package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/pkg/errors"
)

// KafkaClient Kafka 消息队列客户端接口
type KafkaClient interface {
	// Producer 生产者操作
	SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error)
	SendMessages(ctx context.Context, req *SendMessagesRequest) (*SendMessagesResponse, error)
	SendMessageAsync(ctx context.Context, req *SendMessageRequest, callback SendCallback) error

	// Consumer 消费者操作
	Subscribe(ctx context.Context, req *SubscribeRequest) (Consumer, error)
	Unsubscribe(ctx context.Context, consumerID string) error

	// Topic 主题操作
	CreateTopic(ctx context.Context, config *TopicConfig) error
	DeleteTopic(ctx context.Context, topic string) error
	ListTopics(ctx context.Context) ([]string, error)
	GetTopicMetadata(ctx context.Context, topic string) (*TopicMetadata, error)
	TopicExists(ctx context.Context, topic string) (bool, error)

	// ConsumerGroup 消费者组操作
	CreateConsumerGroup(ctx context.Context, config *ConsumerGroupConfig) (ConsumerGroup, error)
	DeleteConsumerGroup(ctx context.Context, groupID string) error
	ListConsumerGroups(ctx context.Context) ([]string, error)
	GetConsumerGroupInfo(ctx context.Context, groupID string) (*ConsumerGroupInfo, error)

	// Admin 管理操作
	GetClusterMetadata(ctx context.Context) (*ClusterMetadata, error)
	GetBrokers(ctx context.Context) ([]*BrokerInfo, error)

	// Health 健康检查
	Ping(ctx context.Context) error
	Close() error
}

// kafkaClient Kafka 客户端实现
type kafkaClient struct {
	config        *KafkaConfig
	producer      sarama.SyncProducer
	asyncProducer sarama.AsyncProducer
	admin         sarama.ClusterAdmin
	consumers     map[string]Consumer
	consumerGroups map[string]ConsumerGroup
	mu            sync.RWMutex
	logger        logger.Logger
	closed        bool
}

// KafkaConfig Kafka 配置
type KafkaConfig struct {
	Brokers           []string      // Broker 地址列表
	ClientID          string        // 客户端ID
	Version           string        // Kafka 版本
	SASL              *SASLConfig   // SASL 配置
	TLS               *TLSConfig    // TLS 配置
	Timeout           time.Duration // 超时时间
	MaxRetries        int           // 最大重试次数
	RetryBackoff      time.Duration // 重试退避时间
	ProducerConfig    *ProducerConfig    // 生产者配置
	ConsumerConfig    *ConsumerConfig    // 消费者配置
	CompressionCodec  CompressionCodec   // 压缩编解码器
	EnableIdempotence bool          // 启用幂等性
	Logger            logger.Logger // 日志记录器
}

// SASLConfig SASL 配置
type SASLConfig struct {
	Enabled   bool   // 是否启用
	Mechanism string // 机制（PLAIN, SCRAM-SHA-256, SCRAM-SHA-512）
	Username  string // 用户名
	Password  string // 密码
}

// TLSConfig TLS 配置
type TLSConfig struct {
	Enabled            bool   // 是否启用
	InsecureSkipVerify bool   // 跳过验证
	CertFile           string // 证书文件
	KeyFile            string // 密钥文件
	CAFile             string // CA文件
}

// ProducerConfig 生产者配置
type ProducerConfig struct {
	RequiredAcks      RequiredAcks      // 需要的确认
	Compression       CompressionCodec  // 压缩
	MaxMessageBytes   int               // 最大消息字节
	ReturnSuccesses   bool              // 返回成功
	ReturnErrors      bool              // 返回错误
	FlushFrequency    time.Duration     // 刷新频率
	FlushMessages     int               // 刷新消息数
	FlushMaxMessages  int               // 最大刷新消息数
	Partitioner       PartitionerType   // 分区器
}

// ConsumerConfig 消费者配置
type ConsumerConfig struct {
	GroupID                string        // 消费者组ID
	AutoCommit             bool          // 自动提交
	AutoCommitInterval     time.Duration // 自动提交间隔
	InitialOffset          OffsetType    // 初始偏移量
	SessionTimeout         time.Duration // 会话超时
	HeartbeatInterval      time.Duration // 心跳间隔
	RebalanceTimeout       time.Duration // 重平衡超时
	MaxProcessingTime      time.Duration // 最大处理时间
	FetchMin               int32         // 最小抓取
	FetchDefault           int32         // 默认抓取
	MaxWaitTime            time.Duration // 最大等待时间
	IsolationLevel         IsolationLevel // 隔离级别
}

// RequiredAcks 需要的确认
type RequiredAcks int16

const (
	NoResponse     RequiredAcks = 0  // 无响应
	WaitForLocal   RequiredAcks = 1  // 等待本地
	WaitForAll     RequiredAcks = -1 // 等待所有
)

// CompressionCodec 压缩编解码器
type CompressionCodec int8

const (
	CompressionNone   CompressionCodec = 0 // 无压缩
	CompressionGZIP   CompressionCodec = 1 // GZIP
	CompressionSnappy CompressionCodec = 2 // Snappy
	CompressionLZ4    CompressionCodec = 3 // LZ4
	CompressionZSTD   CompressionCodec = 4 // ZSTD
)

// PartitionerType 分区器类型
type PartitionerType string

const (
	PartitionerRandom     PartitionerType = "random"     // 随机
	PartitionerRoundRobin PartitionerType = "roundrobin" // 轮询
	PartitionerHash       PartitionerType = "hash"       // 哈希
	PartitionerManual     PartitionerType = "manual"     // 手动
)

// OffsetType 偏移量类型
type OffsetType int64

const (
	OffsetNewest OffsetType = -1 // 最新
	OffsetOldest OffsetType = -2 // 最旧
)

// IsolationLevel 隔离级别
type IsolationLevel int8

const (
	ReadUncommitted IsolationLevel = 0 // 读未提交
	ReadCommitted   IsolationLevel = 1 // 读已提交
)

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	Topic     string            // 主题
	Key       []byte            // 键
	Value     []byte            // 值
	Headers   map[string]string // 头部
	Partition int32             // 分区（-1表示自动）
	Timestamp *time.Time        // 时间戳
}

// SendMessageResponse 发送消息响应
type SendMessageResponse struct {
	Partition int32     // 分区
	Offset    int64     // 偏移量
	Timestamp time.Time // 时间戳
}

// SendMessagesRequest 批量发送消息请求
type SendMessagesRequest struct {
	Messages []*SendMessageRequest // 消息列表
}

// SendMessagesResponse 批量发送消息响应
type SendMessagesResponse struct {
	Responses []*SendMessageResponse // 响应列表
	Errors    []error                // 错误列表
}

// SendCallback 发送回调
type SendCallback func(response *SendMessageResponse, err error)

// SubscribeRequest 订阅请求
type SubscribeRequest struct {
	Topic         string          // 主题
	ConsumerID    string          // 消费者ID
	GroupID       string          // 消费者组ID（可选）
	Handler       MessageHandler  // 消息处理器
	ErrorHandler  ErrorHandler    // 错误处理器
	Config        *ConsumerConfig // 消费者配置
	FromBeginning bool            // 从头开始
}

// MessageHandler 消息处理器
type MessageHandler func(ctx context.Context, message *Message) error

// ErrorHandler 错误处理器
type ErrorHandler func(err error)

// Message 消息
type Message struct {
	Topic     string            // 主题
	Partition int32             // 分区
	Offset    int64             // 偏移量
	Key       []byte            // 键
	Value     []byte            // 值
	Headers   map[string]string // 头部
	Timestamp time.Time         // 时间戳
}

// Consumer 消费者接口
type Consumer interface {
	Start(ctx context.Context) error
	Stop() error
	Commit(ctx context.Context, offset int64) error
	Pause() error
	Resume() error
	GetOffset() int64
}

// consumer 消费者实现
type consumer struct {
	id           string
	topic        string
	groupID      string
	handler      MessageHandler
	errorHandler ErrorHandler
	config       *ConsumerConfig
	consumer     sarama.Consumer
	partition    sarama.PartitionConsumer
	offset       int64
	paused       bool
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	logger       logger.Logger
}

// TopicConfig 主题配置
type TopicConfig struct {
	Name              string            // 名称
	NumPartitions     int32             // 分区数
	ReplicationFactor int16             // 复制因子
	ConfigEntries     map[string]string // 配置条目
}

// TopicMetadata 主题元数据
type TopicMetadata struct {
	Name       string           // 名称
	Partitions []*PartitionInfo // 分区信息
	Config     map[string]string // 配置
}

// PartitionInfo 分区信息
type PartitionInfo struct {
	ID       int32   // ID
	Leader   int32   // 领导者
	Replicas []int32 // 副本
	ISR      []int32 // 同步副本
}

// ConsumerGroupConfig 消费者组配置
type ConsumerGroupConfig struct {
	GroupID           string          // 组ID
	Topics            []string        // 主题列表
	Handler           MessageHandler  // 消息处理器
	ErrorHandler      ErrorHandler    // 错误处理器
	Config            *ConsumerConfig // 消费者配置
	RebalanceStrategy RebalanceStrategy // 重平衡策略
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
	GetMembers() []*MemberInfo
}

// consumerGroup 消费者组实现
type consumerGroup struct {
	groupID      string
	topics       []string
	handler      MessageHandler
	errorHandler ErrorHandler
	config       *ConsumerGroupConfig
	group        sarama.ConsumerGroup
	paused       bool
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	logger       logger.Logger
}

// ConsumerGroupInfo 消费者组信息
type ConsumerGroupInfo struct {
	GroupID       string        // 组ID
	State         string        // 状态
	Protocol      string        // 协议
	ProtocolType  string        // 协议类型
	Members       []*MemberInfo // 成员信息
	Coordinator   *BrokerInfo   // 协调者
}

// MemberInfo 成员信息
type MemberInfo struct {
	MemberID       string   // 成员ID
	ClientID       string   // 客户端ID
	ClientHost     string   // 客户端主机
	MemberMetadata []byte   // 成员元数据
	MemberAssignment []byte // 成员分配
}

// ClusterMetadata 集群元数据
type ClusterMetadata struct {
	ClusterID      string        // 集群ID
	ControllerID   int32         // 控制器ID
	Brokers        []*BrokerInfo // Broker信息
	Topics         []string      // 主题列表
}

// BrokerInfo Broker信息
type BrokerInfo struct {
	ID   int32  // ID
	Host string // 主机
	Port int32  // 端口
	Rack string // 机架
}

// NewKafkaClient 创建 Kafka 客户端
func NewKafkaClient(config *KafkaConfig) (KafkaClient, error) {
	if config == nil {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "config cannot be nil")
	}
	if len(config.Brokers) == 0 {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "brokers cannot be empty")
	}

	// 设置默认值
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryBackoff == 0 {
		config.RetryBackoff = 100 * time.Millisecond
	}

	// 创建 Sarama 配置
	saramaConfig := sarama.NewConfig()
	saramaConfig.ClientID = config.ClientID
	saramaConfig.Net.DialTimeout = config.Timeout
	saramaConfig.Net.ReadTimeout = config.Timeout
	saramaConfig.Net.WriteTimeout = config.Timeout

	// 设置版本
	if config.Version != "" {
		version, err := sarama.ParseKafkaVersion(config.Version)
		if err != nil {
			return nil, errors.Wrap(err, errors.CodeInvalidParameter, "invalid kafka version")
		}
		saramaConfig.Version = version
	}

	// 配置 SASL
	if config.SASL != nil && config.SASL.Enabled {
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.Mechanism = sarama.SASLMechanism(config.SASL.Mechanism)
		saramaConfig.Net.SASL.User = config.SASL.Username
		saramaConfig.Net.SASL.Password = config.SASL.Password
	}

	// 配置 TLS
	if config.TLS != nil && config.TLS.Enabled {
		saramaConfig.Net.TLS.Enable = true
		// TLS 详细配置可以在这里添加
	}

	// 配置生产者
	if config.ProducerConfig != nil {
		saramaConfig.Producer.RequiredAcks = sarama.RequiredAcks(config.ProducerConfig.RequiredAcks)
		saramaConfig.Producer.Compression = sarama.CompressionCodec(config.ProducerConfig.Compression)
		saramaConfig.Producer.Return.Successes = config.ProducerConfig.ReturnSuccesses
		saramaConfig.Producer.Return.Errors = config.ProducerConfig.ReturnErrors
		saramaConfig.Producer.Retry.Max = config.MaxRetries
		saramaConfig.Producer.Retry.Backoff = config.RetryBackoff

		if config.ProducerConfig.MaxMessageBytes > 0 {
			saramaConfig.Producer.MaxMessageBytes = config.ProducerConfig.MaxMessageBytes
		}

		if config.EnableIdempotence {
			saramaConfig.Producer.Idempotent = true
			saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
			saramaConfig.Net.MaxOpenRequests = 1
		}
	}

	// 配置消费者
	if config.ConsumerConfig != nil {
		saramaConfig.Consumer.Return.Errors = true
		saramaConfig.Consumer.Offsets.AutoCommit.Enable = config.ConsumerConfig.AutoCommit
		saramaConfig.Consumer.Offsets.AutoCommit.Interval = config.ConsumerConfig.AutoCommitInterval
		saramaConfig.Consumer.Offsets.Initial = int64(config.ConsumerConfig.InitialOffset)
		saramaConfig.Consumer.Group.Session.Timeout = config.ConsumerConfig.SessionTimeout
		saramaConfig.Consumer.Group.Heartbeat.Interval = config.ConsumerConfig.HeartbeatInterval
		saramaConfig.Consumer.Group.Rebalance.Timeout = config.ConsumerConfig.RebalanceTimeout
		saramaConfig.Consumer.MaxProcessingTime = config.ConsumerConfig.MaxProcessingTime
		saramaConfig.Consumer.Fetch.Min = config.ConsumerConfig.FetchMin
		saramaConfig.Consumer.Fetch.Default = config.ConsumerConfig.FetchDefault
		saramaConfig.Consumer.MaxWaitTime = config.ConsumerConfig.MaxWaitTime
		saramaConfig.Consumer.IsolationLevel = sarama.IsolationLevel(config.ConsumerConfig.IsolationLevel)
	}

	// 创建同步生产者
	producer, err := sarama.NewSyncProducer(config.Brokers, saramaConfig)
	if err != nil {
		return nil, errors.WrapInternalError(err, errors.CodeInternalError, "failed to create sync producer")
	}

	// 创建异步生产者
	asyncProducer, err := sarama.NewAsyncProducer(config.Brokers, saramaConfig)
	if err != nil {
		producer.Close()
		return nil, errors.WrapInternalError(err, errors.CodeInternalError, "failed to create async producer")
	}

	// 创建管理客户端
	admin, err := sarama.NewClusterAdmin(config.Brokers, saramaConfig)
	if err != nil {
		producer.Close()
		asyncProducer.Close()
		return nil, errors.WrapInternalError(err, errors.CodeInternalError, "failed to create cluster admin")
	}

	kc := &kafkaClient{
		config:         config,
		producer:       producer,
		asyncProducer:  asyncProducer,
		admin:          admin,
		consumers:      make(map[string]Consumer),
		consumerGroups: make(map[string]ConsumerGroup),
		logger:         config.Logger,
		closed:         false,
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := kc.Ping(ctx); err != nil {
		kc.Close()
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to ping kafka")
	}

	return kc, nil
}

// SendMessage 发送消息
func (kc *kafkaClient) SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error) {
	if kc.closed {
		return nil, errors.New(errors.CodeInternalError, "kafka client is closed")
	}
	if req == nil {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "send message request cannot be nil")
	}
	if req.Topic == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "topic cannot be empty")
	}

	// 构建消息
	msg := &sarama.ProducerMessage{
		Topic: req.Topic,
		Key:   sarama.ByteEncoder(req.Key),
		Value: sarama.ByteEncoder(req.Value),
	}

	if req.Partition >= 0 {
		msg.Partition = req.Partition
	}

	if req.Timestamp != nil {
		msg.Timestamp = *req.Timestamp
	}

	if len(req.Headers) > 0 {
		headers := make([]sarama.RecordHeader, 0, len(req.Headers))
		for k, v := range req.Headers {
			headers = append(headers, sarama.RecordHeader{
				Key:   []byte(k),
				Value: []byte(v),
			})
		}
		msg.Headers = headers
	}

	// 发送消息
	partition, offset, err := kc.producer.SendMessage(msg)
	if err != nil {
		return nil, errors.WrapInternalError(err, errors.CodeInternalError, "failed to send message")
	}

	return &SendMessageResponse{
		Partition: partition,
		Offset:    offset,
		Timestamp: time.Now(),
	}, nil
}

// SendMessages 批量发送消息
func (kc *kafkaClient) SendMessages(ctx context.Context, req *SendMessagesRequest) (*SendMessagesResponse, error) {
	if kc.closed {
		return nil, errors.New(errors.CodeInternalError, "kafka client is closed")
	}
	if req == nil {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "send messages request cannot be nil")
	}
	if len(req.Messages) == 0 {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "messages cannot be empty")
	}

	// 构建消息列表
	messages := make([]*sarama.ProducerMessage, 0, len(req.Messages))
	for _, msgReq := range req.Messages {
		msg := &sarama.ProducerMessage{
			Topic: msgReq.Topic,
			Key:   sarama.ByteEncoder(msgReq.Key),
			Value: sarama.ByteEncoder(msgReq.Value),
		}

		if msgReq.Partition >= 0 {
			msg.Partition = msgReq.Partition
		}

		if msgReq.Timestamp != nil {
			msg.Timestamp = *msgReq.Timestamp
		}

		if len(msgReq.Headers) > 0 {
			headers := make([]sarama.RecordHeader, 0, len(msgReq.Headers))
			for k, v := range msgReq.Headers {
				headers = append(headers, sarama.RecordHeader{
					Key:   []byte(k),
					Value: []byte(v),
				})
			}
			msg.Headers = headers
		}

		messages = append(messages, msg)
	}

	// 批量发送
	err := kc.producer.SendMessages(messages)

	responses := make([]*SendMessageResponse, 0, len(messages))
	errs := make([]error, 0)

	for _, msg := range messages {
		if msg.Metadata != nil {
			if msgErr, ok := msg.Metadata.(error); ok {
				errs = append(errs, msgErr)
				continue
			}
		}

		responses = append(responses, &SendMessageResponse{
			Partition: msg.Partition,
			Offset:    msg.Offset,
			Timestamp: msg.Timestamp,
		})
	}

	return &SendMessagesResponse{
		Responses: responses,
		Errors:    errs,
	}, err
}

// SendMessageAsync 异步发送消息
func (kc *kafkaClient) SendMessageAsync(ctx context.Context, req *SendMessageRequest, callback SendCallback) error {
	if kc.closed {
		return errors.New(errors.CodeInternalError, "kafka client is closed")
	}
	if req == nil {
		return errors.NewValidationError(errors.CodeInvalidParameter, "send message request cannot be nil")
	}
	if req.Topic == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "topic cannot be empty")
	}

	// 构建消息
	msg := &sarama.ProducerMessage{
		Topic: req.Topic,
		Key:   sarama.ByteEncoder(req.Key),
		Value: sarama.ByteEncoder(req.Value),
		Metadata: callback,
	}

	if req.Partition >= 0 {
		msg.Partition = req.Partition
	}

	if req.Timestamp != nil {
		msg.Timestamp = *req.Timestamp
	}

	if len(req.Headers) > 0 {
		headers := make([]sarama.RecordHeader, 0, len(req.Headers))
		for k, v := range req.Headers {
			headers = append(headers, sarama.RecordHeader{
				Key:   []byte(k),
				Value: []byte(v),
			})
		}
		msg.Headers = headers
	}

	// 异步发送
	kc.asyncProducer.Input() <- msg

	// 处理响应和错误
	go func() {
		select {
		case success := <-kc.asyncProducer.Successes():
			if cb, ok := success.Metadata.(SendCallback); ok {
				cb(&SendMessageResponse{
					Partition: success.Partition,
					Offset:    success.Offset,
					Timestamp: success.Timestamp,
				}, nil)
			}
		case err := <-kc.asyncProducer.Errors():
			if cb, ok := err.Msg.Metadata.(SendCallback); ok {
				cb(nil, err.Err)
			}
		}
	}()

	return nil
}

// Subscribe 订阅主题
func (kc *kafkaClient) Subscribe(ctx context.Context, req *SubscribeRequest) (Consumer, error) {
	if kc.closed {
		return nil, errors.New(errors.CodeInternalError, "kafka client is closed")
	}
	if req == nil {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "subscribe request cannot be nil")
	}
	if req.Topic == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "topic cannot be empty")
	}
	if req.Handler == nil {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "message handler cannot be nil")
	}

	kc.mu.Lock()
	defer kc.mu.Unlock()

	if _, exists := kc.consumers[req.ConsumerID]; exists {
		return nil, errors.New(errors.CodeAlreadyExists, "consumer already exists")
	}

	// 创建 Sarama 配置
	config := sarama.NewConfig()
	config.Version = kc.config.Version != ""
	if config.Version {
		version, _ := sarama.ParseKafkaVersion(kc.config.Version)
		config.Version = version
	}
	config.Consumer.Return.Errors = true

	if req.Config != nil {
		config.Consumer.Offsets.AutoCommit.Enable = req.Config.AutoCommit
		config.Consumer.Offsets.AutoCommit.Interval = req.Config.AutoCommitInterval

		if req.FromBeginning {
			config.Consumer.Offsets.Initial = sarama.OffsetOldest
		} else {
			config.Consumer.Offsets.Initial = int64(req.Config.InitialOffset)
		}
	}

	// 创建消费者
	saramaConsumer, err := sarama.NewConsumer(kc.config.Brokers, config)
	if err != nil {
		return nil, errors.WrapInternalError(err, errors.CodeInternalError, "failed to create consumer")
	}

	// 创建分区消费者（简化版，实际应该支持多分区）
	partitionConsumer, err := saramaConsumer.ConsumePartition(req.Topic, 0, sarama.OffsetNewest)
	if err != nil {
		saramaConsumer.Close()
		return nil, errors.WrapInternalError(err, errors.CodeInternalError, "failed to create partition consumer")
	}

	consumerCtx, cancel := context.WithCancel(context.Background())

	c := &consumer{
		id:           req.ConsumerID,
		topic:        req.Topic,
		groupID:      req.GroupID,
		handler:      req.Handler,
		errorHandler: req.ErrorHandler,
		config:       req.Config,
		consumer:     saramaConsumer,
		partition:    partitionConsumer,
		offset:       0,
		paused:       false,
		ctx:          consumerCtx,
		cancel:       cancel,
		logger:       kc.logger,
	}

	kc.consumers[req.ConsumerID] = c

	return c, nil
}

// Unsubscribe 取消订阅
func (kc *kafkaClient) Unsubscribe(ctx context.Context, consumerID string) error {
	if kc.closed {
		return errors.New(errors.CodeInternalError, "kafka client is closed")
	}

	kc.mu.Lock()
	defer kc.mu.Unlock()

	consumer, exists := kc.consumers[consumerID]
	if !exists {
		return errors.NewNotFoundError(errors.CodeNotFound, "consumer not found")
	}

	if err := consumer.Stop(); err != nil {
		return err
	}

	delete(kc.consumers, consumerID)
	return nil
}

// CreateTopic 创建主题
func (kc *kafkaClient) CreateTopic(ctx context.Context, config *TopicConfig) error {
	if kc.closed {
		return errors.New(errors.CodeInternalError, "kafka client is closed")
	}
	if config == nil {
		return errors.NewValidationError(errors.CodeInvalidParameter, "topic config cannot be nil")
	}
	if config.Name == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "topic name cannot be empty")
	}

	detail := &sarama.TopicDetail{
		NumPartitions:     config.NumPartitions,
		ReplicationFactor: config.ReplicationFactor,
		ConfigEntries:     config.ConfigEntries,
	}

	err := kc.admin.CreateTopic(config.Name, detail, false)
	if err != nil {
		return errors.WrapInternalError(err, errors.CodeInternalError, "failed to create topic")
	}

	return nil
}

// DeleteTopic 删除主题
func (kc *kafkaClient) DeleteTopic(ctx context.Context, topic string) error {
	if kc.closed {
		return errors.New(errors.CodeInternalError, "kafka client is closed")
	}
	if topic == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "topic cannot be empty")
	}

	err := kc.admin.DeleteTopic(topic)
	if err != nil {
		return errors.WrapInternalError(err, errors.CodeInternalError, "failed to delete topic")
	}

	return nil
}

// ListTopics 列出所有主题
func (kc *kafkaClient) ListTopics(ctx context.Context) ([]string, error) {
	if kc.closed {
		return nil, errors.New(errors.CodeInternalError, "kafka client is closed")
	}

	topics, err := kc.admin.ListTopics()
	if err != nil {
		return nil, errors.WrapInternalError(err, errors.CodeInternalError, "failed to list topics")
	}

	topicList := make([]string, 0, len(topics))
	for topic := range topics {
		topicList = append(topicList, topic)
	}

	return topicList, nil
}

// GetTopicMetadata 获取主题元数据
func (kc *kafkaClient) GetTopicMetadata(ctx context.Context, topic string) (*TopicMetadata, error) {
	if kc.closed {
		return nil, errors.New(errors.CodeInternalError, "kafka client is closed")
	}
	if topic == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "topic cannot be empty")
	}

	metadata, err := kc.admin.DescribeTopics([]string{topic})
	if err != nil {
		return nil, errors.WrapInternalError(err, errors.CodeInternalError, "failed to describe topic")
	}

	if len(metadata) == 0 {
		return nil, errors.NewNotFoundError(errors.CodeNotFound, "topic not found")
	}

	topicMeta := metadata[0]
	partitions := make([]*PartitionInfo, 0, len(topicMeta.Partitions))

	for _, p := range topicMeta.Partitions {
		partitions = append(partitions, &PartitionInfo{
			ID:       p.ID,
			Leader:   p.Leader,
			Replicas: p.Replicas,
			ISR:      p.Isr,
		})
	}

	return &TopicMetadata{
		Name:       topicMeta.Name,
		Partitions: partitions,
	}, nil
}

// TopicExists 检查主题是否存在
func (kc *kafkaClient) TopicExists(ctx context.Context, topic string) (bool, error) {
	if kc.closed {
		return false, errors.New(errors.CodeInternalError, "kafka client is closed")
	}
	if topic == "" {
		return false, errors.NewValidationError(errors.CodeInvalidParameter, "topic cannot be empty")
	}

	topics, err := kc.admin.ListTopics()
	if err != nil {
		return false, errors.WrapInternalError(err, errors.CodeInternalError, "failed to list topics")
	}

	_, exists := topics[topic]
	return exists, nil
}

// CreateConsumerGroup 创建消费者组
func (kc *kafkaClient) CreateConsumerGroup(ctx context.Context, config *ConsumerGroupConfig) (ConsumerGroup, error) {
	if kc.closed {
		return nil, errors.New(errors.CodeInternalError, "kafka client is closed")
	}
	if config == nil {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "consumer group config cannot be nil")
	}
	if config.GroupID == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "group id cannot be empty")
	}
	if len(config.Topics) == 0 {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "topics cannot be empty")
	}

	kc.mu.Lock()
	defer kc.mu.Unlock()

	if _, exists := kc.consumerGroups[config.GroupID]; exists {
		return nil, errors.New(errors.CodeAlreadyExists, "consumer group already exists")
	}

	// 创建 Sarama 配置
	saramaConfig := sarama.NewConfig()
	if kc.config.Version != "" {
		version, _ := sarama.ParseKafkaVersion(kc.config.Version)
		saramaConfig.Version = version
	}

	if config.Config != nil {
		saramaConfig.Consumer.Return.Errors = true
		saramaConfig.Consumer.Offsets.AutoCommit.Enable = config.Config.AutoCommit
		saramaConfig.Consumer.Offsets.AutoCommit.Interval = config.Config.AutoCommitInterval
		saramaConfig.Consumer.Offsets.Initial = int64(config.Config.InitialOffset)
		saramaConfig.Consumer.Group.Session.Timeout = config.Config.SessionTimeout
		saramaConfig.Consumer.Group.Heartbeat.Interval = config.Config.HeartbeatInterval
		saramaConfig.Consumer.Group.Rebalance.Timeout = config.Config.RebalanceTimeout
	}

	// 设置重平衡策略
	switch config.RebalanceStrategy {
	case RebalanceStrategyRange:
		saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRange()}
	case RebalanceStrategyRoundRobin:
		saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	case RebalanceStrategySticky:
		saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategySticky()}
	default:
		saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRange()}
	}

	// 创建消费者组
	group, err := sarama.NewConsumerGroup(kc.config.Brokers, config.GroupID, saramaConfig)
	if err != nil {
		return nil, errors.WrapInternalError(err, errors.CodeInternalError, "failed to create consumer group")
	}

	groupCtx, cancel := context.WithCancel(context.Background())

	cg := &consumerGroup{
		groupID:      config.GroupID,
		topics:       config.Topics,
		handler:      config.Handler,
		errorHandler: config.ErrorHandler,
		config:       config,
		group:        group,
		paused:       false,
		ctx:          groupCtx,
		cancel:       cancel,
		logger:       kc.logger,
	}

	kc.consumerGroups[config.GroupID] = cg

	return cg, nil
}

// DeleteConsumerGroup 删除消费者组
func (kc *kafkaClient) DeleteConsumerGroup(ctx context.Context, groupID string) error {
	if kc.closed {
		return errors.New(errors.CodeInternalError, "kafka client is closed")
	}
	if groupID == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "group id cannot be empty")
	}

	kc.mu.Lock()
	defer kc.mu.Unlock()

	cg, exists := kc.consumerGroups[groupID]
	if !exists {
		return errors.NewNotFoundError(errors.CodeNotFound, "consumer group not found")
	}

	if err := cg.Stop(); err != nil {
		return err
	}

	delete(kc.consumerGroups, groupID)

	return kc.admin.DeleteConsumerGroup(groupID)
}

// ListConsumerGroups 列出消费者组
func (kc *kafkaClient) ListConsumerGroups(ctx context.Context) ([]string, error) {
	if kc.closed {
		return nil, errors.New(errors.CodeInternalError, "kafka client is closed")
	}

	groups, err := kc.admin.ListConsumerGroups()
	if err != nil {
		return nil, errors.WrapInternalError(err, errors.CodeInternalError, "failed to list consumer groups")
	}

	groupList := make([]string, 0, len(groups))
	for group := range groups {
		groupList = append(groupList, group)
	}

	return groupList, nil
}

// GetConsumerGroupInfo 获取消费者组信息
func (kc *kafkaClient) GetConsumerGroupInfo(ctx context.Context, groupID string) (*ConsumerGroupInfo, error) {
	if kc.closed {
		return nil, errors.New(errors.CodeInternalError, "kafka client is closed")
	}
	if groupID == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "group id cannot be empty")
	}

	groups, err := kc.admin.DescribeConsumerGroups([]string{groupID})
	if err != nil {
		return nil, errors.WrapInternalError(err, errors.CodeInternalError, "failed to describe consumer group")
	}

	if len(groups) == 0 {
		return nil, errors.NewNotFoundError(errors.CodeNotFound, "consumer group not found")
	}

	group := groups[0]
	members := make([]*MemberInfo, 0, len(group.Members))

	for memberID, member := range group.Members {
		members = append(members, &MemberInfo{
			MemberID:         memberID,
			ClientID:         member.ClientId,
			ClientHost:       member.ClientHost,
			MemberMetadata:   member.MemberMetadata,
			MemberAssignment: member.MemberAssignment,
		})
	}

	return &ConsumerGroupInfo{
		GroupID:      group.GroupId,
		State:        group.State,
		Protocol:     group.Protocol,
		ProtocolType: group.ProtocolType,
		Members:      members,
	}, nil
}

// GetClusterMetadata 获取集群元数据
func (kc *kafkaClient) GetClusterMetadata(ctx context.Context) (*ClusterMetadata, error) {
	if kc.closed {
		return nil, errors.New(errors.CodeInternalError, "kafka client is closed")
	}

	metadata, err := kc.admin.DescribeCluster()
	if err != nil {
		return nil, errors.WrapInternalError(err, errors.CodeInternalError, "failed to describe cluster")
	}

	brokers := make([]*BrokerInfo, 0, len(metadata.Brokers))
	for _, broker := range metadata.Brokers {
		brokers = append(brokers, &BrokerInfo{
			ID:   broker.ID(),
			Host: broker.Addr(),
			Port: int32(broker.ID()),
		})
	}

	topics, err := kc.admin.ListTopics()
	if err != nil {
		return nil, errors.WrapInternalError(err, errors.CodeInternalError, "failed to list topics")
	}

	topicList := make([]string, 0, len(topics))
	for topic := range topics {
		topicList = append(topicList, topic)
	}

	return &ClusterMetadata{
		ClusterID:    metadata.ClusterID,
		ControllerID: metadata.ControllerID,
		Brokers:      brokers,
		Topics:       topicList,
	}, nil
}

// GetBrokers 获取 Broker 列表
func (kc *kafkaClient) GetBrokers(ctx context.Context) ([]*BrokerInfo, error) {
	if kc.closed {
		return nil, errors.New(errors.CodeInternalError, "kafka client is closed")
	}

	metadata, err := kc.admin.DescribeCluster()
	if err != nil {
		return nil, errors.WrapInternalError(err, errors.CodeInternalError, "failed to describe cluster")
	}

	brokers := make([]*BrokerInfo, 0, len(metadata.Brokers))
	for _, broker := range metadata.Brokers {
		brokers = append(brokers, &BrokerInfo{
			ID:   broker.ID(),
			Host: broker.Addr(),
		})
	}

	return brokers, nil
}

// Ping 健康检查
func (kc *kafkaClient) Ping(ctx context.Context) error {
	if kc.closed {
		return errors.New(errors.CodeInternalError, "kafka client is closed")
	}

	// 通过列出主题来检查连接
	_, err := kc.admin.ListTopics()
	if err != nil {
		return errors.WrapDatabaseError(err, errors.CodeDatabaseError, "kafka connection check failed")
	}

	return nil
}

// Close 关闭客户端
func (kc *kafkaClient) Close() error {
	kc.mu.Lock()
	defer kc.mu.Unlock()

	if kc.closed {
		return nil
	}

	kc.closed = true

	// 关闭所有消费者
	for _, consumer := range kc.consumers {
		consumer.Stop()
	}

	// 关闭所有消费者组
	for _, group := range kc.consumerGroups {
		group.Stop()
	}

	// 关闭生产者
	if err := kc.producer.Close(); err != nil {
		kc.logger.Error("failed to close producer", "error", err)
	}

	if err := kc.asyncProducer.Close(); err != nil {
		kc.logger.Error("failed to close async producer", "error", err)
	}

	// 关闭管理客户端
	if err := kc.admin.Close(); err != nil {
		kc.logger.Error("failed to close admin", "error", err)
	}

	return nil
}

// Consumer 实现

// Start 启动消费者
func (c *consumer) Start(ctx context.Context) error {
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				return
			case msg := <-c.partition.Messages():
				c.mu.RLock()
				paused := c.paused
				c.mu.RUnlock()

				if paused {
					continue
				}

				message := &Message{
					Topic:     msg.Topic,
					Partition: msg.Partition,
					Offset:    msg.Offset,
					Key:       msg.Key,
					Value:     msg.Value,
					Timestamp: msg.Timestamp,
					Headers:   make(map[string]string),
				}

				for _, header := range msg.Headers {
					message.Headers[string(header.Key)] = string(header.Value)
				}

				if err := c.handler(c.ctx, message); err != nil {
					if c.errorHandler != nil {
						c.errorHandler(err)
					}
					c.logger.Error("failed to handle message", "error", err)
				}

				c.mu.Lock()
				c.offset = msg.Offset
				c.mu.Unlock()

			case err := <-c.partition.Errors():
				if c.errorHandler != nil {
					c.errorHandler(err)
				}
				c.logger.Error("consumer error", "error", err)
			}
		}
	}()

	return nil
}

// Stop 停止消费者
func (c *consumer) Stop() error {
	c.cancel()

	if err := c.partition.Close(); err != nil {
		return errors.WrapInternalError(err, errors.CodeInternalError, "failed to close partition consumer")
	}

	if err := c.consumer.Close(); err != nil {
		return errors.WrapInternalError(err, errors.CodeInternalError, "failed to close consumer")
	}

	return nil
}

// Commit 提交偏移量
func (c *consumer) Commit(ctx context.Context, offset int64) error {
	// Sarama 的简单消费者不支持手动提交
	// 需要使用消费者组来实现
	return nil
}

// Pause 暂停消费
func (c *consumer) Pause() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = true
	return nil
}

// Resume 恢复消费
func (c *consumer) Resume() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = false
	return nil
}

// GetOffset 获取当前偏移量
func (c *consumer) GetOffset() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.offset
}

// ConsumerGroup 实现

// consumerGroupHandler 消费者组处理器
type consumerGroupHandler struct {
	handler      MessageHandler
	errorHandler ErrorHandler
	logger       logger.Logger
}

// Setup 设置
func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup 清理
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 消费声明
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg := <-claim.Messages():
			if msg == nil {
				return nil
			}

			message := &Message{
				Topic:     msg.Topic,
				Partition: msg.Partition,
				Offset:    msg.Offset,
				Key:       msg.Key,
				Value:     msg.Value,
				Timestamp: msg.Timestamp,
				Headers:   make(map[string]string),
			}

			for _, header := range msg.Headers {
				message.Headers[string(header.Key)] = string(header.Value)
			}

			if err := h.handler(session.Context(), message); err != nil {
				if h.errorHandler != nil {
					h.errorHandler(err)
				}
				h.logger.Error("failed to handle message", "error", err)
				continue
			}

			session.MarkMessage(msg, "")

		case <-session.Context().Done():
			return nil
		}
	}
}

// Start 启动消费者组
func (cg *consumerGroup) Start(ctx context.Context) error {
	handler := &consumerGroupHandler{
		handler:      cg.handler,
		errorHandler: cg.errorHandler,
		logger:       cg.logger,
	}

	go func() {
		for {
			select {
			case <-cg.ctx.Done():
				return
			default:
				if err := cg.group.Consume(cg.ctx, cg.topics, handler); err != nil {
					if cg.errorHandler != nil {
						cg.errorHandler(err)
					}
					cg.logger.Error("consumer group error", "error", err)
				}
			}
		}
	}()

	return nil
}

// Stop 停止消费者组
func (cg *consumerGroup) Stop() error {
	cg.cancel()
	return cg.group.Close()
}

// Pause 暂停消费者组
func (cg *consumerGroup) Pause() error {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	cg.paused = true
	cg.group.PauseAll()
	return nil
}

// Resume 恢复消费者组
func (cg *consumerGroup) Resume() error {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	cg.paused = false
	cg.group.ResumeAll()
	return nil
}

// GetMembers 获取成员列表
func (cg *consumerGroup) GetMembers() []*MemberInfo {
	// Sarama 不提供直接获取成员的方法
	// 需要通过 Admin API 获取
	return nil
}

// SerializeMessage 序列化消息
func SerializeMessage(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// DeserializeMessage 反序列化消息
func DeserializeMessage(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

//Personal.AI order the ending
