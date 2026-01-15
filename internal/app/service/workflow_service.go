package service

import (
	"context"
	"fmt"
	"time"

	"github.com/openeeap/openeeap/internal/app/dto"
	"github.com/openeeap/openeeap/internal/domain/workflow"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/internal/platform/orchestrator"
	"github.com/openeeap/openeeap/pkg/errors"
	"github.com/openeeap/openeeap/pkg/types"
)

// WorkflowService 工作流应用服务接口
type WorkflowService interface {
	// Create 创建工作流
	Create(ctx context.Context, req *dto.CreateWorkflowRequest) (*dto.WorkflowResponse, error)

	// GetByID 根据 ID 获取工作流
	GetByID(ctx context.Context, id string) (*dto.WorkflowResponse, error)

	// List 列出工作流
	List(ctx context.Context, req *dto.ListWorkflowRequest) (*dto.WorkflowListResponse, error)

	// Update 更新工作流
	Update(ctx context.Context, id string, req *dto.UpdateWorkflowRequest) (*dto.WorkflowResponse, error)

	// Delete 删除工作流
	Delete(ctx context.Context, id string) error

	// Run 运行工作流
	Run(ctx context.Context, req *dto.RunWorkflowRequest) (*dto.WorkflowExecutionResponse, error)

	// RunStream 流式运行工作流
	RunStream(ctx context.Context, req *dto.RunWorkflowRequest, streamChan chan<- *dto.WorkflowStreamEvent) error

	// Pause 暂停工作流执行
	Pause(ctx context.Context, executionID string) error

	// Resume 恢复工作流执行
	Resume(ctx context.Context, executionID string) error

	// Cancel 取消工作流执行
	Cancel(ctx context.Context, executionID string) error

	// GetExecutionStatus 获取执行状态
	GetExecutionStatus(ctx context.Context, executionID string) (*dto.WorkflowExecutionStatus, error)

	// ListExecutions 列出执行历史
	ListExecutions(ctx context.Context, workflowID string, req *dto.ListExecutionsRequest) (*dto.ExecutionListResponse, error)
}

// workflowService 工作流应用服务实现
type workflowService struct {
	workflowRepo   workflow.WorkflowRepository
	workflowDomain workflow.WorkflowService
	orchestrator   orchestrator.Orchestrator
	logger         logging.Logger
	tracer         trace.Tracer
	metrics        metrics.MetricsCollector
}

// NewWorkflowService 创建工作流应用服务
func NewWorkflowService(
	workflowRepo workflow.WorkflowRepository,
	workflowDomain workflow.WorkflowService,
	orch orchestrator.Orchestrator,
	logger logging.Logger,
	tracer trace.Tracer,
	metrics metrics.MetricsCollector,
) WorkflowService {
	return &workflowService{
		workflowRepo:   workflowRepo,
		workflowDomain: workflowDomain,
		orchestrator:   orch,
		logger:         logger,
		tracer:         tracer,
		metrics:        metrics,
	}
}

// Create 创建工作流
func (s *workflowService) Create(ctx context.Context, req *dto.CreateWorkflowRequest) (*dto.WorkflowResponse, error) {
	span := s.tracer.StartSpan(ctx, "WorkflowService.Create")
	defer span.End()

	s.logger.InfoCtx(ctx, "Creating workflow", "name", req.Name)

	// 转换 DTO 为领域实体
	wf := &workflow.Workflow{
		ID:          types.NewID(),
		Name:        req.Name,
		Description: req.Description,
		Steps:       s.convertSteps(req.Steps),
		Trigger:     s.convertTrigger(req.Trigger),
		Config:      s.convertWorkflowConfig(req.Config),
		Status:      workflow.StatusDraft,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 领域验证
	if err := s.workflowDomain.ValidateWorkflow(ctx, wf); err != nil {
		s.logger.ErrorCtx(ctx, "Workflow validation failed", "error", err)
		return nil, errors.Wrap(err, errors.CodeValidationFailed, "workflow validation failed")
	}

	// 持久化
	if err := s.workflowRepo.Create(ctx, wf); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to create workflow", "error", err)
		s.metrics.IncrementCounter("workflow_create_errors", nil)
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to create workflow")
	}

	s.metrics.IncrementCounter("workflow_created", map[string]string{"name": req.Name})
	s.logger.InfoCtx(ctx, "Workflow created successfully", "id", wf.ID)

	return s.toWorkflowResponse(wf), nil
}

// GetByID 根据 ID 获取工作流
func (s *workflowService) GetByID(ctx context.Context, id string) (*dto.WorkflowResponse, error) {
	span := s.tracer.StartSpan(ctx, "WorkflowService.GetByID")
	defer span.End()

	wf, err := s.workflowRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Failed to get workflow", "id", id, "error", err)
		return nil, errors.Wrap(err, errors.CodeNotFound, "workflow not found")
	}

	return s.toWorkflowResponse(wf), nil
}

// List 列出工作流
func (s *workflowService) List(ctx context.Context, req *dto.ListWorkflowRequest) (*dto.WorkflowListResponse, error) {
	span := s.tracer.StartSpan(ctx, "WorkflowService.List")
	defer span.End()

	offset := (req.Page - 1) * req.PageSize
	workflows, total, err := s.workflowRepo.List(ctx, req.Status, req.Search, offset, req.PageSize)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Failed to list workflows", "error", err)
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to list workflows")
	}

	items := make([]*dto.WorkflowResponse, len(workflows))
	for i, wf := range workflows {
		items[i] = s.toWorkflowResponse(wf)
	}

	return &dto.WorkflowListResponse{
		Items: items,
		Pagination: dto.Pagination{
			Page:     req.Page,
			PageSize: req.PageSize,
			Total:    total,
		},
	}, nil
}

// Update 更新工作流
func (s *workflowService) Update(ctx context.Context, id string, req *dto.UpdateWorkflowRequest) (*dto.WorkflowResponse, error) {
	span := s.tracer.StartSpan(ctx, "WorkflowService.Update")
	defer span.End()

	s.logger.InfoCtx(ctx, "Updating workflow", "id", id)

	// 获取现有工作流
	wf, err := s.workflowRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "workflow not found")
	}

	// 检查是否可更新
	if wf.Status == workflow.StatusRunning {
		return nil, errors.New(errors.CodeBusinessLogicError, "cannot update running workflow")
	}

	// 更新字段
	if req.Name != nil {
		wf.Name = *req.Name
	}
	if req.Description != nil {
		wf.Description = *req.Description
	}
	if req.Steps != nil {
		wf.Steps = s.convertSteps(*req.Steps)
	}
	if req.Config != nil {
		wf.Config = s.convertWorkflowConfig(req.Config)
	}
	wf.UpdatedAt = time.Now()

	// 领域验证
	if err := s.workflowDomain.ValidateWorkflow(ctx, wf); err != nil {
		return nil, errors.Wrap(err, errors.CodeValidationFailed, "workflow validation failed")
	}

	// 持久化
	if err := s.workflowRepo.Update(ctx, wf); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to update workflow", "error", err)
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to update workflow")
	}

	s.logger.InfoCtx(ctx, "Workflow updated successfully", "id", id)
	return s.toWorkflowResponse(wf), nil
}

// Delete 删除工作流
func (s *workflowService) Delete(ctx context.Context, id string) error {
	span := s.tracer.StartSpan(ctx, "WorkflowService.Delete")
	defer span.End()

	s.logger.InfoCtx(ctx, "Deleting workflow", "id", id)

	// 获取工作流检查状态
	wf, err := s.workflowRepo.GetByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, errors.CodeNotFound, "workflow not found")
	}

	if wf.Status == workflow.StatusRunning {
		return errors.New(errors.CodeBusinessLogicError, "cannot delete running workflow")
	}

	if err := s.workflowRepo.Delete(ctx, id); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to delete workflow", "error", err)
		return errors.Wrap(err, errors.CodeDatabaseError, "failed to delete workflow")
	}

	s.metrics.IncrementCounter("workflow_deleted", nil)
	s.logger.InfoCtx(ctx, "Workflow deleted successfully", "id", id)
	return nil
}

// Run 运行工作流
func (s *workflowService) Run(ctx context.Context, req *dto.RunWorkflowRequest) (*dto.WorkflowExecutionResponse, error) {
	span := s.tracer.StartSpan(ctx, "WorkflowService.Run")
	defer span.End()

	startTime := time.Now()
	s.logger.InfoCtx(ctx, "Running workflow", "workflow_id", req.WorkflowID)

	// 获取工作流定义
	wf, err := s.workflowRepo.GetByID(ctx, req.WorkflowID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "workflow not found")
	}

	// 检查工作流状态
	if !s.workflowDomain.CanExecute(ctx, wf) {
		return nil, errors.New(errors.CodeBusinessLogicError, "workflow cannot be executed")
	}

	// 创建执行上下文
	executionID := types.NewID()
	execCtx := &orchestrator.ExecutionContext{
		ExecutionID: executionID,
		WorkflowID:  req.WorkflowID,
		Input:       req.Input,
		Variables:   req.Variables,
		StartedAt:   startTime,
	}

	// 调用编排器执行工作流
	result, err := s.orchestrator.ExecuteWorkflow(ctx, wf, execCtx)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Workflow execution failed", "execution_id", executionID, "error", err)
		s.metrics.IncrementCounter("workflow_execution_errors", map[string]string{"workflow_id": req.WorkflowID})
		return nil, errors.Wrap(err, errors.CodeExecutionError, "workflow execution failed")
	}

	duration := time.Since(startTime)
	s.metrics.RecordHistogram("workflow_execution_duration_seconds", duration.Seconds(), map[string]string{
		"workflow_id": req.WorkflowID,
		"status":      string(result.Status),
	})

	s.logger.InfoCtx(ctx, "Workflow executed successfully",
		"execution_id", executionID,
		"duration_ms", duration.Milliseconds(),
	)

	return &dto.WorkflowExecutionResponse{
		ExecutionID: executionID,
		Status:      string(result.Status),
		Output:      result.Output,
		Error:       result.Error,
		StartedAt:   result.StartedAt,
		CompletedAt: result.CompletedAt,
		Duration:    duration.Milliseconds(),
		StepResults: s.convertStepResults(result.StepResults),
	}, nil
}

// RunStream 流式运行工作流
func (s *workflowService) RunStream(ctx context.Context, req *dto.RunWorkflowRequest, streamChan chan<- *dto.WorkflowStreamEvent) error {
	span := s.tracer.StartSpan(ctx, "WorkflowService.RunStream")
	defer span.End()
	defer close(streamChan)

	s.logger.InfoCtx(ctx, "Running workflow in stream mode", "workflow_id", req.WorkflowID)

	// 获取工作流定义
	wf, err := s.workflowRepo.GetByID(ctx, req.WorkflowID)
	if err != nil {
		return errors.Wrap(err, errors.CodeNotFound, "workflow not found")
	}

	// 创建执行上下文
	executionID := types.NewID()
	execCtx := &orchestrator.ExecutionContext{
		ExecutionID: executionID,
		WorkflowID:  req.WorkflowID,
		Input:       req.Input,
		Variables:   req.Variables,
		StartedAt:   time.Now(),
	}

	// 创建编排器流式通道
	orchStreamChan := make(chan *orchestrator.StreamEvent, 100)

	// 启动协程转发事件
	go func() {
		for event := range orchStreamChan {
			streamChan <- &dto.WorkflowStreamEvent{
				Type:      event.Type,
				StepID:    event.StepID,
				StepName:  event.StepName,
				Data:      event.Data,
				Error:     event.Error,
				Timestamp: event.Timestamp,
			}
		}
	}()

	// 调用编排器流式执行
	if err := s.orchestrator.ExecuteWorkflowStream(ctx, wf, execCtx, orchStreamChan); err != nil {
		s.logger.ErrorCtx(ctx, "Workflow stream execution failed", "execution_id", executionID, "error", err)
		return errors.Wrap(err, errors.CodeExecutionError, "workflow stream execution failed")
	}

	return nil
}

// Pause 暂停工作流执行
func (s *workflowService) Pause(ctx context.Context, executionID string) error {
	span := s.tracer.StartSpan(ctx, "WorkflowService.Pause")
	defer span.End()

	s.logger.InfoCtx(ctx, "Pausing workflow execution", "execution_id", executionID)

	if err := s.orchestrator.PauseExecution(ctx, executionID); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to pause execution", "execution_id", executionID, "error", err)
		return errors.Wrap(err, errors.CodeExecutionError, "failed to pause workflow execution")
	}

	s.metrics.IncrementCounter("workflow_paused", nil)
	s.logger.InfoCtx(ctx, "Workflow execution paused", "execution_id", executionID)
	return nil
}

// Resume 恢复工作流执行
func (s *workflowService) Resume(ctx context.Context, executionID string) error {
	span := s.tracer.StartSpan(ctx, "WorkflowService.Resume")
	defer span.End()

	s.logger.InfoCtx(ctx, "Resuming workflow execution", "execution_id", executionID)

	if err := s.orchestrator.ResumeExecution(ctx, executionID); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to resume execution", "execution_id", executionID, "error", err)
		return errors.Wrap(err, errors.CodeExecutionError, "failed to resume workflow execution")
	}

	s.metrics.IncrementCounter("workflow_resumed", nil)
	s.logger.InfoCtx(ctx, "Workflow execution resumed", "execution_id", executionID)
	return nil
}

// Cancel 取消工作流执行
func (s *workflowService) Cancel(ctx context.Context, executionID string) error {
	span := s.tracer.StartSpan(ctx, "WorkflowService.Cancel")
	defer span.End()

	s.logger.InfoCtx(ctx, "Cancelling workflow execution", "execution_id", executionID)

	if err := s.orchestrator.CancelExecution(ctx, executionID); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to cancel execution", "execution_id", executionID, "error", err)
		return errors.Wrap(err, errors.CodeExecutionError, "failed to cancel workflow execution")
	}

	s.metrics.IncrementCounter("workflow_cancelled", nil)
	s.logger.InfoCtx(ctx, "Workflow execution cancelled", "execution_id", executionID)
	return nil
}

// GetExecutionStatus 获取执行状态
func (s *workflowService) GetExecutionStatus(ctx context.Context, executionID string) (*dto.WorkflowExecutionStatus, error) {
	span := s.tracer.StartSpan(ctx, "WorkflowService.GetExecutionStatus")
	defer span.End()

	status, err := s.orchestrator.GetExecutionStatus(ctx, executionID)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Failed to get execution status", "execution_id", executionID, "error", err)
		return nil, errors.Wrap(err, errors.CodeNotFound, "execution not found")
	}

	return &dto.WorkflowExecutionStatus{
		ExecutionID:    status.ExecutionID,
		WorkflowID:     status.WorkflowID,
		Status:         string(status.Status),
		CurrentStep:    status.CurrentStep,
		CompletedSteps: status.CompletedSteps,
		TotalSteps:     status.TotalSteps,
		StartedAt:      status.StartedAt,
		UpdatedAt:      status.UpdatedAt,
		Error:          status.Error,
	}, nil
}

// ListExecutions 列出执行历史
func (s *workflowService) ListExecutions(ctx context.Context, workflowID string, req *dto.ListExecutionsRequest) (*dto.ExecutionListResponse, error) {
	span := s.tracer.StartSpan(ctx, "WorkflowService.ListExecutions")
	defer span.End()

	offset := (req.Page - 1) * req.PageSize
	executions, total, err := s.orchestrator.ListExecutions(ctx, workflowID, offset, req.PageSize)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Failed to list executions", "workflow_id", workflowID, "error", err)
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to list executions")
	}

	items := make([]*dto.WorkflowExecutionStatus, len(executions))
	for i, exec := range executions {
		items[i] = &dto.WorkflowExecutionStatus{
			ExecutionID:    exec.ExecutionID,
			WorkflowID:     exec.WorkflowID,
			Status:         string(exec.Status),
			CurrentStep:    exec.CurrentStep,
			CompletedSteps: exec.CompletedSteps,
			TotalSteps:     exec.TotalSteps,
			StartedAt:      exec.StartedAt,
			UpdatedAt:      exec.UpdatedAt,
			Error:          exec.Error,
		}
	}

	return &dto.ExecutionListResponse{
		Items: items,
		Pagination: dto.Pagination{
			Page:     req.Page,
			PageSize: req.PageSize,
			Total:    total,
		},
	}, nil
}

// 辅助方法：转换步骤
func (s *workflowService) convertSteps(dtoSteps []dto.WorkflowStep) []*workflow.WorkflowStep {
	steps := make([]*workflow.WorkflowStep, len(dtoSteps))
	for i, step := range dtoSteps {
		steps[i] = &workflow.WorkflowStep{
			ID:          step.ID,
			Name:        step.Name,
			Type:        workflow.StepType(step.Type),
			AgentID:     step.AgentID,
			Config:      step.Config,
			Condition:   step.Condition,
			RetryPolicy: s.convertRetryPolicy(step.RetryPolicy),
			Timeout:     step.Timeout,
		}
	}
	return steps
}

// 辅助方法：转换触发器
func (s *workflowService) convertTrigger(dtoTrigger *dto.WorkflowTrigger) *workflow.Trigger {
	if dtoTrigger == nil {
		return nil
	}
	return &workflow.Trigger{
		Type:     workflow.TriggerType(dtoTrigger.Type),
		Schedule: dtoTrigger.Schedule,
		Event:    dtoTrigger.Event,
		Webhook:  dtoTrigger.Webhook,
	}
}

// 辅助方法：转换工作流配置
func (s *workflowService) convertWorkflowConfig(dtoConfig *dto.WorkflowConfig) *workflow.Config {
	if dtoConfig == nil {
		return &workflow.Config{}
	}
	return &workflow.Config{
		MaxRetries:      dtoConfig.MaxRetries,
		Timeout:         dtoConfig.Timeout,
		ConcurrentSteps: dtoConfig.ConcurrentSteps,
		ErrorHandling:   workflow.ErrorHandling(dtoConfig.ErrorHandling),
	}
}

// 辅助方法：转换重试策略
func (s *workflowService) convertRetryPolicy(dtoPolicy *dto.RetryPolicy) *workflow.RetryPolicy {
	if dtoPolicy == nil {
		return nil
	}
	return &workflow.RetryPolicy{
		MaxRetries:   dtoPolicy.MaxRetries,
		BackoffType:  workflow.BackoffType(dtoPolicy.BackoffType),
		InitialDelay: dtoPolicy.InitialDelay,
		MaxDelay:     dtoPolicy.MaxDelay,
	}
}

// 辅助方法：转换步骤结果
func (s *workflowService) convertStepResults(results []*orchestrator.StepResult) []dto.StepResult {
	dtoResults := make([]dto.StepResult, len(results))
	for i, result := range results {
		dtoResults[i] = dto.StepResult{
			StepID:      result.StepID,
			StepName:    result.StepName,
			Status:      string(result.Status),
			Output:      result.Output,
			Error:       result.Error,
			StartedAt:   result.StartedAt,
			CompletedAt: result.CompletedAt,
			Duration:    result.Duration.Milliseconds(),
			RetryCount:  result.RetryCount,
		}
	}
	return dtoResults
}

// 辅助方法：转换为响应 DTO
func (s *workflowService) toWorkflowResponse(wf *workflow.Workflow) *dto.WorkflowResponse {
	return &dto.WorkflowResponse{
		ID:          wf.ID,
		Name:        wf.Name,
		Description: wf.Description,
		Steps:       s.convertStepsToDTO(wf.Steps),
		Trigger:     s.convertTriggerToDTO(wf.Trigger),
		Config:      s.convertConfigToDTO(wf.Config),
		Status:      string(wf.Status),
		CreatedAt:   wf.CreatedAt,
		UpdatedAt:   wf.UpdatedAt,
	}
}

// 辅助方法：转换步骤为 DTO
func (s *workflowService) convertStepsToDTO(steps []*workflow.WorkflowStep) []dto.WorkflowStep {
	dtoSteps := make([]dto.WorkflowStep, len(steps))
	for i, step := range steps {
		dtoSteps[i] = dto.WorkflowStep{
			ID:          step.ID,
			Name:        step.Name,
			Type:        string(step.Type),
			AgentID:     step.AgentID,
			Config:      step.Config,
			Condition:   step.Condition,
			RetryPolicy: s.convertRetryPolicyToDTO(step.RetryPolicy),
			Timeout:     step.Timeout,
		}
	}
	return dtoSteps
}

// 辅助方法：转换触发器为 DTO
func (s *workflowService) convertTriggerToDTO(trigger *workflow.Trigger) *dto.WorkflowTrigger {
	if trigger == nil {
		return nil
	}
	return &dto.WorkflowTrigger{
		Type:     string(trigger.Type),
		Schedule: trigger.Schedule,
		Event:    trigger.Event,
		Webhook:  trigger.Webhook,
	}
}

// 辅助方法：转换配置为 DTO
func (s *workflowService) convertConfigToDTO(config *workflow.Config) *dto.WorkflowConfig {
	if config == nil {
		return nil
	}
	return &dto.WorkflowConfig{
		MaxRetries:      config.MaxRetries,
		Timeout:         config.Timeout,
		ConcurrentSteps: config.ConcurrentSteps,
		ErrorHandling:   string(config.ErrorHandling),
	}
}

// 辅助方法：转换重试策略为 DTO
func (s *workflowService) convertRetryPolicyToDTO(policy *workflow.RetryPolicy) *dto.RetryPolicy {
	if policy == nil {
		return nil
	}
	return &dto.RetryPolicy{
		MaxRetries:   policy.MaxRetries,
		BackoffType:  string(policy.BackoffType),
		InitialDelay: policy.InitialDelay,
		MaxDelay:     policy.MaxDelay,
	}
}

//Personal.AI order the ending
