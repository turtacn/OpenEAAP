// internal/platform/rag/reranker.go
package rag

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/pkg/errors"
	"go.opentelemetry.io/otel/codes"
)

// Reranker 定义重排序器接口
type Reranker interface {
	// Rerank 对检索结果重排序
	Rerank(ctx context.Context, req *RerankRequest) ([]*RetrievedChunk, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// RerankRequest 定义重排序请求
type RerankRequest struct {
	Query  string             // 原始查询
	Chunks []*RetrievedChunk  // 待重排序的文档块
	TopK   int                // 返回数量
	Method RerankMethod       // 重排序方法
}

// RerankMethod 定义重排序方法
type RerankMethod string

const (
	RerankMethodScore      RerankMethod = "score"      // 基于原始分数
	RerankMethodMultiFactor RerankMethod = "multi_factor" // 多因子综合排序
	RerankMethodModel      RerankMethod = "model"      // 基于ML模型
	RerankMethodHybrid     RerankMethod = "hybrid"     // 混合方法
)

// rerankerImpl 实现重排序器
type rerankerImpl struct {
	modelClient RerankModelClient // ML重排序模型客户端（可选）
	logger      logging.Logger    // 日志器
	tracer      trace.Tracer      // 追踪器
	config      *RerankConfig     // 配置
}

// RerankConfig 定义重排序配置
type RerankConfig struct {
	// 多因子权重配置
	RelevanceWeight  float32 // 相关性权重
	FreshnessWeight  float32 // 新鲜度权重
	AuthorityWeight  float32 // 权威性权重
	DiversityWeight  float32 // 多样性权重

	// 新鲜度衰减参数
	FreshnessDecayDays float32 // 新鲜度衰减天数

	// 多样性参数
	DiversityThreshold float32 // 多样性阈值（相似度）

	// ML模型配置
	ModelEndpoint   string // 模型端点
	ModelTimeout    int    // 模型超时（秒）
	ModelBatchSize  int    // 批处理大小

	// 默认方法
	DefaultMethod RerankMethod
}

// RerankModelClient 定义ML重排序模型客户端接口
type RerankModelClient interface {
	// Rerank 使用模型重排序
	Rerank(ctx context.Context, query string, chunks []*RetrievedChunk) ([]*ScoredChunk, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// ScoredChunk 定义带分数的文档块
type ScoredChunk struct {
	Chunk *RetrievedChunk
	Score float32
}

// NewReranker 创建重排序器实例
func NewReranker(
	modelClient RerankModelClient,
	logger logging.Logger,
	tracer trace.Tracer,
	config *RerankConfig,
) Reranker {
	if config == nil {
		config = defaultRerankConfig()
	}

	return &rerankerImpl{
		modelClient: modelClient,
		logger:      logger,
		tracer:      tracer,
		config:      config,
	}
}

// Rerank 对检索结果重排序
func (r *rerankerImpl) Rerank(ctx context.Context, req *RerankRequest) ([]*RetrievedChunk, error) {
	startTime := time.Now()

	// 创建 Span
	ctx, span := r.tracer.Start(ctx, "Reranker.Rerank")
	defer span.End()
	// span.AddTag("query", req.Query)
	// span.AddTag("chunks_count", len(req.Chunks))
	// span.AddTag("method", string(req.Method))

	// 验证请求
	if err := r.validateRequest(req); err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidArgument, "invalid rerank request")
	}

	// 如果没有文档块，直接返回
	if len(req.Chunks) == 0 {
		return req.Chunks, nil
	}

	// 应用默认方法
	if req.Method == "" {
		req.Method = r.config.DefaultMethod
	}

	var rerankedChunks []*RetrievedChunk
	var err error

	// 根据方法执行重排序
	switch req.Method {
	case RerankMethodScore:
		rerankedChunks = r.rerankByScore(ctx, req.Chunks)
	case RerankMethodMultiFactor:
		rerankedChunks = r.rerankByMultiFactor(ctx, req.Query, req.Chunks)
	case RerankMethodModel:
		rerankedChunks, err = r.rerankByModel(ctx, req.Query, req.Chunks)
	case RerankMethodHybrid:
		rerankedChunks, err = r.rerankByHybrid(ctx, req.Query, req.Chunks)
	default:
		return nil, errors.New(errors.CodeInvalidArgument,
			fmt.Sprintf("unsupported rerank method: %s", req.Method))
	}

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, err
	}

	// 截断到 TopK
	if req.TopK > 0 && len(rerankedChunks) > req.TopK {
		rerankedChunks = rerankedChunks[:req.TopK]
	}

	latency := time.Since(startTime)
 r.logger.WithContext(ctx).Info("reranking completed", logging.Any("method", req.Method), logging.Any("input_count", len(req.Chunks)))

	return rerankedChunks, nil
}

// rerankByScore 基于原始分数重排序
func (r *rerankerImpl) rerankByScore(ctx context.Context, chunks []*RetrievedChunk) []*RetrievedChunk {
	r.logger.WithContext(ctx).Debug("reranking by score")

	// 创建副本以避免修改原始数据
	reranked := make([]*RetrievedChunk, len(chunks))
	copy(reranked, chunks)

	// 按分数降序排序
	sort.Slice(reranked, func(i, j int) bool {
		return reranked[i].Score > reranked[j].Score
	})

	return reranked
}

// rerankByMultiFactor 基于多因子重排序
func (r *rerankerImpl) rerankByMultiFactor(ctx context.Context, query string, chunks []*RetrievedChunk) []*RetrievedChunk {
	r.logger.WithContext(ctx).Debug("reranking by multi-factor")

	now := time.Now()
	scoredChunks := make([]*ScoredChunk, len(chunks))

	for i, chunk := range chunks {
		// 1. 相关性分数（归一化原始分数）
		relevanceScore := chunk.Score

		// 2. 新鲜度分数
		freshnessScore := r.calculateFreshnessScore(chunk, now)

		// 3. 权威性分数
		authorityScore := r.calculateAuthorityScore(chunk)

		// 4. 多样性分数（与已选文档的差异度）
		diversityScore := r.calculateDiversityScore(chunk, scoredChunks[:i])

		// 综合分数
		finalScore := r.config.RelevanceWeight*relevanceScore +
			r.config.FreshnessWeight*freshnessScore +
			r.config.AuthorityWeight*authorityScore +
			r.config.DiversityWeight*diversityScore

		scoredChunks[i] = &ScoredChunk{
			Chunk: chunk,
			Score: finalScore,
		}

  r.logger.WithContext(ctx).Debug("chunk scores calculated", logging.Any("chunk_id", chunk.ChunkID), logging.Any("relevance", relevanceScore), logging.Any("freshness", freshnessScore), logging.Any("authority", authorityScore), logging.Any("diversity", diversityScore), logging.Any("final", finalScore))
	}

	// 按综合分数降序排序
	sort.Slice(scoredChunks, func(i, j int) bool {
		return scoredChunks[i].Score > scoredChunks[j].Score
	})

	// 提取排序后的文档块
	reranked := make([]*RetrievedChunk, len(scoredChunks))
	for i, sc := range scoredChunks {
		reranked[i] = sc.Chunk
		reranked[i].Score = sc.Score // 更新为综合分数
	}

	return reranked
}

// calculateFreshnessScore 计算新鲜度分数
func (r *rerankerImpl) calculateFreshnessScore(chunk *RetrievedChunk, now time.Time) float32 {
	// 从元数据中获取创建时间
	createdAtStr, ok := chunk.Metadata["created_at"]
	if !ok {
		return 0.5 // 默认中等新鲜度
	}

	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return 0.5
	}

	// 计算天数差
	daysDiff := now.Sub(createdAt).Hours() / 24

	// 指数衰减: score = exp(-days / decay_days)
	score := float32(math.Exp(-daysDiff / float64(r.config.FreshnessDecayDays)))

	return score
}

// calculateAuthorityScore 计算权威性分数
func (r *rerankerImpl) calculateAuthorityScore(chunk *RetrievedChunk) float32 {
	// 基于来源的权威性评分
	source := chunk.Source

	// 定义来源权威性映射
	authorityMap := map[string]float32{
		"官方文档":     1.0,
		"学术论文":     0.9,
		"技术博客":     0.7,
		"社区问答":     0.5,
		"用户评论":     0.3,
	}

	// 默认权威性
	defaultAuthority := float32(0.5)

	if score, ok := authorityMap[source]; ok {
		return score
	}

	// 检查元数据中的权威性标记
	if authorityStr, ok := chunk.Metadata["authority"]; ok {
		if authority, err := parseFloat32(authorityStr); err == nil {
			return authority
		}
	}

	return defaultAuthority
}

// calculateDiversityScore 计算多样性分数
func (r *rerankerImpl) calculateDiversityScore(chunk *RetrievedChunk, selectedChunks []*ScoredChunk) float32 {
	if len(selectedChunks) == 0 {
		return 1.0 // 第一个文档，最大多样性
	}

	// 计算与已选文档的平均相似度
	var totalSimilarity float32
	for _, selected := range selectedChunks {
		similarity := r.calculateSimilarity(chunk.Content, selected.Chunk.Content)
		totalSimilarity += similarity
	}
	avgSimilarity := totalSimilarity / float32(len(selectedChunks))

	// 多样性 = 1 - 相似度
	diversityScore := 1.0 - avgSimilarity

	// 如果相似度超过阈值，惩罚分数
	if avgSimilarity > r.config.DiversityThreshold {
		diversityScore *= 0.5
	}

	return diversityScore
}

// calculateSimilarity 计算文本相似度（简化实现：Jaccard相似度）
func (r *rerankerImpl) calculateSimilarity(text1, text2 string) float32 {
	// 分词（简化：按空格分割）
	tokens1 := tokenize(text1)
	tokens2 := tokenize(text2)

	// 计算 Jaccard 相似度
	intersection := intersect(tokens1, tokens2)
	union := len(tokens1) + len(tokens2) - len(intersection)

	if union == 0 {
		return 0
	}

	return float32(len(intersection)) / float32(union)
}

// rerankByModel 使用ML模型重排序
func (r *rerankerImpl) rerankByModel(ctx context.Context, query string, chunks []*RetrievedChunk) ([]*RetrievedChunk, error) {
	if r.modelClient == nil {
		return nil, errors.New(errors.CodeUnimplemented, "rerank model client not configured")
	}

	r.logger.WithContext(ctx).Debug("reranking by model")

	// 调用模型进行重排序
	scoredChunks, err := r.modelClient.Rerank(ctx, query, chunks)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "model reranking failed")
	}

	// 按模型分数降序排序
	sort.Slice(scoredChunks, func(i, j int) bool {
		return scoredChunks[i].Score > scoredChunks[j].Score
	})

	// 提取排序后的文档块
	reranked := make([]*RetrievedChunk, len(scoredChunks))
	for i, sc := range scoredChunks {
		reranked[i] = sc.Chunk
		reranked[i].Score = sc.Score // 更新为模型分数
	}

	return reranked, nil
}

// rerankByHybrid 混合重排序（结合多因子和模型）
func (r *rerankerImpl) rerankByHybrid(ctx context.Context, query string, chunks []*RetrievedChunk) ([]*RetrievedChunk, error) {
	r.logger.WithContext(ctx).Debug("reranking by hybrid method")

	// 1. 多因子重排序
	multiFactorChunks := r.rerankByMultiFactor(ctx, query, chunks)

	// 2. 模型重排序（如果可用）
	if r.modelClient != nil {
		modelChunks, err := r.rerankByModel(ctx, query, chunks)
		if err != nil {
			r.logger.WithContext(ctx).Warn("model reranking failed, using multi-factor only", logging.Error(err))
			return multiFactorChunks, nil
		}

		// 3. 融合两种排序结果（加权平均排名）
		return r.fuseRankings(multiFactorChunks, modelChunks, 0.5, 0.5), nil
	}

	return multiFactorChunks, nil
}

// fuseRankings 融合两种排序结果
func (r *rerankerImpl) fuseRankings(chunks1, chunks2 []*RetrievedChunk, weight1, weight2 float32) []*RetrievedChunk {
	// 构建排名映射
	rank1Map := make(map[string]int)
	rank2Map := make(map[string]int)

	for i, chunk := range chunks1 {
		rank1Map[chunk.ChunkID] = i + 1
	}
	for i, chunk := range chunks2 {
		rank2Map[chunk.ChunkID] = i + 1
	}

	// 计算融合分数（基于倒数排名）
	chunkScores := make(map[string]float32)
	chunkMap := make(map[string]*RetrievedChunk)

	for _, chunk := range chunks1 {
		id := chunk.ChunkID
		rank1 := float32(rank1Map[id])
		rank2 := float32(rank2Map[id])

		// 融合分数 = weight1/rank1 + weight2/rank2
		fusedScore := weight1/rank1 + weight2/rank2
		chunkScores[id] = fusedScore
		chunkMap[id] = chunk
	}

	// 转换为切片并排序
	fusedChunks := make([]*RetrievedChunk, 0, len(chunkScores))
	for id, score := range chunkScores {
		chunk := chunkMap[id]
		chunk.Score = score
		fusedChunks = append(fusedChunks, chunk)
	}

	sort.Slice(fusedChunks, func(i, j int) bool {
		return fusedChunks[i].Score > fusedChunks[j].Score
	})

	return fusedChunks
}

// validateRequest 验证请求
func (r *rerankerImpl) validateRequest(req *RerankRequest) error {
	if req.Query == "" {
		return fmt.Errorf("query is required")
	}
	if req.Chunks == nil {
		return fmt.Errorf("chunks is required")
	}
	if req.TopK < 0 {
		return fmt.Errorf("topK must be non-negative")
	}
	return nil
}

// HealthCheck 健康检查
func (r *rerankerImpl) HealthCheck(ctx context.Context) error {
	// 检查模型客户端（如果配置）
	if r.modelClient != nil {
		if err := r.modelClient.HealthCheck(ctx); err != nil {
			r.logger.WithContext(ctx).Warn("rerank model client unhealthy", logging.Error(err))
			// 模型不可用时，仍可使用多因子排序，不返回错误
		}
	}
	return nil
}

// defaultRerankConfig 返回默认配置
func defaultRerankConfig() *RerankConfig {
	return &RerankConfig{
		RelevanceWeight:    0.5,  // 相关性权重
		FreshnessWeight:    0.2,  // 新鲜度权重
		AuthorityWeight:    0.2,  // 权威性权重
		DiversityWeight:    0.1,  // 多样性权重
		FreshnessDecayDays: 365,  // 1年衰减期
		DiversityThreshold: 0.7,  // 相似度阈值
		ModelTimeout:       5,    // 模型超时5秒
		ModelBatchSize:     32,   // 批处理大小
		DefaultMethod:      RerankMethodMultiFactor,
	}
}

// 工具函数

// tokenize 分词
func tokenize(text string) []string {
	// 简化实现：按空格分割，转小写
	tokens := make([]string, 0)
	words := splitBySpace(text)
	for _, word := range words {
		if word != "" {
			tokens = append(tokens, toLowerCase(word))
		}
	}
	return tokens
}

// intersect 计算两个切片的交集
func intersect(a, b []string) []string {
	m := make(map[string]bool)
	for _, item := range a {
		m[item] = true
	}

	intersection := make([]string, 0)
	for _, item := range b {
		if m[item] {
			intersection = append(intersection, item)
			delete(m, item) // 避免重复
		}
	}
	return intersection
}

// splitBySpace 按空格分割字符串
func splitBySpace(s string) []string {
	var result []string
	var current string
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// toLowerCase 转小写
func toLowerCase(s string) string {
	result := ""
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			result += string(r + 32)
		} else {
			result += string(r)
		}
	}
	return result
}

// parseFloat32 解析float32
func parseFloat32(s string) (float32, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return float32(f), err
}

//Personal.AI order the ending
