// Package workflow provides domain entities and business logic for workflows.
// It defines the Workflow structure with core fields and domain methods
// for step management, validation, and execution flow control.
package workflow

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Workflow Entity
// ============================================================================

// Workflow represents a workflow orchestrating multiple agents and steps
type Workflow struct {
	// Unique identifier
	ID string `json:"id"`

	// Workflow name
	Name string `json:"name"`

	// Workflow description
	Description string `json:"description"`

	// Workflow version
	Version string `json:"version"`

	// Workflow steps
	Steps []*WorkflowStep `json:"steps"`

	// Trigger configuration
	Trigger TriggerConfig `json:"trigger"`

	// Current status
	Status WorkflowStatus `json:"status"`

	// Owner user ID
	OwnerID string `json:"owner_id"`

	// Tags for categorization
	Tags []string `json:"tags"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata"`

	// Execution configuration
	ExecutionConfig ExecutionConfig `json:"execution_config"`

	// Error handling configuration
	ErrorHandling ErrorHandlingConfig `json:"error_handling"`

	// Notification settings
	Notifications NotificationConfig `json:"notifications"`

	// Execution statistics
	Stats WorkflowStats `json:"stats"`

	// Creation timestamp
	CreatedAt time.Time `json:"created_at"`

	// Last update timestamp
	UpdatedAt time.Time `json:"updated_at"`

	// Last execution timestamp
	LastExecutedAt *time.Time `json:"last_executed_at,omitempty"`

	// Deletion timestamp (soft delete)
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// ============================================================================
// Workflow Step
// ============================================================================

// WorkflowStep represents a single step in the workflow
type WorkflowStep struct {
	// Step ID
	ID string `json:"id"`

	// Step name
	Name string `json:"name"`

	// Step description
	Description string `json:"description"`

	// Step type
	Type StepType `json:"type"`

	// Agent ID (for agent steps)
	AgentID string `json:"agent_id,omitempty"`

	// Step configuration
	Config StepConfig `json:"config"`

	// Input mapping
	InputMapping map[string]string `json:"input_mapping"`

	// Output mapping
	OutputMapping map[string]string `json:"output_mapping"`

	// Dependencies (step IDs that must complete before this)
	Dependencies []string `json:"dependencies"`

	// Condition for execution
	Condition *StepCondition `json:"condition,omitempty"`

	// Retry configuration
	RetryConfig RetryConfig `json:"retry_config"`

	// Timeout in seconds
	TimeoutSeconds int `json:"timeout_seconds"`

	// Order/Position in workflow
	Order int `json:"order"`

	// Enable/disable step
	Enabled bool `json:"enabled"`
}

// ============================================================================
// Step Type
// ============================================================================

// StepType defines the type of workflow step
type StepType string

const (
	// StepTypeAgent for agent execution
	StepTypeAgent StepType = "agent"

	// StepTypeFunction for function call
	StepTypeFunction StepType = "function"

	// StepTypeCondition for conditional logic
	StepTypeCondition StepType = "condition"

	// StepTypeLoop for loop execution
	StepTypeLoop StepType = "loop"

	// StepTypeParallel for parallel execution
	StepTypeParallel StepType = "parallel"

	// StepTypeWait for delay/wait
	StepTypeWait StepType = "wait"

	// StepTypeHumanInput for human intervention
	StepTypeHumanInput StepType = "human_input"

	// StepTypeWebhook for webhook calls
	StepTypeWebhook StepType = "webhook"
)

// ============================================================================
// Step Configuration
// ============================================================================

// StepConfig defines step-specific configuration
type StepConfig struct {
	// Parameters
	Parameters map[string]interface{} `json:"parameters"`

	// Environment variables
	Environment map[string]string `json:"environment"`

	// Function name (for function steps)
	FunctionName string `json:"function_name,omitempty"`

	// Webhook URL (for webhook steps)
	WebhookURL string `json:"webhook_url,omitempty"`

	// Wait duration (for wait steps)
	WaitDuration int `json:"wait_duration,omitempty"`

	// Parallel steps (for parallel steps)
	ParallelSteps []string `json:"parallel_steps,omitempty"`

	// Loop configuration (for loop steps)
	LoopConfig *LoopConfig `json:"loop_config,omitempty"`
}

// LoopConfig defines loop configuration
type LoopConfig struct {
	// Loop type (for_each, while, until)
	Type string `json:"type"`

	// Iteration variable name
	IteratorVar string `json:"iterator_var"`

	// Collection to iterate (for for_each)
	Collection string `json:"collection,omitempty"`

	// Condition (for while/until)
	Condition string `json:"condition,omitempty"`

	// Max iterations
	MaxIterations int `json:"max_iterations"`
}

// ============================================================================
// Step Condition
// ============================================================================

// StepCondition defines conditional execution logic
type StepCondition struct {
	// Condition expression
	Expression string `json:"expression"`

	// Condition type (javascript, jsonpath, simple)
	Type string `json:"type"`

	// Variables for condition evaluation
	Variables map[string]interface{} `json:"variables"`
}

// ============================================================================
// Workflow Status
// ============================================================================

// WorkflowStatus represents the current status of a workflow
type WorkflowStatus string

const (
	// WorkflowStatusDraft indicates workflow is in draft state
	WorkflowStatusDraft WorkflowStatus = "draft"

	// WorkflowStatusActive indicates workflow is active
	WorkflowStatusActive WorkflowStatus = "active"

	// WorkflowStatusInactive indicates workflow is inactive
	WorkflowStatusInactive WorkflowStatus = "inactive"

	// WorkflowStatusRunning indicates workflow is executing
	WorkflowStatusRunning WorkflowStatus = "running"

	// WorkflowStatusPaused indicates workflow is paused
	WorkflowStatusPaused WorkflowStatus = "paused"

	// WorkflowStatusCompleted indicates workflow completed
	WorkflowStatusCompleted WorkflowStatus = "completed"

	// WorkflowStatusFailed indicates workflow failed
	WorkflowStatusFailed WorkflowStatus = "failed"

	// WorkflowStatusArchived indicates workflow is archived
	WorkflowStatusArchived WorkflowStatus = "archived"
)

// ============================================================================
// Trigger Configuration
// ============================================================================

// TriggerConfig defines workflow trigger configuration
type TriggerConfig struct {
	// Trigger type
	Type TriggerType `json:"type"`

	// Schedule (for scheduled triggers)
	Schedule *ScheduleConfig `json:"schedule,omitempty"`

	// Event configuration (for event triggers)
	Event *EventConfig `json:"event,omitempty"`

	// Webhook configuration (for webhook triggers)
	Webhook *WebhookConfig `json:"webhook,omitempty"`

	// Manual trigger settings
	Manual *ManualTriggerConfig `json:"manual,omitempty"`
}

// TriggerType defines the type of trigger
type TriggerType string

const (
	// TriggerTypeManual for manual triggering
	TriggerTypeManual TriggerType = "manual"

	// TriggerTypeSchedule for scheduled execution
	TriggerTypeSchedule TriggerType = "schedule"

	// TriggerTypeEvent for event-based triggering
	TriggerTypeEvent TriggerType = "event"

	// TriggerTypeWebhook for webhook triggering
	TriggerTypeWebhook TriggerType = "webhook"
)

// ScheduleConfig defines schedule configuration
type ScheduleConfig struct {
	// Cron expression
	CronExpression string `json:"cron_expression"`

	// Timezone
	Timezone string `json:"timezone"`

	// Start date
	StartDate *time.Time `json:"start_date,omitempty"`

	// End date
	EndDate *time.Time `json:"end_date,omitempty"`
}

// EventConfig defines event trigger configuration
type EventConfig struct {
	// Event type
	EventType string `json:"event_type"`

	// Event source
	Source string `json:"source"`

	// Event filters
	Filters map[string]interface{} `json:"filters"`
}

// WebhookConfig defines webhook trigger configuration
type WebhookConfig struct {
	// Webhook path
	Path string `json:"path"`

	// Allowed methods
	AllowedMethods []string `json:"allowed_methods"`

	// Authentication required
	RequireAuth bool `json:"require_auth"`

	// Secret for validation
	Secret string `json:"secret,omitempty"`
}

// ManualTriggerConfig defines manual trigger settings
type ManualTriggerConfig struct {
	// Require approval
	RequireApproval bool `json:"require_approval"`

	// Approvers
	Approvers []string `json:"approvers,omitempty"`
}

// ============================================================================
// Execution Configuration
// ============================================================================

// ExecutionConfig defines workflow execution settings
type ExecutionConfig struct {
	// Max concurrent steps
	MaxConcurrentSteps int `json:"max_concurrent_steps"`

	// Execution timeout in seconds
	TimeoutSeconds int `json:"timeout_seconds"`

	// Max retries for failed steps
	MaxRetries int `json:"max_retries"`

	// Continue on error
	ContinueOnError bool `json:"continue_on_error"`

	// Enable step rollback
	EnableRollback bool `json:"enable_rollback"`

	// Execution mode (sequential, parallel, hybrid)
	Mode ExecutionMode `json:"mode"`
}

// ExecutionMode defines execution mode
type ExecutionMode string

const (
	// ExecutionModeSequential for sequential execution
	ExecutionModeSequential ExecutionMode = "sequential"

	// ExecutionModeParallel for parallel execution
	ExecutionModeParallel ExecutionMode = "parallel"

	// ExecutionModeHybrid for hybrid execution
	ExecutionModeHybrid ExecutionMode = "hybrid"
)

// ============================================================================
// Error Handling Configuration
// ============================================================================

// ErrorHandlingConfig defines error handling settings
type ErrorHandlingConfig struct {
	// On error action
	OnError ErrorAction `json:"on_error"`

	// Fallback workflow ID
	FallbackWorkflowID string `json:"fallback_workflow_id,omitempty"`

	// Notify on error
	NotifyOnError bool `json:"notify_on_error"`

	// Log errors
	LogErrors bool `json:"log_errors"`

	// Retry failed steps
	RetryFailedSteps bool `json:"retry_failed_steps"`
}

// ErrorAction defines action on error
type ErrorAction string

const (
	// ErrorActionStop stops workflow on error
	ErrorActionStop ErrorAction = "stop"

	// ErrorActionContinue continues workflow on error
	ErrorActionContinue ErrorAction = "continue"

	// ErrorActionRollback rolls back on error
	ErrorActionRollback ErrorAction = "rollback"

	// ErrorActionFallback executes fallback workflow
	ErrorActionFallback ErrorAction = "fallback"
)

// ============================================================================
// Retry Configuration
// ============================================================================

// RetryConfig defines retry settings
type RetryConfig struct {
	// Enable retry
	Enabled bool `json:"enabled"`

	// Max attempts
	MaxAttempts int `json:"max_attempts"`

	// Delay between retries in seconds
	DelaySeconds int `json:"delay_seconds"`

	// Backoff multiplier
	BackoffMultiplier float64 `json:"backoff_multiplier"`

	// Max delay in seconds
	MaxDelaySeconds int `json:"max_delay_seconds"`
}

// ============================================================================
// Notification Configuration
// ============================================================================

// NotificationConfig defines notification settings
type NotificationConfig struct {
	// Enable notifications
	Enabled bool `json:"enabled"`

	// Notify on start
	NotifyOnStart bool `json:"notify_on_start"`

	// Notify on success
	NotifyOnSuccess bool `json:"notify_on_success"`

	// Notify on failure
	NotifyOnFailure bool `json:"notify_on_failure"`

	// Recipients
	Recipients []string `json:"recipients"`

	// Channels (email, slack, webhook)
	Channels []string `json:"channels"`
}

// ============================================================================
// Workflow Statistics
// ============================================================================

// WorkflowStats tracks workflow execution statistics
type WorkflowStats struct {
	// Total executions
	TotalExecutions int64 `json:"total_executions"`

	// Successful executions
	SuccessfulExecutions int64 `json:"successful_executions"`

	// Failed executions
	FailedExecutions int64 `json:"failed_executions"`

	// Average duration in milliseconds
	AvgDurationMs int64 `json:"avg_duration_ms"`

	// Last execution status
	LastExecutionStatus string `json:"last_execution_status,omitempty"`

	// Last execution duration
	LastExecutionDurationMs int64 `json:"last_execution_duration_ms,omitempty"`
}

// ============================================================================
// Entity Factory
// ============================================================================

// NewWorkflow creates a new workflow entity
func NewWorkflow(name, description string, ownerID string) *Workflow {
	now := time.Now()
	return &Workflow{
		ID:          generateWorkflowID(),
		Name:        name,
		Description: description,
		Version:     "1.0.0",
		Steps:       make([]*WorkflowStep, 0),
		Status:      WorkflowStatusDraft,
		OwnerID:     ownerID,
		Tags:        make([]string, 0),
		Metadata:    make(map[string]interface{}),
		Trigger: TriggerConfig{
			Type: TriggerTypeManual,
		},
		ExecutionConfig: defaultExecutionConfig(),
		ErrorHandling:   defaultErrorHandling(),
		Notifications:   defaultNotificationConfig(),
		Stats:           WorkflowStats{},
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// generateWorkflowID generates a unique workflow ID
func generateWorkflowID() string {
	return fmt.Sprintf("wf_%s", uuid.New().String())
}

// defaultExecutionConfig returns default execution configuration
func defaultExecutionConfig() ExecutionConfig {
	return ExecutionConfig{
		MaxConcurrentSteps: 5,
		TimeoutSeconds:     3600,
		MaxRetries:         3,
		ContinueOnError:    false,
		EnableRollback:     false,
		Mode:               ExecutionModeSequential,
	}
}

// defaultErrorHandling returns default error handling configuration
func defaultErrorHandling() ErrorHandlingConfig {
	return ErrorHandlingConfig{
		OnError:          ErrorActionStop,
		NotifyOnError:    true,
		LogErrors:        true,
		RetryFailedSteps: true,
	}
}

// defaultNotificationConfig returns default notification configuration
func defaultNotificationConfig() NotificationConfig {
	return NotificationConfig{
		Enabled:         false,
		NotifyOnStart:   false,
		NotifyOnSuccess: false,
		NotifyOnFailure: true,
		Recipients:      make([]string, 0),
		Channels:        make([]string, 0),
	}
}

// ============================================================================
// Domain Methods - Step Management
// ============================================================================

// AddStep adds a step to the workflow
func (w *Workflow) AddStep(step *WorkflowStep) error {
	if step == nil {
		return fmt.Errorf("step cannot be nil")
	}

	if err := w.validateStep(step); err != nil {
		return fmt.Errorf("invalid step: %w", err)
	}

	// Check for duplicate step ID
	if w.hasStepID(step.ID) {
		return fmt.Errorf("step with ID %s already exists", step.ID)
	}

	// Set order if not specified
	if step.Order == 0 {
		step.Order = len(w.Steps) + 1
	}

	w.Steps = append(w.Steps, step)
	w.UpdatedAt = time.Now()

	return nil
}

// RemoveStep removes a step from the workflow
func (w *Workflow) RemoveStep(stepID string) error {
	index := w.findStepIndex(stepID)
	if index == -1 {
		return fmt.Errorf("step with ID %s not found", stepID)
	}

	// Check if other steps depend on this step
	if w.hasStepDependents(stepID) {
		return fmt.Errorf("cannot remove step: other steps depend on it")
	}

	w.Steps = append(w.Steps[:index], w.Steps[index+1:]...)
	w.UpdatedAt = time.Now()

	return nil
}

// UpdateStep updates an existing step
func (w *Workflow) UpdateStep(step *WorkflowStep) error {
	if step == nil {
		return fmt.Errorf("step cannot be nil")
	}

	index := w.findStepIndex(step.ID)
	if index == -1 {
		return fmt.Errorf("step with ID %s not found", step.ID)
	}

	if err := w.validateStep(step); err != nil {
		return fmt.Errorf("invalid step: %w", err)
	}

	w.Steps[index] = step
	w.UpdatedAt = time.Now()

	return nil
}

// GetStep retrieves a step by ID
func (w *Workflow) GetStep(stepID string) (*WorkflowStep, error) {
	for _, step := range w.Steps {
		if step.ID == stepID {
			return step, nil
		}
	}
	return nil, fmt.Errorf("step with ID %s not found", stepID)
}

// ReorderSteps reorders workflow steps
func (w *Workflow) ReorderSteps(stepIDs []string) error {
	if len(stepIDs) != len(w.Steps) {
		return fmt.Errorf("step count mismatch")
	}

	newSteps := make([]*WorkflowStep, 0, len(stepIDs))
	for i, stepID := range stepIDs {
		step, err := w.GetStep(stepID)
		if err != nil {
			return err
		}
		step.Order = i + 1
		newSteps = append(newSteps, step)
	}

	w.Steps = newSteps
	w.UpdatedAt = time.Now()

	return nil
}

// ============================================================================
// Domain Methods - Validation
// ============================================================================

// Validate validates the workflow
func (w *Workflow) Validate() error {
	if w.ID == "" {
		return fmt.Errorf("workflow ID is required")
	}

	if w.Name == "" {
		return fmt.Errorf("workflow name is required")
	}

	if len(w.Name) < 3 || len(w.Name) > 100 {
		return fmt.Errorf("workflow name must be between 3 and 100 characters")
	}

	if w.OwnerID == "" {
		return fmt.Errorf("owner ID is required")
	}

	if len(w.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	// Validate each step
	for _, step := range w.Steps {
		if err := w.validateStep(step); err != nil {
			return fmt.Errorf("invalid step %s: %w", step.ID, err)
		}
	}

	// Validate dependencies
	if err := w.validateDependencies(); err != nil {
		return fmt.Errorf("invalid dependencies: %w", err)
	}

	// Validate trigger
	if err := w.validateTrigger(); err != nil {
		return fmt.Errorf("invalid trigger: %w", err)
	}

	return nil
}

// validateStep validates a single step
func (w *Workflow) validateStep(step *WorkflowStep) error {
	if step.ID == "" {
		return fmt.Errorf("step ID is required")
	}

	if step.Name == "" {
		return fmt.Errorf("step name is required")
	}

	if step.Type == "" {
		return fmt.Errorf("step type is required")
	}

	if step.Type == StepTypeAgent && step.AgentID == "" {
		return fmt.Errorf("agent ID is required for agent steps")
	}

	if step.TimeoutSeconds <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	return nil
}

// validateDependencies validates step dependencies
func (w *Workflow) validateDependencies() error {
	stepIDs := make(map[string]bool)
	for _, step := range w.Steps {
		stepIDs[step.ID] = true
	}

	for _, step := range w.Steps {
		for _, depID := range step.Dependencies {
			if !stepIDs[depID] {
				return fmt.Errorf("step %s depends on non-existent step %s", step.ID, depID)
			}
		}
	}

	// Check for circular dependencies
	if w.hasCircularDependencies() {
		return fmt.Errorf("circular dependencies detected")
	}

	return nil
}

// validateTrigger validates trigger configuration
func (w *Workflow) validateTrigger() error {
	if w.Trigger.Type == "" {
		return fmt.Errorf("trigger type is required")
	}

	if w.Trigger.Type == TriggerTypeSchedule && w.Trigger.Schedule == nil {
		return fmt.Errorf("schedule configuration is required for scheduled triggers")
	}

	if w.Trigger.Type == TriggerTypeEvent && w.Trigger.Event == nil {
		return fmt.Errorf("event configuration is required for event triggers")
	}

	return nil
}

// ============================================================================
// Domain Methods - Status
// ============================================================================

// IsActive checks if the workflow is active
func (w *Workflow) IsActive() bool {
	return w.Status == WorkflowStatusActive
}

// IsRunning checks if the workflow is currently running
func (w *Workflow) IsRunning() bool {
	return w.Status == WorkflowStatusRunning
}

// IsPaused checks if the workflow is paused
func (w *Workflow) IsPaused() bool {
	return w.Status == WorkflowStatusPaused
}

// IsCompleted checks if the workflow is completed
func (w *Workflow) IsCompleted() bool {
	return w.Status == WorkflowStatusCompleted
}

// IsFailed checks if the workflow has failed
func (w *Workflow) IsFailed() bool {
	return w.Status == WorkflowStatusFailed
}

// IsDeleted checks if the workflow is soft deleted
func (w *Workflow) IsDeleted() bool {
	return w.DeletedAt != nil
}

// CanExecute checks if the workflow can be executed
func (w *Workflow) CanExecute() bool {
	if w.IsDeleted() {
		return false
	}

	if !w.IsActive() && !w.IsPaused() {
		return false
	}

	if len(w.Steps) == 0 {
		return false
	}

	return true
}

// ============================================================================
// Domain Methods - State Transitions
// ============================================================================

// Activate activates the workflow
func (w *Workflow) Activate() error {
	if w.IsDeleted() {
		return fmt.Errorf("cannot activate deleted workflow")
	}

	if err := w.Validate(); err != nil {
		return fmt.Errorf("cannot activate invalid workflow: %w", err)
	}

	w.Status = WorkflowStatusActive
	w.UpdatedAt = time.Now()
	return nil
}

// Deactivate deactivates the workflow
func (w *Workflow) Deactivate() error {
	if w.IsRunning() {
		return fmt.Errorf("cannot deactivate running workflow")
	}

	w.Status = WorkflowStatusInactive
	w.UpdatedAt = time.Now()
	return nil
}

// Start starts workflow execution
func (w *Workflow) Start() error {
	if !w.CanExecute() {
		return fmt.Errorf("workflow cannot be executed")
	}

	w.Status = WorkflowStatusRunning
	now := time.Now()
	w.LastExecutedAt = &now
	w.UpdatedAt = now
	return nil
}

// Pause pauses workflow execution
func (w *Workflow) Pause() error {
	if !w.IsRunning() {
		return fmt.Errorf("workflow is not running")
	}

	w.Status = WorkflowStatusPaused
	w.UpdatedAt = time.Now()
	return nil
}

// Resume resumes paused workflow
func (w *Workflow) Resume() error {
	if !w.IsPaused() {
		return fmt.Errorf("workflow is not paused")
	}

	w.Status = WorkflowStatusRunning
	w.UpdatedAt = time.Now()
	return nil
}

// Complete marks workflow as completed
func (w *Workflow) Complete(duration time.Duration) {
	w.Status = WorkflowStatusCompleted
	w.Stats.TotalExecutions++
	w.Stats.SuccessfulExecutions++
	w.Stats.LastExecutionStatus = "success"
	w.Stats.LastExecutionDurationMs = duration.Milliseconds()

	// Update average duration
	if w.Stats.TotalExecutions > 0 {
		totalTime := w.Stats.AvgDurationMs * (w.Stats.TotalExecutions - 1)
		totalTime += duration.Milliseconds()
		w.Stats.AvgDurationMs = totalTime / w.Stats.TotalExecutions
	}

	w.UpdatedAt = time.Now()
}

// Fail marks workflow as failed
func (w *Workflow) Fail(duration time.Duration) {
	w.Status = WorkflowStatusFailed
	w.Stats.TotalExecutions++
	w.Stats.FailedExecutions++
	w.Stats.LastExecutionStatus = "failed"
	w.Stats.LastExecutionDurationMs = duration.Milliseconds()
	w.UpdatedAt = time.Now()
}

// Archive archives the workflow
func (w *Workflow) Archive() error {
	if w.IsRunning() {
		return fmt.Errorf("cannot archive running workflow")
	}

	w.Status = WorkflowStatusArchived
	w.UpdatedAt = time.Now()
	return nil
}

// SoftDelete soft deletes the workflow
func (w *Workflow) SoftDelete() error {
	if w.IsRunning() {
		return fmt.Errorf("cannot delete running workflow")
	}

	now := time.Now()
	w.DeletedAt = &now
	w.Status = WorkflowStatusArchived
	w.UpdatedAt = now
	return nil
}

// ============================================================================
// Domain Methods - Utilities
// ============================================================================

// hasStepID checks if a step ID exists
func (w *Workflow) hasStepID(stepID string) bool {
	for _, step := range w.Steps {
		if step.ID == stepID {
			return true
		}
	}
	return false
}

// findStepIndex finds the index of a step
func (w *Workflow) findStepIndex(stepID string) int {
	for i, step := range w.Steps {
		if step.ID == stepID {
			return i
		}
	}
	return -1
}

// hasStepDependents checks if any steps depend on the given step
func (w *Workflow) hasStepDependents(stepID string) bool {
	for _, step := range w.Steps {
		for _, dep := range step.Dependencies {
			if dep == stepID {
				return true
			}
		}
	}
	return false
}

// hasCircularDependencies checks for circular dependencies
func (w *Workflow) hasCircularDependencies() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for _, step := range w.Steps {
		if w.hasCycleDFS(step.ID, visited, recStack) {
			return true
		}
	}

	return false
}

// hasCycleDFS performs DFS to detect cycles
func (w *Workflow) hasCycleDFS(stepID string, visited, recStack map[string]bool) bool {
	visited[stepID] = true
	recStack[stepID] = true

	step, _ := w.GetStep(stepID)
	if step != nil {
		for _, dep := range step.Dependencies {
			if !visited[dep] {
				if w.hasCycleDFS(dep, visited, recStack) {
					return true
				}
			} else if recStack[dep] {
				return true
			}
		}
	}

	recStack[stepID] = false
	return false
}

// Clone creates a copy of the workflow
func (w *Workflow) Clone() *Workflow {
	data, _ := json.Marshal(w)
	var cloned Workflow
	json.Unmarshal(data, &cloned)
	return &cloned
}

// GetSuccessRate returns the success rate
func (w *Workflow) GetSuccessRate() float64 {
	if w.Stats.TotalExecutions == 0 {
		return 0.0
	}
	return float64(w.Stats.SuccessfulExecutions) / float64(w.Stats.TotalExecutions)
}

// ToJSON converts workflow to JSON
func (w *Workflow) ToJSON() ([]byte, error) {
	return json.Marshal(w)
}

//Personal.AI order the ending
