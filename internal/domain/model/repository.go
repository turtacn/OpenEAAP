// Package model provides repository interfaces for model persistence.
// It defines the ModelRepository interface with CRUD operations,
// query methods, and model management capabilities.
package model

import (
	"context"
	"time"
)

// ============================================================================
// Repository Interface
// ============================================================================

// ModelRepository defines the interface for model persistence operations
type ModelRepository interface {
	// Create registers a new model
	Create(ctx context.Context, model *Model) error

	// GetByID retrieves a model by ID
	GetByID(ctx context.Context, id string) (*Model, error)

	// GetByName retrieves a model by name
	GetByName(ctx context.Context, name string) (*Model, error)

	// GetByProviderAndVersion retrieves a model by provider and version
	GetByProviderAndVersion(ctx context.Context, provider, version string) (*Model, error)

	// Update updates an existing model
	Update(ctx context.Context, model *Model) error

	// Delete deletes a model (hard delete)
	Delete(ctx context.Context, id string) error

	// SoftDelete soft deletes a model
	SoftDelete(ctx context.Context, id string) error

	// Restore restores a soft-deleted model
	Restore(ctx context.Context, id string) error

	// List retrieves models with pagination and filters
	List(ctx context.Context, filter ModelFilter) ([]*Model, error)

	// Count returns the total count of models matching the filter
	Count(ctx context.Context, filter ModelFilter) (int64, error)

	// Exists checks if a model exists
	Exists(ctx context.Context, id string) (bool, error)

	// ExistsByName checks if a model with the given name exists
	ExistsByName(ctx context.Context, name string) (bool, error)

	// GetByProvider retrieves models by provider
	GetByProvider(ctx context.Context, provider string, filter ModelFilter) ([]*Model, error)

	// GetByType retrieves models by type
	GetByType(ctx context.Context, modelType ModelType, filter ModelFilter) ([]*Model, error)

	// GetByStatus retrieves models by status
	GetByStatus(ctx context.Context, status ModelStatus, filter ModelFilter) ([]*Model, error)

	// GetByTags retrieves models by tags
	GetByTags(ctx context.Context, tags []string, filter ModelFilter) ([]*Model, error)

	// GetAvailable retrieves all available models
	GetAvailable(ctx context.Context) ([]*Model, error)

	// GetDeprecated retrieves all deprecated models
	GetDeprecated(ctx context.Context) ([]*Model, error)

	// Search searches models by query
	Search(ctx context.Context, query string, filter ModelFilter) ([]*Model, error)

	// UpdateStatus updates model status
	UpdateStatus(ctx context.Context, id string, status ModelStatus) error

	// UpdateConfig updates model configuration
	UpdateConfig(ctx context.Context, id string, config ModelConfig) error

	// UpdatePricing updates model pricing
	UpdatePricing(ctx context.Context, id string, pricing PricingInfo) error

	// UpdateRateLimits updates model rate limits
	UpdateRateLimits(ctx context.Context, id string, limits RateLimits) error

	// UpdatePerformance updates model performance metrics
	UpdatePerformance(ctx context.Context, id string, metrics PerformanceMetrics) error

	// UpdateStats updates model statistics
	UpdateStats(ctx context.Context, id string, stats ModelStats) error

	// RecordRequest records a model request
	RecordRequest(ctx context.Context, id string, success bool, tokens int, cost float64) error

	// UpdateHealthCheck updates health check status
	UpdateHealthCheck(ctx context.Context, id string, healthy bool) error

	// GetModelsForHealthCheck retrieves models that need health check
	GetModelsForHealthCheck(ctx context.Context) ([]*Model, error)

	// BatchCreate creates multiple models
	BatchCreate(ctx context.Context, models []*Model) error

	// BatchUpdate updates multiple models
	BatchUpdate(ctx context.Context, models []*Model) error

	// BatchDelete deletes multiple models
	BatchDelete(ctx context.Context, ids []string) error

	// GetPopular retrieves popular models by usage
	GetPopular(ctx context.Context, limit int) ([]*Model, error)

	// GetRecent retrieves recently added models
	GetRecent(ctx context.Context, limit int) ([]*Model, error)

	// GetByCapability retrieves models by capability
	GetByCapability(ctx context.Context, capability string) ([]*Model, error)

	// GetByPriceRange retrieves models within price range
	GetByPriceRange(ctx context.Context, minCost, maxCost float64) ([]*Model, error)

	// AddTag adds a tag to a model
	AddTag(ctx context.Context, id string, tag string) error

	// RemoveTag removes a tag from a model
	RemoveTag(ctx context.Context, id string, tag string) error

	// GetStatistics retrieves aggregate statistics
	GetStatistics(ctx context.Context, filter ModelFilter) (*ModelStatistics, error)
}

// ============================================================================
// Filter Types
// ============================================================================

// ModelFilter defines filtering options for model queries
type ModelFilter struct {
	// Pagination
	Limit  int
	Offset int

	// Sorting
	SortBy    string
	SortOrder SortOrder

	// Status filter
	Status []ModelStatus

	// Type filter
	Type []ModelType

	// Provider filter
	Providers []string

	// Version filter
	Versions []string

	// Tags filter (any/all)
	Tags    []string
	TagMode TagFilterMode

	// Date range filters
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	UpdatedAfter  *time.Time
	UpdatedBefore *time.Time

	// Availability filters
	OnlyAvailable   bool
	OnlyDeprecated  bool
	IncludeDeleted  bool

	// Capability filters
	RequiredCapabilities []string

	// Price filters
	MinInputTokenCost  *float64
	MaxInputTokenCost  *float64
	MinOutputTokenCost *float64
	MaxOutputTokenCost *float64

	// Performance filters
	MinSuccessRate    *float64
	MaxAvgLatencyMs   *int64
	MinUptimePercent  *float64

	// Rate limit filters
	MinRequestsPerMin *int
	MinTokensPerMin   *int

	// Search query
	SearchQuery string

	// Has free tier
	HasFreeTier *bool

	// Context length filters
	MinContextLength *int
	MaxContextLength *int
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
	// TagFilterModeAny matches models with any of the tags
	TagFilterModeAny TagFilterMode = "any"

	// TagFilterModeAll matches models with all of the tags
	TagFilterModeAll TagFilterMode = "all"
)

// ============================================================================
// Query Builder
// ============================================================================

// NewModelFilter creates a new model filter with default values
func NewModelFilter() ModelFilter {
	return ModelFilter{
		Limit:                50,
		Offset:               0,
		SortBy:               "created_at",
		SortOrder:            SortOrderDesc,
		Status:               make([]ModelStatus, 0),
		Type:                 make([]ModelType, 0),
		Providers:            make([]string, 0),
		Versions:             make([]string, 0),
		Tags:                 make([]string, 0),
		TagMode:              TagFilterModeAny,
		RequiredCapabilities: make([]string, 0),
		OnlyAvailable:        false,
		OnlyDeprecated:       false,
		IncludeDeleted:       false,
	}
}

// WithLimit sets the limit
func (f ModelFilter) WithLimit(limit int) ModelFilter {
	f.Limit = limit
	return f
}

// WithOffset sets the offset
func (f ModelFilter) WithOffset(offset int) ModelFilter {
	f.Offset = offset
	return f
}

// WithSorting sets sorting options
func (f ModelFilter) WithSorting(sortBy string, sortOrder SortOrder) ModelFilter {
	f.SortBy = sortBy
	f.SortOrder = sortOrder
	return f
}

// WithStatus adds status filter
func (f ModelFilter) WithStatus(status ...ModelStatus) ModelFilter {
	f.Status = append(f.Status, status...)
	return f
}

// WithType adds type filter
func (f ModelFilter) WithType(modelType ...ModelType) ModelFilter {
	f.Type = append(f.Type, modelType...)
	return f
}

// WithProviders adds provider filter
func (f ModelFilter) WithProviders(providers ...string) ModelFilter {
	f.Providers = append(f.Providers, providers...)
	return f
}

// WithVersions adds version filter
func (f ModelFilter) WithVersions(versions ...string) ModelFilter {
	f.Versions = append(f.Versions, versions...)
	return f
}

// WithTags sets tags filter
func (f ModelFilter) WithTags(mode TagFilterMode, tags ...string) ModelFilter {
	f.Tags = tags
	f.TagMode = mode
	return f
}

// WithCreatedAfter sets created after filter
func (f ModelFilter) WithCreatedAfter(t time.Time) ModelFilter {
	f.CreatedAfter = &t
	return f
}

// WithCreatedBefore sets created before filter
func (f ModelFilter) WithCreatedBefore(t time.Time) ModelFilter {
	f.CreatedBefore = &t
	return f
}

// WithUpdatedAfter sets updated after filter
func (f ModelFilter) WithUpdatedAfter(t time.Time) ModelFilter {
	f.UpdatedAfter = &t
	return f
}

// WithUpdatedBefore sets updated before filter
func (f ModelFilter) WithUpdatedBefore(t time.Time) ModelFilter {
	f.UpdatedBefore = &t
	return f
}

// WithOnlyAvailable sets only available filter
func (f ModelFilter) WithOnlyAvailable(only bool) ModelFilter {
	f.OnlyAvailable = only
	return f
}

// WithOnlyDeprecated sets only deprecated filter
func (f ModelFilter) WithOnlyDeprecated(only bool) ModelFilter {
	f.OnlyDeprecated = only
	return f
}

// WithIncludeDeleted includes soft-deleted models
func (f ModelFilter) WithIncludeDeleted(include bool) ModelFilter {
	f.IncludeDeleted = include
	return f
}

// WithRequiredCapabilities adds required capabilities filter
func (f ModelFilter) WithRequiredCapabilities(capabilities ...string) ModelFilter {
	f.RequiredCapabilities = append(f.RequiredCapabilities, capabilities...)
	return f
}

// WithPriceRange sets price range filters
func (f ModelFilter) WithPriceRange(minInput, maxInput, minOutput, maxOutput float64) ModelFilter {
	f.MinInputTokenCost = &minInput
	f.MaxInputTokenCost = &maxInput
	f.MinOutputTokenCost = &minOutput
	f.MaxOutputTokenCost = &maxOutput
	return f
}

// WithMinSuccessRate sets minimum success rate filter
func (f ModelFilter) WithMinSuccessRate(rate float64) ModelFilter {
	f.MinSuccessRate = &rate
	return f
}

// WithMaxAvgLatency sets maximum average latency filter
func (f ModelFilter) WithMaxAvgLatency(latencyMs int64) ModelFilter {
	f.MaxAvgLatencyMs = &latencyMs
	return f
}

// WithMinUptime sets minimum uptime percentage filter
func (f ModelFilter) WithMinUptime(percentage float64) ModelFilter {
	f.MinUptimePercent = &percentage
	return f
}

// WithSearchQuery sets search query
func (f ModelFilter) WithSearchQuery(query string) ModelFilter {
	f.SearchQuery = query
	return f
}

// WithHasFreeTier sets has free tier filter
func (f ModelFilter) WithHasFreeTier(has bool) ModelFilter {
	f.HasFreeTier = &has
	return f
}

// WithContextLengthRange sets context length range filters
func (f ModelFilter) WithContextLengthRange(min, max int) ModelFilter {
	f.MinContextLength = &min
	f.MaxContextLength = &max
	return f
}

// ============================================================================
// Pagination Types
// ============================================================================

// PaginatedResult represents paginated query result
type PaginatedResult struct {
	// Models in current page
	Models []*Model

	// Total count of models matching filter
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
// Statistics Types
// ============================================================================

// ModelStatistics represents aggregate statistics
type ModelStatistics struct {
	// Total models
	TotalModels int64

	// Available models
	AvailableModels int64

	// Deprecated models
	DeprecatedModels int64

	// Models by type
	ModelsByType map[ModelType]int64

	// Models by provider
	ModelsByProvider map[string]int64

	// Models by status
	ModelsByStatus map[ModelStatus]int64

	// Total requests across all models
	TotalRequests int64

	// Total successful requests
	TotalSuccessfulRequests int64

	// Total tokens processed
	TotalTokens int64

	// Total cost
	TotalCost float64

	// Average success rate
	AverageSuccessRate float64

	// Average latency
	AverageLatencyMs int64

	// Average uptime
	AverageUptimePercent float64

	// Most popular models
	MostPopular []*ModelPopularity

	// Last updated
	LastUpdated time.Time
}

// ModelPopularity represents model popularity metrics
type ModelPopularity struct {
	ModelID      string
	ModelName    string
	RequestCount int64
	SuccessRate  float64
}

// ============================================================================
// Bulk Operation Types
// ============================================================================

// BulkUpdateRequest represents a bulk update request
type BulkUpdateRequest struct {
	// Model IDs to update
	IDs []string

	// Fields to update
	Status      *ModelStatus
	Tags        []string
	AddTags     []string
	RemoveTags  []string
	Metadata    map[string]interface{}
}

// BulkDeleteRequest represents a bulk delete request
type BulkDeleteRequest struct {
	// Model IDs to delete
	IDs []string

	// Soft delete flag
	Soft bool
}

// ============================================================================
// Transaction Support
// ============================================================================

// TransactionalRepository extends ModelRepository with transaction support
type TransactionalRepository interface {
	ModelRepository

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

// CachedRepository extends ModelRepository with caching capabilities
type CachedRepository interface {
	ModelRepository

	// InvalidateCache invalidates cache for specific model
	InvalidateCache(ctx context.Context, id string) error

	// InvalidateAllCache invalidates all model caches
	InvalidateAllCache(ctx context.Context) error

	// RefreshCache refreshes cache for specific model
	RefreshCache(ctx context.Context, id string) error

	// WarmupCache preloads frequently accessed models
	WarmupCache(ctx context.Context, ids []string) error
}

// ============================================================================
// Event-Aware Repository
// ============================================================================

// ModelEventEmitter defines event emission interface
type ModelEventEmitter interface {
	// EmitModelRegistered emits model registered event
	EmitModelRegistered(ctx context.Context, model *Model) error

	// EmitModelUpdated emits model updated event
	EmitModelUpdated(ctx context.Context, model *Model) error

	// EmitModelDeleted emits model deleted event
	EmitModelDeleted(ctx context.Context, id string) error

	// EmitModelStatusChanged emits status changed event
	EmitModelStatusChanged(ctx context.Context, id string, oldStatus, newStatus ModelStatus) error

	// EmitModelDeprecated emits model deprecated event
	EmitModelDeprecated(ctx context.Context, id string) error

	// EmitModelHealthCheckFailed emits health check failed event
	EmitModelHealthCheckFailed(ctx context.Context, id string) error
}

// EventAwareRepository extends ModelRepository with event emission
type EventAwareRepository interface {
	ModelRepository
	ModelEventEmitter
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

	// Enable auto health checks
	EnableAutoHealthCheck bool

	// Health check interval in seconds
	HealthCheckInterval int
}

// DefaultRepositoryOptions returns default repository options
func DefaultRepositoryOptions() RepositoryOptions {
	return RepositoryOptions{
		EnableCache:           true,
		CacheTTL:              300,
		EnableEvents:          true,
		PoolSize:              10,
		QueryTimeout:          30,
		EnableMetrics:         true,
		EnableTracing:         true,
		EnableAutoHealthCheck: true,
		HealthCheckInterval:   60,
	}
}

// ============================================================================
// Model Versioning Support
// ============================================================================

// ModelVersionRepository defines model version operations
type ModelVersionRepository interface {
	// CreateVersion creates a new model version
	CreateVersion(ctx context.Context, version *ModelVersion) error

	// GetVersion retrieves a specific version
	GetVersion(ctx context.Context, modelID, version string) (*ModelVersion, error)

	// ListVersions lists all versions for a model
	ListVersions(ctx context.Context, modelID string) ([]*ModelVersion, error)

	// GetLatestVersion gets the latest version
	GetLatestVersion(ctx context.Context, modelID string) (*ModelVersion, error)

	// DeleteVersion deletes a version
	DeleteVersion(ctx context.Context, modelID, version string) error

	// SetActiveVersion sets the active version
	SetActiveVersion(ctx context.Context, modelID, version string) error
}

// ModelVersion represents a model version
type ModelVersion struct {
	ModelID     string
	Version     string
	ReleaseDate time.Time
	ChangeLog   string
	Config      ModelConfig
	Pricing     PricingInfo
	IsActive    bool
	Deprecated  bool
	CreatedAt   time.Time
}

// ============================================================================
// Model Registry Support
// ============================================================================

// ModelRegistryRepository extends ModelRepository with registry capabilities
type ModelRegistryRepository interface {
	ModelRepository

	// Register registers a model in the registry
	Register(ctx context.Context, model *Model) error

	// Unregister removes a model from the registry
	Unregister(ctx context.Context, id string) error

	// GetRegistry retrieves all registered models
	GetRegistry(ctx context.Context) ([]*Model, error)

	// Discover discovers available models from providers
	Discover(ctx context.Context, provider string) ([]*Model, error)

	// Sync synchronizes model information with providers
	Sync(ctx context.Context, id string) error

	// ValidateModel validates model configuration
	ValidateModel(ctx context.Context, model *Model) error
}

// ============================================================================
// Model Monitoring Support
// ============================================================================

// ModelMonitoringRepository defines model monitoring operations
type ModelMonitoringRepository interface {
	// RecordMetric records a monitoring metric
	RecordMetric(ctx context.Context, metric *ModelMetric) error

	// GetMetrics retrieves metrics for a model
	GetMetrics(ctx context.Context, modelID string, filter MetricFilter) ([]*ModelMetric, error)

	// GetAggregatedMetrics retrieves aggregated metrics
	GetAggregatedMetrics(ctx context.Context, modelID string, interval time.Duration) (*AggregatedMetrics, error)

	// GetAlerts retrieves active alerts
	GetAlerts(ctx context.Context, modelID string) ([]*ModelAlert, error)

	// CreateAlert creates a monitoring alert
	CreateAlert(ctx context.Context, alert *ModelAlert) error

	// ResolveAlert resolves an alert
	ResolveAlert(ctx context.Context, alertID string) error
}

// ModelMetric represents a monitoring metric
type ModelMetric struct {
	ModelID    string
	MetricType string
	Value      float64
	Timestamp  time.Time
	Labels     map[string]string
}

// MetricFilter defines metric filtering options
type MetricFilter struct {
	StartTime  time.Time
	EndTime    time.Time
	MetricType string
	Limit      int
}

// AggregatedMetrics represents aggregated metrics
type AggregatedMetrics struct {
	ModelID          string
	Interval         time.Duration
	TotalRequests    int64
	SuccessfulReqs   int64
	FailedReqs       int64
	AvgLatencyMs     float64
	P95LatencyMs     float64
	P99LatencyMs     float64
	TotalTokens      int64
	TotalCost        float64
	ErrorRate        float64
	StartTime        time.Time
	EndTime          time.Time
}

// ModelAlert represents a monitoring alert
type ModelAlert struct {
	ID          string
	ModelID     string
	AlertType   string
	Severity    string
	Message     string
	Threshold   float64
	CurrentVal  float64
	CreatedAt   time.Time
	ResolvedAt  *time.Time
}

//Personal.AI order the ending
