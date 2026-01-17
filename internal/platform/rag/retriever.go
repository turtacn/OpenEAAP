// internal/platform/rag/retriever.go
package rag

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/openeeap/openeeap/internal/infrastructure/vector"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/pkg/errors"
	"go.opentelemetry.io/otel/codes"
)

// Retriever 定义检索器接口
type Retriever interface {
	// Retrieve 执行检索
	Retrieve(ctx context.Context, req *RetrieveRequest) ([]*RetrievedChunk, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// RetrieveRequest 定义检索请求
type RetrieveRequest struct {
	Query          string            // 查询文本
	CollectionName string            // 集合名称
	TopK           int               // 返回数量
	Mode           RetrievalMode     // 检索模式
	Metadata       map[string]string // 元数据过滤
	MinScore       float32           // 最小相关性分数
	TimeRange      *TimeRange        // 时间范围过滤
}

// TimeRange 定义时间范围
type TimeRange struct {
	Start time.Time // 开始时间
	End   time.Time // 结束时间
}

// retrieverImpl 实现检索器
type retrieverImpl struct {
	vectorStore   vector.VectorStore  // 向量数据库
	searchEngine  SearchEngine        // 全文搜索引擎（可选）
	knowledgeGraph KnowledgeGraph     // 知识图谱（可选）
	logger        logging.Logger      // 日志器
	tracer        trace.Tracer        // 追踪器
	config        *RetrieverConfig    // 配置
}

// RetrieverConfig 定义检索器配置
type RetrieverConfig struct {
	VectorWeight     float32 // 向量检索权重（混合模式）
	KeywordWeight    float32 // 关键词检索权重（混合模式）
	GraphWeight      float32 // 知识图谱检索权重（混合模式）
	DefaultMinScore  float32 // 默认最小分数
	MaxRetries       int     // 最大重试次数
	TimeoutSeconds   int     // 超时时间
	EnableCache      bool    // 是否启用缓存
	CacheTTLSeconds  int     // 缓存过期时间
}

// SearchEngine 定义全文搜索引擎接口
type SearchEngine interface {
	// Search 执行关键词搜索
	Search(ctx context.Context, query string, topK int, filters map[string]string) ([]*SearchResult, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// SearchResult 定义搜索结果
type SearchResult struct {
	ChunkID    string            // 块ID
	DocumentID string            // 文档ID
	Content    string            // 内容
	Score      float32           // BM25 分数
	Metadata   map[string]string // 元数据
}

// KnowledgeGraph 定义知识图谱接口
type KnowledgeGraph interface {
	// Query 执行图查询
	Query(ctx context.Context, query string, maxDepth int) ([]*GraphResult, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// GraphResult 定义图查询结果
type GraphResult struct {
	EntityID   string            // 实体ID
	EntityType string            // 实体类型
	Content    string            // 内容
	Relations  []string          // 关系
	Score      float32           // 相关性分数
	Metadata   map[string]string // 元数据
}

// NewRetriever 创建检索器实例
func NewRetriever(
	vectorStore vector.VectorStore,
	searchEngine SearchEngine,
	knowledgeGraph KnowledgeGraph,
	logger logging.Logger,
	tracer trace.Tracer,
	config *RetrieverConfig,
) Retriever {
	if config == nil {
		config = defaultRetrieverConfig()
	}

	return &retrieverImpl{
		vectorStore:    vectorStore,
		searchEngine:   searchEngine,
		knowledgeGraph: knowledgeGraph,
		logger:         logger,
		tracer:         tracer,
		config:         config,
	}
}

// Retrieve 执行检索
func (r *retrieverImpl) Retrieve(ctx context.Context, req *RetrieveRequest) ([]*RetrievedChunk, error) {
	startTime := time.Now()

	// 创建 Span
	ctx, span := r.tracer.Start(ctx, "Retriever.Retrieve")
	defer span.End()
	// span.AddTag("query", req.Query)
	// span.AddTag("mode", string(req.Mode))
	// span.AddTag("topK", req.TopK)

	// 验证请求
	if err := r.validateRequest(req); err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidArgument, "invalid retrieve request")
	}

	// 应用默认值
	if req.MinScore == 0 {
		req.MinScore = r.config.DefaultMinScore
	}

	var chunks []*RetrievedChunk
	var err error

	// 根据检索模式执行检索
	switch req.Mode {
	case RetrievalModeVector:
		chunks, err = r.vectorRetrieval(ctx, req)
	case RetrievalModeKeyword:
		chunks, err = r.keywordRetrieval(ctx, req)
	case RetrievalModeGraph:
		chunks, err = r.graphRetrieval(ctx, req)
	case RetrievalModeHybrid:
		chunks, err = r.hybridRetrieval(ctx, req)
	default:
		return nil, errors.ValidationError(
			fmt.Sprintf("unsupported retrieval mode: %s", req.Mode))
	}

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, err
	}

	// 过滤低分结果
	chunks = r.filterByScore(chunks, req.MinScore)

	// 截断到 TopK
	if len(chunks) > req.TopK {
		chunks = chunks[:req.TopK]
	}

	latency := time.Since(startTime)
 r.logger.WithContext(ctx).Info("retrieval completed", logging.Any("query", req.Query), logging.Any("mode", req.Mode), logging.Any("chunks_count", len(chunks)))

	// span.AddTag("chunks_count", len(chunks))

	return chunks, nil
}

// vectorRetrieval 向量检索
func (r *retrieverImpl) vectorRetrieval(ctx context.Context, req *RetrieveRequest) ([]*RetrievedChunk, error) {
	r.logger.WithContext(ctx).Debug("performing vector retrieval", logging.Any("query", req.Query))

	// 构建向量搜索请求
	searchReq := &vector.SearchRequest{
		CollectionName: req.CollectionName,
		QueryText:      req.Query,
		TopK:           req.TopK * 2, // 检索更多候选，后续过滤
		MetricType:     "COSINE",
		Filters:        r.buildVectorFilters(req),
	}

	// 执行向量搜索
	results, err := r.vectorStore.Search(ctx, searchReq)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "vector search failed")
	}

	// 转换为 RetrievedChunk
	chunks := make([]*RetrievedChunk, 0, len(results))
	for _, result := range results {
		chunk := &RetrievedChunk{
			ChunkID:    result.ID,
			DocumentID: result.Metadata["document_id"],
			Content:    result.Content,
			Score:      result.Score,
			Metadata:   result.Metadata,
			Source:     result.Metadata["source"],
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// keywordRetrieval 关键词检索
func (r *retrieverImpl) keywordRetrieval(ctx context.Context, req *RetrieveRequest) ([]*RetrievedChunk, error) {
	if r.searchEngine == nil {
		return nil, errors.InternalError( "search engine not configured")
	}

	r.logger.WithContext(ctx).Debug("performing keyword retrieval", logging.Any("query", req.Query))

	// 执行关键词搜索
	results, err := r.searchEngine.Search(ctx, req.Query, req.TopK*2, req.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "keyword search failed")
	}

	// 转换为 RetrievedChunk
	chunks := make([]*RetrievedChunk, 0, len(results))
	for _, result := range results {
		chunk := &RetrievedChunk{
			ChunkID:    result.ChunkID,
			DocumentID: result.DocumentID,
			Content:    result.Content,
			Score:      result.Score,
			Metadata:   result.Metadata,
			Source:     result.Metadata["source"],
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// graphRetrieval 知识图谱检索
func (r *retrieverImpl) graphRetrieval(ctx context.Context, req *RetrieveRequest) ([]*RetrievedChunk, error) {
	if r.knowledgeGraph == nil {
		return nil, errors.InternalError( "knowledge graph not configured")
	}

	r.logger.WithContext(ctx).Debug("performing graph retrieval", logging.Any("query", req.Query))

	// 执行图查询
	results, err := r.knowledgeGraph.Query(ctx, req.Query, 2) // 最大深度为2
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "graph query failed")
	}

	// 转换为 RetrievedChunk
	chunks := make([]*RetrievedChunk, 0, len(results))
	for _, result := range results {
		chunk := &RetrievedChunk{
			ChunkID:    result.EntityID,
			DocumentID: result.EntityID,
			Content:    r.formatGraphResult(result),
			Score:      result.Score,
			Metadata:   result.Metadata,
			Source:     fmt.Sprintf("知识图谱-%s", result.EntityType),
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// hybridRetrieval 混合检索
func (r *retrieverImpl) hybridRetrieval(ctx context.Context, req *RetrieveRequest) ([]*RetrievedChunk, error) {
	r.logger.WithContext(ctx).Debug("performing hybrid retrieval", logging.Any("query", req.Query))

	// 并行执行多种检索策略
	type retrievalResult struct {
		chunks []*RetrievedChunk
		weight float32
		err    error
	}

	resultChan := make(chan retrievalResult, 3)

	// 向量检索
	go func() {
		chunks, err := r.vectorRetrieval(ctx, req)
		resultChan <- retrievalResult{chunks: chunks, weight: r.config.VectorWeight, err: err}
	}()

	// 关键词检索（如果可用）
	if r.searchEngine != nil {
		go func() {
			chunks, err := r.keywordRetrieval(ctx, req)
			resultChan <- retrievalResult{chunks: chunks, weight: r.config.KeywordWeight, err: err}
		}()
	} else {
		resultChan <- retrievalResult{chunks: nil, weight: 0, err: nil}
	}

	// 知识图谱检索（如果可用）
	if r.knowledgeGraph != nil {
		go func() {
			chunks, err := r.graphRetrieval(ctx, req)
			resultChan <- retrievalResult{chunks: chunks, weight: r.config.GraphWeight, err: err}
		}()
	} else {
		resultChan <- retrievalResult{chunks: nil, weight: 0, err: nil}
	}

	// 收集结果
	var allChunks []*RetrievedChunk
	var weights []float32

	for i := 0; i < 3; i++ {
		result := <-resultChan
		if result.err != nil {
			r.logger.WithContext(ctx).Warn("retrieval strategy failed", logging.Any("error", result.err))
			continue
		}
		if result.chunks != nil {
			for _, chunk := range result.chunks {
				allChunks = append(allChunks, chunk)
				weights = append(weights, result.weight)
			}
		}
	}

	if len(allChunks) == 0 {
		return nil, errors.New("ERR_INTERNAL", "all retrieval strategies failed")
	}

	// 融合分数（RRF - Reciprocal Rank Fusion）
	fusedChunks := r.fuseResults(allChunks, weights)

	return fusedChunks, nil
}

// fuseResults 融合检索结果（RRF算法）
func (r *retrieverImpl) fuseResults(chunks []*RetrievedChunk, weights []float32) []*RetrievedChunk {
	// 使用 Reciprocal Rank Fusion 算法
	k := float32(60.0) // RRF 参数

	// 计算每个文档的融合分数
	scoreMap := make(map[string]float32)
	chunkMap := make(map[string]*RetrievedChunk)

	for i, chunk := range chunks {
		id := chunk.ChunkID
		weight := weights[i]

		// RRF 公式: 1 / (k + rank)
		rank := float32(i + 1)
		rrfScore := weight / (k + rank)

		if existing, ok := scoreMap[id]; ok {
			scoreMap[id] = existing + rrfScore
		} else {
			scoreMap[id] = rrfScore
			chunkMap[id] = chunk
		}
	}

	// 转换为切片并排序
	fusedChunks := make([]*RetrievedChunk, 0, len(scoreMap))
	for id, score := range scoreMap {
		chunk := chunkMap[id]
		chunk.Score = score // 更新为融合分数
		fusedChunks = append(fusedChunks, chunk)
	}

	// 按分数降序排序
	sort.Slice(fusedChunks, func(i, j int) bool {
		return fusedChunks[i].Score > fusedChunks[j].Score
	})

	return fusedChunks
}

// filterByScore 过滤低分结果
func (r *retrieverImpl) filterByScore(chunks []*RetrievedChunk, minScore float32) []*RetrievedChunk {
	filtered := make([]*RetrievedChunk, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk.Score >= minScore {
			filtered = append(filtered, chunk)
		}
	}
	return filtered
}

// buildVectorFilters 构建向量搜索过滤器
func (r *retrieverImpl) buildVectorFilters(req *RetrieveRequest) map[string]interface{} {
	filters := make(map[string]interface{})

	// 添加元数据过滤
	for key, value := range req.Metadata {
		filters[key] = value
	}

	// 添加时间范围过滤
	if req.TimeRange != nil {
		filters["created_at"] = map[string]interface{}{
			"$gte": req.TimeRange.Start,
			"$lte": req.TimeRange.End,
		}
	}

	return filters
}

// formatGraphResult 格式化知识图谱结果
func (r *retrieverImpl) formatGraphResult(result *GraphResult) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("实体: %s (类型: %s)\n", result.EntityID, result.EntityType))
	builder.WriteString(fmt.Sprintf("内容: %s\n", result.Content))

	if len(result.Relations) > 0 {
		builder.WriteString("关系: ")
		builder.WriteString(strings.Join(result.Relations, ", "))
		builder.WriteString("\n")
	}

	return builder.String()
}

// validateRequest 验证请求
func (r *retrieverImpl) validateRequest(req *RetrieveRequest) error {
	if req.Query == "" {
		return fmt.Errorf("query is required")
	}
	if req.CollectionName == "" {
		return fmt.Errorf("collection name is required")
	}
	if req.TopK <= 0 || req.TopK > 100 {
		return fmt.Errorf("topK must be between 1 and 100")
	}
	return nil
}

// HealthCheck 健康检查
func (r *retrieverImpl) HealthCheck(ctx context.Context) error {
	// 检查向量数据库
	if err := r.vectorStore.HealthCheck(ctx); err != nil {
		return errors.Wrap(err, "ERR_UNAVAIL", "vector store unhealthy")
	}

	// 检查搜索引擎（可选）
	if r.searchEngine != nil {
		if err := r.searchEngine.HealthCheck(ctx); err != nil {
			r.logger.WithContext(ctx).Warn("search engine unhealthy", logging.Error(err))
		}
	}

	// 检查知识图谱（可选）
	if r.knowledgeGraph != nil {
		if err := r.knowledgeGraph.HealthCheck(ctx); err != nil {
			r.logger.WithContext(ctx).Warn("knowledge graph unhealthy", logging.Error(err))
		}
	}

	return nil
}

// defaultRetrieverConfig 返回默认配置
func defaultRetrieverConfig() *RetrieverConfig {
	return &RetrieverConfig{
		VectorWeight:    0.6,  // 向量检索权重
		KeywordWeight:   0.3,  // 关键词检索权重
		GraphWeight:     0.1,  // 知识图谱检索权重
		DefaultMinScore: 0.5,  // 最小分数
		MaxRetries:      3,    // 最大重试次数
		TimeoutSeconds:  10,   // 超时时间
		EnableCache:     true, // 启用缓存
		CacheTTLSeconds: 300,  // 缓存5分钟
	}
}

//Personal.AI order the ending
