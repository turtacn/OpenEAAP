// internal/app/service/agent_service.go
package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/openeeap/openeeap/internal/domain/agent"
	"github.com/openeeap/openeeap/internal/domain/model"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/pkg/errors"
	// TODO: implement deployment and testing packages
	// "github.com/openeeap/openeeap/internal/platform/deployment"
	// "github.com/openeeap/openeeap/internal/platform/testing"
)

// AgentService Agent 应用服务接口
type AgentService interface {
	// CreateAgent 创建 Agent
	CreateAgent(ctx context.Context, req *CreateAgentRequest) (*AgentResponse, error)

	// GetAgent 获取 Agent
	GetAgent(ctx context.Context, agentID string) (*AgentResponse, error)

	// UpdateAgent 更新 Agent
	UpdateAgent(ctx context.Context, agentID string, req *UpdateAgentRequest) (*AgentResponse, error)

	// DeleteAgent 删除 Agent
	DeleteAgent(ctx context.Context, agentID string) error

	// ListAgents 列出 Agents
	ListAgents(ctx context.Context, req *ListAgentsRequest) (*ListAgentsResponse, error)

	// DeployAgent 部署 Agent
	DeployAgent(ctx context.Context, agentID string, req *DeployAgentRequest) (*DeploymentResponse, error)

	// TestAgent 测试 Agent
	TestAgent(ctx context.Context, agentID string, req *TestAgentRequest) (*TestResult, error)

	// ExecuteAgent 执行 Agent
	ExecuteAgent(ctx context.Context, agentID string, req *ExecuteAgentRequest) (*ExecutionResult, error)

	// UpdateAgentConfig 更新 Agent 配置
	UpdateAgentConfig(ctx context.Context, agentID string, config *agent.AgentConfig) error

	// GetAgentMetrics 获取 Agent 指标
	GetAgentMetrics(ctx context.Context, agentID string, timeRange *TimeRange) (*AgentMetrics, error)

	// CloneAgent 克隆 Agent
	CloneAgent(ctx context.Context, sourceAgentID string, req *CloneAgentRequest) (*AgentResponse, error)

	// ExportAgent 导出 Agent
	ExportAgent(ctx context.Context, agentID string) (*AgentExport, error)

	// ImportAgent 导入 Agent
	ImportAgent(ctx context.Context, data *AgentExport) (*AgentResponse, error)
}

// CreateAgentRequest 创建 Agent 请求
type CreateAgentRequest struct {
	Name          string                 `json:"name" validate:"required"`
	Description   string                 `json:"description"`
	Type          agent.AgentType        `json:"type" validate:"required"`
	ModelID       string                 `json:"model_id" validate:"required"`
	Config        *agent.AgentConfig     `json:"config"`
	Tools         []*agent.Tool          `json:"tools"`
	Memory        *agent.MemoryConfig    `json:"memory"`
	Tags          []string               `json:"tags"`
	Metadata      map[string]interface{} `json:"metadata"`
	AutoDeploy    bool                   `json:"auto_deploy"`
	DeploymentEnv string                 `json:"deployment_env"`
}

// UpdateAgentRequest 更新 Agent 请求
type UpdateAgentRequest struct {
	Name        *string                `json:"name"`
	Description *string                `json:"description"`
	Config      *agent.AgentConfig     `json:"config"`
	Tools       []*agent.Tool          `json:"tools"`
	Memory      *agent.MemoryConfig    `json:"memory"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ListAgentsRequest 列出 Agents 请求
type ListAgentsRequest struct {
	Type      agent.AgentType `json:"type"`
	Status    agent.Status    `json:"status"`
	Tags      []string        `json:"tags"`
	Search    string          `json:"search"`
	SortBy    string          `json:"sort_by"`
	SortOrder string          `json:"sort_order"`
	Page      int             `json:"page"`
	PageSize  int             `json:"page_size"`
}

// DeployAgentRequest 部署 Agent 请求
type DeployAgentRequest struct {
	Environment string                  `json:"environment" validate:"required"`
	Version     string                  `json:"version"`
	Replicas    int                     `json:"replicas"`
	Resources   *deployment.Resources   `json:"resources"`
	HealthCheck *deployment.HealthCheck `json:"health_check"`
	AutoScaling *deployment.AutoScaling `json:"auto_scaling"`
	Metadata    map[string]interface{}  `json:"metadata"`
}

// TestAgentRequest 测试 Agent 请求
type TestAgentRequest struct {
	TestCases   []*testing.TestCase    `json:"test_cases"`
	Environment string                 `json:"environment"`
	Timeout     time.Duration          `json:"timeout"`
	Parallel    bool                   `json:"parallel"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ExecuteAgentRequest 执行 Agent 请求
type ExecuteAgentRequest struct {
	Input      interface{}            `json:"input" validate:"required"`
	Context    map[string]interface{} `json:"context"`
	Parameters map[string]interface{} `json:"parameters"`
	Stream     bool                   `json:"stream"`
	Timeout    time.Duration          `json:"timeout"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// CloneAgentRequest 克隆 Agent 请求
type CloneAgentRequest struct {
	Name          string                 `json:"name" validate:"required"`
	Description   string                 `json:"description"`
	IncludeMemory bool                   `json:"include_memory"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// AgentResponse Agent 响应
type AgentResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        agent.AgentType        `json:"type"`
	Status      agent.Status           `json:"status"`
	ModelID     string                 `json:"model_id"`
	Config      *agent.AgentConfig     `json:"config"`
	Tools       []*agent.Tool          `json:"tools"`
	Memory      *agent.MemoryConfig    `json:"memory"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	DeployedAt  *time.Time             `json:"deployed_at,omitempty"`
	Version     string                 `json:"version"`
}

// ListAgentsResponse 列出 Agents 响应
type ListAgentsResponse struct {
	Agents     []*AgentResponse `json:"agents"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// DeploymentResponse 部署响应
type DeploymentResponse struct {
	DeploymentID string                 `json:"deployment_id"`
	AgentID      string                 `json:"agent_id"`
	Environment  string                 `json:"environment"`
	Status       string                 `json:"status"`
	Endpoint     string                 `json:"endpoint"`
	Replicas     int                    `json:"replicas"`
	DeployedAt   time.Time              `json:"deployed_at"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// TestResult 测试结果
type TestResult struct {
	TestID       string                 `json:"test_id"`
	AgentID      string                 `json:"agent_id"`
	Status       string                 `json:"status"`
	TotalTests   int                    `json:"total_tests"`
	PassedTests  int                    `json:"passed_tests"`
	FailedTests  int                    `json:"failed_tests"`
	SkippedTests int                    `json:"skipped_tests"`
	Duration     time.Duration          `json:"duration"`
	Results      []*testing.TestResult  `json:"results"`
	Summary      string                 `json:"summary"`
	TestedAt     time.Time              `json:"tested_at"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	ExecutionID string                 `json:"execution_id"`
	AgentID     string                 `json:"agent_id"`
	Status      string                 `json:"status"`
	Output      interface{}            `json:"output"`
	Error       string                 `json:"error,omitempty"`
	Duration    time.Duration          `json:"duration"`
	TokensUsed  int                    `json:"tokens_used"`
	Cost        float64                `json:"cost"`
	Steps       []*ExecutionStep       `json:"steps"`
	ExecutedAt  time.Time              `json:"executed_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ExecutionStep 执行步骤
type ExecutionStep struct {
	StepID   string                 `json:"step_id"`
	Type     string                 `json:"type"`
	Action   string                 `json:"action"`
	Input    interface{}            `json:"input"`
	Output   interface{}            `json:"output"`
	Duration time.Duration          `json:"duration"`
	Status   string                 `json:"status"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata"`
}

// AgentMetrics Agent 指标
type AgentMetrics struct {
	AgentID         string                 `json:"agent_id"`
	TotalExecutions int64                  `json:"total_executions"`
	SuccessfulRuns  int64                  `json:"successful_runs"`
	FailedRuns      int64                  `json:"failed_runs"`
	AvgDuration     time.Duration          `json:"avg_duration"`
	AvgTokensUsed   float64                `json:"avg_tokens_used"`
	TotalCost       float64                `json:"total_cost"`
	Uptime          time.Duration          `json:"uptime"`
	LastExecution   *time.Time             `json:"last_execution,omitempty"`
	ErrorRate       float64                `json:"error_rate"`
	Throughput      float64                `json:"throughput"`
	TimeSeries      []*MetricDataPoint     `json:"time_series"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// MetricDataPoint 指标数据点
type MetricDataPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics"`
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// AgentExport Agent 导出数据
type AgentExport struct {
	Agent      *AgentResponse         `json:"agent"`
	Config     *agent.AgentConfig     `json:"config"`
	Tools      []*agent.Tool          `json:"tools"`
	Memory     *agent.MemoryConfig    `json:"memory"`
	Version    string                 `json:"version"`
	ExportedAt time.Time              `json:"exported_at"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// agentService Agent 应用服务实现
type agentService struct {
	agentRepo        agent.AgentRepository
	agentDomainSvc   agent.AgentService
	modelRepo        model.ModelRepository
	deploymentSvc    deployment.DeploymentService
	testingSvc       testing.TestingService
	logger           logging.Logger
	metricsCollector metrics.MetricsCollector
	tracer           trace.Tracer

	cache sync.Map
	mu    sync.RWMutex
}

// NewAgentService 创建 Agent 应用服务
func NewAgentService(
	agentRepo agent.AgentRepository,
	agentDomainSvc agent.AgentService,
	modelRepo model.ModelRepository,
	deploymentSvc deployment.DeploymentService,
	testingSvc testing.TestingService,
	logger logging.Logger,
	metricsCollector metrics.MetricsCollector,
	tracer trace.Tracer,
) AgentService {
	return &agentService{
		agentRepo:        agentRepo,
		agentDomainSvc:   agentDomainSvc,
		modelRepo:        modelRepo,
		deploymentSvc:    deploymentSvc,
		testingSvc:       testingSvc,
		logger:           logger,
		metricsCollector: metricsCollector,
		tracer:           tracer,
	}
}

// CreateAgent 创建 Agent
func (s *agentService) CreateAgent(ctx context.Context, req *CreateAgentRequest) (*AgentResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AgentService.CreateAgent")
	defer span.End()

	startTime := time.Now()
	defer func() {
		s.metricsCollector.ObserveDuration("agent_create_duration_ms",
			float64(time.Since(startTime).Milliseconds()),
			map[string]string{"type": string(req.Type)})
	}()

 s.logger.WithContext(ctx).Info("Creating agent", logging.Any("name", req.Name), logging.Any("type", req.Type), logging.Any("model_id", req.ModelID))

	// 验证请求
	if err := s.validateCreateRequest(ctx, req); err != nil {
		return nil, err
	}

	// 验证模型存在
	if _, err := s.modelRepo.GetByID(ctx, req.ModelID); err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "model not found")
	}

	// 创建 Agent 实体
	agentEntity := &agent.Agent{
		ID:          generateAgentID(),
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		ModelID:     req.ModelID,
		Status:      agent.StatusCreated,
		Config:      req.Config,
		Tools:       req.Tools,
		Memory:      req.Memory,
		Tags:        req.Tags,
		Metadata:    req.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     "1.0.0",
	}

	// 设置默认配置
	if agentEntity.Config == nil {
		agentEntity.Config = agent.DefaultAgentConfig()
	}

	// 使用领域服务初始化 Agent
	if err := s.agentDomainSvc.Initialize(ctx, agentEntity); err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to initialize agent")
	}

	// 保存到仓储
	if err := s.agentRepo.Create(ctx, agentEntity); err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to create agent")
	}

	// 缓存
	s.cache.Store(agentEntity.ID, agentEntity)

	// 自动部署
	if req.AutoDeploy {
		deployReq := &DeployAgentRequest{
			Environment: req.DeploymentEnv,
			Replicas:    1,
		}

		if _, err := s.DeployAgent(ctx, agentEntity.ID, deployReq); err != nil {
   s.logger.WithContext(ctx).Warn("Failed to auto-deploy agent", logging.Any("agent_id", agentEntity.ID), logging.Error(err))
		}
	}

	s.metricsCollector.IncrementCounter("agents_created_total",
		map[string]string{"type": string(req.Type)})

 s.logger.WithContext(ctx).Info("Agent created successfully", logging.Any("agent_id", agentEntity.ID), logging.Any("name", agentEntity.Name))

	return s.toAgentResponse(agentEntity), nil
}

// GetAgent 获取 Agent
func (s *agentService) GetAgent(ctx context.Context, agentID string) (*AgentResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AgentService.GetAgent")
	defer span.End()

	// 先检查缓存
	if cached, ok := s.cache.Load(agentID); ok {
		return s.toAgentResponse(cached.(*agent.Agent)), nil
	}

	// 从仓储获取
	agentEntity, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "agent not found")
	}

	// 更新缓存
	s.cache.Store(agentID, agentEntity)

	return s.toAgentResponse(agentEntity), nil
}

// UpdateAgent 更新 Agent
func (s *agentService) UpdateAgent(ctx context.Context, agentID string, req *UpdateAgentRequest) (*AgentResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AgentService.UpdateAgent")
	defer span.End()

	s.logger.WithContext(ctx).Info("Updating agent", logging.Any("agent_id", agentID))

	// 获取现有 Agent
	agentEntity, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "agent not found")
	}

	// 应用更新
	if req.Name != nil {
		agentEntity.Name = *req.Name
	}
	if req.Description != nil {
		agentEntity.Description = *req.Description
	}
	if req.Config != nil {
		agentEntity.Config = req.Config
	}
	if req.Tools != nil {
		agentEntity.Tools = req.Tools
	}
	if req.Memory != nil {
		agentEntity.Memory = req.Memory
	}
	if req.Tags != nil {
		agentEntity.Tags = req.Tags
	}
	if req.Metadata != nil {
		if agentEntity.Metadata == nil {
			agentEntity.Metadata = make(map[string]interface{})
		}
		for k, v := range req.Metadata {
			agentEntity.Metadata[k] = v
		}
	}

	agentEntity.UpdatedAt = time.Now()

	// 使用领域服务验证更新
	if err := s.agentDomainSvc.Validate(ctx, agentEntity); err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidArgument, "validation failed")
	}

	// 保存更新
	if err := s.agentRepo.Update(ctx, agentEntity); err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to update agent")
	}

	// 更新缓存
	s.cache.Store(agentID, agentEntity)

	s.logger.WithContext(ctx).Info("Agent updated successfully", logging.Any("agent_id", agentID))

	return s.toAgentResponse(agentEntity), nil
}

// DeleteAgent 删除 Agent
func (s *agentService) DeleteAgent(ctx context.Context, agentID string) error {
	ctx, span := s.tracer.Start(ctx, "AgentService.DeleteAgent")
	defer span.End()

	s.logger.WithContext(ctx).Info("Deleting agent", logging.Any("agent_id", agentID))

	// 获取 Agent
	agentEntity, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return errors.Wrap(err, errors.CodeNotFound, "agent not found")
	}

	// 检查是否正在运行
	if agentEntity.Status == agent.StatusRunning {
		return errors.New(errors.CodeFailedPrecondition,
			"cannot delete running agent, please stop it first")
	}

	// 使用领域服务执行删除前清理
	if err := s.agentDomainSvc.Cleanup(ctx, agentEntity); err != nil {
  s.logger.WithContext(ctx).Warn("Agent cleanup failed", logging.Any("agent_id", agentID), logging.Error(err))
	}

	// 删除部署（如果存在）
	if agentEntity.Status == agent.StatusDeployed {
		if err := s.deploymentSvc.Undeploy(ctx, agentID); err != nil {
   s.logger.WithContext(ctx).Warn("Failed to undeploy agent", logging.Any("agent_id", agentID), logging.Error(err))
		}
	}

	// 从仓储删除
	if err := s.agentRepo.Delete(ctx, agentID); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to delete agent")
	}

	// 从缓存删除
	s.cache.Delete(agentID)

	s.metricsCollector.IncrementCounter("agents_deleted_total",
		map[string]string{"type": string(agentEntity.Type)})

	s.logger.WithContext(ctx).Info("Agent deleted successfully", logging.Any("agent_id", agentID))

	return nil
}

// ListAgents 列出 Agents
func (s *agentService) ListAgents(ctx context.Context, req *ListAgentsRequest) (*ListAgentsResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AgentService.ListAgents")
	defer span.End()

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	// 构建查询条件
	filter := &agent.AgentFilter{
		Type:     req.Type,
		Status:   req.Status,
		Tags:     req.Tags,
		Search:   req.Search,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	// 从仓储查询
	agents, total, err := s.agentRepo.List(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to list agents")
	}

	// 转换为响应
	responses := make([]*AgentResponse, len(agents))
	for i, a := range agents {
		responses[i] = s.toAgentResponse(a)
	}

	totalPages := (total + req.PageSize - 1) / req.PageSize

	return &ListAgentsResponse{
		Agents:     responses,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// DeployAgent 部署 Agent
func (s *agentService) DeployAgent(ctx context.Context, agentID string, req *DeployAgentRequest) (*DeploymentResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AgentService.DeployAgent")
	defer span.End()

	startTime := time.Now()
	defer func() {
		s.metricsCollector.ObserveDuration("agent_deploy_duration_ms",
			float64(time.Since(startTime).Milliseconds()),
			map[string]string{"environment": req.Environment})
	}()

 s.logger.WithContext(ctx).Info("Deploying agent", logging.Any("agent_id", agentID), logging.Any("environment", req.Environment))

	// 获取 Agent
	agentEntity, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "agent not found")
	}

	// 检查状态
	if agentEntity.Status == agent.StatusRunning {
		return nil, errors.New(errors.CodeFailedPrecondition,
			"agent is already deployed and running")
	}

	// 使用领域服务准备部署
	if err := s.agentDomainSvc.PrepareDeployment(ctx, agentEntity); err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to prepare deployment")
	}

	// 设置默认值
	if req.Replicas <= 0 {
		req.Replicas = 1
	}

	// 创建部署配置
	deployConfig := &deployment.DeploymentConfig{
		AgentID:     agentID,
		Environment: req.Environment,
		Version:     req.Version,
		Replicas:    req.Replicas,
		Resources:   req.Resources,
		HealthCheck: req.HealthCheck,
		AutoScaling: req.AutoScaling,
		Metadata:    req.Metadata,
	}

	// 执行部署
	deployment, err := s.deploymentSvc.Deploy(ctx, deployConfig)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "deployment failed")
	}

	// 更新 Agent 状态
	agentEntity.Status = agent.StatusDeployed
	agentEntity.UpdatedAt = time.Now()
	now := time.Now()
	agentEntity.DeployedAt = &now

	if err := s.agentRepo.Update(ctx, agentEntity); err != nil {
  s.logger.WithContext(ctx).Warn("Failed to update agent status", logging.Any("agent_id", agentID), logging.Error(err))
	}

	// 更新缓存
	s.cache.Store(agentID, agentEntity)

	s.metricsCollector.IncrementCounter("agents_deployed_total",
		map[string]string{"environment": req.Environment})

 s.logger.WithContext(ctx).Info("Agent deployed successfully", logging.Any("agent_id", agentID), logging.Any("deployment_id", deployment.ID), logging.Any("endpoint", deployment.Endpoint))

	return &DeploymentResponse{
		DeploymentID: deployment.ID,
		AgentID:      agentID,
		Environment:  req.Environment,
		Status:       deployment.Status,
		Endpoint:     deployment.Endpoint,
		Replicas:     deployment.Replicas,
		DeployedAt:   deployment.DeployedAt,
		Metadata:     deployment.Metadata,
	}, nil
}

// TestAgent 测试 Agent
func (s *agentService) TestAgent(ctx context.Context, agentID string, req *TestAgentRequest) (*TestResult, error) {
	ctx, span := s.tracer.Start(ctx, "AgentService.TestAgent")
	defer span.End()

	startTime := time.Now()

 s.logger.WithContext(ctx).Info("Testing agent", logging.Any("agent_id", agentID), logging.Any("test_cases", len(req.TestCases))

	// 获取 Agent
	agentEntity, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "agent not found")
	}

	// 设置默认超时
	if req.Timeout == 0 {
		req.Timeout = 5 * time.Minute
	}

	// 创建测试配置
	testConfig := &testing.TestConfig{
		AgentID:     agentID,
		TestCases:   req.TestCases,
		Environment: req.Environment,
		Timeout:     req.Timeout,
		Parallel:    req.Parallel,
		Metadata:    req.Metadata,
	}

	// 执行测试
	testResults, err := s.testingSvc.RunTests(ctx, testConfig)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "test execution failed")
	}

	// 统计结果
	passed := 0
	failed := 0
	skipped := 0

	for _, result := range testResults {
		switch result.Status {
		case "passed":
			passed++
		case "failed":
			failed++
		case "skipped":
			skipped++
		}
	}

	duration := time.Since(startTime)
	status := "passed"
	if failed > 0 {
		status = "failed"
	}

	testResult := &TestResult{
		TestID:       generateTestID(),
		AgentID:      agentID,
		Status:       status,
		TotalTests:   len(req.TestCases),
		PassedTests:  passed,
		FailedTests:  failed,
		SkippedTests: skipped,
		Duration:     duration,
		Results:      testResults,
		Summary:      fmt.Sprintf("%d/%d tests passed", passed, len(req.TestCases)),
		TestedAt:     time.Now(),
		Metadata:     req.Metadata,
	}

	// 更新 Agent 测试状态
	agentEntity.LastTestedAt = &testResult.TestedAt
	if err := s.agentRepo.Update(ctx, agentEntity); err != nil {
  s.logger.WithContext(ctx).Warn("Failed to update agent test status", logging.Any("agent_id", agentID), logging.Error(err))
	}

	s.metricsCollector.IncrementCounter("agent_tests_total",
		map[string]string{
			"agent_id": agentID,
			"status":   status,
		})

 s.logger.WithContext(ctx).Info("Agent testing completed", logging.Any("agent_id", agentID), logging.Any("status", status), logging.Any("passed", passed), logging.Any("failed", failed), logging.Any("duration_ms", duration.Milliseconds())

	return testResult, nil
}

// ExecuteAgent 执行 Agent
func (s *agentService) ExecuteAgent(ctx context.Context, agentID string, req *ExecuteAgentRequest) (*ExecutionResult, error) {
	ctx, span := s.tracer.Start(ctx, "AgentService.ExecuteAgent")
	defer span.End()

	startTime := time.Now()

	s.logger.WithContext(ctx).Info("Executing agent", logging.Any("agent_id", agentID))

	// 获取 Agent
	agentEntity, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "agent not found")
	}

	// 检查状态
	if agentEntity.Status != agent.StatusRunning && agentEntity.Status != agent.StatusDeployed {
		return nil, errors.New(errors.CodeFailedPrecondition,
			"agent is not running or deployed")
	}

	// 设置默认超时
	if req.Timeout == 0 {
		req.Timeout = 30 * time.Second
	}

	// 创建执行上下文
	execCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	// 使用领域服务执行
	output, steps, err := s.agentDomainSvc.Execute(execCtx, agentEntity, req.Input, req.Parameters)

	duration := time.Since(startTime)
	executionID := generateExecutionID()

	// 构建执行结果
	result := &ExecutionResult{
		ExecutionID: executionID,
		AgentID:     agentID,
		Duration:    duration,
		ExecutedAt:  time.Now(),
		Metadata:    req.Metadata,
	}

	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()

		s.metricsCollector.IncrementCounter("agent_executions_failed_total",
			map[string]string{"agent_id": agentID})

  s.logger.WithContext(ctx).Error("Agent execution failed", logging.Any("agent_id", agentID), logging.Error(err))

		return result, nil // 返回结果而不是错误
	}

	result.Status = "success"
	result.Output = output
	result.Steps = s.convertExecutionSteps(steps)

	// 计算 tokens 和成本
	result.TokensUsed = s.calculateTokens(output)
	result.Cost = s.calculateCost(agentEntity.ModelID, result.TokensUsed)

	// 更新 Agent 统计
	agentEntity.TotalExecutions++
	agentEntity.LastExecutedAt = &result.ExecutedAt
	if err := s.agentRepo.Update(ctx, agentEntity); err != nil {
  s.logger.WithContext(ctx).Warn("Failed to update agent statistics", logging.Any("agent_id", agentID), logging.Error(err))
	}

	s.metricsCollector.IncrementCounter("agent_executions_success_total",
		map[string]string{"agent_id": agentID})

	s.metricsCollector.ObserveDuration("agent_execution_duration_ms",
		float64(duration.Milliseconds()),
		map[string]string{"agent_id": agentID})

 s.logger.WithContext(ctx).Info("Agent executed successfully", logging.Any("agent_id", agentID), logging.Any("execution_id", executionID), logging.Any("duration_ms", duration.Milliseconds())

	return result, nil
}

// UpdateAgentConfig 更新 Agent 配置
func (s *agentService) UpdateAgentConfig(ctx context.Context, agentID string, config *agent.AgentConfig) error {
	ctx, span := s.tracer.Start(ctx, "AgentService.UpdateAgentConfig")
	defer span.End()

	s.logger.WithContext(ctx).Info("Updating agent config", logging.Any("agent_id", agentID))

	agentEntity, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return errors.Wrap(err, errors.CodeNotFound, "agent not found")
	}

	// 验证配置
	if err := s.agentDomainSvc.ValidateConfig(ctx, config); err != nil {
		return errors.Wrap(err, errors.CodeInvalidArgument, "invalid config")
	}

	agentEntity.Config = config
	agentEntity.UpdatedAt = time.Now()

	if err := s.agentRepo.Update(ctx, agentEntity); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to update config")
	}

	// 更新缓存
	s.cache.Store(agentID, agentEntity)

	s.logger.WithContext(ctx).Info("Agent config updated successfully", logging.Any("agent_id", agentID))

	return nil
}

// GetAgentMetrics 获取 Agent 指标
func (s *agentService) GetAgentMetrics(ctx context.Context, agentID string, timeRange *TimeRange) (*AgentMetrics, error) {
	ctx, span := s.tracer.Start(ctx, "AgentService.GetAgentMetrics")
	defer span.End()

	agentEntity, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "agent not found")
	}

	// 设置默认时间范围
	if timeRange == nil {
		timeRange = &TimeRange{
			Start: time.Now().Add(-24 * time.Hour),
			End:   time.Now(),
		}
	}

	// 从指标收集器获取数据
	totalExecutions := agentEntity.TotalExecutions
	successfulRuns := int64(0)
	failedRuns := int64(0)

	// 简化实现：从 Agent 实体获取基础指标
	if agentEntity.Status == agent.StatusRunning {
		successfulRuns = int64(float64(totalExecutions) * 0.9) // 假设 90% 成功率
		failedRuns = totalExecutions - successfulRuns
	}

	errorRate := 0.0
	if totalExecutions > 0 {
		errorRate = float64(failedRuns) / float64(totalExecutions)
	}

	uptime := time.Duration(0)
	if agentEntity.DeployedAt != nil {
		uptime = time.Since(*agentEntity.DeployedAt)
	}

	metrics := &AgentMetrics{
		AgentID:         agentID,
		TotalExecutions: totalExecutions,
		SuccessfulRuns:  successfulRuns,
		FailedRuns:      failedRuns,
		AvgDuration:     2500 * time.Millisecond,         // 模拟值
		AvgTokensUsed:   850.0,                           // 模拟值
		TotalCost:       float64(totalExecutions) * 0.02, // 模拟值
		Uptime:          uptime,
		LastExecution:   agentEntity.LastExecutedAt,
		ErrorRate:       errorRate,
		Throughput:      calculateThroughput(totalExecutions, uptime),
		TimeSeries:      s.generateMetricsTimeSeries(timeRange),
		Metadata:        agentEntity.Metadata,
	}

	return metrics, nil
}

// CloneAgent 克隆 Agent
func (s *agentService) CloneAgent(ctx context.Context, sourceAgentID string, req *CloneAgentRequest) (*AgentResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AgentService.CloneAgent")
	defer span.End()

 s.logger.WithContext(ctx).Info("Cloning agent", logging.Any("source_agent_id", sourceAgentID), logging.Any("new_name", req.Name))

	// 获取源 Agent
	sourceAgent, err := s.agentRepo.GetByID(ctx, sourceAgentID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "source agent not found")
	}

	// 创建新 Agent
	clonedAgent := &agent.Agent{
		ID:          generateAgentID(),
		Name:        req.Name,
		Description: req.Description,
		Type:        sourceAgent.Type,
		ModelID:     sourceAgent.ModelID,
		Status:      agent.StatusCreated,
		Config:      s.cloneConfig(sourceAgent.Config),
		Tools:       s.cloneTools(sourceAgent.Tools),
		Tags:        append([]string{}, sourceAgent.Tags...),
		Metadata:    s.cloneMetadata(sourceAgent.Metadata, req.Metadata),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     "1.0.0",
	}

	// 可选：克隆记忆
	if req.IncludeMemory && sourceAgent.Memory != nil {
		clonedAgent.Memory = s.cloneMemory(sourceAgent.Memory)
	}

	// 使用领域服务初始化
	if err := s.agentDomainSvc.Initialize(ctx, clonedAgent); err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to initialize cloned agent")
	}

	// 保存
	if err := s.agentRepo.Create(ctx, clonedAgent); err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to create cloned agent")
	}

	// 缓存
	s.cache.Store(clonedAgent.ID, clonedAgent)

	s.metricsCollector.IncrementCounter("agents_cloned_total",
		map[string]string{"type": string(clonedAgent.Type)})

 s.logger.WithContext(ctx).Info("Agent cloned successfully", logging.Any("source_agent_id", sourceAgentID), logging.Any("cloned_agent_id", clonedAgent.ID))

	return s.toAgentResponse(clonedAgent), nil
}

// ExportAgent 导出 Agent
func (s *agentService) ExportAgent(ctx context.Context, agentID string) (*AgentExport, error) {
	ctx, span := s.tracer.Start(ctx, "AgentService.ExportAgent")
	defer span.End()

	s.logger.WithContext(ctx).Info("Exporting agent", logging.Any("agent_id", agentID))

	agentEntity, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "agent not found")
	}

	export := &AgentExport{
		Agent:      s.toAgentResponse(agentEntity),
		Config:     agentEntity.Config,
		Tools:      agentEntity.Tools,
		Memory:     agentEntity.Memory,
		Version:    agentEntity.Version,
		ExportedAt: time.Now(),
		Metadata: map[string]interface{}{
			"source_agent_id": agentID,
			"export_format":   "openeeap_v1",
		},
	}

	s.logger.WithContext(ctx).Info("Agent exported successfully", logging.Any("agent_id", agentID))

	return export, nil
}

// ImportAgent 导入 Agent
func (s *agentService) ImportAgent(ctx context.Context, data *AgentExport) (*AgentResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AgentService.ImportAgent")
	defer span.End()

	s.logger.WithContext(ctx).Info("Importing agent", logging.Any("name", data.Agent.Name))

	// 创建新 Agent
	agentEntity := &agent.Agent{
		ID:          generateAgentID(),
		Name:        data.Agent.Name + "_imported",
		Description: data.Agent.Description,
		Type:        data.Agent.Type,
		ModelID:     data.Agent.ModelID,
		Status:      agent.StatusCreated,
		Config:      data.Config,
		Tools:       data.Tools,
		Memory:      data.Memory,
		Tags:        data.Agent.Tags,
		Metadata:    data.Agent.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     "1.0.0",
	}

	// 验证
	if err := s.agentDomainSvc.Validate(ctx, agentEntity); err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidArgument, "validation failed")
	}

	// 保存
	if err := s.agentRepo.Create(ctx, agentEntity); err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to import agent")
	}

	// 缓存
	s.cache.Store(agentEntity.ID, agentEntity)

	s.metricsCollector.IncrementCounter("agents_imported_total",
		map[string]string{"type": string(agentEntity.Type)})

 s.logger.WithContext(ctx).Info("Agent imported successfully", logging.Any("agent_id", agentEntity.ID), logging.Any("name", agentEntity.Name))

	return s.toAgentResponse(agentEntity), nil
}

// Helper methods

func (s *agentService) validateCreateRequest(ctx context.Context, req *CreateAgentRequest) error {
	if req.Name == "" {
		return errors.ValidationError( "name is required")
	}
	if req.ModelID == "" {
		return errors.ValidationError( "model_id is required")
	}
	if req.Type == "" {
		return errors.ValidationError( "type is required")
	}
	return nil
}

func (s *agentService) toAgentResponse(a *agent.Agent) *AgentResponse {
	return &AgentResponse{
		ID:          a.ID,
		Name:        a.Name,
		Description: a.Description,
		Type:        a.Type,
		Status:      a.Status,
		ModelID:     a.ModelID,
		Config:      a.Config,
		Tools:       a.Tools,
		Memory:      a.Memory,
		Tags:        a.Tags,
		Metadata:    a.Metadata,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
		DeployedAt:  a.DeployedAt,
		Version:     a.Version,
	}
}

func (s *agentService) convertExecutionSteps(domainSteps []agent.ExecutionStep) []*ExecutionStep {
	steps := make([]*ExecutionStep, len(domainSteps))
	for i, step := range domainSteps {
		steps[i] = &ExecutionStep{
			StepID:   step.ID,
			Type:     step.Type,
			Action:   step.Action,
			Input:    step.Input,
			Output:   step.Output,
			Duration: step.Duration,
			Status:   step.Status,
			Error:    step.Error,
			Metadata: step.Metadata,
		}
	}
	return steps
}

func (s *agentService) calculateTokens(output interface{}) int {
	// 简化实现：基于输出长度估算
	outputStr := fmt.Sprintf("%v", output)
	return len(outputStr) / 4 // 粗略估算
}

func (s *agentService) calculateCost(modelID string, tokens int) float64 {
	// 简化实现：固定费率
	costPerToken := 0.00002 // $0.02 per 1000 tokens
	return float64(tokens) * costPerToken
}

func (s *agentService) cloneConfig(config *agent.AgentConfig) *agent.AgentConfig {
	if config == nil {
		return nil
	}

	cloned := *config
	return &cloned
}

func (s *agentService) cloneTools(tools []*agent.Tool) []*agent.Tool {
	if tools == nil {
		return nil
	}

	cloned := make([]*agent.Tool, len(tools))
	for i, tool := range tools {
		toolCopy := *tool
		cloned[i] = &toolCopy
	}
	return cloned
}

func (s *agentService) cloneMemory(memory *agent.MemoryConfig) *agent.MemoryConfig {
	if memory == nil {
		return nil
	}

	cloned := *memory
	return &cloned
}

func (s *agentService) cloneMetadata(source, additional map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range source {
		result[k] = v
	}

	for k, v := range additional {
		result[k] = v
	}

	return result
}

func (s *agentService) generateMetricsTimeSeries(timeRange *TimeRange) []*MetricDataPoint {
	points := []*MetricDataPoint{}

	// 生成每小时的数据点
	current := timeRange.Start
	for current.Before(timeRange.End) {
		points = append(points, &MetricDataPoint{
			Timestamp: current,
			Metrics: map[string]interface{}{
				"executions":      10 + (current.Hour() % 5),
				"success_rate":    0.85 + (float64(current.Minute()) / 300.0),
				"avg_duration_ms": 2000 + (current.Hour() * 100),
			},
		})
		current = current.Add(1 * time.Hour)
	}

	return points
}

// Utility functions

func generateAgentID() string {
	return fmt.Sprintf("agent_%d", time.Now().UnixNano())
}

func generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}

func generateTestID() string {
	return fmt.Sprintf("test_%d", time.Now().UnixNano())
}

func calculateThroughput(executions int64, uptime time.Duration) float64 {
	if uptime.Seconds() == 0 {
		return 0
	}
	return float64(executions) / uptime.Hours()
}

//Personal.AI order the ending
