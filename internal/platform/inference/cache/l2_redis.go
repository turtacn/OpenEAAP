// Package cache provides caching implementations for inference requests.
// This file implements the L2 Redis distributed cache tier with SimHash support.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sangfor.local/hci/hci-common/utils/redis"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/pkg/errors"
)

// L2RedisCache implements a Redis-based distributed cache with SimHash matching
type L2RedisCache struct {
	logger       logging.Logger
	client       *redis.Client
	keyPrefix    string
	ttl          time.Duration
	simHashBits  int

	hitCount     int64
	missCount    int64
	setCount     int64
}

// RedisConfig contains configuration for Redis cache
type RedisConfig struct {
	Address      string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	KeyPrefix    string
	TTL          time.Duration
	SimHashBits  int
}

// NewL2RedisCache creates a new L2 Redis cache instance
func NewL2RedisCache(logger logging.Logger, config *RedisConfig) (*L2RedisCache, error) {
	if config == nil {
		return nil, errors.New(errors.CodeInvalidArgument, "redis config cannot be nil")
	}

	client := redis.NewClient(&redis.Options{
		Addr:         config.Address,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to connect to Redis")
	}

	simHashBits := config.SimHashBits
	if simHashBits == 0 {
		simHashBits = 64 // Default to 64-bit SimHash
	}

	cache := &L2RedisCache{
		logger:      logger,
		client:      client,
		keyPrefix:   config.KeyPrefix,
		ttl:         config.TTL,
		simHashBits: simHashBits,
	}

	logger.Info(context.Background(), "L2 Redis cache initialized",
		"address", config.Address,
		"key_prefix", config.KeyPrefix,
		"ttl", config.TTL,
	)

	return cache, nil
}

// Get retrieves an entry from L2 Redis cache
func (c *L2RedisCache) Get(ctx context.Context, key string) (*CacheEntry, error) {
	startTime := time.Now()

	redisKey := c.buildRedisKey(key)

	// Try exact match first
	data, err := c.client.Get(ctx, redisKey).Bytes()
	if err == nil {
		entry, err := c.deserializeEntry(data)
		if err != nil {
			c.logger.Warn(ctx, "failed to deserialize cache entry", "error", err)
			c.missCount++
			return nil, nil
		}

		// Check expiration
		if time.Now().After(entry.ExpiresAt) {
			_ = c.client.Del(ctx, redisKey)
			c.missCount++
			return nil, nil
		}

		c.hitCount++

		latency := time.Since(startTime)
		c.logger.Debug(ctx, "L2 cache exact hit",
			"key", key,
			"latency_ms", latency.Milliseconds(),
		)

		return entry, nil
	}

	if err != redis.Nil {
		c.logger.Warn(ctx, "Redis GET error", "error", err)
	}

	// Try SimHash fuzzy matching
	entry, err := c.findBySimilarity(ctx, key)
	if err != nil {
		c.logger.Warn(ctx, "SimHash matching failed", "error", err)
		c.missCount++
		return nil, nil
	}

	if entry != nil {
		c.hitCount++

		latency := time.Since(startTime)
		c.logger.Debug(ctx, "L2 cache SimHash hit",
			"key", key,
			"latency_ms", latency.Milliseconds(),
		)

		return entry, nil
	}

	c.missCount++

	latency := time.Since(startTime)
	if latency > 10*time.Millisecond {
		c.logger.Warn(ctx, "L2 cache GET latency exceeded 10ms",
			"latency_ms", latency.Milliseconds(),
		)
	}

	return nil, nil
}

// Set stores an entry in L2 Redis cache
func (c *L2RedisCache) Set(ctx context.Context, entry *CacheEntry) error {
	if entry == nil {
		return errors.New(errors.CodeInvalidArgument, "cache entry cannot be nil")
	}

	startTime := time.Now()

	// Create a copy
	entryCopy := *entry
	entryCopy.Level = LevelL2

	// Calculate TTL
	ttl := c.ttl
	if !entryCopy.ExpiresAt.IsZero() {
		ttl = time.Until(entryCopy.ExpiresAt)
		if ttl <= 0 {
			return errors.New(errors.CodeInvalidArgument, "entry already expired")
		}
	} else {
		entryCopy.ExpiresAt = time.Now().Add(ttl)
	}

	// Serialize entry
	data, err := c.serializeEntry(&entryCopy)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to serialize entry")
	}

	redisKey := c.buildRedisKey(entry.Key)

	// Store in Redis with TTL
	if err := c.client.Set(ctx, redisKey, data, ttl).Err(); err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to set Redis cache")
	}

	// Store SimHash mapping for fuzzy matching
	if err := c.storeSimHashMapping(ctx, entry.Key, &entryCopy, ttl); err != nil {
		c.logger.Warn(ctx, "failed to store SimHash mapping", "error", err)
		// Don't fail the entire operation
	}

	c.setCount++

	latency := time.Since(startTime)
	if latency > 10*time.Millisecond {
		c.logger.Warn(ctx, "L2 cache SET latency exceeded 10ms",
			"latency_ms", latency.Milliseconds(),
		)
	}

	return nil
}

// Delete removes an entry from L2 Redis cache
func (c *L2RedisCache) Delete(ctx context.Context, key string) error {
	redisKey := c.buildRedisKey(key)

	// Delete exact match
	if err := c.client.Del(ctx, redisKey).Err(); err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to delete from Redis")
	}

	// Delete SimHash mapping
	simHashKey := c.buildSimHashKey(key)
	_ = c.client.Del(ctx, simHashKey)

	return nil
}

// Clear removes all entries from L2 Redis cache
func (c *L2RedisCache) Clear(ctx context.Context) error {
	pattern := c.keyPrefix + "*"

	iter := c.client.Scan(ctx, 0, pattern, 1000).Iterator()
	keys := make([]string, 0)

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to scan Redis keys")
	}

	if len(keys) > 0 {
		if err := c.client.Del(ctx, keys...).Err(); err != nil {
			return errors.Wrap(err, errors.CodeInternalError, "failed to delete Redis keys")
		}
	}

	c.logger.Info(ctx, "L2 cache cleared", "deleted_keys", len(keys))

	return nil
}

// Stats returns cache statistics
func (c *L2RedisCache) Stats(ctx context.Context) (*CacheStats, error) {
	stats := &CacheStats{
		L2Hits:   c.hitCount,
		L2Misses: c.missCount,
	}

	total := c.hitCount + c.missCount
	if total > 0 {
		stats.HitRate = float64(c.hitCount) / float64(total)
	}

	return stats, nil
}

// Name returns the cache name
func (c *L2RedisCache) Name() string {
	return "L2RedisCache"
}

// Level returns the cache level
func (c *L2RedisCache) Level() CacheLevel {
	return LevelL2
}

// Close closes the Redis client connection
func (c *L2RedisCache) Close() error {
	return c.client.Close()
}

// buildRedisKey constructs the full Redis key with prefix
func (c *L2RedisCache) buildRedisKey(key string) string {
	return fmt.Sprintf("%s:exact:%s", c.keyPrefix, key)
}

// buildSimHashKey constructs the SimHash Redis key
func (c *L2RedisCache) buildSimHashKey(key string) string {
	return fmt.Sprintf("%s:simhash:%s", c.keyPrefix, key)
}

// serializeEntry serializes a cache entry to JSON
func (c *L2RedisCache) serializeEntry(entry *CacheEntry) ([]byte, error) {
	return json.Marshal(entry)
}

// deserializeEntry deserializes a cache entry from JSON
func (c *L2RedisCache) deserializeEntry(data []byte) (*CacheEntry, error) {
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// storeSimHashMapping stores a SimHash mapping for fuzzy matching
func (c *L2RedisCache) storeSimHashMapping(ctx context.Context, key string, entry *CacheEntry, ttl time.Duration) error {
	// Generate SimHash from request
	requestData, err := json.Marshal(entry.Request)
	if err != nil {
		return err
	}

	simHash := c.computeSimHash(string(requestData))

	// Store mapping: simhash -> key
	simHashKey := c.buildSimHashKey(fmt.Sprintf("%d", simHash))

	// Use ZADD with score as simhash value for range queries
	if err := c.client.ZAdd(ctx, c.keyPrefix+":simhash:index", &redis.Z{
		Score:  float64(simHash),
		Member: key,
	}).Err(); err != nil {
		return err
	}

	// Set expiration on the simhash key
	if err := c.client.Expire(ctx, simHashKey, ttl).Err(); err != nil {
		return err
	}

	return nil
}

// findBySimilarity finds a cache entry by SimHash similarity
func (c *L2RedisCache) findBySimilarity(ctx context.Context, key string) (*CacheEntry, error) {
	// This is a simplified implementation
	// In production, you'd use more sophisticated SimHash matching

	// Generate SimHash for the query key
	simHash := c.computeSimHash(key)

	// Define similarity threshold (Hamming distance)
	maxHammingDistance := 3

	// Search for similar hashes in a range
	minScore := float64(simHash - int64(maxHammingDistance))
	maxScore := float64(simHash + int64(maxHammingDistance))

	members, err := c.client.ZRangeByScore(ctx, c.keyPrefix+":simhash:index", &redis.ZRangeBy{
		Min: fmt.Sprintf("%f", minScore),
		Max: fmt.Sprintf("%f", maxScore),
	}).Result()

	if err != nil {
		return nil, err
	}

	// Find the best match
	for _, member := range members {
		redisKey := c.buildRedisKey(member)
		data, err := c.client.Get(ctx, redisKey).Bytes()
		if err != nil {
			continue
		}

		entry, err := c.deserializeEntry(data)
		if err != nil {
			continue
		}

		// Verify it hasn't expired
		if time.Now().After(entry.ExpiresAt) {
			_ = c.client.Del(ctx, redisKey)
			continue
		}

		// Calculate actual similarity
		similarity := c.calculateSimilarity(key, member)
		entry.Similarity = similarity

		// Return if similarity is high enough (> 0.95)
		if similarity > 0.95 {
			return entry, nil
		}
	}

	return nil, nil
}

// computeSimHash computes a 64-bit SimHash for a string
func (c *L2RedisCache) computeSimHash(text string) int64 {
	// Simplified SimHash implementation
	// In production, use a proper SimHash library

	// Tokenize the text
	tokens := c.tokenize(text)

	// Initialize bit vector
	v := make([]int, c.simHashBits)

	// Process each token
	for _, token := range tokens {
		// Hash the token
		hash := c.hashToken(token)

		// Update bit vector
		for i := 0; i < c.simHashBits; i++ {
			bit := (hash >> i) & 1
			if bit == 1 {
				v[i]++
			} else {
				v[i]--
			}
		}
	}

	// Generate SimHash
	var simHash int64
	for i := 0; i < c.simHashBits; i++ {
		if v[i] > 0 {
			simHash |= (1 << i)
		}
	}

	return simHash
}

// tokenize splits text into tokens
func (c *L2RedisCache) tokenize(text string) []string {
	// Simple whitespace tokenization
	// In production, use a proper tokenizer
	tokens := make([]string, 0)
	current := ""

	for _, char := range text {
		if char == ' ' || char == '\n' || char == '\t' {
			if current != "" {
				tokens = append(tokens, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		tokens = append(tokens, current)
	}

	return tokens
}

// hashToken computes a hash for a token
func (c *L2RedisCache) hashToken(token string) int64 {
	// Simple FNV-1a hash
	var hash int64 = 2166136261

	for _, char := range token {
		hash ^= int64(char)
		hash *= 16777619
	}

	return hash
}

// calculateSimilarity calculates cosine similarity between two keys
func (c *L2RedisCache) calculateSimilarity(key1, key2 string) float64 {
	// Compute SimHashes
	hash1 := c.computeSimHash(key1)
	hash2 := c.computeSimHash(key2)

	// Calculate Hamming distance
	xor := hash1 ^ hash2
	hammingDistance := c.countSetBits(xor)

	// Convert to similarity score (0 to 1)
	similarity := 1.0 - (float64(hammingDistance) / float64(c.simHashBits))

	return similarity
}

// countSetBits counts the number of set bits in an integer
func (c *L2RedisCache) countSetBits(n int64) int {
	count := 0
	for n != 0 {
		count += int(n & 1)
		n >>= 1
	}
	return count
}

// GetMetrics returns detailed cache metrics
func (c *L2RedisCache) GetMetrics(ctx context.Context) map[string]interface{} {
	total := c.hitCount + c.missCount
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.hitCount) / float64(total)
	}

	// Get Redis info
	info := c.client.Info(ctx, "stats").Val()

	return map[string]interface{}{
		"level":      "L2",
		"hit_count":  c.hitCount,
		"miss_count": c.missCount,
		"set_count":  c.setCount,
		"hit_rate":   hitRate,
		"redis_info": info,
	}
}

// HealthCheck checks if Redis connection is healthy
func (c *L2RedisCache) HealthCheck(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "Redis health check failed")
	}
	return nil
}

//Personal.AI order the ending
