package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/openeeap/openeeap/internal/domain/agent"
	"github.com/openeeap/openeeap/internal/infrastructure/repository/postgres"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/pkg/errors"
)

// Orchestrator 编排器接口
type Orchestrator interface {
	// Execute 执行请求
	Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error)

	// ExecuteStream 流式执行请求
	ExecuteStream(ctx context.Context, req *ExecuteRequest, stream ResponseStream) error

	// ExecuteBatch 批量执行请求
	ExecuteBatch(ctx context.Context, requests []*ExecuteRequest) ([]*ExecuteResponse, error)

	// ExecuteAsync 异步执行请求
	ExecuteAsync(ctx context.Context, req *ExecuteRequest, callback ExecuteCallback) error

	// ValidateRequest 验证请求
	ValidateRequest(ctx context.Context, req *ExecuteRequest) error

	// GetExecutionStatus 获取执行状态
	GetExecutionStatus(ctx context.Context, executionID string) (*ExecutionStatus, error)

	// CancelExecution 取消执行
	CancelExecution(ctx context.Context, executionID string) error

	// GetMetrics 获取指标
	GetMetrics(ctx context.Context) (*OrchestratorMetrics, error)

	// Close 关闭编排器
	Close() error
}

// orchestrator 编排器实现
type orchestrator struct {
	parser        Parser
	router        Router
	scheduler     Scheduler
	executor      Executor
	//pluginManager *plugin.PluginManager // TODO: implement plugin manager

	// TODO: implement repository interfaces
	//policyRepo    *postgres.PolicyRepository
	//taskRepo      *postgres.TaskRepository
	//executionRepo *postgres.ExecutionRepository

	config        *OrchestratorConfig
	logger        logging.Logger

	executions    map[string]*executionContext
	mu            sync.RWMutex

	metrics       *orchestratorMetrics
	interceptors  []Interceptor
	middlewares   []Middleware

	closed        bool
	closeChan     chan struct{}
	wg            sync.WaitGroup
}

// OrchestratorConfig 编排器配置
type OrchestratorConfig struct {
	MaxConcurrentExecutions int           // 最大并发执行数
	ExecutionTimeout        time.Duration // 执行超时时间
	StreamBufferSize        int           // 流缓冲区大小
	RetryPolicy             *RetryPolicy  // 重试策略
	EnableMetrics           bool          // 启用指标
	EnableTracing           bool          // 启用追踪
	EnableCaching           bool          // 启用缓存
	CacheExpiration         time.Duration // 缓存过期时间
	EnableRateLimiting      bool          // 启用限流
	RateLimitConfig         *RateLimitConfig // 限流配置
	EnableCircuitBreaker    bool          // 启用熔断器
	CircuitBreakerConfig    *CircuitBreakerConfig // 熔断器配置
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries    int           // 最大重试次数
	InitialDelay  time.Duration // 初始延迟
	MaxDelay      time.Duration // 最大延迟
	BackoffFactor float64       // 退避因子
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	RequestsPerSecond int           // 每秒请求数
	Burst             int           // 突发容量
	WaitTimeout       time.Duration // 等待超时
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	Threshold         int           // 阈值
	Timeout           time.Duration // 超时时间
	HalfOpenRequests  int           // 半开状态请求数
	SuccessThreshold  int           // 成功阈值
}

// ExecuteRequest 执行请求
type ExecuteRequest struct {
	RequestID       string                 // 请求ID
	UserID          string                 // 用户ID
	SessionID       string                 // 会话ID
	Input           string                 // 输入内容
	Context         map[string]interface{} // 上下文
	Options         *ExecuteOptions        // 执行选项
	Metadata        map[string]string      // 元数据
	Priority        int                    // 优先级
	Timeout         time.Duration          // 超时时间
	ParentRequestID string                 // 父请求ID
}

// ExecuteOptions 执行选项
type ExecuteOptions struct {
	Stream            bool                   // 是否流式
	EnablePlugins     bool                   // 启用插件
	EnableCache       bool                   // 启用缓存
	EnableRetry       bool                   // 启用重试
	MaxRetries        int                    // 最大重试次数
	Plugins           []string               // 指定插件列表
	ModelPreferences  *ModelPreferences      // 模型偏好
	OutputFormat      OutputFormat           // 输出格式
	Temperature       float64                // 温度参数
	MaxTokens         int                    // 最大令牌数
	CustomParams      map[string]interface{} // 自定义参数
}

// ModelPreferences 模型偏好
type ModelPreferences struct {
	PreferredModels []string          // 偏好模型列表
	ExcludedModels  []string          // 排除模型列表
	Requirements    *ModelRequirements // 模型要求
}

// ModelRequirements 模型要求
type ModelRequirements struct {
	MinContextLength int      // 最小上下文长度
	RequiredFeatures []string // 必需特性
	MaxLatency       time.Duration // 最大延迟
	MaxCost          float64  // 最大成本
}

// OutputFormat 输出格式
type OutputFormat string

const (
	OutputFormatText OutputFormat = "text" // 文本
	OutputFormatJSON OutputFormat = "json" // JSON
	OutputFormatXML  OutputFormat = "xml"  // XML
)

// ExecuteResponse 执行响应
type ExecuteResponse struct {
	ExecutionID     string                 // 执行ID
	RequestID       string                 // 请求ID
	Output          string                 // 输出内容
	Status          ExecutionStatusType    // 状态
	Error           error                  // 错误
	StartTime       time.Time              // 开始时间
	EndTime         time.Time              // 结束时间
	Duration        time.Duration          // 执行时长
	Model           string                 // 使用的模型
	TokensUsed      int                    // 使用的令牌数
	Cost            float64                // 成本
	Metadata        map[string]interface{} // 元数据
	PluginsExecuted []string               // 执行的插件列表
	CacheHit        bool                   // 是否缓存命中
	RetryCount      int                    // 重试次数
}

// ResponseStream 响应流接口
type ResponseStream interface {
	Send(*StreamChunk) error
	Close() error
}

// StreamChunk 流数据块
type StreamChunk struct {
	ExecutionID string                 // 执行ID
	Type        ChunkType              // 块类型
	Content     string                 // 内容
	Delta       string                 // 增量
	Metadata    map[string]interface{} // 元数据
	IsLast      bool                   // 是否最后一块
	Error       error                  // 错误
}

// ChunkType 块类型
type ChunkType string

const (
	ChunkTypeStart    ChunkType = "start"    // 开始
	ChunkTypeContent  ChunkType = "content"  // 内容
	ChunkTypeMetadata ChunkType = "metadata" // 元数据
	ChunkTypeEnd      ChunkType = "end"      // 结束
	ChunkTypeError    ChunkType = "error"    // 错误
)

// ExecuteCallback 执行回调
type ExecuteCallback func(response *ExecuteResponse, err error)

// ExecutionStatus 执行状态
type ExecutionStatus struct {
	ExecutionID     string              // 执行ID
	RequestID       string              // 请求ID
	Status          ExecutionStatusType // 状态
	Progress        float64             // 进度（0-1）
	StartTime       time.Time           // 开始时间
	CurrentStage    string              // 当前阶段
	Message         string              // 消息
	Metadata        map[string]interface{} // 元数据
}

// ExecutionStatusType 执行状态类型
type ExecutionStatusType string

const (
	ExecutionStatusPending   ExecutionStatusType = "pending"   // 待处理
	ExecutionStatusRunning   ExecutionStatusType = "running"   // 运行中
	ExecutionStatusCompleted ExecutionStatusType = "completed" // 已完成
	ExecutionStatusFailed    ExecutionStatusType = "failed"    // 失败
	ExecutionStatusCancelled ExecutionStatusType = "cancelled" // 已取消
	ExecutionStatusTimeout   ExecutionStatusType = "timeout"   // 超时
)

// executionContext 执行上下文
type executionContext struct {
	req         *ExecuteRequest
	resp        *ExecuteResponse
	ctx         context.Context
	cancel      context.CancelFunc
	status      ExecutionStatusType
	stage       string
	startTime   time.Time
	pluginData  map[string]interface{}
	mu          sync.RWMutex
}

// OrchestratorMetrics 编排器指标
type OrchestratorMetrics struct {
	TotalExecutions     int64         // 总执行次数
	SuccessfulExecutions int64        // 成功执行次数
	FailedExecutions    int64         // 失败执行次数
	CancelledExecutions int64         // 取消执行次数
	AverageDuration     time.Duration // 平均执行时长
	CacheHitRate        float64       // 缓存命中率
	ActiveExecutions    int           // 活跃执行数
	QueuedExecutions    int           // 排队执行数
	PluginMetrics       map[string]*PluginMetrics // 插件指标
}

// PluginMetrics 插件指标
type PluginMetrics struct {
	ExecutionCount  int64         // 执行次数
	SuccessCount    int64         // 成功次数
	FailureCount    int64         // 失败次数
	AverageDuration time.Duration // 平均执行时长
	LastExecutedAt  time.Time     // 最后执行时间
}

// orchestratorMetrics 内部指标
type orchestratorMetrics struct {
	totalExecutions      int64
	successfulExecutions int64
	failedExecutions     int64
	cancelledExecutions  int64
	totalDuration        time.Duration
	cacheHits            int64
	cacheMisses          int64
	pluginMetrics        map[string]*PluginMetrics
	mu                   sync.RWMutex
}

// Interceptor 拦截器接口
type Interceptor interface {
	Before(ctx context.Context, req *ExecuteRequest) error
	After(ctx context.Context, req *ExecuteRequest, resp *ExecuteResponse) error
	OnError(ctx context.Context, req *ExecuteRequest, err error) error
}

// Middleware 中间件接口
type Middleware func(next ExecuteFunc) ExecuteFunc

// ExecuteFunc 执行函数类型
type ExecuteFunc func(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error)

// NewOrchestrator 创建编排器
func NewOrchestrator(
	parser parser.Parser,
	router router.Router,
	scheduler scheduler.Scheduler,
	executor executor.Executor,
	pluginManager plugin.PluginManager,
	policyRepo repository.PolicyRepository,
	taskRepo repository.TaskRepository,
	executionRepo repository.ExecutionRepository,
	config *OrchestratorConfig,
	logger logger.Logger,
) (Orchestrator, error) {
	if parser == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "parser cannot be nil")
	}
	if router == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "router cannot be nil")
	}
	if scheduler == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "scheduler cannot be nil")
	}
	if executor == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "executor cannot be nil")
	}
	if config == nil {
		config = &OrchestratorConfig{
			MaxConcurrentExecutions: 100,
			ExecutionTimeout:        5 * time.Minute,
			StreamBufferSize:        1000,
			EnableMetrics:           true,
		}
	}

	o := &orchestrator{
		parser:        parser,
		router:        router,
		scheduler:     scheduler,
		executor:      executor,
		pluginManager: pluginManager,
		policyRepo:    policyRepo,
		taskRepo:      taskRepo,
		executionRepo: executionRepo,
		config:        config,
		logger:        logger,
		executions:    make(map[string]*executionContext),
		metrics: &orchestratorMetrics{
			pluginMetrics: make(map[string]*PluginMetrics),
		},
		interceptors: make([]Interceptor, 0),
		middlewares:  make([]Middleware, 0),
		closeChan:    make(chan struct{}),
	}

	// 启动后台清理任务
	o.wg.Add(1)
	go o.cleanupLoop()

	return o, nil
}

// Execute 执行请求
func (o *orchestrator) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	if o.closed {
		return nil, errors.New(errors.CodeInternalError, "orchestrator is closed")
	}
	if req == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "request cannot be nil")
	}

	// 验证请求
	if err := o.ValidateRequest(ctx, req); err != nil {
		return nil, err
	}

	// 生成执行ID
	executionID := o.generateExecutionID()

	// 创建执行上下文
	execCtx := o.createExecutionContext(ctx, req, executionID)

	// 注册执行上下文
	o.registerExecution(executionID, execCtx)
	defer o.unregisterExecution(executionID)

	// 设置超时
	timeout := req.Timeout
	if timeout == 0 {
		timeout = o.config.ExecutionTimeout
	}
	execCtx.ctx, execCtx.cancel = context.WithTimeout(ctx, timeout)
	defer execCtx.cancel()

	// 记录开始时间
	startTime := time.Now()
	execCtx.startTime = startTime
	execCtx.status = ExecutionStatusRunning

	// 更新指标
	o.metrics.mu.Lock()
	o.metrics.totalExecutions++
	o.metrics.mu.Unlock()

	// 执行拦截器 Before
	for _, interceptor := range o.interceptors {
		if err := interceptor.Before(execCtx.ctx, req); err != nil {
			return nil, err
		}
	}

	// 构建执行链
	executeFunc := o.buildExecutionChain()

	// 执行请求
	resp, err := executeFunc(execCtx.ctx, req)

	// 计算执行时长
	duration := time.Since(startTime)

	// 更新响应
	if resp == nil {
		resp = &ExecuteResponse{
			ExecutionID: executionID,
			RequestID:   req.RequestID,
		}
	}
	resp.Duration = duration
	resp.StartTime = startTime
	resp.EndTime = time.Now()

	// 处理错误
	if err != nil {
		resp.Status = ExecutionStatusFailed
		resp.Error = err
		execCtx.status = ExecutionStatusFailed

		o.metrics.mu.Lock()
		o.metrics.failedExecutions++
		o.metrics.mu.Unlock()

		// 执行拦截器 OnError
		for _, interceptor := range o.interceptors {
			interceptor.OnError(execCtx.ctx, req, err)
		}

		return resp, err
	}

	// 更新状态
	resp.Status = ExecutionStatusCompleted
	execCtx.status = ExecutionStatusCompleted

	// 更新指标
	o.metrics.mu.Lock()
	o.metrics.successfulExecutions++
	o.metrics.totalDuration += duration
	o.metrics.mu.Unlock()

	// 执行拦截器 After
	for _, interceptor := range o.interceptors {
		if err := interceptor.After(execCtx.ctx, req, resp); err != nil {
			o.logger.Error("interceptor after failed", "error", err)
		}
	}

	// 保存执行记录
	if o.executionRepo != nil {
		go o.saveExecution(req, resp)
	}

	return resp, nil
}

// ExecuteStream 流式执行请求
func (o *orchestrator) ExecuteStream(ctx context.Context, req *ExecuteRequest, stream ResponseStream) error {
	if o.closed {
		return errors.New(errors.CodeInternalError, "orchestrator is closed")
	}
	if req == nil {
		return errors.New(errors.CodeInvalidParameter, "request cannot be nil")
	}
	if stream == nil {
		return errors.New(errors.CodeInvalidParameter, "stream cannot be nil")
	}

	// 验证请求
	if err := o.ValidateRequest(ctx, req); err != nil {
		return err
	}

	// 生成执行ID
	executionID := o.generateExecutionID()

	// 发送开始块
	if err := stream.Send(&StreamChunk{
		ExecutionID: executionID,
		Type:        ChunkTypeStart,
		Metadata:    map[string]interface{}{"request_id": req.RequestID},
	}); err != nil {
		return err
	}

	// 创建执行上下文
	execCtx := o.createExecutionContext(ctx, req, executionID)
	o.registerExecution(executionID, execCtx)
	defer o.unregisterExecution(executionID)

	// 设置超时
	timeout := req.Timeout
	if timeout == 0 {
		timeout = o.config.ExecutionTimeout
	}
	execCtx.ctx, execCtx.cancel = context.WithTimeout(ctx, timeout)
	defer execCtx.cancel()

	// 记录开始时间
	startTime := time.Now()
	execCtx.startTime = startTime
	execCtx.status = ExecutionStatusRunning

	// 阶段1: 解析请求
	execCtx.stage = "parsing"
	parsedReq, err := o.parser.Parse(execCtx.ctx, &parser.ParseRequest{
		Input:   req.Input,
		Context: req.Context,
	})
	if err != nil {
		stream.Send(&StreamChunk{
			ExecutionID: executionID,
			Type:        ChunkTypeError,
			Error:       err,
			IsLast:      true,
		})
		return err
	}

	// 阶段2: 策略检查
	execCtx.stage = "policy_check"
	if err := o.checkPolicies(execCtx.ctx, req, parsedReq); err != nil {
		stream.Send(&StreamChunk{
			ExecutionID: executionID,
			Type:        ChunkTypeError,
			Error:       err,
			IsLast:      true,
		})
		return err
	}

	// 阶段3: 路由
	execCtx.stage = "routing"
	route, err := o.router.Route(execCtx.ctx, &router.RouteRequest{
		Intent:      parsedReq.Intent,
		Context:     req.Context,
		Preferences: req.Options.ModelPreferences,
	})
	if err != nil {
		stream.Send(&StreamChunk{
			ExecutionID: executionID,
			Type:        ChunkTypeError,
			Error:       err,
			IsLast:      true,
		})
		return err
	}

	// 阶段4: 插件前置处理
	execCtx.stage = "plugin_before"
	if req.Options != nil && req.Options.EnablePlugins {
		if err := o.executePluginsBefore(execCtx.ctx, req, parsedReq); err != nil {
			stream.Send(&StreamChunk{
				ExecutionID: executionID,
				Type:        ChunkTypeError,
				Error:       err,
				IsLast:      true,
			})
			return err
		}
	}

	// 阶段5: 调度任务
	execCtx.stage = "scheduling"
	task, err := o.scheduler.Schedule(execCtx.ctx, &scheduler.ScheduleRequest{
		Request:  req,
		Route:    route,
		Priority: req.Priority,
	})
	if err != nil {
		stream.Send(&StreamChunk{
			ExecutionID: executionID,
			Type:        ChunkTypeError,
			Error:       err,
			IsLast:      true,
		})
		return err
	}

	// 阶段6: 执行任务（流式）
	execCtx.stage = "executing"
	streamChan := make(chan *executor.StreamChunk, o.config.StreamBufferSize)
	errChan := make(chan error, 1)

	go func() {
		err := o.executor.ExecuteStream(execCtx.ctx, &executor.ExecuteRequest{
			Task:    task,
			Context: req.Context,
			Options: req.Options,
		}, streamChan)
		if err != nil {
			errChan <- err
		}
		close(streamChan)
	}()

	// 转发流数据
	for {
		select {
		case chunk, ok := <-streamChan:
			if !ok {
				// 流结束
				goto AFTER_PLUGINS
			}
			if err := stream.Send(&StreamChunk{
				ExecutionID: executionID,
				Type:        ChunkTypeContent,
				Content:     chunk.Content,
				Delta:       chunk.Delta,
				Metadata:    chunk.Metadata,
			}); err != nil {
				return err
			}
		case err := <-errChan:
			stream.Send(&StreamChunk{
				ExecutionID: executionID,
				Type:        ChunkTypeError,
				Error:       err,
				IsLast:      true,
			})
			return err
		case <-execCtx.ctx.Done():
			stream.Send(&StreamChunk{
				ExecutionID: executionID,
				Type:        ChunkTypeError,
				Error:       execCtx.ctx.Err(),
				IsLast:      true,
			})
			return execCtx.ctx.Err()
		}
	}

AFTER_PLUGINS:
	// 阶段7: 插件后置处理
	execCtx.stage = "plugin_after"
	if req.Options != nil && req.Options.EnablePlugins {
		if err := o.executePluginsAfter(execCtx.ctx, req, parsedReq, nil); err != nil {
			stream.Send(&StreamChunk{
				ExecutionID: executionID,
				Type:        ChunkTypeError,
				Error:       err,
				IsLast:      true,
			})
			return err
		}
	}

	// 发送结束块
	duration := time.Since(startTime)
	stream.Send(&StreamChunk{
		ExecutionID: executionID,
		Type:        ChunkTypeEnd,
		Metadata: map[string]interface{}{
			"duration": duration.String(),
			"status":   "completed",
		},
		IsLast: true,
	})

	execCtx.status = ExecutionStatusCompleted
	return stream.Close()
}

// ExecuteBatch 批量执行请求
func (o *orchestrator) ExecuteBatch(ctx context.Context, requests []*ExecuteRequest) ([]*ExecuteResponse, error) {
	if o.closed {
		return nil, errors.New(errors.CodeInternalError, "orchestrator is closed")
	}
	if len(requests) == 0 {
		return nil, errors.New(errors.CodeInvalidParameter, "requests cannot be empty")
	}

	responses := make([]*ExecuteResponse, len(requests))
	errChan := make(chan error, len(requests))
	var wg sync.WaitGroup

	for i, req := range requests {
		wg.Add(1)
		go func(idx int, request *ExecuteRequest) {
			defer wg.Done()
			resp, err := o.Execute(ctx, request)
			if err != nil {
				errChan <- err
			}
			responses[idx] = resp
		}(i, req)
	}

	wg.Wait()
	close(errChan)

	// 检查是否有错误
	if len(errChan) > 0 {
		return responses, <-errChan
	}

	return responses, nil
}

// ExecuteAsync 异步执行请求
func (o *orchestrator) ExecuteAsync(ctx context.Context, req *ExecuteRequest, callback ExecuteCallback) error {
	if o.closed {
		return errors.New(errors.CodeInternalError, "orchestrator is closed")
	}
	if req == nil {
		return errors.New(errors.CodeInvalidParameter, "request cannot be nil")
	}
	if callback == nil {
		return errors.New(errors.CodeInvalidParameter, "callback cannot be nil")
	}

	go func() {
		resp, err := o.Execute(ctx, req)
		callback(resp, err)
	}()

	return nil
}

// ValidateRequest 验证请求
func (o *orchestrator) ValidateRequest(ctx context.Context, req *ExecuteRequest) error {
	if req == nil {
		return errors.New(errors.CodeInvalidParameter, "request cannot be nil")
	}
	if req.Input == "" {
		return errors.New(errors.CodeInvalidParameter, "input cannot be empty")
	}
	if req.UserID == "" {
		return errors.New(errors.CodeInvalidParameter, "user id cannot be empty")
	}
	return nil
}

// GetExecutionStatus 获取执行状态
func (o *orchestrator) GetExecutionStatus(ctx context.Context, executionID string) (*ExecutionStatus, error) {
	if executionID == "" {
		return nil, errors.New(errors.CodeInvalidParameter, "execution id cannot be empty")
	}

	o.mu.RLock()
	execCtx, exists := o.executions[executionID]
	o.mu.RUnlock()

	if !exists {
		return nil, errors.New(errors.CodeNotFound, "execution not found")
	}

	execCtx.mu.RLock()
	defer execCtx.mu.RUnlock()

	return &ExecutionStatus{
		ExecutionID:  executionID,
		RequestID:    execCtx.req.RequestID,
		Status:       execCtx.status,
		StartTime:    execCtx.startTime,
		CurrentStage: execCtx.stage,
	}, nil
}

// CancelExecution 取消执行
func (o *orchestrator) CancelExecution(ctx context.Context, executionID string) error {
	if executionID == "" {
		return errors.New(errors.CodeInvalidParameter, "execution id cannot be empty")
	}

	o.mu.RLock()
	execCtx, exists := o.executions[executionID]
	o.mu.RUnlock()

	if !exists {
		return errors.New(errors.CodeNotFound, "execution not found")
	}

	execCtx.cancel()
	execCtx.mu.Lock()
	execCtx.status = ExecutionStatusCancelled
	execCtx.mu.Unlock()

	o.metrics.mu.Lock()
	o.metrics.cancelledExecutions++
	o.metrics.mu.Unlock()

	return nil
}

// GetMetrics 获取指标
func (o *orchestrator) GetMetrics(ctx context.Context) (*OrchestratorMetrics, error) {
	o.metrics.mu.RLock()
	defer o.metrics.mu.RUnlock()

	avgDuration := time.Duration(0)
	if o.metrics.successfulExecutions > 0 {
		avgDuration = time.Duration(int64(o.metrics.totalDuration) / o.metrics.successfulExecutions)
	}

	cacheHitRate := 0.0
	totalCacheRequests := o.metrics.cacheHits + o.metrics.cacheMisses
	if totalCacheRequests > 0 {
		cacheHitRate = float64(o.metrics.cacheHits) / float64(totalCacheRequests)
	}

	o.mu.RLock()
	activeExecutions := len(o.executions)
	o.mu.RUnlock()

	pluginMetrics := make(map[string]*PluginMetrics)
	for name, metrics := range o.metrics.pluginMetrics {
		pluginMetrics[name] = &PluginMetrics{
			ExecutionCount:  metrics.ExecutionCount,
			SuccessCount:    metrics.SuccessCount,
			FailureCount:    metrics.FailureCount,
			AverageDuration: metrics.AverageDuration,
			LastExecutedAt:  metrics.LastExecutedAt,
		}
	}

	return &OrchestratorMetrics{
		TotalExecutions:      o.metrics.totalExecutions,
		SuccessfulExecutions: o.metrics.successfulExecutions,
		FailedExecutions:     o.metrics.failedExecutions,
		CancelledExecutions:  o.metrics.cancelledExecutions,
		AverageDuration:      avgDuration,
		CacheHitRate:         cacheHitRate,
		ActiveExecutions:     activeExecutions,
		PluginMetrics:        pluginMetrics,
	}, nil
}

// Close 关闭编排器
func (o *orchestrator) Close() error {
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return nil
	}
	o.closed = true
	o.mu.Unlock()

	// 关闭信号通道
	close(o.closeChan)

	// 等待所有后台任务完成
	o.wg.Wait()

	// 取消所有正在执行的任务
	o.mu.Lock()
	for _, execCtx := range o.executions {
		execCtx.cancel()
	}
	o.mu.Unlock()

	return nil
}

// buildExecutionChain 构建执行链
func (o *orchestrator) buildExecutionChain() ExecuteFunc {
	// 基础执行函数
	baseFunc := o.executeCore

	// 应用中间件（反向顺序）
	for i := len(o.middlewares) - 1; i >= 0; i-- {
		baseFunc = o.middlewares[i](baseFunc)
	}

	return baseFunc
}

// executeCore 核心执行逻辑
func (o *orchestrator) executeCore(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	executionID := o.extractExecutionID(ctx)

	// 阶段1: 解析请求
	parsedReq, err := o.parser.Parse(ctx, &parser.ParseRequest{
		Input:   req.Input,
		Context: req.Context,
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to parse request")
	}

	// 阶段2: 策略检查
	if err := o.checkPolicies(ctx, req, parsedReq); err != nil {
		return nil, err
	}

	// 阶段3: 检查缓存
	if req.Options != nil && req.Options.EnableCache && o.config.EnableCaching {
		if cached, hit := o.checkCache(ctx, req); hit {
			o.metrics.mu.Lock()
			o.metrics.cacheHits++
			o.metrics.mu.Unlock()

			cached.CacheHit = true
			return cached, nil
		}
		o.metrics.mu.Lock()
		o.metrics.cacheMisses++
		o.metrics.mu.Unlock()
	}

	// 阶段4: 路由
	route, err := o.router.Route(ctx, &router.RouteRequest{
		Intent:      parsedReq.Intent,
		Context:     req.Context,
		Preferences: req.Options.ModelPreferences,
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to route request")
	}

	// 阶段5: 插件前置处理
	if req.Options != nil && req.Options.EnablePlugins {
		if err := o.executePluginsBefore(ctx, req, parsedReq); err != nil {
			return nil, errors.Wrap(err, errors.CodeInternalError, "plugin before failed")
		}
	}

	// 阶段6: 调度任务
	task, err := o.scheduler.Schedule(ctx, &scheduler.ScheduleRequest{
		Request:  req,
		Route:    route,
		Priority: req.Priority,
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to schedule task")
	}

	// 阶段7: 执行任务
	execResp, err := o.executor.Execute(ctx, &executor.ExecuteRequest{
		Task:    task,
		Context: req.Context,
		Options: req.Options,
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to execute task")
	}

	// 阶段8: 插件后置处理
	if req.Options != nil && req.Options.EnablePlugins {
		if err := o.executePluginsAfter(ctx, req, parsedReq, execResp); err != nil {
			o.logger.Error("plugin after failed", "error", err)
		}
	}

	// 构建响应
	resp := &ExecuteResponse{
		ExecutionID:     executionID,
		RequestID:       req.RequestID,
		Output:          execResp.Output,
		Model:           execResp.Model,
		TokensUsed:      execResp.TokensUsed,
		Cost:            execResp.Cost,
		Metadata:        execResp.Metadata,
		PluginsExecuted: o.getExecutedPlugins(ctx),
	}

	// 缓存结果
	if req.Options != nil && req.Options.EnableCache && o.config.EnableCaching {
		o.cacheResult(ctx, req, resp)
	}

	return resp, nil
}

// checkPolicies 检查策略
func (o *orchestrator) checkPolicies(ctx context.Context, req *ExecuteRequest, parsedReq *parser.ParseResponse) error {
	if o.policyRepo == nil {
		return nil
	}

	// 获取用户策略
	policies, err := o.policyRepo.FindByUserID(ctx, req.UserID)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to get policies")
	}

	// 检查每个策略
	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}

		// 检查速率限制
		if policy.RateLimit != nil {
			if err := o.checkRateLimit(ctx, req.UserID, policy.RateLimit); err != nil {
				return err
			}
		}

		// 检查配额
		if policy.Quota != nil {
			if err := o.checkQuota(ctx, req.UserID, policy.Quota); err != nil {
				return err
			}
		}

		// 检查内容过滤
		if policy.ContentFilter != nil {
			if err := o.checkContentFilter(ctx, req.Input, policy.ContentFilter); err != nil {
				return err
			}
		}
	}

	return nil
}

// checkRateLimit 检查速率限制
func (o *orchestrator) checkRateLimit(ctx context.Context, userID string, limit *entity.RateLimit) error {
	// 实现速率限制检查逻辑
	// 这里简化处理，实际应该使用 Redis 或其他分布式存储
	return nil
}

// checkQuota 检查配额
func (o *orchestrator) checkQuota(ctx context.Context, userID string, quota *entity.Quota) error {
	// 实现配额检查逻辑
	return nil
}

// checkContentFilter 检查内容过滤
func (o *orchestrator) checkContentFilter(ctx context.Context, content string, filter *entity.ContentFilter) error {
	// 实现内容过滤检查逻辑
	return nil
}

// executePluginsBefore 执行前置插件
func (o *orchestrator) executePluginsBefore(ctx context.Context, req *ExecuteRequest, parsedReq *parser.ParseResponse) error {
	if o.pluginManager == nil {
		return nil
	}

	var plugins []string
	if req.Options != nil && len(req.Options.Plugins) > 0 {
		plugins = req.Options.Plugins
	} else {
		// 获取所有启用的插件
		allPlugins, err := o.pluginManager.ListPlugins(ctx)
		if err != nil {
			return err
		}
		for _, p := range allPlugins {
			if p.Enabled {
				plugins = append(plugins, p.Name)
			}
		}
	}

	// 执行每个插件的前置处理
	for _, pluginName := range plugins {
		startTime := time.Now()

		err := o.pluginManager.ExecuteBefore(ctx, pluginName, &plugin.BeforeRequest{
			Input:      req.Input,
			ParsedData: parsedReq,
			Context:    req.Context,
		})

		duration := time.Since(startTime)

		// 更新插件指标
		o.updatePluginMetrics(pluginName, err == nil, duration)

		if err != nil {
			return errors.Wrap(err, errors.CodeInternalError, fmt.Sprintf("plugin %s before failed", pluginName))
		}
	}

	return nil
}

// executePluginsAfter 执行后置插件
func (o *orchestrator) executePluginsAfter(ctx context.Context, req *ExecuteRequest, parsedReq *parser.ParseResponse, execResp *executor.ExecuteResponse) error {
	if o.pluginManager == nil {
		return nil
	}

	var plugins []string
	if req.Options != nil && len(req.Options.Plugins) > 0 {
		plugins = req.Options.Plugins
	} else {
		allPlugins, err := o.pluginManager.ListPlugins(ctx)
		if err != nil {
			return err
		}
		for _, p := range allPlugins {
			if p.Enabled {
				plugins = append(plugins, p.Name)
			}
		}
	}

	// 反向执行后置插件
	for i := len(plugins) - 1; i >= 0; i-- {
		pluginName := plugins[i]
		startTime := time.Now()

		err := o.pluginManager.ExecuteAfter(ctx, pluginName, &plugin.AfterRequest{
			Input:      req.Input,
			Output:     execResp.Output,
			ParsedData: parsedReq,
			Context:    req.Context,
		})

		duration := time.Since(startTime)
		o.updatePluginMetrics(pluginName, err == nil, duration)

		if err != nil {
			return errors.Wrap(err, errors.CodeInternalError, fmt.Sprintf("plugin %s after failed", pluginName))
		}
	}

	return nil
}

// updatePluginMetrics 更新插件指标
func (o *orchestrator) updatePluginMetrics(pluginName string, success bool, duration time.Duration) {
	o.metrics.mu.Lock()
	defer o.metrics.mu.Unlock()

	metrics, exists := o.metrics.pluginMetrics[pluginName]
	if !exists {
		metrics = &PluginMetrics{}
		o.metrics.pluginMetrics[pluginName] = metrics
	}

	metrics.ExecutionCount++
	if success {
		metrics.SuccessCount++
	} else {
		metrics.FailureCount++
	}

	// 更新平均执行时长
	totalDuration := time.Duration(metrics.ExecutionCount) * metrics.AverageDuration
	metrics.AverageDuration = (totalDuration + duration) / time.Duration(metrics.ExecutionCount)
	metrics.LastExecutedAt = time.Now()
}

// checkCache 检查缓存
func (o *orchestrator) checkCache(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, bool) {
	// 实现缓存检查逻辑
	// 这里简化处理，实际应该使用 Redis 或其他缓存系统
	return nil, false
}

// cacheResult 缓存结果
func (o *orchestrator) cacheResult(ctx context.Context, req *ExecuteRequest, resp *ExecuteResponse) {
	// 实现缓存存储逻辑
}

// getExecutedPlugins 获取已执行的插件列表
func (o *orchestrator) getExecutedPlugins(ctx context.Context) []string {
	// 从上下文中提取已执行的插件列表
	return []string{}
}

// createExecutionContext 创建执行上下文
func (o *orchestrator) createExecutionContext(ctx context.Context, req *ExecuteRequest, executionID string) *executionContext {
	return &executionContext{
		req:        req,
		resp:       &ExecuteResponse{ExecutionID: executionID},
		ctx:        ctx,
		status:     ExecutionStatusPending,
		pluginData: make(map[string]interface{}),
	}
}

// registerExecution 注册执行
func (o *orchestrator) registerExecution(executionID string, execCtx *executionContext) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.executions[executionID] = execCtx
}

// unregisterExecution 注销执行
func (o *orchestrator) unregisterExecution(executionID string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.executions, executionID)
}

// generateExecutionID 生成执行ID
func (o *orchestrator) generateExecutionID() string {
	return fmt.Sprintf("exec_%d_%s", time.Now().UnixNano(), o.randomString(8))
}

// extractExecutionID 从上下文提取执行ID
func (o *orchestrator) extractExecutionID(ctx context.Context) string {
	if val := ctx.Value("execution_id"); val != nil {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return o.generateExecutionID()
}

// randomString 生成随机字符串
func (o *orchestrator) randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// saveExecution 保存执行记录
func (o *orchestrator) saveExecution(req *ExecuteRequest, resp *ExecuteResponse) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	execution := &entity.Execution{
		ID:          resp.ExecutionID,
		RequestID:   req.RequestID,
		UserID:      req.UserID,
		SessionID:   req.SessionID,
		Input:       req.Input,
		Output:      resp.Output,
		Status:      string(resp.Status),
		Model:       resp.Model,
		TokensUsed:  resp.TokensUsed,
		Cost:        resp.Cost,
		Duration:    resp.Duration,
		StartTime:   resp.StartTime,
		EndTime:     resp.EndTime,
		Error:       "",
		Metadata:    req.Metadata,
	}

	if resp.Error != nil {
		execution.Error = resp.Error.Error()
	}

	if err := o.executionRepo.Create(ctx, execution); err != nil {
		o.logger.Error("failed to save execution", "error", err)
	}
}

// cleanupLoop 清理循环
func (o *orchestrator) cleanupLoop() {
	defer o.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			o.cleanupStaleExecutions()
		case <-o.closeChan:
			return
		}
	}
}

// cleanupStaleExecutions 清理过期执行
func (o *orchestrator) cleanupStaleExecutions() {
	o.mu.Lock()
	defer o.mu.Unlock()

	now := time.Now()
	staleThreshold := 1 * time.Hour

	for id, execCtx := range o.executions {
		execCtx.mu.RLock()
		isStale := now.Sub(execCtx.startTime) > staleThreshold
		status := execCtx.status
		execCtx.mu.RUnlock()

		// 清理已完成、失败或超时的执行
		if isStale && (status == ExecutionStatusCompleted ||
			status == ExecutionStatusFailed ||
			status == ExecutionStatusCancelled ||
			status == ExecutionStatusTimeout) {
			delete(o.executions, id)
		}
	}
}

// AddInterceptor 添加拦截器
func (o *orchestrator) AddInterceptor(interceptor Interceptor) {
	o.interceptors = append(o.interceptors, interceptor)
}

// AddMiddleware 添加中间件
func (o *orchestrator) AddMiddleware(middleware Middleware) {
	o.middlewares = append(o.middlewares, middleware)
}

//Personal.AI order the ending

