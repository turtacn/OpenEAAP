// Package knowledge provides repository interfaces for knowledge persistence.
// It defines DocumentRepository and ChunkRepository interfaces with CRUD operations,
// vector search, and knowledge management capabilities.
package knowledge

import (
	"context"
	"time"
)

// ============================================================================
// Document Repository Interface
// ============================================================================

// DocumentRepository defines the interface for document persistence operations
type DocumentRepository interface {
	// Create creates a new document
	Create(ctx context.Context, document *Document) error

	// GetByID retrieves a document by ID
	GetByID(ctx context.Context, id string) (*Document, error)

	// GetByKnowledgeBaseID retrieves documents by knowledge base ID
	GetByKnowledgeBaseID(ctx context.Context, kbID string, filter DocumentFilter) ([]*Document, error)

	// Update updates an existing document
	Update(ctx context.Context, document *Document) error

	// Delete deletes a document (hard delete)
	Delete(ctx context.Context, id string) error

	// SoftDelete soft deletes a document
	SoftDelete(ctx context.Context, id string) error

	// Restore restores a soft-deleted document
	Restore(ctx context.Context, id string) error

	// List retrieves documents with pagination and filters
	List(ctx context.Context, filter DocumentFilter) ([]*Document, error)

	// Count returns the total count of documents matching the filter
	Count(ctx context.Context, filter DocumentFilter) (int64, error)

	// Exists checks if a document exists
	Exists(ctx context.Context, id string) (bool, error)

	// ExistsByHash checks if a document with the given content hash exists
	ExistsByHash(ctx context.Context, kbID, hash string) (bool, error)

	// GetByContentHash retrieves a document by content hash
	GetByContentHash(ctx context.Context, kbID, hash string) (*Document, error)

	// GetByStatus retrieves documents by status
	GetByStatus(ctx context.Context, kbID string, status DocumentStatus, filter DocumentFilter) ([]*Document, error)

	// GetByType retrieves documents by type
	GetByType(ctx context.Context, kbID string, docType DocumentType, filter DocumentFilter) ([]*Document, error)

	// GetByTags retrieves documents by tags
	GetByTags(ctx context.Context, kbID string, tags []string, filter DocumentFilter) ([]*Document, error)

	// GetByCategory retrieves documents by category
	GetByCategory(ctx context.Context, kbID string, category string, filter DocumentFilter) ([]*Document, error)

	// GetBySensitivity retrieves documents by sensitivity level
	GetBySensitivity(ctx context.Context, kbID string, sensitivity SensitivityLevel, filter DocumentFilter) ([]*Document, error)

	// GetExpired retrieves expired documents
	GetExpired(ctx context.Context, kbID string) ([]*Document, error)

	// GetPendingProcessing retrieves documents pending processing
	GetPendingProcessing(ctx context.Context, kbID string, limit int) ([]*Document, error)

	// GetPendingEmbedding retrieves documents pending embedding
	GetPendingEmbedding(ctx context.Context, kbID string, limit int) ([]*Document, error)

	// Search searches documents by query
	Search(ctx context.Context, kbID string, query SearchQuery) ([]*Document, error)

	// FullTextSearch performs full-text search
	FullTextSearch(ctx context.Context, kbID string, query string, filter DocumentFilter) ([]*Document, error)

	// UpdateStatus updates document status
	UpdateStatus(ctx context.Context, id string, status DocumentStatus) error

	// UpdateEmbeddingStatus updates embedding status
	UpdateEmbeddingStatus(ctx context.Context, id string, status EmbeddingStatus) error

	// UpdateProcessingError updates processing error
	UpdateProcessingError(ctx context.Context, id string, err string) error

	// UpdateMetadata updates document metadata
	UpdateMetadata(ctx context.Context, id string, metadata DocumentMetadata) error

	// UpdateStats updates document statistics
	UpdateStats(ctx context.Context, id string, stats DocumentStats) error

	// RecordAccess records document access
	RecordAccess(ctx context.Context, id string) error

	// AddTag adds a tag to a document
	AddTag(ctx context.Context, id string, tag string) error

	// RemoveTag removes a tag from a document
	RemoveTag(ctx context.Context, id string, tag string) error

	// AddCategory adds a category to a document
	AddCategory(ctx context.Context, id string, category string) error

	// RemoveCategory removes a category from a document
	RemoveCategory(ctx context.Context, id string, category string) error

	// LinkRelatedDocument links related documents
	LinkRelatedDocument(ctx context.Context, id string, relatedID string) error

	// UnlinkRelatedDocument unlinks related documents
	UnlinkRelatedDocument(ctx context.Context, id string, relatedID string) error

	// BatchCreate creates multiple documents
	BatchCreate(ctx context.Context, documents []*Document) error

	// BatchUpdate updates multiple documents
	BatchUpdate(ctx context.Context, documents []*Document) error

	// BatchDelete deletes multiple documents
	BatchDelete(ctx context.Context, ids []string) error

	// GetStatistics retrieves aggregate statistics
	GetStatistics(ctx context.Context, kbID string, filter DocumentFilter) (*DocumentStatistics, error)

	// GetRecentlyAccessed retrieves recently accessed documents
	GetRecentlyAccessed(ctx context.Context, kbID string, limit int) ([]*Document, error)

	// GetMostViewed retrieves most viewed documents
	GetMostViewed(ctx context.Context, kbID string, limit int) ([]*Document, error)

	// GetByOwner retrieves documents by owner
	GetByOwner(ctx context.Context, ownerID string, filter DocumentFilter) ([]*Document, error)

	// GetVersions retrieves all versions of a document
	GetVersions(ctx context.Context, parentDocumentID string) ([]*Document, error)

	// GetLatestVersion retrieves the latest version of a document
	GetLatestVersion(ctx context.Context, parentDocumentID string) (*Document, error)
}

// ============================================================================
// Chunk Repository Interface
// ============================================================================

// ChunkRepository defines the interface for chunk persistence operations
type ChunkRepository interface {
	// Create creates a new chunk
	Create(ctx context.Context, chunk *Chunk) error

	// GetByID retrieves a chunk by ID
	GetByID(ctx context.Context, id string) (*Chunk, error)

	// GetByDocumentID retrieves chunks by document ID
	GetByDocumentID(ctx context.Context, documentID string) ([]*Chunk, error)

	// GetByKnowledgeBaseID retrieves chunks by knowledge base ID
	GetByKnowledgeBaseID(ctx context.Context, kbID string, filter ChunkFilter) ([]*Chunk, error)

	// Update updates an existing chunk
	Update(ctx context.Context, chunk *Chunk) error

	// Delete deletes a chunk
	Delete(ctx context.Context, id string) error

	// DeleteByDocumentID deletes all chunks for a document
	DeleteByDocumentID(ctx context.Context, documentID string) error

	// List retrieves chunks with pagination and filters
	List(ctx context.Context, filter ChunkFilter) ([]*Chunk, error)

	// Count returns the total count of chunks matching the filter
	Count(ctx context.Context, filter ChunkFilter) (int64, error)

	// Exists checks if a chunk exists
	Exists(ctx context.Context, id string) (bool, error)

	// GetPendingIndexing retrieves chunks pending indexing
	GetPendingIndexing(ctx context.Context, kbID string, limit int) ([]*Chunk, error)

	// GetByIndexRange retrieves chunks by index range
	GetByIndexRange(ctx context.Context, documentID string, startIndex, endIndex int) ([]*Chunk, error)

	// UpdateEmbedding updates chunk embedding
	UpdateEmbedding(ctx context.Context, id string, embedding []float64, model string) error

	// MarkIndexed marks chunk as indexed
	MarkIndexed(ctx context.Context, id string) error

	// BatchCreate creates multiple chunks
	BatchCreate(ctx context.Context, chunks []*Chunk) error

	// BatchUpdate updates multiple chunks
	BatchUpdate(ctx context.Context, chunks []*Chunk) error

	// BatchDelete deletes multiple chunks
	BatchDelete(ctx context.Context, ids []string) error

	// VectorSearch performs vector similarity search
	VectorSearch(ctx context.Context, kbID string, vector []float64, limit int, filter ChunkFilter) ([]*Chunk, error)

	// HybridSearch performs hybrid search (vector + keyword)
	HybridSearch(ctx context.Context, kbID string, query HybridSearchQuery) ([]*Chunk, error)

	// GetNeighborChunks retrieves neighboring chunks
	GetNeighborChunks(ctx context.Context, chunkID string, before, after int) ([]*Chunk, error)

	// GetStatistics retrieves chunk statistics
	GetStatistics(ctx context.Context, kbID string) (*ChunkStatistics, error)
}

// ============================================================================
// Embedding Repository Interface
// ============================================================================

// EmbeddingRepository defines the interface for embedding persistence operations
type EmbeddingRepository interface {
	// Create creates a new embedding
	Create(ctx context.Context, embedding *Embedding) error

	// GetByID retrieves an embedding by ID
	GetByID(ctx context.Context, id string) (*Embedding, error)

	// GetByEntityID retrieves embeddings by entity ID
	GetByEntityID(ctx context.Context, entityID string, entityType EmbeddingEntityType) ([]*Embedding, error)

	// Update updates an existing embedding
	Update(ctx context.Context, embedding *Embedding) error

	// Delete deletes an embedding
	Delete(ctx context.Context, id string) error

	// DeleteByEntityID deletes embeddings by entity ID
	DeleteByEntityID(ctx context.Context, entityID string, entityType EmbeddingEntityType) error

	// List retrieves embeddings with filters
	List(ctx context.Context, filter EmbeddingFilter) ([]*Embedding, error)

	// Count returns the total count of embeddings
	Count(ctx context.Context, filter EmbeddingFilter) (int64, error)

	// Exists checks if an embedding exists
	Exists(ctx context.Context, id string) (bool, error)

	// SimilaritySearch performs similarity search
	SimilaritySearch(ctx context.Context, kbID string, vector []float64, limit int, threshold float64) ([]*Embedding, error)

	// BatchCreate creates multiple embeddings
	BatchCreate(ctx context.Context, embeddings []*Embedding) error

	// BatchDelete deletes multiple embeddings
	BatchDelete(ctx context.Context, ids []string) error

	// GetExpired retrieves expired embeddings
	GetExpired(ctx context.Context) ([]*Embedding, error)

	// DeleteExpired deletes expired embeddings
	DeleteExpired(ctx context.Context) error

	// GetByModel retrieves embeddings by model
	GetByModel(ctx context.Context, kbID string, model string) ([]*Embedding, error)

	// UpdateVector updates embedding vector
	UpdateVector(ctx context.Context, id string, vector []float64) error
}

// ============================================================================
// Filter Types
// ============================================================================

// DocumentFilter defines filtering options for document queries
type DocumentFilter struct {
	// Pagination
	Limit  int
	Offset int

	// Sorting
	SortBy    string
	SortOrder SortOrder

	// Status filter
	Status []DocumentStatus

	// Type filter
	Type []DocumentType

	// Sensitivity filter
	Sensitivity []SensitivityLevel

	// Tags filter
	Tags    []string
	TagMode TagFilterMode

	// Categories filter
	Categories []string

	// Date range filters
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	UpdatedAfter  *time.Time
	UpdatedBefore *time.Time

	// Include deleted
	IncludeDeleted bool

	// Only expired
	OnlyExpired bool

	// Owner filter
	OwnerID string

	// Creator filter
	CreatorID string

	// Language filter
	Languages []string

	// Quality score filter
	MinQualityScore *float64

	// Search query
	SearchQuery string

	// Embedding status filter
	EmbeddingStatus []EmbeddingStatus

	// Access control
	UserID     string
	UserGroups []string
}

// ChunkFilter defines filtering options for chunk queries
type ChunkFilter struct {
	// Pagination
	Limit  int
	Offset int

	// Sorting
	SortBy    string
	SortOrder SortOrder

	// Document ID filter
	DocumentID string

	// Date range filters
	CreatedAfter  *time.Time
	CreatedBefore *time.Time

	// Only indexed
	OnlyIndexed bool

	// Has embedding
	HasEmbedding bool

	// Language filter
	Language string

	// Min quality score
	MinQualityScore *float64

	// Token count range
	MinTokenCount *int
	MaxTokenCount *int

	// Embedding model filter
	EmbeddingModel string
}

// EmbeddingFilter defines filtering options for embedding queries
type EmbeddingFilter struct {
	// Pagination
	Limit  int
	Offset int

	// Entity type filter
	EntityType EmbeddingEntityType

	// Model filter
	Model string

	// Date range filters
	CreatedAfter  *time.Time
	CreatedBefore *time.Time

	// Include expired
	IncludeExpired bool

	// Dimension filter
	Dimension int
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
	// TagFilterModeAny matches documents with any of the tags
	TagFilterModeAny TagFilterMode = "any"

	// TagFilterModeAll matches documents with all of the tags
	TagFilterModeAll TagFilterMode = "all"
)

// ============================================================================
// Search Query Types
// ============================================================================

// SearchQuery represents a document search query
type SearchQuery struct {
	// Query text
	Query string

	// Search fields
	Fields []string

	// Filters
	Filter DocumentFilter

	// Highlight options
	Highlight bool

	// Fuzzy matching
	Fuzzy bool

	// Boost factors
	BoostFactors map[string]float64
}

// HybridSearchQuery represents a hybrid search query
type HybridSearchQuery struct {
	// Query text for keyword search
	Query string

	// Vector for similarity search
	Vector []float64

	// Weight for keyword search (0-1)
	KeywordWeight float64

	// Weight for vector search (0-1)
	VectorWeight float64

	// Minimum similarity threshold
	MinSimilarity float64

	// Filters
	Filter ChunkFilter

	// Limit
	Limit int
}

// ============================================================================
// Statistics Types
// ============================================================================

// DocumentStatistics represents document statistics
type DocumentStatistics struct {
	// Total documents
	TotalDocuments int64

	// Documents by status
	ByStatus map[DocumentStatus]int64

	// Documents by type
	ByType map[DocumentType]int64

	// Documents by sensitivity
	BySensitivity map[SensitivityLevel]int64

	// Documents by language
	ByLanguage map[string]int64

	// Total size in bytes
	TotalSizeBytes int64

	// Total word count
	TotalWordCount int64

	// Total view count
	TotalViewCount int64

	// Average quality score
	AverageQualityScore float64

	// Most viewed documents
	MostViewed []*Document

	// Recently added
	RecentlyAdded []*Document

	// Last updated
	LastUpdated time.Time
}

// ChunkStatistics represents chunk statistics
type ChunkStatistics struct {
	// Total chunks
	TotalChunks int64

	// Indexed chunks
	IndexedChunks int64

	// Chunks with embeddings
	ChunksWithEmbeddings int64

	// Average tokens per chunk
	AverageTokens float64

	// Average quality score
	AverageQualityScore float64

	// Total chunks by document
	TotalByDocument map[string]int64

	// Last updated
	LastUpdated time.Time
}

// ============================================================================
// Filter Builder
// ============================================================================

// NewDocumentFilter creates a new document filter with default values
func NewDocumentFilter() DocumentFilter {
	return DocumentFilter{
		Limit:           50,
		Offset:          0,
		SortBy:          "created_at",
		SortOrder:       SortOrderDesc,
		Status:          make([]DocumentStatus, 0),
		Type:            make([]DocumentType, 0),
		Sensitivity:     make([]SensitivityLevel, 0),
		Tags:            make([]string, 0),
		TagMode:         TagFilterModeAny,
		Categories:      make([]string, 0),
		Languages:       make([]string, 0),
		EmbeddingStatus: make([]EmbeddingStatus, 0),
		IncludeDeleted:  false,
		OnlyExpired:     false,
	}
}

// WithLimit sets the limit
func (f DocumentFilter) WithLimit(limit int) DocumentFilter {
	f.Limit = limit
	return f
}

// WithOffset sets the offset
func (f DocumentFilter) WithOffset(offset int) DocumentFilter {
	f.Offset = offset
	return f
}

// WithSorting sets sorting options
func (f DocumentFilter) WithSorting(sortBy string, sortOrder SortOrder) DocumentFilter {
	f.SortBy = sortBy
	f.SortOrder = sortOrder
	return f
}

// WithStatus adds status filter
func (f DocumentFilter) WithStatus(status ...DocumentStatus) DocumentFilter {
	f.Status = append(f.Status, status...)
	return f
}

// WithType adds type filter
func (f DocumentFilter) WithType(docType ...DocumentType) DocumentFilter {
	f.Type = append(f.Type, docType...)
	return f
}

// WithSensitivity adds sensitivity filter
func (f DocumentFilter) WithSensitivity(sensitivity ...SensitivityLevel) DocumentFilter {
	f.Sensitivity = append(f.Sensitivity, sensitivity...)
	return f
}

// WithTags sets tags filter
func (f DocumentFilter) WithTags(mode TagFilterMode, tags ...string) DocumentFilter {
	f.Tags = tags
	f.TagMode = mode
	return f
}

// WithCategories adds categories filter
func (f DocumentFilter) WithCategories(categories ...string) DocumentFilter {
	f.Categories = append(f.Categories, categories...)
	return f
}

// WithOwner sets owner filter
func (f DocumentFilter) WithOwner(ownerID string) DocumentFilter {
	f.OwnerID = ownerID
	return f
}

// WithLanguages adds language filter
func (f DocumentFilter) WithLanguages(languages ...string) DocumentFilter {
	f.Languages = append(f.Languages, languages...)
	return f
}

// WithSearchQuery sets search query
func (f DocumentFilter) WithSearchQuery(query string) DocumentFilter {
	f.SearchQuery = query
	return f
}

// NewChunkFilter creates a new chunk filter with default values
func NewChunkFilter() ChunkFilter {
	return ChunkFilter{
		Limit:     50,
		Offset:    0,
		SortBy:    "index",
		SortOrder: SortOrderAsc,
	}
}

// WithDocumentID sets document ID filter
func (f ChunkFilter) WithDocumentID(documentID string) ChunkFilter {
	f.DocumentID = documentID
	return f
}

// WithOnlyIndexed sets only indexed filter
func (f ChunkFilter) WithOnlyIndexed(only bool) ChunkFilter {
	f.OnlyIndexed = only
	return f
}

// WithHasEmbedding sets has embedding filter
func (f ChunkFilter) WithHasEmbedding(has bool) ChunkFilter {
	f.HasEmbedding = has
	return f
}

// ============================================================================
// Transaction Support
// ============================================================================

// TransactionalDocumentRepository extends DocumentRepository with transaction support
type TransactionalDocumentRepository interface {
	DocumentRepository

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

// CachedDocumentRepository extends DocumentRepository with caching capabilities
type CachedDocumentRepository interface {
	DocumentRepository

	// InvalidateCache invalidates cache for specific document
	InvalidateCache(ctx context.Context, id string) error

	// InvalidateAllCache invalidates all document caches
	InvalidateAllCache(ctx context.Context) error

	// RefreshCache refreshes cache for specific document
	RefreshCache(ctx context.Context, id string) error

	// WarmupCache preloads frequently accessed documents
	WarmupCache(ctx context.Context, ids []string) error
}

// ============================================================================
// Batch Operations
// ============================================================================

// BatchOperation represents a batch operation result
type BatchOperation struct {
	// Success count
	SuccessCount int

	// Failed count
	FailedCount int

	// Failed IDs
	FailedIDs []string

	// Error messages
	Errors map[string]string
}

//Personal.AI order the ending
