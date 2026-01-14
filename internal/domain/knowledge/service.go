// Package knowledge provides domain services for knowledge management.
// It implements document processing workflows including parsing, chunking,
// vectorization, PII detection, and orchestrates the complete knowledge pipeline.
package knowledge

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// Domain Service Interface
// ============================================================================

// KnowledgeService defines the domain service interface for knowledge operations
type KnowledgeService interface {
	// ProcessDocument orchestrates the complete document processing workflow
	ProcessDocument(ctx context.Context, req ProcessDocumentRequest) (*ProcessDocumentResult, error)

	// UploadDocument uploads and processes a new document
	UploadDocument(ctx context.Context, req UploadDocumentRequest) (*Document, error)

	// UpdateDocument updates an existing document
	UpdateDocument(ctx context.Context, id string, req UpdateDocumentRequest) (*Document, error)

	// DeleteDocument deletes a document
	DeleteDocument(ctx context.Context, id string, hard bool) error

	// GetDocument retrieves a document
	GetDocument(ctx context.Context, id string) (*Document, error)

	// SearchDocuments searches documents
	SearchDocuments(ctx context.Context, req SearchDocumentsRequest) (*SearchDocumentsResult, error)

	// SemanticSearch performs semantic search
	SemanticSearch(ctx context.Context, req SemanticSearchRequest) (*SemanticSearchResult, error)

	// HybridSearch performs hybrid search (keyword + semantic)
	HybridSearch(ctx context.Context, req HybridSearchRequest) (*HybridSearchResult, error)

	// ReprocessDocument reprocesses a document
	ReprocessDocument(ctx context.Context, id string) error

	// GenerateEmbeddings generates embeddings for documents
	GenerateEmbeddings(ctx context.Context, kbID string, batchSize int) error

	// DetectPII detects personally identifiable information
	DetectPII(ctx context.Context, documentID string) (*PIIDetectionResult, error)

	// SanitizeDocument sanitizes document by removing/masking PII
	SanitizeDocument(ctx context.Context, documentID string, mode SanitizationMode) (*Document, error)

	// GetDocumentChunks retrieves chunks for a document
	GetDocumentChunks(ctx context.Context, documentID string) ([]*Chunk, error)

	// GetSimilarDocuments finds similar documents
	GetSimilarDocuments(ctx context.Context, documentID string, limit int) ([]*Document, error)

	// GetDocumentStatistics retrieves document statistics
	GetDocumentStatistics(ctx context.Context, kbID string) (*DocumentStatistics, error)

	// ValidateDocument validates document before processing
	ValidateDocument(ctx context.Context, req UploadDocumentRequest) error

	// BatchProcessDocuments processes multiple documents
	BatchProcessDocuments(ctx context.Context, requests []ProcessDocumentRequest) ([]*ProcessDocumentResult, error)
}

// ============================================================================
// Domain Service Implementation
// ============================================================================

// knowledgeService implements the KnowledgeService interface
type knowledgeService struct {
	documentRepo    DocumentRepository
	chunkRepo       ChunkRepository
	embeddingRepo   EmbeddingRepository
	parser          DocumentParser
	chunker         DocumentChunker
	embedder        EmbeddingGenerator
	piiDetector     PIIDetector
	qualityAnalyzer QualityAnalyzer
	eventEmitter    KnowledgeEventEmitter
	mu              sync.RWMutex
}

// NewKnowledgeService creates a new knowledge domain service
func NewKnowledgeService(
	documentRepo DocumentRepository,
	chunkRepo ChunkRepository,
	embeddingRepo EmbeddingRepository,
	parser DocumentParser,
	chunker DocumentChunker,
	embedder EmbeddingGenerator,
	piiDetector PIIDetector,
	qualityAnalyzer QualityAnalyzer,
	eventEmitter KnowledgeEventEmitter,
) KnowledgeService {
	return &knowledgeService{
		documentRepo:    documentRepo,
		chunkRepo:       chunkRepo,
		embeddingRepo:   embeddingRepo,
		parser:          parser,
		chunker:         chunker,
		embedder:        embedder,
		piiDetector:     piiDetector,
		qualityAnalyzer: qualityAnalyzer,
		eventEmitter:    eventEmitter,
	}
}

// ============================================================================
// Request/Response Types
// ============================================================================

// UploadDocumentRequest represents document upload request
type UploadDocumentRequest struct {
	KnowledgeBaseID string
	Title           string
	Content         string
	Type            DocumentType
	Source          DocumentSource
	Metadata        DocumentMetadata
	Language        string
	Sensitivity     SensitivityLevel
	Tags            []string
	Categories      []string
	OwnerID         string
	CreatorID       string
	AutoProcess     bool
}

// UpdateDocumentRequest represents document update request
type UpdateDocumentRequest struct {
	Title       *string
	Content     *string
	Metadata    *DocumentMetadata
	Tags        []string
	Categories  []string
	Sensitivity *SensitivityLevel
}

// ProcessDocumentRequest represents document processing request
type ProcessDocumentRequest struct {
	DocumentID      string
	ParseOptions    ParseOptions
	ChunkOptions    ChunkOptions
	EmbedOptions    EmbedOptions
	DetectPII       bool
	AnalyzeQuality  bool
	GenerateSummary bool
}

// ProcessDocumentResult represents processing result
type ProcessDocumentResult struct {
	Document       *Document
	ChunkCount     int
	EmbeddingCount int
	PIIFound       bool
	PIIResults     *PIIDetectionResult
	QualityScore   float64
	ProcessingTime time.Duration
	Errors         []string
}

// SearchDocumentsRequest represents document search request
type SearchDocumentsRequest struct {
	KnowledgeBaseID string
	Query           string
	Filter          DocumentFilter
	Highlight       bool
}

// SearchDocumentsResult represents search result
type SearchDocumentsResult struct {
	Documents  []*Document
	TotalCount int64
	SearchTime time.Duration
}

// SemanticSearchRequest represents semantic search request
type SemanticSearchRequest struct {
	KnowledgeBaseID string
	Query           string
	Limit           int
	MinSimilarity   float64
	Filter          ChunkFilter
	IncludeContext  bool
}

// SemanticSearchResult represents semantic search result
type SemanticSearchResult struct {
	Results    []*SemanticSearchMatch
	SearchTime time.Duration
}

// SemanticSearchMatch represents a semantic search match
type SemanticSearchMatch struct {
	Chunk           *Chunk
	Document        *Document
	SimilarityScore float64
	Context         string
}

// HybridSearchRequest represents hybrid search request
type HybridSearchRequest struct {
	KnowledgeBaseID string
	Query           string
	Limit           int
	KeywordWeight   float64
	SemanticWeight  float64
	Filter          ChunkFilter
}

// HybridSearchResult represents hybrid search result
type HybridSearchResult struct {
	Results    []*HybridSearchMatch
	SearchTime time.Duration
}

// HybridSearchMatch represents a hybrid search match
type HybridSearchMatch struct {
	Chunk            *Chunk
	Document         *Document
	CombinedScore    float64
	KeywordScore     float64
	SemanticScore    float64
	Highlights       []string
}

// PIIDetectionResult represents PII detection result
type PIIDetectionResult struct {
	DocumentID string
	PIIFound   bool
	Entities   []PIIEntity
	RiskLevel  PIIRiskLevel
	Suggestions []string
}

// PIIEntity represents a PII entity
type PIIEntity struct {
	Type       PIIType
	Value      string
	Position   int
	Confidence float64
	Context    string
}

// PIIType defines types of PII
type PIIType string

const (
	PIITypeEmail          PIIType = "email"
	PIITypePhone          PIIType = "phone"
	PIITypeSSN            PIIType = "ssn"
	PIITypeCreditCard     PIIType = "credit_card"
	PIITypeAddress        PIIType = "address"
	PIITypeName           PIIType = "name"
	PIITypeDateOfBirth    PIIType = "date_of_birth"
	PIITypeIPAddress      PIIType = "ip_address"
	PIITypeBankAccount    PIIType = "bank_account"
)

// PIIRiskLevel defines PII risk levels
type PIIRiskLevel string

const (
	PIIRiskNone     PIIRiskLevel = "none"
	PIIRiskLow      PIIRiskLevel = "low"
	PIIRiskMedium   PIIRiskLevel = "medium"
	PIIRiskHigh     PIIRiskLevel = "high"
	PIIRiskCritical PIIRiskLevel = "critical"
)

// SanitizationMode defines sanitization modes
type SanitizationMode string

const (
	SanitizationModeRemove SanitizationMode = "remove"
	SanitizationModeMask   SanitizationMode = "mask"
	SanitizationModeHash   SanitizationMode = "hash"
)

// ============================================================================
// Upload Document
// ============================================================================

// UploadDocument uploads and processes a new document
func (s *knowledgeService) UploadDocument(ctx context.Context, req UploadDocumentRequest) (*Document, error) {
	// Validate request
	if err := s.ValidateDocument(ctx, req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create document entity
	document := NewDocument(
		req.KnowledgeBaseID,
		req.Title,
		req.Content,
		req.Type,
		req.OwnerID,
		req.CreatorID,
	)

	// Set additional fields
	document.Source = req.Source
	document.Metadata = req.Metadata
	document.Language = req.Language
	document.Sensitivity = req.Sensitivity
	document.Tags = req.Tags
	document.Categories = req.Categories

	// Calculate content hash
	document.ContentHash = s.calculateContentHash(req.Content)

	// Calculate statistics
	document.CharCount = len(req.Content)
	document.WordCount = s.countWords(req.Content)
	document.SizeBytes = int64(len(req.Content))

	// Check for duplicate
	exists, err := s.documentRepo.ExistsByHash(ctx, req.KnowledgeBaseID, document.ContentHash)
	if err != nil {
		return nil, fmt.Errorf("failed to check duplicate: %w", err)
	}
	if exists {
		existing, _ := s.documentRepo.GetByContentHash(ctx, req.KnowledgeBaseID, document.ContentHash)
		return existing, fmt.Errorf("duplicate document found")
	}

	// Persist document
	if err := s.documentRepo.Create(ctx, document); err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	// Emit event
	if s.eventEmitter != nil {
		s.eventEmitter.EmitDocumentUploaded(ctx, document)
	}

	// Auto-process if requested
	if req.AutoProcess {
		processReq := ProcessDocumentRequest{
			DocumentID:      document.ID,
			DetectPII:       true,
			AnalyzeQuality:  true,
			GenerateSummary: true,
		}
		go s.ProcessDocument(context.Background(), processReq)
	}

	return document, nil
}

// ============================================================================
// Process Document
// ============================================================================

// ProcessDocument orchestrates the complete document processing workflow
func (s *knowledgeService) ProcessDocument(ctx context.Context, req ProcessDocumentRequest) (*ProcessDocumentResult, error) {
	startTime := time.Now()
	result := &ProcessDocumentResult{
		Errors: make([]string, 0),
	}

	// Retrieve document
	document, err := s.documentRepo.GetByID(ctx, req.DocumentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	// Update status to processing
	document.MarkProcessing()
	s.documentRepo.Update(ctx, document)

	// Step 1: Parse document
	parsedContent, parseErr := s.parseDocument(ctx, document, req.ParseOptions)
	if parseErr != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("parsing failed: %v", parseErr))
		document.MarkFailed(parseErr.Error())
		s.documentRepo.Update(ctx, document)
		return result, parseErr
	}

	// Update document content with parsed content
	if parsedContent != document.Content {
		document.UpdateContent(parsedContent)
		s.documentRepo.Update(ctx, document)
	}

	// Step 2: Detect PII if requested
	if req.DetectPII {
		piiResult, piiErr := s.detectDocumentPII(ctx, document)
		if piiErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("PII detection failed: %v", piiErr))
		} else {
			result.PIIFound = piiResult.PIIFound
			result.PIIResults = piiResult

			// Update document sensitivity based on PII
			if piiResult.PIIFound {
				s.updateSensitivityBasedOnPII(document, piiResult)
			}
		}
	}

	// Step 3: Analyze quality if requested
	if req.AnalyzeQuality {
		qualityScore, qualityErr := s.analyzeDocumentQuality(ctx, document)
		if qualityErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("quality analysis failed: %v", qualityErr))
		} else {
			document.QualityScore = qualityScore
			result.QualityScore = qualityScore
		}
	}

	// Step 4: Chunk document
	chunks, chunkErr := s.chunkDocument(ctx, document, req.ChunkOptions)
	if chunkErr != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("chunking failed: %v", chunkErr))
		document.MarkFailed(chunkErr.Error())
		s.documentRepo.Update(ctx, document)
		return result, chunkErr
	}

	result.ChunkCount = len(chunks)
	document.ChunkCount = len(chunks)

	// Step 5: Generate embeddings
	embeddingCount, embedErr := s.generateChunkEmbeddings(ctx, chunks, req.EmbedOptions)
	if embedErr != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("embedding generation failed: %v", embedErr))
	}
	result.EmbeddingCount = embeddingCount

	// Update document status
	if len(result.Errors) == 0 {
		document.MarkCompleted()
		document.EmbeddingStatus = EmbeddingStatusCompleted
		now := time.Now()
		document.IndexedAt = &now
	} else {
		document.MarkFailed(strings.Join(result.Errors, "; "))
	}

	s.documentRepo.Update(ctx, document)

	result.Document = document
	result.ProcessingTime = time.Since(startTime)

	// Emit completion event
	if s.eventEmitter != nil {
		s.eventEmitter.EmitDocumentProcessed(ctx, document)
	}

	return result, nil
}

// ============================================================================
// Document Parsing
// ============================================================================

// parseDocument parses document content
func (s *knowledgeService) parseDocument(ctx context.Context, document *Document, options ParseOptions) (string, error) {
	if s.parser == nil {
		return document.Content, nil
	}

	parsed, err := s.parser.Parse(ctx, document.Content, document.Type, options)
	if err != nil {
		return "", fmt.Errorf("parsing failed: %w", err)
	}

	return parsed, nil
}

// ============================================================================
// Document Chunking
// ============================================================================

// chunkDocument splits document into chunks
func (s *knowledgeService) chunkDocument(ctx context.Context, document *Document, options ChunkOptions) ([]*Chunk, error) {
	if s.chunker == nil {
		return nil, fmt.Errorf("chunker not configured")
	}

	// Delete existing chunks
	s.chunkRepo.DeleteByDocumentID(ctx, document.ID)

	// Generate chunks
	chunkTexts, err := s.chunker.Chunk(ctx, document.Content, options)
	if err != nil {
		return nil, fmt.Errorf("chunking failed: %w", err)
	}

	chunks := make([]*Chunk, 0, len(chunkTexts))
	for i, text := range chunkTexts {
		chunk := NewChunk(document.ID, document.KnowledgeBaseID, text, i)
		chunk.Language = document.Language
		chunk.TokenCount = s.estimateTokenCount(text)

		// Calculate quality score
		if s.qualityAnalyzer != nil {
			chunk.QualityScore = s.qualityAnalyzer.AnalyzeChunk(text)
		}

		chunks = append(chunks, chunk)
	}

	// Batch create chunks
	if err := s.chunkRepo.BatchCreate(ctx, chunks); err != nil {
		return nil, fmt.Errorf("failed to create chunks: %w", err)
	}

	return chunks, nil
}

// ============================================================================
// Embedding Generation
// ============================================================================

// generateChunkEmbeddings generates embeddings for chunks
func (s *knowledgeService) generateChunkEmbeddings(ctx context.Context, chunks []*Chunk, options EmbedOptions) (int, error) {
	if s.embedder == nil {
		return 0, fmt.Errorf("embedder not configured")
	}

	count := 0
	batchSize := 32
	batches := s.batchChunks(chunks, batchSize)

	for _, batch := range batches {
		texts := make([]string, len(batch))
		for i, chunk := range batch {
			texts[i] = chunk.Content
		}

		// Generate embeddings
		vectors, err := s.embedder.GenerateBatch(ctx, texts, options)
		if err != nil {
			return count, fmt.Errorf("embedding generation failed: %w", err)
		}

		// Update chunks with embeddings
		for i, chunk := range batch {
			if i < len(vectors) {
				chunk.SetEmbedding(vectors[i], options.Model)
				chunk.MarkIndexed()

				// Update chunk
				if err := s.chunkRepo.Update(ctx, chunk); err != nil {
					continue
				}

				// Create embedding entity
				embedding := NewEmbedding(
					chunk.ID,
					EmbeddingEntityChunk,
					chunk.KnowledgeBaseID,
					chunk.Content,
					options.Model,
					vectors[i],
				)
				embedding.TokenCount = chunk.TokenCount

				if err := s.embeddingRepo.Create(ctx, embedding); err != nil {
					continue
				}

				count++
			}
		}
	}

	return count, nil
}

// GenerateEmbeddings generates embeddings for documents in batch
func (s *knowledgeService) GenerateEmbeddings(ctx context.Context, kbID string, batchSize int) error {
	// Get documents pending embedding
	documents, err := s.documentRepo.GetPendingEmbedding(ctx, kbID, batchSize)
	if err != nil {
		return fmt.Errorf("failed to get pending documents: %w", err)
	}

	for _, document := range documents {
		processReq := ProcessDocumentRequest{
			DocumentID:   document.ID,
			DetectPII:    false,
			AnalyzeQuality: false,
		}

		_, err := s.ProcessDocument(ctx, processReq)
		if err != nil {
			document.MarkFailed(err.Error())
			s.documentRepo.Update(ctx, document)
			continue
		}
	}

	return nil
}

// ============================================================================
// PII Detection
// ============================================================================

// DetectPII detects personally identifiable information
func (s *knowledgeService) DetectPII(ctx context.Context, documentID string) (*PIIDetectionResult, error) {
	document, err := s.documentRepo.GetByID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return s.detectDocumentPII(ctx, document)
}

// detectDocumentPII performs PII detection on document
func (s *knowledgeService) detectDocumentPII(ctx context.Context, document *Document) (*PIIDetectionResult, error) {
	if s.piiDetector == nil {
		return &PIIDetectionResult{
			DocumentID: document.ID,
			PIIFound:   false,
			Entities:   make([]PIIEntity, 0),
			RiskLevel:  PIIRiskNone,
		}, nil
	}

	entities, err := s.piiDetector.Detect(ctx, document.Content)
	if err != nil {
		return nil, fmt.Errorf("PII detection failed: %w", err)
	}

	result := &PIIDetectionResult{
		DocumentID:  document.ID,
		PIIFound:    len(entities) > 0,
		Entities:    entities,
		RiskLevel:   s.calculatePIIRiskLevel(entities),
		Suggestions: s.generatePIISuggestions(entities),
	}

	return result, nil
}

// updateSensitivityBasedOnPII updates document sensitivity based on PII
func (s *knowledgeService) updateSensitivityBasedOnPII(document *Document, piiResult *PIIDetectionResult) {
	switch piiResult.RiskLevel {
	case PIIRiskCritical:
		document.Sensitivity = SensitivitySecret
	case PIIRiskHigh:
		document.Sensitivity = SensitivityRestricted
	case PIIRiskMedium:
		document.Sensitivity = SensitivityConfidential
	case PIIRiskLow:
		if document.Sensitivity == SensitivityPublic {
			document.Sensitivity = SensitivityInternal
		}
	}
}

// calculatePIIRiskLevel calculates overall PII risk level
func (s *knowledgeService) calculatePIIRiskLevel(entities []PIIEntity) PIIRiskLevel {
	if len(entities) == 0 {
		return PIIRiskNone
	}

	highRiskTypes := map[PIIType]bool{
		PIITypeSSN:         true,
		PIITypeCreditCard:  true,
		PIITypeBankAccount: true,
	}

	mediumRiskTypes := map[PIIType]bool{
		PIITypeDateOfBirth: true,
		PIITypeAddress:     true,
	}

	criticalCount := 0
	highCount := 0
	mediumCount := 0

	for _, entity := range entities {
		if highRiskTypes[entity.Type] {
			criticalCount++
		} else if mediumRiskTypes[entity.Type] {
			highCount++
		} else {
			mediumCount++
		}
	}

	if criticalCount > 0 {
		return PIIRiskCritical
	}
	if highCount > 2 {
		return PIIRiskHigh
	}
	if highCount > 0 || mediumCount > 5 {
		return PIIRiskMedium
	}
	if mediumCount > 0 {
		return PIIRiskLow
	}

	return PIIRiskNone
}

// generatePIISuggestions generates suggestions for handling PII
func (s *knowledgeService) generatePIISuggestions(entities []PIIEntity) []string {
	suggestions := make([]string, 0)

	typeCount := make(map[PIIType]int)
	for _, entity := range entities {
		typeCount[entity.Type]++
	}

	for piiType, count := range typeCount {
		switch piiType {
		case PIITypeSSN, PIITypeCreditCard, PIITypeBankAccount:
			suggestions = append(suggestions, fmt.Sprintf("Remove or mask %d %s occurrence(s)", count, piiType))
		case PIITypeEmail, PIITypePhone:
			suggestions = append(suggestions, fmt.Sprintf("Consider masking %d %s occurrence(s)", count, piiType))
		}
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Review document for sensitive information")
	}

	return suggestions
}

// ============================================================================
// Document Sanitization
// ============================================================================

// SanitizeDocument sanitizes document by removing/masking PII
func (s *knowledgeService) SanitizeDocument(ctx context.Context, documentID string, mode SanitizationMode) (*Document, error) {
	document, err := s.documentRepo.GetByID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	// Detect PII
	piiResult, err := s.detectDocumentPII(ctx, document)
	if err != nil {
		return nil, fmt.Errorf("PII detection failed: %w", err)
	}

	if !piiResult.PIIFound {
		return document, nil
	}

	// Sanitize content
	sanitized := document.Content
	for _, entity := range piiResult.Entities {
		replacement := s.getSanitizationReplacement(entity, mode)
		sanitized = strings.ReplaceAll(sanitized, entity.Value, replacement)
	}

	// Update document
	document.UpdateContent(sanitized)
	document.ContentHash = s.calculateContentHash(sanitized)

	if err := s.documentRepo.Update(ctx, document); err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	// Reprocess document
	s.ReprocessDocument(ctx, documentID)

	return document, nil
}

// getSanitizationReplacement gets replacement text for PII
func (s *knowledgeService) getSanitizationReplacement(entity PIIEntity, mode SanitizationMode) string {
	switch mode {
	case SanitizationModeRemove:
		return "[REDACTED]"
	case SanitizationModeMask:
		return s.maskValue(entity.Value)
	case SanitizationModeHash:
		return s.hashValue(entity.Value)
	default:
		return "[REDACTED]"
	}
}

// maskValue masks a value
func (s *knowledgeService) maskValue(value string) string {
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

// hashValue hashes a value
func (s *knowledgeService) hashValue(value string) string {
	hash := sha256.Sum256([]byte(value))
	return fmt.Sprintf("[HASH:%x]", hash[:8])
}

// ============================================================================
// Quality Analysis
// ============================================================================

// analyzeDocumentQuality analyzes document quality
func (s *knowledgeService) analyzeDocumentQuality(ctx context.Context, document *Document) (float64, error) {
	if s.qualityAnalyzer == nil {
		return 0.5, nil
	}

	score := s.qualityAnalyzer.AnalyzeDocument(document.Content)
	return score, nil
}

// ============================================================================
// Search Operations
// ============================================================================

// SearchDocuments searches documents
func (s *knowledgeService) SearchDocuments(ctx context.Context, req SearchDocumentsRequest) (*SearchDocumentsResult, error) {
	startTime := time.Now()

	documents, err := s.documentRepo.FullTextSearch(ctx, req.KnowledgeBaseID, req.Query, req.Filter)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	total, _ := s.documentRepo.Count(ctx, req.Filter)

	return &SearchDocumentsResult{
		Documents:  documents,
		TotalCount: total,
		SearchTime: time.Since(startTime),
	}, nil
}

// SemanticSearch performs semantic search
func (s *knowledgeService) SemanticSearch(ctx context.Context, req SemanticSearchRequest) (*SemanticSearchResult, error) {
	startTime := time.Now()

	// Generate query embedding
	queryVector, err := s.embedder.Generate(ctx, req.Query, EmbedOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search similar chunks
	chunks, err := s.chunkRepo.VectorSearch(ctx, req.KnowledgeBaseID, queryVector, req.Limit, req.Filter)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Build results
	results := make([]*SemanticSearchMatch, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk.SimilarityScore < req.MinSimilarity {
			continue
		}

		document, _ := s.documentRepo.GetByID(ctx, chunk.DocumentID)

		match := &SemanticSearchMatch{
			Chunk:           chunk,
			Document:        document,
			SimilarityScore: chunk.SimilarityScore,
		}

		if req.IncludeContext {
			match.Context = s.buildContext(chunk)
		}

		results = append(results, match)
	}

	return &SemanticSearchResult{
		Results:    results,
		SearchTime: time.Since(startTime),
	}, nil
}

// HybridSearch performs hybrid search
func (s *knowledgeService) HybridSearch(ctx context.Context, req HybridSearchRequest) (*HybridSearchResult, error) {
	startTime := time.Now()

	// Generate query embedding
	queryVector, err := s.embedder.Generate(ctx, req.Query, EmbedOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Perform hybrid search
	hybridQuery := HybridSearchQuery{
		Query:         req.Query,
		Vector:        queryVector,
		KeywordWeight: req.KeywordWeight,
		VectorWeight:  req.SemanticWeight,
		Filter:        req.Filter,
		Limit:         req.Limit,
	}

	chunks, err := s.chunkRepo.HybridSearch(ctx, req.KnowledgeBaseID, hybridQuery)
	if err != nil {
		return nil, fmt.Errorf("hybrid search failed: %w", err)
	}

	// Build results
	results := make([]*HybridSearchMatch, 0, len(chunks))
	for _, chunk := range chunks {
		document, _ := s.documentRepo.GetByID(ctx, chunk.DocumentID)

		results = append(results, &HybridSearchMatch{
			Chunk:         chunk,
			Document:      document,
			CombinedScore: chunk.RelevanceScore,
			SemanticScore: chunk.SimilarityScore,
		})
	}

	return &HybridSearchResult{
		Results:    results,
		SearchTime: time.Since(startTime),
	}, nil
}

// GetSimilarDocuments finds similar documents
func (s *knowledgeService) GetSimilarDocuments(ctx context.Context, documentID string, limit int) ([]*Document, error) {
	// Get document chunks
	chunks, err := s.chunkRepo.GetByDocumentID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunks: %w", err)
	}

	if len(chunks) == 0 {
		return []*Document{}, nil
	}

	// Use first chunk's embedding for similarity search
	if !chunks[0].HasEmbedding() {
		return []*Document{}, nil
	}

	// Search similar chunks
	similarChunks, err := s.chunkRepo.VectorSearch(
		ctx,
		chunks[0].KnowledgeBaseID,
		chunks[0].Embedding,
		limit*3,
		NewChunkFilter(),
	)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Deduplicate by document ID
	documentIDs := make(map[string]bool)
	documents := make([]*Document, 0)

	for _, chunk := range similarChunks {
		if chunk.DocumentID == documentID {
			continue
		}
		if documentIDs[chunk.DocumentID] {
			continue
		}
		if len(documents) >= limit {
			break
		}

		document, err := s.documentRepo.GetByID(ctx, chunk.DocumentID)
		if err != nil {
			continue
		}

		document.RelevanceScore = chunk.SimilarityScore
		documents = append(documents, document)
		documentIDs[chunk.DocumentID] = true
	}

	return documents, nil
}

// ============================================================================
// Document Operations
// ============================================================================

// GetDocument retrieves a document
func (s *knowledgeService) GetDocument(ctx context.Context, id string) (*Document, error) {
	document, err := s.documentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	// Record access
	s.documentRepo.RecordAccess(ctx, id)

	return document, nil
}

// UpdateDocument updates an existing document
func (s *knowledgeService) UpdateDocument(ctx context.Context, id string, req UpdateDocumentRequest) (*Document, error) {
	document, err := s.documentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	// Update fields
	updated := false

	if req.Title != nil && *req.Title != document.Title {
		document.Title = *req.Title
		updated = true
	}

	if req.Content != nil && *req.Content != document.Content {
		document.UpdateContent(*req.Content)
		document.ContentHash = s.calculateContentHash(*req.Content)
		updated = true
	}

	if req.Metadata != nil {
		document.Metadata = *req.Metadata
		updated = true
	}

	if len(req.Tags) > 0 {
		document.Tags = req.Tags
		updated = true
	}

	if len(req.Categories) > 0 {
		document.Categories = req.Categories
		updated = true
	}

	if req.Sensitivity != nil && *req.Sensitivity != document.Sensitivity {
		document.Sensitivity = *req.Sensitivity
		updated = true
	}

	if !updated {
		return document, nil
	}

	document.UpdatedAt = time.Now()

	if err := s.documentRepo.Update(ctx, document); err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	// Reprocess if content changed
	if req.Content != nil {
		go s.ReprocessDocument(context.Background(), id)
	}

	// Emit event
	if s.eventEmitter != nil {
		s.eventEmitter.EmitDocumentUpdated(ctx, document)
	}

	return document, nil
}

// DeleteDocument deletes a document
func (s *knowledgeService) DeleteDocument(ctx context.Context, id string, hard bool) error {
	document, err := s.documentRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get document: %w", err)
	}

	if hard {
		// Delete chunks
		s.chunkRepo.DeleteByDocumentID(ctx, id)

		// Delete document
		if err := s.documentRepo.Delete(ctx, id); err != nil {
			return fmt.Errorf("failed to delete document: %w", err)
		}
	} else {
		// Soft delete
		if err := s.documentRepo.SoftDelete(ctx, id); err != nil {
			return fmt.Errorf("failed to soft delete document: %w", err)
		}
	}

	// Emit event
	if s.eventEmitter != nil {
		s.eventEmitter.EmitDocumentDeleted(ctx, document)
	}

	return nil
}

// ReprocessDocument reprocesses a document
func (s *knowledgeService) ReprocessDocument(ctx context.Context, id string) error {
	processReq := ProcessDocumentRequest{
		DocumentID:      id,
		DetectPII:       true,
		AnalyzeQuality:  true,
		GenerateSummary: true,
	}

	_, err := s.ProcessDocument(ctx, processReq)
	return err
}

// GetDocumentChunks retrieves chunks for a document
func (s *knowledgeService) GetDocumentChunks(ctx context.Context, documentID string) ([]*Chunk, error) {
	chunks, err := s.chunkRepo.GetByDocumentID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunks: %w", err)
	}
	return chunks, nil
}

// GetDocumentStatistics retrieves document statistics
func (s *knowledgeService) GetDocumentStatistics(ctx context.Context, kbID string) (*DocumentStatistics, error) {
	stats, err := s.documentRepo.GetStatistics(ctx, kbID, NewDocumentFilter())
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}
	return stats, nil
}

// ============================================================================
// Batch Operations
// ============================================================================

// BatchProcessDocuments processes multiple documents
func (s *knowledgeService) BatchProcessDocuments(ctx context.Context, requests []ProcessDocumentRequest) ([]*ProcessDocumentResult, error) {
	results := make([]*ProcessDocumentResult, len(requests))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Limit concurrency

	for i, req := range requests {
		wg.Add(1)
		go func(index int, request ProcessDocumentRequest) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result, err := s.ProcessDocument(ctx, request)
			if err != nil {
				result = &ProcessDocumentResult{
					Errors: []string{err.Error()},
				}
			}
			results[index] = result
		}(i, req)
	}

	wg.Wait()
	return results, nil
}

// ============================================================================
// Validation
// ============================================================================

// ValidateDocument validates document before processing
func (s *knowledgeService) ValidateDocument(ctx context.Context, req UploadDocumentRequest) error {
	if req.KnowledgeBaseID == "" {
		return fmt.Errorf("knowledge base ID is required")
	}

	if req.Title == "" {
		return fmt.Errorf("title is required")
	}

	if req.Content == "" {
		return fmt.Errorf("content is required")
	}

	if req.Type == "" {
		return fmt.Errorf("document type is required")
	}

	if req.OwnerID == "" {
		return fmt.Errorf("owner ID is required")
	}

	if req.CreatorID == "" {
		return fmt.Errorf("creator ID is required")
	}

	// Validate content size
	maxSize := int64(100 * 1024 * 1024) // 100MB
	if int64(len(req.Content)) > maxSize {
		return fmt.Errorf("content exceeds maximum size of %d bytes", maxSize)
	}

	return nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// calculateContentHash calculates SHA-256 hash of content
func (s *knowledgeService) calculateContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// countWords counts words in text
func (s *knowledgeService) countWords(text string) int {
	words := strings.Fields(text)
	return len(words)
}

// estimateTokenCount estimates token count (rough approximation)
func (s *knowledgeService) estimateTokenCount(text string) int {
	// Simple estimation: ~4 characters per token
	return len(text) / 4
}

// batchChunks splits chunks into batches
func (s *knowledgeService) batchChunks(chunks []*Chunk, batchSize int) [][]*Chunk {
	batches := make([][]*Chunk, 0)
	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		batches = append(batches, chunks[i:end])
	}
	return batches
}

// buildContext builds context around a chunk
func (s *knowledgeService) buildContext(chunk *Chunk) string {
	context := chunk.Content
	if chunk.ContextBefore != "" {
		context = chunk.ContextBefore + "\n...\n" + context
	}
	if chunk.ContextAfter != "" {
		context = context + "\n...\n" + chunk.ContextAfter
	}
	return context
}

// ============================================================================
// External Service Interfaces
// ============================================================================

// DocumentParser defines document parsing interface
type DocumentParser interface {
	Parse(ctx context.Context, content string, docType DocumentType, options ParseOptions) (string, error)
}

// ParseOptions defines parsing options
type ParseOptions struct {
	ExtractMetadata bool
	PreserveFormat  bool
	Language        string
}

// DocumentChunker defines document chunking interface
type DocumentChunker interface {
	Chunk(ctx context.Context, content string, options ChunkOptions) ([]string, error)
}

// ChunkOptions defines chunking options
type ChunkOptions struct {
	ChunkSize    int
	ChunkOverlap int
	Strategy     ChunkStrategy
}

// ChunkStrategy defines chunking strategy
type ChunkStrategy string

const (
	ChunkStrategyFixed      ChunkStrategy = "fixed"
	ChunkStrategySentence   ChunkStrategy = "sentence"
	ChunkStrategyParagraph  ChunkStrategy = "paragraph"
	ChunkStrategySemantic   ChunkStrategy = "semantic"
)

// EmbeddingGenerator defines embedding generation interface
type EmbeddingGenerator interface {
	Generate(ctx context.Context, text string, options EmbedOptions) ([]float64, error)
	GenerateBatch(ctx context.Context, texts []string, options EmbedOptions) ([][]float64, error)
}

// EmbedOptions defines embedding options
type EmbedOptions struct {
	Model      string
	Dimensions int
	Normalize  bool
}

// PIIDetector defines PII detection interface
type PIIDetector interface {
	Detect(ctx context.Context, text string) ([]PIIEntity, error)
}

// QualityAnalyzer defines quality analysis interface
type QualityAnalyzer interface {
	AnalyzeDocument(content string) float64
	AnalyzeChunk(content string) float64
}

// KnowledgeEventEmitter defines event emission interface
type KnowledgeEventEmitter interface {
	EmitDocumentUploaded(ctx context.Context, document *Document)
	EmitDocumentProcessed(ctx context.Context, document *Document)
	EmitDocumentUpdated(ctx context.Context, document *Document)
	EmitDocumentDeleted(ctx context.Context, document *Document)
}

// ============================================================================
// Advanced Features
// ============================================================================

// DocumentDeduplicator handles document deduplication
type DocumentDeduplicator struct {
	repo DocumentRepository
}

// NewDocumentDeduplicator creates a new deduplicator
func NewDocumentDeduplicator(repo DocumentRepository) *DocumentDeduplicator {
	return &DocumentDeduplicator{repo: repo}
}

// FindDuplicates finds duplicate documents
func (d *DocumentDeduplicator) FindDuplicates(ctx context.Context, kbID string) (map[string][]string, error) {
	filter := NewDocumentFilter()
	documents, err := d.repo.GetByKnowledgeBaseID(ctx, kbID, filter)
	if err != nil {
		return nil, err
	}

	hashMap := make(map[string][]string)
	for _, doc := range documents {
		hashMap[doc.ContentHash] = append(hashMap[doc.ContentHash], doc.ID)
	}

	duplicates := make(map[string][]string)
	for hash, ids := range hashMap {
		if len(ids) > 1 {
			duplicates[hash] = ids
		}
	}

	return duplicates, nil
}

// ============================================================================
// Similarity Calculator
// ============================================================================

// SimilarityCalculator calculates similarity between embeddings
type SimilarityCalculator struct{}

// NewSimilarityCalculator creates a new similarity calculator
func NewSimilarityCalculator() *SimilarityCalculator {
	return &SimilarityCalculator{}
}

// CosineSimilarity calculates cosine similarity
func (c *SimilarityCalculator) CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// EuclideanDistance calculates Euclidean distance
func (c *SimilarityCalculator) EuclideanDistance(a, b []float64) float64 {
	if len(a) != len(b) {
		return math.MaxFloat64
	}

	var sum float64
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return math.Sqrt(sum)
}

// DotProduct calculates dot product
func (c *SimilarityCalculator) DotProduct(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var product float64
	for i := 0; i < len(a); i++ {
		product += a[i] * b[i]
	}

	return product
}

//Personal.AI order the ending
