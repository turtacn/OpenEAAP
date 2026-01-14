package runtime

import (
	"context"
	"time"
)

// Manager 运行时管理器接口
type Manager interface {
	// RegisterRuntime 注册运行时
	RegisterRuntime(runtime Runtime) error

	// UnregisterRuntime 注销运行时
	UnregisterRuntime(runtimeID string) error

	// GetRuntime 获取运行时
	GetRuntime(runtimeID string) (Runtime, error)

	// ListRuntimes 列出所有运行时
	ListRuntimes() ([]Runtime, error)

	// Execute 执行任务
	Execute(ctx context.Context, runtimeID string, req *ExecuteRequest) (*ExecuteResponse, error)

	// ExecuteStream 流式执行任务
	ExecuteStream(ctx context.Context, runtimeID string, req *ExecuteRequest) (<-chan *StreamChunk, <-chan error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context, runtimeID string) (*HealthCheckResult, error)

	// GetMetrics 获取运行时指标
	GetMetrics(ctx context.Context, runtimeID string) (*RuntimeMetrics, error)

	// Shutdown 关闭管理器
	Shutdown(ctx context.Context) error
}

// Runtime 运行时接口
type Runtime interface {
	// ID 获取运行时ID
	ID() string

	// Type 获取运行时类型
	Type() RuntimeType

	// Name 获取运行时名称
	Name() string

	// Version 获取运行时版本
	Version() string

	// Initialize 初始化运行时
	Initialize(ctx context.Context, config *RuntimeConfig) error

	// Execute 执行任务
	Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error)

	// ExecuteStream 流式执行任务
	ExecuteStream(ctx context.Context, req *ExecuteRequest) (<-chan *StreamChunk, <-chan error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) (*HealthCheckResult, error)

	// Metadata 获取运行时元数据
	Metadata() *RuntimeMetadata

	// Shutdown 关闭运行时
	Shutdown(ctx context.Context) error

	// IsReady 检查是否就绪
	IsReady() bool

	// GetStatus 获取运行时状态
	GetStatus() RuntimeStatus

	// UpdateConfig 更新配置
	UpdateConfig(ctx context.Context, config *RuntimeConfig) error
}

// PluginRuntime 插件运行时接口
type PluginRuntime interface {
	Runtime

	// LoadPlugin 加载插件
	LoadPlugin(ctx context.Context, plugin *Plugin) error

	// UnloadPlugin 卸载插件
	UnloadPlugin(ctx context.Context, pluginID string) error

	// ListPlugins 列出所有插件
	ListPlugins() ([]*Plugin, error)

	// GetPlugin 获取插件
	GetPlugin(pluginID string) (*Plugin, error)

	// ExecutePlugin 执行插件
	ExecutePlugin(ctx context.Context, pluginID string, req *PluginExecuteRequest) (*PluginExecuteResponse, error)

	// ValidatePlugin 验证插件
	ValidatePlugin(ctx context.Context, plugin *Plugin) error
}

// RuntimeType 运行时类型
type RuntimeType string

const (
	RuntimeTypeNative    RuntimeType = "native"     // 原生运行时
	RuntimeTypeLangChain RuntimeType = "langchain"  // LangChain运行时
	RuntimeTypeAutoGPT   RuntimeType = "autogpt"    // AutoGPT运行时
	RuntimeTypeCrewAI    RuntimeType = "crewai"     // CrewAI运行时
	RuntimeTypeCustom    RuntimeType = "custom"     // 自定义运行时
	RuntimeTypePlugin    RuntimeType = "plugin"     // 插件运行时
)

// RuntimeStatus 运行时状态
type RuntimeStatus string

const (
	RuntimeStatusInitializing RuntimeStatus = "initializing" // 初始化中
	RuntimeStatusReady        RuntimeStatus = "ready"        // 就绪
	RuntimeStatusBusy         RuntimeStatus = "busy"         // 繁忙
	RuntimeStatusError        RuntimeStatus = "error"        // 错误
	RuntimeStatusShutdown     RuntimeStatus = "shutdown"     // 已关闭
	RuntimeStatusMaintenance  RuntimeStatus = "maintenance"  // 维护中
)

// RuntimeConfig 运行时配置
type RuntimeConfig struct {
	ID                  string                 // 运行时ID
	Type                RuntimeType            // 运行时类型
	Name                string                 // 名称
	Version             string                 // 版本
	Endpoint            string                 // 端点
	Timeout             time.Duration          // 超时时间
	MaxConcurrency      int                    // 最大并发数
	EnableStreaming     bool                   // 启用流式输出
	EnableHealthCheck   bool                   // 启用健康检查
	HealthCheckInterval time.Duration          // 健康检查间隔
	RetryPolicy         *RetryPolicy           // 重试策略
	ResourceLimits      *ResourceLimits        // 资源限制
	Plugins             []*PluginConfig        // 插件配置列表
	Environment         map[string]string      // 环境变量
	Metadata            map[string]interface{} // 元数据
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries      int           // 最大重试次数
	InitialInterval time.Duration // 初始重试间隔
	MaxInterval     time.Duration // 最大重试间隔
	Multiplier      float64       // 退避倍数
	RandomFactor    float64       // 随机因子
}

// ResourceLimits 资源限制
type ResourceLimits struct {
	MaxCPU        float64 // 最大CPU使用率（核心数）
	MaxMemory     int64   // 最大内存使用（字节）
	MaxDisk       int64   // 最大磁盘使用（字节）
	MaxGoroutines int     // 最大协程数
	MaxConnections int    // 最大连接数
}

// PluginConfig 插件配置
type PluginConfig struct {
	ID          string                 // 插件ID
	Name        string                 // 插件名称
	Version     string                 // 插件版本
	Type        PluginType             // 插件类型
	Enabled     bool                   // 是否启用
	Path        string                 // 插件路径
	EntryPoint  string                 // 入口点
	Config      map[string]interface{} // 插件配置
	Dependencies []string              // 依赖项
	Permissions []string               // 权限列表
	Metadata    map[string]interface{} // 元数据
}

// PluginType 插件类型
type PluginType string

const (
	PluginTypeTool       PluginType = "tool"       // 工具插件
	PluginTypeMemory     PluginType = "memory"     // 记忆插件
	PluginTypeRetriever  PluginType = "retriever"  // 检索插件
	PluginTypeParser     PluginType = "parser"     // 解析器插件
	PluginTypeFormatter  PluginType = "formatter"  // 格式化器插件
	PluginTypeValidator  PluginType = "validator"  // 验证器插件
	PluginTypeMiddleware PluginType = "middleware" // 中间件插件
	PluginTypeCustom     PluginType = "custom"     // 自定义插件
)

// ExecuteRequest 执行请求
type ExecuteRequest struct {
	AgentID     string                 // Agent ID
	Input       string                 // 输入
	Context     map[string]interface{} // 上下文
	Options     *ExecuteOptions        // 执行选项
	Metadata    map[string]interface{} // 元数据
}

// ExecuteOptions 执行选项
type ExecuteOptions struct {
	Timeout          time.Duration          // 超时时间
	MaxTokens        int                    // 最大令牌数
	Temperature      float64                // 温度
	TopP             float64                // Top-P
	EnableStreaming  bool                   // 启用流式输出
	EnablePlugins    bool                   // 启用插件
	Plugins          []string               // 启用的插件列表
	StopSequences    []string               // 停止序列
	PresencePenalty  float64                // 存在惩罚
	FrequencyPenalty float64                // 频率惩罚
	LogitBias        map[string]float64     // Logit偏差
	User             string                 // 用户标识
	Metadata         map[string]interface{} // 元数据
}

// ExecuteResponse 执行响应
type ExecuteResponse struct {
	Output        string                 // 输出
	Model         string                 // 使用的模型
	TokensUsed    int                    // 使用的令牌数
	InputTokens   int                    // 输入令牌数
	OutputTokens  int                    // 输出令牌数
	FinishReason  string                 // 完成原因
	Duration      time.Duration          // 执行时长
	PluginResults []*PluginExecuteResponse // 插件执行结果
	Metadata      map[string]interface{} // 元数据
	Error         error                  // 错误
}

// StreamChunk 流式数据块
type StreamChunk struct {
	Content      string                 // 内容
	Delta        string                 // 增量内容
	TokensUsed   int                    // 已使用令牌数
	Finished     bool                   // 是否完成
	FinishReason string                 // 完成原因
	Metadata     map[string]interface{} // 元数据
	Error        error                  // 错误
	Timestamp    time.Time              // 时间戳
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	Status       HealthStatus           // 健康状态
	Message      string                 // 消息
	Details      map[string]interface{} // 详细信息
	Timestamp    time.Time              // 时间戳
	ResponseTime time.Duration          // 响应时间
	Checks       []*ComponentHealth     // 组件健康状态
}

// HealthStatus 健康状态
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"   // 健康
	HealthStatusDegraded  HealthStatus = "degraded"  // 降级
	HealthStatusUnhealthy HealthStatus = "unhealthy" // 不健康
	HealthStatusUnknown   HealthStatus = "unknown"   // 未知
)

// ComponentHealth 组件健康状态
type ComponentHealth struct {
	Name         string                 // 组件名称
	Status       HealthStatus           // 健康状态
	Message      string                 // 消息
	Details      map[string]interface{} // 详细信息
	Timestamp    time.Time              // 时间戳
	ResponseTime time.Duration          // 响应时间
}

// RuntimeMetadata 运行时元数据
type RuntimeMetadata struct {
	ID              string                 // 运行时ID
	Type            RuntimeType            // 运行时类型
	Name            string                 // 名称
	Version         string                 // 版本
	Description     string                 // 描述
	Author          string                 // 作者
	Homepage        string                 // 主页
	Repository      string                 // 仓库地址
	License         string                 // 许可证
	Tags            []string               // 标签
	Capabilities    []string               // 能力列表
	SupportedModels []string               // 支持的模型列表
	SupportedPlugins []string              // 支持的插件类型列表
	Features        *RuntimeFeatures       // 特性
	Requirements    *RuntimeRequirements   // 要求
	Limits          *RuntimeLimits         // 限制
	CreatedAt       time.Time              // 创建时间
	UpdatedAt       time.Time              // 更新时间
	Metadata        map[string]interface{} // 额外元数据
}

// RuntimeFeatures 运行时特性
type RuntimeFeatures struct {
	SupportsStreaming     bool     // 支持流式输出
	SupportsPlugins       bool     // 支持插件
	SupportsMemory        bool     // 支持记忆
	SupportsRetrieval     bool     // 支持检索
	SupportsFunctionCall  bool     // 支持函数调用
	SupportsVision        bool     // 支持视觉
	SupportsAudio         bool     // 支持音频
	SupportsMultimodal    bool     // 支持多模态
	SupportsAsync         bool     // 支持异步执行
	SupportsBatch         bool     // 支持批处理
	Languages             []string // 支持的编程语言
}

// RuntimeRequirements 运行时要求
type RuntimeRequirements struct {
	MinCPU        float64 // 最小CPU（核心数）
	MinMemory     int64   // 最小内存（字节）
	MinDisk       int64   // 最小磁盘空间（字节）
	Dependencies  []string // 依赖项
	Environment   []string // 环境变量要求
	Permissions   []string // 所需权限
}

// RuntimeLimits 运行时限制
type RuntimeLimits struct {
	MaxConcurrentRequests int           // 最大并发请求数
	MaxRequestSize        int64         // 最大请求大小（字节）
	MaxResponseSize       int64         // 最大响应大小（字节）
	MaxExecutionTime      time.Duration // 最大执行时间
	MaxTokensPerRequest   int           // 每个请求最大令牌数
	RateLimit             int           // 速率限制（请求/秒）
}

// RuntimeMetrics 运行时指标
type RuntimeMetrics struct {
	TotalRequests        int64         // 总请求数
	SuccessfulRequests   int64         // 成功请求数
	FailedRequests       int64         // 失败请求数
	ActiveRequests       int64         // 活跃请求数
	AverageResponseTime  time.Duration // 平均响应时间
	P50ResponseTime      time.Duration // P50响应时间
	P95ResponseTime      time.Duration // P95响应时间
	P99ResponseTime      time.Duration // P99响应时间
	TotalTokensProcessed int64         // 总处理令牌数
	AverageTokensPerReq  int           // 平均每请求令牌数
	TotalCost            float64       // 总成本
	ErrorRate            float64       // 错误率
	Uptime               time.Duration // 运行时间
	CPUUsage             float64       // CPU使用率
	MemoryUsage          int64         // 内存使用量（字节）
	DiskUsage            int64         // 磁盘使用量（字节）
	GoroutineCount       int           // 协程数量
	ConnectionCount      int           // 连接数量
	LastRequestTime      time.Time     // 最后请求时间
	Timestamp            time.Time     // 时间戳
}

// Plugin 插件
type Plugin struct {
	ID           string                 // 插件ID
	Name         string                 // 插件名称
	Version      string                 // 版本
	Type         PluginType             // 插件类型
	Description  string                 // 描述
	Author       string                 // 作者
	Homepage     string                 // 主页
	Repository   string                 // 仓库地址
	License      string                 // 许可证
	Enabled      bool                   // 是否启用
	Path         string                 // 插件路径
	EntryPoint   string                 // 入口点
	Config       map[string]interface{} // 配置
	Dependencies []string               // 依赖项
	Permissions  []string               // 权限列表
	Capabilities []string               // 能力列表
	Tags         []string               // 标签
	Schema       *PluginSchema          // 插件模式
	Status       PluginStatus           // 状态
	CreatedAt    time.Time              // 创建时间
	UpdatedAt    time.Time              // 更新时间
	Metadata     map[string]interface{} // 元数据
}

// PluginStatus 插件状态
type PluginStatus string

const (
	PluginStatusLoaded   PluginStatus = "loaded"   // 已加载
	PluginStatusActive   PluginStatus = "active"   // 活跃
	PluginStatusInactive PluginStatus = "inactive" // 不活跃
	PluginStatusError    PluginStatus = "error"    // 错误
	PluginStatusUnloaded PluginStatus = "unloaded" // 已卸载
)

// PluginSchema 插件模式
type PluginSchema struct {
	InputSchema  map[string]interface{} // 输入模式
	OutputSchema map[string]interface{} // 输出模式
	ConfigSchema map[string]interface{} // 配置模式
	Examples     []PluginExample        // 示例列表
}

// PluginExample 插件示例
type PluginExample struct {
	Name        string                 // 示例名称
	Description string                 // 描述
	Input       map[string]interface{} // 输入示例
	Output      map[string]interface{} // 输出示例
	Config      map[string]interface{} // 配置示例
}

// PluginExecuteRequest 插件执行请求
type PluginExecuteRequest struct {
	PluginID string                 // 插件ID
	Input    map[string]interface{} // 输入
	Config   map[string]interface{} // 配置
	Context  map[string]interface{} // 上下文
	Metadata map[string]interface{} // 元数据
}

// PluginExecuteResponse 插件执行响应
type PluginExecuteResponse struct {
	PluginID  string                 // 插件ID
	Output    map[string]interface{} // 输出
	Duration  time.Duration          // 执行时长
	Metadata  map[string]interface{} // 元数据
	Error     error                  // 错误
	Timestamp time.Time              // 时间戳
}

// RuntimeEvent 运行时事件
type RuntimeEvent struct {
	Type      RuntimeEventType       // 事件类型
	RuntimeID string                 // 运行时ID
	Message   string                 // 消息
	Data      map[string]interface{} // 数据
	Timestamp time.Time              // 时间戳
}

// RuntimeEventType 运行时事件类型
type RuntimeEventType string

const (
	RuntimeEventTypeStarted      RuntimeEventType = "started"       // 已启动
	RuntimeEventTypeStopped      RuntimeEventType = "stopped"       // 已停止
	RuntimeEventTypeError        RuntimeEventType = "error"         // 错误
	RuntimeEventTypeHealthChange RuntimeEventType = "health_change" // 健康状态变更
	RuntimeEventTypeConfigChange RuntimeEventType = "config_change" // 配置变更
	RuntimeEventTypePluginLoaded RuntimeEventType = "plugin_loaded" // 插件已加载
	RuntimeEventTypePluginUnloaded RuntimeEventType = "plugin_unloaded" // 插件已卸载
	RuntimeEventTypeMetricsUpdate RuntimeEventType = "metrics_update" // 指标更新
)

// RuntimeEventHandler 运行时事件处理器
type RuntimeEventHandler interface {
	// HandleEvent 处理事件
	HandleEvent(event *RuntimeEvent) error
}

// RuntimeFactory 运行时工厂接口
type RuntimeFactory interface {
	// CreateRuntime 创建运行时
	CreateRuntime(config *RuntimeConfig) (Runtime, error)

	// SupportedTypes 支持的运行时类型
	SupportedTypes() []RuntimeType

	// ValidateConfig 验证配置
	ValidateConfig(config *RuntimeConfig) error
}

//Personal.AI order the ending
