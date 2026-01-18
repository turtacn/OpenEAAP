// internal/platform/training/training_service.go
package training

import (
	"context"
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

// TrainingService 训练服务接口
type TrainingService interface {
	// CreateTask 创建训练任务
	CreateTask(ctx context.Context, req *TrainingRequest) (*TrainingTask, error)

	// StartTask 启动训练任务
	StartTask(ctx context.Context, taskID string) error

	// StopTask 停止训练任务
	StopTask(ctx context.Context, taskID string) error

	// PauseTask 暂停训练任务
	PauseTask(ctx context.Context, taskID string) error

	// ResumeTask 恢复训练任务
	ResumeTask(ctx context.Context, taskID string) error

	// GetTask 获取训练任务
	GetTask(ctx context.Context, taskID string) (*TrainingTask, error)

	// ListTasks 列出训练任务
	ListTasks(ctx context.Context, filter *TaskFilter) ([]*TrainingTask, error)

	// DeleteTask 删除训练任务
	DeleteTask(ctx context.Context, taskID string) error

	// GetTaskMetrics 获取任务指标
	GetTaskMetrics(ctx context.Context, taskID string) (*TaskMetrics, error)

	// GetTaskLogs 获取任务日志
	GetTaskLogs(ctx context.Context, taskID string, limit int) ([]string, error)
}

// TrainingRequest 训练请求
type TrainingRequest struct {
	ModelID      string                 `json:"model_id"`
	TrainingType types.TrainingType     `json:"training_type"`
	Dataset      *TrainingDataset       `json:"dataset"`
	Config       map[string]interface{} `json:"config"`
	Priority     int                    `json:"priority"`
	Resources    *ResourceConfig        `json:"resources"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// TrainingDataset 训练数据集
type TrainingDataset struct {
	ID          string            `json:"id"`
	Samples     []*TrainingSample `json:"samples"`
	SampleCount int               `json:"sample_count"`
	SplitRatio  *SplitRatio       `json:"split_ratio"`
}

// TrainingSample 训练样本
type TrainingSample struct {
	Input     string                 `json:"input"`
	Output    string                 `json:"output"`
	Preferred string                 `json:"preferred"`
	Rejected  string                 `json:"rejected"`
	Reward    float64                `json:"reward"`
	Weight    float64                `json:"weight"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// SplitRatio 数据集分割比例
type SplitRatio struct {
	Train      float64 `json:"train"`
	Validation float64 `json:"validation"`
	Test       float64 `json:"test"`
}

// ResourceConfig 资源配置
type ResourceConfig struct {
	GPUCount    int    `json:"gpu_count"`
	GPUType     string `json:"gpu_type"`
	CPUCores    int    `json:"cpu_cores"`
	MemoryGB    int    `json:"memory_gb"`
	StorageGB   int    `json:"storage_gb"`
	MaxDuration int    `json:"max_duration_hours"`
}

// TrainingTask 训练任务
type TrainingTask struct {
	ID           string                 `json:"id"`
	ModelID      string                 `json:"model_id"`
	TrainingType types.TrainingType     `json:"training_type"`
	Status       TaskStatus             `json:"status"`
	Progress     float64                `json:"progress"`
	Epoch        int                    `json:"epoch"`
	TotalEpochs  int                    `json:"total_epochs"`
	CurrentLoss  float64                `json:"current_loss"`
	BestLoss     float64                `json:"best_loss"`
	ModelVersion string                 `json:"model_version"`
	Metrics      map[string]float64     `json:"metrics"`
	Config       map[string]interface{} `json:"config"`
	Resources    *ResourceConfig        `json:"resources"`
	StartedAt    *time.Time             `json:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at"`
	ErrorMessage string                 `json:"error_message"`
	Checkpoints  []*Checkpoint          `json:"checkpoints"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusPaused    TaskStatus = "paused"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// Checkpoint 检查点
type Checkpoint struct {
	ID        string             `json:"id"`
	Epoch     int                `json:"epoch"`
	Step      int                `json:"step"`
	Loss      float64            `json:"loss"`
	Metrics   map[string]float64 `json:"metrics"`
	Path      string             `json:"path"`
	Size      int64              `json:"size"`
	CreatedAt time.Time          `json:"created_at"`
}

// TaskFilter 任务过滤器
type TaskFilter struct {
	ModelID      string             `json:"model_id"`
	Status       TaskStatus         `json:"status"`
	TrainingType types.TrainingType `json:"training_type"`
	StartTime    *time.Time         `json:"start_time"`
	EndTime      *time.Time         `json:"end_time"`
	Limit        int                `json:"limit"`
	Offset       int                `json:"offset"`
}

// TaskMetrics 任务指标
type TaskMetrics struct {
	TaskID         string                   `json:"task_id"`
	TrainingLoss   []MetricPoint            `json:"training_loss"`
	ValidationLoss []MetricPoint            `json:"validation_loss"`
	LearningRate   []MetricPoint            `json:"learning_rate"`
	CustomMetrics  map[string][]MetricPoint `json:"custom_metrics"`
	ResourceUsage  *ResourceUsage           `json:"resource_usage"`
	LastUpdated    time.Time                `json:"last_updated"`
}

// MetricPoint 指标数据点
type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Step      int       `json:"step"`
	Value     float64   `json:"value"`
}

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	GPUUtilization []float64 `json:"gpu_utilization"`
	GPUMemoryUsage []float64 `json:"gpu_memory_usage"`
	CPUUtilization float64   `json:"cpu_utilization"`
	MemoryUsage    float64   `json:"memory_usage"`
	DiskIO         float64   `json:"disk_io"`
}

// TrainingEngine 训练引擎接口
type TrainingEngine interface {
	// Train 执行训练
	Train(ctx context.Context, task *TrainingTask, dataset *TrainingDataset) error

	// Stop 停止训练
	Stop(ctx context.Context, taskID string) error

	// Pause 暂停训练
	Pause(ctx context.Context, taskID string) error

	// Resume 恢复训练
	Resume(ctx context.Context, taskID string) error

	// GetProgress 获取训练进度
	GetProgress(ctx context.Context, taskID string) (*TrainingProgress, error)
}

// TrainingProgress 训练进度
type TrainingProgress struct {
	TaskID    string             `json:"task_id"`
	Progress  float64            `json:"progress"`
	Epoch     int                `json:"epoch"`
	Step      int                `json:"step"`
	Loss      float64            `json:"loss"`
	Metrics   map[string]float64 `json:"metrics"`
	ETA       time.Duration      `json:"eta"`
	UpdatedAt time.Time          `json:"updated_at"`
}

// TaskRepository 任务仓储接口
type TaskRepository interface {
	Create(ctx context.Context, task *TrainingTask) error
	GetByID(ctx context.Context, id string) (*TrainingTask, error)
	Update(ctx context.Context, task *TrainingTask) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter *TaskFilter) ([]*TrainingTask, error)
	SaveCheckpoint(ctx context.Context, taskID string, checkpoint *Checkpoint) error
	SaveMetrics(ctx context.Context, taskID string, metrics *TaskMetrics) error
	GetMetrics(ctx context.Context, taskID string) (*TaskMetrics, error)
}

// trainingService 训练服务实现
type trainingService struct {
	taskRepo         TaskRepository
	modelRepo        interface{}
	rlhfEngine       TrainingEngine
	dpoEngine        TrainingEngine
	messageQueue     message.MessageQueue
	logger           logging.Logger
	metricsCollector metrics.MetricsCollector
	tracer           trace.Tracer

	config       *TrainingConfig
	runningTasks sync.Map
	taskQueue    chan *TrainingTask
	workerPool   *WorkerPool
	mu           sync.RWMutex
}

// TrainingConfig 训练服务配置
type TrainingConfig struct {
	MaxConcurrentTasks    int           `yaml:"max_concurrent_tasks"`
	DefaultEpochs         int           `yaml:"default_epochs"`
	DefaultBatchSize      int           `yaml:"default_batch_size"`
	DefaultLearningRate   float64       `yaml:"default_learning_rate"`
	CheckpointInterval    int           `yaml:"checkpoint_interval"`
	EvaluationInterval    int           `yaml:"evaluation_interval"`
	EarlyStoppingEnabled  bool          `yaml:"early_stopping_enabled"`
	EarlyStoppingPatience int           `yaml:"early_stopping_patience"`
	TaskQueueSize         int           `yaml:"task_queue_size"`
	MetricsFlushInterval  time.Duration `yaml:"metrics_flush_interval"`
	MaxRetries            int           `yaml:"max_retries"`
	RetryDelay            time.Duration `yaml:"retry_delay"`
}

// WorkerPool 工作池
type WorkerPool struct {
	workers      int
	taskQueue    chan *TrainingTask
	stopChan     chan struct{}
	wg           sync.WaitGroup
	trainingFunc func(context.Context, *TrainingTask) error
}

// NewTrainingService 创建训练服务
func NewTrainingService(
	taskRepo TaskRepository,
	modelRepo interface{},
	rlhfEngine TrainingEngine,
	dpoEngine TrainingEngine,
	messageQueue message.MessageQueue,
	logger logging.Logger,
	metricsCollector metrics.MetricsCollector,
	tracer trace.Tracer,
	config *TrainingConfig,
) TrainingService {
	service := &trainingService{
		taskRepo:         taskRepo,
		modelRepo:        modelRepo,
		rlhfEngine:       rlhfEngine,
		dpoEngine:        dpoEngine,
		messageQueue:     messageQueue,
		logger:           logger,
		metricsCollector: metricsCollector,
		tracer:           tracer,
		config:           config,
		taskQueue:        make(chan *TrainingTask, config.TaskQueueSize),
	}

	// 初始化工作池
	service.workerPool = &WorkerPool{
		workers:      config.MaxConcurrentTasks,
		taskQueue:    service.taskQueue,
		stopChan:     make(chan struct{}),
		trainingFunc: service.executeTraining,
	}

	// 启动工作协程
	service.workerPool.Start()

	return service
}

// CreateTask 创建训练任务
func (s *trainingService) CreateTask(ctx context.Context, req *TrainingRequest) (*TrainingTask, error) {
	ctx, span := s.tracer.Start(ctx, "TrainingService.CreateTask")
	defer span.End()

	startTime := time.Now()
	defer func() {
		// s.metricsCollector.ObserveDuration("training_create_task_duration_ms",
		// 	float64(time.Since(startTime).Milliseconds()),
		// 	map[string]string{"training_type": string(req.TrainingType)})
		_ = startTime // Suppress unused variable warning
	}()

 s.logger.WithContext(ctx).Info("Creating training task", logging.Any("model_id", req.ModelID), logging.Any("type", req.TrainingType))

	// 验证模型存在
	if _, err := // s.modelRepo.GetByID(ctx, req.ModelID); err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "model not found")
	}

	// 验证数据集
	if req.Dataset == nil || len(req.Dataset.Samples) == 0 {
		return nil, errors.NewInternalError(errors.CodeInvalidArgument, "dataset is required")
	}

	// 应用默认配置
	config := s.applyDefaultConfig(req.Config)

	// 创建任务
	task := &TrainingTask{
		ID:           generateTaskID(),
		ModelID:      req.ModelID,
		TrainingType: req.TrainingType,
		Status:       TaskStatusPending,
		Progress:     0.0,
		TotalEpochs:  s.getEpochs(config),
		Config:       config,
		Resources:    req.Resources,
		Checkpoints:  []*Checkpoint{},
		Metadata:     req.Metadata,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 保存任务
	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to create task")
	}

	s.metricsCollector.IncrementCounter("training_task_created",
		map[string]string{
			"model_id":      req.ModelID,
			"training_type": string(req.TrainingType),
		})

 s.logger.WithContext(ctx).Info("Training task created successfully", logging.Any("task_id", task.ID), logging.Any("model_id", req.ModelID))

	return task, nil
}

// StartTask 启动训练任务
func (s *trainingService) StartTask(ctx context.Context, taskID string) error {
	ctx, span := s.tracer.Start(ctx, "TrainingService.StartTask")
	defer span.End()

	s.logger.WithContext(ctx).Info("Starting training task", logging.Any("task_id", taskID))

	// 获取任务
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return errors.Wrap(err, errors.CodeNotFound, "task not found")
	}

	// 检查状态
	if task.Status != TaskStatusPending && task.Status != TaskStatusPaused {
		return errors.ValidationError(
			fmt.Sprintf("cannot start task with status: %s", task.Status))
	}

	// 更新状态
	now := time.Now()
	task.Status = TaskStatusRunning
	task.StartedAt = &now
	task.UpdatedAt = now

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to update task")
	}

	// 提交到任务队列
	select {
	case s.taskQueue <- task:
		s.runningTasks.Store(taskID, task)
		s.metricsCollector.IncrementCounter("training_task_started",
			map[string]string{"task_id": taskID})
		s.logger.WithContext(ctx).Info("Training task started", logging.Any("task_id", taskID))
	default:
		task.Status = TaskStatusPending
		s.taskRepo.Update(ctx, task)
		return errors.NewInternalError("ERR_EXHAUSTED", "task queue is full")
	}

	return nil
}

// StopTask 停止训练任务
func (s *trainingService) StopTask(ctx context.Context, taskID string) error {
	ctx, span := s.tracer.Start(ctx, "TrainingService.StopTask")
	defer span.End()

	s.logger.WithContext(ctx).Info("Stopping training task", logging.Any("task_id", taskID))

	// 获取任务
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return errors.Wrap(err, errors.CodeNotFound, "task not found")
	}

	// 检查状态
	if task.Status != TaskStatusRunning && task.Status != TaskStatusPaused {
		return errors.ValidationError(
			fmt.Sprintf("cannot stop task with status: %s", task.Status))
	}

	// 调用训练引擎停止
	engine := s.getEngine(task.TrainingType)
	if err := engine.Stop(ctx, taskID); err != nil {
		s.logger.WithContext(ctx).Warn("Failed to stop training engine", logging.Error(err))
	}

	// 更新状态
	now := time.Now()
	task.Status = TaskStatusCancelled
	task.CompletedAt = &now
	task.UpdatedAt = now

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to update task")
	}

	s.runningTasks.Delete(taskID)

	s.metricsCollector.IncrementCounter("training_task_stopped",
		map[string]string{"task_id": taskID})

	s.logger.WithContext(ctx).Info("Training task stopped", logging.Any("task_id", taskID))

	return nil
}

// PauseTask 暂停训练任务
func (s *trainingService) PauseTask(ctx context.Context, taskID string) error {
	ctx, span := s.tracer.Start(ctx, "TrainingService.PauseTask")
	defer span.End()

	s.logger.WithContext(ctx).Info("Pausing training task", logging.Any("task_id", taskID))

	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return errors.Wrap(err, errors.CodeNotFound, "task not found")
	}

	if task.Status != TaskStatusRunning {
		return errors.ValidationError(
			fmt.Sprintf("cannot pause task with status: %s", task.Status))
	}

	engine := s.getEngine(task.TrainingType)
	if err := engine.Pause(ctx, taskID); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to pause training")
	}

	task.Status = TaskStatusPaused
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to update task")
	}

	s.metricsCollector.IncrementCounter("training_task_paused",
		map[string]string{"task_id": taskID})

	s.logger.WithContext(ctx).Info("Training task paused", logging.Any("task_id", taskID))

	return nil
}

// ResumeTask 恢复训练任务
func (s *trainingService) ResumeTask(ctx context.Context, taskID string) error {
	ctx, span := s.tracer.Start(ctx, "TrainingService.ResumeTask")
	defer span.End()

	s.logger.WithContext(ctx).Info("Resuming training task", logging.Any("task_id", taskID))

	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return errors.Wrap(err, errors.CodeNotFound, "task not found")
	}

	if task.Status != TaskStatusPaused {
		return errors.ValidationError(
			fmt.Sprintf("cannot resume task with status: %s", task.Status))
	}

	engine := s.getEngine(task.TrainingType)
	if err := engine.Resume(ctx, taskID); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to resume training")
	}

	task.Status = TaskStatusRunning
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to update task")
	}

	s.metricsCollector.IncrementCounter("training_task_resumed",
		map[string]string{"task_id": taskID})

	s.logger.WithContext(ctx).Info("Training task resumed", logging.Any("task_id", taskID))

	return nil
}

// GetTask 获取训练任务
func (s *trainingService) GetTask(ctx context.Context, taskID string) (*TrainingTask, error) {
	ctx, span := s.tracer.Start(ctx, "TrainingService.GetTask")
	defer span.End()

	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "task not found")
	}

	return task, nil
}

// ListTasks 列出训练任务
func (s *trainingService) ListTasks(ctx context.Context, filter *TaskFilter) ([]*TrainingTask, error) {
	ctx, span := s.tracer.Start(ctx, "TrainingService.ListTasks")
	defer span.End()

	tasks, err := s.taskRepo.List(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to list tasks")
	}

	return tasks, nil
}

// DeleteTask 删除训练任务
func (s *trainingService) DeleteTask(ctx context.Context, taskID string) error {
	ctx, span := s.tracer.Start(ctx, "TrainingService.DeleteTask")
	defer span.End()

	s.logger.WithContext(ctx).Info("Deleting training task", logging.Any("task_id", taskID))

	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return errors.Wrap(err, errors.CodeNotFound, "task not found")
	}

	if task.Status == TaskStatusRunning {
		return errors.NewInternalError(errors.CodeInvalidArgument, "cannot delete running task")
	}

	if err := s.taskRepo.Delete(ctx, taskID); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to delete task")
	}

	s.metricsCollector.IncrementCounter("training_task_deleted",
		map[string]string{"task_id": taskID})

	s.logger.WithContext(ctx).Info("Training task deleted", logging.Any("task_id", taskID))

	return nil
}

// GetTaskMetrics 获取任务指标
func (s *trainingService) GetTaskMetrics(ctx context.Context, taskID string) (*TaskMetrics, error) {
	ctx, span := s.tracer.Start(ctx, "TrainingService.GetTaskMetrics")
	defer span.End()

	taskMetrics, err := s.taskRepo.GetMetrics(ctx, taskID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "metrics not found")
	}

	return taskMetrics, nil
}

// GetTaskLogs 获取任务日志
func (s *trainingService) GetTaskLogs(ctx context.Context, taskID string, limit int) ([]string, error) {
	ctx, span := s.tracer.Start(ctx, "TrainingService.GetTaskLogs")
	defer span.End()

	// 简化实现：实际应从日志系统查询
	logs := []string{
		fmt.Sprintf("[%s] Training task %s started", time.Now().Format(time.RFC3339), taskID),
		fmt.Sprintf("[%s] Loading dataset...", time.Now().Format(time.RFC3339)),
		fmt.Sprintf("[%s] Training in progress...", time.Now().Format(time.RFC3339)),
	}

	if limit > 0 && len(logs) > limit {
		logs = logs[len(logs)-limit:]
	}

	return logs, nil
}

// executeTraining 执行训练
func (s *trainingService) executeTraining(ctx context.Context, task *TrainingTask) error {
	s.logger.WithContext(ctx).Info("Executing training", logging.Any("task_id", task.ID))

	// 获取数据集（简化实现）
	dataset := &TrainingDataset{
		ID:          task.ID,
		SampleCount: 1000,
		SplitRatio: &SplitRatio{
			Train:      0.8,
			Validation: 0.1,
			Test:       0.1,
		},
	}

	// 选择训练引擎
	engine := s.getEngine(task.TrainingType)

	// 执行训练
	if err := engine.Train(ctx, task, dataset); err != nil {
		s.handleTrainingError(ctx, task, err)
		return err
	}

	// 更新任务状态
	now := time.Now()
	task.Status = TaskStatusCompleted
	task.CompletedAt = &now
	task.Progress = 1.0

	if err := s.taskRepo.Update(ctx, task); err != nil {
		s.logger.WithContext(ctx).Error("Failed to update task after completion", logging.Error(err))
	}

	s.runningTasks.Delete(task.ID)

	s.metricsCollector.IncrementCounter("training_task_completed",
		map[string]string{"task_id": task.ID})

	s.logger.WithContext(ctx).Info("Training completed successfully", logging.Any("task_id", task.ID))

	return nil
}

// handleTrainingError 处理训练错误
func (s *trainingService) handleTrainingError(ctx context.Context, task *TrainingTask, err error) {
	s.logger.WithContext(ctx).Error("Training failed", logging.Any("task_id", task.ID), logging.Error(err))

	now := time.Now()
	task.Status = TaskStatusFailed
	task.CompletedAt = &now
	task.ErrorMessage = err.Error()

	if updateErr := s.taskRepo.Update(ctx, task); updateErr != nil {
		s.logger.WithContext(ctx).Error("Failed to update task after error", logging.Error(updateErr))
	}

	s.runningTasks.Delete(task.ID)

	s.metricsCollector.IncrementCounter("training_task_failed",
		map[string]string{"task_id": task.ID})
}

// Helper functions

func (s *trainingService) getEngine(trainingType types.TrainingType) TrainingEngine {
	switch trainingType {
	case types.TrainingTypeRLHF:
		return s.rlhfEngine
	case types.TrainingTypeDPO:
		return s.dpoEngine
	default:
		return s.dpoEngine
	}
}

func (s *trainingService) applyDefaultConfig(config map[string]interface{}) map[string]interface{} {
	if config == nil {
		config = make(map[string]interface{})
	}

	if _, exists := config["epochs"]; !exists {
		config["epochs"] = s.config.DefaultEpochs
	}
	if _, exists := config["batch_size"]; !exists {
		config["batch_size"] = s.config.DefaultBatchSize
	}
	if _, exists := config["learning_rate"]; !exists {
		config["learning_rate"] = s.config.DefaultLearningRate
	}

	return config
}

func (s *trainingService) getEpochs(config map[string]interface{}) int {
	if epochs, ok := config["epochs"].(float64); ok {
		return int(epochs)
	}
	return s.config.DefaultEpochs
}

// WorkerPool methods

func (w *WorkerPool) Start() {
	for i := 0; i < w.workers; i++ {
		w.wg.Add(1)
		go w.worker()
	}
}

func (w *WorkerPool) Stop() {
	close(w.stopChan)
	w.wg.Wait()
}

func (w *WorkerPool) worker() {
	defer w.wg.Done()

	for {
		select {
		case task := <-w.taskQueue:
			w.trainingFunc(context.Background(), task)
		case <-w.stopChan:
			return
		}
	}
}

// Utility functions

func generateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}

//Personal.AI order the ending
