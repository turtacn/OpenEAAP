// internal/platform/rag/generator.go
package rag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/pkg/errors"
)

// Generator 定义生成器接口
type Generator interface {
	// Generate 生成答案
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)

	// GenerateStream 流式生成答案
	GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *GenerateChunk, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// GenerateRequest 定义生成请求
type GenerateRequest struct {
	Query        string  // 用户查询
	Context      string  // 检索上下文
	ModelName    string  // 模型名称
	Temperature  float32 // 生成温度
	MaxTokens    int     // 最大生成长度
	SystemPrompt string  // 系统提示词（可选）
	EnableVerify bool    // 是否启用验证
}

// GenerateResponse 定义生成响应
type GenerateResponse struct {
	Answer       string        // 生成的答案
	Sources      []string      // 引用来源
	TokensUsed   int           // 使用的token数
	Latency      time.Duration // 延迟
	Verified     bool          // 是否通过验证
	VerifyIssues []string      // 验证问题
}

// GenerateChunk 定义流式生成块
type GenerateChunk struct {
	Content string // 内容
	Done    bool   // 是否完成
	Error   error  // 错误
}

// generatorImpl 实现生成器
type generatorImpl struct {
	llmClient LLMClient        // LLM客户端
	verifier  AnswerVerifier   // 答案验证器
	logger    logging.Logger   // 日志器
	tracer    trace.Tracer     // 追踪器
	config    *GeneratorConfig // 配置
}

// GeneratorConfig 定义生成器配置
type GeneratorConfig struct {
	DefaultModelName    string  // 默认模型名称
	DefaultTemperature  float32 // 默认温度
	DefaultMaxTokens    int     // 默认最大长度
	DefaultSystemPrompt string  // 默认系统提示词
	EnableVerification  bool    // 是否启用验证
	TimeoutSeconds      int     // 超时时间
	MaxRetries          int     // 最大重试次数
}

// LLMClient 定义LLM客户端接口
type LLMClient interface {
	// Complete 同步补全
	Complete(ctx context.Context, req *LLMRequest) (*LLMResponse, error)

	// CompleteStream 流式补全
	CompleteStream(ctx context.Context, req *LLMRequest) (<-chan *LLMChunk, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// LLMRequest 定义LLM请求
type LLMRequest struct {
	Messages    []Message // 消息列表
	ModelName   string    // 模型名称
	Temperature float32   // 温度
	MaxTokens   int       // 最大长度
	StopWords   []string  // 停止词
}

// Message 定义消息
type Message struct {
	Role    string // 角色：system/user/assistant
	Content string // 内容
}

// LLMResponse 定义LLM响应
type LLMResponse struct {
	Content      string // 生成内容
	TokensUsed   int    // 使用的token数
	FinishReason string // 结束原因
}

// LLMChunk 定义LLM流式块
type LLMChunk struct {
	Content string // 内容
	Done    bool   // 是否完成
	Error   error  // 错误
}

// AnswerVerifier 定义答案验证器接口
type AnswerVerifier interface {
	// Verify 验证答案
	Verify(ctx context.Context, query, answer, context string) (*VerificationResult, error)
}

// VerificationResult 定义验证结果
type VerificationResult struct {
	Passed           bool     // 是否通过
	HasHallucination bool     // 是否有幻觉
	CitationValid    bool     // 引用是否有效
	FactCheckPassed  bool     // 事实检查是否通过
	Issues           []string // 问题列表
}

// NewGenerator 创建生成器实例
func NewGenerator(
	llmClient LLMClient,
	verifier AnswerVerifier,
	logger logging.Logger,
	tracer trace.Tracer,
	config *GeneratorConfig,
) Generator {
	if config == nil {
		config = defaultGeneratorConfig()
	}

	return &generatorImpl{
		llmClient: llmClient,
		verifier:  verifier,
		logger:    logger,
		tracer:    tracer,
		config:    config,
	}
}

// Generate 生成答案
func (g *generatorImpl) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	startTime := time.Now()

	// 创建 Span
	span := g.tracer.StartSpan(ctx, "Generator.Generate")
	defer span.End()
	span.AddTag("query", req.Query)
	span.AddTag("model", req.ModelName)

	// 应用默认值
	g.applyDefaults(req)

	// 验证请求
	if err := g.validateRequest(req); err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidArgument, "invalid generate request")
	}

	// 构建 Prompt
	messages := g.buildPrompt(req)

	// 调用 LLM
	llmReq := &LLMRequest{
		Messages:    messages,
		ModelName:   req.ModelName,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		StopWords:   []string{},
	}

	llmResp, err := g.llmClient.Complete(ctx, llmReq)
	if err != nil {
		span.SetStatus(trace.StatusError, err.Error())
		return nil, errors.Wrap(err, errors.CodeInternal, "LLM completion failed")
	}

	answer := llmResp.Content

	// 提取引用来源
	sources := g.extractSources(answer, req.Context)

	// 答案验证（如果启用）
	verified := true
	var verifyIssues []string

	if req.EnableVerify || g.config.EnableVerification {
		if g.verifier != nil {
			verifyResult, err := g.verifier.Verify(ctx, req.Query, answer, req.Context)
			if err != nil {
				g.logger.Warn(ctx, "answer verification failed", "error", err)
			} else {
				verified = verifyResult.Passed
				verifyIssues = verifyResult.Issues

				if !verified {
					g.logger.Warn(ctx, "answer verification failed",
						"has_hallucination", verifyResult.HasHallucination,
						"citation_valid", verifyResult.CitationValid,
						"fact_check_passed", verifyResult.FactCheckPassed,
						"issues", verifyIssues)
				}
			}
		}
	}

	latency := time.Since(startTime)

	response := &GenerateResponse{
		Answer:       answer,
		Sources:      sources,
		TokensUsed:   llmResp.TokensUsed,
		Latency:      latency,
		Verified:     verified,
		VerifyIssues: verifyIssues,
	}

	g.logger.Info(ctx, "generation completed",
		"query", req.Query,
		"answer_length", len(answer),
		"tokens_used", llmResp.TokensUsed,
		"verified", verified,
		"latency_ms", latency.Milliseconds())

	span.AddTag("tokens_used", llmResp.TokensUsed)
	span.AddTag("verified", verified)

	return response, nil
}

// GenerateStream 流式生成答案
func (g *generatorImpl) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *GenerateChunk, error) {
	// 应用默认值
	g.applyDefaults(req)

	// 验证请求
	if err := g.validateRequest(req); err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidArgument, "invalid generate request")
	}

	// 构建 Prompt
	messages := g.buildPrompt(req)

	// 调用 LLM 流式接口
	llmReq := &LLMRequest{
		Messages:    messages,
		ModelName:   req.ModelName,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		StopWords:   []string{},
	}

	llmChunkChan, err := g.llmClient.CompleteStream(ctx, llmReq)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "LLM stream failed")
	}

	// 转换为 GenerateChunk
	genChunkChan := make(chan *GenerateChunk, 10)

	go func() {
		defer close(genChunkChan)

		for llmChunk := range llmChunkChan {
			genChunk := &GenerateChunk{
				Content: llmChunk.Content,
				Done:    llmChunk.Done,
				Error:   llmChunk.Error,
			}
			genChunkChan <- genChunk

			if llmChunk.Error != nil || llmChunk.Done {
				break
			}
		}
	}()

	return genChunkChan, nil
}

// buildPrompt 构建 Prompt
func (g *generatorImpl) buildPrompt(req *GenerateRequest) []Message {
	messages := make([]Message, 0, 3)

	// 1. 系统提示词
	systemPrompt := req.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = g.config.DefaultSystemPrompt
	}

	messages = append(messages, Message{
		Role:    "system",
		Content: systemPrompt,
	})

	// 2. 构建用户消息（包含检索上下文）
	userContent := g.buildUserMessage(req.Query, req.Context)

	messages = append(messages, Message{
		Role:    "user",
		Content: userContent,
	})

	return messages
}

// buildUserMessage 构建用户消息
func (g *generatorImpl) buildUserMessage(query, context string) string {
	var builder strings.Builder

	// 添加上下文
	if context != "" {
		builder.WriteString("参考以下上下文信息回答问题:\n\n")
		builder.WriteString("【上下文】\n")
		builder.WriteString(context)
		builder.WriteString("\n\n")
	}

	// 添加查询
	builder.WriteString("【问题】\n")
	builder.WriteString(query)
	builder.WriteString("\n\n")

	// 添加指导
	builder.WriteString("【要求】\n")
	builder.WriteString("1. 基于上述上下文回答问题，不要编造信息\n")
	builder.WriteString("2. 如果上下文中没有相关信息，请明确说明\n")
	builder.WriteString("3. 在答案中标注信息来源（使用[文档X]格式）\n")
	builder.WriteString("4. 保持答案简洁、准确、客观\n")

	return builder.String()
}

// extractSources 提取引用来源
func (g *generatorImpl) extractSources(answer, context string) []string {
	sources := make([]string, 0)

	// 查找 [文档X] 格式的引用
	for i := 1; i <= 10; i++ {
		citation := fmt.Sprintf("[文档 %d]", i)
		if strings.Contains(answer, citation) {
			sources = append(sources, citation)
		}
	}

	// 如果没有显式引用，从上下文中提取来源
	if len(sources) == 0 && context != "" {
		lines := strings.Split(context, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "[文档") && strings.Contains(line, "来源:") {
				parts := strings.Split(line, "来源:")
				if len(parts) >= 2 {
					source := strings.TrimSpace(parts[1])
					if source != "" && !contains(sources, source) {
						sources = append(sources, source)
					}
				}
			}
		}
	}

	return sources
}

// applyDefaults 应用默认配置
func (g *generatorImpl) applyDefaults(req *GenerateRequest) {
	if req.ModelName == "" {
		req.ModelName = g.config.DefaultModelName
	}
	if req.Temperature == 0 {
		req.Temperature = g.config.DefaultTemperature
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = g.config.DefaultMaxTokens
	}
}

// validateRequest 验证请求
func (g *generatorImpl) validateRequest(req *GenerateRequest) error {
	if req.Query == "" {
		return fmt.Errorf("query is required")
	}
	if req.Temperature < 0 || req.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if req.MaxTokens <= 0 || req.MaxTokens > 8192 {
		return fmt.Errorf("maxTokens must be between 1 and 8192")
	}
	return nil
}

// HealthCheck 健康检查
func (g *generatorImpl) HealthCheck(ctx context.Context) error {
	// 检查 LLM 客户端
	if err := g.llmClient.HealthCheck(ctx); err != nil {
		return errors.Wrap(err, errors.CodeUnavailable, "LLM client unhealthy")
	}

	return nil
}

// defaultGeneratorConfig 返回默认配置
func defaultGeneratorConfig() *GeneratorConfig {
	return &GeneratorConfig{
		DefaultModelName:   "gpt-4",
		DefaultTemperature: 0.7,
		DefaultMaxTokens:   2048,
		DefaultSystemPrompt: "你是一个专业的AI助手，擅长基于提供的上下文信息回答问题。" +
			"你的回答应该准确、客观、简洁，并始终基于给定的上下文，不编造信息。",
		EnableVerification: true,
		TimeoutSeconds:     30,
		MaxRetries:         3,
	}
}

// answerVerifierImpl 实现答案验证器（简化版本）
type answerVerifierImpl struct {
	logger logging.Logger
}

// NewAnswerVerifier 创建答案验证器
func NewAnswerVerifier(logger logging.Logger) AnswerVerifier {
	return &answerVerifierImpl{
		logger: logger,
	}
}

// Verify 验证答案
func (v *answerVerifierImpl) Verify(ctx context.Context, query, answer, context string) (*VerificationResult, error) {
	result := &VerificationResult{
		Passed:           true,
		HasHallucination: false,
		CitationValid:    true,
		FactCheckPassed:  true,
		Issues:           []string{},
	}

	// 1. 检查答案长度
	if len(answer) < 10 {
		result.Issues = append(result.Issues, "答案过短")
		result.Passed = false
	}

	// 2. 检查是否包含"我不知道"等表述
	uncertainPhrases := []string{"不确定", "不清楚", "无法回答", "没有相关信息"}
	for _, phrase := range uncertainPhrases {
		if strings.Contains(answer, phrase) {
			v.logger.Debug(ctx, "answer contains uncertainty phrase", "phrase", phrase)
		}
	}

	// 3. 检查引用有效性
	if context != "" {
		hasReference := false

		// 检查答案是否引用了上下文
		contextSnippets := extractContextSnippets(context, 50)
		for _, snippet := range contextSnippets {
			if strings.Contains(answer, snippet) || containsSimilar(answer, snippet) {
				hasReference = true
				break
			}
		}

		if !hasReference {
			result.Issues = append(result.Issues, "答案未引用上下文信息，可能存在幻觉")
			result.HasHallucination = true
			result.CitationValid = false
			result.Passed = false
		}
	}

	// 4. 检查事实一致性（简化版本）
	if context != "" && !checkFactConsistency(answer, context) {
		result.Issues = append(result.Issues, "答案与上下文存在事实不一致")
		result.FactCheckPassed = false
		result.Passed = false
	}

	return result, nil
}

// extractContextSnippets 提取上下文片段
func extractContextSnippets(context string, maxLength int) []string {
	snippets := make([]string, 0)

	lines := strings.Split(context, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 20 && !strings.HasPrefix(line, "[文档") {
			snippet := line
			if len(snippet) > maxLength {
				snippet = snippet[:maxLength]
			}
			snippets = append(snippets, snippet)
		}
	}

	return snippets
}

// containsSimilar 检查是否包含相似内容（简化实现）
func containsSimilar(text, snippet string) bool {
	// 简化：检查是否包含片段的主要词汇
	words := strings.Fields(snippet)
	matchCount := 0

	for _, word := range words {
		if len(word) > 2 && strings.Contains(text, word) {
			matchCount++
		}
	}

	// 如果超过50%的词匹配，认为相似
	return matchCount > len(words)/2
}

// checkFactConsistency 检查事实一致性（简化实现）
func checkFactConsistency(answer, context string) bool {
	// 简化：检查答案中的数字、日期等关键信息是否在上下文中
	// 实际应用中应使用更复杂的NLP技术

	// 提取答案中的数字
	answerNumbers := extractNumbers(answer)
	contextNumbers := extractNumbers(context)

	// 检查答案中的数字是否都在上下文中
	for _, num := range answerNumbers {
		if !contains(contextNumbers, num) {
			return false
		}
	}

	return true
}

// extractNumbers 提取文本中的数字
func extractNumbers(text string) []string {
	numbers := make([]string, 0)
	var current string

	for _, r := range text {
		if (r >= '0' && r <= '9') || r == '.' {
			current += string(r)
		} else {
			if current != "" {
				numbers = append(numbers, current)
				current = ""
			}
		}
	}

	if current != "" {
		numbers = append(numbers, current)
	}

	return numbers
}

// contains 检查切片是否包含元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

//Personal.AI order the ending
