// Package workflow provides domain services for workflow business logic.
// It implements complex orchestration, execution management, state transitions,
// step scheduling, dependency resolution, and error handling.
package workflow

import (
	"context"
	"fmt"
	"time"
)

// ============================================================================
// Domain Service Interface
// ============================================================================

// WorkflowService defines the domain service interface for workflow operations
type WorkflowService interface {
	// CreateWorkflow creates a new workflow with validation
	CreateWorkflow(ctx context.Context, req CreateWorkflowRequest) (*Workflow, error)

	// UpdateWorkflow updates an existing workflow
	UpdateWorkflow(ctx context.Context, id string, req UpdateWorkflowRequest) (*Workflow, error)

	// DeleteWorkflow deletes a workflow
	DeleteWorkflow(ctx context.Context, id string, hard bool) error

	// GetWorkflow retrieves a workflow by ID
	GetWorkflow(ctx context.Context, id string) (*Workflow, error)

	// ListWorkflows lists workflows with filtering
	ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]*Workflow, error)

	// ActivateWorkflow activates a workflow
	ActivateWorkflow(ctx context.Context, id string) (*Workflow, error)

	// DeactivateWorkflow deactivates a workflow
	DeactivateWorkflow(ctx context.Context, id string) (*Workflow, error)

	// ExecuteWorkflow executes a workflow
	ExecuteWorkflow(ctx context.Context, id string, input map[string]interface{}) (*ExecutionResult, error)

	// PauseExecution pauses a running workflow execution
	PauseExecution(ctx context.Context, executionID string) error

	// ResumeExecution resumes a paused workflow execution
	ResumeExecution(ctx context.Context, executionID string) error

	// CancelExecution cancels a running workflow execution
	CancelExecution(ctx context.Context, executionID string) error

	// ValidateWorkflow validates workflow configuration
	ValidateWorkflow(ctx context.Context, workflow *Workflow) error

	// CloneWorkflow creates a copy of a workflow
	CloneWorkflow(ctx context.Context, id string, newName string) (*Workflow, error)

	// AddStep adds a step to a workflow
	AddStep(ctx context.Context, workflowID string, step *WorkflowStep) error

	// RemoveStep removes a step from a workflow
	RemoveStep(ctx context.Context, workflowID string, stepID string) error

	// UpdateStep updates a workflow step
	UpdateStep(ctx context.Context, workflowID string, step *WorkflowStep) error

	// ReorderSteps reorders workflow steps
	ReorderSteps(ctx context.Context, workflowID string, stepIDs []string) error

	// GetExecutionHistory retrieves execution history
	GetExecutionHistory(ctx context.Context, workflowID string, filter ExecutionFilter) ([]*WorkflowExecution, error)

	// GetExecutionStats retrieves execution statistics
	GetExecutionStats(ctx context.Context, workflowID string) (*WorkflowStats, error)

	// ScheduleWorkflow schedules a workflow for execution
	ScheduleWorkflow(ctx context.Context, id string, schedule ScheduleConfig) error

	// TriggerScheduledWorkflows triggers workflows due for scheduled execution
	TriggerScheduledWorkflows(ctx context.Context) error
}

// ============================================================================
// Domain Service Implementation
// ============================================================================

// workflowService implements the WorkflowService interface
type workflowService struct {
	repo              WorkflowRepository
	executionRepo     ExecutionHistoryRepository
	validator         WorkflowValidator
	orchestrator      WorkflowOrchestrator
	stepScheduler     StepScheduler
	dependencyChecker DependencyChecker
	eventEmitter      EventEmitter
}

// NewWorkflowService creates a new workflow domain service
func NewWorkflowService(
	repo WorkflowRepository,
	executionRepo ExecutionHistoryRepository,
	validator WorkflowValidator,
	orchestrator WorkflowOrchestrator,
	stepScheduler StepScheduler,
	dependencyChecker DependencyChecker,
	eventEmitter EventEmitter,
) WorkflowService {
	return &workflowService{
		repo:              repo,
		executionRepo:     executionRepo,
		validator:         validator,
		orchestrator:      orchestrator,
		stepScheduler:     stepScheduler,
		dependencyChecker: dependencyChecker,
		eventEmitter:      eventEmitter,
	}
}

// ============================================================================
// Request/Response Types
// ============================================================================

// CreateWorkflowRequest represents workflow creation request
type CreateWorkflowRequest struct {
	Name            string
	Description     string
	Steps           []*WorkflowStep
	Trigger         TriggerConfig
	OwnerID         string
	Tags            []string
	Metadata        map[string]interface{}
	ExecutionConfig ExecutionConfig
	ErrorHandling   ErrorHandlingConfig
	Notifications   NotificationConfig
}

// UpdateWorkflowRequest represents workflow update request
type UpdateWorkflowRequest struct {
	Name            *string
	Description     *string
	Steps           []*WorkflowStep
	Trigger         *TriggerConfig
	Tags            []string
	Metadata        map[string]interface{}
	ExecutionConfig *ExecutionConfig
	ErrorHandling   *ErrorHandlingConfig
	Notifications   *NotificationConfig
}

// ExecutionResult represents workflow execution result
type ExecutionResult struct {
	ExecutionID   string
	WorkflowID    string
	Status        string
	StartedAt     time.Time
	CompletedAt   *time.Time
	Duration      time.Duration
	StepResults   []*StepResult
	Output        map[string]interface{}
	Error         string
	Metadata      map[string]interface{}
}

// StepResult represents step execution result
type StepResult struct {
	StepID      string
	StepName    string
	Status      string
	StartedAt   time.Time
	CompletedAt *time.Time
	Duration    time.Duration
	Output      map[string]interface{}
	Error       string
	RetryCount  int
}

// ============================================================================
// Create Workflow
// ============================================================================

// CreateWorkflow creates a new workflow with full validation
func (s *workflowService) CreateWorkflow(ctx context.Context, req CreateWorkflowRequest) (*Workflow, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid create request: %w", err)
	}

	// Check if workflow with same name exists
	exists, err := s.repo.ExistsByName(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check workflow existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("workflow with name '%s' already exists", req.Name)
	}

	// Create workflow entity
	workflow := NewWorkflow(req.Name, req.Description, req.OwnerID)
	workflow.Steps = req.Steps
	workflow.Trigger = req.Trigger
	workflow.Tags = req.Tags
	workflow.Metadata = req.Metadata
	workflow.ExecutionConfig = req.ExecutionConfig
	workflow.ErrorHandling = req.ErrorHandling
	workflow.Notifications = req.Notifications

	// Validate workflow
	if err := s.validator.Validate(ctx, workflow); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	// Validate dependencies
	if err := s.dependencyChecker.ValidateDependencies(ctx, workflow); err != nil {
		return nil, fmt.Errorf("dependency validation failed: %w", err)
	}

	// Validate step configurations
	if err := s.validateStepConfigurations(workflow); err != nil {
		return nil, fmt.Errorf("step configuration validation failed: %w", err)
	}

	// Persist workflow
	if err := s.repo.Create(ctx, workflow); err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	// Emit workflow created event
	if s.eventEmitter != nil {
		s.eventEmitter.EmitWorkflowCreated(ctx, workflow)
	}

	return workflow, nil
}

// validateCreateRequest validates create request
func (s *workflowService) validateCreateRequest(req CreateWorkflowRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}

	if len(req.Name) < 3 || len(req.Name) > 100 {
		return fmt.Errorf("name must be between 3 and 100 characters")
	}

	if req.OwnerID == "" {
		return fmt.Errorf("owner ID is required")
	}

	if len(req.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	return nil
}

// ============================================================================
// Update Workflow
// ============================================================================

// UpdateWorkflow updates an existing workflow
func (s *workflowService) UpdateWorkflow(ctx context.Context, id string, req UpdateWorkflowRequest) (*Workflow, error) {
	// Retrieve existing workflow
	workflow, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	if workflow.IsDeleted() {
		return nil, fmt.Errorf("cannot update deleted workflow")
	}

	if workflow.IsRunning() {
		return nil, fmt.Errorf("cannot update running workflow")
	}

	// Apply updates
	if req.Name != nil {
		workflow.Name = *req.Name
	}

	if req.Description != nil {
		workflow.Description = *req.Description
	}

	if req.Steps != nil {
		workflow.Steps = req.Steps
	}

	if req.Trigger != nil {
		workflow.Trigger = *req.Trigger
	}

	if req.Tags != nil {
		workflow.Tags = req.Tags
	}

	if req.Metadata != nil {
		for k, v := range req.Metadata {
			workflow.Metadata[k] = v
		}
	}

	if req.ExecutionConfig != nil {
		workflow.ExecutionConfig = *req.ExecutionConfig
	}

	if req.ErrorHandling != nil {
		workflow.ErrorHandling = *req.ErrorHandling
	}

	if req.Notifications != nil {
		workflow.Notifications = *req.Notifications
	}

	workflow.UpdatedAt = time.Now()

	// Validate updated workflow
	if err := s.validator.Validate(ctx, workflow); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Update in repository
	if err := s.repo.Update(ctx, workflow); err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	// Emit workflow updated event
	if s.eventEmitter != nil {
		s.eventEmitter.EmitWorkflowUpdated(ctx, workflow)
	}

	return workflow, nil
}

// ============================================================================
// Delete Workflow
// ============================================================================

// DeleteWorkflow deletes a workflow
func (s *workflowService) DeleteWorkflow(ctx context.Context, id string, hard bool) error {
	// Retrieve workflow
	workflow, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	// Check if workflow is currently running
	if workflow.IsRunning() {
		return fmt.Errorf("cannot delete running workflow")
	}

	// Check dependencies
	if err := s.dependencyChecker.CheckDependenciesForDeletion(ctx, workflow); err != nil {
		return fmt.Errorf("cannot delete workflow due to dependencies: %w", err)
	}

	if hard {
		if err := s.repo.Delete(ctx, id); err != nil {
			return fmt.Errorf("failed to delete workflow: %w", err)
		}
	} else {
		if err := s.repo.SoftDelete(ctx, id); err != nil {
			return fmt.Errorf("failed to soft delete workflow: %w", err)
		}
	}

	// Emit workflow deleted event
	if s.eventEmitter != nil {
		s.eventEmitter.EmitWorkflowDeleted(ctx, id)
	}

	return nil
}

// ============================================================================
// Get Workflow
// ============================================================================

// GetWorkflow retrieves a workflow by ID
func (s *workflowService) GetWorkflow(ctx context.Context, id string) (*Workflow, error) {
	workflow, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	if workflow.IsDeleted() {
		return nil, fmt.Errorf("workflow not found")
	}

	return workflow, nil
}

// ============================================================================
// List Workflows
// ============================================================================

// ListWorkflows lists workflows with filtering
func (s *workflowService) ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]*Workflow, error) {
	workflows, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	return workflows, nil
}

// ============================================================================
// State Transitions
// ============================================================================

// ActivateWorkflow activates a workflow
func (s *workflowService) ActivateWorkflow(ctx context.Context, id string) (*Workflow, error) {
	workflow, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// Validate workflow before activation
	if err := s.validator.Validate(ctx, workflow); err != nil {
		return nil, fmt.Errorf("cannot activate invalid workflow: %w", err)
	}

	// Validate dependencies
	if err := s.dependencyChecker.ValidateDependencies(ctx, workflow); err != nil {
		return nil, fmt.Errorf("cannot activate workflow with invalid dependencies: %w", err)
	}

	// Perform state transition
	if err := workflow.Activate(); err != nil {
		return nil, fmt.Errorf("failed to activate workflow: %w", err)
	}

	// Update status in repository
	if err := s.repo.UpdateStatus(ctx, id, WorkflowStatusActive); err != nil {
		return nil, fmt.Errorf("failed to update workflow status: %w", err)
	}

	// Emit status changed event
	if s.eventEmitter != nil {
		s.eventEmitter.EmitWorkflowStatusChanged(ctx, id, WorkflowStatusInactive, WorkflowStatusActive)
	}

	return workflow, nil
}

// DeactivateWorkflow deactivates a workflow
func (s *workflowService) DeactivateWorkflow(ctx context.Context, id string) (*Workflow, error) {
	workflow, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// Check if workflow is running
	if workflow.IsRunning() {
		return nil, fmt.Errorf("cannot deactivate running workflow")
	}

	// Perform state transition
	if err := workflow.Deactivate(); err != nil {
		return nil, fmt.Errorf("failed to deactivate workflow: %w", err)
	}

	// Update status
	if err := s.repo.UpdateStatus(ctx, id, WorkflowStatusInactive); err != nil {
		return nil, fmt.Errorf("failed to update workflow status: %w", err)
	}

	// Emit status changed event
	if s.eventEmitter != nil {
		s.eventEmitter.EmitWorkflowStatusChanged(ctx, id, WorkflowStatusActive, WorkflowStatusInactive)
	}

	return workflow, nil
}

// ============================================================================
// Workflow Execution
// ============================================================================

// ExecuteWorkflow executes a workflow
func (s *workflowService) ExecuteWorkflow(ctx context.Context, id string, input map[string]interface{}) (*ExecutionResult, error) {
	// Retrieve workflow
	workflow, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// Check if workflow can be executed
	if !workflow.CanExecute() {
		return nil, fmt.Errorf("workflow cannot be executed in current state")
	}

	// Start workflow execution
	if err := workflow.Start(); err != nil {
		return nil, fmt.Errorf("failed to start workflow: %w", err)
	}

	// Update workflow status
	if err := s.repo.UpdateStatus(ctx, id, WorkflowStatusRunning); err != nil {
		return nil, fmt.Errorf("failed to update workflow status: %w", err)
	}

	// Emit execution started event
	if s.eventEmitter != nil {
		s.eventEmitter.EmitWorkflowExecutionStarted(ctx, id)
	}

	// Create execution record
	execution := &WorkflowExecution{
		ID:         generateExecutionID(),
		WorkflowID: id,
		Status:     "running",
		StartedAt:  time.Now(),
		Input:      input,
		Metadata:   make(map[string]interface{}),
	}

	// Save execution record
	if err := s.executionRepo.SaveExecution(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to save execution: %w", err)
	}

	// Orchestrate workflow execution
	result, err := s.orchestrator.Execute(ctx, workflow, input)
	if err != nil {
		// Mark workflow as failed
		duration := time.Since(execution.StartedAt)
		workflow.Fail(duration)
		s.repo.UpdateStatus(ctx, id, WorkflowStatusFailed)
		s.repo.UpdateStats(ctx, id, workflow.Stats)

		// Update execution record
		now := time.Now()
		execution.Status = "failed"
		execution.CompletedAt = &now
		execution.Duration = duration.Milliseconds()
		execution.Error = err.Error()
		s.executionRepo.SaveExecution(ctx, execution)

		// Emit execution failed event
		if s.eventEmitter != nil {
			s.eventEmitter.EmitWorkflowExecutionCompleted(ctx, id, false)
		}

		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	// Mark workflow as completed
	duration := time.Since(execution.StartedAt)
	workflow.Complete(duration)
	s.repo.UpdateStatus(ctx, id, WorkflowStatusCompleted)
	s.repo.UpdateStats(ctx, id, workflow.Stats)

	// Update execution record
	now := time.Now()
	execution.Status = "completed"
	execution.CompletedAt = &now
	execution.Duration = duration.Milliseconds()
	execution.Output = result.Output
	s.executionRepo.SaveExecution(ctx, execution)

	// Emit execution completed event
	if s.eventEmitter != nil {
		s.eventEmitter.EmitWorkflowExecutionCompleted(ctx, id, true)
	}

	return result, nil
}

// ============================================================================
// Execution Control
// ============================================================================

// PauseExecution pauses a running workflow execution
func (s *workflowService) PauseExecution(ctx context.Context, executionID string) error {
	execution, err := s.executionRepo.GetExecution(ctx, executionID)
	if err != nil {
		return fmt.Errorf("failed to get execution: %w", err)
	}

	workflow, err := s.repo.GetByID(ctx, execution.WorkflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	if err := workflow.Pause(); err != nil {
		return fmt.Errorf("failed to pause workflow: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, workflow.ID, WorkflowStatusPaused); err != nil {
		return fmt.Errorf("failed to update workflow status: %w", err)
	}

	return nil
}

// ResumeExecution resumes a paused workflow execution
func (s *workflowService) ResumeExecution(ctx context.Context, executionID string) error {
	execution, err := s.executionRepo.GetExecution(ctx, executionID)
	if err != nil {
		return fmt.Errorf("failed to get execution: %w", err)
	}

	workflow, err := s.repo.GetByID(ctx, execution.WorkflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	if err := workflow.Resume(); err != nil {
		return fmt.Errorf("failed to resume workflow: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, workflow.ID, WorkflowStatusRunning); err != nil {
		return fmt.Errorf("failed to update workflow status: %w", err)
	}

	return nil
}

// CancelExecution cancels a running workflow execution
func (s *workflowService) CancelExecution(ctx context.Context, executionID string) error {
	execution, err := s.executionRepo.GetExecution(ctx, executionID)
	if err != nil {
		return fmt.Errorf("failed to get execution: %w", err)
	}

	workflow, err := s.repo.GetByID(ctx, execution.WorkflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	if err := workflow.Deactivate(); err != nil {
		return fmt.Errorf("failed to cancel workflow: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, workflow.ID, WorkflowStatusInactive); err != nil {
		return fmt.Errorf("failed to update workflow status: %w", err)
	}

	return nil
}

// ============================================================================
// Validation
// ============================================================================

// ValidateWorkflow validates workflow configuration
func (s *workflowService) ValidateWorkflow(ctx context.Context, workflow *Workflow) error {
	return s.validator.Validate(ctx, workflow)
}

// validateStepConfigurations validates all step configurations
func (s *workflowService) validateStepConfigurations(workflow *Workflow) error {
	for _, step := range workflow.Steps {
		if err := s.validateStepConfiguration(step); err != nil {
			return fmt.Errorf("invalid step %s: %w", step.ID, err)
		}
	}
	return nil
}

// validateStepConfiguration validates a single step configuration
func (s *workflowService) validateStepConfiguration(step *WorkflowStep) error {
	if step.Type == StepTypeAgent && step.AgentID == "" {
		return fmt.Errorf("agent ID is required for agent steps")
	}

	if step.Type == StepTypeFunction && step.Config.FunctionName == "" {
		return fmt.Errorf("function name is required for function steps")
	}

	if step.Type == StepTypeWebhook && step.Config.WebhookURL == "" {
		return fmt.Errorf("webhook URL is required for webhook steps")
	}

	return nil
}

// ============================================================================
// Clone Workflow
// ============================================================================

// CloneWorkflow creates a copy of a workflow
func (s *workflowService) CloneWorkflow(ctx context.Context, id string, newName string) (*Workflow, error) {
	// Get original workflow
	original, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get original workflow: %w", err)
	}

	// Check if new name is available
	exists, err := s.repo.ExistsByName(ctx, newName)
	if err != nil {
		return nil, fmt.Errorf("failed to check name availability: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("workflow with name '%s' already exists", newName)
	}

	// Clone workflow
	cloned := original.Clone()
	cloned.ID = generateWorkflowID()
	cloned.Name = newName
	cloned.Status = WorkflowStatusDraft
	cloned.Stats = WorkflowStats{}
	now := time.Now()
	cloned.CreatedAt = now
	cloned.UpdatedAt = now
	cloned.LastExecutedAt = nil

	// Create cloned workflow
	if err := s.repo.Create(ctx, cloned); err != nil {
		return nil, fmt.Errorf("failed to create cloned workflow: %w", err)
	}

	return cloned, nil
}

// ============================================================================
// Step Management
// ============================================================================

// AddStep adds a step to a workflow
func (s *workflowService) AddStep(ctx context.Context, workflowID string, step *WorkflowStep) error {
	workflow, err := s.repo.GetByID(ctx, workflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	if err := workflow.AddStep(step); err != nil {
		return fmt.Errorf("failed to add step: %w", err)
	}

	if err := s.repo.Update(ctx, workflow); err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	return nil
}

// RemoveStep removes a step from a workflow
func (s *workflowService) RemoveStep(ctx context.Context, workflowID string, stepID string) error {
	workflow, err := s.repo.GetByID(ctx, workflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	if err := workflow.RemoveStep(stepID); err != nil {
		return fmt.Errorf("failed to remove step: %w", err)
	}

	if err := s.repo.Update(ctx, workflow); err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	return nil
}

// UpdateStep updates a workflow step
func (s *workflowService) UpdateStep(ctx context.Context, workflowID string, step *WorkflowStep) error {
	workflow, err := s.repo.GetByID(ctx, workflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	if err := workflow.UpdateStep(step); err != nil {
		return fmt.Errorf("failed to update step: %w", err)
	}

	if err := s.repo.Update(ctx, workflow); err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	return nil
}

// ReorderSteps reorders workflow steps
func (s *workflowService) ReorderSteps(ctx context.Context, workflowID string, stepIDs []string) error {
	workflow, err := s.repo.GetByID(ctx, workflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	if err := workflow.ReorderSteps(stepIDs); err != nil {
		return fmt.Errorf("failed to reorder steps: %w", err)
	}

	if err := s.repo.Update(ctx, workflow); err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	return nil
}

// ============================================================================
// Execution History
// ============================================================================

// GetExecutionHistory retrieves execution history
func (s *workflowService) GetExecutionHistory(ctx context.Context, workflowID string, filter ExecutionFilter) ([]*WorkflowExecution, error) {
	executions, err := s.executionRepo.ListExecutions(ctx, workflowID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution history: %w", err)
	}

	return executions, nil
}

// GetExecutionStats retrieves execution statistics
func (s *workflowService) GetExecutionStats(ctx context.Context, workflowID string) (*WorkflowStats, error) {
	stats, err := s.executionRepo.GetExecutionStats(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution stats: %w", err)
	}

	return stats, nil
}

// ============================================================================
// Scheduling
// ============================================================================

// ScheduleWorkflow schedules a workflow for execution
func (s *workflowService) ScheduleWorkflow(ctx context.Context, id string, schedule ScheduleConfig) error {
	workflow, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	workflow.Trigger = TriggerConfig{
		Type:     TriggerTypeSchedule,
		Schedule: &schedule,
	}

	if err := s.repo.UpdateTrigger(ctx, id, workflow.Trigger); err != nil {
		return fmt.Errorf("failed to update trigger: %w", err)
	}

	return nil
}

// TriggerScheduledWorkflows triggers workflows due for scheduled execution
func (s *workflowService) TriggerScheduledWorkflows(ctx context.Context) error {
	workflows, err := s.repo.GetByDueSchedule(ctx, time.Now())
	if err != nil {
		return fmt.Errorf("failed to get scheduled workflows: %w", err)
	}

	for _, workflow := range workflows {
		go func(wf *Workflow) {
			_, err := s.ExecuteWorkflow(ctx, wf.ID, make(map[string]interface{}))
			if err != nil {
				// Log error but continue
				fmt.Printf("Failed to execute scheduled workflow %s: %v\n", wf.ID, err)
			}
		}(workflow)
	}

	return nil
}

// ============================================================================
// Supporting Interfaces
// ============================================================================

// WorkflowValidator defines validation interface
type WorkflowValidator interface {
	Validate(ctx context.Context, workflow *Workflow) error
}

// WorkflowOrchestrator defines orchestration interface
type WorkflowOrchestrator interface {
	Execute(ctx context.Context, workflow *Workflow, input map[string]interface{}) (*ExecutionResult, error)
}

// StepScheduler defines step scheduling interface
type StepScheduler interface {
	Schedule(ctx context.Context, workflow *Workflow) ([]*WorkflowStep, error)
}

// DependencyChecker defines dependency checking interface
type DependencyChecker interface {
	ValidateDependencies(ctx context.Context, workflow *Workflow) error
	CheckDependenciesForDeletion(ctx context.Context, workflow *Workflow) error
}

// ============================================================================
// Utility Functions
// ============================================================================

// generateExecutionID generates a unique execution ID
func generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}

//Personal.AI order the ending
