// internal/platform/learning/learning_engine.go
package learning

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/openeeap/openeeap/internal/domain/model"
	"github.com/openeeap/openeeap/internal/infrastructure/message"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/internal/platform/training"
	"github.com/openeeap/openeeap/pkg/errors"
	"github.com/openeeap/openeeap/pkg/types"
)

// OnlineLearningEngine 在线学习引擎接口
type OnlineLearningEngine interface {
	// Start 启动学习引擎
	Start(ctx context.Context) error

	// Stop 停止学习引擎
	Stop(ctx context.Context) error

	// ProcessFeedback 处理单条反馈
	ProcessFeedback(ctx context.Context, feedback *Feedback) error

	// TriggerOptimization 触发优化任务
	TriggerOptimization(ctx context.Context, req *OptimizationRequest) (*OptimizationResult, error)

	// GetLearningStats 获取学习统计
	GetLearningStats(ctx context.Context, modelID string) (*LearningStats, error)
}

// Feedback 反馈数据
type Feedback struct {
	ID            string                 `json:"id"`
	ModelID       string                 `json:"model_id"`
	RequestID     string                 `json:"request_id"`
	Input         string                 `json:"input"`
	Output        string                 `json:"output"`
	Rating        float64                `json:"rating"`        // 1-5 评分
	Correction    string                 `json:"correction"`    // 用户修正
	FeedbackType  types.FeedbackType     `json:"feedback_type"` // Positive/Negative/Neutral
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
	ProcessedAt   *time.Time             `json:"processed_at"`
	AutoEvaluated bool                   `json:"auto_evaluated"`
}

// OptimizationRequest 优化请求
type OptimizationRequest struct {
	ModelID          string                 `json:"model_id"`
	OptimizationType types.TrainingType     `json:"optimization_type"` // RLHF/DPO/PromptTuning
	Config           map[string]interface{} `json:"config"`
	Priority         int                    `json:"priority"`
	ForceTrigger     bool                   `json:"force_trigger"`
}

// OptimizationResult 优化结果
type OptimizationResult struct {
	TaskID       string                 `json:"task_id"`
	Status       string                 `json:"status"`
	ModelVersion string                 `json:"model_version"`
	Improvements map[string]float64     `json:"improvements"` // 指标改善
	StartedAt    time.Time              `json:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// LearningStats 学习统计
type LearningStats struct {
	ModelID              string             `json:"model_id"`
	TotalFeedbacks       int64              `json:"total_feedbacks"`
	PositiveFeedbacks    int64              `json:"positive_feedbacks"`
	NegativeFeedbacks    int64              `json:"negative_feedbacks"`
	AverageRating        float64            `json:"average_rating"`
	LastOptimizedAt      *time.Time         `json:"last_optimized_at"`
	OptimizationCount    int64              `json:"optimization_count"`
	PerformanceTrend     []PerformancePoint `json:"performance_trend"`
	PendingFeedbackCount int64              `json:"pending_feedback_count"`
}

// PerformancePoint 性能数据点
type PerformancePoint struct {
	Timestamp time.Time          `json:"timestamp"`
	Metrics   map[string]float64 `json:"metrics"`
}

// learningEngine 在线学习引擎实现
type learningEngine struct {
	feedbackCollector FeedbackCollector
	optimizer         Optimizer
	trainingService   training.TrainingService
	messageQueue      message.MessageQueue
	modelRepo         model.ModelRepository
	logger            logging.Logger
	metricsCollector  metrics.MetricsCollector
	tracer            trace.Tracer

	config       *LearningConfig
	runningTasks sync.Map // 运行中的优化任务
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// LearningConfig 学习引擎配置
type LearningConfig struct {
	// 反馈收集
	FeedbackBufferSize    int           `yaml:"feedback_buffer_size"`
	FeedbackBatchSize     int           `yaml:"feedback_batch_size"`
	FeedbackFlushInterval time.Duration `yaml:"feedback_flush_interval"`

	// 数据清洗
	MinRatingThreshold   float64 `yaml:"min_rating_threshold"`
	MinFeedbackCount     int     `yaml:"min_feedback_count"`
	DataQualityThreshold float64 `yaml:"data_quality_threshold"`

	// 优化触发
	OptimizationInterval   time.Duration `yaml:"optimization_interval"`
	MinFeedbackForOptimize int           `yaml:"min_feedback_for_optimize"`
	AutoOptimizeEnabled    bool          `yaml:"auto_optimize_enabled"`

	// 性能监控
	MetricsWindowSize      int     `yaml:"metrics_window_size"`
	PerformanceDegradation float64 `yaml:"performance_degradation_threshold"`
}

// NewLearningEngine 创建在线学习引擎
func NewLearningEngine(
	feedbackCollector FeedbackCollector,
	optimizer Optimizer,
	trainingService training.TrainingService,
	messageQueue message.MessageQueue,
	modelRepo model.ModelRepository,
	logger logging.Logger,
	metricsCollector metrics.MetricsCollector,
	tracer trace.Tracer,
	config *LearningConfig,
) OnlineLearningEngine {
	ctx, cancel := context.WithCancel(context.Background())

	return &learningEngine{
		feedbackCollector: feedbackCollector,
		optimizer:         optimizer,
		trainingService:   trainingService,
		messageQueue:      messageQueue,
		modelRepo:         modelRepo,
		logger:            logger,
		metricsCollector:  metricsCollector,
		tracer:            tracer,
		config:            config,
		ctx:               ctx,
		cancel:            cancel,
	}
}

// Start 启动学习引擎
func (e *learningEngine) Start(ctx context.Context) error {
	ctx, span := e.tracer.Start(ctx, "LearningEngine.Start")
	defer span.End()

	e.logger.WithContext(ctx).Info("Starting online learning engine")

	// 启动反馈处理协程
	e.wg.Add(1)
	go e.processFeedbackLoop()

	// 启动自动优化协程
	if e.config.AutoOptimizeEnabled {
		e.wg.Add(1)
		go e.autoOptimizationLoop()
	}

	// 启动性能监控协程
	e.wg.Add(1)
	go e.performanceMonitorLoop()

	// 订阅反馈消息队列
	if err := e.subscribeFeedbackQueue(ctx); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to subscribe feedback queue")
	}

	e.metricsCollector.IncrementCounter("learning_engine_started", nil)
	e.logger.WithContext(ctx).Info("Online learning engine started successfully")

	return nil
}

// Stop 停止学习引擎
func (e *learningEngine) Stop(ctx context.Context) error {
	ctx, span := e.tracer.Start(ctx, "LearningEngine.Stop")
	defer span.End()

	e.logger.WithContext(ctx).Info("Stopping online learning engine")

	e.cancel()
	e.wg.Wait()

	e.metricsCollector.IncrementCounter("learning_engine_stopped", nil)
	e.logger.WithContext(ctx).Info("Online learning engine stopped successfully")

	return nil
}

// ProcessFeedback 处理单条反馈
func (e *learningEngine) ProcessFeedback(ctx context.Context, feedback *Feedback) error {
	ctx, span := e.tracer.Start(ctx, "LearningEngine.ProcessFeedback")
	defer span.End()

	startTime := time.Now()
	defer func() {
		e.metricsCollector.ObserveDuration("learning_process_feedback_duration_ms",
			float64(time.Since(startTime).Milliseconds()),
			map[string]string{"model_id": feedback.ModelID})
	}()

	// 验证反馈数据
	if err := e.validateFeedback(feedback); err != nil {
		e.logger.WithContext(ctx).Warn("Invalid feedback", logging.Error(err), logging.Any("feedback_id", feedback.ID))
		e.metricsCollector.IncrementCounter("learning_invalid_feedback",
			map[string]string{"model_id": feedback.ModelID})
		return errors.Wrap(err, errors.CodeInvalidArgument, "invalid feedback")
	}

	// 收集反馈
	if err := e.feedbackCollector.Collect(ctx, feedback); err != nil {
		e.logger.WithContext(ctx).Error("Failed to collect feedback", logging.Error(err), logging.Any("feedback_id", feedback.ID))
		e.metricsCollector.IncrementCounter("learning_collect_feedback_failed",
			map[string]string{"model_id": feedback.ModelID})
		return errors.Wrap(err, "ERR_INTERNAL", "failed to collect feedback")
	}

	feedback.ProcessedAt = &startTime

	e.metricsCollector.IncrementCounter("learning_feedback_processed",
		map[string]string{
			"model_id": feedback.ModelID,
			"type":     string(feedback.FeedbackType),
		})

 e.logger.WithContext(ctx).Info("Feedback processed successfully", logging.Any("feedback_id", feedback.ID), logging.Any("model_id", feedback.ModelID))

	// 检查是否需要触发优化
	if e.config.AutoOptimizeEnabled {
		if err := e.checkAndTriggerOptimization(ctx, feedback.ModelID); err != nil {
			e.logger.WithContext(ctx).Warn("Failed to check optimization trigger", logging.Error(err))
		}
	}

	return nil
}

// TriggerOptimization 触发优化任务
func (e *learningEngine) TriggerOptimization(ctx context.Context, req *OptimizationRequest) (*OptimizationResult, error) {
	ctx, span := e.tracer.Start(ctx, "LearningEngine.TriggerOptimization")
	defer span.End()

 e.logger.WithContext(ctx).Info("Triggering model optimization", logging.Any("model_id", req.ModelID), logging.Any("type", req.OptimizationType))

	// 检查是否已有运行中的任务
	if !req.ForceTrigger {
		if _, exists := e.runningTasks.Load(req.ModelID); exists {
			return nil, errors.NewConflictError(errors.CodeConflict,
				fmt.Sprintf("optimization task already running for model %s", req.ModelID))
		}
	}

	// 检查反馈数量是否足够
	stats, err := e.GetLearningStats(ctx, req.ModelID)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to get learning stats")
	}

	if stats.PendingFeedbackCount < int64(e.config.MinFeedbackForOptimize) && !req.ForceTrigger {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter,
			fmt.Sprintf("insufficient feedback count: %d < %d",
				stats.PendingFeedbackCount, e.config.MinFeedbackForOptimize))
	}

	// 准备训练数据
	trainingData, err := e.optimizer.PrepareTrainingData(ctx, req.ModelID)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to prepare training data")
	}

	// 创建训练任务
	trainingReq := &training.TrainingRequest{
		ModelID:      req.ModelID,
		TrainingType: req.OptimizationType,
		Dataset:      trainingData,
		Config:       req.Config,
		Priority:     req.Priority,
	}

	task, err := e.trainingService.CreateTask(ctx, trainingReq)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to create training task")
	}

	// 启动训练
	if err := e.trainingService.StartTask(ctx, task.ID); err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to start training task")
	}

	result := &OptimizationResult{
		TaskID:    task.ID,
		Status:    "running",
		StartedAt: time.Now(),
		Metadata: map[string]interface{}{
			"feedback_count": stats.PendingFeedbackCount,
			"training_type":  req.OptimizationType,
		},
	}

	e.runningTasks.Store(req.ModelID, result)

	e.metricsCollector.IncrementCounter("learning_optimization_triggered",
		map[string]string{
			"model_id": req.ModelID,
			"type":     string(req.OptimizationType),
		})

	// 异步监控训练任务
	go e.monitorTrainingTask(context.Background(), req.ModelID, task.ID)

	return result, nil
}

// GetLearningStats 获取学习统计
func (e *learningEngine) GetLearningStats(ctx context.Context, modelID string) (*LearningStats, error) {
	ctx, span := e.tracer.Start(ctx, "LearningEngine.GetLearningStats")
	defer span.End()

	stats, err := e.feedbackCollector.GetStats(ctx, modelID)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to get feedback stats")
	}

	// 获取性能趋势
	trend, err := e.getPerformanceTrend(ctx, modelID)
	if err != nil {
		e.logger.WithContext(ctx).Warn("Failed to get performance trend", logging.Error(err))
		trend = []PerformancePoint{}
	}

	return &LearningStats{
		ModelID:              modelID,
		TotalFeedbacks:       stats.TotalCount,
		PositiveFeedbacks:    stats.PositiveCount,
		NegativeFeedbacks:    stats.NegativeCount,
		AverageRating:        stats.AverageRating,
		LastOptimizedAt:      stats.LastOptimizedAt,
		OptimizationCount:    stats.OptimizationCount,
		PerformanceTrend:     trend,
		PendingFeedbackCount: stats.PendingCount,
	}, nil
}

// processFeedbackLoop 反馈处理循环
func (e *learningEngine) processFeedbackLoop() {
	defer e.wg.Done()

	ticker := time.NewTicker(e.config.FeedbackFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			e.logger.Info("Feedback processing loop stopped")
			return
		case <-ticker.C:
			if err := e.flushFeedbacks(context.Background()); err != nil {
				e.logger.Error("Failed to flush feedbacks", "error", err)
			}
		}
	}
}

// autoOptimizationLoop 自动优化循环
func (e *learningEngine) autoOptimizationLoop() {
	defer e.wg.Done()

	ticker := time.NewTicker(e.config.OptimizationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			e.logger.Info("Auto optimization loop stopped")
			return
		case <-ticker.C:
			if err := e.checkAllModelsForOptimization(context.Background()); err != nil {
				e.logger.Error("Failed to check models for optimization", "error", err)
			}
		}
	}
}

// performanceMonitorLoop 性能监控循环
func (e *learningEngine) performanceMonitorLoop() {
	defer e.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			e.logger.Info("Performance monitor loop stopped")
			return
		case <-ticker.C:
			if err := e.monitorPerformanceDegradation(context.Background()); err != nil {
				e.logger.Error("Failed to monitor performance", "error", err)
			}
		}
	}
}

// subscribeFeedbackQueue 订阅反馈消息队列
func (e *learningEngine) subscribeFeedbackQueue(ctx context.Context) error {
	return e.messageQueue.Subscribe(ctx, "feedback", func(msg []byte) error {
		var feedback Feedback
		if err := json.Unmarshal(msg, &feedback); err != nil {
			e.logger.WithContext(ctx).Error("Failed to unmarshal feedback message", logging.Error(err))
			return err
		}
		return e.ProcessFeedback(ctx, &feedback)
	})
}

// validateFeedback 验证反馈数据
func (e *learningEngine) validateFeedback(feedback *Feedback) error {
	if feedback.ModelID == "" {
		return errors.ValidationError( "model_id is required")
	}
	if feedback.Input == "" {
		return errors.ValidationError( "input is required")
	}
	if feedback.Rating < 1 || feedback.Rating > 5 {
		return errors.ValidationError( "rating must be between 1 and 5")
	}
	return nil
}

// checkAndTriggerOptimization 检查并触发优化
func (e *learningEngine) checkAndTriggerOptimization(ctx context.Context, modelID string) error {
	stats, err := e.GetLearningStats(ctx, modelID)
	if err != nil {
		return err
	}

	if stats.PendingFeedbackCount >= int64(e.config.MinFeedbackForOptimize) {
		req := &OptimizationRequest{
			ModelID:          modelID,
			OptimizationType: types.TrainingTypeDPO,
			Priority:         5,
		}

		if _, err := e.TriggerOptimization(ctx, req); err != nil {
			return err
		}
	}

	return nil
}

// checkAllModelsForOptimization 检查所有模型是否需要优化
func (e *learningEngine) checkAllModelsForOptimization(ctx context.Context) error {
	models, err := e.modelRepo.List(ctx, nil)
	if err != nil {
		return err
	}

	for _, mdl := range models {
		if err := e.checkAndTriggerOptimization(ctx, mdl.ID); err != nil {
   e.logger.WithContext(ctx).Warn("Failed to check model for optimization", logging.Any("model_id", mdl.ID), logging.Error(err))
		}
	}

	return nil
}

// monitorPerformanceDegradation 监控性能退化
func (e *learningEngine) monitorPerformanceDegradation(ctx context.Context) error {
	models, err := e.modelRepo.List(ctx, nil)
	if err != nil {
		return err
	}

	for _, mdl := range models {
		trend, err := e.getPerformanceTrend(ctx, mdl.ID)
		if err != nil {
			continue
		}

		if len(trend) < 2 {
			continue
		}

		// 检查性能是否下降
		latest := trend[len(trend)-1]
		previous := trend[len(trend)-2]

		for metric, latestValue := range latest.Metrics {
			if previousValue, ok := previous.Metrics[metric]; ok {
				degradation := (previousValue - latestValue) / previousValue
				if degradation > e.config.PerformanceDegradation {
     e.logger.WithContext(ctx).Warn("Performance degradation detected", logging.Any("model_id", mdl.ID), logging.Any("metric", metric), logging.Any("degradation", degradation))

					e.metricsCollector.IncrementCounter("learning_performance_degradation_detected",
						map[string]string{"model_id": mdl.ID, "metric": metric})
				}
			}
		}
	}

	return nil
}

// monitorTrainingTask 监控训练任务
func (e *learningEngine) monitorTrainingTask(ctx context.Context, modelID, taskID string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			task, err := e.trainingService.GetTask(ctx, taskID)
			if err != nil {
				e.logger.WithContext(ctx).Error("Failed to get training task", logging.Error(err), logging.Any("task_id", taskID))
				continue
			}

			if task.Status == "completed" {
				completedAt := time.Now()
				if result, ok := e.runningTasks.Load(modelID); ok {
					optimResult := result.(*OptimizationResult)
					optimResult.Status = "completed"
					optimResult.CompletedAt = &completedAt
					optimResult.ModelVersion = task.ModelVersion
					optimResult.Improvements = task.Metrics
				}

				e.runningTasks.Delete(modelID)

				e.metricsCollector.IncrementCounter("learning_optimization_completed",
					map[string]string{"model_id": modelID})

    e.logger.WithContext(ctx).Info("Optimization completed", logging.Any("model_id", modelID), logging.Any("task_id", taskID))
				return
			}

			if task.Status == "failed" {
				e.runningTasks.Delete(modelID)

				e.metricsCollector.IncrementCounter("learning_optimization_failed",
					map[string]string{"model_id": modelID})

    e.logger.WithContext(ctx).Error("Optimization failed", logging.Any("model_id", modelID), logging.Any("task_id", taskID))
				return
			}
		}
	}
}

// flushFeedbacks 刷新反馈缓冲区
func (e *learningEngine) flushFeedbacks(ctx context.Context) error {
	return e.feedbackCollector.Flush(ctx)
}

// getPerformanceTrend 获取性能趋势
func (e *learningEngine) getPerformanceTrend(ctx context.Context, modelID string) ([]PerformancePoint, error) {
	// 从指标系统查询性能趋势数据
	// 这里简化实现，实际应查询时序数据库
	return []PerformancePoint{}, nil
}

//Personal.AI order the ending
