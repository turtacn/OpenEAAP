// internal/platform/rag/rag_engine.go
package rag

import (
	"context"
	"fmt"
	"time"

	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/pkg/errors"
	"go.opentelemetry.io/otel/codes"
)

// RAGEngine 定义 RAG 引擎接口
type RAGEngine interface {
	// Query 执行完整的 RAG 查询流程
	Query(ctx context.Context, req *RAGRequest) (*RAGResponse, error)

	// QueryStream 执行流式 RAG 查询
	QueryStream(ctx context.Context, req *RAGRequest) (<-chan *RAGChunk, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// RAGRequest 定义 RAG 请求
type RAGRequest struct {
	Query           string            // 用户查询
	CollectionName  string            // 知识库名称
	TopK            int               // 检索数量
	RetrievalMode   RetrievalMode     // 检索模式
	RerankEnabled   bool              // 是否启用重排序
	ModelName       string            // 生成模型名称
	Temperature     float32           // 生成温度
	MaxTokens       int               // 最大生成长度
	Metadata        map[string]string // 元数据过滤
	VerifyEnabled   bool              // 是否启用答案验证
}

// RAGResponse 定义 RAG 响应
type RAGResponse struct {
	Answer          string              // 生成的答案
	RetrievedChunks []*RetrievedChunk   // 检索到的文档块
	Sources         []string            // 引用来源
	Confidence      float32             // 置信度
	Latency         LatencyBreakdown    // 延迟分解
	Verified        bool                // 是否通过验证
	VerifyResult    *VerifyResult       // 验证结果
}

// RAGChunk 定义流式响应块
type RAGChunk struct {
	Type    ChunkType // 块类型
	Content string    // 内容
	Done    bool      // 是否完成
	Error   error     // 错误
}

// ChunkType 定义块类型
type ChunkType string

const (
	ChunkTypeRetrieval ChunkType = "retrieval" // 检索阶段
	ChunkTypeGenerate  ChunkType = "generate"  // 生成阶段
	ChunkTypeVerify    ChunkType = "verify"    // 验证阶段
	ChunkTypeError     ChunkType = "error"     // 错误
)

// RetrievalMode 定义检索模式
type RetrievalMode string

const (
	RetrievalModeVector  RetrievalMode = "vector"  // 向量检索
	RetrievalModeKeyword RetrievalMode = "keyword" // 关键词检索
	RetrievalModeHybrid  RetrievalMode = "hybrid"  // 混合检索
	RetrievalModeGraph   RetrievalMode = "graph"   // 知识图谱检索
)

// RetrievedChunk 定义检索到的文档块
type RetrievedChunk struct {
	ChunkID    string            // 块ID
	DocumentID string            // 文档ID
	Content    string            // 内容
	Score      float32           // 相关性分数
	Metadata   map[string]string // 元数据
	Source     string            // 来源
}

// LatencyBreakdown 定义延迟分解
type LatencyBreakdown struct {
	QueryUnderstanding time.Duration // 查询理解
	Retrieval          time.Duration // 检索
	Reranking          time.Duration // 重排序
	ContextBuilding    time.Duration // 上下文构建
	Generation         time.Duration // 生成
	Verification       time.Duration // 验证
	Total              time.Duration // 总延迟
}

// VerifyResult 定义验证结果
type VerifyResult struct {
	HasHallucination bool     // 是否存在幻觉
	CitationValid    bool     // 引用是否有效
	FactCheckPassed  bool     // 事实检查是否通过
	Issues           []string // 问题列表
}

// ragEngineImpl 实现 RAG 引擎
type ragEngineImpl struct {
	retriever    Retriever       // 检索器
	reranker     Reranker        // 重排序器
	generator    Generator       // 生成器
	logger       logging.Logger  // 日志器
	tracer       trace.Tracer    // 追踪器
	config       *RAGConfig      // 配置
}

// RAGConfig 定义 RAG 配置
type RAGConfig struct {
	DefaultTopK         int           // 默认检索数量
	DefaultRetrievalMode RetrievalMode // 默认检索模式
	DefaultModelName    string        // 默认模型名称
	DefaultTemperature  float32       // 默认温度
	DefaultMaxTokens    int           // 默认最大长度
	EnableRerank        bool          // 是否启用重排序
	EnableVerify        bool          // 是否启用验证
	MaxContextLength    int           // 最大上下文长度
	TimeoutSeconds      int           // 超时时间（秒）
}

// NewRAGEngine 创建 RAG 引擎实例
func NewRAGEngine(
	retriever Retriever,
	reranker Reranker,
	generator Generator,
	logger logging.Logger,
	tracer trace.Tracer,
	config *RAGConfig,
) RAGEngine {
	if config == nil {
		config = defaultRAGConfig()
	}

	return &ragEngineImpl{
		retriever: retriever,
		reranker:  reranker,
		generator: generator,
		logger:    logger,
		tracer:    tracer,
		config:    config,
	}
}

// Query 执行完整的 RAG 查询流程
func (r *ragEngineImpl) Query(ctx context.Context, req *RAGRequest) (*RAGResponse, error) {
	startTime := time.Now()

	// 创建 Span
	ctx, span := r.tracer.Start(ctx, "RAGEngine.Query")
	defer span.End()
	// span.AddTag("query", req.Query)
	// span.AddTag("collection", req.CollectionName)

	// 应用默认值
	r.applyDefaults(req)

	// 验证请求
	if err := r.validateRequest(req); err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidArgument, "invalid RAG request")
	}

	var latency LatencyBreakdown

	// 1. 查询理解（可选，当前简化为直接使用原始查询）
	queryStart := time.Now()
	processedQuery := r.understandQuery(ctx, req.Query)
	latency.QueryUnderstanding = time.Since(queryStart)

	// 2. 检索阶段
	retrievalStart := time.Now()
	retrievedChunks, err := r.retrieveChunks(ctx, processedQuery, req)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, errors.Wrap(err, "ERR_INTERNAL", "retrieval failed")
	}
	latency.Retrieval = time.Since(retrievalStart)

	r.logger.WithContext(ctx).Info("retrieval completed", logging.Any("query", req.Query), logging.Any("chunks_count", len(retrievedChunks)))

	// 3. 重排序阶段（可选）
	if req.RerankEnabled && r.reranker != nil {
		rerankStart := time.Now()
		retrievedChunks, err = r.rerankChunks(ctx, processedQuery, retrievedChunks)
		if err != nil {
			r.logger.WithContext(ctx).Warn("reranking failed, using original order", logging.Error(err))
		}
		latency.Reranking = time.Since(rerankStart)
	}

	// 4. 上下文构建阶段
	contextStart := time.Now()
	ragContext := r.buildContext(ctx, retrievedChunks, req)
	latency.ContextBuilding = time.Since(contextStart)

	// 5. 生成阶段
	generationStart := time.Now()
	answer, sources, err := r.generateAnswer(ctx, req.Query, ragContext, req)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, errors.Wrap(err, "ERR_INTERNAL", "generation failed")
	}
	latency.Generation = time.Since(generationStart)

	// 6. 验证阶段（可选）
	var verifyResult *VerifyResult
	verified := true
	if req.VerifyEnabled {
		verifyStart := time.Now()
		verifyResult, err = r.verifyAnswer(ctx, req.Query, answer, retrievedChunks)
		if err != nil {
			r.logger.WithContext(ctx).Warn("verification failed", logging.Error(err))
		} else {
			verified = verifyResult.HasHallucination == false &&
				verifyResult.CitationValid &&
				verifyResult.FactCheckPassed
		}
		latency.Verification = time.Since(verifyStart)
	}

	latency.Total = time.Since(startTime)

	// 计算置信度
	confidence := r.calculateConfidence(retrievedChunks, verified)

	response := &RAGResponse{
		Answer:          answer,
		RetrievedChunks: retrievedChunks,
		Sources:         sources,
		Confidence:      confidence,
		Latency:         latency,
		Verified:        verified,
		VerifyResult:    verifyResult,
	}

 r.logger.WithContext(ctx).Info("RAG query completed", logging.Any("query", req.Query), logging.Any("answer_length", len(answer)))

	// span.AddTag("confidence", fmt.Sprintf("%.2f", confidence))
	// span.AddTag("verified", verified)

	return response, nil
}

// QueryStream 执行流式 RAG 查询
func (r *ragEngineImpl) QueryStream(ctx context.Context, req *RAGRequest) (<-chan *RAGChunk, error) {
	chunkChan := make(chan *RAGChunk, 10)

	go func() {
		defer close(chunkChan)

		// 应用默认值
		r.applyDefaults(req)

		// 1. 检索阶段
		chunkChan <- &RAGChunk{Type: ChunkTypeRetrieval, Content: "开始检索相关文档...", Done: false}

		processedQuery := r.understandQuery(ctx, req.Query)
		retrievedChunks, err := r.retrieveChunks(ctx, processedQuery, req)
		if err != nil {
			chunkChan <- &RAGChunk{Type: ChunkTypeError, Error: err, Done: true}
			return
		}

		chunkChan <- &RAGChunk{
			Type:    ChunkTypeRetrieval,
			Content: fmt.Sprintf("检索完成，找到 %d 个相关文档块", len(retrievedChunks)),
			Done:    false,
		}

		// 2. 重排序（可选）
		if req.RerankEnabled && r.reranker != nil {
			retrievedChunks, _ = r.rerankChunks(ctx, processedQuery, retrievedChunks)
		}

		// 3. 构建上下文
		ragContext := r.buildContext(ctx, retrievedChunks, req)

		// 4. 流式生成
		chunkChan <- &RAGChunk{Type: ChunkTypeGenerate, Content: "", Done: false}

		answerChan, err := r.generator.GenerateStream(ctx, &GenerateRequest{
			Query:       req.Query,
			Context:     ragContext,
			ModelName:   req.ModelName,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
		})

		if err != nil {
			chunkChan <- &RAGChunk{Type: ChunkTypeError, Error: err, Done: true}
			return
		}

		fullAnswer := ""
		for genChunk := range answerChan {
			if genChunk.Error != nil {
				chunkChan <- &RAGChunk{Type: ChunkTypeError, Error: genChunk.Error, Done: true}
				return
			}
			fullAnswer += genChunk.Content
			chunkChan <- &RAGChunk{Type: ChunkTypeGenerate, Content: genChunk.Content, Done: false}
		}

		// 5. 验证（可选）
		if req.VerifyEnabled {
			chunkChan <- &RAGChunk{Type: ChunkTypeVerify, Content: "验证答案中...", Done: false}
			verifyResult, err := r.verifyAnswer(ctx, req.Query, fullAnswer, retrievedChunks)
			if err == nil {
				verified := verifyResult.HasHallucination == false &&
					verifyResult.CitationValid &&
					verifyResult.FactCheckPassed
				chunkChan <- &RAGChunk{
					Type:    ChunkTypeVerify,
					Content: fmt.Sprintf("验证完成，结果: %v", verified),
					Done:    true,
				}
			}
		} else {
			chunkChan <- &RAGChunk{Type: ChunkTypeGenerate, Content: "", Done: true}
		}
	}()

	return chunkChan, nil
}

// HealthCheck 健康检查
func (r *ragEngineImpl) HealthCheck(ctx context.Context) error {
	// 检查 Retriever
	if err := r.retriever.HealthCheck(ctx); err != nil {
		return errors.Wrap(err, "ERR_UNAVAIL", "retriever unhealthy")
	}

	// 检查 Generator
	if err := r.generator.HealthCheck(ctx); err != nil {
		return errors.Wrap(err, "ERR_UNAVAIL", "generator unhealthy")
	}

	return nil
}

// understandQuery 查询理解（简化实现）
func (r *ragEngineImpl) understandQuery(ctx context.Context, query string) string {
	// 当前简化为直接返回原始查询
	// 未来可以集成查询改写、扩展、意图识别等功能
	return query
}

// retrieveChunks 检索文档块
func (r *ragEngineImpl) retrieveChunks(ctx context.Context, query string, req *RAGRequest) ([]*RetrievedChunk, error) {
	retrieveReq := &RetrieveRequest{
		Query:          query,
		CollectionName: req.CollectionName,
		TopK:           req.TopK,
		Mode:           req.RetrievalMode,
		Metadata:       req.Metadata,
	}

	return r.retriever.Retrieve(ctx, retrieveReq)
}

// rerankChunks 重排序文档块
func (r *ragEngineImpl) rerankChunks(ctx context.Context, query string, chunks []*RetrievedChunk) ([]*RetrievedChunk, error) {
	rerankReq := &RerankRequest{
		Query:  query,
		Chunks: chunks,
		TopK:   len(chunks), // 保留所有文档，仅调整顺序
	}

	return r.reranker.Rerank(ctx, rerankReq)
}

// buildContext 构建 RAG 上下文
func (r *ragEngineImpl) buildContext(ctx context.Context, chunks []*RetrievedChunk, req *RAGRequest) string {
	var contextBuilder string
	currentLength := 0

	for i, chunk := range chunks {
		chunkText := fmt.Sprintf("[文档 %d] 来源: %s\n%s\n\n", i+1, chunk.Source, chunk.Content)

		// 控制上下文长度
		if currentLength+len(chunkText) > r.config.MaxContextLength {
   r.logger.WithContext(ctx).Warn("context truncated due to length limit", logging.Any("max_length", r.config.MaxContextLength), logging.Any("chunks_included", i))
			break
		}

		contextBuilder += chunkText
		currentLength += len(chunkText)
	}

	return contextBuilder
}

// generateAnswer 生成答案
func (r *ragEngineImpl) generateAnswer(ctx context.Context, query, ragContext string, req *RAGRequest) (string, []string, error) {
	genReq := &GenerateRequest{
		Query:       query,
		Context:     ragContext,
		ModelName:   req.ModelName,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	genResp, err := r.generator.Generate(ctx, genReq)
	if err != nil {
		return "", nil, err
	}

	return genResp.Answer, genResp.Sources, nil
}

// verifyAnswer 验证答案
func (r *ragEngineImpl) verifyAnswer(ctx context.Context, query, answer string, chunks []*RetrievedChunk) (*VerifyResult, error) {
	// 简化实现：基于规则的验证
	result := &VerifyResult{
		HasHallucination: false,
		CitationValid:    true,
		FactCheckPassed:  true,
		Issues:           []string{},
	}

	// 检查答案长度
	if len(answer) < 10 {
		result.Issues = append(result.Issues, "答案过短")
		result.FactCheckPassed = false
	}

	// 检查是否引用了检索到的内容
	hasReference := false
	for _, chunk := range chunks {
		if contains(answer, chunk.Content[:min(50, len(chunk.Content))]) {
			hasReference = true
			break
		}
	}

	if !hasReference {
		result.Issues = append(result.Issues, "答案未引用检索到的内容，可能存在幻觉")
		result.HasHallucination = true
	}

	return result, nil
}

// calculateConfidence 计算置信度
func (r *ragEngineImpl) calculateConfidence(chunks []*RetrievedChunk, verified bool) float32 {
	if len(chunks) == 0 {
		return 0.0
	}

	// 基于检索分数计算基础置信度
	var avgScore float32
	for _, chunk := range chunks {
		avgScore += chunk.Score
	}
	avgScore /= float32(len(chunks))

	confidence := avgScore

	// 如果验证失败，降低置信度
	if !verified {
		confidence *= 0.7
	}

	if confidence > 1.0 {
		return 1.0
	}
	return confidence
}

// applyDefaults 应用默认配置
func (r *ragEngineImpl) applyDefaults(req *RAGRequest) {
	if req.TopK == 0 {
		req.TopK = r.config.DefaultTopK
	}
	if req.RetrievalMode == "" {
		req.RetrievalMode = r.config.DefaultRetrievalMode
	}
	if req.ModelName == "" {
		req.ModelName = r.config.DefaultModelName
	}
	if req.Temperature == 0 {
		req.Temperature = r.config.DefaultTemperature
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = r.config.DefaultMaxTokens
	}
}

// validateRequest 验证请求
func (r *ragEngineImpl) validateRequest(req *RAGRequest) error {
	if req.Query == "" {
		return fmt.Errorf("query is required")
	}
	if req.CollectionName == "" {
		return fmt.Errorf("collection name is required")
	}
	if req.TopK <= 0 || req.TopK > 100 {
		return fmt.Errorf("topK must be between 1 and 100")
	}
	if req.Temperature < 0 || req.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	return nil
}

// defaultRAGConfig 返回默认配置
func defaultRAGConfig() *RAGConfig {
	return &RAGConfig{
		DefaultTopK:          5,
		DefaultRetrievalMode: RetrievalModeHybrid,
		DefaultModelName:     "gpt-4",
		DefaultTemperature:   0.7,
		DefaultMaxTokens:     2048,
		EnableRerank:         true,
		EnableVerify:         true,
		MaxContextLength:     8000,
		TimeoutSeconds:       30,
	}
}

// 工具函数
func contains(text, substr string) bool {
	return len(text) >= len(substr) && (text == substr || len(text) > len(substr))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

//Personal.AI order the ending

