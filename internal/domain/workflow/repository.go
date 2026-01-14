// Package workflow provides repository interfaces for workflow persistence.
// It defines the WorkflowRepository interface with CRUD operations,
// query methods, and execution tracking capabilities.
package workflow

import (
	"context"
	"time"
)

// ============================================================================
// Repository Interface
// ============================================================================

// WorkflowRepository defines the interface for workflow persistence operations
type WorkflowRepository interface {
	// Create creates a new workflow
	Create(ctx context.Context, workflow *Workflow) error

	// GetByID retrieves a workflow by ID
	GetByID(ctx context.Context, id string) (*Workflow, error)

	// GetByName retrieves a workflow by name
	GetByName(ctx context.Context, name string) (*Workflow, error)

	// Update updates an existing workflow
	Update(ctx context.Context, workflow *Workflow) error

	// Delete deletes a workflow (hard delete)
	Delete(ctx context.Context, id string) error

	// SoftDelete soft deletes a workflow
	SoftDelete(ctx context.Context, id string) error

	// List retrieves workflows with pagination and filters
	List(ctx context.Context, filter WorkflowFilter) ([]*Workflow, error)

	// Count returns the total count of workflows matching the filter
	Count(ctx context.Context, filter WorkflowFilter) (int64, error)

	// Exists checks if a workflow exists
	Exists(ctx context.Context, id string) (bool, error)

	// ExistsByName checks if a workflow with the given name exists
	ExistsByName(ctx context.Context, name string) (bool, error)

	// GetByOwner retrieves workflows by owner ID
	GetByOwner(ctx context.Context, ownerID string, filter WorkflowFilter) ([]*Workflow, error)

	// GetByStatus retrieves workflows by status
	GetByStatus(ctx context.Context, status WorkflowStatus, filter WorkflowFilter) ([]*Workflow, error)

	// GetByTriggerType retrieves workflows by trigger type
	GetByTriggerType(ctx context.Context, triggerType TriggerType, filter WorkflowFilter) ([]*Workflow, error)

	// GetByTags retrieves workflows by tags
	GetByTags(ctx context.Context, tags []string, filter WorkflowFilter) ([]*Workflow, error)

	// Search searches workflows by query
	Search(ctx context.Context, query string, filter WorkflowFilter) ([]*Workflow, error)

	// UpdateStatus updates workflow status
	UpdateStatus(ctx context.Context, id string, status WorkflowStatus) error

	// UpdateStats updates workflow execution statistics
	UpdateStats(ctx context.Context, id string, stats WorkflowStats) error

	// IncrementExecutionCount increments execution count
	IncrementExecutionCount(ctx context.Context, id string, success bool) error

	// GetActive retrieves all active workflows
	GetActive(ctx context.Context) ([]*Workflow, error)

	// GetScheduled retrieves workflows with schedule triggers
	GetScheduled(ctx context.Context) ([]*Workflow, error)

	// GetRecent retrieves recently created workflows
	GetRecent(ctx context.Context, limit int) ([]*Workflow, error)

	// GetPopular retrieves popular workflows by execution count
	GetPopular(ctx context.Context, limit int) ([]*Workflow, error)

	// BatchCreate creates multiple workflows
	BatchCreate(ctx context.Context, workflows []*Workflow) error

	// BatchUpdate updates multiple workflows
	BatchUpdate(ctx context.Context, workflows []*Workflow) error

	// BatchDelete deletes multiple workflows
	BatchDelete(ctx context.Context, ids []string) error

	// Restore restores a soft-deleted workflow
	Restore(ctx context.Context, id string) error

	// Archive archives a workflow
	Archive(ctx context.Context, id string) error

	// GetArchived retrieves archived workflows
	GetArchived(ctx context.Context, filter WorkflowFilter) ([]*Workflow, error)

	// AddStep adds a step to a workflow
	AddStep(ctx context.Context, workflowID string, step *WorkflowStep) error

	// RemoveStep removes a step from a workflow
	RemoveStep(ctx context.Context, workflowID string, stepID string) error

	// UpdateStep updates a workflow step
	UpdateStep(ctx context.Context, workflowID string, step *WorkflowStep) error

	// GetSteps retrieves all steps for a workflow
	GetSteps(ctx context.Context, workflowID string) ([]*WorkflowStep, error)

	// GetStep retrieves a specific step
	GetStep(ctx context.Context, workflowID string, stepID string) (*WorkflowStep, error)

	// UpdateTrigger updates workflow trigger configuration
	UpdateTrigger(ctx context.Context, id string, trigger TriggerConfig) error

	// GetByDueSchedule retrieves workflows due for scheduled execution
	GetByDueSchedule(ctx context.Context, before time.Time) ([]*Workflow, error)
}

// ============================================================================
// Filter Types
// ============================================================================

// WorkflowFilter defines filtering options for workflow queries
type WorkflowFilter struct {
	// Pagination
	Limit  int
	Offset int

	// Sorting
	SortBy    string
	SortOrder SortOrder

	// Status filter
	Status []WorkflowStatus

	// Trigger type filter
	TriggerType []TriggerType

	// Owner filter
	OwnerID string

	// Tags filter (any/all)
	Tags    []string
	TagMode TagFilterMode

	// Date range filters
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	UpdatedAfter  *time.Time
	UpdatedBefore *time.Time

	// Execution stats filters
	MinExecutions  *int64
	MaxExecutions  *int64
	MinSuccessRate *float64

	// Include deleted (soft-deleted) workflows
	IncludeDeleted bool

	// Include archived workflows
	IncludeArchived bool

	// Search query
	SearchQuery string

	// Version filter
	Version string

	// Has schedule
	HasSchedule *bool

	// Has webhook
	HasWebhook *bool

	// Step count filters
	MinSteps *int
	MaxSteps *int
}

// SortOrder defines sort direction
type SortOrder string

const (
	// SortOrderAsc for ascending order
	SortOrderAsc SortOrder = "asc"

	// SortOrderDesc for descending order
	SortOrderDesc SortOrder = "desc"
)

// TagFilterMode defines how to filter by tags
type TagFilterMode string

const (
	// TagFilterModeAny matches workflows with any of the tags
	TagFilterModeAny TagFilterMode = "any"

	// TagFilterModeAll matches workflows with all of the tags
	TagFilterModeAll TagFilterMode = "all"
)

// ============================================================================
// Query Builder
// ============================================================================

// NewWorkflowFilter creates a new workflow filter with default values
func NewWorkflowFilter() WorkflowFilter {
	return WorkflowFilter{
		Limit:           50,
		Offset:          0,
		SortBy:          "created_at",
		SortOrder:       SortOrderDesc,
		Status:          make([]WorkflowStatus, 0),
		TriggerType:     make([]TriggerType, 0),
		Tags:            make([]string, 0),
		TagMode:         TagFilterModeAny,
		IncludeDeleted:  false,
		IncludeArchived: false,
	}
}

// WithLimit sets the limit
func (f WorkflowFilter) WithLimit(limit int) WorkflowFilter {
	f.Limit = limit
	return f
}

// WithOffset sets the offset
func (f WorkflowFilter) WithOffset(offset int) WorkflowFilter {
	f.Offset = offset
	return f
}

// WithSorting sets sorting options
func (f WorkflowFilter) WithSorting(sortBy string, sortOrder SortOrder) WorkflowFilter {
	f.SortBy = sortBy
	f.SortOrder = sortOrder
	return f
}

// WithStatus adds status filter
func (f WorkflowFilter) WithStatus(status ...WorkflowStatus) WorkflowFilter {
	f.Status = append(f.Status, status...)
	return f
}

// WithTriggerType adds trigger type filter
func (f WorkflowFilter) WithTriggerType(triggerType ...TriggerType) WorkflowFilter {
	f.TriggerType = append(f.TriggerType, triggerType...)
	return f
}

// WithOwner sets owner filter
func (f WorkflowFilter) WithOwner(ownerID string) WorkflowFilter {
	f.OwnerID = ownerID
	return f
}

// WithTags sets tags filter
func (f WorkflowFilter) WithTags(mode TagFilterMode, tags ...string) WorkflowFilter {
	f.Tags = tags
	f.TagMode = mode
	return f
}

// WithCreatedAfter sets created after filter
func (f WorkflowFilter) WithCreatedAfter(t time.Time) WorkflowFilter {
	f.CreatedAfter = &t
	return f
}

// WithCreatedBefore sets created before filter
func (f WorkflowFilter) WithCreatedBefore(t time.Time) WorkflowFilter {
	f.CreatedBefore = &t
	return f
}

// WithUpdatedAfter sets updated after filter
func (f WorkflowFilter) WithUpdatedAfter(t time.Time) WorkflowFilter {
	f.UpdatedAfter = &t
	return f
}

// WithUpdatedBefore sets updated before filter
func (f WorkflowFilter) WithUpdatedBefore(t time.Time) WorkflowFilter {
	f.UpdatedBefore = &t
	return f
}

// WithMinExecutions sets minimum executions filter
func (f WorkflowFilter) WithMinExecutions(count int64) WorkflowFilter {
	f.MinExecutions = &count
	return f
}

// WithMaxExecutions sets maximum executions filter
func (f WorkflowFilter) WithMaxExecutions(count int64) WorkflowFilter {
	f.MaxExecutions = &count
	return f
}

// WithMinSuccessRate sets minimum success rate filter
func (f WorkflowFilter) WithMinSuccessRate(rate float64) WorkflowFilter {
	f.MinSuccessRate = &rate
	return f
}

// WithIncludeDeleted includes soft-deleted workflows
func (f WorkflowFilter) WithIncludeDeleted(include bool) WorkflowFilter {
	f.IncludeDeleted = include
	return f
}

// WithIncludeArchived includes archived workflows
func (f WorkflowFilter) WithIncludeArchived(include bool) WorkflowFilter {
	f.IncludeArchived = include
	return f
}

// WithSearchQuery sets search query
func (f WorkflowFilter) WithSearchQuery(query string) WorkflowFilter {
	f.SearchQuery = query
	return f
}

// WithVersion sets version filter
func (f WorkflowFilter) WithVersion(version string) WorkflowFilter {
	f.Version = version
	return f
}

// WithHasSchedule sets has schedule filter
func (f WorkflowFilter) WithHasSchedule(hasSchedule bool) WorkflowFilter {
	f.HasSchedule = &hasSchedule
	return f
}

// WithHasWebhook sets has webhook filter
func (f WorkflowFilter) WithHasWebhook(hasWebhook bool) WorkflowFilter {
	f.HasWebhook = &hasWebhook
	return f
}

// WithMinSteps sets minimum steps filter
func (f WorkflowFilter) WithMinSteps(count int) WorkflowFilter {
	f.MinSteps = &count
	return f
}

// WithMaxSteps sets maximum steps filter
func (f WorkflowFilter) WithMaxSteps(count int) WorkflowFilter {
	f.MaxSteps = &count
	return f
}

// ============================================================================
// Pagination Types
// ============================================================================

// PaginatedResult represents paginated query result
type PaginatedResult struct {
	// Workflows in current page
	Workflows []*Workflow

	// Total count of workflows matching filter
	Total int64

	// Current page number (1-based)
	Page int

	// Page size
	PageSize int

	// Total pages
	TotalPages int

	// Has next page
	HasNext bool

	// Has previous page
	HasPrevious bool
}

// ============================================================================
// Bulk Operation Types
// ============================================================================

// BulkUpdateRequest represents a bulk update request
type BulkUpdateRequest struct {
	// Workflow IDs to update
	IDs []string

	// Fields to update
	Status      *WorkflowStatus
	Tags        []string
	Metadata    map[string]interface{}
	AddTags     []string
	RemoveTags  []string
}

// BulkDeleteRequest represents a bulk delete request
type BulkDeleteRequest struct {
	// Workflow IDs to delete
	IDs []string

	// Soft delete flag
	Soft bool
}

// ============================================================================
// Transaction Support
// ============================================================================

// TransactionalRepository extends WorkflowRepository with transaction support
type TransactionalRepository interface {
	WorkflowRepository

	// BeginTx begins a transaction
	BeginTx(ctx context.Context) (context.Context, error)

	// CommitTx commits the transaction
	CommitTx(ctx context.Context) error

	// RollbackTx rolls back the transaction
	RollbackTx(ctx context.Context) error

	// WithTx executes a function within a transaction
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// ============================================================================
// Cache-Aware Repository
// ============================================================================

// CachedRepository extends WorkflowRepository with caching capabilities
type CachedRepository interface {
	WorkflowRepository

	// InvalidateCache invalidates cache for specific workflow
	InvalidateCache(ctx context.Context, id string) error

	// InvalidateAllCache invalidates all workflow caches
	InvalidateAllCache(ctx context.Context) error

	// RefreshCache refreshes cache for specific workflow
	RefreshCache(ctx context.Context, id string) error

	// WarmupCache preloads frequently accessed workflows
	WarmupCache(ctx context.Context, ids []string) error
}

// ============================================================================
// Event-Aware Repository
// ============================================================================

// EventEmitter defines event emission interface
type EventEmitter interface {
	// EmitWorkflowCreated emits workflow created event
	EmitWorkflowCreated(ctx context.Context, workflow *Workflow) error

	// EmitWorkflowUpdated emits workflow updated event
	EmitWorkflowUpdated(ctx context.Context, workflow *Workflow) error

	// EmitWorkflowDeleted emits workflow deleted event
	EmitWorkflowDeleted(ctx context.Context, id string) error

	// EmitWorkflowStatusChanged emits status changed event
	EmitWorkflowStatusChanged(ctx context.Context, id string, oldStatus, newStatus WorkflowStatus) error

	// EmitWorkflowExecutionStarted emits execution started event
	EmitWorkflowExecutionStarted(ctx context.Context, id string) error

	// EmitWorkflowExecutionCompleted emits execution completed event
	EmitWorkflowExecutionCompleted(ctx context.Context, id string, success bool) error
}

// EventAwareRepository extends WorkflowRepository with event emission
type EventAwareRepository interface {
	WorkflowRepository
	EventEmitter
}

// ============================================================================
// Repository Options
// ============================================================================

// RepositoryOptions defines repository configuration options
type RepositoryOptions struct {
	// Enable caching
	EnableCache bool

	// Cache TTL in seconds
	CacheTTL int

	// Enable events
	EnableEvents bool

	// Connection pool size
	PoolSize int

	// Query timeout in seconds
	QueryTimeout int

	// Enable metrics
	EnableMetrics bool

	// Enable tracing
	EnableTracing bool

	// Enable step versioning
	EnableStepVersioning bool
}

// DefaultRepositoryOptions returns default repository options
func DefaultRepositoryOptions() RepositoryOptions {
	return RepositoryOptions{
		EnableCache:          true,
		CacheTTL:             300,
		EnableEvents:         true,
		PoolSize:             10,
		QueryTimeout:         30,
		EnableMetrics:        true,
		EnableTracing:        true,
		EnableStepVersioning: false,
	}
}

// ============================================================================
// Execution History Support
// ============================================================================

// ExecutionHistoryRepository defines execution history operations
type ExecutionHistoryRepository interface {
	// SaveExecution saves workflow execution record
	SaveExecution(ctx context.Context, execution *WorkflowExecution) error

	// GetExecution retrieves execution by ID
	GetExecution(ctx context.Context, executionID string) (*WorkflowExecution, error)

	// ListExecutions lists executions for a workflow
	ListExecutions(ctx context.Context, workflowID string, filter ExecutionFilter) ([]*WorkflowExecution, error)

	// GetLatestExecution gets the latest execution for a workflow
	GetLatestExecution(ctx context.Context, workflowID string) (*WorkflowExecution, error)

	// GetExecutionStats gets execution statistics
	GetExecutionStats(ctx context.Context, workflowID string) (*WorkflowStats, error)
}

// WorkflowExecution represents a workflow execution record
type WorkflowExecution struct {
	ID               string
	WorkflowID       string
	Status           string
	StartedAt        time.Time
	CompletedAt      *time.Time
	Duration         int64
	StepExecutions   []*StepExecution
	Input            map[string]interface{}
	Output           map[string]interface{}
	Error            string
	TriggeredBy      string
	Metadata         map[string]interface{}
}

// StepExecution represents a step execution record
type StepExecution struct {
	StepID      string
	Status      string
	StartedAt   time.Time
	CompletedAt *time.Time
	Duration    int64
	Input       map[string]interface{}
	Output      map[string]interface{}
	Error       string
	RetryCount  int
}

// ExecutionFilter defines filtering options for execution queries
type ExecutionFilter struct {
	Limit         int
	Offset        int
	Status        []string
	StartedAfter  *time.Time
	StartedBefore *time.Time
	TriggeredBy   string
}

//Personal.AI order the ending
