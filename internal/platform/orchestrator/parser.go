package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/openeeap/openeeap/internal/domain/agent"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/pkg/errors"
)

// Parser 请求解析器接口
type Parser interface {
	// Parse 解析请求
	Parse(ctx context.Context, req *ParseRequest) (*ParseResponse, error)

	// ParseIntent 解析意图
	ParseIntent(ctx context.Context, input string) (*Intent, error)

	// ExtractParameters 提取参数
	ExtractParameters(ctx context.Context, input string, schema *ParameterSchema) (map[string]interface{}, error)

	// ExtractContext 提取上下文
	ExtractContext(ctx context.Context, input string, history []*Message) (*ContextInfo, error)

	// ValidateInput 验证输入
	ValidateInput(ctx context.Context, input string) error
}

// parser 解析器实现
type parser struct {
	llmClient       llm.LLMClient
	intentDetector  IntentDetector
	paramExtractor  ParameterExtractor
	contextBuilder  ContextBuilder
	config          *ParserConfig
	logger          logger.Logger
	cache           *parseCache
	patterns        []*ParsePattern
}

// ParserConfig 解析器配置
type ParserConfig struct {
	EnableLLM           bool          // 启用LLM解析
	EnableCache         bool          // 启用缓存
	CacheTTL            time.Duration // 缓存TTL
	MaxInputLength      int           // 最大输入长度
	MinInputLength      int           // 最小输入长度
	EnableIntentDetection bool        // 启用意图检测
	EnableParameterExtraction bool    // 启用参数提取
	EnableContextExtraction bool      // 启用上下文提取
	IntentConfidenceThreshold float64 // 意图置信度阈值
	DefaultLanguage       string      // 默认语言
	SupportedLanguages    []string    // 支持的语言列表
}

// ParseRequest 解析请求
type ParseRequest struct {
	Input      string                 // 输入内容
	Context    map[string]interface{} // 上下文
	History    []*Message             // 历史消息
	Language   string                 // 语言
	Options    *ParseOptions          // 解析选项
	Metadata   map[string]string      // 元数据
}

// ParseOptions 解析选项
type ParseOptions struct {
	EnableLLM         bool     // 启用LLM
	DetectIntent      bool     // 检测意图
	ExtractParameters bool     // 提取参数
	ExtractContext    bool     // 提取上下文
	ExpectedIntents   []string // 期望的意图列表
	ParameterSchema   *ParameterSchema // 参数模式
}

// ParseResponse 解析响应
type ParseResponse struct {
	Input          string                 // 输入内容
	Intent         *Intent                // 意图
	Parameters     map[string]interface{} // 参数
	Context        *ContextInfo           // 上下文信息
	Entities       []*Entity              // 实体列表
	Language       string                 // 语言
	Confidence     float64                // 置信度
	ProcessingTime time.Duration          // 处理时间
	Metadata       map[string]interface{} // 元数据
}

// Intent 意图
type Intent struct {
	Name        string                 // 意图名称
	Type        IntentType             // 意图类型
	Confidence  float64                // 置信度
	Description string                 // 描述
	Category    string                 // 分类
	Synonyms    []string               // 同义词
	Examples    []string               // 示例
	Metadata    map[string]interface{} // 元数据
}

// IntentType 意图类型
type IntentType string

const (
	IntentTypeQuery       IntentType = "query"       // 查询
	IntentTypeCommand     IntentType = "command"     // 命令
	IntentTypeQuestion    IntentType = "question"    // 问题
	IntentTypeStatement   IntentType = "statement"   // 陈述
	IntentTypeRequest     IntentType = "request"     // 请求
	IntentTypeConfirmation IntentType = "confirmation" // 确认
	IntentTypeNegation    IntentType = "negation"    // 否定
	IntentTypeGreeting    IntentType = "greeting"    // 问候
	IntentTypeFarewell    IntentType = "farewell"    // 告别
	IntentTypeUnknown     IntentType = "unknown"     // 未知
)

// ParameterSchema 参数模式
type ParameterSchema struct {
	Parameters []*ParameterDefinition // 参数定义列表
}

// ParameterDefinition 参数定义
type ParameterDefinition struct {
	Name        string                 // 参数名称
	Type        ParameterType          // 参数类型
	Required    bool                   // 是否必需
	Description string                 // 描述
	Default     interface{}            // 默认值
	Enum        []interface{}          // 枚举值
	Pattern     string                 // 正则表达式模式
	Minimum     *float64               // 最小值
	Maximum     *float64               // 最大值
	MinLength   *int                   // 最小长度
	MaxLength   *int                   // 最大长度
	Format      string                 // 格式
	Examples    []interface{}          // 示例
	Metadata    map[string]interface{} // 元数据
}

// ParameterType 参数类型
type ParameterType string

const (
	ParameterTypeString  ParameterType = "string"  // 字符串
	ParameterTypeInteger ParameterType = "integer" // 整数
	ParameterTypeNumber  ParameterType = "number"  // 数字
	ParameterTypeBoolean ParameterType = "boolean" // 布尔值
	ParameterTypeArray   ParameterType = "array"   // 数组
	ParameterTypeObject  ParameterType = "object"  // 对象
	ParameterTypeDate    ParameterType = "date"    // 日期
	ParameterTypeTime    ParameterType = "time"    // 时间
	ParameterTypeDateTime ParameterType = "datetime" // 日期时间
	ParameterTypeEnum    ParameterType = "enum"    // 枚举
)

// ContextInfo 上下文信息
type ContextInfo struct {
	SessionID       string                 // 会话ID
	UserID          string                 // 用户ID
	ConversationID  string                 // 对话ID
	Timestamp       time.Time              // 时间戳
	Location        *Location              // 位置信息
	Device          *DeviceInfo            // 设备信息
	UserProfile     *UserProfile           // 用户画像
	PreviousIntent  *Intent                // 前一个意图
	Variables       map[string]interface{} // 变量
	State           map[string]interface{} // 状态
	Metadata        map[string]interface{} // 元数据
}

// Location 位置信息
type Location struct {
	Country    string  // 国家
	Region     string  // 地区
	City       string  // 城市
	Latitude   float64 // 纬度
	Longitude  float64 // 经度
	Timezone   string  // 时区
}

// DeviceInfo 设备信息
type DeviceInfo struct {
	Type        string // 设备类型
	Platform    string // 平台
	OS          string // 操作系统
	Browser     string // 浏览器
	Version     string // 版本
	UserAgent   string // 用户代理
}

// UserProfile 用户画像
type UserProfile struct {
	Name        string   // 姓名
	Email       string   // 邮箱
	Language    string   // 语言
	Preferences map[string]interface{} // 偏好
	History     []*Intent // 历史意图
	Tags        []string // 标签
}

// Entity 实体
type Entity struct {
	Name       string                 // 实体名称
	Type       EntityType             // 实体类型
	Value      interface{}            // 值
	Confidence float64                // 置信度
	StartPos   int                    // 起始位置
	EndPos     int                    // 结束位置
	Metadata   map[string]interface{} // 元数据
}

// EntityType 实体类型
type EntityType string

const (
	EntityTypePerson       EntityType = "person"       // 人物
	EntityTypeOrganization EntityType = "organization" // 组织
	EntityTypeLocation     EntityType = "location"     // 地点
	EntityTypeDate         EntityType = "date"         // 日期
	EntityTypeTime         EntityType = "time"         // 时间
	EntityTypeMoney        EntityType = "money"        // 金额
	EntityTypePercentage   EntityType = "percentage"   // 百分比
	EntityTypeEmail        EntityType = "email"        // 邮箱
	EntityTypeURL          EntityType = "url"          // URL
	EntityTypePhone        EntityType = "phone"        // 电话
	EntityTypeCustom       EntityType = "custom"       // 自定义
)

// Message 消息
type Message struct {
	Role      string    // 角色
	Content   string    // 内容
	Timestamp time.Time // 时间戳
}

// ParsePattern 解析模式
type ParsePattern struct {
	Name        string         // 模式名称
	Pattern     *regexp.Regexp // 正则表达式
	Intent      string         // 意图
	Parameters  map[string]int // 参数映射（参数名 -> 捕获组索引）
	Priority    int            // 优先级
}

// parseCache 解析缓存
type parseCache struct {
	data map[string]*ParseResponse
	ttl  time.Duration
}

// IntentDetector 意图检测器接口
type IntentDetector interface {
	Detect(ctx context.Context, input string, options *DetectOptions) (*Intent, error)
}

// DetectOptions 检测选项
type DetectOptions struct {
	ExpectedIntents []string // 期望的意图列表
	UseLL           bool     // 使用LLM
	Threshold       float64  // 阈值
}

// ParameterExtractor 参数提取器接口
type ParameterExtractor interface {
	Extract(ctx context.Context, input string, schema *ParameterSchema) (map[string]interface{}, error)
}

// ContextBuilder 上下文构建器接口
type ContextBuilder interface {
	Build(ctx context.Context, input string, history []*Message, existing map[string]interface{}) (*ContextInfo, error)
}

// NewParser 创建解析器
func NewParser(
	llmClient llm.LLMClient,
	config *ParserConfig,
	logger logger.Logger,
) (Parser, error) {
	if config == nil {
		config = &ParserConfig{
			EnableLLM:                 true,
			EnableCache:               true,
			CacheTTL:                  5 * time.Minute,
			MaxInputLength:            10000,
			MinInputLength:            1,
			EnableIntentDetection:     true,
			EnableParameterExtraction: true,
			EnableContextExtraction:   true,
			IntentConfidenceThreshold: 0.6,
			DefaultLanguage:           "en",
			SupportedLanguages:        []string{"en", "zh", "es", "fr", "de"},
		}
	}

	p := &parser{
		llmClient:      llmClient,
		config:         config,
		logger:         logger,
		cache:          &parseCache{data: make(map[string]*ParseResponse), ttl: config.CacheTTL},
		patterns:       make([]*ParsePattern, 0),
	}

	// 初始化组件
	p.intentDetector = newIntentDetector(llmClient, config, logger)
	p.paramExtractor = newParameterExtractor(llmClient, logger)
	p.contextBuilder = newContextBuilder(logger)

	// 加载预定义模式
	p.loadPredefinedPatterns()

	return p, nil
}

// Parse 解析请求
func (p *parser) Parse(ctx context.Context, req *ParseRequest) (*ParseResponse, error) {
	if req == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "parse request cannot be nil")
	}

	startTime := time.Now()

	// 验证输入
	if err := p.ValidateInput(ctx, req.Input); err != nil {
		return nil, err
	}

	// 检查缓存
	if p.config.EnableCache {
		if cached := p.getFromCache(req.Input); cached != nil {
			return cached, nil
		}
	}

	// 规范化输入
	normalizedInput := p.normalizeInput(req.Input)

	// 检测语言
	language := req.Language
	if language == "" {
		language = p.detectLanguage(normalizedInput)
	}

	resp := &ParseResponse{
		Input:    req.Input,
		Language: language,
		Metadata: make(map[string]interface{}),
	}

	// 1. 尝试模式匹配
	if matched, intent, params := p.matchPattern(normalizedInput); matched {
		resp.Intent = intent
		resp.Parameters = params
		resp.Confidence = 1.0
	}

	// 2. 意图检测
	if req.Options == nil || req.Options.DetectIntent {
		intent, err := p.ParseIntent(ctx, normalizedInput)
		if err != nil {
			p.logger.Warn("intent detection failed", "error", err)
		} else if intent != nil && intent.Confidence >= p.config.IntentConfidenceThreshold {
			resp.Intent = intent
			resp.Confidence = intent.Confidence
		}
	}

	// 3. 参数提取
	if req.Options == nil || req.Options.ExtractParameters {
		var schema *ParameterSchema
		if req.Options != nil {
			schema = req.Options.ParameterSchema
		}
		params, err := p.ExtractParameters(ctx, normalizedInput, schema)
		if err != nil {
			p.logger.Warn("parameter extraction failed", "error", err)
		} else {
			resp.Parameters = params
		}
	}

	// 4. 实体识别
	entities := p.extractEntities(normalizedInput)
	resp.Entities = entities

	// 5. 上下文提取
	if req.Options == nil || req.Options.ExtractContext {
		contextInfo, err := p.ExtractContext(ctx, normalizedInput, req.History)
		if err != nil {
			p.logger.Warn("context extraction failed", "error", err)
		} else {
			// 合并请求中的上下文
			if req.Context != nil {
				if contextInfo.Variables == nil {
					contextInfo.Variables = make(map[string]interface{})
				}
				for k, v := range req.Context {
					contextInfo.Variables[k] = v
				}
			}
			resp.Context = contextInfo
		}
	}

	// 如果没有检测到意图，设置为未知
	if resp.Intent == nil {
		resp.Intent = &Intent{
			Name:       "unknown",
			Type:       IntentTypeUnknown,
			Confidence: 0.0,
		}
	}

	// 计算处理时间
	resp.ProcessingTime = time.Since(startTime)

	// 缓存结果
	if p.config.EnableCache && resp.Confidence >= p.config.IntentConfidenceThreshold {
		p.putToCache(req.Input, resp)
	}

	return resp, nil
}

// ParseIntent 解析意图
func (p *parser) ParseIntent(ctx context.Context, input string) (*Intent, error) {
	if !p.config.EnableIntentDetection {
		return nil, nil
	}

	// 使用意图检测器
	intent, err := p.intentDetector.Detect(ctx, input, &DetectOptions{
		UseLLM:    p.config.EnableLLM,
		Threshold: p.config.IntentConfidenceThreshold,
	})
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to detect intent")
	}

	return intent, nil
}

// ExtractParameters 提取参数
func (p *parser) ExtractParameters(ctx context.Context, input string, schema *ParameterSchema) (map[string]interface{}, error) {
	if !p.config.EnableParameterExtraction {
		return nil, nil
	}

	// 使用参数提取器
	params, err := p.paramExtractor.Extract(ctx, input, schema)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to extract parameters")
	}

	return params, nil
}

// ExtractContext 提取上下文
func (p *parser) ExtractContext(ctx context.Context, input string, history []*Message) (*ContextInfo, error) {
	if !p.config.EnableContextExtraction {
		return nil, nil
	}

	// 使用上下文构建器
	contextInfo, err := p.contextBuilder.Build(ctx, input, history, nil)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to build context")
	}

	return contextInfo, nil
}

// ValidateInput 验证输入
func (p *parser) ValidateInput(ctx context.Context, input string) error {
	if input == "" {
		return errors.New(errors.CodeInvalidParameter, "input cannot be empty")
	}

	inputLen := len([]rune(input))
	if inputLen < p.config.MinInputLength {
		return errors.New(errors.CodeInvalidParameter, fmt.Sprintf("input too short (min: %d)", p.config.MinInputLength))
	}

	if inputLen > p.config.MaxInputLength {
		return errors.New(errors.CodeInvalidParameter, fmt.Sprintf("input too long (max: %d)", p.config.MaxInputLength))
	}

	return nil
}

// normalizeInput 规范化输入
func (p *parser) normalizeInput(input string) string {
	// 去除首尾空格
	normalized := strings.TrimSpace(input)

	// 替换多个空格为单个空格
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

	return normalized
}

// detectLanguage 检测语言
func (p *parser) detectLanguage(input string) string {
	// 简单的语言检测逻辑
	// 实际应该使用专门的语言检测库

	// 检测中文字符
	if regexp.MustCompile(`[\x{4e00}-\x{9fa5}]`).MatchString(input) {
		return "zh"
	}

	// 默认返回英语
	return p.config.DefaultLanguage
}

// matchPattern 模式匹配
func (p *parser) matchPattern(input string) (bool, *Intent, map[string]interface{}) {
	for _, pattern := range p.patterns {
		if matches := pattern.Pattern.FindStringSubmatch(input); matches != nil {
			intent := &Intent{
				Name:       pattern.Intent,
				Confidence: 1.0,
			}

			params := make(map[string]interface{})
			for paramName, groupIdx := range pattern.Parameters {
				if groupIdx < len(matches) {
					params[paramName] = matches[groupIdx]
				}
			}

			return true, intent, params
		}
	}

	return false, nil, nil
}

// extractEntities 提取实体
func (p *parser) extractEntities(input string) []*Entity {
	entities := make([]*Entity, 0)

	// 提取邮箱
	emailPattern := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	for _, match := range emailPattern.FindAllStringIndex(input, -1) {
		entities = append(entities, &Entity{
			Type:       EntityTypeEmail,
			Value:      input[match[0]:match[1]],
			StartPos:   match[0],
			EndPos:     match[1],
			Confidence: 1.0,
		})
	}

	// 提取URL
	urlPattern := regexp.MustCompile(`https?://[^\s]+`)
	for _, match := range urlPattern.FindAllStringIndex(input, -1) {
		entities = append(entities, &Entity{
			Type:       EntityTypeURL,
			Value:      input[match[0]:match[1]],
			StartPos:   match[0],
			EndPos:     match[1],
			Confidence: 1.0,
		})
	}

	// 提取电话号码
	phonePattern := regexp.MustCompile(`\+?[1-9]\d{1,14}`)
	for _, match := range phonePattern.FindAllStringIndex(input, -1) {
		entities = append(entities, &Entity{
			Type:       EntityTypePhone,
			Value:      input[match[0]:match[1]],
			StartPos:   match[0],
			EndPos:     match[1],
			Confidence: 0.8,
		})
	}

	return entities
}

// loadPredefinedPatterns 加载预定义模式
func (p *parser) loadPredefinedPatterns() {
	// 问候模式
	p.patterns = append(p.patterns, &ParsePattern{
		Name:     "greeting",
		Pattern:  regexp.MustCompile(`(?i)^(hi|hello|hey|greetings)`),
		Intent:   "greeting",
		Priority: 10,
	})

	// 告别模式
	p.patterns = append(p.patterns, &ParsePattern{
		Name:     "farewell",
		Pattern:  regexp.MustCompile(`(?i)^(bye|goodbye|see you|farewell)`),
		Intent:   "farewell",
		Priority: 10,
	})

	// 帮助请求模式
	p.patterns = append(p.patterns, &ParsePattern{
		Name:     "help_request",
		Pattern:  regexp.MustCompile(`(?i)(help|assist|support)`),
		Intent:   "help_request",
		Priority: 5,
	})
}

// getFromCache 从缓存获取
func (p *parser) getFromCache(input string) *ParseResponse {
	if resp, exists := p.cache.data[input]; exists {
		return resp
	}
	return nil
}

// putToCache 放入缓存
func (p *parser) putToCache(input string, resp *ParseResponse) {
	p.cache.data[input] = resp

	// 启动清理定时器
	time.AfterFunc(p.cache.ttl, func() {
		delete(p.cache.data, input)
	})
}

// intentDetector 意图检测器实现
type intentDetector struct {
	llmClient llm.LLMClient
	config    *ParserConfig
	logger    logger.Logger
}

func newIntentDetector(llmClient llm.LLMClient, config *ParserConfig, logger logger.Logger) IntentDetector {
	return &intentDetector{
		llmClient: llmClient,
		config:    config,
		logger:    logger,
	}
}

func (d *intentDetector) Detect(ctx context.Context, input string, options *DetectOptions) (*Intent, error) {
	if !options.UseLLM || d.llmClient == nil {
		// 使用规则引擎的简单意图检测
		return d.detectWithRules(input)
	}

	// 使用LLM进行意图检测
	return d.detectWithLLM(ctx, input, options)
}

func (d *intentDetector) detectWithRules(input string) (*Intent, error) {
	input = strings.ToLower(input)

	// 简单的规则判断
	if strings.Contains(input, "?") {
		return &Intent{
			Name:       "question",
			Type:       IntentTypeQuestion,
			Confidence: 0.7,
		}, nil
	}

	if strings.HasPrefix(input, "please") || strings.HasPrefix(input, "can you") {
		return &Intent{
			Name:       "request",
			Type:       IntentTypeRequest,
			Confidence: 0.7,
		}, nil
	}

	return &Intent{
		Name:       "statement",
		Type:       IntentTypeStatement,
		Confidence: 0.5,
	}, nil
}

func (d *intentDetector) detectWithLLM(ctx context.Context, input string, options *DetectOptions) (*Intent, error) {
	prompt := fmt.Sprintf(`Analyze the following user input and determine the intent.

User input: "%s"

Respond with a JSON object containing:
- name: the intent name
- type: the intent type (query, command, question, statement, request, etc.)
- confidence: confidence score (0-1)
- description: brief description of the intent

JSON:`, input)

	resp, err := d.llmClient.Complete(ctx, &llm.CompletionRequest{
		Prompt:      prompt,
		MaxTokens:   200,
		Temperature: 0.3,
	})
	if err != nil {
		return nil, err
	}

	var intent Intent
	if err := json.Unmarshal([]byte(resp.Choices[0].Text), &intent); err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to parse intent response")
	}

	return &intent, nil
}

// parameterExtractor 参数提取器实现
type parameterExtractor struct {
	llmClient llm.LLMClient
	logger    logger.Logger
}

func newParameterExtractor(llmClient llm.LLMClient, logger logger.Logger) ParameterExtractor {
	return &parameterExtractor{
		llmClient: llmClient,
		logger:    logger,
	}
}

func (e *parameterExtractor) Extract(ctx context.Context, input string, schema *ParameterSchema) (map[string]interface{}, error) {
	if schema == nil || len(schema.Parameters) == 0 {
		return make(map[string]interface{}), nil
	}

	// 使用LLM提取参数
	if e.llmClient != nil {
		return e.extractWithLLM(ctx, input, schema)
	}

	// 使用正则表达式提取参数
	return e.extractWithRegex(input, schema)
}

func (e *parameterExtractor) extractWithLLM(ctx context.Context, input string, schema *ParameterSchema) (map[string]interface{}, error) {
	schemaJSON, _ := json.Marshal(schema)

	prompt := fmt.Sprintf(`Extract parameters from the following user input based on the schema.

User input: "%s"

Parameter schema:
%s

Respond with a JSON object containing the extracted parameters.

JSON:`, input, string(schemaJSON))

	resp, err := e.llmClient.Complete(ctx, &llm.CompletionRequest{
		Prompt:      prompt,
		MaxTokens:   500,
		Temperature: 0.2,
	})
	if err != nil {
		return nil, err
	}

	var params map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Text), &params); err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to parse parameters")
	}

	return params, nil
}

func (e *parameterExtractor) extractWithRegex(input string, schema *ParameterSchema) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	for _, param := range schema.Parameters {
		if param.Pattern != "" {
			pattern := regexp.MustCompile(param.Pattern)
			if matches := pattern.FindStringSubmatch(input); len(matches) > 1 {
				params[param.Name] = matches[1]
			}
		}
	}

	return params, nil
}

// contextBuilder 上下文构建器实现
type contextBuilder struct {
	logger logger.Logger
}

func newContextBuilder(logger logger.Logger) ContextBuilder {
	return &contextBuilder{
		logger: logger,
	}
}

func (b *contextBuilder) Build(ctx context.Context, input string, history []*Message, existing map[string]interface{}) (*ContextInfo, error) {
	contextInfo := &ContextInfo{
		Timestamp: time.Now(),
		Variables: make(map[string]interface{}),
		State:     make(map[string]interface{}),
		Metadata:  make(map[string]interface{}),
	}

	// 合并现有上下文
	if existing != nil {
		for k, v := range existing {
			contextInfo.Variables[k] = v
		}
	}

	// 从历史消息中提取上下文
	if len(history) > 0 {
		lastMsg := history[len(history)-1]
		contextInfo.Metadata["last_message_time"] = lastMsg.Timestamp
		contextInfo.Metadata["conversation_length"] = len(history)
	}

	return contextInfo, nil
}

//Personal.AI order the ending
