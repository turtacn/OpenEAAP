package orchestrator

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"sync"
	"time"

	"github.com/openeeap/openeeap/internal/domain/entity"
	"github.com/openeeap/openeeap/internal/domain/repository"
	"github.com/openeeap/openeeap/pkg/errors"
	"github.com/openeeap/openeeap/pkg/logger"
)

// Router 路由器接口
type Router interface {
	// Route 路由请求
	Route(ctx context.Context, req *RouteRequest) (*RouteResponse, error)

	// SelectRuntime 选择运行时
	SelectRuntime(ctx context.Context, req *RuntimeSelectionRequest) (*RuntimeInfo, error)

	// SelectModel 选择模型
	SelectModel(ctx context.Context, req *ModelSelectionRequest) (*ModelInfo, error)

	// RegisterRuntime 注册运行时
	RegisterRuntime(runtime *RuntimeInfo) error

	// UnregisterRuntime 注销运行时
	UnregisterRuntime(runtimeID string) error

	// GetRuntimeStatus 获取运行时状态
	GetRuntimeStatus(ctx context.Context, runtimeID string) (*RuntimeStatus, error)

	// UpdateRuntimeMetrics 更新运行时指标
	UpdateRuntimeMetrics(runtimeID string, metrics *RuntimeMetrics) error

	// GetMetrics 获取路由器指标
	GetMetrics(ctx context.Context) (*RouterMetrics, error)
}

// router 路由器实现
type router struct {
	agentRepo     repository.AgentRepository
	modelRepo     repository.ModelRepository
	config        *RouterConfig
	logger        logger.Logger

	runtimes      map[string]*RuntimeInfo
	runtimesMu    sync.RWMutex

	strategies    map[string]RoutingStrategy
	loadBalancers map[string]LoadBalancer

	metrics       *routerMetrics
	healthChecker *healthChecker

	closed        bool
	closeChan     chan struct{}
	wg            sync.WaitGroup
}

// RouterConfig 路由器配置
type RouterConfig struct {
	DefaultStrategy          RoutingStrategyType  // 默认路由策略
	DefaultLoadBalancer      LoadBalancerType     // 默认负载均衡器
	EnableHealthCheck        bool                 // 启用健康检查
	HealthCheckInterval      time.Duration        // 健康检查间隔
	EnableMetrics            bool                 // 启用指标
	EnableCircuitBreaker     bool                 // 启用熔断器
	CircuitBreakerThreshold  int                  // 熔断器阈值
	CircuitBreakerTimeout    time.Duration        // 熔断器超时
	EnableRateLimiting       bool                 // 启用限流
	MaxRequestsPerSecond     int                  // 每秒最大请求数
	EnableCaching            bool                 // 启用缓存
	CacheTTL                 time.Duration        // 缓存TTL
	EnableStickySessions     bool                 // 启用粘性会话
	SessionAffinityTimeout   time.Duration        // 会话亲和性超时
	EnableFailover           bool                 // 启用故障转移
	MaxFailoverAttempts      int                  // 最大故障转移次数
	EnableWeightedRouting    bool                 // 启用加权路由
	EnableCostOptimization   bool                 // 启用成本优化
	EnableLatencyOptimization bool                // 启用延迟优化
}

// RouteRequest 路由请求
type RouteRequest struct {
	RequestID       string                 // 请求ID
	SessionID       string                 // 会话ID
	UserID          string                 // 用户ID
	Intent          *Intent                // 意图
	Context         map[string]interface{} // 上下文
	Preferences     *ModelPreferences      // 模型偏好
	Requirements    *RouteRequirements     // 路由要求
	Metadata        map[string]string      // 元数据
}

// RouteRequirements 路由要求
type RouteRequirements struct {
	RuntimeType     RuntimeType    // 运行时类型
	ModelType       string         // 模型类型
	MinQuality      float64        // 最小质量
	MaxLatency      time.Duration  // 最大延迟
	MaxCost         float64        // 最大成本
	RequiredFeatures []string      // 必需特性
	PreferredRegion string         // 偏好地区
}

// RouteResponse 路由响应
type RouteResponse struct {
	Runtime         *RuntimeInfo           // 运行时信息
	Model           *ModelInfo             // 模型信息
	Strategy        RoutingStrategyType    // 使用的策略
	LoadBalancer    LoadBalancerType       // 使用的负载均衡器
	Score           float64                // 评分
	Metadata        map[string]interface{} // 元数据
	RoutingTime     time.Duration          // 路由时间
}

// RuntimeSelectionRequest 运行时选择请求
type RuntimeSelectionRequest struct {
	RuntimeType      RuntimeType            // 运行时类型
	Requirements     *RouteRequirements     // 要求
	Strategy         RoutingStrategyType    // 策略
	LoadBalancer     LoadBalancerType       // 负载均衡器
	Context          map[string]interface{} // 上下文
}

// ModelSelectionRequest 模型选择请求
type ModelSelectionRequest struct {
	Intent           *Intent                // 意图
	Preferences      *ModelPreferences      // 偏好
	Requirements     *RouteRequirements     // 要求
	Context          map[string]interface{} // 上下文
}

// RuntimeInfo 运行时信息
type RuntimeInfo struct {
	ID              string                 // 运行时ID
	Name            string                 // 名称
	Type            RuntimeType            // 类型
	Version         string                 // 版本
	Endpoint        string                 // 端点
	Status          RuntimeStatusType      // 状态
	Capacity        *RuntimeCapacity       // 容量
	Metrics         *RuntimeMetrics        // 指标
	Features        []string               // 特性列表
	SupportedModels []string               // 支持的模型列表
	Region          string                 // 地区
	Weight          int                    // 权重
	Priority        int                    // 优先级
	Tags            []string               // 标签
	Metadata        map[string]interface{} // 元数据
	RegisteredAt    time.Time              // 注册时间
	LastHealthCheck time.Time              // 最后健康检查时间
}

// RuntimeType 运行时类型
type RuntimeType string

const (
	RuntimeTypeNative    RuntimeType = "native"     // 原生
	RuntimeTypeLangChain RuntimeType = "langchain"  // LangChain
	RuntimeTypeAutoGPT   RuntimeType = "autogpt"    // AutoGPT
	RuntimeTypeCrewAI    RuntimeType = "crewai"     // CrewAI
	RuntimeTypeCustom    RuntimeType = "custom"     // 自定义
)

// RuntimeStatusType 运行时状态类型
type RuntimeStatusType string

const (
	RuntimeStatusHealthy   RuntimeStatusType = "healthy"   // 健康
	RuntimeStatusDegraded  RuntimeStatusType = "degraded"  // 降级
	RuntimeStatusUnhealthy RuntimeStatusType = "unhealthy" // 不健康
	RuntimeStatusOffline   RuntimeStatusType = "offline"   // 离线
)

// RuntimeCapacity 运行时容量
type RuntimeCapacity struct {
	MaxConcurrentRequests int     // 最大并发请求数
	CurrentLoad           int     // 当前负载
	CPULimit              float64 // CPU限制
	MemoryLimit           int64   // 内存限制
	DiskLimit             int64   // 磁盘限制
	AvailableSlots        int     // 可用槽位
}

// RuntimeMetrics 运行时指标
type RuntimeMetrics struct {
	TotalRequests       int64         // 总请求数
	SuccessfulRequests  int64         // 成功请求数
	FailedRequests      int64         // 失败请求数
	AverageLatency      time.Duration // 平均延迟
	P95Latency          time.Duration // P95延迟
	P99Latency          time.Duration // P99延迟
	ErrorRate           float64       // 错误率
	Throughput          float64       // 吞吐量（请求/秒）
	CPUUsage            float64       // CPU使用率
	MemoryUsage         int64         // 内存使用
	ActiveConnections   int           // 活跃连接数
	LastRequestTime     time.Time     // 最后请求时间
	LastUpdateTime      time.Time     // 最后更新时间
}

// RuntimeStatus 运行时状态
type RuntimeStatus struct {
	RuntimeID       string            // 运行时ID
	Status          RuntimeStatusType // 状态
	Capacity        *RuntimeCapacity  // 容量
	Metrics         *RuntimeMetrics   // 指标
	HealthScore     float64           // 健康评分
	LastHealthCheck time.Time         // 最后健康检查时间
	Message         string            // 消息
}

// ModelInfo 模型信息
type ModelInfo struct {
	ID              string                 // 模型ID
	Name            string                 // 名称
	Provider        string                 // 提供商
	Type            string                 // 类型
	Version         string                 // 版本
	Endpoint        string                 // 端点
	ContextLength   int                    // 上下文长度
	Features        []string               // 特性列表
	Pricing         *ModelPricing          // 价格信息
	Performance     *ModelPerformance      // 性能信息
	Capabilities    *ModelCapabilities     // 能力
	Metadata        map[string]interface{} // 元数据
}

// ModelPricing 模型价格
type ModelPricing struct {
	InputCostPer1K  float64 // 每1K输入令牌成本
	OutputCostPer1K float64 // 每1K输出令牌成本
	Currency        string  // 货币
}

// ModelPerformance 模型性能
type ModelPerformance struct {
	AverageLatency  time.Duration // 平均延迟
	QualityScore    float64       // 质量评分
	ReliabilityScore float64      // 可靠性评分
	TokensPerSecond float64       // 每秒令牌数
}

// ModelCapabilities 模型能力
type ModelCapabilities struct {
	SupportsStreaming    bool     // 支持流式
	SupportsEmbedding    bool     // 支持嵌入
	SupportsFunctionCall bool     // 支持函数调用
	SupportsVision       bool     // 支持视觉
	SupportsAudio        bool     // 支持音频
	Languages            []string // 支持的语言
}

// RoutingStrategy 路由策略接口
type RoutingStrategy interface {
	Select(ctx context.Context, candidates []*RuntimeInfo, req *RouteRequest) (*RuntimeInfo, error)
}

// RoutingStrategyType 路由策略类型
type RoutingStrategyType string

const (
	RoutingStrategyRoundRobin    RoutingStrategyType = "round_robin"    // 轮询
	RoutingStrategyLeastLoad     RoutingStrategyType = "least_load"     // 最少负载
	RoutingStrategyWeighted      RoutingStrategyType = "weighted"       // 加权
	RoutingStrategyRandom        RoutingStrategyType = "random"         // 随机
	RoutingStrategyHash          RoutingStrategyType = "hash"           // 哈希
	RoutingStrategyLatency       RoutingStrategyType = "latency"        // 延迟优先
	RoutingStrategyCost          RoutingStrategyType = "cost"           // 成本优先
	RoutingStrategyQuality       RoutingStrategyType = "quality"        // 质量优先
	RoutingStrategyComposite     RoutingStrategyType = "composite"      // 组合策略
)

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	Next(ctx context.Context, runtimes []*RuntimeInfo) (*RuntimeInfo, error)
}

// LoadBalancerType 负载均衡器类型
type LoadBalancerType string

const (
	LoadBalancerTypeRoundRobin LoadBalancerType = "round_robin" // 轮询
	LoadBalancerTypeLeastConn  LoadBalancerType = "least_conn"  // 最少连接
	LoadBalancerTypeWeighted   LoadBalancerType = "weighted"    // 加权
	LoadBalancerTypeRandom     LoadBalancerType = "random"      // 随机
	LoadBalancerTypeIPHash     LoadBalancerType = "ip_hash"     // IP哈希
)

// RouterMetrics 路由器指标
type RouterMetrics struct {
	TotalRoutes         int64                       // 总路由数
	SuccessfulRoutes    int64                       // 成功路由数
	FailedRoutes        int64                       // 失败路由数
	AverageRoutingTime  time.Duration               // 平均路由时间
	RuntimeMetrics      map[string]*RuntimeMetrics  // 运行时指标
	StrategyUsage       map[string]int64            // 策略使用次数
	LoadBalancerUsage   map[string]int64            // 负载均衡器使用次数
}

// routerMetrics 内部指标
type routerMetrics struct {
	totalRoutes        int64
	successfulRoutes   int64
	failedRoutes       int64
	totalRoutingTime   time.Duration
	strategyUsage      map[string]int64
	loadBalancerUsage  map[string]int64
	mu                 sync.RWMutex
}

// healthChecker 健康检查器
type healthChecker struct {
	router   *router
	interval time.Duration
	logger   logger.Logger
}

// NewRouter 创建路由器
func NewRouter(
	agentRepo repository.AgentRepository,
	modelRepo repository.ModelRepository,
	config *RouterConfig,
	logger logger.Logger,
) (Router, error) {
	if config == nil {
		config = &RouterConfig{
			DefaultStrategy:          RoutingStrategyComposite,
			DefaultLoadBalancer:      LoadBalancerTypeRoundRobin,
			EnableHealthCheck:        true,
			HealthCheckInterval:      30 * time.Second,
			EnableMetrics:            true,
			EnableCircuitBreaker:     true,
			CircuitBreakerThreshold:  5,
			CircuitBreakerTimeout:    1 * time.Minute,
			EnableFailover:           true,
			MaxFailoverAttempts:      3,
			EnableWeightedRouting:    true,
			EnableCostOptimization:   true,
			EnableLatencyOptimization: true,
		}
	}

	r := &router{
		agentRepo:     agentRepo,
		modelRepo:     modelRepo,
		config:        config,
		logger:        logger,
		runtimes:      make(map[string]*RuntimeInfo),
		strategies:    make(map[string]RoutingStrategy),
		loadBalancers: make(map[string]LoadBalancer),
		metrics: &routerMetrics{
			strategyUsage:     make(map[string]int64),
			loadBalancerUsage: make(map[string]int64),
		},
		closeChan:     make(chan struct{}),
	}

	// 初始化路由策略
	r.initializeStrategies()

	// 初始化负载均衡器
	r.initializeLoadBalancers()

	// 初始化健康检查器
	if config.EnableHealthCheck {
		r.healthChecker = &healthChecker{
			router:   r,
			interval: config.HealthCheckInterval,
			logger:   logger,
		}
		r.wg.Add(1)
		go r.healthChecker.start()
	}

	return r, nil
}

// Route 路由请求
func (r *router) Route(ctx context.Context, req *RouteRequest) (*RouteResponse, error) {
	if r.closed {
		return nil, errors.New(errors.CodeInternalError, "router is closed")
	}
	if req == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "route request cannot be nil")
	}

	startTime := time.Now()

	// 更新指标
	r.metrics.mu.Lock()
	r.metrics.totalRoutes++
	r.metrics.mu.Unlock()

	// 1. 选择运行时
	runtimeReq := &RuntimeSelectionRequest{
		Requirements: req.Requirements,
		Strategy:     r.config.DefaultStrategy,
		LoadBalancer: r.config.DefaultLoadBalancer,
		Context:      req.Context,
	}

	runtime, err := r.SelectRuntime(ctx, runtimeReq)
	if err != nil {
		r.metrics.mu.Lock()
		r.metrics.failedRoutes++
		r.metrics.mu.Unlock()
		return nil, err
	}

	// 2. 选择模型
	modelReq := &ModelSelectionRequest{
		Intent:       req.Intent,
		Preferences:  req.Preferences,
		Requirements: req.Requirements,
		Context:      req.Context,
	}

	model, err := r.SelectModel(ctx, modelReq)
	if err != nil {
		r.metrics.mu.Lock()
		r.metrics.failedRoutes++
		r.metrics.mu.Unlock()
		return nil, err
	}

	// 计算路由时间
	routingTime := time.Since(startTime)

	// 更新成功指标
	r.metrics.mu.Lock()
	r.metrics.successfulRoutes++
	r.metrics.totalRoutingTime += routingTime
	r.metrics.mu.Unlock()

	// 构建响应
	resp := &RouteResponse{
		Runtime:      runtime,
		Model:        model,
		Strategy:     runtimeReq.Strategy,
		LoadBalancer: runtimeReq.LoadBalancer,
		Score:        r.calculateScore(runtime, model),
		Metadata:     make(map[string]interface{}),
		RoutingTime:  routingTime,
	}

	return resp, nil
}

// SelectRuntime 选择运行时
func (r *router) SelectRuntime(ctx context.Context, req *RuntimeSelectionRequest) (*RuntimeInfo, error) {
	// 获取所有可用运行时
	candidates := r.getAvailableRuntimes(req.RuntimeType)
	if len(candidates) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no available runtime found")
	}

	// 过滤运行时
	filtered := r.filterRuntimes(candidates, req.Requirements)
	if len(filtered) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no runtime matches requirements")
	}

	// 应用负载均衡
	lb := r.getLoadBalancer(req.LoadBalancer)
	runtime, err := lb.Next(ctx, filtered)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "load balancer failed")
	}

	// 更新负载均衡器使用统计
	r.metrics.mu.Lock()
	r.metrics.loadBalancerUsage[string(req.LoadBalancer)]++
	r.metrics.mu.Unlock()

	return runtime, nil
}

// SelectModel 选择模型
func (r *router) SelectModel(ctx context.Context, req *ModelSelectionRequest) (*ModelInfo, error) {
	if r.modelRepo == nil {
		return nil, errors.New(errors.CodeInternalError, "model repository not configured")
	}

	// 获取所有可用模型
	models, err := r.modelRepo.FindAll(ctx)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to get models")
	}

	if len(models) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no models available")
	}

	// 过滤模型
	candidates := r.filterModels(models, req)

	if len(candidates) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no model matches requirements")
	}

	// 选择最佳模型
	bestModel := r.selectBestModel(candidates, req)

	// 转换为 ModelInfo
	modelInfo := &ModelInfo{
		ID:            bestModel.ID,
		Name:          bestModel.Name,
		Provider:      bestModel.Provider,
		Type:          bestModel.Type,
		Version:       bestModel.Version,
		Endpoint:      bestModel.Endpoint,
		ContextLength: bestModel.ContextLength,
		Features:      bestModel.Features,
		Metadata:      bestModel.Metadata,
	}

	return modelInfo, nil
}

// RegisterRuntime 注册运行时
func (r *router) RegisterRuntime(runtime *RuntimeInfo) error {
	if runtime == nil {
		return errors.New(errors.CodeInvalidParameter, "runtime cannot be nil")
	}
	if runtime.ID == "" {
		return errors.New(errors.CodeInvalidParameter, "runtime id cannot be empty")
	}

	r.runtimesMu.Lock()
	defer r.runtimesMu.Unlock()

	runtime.RegisteredAt = time.Now()
	runtime.Status = RuntimeStatusHealthy
	r.runtimes[runtime.ID] = runtime

	r.logger.Info("runtime registered", "runtime_id", runtime.ID, "type", runtime.Type)
	return nil
}

// UnregisterRuntime 注销运行时
func (r *router) UnregisterRuntime(runtimeID string) error {
	if runtimeID == "" {
		return errors.New(errors.CodeInvalidParameter, "runtime id cannot be empty")
	}

	r.runtimesMu.Lock()
	defer r.runtimesMu.Unlock()

	if _, exists := r.runtimes[runtimeID]; !exists {
		return errors.New(errors.CodeNotFound, "runtime not found")
	}

	delete(r.runtimes, runtimeID)
	r.logger.Info("runtime unregistered", "runtime_id", runtimeID)
	return nil
}

// GetRuntimeStatus 获取运行时状态
func (r *router) GetRuntimeStatus(ctx context.Context, runtimeID string) (*RuntimeStatus, error) {
	if runtimeID == "" {
		return nil, errors.New(errors.CodeInvalidParameter, "runtime id cannot be empty")
	}

	r.runtimesMu.RLock()
	runtime, exists := r.runtimes[runtimeID]
	r.runtimesMu.RUnlock()

	if !exists {
		return nil, errors.New(errors.CodeNotFound, "runtime not found")
	}

	healthScore := r.calculateHealthScore(runtime)

	return &RuntimeStatus{
		RuntimeID:       runtime.ID,
		Status:          runtime.Status,
		Capacity:        runtime.Capacity,
		Metrics:         runtime.Metrics,
		HealthScore:     healthScore,
		LastHealthCheck: runtime.LastHealthCheck,
	}, nil
}

// UpdateRuntimeMetrics 更新运行时指标
func (r *router) UpdateRuntimeMetrics(runtimeID string, metrics *RuntimeMetrics) error {
	if runtimeID == "" {
		return errors.New(errors.CodeInvalidParameter, "runtime id cannot be empty")
	}
	if metrics == nil {
		return errors.New(errors.CodeInvalidParameter, "metrics cannot be nil")
	}

	r.runtimesMu.Lock()
	defer r.runtimesMu.Unlock()

	runtime, exists := r.runtimes[runtimeID]
	if !exists {
		return errors.New(errors.CodeNotFound, "runtime not found")
	}

	runtime.Metrics = metrics
	runtime.Metrics.LastUpdateTime = time.Now()

	// 更新运行时状态
	r.updateRuntimeStatus(runtime)

	return nil
}

// GetMetrics 获取路由器指标
func (r *router) GetMetrics(ctx context.Context) (*RouterMetrics, error) {
	r.metrics.mu.RLock()
	defer r.metrics.mu.RUnlock()

	avgRoutingTime := time.Duration(0)
	if r.metrics.successfulRoutes > 0 {
		avgRoutingTime = time.Duration(int64(r.metrics.totalRoutingTime) / r.metrics.successfulRoutes)
	}

	// 复制运行时指标
	runtimeMetrics := make(map[string]*RuntimeMetrics)
	r.runtimesMu.RLock()
	for id, runtime := range r.runtimes {
		if runtime.Metrics != nil {
			runtimeMetrics[id] = &RuntimeMetrics{
				TotalRequests:      runtime.Metrics.TotalRequests,
				SuccessfulRequests: runtime.Metrics.SuccessfulRequests,
				FailedRequests:     runtime.Metrics.FailedRequests,
				AverageLatency:     runtime.Metrics.AverageLatency,
				P95Latency:         runtime.Metrics.P95Latency,
				P99Latency:         runtime.Metrics.P99Latency,
				ErrorRate:          runtime.Metrics.ErrorRate,
				Throughput:         runtime.Metrics.Throughput,
			}
		}
	}
	r.runtimesMu.RUnlock()

	// 复制策略使用统计
	strategyUsage := make(map[string]int64)
	for k, v := range r.metrics.strategyUsage {
		strategyUsage[k] = v
	}

	// 复制负载均衡器使用统计
	loadBalancerUsage := make(map[string]int64)
	for k, v := range r.metrics.loadBalancerUsage {
		loadBalancerUsage[k] = v
	}

	return &RouterMetrics{
		TotalRoutes:        r.metrics.totalRoutes,
		SuccessfulRoutes:   r.metrics.successfulRoutes,
		FailedRoutes:       r.metrics.failedRoutes,
		AverageRoutingTime: avgRoutingTime,
		RuntimeMetrics:     runtimeMetrics,
		StrategyUsage:      strategyUsage,
		LoadBalancerUsage:  loadBalancerUsage,
	}, nil
}

// initializeStrategies 初始化路由策略
func (r *router) initializeStrategies() {
	r.strategies[string(RoutingStrategyRoundRobin)] = &roundRobinStrategy{}
	r.strategies[string(RoutingStrategyLeastLoad)] = &leastLoadStrategy{}
	r.strategies[string(RoutingStrategyWeighted)] = &weightedStrategy{}
	r.strategies[string(RoutingStrategyRandom)] = &randomStrategy{}
	r.strategies[string(RoutingStrategyHash)] = &hashStrategy{}
	r.strategies[string(RoutingStrategyLatency)] = &latencyStrategy{}
	r.strategies[string(RoutingStrategyCost)] = &costStrategy{}
	r.strategies[string(RoutingStrategyQuality)] = &qualityStrategy{}
	r.strategies[string(RoutingStrategyComposite)] = &compositeStrategy{config: r.config}
}

// initializeLoadBalancers 初始化负载均衡器
func (r *router) initializeLoadBalancers() {
	r.loadBalancers[string(LoadBalancerTypeRoundRobin)] = &roundRobinLoadBalancer{}
	r.loadBalancers[string(LoadBalancerTypeLeastConn)] = &leastConnLoadBalancer{}
	r.loadBalancers[string(LoadBalancerTypeWeighted)] = &weightedLoadBalancer{}
	r.loadBalancers[string(LoadBalancerTypeRandom)] = &randomLoadBalancer{}
	r.loadBalancers[string(LoadBalancerTypeIPHash)] = &ipHashLoadBalancer{}
}

// getAvailableRuntimes 获取可用运行时
func (r *router) getAvailableRuntimes(runtimeType RuntimeType) []*RuntimeInfo {
	r.runtimesMu.RLock()
	defer r.runtimesMu.RUnlock()

	var runtimes []*RuntimeInfo
	for _, runtime := range r.runtimes {
		if runtime.Status == RuntimeStatusHealthy || runtime.Status == RuntimeStatusDegraded {
			if runtimeType == "" || runtime.Type == runtimeType {
				runtimes = append(runtimes, runtime)
			}
		}
	}

	return runtimes
}

// filterRuntimes 过滤运行时
func (r *router) filterRuntimes(runtimes []*RuntimeInfo, requirements *RouteRequirements) []*RuntimeInfo {
	if requirements == nil {
		return runtimes
	}

	var filtered []*RuntimeInfo
	for _, runtime := range runtimes {
		if r.matchesRequirements(runtime, requirements) {
			filtered = append(filtered, runtime)
		}
	}

	return filtered
}

// matchesRequirements 检查运行时是否满足要求
func (r *router) matchesRequirements(runtime *RuntimeInfo, requirements *RouteRequirements) bool {
	// 检查必需特性
	if len(requirements.RequiredFeatures) > 0 {
		featureSet := make(map[string]bool)
		for _, feature := range runtime.Features {
			featureSet[feature] = true
		}
		for _, required := range requirements.RequiredFeatures {
			if !featureSet[required] {
				return false
			}
		}
	}

	// 检查地区偏好
	if requirements.PreferredRegion != "" && runtime.Region != requirements.PreferredRegion {
		return false
	}

	// 检查容量
	if runtime.Capacity != nil && runtime.Capacity.AvailableSlots <= 0 {
		return false
	}

	return true
}

// filterModels 过滤模型
func (r *router) filterModels(models []*entity.Model, req *ModelSelectionRequest) []*entity.Model {
	var filtered []*entity.Model

	for _, model := range models {
		if !model.Enabled {
			continue
		}

		// 检查偏好模型
		if req.Preferences != nil && len(req.Preferences.PreferredModels) > 0 {
			found := false
			for _, preferred := range req.Preferences.PreferredModels {
				if model.Name == preferred {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// 检查排除模型
		if req.Preferences != nil && len(req.Preferences.ExcludedModels) > 0 {
			excluded := false
			for _, ex := range req.Preferences.ExcludedModels {
				if model.Name == ex {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}
		}

		// 检查模型要求
		if req.Preferences != nil && req.Preferences.Requirements != nil {
			reqs := req.Preferences.Requirements

			// 检查最小上下文长度
			if reqs.MinContextLength > 0 && model.ContextLength < reqs.MinContextLength {
				continue
			}

			// 检查必需特性
			if len(reqs.RequiredFeatures) > 0 {
				featureSet := make(map[string]bool)
				for _, feature := range model.Features {
					featureSet[feature] = true
				}
				hasAllFeatures := true
				for _, required := range reqs.RequiredFeatures {
					if !featureSet[required] {
						hasAllFeatures = false
						break
					}
				}
				if !hasAllFeatures {
					continue
				}
			}
		}

		filtered = append(filtered, model)
	}

	return filtered
}

// selectBestModel 选择最佳模型
func (r *router) selectBestModel(models []*entity.Model, req *ModelSelectionRequest) *entity.Model {
	if len(models) == 1 {
		return models[0]
	}

	// 使用综合评分选择最佳模型
	var bestModel *entity.Model
	var bestScore float64

	for _, model := range models {
		score := r.calculateModelScore(model, req)
		if bestModel == nil || score > bestScore {
			bestModel = model
			bestScore = score
		}
	}

	return bestModel
}

// calculateModelScore 计算模型评分
func (r *router) calculateModelScore(model *entity.Model, req *ModelSelectionRequest) float64 {
	score := 0.0

	// 质量权重 (40%)
	if model.Metadata != nil {
		if quality, ok := model.Metadata["quality_score"].(float64); ok {
			score += quality * 0.4
		}
	}

	// 成本权重 (30%) - 成本越低越好
	if r.config.EnableCostOptimization {
		costScore := 1.0 // 默认满分
		if req.Requirements != nil && req.Requirements.MaxCost > 0 {
			// 假设从元数据中获取成本
			if cost, ok := model.Metadata["cost_per_1k"].(float64); ok {
				if cost <= req.Requirements.MaxCost {
					costScore = 1.0 - (cost / req.Requirements.MaxCost * 0.5)
				} else {
					costScore = 0.0
				}
			}
		}
		score += costScore * 0.3
	}

	// 延迟权重 (30%) - 延迟越低越好
	if r.config.EnableLatencyOptimization {
		latencyScore := 1.0 // 默认满分
		if req.Requirements != nil && req.Requirements.MaxLatency > 0 {
			if avgLatency, ok := model.Metadata["avg_latency"].(time.Duration); ok {
				if avgLatency <= req.Requirements.MaxLatency {
					latencyScore = 1.0 - (float64(avgLatency) / float64(req.Requirements.MaxLatency) * 0.5)
				} else {
					latencyScore = 0.0
				}
			}
		}
		score += latencyScore * 0.3
	}

	return score
}

// calculateScore 计算路由评分
func (r *router) calculateScore(runtime *RuntimeInfo, model *ModelInfo) float64 {
	score := 0.0

	// 运行时健康评分 (50%)
	healthScore := r.calculateHealthScore(runtime)
	score += healthScore * 0.5

	// 模型质量评分 (30%)
	if model.Performance != nil {
		score += model.Performance.QualityScore * 0.3
	}

	// 延迟评分 (20%)
	if runtime.Metrics != nil {
		latencyScore := 1.0
		if runtime.Metrics.AverageLatency > 0 {
			// 假设1秒为基准，延迟越低评分越高
			latencyScore = 1.0 - (float64(runtime.Metrics.AverageLatency) / float64(time.Second))
			if latencyScore < 0 {
				latencyScore = 0
			}
		}
		score += latencyScore * 0.2
	}

	return score
}

// calculateHealthScore 计算健康评分
func (r *router) calculateHealthScore(runtime *RuntimeInfo) float64 {
	score := 1.0

	if runtime.Metrics == nil {
		return 0.5 // 无指标时返回中等评分
	}

	// 错误率影响 (40%)
	errorImpact := (1.0 - runtime.Metrics.ErrorRate) * 0.4
	score = errorImpact

	// 负载影响 (30%)
	loadScore := 1.0
	if runtime.Capacity != nil && runtime.Capacity.MaxConcurrentRequests > 0 {
		loadRatio := float64(runtime.Capacity.CurrentLoad) / float64(runtime.Capacity.MaxConcurrentRequests)
		loadScore = 1.0 - loadRatio
		if loadScore < 0 {
			loadScore = 0
		}
	}
	score += loadScore * 0.3

	// 可用性影响 (30%)
	availabilityScore := 1.0
	if runtime.Status != RuntimeStatusHealthy {
		availabilityScore = 0.5
	}
	if runtime.Status == RuntimeStatusUnhealthy {
		availabilityScore = 0.0
	}
	score += availabilityScore * 0.3

	return score
}

// getLoadBalancer 获取负载均衡器
func (r *router) getLoadBalancer(lbType LoadBalancerType) LoadBalancer {
	if lb, exists := r.loadBalancers[string(lbType)]; exists {
		return lb
	}
	return r.loadBalancers[string(LoadBalancerTypeRoundRobin)]
}

// updateRuntimeStatus 更新运行时状态
func (r *router) updateRuntimeStatus(runtime *RuntimeInfo) {
	if runtime.Metrics == nil {
		return
	}

	// 根据指标更新状态
	if runtime.Metrics.ErrorRate > 0.5 {
		runtime.Status = RuntimeStatusUnhealthy
	} else if runtime.Metrics.ErrorRate > 0.2 {
		runtime.Status = RuntimeStatusDegraded
	} else {
		runtime.Status = RuntimeStatusHealthy
	}

	// 检查容量
	if runtime.Capacity != nil {
		loadRatio := float64(runtime.Capacity.CurrentLoad) / float64(runtime.Capacity.MaxConcurrentRequests)
		if loadRatio > 0.95 {
			runtime.Status = RuntimeStatusDegraded
		}
	}
}

// 健康检查器方法
func (h *healthChecker) start() {
	defer h.router.wg.Done()

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.checkAll()
		case <-h.router.closeChan:
			return
		}
	}
}

func (h *healthChecker) checkAll() {
	h.router.runtimesMu.RLock()
	runtimes := make([]*RuntimeInfo, 0, len(h.router.runtimes))
	for _, runtime := range h.router.runtimes {
		runtimes = append(runtimes, runtime)
	}
	h.router.runtimesMu.RUnlock()

	for _, runtime := range runtimes {
		h.check(runtime)
	}
}

func (h *healthChecker) check(runtime *RuntimeInfo) {
	// 实现健康检查逻辑
	// 这里简化处理，实际应该发送健康检查请求到运行时端点

	runtime.LastHealthCheck = time.Now()

	// 基于指标更新状态
	h.router.updateRuntimeStatus(runtime)

	h.logger.Debug("health check completed",
		"runtime_id", runtime.ID,
		"status", runtime.Status,
		"health_score", h.router.calculateHealthScore(runtime))
}

// 路由策略实现

// roundRobinStrategy 轮询策略
type roundRobinStrategy struct {
	counter uint64
	mu      sync.Mutex
}

func (s *roundRobinStrategy) Select(ctx context.Context, candidates []*RuntimeInfo, req *RouteRequest) (*RuntimeInfo, error) {
	if len(candidates) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no candidates")
	}

	s.mu.Lock()
	index := int(s.counter % uint64(len(candidates)))
	s.counter++
	s.mu.Unlock()

	return candidates[index], nil
}

// leastLoadStrategy 最少负载策略
type leastLoadStrategy struct{}

func (s *leastLoadStrategy) Select(ctx context.Context, candidates []*RuntimeInfo, req *RouteRequest) (*RuntimeInfo, error) {
	if len(candidates) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no candidates")
	}

	var selected *RuntimeInfo
	minLoad := -1

	for _, candidate := range candidates {
		if candidate.Capacity == nil {
			continue
		}

		load := candidate.Capacity.CurrentLoad
		if minLoad == -1 || load < minLoad {
			minLoad = load
			selected = candidate
		}
	}

	if selected == nil {
		return candidates[0], nil
	}

	return selected, nil
}

// weightedStrategy 加权策略
type weightedStrategy struct{}

func (s *weightedStrategy) Select(ctx context.Context, candidates []*RuntimeInfo, req *RouteRequest) (*RuntimeInfo, error) {
	if len(candidates) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no candidates")
	}

	// 计算总权重
	totalWeight := 0
	for _, candidate := range candidates {
		totalWeight += candidate.Weight
	}

	if totalWeight == 0 {
		return candidates[0], nil
	}

	// 随机选择
	r := rand.Intn(totalWeight)
	for _, candidate := range candidates {
		r -= candidate.Weight
		if r < 0 {
			return candidate, nil
		}
	}

	return candidates[len(candidates)-1], nil
}

// randomStrategy 随机策略
type randomStrategy struct{}

func (s *randomStrategy) Select(ctx context.Context, candidates []*RuntimeInfo, req *RouteRequest) (*RuntimeInfo, error) {
	if len(candidates) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no candidates")
	}

	index := rand.Intn(len(candidates))
	return candidates[index], nil
}

// hashStrategy 哈希策略
type hashStrategy struct{}

func (s *hashStrategy) Select(ctx context.Context, candidates []*RuntimeInfo, req *RouteRequest) (*RuntimeInfo, error) {
	if len(candidates) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no candidates")
	}

	// 使用会话ID进行哈希
	h := fnv.New32a()
	h.Write([]byte(req.SessionID))
	index := int(h.Sum32()) % len(candidates)

	return candidates[index], nil
}

// latencyStrategy 延迟优先策略
type latencyStrategy struct{}

func (s *latencyStrategy) Select(ctx context.Context, candidates []*RuntimeInfo, req *RouteRequest) (*RuntimeInfo, error) {
	if len(candidates) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no candidates")
	}

	var selected *RuntimeInfo
	var minLatency time.Duration

	for _, candidate := range candidates {
		if candidate.Metrics == nil {
			continue
		}

		latency := candidate.Metrics.AverageLatency
		if selected == nil || latency < minLatency {
			minLatency = latency
			selected = candidate
		}
	}

	if selected == nil {
		return candidates[0], nil
	}

	return selected, nil
}

// costStrategy 成本优先策略
type costStrategy struct{}

func (s *costStrategy) Select(ctx context.Context, candidates []*RuntimeInfo, req *RouteRequest) (*RuntimeInfo, error) {
	if len(candidates) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no candidates")
	}

	// 简化处理，选择第一个
	return candidates[0], nil
}

// qualityStrategy 质量优先策略
type qualityStrategy struct{}

func (s *qualityStrategy) Select(ctx context.Context, candidates []*RuntimeInfo, req *RouteRequest) (*RuntimeInfo, error) {
	if len(candidates) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no candidates")
	}

	var selected *RuntimeInfo
	var maxQuality float64

	for _, candidate := range candidates {
		if candidate.Metadata == nil {
			continue
		}

		if quality, ok := candidate.Metadata["quality_score"].(float64); ok {
			if selected == nil || quality > maxQuality {
				maxQuality = quality
				selected = candidate
			}
		}
	}

	if selected == nil {
		return candidates[0], nil
	}

	return selected, nil
}

// compositeStrategy 组合策略
type compositeStrategy struct {
	config *RouterConfig
}

func (s *compositeStrategy) Select(ctx context.Context, candidates []*RuntimeInfo, req *RouteRequest) (*RuntimeInfo, error) {
	if len(candidates) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no candidates")
	}

	// 综合评分选择
	var selected *RuntimeInfo
	var maxScore float64

	for _, candidate := range candidates {
		score := s.calculateScore(candidate)
		if selected == nil || score > maxScore {
			maxScore = score
			selected = candidate
		}
	}

	return selected, nil
}

func (s *compositeStrategy) calculateScore(runtime *RuntimeInfo) float64 {
	score := 0.0

	// 健康状态 (30%)
	if runtime.Status == RuntimeStatusHealthy {
		score += 0.3
	} else if runtime.Status == RuntimeStatusDegraded {
		score += 0.15
	}

	// 负载 (25%)
	if runtime.Capacity != nil && runtime.Capacity.MaxConcurrentRequests > 0 {
		loadRatio := float64(runtime.Capacity.CurrentLoad) / float64(runtime.Capacity.MaxConcurrentRequests)
		score += (1.0 - loadRatio) * 0.25
	}

	// 错误率 (25%)
	if runtime.Metrics != nil {
		score += (1.0 - runtime.Metrics.ErrorRate) * 0.25
	}

	// 延迟 (20%)
	if runtime.Metrics != nil && runtime.Metrics.AverageLatency > 0 {
		latencyScore := 1.0 - (float64(runtime.Metrics.AverageLatency) / float64(5*time.Second))
		if latencyScore < 0 {
			latencyScore = 0
		}
		score += latencyScore * 0.2
	}

	return score
}

// 负载均衡器实现

// roundRobinLoadBalancer 轮询负载均衡器
type roundRobinLoadBalancer struct {
	counter uint64
	mu      sync.Mutex
}

func (lb *roundRobinLoadBalancer) Next(ctx context.Context, runtimes []*RuntimeInfo) (*RuntimeInfo, error) {
	if len(runtimes) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no runtimes available")
	}

	lb.mu.Lock()
	index := int(lb.counter % uint64(len(runtimes)))
	lb.counter++
	lb.mu.Unlock()

	return runtimes[index], nil
}

// leastConnLoadBalancer 最少连接负载均衡器
type leastConnLoadBalancer struct{}

func (lb *leastConnLoadBalancer) Next(ctx context.Context, runtimes []*RuntimeInfo) (*RuntimeInfo, error) {
	if len(runtimes) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no runtimes available")
	}

	var selected *RuntimeInfo
	minConn := -1

	for _, runtime := range runtimes {
		if runtime.Metrics == nil {
			continue
		}

		conn := runtime.Metrics.ActiveConnections
		if minConn == -1 || conn < minConn {
			minConn = conn
			selected = runtime
		}
	}

	if selected == nil {
		return runtimes[0], nil
	}

	return selected, nil
}

// weightedLoadBalancer 加权负载均衡器
type weightedLoadBalancer struct{}

func (lb *weightedLoadBalancer) Next(ctx context.Context, runtimes []*RuntimeInfo) (*RuntimeInfo, error) {
	if len(runtimes) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no runtimes available")
	}

	totalWeight := 0
	for _, runtime := range runtimes {
		totalWeight += runtime.Weight
	}

	if totalWeight == 0 {
		return runtimes[0], nil
	}

	r := rand.Intn(totalWeight)
	for _, runtime := range runtimes {
		r -= runtime.Weight
		if r < 0 {
			return runtime, nil
		}
	}

	return runtimes[len(runtimes)-1], nil
}

// randomLoadBalancer 随机负载均衡器
type randomLoadBalancer struct{}

func (lb *randomLoadBalancer) Next(ctx context.Context, runtimes []*RuntimeInfo) (*RuntimeInfo, error) {
	if len(runtimes) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no runtimes available")
	}

	index := rand.Intn(len(runtimes))
	return runtimes[index], nil
}

// ipHashLoadBalancer IP哈希负载均衡器
type ipHashLoadBalancer struct{}

func (lb *ipHashLoadBalancer) Next(ctx context.Context, runtimes []*RuntimeInfo) (*RuntimeInfo, error) {
	if len(runtimes) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no runtimes available")
	}

	// 从上下文获取IP（简化处理）
	h := fnv.New32a()
	h.Write([]byte("default-ip"))
	index := int(h.Sum32()) % len(runtimes)

	return runtimes[index], nil
}

//Personal.AI order the ending
