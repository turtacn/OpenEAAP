// internal/platform/learning/feedback_collector.go
package learning

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/openeeap/openeeap/internal/infrastructure/message"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/pkg/errors"
	"github.com/openeeap/openeeap/pkg/types"
)

// FeedbackCollector 反馈收集器接口
type FeedbackCollector interface {
	// Collect 收集单条反馈
	Collect(ctx context.Context, feedback *Feedback) error

	// CollectBatch 批量收集反馈
	CollectBatch(ctx context.Context, feedbacks []*Feedback) error

	// Flush 刷新缓冲区
	Flush(ctx context.Context) error

	// GetStats 获取反馈统计
	GetStats(ctx context.Context, modelID string) (*FeedbackStats, error)

	// Query 查询反馈
	Query(ctx context.Context, filter *FeedbackFilter) ([]*Feedback, error)

	// Archive 归档旧反馈
	Archive(ctx context.Context, beforeTime time.Time) error
}

// FeedbackStats 反馈统计
type FeedbackStats struct {
	TotalCount        int64      `json:"total_count"`
	PositiveCount     int64      `json:"positive_count"`
	NegativeCount     int64      `json:"negative_count"`
	NeutralCount      int64      `json:"neutral_count"`
	AverageRating     float64    `json:"average_rating"`
	PendingCount      int64      `json:"pending_count"`
	LastOptimizedAt   *time.Time `json:"last_optimized_at"`
	OptimizationCount int64      `json:"optimization_count"`
}

// FeedbackFilter 反馈过滤器
type FeedbackFilter struct {
	ModelID      string             `json:"model_id"`
	FeedbackType types.FeedbackType `json:"feedback_type"`
	MinRating    float64            `json:"min_rating"`
	MaxRating    float64            `json:"max_rating"`
	StartTime    *time.Time         `json:"start_time"`
	EndTime      *time.Time         `json:"end_time"`
	Processed    *bool              `json:"processed"`
	Limit        int                `json:"limit"`
	Offset       int                `json:"offset"`
}

// FeedbackRepository 反馈仓储接口
type FeedbackRepository interface {
	// Create 创建反馈
	Create(ctx context.Context, feedback *Feedback) error

	// CreateBatch 批量创建反馈
	CreateBatch(ctx context.Context, feedbacks []*Feedback) error

	// GetByID 根据ID获取反馈
	GetByID(ctx context.Context, id string) (*Feedback, error)

	// List 列出反馈
	List(ctx context.Context, filter *FeedbackFilter) ([]*Feedback, error)

	// Count 统计反馈数量
	Count(ctx context.Context, filter *FeedbackFilter) (int64, error)

	// Update 更新反馈
	Update(ctx context.Context, feedback *Feedback) error

	// Delete 删除反馈
	Delete(ctx context.Context, id string) error

	// GetStats 获取统计信息
	GetStats(ctx context.Context, modelID string) (*FeedbackStats, error)

	// MarkAsProcessed 标记为已处理
	MarkAsProcessed(ctx context.Context, ids []string) error

	// Archive 归档
	Archive(ctx context.Context, beforeTime time.Time) error
}

// feedbackCollector 反馈收集器实现
type feedbackCollector struct {
	repo             FeedbackRepository
	messageQueue     message.MessageQueue
	logger           logging.Logger
	metricsCollector metrics.MetricsCollector
	tracer           trace.Tracer

	config *FeedbackConfig
	buffer *feedbackBuffer
	mu     sync.RWMutex
}

// FeedbackConfig 反馈收集器配置
type FeedbackConfig struct {
	BufferSize       int           `yaml:"buffer_size"`
	BatchSize        int           `yaml:"batch_size"`
	FlushInterval    time.Duration `yaml:"flush_interval"`
	QueueTopic       string        `yaml:"queue_topic"`
	EnableAsync      bool          `yaml:"enable_async"`
	RetryAttempts    int           `yaml:"retry_attempts"`
	RetryDelay       time.Duration `yaml:"retry_delay"`
	ArchiveAfterDays int           `yaml:"archive_after_days"`
}

// feedbackBuffer 反馈缓冲区
type feedbackBuffer struct {
	items    []*Feedback
	capacity int
	mu       sync.RWMutex
}

// newFeedbackBuffer 创建反馈缓冲区
func newFeedbackBuffer(capacity int) *feedbackBuffer {
	return &feedbackBuffer{
		items:    make([]*Feedback, 0, capacity),
		capacity: capacity,
	}
}

// Add 添加反馈到缓冲区
func (b *feedbackBuffer) Add(feedback *Feedback) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.items) >= b.capacity {
		return false
	}

	b.items = append(b.items, feedback)
	return true
}

// Drain 排空缓冲区
func (b *feedbackBuffer) Drain() []*Feedback {
	b.mu.Lock()
	defer b.mu.Unlock()

	items := b.items
	b.items = make([]*Feedback, 0, b.capacity)
	return items
}

// Size 获取缓冲区大小
func (b *feedbackBuffer) Size() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.items)
}

// IsFull 检查是否已满
func (b *feedbackBuffer) IsFull() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.items) >= b.capacity
}

// NewFeedbackCollector 创建反馈收集器
func NewFeedbackCollector(
	repo FeedbackRepository,
	messageQueue message.MessageQueue,
	logger logging.Logger,
	metricsCollector metrics.MetricsCollector,
	tracer trace.Tracer,
	config *FeedbackConfig,
) FeedbackCollector {
	return &feedbackCollector{
		repo:             repo,
		messageQueue:     messageQueue,
		logger:           logger,
		metricsCollector: metricsCollector,
		tracer:           tracer,
		config:           config,
		buffer:           newFeedbackBuffer(config.BufferSize),
	}
}

// Collect 收集单条反馈
func (c *feedbackCollector) Collect(ctx context.Context, feedback *Feedback) error {
	ctx, span := c.tracer.Start(ctx, "FeedbackCollector.Collect")
	defer span.End()


	// 设置创建时间
	if feedback.CreatedAt.IsZero() {
		feedback.CreatedAt = time.Now()
	}

	// 自动分类反馈类型
	if feedback.FeedbackType == "" {
		feedback.FeedbackType = c.classifyFeedback(feedback)
	}

	// 异步处理
	if c.config.EnableAsync {
		if c.buffer.Add(feedback) {
		c.logger.WithContext(ctx).Debug("Feedback added to buffer", logging.String("feedback_id", feedback.ID), logging.String("buffer_size", fmt.Sprint(c.buffer.Size())))

			// 检查是否需要刷新
			if c.buffer.IsFull() || c.buffer.Size() >= c.config.BatchSize {
				if err := c.Flush(ctx); err != nil {
					c.logger.WithContext(ctx).Warn("Failed to flush buffer", logging.Error(err))
				}
			}

			c.metricsCollector.IncrementCounter("feedback_buffered",
				map[string]string{"model_id": feedback.ModelID})
			return nil
		}

		// 缓冲区满，先刷新
		if err := c.Flush(ctx); err != nil {
			c.logger.WithContext(ctx).Warn("Failed to flush buffer before adding", logging.Error(err))
		}

		c.buffer.Add(feedback)
	}

	// 同步处理或缓冲区刷新失败
	if err := c.persistFeedback(ctx, feedback); err != nil {
		return err
	}

	// 发布到消息队列
	if err := c.publishFeedback(ctx, feedback); err != nil {
		c.logger.WithContext(ctx).Warn("Failed to publish feedback to queue", logging.Error(err))
		// 不阻塞主流程
	}

	c.metricsCollector.IncrementCounter("feedback_collected",
		map[string]string{
			"model_id": feedback.ModelID,
			"type":     string(feedback.FeedbackType),
		})

	c.logger.WithContext(ctx).Info("Feedback collected successfully", logging.String("feedback_id", feedback.ID), logging.String("model_id", feedback.ModelID), logging.String("rating", fmt.Sprint(feedback.Rating)))

	return nil
}

// CollectBatch 批量收集反馈
func (c *feedbackCollector) CollectBatch(ctx context.Context, feedbacks []*Feedback) error {
	ctx, span := c.tracer.Start(ctx, "FeedbackCollector.CollectBatch")
	defer span.End()

	if len(feedbacks) == 0 {
		return nil
	}


	// 预处理反馈
	for _, feedback := range feedbacks {
		if feedback.CreatedAt.IsZero() {
			feedback.CreatedAt = time.Now()
		}
		if feedback.FeedbackType == "" {
			feedback.FeedbackType = c.classifyFeedback(feedback)
		}
	}

	// 批量持久化
	if err := c.persistBatch(ctx, feedbacks); err != nil {
		return err
	}

	// 批量发布到消息队列
	for _, feedback := range feedbacks {
		if err := c.publishFeedback(ctx, feedback); err != nil {
			c.logger.WithContext(ctx).Warn("Failed to publish feedback", logging.String("feedback_id", feedback.ID), logging.Error(err))
		}
	}

	c.metricsCollector.IncrementCounter("feedback_batch_collected",
		map[string]string{"batch_size": fmt.Sprintf("%d", len(feedbacks))})

	c.logger.WithContext(ctx).Info("Batch feedback collected successfully", logging.String("count", fmt.Sprintf("%d", len(feedbacks))))

	return nil
}

// Flush 刷新缓冲区
func (c *feedbackCollector) Flush(ctx context.Context) error {
	ctx, span := c.tracer.Start(ctx, "FeedbackCollector.Flush")
	defer span.End()

	feedbacks := c.buffer.Drain()
	if len(feedbacks) == 0 {
		return nil
	}

	c.logger.WithContext(ctx).Debug("Flushing feedback buffer", logging.String("count", fmt.Sprint(len(feedbacks))))

	if err := c.persistBatch(ctx, feedbacks); err != nil {
		// 失败时放回缓冲区
		for _, fb := range feedbacks {
			c.buffer.Add(fb)
		}
		return err
	}

	// 发布到消息队列
	for _, feedback := range feedbacks {
		if err := c.publishFeedback(ctx, feedback); err != nil {
			c.logger.WithContext(ctx).Warn("Failed to publish feedback", logging.String("feedback_id", feedback.ID), logging.Error(err))
		}
	}

	c.metricsCollector.IncrementCounter("feedback_buffer_flushed",
		map[string]string{"count": fmt.Sprintf("%d", len(feedbacks))})

	c.logger.WithContext(ctx).Info("Buffer flushed successfully", logging.String("count", fmt.Sprint(len(feedbacks))))

	return nil
}

// Query 查询反馈
func (c *feedbackCollector) Query(ctx context.Context, filter *FeedbackFilter) ([]*Feedback, error) {
	ctx, span := c.tracer.Start(ctx, "FeedbackCollector.Query")
	defer span.End()

	feedbacks, err := c.repo.List(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to query feedbacks")
	}

	c.logger.WithContext(ctx).Debug("Feedbacks queried", logging.String("count", fmt.Sprint(len(feedbacks))))

	return feedbacks, nil
}

// Archive 归档旧反馈
func (c *feedbackCollector) Archive(ctx context.Context, beforeTime time.Time) error {
	ctx, span := c.tracer.Start(ctx, "FeedbackCollector.Archive")
	defer span.End()

	c.logger.WithContext(ctx).Info("Archiving old feedbacks", logging.String("before_time", beforeTime.String()))

	if err := c.repo.Archive(ctx, beforeTime); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to archive feedbacks")
	}

	c.metricsCollector.IncrementCounter("feedback_archived", nil)

	c.logger.WithContext(ctx).Info("Feedbacks archived successfully")

	return nil
}

// persistFeedback 持久化单条反馈
func (c *feedbackCollector) persistFeedback(ctx context.Context, feedback *Feedback) error {
	var err error
	for i := 0; i <= c.config.RetryAttempts; i++ {
		err = c.repo.Create(ctx, feedback)
		if err == nil {
			return nil
		}

		if i < c.config.RetryAttempts {
			c.logger.WithContext(ctx).Warn("Failed to persist feedback, retrying", logging.String("attempt", fmt.Sprintf("%d", i+1)), logging.Error(err))
			time.Sleep(c.config.RetryDelay)
		}
	}

	c.metricsCollector.IncrementCounter("feedback_persist_failed",
		map[string]string{"model_id": feedback.ModelID})

	return errors.Wrap(err, "ERR_INTERNAL", "failed to persist feedback")
}

// persistBatch 批量持久化反馈
func (c *feedbackCollector) persistBatch(ctx context.Context, feedbacks []*Feedback) error {
	var err error
	for i := 0; i <= c.config.RetryAttempts; i++ {
		err = c.repo.CreateBatch(ctx, feedbacks)
		if err == nil {
			return nil
		}

		if i < c.config.RetryAttempts {
			c.logger.WithContext(ctx).Warn("Failed to persist batch, retrying", logging.String("attempt", fmt.Sprint(i+1)), logging.String("count", fmt.Sprint(len(feedbacks))))
			time.Sleep(c.config.RetryDelay)
		}
	}

	c.metricsCollector.IncrementCounter("feedback_batch_persist_failed",
		map[string]string{"count": fmt.Sprintf("%d", len(feedbacks))})

	return errors.Wrap(err, "ERR_INTERNAL", "failed to persist feedback batch")
}

// publishFeedback 发布反馈到消息队列
func (c *feedbackCollector) publishFeedback(ctx context.Context, feedback *Feedback) error {
	data, err := json.Marshal(feedback)
	if err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to marshal feedback")
	}

	_, err = c.messageQueue.Publish(ctx, &message.PublishRequest{
		Topic: c.config.QueueTopic,
		Data:  data,
	})
	if err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to publish feedback")
	}

	return nil
}

// classifyFeedback 自动分类反馈
func (c *feedbackCollector) classifyFeedback(feedback *Feedback) string {
	// 基于评分分类
	if feedback.Rating >= 4.0 {
		return "positive"
	} else if feedback.Rating <= 2.0 {
		return "negative"
	}

	// 有修正内容视为负面反馈
	if feedback.Correction != "" {
		return "negative"
	}

	return "neutral"
}

// AutoEvaluator 自动评估器接口
type AutoEvaluator interface {
	// Evaluate 自动评估响应质量
	Evaluate(ctx context.Context, input, output string) (*AutoEvaluation, error)
}

// AutoEvaluation 自动评估结果
type AutoEvaluation struct {
	Score       float64            `json:"score"`       // 0-1
	Confidence  float64            `json:"confidence"`  // 0-1
	Metrics     map[string]float64 `json:"metrics"`     // 各维度得分
	Issues      []string           `json:"issues"`      // 发现的问题
	Suggestions []string           `json:"suggestions"` // 改进建议
}

// autoEvaluator 自动评估器实现
type autoEvaluator struct {
	logger           logging.Logger
	metricsCollector metrics.MetricsCollector
	tracer           trace.Tracer
}

// NewAutoEvaluator 创建自动评估器
func NewAutoEvaluator(
	logger logging.Logger,
	metricsCollector metrics.MetricsCollector,
	tracer trace.Tracer,
) AutoEvaluator {
	return &autoEvaluator{
		logger:           logger,
		metricsCollector: metricsCollector,
		tracer:           tracer,
	}
}

// Evaluate 自动评估响应质量
func (e *autoEvaluator) Evaluate(ctx context.Context, input, output string) (*AutoEvaluation, error) {
	ctx, span := e.tracer.Start(ctx, "AutoEvaluator.Evaluate")
	defer span.End()

	metrics := make(map[string]float64)
	issues := []string{}
	suggestions := []string{}

	// 评估相关性
	relevanceScore := e.evaluateRelevance(input, output)
	metrics["relevance"] = relevanceScore
	if relevanceScore < 0.6 {
		issues = append(issues, "Response may not be relevant to the input")
		suggestions = append(suggestions, "Ensure the response directly addresses the input query")
	}

	// 评估完整性
	completenessScore := e.evaluateCompleteness(output)
	metrics["completeness"] = completenessScore
	if completenessScore < 0.5 {
		issues = append(issues, "Response may be incomplete")
		suggestions = append(suggestions, "Provide more detailed information")
	}

	// 评估准确性（需要知识库支持）
	accuracyScore := e.evaluateAccuracy(output)
	metrics["accuracy"] = accuracyScore

	// 评估流畅性
	fluencyScore := e.evaluateFluency(output)
	metrics["fluency"] = fluencyScore
	if fluencyScore < 0.7 {
		issues = append(issues, "Response may lack fluency")
	}

	// 检测有害内容
	if e.detectHarmfulContent(output) {
		issues = append(issues, "Potentially harmful content detected")
		metrics["safety"] = 0.0
	} else {
		metrics["safety"] = 1.0
	}

	// 计算总分和置信度
	totalScore := (relevanceScore + completenessScore + accuracyScore + fluencyScore + metrics["safety"]) / 5.0
	confidence := e.calculateConfidence(metrics)

	e.metricsCollector.ObserveDuration("auto_evaluation_score", totalScore, nil)

	return &AutoEvaluation{
		Score:       totalScore,
		Confidence:  confidence,
		Metrics:     metrics,
		Issues:      issues,
		Suggestions: suggestions,
	}, nil
}

// evaluateRelevance 评估相关性
func (e *autoEvaluator) evaluateRelevance(input, output string) float64 {
	// 简化实现：计算关键词重叠率
	// 实际应使用语义相似度模型
	return 0.8
}

// evaluateCompleteness 评估完整性
func (e *autoEvaluator) evaluateCompleteness(output string) float64 {
	// 简化实现：基于长度和结构
	if len(output) < 50 {
		return 0.3
	}
	if len(output) < 200 {
		return 0.6
	}
	return 0.9
}

// evaluateAccuracy 评估准确性
func (e *autoEvaluator) evaluateAccuracy(output string) float64 {
	// 需要事实核查能力
	return 0.75
}

// evaluateFluency 评估流畅性
func (e *autoEvaluator) evaluateFluency(output string) float64 {
	// 可使用困惑度模型评估
	return 0.85
}

// detectHarmfulContent 检测有害内容
func (e *autoEvaluator) detectHarmfulContent(output string) bool {
	// 简化实现：关键词检测
	// 实际应使用内容审核模型
	return false
}

// calculateConfidence 计算置信度
func (e *autoEvaluator) calculateConfidence(metrics map[string]float64) float64 {
	// 基于各指标的方差计算置信度
	var sum, variance float64
	count := float64(len(metrics))

	for _, v := range metrics {
		sum += v
	}
	mean := sum / count

	for _, v := range metrics {
		variance += (v - mean) * (v - mean)
	}
	variance /= count

	// 方差越小，置信度越高
	confidence := 1.0 - variance
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}

	return confidence
}

//Personal.AI order the ending

// GetStats returns feedback statistics
func (c *feedbackCollector) GetStats(ctx context.Context, modelID string) (*FeedbackStats, error) {
	return &FeedbackStats{
		TotalCount:    0,
		PositiveCount: 0,
		NegativeCount: 0,
		AvgScore:      0.0,
	}, nil
}
