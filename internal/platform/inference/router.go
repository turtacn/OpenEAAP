// Package inference provides model routing capabilities for selecting
// the optimal inference engine based on request characteristics.
package inference

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"sync"
	"time"

	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/pkg/errors"
	"github.com/openeeap/openeeap/pkg/types"
)

// RoutingStrategy defines the routing strategy type
type RoutingStrategy string

const (
	// StrategyRoundRobin routes requests in round-robin fashion
	StrategyRoundRobin RoutingStrategy = "round_robin"

	// StrategyWeightedRandom routes based on engine weights
	StrategyWeightedRandom RoutingStrategy = "weighted_random"

	// StrategyLeastLatency routes to engine with lowest latency
	StrategyLeastLatency RoutingStrategy = "least_latency"

	// StrategyLeastLoad routes to engine with lowest load
	StrategyLeastLoad RoutingStrategy = "least_load"

	// StrategyABTest routes for A/B testing
	StrategyABTest RoutingStrategy = "ab_test"

	// StrategyCostOptimized routes based on cost optimization
	StrategyCostOptimized RoutingStrategy = "cost_optimized"

	// StrategyLatencyOptimized routes based on latency requirements
	StrategyLatencyOptimized RoutingStrategy = "latency_optimized"
)

// EngineMetrics contains runtime metrics for an engine
type EngineMetrics struct {
	AverageLatency    time.Duration
	P95Latency        time.Duration
	P99Latency        time.Duration
	ErrorRate         float64
	CurrentLoad       int
	MaxConcurrency    int
	TotalRequests     int64
	SuccessfulRequests int64
	FailedRequests    int64
	LastUpdated       time.Time
}

// EngineConfig contains configuration for an inference engine
type EngineConfig struct {
	Name           string
	Engine         InferenceEngine
	Weight         int
	Priority       int
	MaxConcurrency int
	CostPerToken   float64
	MaxLatency     time.Duration
	Enabled        bool
	ABTestGroup    string
	Tags           map[string]string
}

// RoutingRule defines routing rules based on request characteristics
type RoutingRule struct {
	Name         string
	Priority     int
	Condition    RoutingCondition
	TargetEngine string
	Enabled      bool
}

// RoutingCondition defines conditions for routing
type RoutingCondition struct {
	ModelPattern      string
	MaxLatency        *time.Duration
	MaxCostPerToken   *float64
	MinComplexity     *int
	MaxComplexity     *int
	RequiredTags      map[string]string
	UserIDPattern     string
	TenantIDPattern   string
}

// ABTestConfig contains A/B test configuration
type ABTestConfig struct {
	Name            string
	Enabled         bool
	TrafficSplit    map[string]int // engine -> percentage
	HashKey         string         // "user_id" or "request_id"
	StartTime       time.Time
	EndTime         time.Time
}

// Router handles model routing and engine selection
type Router struct {
	logger           logging.Logger
	metricsCollector *metrics.MetricsCollector

	engines          map[string]*EngineConfig
	enginesMu        sync.RWMutex

	metrics          map[string]*EngineMetrics
	metricsMu        sync.RWMutex

	rules            []*RoutingRule
	rulesMu          sync.RWMutex

	strategy         RoutingStrategy
	abTestConfig     *ABTestConfig

	roundRobinIndex  int
	roundRobinMu     sync.Mutex

	rand             *rand.Rand
	randMu           sync.Mutex
}

// RouterConfig contains configuration for the router
type RouterConfig struct {
	Strategy         RoutingStrategy
	DefaultEngine    string
	ABTestConfig     *ABTestConfig
	MetricsInterval  time.Duration
}

// NewRouter creates a new model router
func NewRouter(
	logger logging.Logger,
	metricsCollector metrics.MetricsCollector,
	config *RouterConfig,
) *Router {
	r := &Router{
		logger:           logger,
		metricsCollector: metricsCollector,
		engines:          make(map[string]*EngineConfig),
		metrics:          make(map[string]*EngineMetrics),
		rules:            make([]*RoutingRule, 0),
		strategy:         config.Strategy,
		abTestConfig:     config.ABTestConfig,
		rand:             rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// Start metrics collector
	if config.MetricsInterval > 0 {
		go r.collectMetrics(config.MetricsInterval)
	}

	return r
}

// RegisterEngine registers an inference engine
func (r *Router) RegisterEngine(config *EngineConfig) error {
	if config.Name == "" {
		return errors.ValidationError( "engine name is required")
	}

	if config.Engine == nil {
		return errors.ValidationError( "engine instance is required")
	}

	r.enginesMu.Lock()
	defer r.enginesMu.Unlock()

	r.engines[config.Name] = config

	// Initialize metrics
	r.metricsMu.Lock()
	r.metrics[config.Name] = &EngineMetrics{
		LastUpdated: time.Now(),
	}
	r.metricsMu.Unlock()

	r.logger.Info(context.Background(), "engine registered",
		"name", config.Name,
		"weight", config.Weight,
		"priority", config.Priority,
	)

	return nil
}

// UnregisterEngine unregisters an inference engine
func (r *Router) UnregisterEngine(name string) error {
	r.enginesMu.Lock()
	defer r.enginesMu.Unlock()

	if _, exists := r.engines[name]; !exists {
		return errors.New(errors.CodeNotFound, fmt.Sprintf("engine %s not found", name))
	}

	delete(r.engines, name)

	r.metricsMu.Lock()
	delete(r.metrics, name)
	r.metricsMu.Unlock()

	r.logger.Info(context.Background(), "engine unregistered", "name", name)

	return nil
}

// AddRoutingRule adds a routing rule
func (r *Router) AddRoutingRule(rule *RoutingRule) error {
	if rule.Name == "" {
		return errors.ValidationError( "rule name is required")
	}

	r.rulesMu.Lock()
	defer r.rulesMu.Unlock()

	r.rules = append(r.rules, rule)

	// Sort rules by priority (higher priority first)
	r.sortRules()

	r.logger.Info(context.Background(), "routing rule added",
		"name", rule.Name,
		"priority", rule.Priority,
		"target", rule.TargetEngine,
	)

	return nil
}

// Route selects the appropriate engine for a request
func (r *Router) Route(ctx context.Context, req *InferenceRequest) (InferenceEngine, error) {
	startTime := time.Now()

	// Try rule-based routing first
	engineName, matchedRule := r.matchRoutingRule(ctx, req)
	if engineName != "" {
  r.logger.WithContext(ctx).Info("routing by rule", logging.Any("rule", matchedRule), logging.Any("engine", engineName))

		engine, err := r.getEngine(engineName)
		if err == nil {
			r.recordRoutingMetrics("rule", engineName, time.Since(startTime))
			return engine, nil
		}

  r.logger.WithContext(ctx).Warn("rule-matched engine not available", logging.Any("engine", engineName), logging.Error(err))
	}

	// Fallback to strategy-based routing
	var engine InferenceEngine
	var err error

	switch r.strategy {
	case StrategyRoundRobin:
		engine, err = r.routeRoundRobin(ctx)

	case StrategyWeightedRandom:
		engine, err = r.routeWeightedRandom(ctx)

	case StrategyLeastLatency:
		engine, err = r.routeLeastLatency(ctx)

	case StrategyLeastLoad:
		engine, err = r.routeLeastLoad(ctx)

	case StrategyABTest:
		engine, err = r.routeABTest(ctx, req)

	case StrategyCostOptimized:
		engine, err = r.routeCostOptimized(ctx, req)

	case StrategyLatencyOptimized:
		engine, err = r.routeLatencyOptimized(ctx, req)

	default:
		err = errors.New("ERR_INTERNAL", fmt.Sprintf("unknown routing strategy: %s", r.strategy))
	}

	if err != nil {
		r.recordRoutingMetrics("failed", "", time.Since(startTime))
		return nil, err
	}

	r.recordRoutingMetrics(string(r.strategy), engine.Name(), time.Since(startTime))

	return engine, nil
}

// matchRoutingRule finds the first matching routing rule
func (r *Router) matchRoutingRule(ctx context.Context, req *InferenceRequest) (string, string) {
	r.rulesMu.RLock()
	defer r.rulesMu.RUnlock()

	for _, rule := range r.rules {
		if !rule.Enabled {
			continue
		}

		if r.ruleMatches(ctx, rule, req) {
			return rule.TargetEngine, rule.Name
		}
	}

	return "", ""
}

// ruleMatches checks if a rule matches the request
func (r *Router) ruleMatches(ctx context.Context, rule *RoutingRule, req *InferenceRequest) bool {
	cond := rule.Condition

	// Check model pattern
	if cond.ModelPattern != "" {
		// Simple pattern matching (can be enhanced with regex)
		if req.Model != cond.ModelPattern {
			return false
		}
	}

	// Check latency requirement
	if cond.MaxLatency != nil {
		// Would need to estimate request complexity
		complexity := r.estimateComplexity(req)
		if complexity > 0 {
			// Check if any engine can meet the latency requirement
			canMeetLatency := false
			r.enginesMu.RLock()
			if cfg, exists := r.engines[rule.TargetEngine]; exists {
				if cfg.MaxLatency <= *cond.MaxLatency {
					canMeetLatency = true
				}
			}
			r.enginesMu.RUnlock()

			if !canMeetLatency {
				return false
			}
		}
	}

	// Check cost requirement
	if cond.MaxCostPerToken != nil {
		r.enginesMu.RLock()
		cfg, exists := r.engines[rule.TargetEngine]
		r.enginesMu.RUnlock()

		if exists && cfg.CostPerToken > *cond.MaxCostPerToken {
			return false
		}
	}

	// Check complexity requirements
	complexity := r.estimateComplexity(req)
	if cond.MinComplexity != nil && complexity < *cond.MinComplexity {
		return false
	}
	if cond.MaxComplexity != nil && complexity > *cond.MaxComplexity {
		return false
	}

	// Check required tags
	if len(cond.RequiredTags) > 0 {
		r.enginesMu.RLock()
		cfg, exists := r.engines[rule.TargetEngine]
		r.enginesMu.RUnlock()

		if !exists {
			return false
		}

		for key, value := range cond.RequiredTags {
			if cfg.Tags[key] != value {
				return false
			}
		}
	}

	return true
}

// routeRoundRobin implements round-robin routing
func (r *Router) routeRoundRobin(ctx context.Context) (InferenceEngine, error) {
	engines := r.getAvailableEngines()
	if len(engines) == 0 {
		return nil, errors.New("ERR_INTERNAL", "no available engines")
	}

	r.roundRobinMu.Lock()
	defer r.roundRobinMu.Unlock()

	index := r.roundRobinIndex % len(engines)
	r.roundRobinIndex++

	return engines[index].Engine, nil
}

// routeWeightedRandom implements weighted random routing
func (r *Router) routeWeightedRandom(ctx context.Context) (InferenceEngine, error) {
	engines := r.getAvailableEngines()
	if len(engines) == 0 {
		return nil, errors.New("ERR_INTERNAL", "no available engines")
	}

	// Calculate total weight
	totalWeight := 0
	for _, cfg := range engines {
		totalWeight += cfg.Weight
	}

	if totalWeight == 0 {
		// All weights are zero, fallback to random
		r.randMu.Lock()
		idx := r.rand.Intn(len(engines))
		r.randMu.Unlock()
		return engines[idx].Engine, nil
	}

	// Select based on weight
	r.randMu.Lock()
	randVal := r.rand.Intn(totalWeight)
	r.randMu.Unlock()

	accumulated := 0
	for _, cfg := range engines {
		accumulated += cfg.Weight
		if randVal < accumulated {
			return cfg.Engine, nil
		}
	}

	// Fallback to last engine
	return engines[len(engines)-1].Engine, nil
}

// routeLeastLatency routes to the engine with lowest latency
func (r *Router) routeLeastLatency(ctx context.Context) (InferenceEngine, error) {
	engines := r.getAvailableEngines()
	if len(engines) == 0 {
		return nil, errors.New("ERR_INTERNAL", "no available engines")
	}

	r.metricsMu.RLock()
	defer r.metricsMu.RUnlock()

	var bestEngine *EngineConfig
	var bestLatency time.Duration = time.Hour // Initialize with very high value

	for _, cfg := range engines {
		metrics, exists := r.metrics[cfg.Name]
		if !exists {
			continue
		}

		avgLatency := metrics.AverageLatency
		if avgLatency == 0 || avgLatency < bestLatency {
			bestLatency = avgLatency
			bestEngine = cfg
		}
	}

	if bestEngine == nil {
		// No metrics available, fallback to first engine
		return engines[0].Engine, nil
	}

	return bestEngine.Engine, nil
}

// routeLeastLoad routes to the engine with lowest current load
func (r *Router) routeLeastLoad(ctx context.Context) (InferenceEngine, error) {
	engines := r.getAvailableEngines()
	if len(engines) == 0 {
		return nil, errors.New("ERR_INTERNAL", "no available engines")
	}

	r.metricsMu.RLock()
	defer r.metricsMu.RUnlock()

	var bestEngine *EngineConfig
	bestLoadRatio := 1.0 // 100% load

	for _, cfg := range engines {
		metrics, exists := r.metrics[cfg.Name]
		if !exists {
			continue
		}

		if cfg.MaxConcurrency == 0 {
			continue
		}

		loadRatio := float64(metrics.CurrentLoad) / float64(cfg.MaxConcurrency)
		if loadRatio < bestLoadRatio {
			bestLoadRatio = loadRatio
			bestEngine = cfg
		}
	}

	if bestEngine == nil {
		// No metrics available, fallback to first engine
		return engines[0].Engine, nil
	}

	return bestEngine.Engine, nil
}

// routeABTest routes based on A/B testing configuration
func (r *Router) routeABTest(ctx context.Context, req *InferenceRequest) (InferenceEngine, error) {
	if r.abTestConfig == nil || !r.abTestConfig.Enabled {
		return r.routeWeightedRandom(ctx)
	}

	// Check if test is within time range
	now := time.Now()
	if now.Before(r.abTestConfig.StartTime) || now.After(r.abTestConfig.EndTime) {
		return r.routeWeightedRandom(ctx)
	}

	// Get hash key value from context or request
	var hashValue string
	switch r.abTestConfig.HashKey {
	case "user_id":
		if userID, ok := ctx.Value(types.ContextKeyUserID).(string); ok {
			hashValue = userID
		}
	case "request_id":
		if reqID, ok := ctx.Value(types.ContextKeyRequestID).(string); ok {
			hashValue = reqID
		}
	}

	if hashValue == "" {
		// Fallback to random if no hash key
		return r.routeWeightedRandom(ctx)
	}

	// Hash the value to get consistent routing
	h := fnv.New32a()
	h.Write([]byte(hashValue))
	hashNum := int(h.Sum32())

	// Calculate cumulative percentages
	total := 0
	splits := make(map[string]int)
	for engine, pct := range r.abTestConfig.TrafficSplit {
		total += pct
		splits[engine] = total
	}

	// Normalize to 100
	bucket := hashNum % 100

	for engine, cumulative := range splits {
		if bucket < cumulative {
			return r.getEngine(engine)
		}
	}

	// Fallback
	return r.routeWeightedRandom(ctx)
}

// routeCostOptimized routes based on cost optimization
func (r *Router) routeCostOptimized(ctx context.Context, req *InferenceRequest) (InferenceEngine, error) {
	engines := r.getAvailableEngines()
	if len(engines) == 0 {
		return nil, errors.New("ERR_INTERNAL", "no available engines")
	}

	var bestEngine *EngineConfig
	bestCost := float64(1000000) // Initialize with very high value

	for _, cfg := range engines {
		if cfg.CostPerToken < bestCost {
			bestCost = cfg.CostPerToken
			bestEngine = cfg
		}
	}

	if bestEngine == nil {
		return engines[0].Engine, nil
	}

	return bestEngine.Engine, nil
}

// routeLatencyOptimized routes based on latency requirements
func (r *Router) routeLatencyOptimized(ctx context.Context, req *InferenceRequest) (InferenceEngine, error) {
	engines := r.getAvailableEngines()
	if len(engines) == 0 {
		return nil, errors.New("ERR_INTERNAL", "no available engines")
	}

	// Get latency requirement from context
	requiredLatency, _ := ctx.Value(types.ContextKeyMaxLatency).(time.Duration)

	r.metricsMu.RLock()
	defer r.metricsMu.RUnlock()

	// Filter engines that can meet latency requirement
	var candidates []*EngineConfig
	for _, cfg := range engines {
		metrics, exists := r.metrics[cfg.Name]
		if !exists {
			continue
		}

		if requiredLatency > 0 && metrics.P95Latency > requiredLatency {
			continue
		}

		candidates = append(candidates, cfg)
	}

	if len(candidates) == 0 {
		// No engine meets requirement, return fastest
		return r.routeLeastLatency(ctx)
	}

	// Among candidates, return the one with lowest cost
	var bestEngine *EngineConfig
	bestCost := float64(1000000)

	for _, cfg := range candidates {
		if cfg.CostPerToken < bestCost {
			bestCost = cfg.CostPerToken
			bestEngine = cfg
		}
	}

	return bestEngine.Engine, nil
}

// getEngine retrieves an engine by name
func (r *Router) getEngine(name string) (InferenceEngine, error) {
	r.enginesMu.RLock()
	defer r.enginesMu.RUnlock()

	cfg, exists := r.engines[name]
	if !exists {
		return nil, errors.New(errors.CodeNotFound, fmt.Sprintf("engine %s not found", name))
	}

	if !cfg.Enabled {
		return nil, errors.New("ERR_INTERNAL", fmt.Sprintf("engine %s is disabled", name))
	}

	return cfg.Engine, nil
}

// getAvailableEngines returns all enabled engines
func (r *Router) getAvailableEngines() []*EngineConfig {
	r.enginesMu.RLock()
	defer r.enginesMu.RUnlock()

	engines := make([]*EngineConfig, 0, len(r.engines))
	for _, cfg := range r.engines {
		if cfg.Enabled {
			engines = append(engines, cfg)
		}
	}

	return engines
}

// GetAllEngines returns all registered engines (for health checks)
func (r *Router) GetAllEngines() []InferenceEngine {
	r.enginesMu.RLock()
	defer r.enginesMu.RUnlock()

	engines := make([]InferenceEngine, 0, len(r.engines))
	for _, cfg := range r.engines {
		engines = append(engines, cfg.Engine)
	}

	return engines
}

// UpdateEngineMetrics updates metrics for an engine
func (r *Router) UpdateEngineMetrics(engineName string, metrics *EngineMetrics) {
	r.metricsMu.Lock()
	defer r.metricsMu.Unlock()

	metrics.LastUpdated = time.Now()
	r.metrics[engineName] = metrics
}

// estimateComplexity estimates the complexity of a request
func (r *Router) estimateComplexity(req *InferenceRequest) int {
	// Simple heuristic based on message count and max tokens
	complexity := len(req.Messages) * 10

	if req.MaxTokens > 0 {
		complexity += req.MaxTokens / 100
	}

	// Factor in temperature (higher temperature = more complex)
	if req.Temperature > 0.7 {
		complexity += 20
	}

	return complexity
}

// sortRules sorts routing rules by priority
func (r *Router) sortRules() {
	// Simple bubble sort (efficient for small lists)
	for i := 0; i < len(r.rules)-1; i++ {
		for j := 0; j < len(r.rules)-i-1; j++ {
			if r.rules[j].Priority < r.rules[j+1].Priority {
				r.rules[j], r.rules[j+1] = r.rules[j+1], r.rules[j]
			}
		}
	}
}

// collectMetrics periodically collects metrics from engines
func (r *Router) collectMetrics(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		r.enginesMu.RLock()
		engines := make([]*EngineConfig, 0, len(r.engines))
		for _, cfg := range r.engines {
			engines = append(engines, cfg)
		}
		r.enginesMu.RUnlock()

		for _, cfg := range engines {
			r.metricsCollector.RecordGauge(
				"router_engine_enabled",
				func() float64 {
					if cfg.Enabled {
						return 1
					}
					return 0
				}(),
				map[string]string{"engine": cfg.Name},
			)
		}
	}
}

// recordRoutingMetrics records routing operation metrics
func (r *Router) recordRoutingMetrics(strategy, engine string, latency time.Duration) {
	if r.metricsCollector == nil {
		return
	}

	r.metricsCollector.IncrementCounter("router_route_total", 1,
		map[string]string{
			"strategy": strategy,
			"engine":   engine,
		})

	r.metricsCollector.RecordDuration("router_route_latency_ms", float64(latency.Milliseconds()),
		map[string]string{
			"strategy": strategy,
		})
}

//Personal.AI order the ending
