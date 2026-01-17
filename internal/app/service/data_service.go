package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/openeeap/openeeap/internal/app/dto"
	"github.com/openeeap/openeeap/internal/domain/knowledge"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/internal/platform/rag"
	"github.com/openeeap/openeeap/pkg/errors"
	"github.com/openeeap/openeeap/pkg/types"
)

// DataService 数据应用服务接口
type DataService interface {
	// UploadDocument 上传文档
	UploadDocument(ctx context.Context, req *dto.UploadDocumentRequest) (*dto.DocumentResponse, error)

	// UploadDocumentStream 流式上传文档
	UploadDocumentStream(ctx context.Context, reader io.Reader, metadata *dto.DocumentMetadata) (*dto.DocumentResponse, error)

	// GetDocumentByID 根据 ID 获取文档
	GetDocumentByID(ctx context.Context, id string) (*dto.DocumentResponse, error)

	// ListDocuments 列出文档
	ListDocuments(ctx context.Context, req *dto.ListDocumentsRequest) (*dto.DocumentListResponse, error)

	// UpdateDocument 更新文档元数据
	UpdateDocument(ctx context.Context, id string, req *dto.UpdateDocumentRequest) (*dto.DocumentResponse, error)

	// DeleteDocument 删除文档
	DeleteDocument(ctx context.Context, id string) error

	// ProcessDocument 处理文档（分块、向量化、索引）
	ProcessDocument(ctx context.Context, id string) (*dto.ProcessingStatus, error)

	// GetProcessingStatus 获取处理状态
	GetProcessingStatus(ctx context.Context, id string) (*dto.ProcessingStatus, error)

	// SearchDocuments 搜索文档
	SearchDocuments(ctx context.Context, req *dto.SearchDocumentsRequest) (*dto.SearchResponse, error)

	// SemanticSearch 语义搜索
	SemanticSearch(ctx context.Context, req *dto.SemanticSearchRequest) (*dto.SemanticSearchResponse, error)

	// HybridSearch 混合搜索（向量 + 关键词）
	HybridSearch(ctx context.Context, req *dto.HybridSearchRequest) (*dto.SearchResponse, error)

	// CreateCollection 创建文档集合
	CreateCollection(ctx context.Context, req *dto.CreateCollectionRequest) (*dto.CollectionResponse, error)

	// GetCollection 获取集合信息
	GetCollection(ctx context.Context, id string) (*dto.CollectionResponse, error)

	// ListCollections 列出集合
	ListCollections(ctx context.Context, req *dto.ListCollectionsRequest) (*dto.CollectionListResponse, error)

	// DeleteCollection 删除集合
	DeleteCollection(ctx context.Context, id string) error

	// GetStatistics 获取数据统计
	GetStatistics(ctx context.Context) (*dto.DataStatistics, error)
}

// dataService 数据应用服务实现
type dataService struct {
	documentRepo    knowledge.DocumentRepository
	chunkRepo       knowledge.ChunkRepository
	knowledgeDomain knowledge.KnowledgeService
	ragEngine       rag.RAGEngine
	logger          logging.Logger
	tracer          trace.Tracer
	metrics         metrics.MetricsCollector
}

// NewDataService 创建数据应用服务
func NewDataService(
	documentRepo knowledge.DocumentRepository,
	chunkRepo knowledge.ChunkRepository,
	knowledgeDomain knowledge.KnowledgeService,
	ragEngine rag.RAGEngine,
	logger logging.Logger,
	tracer trace.Tracer,
	metrics metrics.MetricsCollector,
) DataService {
	return &dataService{
		documentRepo:    documentRepo,
		chunkRepo:       chunkRepo,
		knowledgeDomain: knowledgeDomain,
		ragEngine:       ragEngine,
		logger:          logger,
		tracer:          tracer,
		metrics:         metrics,
	}
}

// UploadDocument 上传文档
func (s *dataService) UploadDocument(ctx context.Context, req *dto.UploadDocumentRequest) (*dto.DocumentResponse, error) {
	span := s.tracer.StartSpan(ctx, "DataService.UploadDocument")
	defer span.End()

	s.logger.InfoCtx(ctx, "Uploading document", "filename", req.Filename, "type", req.ContentType)

	// 转换 DTO 为领域实体
	doc := &knowledge.Document{
		ID:           types.NewID(),
		Title:        req.Title,
		Filename:     req.Filename,
		ContentType:  req.ContentType,
		Content:      req.Content,
		Metadata:     s.convertMetadata(req.Metadata),
		CollectionID: req.CollectionID,
		Source:       req.Source,
		Status:       knowledge.StatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 领域验证
	if err := s.knowledgeDomain.ValidateDocument(ctx, doc); err != nil {
		s.logger.ErrorCtx(ctx, "Document validation failed", "error", err)
		return nil, errors.Wrap(err, errors.CodeValidationFailed, "document validation failed")
	}

	// 持久化
	if err := s.documentRepo.Create(ctx, doc); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to upload document", "error", err)
		s.metrics.IncrementCounter("document_upload_errors", nil)
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to upload document")
	}

	s.metrics.IncrementCounter("documents_uploaded", map[string]string{
		"type": req.ContentType,
	})
	s.logger.InfoCtx(ctx, "Document uploaded successfully", "id", doc.ID)

	// 异步触发文档处理
	if req.AutoProcess {
		go func() {
			processCtx := context.Background()
			if _, err := s.ProcessDocument(processCtx, doc.ID); err != nil {
				s.logger.ErrorCtx(processCtx, "Auto-processing failed", "doc_id", doc.ID, "error", err)
			}
		}()
	}

	return s.toDocumentResponse(doc), nil
}

// UploadDocumentStream 流式上传文档
func (s *dataService) UploadDocumentStream(ctx context.Context, reader io.Reader, metadata *dto.DocumentMetadata) (*dto.DocumentResponse, error) {
	span := s.tracer.StartSpan(ctx, "DataService.UploadDocumentStream")
	defer span.End()

	s.logger.InfoCtx(ctx, "Uploading document stream", "filename", metadata.Filename)

	// 读取流式内容
	content, err := io.ReadAll(reader)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Failed to read document stream", "error", err)
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to read document stream")
	}

	doc := &knowledge.Document{
		ID:           types.NewID(),
		Title:        metadata.Title,
		Filename:     metadata.Filename,
		ContentType:  metadata.ContentType,
		Content:      string(content),
		Metadata:     s.convertMetadata(metadata),
		CollectionID: metadata.CollectionID,
		Source:       metadata.Source,
		Status:       knowledge.StatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.documentRepo.Create(ctx, doc); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to upload document stream", "error", err)
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to upload document stream")
	}

	s.metrics.IncrementCounter("documents_uploaded_stream", nil)
	s.logger.InfoCtx(ctx, "Document stream uploaded successfully", "id", doc.ID)

	return s.toDocumentResponse(doc), nil
}

// GetDocumentByID 根据 ID 获取文档
func (s *dataService) GetDocumentByID(ctx context.Context, id string) (*dto.DocumentResponse, error) {
	span := s.tracer.StartSpan(ctx, "DataService.GetDocumentByID")
	defer span.End()

	doc, err := s.documentRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Failed to get document", "id", id, "error", err)
		return nil, errors.Wrap(err, errors.CodeNotFound, "document not found")
	}

	return s.toDocumentResponse(doc), nil
}

// ListDocuments 列出文档
func (s *dataService) ListDocuments(ctx context.Context, req *dto.ListDocumentsRequest) (*dto.DocumentListResponse, error) {
	span := s.tracer.StartSpan(ctx, "DataService.ListDocuments")
	defer span.End()

	offset := (req.Page - 1) * req.PageSize
	docs, total, err := s.documentRepo.List(ctx, req.CollectionID, req.Status, req.Search, offset, req.PageSize)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Failed to list documents", "error", err)
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to list documents")
	}

	items := make([]*dto.DocumentResponse, len(docs))
	for i, doc := range docs {
		items[i] = s.toDocumentResponse(doc)
	}

	return &dto.DocumentListResponse{
		Items: items,
		Pagination: dto.Pagination{
			Page:     req.Page,
			PageSize: req.PageSize,
			Total:    total,
		},
	}, nil
}

// UpdateDocument 更新文档元数据
func (s *dataService) UpdateDocument(ctx context.Context, id string, req *dto.UpdateDocumentRequest) (*dto.DocumentResponse, error) {
	span := s.tracer.StartSpan(ctx, "DataService.UpdateDocument")
	defer span.End()

	s.logger.InfoCtx(ctx, "Updating document", "id", id)

	doc, err := s.documentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "document not found")
	}

	// 更新字段
	if req.Title != nil {
		doc.Title = *req.Title
	}
	if req.Metadata != nil {
		doc.Metadata = s.convertMetadata(req.Metadata)
	}
	if req.CollectionID != nil {
		doc.CollectionID = *req.CollectionID
	}
	doc.UpdatedAt = time.Now()

	if err := s.documentRepo.Update(ctx, doc); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to update document", "error", err)
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to update document")
	}

	s.logger.InfoCtx(ctx, "Document updated successfully", "id", id)
	return s.toDocumentResponse(doc), nil
}

// DeleteDocument 删除文档
func (s *dataService) DeleteDocument(ctx context.Context, id string) error {
	span := s.tracer.StartSpan(ctx, "DataService.DeleteDocument")
	defer span.End()

	s.logger.InfoCtx(ctx, "Deleting document", "id", id)

	// 删除文档的所有分块
	if err := s.chunkRepo.DeleteByDocumentID(ctx, id); err != nil {
		s.logger.WarnCtx(ctx, "Failed to delete document chunks", "doc_id", id, "error", err)
	}

	if err := s.documentRepo.Delete(ctx, id); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to delete document", "error", err)
		return errors.Wrap(err, errors.CodeDatabaseError, "failed to delete document")
	}

	s.metrics.IncrementCounter("documents_deleted", nil)
	s.logger.InfoCtx(ctx, "Document deleted successfully", "id", id)
	return nil
}

// ProcessDocument 处理文档（分块、向量化、索引）
func (s *dataService) ProcessDocument(ctx context.Context, id string) (*dto.ProcessingStatus, error) {
	span := s.tracer.StartSpan(ctx, "DataService.ProcessDocument")
	defer span.End()

	startTime := time.Now()
	s.logger.InfoCtx(ctx, "Processing document", "id", id)

	// 获取文档
	doc, err := s.documentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "document not found")
	}

	// 更新状态为处理中
	doc.Status = knowledge.StatusProcessing
	doc.UpdatedAt = time.Now()
	if err := s.documentRepo.Update(ctx, doc); err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to update document status")
	}

	// 调用知识领域服务处理文档
	result, err := s.knowledgeDomain.ProcessDocument(ctx, doc)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Document processing failed", "id", id, "error", err)
		doc.Status = knowledge.StatusFailed
		doc.ProcessingError = err.Error()
		s.documentRepo.Update(ctx, doc)
		s.metrics.IncrementCounter("document_processing_errors", nil)
		return nil, errors.Wrap(err, errors.CodeExecutionError, "document processing failed")
	}

	// 更新状态为已处理
	doc.Status = knowledge.StatusProcessed
	doc.ChunkCount = result.ChunkCount
	doc.ProcessedAt = time.Now()
	doc.UpdatedAt = time.Now()
	if err := s.documentRepo.Update(ctx, doc); err != nil {
		s.logger.WarnCtx(ctx, "Failed to update document status after processing", "error", err)
	}

	duration := time.Since(startTime)
	s.metrics.RecordHistogram("document_processing_duration_seconds", duration.Seconds(), map[string]string{
		"content_type": doc.ContentType,
	})

	s.logger.InfoCtx(ctx, "Document processed successfully",
		"id", id,
		"chunks", result.ChunkCount,
		"duration_ms", duration.Milliseconds(),
	)

	return &dto.ProcessingStatus{
		DocumentID:  id,
		Status:      string(doc.Status),
		ChunkCount:  result.ChunkCount,
		PIIDetected: result.PIIDetected,
		ProcessedAt: doc.ProcessedAt,
		Duration:    duration.Milliseconds(),
	}, nil
}

// GetProcessingStatus 获取处理状态
func (s *dataService) GetProcessingStatus(ctx context.Context, id string) (*dto.ProcessingStatus, error) {
	span := s.tracer.StartSpan(ctx, "DataService.GetProcessingStatus")
	defer span.End()

	doc, err := s.documentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "document not found")
	}

	var duration int64
	if !doc.ProcessedAt.IsZero() {
		duration = doc.ProcessedAt.Sub(doc.CreatedAt).Milliseconds()
	}

	return &dto.ProcessingStatus{
		DocumentID:  id,
		Status:      string(doc.Status),
		ChunkCount:  doc.ChunkCount,
		ProcessedAt: doc.ProcessedAt,
		Error:       doc.ProcessingError,
		Duration:    duration,
	}, nil
}

// SearchDocuments 搜索文档
func (s *dataService) SearchDocuments(ctx context.Context, req *dto.SearchDocumentsRequest) (*dto.SearchResponse, error) {
	span := s.tracer.StartSpan(ctx, "DataService.SearchDocuments")
	defer span.End()

	s.logger.InfoCtx(ctx, "Searching documents", "query", req.Query)

	offset := (req.Page - 1) * req.PageSize
	docs, total, err := s.documentRepo.Search(ctx, req.Query, req.CollectionID, offset, req.PageSize)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Document search failed", "error", err)
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "document search failed")
	}

	items := make([]*dto.SearchResult, len(docs))
	for i, doc := range docs {
		items[i] = &dto.SearchResult{
			DocumentID: doc.ID,
			Title:      doc.Title,
			Filename:   doc.Filename,
			Excerpt:    s.generateExcerpt(doc.Content, req.Query, 200),
			Score:      1.0, // 关键词搜索默认分数
			Highlights: []string{},
		}
	}

	return &dto.SearchResponse{
		Items: items,
		Pagination: dto.Pagination{
			Page:     req.Page,
			PageSize: req.PageSize,
			Total:    total,
		},
		Query: req.Query,
	}, nil
}

// SemanticSearch 语义搜索
func (s *dataService) SemanticSearch(ctx context.Context, req *dto.SemanticSearchRequest) (*dto.SemanticSearchResponse, error) {
	span := s.tracer.StartSpan(ctx, "DataService.SemanticSearch")
	defer span.End()

	startTime := time.Now()
	s.logger.InfoCtx(ctx, "Performing semantic search", "query", req.Query)

	// 调用 RAG 引擎进行语义检索
	ragReq := &rag.RetrievalRequest{
		Query:        req.Query,
		CollectionID: req.CollectionID,
		TopK:         req.TopK,
		MinScore:     req.MinScore,
		Filters:      req.Filters,
	}

	results, err := s.ragEngine.Retrieve(ctx, ragReq)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Semantic search failed", "error", err)
		s.metrics.IncrementCounter("semantic_search_errors", nil)
		return nil, errors.Wrap(err, errors.CodeExecutionError, "semantic search failed")
	}

	items := make([]*dto.SemanticResult, len(results))
	for i, result := range results {
		items[i] = &dto.SemanticResult{
			ChunkID:    result.ChunkID,
			DocumentID: result.DocumentID,
			Content:    result.Content,
			Score:      result.Score,
			Metadata:   result.Metadata,
		}
	}

	duration := time.Since(startTime)
	s.metrics.RecordHistogram("semantic_search_duration_seconds", duration.Seconds(), nil)

	s.logger.InfoCtx(ctx, "Semantic search completed",
		"results", len(items),
		"duration_ms", duration.Milliseconds(),
	)

	return &dto.SemanticSearchResponse{
		Items:    items,
		Query:    req.Query,
		Duration: duration.Milliseconds(),
	}, nil
}

// HybridSearch 混合搜索（向量 + 关键词）
func (s *dataService) HybridSearch(ctx context.Context, req *dto.HybridSearchRequest) (*dto.SearchResponse, error) {
	span := s.tracer.StartSpan(ctx, "DataService.HybridSearch")
	defer span.End()

	startTime := time.Now()
	s.logger.InfoCtx(ctx, "Performing hybrid search", "query", req.Query)

	// 调用 RAG 引擎进行混合检索
	ragReq := &rag.HybridRetrievalRequest{
		Query:         req.Query,
		CollectionID:  req.CollectionID,
		TopK:          req.TopK,
		VectorWeight:  req.VectorWeight,
		KeywordWeight: req.KeywordWeight,
		MinScore:      req.MinScore,
	}

	results, err := s.ragEngine.HybridRetrieve(ctx, ragReq)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Hybrid search failed", "error", err)
		s.metrics.IncrementCounter("hybrid_search_errors", nil)
		return nil, errors.Wrap(err, errors.CodeExecutionError, "hybrid search failed")
	}

	items := make([]*dto.SearchResult, len(results))
	for i, result := range results {
		items[i] = &dto.SearchResult{
			DocumentID: result.DocumentID,
			Title:      result.Title,
			Filename:   result.Filename,
			Excerpt:    result.Content,
			Score:      result.Score,
			Highlights: result.Highlights,
		}
	}

	duration := time.Since(startTime)
	s.metrics.RecordHistogram("hybrid_search_duration_seconds", duration.Seconds(), nil)

	s.logger.InfoCtx(ctx, "Hybrid search completed",
		"results", len(items),
		"duration_ms", duration.Milliseconds(),
	)

	return &dto.SearchResponse{
		Items: items,
		Pagination: dto.Pagination{
			Page:     1,
			PageSize: req.TopK,
			Total:    int64(len(items)),
		},
		Query: req.Query,
	}, nil
}

// CreateCollection 创建文档集合
func (s *dataService) CreateCollection(ctx context.Context, req *dto.CreateCollectionRequest) (*dto.CollectionResponse, error) {
	span := s.tracer.StartSpan(ctx, "DataService.CreateCollection")
	defer span.End()

	s.logger.InfoCtx(ctx, "Creating collection", "name", req.Name)

	collection := &knowledge.Collection{
		ID:          types.NewID(),
		Name:        req.Name,
		Description: req.Description,
		Config:      s.convertCollectionConfig(req.Config),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.documentRepo.CreateCollection(ctx, collection); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to create collection", "error", err)
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to create collection")
	}

	s.metrics.IncrementCounter("collections_created", nil)
	s.logger.InfoCtx(ctx, "Collection created successfully", "id", collection.ID)

	return s.toCollectionResponse(collection), nil
}

// GetCollection 获取集合信息
func (s *dataService) GetCollection(ctx context.Context, id string) (*dto.CollectionResponse, error) {
	span := s.tracer.StartSpan(ctx, "DataService.GetCollection")
	defer span.End()

	collection, err := s.documentRepo.GetCollection(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "collection not found")
	}

	return s.toCollectionResponse(collection), nil
}

// ListCollections 列出集合
func (s *dataService) ListCollections(ctx context.Context, req *dto.ListCollectionsRequest) (*dto.CollectionListResponse, error) {
	span := s.tracer.StartSpan(ctx, "DataService.ListCollections")
	defer span.End()

	offset := (req.Page - 1) * req.PageSize
	collections, total, err := s.documentRepo.ListCollections(ctx, req.Search, offset, req.PageSize)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Failed to list collections", "error", err)
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to list collections")
	}

	items := make([]*dto.CollectionResponse, len(collections))
	for i, collection := range collections {
		items[i] = s.toCollectionResponse(collection)
	}

	return &dto.CollectionListResponse{
		Items: items,
		Pagination: dto.Pagination{
			Page:     req.Page,
			PageSize: req.PageSize,
			Total:    total,
		},
	}, nil
}

// DeleteCollection 删除集合
func (s *dataService) DeleteCollection(ctx context.Context, id string) error {
	span := s.tracer.StartSpan(ctx, "DataService.DeleteCollection")
	defer span.End()

	s.logger.InfoCtx(ctx, "Deleting collection", "id", id)

	// 检查集合中是否有文档
	docCount, err := s.documentRepo.CountByCollection(ctx, id)
	if err != nil {
		return errors.Wrap(err, errors.CodeDatabaseError, "failed to count documents")
	}

	if docCount > 0 {
		return errors.New(errors.CodeBusinessLogicError, "cannot delete collection with documents")
	}

	if err := s.documentRepo.DeleteCollection(ctx, id); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to delete collection", "error", err)
		return errors.Wrap(err, errors.CodeDatabaseError, "failed to delete collection")
	}

	s.metrics.IncrementCounter("collections_deleted", nil)
	s.logger.InfoCtx(ctx, "Collection deleted successfully", "id", id)
	return nil
}

// GetStatistics 获取数据统计
func (s *dataService) GetStatistics(ctx context.Context) (*dto.DataStatistics, error) {
	span := s.tracer.StartSpan(ctx, "DataService.GetStatistics")
	defer span.End()

	stats, err := s.documentRepo.GetStatistics(ctx)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Failed to get statistics", "error", err)
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get statistics")
	}

	return &dto.DataStatistics{
		TotalDocuments:     stats.TotalDocuments,
		ProcessedDocuments: stats.ProcessedDocuments,
		TotalChunks:        stats.TotalChunks,
		TotalCollections:   stats.TotalCollections,
		StorageSizeBytes:   stats.StorageSizeBytes,
		LastUpdated:        time.Now(),
	}, nil
}

// 辅助方法：转换元数据
func (s *dataService) convertMetadata(dtoMetadata *dto.DocumentMetadata) map[string]interface{} {
	if dtoMetadata == nil {
		return make(map[string]interface{})
	}
	return map[string]interface{}{
		"filename":      dtoMetadata.Filename,
		"content_type":  dtoMetadata.ContentType,
		"title":         dtoMetadata.Title,
		"source":        dtoMetadata.Source,
		"collection_id": dtoMetadata.CollectionID,
	}
}

// 辅助方法：转换集合配置
func (s *dataService) convertCollectionConfig(dtoConfig *dto.CollectionConfig) *knowledge.CollectionConfig {
	if dtoConfig == nil {
		return &knowledge.CollectionConfig{}
	}
	return &knowledge.CollectionConfig{
		ChunkSize:      dtoConfig.ChunkSize,
		ChunkOverlap:   dtoConfig.ChunkOverlap,
		EmbeddingModel: dtoConfig.EmbeddingModel,
		EnablePII:      dtoConfig.EnablePII,
	}
}

// 辅助方法：生成摘要
func (s *dataService) generateExcerpt(content string, query string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}
	// 简单截取，实际应实现更智能的摘要算法
	return content[:maxLength] + "..."
}

// 辅助方法：转换为文档响应 DTO
func (s *dataService) toDocumentResponse(doc *knowledge.Document) *dto.DocumentResponse {
	return &dto.DocumentResponse{
		ID:           doc.ID,
		Title:        doc.Title,
		Filename:     doc.Filename,
		ContentType:  doc.ContentType,
		CollectionID: doc.CollectionID,
		Source:       doc.Source,
		Status:       string(doc.Status),
		ChunkCount:   doc.ChunkCount,
		CreatedAt:    doc.CreatedAt,
		UpdatedAt:    doc.UpdatedAt,
		ProcessedAt:  doc.ProcessedAt,
	}
}

// 辅助方法：转换为集合响应 DTO
func (s *dataService) toCollectionResponse(collection *knowledge.Collection) *dto.CollectionResponse {
	return &dto.CollectionResponse{
		ID:          collection.ID,
		Name:        collection.Name,
		Description: collection.Description,
		Config:      s.convertCollectionConfigToDTO(collection.Config),
		CreatedAt:   collection.CreatedAt,
		UpdatedAt:   collection.UpdatedAt,
	}
}

// 辅助方法：转换集合配置为 DTO
func (s *dataService) convertCollectionConfigToDTO(config *knowledge.CollectionConfig) *dto.CollectionConfig {
	if config == nil {
		return nil
	}
	return &dto.CollectionConfig{
		ChunkSize:      config.ChunkSize,
		ChunkOverlap:   config.ChunkOverlap,
		EmbeddingModel: config.EmbeddingModel,
		EnablePII:      config.EnablePII,
	}
}

//Personal.AI order the ending
