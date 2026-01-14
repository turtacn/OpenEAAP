package orchestrator

import (
	"context"
	"container/heap"
	"fmt"
	"sync"
	"time"

	"github.com/openeeap/openeeap/pkg/errors"
	"github.com/openeeap/openeeap/pkg/logger"
)

// Scheduler 调度器接口
type Scheduler interface {
	// Schedule 调度任务
	Schedule(ctx context.Context, req *ScheduleRequest) (*Task, error)

	// Cancel 取消任务
	Cancel(ctx context.Context, taskID string) error

	// Retry 重试任务
	Retry(ctx context.Context, taskID string) error

	// GetTask 获取任务
	GetTask(ctx context.Context, taskID string) (*Task, error)

	// ListTasks 列出任务
	ListTasks(ctx context.Context, filter *TaskFilter) ([]*Task, error)

	// GetMetrics 获取调度器指标
	GetMetrics(ctx context.Context) (*SchedulerMetrics, error)

	// Start 启动调度器
	Start() error

	// Stop 停止调度器
	Stop() error
}

// scheduler 调度器实现
type scheduler struct {
	config        *SchedulerConfig
	logger        logger.Logger

	taskQueue     *priorityQueue
	taskMap       map[string]*Task
	taskMapMu     sync.RWMutex

	workers       []*worker
	workerPool    chan *worker

	metrics       *schedulerMetrics

	running       bool
	runningMu     sync.RWMutex

	closeChan     chan struct{}
	wg            sync.WaitGroup
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	MaxWorkers             int           // 最大工作线程数
	MaxQueueSize           int           // 最大队列大小
	DefaultTimeout         time.Duration // 默认超时时间
	MaxRetries             int           // 最大重试次数
	RetryDelay             time.Duration // 重试延迟
	EnablePriorityScheduling bool        // 启用优先级调度
	EnableConcurrencyControl bool        // 启用并发控制
	MaxConcurrentTasks     int           // 最大并发任务数
	TaskCleanupInterval    time.Duration // 任务清理间隔
	CompletedTaskRetention time.Duration // 已完成任务保留时间
	FailedTaskRetention    time.Duration // 失败任务保留时间
	EnableMetrics          bool          // 启用指标
	EnableTaskHistory      bool          // 启用任务历史
}

// ScheduleRequest 调度请求
type ScheduleRequest struct {
	Request      *ExecuteRequest        // 执行请求
	Route        *RouteResponse         // 路由响应
	Priority     int                    // 优先级
	Timeout      time.Duration          // 超时时间
	MaxRetries   int                    // 最大重试次数
	RetryPolicy  RetryPolicy            // 重试策略
	Metadata     map[string]interface{} // 元数据
}

// Task 任务
type Task struct {
	ID              string                 // 任务ID
	Type            TaskType               // 任务类型
	Status          TaskStatus             // 任务状态
	Priority        int                    // 优先级
	Request         *ExecuteRequest        // 执行请求
	Route           *RouteResponse         // 路由响应
	Timeout         time.Duration          // 超时时间
	MaxRetries      int                    // 最大重试次数
	CurrentRetry    int                    // 当前重试次数
	RetryPolicy     RetryPolicy            // 重试策略
	Result          interface{}            // 结果
	Error           error                  // 错误
	CreatedAt       time.Time              // 创建时间
	ScheduledAt     time.Time              // 调度时间
	StartedAt       time.Time              // 开始时间
	CompletedAt     time.Time              // 完成时间
	Duration        time.Duration          // 执行时长
	WorkerID        string                 // 工作线程ID
	Metadata        map[string]interface{} // 元数据
	CancelFunc      context.CancelFunc     // 取消函数
	ctx             context.Context        // 上下文
	queueIndex      int                    // 队列索引
}

// TaskType 任务类型
type TaskType string

const (
	TaskTypeExecution  TaskType = "execution"  // 执行任务
	TaskTypeRetry      TaskType = "retry"      // 重试任务
	TaskTypeScheduled  TaskType = "scheduled"  // 定时任务
	TaskTypeBackground TaskType = "background" // 后台任务
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"    // 待处理
	TaskStatusQueued     TaskStatus = "queued"     // 已入队
	TaskStatusRunning    TaskStatus = "running"    // 运行中
	TaskStatusCompleted  TaskStatus = "completed"  // 已完成
	TaskStatusFailed     TaskStatus = "failed"     // 失败
	TaskStatusCancelled  TaskStatus = "cancelled"  // 已取消
	TaskStatusTimeout    TaskStatus = "timeout"    // 超时
	TaskStatusRetrying   TaskStatus = "retrying"   // 重试中
)

// RetryPolicy 重试策略
type RetryPolicy string

const (
	RetryPolicyNone        RetryPolicy = "none"        // 不重试
	RetryPolicyImmediate   RetryPolicy = "immediate"   // 立即重试
	RetryPolicyLinear      RetryPolicy = "linear"      // 线性延迟
	RetryPolicyExponential RetryPolicy = "exponential" // 指数退避
	RetryPolicyCustom      RetryPolicy = "custom"      // 自定义
)

// TaskFilter 任务过滤器
type TaskFilter struct {
	Status      []TaskStatus // 状态列表
	Priority    *int         // 优先级
	Type        TaskType     // 类型
	UserID      string       // 用户ID
	CreatedFrom time.Time    // 创建起始时间
	CreatedTo   time.Time    // 创建结束时间
	Limit       int          // 限制数量
	Offset      int          // 偏移量
}

// SchedulerMetrics 调度器指标
type SchedulerMetrics struct {
	TotalTasks          int64         // 总任务数
	CompletedTasks      int64         // 已完成任务数
	FailedTasks         int64         // 失败任务数
	CancelledTasks      int64         // 已取消任务数
	TimeoutTasks        int64         // 超时任务数
	RunningTasks        int64         // 运行中任务数
	QueuedTasks         int64         // 队列中任务数
	AverageWaitTime     time.Duration // 平均等待时间
	AverageExecutionTime time.Duration // 平均执行时间
	WorkerUtilization   float64       // 工作线程利用率
	QueueUtilization    float64       // 队列利用率
	ThroughputPerSecond float64       // 每秒吞吐量
}

// schedulerMetrics 内部指标
type schedulerMetrics struct {
	totalTasks       int64
	completedTasks   int64
	failedTasks      int64
	cancelledTasks   int64
	timeoutTasks     int64
	totalWaitTime    time.Duration
	totalExecTime    time.Duration
	mu               sync.RWMutex
}

// worker 工作线程
type worker struct {
	id        string
	scheduler *scheduler
	taskChan  chan *Task
	running   bool
	mu        sync.RWMutex
}

// priorityQueue 优先级队列
type priorityQueue struct {
	tasks []*Task
	mu    sync.RWMutex
}

// NewScheduler 创建调度器
func NewScheduler(config *SchedulerConfig, logger logger.Logger) (Scheduler, error) {
	if config == nil {
		config = &SchedulerConfig{
			MaxWorkers:               10,
			MaxQueueSize:             1000,
			DefaultTimeout:           5 * time.Minute,
			MaxRetries:               3,
			RetryDelay:               1 * time.Second,
			EnablePriorityScheduling: true,
			EnableConcurrencyControl: true,
			MaxConcurrentTasks:       100,
			TaskCleanupInterval:      10 * time.Minute,
			CompletedTaskRetention:   1 * time.Hour,
			FailedTaskRetention:      24 * time.Hour,
			EnableMetrics:            true,
			EnableTaskHistory:        true,
		}
	}

	s := &scheduler{
		config:     config,
		logger:     logger,
		taskQueue:  &priorityQueue{tasks: make([]*Task, 0)},
		taskMap:    make(map[string]*Task),
		workerPool: make(chan *worker, config.MaxWorkers),
		metrics:    &schedulerMetrics{},
		closeChan:  make(chan struct{}),
	}

	// 初始化工作线程
	s.workers = make([]*worker, config.MaxWorkers)
	for i := 0; i < config.MaxWorkers; i++ {
		w := &worker{
			id:        fmt.Sprintf("worker-%d", i),
			scheduler: s,
			taskChan:  make(chan *Task, 1),
		}
		s.workers[i] = w
	}

	return s, nil
}

// Schedule 调度任务
func (s *scheduler) Schedule(ctx context.Context, req *ScheduleRequest) (*Task, error) {
	if req == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "schedule request cannot be nil")
	}

	s.runningMu.RLock()
	if !s.running {
		s.runningMu.RUnlock()
		return nil, errors.New(errors.CodeInternalError, "scheduler not running")
	}
	s.runningMu.RUnlock()

	// 检查队列大小
	if s.config.EnableConcurrencyControl {
		queueSize := s.taskQueue.Len()
		if queueSize >= s.config.MaxQueueSize {
			return nil, errors.New(errors.CodeResourceExhausted, "task queue is full")
		}
	}

	// 创建任务
	task := &Task{
		ID:           s.generateTaskID(),
		Type:         TaskTypeExecution,
		Status:       TaskStatusPending,
		Priority:     req.Priority,
		Request:      req.Request,
		Route:        req.Route,
		Timeout:      req.Timeout,
		MaxRetries:   req.MaxRetries,
		CurrentRetry: 0,
		RetryPolicy:  req.RetryPolicy,
		CreatedAt:    time.Now(),
		Metadata:     req.Metadata,
	}

	// 设置默认值
	if task.Timeout == 0 {
		task.Timeout = s.config.DefaultTimeout
	}
	if task.MaxRetries == 0 {
		task.MaxRetries = s.config.MaxRetries
	}
	if task.RetryPolicy == "" {
		task.RetryPolicy = RetryPolicyLinear
	}

	// 创建任务上下文
	taskCtx, cancel := context.WithTimeout(ctx, task.Timeout)
	task.ctx = taskCtx
	task.CancelFunc = cancel

	// 存储任务
	s.taskMapMu.Lock()
	s.taskMap[task.ID] = task
	s.taskMapMu.Unlock()

	// 加入队列
	task.Status = TaskStatusQueued
	task.ScheduledAt = time.Now()
	s.taskQueue.Push(task)

	// 更新指标
	s.metrics.mu.Lock()
	s.metrics.totalTasks++
	s.metrics.mu.Unlock()

	s.logger.Debug("task scheduled",
		"task_id", task.ID,
		"priority", task.Priority,
		"timeout", task.Timeout)

	return task, nil
}

// Cancel 取消任务
func (s *scheduler) Cancel(ctx context.Context, taskID string) error {
	if taskID == "" {
		return errors.New(errors.CodeInvalidParameter, "task id cannot be empty")
	}

	s.taskMapMu.RLock()
	task, exists := s.taskMap[taskID]
	s.taskMapMu.RUnlock()

	if !exists {
		return errors.New(errors.CodeNotFound, "task not found")
	}

	// 取消任务
	if task.CancelFunc != nil {
		task.CancelFunc()
	}

	task.Status = TaskStatusCancelled
	task.CompletedAt = time.Now()
	task.Duration = task.CompletedAt.Sub(task.StartedAt)

	// 更新指标
	s.metrics.mu.Lock()
	s.metrics.cancelledTasks++
	s.metrics.mu.Unlock()

	s.logger.Info("task cancelled", "task_id", taskID)
	return nil
}

// Retry 重试任务
func (s *scheduler) Retry(ctx context.Context, taskID string) error {
	if taskID == "" {
		return errors.New(errors.CodeInvalidParameter, "task id cannot be empty")
	}

	s.taskMapMu.RLock()
	task, exists := s.taskMap[taskID]
	s.taskMapMu.RUnlock()

	if !exists {
		return errors.New(errors.CodeNotFound, "task not found")
	}

	// 检查是否可以重试
	if task.CurrentRetry >= task.MaxRetries {
		return errors.New(errors.CodeInvalidParameter, "max retries exceeded")
	}

	// 创建重试任务
	retryTask := &Task{
		ID:           s.generateTaskID(),
		Type:         TaskTypeRetry,
		Status:       TaskStatusPending,
		Priority:     task.Priority,
		Request:      task.Request,
		Route:        task.Route,
		Timeout:      task.Timeout,
		MaxRetries:   task.MaxRetries,
		CurrentRetry: task.CurrentRetry + 1,
		RetryPolicy:  task.RetryPolicy,
		CreatedAt:    time.Now(),
		Metadata:     task.Metadata,
	}

	// 计算重试延迟
	delay := s.calculateRetryDelay(retryTask)
	time.Sleep(delay)

	// 创建任务上下文
	taskCtx, cancel := context.WithTimeout(ctx, retryTask.Timeout)
	retryTask.ctx = taskCtx
	retryTask.CancelFunc = cancel

	// 存储任务
	s.taskMapMu.Lock()
	s.taskMap[retryTask.ID] = retryTask
	s.taskMapMu.Unlock()

	// 加入队列
	retryTask.Status = TaskStatusQueued
	retryTask.ScheduledAt = time.Now()
	s.taskQueue.Push(retryTask)

	s.logger.Info("task retrying",
		"task_id", taskID,
		"retry_task_id", retryTask.ID,
		"retry_count", retryTask.CurrentRetry)

	return nil
}

// GetTask 获取任务
func (s *scheduler) GetTask(ctx context.Context, taskID string) (*Task, error) {
	if taskID == "" {
		return nil, errors.New(errors.CodeInvalidParameter, "task id cannot be empty")
	}

	s.taskMapMu.RLock()
	task, exists := s.taskMap[taskID]
	s.taskMapMu.RUnlock()

	if !exists {
		return nil, errors.New(errors.CodeNotFound, "task not found")
	}

	return task, nil
}

// ListTasks 列出任务
func (s *scheduler) ListTasks(ctx context.Context, filter *TaskFilter) ([]*Task, error) {
	s.taskMapMu.RLock()
	defer s.taskMapMu.RUnlock()

	var tasks []*Task
	for _, task := range s.taskMap {
		if s.matchesFilter(task, filter) {
			tasks = append(tasks, task)
		}
	}

	// 应用分页
	if filter != nil {
		if filter.Offset > 0 && filter.Offset < len(tasks) {
			tasks = tasks[filter.Offset:]
		}
		if filter.Limit > 0 && filter.Limit < len(tasks) {
			tasks = tasks[:filter.Limit]
		}
	}

	return tasks, nil
}

// GetMetrics 获取调度器指标
func (s *scheduler) GetMetrics(ctx context.Context) (*SchedulerMetrics, error) {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	queueSize := s.taskQueue.Len()
	runningTasks := s.countRunningTasks()

	avgWaitTime := time.Duration(0)
	if s.metrics.totalTasks > 0 {
		avgWaitTime = time.Duration(int64(s.metrics.totalWaitTime) / s.metrics.totalTasks)
	}

	avgExecTime := time.Duration(0)
	if s.metrics.completedTasks > 0 {
		avgExecTime = time.Duration(int64(s.metrics.totalExecTime) / s.metrics.completedTasks)
	}

	workerUtilization := 0.0
	if s.config.MaxWorkers > 0 {
		workerUtilization = float64(runningTasks) / float64(s.config.MaxWorkers)
	}

	queueUtilization := 0.0
	if s.config.MaxQueueSize > 0 {
		queueUtilization = float64(queueSize) / float64(s.config.MaxQueueSize)
	}

	return &SchedulerMetrics{
		TotalTasks:           s.metrics.totalTasks,
		CompletedTasks:       s.metrics.completedTasks,
		FailedTasks:          s.metrics.failedTasks,
		CancelledTasks:       s.metrics.cancelledTasks,
		TimeoutTasks:         s.metrics.timeoutTasks,
		RunningTasks:         int64(runningTasks),
		QueuedTasks:          int64(queueSize),
		AverageWaitTime:      avgWaitTime,
		AverageExecutionTime: avgExecTime,
		WorkerUtilization:    workerUtilization,
		QueueUtilization:     queueUtilization,
	}, nil
}

// Start 启动调度器
func (s *scheduler) Start() error {
	s.runningMu.Lock()
	if s.running {
		s.runningMu.Unlock()
		return errors.New(errors.CodeInternalError, "scheduler already running")
	}
	s.running = true
	s.runningMu.Unlock()

	// 启动工作线程
	for _, w := range s.workers {
		s.wg.Add(1)
		go w.start()
	}

	// 启动调度循环
	s.wg.Add(1)
	go s.scheduleLoop()

	// 启动清理循环
	if s.config.TaskCleanupInterval > 0 {
		s.wg.Add(1)
		go s.cleanupLoop()
	}

	s.logger.Info("scheduler started",
		"max_workers", s.config.MaxWorkers,
		"max_queue_size", s.config.MaxQueueSize)

	return nil
}

// Stop 停止调度器
func (s *scheduler) Stop() error {
	s.runningMu.Lock()
	if !s.running {
		s.runningMu.Unlock()
		return errors.New(errors.CodeInternalError, "scheduler not running")
	}
	s.running = false
	s.runningMu.Unlock()

	// 关闭信号通道
	close(s.closeChan)

	// 等待所有任务完成
	s.wg.Wait()

	// 停止所有工作线程
	for _, w := range s.workers {
		w.stop()
	}

	s.logger.Info("scheduler stopped")
	return nil
}

// scheduleLoop 调度循环
func (s *scheduler) scheduleLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.dispatch()
		case <-s.closeChan:
			return
		}
	}
}

// dispatch 分发任务
func (s *scheduler) dispatch() {
	// 检查是否有可用工作线程
	select {
	case worker := <-s.workerPool:
		// 从队列获取任务
		task := s.taskQueue.Pop()
		if task == nil {
			// 没有任务，归还工作线程
			s.workerPool <- worker
			return
		}

		// 分配任务给工作线程
		task.WorkerID = worker.id
		task.Status = TaskStatusRunning
		task.StartedAt = time.Now()

		// 计算等待时间
		waitTime := task.StartedAt.Sub(task.ScheduledAt)
		s.metrics.mu.Lock()
		s.metrics.totalWaitTime += waitTime
		s.metrics.mu.Unlock()

		// 发送任务
		select {
		case worker.taskChan <- task:
		default:
			// 工作线程繁忙，重新入队
			s.taskQueue.Push(task)
			s.workerPool <- worker
		}
	default:
		// 没有可用工作线程
		return
	}
}

// cleanupLoop 清理循环
func (s *scheduler) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.TaskCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanup()
		case <-s.closeChan:
			return
		}
	}
}

// cleanup 清理过期任务
func (s *scheduler) cleanup() {
	now := time.Now()

	s.taskMapMu.Lock()
	defer s.taskMapMu.Unlock()

	for id, task := range s.taskMap {
		shouldRemove := false

		switch task.Status {
		case TaskStatusCompleted:
			if now.Sub(task.CompletedAt) > s.config.CompletedTaskRetention {
				shouldRemove = true
			}
		case TaskStatusFailed, TaskStatusCancelled, TaskStatusTimeout:
			if now.Sub(task.CompletedAt) > s.config.FailedTaskRetention {
				shouldRemove = true
			}
		}

		if shouldRemove {
			delete(s.taskMap, id)
			s.logger.Debug("task cleaned up", "task_id", id, "status", task.Status)
		}
	}
}

// calculateRetryDelay 计算重试延迟
func (s *scheduler) calculateRetryDelay(task *Task) time.Duration {
	switch task.RetryPolicy {
	case RetryPolicyNone:
		return 0
	case RetryPolicyImmediate:
		return 0
	case RetryPolicyLinear:
		return time.Duration(task.CurrentRetry) * s.config.RetryDelay
	case RetryPolicyExponential:
		delay := s.config.RetryDelay
		for i := 0; i < task.CurrentRetry; i++ {
			delay *= 2
		}
		return delay
	default:
		return s.config.RetryDelay
	}
}

// matchesFilter 检查任务是否匹配过滤器
func (s *scheduler) matchesFilter(task *Task, filter *TaskFilter) bool {
	if filter == nil {
		return true
	}

	// 检查状态
	if len(filter.Status) > 0 {
		found := false
		for _, status := range filter.Status {
			if task.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 检查优先级
	if filter.Priority != nil && task.Priority != *filter.Priority {
		return false
	}

	// 检查类型
	if filter.Type != "" && task.Type != filter.Type {
		return false
	}

	// 检查用户ID
	if filter.UserID != "" && task.Request.UserID != filter.UserID {
		return false
	}

	// 检查创建时间
	if !filter.CreatedFrom.IsZero() && task.CreatedAt.Before(filter.CreatedFrom) {
		return false
	}
	if !filter.CreatedTo.IsZero() && task.CreatedAt.After(filter.CreatedTo) {
		return false
	}

	return true
}

// countRunningTasks 统计运行中的任务数
func (s *scheduler) countRunningTasks() int {
	s.taskMapMu.RLock()
	defer s.taskMapMu.RUnlock()

	count := 0
	for _, task := range s.taskMap {
		if task.Status == TaskStatusRunning {
			count++
		}
	}
	return count
}

// generateTaskID 生成任务ID
func (s *scheduler) generateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}

// worker方法

// start 启动工作线程
func (w *worker) start() {
	defer w.scheduler.wg.Done()

	w.mu.Lock()
	w.running = true
	w.mu.Unlock()

	// 注册为可用工作线程
	w.scheduler.workerPool <- w

	for {
		select {
		case task := <-w.taskChan:
			w.executeTask(task)
			// 任务完成后重新注册为可用
			w.scheduler.workerPool <- w
		case <-w.scheduler.closeChan:
			return
		}
	}
}

// stop 停止工作线程
func (w *worker) stop() {
	w.mu.Lock()
	w.running = false
	w.mu.Unlock()
}

// executeTask 执行任务
func (w *worker) executeTask(task *Task) {
	defer func() {
		if r := recover(); r != nil {
			task.Status = TaskStatusFailed
			task.Error = fmt.Errorf("panic: %v", r)
			task.CompletedAt = time.Now()
			task.Duration = task.CompletedAt.Sub(task.StartedAt)

			w.scheduler.metrics.mu.Lock()
			w.scheduler.metrics.failedTasks++
			w.scheduler.metrics.mu.Unlock()

			w.scheduler.logger.Error("task panicked",
				"task_id", task.ID,
				"worker_id", w.id,
				"error", task.Error)
		}
	}()

	// 执行任务（这里需要调用实际的执行器）
	// 简化处理，实际应该调用 executor
	select {
	case <-task.ctx.Done():
		// 任务被取消或超时
		if task.ctx.Err() == context.DeadlineExceeded {
			task.Status = TaskStatusTimeout
			w.scheduler.metrics.mu.Lock()
			w.scheduler.metrics.timeoutTasks++
			w.scheduler.metrics.mu.Unlock()
		} else {
			task.Status = TaskStatusCancelled
			w.scheduler.metrics.mu.Lock()
			w.scheduler.metrics.cancelledTasks++
			w.scheduler.metrics.mu.Unlock()
		}
	case <-time.After(100 * time.Millisecond): // 模拟执行
		task.Status = TaskStatusCompleted
		w.scheduler.metrics.mu.Lock()
		w.scheduler.metrics.completedTasks++
		w.scheduler.metrics.totalExecTime += task.Duration
		w.scheduler.metrics.mu.Unlock()
	}

	task.CompletedAt = time.Now()
	task.Duration = task.CompletedAt.Sub(task.StartedAt)

	// 如果失败且可以重试，加入重试队列
	if task.Status == TaskStatusFailed && task.CurrentRetry < task.MaxRetries {
		w.scheduler.Retry(context.Background(), task.ID)
	}

	w.scheduler.logger.Debug("task completed",
		"task_id", task.ID,
		"worker_id", w.id,
		"status", task.Status,
		"duration", task.Duration)
}

// 优先级队列实现

func (pq *priorityQueue) Len() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return len(pq.tasks)
}

func (pq *priorityQueue) Less(i, j int) bool {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	// 优先级高的在前
	return pq.tasks[i].Priority > pq.tasks[j].Priority
}

func (pq *priorityQueue) Swap(i, j int) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.tasks[i], pq.tasks[j] = pq.tasks[j], pq.tasks[i]
	pq.tasks[i].queueIndex = i
	pq.tasks[j].queueIndex = j
}

func (pq *priorityQueue) Push(task *Task) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	n := len(pq.tasks)
	task.queueIndex = n
	pq.tasks = append(pq.tasks, task)
	heap.Fix(pq, n)
}

func (pq *priorityQueue) Pop() *Task {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if len(pq.tasks) == 0 {
		return nil
	}
	old := pq.tasks
	n := len(old)
	task := old[n-1]
	old[n-1] = nil
	task.queueIndex = -1
	pq.tasks = old[0 : n-1]
	return task
}

//Personal.AI order the ending
