package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/openeeap/openeeap/internal/domain/agent"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/internal/platform/runtime"
	"github.com/openeeap/openeeap/pkg/errors"
)

// Executor 执行器接口
type Executor interface {
	// Execute 执行任务
	Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error)

	// ExecuteAsync 异步执行任务
	ExecuteAsync(ctx context.Context, req *ExecuteRequest) (<-chan *ExecuteResponse, <-chan error)

	// ExecuteStream 流式执行任务
	ExecuteStream(ctx context.Context, req *ExecuteRequest) (<-chan *StreamChunk, <-chan error)

	// GetExecution 获取执行状态
	GetExecution(ctx context.Context, executionID string) (*ExecutionInfo, error)

	// CancelExecution 取消执行
	CancelExecution(ctx context.Context, executionID string) error

	// GetMetrics 获取执行器指标
	GetMetrics(ctx context.Context) (*ExecutorMetrics, error)
}

// executor 执行器实现
type executor struct {
	runtimeManager runtime.Manager
	llmClient      llm.LLMClient
	tracer         trace.Tracer
	config         *ExecutorConfig
	logger         logger.Logger

	executions     map[string]*executionState
	executionsMu   sync.RWMutex

	metrics        *executorMetrics

	closed         bool
	closeChan      chan struct{}
	wg             sync.WaitGroup
}

// ExecutorConfig 执行器配置
type ExecutorConfig struct {
	DefaultTimeout         time.Duration // 默认超时时间
	MaxConcurrentExecutions int          // 最大并发执行数
	EnableTracing          bool          // 启用追踪
	EnableMetrics          bool          // 启用指标
	EnableStreaming        bool          // 启用流式输出
	EnableErrorRecovery    bool          // 启用错误恢复
	EnableResultCaching    bool          // 启用结果缓存
	ExecutionRetention     time.Duration // 执行记录保留时间
	StreamBufferSize       int           // 流缓冲区大小
	MetricsInterval        time.Duration // 指标收集间隔
}

// ExecuteRequest 执行请求
type ExecuteRequest struct {
	Task           *Task                  // 任务
	Context        map[string]interface{} // 上下文
	Options        *ExecuteOptions        // 执行选项
}

// ExecuteOptions 执行选项
type ExecuteOptions struct {
	EnableStreaming   bool                   // 启用流式
	EnableCache       bool                   // 启用缓存
	EnablePlugins     bool                   // 启用插件
	Timeout           time.Duration          // 超时时间
	MaxTokens         int                    // 最大令牌数
	Temperature       float64                // 温度
	TopP              float64                // Top-P
	Plugins           []string               // 插件列表
	ModelPreferences  *ModelPreferences      // 模型偏好
	Metadata          map[string]interface{} // 元数据
}

// ExecuteResponse 执行响应
type ExecuteResponse struct {
	ExecutionID    string                 // 执行ID
	Output         string                 // 输出
	Model          string                 // 模型
	TokensUsed     int                    // 使用的令牌数
	Cost           float64                // 成本
	Duration       time.Duration          // 执行时长
	Metadata       map[string]interface{} // 元数据
	Trace          *ExecutionTrace        // 执行追踪
	Error          error                  // 错误
}

// StreamChunk 流式数据块
type StreamChunk struct {
	ExecutionID    string                 // 执行ID
	Content        string                 // 内容
	Delta          string                 // 增量内容
	TokensUsed     int                    // 已使用令牌数
	Finished       bool                   // 是否完成
	Metadata       map[string]interface{} // 元数据
	Timestamp      time.Time              // 时间戳
}

// ExecutionInfo 执行信息
type ExecutionInfo struct {
	ExecutionID    string                 // 执行ID
	TaskID         string                 // 任务ID
	Status         ExecutionStatus        // 状态
	StartTime      time.Time              // 开始时间
	EndTime        time.Time              // 结束时间
	Duration       time.Duration          // 执行时长
	Runtime        string                 // 运行时
	Model          string                 // 模型
	TokensUsed     int                    // 使用的令牌数
	Cost           float64                // 成本
	Error          error                  // 错误
	Metadata       map[string]interface{} // 元数据
}

// ExecutionStatus 执行状态
type ExecutionStatus string

const (
	ExecutionStatusPending    ExecutionStatus = "pending"    // 待处理
	ExecutionStatusRunning    ExecutionStatus = "running"    // 运行中
	ExecutionStatusCompleted  ExecutionStatus = "completed"  // 已完成
	ExecutionStatusFailed     ExecutionStatus = "failed"     // 失败
	ExecutionStatusCancelled  ExecutionStatus = "cancelled"  // 已取消
	ExecutionStatusTimeout    ExecutionStatus = "timeout"    // 超时
)

// ExecutionTrace 执行追踪
type ExecutionTrace struct {
	TraceID        string                 // 追踪ID
	SpanID         string                 // Span ID
	ParentSpanID   string                 // 父Span ID
	StartTime      time.Time              // 开始时间
	EndTime        time.Time              // 结束时间
	Duration       time.Duration          // 时长
	Steps          []*TraceStep           // 步骤列表
	Metadata       map[string]interface{} // 元数据
}

// TraceStep 追踪步骤
type TraceStep struct {
	Name           string                 // 步骤名称
	Type           string                 // 步骤类型
	StartTime      time.Time              // 开始时间
	EndTime        time.Time              // 结束时间
	Duration       time.Duration          // 时长
	Input          interface{}            // 输入
	Output         interface{}            // 输出
	Error          error                  // 错误
	Metadata       map[string]interface{} // 元数据
}

// ExecutorMetrics 执行器指标
type ExecutorMetrics struct {
	TotalExecutions       int64         // 总执行数
	SuccessfulExecutions  int64         // 成功执行数
	FailedExecutions      int64         // 失败执行数
	CancelledExecutions   int64         // 取消执行数
	TimeoutExecutions     int64         // 超时执行数
	RunningExecutions     int64         // 运行中执行数
	AverageExecutionTime  time.Duration // 平均执行时间
	AverageTokensUsed     int           // 平均令牌使用数
	TotalCost             float64       // 总成本
	ErrorRate             float64       // 错误率
	ThroughputPerSecond   float64       // 每秒吞吐量
}

// executorMetrics 内部指标
type executorMetrics struct {
	totalExecutions      int64
	successfulExecutions int64
	failedExecutions     int64
	cancelledExecutions  int64
	timeoutExecutions    int64
	totalExecTime        time.Duration
	totalTokensUsed      int64
	totalCost            float64
	mu                   sync.RWMutex
}

// executionState 执行状态
type executionState struct {
	id             string
	taskID         string
	status         ExecutionStatus
	runtime        string
	model          string
	startTime      time.Time
	endTime        time.Time
	duration       time.Duration
	tokensUsed     int
	cost           float64
	trace          *ExecutionTrace
	ctx            context.Context
	cancel         context.CancelFunc
	error          error
	metadata       map[string]interface{}
	mu             sync.RWMutex
}

// NewExecutor 创建执行器
func NewExecutor(
	runtimeManager runtime.Manager,
	llmClient llm.LLMClient,
	tracer trace.Tracer,
	config *ExecutorConfig,
	logger logger.Logger,
) (Executor, error) {
	if config == nil {
		config = &ExecutorConfig{
			DefaultTimeout:          5 * time.Minute,
			MaxConcurrentExecutions: 100,
			EnableTracing:           true,
			EnableMetrics:           true,
			EnableStreaming:         true,
			EnableErrorRecovery:     true,
			EnableResultCaching:     true,
			ExecutionRetention:      1 * time.Hour,
			StreamBufferSize:        100,
			MetricsInterval:         1 * time.Minute,
		}
	}

	e := &executor{
		runtimeManager: runtimeManager,
		llmClient:      llmClient,
		tracer:         tracer,
		config:         config,
		logger:         logger,
		executions:     make(map[string]*executionState),
		metrics:        &executorMetrics{},
		closeChan:      make(chan struct{}),
	}

	// 启动清理循环
	if config.ExecutionRetention > 0 {
		e.wg.Add(1)
		go e.cleanupLoop()
	}

	// 启动指标收集循环
	if config.EnableMetrics && config.MetricsInterval > 0 {
		e.wg.Add(1)
		go e.metricsLoop()
	}

	return e, nil
}

// Execute 执行任务
func (e *executor) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	if e.closed {
		return nil, errors.New("ERR_INTERNAL", "executor is closed")
	}
	if req == nil || req.Task == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "execute request or task cannot be nil")
	}

	// 检查并发限制
	if e.config.MaxConcurrentExecutions > 0 {
		runningCount := e.countRunningExecutions()
		if runningCount >= e.config.MaxConcurrentExecutions {
			return nil, errors.New(errors.CodeResourceExhausted, "max concurrent executions reached")
		}
	}

	// 创建执行状态
	executionID := e.generateExecutionID()
	execState := e.createExecutionState(ctx, executionID, req)

	// 注册执行
	e.registerExecution(execState)
	defer e.unregisterExecution(executionID)

	// 开始追踪
	var span trace.Span
	if e.config.EnableTracing && e.tracer != nil {
		var traceCtx context.Context
		traceCtx, span = e.tracer.Start(ctx, "executor.Execute",
			trace.WithAttributes(
				trace.String("execution_id", executionID),
				trace.String("task_id", req.Task.ID),
			))
		ctx = traceCtx
		defer span.End()
	}

	// 更新指标
	e.metrics.mu.Lock()
	e.metrics.totalExecutions++
	e.metrics.mu.Unlock()

	// 执行任务
	startTime := time.Now()
	execState.startTime = startTime
	execState.status = ExecutionStatusRunning

	// 获取超时设置
	timeout := e.config.DefaultTimeout
	if req.Options != nil && req.Options.Timeout > 0 {
		timeout = req.Options.Timeout
	}

	// 创建超时上下文
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	execState.ctx = execCtx
	execState.cancel = cancel

	// 记录步骤：准备
	e.recordTraceStep(execState, "prepare", "preparation", nil, nil, nil)

	// 选择运行时并执行
	var output string
	var tokensUsed int
	var cost float64
	var execErr error

	switch req.Task.Route.Runtime.Type {
	case RuntimeTypeNative:
		output, tokensUsed, cost, execErr = e.executeNative(execCtx, req, execState)
	case RuntimeTypeLangChain:
		output, tokensUsed, cost, execErr = e.executeLangChain(execCtx, req, execState)
	case RuntimeTypeAutoGPT:
		output, tokensUsed, cost, execErr = e.executeAutoGPT(execCtx, req, execState)
	default:
		execErr = errors.New(errors.CodeInvalidParameter,
			fmt.Sprintf("unsupported runtime type: %s", req.Task.Route.Runtime.Type))
	}

	// 记录结果
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	execState.mu.Lock()
	execState.endTime = endTime
	execState.duration = duration
	execState.tokensUsed = tokensUsed
	execState.cost = cost
	execState.error = execErr
	execState.mu.Unlock()

	// 更新状态
	if execErr != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			execState.status = ExecutionStatusTimeout
			e.metrics.mu.Lock()
			e.metrics.timeoutExecutions++
			e.metrics.mu.Unlock()
		} else if execCtx.Err() == context.Canceled {
			execState.status = ExecutionStatusCancelled
			e.metrics.mu.Lock()
			e.metrics.cancelledExecutions++
			e.metrics.mu.Unlock()
		} else {
			execState.status = ExecutionStatusFailed
			e.metrics.mu.Lock()
			e.metrics.failedExecutions++
			e.metrics.mu.Unlock()
		}
	} else {
		execState.status = ExecutionStatusCompleted
		e.metrics.mu.Lock()
		e.metrics.successfulExecutions++
		e.metrics.totalExecTime += duration
		e.metrics.totalTokensUsed += int64(tokensUsed)
		e.metrics.totalCost += cost
		e.metrics.mu.Unlock()
	}

	// 记录步骤：完成
	e.recordTraceStep(execState, "complete", "completion", nil, output, execErr)

	// 记录追踪
	if span != nil {
		span.SetAttributes(
			trace.String("status", string(execState.status)),
			trace.Int("tokens_used", tokensUsed),
			trace.Float64("cost", cost),
		)
		if execErr != nil {
			span.RecordError(execErr)
		}
	}

	e.logger.Info("task executed",
		"execution_id", executionID,
		"task_id", req.Task.ID,
		"status", execState.status,
		"duration", duration,
		"tokens_used", tokensUsed,
		"cost", cost)

	// 构建响应
	resp := &ExecuteResponse{
		ExecutionID: executionID,
		Output:      output,
		Model:       execState.model,
		TokensUsed:  tokensUsed,
		Cost:        cost,
		Duration:    duration,
		Metadata:    execState.metadata,
		Trace:       execState.trace,
		Error:       execErr,
	}

	return resp, execErr
}

// ExecuteAsync 异步执行任务
func (e *executor) ExecuteAsync(ctx context.Context, req *ExecuteRequest) (<-chan *ExecuteResponse, <-chan error) {
	respChan := make(chan *ExecuteResponse, 1)
	errChan := make(chan error, 1)

	go func() {
		defer close(respChan)
		defer close(errChan)

		resp, err := e.Execute(ctx, req)
		if err != nil {
			errChan <- err
			return
		}
		respChan <- resp
	}()

	return respChan, errChan
}

// ExecuteStream 流式执行任务
func (e *executor) ExecuteStream(ctx context.Context, req *ExecuteRequest) (<-chan *StreamChunk, <-chan error) {
	chunkChan := make(chan *StreamChunk, e.config.StreamBufferSize)
	errChan := make(chan error, 1)

	if !e.config.EnableStreaming {
		errChan <- errors.New(errors.CodeInvalidParameter, "streaming not enabled")
		close(chunkChan)
		close(errChan)
		return chunkChan, errChan
	}

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		executionID := e.generateExecutionID()
		execState := e.createExecutionState(ctx, executionID, req)
		e.registerExecution(execState)
		defer e.unregisterExecution(executionID)

		execState.status = ExecutionStatusRunning
		execState.startTime = time.Now()

		// 使用LLM流式API
		if e.llmClient != nil {
			streamReq := &llm.CompletionRequest{
				Prompt:      req.Task.Request.Input,
				MaxTokens:   2000,
				Temperature: 0.7,
				Stream:      true,
			}

			if req.Options != nil {
				if req.Options.MaxTokens > 0 {
					streamReq.MaxTokens = req.Options.MaxTokens
				}
				if req.Options.Temperature > 0 {
					streamReq.Temperature = req.Options.Temperature
				}
			}

			fullContent := ""
			totalTokens := 0

			streamChan, streamErr := e.llmClient.Stream(ctx, streamReq)
			if streamErr != nil {
				errChan <- streamErr
				return
			}

			for chunk := range streamChan {
				if chunk.Error != nil {
					errChan <- chunk.Error
					break
				}

				fullContent += chunk.Delta
				totalTokens += chunk.TokensUsed

				chunkChan <- &StreamChunk{
					ExecutionID: executionID,
					Content:     fullContent,
					Delta:       chunk.Delta,
					TokensUsed:  totalTokens,
					Finished:    chunk.Finished,
					Metadata:    chunk.Metadata,
					Timestamp:   time.Now(),
				}

				if chunk.Finished {
					break
				}
			}

			execState.status = ExecutionStatusCompleted
			execState.endTime = time.Now()
			execState.duration = execState.endTime.Sub(execState.startTime)
			execState.tokensUsed = totalTokens
		}
	}()

	return chunkChan, errChan
}

// GetExecution 获取执行状态
func (e *executor) GetExecution(ctx context.Context, executionID string) (*ExecutionInfo, error) {
	if executionID == "" {
		return nil, errors.New(errors.CodeInvalidParameter, "execution id cannot be empty")
	}

	e.executionsMu.RLock()
	execState, exists := e.executions[executionID]
	e.executionsMu.RUnlock()

	if !exists {
		return nil, errors.New(errors.CodeNotFound, "execution not found")
	}

	execState.mu.RLock()
	defer execState.mu.RUnlock()

	return &ExecutionInfo{
		ExecutionID: execState.id,
		TaskID:      execState.taskID,
		Status:      execState.status,
		StartTime:   execState.startTime,
		EndTime:     execState.endTime,
		Duration:    execState.duration,
		Runtime:     execState.runtime,
		Model:       execState.model,
		TokensUsed:  execState.tokensUsed,
		Cost:        execState.cost,
		Error:       execState.error,
		Metadata:    execState.metadata,
	}, nil
}

// CancelExecution 取消执行
func (e *executor) CancelExecution(ctx context.Context, executionID string) error {
	if executionID == "" {
		return errors.New(errors.CodeInvalidParameter, "execution id cannot be empty")
	}

	e.executionsMu.RLock()
	execState, exists := e.executions[executionID]
	e.executionsMu.RUnlock()

	if !exists {
		return errors.New(errors.CodeNotFound, "execution not found")
	}

	execState.mu.Lock()
	if execState.cancel != nil {
		execState.cancel()
	}
	execState.status = ExecutionStatusCancelled
	execState.endTime = time.Now()
	execState.duration = execState.endTime.Sub(execState.startTime)
	execState.mu.Unlock()

	e.metrics.mu.Lock()
	e.metrics.cancelledExecutions++
	e.metrics.mu.Unlock()

	e.logger.Info("execution cancelled", "execution_id", executionID)
	return nil
}

// GetMetrics 获取执行器指标
func (e *executor) GetMetrics(ctx context.Context) (*ExecutorMetrics, error) {
	e.metrics.mu.RLock()
	defer e.metrics.mu.RUnlock()

	runningCount := e.countRunningExecutions()

	avgExecTime := time.Duration(0)
	if e.metrics.successfulExecutions > 0 {
		avgExecTime = time.Duration(int64(e.metrics.totalExecTime) / e.metrics.successfulExecutions)
	}

	avgTokens := 0
	if e.metrics.successfulExecutions > 0 {
		avgTokens = int(e.metrics.totalTokensUsed / e.metrics.successfulExecutions)
	}

	errorRate := 0.0
	if e.metrics.totalExecutions > 0 {
		errorRate = float64(e.metrics.failedExecutions) / float64(e.metrics.totalExecutions)
	}

	return &ExecutorMetrics{
		TotalExecutions:      e.metrics.totalExecutions,
		SuccessfulExecutions: e.metrics.successfulExecutions,
		FailedExecutions:     e.metrics.failedExecutions,
		CancelledExecutions:  e.metrics.cancelledExecutions,
		TimeoutExecutions:    e.metrics.timeoutExecutions,
		RunningExecutions:    int64(runningCount),
		AverageExecutionTime: avgExecTime,
		AverageTokensUsed:    avgTokens,
		TotalCost:            e.metrics.totalCost,
		ErrorRate:            errorRate,
	}, nil
}

// executeNative 执行原生任务
func (e *executor) executeNative(ctx context.Context, req *ExecuteRequest, execState *executionState) (string, int, float64, error) {
	e.recordTraceStep(execState, "execute_native", "execution", req.Task.Request.Input, nil, nil)

	// 调用LLM客户端
	if e.llmClient == nil {
		return "", 0, 0, errors.New("ERR_INTERNAL", "llm client not configured")
	}

	llmReq := &llm.CompletionRequest{
		Prompt:      req.Task.Request.Input,
		MaxTokens:   2000,
		Temperature: 0.7,
	}

	if req.Options != nil {
		if req.Options.MaxTokens > 0 {
			llmReq.MaxTokens = req.Options.MaxTokens
		}
		if req.Options.Temperature > 0 {
			llmReq.Temperature = req.Options.Temperature
		}
	}

	resp, err := e.llmClient.Complete(ctx, llmReq)
	if err != nil {
		e.recordTraceStep(execState, "execute_native", "execution", req.Task.Request.Input, nil, err)
		return "", 0, 0, errors.Wrap(err, "ERR_INTERNAL", "llm completion failed")
	}

	output := ""
	if len(resp.Choices) > 0 {
		output = resp.Choices[0].Text
	}

	tokensUsed := resp.Usage.TotalTokens
	cost := e.calculateCost(req.Task.Route.Model, tokensUsed)

	execState.model = resp.Model

	e.recordTraceStep(execState, "execute_native", "execution", req.Task.Request.Input, output, nil)
	return output, tokensUsed, cost, nil
}

// executeLangChain 执行LangChain任务
func (e *executor) executeLangChain(ctx context.Context, req *ExecuteRequest, execState *executionState) (string, int, float64, error) {
	e.recordTraceStep(execState, "execute_langchain", "execution", req.Task.Request.Input, nil, nil)

	// 调用LangChain运行时
	if e.runtimeManager == nil {
		return "", 0, 0, errors.New("ERR_INTERNAL", "runtime manager not configured")
	}

	runtimeReq := &runtime.ExecuteRequest{
		AgentID: req.Task.Request.AgentID,
		Input:   req.Task.Request.Input,
		Context: req.Context,
	}

	runtimeResp, err := e.runtimeManager.Execute(ctx, req.Task.Route.Runtime.ID, runtimeReq)
	if err != nil {
		e.recordTraceStep(execState, "execute_langchain", "execution", req.Task.Request.Input, nil, err)
		return "", 0, 0, errors.Wrap(err, "ERR_INTERNAL", "langchain execution failed")
	}

	tokensUsed := runtimeResp.TokensUsed
	cost := e.calculateCost(req.Task.Route.Model, tokensUsed)

	e.recordTraceStep(execState, "execute_langchain", "execution", req.Task.Request.Input, runtimeResp.Output, nil)
	return runtimeResp.Output, tokensUsed, cost, nil
}

// executeAutoGPT 执行AutoGPT任务
func (e *executor) executeAutoGPT(ctx context.Context, req *ExecuteRequest, execState *executionState) (string, int, float64, error) {
	e.recordTraceStep(execState, "execute_autogpt", "execution", req.Task.Request.Input, nil, nil)

	// 调用AutoGPT运行时
	if e.runtimeManager == nil {
		return "", 0, 0, errors.New("ERR_INTERNAL", "runtime manager not configured")
	}

	runtimeReq := &runtime.ExecuteRequest{
		AgentID: req.Task.Request.AgentID,
		Input:   req.Task.Request.Input,
		Context: req.Context,
	}

	runtimeResp, err := e.runtimeManager.Execute(ctx, req.Task.Route.Runtime.ID, runtimeReq)
	if err != nil {
		e.recordTraceStep(execState, "execute_autogpt", "execution", req.Task.Request.Input, nil, err)
		return "", 0, 0, errors.Wrap(err, "ERR_INTERNAL", "autogpt execution failed")
	}

	tokensUsed := runtimeResp.TokensUsed
	cost := e.calculateCost(req.Task.Route.Model, tokensUsed)

	e.recordTraceStep(execState, "execute_autogpt", "execution", req.Task.Request.Input, runtimeResp.Output, nil)
	return runtimeResp.Output, tokensUsed, cost, nil
}

// createExecutionState 创建执行状态
func (e *executor) createExecutionState(ctx context.Context, executionID string, req *ExecuteRequest) *executionState {
	execState := &executionState{
		id:       executionID,
		taskID:   req.Task.ID,
		status:   ExecutionStatusPending,
		metadata: make(map[string]interface{}),
	}

	if req.Task.Route != nil {
		if req.Task.Route.Runtime != nil {
			execState.runtime = string(req.Task.Route.Runtime.Type)
		}
		if req.Task.Route.Model != nil {
			execState.model = req.Task.Route.Model.Name
		}
	}

	if e.config.EnableTracing {
		execState.trace = &ExecutionTrace{
			TraceID:   executionID,
			Steps:     make([]*TraceStep, 0),
			Metadata:  make(map[string]interface{}),
		}
	}

	return execState
}

// registerExecution 注册执行
func (e *executor) registerExecution(execState *executionState) {
	e.executionsMu.Lock()
	defer e.executionsMu.Unlock()
	e.executions[execState.id] = execState
}

// unregisterExecution 注销执行
func (e *executor) unregisterExecution(executionID string) {
	// 不立即删除，由清理循环处理
}

// recordTraceStep 记录追踪步骤
func (e *executor) recordTraceStep(execState *executionState, name, stepType string, input, output interface{}, err error) {
	if !e.config.EnableTracing || execState.trace == nil {
		return
	}

	execState.mu.Lock()
	defer execState.mu.Unlock()

	step := &TraceStep{
		Name:      name,
		Type:      stepType,
		StartTime: time.Now(),
		Input:     input,
		Output:    output,
		Error:     err,
		Metadata:  make(map[string]interface{}),
	}

	execState.trace.Steps = append(execState.trace.Steps, step)
}

// calculateCost 计算成本
func (e *executor) calculateCost(model *ModelInfo, tokensUsed int) float64 {
	if model == nil || model.Pricing == nil {
		return 0.0
	}

	// 简化计算，假设输入和输出令牌各占一半
	inputTokens := tokensUsed / 2
	outputTokens := tokensUsed - inputTokens

	inputCost := float64(inputTokens) / 1000.0 * model.Pricing.InputCostPer1K
	outputCost := float64(outputTokens) / 1000.0 * model.Pricing.OutputCostPer1K

	return inputCost + outputCost
}

// countRunningExecutions 统计运行中的执行数
func (e *executor) countRunningExecutions() int {
	e.executionsMu.RLock()
	defer e.executionsMu.RUnlock()

	count := 0
	for _, exec := range e.executions {
		exec.mu.RLock()
		if exec.status == ExecutionStatusRunning {
			count++
		}
		exec.mu.RUnlock()
	}
	return count
}

// cleanupLoop 清理循环
func (e *executor) cleanupLoop() {
	defer e.wg.Done()

	ticker := time.NewTicker(e.config.ExecutionRetention / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.cleanup()
		case <-e.closeChan:
			return
		}
	}
}

// cleanup 清理过期执行记录
func (e *executor) cleanup() {
	now := time.Now()

	e.executionsMu.Lock()
	defer e.executionsMu.Unlock()

	for id, exec := range e.executions {
		exec.mu.RLock()
		shouldRemove := false

		// 只清理已完成的执行
		if exec.status == ExecutionStatusCompleted ||
			exec.status == ExecutionStatusFailed ||
			exec.status == ExecutionStatusCancelled ||
			exec.status == ExecutionStatusTimeout {
			if !exec.endTime.IsZero() && now.Sub(exec.endTime) > e.config.ExecutionRetention {
				shouldRemove = true
			}
		}
		exec.mu.RUnlock()

		if shouldRemove {
			delete(e.executions, id)
			e.logger.Debug("execution cleaned up", "execution_id", id)
		}
	}
}

// metricsLoop 指标收集循环
func (e *executor) metricsLoop() {
	defer e.wg.Done()

	ticker := time.NewTicker(e.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.collectMetrics()
		case <-e.closeChan:
			return
		}
	}
}

// collectMetrics 收集指标
func (e *executor) collectMetrics() {
	metrics, err := e.GetMetrics(context.Background())
	if err != nil {
		e.logger.Error("failed to collect metrics", "error", err)
		return
	}

	e.logger.Debug("executor metrics",
		"total_executions", metrics.TotalExecutions,
		"successful_executions", metrics.SuccessfulExecutions,
		"failed_executions", metrics.FailedExecutions,
		"running_executions", metrics.RunningExecutions,
		"avg_execution_time", metrics.AverageExecutionTime,
		"avg_tokens_used", metrics.AverageTokensUsed,
		"total_cost", metrics.TotalCost,
		"error_rate", metrics.ErrorRate)
}

// generateExecutionID 生成执行ID
func (e *executor) generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}

// Close 关闭执行器
func (e *executor) Close() error {
	if e.closed {
		return nil
	}

	e.closed = true
	close(e.closeChan)
	e.wg.Wait()

	// 取消所有运行中的执行
	e.executionsMu.RLock()
	executions := make([]*executionState, 0, len(e.executions))
	for _, exec := range e.executions {
		executions = append(executions, exec)
	}
	e.executionsMu.RUnlock()

	for _, exec := range executions {
		exec.mu.Lock()
		if exec.status == ExecutionStatusRunning && exec.cancel != nil {
			exec.cancel()
			exec.status = ExecutionStatusCancelled
		}
		exec.mu.Unlock()
	}

	e.logger.Info("executor closed")
	return nil
}

// ModelPreferences 模型偏好
type ModelPreferences struct {
	PreferredModels  []string              // 偏好的模型列表
	ExcludedModels   []string              // 排除的模型列表
	Requirements     *ModelRequirements    // 模型要求
	CostLimit        float64               // 成本限制
	LatencyLimit     time.Duration         // 延迟限制
	Metadata         map[string]interface{} // 元数据
}

// ModelRequirements 模型要求
type ModelRequirements struct {
	MinContextLength   int      // 最小上下文长度
	RequiredFeatures   []string // 必需特性
	MinQualityScore    float64  // 最小质量评分
	MaxCostPer1KTokens float64  // 最大每1K令牌成本
}

// Intent 意图
type Intent struct {
	Type        string                 // 意图类型
	Action      string                 // 动作
	Entities    map[string]interface{} // 实体
	Confidence  float64                // 置信度
	Context     map[string]interface{} // 上下文
	Metadata    map[string]interface{} // 元数据
}

// Helper functions for execution results serialization

// MarshalExecutionResult 序列化执行结果
func MarshalExecutionResult(result interface{}) ([]byte, error) {
	return json.Marshal(result)
}

// UnmarshalExecutionResult 反序列化执行结果
func UnmarshalExecutionResult(data []byte, result interface{}) error {
	return json.Unmarshal(data, result)
}

// ExecutionContext 执行上下文
type ExecutionContext struct {
	ExecutionID     string                 // 执行ID
	TaskID          string                 // 任务ID
	UserID          string                 // 用户ID
	SessionID       string                 // 会话ID
	Variables       map[string]interface{} // 变量
	State           map[string]interface{} // 状态
	Metadata        map[string]interface{} // 元数据
	ParentExecID    string                 // 父执行ID
	ChildExecIDs    []string               // 子执行ID列表
	CreatedAt       time.Time              // 创建时间
	UpdatedAt       time.Time              // 更新时间
}

// NewExecutionContext 创建执行上下文
func NewExecutionContext(executionID, taskID, userID, sessionID string) *ExecutionContext {
	now := time.Now()
	return &ExecutionContext{
		ExecutionID: executionID,
		TaskID:      taskID,
		UserID:      userID,
		SessionID:   sessionID,
		Variables:   make(map[string]interface{}),
		State:       make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
		ChildExecIDs: make([]string, 0),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// SetVariable 设置变量
func (ec *ExecutionContext) SetVariable(key string, value interface{}) {
	ec.Variables[key] = value
	ec.UpdatedAt = time.Now()
}

// GetVariable 获取变量
func (ec *ExecutionContext) GetVariable(key string) (interface{}, bool) {
	value, exists := ec.Variables[key]
	return value, exists
}

// SetState 设置状态
func (ec *ExecutionContext) SetState(key string, value interface{}) {
	ec.State[key] = value
	ec.UpdatedAt = time.Now()
}

// GetState 获取状态
func (ec *ExecutionContext) GetState(key string) (interface{}, bool) {
	value, exists := ec.State[key]
	return value, exists
}

// AddChildExecution 添加子执行
func (ec *ExecutionContext) AddChildExecution(childExecID string) {
	ec.ChildExecIDs = append(ec.ChildExecIDs, childExecID)
	ec.UpdatedAt = time.Now()
}

// Clone 克隆执行上下文
func (ec *ExecutionContext) Clone() *ExecutionContext {
	cloned := &ExecutionContext{
		ExecutionID:  ec.ExecutionID,
		TaskID:       ec.TaskID,
		UserID:       ec.UserID,
		SessionID:    ec.SessionID,
		ParentExecID: ec.ParentExecID,
		CreatedAt:    ec.CreatedAt,
		UpdatedAt:    time.Now(),
		Variables:    make(map[string]interface{}),
		State:        make(map[string]interface{}),
		Metadata:     make(map[string]interface{}),
		ChildExecIDs: make([]string, len(ec.ChildExecIDs)),
	}

	// 深拷贝变量
	for k, v := range ec.Variables {
		cloned.Variables[k] = v
	}

	// 深拷贝状态
	for k, v := range ec.State {
		cloned.State[k] = v
	}

	// 深拷贝元数据
	for k, v := range ec.Metadata {
		cloned.Metadata[k] = v
	}

	// 拷贝子执行ID列表
	copy(cloned.ChildExecIDs, ec.ChildExecIDs)

	return cloned
}

// ExecutionError 执行错误
type ExecutionError struct {
	Code       string                 // 错误代码
	Message    string                 // 错误消息
	Type       string                 // 错误类型
	Recoverable bool                  // 是否可恢复
	Retryable  bool                   // 是否可重试
	Details    map[string]interface{} // 错误详情
	StackTrace string                 // 堆栈跟踪
	Timestamp  time.Time              // 时间戳
}

// NewExecutionError 创建执行错误
func NewExecutionError(code, message, errorType string) *ExecutionError {
	return &ExecutionError{
		Code:       code,
		Message:    message,
		Type:       errorType,
		Recoverable: false,
		Retryable:  false,
		Details:    make(map[string]interface{}),
		Timestamp:  time.Now(),
	}
}

// Error 实现error接口
func (e *ExecutionError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Type, e.Message)
}

// WithDetails 添加错误详情
func (e *ExecutionError) WithDetails(details map[string]interface{}) *ExecutionError {
	e.Details = details
	return e
}

// WithStackTrace 添加堆栈跟踪
func (e *ExecutionError) WithStackTrace(stackTrace string) *ExecutionError {
	e.StackTrace = stackTrace
	return e
}

// SetRecoverable 设置是否可恢复
func (e *ExecutionError) SetRecoverable(recoverable bool) *ExecutionError {
	e.Recoverable = recoverable
	return e
}

// SetRetryable 设置是否可重试
func (e *ExecutionError) SetRetryable(retryable bool) *ExecutionError {
	e.Retryable = retryable
	return e
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	Success      bool                   // 是否成功
	Output       interface{}            // 输出
	Error        *ExecutionError        // 错误
	TokensUsed   int                    // 使用的令牌数
	Cost         float64                // 成本
	Duration     time.Duration          // 执行时长
	Metadata     map[string]interface{} // 元数据
	Artifacts    []string               // 产物列表
	Checkpoints  []string               // 检查点列表
	Timestamp    time.Time              // 时间戳
}

// NewExecutionResult 创建执行结果
func NewExecutionResult(success bool) *ExecutionResult {
	return &ExecutionResult{
		Success:     success,
		Metadata:    make(map[string]interface{}),
		Artifacts:   make([]string, 0),
		Checkpoints: make([]string, 0),
		Timestamp:   time.Now(),
	}
}

// WithOutput 设置输出
func (r *ExecutionResult) WithOutput(output interface{}) *ExecutionResult {
	r.Output = output
	return r
}

// WithError 设置错误
func (r *ExecutionResult) WithError(err *ExecutionError) *ExecutionResult {
	r.Error = err
	r.Success = false
	return r
}

// WithTokensUsed 设置令牌使用数
func (r *ExecutionResult) WithTokensUsed(tokens int) *ExecutionResult {
	r.TokensUsed = tokens
	return r
}

// WithCost 设置成本
func (r *ExecutionResult) WithCost(cost float64) *ExecutionResult {
	r.Cost = cost
	return r
}

// WithDuration 设置执行时长
func (r *ExecutionResult) WithDuration(duration time.Duration) *ExecutionResult {
	r.Duration = duration
	return r
}

// AddArtifact 添加产物
func (r *ExecutionResult) AddArtifact(artifact string) *ExecutionResult {
	r.Artifacts = append(r.Artifacts, artifact)
	return r
}

// AddCheckpoint 添加检查点
func (r *ExecutionResult) AddCheckpoint(checkpoint string) *ExecutionResult {
	r.Checkpoints = append(r.Checkpoints, checkpoint)
	return r
}

// IsSuccess 检查是否成功
func (r *ExecutionResult) IsSuccess() bool {
	return r.Success && r.Error == nil
}

// IsRetryable 检查是否可重试
func (r *ExecutionResult) IsRetryable() bool {
	return r.Error != nil && r.Error.Retryable
}

// IsRecoverable 检查是否可恢复
func (r *ExecutionResult) IsRecoverable() bool {
	return r.Error != nil && r.Error.Recoverable
}

//Personal.AI order the ending
