// Package model provides domain services for model business logic.
// It implements model health checking, version management, routing selection,
// load balancing, fallback strategies, and monitoring.
package model

import (
   "context"
   "fmt"
   "math/rand"
   "sort"
   "sync"
   "time"
)

// ============================================================================
// Domain Service Interface
// ============================================================================

// ModelService defines the domain service interface for model operations
type ModelService interface {
   // RegisterModel registers a new model
   RegisterModel(ctx context.Context, req RegisterModelRequest) (*Model, error)

   // UpdateModel updates an existing model
   UpdateModel(ctx context.Context, id string, req UpdateModelRequest) (*Model, error)

   // DeleteModel deletes a model
   DeleteModel(ctx context.Context, id string, hard bool) error

   // GetModel retrieves a model by ID
   GetModel(ctx context.Context, id string) (*Model, error)

   // ListModels lists models with filtering
   ListModels(ctx context.Context, filter ModelFilter) ([]*Model, error)

   // SelectModel selects the best model for a request
   SelectModel(ctx context.Context, req SelectionRequest) (*Model, error)

   // HealthCheck performs health check on a model
   HealthCheck(ctx context.Context, id string) (*HealthCheckResult, error)

   // HealthCheckAll performs health check on all models
   HealthCheckAll(ctx context.Context) ([]*HealthCheckResult, error)

   // DeprecateModel marks a model as deprecated
   DeprecateModel(ctx context.Context, id string, message string, replacementID string) error

   // GetModelVersion retrieves a specific model version
   GetModelVersion(ctx context.Context, modelID, version string) (*ModelVersion, error)

   // ListModelVersions lists all versions for a model
   ListModelVersions(ctx context.Context, modelID string) ([]*ModelVersion, error)

   // SetActiveVersion sets the active version for a model
   SetActiveVersion(ctx context.Context, modelID, version string) error

   // DiscoverModels discovers available models from providers
   DiscoverModels(ctx context.Context, provider string) ([]*Model, error)

   // SyncModel synchronizes model information with provider
   SyncModel(ctx context.Context, id string) (*Model, error)

   // GetStatistics retrieves model statistics
   GetStatistics(ctx context.Context, filter ModelFilter) (*ModelStatistics, error)

   // RecordUsage records model usage
   RecordUsage(ctx context.Context, id string, usage UsageRecord) error

   // GetRecommendations gets model recommendations based on requirements
   GetRecommendations(ctx context.Context, req RecommendationRequest) ([]*ModelRecommendation, error)

   // ValidateModel validates model configuration
   ValidateModel(ctx context.Context, model *Model) error
}

// ============================================================================
// Domain Service Implementation
// ============================================================================

// modelService implements the ModelService interface
type modelService struct {
   repo              ModelRepository
   versionRepo       ModelVersionRepository
   monitoringRepo    ModelMonitoringRepository
   healthChecker     HealthChecker
   routingStrategy   RoutingStrategy
   loadBalancer      LoadBalancer
   fallbackHandler   FallbackHandler
   costOptimizer     CostOptimizer
   eventEmitter      ModelEventEmitter
   metricsCollector  MetricsCollector
   mu                sync.RWMutex
   healthCheckCache  map[string]*HealthCheckResult
   cacheTTL          time.Duration
}

// NewModelService creates a new model domain service
func NewModelService(
   repo ModelRepository,
   versionRepo ModelVersionRepository,
   monitoringRepo ModelMonitoringRepository,
   healthChecker HealthChecker,
   routingStrategy RoutingStrategy,
   loadBalancer LoadBalancer,
   fallbackHandler FallbackHandler,
   costOptimizer CostOptimizer,
   eventEmitter ModelEventEmitter,
   metricsCollector MetricsCollector,
) ModelService {
   return &modelService{
       repo:             repo,
       versionRepo:      versionRepo,
       monitoringRepo:   monitoringRepo,
       healthChecker:    healthChecker,
       routingStrategy:  routingStrategy,
       loadBalancer:     loadBalancer,
       fallbackHandler:  fallbackHandler,
       costOptimizer:    costOptimizer,
       eventEmitter:     eventEmitter,
       metricsCollector: metricsCollector,
       healthCheckCache: make(map[string]*HealthCheckResult),
       cacheTTL:         5 * time.Minute,
   }
}

// ============================================================================
// Request/Response Types
// ============================================================================

// RegisterModelRequest represents model registration request
type RegisterModelRequest struct {
   Name         string
   DisplayName  string
   Description  string
   Type         ModelType
   Provider     string
   Version      string
   Endpoint     string
   Config       ModelConfig
   Capabilities ModelCapabilities
   Pricing      PricingInfo
   RateLimits   RateLimits
   HealthCheck  HealthCheckConfig
   Tags         []string
   Metadata     map[string]interface{}
}

// UpdateModelRequest represents model update request
type UpdateModelRequest struct {
   DisplayName  *string
   Description  *string
   Endpoint     *string
   Config       *ModelConfig
   Capabilities *ModelCapabilities
   Pricing      *PricingInfo
   RateLimits   *RateLimits
   HealthCheck  *HealthCheckConfig
   Tags         []string
   Metadata     map[string]interface{}
}

// SelectionRequest represents model selection request
type SelectionRequest struct {
   Type              ModelType
   RequiredCapabilities []string
   MaxCost           *float64
   MinSuccessRate    *float64
   MaxLatency        *int64
   PreferredProviders []string
   ExcludedModels    []string
   InputTokens       int
   OutputTokens      int
   Priority          SelectionPriority
}

// SelectionPriority defines selection priority
type SelectionPriority string

const (
   // PriorityPerformance prioritizes performance
   PriorityPerformance SelectionPriority = "performance"

   // PriorityCost prioritizes cost
   PriorityCost SelectionPriority = "cost"

   // PriorityBalance balances performance and cost
   PriorityBalance SelectionPriority = "balance"

   // PriorityReliability prioritizes reliability
   PriorityReliability SelectionPriority = "reliability"
)

// HealthCheckResult represents health check result
type HealthCheckResult struct {
   ModelID       string
   Healthy       bool
   ResponseTime  time.Duration
   ErrorMessage  string
   CheckedAt     time.Time
   StatusCode    int
   Metadata      map[string]interface{}
}

// UsageRecord represents a usage record
type UsageRecord struct {
   RequestID    string
   InputTokens  int
   OutputTokens int
   Success      bool
   LatencyMs    int64
   Cost         float64
   ErrorType    string
   Timestamp    time.Time
}

// RecommendationRequest represents recommendation request
type RecommendationRequest struct {
   UseCase        string
   Requirements   map[string]interface{}
   Budget         *float64
   LatencyTarget  *int64
   Languages      []string
   Limit          int
}

// ModelRecommendation represents a model recommendation
type ModelRecommendation struct {
   Model       *Model
   Score       float64
   Reasoning   string
   EstimatedCost float64
   Confidence  float64
}

// ============================================================================
// Register Model
// ============================================================================

// RegisterModel registers a new model
func (s *modelService) RegisterModel(ctx context.Context, req RegisterModelRequest) (*Model, error) {
   // Validate request
   if err := s.validateRegisterRequest(req); err != nil {
       return nil, fmt.Errorf("invalid register request: %w", err)
   }

   // Check if model already exists
   exists, err := s.repo.ExistsByName(ctx, req.Name)
   if err != nil {
       return nil, fmt.Errorf("failed to check model existence: %w", err)
   }
   if exists {
       return nil, fmt.Errorf("model with name '%s' already exists", req.Name)
   }

   // Create model entity
   model := NewModel(req.Name, req.DisplayName, string(req.Type), req.Provider, req.Version)
   model.Description = req.Description
   model.Endpoint = req.Endpoint
   model.Config = req.Config
   model.Capabilities = req.Capabilities
   model.Pricing = req.Pricing
   model.RateLimits = req.RateLimits
   model.HealthCheck = req.HealthCheck
   model.Tags = req.Tags
   model.Metadata = req.Metadata

   // Validate model
   if err := model.Validate(); err != nil {
       return nil, fmt.Errorf("model validation failed: %w", err)
   }

   // Persist model
   if err := s.repo.Create(ctx, model); err != nil {
       return nil, fmt.Errorf("failed to create model: %w", err)
   }

   // Create initial version
   version := &ModelVersion{
       ModelID:     model.ID,
       Version:     req.Version,
       ReleaseDate: time.Now(),
       ChangeLog:   "Initial release",
       Config:      req.Config,
       Pricing:     req.Pricing,
       IsActive:    true,
       Deprecated:  false,
       CreatedAt:   time.Now(),
   }
   if err := s.versionRepo.CreateVersion(ctx, version); err != nil {
       return nil, fmt.Errorf("failed to create model version: %w", err)
   }

   // Emit model registered event
   if s.eventEmitter != nil {
       s.eventEmitter.EmitModelRegistered(ctx, model)
   }

   // Collect metrics
   if s.metricsCollector != nil {
       s.metricsCollector.RecordModelRegistered(model)
   }

   // Perform initial health check
   go s.performHealthCheck(context.Background(), model.ID)

   return model, nil
}

// validateRegisterRequest validates registration request
func (s *modelService) validateRegisterRequest(req RegisterModelRequest) error {
   if req.Name == "" {
       return fmt.Errorf("name is required")
   }

   if req.Type == "" {
       return fmt.Errorf("type is required")
   }

   if req.Provider == "" {
       return fmt.Errorf("provider is required")
   }

   if req.Version == "" {
       return fmt.Errorf("version is required")
   }

   if req.Config.BaseURL == "" && req.Endpoint == "" {
       return fmt.Errorf("endpoint or base URL is required")
   }

   return nil
}

// ============================================================================
// Update Model
// ============================================================================

// UpdateModel updates an existing model
func (s *modelService) UpdateModel(ctx context.Context, id string, req UpdateModelRequest) (*Model, error) {
   // Retrieve existing model
   model, err := s.repo.GetByID(ctx, id)
   if err != nil {
       return nil, fmt.Errorf("failed to get model: %w", err)
   }

   if model.IsDeleted() {
       return nil, fmt.Errorf("cannot update deleted model")
   }

   // Apply updates
   if req.DisplayName != nil {
       model.DisplayName = *req.DisplayName
   }

   if req.Description != nil {
       model.Description = *req.Description
   }

   if req.Endpoint != nil {
       model.Endpoint = *req.Endpoint
   }

   if req.Config != nil {
       model.UpdateConfig(*req.Config)
   }

   if req.Capabilities != nil {
       model.Capabilities = *req.Capabilities
   }

   if req.Pricing != nil {
       model.UpdatePricing(*req.Pricing)
   }

   if req.RateLimits != nil {
       model.UpdateRateLimits(*req.RateLimits)
   }

   if req.HealthCheck != nil {
       model.HealthCheck = *req.HealthCheck
   }

   if req.Tags != nil {
       model.Tags = req.Tags
   }

   if req.Metadata != nil {
       for k, v := range req.Metadata {
           model.Metadata[k] = v
       }
   }

   model.UpdatedAt = time.Now()

   // Validate updated model
   if err := model.Validate(); err != nil {
       return nil, fmt.Errorf("validation failed: %w", err)
   }

   // Update in repository
   if err := s.repo.Update(ctx, model); err != nil {
       return nil, fmt.Errorf("failed to update model: %w", err)
   }

   // Invalidate health check cache
   s.invalidateHealthCheckCache(id)

   // Emit model updated event
   if s.eventEmitter != nil {
       s.eventEmitter.EmitModelUpdated(ctx, model)
   }

   return model, nil
}

// ============================================================================
// Delete Model
// ============================================================================

// DeleteModel deletes a model
func (s *modelService) DeleteModel(ctx context.Context, id string, hard bool) error {
   // Retrieve model
   model, err := s.repo.GetByID(ctx, id)
   if err != nil {
       return fmt.Errorf("failed to get model: %w", err)
   }

   if hard {
       if err := s.repo.Delete(ctx, id); err != nil {
           return fmt.Errorf("failed to delete model: %w", err)
       }
   } else {
       model.SoftDelete()
       if err := s.repo.Update(ctx, model); err != nil {
           return fmt.Errorf("failed to soft delete model: %w", err)
       }
   }

   // Invalidate health check cache
   s.invalidateHealthCheckCache(id)

   // Emit model deleted event
   if s.eventEmitter != nil {
       s.eventEmitter.EmitModelDeleted(ctx, id)
   }

   return nil
}

// ============================================================================
// Get Model
// ============================================================================

// GetModel retrieves a model by ID
func (s *modelService) GetModel(ctx context.Context, id string) (*Model, error) {
   model, err := s.repo.GetByID(ctx, id)
   if err != nil {
       return nil, fmt.Errorf("failed to get model: %w", err)
   }

   if model.IsDeleted() {
       return nil, fmt.Errorf("model not found")
   }

   return model, nil
}

// ============================================================================
// List Models
// ============================================================================

// ListModels lists models with filtering
func (s *modelService) ListModels(ctx context.Context, filter ModelFilter) ([]*Model, error) {
   models, err := s.repo.List(ctx, filter)
   if err != nil {
       return nil, fmt.Errorf("failed to list models: %w", err)
   }

   return models, nil
}

// ============================================================================
// Model Selection
// ============================================================================

// SelectModel selects the best model for a request
func (s *modelService) SelectModel(ctx context.Context, req SelectionRequest) (*Model, error) {
   // Get available models matching criteria
   filter := s.buildFilterFromSelection(req)
   models, err := s.repo.List(ctx, filter)
   if err != nil {
       return nil, fmt.Errorf("failed to list models: %w", err)
   }

   // Filter by required capabilities
   models = s.filterByCapabilities(models, req.RequiredCapabilities)

   // Exclude specified models
   models = s.excludeModels(models, req.ExcludedModels)

   // Filter by health status
   models = s.filterByHealth(models)

   if len(models) == 0 {
       return nil, fmt.Errorf("no suitable models found")
   }

   // Apply routing strategy
   var selectedModel *Model
   switch req.Priority {
   case PriorityCost:
       selectedModel = s.selectByCost(models, req.InputTokens, req.OutputTokens)
   case PriorityPerformance:
       selectedModel = s.selectByPerformance(models)
   case PriorityReliability:
       selectedModel = s.selectByReliability(models)
   default: // PriorityBalance
       selectedModel = s.selectBalanced(models, req.InputTokens, req.OutputTokens)
   }

   if selectedModel == nil {
       return nil, fmt.Errorf("model selection failed")
   }

   // Apply load balancing if multiple instances available
   if s.loadBalancer != nil {
       selectedModel = s.loadBalancer.SelectInstance(selectedModel)
   }

   return selectedModel, nil
}

// buildFilterFromSelection builds filter from selection request
func (s *modelService) buildFilterFromSelection(req SelectionRequest) ModelFilter {
   filter := NewModelFilter()
   filter = filter.WithType(req.Type)
   filter = filter.WithOnlyAvailable(true)

   if len(req.PreferredProviders) > 0 {
       filter = filter.WithProviders(req.PreferredProviders...)
   }

   if req.MinSuccessRate != nil {
       filter = filter.WithMinSuccessRate(*req.MinSuccessRate)
   }

   if req.MaxLatency != nil {
       filter = filter.WithMaxAvgLatency(*req.MaxLatency)
   }

   return filter
}

// filterByCapabilities filters models by required capabilities
func (s *modelService) filterByCapabilities(models []*Model, capabilities []string) []*Model {
   if len(capabilities) == 0 {
       return models
   }

   filtered := make([]*Model, 0)
   for _, model := range models {
       hasAll := true
       for _, cap := range capabilities {
           if !model.SupportsCapability(cap) {
               hasAll = false
               break
           }
       }
       if hasAll {
           filtered = append(filtered, model)
       }
   }

   return filtered
}

// excludeModels excludes specified models
func (s *modelService) excludeModels(models []*Model, excludedIDs []string) []*Model {
   if len(excludedIDs) == 0 {
       return models
   }

   excluded := make(map[string]bool)
   for _, id := range excludedIDs {
       excluded[id] = true
   }

   filtered := make([]*Model, 0)
   for _, model := range models {
       if !excluded[model.ID] {
           filtered = append(filtered, model)
       }
   }

   return filtered
}

// filterByHealth filters models by health status
func (s *modelService) filterByHealth(models []*Model) []*Model {
   filtered := make([]*Model, 0)
   for _, model := range models {
       if s.isModelHealthy(model.ID) {
           filtered = append(filtered, model)
       }
   }
   return filtered
}

// selectByCost selects model with lowest cost
func (s *modelService) selectByCost(models []*Model, inputTokens, outputTokens int) *Model {
   if len(models) == 0 {
       return nil
   }

   var bestModel *Model
   lowestCost := float64(0)

   for _, model := range models {
       cost := model.GetCost(inputTokens, outputTokens)
       if bestModel == nil || cost < lowestCost {
           bestModel = model
           lowestCost = cost
       }
   }

   return bestModel
}

// selectByPerformance selects model with best performance
func (s *modelService) selectByPerformance(models []*Model) *Model {
   if len(models) == 0 {
       return nil
   }

   sort.Slice(models, func(i, j int) bool {
       return models[i].Performance.AvgLatencyMs < models[j].Performance.AvgLatencyMs
   })

   return models[0]
}

// selectByReliability selects model with highest reliability
func (s *modelService) selectByReliability(models []*Model) *Model {
   if len(models) == 0 {
       return nil
   }

   sort.Slice(models, func(i, j int) bool {
       return models[i].GetSuccessRate() > models[j].GetSuccessRate()
   })

   return models[0]
}

// selectBalanced selects model with best balance of cost and performance
func (s *modelService) selectBalanced(models []*Model, inputTokens, outputTokens int) *Model {
   if len(models) == 0 {
       return nil
   }

   type scoredModel struct {
       model *Model
       score float64
   }

   scored := make([]scoredModel, 0)
   for _, model := range models {
       cost := model.GetCost(inputTokens, outputTokens)
       latency := float64(model.Performance.AvgLatencyMs)
       reliability := model.GetSuccessRate()

       // Normalize and combine scores (lower is better)
       costScore := cost / 100.0           // Normalize cost
       latencyScore := latency / 1000.0    // Normalize latency
       reliabilityScore := 1.0 - reliability // Invert reliability

       // Weighted combination
       combinedScore := (costScore * 0.4) + (latencyScore * 0.3) + (reliabilityScore * 0.3)

       scored = append(scored, scoredModel{
           model: model,
           score: combinedScore,
       })
   }

   // Sort by score (lower is better)
   sort.Slice(scored, func(i, j int) bool {
       return scored[i].score < scored[j].score
   })

   return scored[0].model
}

// ============================================================================
// Health Check
// ============================================================================

// HealthCheck performs health check on a model
func (s *modelService) HealthCheck(ctx context.Context, id string) (*HealthCheckResult, error) {
   // Check cache first
   if cached := s.getCachedHealthCheck(id); cached != nil {
       return cached, nil
   }

   // Perform health check
   result, err := s.performHealthCheck(ctx, id)
   if err != nil {
       return nil, fmt.Errorf("health check failed: %w", err)
   }

   // Cache result
   s.cacheHealthCheck(id, result)

   return result, nil
}

// performHealthCheck performs actual health check
func (s *modelService) performHealthCheck(ctx context.Context, id string) (*HealthCheckResult, error) {
   model, err := s.repo.GetByID(ctx, id)
   if err != nil {
       return nil, fmt.Errorf("failed to get model: %w", err)
   }

   startTime := time.Now()
   result := &HealthCheckResult{
       ModelID:   id,
       CheckedAt: startTime,
   }

   // Use health checker to perform check
   if s.healthChecker != nil {
       healthy, responseTime, statusCode, errMsg := s.healthChecker.Check(ctx, model)
       result.Healthy = healthy
       result.ResponseTime = responseTime
       result.StatusCode = statusCode
       result.ErrorMessage = errMsg
   } else {
       // Default check: just verify model is available
       result.Healthy = model.IsAvailable()
       result.ResponseTime = time.Since(startTime)
   }

   // Update model health status
   model.UpdateHealthCheck(result.Healthy)
   s.repo.UpdateHealthCheck(ctx, id, result.Healthy)

   // Emit event if health check failed
   if !result.Healthy && s.eventEmitter != nil {
       s.eventEmitter.EmitModelHealthCheckFailed(ctx, id)
   }

   // Record metric
   if s.monitoringRepo != nil {
       metric := &ModelMetric{
           ModelID:    id,
           MetricType: "health_check",
           Value:      boolToFloat(result.Healthy),
           Timestamp:  result.CheckedAt,
           Labels: map[string]string{
               "status_code": fmt.Sprintf("%d", result.StatusCode),
           },
       }
       s.monitoringRepo.RecordMetric(ctx, metric)
   }

   return result, nil
}

// HealthCheckAll performs health check on all models
func (s *modelService) HealthCheckAll(ctx context.Context) ([]*HealthCheckResult, error) {
   models, err := s.repo.GetModelsForHealthCheck(ctx)
   if err != nil {
       return nil, fmt.Errorf("failed to get models: %w", err)
   }

   results := make([]*HealthCheckResult, 0, len(models))
   var wg sync.WaitGroup
   resultChan := make(chan *HealthCheckResult, len(models))

   for _, model := range models {
       if !model.ShouldPerformHealthCheck() {
           continue
       }

       wg.Add(1)
       go func(m *Model) {
           defer wg.Done()
           result, err := s.performHealthCheck(ctx, m.ID)
           if err == nil {
               resultChan <- result
           }
       }(model)
   }

   go func() {
       wg.Wait()
       close(resultChan)
   }()

   for result := range resultChan {
       results = append(results, result)
   }

   return results, nil
}

// getCachedHealthCheck retrieves cached health check result
func (s *modelService) getCachedHealthCheck(id string) *HealthCheckResult {
   s.mu.RLock()
   defer s.mu.RUnlock()

   cached, ok := s.healthCheckCache[id]
   if !ok {
       return nil
   }

   // Check if cache is still valid
   if time.Since(cached.CheckedAt) > s.cacheTTL {
       return nil
   }

   return cached
}

// cacheHealthCheck caches health check result
func (s *modelService) cacheHealthCheck(id string, result *HealthCheckResult) {
   s.mu.Lock()
   defer s.mu.Unlock()
   s.healthCheckCache[id] = result
}

// invalidateHealthCheckCache invalidates health check cache
func (s *modelService) invalidateHealthCheckCache(id string) {
   s.mu.Lock()
   defer s.mu.Unlock()
   delete(s.healthCheckCache, id)
}

// isModelHealthy checks if model is healthy
func (s *modelService) isModelHealthy(id string) bool {
   result := s.getCachedHealthCheck(id)
   if result == nil {
       return true // Assume healthy if no recent check
   }
   return result.Healthy
}

// ============================================================================
// Model Deprecation
// ============================================================================

// DeprecateModel marks a model as deprecated
func (s *modelService) DeprecateModel(ctx context.Context, id string, message string, replacementID string) error {
   model, err := s.repo.GetByID(ctx, id)
   if err != nil {
       return fmt.Errorf("failed to get model: %w", err)
   }

   // Validate replacement model if specified
   if replacementID != "" {
       replacement, err := s.repo.GetByID(ctx, replacementID)
       if err != nil {
           return fmt.Errorf("invalid replacement model: %w", err)
       }
       if !replacement.IsAvailable() {
           return fmt.Errorf("replacement model is not available")
       }
   }

   model.MarkDeprecated(message, replacementID)

   if err := s.repo.Update(ctx, model); err != nil {
       return fmt.Errorf("failed to update model: %w", err)
   }

   // Emit deprecated event
   if s.eventEmitter != nil {
       s.eventEmitter.EmitModelDeprecated(ctx, id)
   }

   return nil
}

// ============================================================================
// Version Management
// ============================================================================

// GetModelVersion retrieves a specific model version
func (s *modelService) GetModelVersion(ctx context.Context, modelID, version string) (*ModelVersion, error) {
   modelVersion, err := s.versionRepo.GetVersion(ctx, modelID, version)
   if err != nil {
       return nil, fmt.Errorf("failed to get model version: %w", err)
   }

   return modelVersion, nil
}

// ListModelVersions lists all versions for a model
func (s *modelService) ListModelVersions(ctx context.Context, modelID string) ([]*ModelVersion, error) {
   versions, err := s.versionRepo.ListVersions(ctx, modelID)
   if err != nil {
       return nil, fmt.Errorf("failed to list versions: %w", err)
   }

   return versions, nil
}

// SetActiveVersion sets the active version for a model
func (s *modelService) SetActiveVersion(ctx context.Context, modelID, version string) error {
   // Verify version exists
   _, err := s.versionRepo.GetVersion(ctx, modelID, version)
   if err != nil {
       return fmt.Errorf("version not found: %w", err)
   }

   if err := s.versionRepo.SetActiveVersion(ctx, modelID, version); err != nil {
       return fmt.Errorf("failed to set active version: %w", err)
   }

   // Invalidate caches
   s.invalidateHealthCheckCache(modelID)

   return nil
}

// ============================================================================
// Model Discovery
// ============================================================================

// DiscoverModels discovers available models from providers
func (s *modelService) DiscoverModels(ctx context.Context, provider string) ([]*Model, error) {
   // This would typically integrate with provider APIs
   // For now, return placeholder
   return nil, fmt.Errorf("model discovery not yet implemented")
}

// SyncModel synchronizes model information with provider
func (s *modelService) SyncModel(ctx context.Context, id string) (*Model, error) {
   model, err := s.repo.GetByID(ctx, id)
   if err != nil {
       return nil, fmt.Errorf("failed to get model: %w", err)
   }

   // This would typically fetch latest info from provider
   // For now, just perform health check
   _, err = s.performHealthCheck(ctx, id)
   if err != nil {
       return nil, fmt.Errorf("sync failed: %w", err)
   }

   return model, nil
}

// ============================================================================
// Statistics
// ============================================================================

// GetStatistics retrieves model statistics
func (s *modelService) GetStatistics(ctx context.Context, filter ModelFilter) (*ModelStatistics, error) {
   stats, err := s.repo.GetStatistics(ctx, filter)
   if err != nil {
       return nil, fmt.Errorf("failed to get statistics: %w", err)
   }

   return stats, nil
}

// ============================================================================
// Usage Recording
// ============================================================================

// RecordUsage records model usage
func (s *modelService) RecordUsage(ctx context.Context, id string, usage UsageRecord) error {
   model, err := s.repo.GetByID(ctx, id)
   if err != nil {
       return fmt.Errorf("failed to get model: %w", err)
   }

   // Calculate cost
   cost := model.GetCost(usage.InputTokens, usage.OutputTokens)
   totalTokens := usage.InputTokens + usage.OutputTokens

   // Update model statistics
   model.RecordRequest(usage.Success, totalTokens, cost)

   if err := s.repo.UpdateStats(ctx, id, model.Stats); err != nil {
       return fmt.Errorf("failed to update stats: %w", err)
   }

   // Record metric
   if s.monitoringRepo != nil {
       metric := &ModelMetric{
           ModelID:    id,
           MetricType: "usage",
           Value:      float64(totalTokens),
           Timestamp:  usage.Timestamp,
           Labels: map[string]string{
               "success":      fmt.Sprintf("%v", usage.Success),
               "request_id":   usage.RequestID,
               "input_tokens": fmt.Sprintf("%d", usage.InputTokens),
               "output_tokens": fmt.Sprintf("%d", usage.OutputTokens),
           },
       }
       s.monitoringRepo.RecordMetric(ctx, metric)

       // Record cost metric
       costMetric := &ModelMetric{
           ModelID:    id,
           MetricType: "cost",
           Value:      cost,
           Timestamp:  usage.Timestamp,
           Labels: map[string]string{
               "request_id": usage.RequestID,
           },
       }
       s.monitoringRepo.RecordMetric(ctx, costMetric)

       // Record latency metric
       latencyMetric := &ModelMetric{
           ModelID:    id,
           MetricType: "latency",
           Value:      float64(usage.LatencyMs),
           Timestamp:  usage.Timestamp,
           Labels: map[string]string{
               "request_id": usage.RequestID,
           },
       }
       s.monitoringRepo.RecordMetric(ctx, latencyMetric)
   }

   // Collect metrics
   if s.metricsCollector != nil {
       s.metricsCollector.RecordUsage(id, usage)
   }

   return nil
}

// ============================================================================
// Model Recommendations
// ============================================================================

// GetRecommendations gets model recommendations based on requirements
func (s *modelService) GetRecommendations(ctx context.Context, req RecommendationRequest) ([]*ModelRecommendation, error) {
   // Build filter from requirements
   filter := s.buildRecommendationFilter(req)

   // Get candidate models
   models, err := s.repo.List(ctx, filter)
   if err != nil {
       return nil, fmt.Errorf("failed to list models: %w", err)
   }

   // Score and rank models
   recommendations := make([]*ModelRecommendation, 0)
   for _, model := range models {
       score, reasoning := s.scoreModel(model, req)

       // Estimate cost
       estimatedCost := s.estimateModelCost(model, req)

       // Calculate confidence
       confidence := s.calculateConfidence(model, req)

       recommendations = append(recommendations, &ModelRecommendation{
           Model:         model,
           Score:         score,
           Reasoning:     reasoning,
           EstimatedCost: estimatedCost,
           Confidence:    confidence,
       })
   }

   // Sort by score (descending)
   sort.Slice(recommendations, func(i, j int) bool {
       return recommendations[i].Score > recommendations[j].Score
   })

   // Limit results
   if req.Limit > 0 && len(recommendations) > req.Limit {
       recommendations = recommendations[:req.Limit]
   }

   return recommendations, nil
}

// buildRecommendationFilter builds filter from recommendation request
func (s *modelService) buildRecommendationFilter(req RecommendationRequest) ModelFilter {
   filter := NewModelFilter()
   filter = filter.WithOnlyAvailable(true)

   if req.Budget != nil {
       filter = filter.WithPriceRange(0, *req.Budget, 0, *req.Budget)
   }

   if req.LatencyTarget != nil {
       filter = filter.WithMaxAvgLatency(*req.LatencyTarget)
   }

   return filter
}

// scoreModel scores a model based on requirements
func (s *modelService) scoreModel(model *Model, req RecommendationRequest) (float64, string) {
   var score float64
   var reasons []string

   // Score based on performance
   if model.Performance.AvgLatencyMs < 500 {
       score += 30
       reasons = append(reasons, "Low latency")
   } else if model.Performance.AvgLatencyMs < 1000 {
       score += 20
       reasons = append(reasons, "Moderate latency")
   } else {
       score += 10
       reasons = append(reasons, "Higher latency")
   }

   // Score based on reliability
   successRate := model.GetSuccessRate()
   if successRate > 0.99 {
       score += 30
       reasons = append(reasons, "Excellent reliability")
   } else if successRate > 0.95 {
       score += 20
       reasons = append(reasons, "Good reliability")
   } else {
       score += 10
       reasons = append(reasons, "Moderate reliability")
   }

   // Score based on cost efficiency
   avgCost := model.GetAverageCostPerRequest()
   if avgCost < 0.01 {
       score += 20
       reasons = append(reasons, "Very cost-effective")
   } else if avgCost < 0.05 {
       score += 15
       reasons = append(reasons, "Cost-effective")
   } else {
       score += 5
       reasons = append(reasons, "Higher cost")
   }

   // Score based on uptime
   if model.Performance.UptimePercentage > 99.9 {
       score += 20
       reasons = append(reasons, "Excellent uptime")
   } else if model.Performance.UptimePercentage > 99.0 {
       score += 10
       reasons = append(reasons, "Good uptime")
   }

   // Bonus for matching languages
   if len(req.Languages) > 0 {
       for _, lang := range req.Languages {
           for _, supported := range model.Capabilities.SupportedLanguages {
               if lang == supported {
                   score += 5
                   break
               }
           }
       }
   }

   reasoning := fmt.Sprintf("Score breakdown: %s", joinStrings(reasons, ", "))
   return score, reasoning
}

// estimateModelCost estimates cost for a model
func (s *modelService) estimateModelCost(model *Model, req RecommendationRequest) float64 {
   // Rough estimation based on average usage
   avgInputTokens := 1000
   avgOutputTokens := 500

   return model.GetCost(avgInputTokens, avgOutputTokens)
}

// calculateConfidence calculates confidence in recommendation
func (s *modelService) calculateConfidence(model *Model, req RecommendationRequest) float64 {
   var confidence float64 = 0.5 // Base confidence

   // Increase confidence based on usage statistics
   if model.Stats.TotalRequests > 10000 {
       confidence += 0.3
   } else if model.Stats.TotalRequests > 1000 {
       confidence += 0.2
   } else if model.Stats.TotalRequests > 100 {
       confidence += 0.1
   }

   // Increase confidence based on recent activity
   if model.Stats.LastRequestAt != nil {
       hoursSinceLastRequest := time.Since(*model.Stats.LastRequestAt).Hours()
       if hoursSinceLastRequest < 1 {
           confidence += 0.1
       } else if hoursSinceLastRequest < 24 {
           confidence += 0.05
       }
   }

   // Decrease confidence for deprecated models
   if model.IsDeprecated() {
       confidence -= 0.2
   }

   // Ensure confidence is between 0 and 1
   if confidence > 1.0 {
       confidence = 1.0
   } else if confidence < 0.0 {
       confidence = 0.0
   }

   return confidence
}

// ============================================================================
// Model Validation
// ============================================================================

// ValidateModel validates model configuration
func (s *modelService) ValidateModel(ctx context.Context, model *Model) error {
   // Basic validation
   if err := model.Validate(); err != nil {
       return err
   }

   // Perform test health check
   if s.healthChecker != nil {
       healthy, _, _, errMsg := s.healthChecker.Check(ctx, model)
       if !healthy {
           return fmt.Errorf("health check failed: %s", errMsg)
       }
   }

   // Validate pricing configuration
   if model.Pricing.InputTokenCost < 0 || model.Pricing.OutputTokenCost < 0 {
       return fmt.Errorf("invalid pricing configuration")
   }

   // Validate rate limits
   if model.RateLimits.RequestsPerMinute <= 0 || model.RateLimits.TokensPerMinute <= 0 {
       return fmt.Errorf("invalid rate limits")
   }

   // Validate capabilities
   if model.Capabilities.MaxContextLength <= 0 {
       return fmt.Errorf("invalid max context length")
   }

   return nil
}

// ============================================================================
// Helper Interfaces
// ============================================================================

// HealthChecker defines health check interface
type HealthChecker interface {
   Check(ctx context.Context, model *Model) (healthy bool, responseTime time.Duration, statusCode int, errMsg string)
}

// RoutingStrategy defines routing strategy interface
type RoutingStrategy interface {
   SelectModel(ctx context.Context, models []*Model, req SelectionRequest) *Model
}

// LoadBalancer defines load balancing interface
type LoadBalancer interface {
   SelectInstance(model *Model) *Model
}

// FallbackHandler defines fallback handling interface
type FallbackHandler interface {
   HandleFailure(ctx context.Context, model *Model, err error) (*Model, error)
}

// CostOptimizer defines cost optimization interface
type CostOptimizer interface {
   OptimizeSelection(ctx context.Context, models []*Model, budget float64) *Model
}

// MetricsCollector defines metrics collection interface
type MetricsCollector interface {
   RecordModelRegistered(model *Model)
   RecordUsage(modelID string, usage UsageRecord)
   RecordHealthCheck(modelID string, healthy bool, latency time.Duration)
}

// ============================================================================
// Utility Functions
// ============================================================================

// boolToFloat converts boolean to float (1.0 for true, 0.0 for false)
func boolToFloat(b bool) float64 {
   if b {
       return 1.0
   }
   return 0.0
}

// joinStrings joins strings with separator
func joinStrings(strs []string, sep string) string {
   result := ""
   for i, s := range strs {
       if i > 0 {
           result += sep
       }
       result += s
   }
   return result
}

// ============================================================================
// Advanced Features
// ============================================================================

// AutoScalingService handles model auto-scaling
type AutoScalingService struct {
   service       ModelService
   repo          ModelRepository
   scaleUpThreshold   float64
   scaleDownThreshold float64
}

// NewAutoScalingService creates a new auto-scaling service
func NewAutoScalingService(service ModelService, repo ModelRepository) *AutoScalingService {
   return &AutoScalingService{
       service:            service,
       repo:               repo,
       scaleUpThreshold:   0.8,  // 80% utilization
       scaleDownThreshold: 0.3,  // 30% utilization
   }
}

// MonitorAndScale monitors models and triggers scaling
func (a *AutoScalingService) MonitorAndScale(ctx context.Context) error {
   models, err := a.repo.GetAvailable(ctx)
   if err != nil {
       return fmt.Errorf("failed to get models: %w", err)
   }

   for _, model := range models {
       // Calculate utilization based on requests
       utilization := a.calculateUtilization(model)

       if utilization > a.scaleUpThreshold {
           // Scale up logic would go here
           continue
       }

       if utilization < a.scaleDownThreshold {
           // Scale down logic would go here
           continue
       }
   }

   return nil
}

// calculateUtilization calculates model utilization
func (a *AutoScalingService) calculateUtilization(model *Model) float64 {
   if model.Stats.TotalRequests == 0 {
       return 0.0
   }

   // Simple utilization based on requests vs capacity
   capacity := float64(model.RateLimits.RequestsPerMinute * 60) // per hour
   usage := float64(model.Stats.TotalRequests)

   return usage / capacity
}

// ============================================================================
// Circuit Breaker Pattern
// ============================================================================

// CircuitBreaker implements circuit breaker pattern for model calls
type CircuitBreaker struct {
   failureThreshold int
   timeout          time.Duration
   mu               sync.RWMutex
   failures         map[string]int
   lastFailure      map[string]time.Time
   state            map[string]CircuitState
}

// CircuitState represents circuit breaker state
type CircuitState string

const (
   StateClosed    CircuitState = "closed"
   StateOpen      CircuitState = "open"
   StateHalfOpen  CircuitState = "half_open"
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold int, timeout time.Duration) *CircuitBreaker {
   return &CircuitBreaker{
       failureThreshold: failureThreshold,
       timeout:          timeout,
       failures:         make(map[string]int),
       lastFailure:      make(map[string]time.Time),
       state:            make(map[string]CircuitState),
   }
}

// Call executes a call through the circuit breaker
func (cb *CircuitBreaker) Call(modelID string, fn func() error) error {
   cb.mu.RLock()
   state := cb.state[modelID]
   cb.mu.RUnlock()

   if state == StateOpen {
       // Check if timeout has passed
       cb.mu.RLock()
       lastFail := cb.lastFailure[modelID]
       cb.mu.RUnlock()

       if time.Since(lastFail) > cb.timeout {
           // Transition to half-open
           cb.mu.Lock()
           cb.state[modelID] = StateHalfOpen
           cb.mu.Unlock()
       } else {
           return fmt.Errorf("circuit breaker is open for model %s", modelID)
       }
   }

   // Execute call
   err := fn()

   if err != nil {
       cb.recordFailure(modelID)
       return err
   }

   cb.recordSuccess(modelID)
   return nil
}

// recordFailure records a failure
func (cb *CircuitBreaker) recordFailure(modelID string) {
   cb.mu.Lock()
   defer cb.mu.Unlock()

   cb.failures[modelID]++
   cb.lastFailure[modelID] = time.Now()

   if cb.failures[modelID] >= cb.failureThreshold {
       cb.state[modelID] = StateOpen
   }
}

// recordSuccess records a success
func (cb *CircuitBreaker) recordSuccess(modelID string) {
   cb.mu.Lock()
   defer cb.mu.Unlock()

   cb.failures[modelID] = 0
   cb.state[modelID] = StateClosed
}

// ============================================================================
// Rate Limiter
// ============================================================================

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
   mu       sync.Mutex
   buckets  map[string]*TokenBucket
}

// TokenBucket represents a token bucket
type TokenBucket struct {
   capacity    int
   tokens      int
   refillRate  int
   lastRefill  time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
   return &RateLimiter{
       buckets: make(map[string]*TokenBucket),
   }
}

// Allow checks if request is allowed
func (rl *RateLimiter) Allow(modelID string, limits RateLimits) bool {
   rl.mu.Lock()
   defer rl.mu.Unlock()

   bucket, exists := rl.buckets[modelID]
   if !exists {
       bucket = &TokenBucket{
           capacity:   limits.RequestsPerMinute,
           tokens:     limits.RequestsPerMinute,
           refillRate: limits.RequestsPerMinute,
           lastRefill: time.Now(),
       }
       rl.buckets[modelID] = bucket
   }

   // Refill tokens
   now := time.Now()
   elapsed := now.Sub(bucket.lastRefill)
   tokensToAdd := int(elapsed.Minutes() * float64(bucket.refillRate))

   if tokensToAdd > 0 {
       bucket.tokens += tokensToAdd
       if bucket.tokens > bucket.capacity {
           bucket.tokens = bucket.capacity
       }
       bucket.lastRefill = now
   }

   // Check if we have tokens
   if bucket.tokens > 0 {
       bucket.tokens--
       return true
   }

   return false
}

// ============================================================================
// Model Pool Manager
// ============================================================================

// ModelPoolManager manages a pool of model instances
type ModelPoolManager struct {
   mu      sync.RWMutex
   pools   map[string]*ModelPool
   service ModelService
}

// ModelPool represents a pool of model instances
type ModelPool struct {
   ModelID   string
   Instances []*ModelInstance
   Strategy  LoadBalancingStrategy
   Index     int
}

// ModelInstance represents a model instance
type ModelInstance struct {
   ID        string
   ModelID   string
   Endpoint  string
   Healthy   bool
   Load      int
   LastCheck time.Time
}

// LoadBalancingStrategy defines load balancing strategy
type LoadBalancingStrategy string

const (
   StrategyRoundRobin    LoadBalancingStrategy = "round_robin"
   StrategyLeastLoad     LoadBalancingStrategy = "least_load"
   StrategyRandom        LoadBalancingStrategy = "random"
   StrategyWeightedRandom LoadBalancingStrategy = "weighted_random"
)

// NewModelPoolManager creates a new model pool manager
func NewModelPoolManager(service ModelService) *ModelPoolManager {
   return &ModelPoolManager{
       pools:   make(map[string]*ModelPool),
       service: service,
   }
}

// GetInstance gets an instance from the pool
func (m *ModelPoolManager) GetInstance(modelID string) (*ModelInstance, error) {
   m.mu.RLock()
   pool, exists := m.pools[modelID]
   m.mu.RUnlock()

   if !exists {
       return nil, fmt.Errorf("no pool for model %s", modelID)
   }

   return m.selectInstance(pool), nil
}

// selectInstance selects an instance based on strategy
func (m *ModelPoolManager) selectInstance(pool *ModelPool) *ModelInstance {
   healthyInstances := make([]*ModelInstance, 0)
   for _, instance := range pool.Instances {
       if instance.Healthy {
           healthyInstances = append(healthyInstances, instance)
       }
   }

   if len(healthyInstances) == 0 {
       return nil
   }

   switch pool.Strategy {
   case StrategyRoundRobin:
       instance := healthyInstances[pool.Index%len(healthyInstances)]
       pool.Index++
       return instance

   case StrategyLeastLoad:
       var selected *ModelInstance
       minLoad := int(^uint(0) >> 1) // Max int
       for _, instance := range healthyInstances {
           if instance.Load < minLoad {
               minLoad = instance.Load
               selected = instance
           }
       }
       return selected

   case StrategyRandom:
       return healthyInstances[rand.Intn(len(healthyInstances))]

   default:
       return healthyInstances[0]
   }
}

//Personal.AI order the ending
