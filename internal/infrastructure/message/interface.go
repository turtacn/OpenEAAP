package message

import (
	"context"
	"time"
)

// MessageQueue 消息队列接口
// 抽象消息队列操作，支持多种消息队列实现
type MessageQueue interface {
	// Producer 生产者操作
	Publish(ctx context.Context, req *PublishRequest) (*PublishResponse, error)
	PublishBatch(ctx context.Context, req *PublishBatchRequest) (*PublishBatchResponse, error)
	PublishAsync(ctx context.Context, req *PublishRequest, callback PublishCallback) error
	PublishDelayed(ctx context.Context, req *PublishDelayedRequest) (*PublishResponse, error)
	PublishScheduled(ctx context.Context, req *PublishScheduledRequest) (*PublishResponse, error)

	// Consumer 消费者操作
	Subscribe(ctx context.Context, req *SubscribeRequest) (Subscription, error)
	Unsubscribe(ctx context.Context, subscriptionID string) error
	CreateConsumerGroup(ctx context.Context, config *ConsumerGroupConfig) (ConsumerGroup, error)
	DeleteConsumerGroup(ctx context.Context, groupID string) error

	// Queue 队列操作
	CreateQueue(ctx context.Context, config *QueueConfig) error
	DeleteQueue(ctx context.Context, name string) error
	PurgeQueue(ctx context.Context, name string) error
	QueueExists(ctx context.Context, name string) (bool, error)
	GetQueueInfo(ctx context.Context, name string) (*QueueInfo, error)
	ListQueues(ctx context.Context) ([]*QueueInfo, error)

	// Topic 主题操作
	CreateTopic(ctx context.Context, config *TopicConfig) error
	DeleteTopic(ctx context.Context, name string) error
	TopicExists(ctx context.Context, name string) (bool, error)
	GetTopicInfo(ctx context.Context, name string) (*TopicInfo, error)
	ListTopics(ctx context.Context) ([]string, error)

	// Exchange 交换机操作（仅 RabbitMQ）
	CreateExchange(ctx context.Context, config *ExchangeConfig) error
	DeleteExchange(ctx context.Context, name string) error
	BindQueue(ctx context.Context, req *BindQueueRequest) error
	UnbindQueue(ctx context.Context, req *UnbindQueueRequest) error

	// Message 消息操作
	GetMessage(ctx context.Context, queue string, timeout time.Duration) (*Message, error)
	AckMessage(ctx context.Context, req *AckMessageRequest) error
	NackMessage(ctx context.Context, req *NackMessageRequest) error
	RejectMessage(ctx context.Context, req *RejectMessageRequest) error
	RequeuMessage(ctx context.Context, req *RequeueMessageRequest) error

	// DeadLetter 死信队列操作
	CreateDeadLetterQueue(ctx context.Context, config *DeadLetterConfig) error
	GetDeadLetterMessages(ctx context.Context, queue string, limit int) ([]*Message, error)
	RedriveDeadLetterMessage(ctx context.Context, req *RedriveMessageRequest) error

	// Priority 优先级队列操作
	CreatePriorityQueue(ctx context.Context, config *PriorityQueueConfig) error
	PublishWithPriority(ctx context.Context, req *PublishPriorityRequest) (*PublishResponse, error)

	// Transaction 事务操作
	BeginTransaction(ctx context.Context) (Transaction, error)

	// Monitoring 监控操作
	GetQueueStats(ctx context.Context, queue string) (*QueueStats, error)
	GetConsumerStats(ctx context.Context, consumerID string) (*ConsumerStats, error)
	GetClusterInfo(ctx context.Context) (*ClusterInfo, error)

	// Health 健康检查
	Ping(ctx context.Context) error
	Close() error
}

// PublishRequest 发布消息请求
type PublishRequest struct {
	Queue       string            // 队列名称
	Topic       string            // 主题名称（Kafka/NATS）
	Exchange    string            // 交换机名称（RabbitMQ）
	RoutingKey  string            // 路由键（RabbitMQ）
	Body        []byte            // 消息体
	Headers     map[string]string // 消息头
	ContentType string            // 内容类型
	MessageID   string            // 消息ID
	Priority    uint8             // 优先级（0-9）
	Expiration  time.Duration     // 过期时间
	Persistent  bool              // 是否持久化
	Mandatory   bool              // 是否强制（RabbitMQ）
	Immediate   bool              // 是否立即（RabbitMQ）
	Metadata    map[string]interface{} // 元数据
}

// PublishResponse 发布消息响应
type PublishResponse struct {
	MessageID   string    // 消息ID
	Queue       string    // 队列
	Topic       string    // 主题
	Partition   int32     // 分区（Kafka）
	Offset      int64     // 偏移量（Kafka）
	PublishedAt time.Time // 发布时间
}

// PublishBatchRequest 批量发布消息请求
type PublishBatchRequest struct {
	Messages []*PublishRequest // 消息列表
}

// PublishBatchResponse 批量发布消息响应
type PublishBatchResponse struct {
	Responses []*PublishResponse // 响应列表
	Errors    []error            // 错误列表
	Succeeded int                // 成功数量
	Failed    int                // 失败数量
}

// PublishCallback 发布回调
type PublishCallback func(response *PublishResponse, err error)

// PublishDelayedRequest 延迟发布消息请求
type PublishDelayedRequest struct {
	PublishRequest
	Delay time.Duration // 延迟时间
}

// PublishScheduledRequest 定时发布消息请求
type PublishScheduledRequest struct {
	PublishRequest
	ScheduleTime time.Time // 定时时间
}

// SubscribeRequest 订阅请求
type SubscribeRequest struct {
	SubscriptionID string          // 订阅ID
	Queue          string          // 队列名称
	Topic          string          // 主题名称
	Exchange       string          // 交换机名称（RabbitMQ）
	RoutingKey     string          // 路由键（RabbitMQ）
	ConsumerTag    string          // 消费者标签（RabbitMQ）
	Handler        MessageHandler  // 消息处理器
	ErrorHandler   ErrorHandler    // 错误处理器
	Config         *SubscribeConfig // 订阅配置
	AutoAck        bool            // 自动确认
	Exclusive      bool            // 独占
	NoLocal        bool            // 不接收本地消息
	NoWait         bool            // 不等待
	Args           map[string]interface{} // 参数
}

// SubscribeConfig 订阅配置
type SubscribeConfig struct {
	PrefetchCount    int           // 预取数量
	PrefetchSize     int           // 预取大小
	MaxRetries       int           // 最大重试次数
	RetryDelay       time.Duration // 重试延迟
	AckTimeout       time.Duration // 确认超时
	ProcessTimeout   time.Duration // 处理超时
	ConcurrentLimit  int           // 并发限制
	EnableDeadLetter bool          // 启用死信队列
	DeadLetterQueue  string        // 死信队列名称
}

// MessageHandler 消息处理器
type MessageHandler func(ctx context.Context, message *Message) error

// ErrorHandler 错误处理器
type ErrorHandler func(err error)

// Message 消息
type Message struct {
	ID            string                 // 消息ID
	Queue         string                 // 队列
	Topic         string                 // 主题
	Exchange      string                 // 交换机
	RoutingKey    string                 // 路由键
	Body          []byte                 // 消息体
	Headers       map[string]string      // 消息头
	ContentType   string                 // 内容类型
	Priority      uint8                  // 优先级
	Timestamp     time.Time              // 时间戳
	Expiration    time.Duration          // 过期时间
	DeliveryTag   uint64                 // 投递标签（RabbitMQ）
	Redelivered   bool                   // 是否重新投递
	DeliveryCount int                    // 投递次数
	Partition     int32                  // 分区（Kafka）
	Offset        int64                  // 偏移量（Kafka）
	Metadata      map[string]interface{} // 元数据
	Acknowledger  Acknowledger           // 确认器
}

// Acknowledger 消息确认器
type Acknowledger interface {
	Ack() error
	Nack(requeue bool) error
	Reject(requeue bool) error
}

// Subscription 订阅接口
type Subscription interface {
	Start(ctx context.Context) error
	Stop() error
	Pause() error
	Resume() error
	GetStats() *SubscriptionStats
	GetID() string
}

// SubscriptionStats 订阅统计
type SubscriptionStats struct {
	SubscriptionID   string        // 订阅ID
	Queue            string        // 队列
	MessagesReceived int64         // 接收的消息数
	MessagesAcked    int64         // 确认的消息数
	MessagesNacked   int64         // 拒绝的消息数
	MessagesFailed   int64         // 失败的消息数
	LastMessageAt    time.Time     // 最后消息时间
	ProcessingTime   time.Duration // 平均处理时间
	IsActive         bool          // 是否活跃
}

// ConsumerGroupConfig 消费者组配置
type ConsumerGroupConfig struct {
	GroupID           string          // 组ID
	Queues            []string        // 队列列表
	Topics            []string        // 主题列表
	Handler           MessageHandler  // 消息处理器
	ErrorHandler      ErrorHandler    // 错误处理器
	Config            *SubscribeConfig // 订阅配置
	RebalanceStrategy RebalanceStrategy // 重平衡策略
	SessionTimeout    time.Duration   // 会话超时
	HeartbeatInterval time.Duration   // 心跳间隔
}

// RebalanceStrategy 重平衡策略
type RebalanceStrategy string

const (
	RebalanceStrategyRange      RebalanceStrategy = "range"      // 范围
	RebalanceStrategyRoundRobin RebalanceStrategy = "roundrobin" // 轮询
	RebalanceStrategySticky     RebalanceStrategy = "sticky"     // 粘性
)

// ConsumerGroup 消费者组接口
type ConsumerGroup interface {
	Start(ctx context.Context) error
	Stop() error
	Pause() error
	Resume() error
	GetMembers() []*ConsumerMember
	GetStats() *ConsumerGroupStats
}

// ConsumerMember 消费者成员
type ConsumerMember struct {
	MemberID   string   // 成员ID
	ClientID   string   // 客户端ID
	Host       string   // 主机
	Partitions []int32  // 分区列表
	Topics     []string // 主题列表
}

// ConsumerGroupStats 消费者组统计
type ConsumerGroupStats struct {
	GroupID          string    // 组ID
	MemberCount      int       // 成员数量
	MessagesConsumed int64     // 消费的消息数
	Lag              int64     // 延迟
	State            string    // 状态
	LastRebalance    time.Time // 最后重平衡时间
}

// QueueConfig 队列配置
type QueueConfig struct {
	Name        string            // 队列名称
	Durable     bool              // 是否持久化
	AutoDelete  bool              // 是否自动删除
	Exclusive   bool              // 是否独占
	NoWait      bool              // 是否等待
	Args        map[string]interface{} // 参数
	MaxLength   int               // 最大长度
	MaxBytes    int64             // 最大字节数
	MessageTTL  time.Duration     // 消息TTL
	Expires     time.Duration     // 队列过期时间
	DeadLetter  *DeadLetterConfig // 死信配置
	Priority    int               // 优先级
}

// QueueInfo 队列信息
type QueueInfo struct {
	Name              string                 // 队列名称
	Messages          int                    // 消息数量
	MessagesReady     int                    // 就绪消息数
	MessagesUnacked   int                    // 未确认消息数
	Consumers         int                    // 消费者数量
	Durable           bool                   // 是否持久化
	AutoDelete        bool                   // 是否自动删除
	Exclusive         bool                   // 是否独占
	Args              map[string]interface{} // 参数
	CreatedAt         time.Time              // 创建时间
	LastActivityAt    time.Time              // 最后活动时间
}

// TopicConfig 主题配置
type TopicConfig struct {
	Name              string            // 主题名称
	Partitions        int               // 分区数
	ReplicationFactor int               // 复制因子
	Config            map[string]string // 配置
	RetentionTime     time.Duration     // 保留时间
	RetentionBytes    int64             // 保留字节数
	SegmentBytes      int64             // 段字节数
	MaxMessageBytes   int               // 最大消息字节数
}

// TopicInfo 主题信息
type TopicInfo struct {
	Name       string           // 主题名称
	Partitions []*PartitionInfo // 分区信息
	Config     map[string]string // 配置
	CreatedAt  time.Time        // 创建时间
}

// PartitionInfo 分区信息
type PartitionInfo struct {
	ID       int32   // 分区ID
	Leader   int32   // 领导者
	Replicas []int32 // 副本列表
	ISR      []int32 // 同步副本列表
	Offset   int64   // 偏移量
	Lag      int64   // 延迟
}

// ExchangeConfig 交换机配置（RabbitMQ）
type ExchangeConfig struct {
	Name       string                 // 交换机名称
	Type       ExchangeType           // 交换机类型
	Durable    bool                   // 是否持久化
	AutoDelete bool                   // 是否自动删除
	Internal   bool                   // 是否内部
	NoWait     bool                   // 是否等待
	Args       map[string]interface{} // 参数
}

// ExchangeType 交换机类型
type ExchangeType string

const (
	ExchangeTypeDirect  ExchangeType = "direct"  // 直接
	ExchangeTypeFanout  ExchangeType = "fanout"  // 扇出
	ExchangeTypeTopic   ExchangeType = "topic"   // 主题
	ExchangeTypeHeaders ExchangeType = "headers" // 头部
)

// BindQueueRequest 绑定队列请求
type BindQueueRequest struct {
	Queue      string                 // 队列名称
	Exchange   string                 // 交换机名称
	RoutingKey string                 // 路由键
	NoWait     bool                   // 是否等待
	Args       map[string]interface{} // 参数
}

// UnbindQueueRequest 解绑队列请求
type UnbindQueueRequest struct {
	Queue      string                 // 队列名称
	Exchange   string                 // 交换机名称
	RoutingKey string                 // 路由键
	Args       map[string]interface{} // 参数
}

// AckMessageRequest 确认消息请求
type AckMessageRequest struct {
	Queue       string // 队列
	DeliveryTag uint64 // 投递标签
	MessageID   string // 消息ID
	Multiple    bool   // 是否批量
}

// NackMessageRequest 拒绝消息请求
type NackMessageRequest struct {
	Queue       string // 队列
	DeliveryTag uint64 // 投递标签
	MessageID   string // 消息ID
	Multiple    bool   // 是否批量
	Requeue     bool   // 是否重新入队
}

// RejectMessageRequest 拒绝消息请求
type RejectMessageRequest struct {
	Queue       string // 队列
	DeliveryTag uint64 // 投递标签
	MessageID   string // 消息ID
	Requeue     bool   // 是否重新入队
}

// RequeueMessageRequest 重新入队消息请求
type RequeueMessageRequest struct {
	Queue       string        // 队列
	DeliveryTag uint64        // 投递标签
	MessageID   string        // 消息ID
	Delay       time.Duration // 延迟时间
}

// DeadLetterConfig 死信配置
type DeadLetterConfig struct {
	Enabled         bool          // 是否启用
	Queue           string        // 死信队列
	Exchange        string        // 死信交换机
	RoutingKey      string        // 死信路由键
	MaxRetries      int           // 最大重试次数
	RetryDelay      time.Duration // 重试延迟
	TTL             time.Duration // TTL
}

// RedriveMessageRequest 重新驱动消息请求
type RedriveMessageRequest struct {
	SourceQueue      string // 源队列
	DestinationQueue string // 目标队列
	MessageID        string // 消息ID
	MaxCount         int    // 最大数量
}

// PriorityQueueConfig 优先级队列配置
type PriorityQueueConfig struct {
	QueueConfig
	MaxPriority uint8 // 最大优先级
}

// PublishPriorityRequest 优先级发布请求
type PublishPriorityRequest struct {
	PublishRequest
	Priority uint8 // 优先级（0-9）
}

// Transaction 事务接口
type Transaction interface {
	Publish(ctx context.Context, req *PublishRequest) error
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// QueueStats 队列统计
type QueueStats struct {
	Name                    string        // 队列名称
	Messages                int           // 消息数量
	MessagesReady           int           // 就绪消息数
	MessagesUnacked         int           // 未确认消息数
	MessagesRate            float64       // 消息速率（消息/秒）
	PublishRate             float64       // 发布速率（消息/秒）
	DeliverRate             float64       // 投递速率（消息/秒）
	AckRate                 float64       // 确认速率（消息/秒）
	Consumers               int           // 消费者数量
	ConsumerUtilization     float64       // 消费者利用率
	MemoryUsed              int64         // 内存使用（字节）
	AvgMessageSize          int64         // 平均消息大小
	OldestMessageAge        time.Duration // 最旧消息年龄
	LastActivityAt          time.Time     // 最后活动时间
}

// ConsumerStats 消费者统计
type ConsumerStats struct {
	ConsumerID       string        // 消费者ID
	Queue            string        // 队列
	MessagesConsumed int64         // 消费的消息数
	MessagesAcked    int64         // 确认的消息数
	MessagesNacked   int64         // 拒绝的消息数
	MessagesFailed   int64         // 失败的消息数
	ProcessingTime   time.Duration // 平均处理时间
	LastMessageAt    time.Time     // 最后消息时间
	IsActive         bool          // 是否活跃
}

// ClusterInfo 集群信息
type ClusterInfo struct {
	ClusterID   string        // 集群ID
	Nodes       []*NodeInfo   // 节点信息
	TotalQueues int           // 总队列数
	TotalTopics int           // 总主题数
	Status      ClusterStatus // 状态
	Version     string        // 版本
}

// NodeInfo 节点信息
type NodeInfo struct {
	ID          string    // 节点ID
	Host        string    // 主机
	Port        int       // 端口
	Status      string    // 状态
	IsLeader    bool      // 是否领导者
	Uptime      time.Duration // 运行时间
	MemoryUsed  int64     // 内存使用
	CPUUsage    float64   // CPU使用率
}

// ClusterStatus 集群状态
type ClusterStatus string

const (
	ClusterStatusHealthy   ClusterStatus = "healthy"   // 健康
	ClusterStatusDegraded  ClusterStatus = "degraded"  // 降级
	ClusterStatusUnhealthy ClusterStatus = "unhealthy" // 不健康
)

// MessageQueueFactory 消息队列工厂接口
type MessageQueueFactory interface {
	// Create 创建消息队列实例
	Create(config map[string]interface{}) (MessageQueue, error)
	// Type 返回消息队列类型
	Type() string
}

// MessageQueueType 消息队列类型
type MessageQueueType string

const (
	MessageQueueTypeKafka     MessageQueueType = "kafka"     // Kafka
	MessageQueueTypeRabbitMQ  MessageQueueType = "rabbitmq"  // RabbitMQ
	MessageQueueTypeNATS      MessageQueueType = "nats"      // NATS
	MessageQueueTypeRedis     MessageQueueType = "redis"     // Redis
	MessageQueueTypeAmazonSQS MessageQueueType = "amazonsqs" // Amazon SQS
	MessageQueueTypeGooglePubSub MessageQueueType = "googlepubsub" // Google Pub/Sub
	MessageQueueTypeAzureServiceBus MessageQueueType = "azureservicebus" // Azure Service Bus
)

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries     int           // 最大重试次数
	InitialDelay   time.Duration // 初始延迟
	MaxDelay       time.Duration // 最大延迟
	BackoffFactor  float64       // 退避因子
	RetryableErrors []string     // 可重试错误
}

// MessageFilter 消息过滤器
type MessageFilter struct {
	Headers  map[string]string      // 头部过滤
	Body     []byte                 // 消息体过滤
	Metadata map[string]interface{} // 元数据过滤
	Priority *uint8                 // 优先级过滤
}

// MessageTransformer 消息转换器
type MessageTransformer func(message *Message) (*Message, error)

// MessageValidator 消息验证器
type MessageValidator func(message *Message) error

// MessageInterceptor 消息拦截器
type MessageInterceptor interface {
	OnPublish(ctx context.Context, message *Message) error
	OnConsume(ctx context.Context, message *Message) error
}

//Personal.AI order the ending
