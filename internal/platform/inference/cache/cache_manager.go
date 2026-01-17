// Package cache provides a three-tier caching system for inference requests
// with L1 local cache, L2 Redis cache, and L3 vector similarity cache.
package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/pkg/errors"
)

// CacheLevel represents the cache tier level
type CacheLevel string

const (
	// LevelL1 represents local in-memory cache
	LevelL1 CacheLevel = "L1"

	// LevelL2 represents Redis distributed cache
	LevelL2 CacheLevel = "L2"

	// LevelL3 represents vector similarity cache
	LevelL3 CacheLevel = "L3"
)

// CacheEntry represents a cached inference response
type CacheEntry struct {
	Key        string                 `json:"key"`
	Request    interface{}            `json:"request"`
	Response   interface{}            `json:"response"`
	Level      CacheLevel             `json:"level"`
	Similarity float64                `json:"similarity,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	ExpiresAt  time.Time              `json:"expires_at"`
	AccessCount int64                 `json:"access_count"`
	LastAccess time.Time              `json:"last_access"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// CacheConfig contains configuration for the cache manager
type CacheConfig struct {
	L1Enabled          bool
	L1TTL              time.Duration
	L1MaxSize          int

	L2Enabled          bool
	L2TTL              time.Duration
	L2KeyPrefix        string

	L3Enabled          bool
	L3TTL              time.Duration
	L3SimilarityThreshold float64
	L3MaxResults       int

	EnableMetrics      bool
	EnableAdaptiveTTL  bool
	MinTTL             time.Duration
	MaxTTL             time.Duration
}

// CacheStats contains cache statistics
type CacheStats struct {
	L1Hits     int64
	L1Misses   int64
	L2Hits     int64
	L2Misses   int64
	L3Hits     int64
	L3Misses   int64
	TotalHits  int64
	TotalMisses int64
	HitRate    float64
	AvgLatency time.Duration
}

// Cache defines the interface for cache implementations
type Cache interface {
	// Get retrieves a value from cache
	Get(ctx context.Context, key string) (*CacheEntry, error)

	// Set stores a value in cache
	Set(ctx context.Context, entry *CacheEntry) error

	// Delete removes a value from cache
	Delete(ctx context.Context, key string) error

	// Clear removes all entries from cache
	Clear(ctx context.Context) error

	// Stats returns cache statistics
	Stats(ctx context.Context) (*CacheStats, error)

	// Name returns the cache name
	Name() string

	// Level returns the cache level
	Level() CacheLevel
}

// CacheManager manages the three-tier cache hierarchy
type CacheManager struct {
	logger           logging.Logger
	metricsCollector *metrics.MetricsCollector
	config           *CacheConfig

	l1Cache          Cache
	l2Cache          Cache
	l3Cache          Cache

	stats            *CacheStats
}

// NewCacheManager creates a new cache manager
func NewCacheManager(
	logger logging.Logger,
	metricsCollector *metrics.MetricsCollector,
	config *CacheConfig,
	l1Cache Cache,
	l2Cache Cache,
	l3Cache Cache,
) *CacheManager {
	return &CacheManager{
		logger:           logger,
		metricsCollector: metricsCollector,
		config:           config,
		l1Cache:          l1Cache,
		l2Cache:          l2Cache,
		l3Cache:          l3Cache,
		stats:            &CacheStats{},
	}
}

// Get attempts to retrieve a cached response through the cache hierarchy
func (cm *CacheManager) Get(ctx context.Context, req interface{}) (interface{}, error) {
	startTime := time.Now()

	// Generate cache key
	key, err := cm.generateCacheKey(req)
	if err != nil {
		return nil, errors.WrapInternalError(err, "ERR_INTERNAL", "failed to generate cache key")
	}

	// Try L1 cache first (fastest, < 1ms)
	if cm.config.L1Enabled && cm.l1Cache != nil {
		entry, err := cm.l1Cache.Get(ctx, key)
		if err == nil && entry != nil {
			cm.recordCacheHit(LevelL1, time.Since(startTime))
			cm.logger.WithContext(ctx).Debug("L1 cache hit", logging.Any("key", key))

			// Update access statistics
			cm.updateAccessStats(ctx, entry, LevelL1)

			return entry.Response, nil
		}
	}

	// Try L2 cache (moderate speed, < 10ms)
	if cm.config.L2Enabled && cm.l2Cache != nil {
		entry, err := cm.l2Cache.Get(ctx, key)
		if err == nil && entry != nil {
			cm.recordCacheHit(LevelL2, time.Since(startTime))
			cm.logger.WithContext(ctx).Debug("L2 cache hit", logging.Any("key", key))

			// Promote to L1 for faster future access
			if cm.config.L1Enabled && cm.l1Cache != nil {
				l1Entry := *entry
				l1Entry.Level = LevelL1
				l1Entry.ExpiresAt = time.Now().Add(cm.config.L1TTL)

				if err := cm.l1Cache.Set(ctx, &l1Entry); err != nil {
					cm.logger.WithContext(ctx).Warn("failed to promote to L1", logging.Error(err))
				}
			}

			cm.updateAccessStats(ctx, entry, LevelL2)
			return entry.Response, nil
		}
	}

	// Try L3 vector similarity cache (slower, < 50ms)
	if cm.config.L3Enabled && cm.l3Cache != nil {
		entry, err := cm.l3Cache.Get(ctx, key)
		if err == nil && entry != nil {
			// Check similarity threshold
			if entry.Similarity >= cm.config.L3SimilarityThreshold {
				cm.recordCacheHit(LevelL3, time.Since(startTime))
    cm.logger.WithContext(ctx).Debug("L3 cache hit", logging.Any("key", key), logging.Any("similarity", entry.Similarity))

				// Promote to L2 and L1
				if cm.config.L2Enabled && cm.l2Cache != nil {
					l2Entry := *entry
					l2Entry.Level = LevelL2
					l2Entry.ExpiresAt = time.Now().Add(cm.config.L2TTL)
					l2Entry.Key = key // Use exact key for L2

					if err := cm.l2Cache.Set(ctx, &l2Entry); err != nil {
						cm.logger.WithContext(ctx).Warn("failed to promote to L2", logging.Error(err))
					}
				}

				if cm.config.L1Enabled && cm.l1Cache != nil {
					l1Entry := *entry
					l1Entry.Level = LevelL1
					l1Entry.ExpiresAt = time.Now().Add(cm.config.L1TTL)
					l1Entry.Key = key

					if err := cm.l1Cache.Set(ctx, &l1Entry); err != nil {
						cm.logger.WithContext(ctx).Warn("failed to promote to L1", logging.Error(err))
					}
				}

				cm.updateAccessStats(ctx, entry, LevelL3)
				return entry.Response, nil
			}
		}
	}

	// Cache miss
	cm.recordCacheMiss(time.Since(startTime))
	return nil, nil
}

// Set stores a response in all enabled cache levels
func (cm *CacheManager) Set(ctx context.Context, req interface{}, resp interface{}) error {
	startTime := time.Now()

	// Generate cache key
	key, err := cm.generateCacheKey(req)
	if err != nil {
		return errors.WrapInternalError(err, "ERR_INTERNAL", "failed to generate cache key")
	}

	now := time.Now()

	// Calculate adaptive TTL based on request characteristics
	ttl := cm.calculateAdaptiveTTL(ctx, req, resp)

	// Store in L3 first (for vector similarity)
	if cm.config.L3Enabled && cm.l3Cache != nil {
		l3Entry := &CacheEntry{
			Key:        key,
			Request:    req,
			Response:   resp,
			Level:      LevelL3,
			CreatedAt:  now,
			ExpiresAt:  now.Add(ttl.L3),
			AccessCount: 0,
			LastAccess: now,
		}

		if err := cm.l3Cache.Set(ctx, l3Entry); err != nil {
			cm.logger.WithContext(ctx).Warn("failed to set L3 cache", logging.Error(err))
		} else {
			cm.logger.WithContext(ctx).Debug("L3 cache set", logging.Any("key", key), logging.Any("ttl", ttl.L3))
		}
	}

	// Store in L2 (distributed cache)
	if cm.config.L2Enabled && cm.l2Cache != nil {
		l2Entry := &CacheEntry{
			Key:        key,
			Request:    req,
			Response:   resp,
			Level:      LevelL2,
			CreatedAt:  now,
			ExpiresAt:  now.Add(ttl.L2),
			AccessCount: 0,
			LastAccess: now,
		}

		if err := cm.l2Cache.Set(ctx, l2Entry); err != nil {
			cm.logger.WithContext(ctx).Warn("failed to set L2 cache", logging.Error(err))
		} else {
			cm.logger.WithContext(ctx).Debug("L2 cache set", logging.Any("key", key), logging.Any("ttl", ttl.L2))
		}
	}

	// Store in L1 (local cache)
	if cm.config.L1Enabled && cm.l1Cache != nil {
		l1Entry := &CacheEntry{
			Key:        key,
			Request:    req,
			Response:   resp,
			Level:      LevelL1,
			CreatedAt:  now,
			ExpiresAt:  now.Add(ttl.L1),
			AccessCount: 0,
			LastAccess: now,
		}

		if err := cm.l1Cache.Set(ctx, l1Entry); err != nil {
			cm.logger.WithContext(ctx).Warn("failed to set L1 cache", logging.Error(err))
		} else {
			cm.logger.WithContext(ctx).Debug("L1 cache set", logging.Any("key", key), logging.Any("ttl", ttl.L1))
		}
	}

	cm.recordCacheSet(time.Since(startTime))

	return nil
}

// Invalidate removes entries from all cache levels
func (cm *CacheManager) Invalidate(ctx context.Context, req interface{}) error {
	key, err := cm.generateCacheKey(req)
	if err != nil {
		return errors.WrapInternalError(err, "ERR_INTERNAL", "failed to generate cache key")
	}

	var errs []error

	if cm.config.L1Enabled && cm.l1Cache != nil {
		if err := cm.l1Cache.Delete(ctx, key); err != nil {
			errs = append(errs, err)
		}
	}

	if cm.config.L2Enabled && cm.l2Cache != nil {
		if err := cm.l2Cache.Delete(ctx, key); err != nil {
			errs = append(errs, err)
		}
	}

	if cm.config.L3Enabled && cm.l3Cache != nil {
		if err := cm.l3Cache.Delete(ctx, key); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
  cm.logger.WithContext(ctx).Warn("cache invalidation had errors", logging.Any("key", key), logging.Any("error_count", len(errs)))
	}

	cm.logger.WithContext(ctx).Info("cache invalidated", logging.Any("key", key))

	return nil
}

// InvalidatePattern invalidates all entries matching a pattern
func (cm *CacheManager) InvalidatePattern(ctx context.Context, pattern string) error {
	cm.logger.WithContext(ctx).Info("invalidating cache pattern", logging.Any("pattern", pattern))

	// This would require cache implementations to support pattern matching
	// For now, just clear all caches
	return cm.ClearAll(ctx)
}

// ClearAll clears all cache levels
func (cm *CacheManager) ClearAll(ctx context.Context) error {
	var errs []error

	if cm.config.L1Enabled && cm.l1Cache != nil {
		if err := cm.l1Cache.Clear(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if cm.config.L2Enabled && cm.l2Cache != nil {
		if err := cm.l2Cache.Clear(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if cm.config.L3Enabled && cm.l3Cache != nil {
		if err := cm.l3Cache.Clear(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.New("ERR_INTERNAL", fmt.Sprintf("failed to clear %d cache levels", len(errs)))
	}

	cm.logger.WithContext(ctx).Info("all caches cleared")

	return nil
}

// GetStats returns aggregated cache statistics
func (cm *CacheManager) GetStats(ctx context.Context) (*CacheStats, error) {
	aggregated := &CacheStats{
		L1Hits:   cm.stats.L1Hits,
		L1Misses: cm.stats.L1Misses,
		L2Hits:   cm.stats.L2Hits,
		L2Misses: cm.stats.L2Misses,
		L3Hits:   cm.stats.L3Hits,
		L3Misses: cm.stats.L3Misses,
	}

	aggregated.TotalHits = aggregated.L1Hits + aggregated.L2Hits + aggregated.L3Hits
	aggregated.TotalMisses = aggregated.L1Misses + aggregated.L2Misses + aggregated.L3Misses

	total := aggregated.TotalHits + aggregated.TotalMisses
	if total > 0 {
		aggregated.HitRate = float64(aggregated.TotalHits) / float64(total)
	}

	return aggregated, nil
}

// generateCacheKey generates a unique cache key from a request
func (cm *CacheManager) generateCacheKey(req interface{}) (string, error) {
	// Serialize request to JSON
	data, err := json.Marshal(req)
	if err != nil {
		return "", errors.WrapInternalError(err, "ERR_INTERNAL", "failed to marshal request")
	}

	// Generate SHA-256 hash
	hash := sha256.Sum256(data)
	key := hex.EncodeToString(hash[:])

	return key, nil
}

// TTLSet contains TTL values for each cache level
type TTLSet struct {
	L1 time.Duration
	L2 time.Duration
	L3 time.Duration
}

// calculateAdaptiveTTL calculates adaptive TTL based on request/response characteristics
func (cm *CacheManager) calculateAdaptiveTTL(ctx context.Context, req interface{}, resp interface{}) TTLSet {
	if !cm.config.EnableAdaptiveTTL {
		return TTLSet{
			L1: cm.config.L1TTL,
			L2: cm.config.L2TTL,
			L3: cm.config.L3TTL,
		}
	}

	// Base TTL
	baseTTL := cm.config.L2TTL

	// Factor 1: Request complexity (longer responses get longer TTL)
	complexityFactor := 1.0
	if respData, err := json.Marshal(resp); err == nil {
		responseSize := len(respData)
		if responseSize > 10000 {
			complexityFactor = 2.0 // Large responses
		} else if responseSize > 5000 {
			complexityFactor = 1.5
		}
	}

	// Factor 2: Time of day (cache longer during peak hours)
	timeFactor := 1.0
	hour := time.Now().Hour()
	if hour >= 9 && hour <= 17 {
		timeFactor = 1.5 // Business hours
	}

	// Calculate final TTL
	calculatedTTL := time.Duration(float64(baseTTL) * complexityFactor * timeFactor)

	// Clamp to min/max
	if calculatedTTL < cm.config.MinTTL {
		calculatedTTL = cm.config.MinTTL
	}
	if calculatedTTL > cm.config.MaxTTL {
		calculatedTTL = cm.config.MaxTTL
	}

	return TTLSet{
		L1: calculatedTTL / 4,  // L1 is shorter-lived
		L2: calculatedTTL,
		L3: calculatedTTL * 2,  // L3 is longer-lived
	}
}

// updateAccessStats updates access statistics for a cache entry
func (cm *CacheManager) updateAccessStats(ctx context.Context, entry *CacheEntry, level CacheLevel) {
	entry.AccessCount++
	entry.LastAccess = time.Now()

	// Update in the cache (best effort, don't fail if update fails)
	switch level {
	case LevelL1:
		if cm.l1Cache != nil {
			_ = cm.l1Cache.Set(ctx, entry)
		}
	case LevelL2:
		if cm.l2Cache != nil {
			_ = cm.l2Cache.Set(ctx, entry)
		}
	case LevelL3:
		if cm.l3Cache != nil {
			_ = cm.l3Cache.Set(ctx, entry)
		}
	}
}

// recordCacheHit records a cache hit metric
func (cm *CacheManager) recordCacheHit(level CacheLevel, latency time.Duration) {
	switch level {
	case LevelL1:
		cm.stats.L1Hits++
	case LevelL2:
		cm.stats.L2Hits++
	case LevelL3:
		cm.stats.L3Hits++
	}

	if cm.metricsCollector != nil {
		cm.metricsCollector.IncrementCounter("cache_hits_total",
			map[string]string{
				"level": string(level),
			})

		cm.metricsCollector.ObserveDuration("cache_latency_ms", float64(latency.Milliseconds()),
			map[string]string{
				"level":  string(level),
				"result": "hit",
			})
	}
}

// recordCacheMiss records a cache miss metric
func (cm *CacheManager) recordCacheMiss(latency time.Duration) {
	cm.stats.L1Misses++
	cm.stats.L2Misses++
	cm.stats.L3Misses++

	if cm.metricsCollector != nil {
		cm.metricsCollector.IncrementCounter("cache_misses_total", nil)

		cm.metricsCollector.ObserveDuration("cache_latency_ms", float64(latency.Milliseconds()),
			map[string]string{
				"result": "miss",
			})
	}
}

// recordCacheSet records a cache set operation metric
func (cm *CacheManager) recordCacheSet(latency time.Duration) {
	if cm.metricsCollector != nil {
		cm.metricsCollector.IncrementCounter("cache_sets_total", nil)

		cm.metricsCollector.ObserveDuration("cache_set_latency_ms", float64(latency.Milliseconds()), nil)
	}
}

//Personal.AI order the ending
