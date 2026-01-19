// Package cache provides caching implementations for inference requests.
// This file implements the L1 local in-memory cache tier.
package cache

import (
	"context"
	"sync"
	"time"

	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/pkg/errors"
)

// L1LocalCache implements a local in-memory cache using LRU eviction
type L1LocalCache struct {
	logger    logging.Logger
	maxSize   int
	ttl       time.Duration

	entries   map[string]*l1CacheNode
	entryMu   sync.RWMutex

	head      *l1CacheNode
	tail      *l1CacheNode

	hitCount  int64
	missCount int64
	setCount  int64
}

// l1CacheNode represents a node in the LRU doubly-linked list
type l1CacheNode struct {
	key       string
	entry     *CacheEntry
	prev      *l1CacheNode
	next      *l1CacheNode
	expiresAt time.Time
}

// NewL1LocalCache creates a new L1 local cache instance
func NewL1LocalCache(logger logging.Logger, maxSize int, ttl time.Duration) *L1LocalCache {
	cache := &L1LocalCache{
		logger:  logger,
		maxSize: maxSize,
		ttl:     ttl,
		entries: make(map[string]*l1CacheNode, maxSize),
	}

	// Initialize doubly-linked list with sentinel nodes
	cache.head = &l1CacheNode{}
	cache.tail = &l1CacheNode{}
	cache.head.next = cache.tail
	cache.tail.prev = cache.head

	// Start background cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// Get retrieves an entry from L1 cache
func (c *L1LocalCache) Get(ctx context.Context, key string) (*CacheEntry, error) {
	c.entryMu.Lock()
	defer c.entryMu.Unlock()

	node, exists := c.entries[key]
	if !exists {
		c.missCount++
		return nil, nil
	}

	// Check expiration
	if time.Now().After(node.expiresAt) {
		c.removeNode(node)
		delete(c.entries, key)
		c.missCount++
		return nil, nil
	}

	// Move to front (most recently used)
	c.moveToFront(node)

	c.hitCount++

	// Return a copy to prevent external modifications
	entryCopy := *node.entry
	return &entryCopy, nil
}

// Set stores an entry in L1 cache
func (c *L1LocalCache) Set(ctx context.Context, entry *CacheEntry) error {
	if entry == nil {
		return errors.ValidationError( "cache entry cannot be nil")
	}

	c.entryMu.Lock()
	defer c.entryMu.Unlock()

	// Create a copy to prevent external modifications
	entryCopy := *entry
	entryCopy.Level = LevelL1

	// Calculate expiration time
	expiresAt := entryCopy.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(c.ttl)
		entryCopy.ExpiresAt = expiresAt
	}

	key := entry.Key

	// Check if key already exists
	if node, exists := c.entries[key]; exists {
		// Update existing node
		node.entry = &entryCopy
		node.expiresAt = expiresAt
		c.moveToFront(node)
	} else {
		// Create new node
		newNode := &l1CacheNode{
			key:       key,
			entry:     &entryCopy,
			expiresAt: expiresAt,
		}

		// Add to front of list
		c.addToFront(newNode)
		c.entries[key] = newNode

		// Evict if necessary
		if len(c.entries) > c.maxSize {
			c.evictLRU()
		}
	}

	c.setCount++

	return nil
}

// Delete removes an entry from L1 cache
func (c *L1LocalCache) Delete(ctx context.Context, key string) error {
	c.entryMu.Lock()
	defer c.entryMu.Unlock()

	node, exists := c.entries[key]
	if !exists {
		return nil
	}

	c.removeNode(node)
	delete(c.entries, key)

	return nil
}

// Clear removes all entries from L1 cache
func (c *L1LocalCache) Clear(ctx context.Context) error {
	c.entryMu.Lock()
	defer c.entryMu.Unlock()

	// Reset map
	c.entries = make(map[string]*l1CacheNode, c.maxSize)

	// Reset linked list
	c.head.next = c.tail
	c.tail.prev = c.head

	c.logger.WithContext(ctx).Info("L1 cache cleared")

	return nil
}

// Stats returns cache statistics
func (c *L1LocalCache) Stats(ctx context.Context) (*CacheStats, error) {
	c.entryMu.RLock()
	defer c.entryMu.RUnlock()

	stats := &CacheStats{
		L1Hits:   c.hitCount,
		L1Misses: c.missCount,
	}

	total := c.hitCount + c.missCount
	if total > 0 {
		stats.HitRate = float64(c.hitCount) / float64(total)
	}

	return stats, nil
}

// Name returns the cache name
func (c *L1LocalCache) Name() string {
	return "L1LocalCache"
}

// Level returns the cache level
func (c *L1LocalCache) Level() CacheLevel {
	return LevelL1
}

// Size returns the current number of entries in cache
func (c *L1LocalCache) Size() int {
	c.entryMu.RLock()
	defer c.entryMu.RUnlock()
	return len(c.entries)
}

// moveToFront moves a node to the front of the LRU list
func (c *L1LocalCache) moveToFront(node *l1CacheNode) {
	if node == c.head.next {
		// Already at front
		return
	}

	c.removeNode(node)
	c.addToFront(node)
}

// addToFront adds a node to the front of the LRU list
func (c *L1LocalCache) addToFront(node *l1CacheNode) {
	node.prev = c.head
	node.next = c.head.next
	c.head.next.prev = node
	c.head.next = node
}

// removeNode removes a node from the LRU list
func (c *L1LocalCache) removeNode(node *l1CacheNode) {
	if node.prev != nil {
		node.prev.next = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	}
}

// evictLRU evicts the least recently used entry
func (c *L1LocalCache) evictLRU() {
	if c.tail.prev == c.head {
		// List is empty
		return
	}

	lruNode := c.tail.prev
	c.removeNode(lruNode)
	delete(c.entries, lruNode.key)

	c.logger.Debug("L1 cache evicted LRU entry", logging.String("key", lruNode.key))
}

// cleanupExpired periodically removes expired entries
func (c *L1LocalCache) cleanupExpired() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.performCleanup()
	}
}

// performCleanup removes all expired entries
func (c *L1LocalCache) performCleanup() {
	c.entryMu.Lock()
	defer c.entryMu.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	// Traverse from tail (least recently used)
	current := c.tail.prev
	for current != c.head {
		if now.After(current.expiresAt) {
			expiredKeys = append(expiredKeys, current.key)
		}
		current = current.prev
	}

	// Remove expired entries
	for _, key := range expiredKeys {
		if node, exists := c.entries[key]; exists {
			c.removeNode(node)
			delete(c.entries, key)
		}
	}

	if len(expiredKeys) > 0 {
		c.logger.Debug("L1 cache cleanup completed",
			logging.Int("expired_count", len(expiredKeys)),
			logging.Int("remaining_count", len(c.entries)))
	}
}

// GetMetrics returns detailed cache metrics
func (c *L1LocalCache) GetMetrics() map[string]interface{} {
	c.entryMu.RLock()
	defer c.entryMu.RUnlock()

	total := c.hitCount + c.missCount
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.hitCount) / float64(total)
	}

	return map[string]interface{}{
		"level":       "L1",
		"size":        len(c.entries),
		"max_size":    c.maxSize,
		"hit_count":   c.hitCount,
		"miss_count":  c.missCount,
		"set_count":   c.setCount,
		"hit_rate":    hitRate,
		"utilization": float64(len(c.entries)) / float64(c.maxSize),
	}
}

// Warmup pre-populates the cache with frequently accessed entries
func (c *L1LocalCache) Warmup(ctx context.Context, entries []*CacheEntry) error {
	for _, entry := range entries {
		if err := c.Set(ctx, entry); err != nil {
			c.logger.WithContext(ctx).Warn("failed to warmup entry", logging.Any("key", entry.Key), logging.Error(err))
		}
	}

	c.logger.WithContext(ctx).Info("L1 cache warmup completed", logging.Any("count", len(entries)))

	return nil
}

// Snapshot returns a snapshot of all current cache entries
// func (c *L1LocalCache) Snapshot() []*CacheEntry {
// 	c.entryMu.RLock()
// 	defer c.entryMu.RUnlock()

// 	snapshot := make([]*CacheEntry, 0, len(c.entries))

// 	current := c.head.next
// 	for current != c.tail {
// 		entryCopy := *current.entry
// 		snapshot = append(snapshot, &entryCopy)
// 		current = current.next
// 	}
//
// 	return snapshot
// }
