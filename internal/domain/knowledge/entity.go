// Package knowledge provides domain entities for knowledge management.
// It defines Document, Chunk, Embedding, and related structures for
// organizing, indexing, and retrieving knowledge base content.
package knowledge

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Document Entity
// ============================================================================

// Document represents a knowledge document entity
type Document struct {
	// Unique identifier
	ID string `json:"id"`

	// Knowledge base ID
	KnowledgeBaseID string `json:"knowledge_base_id"`

	// Document title
	Title string `json:"title"`

	// Document content
	Content string `json:"content"`

	// Document type (text, pdf, markdown, html, etc.)
	Type DocumentType `json:"type"`

	// Source information
	Source DocumentSource `json:"source"`

	// Document metadata
	Metadata DocumentMetadata `json:"metadata"`

	// Language code (en, zh, es, etc.)
	Language string `json:"language"`

	// Content hash for deduplication
	ContentHash string `json:"content_hash"`

	// File size in bytes
	SizeBytes int64 `json:"size_bytes"`

	// Word count
	WordCount int `json:"word_count"`

	// Character count
	CharCount int `json:"char_count"`

	// Sensitivity level
	Sensitivity SensitivityLevel `json:"sensitivity"`

	// Access control
	AccessControl AccessControl `json:"access_control"`

	// Processing status
	Status DocumentStatus `json:"status"`

	// Processing error if any
	ProcessingError string `json:"processing_error,omitempty"`

	// Tags for categorization
	Tags []string `json:"tags"`

	// Categories
	Categories []string `json:"categories"`

	// Related document IDs
	RelatedDocuments []string `json:"related_documents"`

	// Version number
	Version int `json:"version"`

	// Parent document ID (for versions)
	ParentDocumentID string `json:"parent_document_id,omitempty"`

	// Chunks associated with this document
	ChunkCount int `json:"chunk_count"`

	// Embedding status
	EmbeddingStatus EmbeddingStatus `json:"embedding_status"`

	// Quality score (0-1)
	QualityScore float64 `json:"quality_score"`

	// Relevance score for search results
	RelevanceScore float64 `json:"relevance_score,omitempty"`

	// Statistics
	Stats DocumentStats `json:"stats"`

	// Owner user ID
	OwnerID string `json:"owner_id"`

	// Creator user ID
	CreatorID string `json:"creator_id"`

	// Last modified by user ID
	LastModifiedBy string `json:"last_modified_by,omitempty"`

	// Creation timestamp
	CreatedAt time.Time `json:"created_at"`

	// Last update timestamp
	UpdatedAt time.Time `json:"updated_at"`

	// Last accessed timestamp
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty"`

	// Indexed timestamp
	IndexedAt *time.Time `json:"indexed_at,omitempty"`

	// Expiration timestamp (for temporary documents)
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// Deletion timestamp (soft delete)
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// ============================================================================
// Document Type
// ============================================================================

// DocumentType defines the type of document
type DocumentType string

const (
	// DocumentTypeText for plain text documents
	DocumentTypeText DocumentType = "text"

	// DocumentTypePDF for PDF documents
	DocumentTypePDF DocumentType = "pdf"

	// DocumentTypeMarkdown for Markdown documents
	DocumentTypeMarkdown DocumentType = "markdown"

	// DocumentTypeHTML for HTML documents
	DocumentTypeHTML DocumentType = "html"

	// DocumentTypeWord for Word documents
	DocumentTypeWord DocumentType = "word"

	// DocumentTypeExcel for Excel documents
	DocumentTypeExcel DocumentType = "excel"

	// DocumentTypePowerPoint for PowerPoint documents
	DocumentTypePowerPoint DocumentType = "powerpoint"

	// DocumentTypeCode for source code
	DocumentTypeCode DocumentType = "code"

	// DocumentTypeJSON for JSON documents
	DocumentTypeJSON DocumentType = "json"

	// DocumentTypeXML for XML documents
	DocumentTypeXML DocumentType = "xml"

	// DocumentTypeCSV for CSV documents
	DocumentTypeCSV DocumentType = "csv"

	// DocumentTypeEmail for email messages
	DocumentTypeEmail DocumentType = "email"

	// DocumentTypeWebPage for web pages
	DocumentTypeWebPage DocumentType = "webpage"
)

// ============================================================================
// Document Status
// ============================================================================

// DocumentStatus represents document processing status
type DocumentStatus string

const (
	// StatusPending indicates document is pending processing
	StatusPending DocumentStatus = "pending"

	// StatusProcessing indicates document is being processed
	StatusProcessing DocumentStatus = "processing"

	// StatusCompleted indicates processing is completed
	StatusCompleted DocumentStatus = "completed"

	// StatusFailed indicates processing failed
	StatusFailed DocumentStatus = "failed"

	// StatusDeleted indicates document is deleted
	StatusDeleted DocumentStatus = "deleted"
)

// ============================================================================
// Embedding Status
// ============================================================================

// EmbeddingStatus represents embedding generation status
type EmbeddingStatus string

const (
	// EmbeddingStatusNone indicates no embeddings
	EmbeddingStatusNone EmbeddingStatus = "none"

	// EmbeddingStatusPending indicates embeddings are pending
	EmbeddingStatusPending EmbeddingStatus = "pending"

	// EmbeddingStatusInProgress indicates embeddings are being generated
	EmbeddingStatusInProgress EmbeddingStatus = "in_progress"

	// EmbeddingStatusCompleted indicates embeddings are completed
	EmbeddingStatusCompleted EmbeddingStatus = "completed"

	// EmbeddingStatusFailed indicates embedding generation failed
	EmbeddingStatusFailed EmbeddingStatus = "failed"
)

// ============================================================================
// Sensitivity Level
// ============================================================================

// SensitivityLevel defines document sensitivity
type SensitivityLevel string

const (
	// SensitivityPublic for public documents
	SensitivityPublic SensitivityLevel = "public"

	// SensitivityInternal for internal documents
	SensitivityInternal SensitivityLevel = "internal"

	// SensitivityConfidential for confidential documents
	SensitivityConfidential SensitivityLevel = "confidential"

	// SensitivityRestricted for restricted documents
	SensitivityRestricted SensitivityLevel = "restricted"

	// SensitivitySecret for secret documents
	SensitivitySecret SensitivityLevel = "secret"
)

// ============================================================================
// Document Source
// ============================================================================

// DocumentSource contains source information
type DocumentSource struct {
	// Source type (upload, api, scrape, etc.)
	Type string `json:"type"`

	// Original URL if applicable
	URL string `json:"url,omitempty"`

	// Original filename
	FileName string `json:"file_name,omitempty"`

	// File path
	FilePath string `json:"file_path,omitempty"`

	// MIME type
	MIMEType string `json:"mime_type,omitempty"`

	// Author
	Author string `json:"author,omitempty"`

	// Publisher
	Publisher string `json:"publisher,omitempty"`

	// Publication date
	PublicationDate *time.Time `json:"publication_date,omitempty"`

	// Source metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ============================================================================
// Document Metadata
// ============================================================================

// DocumentMetadata contains document metadata
type DocumentMetadata struct {
	// Custom fields
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`

	// Keywords
	Keywords []string `json:"keywords,omitempty"`

	// Summary
	Summary string `json:"summary,omitempty"`

	// Abstract
	Abstract string `json:"abstract,omitempty"`

	// Description
	Description string `json:"description,omitempty"`

	// Subject
	Subject string `json:"subject,omitempty"`

	// Copyright
	Copyright string `json:"copyright,omitempty"`

	// License
	License string `json:"license,omitempty"`

	// DOI (Digital Object Identifier)
	DOI string `json:"doi,omitempty"`

	// ISBN
	ISBN string `json:"isbn,omitempty"`

	// Entities extracted from document
	Entities []Entity `json:"entities,omitempty"`

	// Topics
	Topics []Topic `json:"topics,omitempty"`
}

// Entity represents a named entity
type Entity struct {
	Type       string  `json:"type"`        // person, organization, location, etc.
	Value      string  `json:"value"`       // entity value
	Confidence float64 `json:"confidence"`  // confidence score
	Count      int     `json:"count"`       // occurrence count
}

// Topic represents a topic
type Topic struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	Keywords   []string `json:"keywords,omitempty"`
}

// ============================================================================
// Access Control
// ============================================================================

// AccessControl defines access control settings
type AccessControl struct {
	// Is public
	IsPublic bool `json:"is_public"`

	// Allowed user IDs
	AllowedUsers []string `json:"allowed_users,omitempty"`

	// Allowed group IDs
	AllowedGroups []string `json:"allowed_groups,omitempty"`

	// Denied user IDs
	DeniedUsers []string `json:"denied_users,omitempty"`

	// Required permissions
	RequiredPermissions []string `json:"required_permissions,omitempty"`

	// Access level
	AccessLevel AccessLevel `json:"access_level"`
}

// AccessLevel defines access level
type AccessLevel string

const (
	// AccessLevelRead for read access
	AccessLevelRead AccessLevel = "read"

	// AccessLevelWrite for write access
	AccessLevelWrite AccessLevel = "write"

	// AccessLevelAdmin for admin access
	AccessLevelAdmin AccessLevel = "admin"
)

// ============================================================================
// Document Statistics
// ============================================================================

// DocumentStats contains document statistics
type DocumentStats struct {
	// View count
	ViewCount int64 `json:"view_count"`

	// Download count
	DownloadCount int64 `json:"download_count"`

	// Search hit count
	SearchHitCount int64 `json:"search_hit_count"`

	// Reference count (how many times referenced)
	ReferenceCount int64 `json:"reference_count"`

	// Average rating
	AverageRating float64 `json:"average_rating"`

	// Rating count
	RatingCount int64 `json:"rating_count"`
}

// ============================================================================
// Chunk Entity
// ============================================================================

// Chunk represents a document chunk for embedding
type Chunk struct {
	// Unique identifier
	ID string `json:"id"`

	// Document ID
	DocumentID string `json:"document_id"`

	// Knowledge base ID
	KnowledgeBaseID string `json:"knowledge_base_id"`

	// Chunk content
	Content string `json:"content"`

	// Chunk index within document
	Index int `json:"index"`

	// Start position in original document
	StartPosition int `json:"start_position"`

	// End position in original document
	EndPosition int `json:"end_position"`

	// Token count
	TokenCount int `json:"token_count"`

	// Character count
	CharCount int `json:"char_count"`

	// Chunk metadata
	Metadata ChunkMetadata `json:"metadata"`

	// Embedding vector
	Embedding []float64 `json:"embedding,omitempty"`

	// Embedding model used
	EmbeddingModel string `json:"embedding_model"`

	// Embedding dimension
	EmbeddingDimension int `json:"embedding_dimension"`

	// Context window (surrounding text)
	ContextBefore string `json:"context_before,omitempty"`
	ContextAfter  string `json:"context_after,omitempty"`

	// Relevance score (for search results)
	RelevanceScore float64 `json:"relevance_score,omitempty"`

	// Similarity score (for similarity search)
	SimilarityScore float64 `json:"similarity_score,omitempty"`

	// Quality score
	QualityScore float64 `json:"quality_score"`

	// Is indexed
	IsIndexed bool `json:"is_indexed"`

	// Language
	Language string `json:"language"`

	// Creation timestamp
	CreatedAt time.Time `json:"created_at"`

	// Last update timestamp
	UpdatedAt time.Time `json:"updated_at"`

	// Indexed timestamp
	IndexedAt *time.Time `json:"indexed_at,omitempty"`
}

// ============================================================================
// Chunk Metadata
// ============================================================================

// ChunkMetadata contains chunk metadata
type ChunkMetadata struct {
	// Heading/section title
	Title string `json:"title,omitempty"`

	// Section level
	SectionLevel int `json:"section_level,omitempty"`

	// Page number
	PageNumber int `json:"page_number,omitempty"`

	// Paragraph index
	ParagraphIndex int `json:"paragraph_index,omitempty"`

	// Sentence indices
	SentenceIndices []int `json:"sentence_indices,omitempty"`

	// Contains code
	ContainsCode bool `json:"contains_code,omitempty"`

	// Contains table
	ContainsTable bool `json:"contains_table,omitempty"`

	// Contains list
	ContainsList bool `json:"contains_list,omitempty"`

	// Contains image reference
	ContainsImage bool `json:"contains_image,omitempty"`

	// Custom metadata
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}

// ============================================================================
// Embedding Entity
// ============================================================================

// Embedding represents a vector embedding
type Embedding struct {
	// Unique identifier
	ID string `json:"id"`

	// Entity ID (document, chunk, etc.)
	EntityID string `json:"entity_id"`

	// Entity type
	EntityType EmbeddingEntityType `json:"entity_type"`

	// Knowledge base ID
	KnowledgeBaseID string `json:"knowledge_base_id"`

	// Vector embedding
	Vector []float64 `json:"vector"`

	// Vector dimension
	Dimension int `json:"dimension"`

	// Model used for embedding
	Model string `json:"model"`

	// Model version
	ModelVersion string `json:"model_version"`

	// Text that was embedded
	Text string `json:"text"`

	// Token count
	TokenCount int `json:"token_count"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Normalization applied
	Normalized bool `json:"normalized"`

	// Creation timestamp
	CreatedAt time.Time `json:"created_at"`

	// Expiration timestamp
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// ============================================================================
// Embedding Entity Type
// ============================================================================

// EmbeddingEntityType defines the type of entity being embedded
type EmbeddingEntityType string

const (
	// EmbeddingEntityDocument for document embeddings
	EmbeddingEntityDocument EmbeddingEntityType = "document"

	// EmbeddingEntityChunk for chunk embeddings
	EmbeddingEntityChunk EmbeddingEntityType = "chunk"

	// EmbeddingEntityQuery for query embeddings
	EmbeddingEntityQuery EmbeddingEntityType = "query"
)

// ============================================================================
// Entity Factory Methods
// ============================================================================

// NewDocument creates a new document entity
func NewDocument(knowledgeBaseID, title, content string, docType DocumentType, ownerID, creatorID string) *Document {
	now := time.Now()
	return &Document{
		ID:              generateDocumentID(),
		KnowledgeBaseID: knowledgeBaseID,
		Title:           title,
		Content:         content,
		Type:            docType,
		Language:        "en",
		Sensitivity:     SensitivityInternal,
		Status:          StatusPending,
		EmbeddingStatus: EmbeddingStatusNone,
		Tags:            make([]string, 0),
		Categories:      make([]string, 0),
		RelatedDocuments: make([]string, 0),
		Version:         1,
		QualityScore:    0.0,
		OwnerID:         ownerID,
		CreatorID:       creatorID,
		Source:          DocumentSource{},
		Metadata: DocumentMetadata{
			CustomFields: make(map[string]interface{}),
		},
		AccessControl: AccessControl{
			IsPublic:    false,
			AccessLevel: AccessLevelRead,
		},
		Stats:     DocumentStats{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewChunk creates a new chunk entity
func NewChunk(documentID, knowledgeBaseID, content string, index int) *Chunk {
	now := time.Now()
	return &Chunk{
		ID:              generateChunkID(),
		DocumentID:      documentID,
		KnowledgeBaseID: knowledgeBaseID,
		Content:         content,
		Index:           index,
		TokenCount:      0,
		CharCount:       len(content),
		QualityScore:    0.0,
		IsIndexed:       false,
		Language:        "en",
		Metadata: ChunkMetadata{
			CustomFields: make(map[string]interface{}),
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewEmbedding creates a new embedding entity
func NewEmbedding(entityID string, entityType EmbeddingEntityType, knowledgeBaseID, text, model string, vector []float64) *Embedding {
	return &Embedding{
		ID:              generateEmbeddingID(),
		EntityID:        entityID,
		EntityType:      entityType,
		KnowledgeBaseID: knowledgeBaseID,
		Vector:          vector,
		Dimension:       len(vector),
		Model:           model,
		Text:            text,
		TokenCount:      0,
		Metadata:        make(map[string]interface{}),
		Normalized:      false,
		CreatedAt:       time.Now(),
	}
}

// generateDocumentID generates a unique document ID
func generateDocumentID() string {
	return fmt.Sprintf("doc_%s", uuid.New().String())
}

// generateChunkID generates a unique chunk ID
func generateChunkID() string {
	return fmt.Sprintf("chunk_%s", uuid.New().String())
}

// generateEmbeddingID generates a unique embedding ID
func generateEmbeddingID() string {
	return fmt.Sprintf("emb_%s", uuid.New().String())
}

// ============================================================================
// Domain Methods - Document
// ============================================================================

// IsDeleted checks if document is soft deleted
func (d *Document) IsDeleted() bool {
	return d.DeletedAt != nil
}

// IsExpired checks if document is expired
func (d *Document) IsExpired() bool {
	return d.ExpiresAt != nil && time.Now().After(*d.ExpiresAt)
}

// IsProcessed checks if document is fully processed
func (d *Document) IsProcessed() bool {
	return d.Status == StatusCompleted
}

// HasEmbeddings checks if document has embeddings
func (d *Document) HasEmbeddings() bool {
	return d.EmbeddingStatus == EmbeddingStatusCompleted
}

// CanAccess checks if user can access document
func (d *Document) CanAccess(userID string, userGroups []string) bool {
	if d.AccessControl.IsPublic {
		return true
	}

	// Check denied users
	for _, deniedUser := range d.AccessControl.DeniedUsers {
		if deniedUser == userID {
			return false
		}
	}

	// Check allowed users
	for _, allowedUser := range d.AccessControl.AllowedUsers {
		if allowedUser == userID {
			return true
		}
	}

	// Check allowed groups
	for _, allowedGroup := range d.AccessControl.AllowedGroups {
		for _, userGroup := range userGroups {
			if allowedGroup == userGroup {
				return true
			}
		}
	}

	// Owner always has access
	return d.OwnerID == userID
}

// UpdateContent updates document content
func (d *Document) UpdateContent(content string) {
	d.Content = content
	d.CharCount = len(content)
	d.UpdatedAt = time.Now()
	d.Status = StatusPending
	d.EmbeddingStatus = EmbeddingStatusNone
}

// MarkProcessing marks document as processing
func (d *Document) MarkProcessing() {
	d.Status = StatusProcessing
	d.UpdatedAt = time.Now()
}

// MarkCompleted marks document as completed
func (d *Document) MarkCompleted() {
	d.Status = StatusCompleted
	d.UpdatedAt = time.Now()
}

// MarkFailed marks document as failed
func (d *Document) MarkFailed(err string) {
	d.Status = StatusFailed
	d.ProcessingError = err
	d.UpdatedAt = time.Now()
}

// SoftDelete soft deletes the document
func (d *Document) SoftDelete() {
	now := time.Now()
	d.DeletedAt = &now
	d.Status = StatusDeleted
	d.UpdatedAt = now
}

// RecordAccess records document access
func (d *Document) RecordAccess() {
	now := time.Now()
	d.LastAccessedAt = &now
	d.Stats.ViewCount++
}

// AddTag adds a tag
func (d *Document) AddTag(tag string) {
	for _, t := range d.Tags {
		if t == tag {
			return
		}
	}
	d.Tags = append(d.Tags, tag)
	d.UpdatedAt = time.Now()
}

// RemoveTag removes a tag
func (d *Document) RemoveTag(tag string) {
	for i, t := range d.Tags {
		if t == tag {
			d.Tags = append(d.Tags[:i], d.Tags[i+1:]...)
			d.UpdatedAt = time.Now()
			return
		}
	}
}

// Validate validates the document
func (d *Document) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("document ID is required")
	}
	if d.KnowledgeBaseID == "" {
		return fmt.Errorf("knowledge base ID is required")
	}
	if d.Title == "" {
		return fmt.Errorf("title is required")
	}
	if d.Type == "" {
		return fmt.Errorf("document type is required")
	}
	if d.OwnerID == "" {
		return fmt.Errorf("owner ID is required")
	}
	return nil
}

// ToJSON converts document to JSON
func (d *Document) ToJSON() ([]byte, error) {
	return json.Marshal(d)
}

// ============================================================================
// Domain Methods - Chunk
// ============================================================================

// MarkIndexed marks chunk as indexed
func (c *Chunk) MarkIndexed() {
	now := time.Now()
	c.IsIndexed = true
	c.IndexedAt = &now
	c.UpdatedAt = now
}

// SetEmbedding sets the embedding vector
func (c *Chunk) SetEmbedding(vector []float64, model string) {
	c.Embedding = vector
	c.EmbeddingModel = model
	c.EmbeddingDimension = len(vector)
	c.UpdatedAt = time.Now()
}

// HasEmbedding checks if chunk has embedding
func (c *Chunk) HasEmbedding() bool {
	return len(c.Embedding) > 0
}

// Validate validates the chunk
func (c *Chunk) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("chunk ID is required")
	}
	if c.DocumentID == "" {
		return fmt.Errorf("document ID is required")
	}
	if c.KnowledgeBaseID == "" {
		return fmt.Errorf("knowledge base ID is required")
	}
	if c.Content == "" {
		return fmt.Errorf("content is required")
	}
	return nil
}

// ToJSON converts chunk to JSON
func (c *Chunk) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// ============================================================================
// Domain Methods - Embedding
// ============================================================================

// IsExpired checks if embedding is expired
func (e *Embedding) IsExpired() bool {
	return e.ExpiresAt != nil && time.Now().After(*e.ExpiresAt)
}

// Normalize normalizes the embedding vector
func (e *Embedding) Normalize() {
	if e.Normalized {
		return
	}

	var norm float64
	for _, v := range e.Vector {
		norm += v * v
	}
	norm = 1.0 / (norm + 1e-10)

	for i := range e.Vector {
		e.Vector[i] *= norm
	}

	e.Normalized = true
}

// Validate validates the embedding
func (e *Embedding) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("embedding ID is required")
	}
	if e.EntityID == "" {
		return fmt.Errorf("entity ID is required")
	}
	if e.KnowledgeBaseID == "" {
		return fmt.Errorf("knowledge base ID is required")
	}
	if len(e.Vector) == 0 {
		return fmt.Errorf("vector is required")
	}
	if e.Model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

// ToJSON converts embedding to JSON
func (e *Embedding) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

//Personal.AI order the ending
