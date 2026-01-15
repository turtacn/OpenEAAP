package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/openeeap/openeeap/internal/app/dto"
	"github.com/openeeap/openeeap/internal/app/service"
	"github.com/openeeap/openeeap/internal/domain/workflow"
	"github.com/openeeap/openeeap/internal/infrastructure/repository/postgres"
	"github.com/openeeap/openeeap/internal/platform/orchestrator"
	"github.com/openeeap/openeeap/pkg/types"
	"github.com/openeeap/openeeap/test/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// WorkflowIntegrationTestSuite 工作流集成测试套件
type WorkflowIntegrationTestSuite struct {
	suite.Suite
	ctx             context.Context
	pgContainer     testcontainers.Container
	redisContainer  testcontainers.Container
	db              *gorm.DB
	workflowRepo    workflow.WorkflowRepository
	workflowService *service.WorkflowService
	orchestratorSvc *orchestrator.Orchestrator
	cleanupFuncs    []func()
}

// SetupSuite 测试套件初始化
func (s *WorkflowIntegrationTestSuite) SetupSuite() {
	s.ctx = context.Background()

	// 启动 PostgreSQL 容器
	pgReq := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "openeeap_test",
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test123",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	pgContainer, err := testcontainers.GenericContainer(s.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: pgReq,
		Started:          true,
	})
	require.NoError(s.T(), err)
	s.pgContainer = pgContainer

	// 启动 Redis 容器
	redisReq := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	redisContainer, err := testcontainers.GenericContainer(s.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: redisReq,
		Started:          true,
	})
	require.NoError(s.T(), err)
	s.redisContainer = redisContainer

	// 获取 PostgreSQL 连接信息
	pgHost, err := pgContainer.Host(s.ctx)
	require.NoError(s.T(), err)
	pgPort, err := pgContainer.MappedPort(s.ctx, "5432")
	require.NoError(s.T(), err)

	// 初始化数据库连接
	dsn := fmt.Sprintf("host=%s port=%s user=test password=test123 dbname=openeeap_test sslmode=disable",
		pgHost, pgPort.Port())
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(s.T(), err)
	s.db = db

	// 自动迁移表结构
	err = db.AutoMigrate(
		&workflow.Workflow{},
		&workflow.WorkflowStep{},
		&workflow.WorkflowExecution{},
		&workflow.StepExecution{},
	)
	require.NoError(s.T(), err)

	// 初始化仓储和服务
	s.workflowRepo = postgres.NewWorkflowRepository(db)
	s.workflowService = service.NewWorkflowService(s.workflowRepo)
	s.orchestratorSvc = orchestrator.NewOrchestrator( /* dependencies */ )
}

// TearDownSuite 测试套件清理
func (s *WorkflowIntegrationTestSuite) TearDownSuite() {
	// 执行所有清理函数
	for _, cleanup := range s.cleanupFuncs {
		cleanup()
	}

	// 关闭容器
	if s.pgContainer != nil {
		s.pgContainer.Terminate(s.ctx)
	}
	if s.redisContainer != nil {
		s.redisContainer.Terminate(s.ctx)
	}
}

// SetupTest 每个测试前清理数据
func (s *WorkflowIntegrationTestSuite) SetupTest() {
	s.db.Exec("TRUNCATE TABLE workflows, workflow_steps, workflow_executions, step_executions CASCADE")
}

// TestCreateWorkflow 测试创建工作流
func (s *WorkflowIntegrationTestSuite) TestCreateWorkflow() {
	// 准备测试数据
	req := &dto.CreateWorkflowRequest{
		Name:        "安全事件响应流程",
		Description: "自动化安全事件分析和响应",
		Steps: []dto.WorkflowStepRequest{
			{
				Name:    "事件收集",
				Type:    "agent",
				AgentID: "agent-001",
				Order:   1,
				Config:  json.RawMessage(`{"source": "siem"}`),
			},
			{
				Name:    "威胁分析",
				Type:    "agent",
				AgentID: "agent-002",
				Order:   2,
				Config:  json.RawMessage(`{"model": "gpt-4"}`),
			},
			{
				Name:    "生成报告",
				Type:    "agent",
				AgentID: "agent-003",
				Order:   3,
				Config:  json.RawMessage(`{"format": "pdf"}`),
			},
		},
		Trigger: dto.WorkflowTrigger{
			Type:   "manual",
			Config: json.RawMessage(`{}`),
		},
	}

	// 创建工作流
	wf, err := s.workflowService.CreateWorkflow(s.ctx, req)
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), wf.ID)
	assert.Equal(s.T(), req.Name, wf.Name)
	assert.Equal(s.T(), types.WorkflowStatusDraft, wf.Status)
	assert.Len(s.T(), wf.Steps, 3)

	// 验证步骤顺序
	for i, step := range wf.Steps {
		assert.Equal(s.T(), i+1, step.Order)
	}

	// 验证数据库持久化
	dbWf, err := s.workflowRepo.GetByID(s.ctx, wf.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), wf.ID, dbWf.ID)
	assert.Equal(s.T(), wf.Name, dbWf.Name)
}

// TestRunWorkflow 测试运行工作流
func (s *WorkflowIntegrationTestSuite) TestRunWorkflow() {
	// 创建测试工作流
	wf := fixtures.CreateTestWorkflow(s.T(), s.workflowRepo)

	// 运行工作流
	req := &dto.RunWorkflowRequest{
		WorkflowID: wf.ID,
		Input: map[string]interface{}{
			"alert_id": "ALT-2026-001",
			"severity": "high",
		},
	}

	execution, err := s.workflowService.RunWorkflow(s.ctx, req)
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), execution.ID)
	assert.Equal(s.T(), wf.ID, execution.WorkflowID)
	assert.Equal(s.T(), types.ExecutionStatusRunning, execution.Status)

	// 等待工作流执行完成（模拟异步执行）
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var finalExecution *workflow.WorkflowExecution
	for {
		select {
		case <-timeout:
			s.T().Fatal("工作流执行超时")
		case <-ticker.C:
			exec, err := s.workflowService.GetExecutionStatus(s.ctx, execution.ID)
			require.NoError(s.T(), err)

			if exec.Status == types.ExecutionStatusCompleted ||
				exec.Status == types.ExecutionStatusFailed {
				finalExecution = exec
				goto Done
			}
		}
	}

Done:
	// 验证执行结果
	assert.NotNil(s.T(), finalExecution)
	assert.Equal(s.T(), types.ExecutionStatusCompleted, finalExecution.Status)
	assert.NotNil(s.T(), finalExecution.CompletedAt)

	// 验证步骤执行顺序
	stepExecs, err := s.workflowService.GetStepExecutions(s.ctx, execution.ID)
	require.NoError(s.T(), err)
	assert.Len(s.T(), stepExecs, len(wf.Steps))

	for i, stepExec := range stepExecs {
		assert.Equal(s.T(), i+1, stepExec.Order)
		assert.Equal(s.T(), types.ExecutionStatusCompleted, stepExec.Status)
		assert.NotNil(s.T(), stepExec.Output)
	}
}

// TestPauseAndResumeWorkflow 测试暂停和恢复工作流
func (s *WorkflowIntegrationTestSuite) TestPauseAndResumeWorkflow() {
	// 创建长时间运行的工作流
	wf := fixtures.CreateLongRunningWorkflow(s.T(), s.workflowRepo)

	// 启动工作流
	req := &dto.RunWorkflowRequest{
		WorkflowID: wf.ID,
		Input:      map[string]interface{}{"data": "test"},
	}

	execution, err := s.workflowService.RunWorkflow(s.ctx, req)
	require.NoError(s.T(), err)

	// 等待第一步完成
	time.Sleep(2 * time.Second)

	// 暂停工作流
	err = s.workflowService.PauseWorkflow(s.ctx, execution.ID)
	require.NoError(s.T(), err)

	// 验证暂停状态
	exec, err := s.workflowService.GetExecutionStatus(s.ctx, execution.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), types.ExecutionStatusPaused, exec.Status)
	assert.NotNil(s.T(), exec.PausedAt)

	// 记录暂停前的进度
	stepExecsBefore, err := s.workflowService.GetStepExecutions(s.ctx, execution.ID)
	require.NoError(s.T(), err)
	completedBefore := 0
	for _, stepExec := range stepExecsBefore {
		if stepExec.Status == types.ExecutionStatusCompleted {
			completedBefore++
		}
	}

	// 等待一段时间，确保暂停期间不执行新步骤
	time.Sleep(3 * time.Second)
	stepExecsAfterPause, err := s.workflowService.GetStepExecutions(s.ctx, execution.ID)
	require.NoError(s.T(), err)
	completedAfterPause := 0
	for _, stepExec := range stepExecsAfterPause {
		if stepExec.Status == types.ExecutionStatusCompleted {
			completedAfterPause++
		}
	}
	assert.Equal(s.T(), completedBefore, completedAfterPause, "暂停期间不应执行新步骤")

	// 恢复工作流
	err = s.workflowService.ResumeWorkflow(s.ctx, execution.ID)
	require.NoError(s.T(), err)

	// 验证恢复状态
	exec, err = s.workflowService.GetExecutionStatus(s.ctx, execution.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), types.ExecutionStatusRunning, exec.Status)
	assert.NotNil(s.T(), exec.ResumedAt)

	// 等待工作流完成
	time.Sleep(10 * time.Second)

	// 验证最终完成
	finalExec, err := s.workflowService.GetExecutionStatus(s.ctx, execution.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), types.ExecutionStatusCompleted, finalExec.Status)

	// 验证所有步骤都已完成
	finalStepExecs, err := s.workflowService.GetStepExecutions(s.ctx, execution.ID)
	require.NoError(s.T(), err)
	for _, stepExec := range finalStepExecs {
		assert.Equal(s.T(), types.ExecutionStatusCompleted, stepExec.Status)
	}
}

// TestWorkflowStepFailure 测试步骤失败处理
func (s *WorkflowIntegrationTestSuite) TestWorkflowStepFailure() {
	// 创建包含失败步骤的工作流
	wf := fixtures.CreateWorkflowWithFailingStep(s.T(), s.workflowRepo)

	// 运行工作流
	req := &dto.RunWorkflowRequest{
		WorkflowID: wf.ID,
		Input:      map[string]interface{}{"trigger_error": true},
	}

	execution, err := s.workflowService.RunWorkflow(s.ctx, req)
	require.NoError(s.T(), err)

	// 等待工作流失败
	time.Sleep(5 * time.Second)

	// 验证执行状态
	exec, err := s.workflowService.GetExecutionStatus(s.ctx, execution.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), types.ExecutionStatusFailed, exec.Status)
	assert.NotNil(s.T(), exec.Error)

	// 验证失败步骤
	stepExecs, err := s.workflowService.GetStepExecutions(s.ctx, execution.ID)
	require.NoError(s.T(), err)

	failedStep := findFailedStep(stepExecs)
	assert.NotNil(s.T(), failedStep)
	assert.Equal(s.T(), types.ExecutionStatusFailed, failedStep.Status)
	assert.NotEmpty(s.T(), failedStep.Error)

	// 验证后续步骤未执行
	for _, stepExec := range stepExecs {
		if stepExec.Order > failedStep.Order {
			assert.Equal(s.T(), types.ExecutionStatusPending, stepExec.Status)
		}
	}
}

// TestWorkflowRetry 测试工作流重试
func (s *WorkflowIntegrationTestSuite) TestWorkflowRetry() {
	// 创建可重试的工作流
	wf := fixtures.CreateRetryableWorkflow(s.T(), s.workflowRepo)

	// 首次运行（预期失败）
	req := &dto.RunWorkflowRequest{
		WorkflowID: wf.ID,
		Input:      map[string]interface{}{"retry_count": 0},
	}

	execution, err := s.workflowService.RunWorkflow(s.ctx, req)
	require.NoError(s.T(), err)

	// 等待首次执行失败
	time.Sleep(3 * time.Second)

	exec, err := s.workflowService.GetExecutionStatus(s.ctx, execution.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), types.ExecutionStatusFailed, exec.Status)

	// 重试工作流
	retryReq := &dto.RetryWorkflowRequest{
		ExecutionID: execution.ID,
		FromStep:    "failing_step",
	}

	retryExecution, err := s.workflowService.RetryWorkflow(s.ctx, retryReq)
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), retryExecution.ID)
	assert.Equal(s.T(), execution.ID, retryExecution.ParentExecutionID)

	// 等待重试完成
	time.Sleep(5 * time.Second)

	// 验证重试成功
	retryExec, err := s.workflowService.GetExecutionStatus(s.ctx, retryExecution.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), types.ExecutionStatusCompleted, retryExec.Status)
}

// TestConcurrentWorkflowExecutions 测试并发工作流执行
func (s *WorkflowIntegrationTestSuite) TestConcurrentWorkflowExecutions() {
	wf := fixtures.CreateTestWorkflow(s.T(), s.workflowRepo)

	// 并发启动多个工作流执行
	concurrency := 5
	executions := make([]*workflow.WorkflowExecution, concurrency)
	errChan := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(index int) {
			req := &dto.RunWorkflowRequest{
				WorkflowID: wf.ID,
				Input: map[string]interface{}{
					"execution_index": index,
				},
			}
			exec, err := s.workflowService.RunWorkflow(s.ctx, req)
			if err != nil {
				errChan <- err
				return
			}
			executions[index] = exec
			errChan <- nil
		}(i)
	}

	// 收集启动结果
	for i := 0; i < concurrency; i++ {
		err := <-errChan
		require.NoError(s.T(), err)
	}

	// 验证所有执行都已启动
	for _, exec := range executions {
		assert.NotEmpty(s.T(), exec.ID)
		assert.Equal(s.T(), wf.ID, exec.WorkflowID)
	}

	// 等待所有执行完成
	time.Sleep(15 * time.Second)

	// 验证所有执行都已完成
	for _, exec := range executions {
		finalExec, err := s.workflowService.GetExecutionStatus(s.ctx, exec.ID)
		require.NoError(s.T(), err)
		assert.Equal(s.T(), types.ExecutionStatusCompleted, finalExec.Status)
	}
}

// TestWorkflowStepDataPassing 测试步骤间数据传递
func (s *WorkflowIntegrationTestSuite) TestWorkflowStepDataPassing() {
	// 创建数据传递工作流
	wf := fixtures.CreateDataPassingWorkflow(s.T(), s.workflowRepo)

	// 运行工作流
	req := &dto.RunWorkflowRequest{
		WorkflowID: wf.ID,
		Input: map[string]interface{}{
			"initial_value": 100,
		},
	}

	execution, err := s.workflowService.RunWorkflow(s.ctx, req)
	require.NoError(s.T(), err)

	// 等待完成
	time.Sleep(8 * time.Second)

	// 获取步骤执行结果
	stepExecs, err := s.workflowService.GetStepExecutions(s.ctx, execution.ID)
	require.NoError(s.T(), err)

	// 验证数据传递链
	for i, stepExec := range stepExecs {
		assert.Equal(s.T(), types.ExecutionStatusCompleted, stepExec.Status)
		assert.NotNil(s.T(), stepExec.Output)

		// 验证每步输出作为下一步输入
		if i > 0 {
			prevOutput := stepExecs[i-1].Output
			currentInput := stepExec.Input
			// 验证输入包含前一步的输出
			assert.Contains(s.T(), currentInput, "previous_step_output")
		}
	}

	// 验证最终输出
	finalExec, err := s.workflowService.GetExecutionStatus(s.ctx, execution.ID)
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), finalExec.Output)

	var output map[string]interface{}
	err = json.Unmarshal(finalExec.Output, &output)
	require.NoError(s.T(), err)
	assert.Contains(s.T(), output, "final_result")
}

// Helper functions

func findFailedStep(stepExecs []*workflow.StepExecution) *workflow.StepExecution {
	for _, stepExec := range stepExecs {
		if stepExec.Status == types.ExecutionStatusFailed {
			return stepExec
		}
	}
	return nil
}

// TestWorkflowIntegration 测试入口
func TestWorkflowIntegration(t *testing.T) {
	suite.Run(t, new(WorkflowIntegrationTestSuite))
}

//Personal.AI order the ending
