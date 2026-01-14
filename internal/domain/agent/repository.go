// Package agent provides repository interfaces for agent persistence.
// It defines the AgentRepository interface with CRUD operations and query methods.
// Concrete implementations are provided by the infrastructure layer.
package agent

import (
	"context"
	"time"
)

// ============================================================================
// Repository Interface
// ============================================================================

// AgentRepository defines the interface for agent persistence operations
type AgentRepository interface {
	// Create creates a new agent
	Create(ctx context.Context, agent *Agent) error

	// GetByID retrieves an agent by ID
	GetByID(ctx context.Context, id string) (*Agent, error)

	// GetByName retrieves an agent by name
	GetByName(ctx context.Context, name string) (*Agent, error)

	// Update updates an existing agent
	Update(ctx context.Context, agent *Agent) error

	// Delete deletes an agent (hard delete)
	Delete(ctx context.Context, id string) error

	// SoftDelete soft deletes an agent
	SoftDelete(ctx context.Context, id string) error

	// List retrieves agents with pagination and filters
	List(ctx context.Context, filter AgentFilter) ([]*Agent, error)

	// Count returns the total count of agents matching the filter
	Count(ctx context.Context, filter AgentFilter) (int64, error)

	// Exists checks if an agent exists
	Exists(ctx context.Context, id string) (bool, error)

	// ExistsByName checks if an agent with the given name exists
	ExistsByName(ctx context.Context, name string) (bool, error)

	// GetByOwner retrieves agents by owner ID
	GetByOwner(ctx context.Context, ownerID string, filter AgentFilter) ([]*Agent, error)

	// GetByStatus retrieves agents by status
	GetByStatus(ctx context.Context, status AgentStatus, filter AgentFilter) ([]*Agent, error)

	// GetByRuntimeType retrieves agents by runtime type
	GetByRuntimeType(ctx context.Context, runtimeType RuntimeType, filter AgentFilter) ([]*Agent, error)

	// GetByTags retrieves agents by tags
	GetByTags(ctx context.Context, tags []string, filter AgentFilter) ([]*Agent, error)

	// Search searches agents by query
	Search(ctx context.Context, query string, filter AgentFilter) ([]*Agent, error)

	// UpdateStatus updates agent status
	UpdateStatus(ctx context.Context, id string, status AgentStatus) error

	// UpdateStats updates agent execution statistics
	UpdateStats(ctx context.Context, id string, stats ExecutionStats) error

	// IncrementExecutionCount increments execution count
	IncrementExecutionCount(ctx context.Context, id string, success bool) error

	// GetActive retrieves all active agents
	GetActive(ctx context.Context) ([]*Agent, error)

	// GetRecent retrieves recently created agents
	GetRecent(ctx context.Context, limit int) ([]*Agent, error)

	// GetPopular retrieves popular agents by execution count
	GetPopular(ctx context.Context, limit int) ([]*Agent, error)

	// BatchCreate creates multiple agents
	BatchCreate(ctx context.Context, agents []*Agent) error

	// BatchUpdate updates multiple agents
	BatchUpdate(ctx context.Context, agents []*Agent) error

	// BatchDelete deletes multiple agents
	BatchDelete(ctx context.Context, ids []string) error

	// Restore restores a soft-deleted agent
	Restore(ctx context.Context, id string) error

	// Archive archives an agent
	Archive(ctx context.Context, id string) error

	// GetArchived retrieves archived agents
	GetArchived(ctx context.Context, filter AgentFilter) ([]*Agent, error)
}

// ============================================================================
// Filter Types
// ============================================================================

// AgentFilter defines filtering options for agent queries
type AgentFilter struct {
	// Pagination
	Limit  int
	Offset int

	// Sorting
	SortBy    string
	SortOrder SortOrder

	// Status filter
	Status []AgentStatus

	// Runtime type filter
	RuntimeType []RuntimeType

	// Owner filter
	OwnerID string

	// Tags filter (any/all)
	Tags    []string
	TagMode TagFilterMode

	// Capabilities filter
	Capabilities    []string
	CapabilityMode  TagFilterMode

	// Date range filters
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	UpdatedAfter  *time.Time
	UpdatedBefore *time.Time

	// Execution stats filters
	MinExecutions *int64
	MaxExecutions *int64
	MinSuccessRate *float64

	// Include deleted (soft-deleted) agents
	IncludeDeleted bool

	// Include archived agents
	IncludeArchived bool

	// Search query
	SearchQuery string

	// Model provider filter
	ModelProvider string

	// Model name filter
	ModelName string

	// Version filter
	Version string
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
	// TagFilterModeAny matches agents with any of the tags
	TagFilterModeAny TagFilterMode = "any"

	// TagFilterModeAll matches agents with all of the tags
	TagFilterModeAll TagFilterMode = "all"
)

// ============================================================================
// Query Builder
// ============================================================================

// NewAgentFilter creates a new agent filter with default values
func NewAgentFilter() AgentFilter {
	return AgentFilter{
		Limit:           50,
		Offset:          0,
		SortBy:          "created_at",
		SortOrder:       SortOrderDesc,
		Status:          make([]AgentStatus, 0),
		RuntimeType:     make([]RuntimeType, 0),
		Tags:            make([]string, 0),
		TagMode:         TagFilterModeAny,
		Capabilities:    make([]string, 0),
		CapabilityMode:  TagFilterModeAny,
		IncludeDeleted:  false,
		IncludeArchived: false,
	}
}

// WithLimit sets the limit
func (f AgentFilter) WithLimit(limit int) AgentFilter {
	f.Limit = limit
	return f
}

// WithOffset sets the offset
func (f AgentFilter) WithOffset(offset int) AgentFilter {
	f.Offset = offset
	return f
}

// WithSorting sets sorting options
func (f AgentFilter) WithSorting(sortBy string, sortOrder SortOrder) AgentFilter {
	f.SortBy = sortBy
	f.SortOrder = sortOrder
	return f
}

// WithStatus adds status filter
func (f AgentFilter) WithStatus(status ...AgentStatus) AgentFilter {
	f.Status = append(f.Status, status...)
	return f
}

// WithRuntimeType adds runtime type filter
func (f AgentFilter) WithRuntimeType(runtimeType ...RuntimeType) AgentFilter {
	f.RuntimeType = append(f.RuntimeType, runtimeType...)
	return f
}

// WithOwner sets owner filter
func (f AgentFilter) WithOwner(ownerID string) AgentFilter {
	f.OwnerID = ownerID
	return f
}

// WithTags sets tags filter
func (f AgentFilter) WithTags(mode TagFilterMode, tags ...string) AgentFilter {
	f.Tags = tags
	f.TagMode = mode
	return f
}

// WithCapabilities sets capabilities filter
func (f AgentFilter) WithCapabilities(mode TagFilterMode, capabilities ...string) AgentFilter {
	f.Capabilities = capabilities
	f.CapabilityMode = mode
	return f
}

// WithCreatedAfter sets created after filter
func (f AgentFilter) WithCreatedAfter(t time.Time) AgentFilter {
	f.CreatedAfter = &t
	return f
}

// WithCreatedBefore sets created before filter
func (f AgentFilter) WithCreatedBefore(t time.Time) AgentFilter {
	f.CreatedBefore = &t
	return f
}

// WithUpdatedAfter sets updated after filter
func (f AgentFilter) WithUpdatedAfter(t time.Time) AgentFilter {
	f.UpdatedAfter = &t
	return f
}

// WithUpdatedBefore sets updated before filter
func (f AgentFilter) WithUpdatedBefore(t time.Time) AgentFilter {
	f.UpdatedBefore = &t
	return f
}

// WithMinExecutions sets minimum executions filter
func (f AgentFilter) WithMinExecutions(count int64) AgentFilter {
	f.MinExecutions = &count
	return f
}

// WithMaxExecutions sets maximum executions filter
func (f AgentFilter) WithMaxExecutions(count int64) AgentFilter {
	f.MaxExecutions = &count
	return f
}

// WithMinSuccessRate sets minimum success rate filter
func (f AgentFilter) WithMinSuccessRate(rate float64) AgentFilter {
	f.MinSuccessRate = &rate
	return f
}

// WithIncludeDeleted includes soft-deleted agents
func (f AgentFilter) WithIncludeDeleted(include bool) AgentFilter {
	f.IncludeDeleted = include
	return f
}

// WithIncludeArchived includes archived agents
func (f AgentFilter) WithIncludeArchived(include bool) AgentFilter {
	f.IncludeArchived = include
	return f
}

// WithSearchQuery sets search query
func (f AgentFilter) WithSearchQuery(query string) AgentFilter {
	f.SearchQuery = query
	return f
}

// WithModelProvider sets model provider filter
func (f AgentFilter) WithModelProvider(provider string) AgentFilter {
	f.ModelProvider = provider
	return f
}

// WithModelName sets model name filter
func (f AgentFilter) WithModelName(name string) AgentFilter {
	f.ModelName = name
	return f
}

// WithVersion sets version filter
func (f AgentFilter) WithVersion(version string) AgentFilter {
	f.Version = version
	return f
}

// ============================================================================
// Pagination Types
// ============================================================================

// PaginatedResult represents paginated query result
type PaginatedResult struct {
	// Agents in current page
	Agents []*Agent

	// Total count of agents matching filter
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
	// Agent IDs to update
	IDs []string

	// Fields to update
	Status      *AgentStatus
	Tags        []string
	Metadata    map[string]interface{}
	AddTags     []string
	RemoveTags  []string
}

// BulkDeleteRequest represents a bulk delete request
type BulkDeleteRequest struct {
	// Agent IDs to delete
	IDs []string

	// Soft delete flag
	Soft bool
}

// ============================================================================
// Transaction Support
// ============================================================================

// TransactionalRepository extends AgentRepository with transaction support
type TransactionalRepository interface {
	AgentRepository

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

// CachedRepository extends AgentRepository with caching capabilities
type CachedRepository interface {
	AgentRepository

	// InvalidateCache invalidates cache for specific agent
	InvalidateCache(ctx context.Context, id string) error

	// InvalidateAllCache invalidates all agent caches
	InvalidateAllCache(ctx context.Context) error

	// RefreshCache refreshes cache for specific agent
	RefreshCache(ctx context.Context, id string) error

	// WarmupCache preloads frequently accessed agents
	WarmupCache(ctx context.Context, ids []string) error
}

// ============================================================================
// Event-Aware Repository
// ============================================================================

// EventEmitter defines event emission interface
type EventEmitter interface {
	// EmitAgentCreated emits agent created event
	EmitAgentCreated(ctx context.Context, agent *Agent) error

	// EmitAgentUpdated emits agent updated event
	EmitAgentUpdated(ctx context.Context, agent *Agent) error

	// EmitAgentDeleted emits agent deleted event
	EmitAgentDeleted(ctx context.Context, id string) error

	// EmitAgentStatusChanged emits status changed event
	EmitAgentStatusChanged(ctx context.Context, id string, oldStatus, newStatus AgentStatus) error
}

// EventAwareRepository extends AgentRepository with event emission
type EventAwareRepository interface {
	AgentRepository
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
}

// DefaultRepositoryOptions returns default repository options
func DefaultRepositoryOptions() RepositoryOptions {
	return RepositoryOptions{
		EnableCache:   true,
		CacheTTL:      300,
		EnableEvents:  true,
		PoolSize:      10,
		QueryTimeout:  30,
		EnableMetrics: true,
		EnableTracing: true,
	}
}

//Personal.AI order the ending
