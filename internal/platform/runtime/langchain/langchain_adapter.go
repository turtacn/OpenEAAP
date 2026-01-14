package langchain

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/openeeap/openeeap/internal/platform/runtime"
	"github.com/openeeap/openeeap/pkg/errors"
	"github.com/openeeap/openeeap/pkg/logger"
)

// LangChainAdapter LangChain适配器
type LangChainAdapter struct {
	id            string
	name          string
	version       string
	config        *runtime.RuntimeConfig
	logger        logger.Logger
	status        runtime.RuntimeStatus
	metadata      *runtime.RuntimeMetadata
	metrics       *adapterMetrics

	// LangChain客户端
	client        LangChainClient

	// 插件管理
	plugins       map[string]*runtime.Plugin
	pluginsMu     sync.RWMutex

	mu            sync.RWMutex
	shutdownChan  chan struct{}
	wg            sync.WaitGroup
}

// adapterMetrics 适配器指标
type adapterMetrics struct {
	totalRequests      int64
	successfulRequests int64
	failedRequests     int64
	activeRequests     int64
	totalTokens        int64
	totalCost          float64
	totalResponseTime  time.Duration
	mu                 sync.RWMutex
}

// LangChainClient LangChain客户端接口
type LangChainClient interface {
	// Initialize 初始化客户端
	Initialize(config *LangChainConfig) error

	// CreateChain 创建链
	CreateChain(chainConfig *ChainConfig) (Chain, error)

	// CreateAgent 创建代理
	CreateAgent(agentConfig *AgentConfig) (Agent, error)

	// Invoke 调用
	Invoke(ctx context.Context, input *InvokeInput) (*InvokeOutput, error)

	// Stream 流式调用
	Stream(ctx context.Context, input *InvokeInput) (<-chan *StreamOutput, <-chan error)

	// Close 关闭客户端
	Close() error
}

// Chain LangChain链接口
type Chain interface {
	// Run 运行链
	Run(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)

	// RunStream 流式运行链
	RunStream(ctx context.Context, input map[string]interface{}) (<-chan map[string]interface{}, <-chan error)

	// GetType 获取链类型
	GetType() string

	// GetConfig 获取配置
	GetConfig() *ChainConfig
}

// Agent LangChain代理接口
type Agent interface {
	// Execute 执行代理
	Execute(ctx context.Context, input string) (*AgentOutput, error)

	// ExecuteStream 流式执行代理
	ExecuteStream(ctx context.Context, input string) (<-chan *AgentStreamOutput, <-chan error)

	// AddTool 添加工具
	AddTool(tool *Tool) error

	// RemoveTool 移除工具
	RemoveTool(toolName string) error

	// ListTools 列出工具
	ListTools() []*Tool

	// GetConfig 获取配置
	GetConfig() *AgentConfig
}

// LangChainConfig LangChain配置
type LangChainConfig struct {
	APIKey          string                 // API密钥
	BaseURL         string                 // 基础URL
	Model           string                 // 模型
	Temperature     float64                // 温度
	MaxTokens       int                    // 最大令牌数
	Timeout         time.Duration          // 超时时间
	EnableCaching   bool                   // 启用缓存
	EnableTracing   bool                   // 启用追踪
	Verbose         bool                   // 详细模式
	Metadata        map[string]interface{} // 元数据
}

// ChainConfig 链配置
type ChainConfig struct {
	Type            ChainType              // 链类型
	Name            string                 // 链名称
	LLMConfig       *LLMConfig             // LLM配置
	PromptTemplate  string                 // 提示词模板
	Memory          *MemoryConfig          // 记忆配置
	Tools           []*Tool                // 工具列表
	OutputParser    string                 // 输出解析器
	Metadata        map[string]interface{} // 元数据
}

// ChainType 链类型
type ChainType string

const (
	ChainTypeLLM              ChainType = "llm"               // LLM链
	ChainTypeConversation     ChainType = "conversation"      // 对话链
	ChainTypeSequential       ChainType = "sequential"        // 顺序链
	ChainTypeRetrieval        ChainType = "retrieval"         // 检索链
	ChainTypeReAct            ChainType = "react"             // ReAct链
	ChainTypeConversationalReAct ChainType = "conversational_react" // 对话ReAct链
)

// AgentConfig 代理配置
type AgentConfig struct {
	Type            AgentType              // 代理类型
	Name            string                 // 代理名称
	LLMConfig       *LLMConfig             // LLM配置
	Tools           []*Tool                // 工具列表
	Memory          *MemoryConfig          // 记忆配置
	MaxIterations   int                    // 最大迭代次数
	MaxExecutionTime time.Duration         // 最大执行时间
	EarlyStoppingMethod string             // 早停方法
	Verbose         bool                   // 详细模式
	Metadata        map[string]interface{} // 元数据
}

// AgentType 代理类型
type AgentType string

const (
	AgentTypeZeroShot         AgentType = "zero_shot"          // Zero-shot代理
	AgentTypeReAct            AgentType = "react"              // ReAct代理
	AgentTypeConversational   AgentType = "conversational"     // 对话代理
	AgentTypeStructuredChat   AgentType = "structured_chat"    // 结构化聊天代理
	AgentTypeSelfAsk          AgentType = "self_ask"           // Self-ask代理
)

// LLMConfig LLM配置
type LLMConfig struct {
	Provider        string                 // 提供商
	Model           string                 // 模型
	Temperature     float64                // 温度
	MaxTokens       int                    // 最大令牌数
	TopP            float64                // Top-P
	FrequencyPenalty float64               // 频率惩罚
	PresencePenalty  float64               // 存在惩罚
	StopSequences   []string               // 停止序列
	Metadata        map[string]interface{} // 元数据
}

// MemoryConfig 记忆配置
type MemoryConfig struct {
	Type            MemoryType             // 记忆类型
	MaxTokenLimit   int                    // 最大令牌限制
	ReturnMessages  bool                   // 返回消息
	InputKey        string                 // 输入键
	OutputKey       string                 // 输出键
	MemoryKey       string                 // 记忆键
	Metadata        map[string]interface{} // 元数据
}

// MemoryType 记忆类型
type MemoryType string

const (
	MemoryTypeBuffer          MemoryType = "buffer"           // 缓冲记忆
	MemoryTypeBufferWindow    MemoryType = "buffer_window"    // 窗口缓冲记忆
	MemoryTypeSummary         MemoryType = "summary"          // 摘要记忆
	MemoryTypeConversation    MemoryType = "conversation"     // 对话记忆
	MemoryTypeEntity          MemoryType = "entity"           // 实体记忆
	MemoryTypeKnowledgeGraph  MemoryType = "knowledge_graph"  // 知识图谱记忆
)

// Tool 工具
type Tool struct {
	Name        string                 // 工具名称
	Description string                 // 工具描述
	Function    string                 // 函数名称
	Parameters  map[string]interface{} // 参数模式
	ReturnDirect bool                  // 直接返回
	Metadata    map[string]interface{} // 元数据
}

// InvokeInput 调用输入
type InvokeInput struct {
	ChainType   ChainType              // 链类型
	AgentType   AgentType              // 代理类型
	Input       string                 // 输入
	Variables   map[string]interface{} // 变量
	Context     map[string]interface{} // 上下文
	Options     map[string]interface{} // 选项
}

// InvokeOutput 调用输出
type InvokeOutput struct {
	Output      string                 // 输出
	TokensUsed  int                    // 使用的令牌数
	IntermediateSteps []IntermediateStep // 中间步骤
	Metadata    map[string]interface{} // 元数据
	Error       error                  // 错误
}

// StreamOutput 流式输出
type StreamOutput struct {
	Content     string                 // 内容
	Delta       string                 // 增量内容
	Finished    bool                   // 是否完成
	Metadata    map[string]interface{} // 元数据
	Timestamp   time.Time              // 时间戳
}

// AgentOutput 代理输出
type AgentOutput struct {
	Output      string                 // 输出
	TokensUsed  int                    // 使用的令牌数
	Steps       []AgentStep            // 代理步骤
	Metadata    map[string]interface{} // 元数据
}

// AgentStreamOutput 代理流式输出
type AgentStreamOutput struct {
	Content     string                 // 内容
	Delta       string                 // 增量内容
	StepUpdate  *AgentStep             // 步骤更新
	Finished    bool                   // 是否完成
	Metadata    map[string]interface{} // 元数据
	Timestamp   time.Time              // 时间戳
}

// AgentStep 代理步骤
type AgentStep struct {
	Action      string                 // 动作
	ActionInput map[string]interface{} // 动作输入
	Observation string                 // 观察
	Thought     string                 // 思考
	Timestamp   time.Time              // 时间戳
}

// IntermediateStep 中间步骤
type IntermediateStep struct {
	Action      string                 // 动作
	Input       interface{}            // 输入
	Output      interface{}            // 输出
	Metadata    map[string]interface{} // 元数据
}

// NewLangChainAdapter 创建LangChain适配器
func NewLangChainAdapter(
	config *runtime.RuntimeConfig,
	logger logger.Logger,
	client LangChainClient,
) (*LangChainAdapter, error) {
	if config == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "config cannot be nil")
	}
	if logger == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "logger cannot be nil")
	}
	if client == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "client cannot be nil")
	}

	adapter := &LangChainAdapter{
		id:           config.ID,
		name:         config.Name,
		version:      config.Version,
		config:       config,
		logger:       logger,
		client:       client,
		status:       runtime.RuntimeStatusInitializing,
		plugins:      make(map[string]*runtime.Plugin),
		metrics:      &adapterMetrics{},
		shutdownChan: make(chan struct{}),
	}

	// 初始化元数据
	adapter.metadata = &runtime.RuntimeMetadata{
		ID:          config.ID,
		Type:        runtime.RuntimeTypeLangChain,
		Name:        config.Name,
		Version:     config.Version,
		Description: "LangChain Runtime Adapter",
		Author:      "OpenEEAP",
		License:     "MIT",
		Capabilities: []string{
			"chains",
			"agents",
			"tools",
			"memory",
			"streaming",
			"retrieval",
		},
		Features: &runtime.RuntimeFeatures{
			SupportsStreaming:    config.EnableStreaming,
			SupportsPlugins:      true,
			SupportsMemory:       true,
			SupportsRetrieval:    true,
			SupportsFunctionCall: true,
			SupportsAsync:        true,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return adapter, nil
}

// ID 获取运行时ID
func (lca *LangChainAdapter) ID() string {
	return lca.id
}

// Type 获取运行时类型
func (lca *LangChainAdapter) Type() runtime.RuntimeType {
	return runtime.RuntimeTypeLangChain
}

// Name 获取运行时名称
func (lca *LangChainAdapter) Name() string {
	return lca.name
}

// Version 获取运行时版本
func (lca *LangChainAdapter) Version() string {
	return lca.version
}

// Initialize 初始化运行时
func (lca *LangChainAdapter) Initialize(ctx context.Context, config *runtime.RuntimeConfig) error {
	lca.mu.Lock()
	defer lca.mu.Unlock()

	if lca.status == runtime.RuntimeStatusReady {
		return errors.New(errors.CodeInternalError, "runtime already initialized")
	}

	// 验证配置
	if err := lca.validateConfig(config); err != nil {
		return errors.Wrap(err, errors.CodeInvalidParameter, "invalid configuration")
	}

	// 更新配置
	if config != nil {
		lca.config = config
	}

	// 初始化LangChain客户端
	lcConfig := lca.buildLangChainConfig()
	if err := lca.client.Initialize(lcConfig); err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to initialize langchain client")
	}

	// 加载插件
	if len(lca.config.Plugins) > 0 {
		for _, pluginConfig := range lca.config.Plugins {
			plugin := lca.convertPluginConfig(pluginConfig)
			if err := lca.LoadPlugin(ctx, plugin); err != nil {
				lca.logger.Warn("failed to load plugin",
					"plugin_id", plugin.ID,
					"error", err)
			}
		}
	}

	// 启动健康检查
	if lca.config.EnableHealthCheck {
		lca.wg.Add(1)
		go lca.healthCheckLoop()
	}

	lca.status = runtime.RuntimeStatusReady
	lca.logger.Info("langchain adapter initialized",
		"runtime_id", lca.id,
		"version", lca.version)

	return nil
}

// Execute 执行任务
func (lca *LangChainAdapter) Execute(ctx context.Context, req *runtime.ExecuteRequest) (*runtime.ExecuteResponse, error) {
	if !lca.IsReady() {
		return nil, errors.New(errors.CodeInternalError, "runtime not ready")
	}

	// 更新指标
	lca.metrics.mu.Lock()
	lca.metrics.totalRequests++
	lca.metrics.activeRequests++
	lca.metrics.mu.Unlock()

	defer func() {
		lca.metrics.mu.Lock()
		lca.metrics.activeRequests--
		lca.metrics.mu.Unlock()
	}()

	startTime := time.Now()

	// 转换请求格式
	invokeInput := lca.convertToInvokeInput(req)

	// 调用LangChain
	output, err := lca.client.Invoke(ctx, invokeInput)

	duration := time.Since(startTime)

	// 更新指标
	lca.metrics.mu.Lock()
	if err != nil {
		lca.metrics.failedRequests++
	} else {
		lca.metrics.successfulRequests++
		lca.metrics.totalTokens += int64(output.TokensUsed)
		lca.metrics.totalCost += lca.estimateCost(output.TokensUsed)
	}
	lca.metrics.totalResponseTime += duration
	lca.metrics.mu.Unlock()

	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "langchain execution failed")
	}

	// 转换响应格式
	resp := lca.convertFromInvokeOutput(output, duration)

	lca.logger.Debug("langchain execution completed",
		"request_id", req.AgentID,
		"duration", duration,
		"tokens_used", output.TokensUsed)

	return resp, nil
}

// ExecuteStream 流式执行任务
func (lca *LangChainAdapter) ExecuteStream(ctx context.Context, req *runtime.ExecuteRequest) (<-chan *runtime.StreamChunk, <-chan error) {
	chunkChan := make(chan *runtime.StreamChunk, 10)
	errChan := make(chan error, 1)

	if !lca.config.EnableStreaming {
		errChan <- errors.New(errors.CodeInvalidParameter, "streaming not enabled")
		close(chunkChan)
		close(errChan)
		return chunkChan, errChan
	}

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		// 转换请求格式
		invokeInput := lca.convertToInvokeInput(req)

		// 调用LangChain流式API
		streamChan, streamErrChan := lca.client.Stream(ctx, invokeInput)

		fullContent := ""
		for {
			select {
			case streamOutput, ok := <-streamChan:
				if !ok {
					return
				}

				fullContent += streamOutput.Delta

				// 转换流式输出
				chunk := &runtime.StreamChunk{
					Content:   fullContent,
					Delta:     streamOutput.Delta,
					Finished:  streamOutput.Finished,
					Metadata:  streamOutput.Metadata,
					Timestamp: streamOutput.Timestamp,
				}

				select {
				case chunkChan <- chunk:
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				}

				if streamOutput.Finished {
					return
				}

			case err := <-streamErrChan:
				if err != nil {
					errChan <- errors.Wrap(err, errors.CodeInternalError, "langchain stream failed")
				}
				return

			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}
		}
	}()

	return chunkChan, errChan
}

// HealthCheck 健康检查
func (lca *LangChainAdapter) HealthCheck(ctx context.Context) (*runtime.HealthCheckResult, error) {
	startTime := time.Now()

	result := &runtime.HealthCheckResult{
		Status:    runtime.HealthStatusHealthy,
		Details:   make(map[string]interface{}),
		Timestamp: time.Now(),
		Checks:    make([]*runtime.ComponentHealth, 0),
	}

	// 检查LangChain客户端
	clientHealth := &runtime.ComponentHealth{
		Name:      "langchain_client",
		Status:    runtime.HealthStatusHealthy,
		Message:   "LangChain client healthy",
		Timestamp: time.Now(),
	}
	result.Checks = append(result.Checks, clientHealth)

	// 检查插件
	lca.pluginsMu.RLock()
	pluginCount := len(lca.plugins)
	lca.pluginsMu.RUnlock()

	pluginHealth := &runtime.ComponentHealth{
		Name:      "plugins",
		Status:    runtime.HealthStatusHealthy,
		Message:   fmt.Sprintf("%d plugins loaded", pluginCount),
		Timestamp: time.Now(),
	}
	result.Checks = append(result.Checks, pluginHealth)

	result.ResponseTime = time.Since(startTime)

	// 添加指标信息
	lca.metrics.mu.RLock()
	result.Details["total_requests"] = lca.metrics.totalRequests
	result.Details["active_requests"] = lca.metrics.activeRequests
	result.Details["success_rate"] = lca.calculateSuccessRate()
	lca.metrics.mu.RUnlock()

	return result, nil
}

// Metadata 获取运行时元数据
func (lca *LangChainAdapter) Metadata() *runtime.RuntimeMetadata {
	lca.mu.RLock()
	defer lca.mu.RUnlock()
	return lca.metadata
}

// Shutdown 关闭运行时
func (lca *LangChainAdapter) Shutdown(ctx context.Context) error {
	lca.mu.Lock()
	defer lca.mu.Unlock()

	if lca.status == runtime.RuntimeStatusShutdown {
		return nil
	}

	lca.logger.Info("shutting down langchain adapter", "runtime_id", lca.id)

	// 发送关闭信号
	close(lca.shutdownChan)

	// 等待所有goroutine完成
	done := make(chan struct{})
	go func() {
		lca.wg.Wait()
		close(done)
	}()

	// 等待关闭完成或超时
	select {
	case <-done:
		lca.logger.Info("langchain adapter shutdown completed", "runtime_id", lca.id)
	case <-ctx.Done():
		return errors.New(errors.CodeDeadlineExceeded, "shutdown timeout")
	}

	// 关闭客户端
	if err := lca.client.Close(); err != nil {
		lca.logger.Warn("failed to close langchain client", "error", err)
	}

	lca.status = runtime.RuntimeStatusShutdown
	return nil
}

// IsReady 检查是否就绪
func (lca *LangChainAdapter) IsReady() bool {
	lca.mu.RLock()
	defer lca.mu.RUnlock()
	return lca.status == runtime.RuntimeStatusReady
}

// GetStatus 获取运行时状态
func (lca *LangChainAdapter) GetStatus() runtime.RuntimeStatus {
	lca.mu.RLock()
	defer lca.mu.RUnlock()
	return lca.status
}

// UpdateConfig 更新配置
func (lca *LangChainAdapter) UpdateConfig(ctx context.Context, config *runtime.RuntimeConfig) error {
	lca.mu.Lock()
	defer lca.mu.Unlock()

	if err := lca.validateConfig(config); err != nil {
		return errors.Wrap(err, errors.CodeInvalidParameter, "invalid configuration")
	}

	lca.config = config
	lca.metadata.UpdatedAt = time.Now()

	lca.logger.Info("langchain adapter configuration updated", "runtime_id", lca.id)
	return nil
}

// LoadPlugin 加载插件
func (lca *LangChainAdapter) LoadPlugin(ctx context.Context, plugin *runtime.Plugin) error {
	if plugin == nil {
		return errors.New(errors.CodeInvalidParameter, "plugin cannot be nil")
	}

	lca.pluginsMu.Lock()
	defer lca.pluginsMu.Unlock()

	// 检查插件是否已加载
	if _, exists := lca.plugins[plugin.ID]; exists {
		return errors.New(errors.CodeAlreadyExists, "plugin already loaded")
	}

	// 验证插件
	if err := lca.ValidatePlugin(ctx, plugin); err != nil {
		return errors.Wrap(err, errors.CodeInvalidParameter, "plugin validation failed")
	}

	// 加载插件（实际实现需要调用LangChain SDK）
	plugin.Status = runtime.PluginStatusLoaded
	plugin.UpdatedAt = time.Now()

	lca.plugins[plugin.ID] = plugin

	lca.logger.Info("plugin loaded",
		"plugin_id", plugin.ID,
		"plugin_name", plugin.Name,
		"plugin_type", plugin.Type)

	return nil
}

// UnloadPlugin 卸载插件
func (lca *LangChainAdapter) UnloadPlugin(ctx context.Context, pluginID string) error {
	if pluginID == "" {
		return errors.New(errors.CodeInvalidParameter, "plugin id cannot be empty")
	}

	lca.pluginsMu.Lock()
	defer lca.pluginsMu.Unlock()

	plugin, exists := lca.plugins[pluginID]
	if !exists {
		return errors.New(errors.CodeNotFound, "plugin not found")
	}

	// 卸载插件（实际实现需要调用LangChain SDK）
	plugin.Status = runtime.PluginStatusUnloaded
	plugin.UpdatedAt = time.Now()

	delete(lca.plugins, pluginID)

	lca.logger.Info("plugin unloaded", "plugin_id", pluginID)
	return nil
}

// ListPlugins 列出所有插件
func (lca *LangChainAdapter) ListPlugins() ([]*runtime.Plugin, error) {
	lca.pluginsMu.RLock()
	defer lca.pluginsMu.RUnlock()

	plugins := make([]*runtime.Plugin, 0, len(lca.plugins))
	for _, plugin := range lca.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

// GetPlugin 获取插件
func (lca *LangChainAdapter) GetPlugin(pluginID string) (*runtime.Plugin, error) {
	if pluginID == "" {
		return nil, errors.New(errors.CodeInvalidParameter, "plugin id cannot be empty")
	}

	lca.pluginsMu.RLock()
	defer lca.pluginsMu.RUnlock()

	plugin, exists := lca.plugins[pluginID]
	if !exists {
		return nil, errors.New(errors.CodeNotFound, "plugin not found")
	}

	return plugin, nil
}

// ExecutePlugin 执行插件
func (lca *LangChainAdapter) ExecutePlugin(ctx context.Context, pluginID string, req *runtime.PluginExecuteRequest) (*runtime.PluginExecuteResponse, error) {
	plugin, err := lca.GetPlugin(pluginID)
	if err != nil {
		return nil, err
	}

	if plugin.Status != runtime.PluginStatusLoaded && plugin.Status != runtime.PluginStatusActive {
		return nil, errors.New(errors.CodeInvalidParameter, "plugin not active")
	}

	startTime := time.Now()

	// 执行插件（实际实现需要调用LangChain SDK）
	// 这里是简化实现
	output := make(map[string]interface{})
	output["result"] = "plugin executed successfully"

	duration := time.Since(startTime)

	resp := &runtime.PluginExecuteResponse{
		PluginID:  pluginID,
		Output:    output,
		Duration:  duration,
		Metadata:  make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	lca.logger.Debug("plugin executed",
		"plugin_id", pluginID,
		"duration", duration)

	return resp, nil
}

// ValidatePlugin 验证插件
func (lca *LangChainAdapter) ValidatePlugin(ctx context.Context, plugin *runtime.Plugin) error {
	if plugin == nil {
		return errors.New(errors.CodeInvalidParameter, "plugin cannot be nil")
	}

	if plugin.ID == "" {
		return errors.New(errors.CodeInvalidParameter, "plugin id cannot be empty")
	}

	if plugin.Name == "" {
		return errors.New(errors.CodeInvalidParameter, "plugin name cannot be empty")
	}

	if plugin.Type == "" {
		return errors.New(errors.CodeInvalidParameter, "plugin type cannot be empty")
	}

	// 验证插件类型是否支持
	supportedTypes := []runtime.PluginType{
		runtime.PluginTypeTool,
		runtime.PluginTypeMemory,
		runtime.PluginTypeRetriever,
		runtime.PluginTypeParser,
	}

	supported := false
	for _, t := range supportedTypes {
		if plugin.Type == t {
			supported = true
			break
		}
	}

	if !supported {
		return errors.New(errors.CodeInvalidParameter,
			fmt.Sprintf("unsupported plugin type: %s", plugin.Type))
	}

	return nil
}

// 私有方法

// validateConfig 验证配置
func (lca *LangChainAdapter) validateConfig(config *runtime.RuntimeConfig) error {
	if config == nil {
		return errors.New(errors.CodeInvalidParameter, "config cannot be nil")
	}

	if config.ID == "" {
		return errors.New(errors.CodeInvalidParameter, "runtime id cannot be empty")
	}

	if config.Type != runtime.RuntimeTypeLangChain {
		return errors.New(errors.CodeInvalidParameter, "invalid runtime type")
	}

	return nil
}

// buildLangChainConfig 构建LangChain配置
func (lca *LangChainAdapter) buildLangChainConfig() *LangChainConfig {
	return &LangChainConfig{
		APIKey:        lca.config.Environment["LANGCHAIN_API_KEY"],
		BaseURL:       lca.config.Endpoint,
		Timeout:       lca.config.Timeout,
		EnableCaching: lca.config.EnableResultCaching,
		EnableTracing: lca.config.EnableTracing,
		Verbose:       false,
		Metadata:      lca.config.Metadata,
	}
}

// convertToInvokeInput 转换为调用输入
func (lca *LangChainAdapter) convertToInvokeInput(req *runtime.ExecuteRequest) *InvokeInput {
	input := &InvokeInput{
		Input:     req.Input,
		Variables: make(map[string]interface{}),
		Context:   req.Context,
		Options:   make(map[string]interface{}),
	}

	// 根据请求选项确定使用链还是代理
	if req.Options != nil {
		if req.Options.EnablePlugins && len(req.Options.Plugins) > 0 {
			input.AgentType = AgentTypeReAct
			input.Options["tools"] = req.Options.Plugins
		} else {
			input.ChainType = ChainTypeConversation
		}

		// 转换其他选项
		if req.Options.MaxTokens > 0 {
			input.Options["max_tokens"] = req.Options.MaxTokens
		}
		if req.Options.Temperature > 0 {
			input.Options["temperature"] = req.Options.Temperature
		}
		if req.Options.TopP > 0 {
			input.Options["top_p"] = req.Options.TopP
		}
		if len(req.Options.StopSequences) > 0 {
			input.Options["stop_sequences"] = req.Options.StopSequences
		}
	}

	// 转换上下文变量
	if req.Context != nil {
		for k, v := range req.Context {
			input.Variables[k] = v
		}
	}

	return input
}

// convertFromInvokeOutput 转换调用输出
func (lca *LangChainAdapter) convertFromInvokeOutput(output *InvokeOutput, duration time.Duration) *runtime.ExecuteResponse {
	resp := &runtime.ExecuteResponse{
		Output:       output.Output,
		TokensUsed:   output.TokensUsed,
		Duration:     duration,
		Metadata:     make(map[string]interface{}),
		Error:        output.Error,
	}

	// 转换中间步骤
	if len(output.IntermediateSteps) > 0 {
		steps := make([]map[string]interface{}, 0, len(output.IntermediateSteps))
		for _, step := range output.IntermediateSteps {
			steps = append(steps, map[string]interface{}{
				"action": step.Action,
				"input":  step.Input,
				"output": step.Output,
			})
		}
		resp.Metadata["intermediate_steps"] = steps
	}

	// 添加其他元数据
	if output.Metadata != nil {
		for k, v := range output.Metadata {
			resp.Metadata[k] = v
		}
	}

	// 添加成本估算
	resp.Metadata["estimated_cost"] = lca.estimateCost(output.TokensUsed)

	return resp
}

// convertPluginConfig 转换插件配置
func (lca *LangChainAdapter) convertPluginConfig(config *runtime.PluginConfig) *runtime.Plugin {
	return &runtime.Plugin{
		ID:           config.ID,
		Name:         config.Name,
		Version:      config.Version,
		Type:         config.Type,
		Description:  fmt.Sprintf("Plugin: %s", config.Name),
		Enabled:      config.Enabled,
		Path:         config.Path,
		EntryPoint:   config.EntryPoint,
		Config:       config.Config,
		Dependencies: config.Dependencies,
		Permissions:  config.Permissions,
		Status:       runtime.PluginStatusInactive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Metadata:     config.Metadata,
	}
}

// healthCheckLoop 健康检查循环
func (lca *LangChainAdapter) healthCheckLoop() {
	defer lca.wg.Done()

	ticker := time.NewTicker(lca.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, err := lca.HealthCheck(ctx)
			cancel()

			if err != nil {
				lca.logger.Warn("health check failed", "error", err)
			}
		case <-lca.shutdownChan:
			return
		}
	}
}

// calculateSuccessRate 计算成功率
func (lca *LangChainAdapter) calculateSuccessRate() float64 {
	if lca.metrics.totalRequests == 0 {
		return 0.0
	}
	return float64(lca.metrics.successfulRequests) / float64(lca.metrics.totalRequests)
}

// estimateCost 估算成本
func (lca *LangChainAdapter) estimateCost(tokens int) float64 {
	// 简化计算：假设每1K令牌$0.002
	return float64(tokens) / 1000.0 * 0.002
}

// ConvertToolToLangChain 转换工具为LangChain格式
func ConvertToolToLangChain(tool interface{}) *Tool {
	// 简化实现，实际需要根据工具类型进行转换
	return &Tool{
		Name:        "custom_tool",
		Description: "Custom tool",
		Parameters:  make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
	}
}

// ConvertMemoryToLangChain 转换记忆为LangChain格式
func ConvertMemoryToLangChain(memory interface{}) *MemoryConfig {
	// 简化实现，实际需要根据记忆类型进行转换
	return &MemoryConfig{
		Type:          MemoryTypeBuffer,
		MaxTokenLimit: 2000,
		ReturnMessages: true,
		InputKey:      "input",
		OutputKey:     "output",
		MemoryKey:     "history",
	}
}

// BuildChainFromConfig 从配置构建链
func (lca *LangChainAdapter) BuildChainFromConfig(config *ChainConfig) (Chain, error) {
	if config == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "chain config cannot be nil")
	}

	// 调用LangChain客户端创建链
	chain, err := lca.client.CreateChain(config)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to create chain")
	}

	lca.logger.Info("chain created",
		"chain_type", config.Type,
		"chain_name", config.Name)

	return chain, nil
}

// BuildAgentFromConfig 从配置构建代理
func (lca *LangChainAdapter) BuildAgentFromConfig(config *AgentConfig) (Agent, error) {
	if config == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "agent config cannot be nil")
	}

	// 调用LangChain客户端创建代理
	agent, err := lca.client.CreateAgent(config)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to create agent")
	}

	lca.logger.Info("agent created",
		"agent_type", config.Type,
		"agent_name", config.Name)

	return agent, nil
}

// SerializeChainConfig 序列化链配置
func SerializeChainConfig(config *ChainConfig) ([]byte, error) {
	if config == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "config cannot be nil")
	}
	return json.Marshal(config)
}

// DeserializeChainConfig 反序列化链配置
func DeserializeChainConfig(data []byte) (*ChainConfig, error) {
	if len(data) == 0 {
		return nil, errors.New(errors.CodeInvalidParameter, "data cannot be empty")
	}

	var config ChainConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to unmarshal config")
	}

	return &config, nil
}

// SerializeAgentConfig 序列化代理配置
func SerializeAgentConfig(config *AgentConfig) ([]byte, error) {
	if config == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "config cannot be nil")
	}
	return json.Marshal(config)
}

// DeserializeAgentConfig 反序列化代理配置
func DeserializeAgentConfig(data []byte) (*AgentConfig, error) {
	if len(data) == 0 {
		return nil, errors.New(errors.CodeInvalidParameter, "data cannot be empty")
	}

	var config AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to unmarshal config")
	}

	return &config, nil
}

// GetMetrics 获取适配器指标
func (lca *LangChainAdapter) GetMetrics() map[string]interface{} {
	lca.metrics.mu.RLock()
	defer lca.metrics.mu.RUnlock()

	avgResponseTime := time.Duration(0)
	if lca.metrics.totalRequests > 0 {
		avgResponseTime = lca.metrics.totalResponseTime / time.Duration(lca.metrics.totalRequests)
	}

	return map[string]interface{}{
		"total_requests":       lca.metrics.totalRequests,
		"successful_requests":  lca.metrics.successfulRequests,
		"failed_requests":      lca.metrics.failedRequests,
		"active_requests":      lca.metrics.activeRequests,
		"total_tokens":         lca.metrics.totalTokens,
		"total_cost":           lca.metrics.totalCost,
		"avg_response_time":    avgResponseTime,
		"success_rate":         lca.calculateSuccessRate(),
	}
}

// ResetMetrics 重置指标
func (lca *LangChainAdapter) ResetMetrics() {
	lca.metrics.mu.Lock()
	defer lca.metrics.mu.Unlock()

	lca.metrics.totalRequests = 0
	lca.metrics.successfulRequests = 0
	lca.metrics.failedRequests = 0
	lca.metrics.totalTokens = 0
	lca.metrics.totalCost = 0
	lca.metrics.totalResponseTime = 0

	lca.logger.Info("metrics reset", "runtime_id", lca.id)
}

// GetPluginCount 获取插件数量
func (lca *LangChainAdapter) GetPluginCount() int {
	lca.pluginsMu.RLock()
	defer lca.pluginsMu.RUnlock()
	return len(lca.plugins)
}

// GetActivePluginCount 获取活跃插件数量
func (lca *LangChainAdapter) GetActivePluginCount() int {
	lca.pluginsMu.RLock()
	defer lca.pluginsMu.RUnlock()

	count := 0
	for _, plugin := range lca.plugins {
		if plugin.Status == runtime.PluginStatusActive {
			count++
		}
	}
	return count
}

// EnablePlugin 启用插件
func (lca *LangChainAdapter) EnablePlugin(ctx context.Context, pluginID string) error {
	lca.pluginsMu.Lock()
	defer lca.pluginsMu.Unlock()

	plugin, exists := lca.plugins[pluginID]
	if !exists {
		return errors.New(errors.CodeNotFound, "plugin not found")
	}

	if plugin.Status == runtime.PluginStatusActive {
		return errors.New(errors.CodeInvalidParameter, "plugin already active")
	}

	plugin.Status = runtime.PluginStatusActive
	plugin.Enabled = true
	plugin.UpdatedAt = time.Now()

	lca.logger.Info("plugin enabled", "plugin_id", pluginID)
	return nil
}

// DisablePlugin 禁用插件
func (lca *LangChainAdapter) DisablePlugin(ctx context.Context, pluginID string) error {
	lca.pluginsMu.Lock()
	defer lca.pluginsMu.Unlock()

	plugin, exists := lca.plugins[pluginID]
	if !exists {
		return errors.New(errors.CodeNotFound, "plugin not found")
	}

	if plugin.Status == runtime.PluginStatusInactive {
		return errors.New(errors.CodeInvalidParameter, "plugin already inactive")
	}

	plugin.Status = runtime.PluginStatusInactive
	plugin.Enabled = false
	plugin.UpdatedAt = time.Now()

	lca.logger.Info("plugin disabled", "plugin_id", pluginID)
	return nil
}

// UpdatePluginConfig 更新插件配置
func (lca *LangChainAdapter) UpdatePluginConfig(ctx context.Context, pluginID string, config map[string]interface{}) error {
	lca.pluginsMu.Lock()
	defer lca.pluginsMu.Unlock()

	plugin, exists := lca.plugins[pluginID]
	if !exists {
		return errors.New(errors.CodeNotFound, "plugin not found")
	}

	if config != nil {
		plugin.Config = config
		plugin.UpdatedAt = time.Now()
	}

	lca.logger.Info("plugin config updated", "plugin_id", pluginID)
	return nil
}

//Personal.AI order the ending
