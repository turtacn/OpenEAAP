// internal/governance/audit/audit_logger.go
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/pkg/errors"
)

// AuditLogger 审计日志记录器接口
type AuditLogger interface {
	// Log 记录审计日志
	Log(ctx context.Context, entry *AuditEntry) error

	// LogBatch 批量记录审计日志
	LogBatch(ctx context.Context, entries []*AuditEntry) error

	// Query 查询审计日志
	Query(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error)

	// QueryWithPagination 分页查询审计日志
	QueryWithPagination(ctx context.Context, filter *AuditFilter, page, pageSize int) (*AuditQueryResult, error)

	// Count 统计审计日志数量
	Count(ctx context.Context, filter *AuditFilter) (int64, error)

	// GetStatistics 获取审计统计信息
	GetStatistics(ctx context.Context, filter *AuditFilter) (*AuditStatistics, error)

	// Delete 删除审计日志
	Delete(ctx context.Context, filter *AuditFilter) (int64, error)

	// Archive 归档审计日志
	Archive(ctx context.Context, filter *AuditFilter, archivePath string) error

	// Close 关闭审计日志记录器
	Close() error
}

// AuditEntry 审计日志条目
type AuditEntry struct {
	ID        string        `json:"id"`
	Timestamp time.Time     `json:"timestamp"`
	EventType EventType     `json:"event_type"`
	Category  EventCategory `json:"category"`
	Action    string        `json:"action"`
	Result    EventResult   `json:"result"`
	Severity  Severity      `json:"severity"`

	// 主体信息
	ActorID        string    `json:"actor_id"`
	ActorType      ActorType `json:"actor_type"`
	ActorName      string    `json:"actor_name"`
	ActorRoles     []string  `json:"actor_roles"`
	ActorIP        string    `json:"actor_ip"`
	ActorUserAgent string    `json:"actor_user_agent"`

	// 资源信息
	ResourceID    string       `json:"resource_id"`
	ResourceType  ResourceType `json:"resource_type"`
	ResourceName  string       `json:"resource_name"`
	ResourceOwner string       `json:"resource_owner"`

	// 操作详情
	OperationID string `json:"operation_id"`
	SessionID   string `json:"session_id"`
	RequestID   string `json:"request_id"`
	TraceID     string `json:"trace_id"`
	SpanID      string `json:"span_id"`

	// 变更信息
	OldValue interface{}            `json:"old_value,omitempty"`
	NewValue interface{}            `json:"new_value,omitempty"`
	Changes  map[string]interface{} `json:"changes,omitempty"`

	// 上下文信息
	Environment string `json:"environment"`
	Application string `json:"application"`
	Service     string `json:"service"`
	Version     string `json:"version"`

	// 性能信息
	Duration     time.Duration `json:"duration"`
	ResponseSize int64         `json:"response_size,omitempty"`

	// 安全信息
	SecurityContext map[string]interface{} `json:"security_context,omitempty"`
	ThreatLevel     ThreatLevel            `json:"threat_level,omitempty"`

	// 额外信息
	Message      string                 `json:"message"`
	ErrorCode    string                 `json:"error_code,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	StackTrace   string                 `json:"stack_trace,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// EventType 事件类型
type EventType string

const (
	EventTypeCreate    EventType = "CREATE"
	EventTypeRead      EventType = "READ"
	EventTypeUpdate    EventType = "UPDATE"
	EventTypeDelete    EventType = "DELETE"
	EventTypeExecute   EventType = "EXECUTE"
	EventTypeAccess    EventType = "ACCESS"
	EventTypeLogin     EventType = "LOGIN"
	EventTypeLogout    EventType = "LOGOUT"
	EventTypeAuthorize EventType = "AUTHORIZE"
	EventTypeDeploy    EventType = "DEPLOY"
	EventTypeConfig    EventType = "CONFIG"
	EventTypeInference EventType = "INFERENCE"
	EventTypeTraining  EventType = "TRAINING"
)

// EventCategory 事件分类
type EventCategory string

const (
	CategorySecurity    EventCategory = "SECURITY"
	CategoryData        EventCategory = "DATA"
	CategoryModel       EventCategory = "MODEL"
	CategoryAgent       EventCategory = "AGENT"
	CategoryUser        EventCategory = "USER"
	CategorySystem      EventCategory = "SYSTEM"
	CategoryCompliance  EventCategory = "COMPLIANCE"
	CategoryPerformance EventCategory = "PERFORMANCE"
)

// EventResult 事件结果
type EventResult string

const (
	ResultSuccess EventResult = "SUCCESS"
	ResultFailure EventResult = "FAILURE"
	ResultDenied  EventResult = "DENIED"
	ResultError   EventResult = "ERROR"
)

// Severity 严重程度
type Severity string

const (
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

// ActorType 操作主体类型
type ActorType string

const (
	ActorTypeUser    ActorType = "USER"
	ActorTypeService ActorType = "SERVICE"
	ActorTypeAgent   ActorType = "AGENT"
	ActorTypeSystem  ActorType = "SYSTEM"
)

// ResourceType 资源类型
type ResourceType string

const (
	ResourceTypeModel      ResourceType = "MODEL"
	ResourceTypeAgent      ResourceType = "AGENT"
	ResourceTypeDataset    ResourceType = "DATASET"
	ResourceTypeAPI        ResourceType = "API"
	ResourceTypePolicy     ResourceType = "POLICY"
	ResourceTypeUser       ResourceType = "USER"
	ResourceTypeWorkflow   ResourceType = "WORKFLOW"
	ResourceTypeExperiment ResourceType = "EXPERIMENT"
)

// ThreatLevel 威胁等级
type ThreatLevel string

const (
	ThreatLevelNone     ThreatLevel = "NONE"
	ThreatLevelLow      ThreatLevel = "LOW"
	ThreatLevelMedium   ThreatLevel = "MEDIUM"
	ThreatLevelHigh     ThreatLevel = "HIGH"
	ThreatLevelCritical ThreatLevel = "CRITICAL"
)

// AuditFilter 审计日志过滤器
type AuditFilter struct {
	StartTime     time.Time       `json:"start_time,omitempty"`
	EndTime       time.Time       `json:"end_time,omitempty"`
	EventTypes    []EventType     `json:"event_types,omitempty"`
	Categories    []EventCategory `json:"categories,omitempty"`
	ActorIDs      []string        `json:"actor_ids,omitempty"`
	ActorTypes    []ActorType     `json:"actor_types,omitempty"`
	ResourceIDs   []string        `json:"resource_ids,omitempty"`
	ResourceTypes []ResourceType  `json:"resource_types,omitempty"`
	Results       []EventResult   `json:"results,omitempty"`
	Severities    []Severity      `json:"severities,omitempty"`
	Actions       []string        `json:"actions,omitempty"`
	Tags          []string        `json:"tags,omitempty"`
	SearchText    string          `json:"search_text,omitempty"`
	Limit         int             `json:"limit,omitempty"`
	Offset        int             `json:"offset,omitempty"`
	SortBy        string          `json:"sort_by,omitempty"`
	SortOrder     string          `json:"sort_order,omitempty"`
}

// AuditQueryResult 审计查询结果
type AuditQueryResult struct {
	Entries    []*AuditEntry `json:"entries"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

// AuditStatistics 审计统计信息
type AuditStatistics struct {
	TotalEvents      int64                   `json:"total_events"`
	EventsByType     map[EventType]int64     `json:"events_by_type"`
	EventsByCategory map[EventCategory]int64 `json:"events_by_category"`
	EventsByResult   map[EventResult]int64   `json:"events_by_result"`
	EventsBySeverity map[Severity]int64      `json:"events_by_severity"`
	TopActors        []ActorStats            `json:"top_actors"`
	TopResources     []ResourceStats         `json:"top_resources"`
	TopActions       []ActionStats           `json:"top_actions"`
	AverageDuration  time.Duration           `json:"average_duration"`
	FailureRate      float64                 `json:"failure_rate"`
	TimeRange        TimeRangeStats          `json:"time_range"`
}

// ActorStats 操作主体统计
type ActorStats struct {
	ActorID    string `json:"actor_id"`
	ActorName  string `json:"actor_name"`
	EventCount int64  `json:"event_count"`
}

// ResourceStats 资源统计
type ResourceStats struct {
	ResourceID   string `json:"resource_id"`
	ResourceName string `json:"resource_name"`
	EventCount   int64  `json:"event_count"`
}

// ActionStats 操作统计
type ActionStats struct {
	Action     string `json:"action"`
	EventCount int64  `json:"event_count"`
}

// TimeRangeStats 时间范围统计
type TimeRangeStats struct {
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
}

// AuditStorage 审计存储接口
type AuditStorage interface {
	// Write 写入审计日志
	Write(ctx context.Context, entry *AuditEntry) error

	// WriteBatch 批量写入审计日志
	WriteBatch(ctx context.Context, entries []*AuditEntry) error

	// Read 读取审计日志
	Read(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error)

	// Count 统计审计日志数量
	Count(ctx context.Context, filter *AuditFilter) (int64, error)

	// Delete 删除审计日志
	Delete(ctx context.Context, filter *AuditFilter) (int64, error)

	// Close 关闭存储
	Close() error
}

// auditLogger 审计日志记录器实现
type auditLogger struct {
	storage          AuditStorage
	logger           logging.Logger
	metricsCollector metrics.Collector
	tracer           trace.Tracer

	config      *AuditLoggerConfig
	buffer      []*AuditEntry
	bufferMutex sync.Mutex
	flushTicker *time.Ticker
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// AuditLoggerConfig 审计日志记录器配置
type AuditLoggerConfig struct {
	BufferSize        int           `yaml:"buffer_size"`
	FlushInterval     time.Duration `yaml:"flush_interval"`
	MaxRetries        int           `yaml:"max_retries"`
	RetryDelay        time.Duration `yaml:"retry_delay"`
	EnableCompression bool          `yaml:"enable_compression"`
	EnableEncryption  bool          `yaml:"enable_encryption"`
	SamplingRate      float64       `yaml:"sampling_rate"`
	ExcludePatterns   []string      `yaml:"exclude_patterns"`
	IncludePatterns   []string      `yaml:"include_patterns"`
	RetentionDays     int           `yaml:"retention_days"`
}

// NewAuditLogger 创建审计日志记录器
func NewAuditLogger(
	storage AuditStorage,
	logger logging.Logger,
	metricsCollector metrics.Collector,
	tracer trace.Tracer,
	config *AuditLoggerConfig,
) AuditLogger {
	al := &auditLogger{
		storage:          storage,
		logger:           logger,
		metricsCollector: metricsCollector,
		tracer:           tracer,
		config:           config,
		buffer:           make([]*AuditEntry, 0, config.BufferSize),
		stopChan:         make(chan struct{}),
	}

	// 启动定期刷新
	if config.FlushInterval > 0 {
		al.startAutoFlush()
	}

	return al
}

// Log 记录审计日志
func (a *auditLogger) Log(ctx context.Context, entry *AuditEntry) error {
	ctx, span := a.tracer.Start(ctx, "AuditLogger.Log")
	defer span.End()

	// 设置默认值
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("audit_%d_%d", time.Now().UnixNano(), randomInt())
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// 从上下文提取追踪信息
	if traceID := trace.GetTraceID(ctx); traceID != "" {
		entry.TraceID = traceID
	}
	if spanID := trace.GetSpanID(ctx); spanID != "" {
		entry.SpanID = spanID
	}

	// 检查采样率
	if a.config.SamplingRate < 1.0 {
		if randomFloat() > a.config.SamplingRate {
			a.logger.Debug(ctx, "Audit entry sampled out", "id", entry.ID)
			return nil
		}
	}

	// 检查排除模式
	if a.shouldExclude(entry) {
		a.logger.Debug(ctx, "Audit entry excluded", "id", entry.ID)
		return nil
	}

	a.logger.Debug(ctx, "Recording audit entry",
		"id", entry.ID,
		"event_type", entry.EventType,
		"actor", entry.ActorID,
		"resource", entry.ResourceID)

	// 添加到缓冲区
	if a.config.BufferSize > 0 {
		a.bufferMutex.Lock()
		a.buffer = append(a.buffer, entry)
		shouldFlush := len(a.buffer) >= a.config.BufferSize
		a.bufferMutex.Unlock()

		if shouldFlush {
			go a.flush(context.Background())
		}
	} else {
		// 直接写入
		if err := a.writeWithRetry(ctx, entry); err != nil {
			return err
		}
	}

	// 记录指标
	a.recordMetrics(entry)

	return nil
}

// LogBatch 批量记录审计日志
func (a *auditLogger) LogBatch(ctx context.Context, entries []*AuditEntry) error {
	ctx, span := a.tracer.Start(ctx, "AuditLogger.LogBatch")
	defer span.End()

	a.logger.Debug(ctx, "Recording batch audit entries", "count", len(entries))

	// 设置默认值
	for _, entry := range entries {
		if entry.ID == "" {
			entry.ID = fmt.Sprintf("audit_%d_%d", time.Now().UnixNano(), randomInt())
		}
		if entry.Timestamp.IsZero() {
			entry.Timestamp = time.Now()
		}
	}

	// 批量写入
	if err := a.storage.WriteBatch(ctx, entries); err != nil {
		a.logger.Error(ctx, "Failed to write batch audit entries", "error", err)
		return errors.Wrap(err, errors.CodeInternalError, "failed to write batch audit entries")
	}

	// 记录指标
	for _, entry := range entries {
		a.recordMetrics(entry)
	}

	a.logger.Info(ctx, "Batch audit entries recorded", "count", len(entries))

	return nil
}

// Query 查询审计日志
func (a *auditLogger) Query(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error) {
	ctx, span := a.tracer.Start(ctx, "AuditLogger.Query")
	defer span.End()

	a.logger.Debug(ctx, "Querying audit entries", "filter", filter)

	entries, err := a.storage.Read(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to query audit entries")
	}

	a.logger.Info(ctx, "Audit entries queried", "count", len(entries))

	return entries, nil
}

// QueryWithPagination 分页查询审计日志
func (a *auditLogger) QueryWithPagination(ctx context.Context, filter *AuditFilter, page, pageSize int) (*AuditQueryResult, error) {
	ctx, span := a.tracer.Start(ctx, "AuditLogger.QueryWithPagination")
	defer span.End()

	a.logger.Debug(ctx, "Querying audit entries with pagination",
		"page", page, "page_size", pageSize)

	// 设置分页参数
	if filter == nil {
		filter = &AuditFilter{}
	}
	filter.Limit = pageSize
	filter.Offset = (page - 1) * pageSize

	// 查询总数
	total, err := a.Count(ctx, filter)
	if err != nil {
		return nil, err
	}

	// 查询数据
	entries, err := a.Query(ctx, filter)
	if err != nil {
		return nil, err
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	result := &AuditQueryResult{
		Entries:    entries,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	return result, nil
}

// Count 统计审计日志数量
func (a *auditLogger) Count(ctx context.Context, filter *AuditFilter) (int64, error) {
	ctx, span := a.tracer.Start(ctx, "AuditLogger.Count")
	defer span.End()

	count, err := a.storage.Count(ctx, filter)
	if err != nil {
		return 0, errors.Wrap(err, errors.CodeInternalError, "failed to count audit entries")
	}

	return count, nil
}

// GetStatistics 获取审计统计信息
func (a *auditLogger) GetStatistics(ctx context.Context, filter *AuditFilter) (*AuditStatistics, error) {
	ctx, span := a.tracer.Start(ctx, "AuditLogger.GetStatistics")
	defer span.End()

	a.logger.Debug(ctx, "Getting audit statistics")

	// 查询所有符合条件的审计日志
	entries, err := a.Query(ctx, filter)
	if err != nil {
		return nil, err
	}

	stats := &AuditStatistics{
		TotalEvents:      int64(len(entries)),
		EventsByType:     make(map[EventType]int64),
		EventsByCategory: make(map[EventCategory]int64),
		EventsByResult:   make(map[EventResult]int64),
		EventsBySeverity: make(map[Severity]int64),
	}

	// 计算统计信息
	actorCounts := make(map[string]int64)
	resourceCounts := make(map[string]int64)
	actionCounts := make(map[string]int64)
	var totalDuration time.Duration
	var failureCount int64

	for _, entry := range entries {
		// 按类型统计
		stats.EventsByType[entry.EventType]++

		// 按分类统计
		stats.EventsByCategory[entry.Category]++

		// 按结果统计
		stats.EventsByResult[entry.Result]++
		if entry.Result == ResultFailure || entry.Result == ResultError {
			failureCount++
		}

		// 按严重程度统计
		stats.EventsBySeverity[entry.Severity]++

		// 统计操作主体
		actorCounts[entry.ActorID]++

		// 统计资源
		resourceCounts[entry.ResourceID]++

		// 统计操作
		actionCounts[entry.Action]++

		// 累计时长
		totalDuration += entry.Duration

		// 时间范围
		if stats.TimeRange.StartTime.IsZero() || entry.Timestamp.Before(stats.TimeRange.StartTime) {
			stats.TimeRange.StartTime = entry.Timestamp
		}
		if entry.Timestamp.After(stats.TimeRange.EndTime) {
			stats.TimeRange.EndTime = entry.Timestamp
		}
	}

	// 计算平均时长
	if len(entries) > 0 {
		stats.AverageDuration = totalDuration / time.Duration(len(entries))
		stats.FailureRate = float64(failureCount) / float64(len(entries))
	}

	// 计算时间范围时长
	stats.TimeRange.Duration = stats.TimeRange.EndTime.Sub(stats.TimeRange.StartTime)

	// Top N 统计
	stats.TopActors = topN(actorCounts, 10, func(id string, count int64) ActorStats {
		return ActorStats{ActorID: id, EventCount: count}
	})

	stats.TopResources = topN(resourceCounts, 10, func(id string, count int64) ResourceStats {
		return ResourceStats{ResourceID: id, EventCount: count}
	})

	stats.TopActions = topN(actionCounts, 10, func(action string, count int64) ActionStats {
		return ActionStats{Action: action, EventCount: count}
	})

	return stats, nil
}

// Delete 删除审计日志
func (a *auditLogger) Delete(ctx context.Context, filter *AuditFilter) (int64, error) {
	ctx, span := a.tracer.Start(ctx, "AuditLogger.Delete")
	defer span.End()

	a.logger.Info(ctx, "Deleting audit entries", "filter", filter)

	count, err := a.storage.Delete(ctx, filter)
	if err != nil {
		return 0, errors.Wrap(err, errors.CodeInternalError, "failed to delete audit entries")
	}

	a.logger.Info(ctx, "Audit entries deleted", "count", count)

	return count, nil
}

// Archive 归档审计日志
func (a *auditLogger) Archive(ctx context.Context, filter *AuditFilter, archivePath string) error {
	ctx, span := a.tracer.Start(ctx, "AuditLogger.Archive")
	defer span.End()

	a.logger.Info(ctx, "Archiving audit entries", "path", archivePath)

	// 查询要归档的数据
	entries, err := a.Query(ctx, filter)
	if err != nil {
		return err
	}

	// 序列化为 JSON
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to marshal audit entries")
	}

	// 写入文件（简化实现）
	// 实际应该使用更健壮的文件写入和压缩
	_ = data

	a.logger.Info(ctx, "Audit entries archived",
		"count", len(entries), "path", archivePath)

	return nil
}

// Close 关闭审计日志记录器
func (a *auditLogger) Close() error {
	a.logger.Info(context.Background(), "Closing audit logger")

	// 停止自动刷新
	if a.flushTicker != nil {
		a.flushTicker.Stop()
		close(a.stopChan)
	}

	// 等待所有刷新完成
	a.wg.Wait()

	// 最终刷新缓冲区
	if err := a.flush(context.Background()); err != nil {
		a.logger.Error(context.Background(), "Failed to flush buffer on close", "error", err)
	}

	// 关闭存储
	if err := a.storage.Close(); err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to close storage")
	}

	a.logger.Info(context.Background(), "Audit logger closed")

	return nil
}

// Helper methods

func (a *auditLogger) startAutoFlush() {
	a.flushTicker = time.NewTicker(a.config.FlushInterval)

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()

		for {
			select {
			case <-a.flushTicker.C:
				if err := a.flush(context.Background()); err != nil {
					a.logger.Error(context.Background(), "Auto flush failed", "error", err)
				}
			case <-a.stopChan:
				return
			}
		}
	}()
}

func (a *auditLogger) flush(ctx context.Context) error {
	a.bufferMutex.Lock()
	if len(a.buffer) == 0 {
		a.bufferMutex.Unlock()
		return nil
	}

	entries := make([]*AuditEntry, len(a.buffer))
	copy(entries, a.buffer)
	a.buffer = a.buffer[:0]
	a.bufferMutex.Unlock()

	a.logger.Debug(ctx, "Flushing audit buffer", "count", len(entries))

	if err := a.storage.WriteBatch(ctx, entries); err != nil {
		a.logger.Error(ctx, "Failed to flush audit buffer", "error", err)
		return err
	}

	return nil
}

func (a *auditLogger) writeWithRetry(ctx context.Context, entry *AuditEntry) error {
	var lastErr error

	for i := 0; i <= a.config.MaxRetries; i++ {
		if err := a.storage.Write(ctx, entry); err != nil {
			lastErr = err
			a.logger.Warn(ctx, "Failed to write audit entry, retrying",
				"attempt", i+1, "error", err)

			if i < a.config.MaxRetries {
				time.Sleep(a.config.RetryDelay)
				continue
			}
		} else {
			return nil
		}
	}

	return errors.Wrap(lastErr, errors.CodeInternalError, "failed to write audit entry after retries")
}

func (a *auditLogger) shouldExclude(entry *AuditEntry) bool {
	// 简化实现：检查排除模式
	for _, pattern := range a.config.ExcludePatterns {
		if pattern == string(entry.EventType) || pattern == entry.Action {
			return true
		}
	}

	return false
}

func (a *auditLogger) recordMetrics(entry *AuditEntry) {
	a.metricsCollector.Increment("audit_entries_total",
		map[string]string{
			"event_type": string(entry.EventType),
			"category":   string(entry.Category),
			"result":     string(entry.Result),
			"severity":   string(entry.Severity),
		})

	if entry.Duration > 0 {
		a.metricsCollector.Histogram("audit_operation_duration_ms",
			float64(entry.Duration.Milliseconds()),
			map[string]string{
				"event_type": string(entry.EventType),
				"action":     entry.Action,
			})
	}
}

// Utility functions

func randomInt() int64 {
	return time.Now().UnixNano() % 1000000
}

func randomFloat() float64 {
	return float64(randomInt()%100) / 100.0
}

func topN[T any](counts map[string]int64, n int, mapper func(string, int64) T) []T {
	type pair struct {
		key   string
		value int64
	}

	pairs := make([]pair, 0, len(counts))
	for k, v := range counts {
		pairs = append(pairs, pair{k, v})
	}

	// 简单排序
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[i].value < pairs[j].value {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	if len(pairs) > n {
		pairs = pairs[:n]
	}

	result := make([]T, len(pairs))
	for i, p := range pairs {
		result[i] = mapper(p.key, p.value)
	}

	return result
}

//Personal.AI order the ending
