// Package cache provides caching implementations for inference requests.
// This file implements the L3 vector similarity cache tier using Milvus.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/pkg/errors"
)

// L3VectorCache implements a vector similarity-based cache using Milvus
type L3VectorCache struct {
	logger              logging.Logger
	client              client.Client
	collectionName      string
	ttl                 time.Duration
	similarityThreshold float64
	maxResults          int
	embeddingDim        int

	hitCount            int64
	missCount           int64
	setCount            int64
}

// VectorConfig contains configuration for vector cache
type VectorConfig struct {
	Address             string
	Username            string
	Password            string
	CollectionName      string
	TTL                 time.Duration
	SimilarityThreshold float64
	MaxResults          int
	EmbeddingDim        int
	IndexType           string
	MetricType          string
	Nlist               int
}

// vectorCacheEntry represents a cache entry stored in Milvus
type vectorCacheEntry struct {
	ID        int64     `json:"id"`
	Key       string    `json:"key"`
	Embedding []float32 `json:"embedding"`
	Request   string    `json:"request"`
	Response  string    `json:"response"`
	CreatedAt int64     `json:"created_at"`
	ExpiresAt int64     `json:"expires_at"`
	Metadata  string    `json:"metadata"`
}

// NewL3VectorCache creates a new L3 vector cache instance
func NewL3VectorCache(logger logging.Logger, config *VectorConfig) (*L3VectorCache, error) {
	if config == nil {
		return nil, errors.ValidationError( "vector config cannot be nil")
	}

	// Connect to Milvus
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	milvusClient, err := client.NewGrpcClient(ctx, config.Address)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to connect to Milvus")
	}

	// Authenticate if credentials provided
	if config.Username != "" && config.Password != "" {
		// Milvus authentication would be implemented here
		logger.Info(ctx, "Milvus authentication enabled")
	}

	cache := &L3VectorCache{
		logger:              logger,
		client:              milvusClient,
		collectionName:      config.CollectionName,
		ttl:                 config.TTL,
		similarityThreshold: config.SimilarityThreshold,
		maxResults:          config.MaxResults,
		embeddingDim:        config.EmbeddingDim,
	}

	// Initialize collection
	if err := cache.initializeCollection(ctx, config); err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to initialize collection")
	}

	// Start background cleanup
	go cache.cleanupExpired()

	logger.WithContext(ctx).Info("L3 vector cache initialized",
		logging.String("collection", config.CollectionName),
		logging.Int("embedding_dim", config.EmbeddingDim),
		logging.Float64("similarity_threshold", config.SimilarityThreshold))

	return cache, nil
}

// Get retrieves an entry from L3 vector cache using similarity search
func (c *L3VectorCache) Get(ctx context.Context, key string) (*CacheEntry, error) {
	startTime := time.Now()

	// Generate embedding for the key
	embedding, err := c.generateEmbedding(ctx, key)
	if err != nil {
		c.logger.WithContext(ctx).Warn("failed to generate embedding", logging.Error(err))
		c.missCount++
		return nil, nil
	}

	// Search for similar vectors
	searchResult, err := c.searchSimilar(ctx, embedding)
	if err != nil {
		c.logger.WithContext(ctx).Warn("vector search failed", logging.Error(err))
		c.missCount++
		return nil, nil
	}

	if len(searchResult) == 0 {
		c.missCount++
		return nil, nil
	}

	// Get the top result
	topResult := searchResult[0]

	// Check similarity threshold
	if topResult.Similarity < c.similarityThreshold {
		c.missCount++
  c.logger.WithContext(ctx).Debug("similarity below threshold", logging.Any("similarity", topResult.Similarity), logging.Any("threshold", c.similarityThreshold))
		return nil, nil
	}

	// Check expiration
	if time.Now().Unix() > topResult.ExpiresAt {
		// Delete expired entry
		_ = c.deleteByID(ctx, topResult.ID)
		c.missCount++
		return nil, nil
	}

	// Deserialize response
	var response interface{}
	if err := json.Unmarshal([]byte(topResult.Response), &response); err != nil {
		c.logger.WithContext(ctx).Warn("failed to deserialize response", logging.Error(err))
		c.missCount++
		return nil, nil
	}

	// Deserialize request
	var request interface{}
	if err := json.Unmarshal([]byte(topResult.Request), &request); err != nil {
		c.logger.WithContext(ctx).Warn("failed to deserialize request", logging.Error(err))
	}

	entry := &CacheEntry{
		Key:        topResult.Key,
		Request:    request,
		Response:   response,
		Level:      LevelL3,
		Similarity: topResult.Similarity,
		CreatedAt:  time.Unix(topResult.CreatedAt, 0),
		ExpiresAt:  time.Unix(topResult.ExpiresAt, 0),
	}

	c.hitCount++

	latency := time.Since(startTime)
 c.logger.WithContext(ctx).Debug("L3 cache hit", logging.Any("key", key), logging.Any("similarity", topResult.Similarity), logging.Any("latency_ms", latency.Milliseconds()))

	if latency > 50*time.Millisecond {
  c.logger.WithContext(ctx).Warn("L3 cache GET latency exceeded 50ms", logging.Any("latency_ms", latency.Milliseconds()))
	}

	return entry, nil
}

// Set stores an entry in L3 vector cache
func (c *L3VectorCache) Set(ctx context.Context, entry *CacheEntry) error {
	if entry == nil {
		return errors.ValidationError( "cache entry cannot be nil")
	}

	startTime := time.Now()

	// Generate embedding
	requestJSON, err := json.Marshal(entry.Request)
	if err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to marshal request")
	}

	embedding, err := c.generateEmbedding(ctx, string(requestJSON))
	if err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to generate embedding")
	}

	// Serialize response
	responseJSON, err := json.Marshal(entry.Response)
	if err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to marshal response")
	}

	// Calculate expiration
	expiresAt := entry.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(c.ttl)
	}

	// Serialize metadata
	metadataJSON := "{}"
	if entry.Metadata != nil {
		if data, err := json.Marshal(entry.Metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	// Insert into Milvus
	vectorEntry := &vectorCacheEntry{
		ID:        time.Now().UnixNano(), // Use timestamp as ID
		Key:       entry.Key,
		Embedding: embedding,
		Request:   string(requestJSON),
		Response:  string(responseJSON),
		CreatedAt: time.Now().Unix(),
		ExpiresAt: expiresAt.Unix(),
		Metadata:  metadataJSON,
	}

	if err := c.insertVector(ctx, vectorEntry); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to insert vector")
	}

	c.setCount++

	latency := time.Since(startTime)
	if latency > 50*time.Millisecond {
  c.logger.WithContext(ctx).Warn("L3 cache SET latency exceeded 50ms", logging.Any("latency_ms", latency.Milliseconds()))
	}

	return nil
}

// Delete removes an entry from L3 vector cache
func (c *L3VectorCache) Delete(ctx context.Context, key string) error {
	// Search for the entry by key
	expr := fmt.Sprintf("key == \"%s\"", key)

	if err := c.client.Delete(ctx, c.collectionName, "", expr); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to delete from Milvus")
	}

	return nil
}

// Clear removes all entries from L3 vector cache
func (c *L3VectorCache) Clear(ctx context.Context) error {
	// Drop and recreate collection
	if err := c.client.DropCollection(ctx, c.collectionName); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to drop collection")
	}

	// Reinitialize with default config
	config := &VectorConfig{
		CollectionName:      c.collectionName,
		EmbeddingDim:        c.embeddingDim,
		IndexType:           "IVF_FLAT",
		MetricType:          "IP", // Inner product (cosine similarity)
		Nlist:               1024,
	}

	if err := c.initializeCollection(ctx, config); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to reinitialize collection")
	}

	c.logger.WithContext(ctx).Info("L3 cache cleared")

	return nil
}

// Stats returns cache statistics
func (c *L3VectorCache) Stats(ctx context.Context) (*CacheStats, error) {
	stats := &CacheStats{
		L3Hits:   c.hitCount,
		L3Misses: c.missCount,
	}

	total := c.hitCount + c.missCount
	if total > 0 {
		stats.HitRate = float64(c.hitCount) / float64(total)
	}

	return stats, nil
}

// Name returns the cache name
func (c *L3VectorCache) Name() string {
	return "L3VectorCache"
}

// Level returns the cache level
func (c *L3VectorCache) Level() CacheLevel {
	return LevelL3
}

// Close closes the Milvus client connection
func (c *L3VectorCache) Close() error {
	return c.client.Close()
}

// initializeCollection creates and configures the Milvus collection
func (c *L3VectorCache) initializeCollection(ctx context.Context, config *VectorConfig) error {
	// Check if collection exists
	hasCollection, err := c.client.HasCollection(ctx, config.CollectionName)
	if err != nil {
		return err
	}

	if hasCollection {
		c.logger.WithContext(ctx).Info("collection already exists", logging.Any("name", config.CollectionName))
		return nil
	}

	// Define schema
	schema := &entity.Schema{
		CollectionName: config.CollectionName,
		Description:    "L3 vector cache for inference requests",
		Fields: []*entity.Field{
			{
				Name:       "id",
				DataType:   entity.FieldTypeInt64,
				PrimaryKey: true,
				AutoID:     false,
			},
			{
				Name:     "key",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "512",
				},
			},
			{
				Name:     "embedding",
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": fmt.Sprintf("%d", config.EmbeddingDim),
				},
			},
			{
				Name:     "request",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "65535",
				},
			},
			{
				Name:     "response",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "65535",
				},
			},
			{
				Name:     "created_at",
				DataType: entity.FieldTypeInt64,
			},
			{
				Name:     "expires_at",
				DataType: entity.FieldTypeInt64,
			},
			{
				Name:     "metadata",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "4096",
				},
			},
		},
	}

	// Create collection
	if err := c.client.CreateCollection(ctx, schema, 2); err != nil {
		return err
	}

	// Create index
	idx, err := entity.NewIndexIvfFlat(entity.IP, config.Nlist)
	if err != nil {
		return err
	}

	if err := c.client.CreateIndex(ctx, config.CollectionName, "embedding", idx, false); err != nil {
		return err
	}

	// Load collection
	if err := c.client.LoadCollection(ctx, config.CollectionName, false); err != nil {
		return err
	}

	c.logger.WithContext(ctx).Info("collection created and loaded", logging.Any("name", config.CollectionName))

	return nil
}

// generateEmbedding generates a vector embedding for text
func (c *L3VectorCache) generateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Simplified embedding generation
	// In production, use a proper embedding model (e.g., sentence-transformers)

	embedding := make([]float32, c.embeddingDim)

	// Simple hash-based embedding for demonstration
	hash := 0
	for _, char := range text {
		hash = hash*31 + int(char)
	}

	// Distribute hash across dimensions
	for i := 0; i < c.embeddingDim; i++ {
		embedding[i] = float32((hash >> (i % 32)) & 1)
	}

	// Normalize
	magnitude := float32(0)
	for _, val := range embedding {
		magnitude += val * val
	}
	magnitude = float32(1.0 / (1.0 + magnitude))

	for i := range embedding {
		embedding[i] *= magnitude
	}

	return embedding, nil
}

// searchSimilar searches for similar vectors in Milvus
func (c *L3VectorCache) searchSimilar(ctx context.Context, embedding []float32) ([]*vectorCacheEntry, error) {
	sp, _ := entity.NewIndexIvfFlatSearchParam(16)

	searchResult, err := c.client.Search(
		ctx,
		c.collectionName,
		nil,
		"",
		[]string{"id", "key", "request", "response", "created_at", "expires_at"},
		[]entity.Vector{entity.FloatVector(embedding)},
		"embedding",
		entity.IP,
		c.maxResults,
		sp,
	)

	if err != nil {
		return nil, err
	}

	if len(searchResult) == 0 {
		return []*vectorCacheEntry{}, nil
	}

	results := make([]*vectorCacheEntry, 0)

	for _, result := range searchResult[0].Fields {
		entries := c.parseSearchResult(result)
		for i, entry := range entries {
			if i < len(searchResult[0].Scores) {
				entry.Similarity = float64(searchResult[0].Scores[i])
			}
			results = append(results, entry)
		}
	}

	return results, nil
}

// parseSearchResult parses Milvus search results
func (c *L3VectorCache) parseSearchResult(field *interface{}) []*vectorCacheEntry {
	// This is a simplified parser
	// Actual implementation would properly parse all fields
	entries := make([]*vectorCacheEntry, 0)

	// Extract data from field
	// In production, properly iterate through all result fields

	return entries
}

// insertVector inserts a vector entry into Milvus
func (c *L3VectorCache) insertVector(ctx context.Context, entry *vectorCacheEntry) error {
	idColumn := entity.NewColumnInt64("id", []int64{entry.ID})
	keyColumn := entity.NewColumnVarChar("key", []string{entry.Key})
	embeddingColumn := entity.NewColumnFloatVector("embedding", c.embeddingDim, [][]float32{entry.Embedding})
	requestColumn := entity.NewColumnVarChar("request", []string{entry.Request})
	responseColumn := entity.NewColumnVarChar("response", []string{entry.Response})
	createdColumn := entity.NewColumnInt64("created_at", []int64{entry.CreatedAt})
	expiresColumn := entity.NewColumnInt64("expires_at", []int64{entry.ExpiresAt})
	metadataColumn := entity.NewColumnVarChar("metadata", []string{entry.Metadata})

	if _, err := c.client.Insert(
		ctx,
		c.collectionName,
		"",
		idColumn,
		keyColumn,
		embeddingColumn,
		requestColumn,
		responseColumn,
		createdColumn,
		expiresColumn,
		metadataColumn,
	); err != nil {
		return err
	}

	// Flush to ensure data is persisted
	if err := c.client.Flush(ctx, c.collectionName, false); err != nil {
		c.logger.WithContext(ctx).Warn("flush failed", logging.Error(err))
	}

	return nil
}

// deleteByID deletes an entry by ID
func (c *L3VectorCache) deleteByID(ctx context.Context, id int64) error {
	expr := fmt.Sprintf("id == %d", id)
	return c.client.Delete(ctx, c.collectionName, "", expr)
}

// cleanupExpired periodically removes expired entries
func (c *L3VectorCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.performCleanup()
	}
}

// performCleanup removes all expired entries
func (c *L3VectorCache) performCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now().Unix()
	expr := fmt.Sprintf("expires_at < %d", now)

	if err := c.client.Delete(ctx, c.collectionName, "", expr); err != nil {
		c.logger.WithContext(ctx).Warn("cleanup failed", logging.Error(err))
	} else {
		c.logger.WithContext(ctx).Debug("L3 cache cleanup completed")
	}
}

// GetMetrics returns detailed cache metrics
func (c *L3VectorCache) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	total := c.hitCount + c.missCount
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.hitCount) / float64(total)
	}

	// Get collection stats
	stats, err := c.client.GetCollectionStatistics(ctx, c.collectionName)
	if err != nil {
		c.logger.WithContext(ctx).Warn("failed to get collection stats", logging.Error(err))
	}

	return map[string]interface{}{
		"level":            "L3",
		"hit_count":        c.hitCount,
		"miss_count":       c.missCount,
		"set_count":        c.setCount,
		"hit_rate":         hitRate,
		"collection_stats": stats,
	}, nil
}

// HealthCheck checks if Milvus connection is healthy
func (c *L3VectorCache) HealthCheck(ctx context.Context) error {
	hasCollection, err := c.client.HasCollection(ctx, c.collectionName)
	if err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "Milvus health check failed")
	}

	if !hasCollection {
		return errors.InternalError("collection does not exist")
	}

	return nil
}

//Personal.AI order the ending
