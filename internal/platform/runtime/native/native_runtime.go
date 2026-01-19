package native

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/openeeap/openeeap/internal/domain/agent"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/platform/runtime"
	"github.com/openeeap/openeeap/pkg/errors"
)

// NativeRuntime 原生运行时实现
type NativeRuntime struct {
	id             string
	name           string
	version        string
	config         *runtime.RuntimeConfig
	logger         logging.Logger
	// llmClient      llm.LLMClient
	toolManager    ToolManager
	memoryManager  MemoryManager
	status         runtime.RuntimeStatus
	metadata       *runtime.RuntimeMetadata
	metrics        *runtimeMetrics
	mu             sync.RWMutex
	shutdownChan   chan struct{}
	wg             sync.WaitGroup
}

// runtimeMetrics 运行时指标
type runtimeMetrics struct {
	totalRequests       int64
	successfulRequests  int64
	failedRequests      int64
	activeRequests      int64
	totalTokens         int64
	totalCost           float64
	totalResponseTime   time.Duration
	mu                  sync.RWMutex
}

// ToolManager 工具管理器接口
type ToolManager interface {
	// GetTool 获取工具
	GetTool(name string) (Tool, error)

	// ListTools 列出所有工具
	ListTools() []Tool

	// ExecuteTool 执行工具
	ExecuteTool(ctx context.Context, name string, input map[string]interface{}) (interface{}, error)
}

// Tool 工具接口
type Tool interface {
	// Name 工具名称
	Name() string

	// Description 工具描述
	Description() string

	// Execute 执行工具
	Execute(ctx context.Context, input map[string]interface{}) (interface{}, error)

	// Schema 工具参数模式
	Schema() map[string]interface{}
}

// MemoryManager 记忆管理器接口
type MemoryManager interface {
	// Store 存储记忆
	Store(ctx context.Context, memory *Memory) error

	// Retrieve 检索记忆
	Retrieve(ctx context.Context, query string, limit int) ([]*Memory, error)

	// Clear 清除记忆
	Clear(ctx context.Context, sessionID string) error
}

// Memory 记忆
type Memory struct {
	ID        string                 // 记忆ID
	SessionID string                 // 会话ID
	Type      MemoryType             // 记忆类型
	Content   string                 // 内容
	Metadata  map[string]interface{} // 元数据
	Timestamp time.Time              // 时间戳
}

// MemoryType 记忆类型
type MemoryType string

const (
	MemoryTypeConversation MemoryType = "conversation" // 对话记忆
	MemoryTypeContext      MemoryType = "context"      // 上下文记忆
	MemoryTypeLongTerm     MemoryType = "long_term"    // 长期记忆
	MemoryTypeWorking      MemoryType = "working"      // 工作记忆
)

// ReActStep ReAct步骤
type ReActStep struct {
	StepNumber int                    // 步骤编号
	Thought    string                 // 思考
	Action     string                 // 动作
	ActionInput map[string]interface{} // 动作输入
	Observation string                 // 观察
	Timestamp   time.Time              // 时间戳
}

// ReActResult ReAct结果
type ReActResult struct {
	Steps        []*ReActStep // 步骤列表
	FinalAnswer  string       // 最终答案
	TokensUsed   int          // 使用的令牌数
	TotalCost    float64      // 总成本
	Duration     time.Duration // 执行时长
	Error        error        // 错误
}

const (
	// ReAct模式的系统提示词
	reactSystemPrompt = `You are an AI assistant that uses the ReAct (Reasoning and Acting) pattern to solve problems.

For each task, follow these steps:
1. Thought: Think about what you need to do
2. Action: Choose an action to take (use a tool or provide a final answer)
3. Observation: Observe the result of the action
4. Repeat steps 1-3 until you can provide a final answer

Available actions:
{{TOOLS}}

When you want to use a tool, format your response as:
Thought: [your reasoning]
Action: [tool_name]
Action Input: [JSON input for the tool]

When you have enough information to answer, format your response as:
Thought: [your reasoning]
Final Answer: [your answer]

Important:
- Always start with a Thought
- Only use one Action per response
- Action Input must be valid JSON
- Continue reasoning until you can provide a Final Answer`

	// 最大ReAct步骤数
	maxReActSteps = 10
)

// NewNativeRuntime 创建原生运行时
func NewNativeRuntime(
	config *runtime.RuntimeConfig,
	logger logger.Logger,
	llmClient llm.LLMClient,
	toolManager ToolManager,
	memoryManager MemoryManager,
) (*NativeRuntime, error) {
	if config == nil {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "config cannot be nil")
	}
	if logger == nil {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "logger cannot be nil")
	}
	if llmClient == nil {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "llm client cannot be nil")
	}

	nr := &NativeRuntime{
		id:            config.ID,
		name:          config.Name,
		version:       config.Version,
		config:        config,
		logger:        logger,
		llmClient:     llmClient,
		toolManager:   toolManager,
		memoryManager: memoryManager,
		status:        runtime.RuntimeStatusInitializing,
		metrics:       &runtimeMetrics{},
		shutdownChan:  make(chan struct{}),
	}

	// 初始化元数据
	nr.metadata = &runtime.RuntimeMetadata{
		ID:          config.ID,
		Type:        runtime.RuntimeTypeNative,
		Name:        config.Name,
		Version:     config.Version,
		Description: "Native ReAct-based Agent Runtime",
		Author:      "OpenEEAP",
		License:     "MIT",
		Capabilities: []string{
			"reasoning",
			"tool_use",
			"memory",
			"streaming",
		},
		Features: &runtime.RuntimeFeatures{
			SupportsStreaming:    config.EnableStreaming,
			SupportsPlugins:      false,
			SupportsMemory:       memoryManager != nil,
			SupportsRetrieval:    false,
			SupportsFunctionCall: true,
			SupportsAsync:        true,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
//
	return nr, nil

// ID 获取运行时ID
func (nr *NativeRuntime) ID() string {
	return nr.id
}

// Type 获取运行时类型
func (nr *NativeRuntime) Type() runtime.RuntimeType {
	return runtime.RuntimeTypeNative
}

// Name 获取运行时名称
func (nr *NativeRuntime) Name() string {
	return nr.name
}

// Version 获取运行时版本
func (nr *NativeRuntime) Version() string {
	return nr.version
}

// Initialize 初始化运行时
func (nr *NativeRuntime) Initialize(ctx context.Context, config *runtime.RuntimeConfig) error {
	nr.mu.Lock()
	defer nr.mu.Unlock()

	if nr.status == runtime.RuntimeStatusReady {
		return errors.NewInternalError("ERR_INTERNAL", "runtime already initialized")
	}

	// 验证配置
	if err := nr.validateConfig(config); err != nil {
		return errors.Wrap(err, errors.CodeInvalidParameter, "invalid configuration")
	}

	// 更新配置
	if config != nil {
		nr.config = config
	}

	// 测试LLM连接
	if err := nr.testLLMConnection(ctx); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to connect to LLM")
	}

	// 启动健康检查
	if nr.config.EnableHealthCheck {
		nr.wg.Add(1)
		go nr.healthCheckLoop()
	}

	nr.status = runtime.RuntimeStatusReady
	nr.logger.Info("native runtime initialized",
		logging.String("runtime_id", nr.id),
		logging.String("version", nr.version))

	return nil
}

// Execute 执行任务
func (nr *NativeRuntime) Execute(ctx context.Context, req *runtime.ExecuteRequest) (*runtime.ExecuteResponse, error) {
	if !nr.IsReady() {
		return nil, errors.NewInternalError("ERR_INTERNAL", "runtime not ready")
	}

	// 更新指标
	nr.metrics.mu.Lock()
	nr.metrics.totalRequests++
	nr.metrics.activeRequests++
	nr.metrics.mu.Unlock()

	defer func() {
		nr.metrics.mu.Lock()
		nr.metrics.activeRequests--
		nr.metrics.mu.Unlock()
	}()

	startTime := time.Now()

	// 执行ReAct循环
	result, err := nr.executeReAct(ctx, req)

	duration := time.Since(startTime)

	// 更新指标
	nr.metrics.mu.Lock()
	if err != nil {
		nr.metrics.failedRequests++
	} else {
		nr.metrics.successfulRequests++
		nr.metrics.totalTokens += int64(result.TokensUsed)
		nr.metrics.totalCost += result.TotalCost
	}
	nr.metrics.totalResponseTime += duration
	nr.metrics.mu.Unlock()

	if err != nil {
		return nil, err
	}

	// 构建响应
	resp := &runtime.ExecuteResponse{
		Output:       result.FinalAnswer,
		TokensUsed:   result.TokensUsed,
		Duration:     duration,
		Metadata: map[string]interface{}{
			"steps": result.Steps,
			"cost":  result.TotalCost,
		},
		Error: result.Error,
	}

	return resp, nil
}

// ExecuteStream 流式执行任务
func (nr *NativeRuntime) ExecuteStream(ctx context.Context, req *runtime.ExecuteRequest) (<-chan *runtime.StreamChunk, <-chan error) {
	chunkChan := make(chan *runtime.StreamChunk, 10)
	errChan := make(chan error, 1)

	if !nr.config.EnableStreaming {
		errChan <- errors.NewValidationError(errors.CodeInvalidParameter, "streaming not enabled")
		close(chunkChan)
		close(errChan)
		return chunkChan, errChan
	}

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		// 执行ReAct循环并流式输出
		if err := nr.executeReActStream(ctx, req, chunkChan); err != nil {
			errChan <- err
		}
	}()

	return chunkChan, errChan
}

// HealthCheck 健康检查
func (nr *NativeRuntime) HealthCheck(ctx context.Context) (*runtime.HealthCheckResult, error) {
	startTime := time.Now()

	result := &runtime.HealthCheckResult{
		Status:    runtime.HealthStatusHealthy,
		Details:   make(map[string]interface{}),
		Timestamp: time.Now(),
		Checks:    make([]*runtime.ComponentHealth, 0),
	}

	// 检查LLM连接
	llmHealth := nr.checkLLMHealth(ctx)
	result.Checks = append(result.Checks, llmHealth)
	if llmHealth.Status != runtime.HealthStatusHealthy {
		result.Status = runtime.HealthStatusDegraded
	}

	// 检查工具管理器
	if nr.toolManager != nil {
		toolHealth := &runtime.ComponentHealth{
			Name:      "tool_manager",
			Status:    runtime.HealthStatusHealthy,
			Timestamp: time.Now(),
		}
		result.Checks = append(result.Checks, toolHealth)
	}

	// 检查记忆管理器
	if nr.memoryManager != nil {
		memoryHealth := &runtime.ComponentHealth{
			Name:      "memory_manager",
			Status:    runtime.HealthStatusHealthy,
			Timestamp: time.Now(),
		}
		result.Checks = append(result.Checks, memoryHealth)
	}

	result.ResponseTime = time.Since(startTime)

	// 添加指标信息
	nr.metrics.mu.RLock()
	result.Details["total_requests"] = nr.metrics.totalRequests
	result.Details["active_requests"] = nr.metrics.activeRequests
	result.Details["success_rate"] = nr.calculateSuccessRate()
	nr.metrics.mu.RUnlock()

	return result, nil
}

// Metadata 获取运行时元数据
func (nr *NativeRuntime) Metadata() *runtime.RuntimeMetadata {
	nr.mu.RLock()
	defer nr.mu.RUnlock()
	return nr.metadata
}

// Shutdown 关闭运行时
func (nr *NativeRuntime) Shutdown(ctx context.Context) error {
	nr.mu.Lock()
	defer nr.mu.Unlock()

	if nr.status == runtime.RuntimeStatusShutdown {
		return nil
	}

	nr.logger.Info("shutting down native runtime", logging.String("runtime_id", nr.id))

	// 发送关闭信号
	close(nr.shutdownChan)

	// 等待所有goroutine完成
	done := make(chan struct{})
	go func() {
		nr.wg.Wait()
		close(done)
	}()

	// 等待关闭完成或超时
	select {
	case <-done:
		nr.logger.Info("native runtime shutdown completed", logging.String("runtime_id", nr.id))
	case <-ctx.Done():
		return errors.NewTimeoutError("DEADLINE_EXCEEDED", "shutdown timeout")
	}

	nr.status = runtime.RuntimeStatusShutdown
	return nil
}

// IsReady 检查是否就绪
func (nr *NativeRuntime) IsReady() bool {
	nr.mu.RLock()
	defer nr.mu.RUnlock()
	return nr.status == runtime.RuntimeStatusReady
}

// GetStatus 获取运行时状态
func (nr *NativeRuntime) GetStatus() runtime.RuntimeStatus {
	nr.mu.RLock()
	defer nr.mu.RUnlock()
	return nr.status
}

// UpdateConfig 更新配置
func (nr *NativeRuntime) UpdateConfig(ctx context.Context, config *runtime.RuntimeConfig) error {
	nr.mu.Lock()
	defer nr.mu.Unlock()

	if err := nr.validateConfig(config); err != nil {
		return errors.Wrap(err, errors.CodeInvalidParameter, "invalid configuration")
	}

	nr.config = config
	nr.metadata.UpdatedAt = time.Now()

	nr.logger.Info("native runtime configuration updated", logging.String("runtime_id", nr.id))
	return nil
}

// executeReAct 执行ReAct循环
func (nr *NativeRuntime) executeReAct(ctx context.Context, req *runtime.ExecuteRequest) (*ReActResult, error) {
	result := &ReActResult{
		Steps: make([]*ReActStep, 0),
	}

	// 准备系统提示词
	systemPrompt := nr.buildSystemPrompt()

	// 获取历史记忆
	var conversationHistory string
	if nr.memoryManager != nil {
		memories, err := nr.memoryManager.Retrieve(ctx, req.Input, 5)
		if err != nil {
			nr.logger.Warn("failed to retrieve memories", logging.String("error", err))
		} else {
			conversationHistory = nr.formatMemories(memories)
		}
	}

	// 构建初始提示
	userPrompt := fmt.Sprintf("%s\n\nTask: %s", conversationHistory, req.Input)

	totalTokens := 0
	totalCost := 0.0

	// ReAct循环
	for step := 1; step <= maxReActSteps; step++ {
		// 构建消息
// 		messages := []llm.Message{
// 			{Role: "system", Content: systemPrompt},
// 			{Role: "user", Content: userPrompt},
// 		}

// 		// 添加之前的步骤到上下文
// 		/*
// 		for _, s := range result.Steps {
// 			messages = append(messages,
// 				llm.Message{
// 					Role:    "assistant",
// 					Content: nr.formatStepForContext(s),
// 				},
// 			)
// 		}
// 		*/

// 		// 调用LLM
// 		llmResp, err := nr.llmClient.Chat(ctx, &llm.ChatRequest{
// 			Messages:    messages,
// 			MaxTokens:   1000,
// 			Temperature: 0.7,
// 		})
		if err != nil {
			return nil, errors.Wrap(err, "ERR_INTERNAL", "llm call failed")
		}

		totalTokens += llmResp.Usage.TotalTokens
		totalCost += nr.estimateCost(llmResp.Usage.TotalTokens)

		// 解析响应
		response := llmResp.Choices[0].Message.Content
		reactStep, isFinal, err := nr.parseReActResponse(response)
		if err != nil {
			return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to parse response")
		}

		reactStep.StepNumber = step
		reactStep.Timestamp = time.Now()
		result.Steps = append(result.Steps, reactStep)

		// 检查是否得到最终答案
		if isFinal {
			result.FinalAnswer = strings.TrimPrefix(response, "Final Answer:")
			result.FinalAnswer = strings.TrimSpace(result.FinalAnswer)
			break
		}

		// 执行工具
		if reactStep.Action != "" && nr.toolManager != nil {
			observation, err := nr.toolManager.ExecuteTool(ctx, reactStep.Action, reactStep.ActionInput)
			if err != nil {
				reactStep.Observation = fmt.Sprintf("Error: %v", err)
			} else {
				observationStr, _ := json.Marshal(observation)
				reactStep.Observation = string(observationStr)
			}

			// 更新用户提示，包含观察结果
			userPrompt = fmt.Sprintf("Observation: %s", reactStep.Observation)
		}

		// 检查是否达到最大步骤数
		if step >= maxReActSteps {
			return nil, errors.NewTimeoutError("DEADLINE_EXCEEDED", "max steps exceeded")
		}
	}

	result.TokensUsed = totalTokens
	result.TotalCost = totalCost

	// 存储记忆
	if nr.memoryManager != nil {
		memory := &Memory{
			ID:        fmt.Sprintf("mem_%d", time.Now().UnixNano()),
			SessionID: req.Metadata["session_id"].(string),
			Type:      MemoryTypeConversation,
			Content:   fmt.Sprintf("Q: %s\nA: %s", req.Input, result.FinalAnswer),
			Timestamp: time.Now(),
		}
		if err := nr.memoryManager.Store(ctx, memory); err != nil {
			nr.logger.Warn("failed to store memory", logging.String("error", err))
		}
	}

	return result, nil
}

// executeReActStream 流式执行ReAct循环
func (nr *NativeRuntime) executeReActStream(ctx context.Context, req *runtime.ExecuteRequest, chunkChan chan<- *runtime.StreamChunk) error {
	// 简化实现：执行完整ReAct后流式输出最终答案
	result, err := nr.executeReAct(ctx, req)
	if err != nil {
		return err
	}

	// 分块发送最终答案
	words := strings.Fields(result.FinalAnswer)
	for i, word := range words {
		chunk := &runtime.StreamChunk{
			Content:   strings.Join(words[:i+1], " "),
			Delta:     word + " ",
			Finished:  i == len(words)-1,
			Timestamp: time.Now(),
		}

		select {
		case chunkChan <- chunk:
		case <-ctx.Done():
			return ctx.Err()
		}

		time.Sleep(50 * time.Millisecond)
	}

	return nil
}

// buildSystemPrompt 构建系统提示词
func (nr *NativeRuntime) buildSystemPrompt() string {
	prompt := reactSystemPrompt

	// 添加可用工具列表
	if nr.toolManager != nil {
		tools := nr.toolManager.ListTools()
		toolDescriptions := make([]string, 0, len(tools))
		for _, tool := range tools {
			toolDescriptions = append(toolDescriptions,
				fmt.Sprintf("- %s: %s", tool.Name(), tool.Description()))
		}
		prompt = strings.Replace(prompt, "{{TOOLS}}", strings.Join(toolDescriptions, "\n"), 1)
	} else {
		prompt = strings.Replace(prompt, "{{TOOLS}}", "No tools available", 1)
	}

	return prompt
}

// parseReActResponse 解析ReAct响应
func (nr *NativeRuntime) parseReActResponse(response string) (*ReActStep, bool, error) {
	step := &ReActStep{}

	// 检查是否是最终答案
	if strings.Contains(response, "Final Answer:") {
		return step, true, nil
	}

	// 解析Thought
	if thoughtIdx := strings.Index(response, "Thought:"); thoughtIdx != -1 {
		thoughtEnd := strings.Index(response[thoughtIdx:], "\n")
		if thoughtEnd == -1 {
			thoughtEnd = len(response)
		} else {
			thoughtEnd += thoughtIdx
		}
		step.Thought = strings.TrimSpace(response[thoughtIdx+8 : thoughtEnd])
	}

	// 解析Action
	if actionIdx := strings.Index(response, "Action:"); actionIdx != -1 {
		actionEnd := strings.Index(response[actionIdx:], "\n")
		if actionEnd == -1 {
			actionEnd = len(response)
		} else {
			actionEnd += actionIdx
		}
		step.Action = strings.TrimSpace(response[actionIdx+7 : actionEnd])
	}

	// 解析Action Input
	if inputIdx := strings.Index(response, "Action Input:"); inputIdx != -1 {
		inputStart := inputIdx + 13
		inputEnd := len(response)

		// 尝试查找下一个标记或响应结束
		if nextIdx := strings.Index(response[inputStart:], "\n\n"); nextIdx != -1 {
			inputEnd = inputStart + nextIdx
		}

		inputStr := strings.TrimSpace(response[inputStart:inputEnd])

		// 尝试解析JSON
		var input map[string]interface{}
		if err := json.Unmarshal([]byte(inputStr), &input); err != nil {
			nr.logger.Warn("failed to parse action input as JSON", logging.String("input", inputStr), logging.String("error", err))
			// 如果不是JSON，使用原始字符串
			step.ActionInput = map[string]interface{}{"input": inputStr}
		} else {
			step.ActionInput = input
		}
	}

	return step, false, nil
}

// formatStepForContext 格式化步骤用于上下文
func (nr *NativeRuntime) formatStepForContext(step *ReActStep) string {
	var builder strings.Builder

	if step.Thought != "" {
		builder.WriteString(fmt.Sprintf("Thought: %s\n", step.Thought))
	}
	if step.Action != "" {
		builder.WriteString(fmt.Sprintf("Action: %s\n", step.Action))
		if step.ActionInput != nil {
			inputJSON, _ := json.Marshal(step.ActionInput)
			builder.WriteString(fmt.Sprintf("Action Input: %s\n", string(inputJSON)))
		}
	}
	if step.Observation != "" {
		builder.WriteString(fmt.Sprintf("Observation: %s\n", step.Observation))
	}

	return builder.String()
}

// formatMemories 格式化记忆
func (nr *NativeRuntime) formatMemories(memories []*Memory) string {
	if len(memories) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("Previous conversation:\n")
	for _, mem := range memories {
		builder.WriteString(fmt.Sprintf("- %s\n", mem.Content))
	}

	return builder.String()
}

// validateConfig 验证配置
func (nr *NativeRuntime) validateConfig(config *runtime.RuntimeConfig) error {
	if config == nil {
		return errors.NewValidationError(errors.CodeInvalidParameter, "config cannot be nil")
	}

	if config.ID == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "runtime id cannot be empty")
	}

	if config.Type != runtime.RuntimeTypeNative {
		return errors.NewValidationError(errors.CodeInvalidParameter, "invalid runtime type")
	}

	return nil
}

// testLLMConnection 测试LLM连接
func (nr *NativeRuntime) testLLMConnection(ctx context.Context) error {
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := nr.llmClient.Chat(testCtx, &llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: "test"},
		},
		MaxTokens: 10,
	})

	return err
}

// checkLLMHealth 检查LLM健康状态
func (nr *NativeRuntime) checkLLMHealth(ctx context.Context) *runtime.ComponentHealth {
	health := &runtime.ComponentHealth{
		Name:      "llm_client",
		Timestamp: time.Now(),
	}

	if err := nr.testLLMConnection(ctx); err != nil {
		health.Status = runtime.HealthStatusUnhealthy
		health.Message = fmt.Sprintf("LLM connection failed: %v", err)
	} else {
		health.Status = runtime.HealthStatusHealthy
		health.Message = "LLM connection healthy"
	}

	return health
}

// healthCheckLoop 健康检查循环
func (nr *NativeRuntime) healthCheckLoop() {
	defer nr.wg.Done()

	ticker := time.NewTicker(nr.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, err := nr.HealthCheck(ctx)
			cancel()

			if err != nil {
				nr.logger.Warn("health check failed", logging.String("error", err))
			}
		case <-nr.shutdownChan:
			return
		}
	}
}

// calculateSuccessRate 计算成功率
func (nr *NativeRuntime) calculateSuccessRate() float64 {
	if nr.metrics.totalRequests == 0 {
		return 0.0
	}
	return float64(nr.metrics.successfulRequests) / float64(nr.metrics.totalRequests)
}

// estimateCost 估算成本
func (nr *NativeRuntime) estimateCost(tokens int) float64 {
	// 简化计算：假设每1K令牌$0.002
	return float64(tokens) / 1000.0 * 0.002
}

//Personal.AI order the ending
