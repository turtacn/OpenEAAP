// internal/governance/audit/audit_query.go
package audit

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/pkg/errors"
)

// AuditQueryService 审计日志查询服务接口
type AuditQueryService interface {
	// Query 查询审计日志
	Query(ctx context.Context, query *QueryRequest) (*QueryResponse, error)

	// QueryByUser 按用户查询
	QueryByUser(ctx context.Context, userID string, options *QueryOptions) ([]*AuditEntry, error)

	// QueryByResource 按资源查询
	QueryByResource(ctx context.Context, resourceID string, options *QueryOptions) ([]*AuditEntry, error)

	// QueryByTimeRange 按时间范围查询
	QueryByTimeRange(ctx context.Context, startTime, endTime time.Time, options *QueryOptions) ([]*AuditEntry, error)

	// QueryByAction 按操作类型查询
	QueryByAction(ctx context.Context, action string, options *QueryOptions) ([]*AuditEntry, error)

	// AdvancedQuery 高级查询
	AdvancedQuery(ctx context.Context, criteria *QueryCriteria) (*QueryResponse, error)

	// GenerateReport 生成审计报告
	GenerateReport(ctx context.Context, request *ReportRequest) (*AuditReport, error)

	// ExportReport 导出审计报告
	ExportReport(ctx context.Context, report *AuditReport, format ExportFormat, writer io.Writer) error

	// GetTrends 获取趋势分析
	GetTrends(ctx context.Context, request *TrendRequest) (*TrendAnalysis, error)

	// GetAnomalies 获取异常检测结果
	GetAnomalies(ctx context.Context, request *AnomalyRequest) ([]*Anomaly, error)
}

// QueryRequest 查询请求
type QueryRequest struct {
	Filter      *AuditFilter  `json:"filter"`
	Options     *QueryOptions `json:"options"`
	Aggregation *Aggregation  `json:"aggregation,omitempty"`
}

// QueryOptions 查询选项
type QueryOptions struct {
	Page         int      `json:"page"`
	PageSize     int      `json:"page_size"`
	SortBy       string   `json:"sort_by"`
	SortOrder    string   `json:"sort_order"` // asc/desc
	Fields       []string `json:"fields,omitempty"`
	IncludeTotal bool     `json:"include_total"`
	IncludeStats bool     `json:"include_stats"`
}

// QueryCriteria 查询条件
type QueryCriteria struct {
	Conditions []*Condition  `json:"conditions"`
	Logic      LogicOperator `json:"logic"` // AND/OR
	GroupBy    []string      `json:"group_by,omitempty"`
	Having     *Condition    `json:"having,omitempty"`
	OrderBy    []OrderBy     `json:"order_by,omitempty"`
	Limit      int           `json:"limit,omitempty"`
	Offset     int           `json:"offset,omitempty"`
}

// Condition 查询条件
type Condition struct {
	Field    string        `json:"field"`
	Operator CondOperator  `json:"operator"`
	Value    interface{}   `json:"value"`
	Values   []interface{} `json:"values,omitempty"`
}

// LogicOperator 逻辑操作符
type LogicOperator string

const (
	LogicAND LogicOperator = "AND"
	LogicOR  LogicOperator = "OR"
)

// CondOperator 条件操作符
type CondOperator string

const (
	OpEquals       CondOperator = "eq"
	OpNotEquals    CondOperator = "ne"
	OpGreaterThan  CondOperator = "gt"
	OpGreaterEqual CondOperator = "gte"
	OpLessThan     CondOperator = "lt"
	OpLessEqual    CondOperator = "lte"
	OpIn           CondOperator = "in"
	OpNotIn        CondOperator = "nin"
	OpContains     CondOperator = "contains"
	OpStartsWith   CondOperator = "starts_with"
	OpEndsWith     CondOperator = "ends_with"
	OpRegex        CondOperator = "regex"
)

// OrderBy 排序条件
type OrderBy struct {
	Field string `json:"field"`
	Order string `json:"order"` // asc/desc
}

// Aggregation 聚合查询
type Aggregation struct {
	Type   AggregationType `json:"type"`
	Field  string          `json:"field"`
	Bucket *BucketConfig   `json:"bucket,omitempty"`
}

// AggregationType 聚合类型
type AggregationType string

const (
	AggCount    AggregationType = "count"
	AggSum      AggregationType = "sum"
	AggAvg      AggregationType = "avg"
	AggMin      AggregationType = "min"
	AggMax      AggregationType = "max"
	AggTerms    AggregationType = "terms"
	AggDateHist AggregationType = "date_histogram"
)

// BucketConfig 桶配置
type BucketConfig struct {
	Interval string `json:"interval"` // 1h, 1d, etc.
	Size     int    `json:"size"`
}

// QueryResponse 查询响应
type QueryResponse struct {
	Entries      []*AuditEntry          `json:"entries"`
	Total        int64                  `json:"total"`
	Page         int                    `json:"page"`
	PageSize     int                    `json:"page_size"`
	Statistics   *AuditStatistics       `json:"statistics,omitempty"`
	Aggregations map[string]interface{} `json:"aggregations,omitempty"`
}

// ReportRequest 报告请求
type ReportRequest struct {
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	Filter        *AuditFilter           `json:"filter"`
	ReportType    ReportType             `json:"report_type"`
	Sections      []ReportSection        `json:"sections"`
	Format        ExportFormat           `json:"format"`
	Grouping      []string               `json:"grouping,omitempty"`
	TimeGrain     string                 `json:"time_grain,omitempty"` // hour/day/week/month
	IncludeCharts bool                   `json:"include_charts"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ReportType 报告类型
type ReportType string

const (
	ReportTypeSummary     ReportType = "summary"
	ReportTypeDetailed    ReportType = "detailed"
	ReportTypeCompliance  ReportType = "compliance"
	ReportTypeSecurity    ReportType = "security"
	ReportTypePerformance ReportType = "performance"
	ReportTypeCustom      ReportType = "custom"
)

// ReportSection 报告节
type ReportSection string

const (
	SectionOverview        ReportSection = "overview"
	SectionByUser          ReportSection = "by_user"
	SectionByResource      ReportSection = "by_resource"
	SectionByAction        ReportSection = "by_action"
	SectionByResult        ReportSection = "by_result"
	SectionTimeline        ReportSection = "timeline"
	SectionAnomalies       ReportSection = "anomalies"
	SectionRecommendations ReportSection = "recommendations"
)

// ExportFormat 导出格式
type ExportFormat string

const (
	FormatJSON  ExportFormat = "json"
	FormatCSV   ExportFormat = "csv"
	FormatPDF   ExportFormat = "pdf"
	FormatHTML  ExportFormat = "html"
	FormatExcel ExportFormat = "excel"
)

// AuditReport 审计报告
type AuditReport struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	ReportType  ReportType     `json:"report_type"`
	GeneratedAt time.Time      `json:"generated_at"`
	GeneratedBy string         `json:"generated_by"`
	TimeRange   TimeRangeStats `json:"time_range"`

	Summary         *ReportSummary           `json:"summary"`
	UserActivity    []*UserActivityReport    `json:"user_activity,omitempty"`
	ResourceAccess  []*ResourceAccessReport  `json:"resource_access,omitempty"`
	ActionBreakdown []*ActionBreakdownReport `json:"action_breakdown,omitempty"`
	Timeline        []*TimelinePoint         `json:"timeline,omitempty"`
	Anomalies       []*Anomaly               `json:"anomalies,omitempty"`
	Recommendations []*Recommendation        `json:"recommendations,omitempty"`

	Charts   []*ChartData           `json:"charts,omitempty"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ReportSummary 报告摘要
type ReportSummary struct {
	TotalEvents       int64         `json:"total_events"`
	UniqueUsers       int64         `json:"unique_users"`
	UniqueResources   int64         `json:"unique_resources"`
	SuccessRate       float64       `json:"success_rate"`
	FailureRate       float64       `json:"failure_rate"`
	AverageDuration   time.Duration `json:"average_duration"`
	PeakActivityTime  time.Time     `json:"peak_activity_time"`
	TopEventType      EventType     `json:"top_event_type"`
	TopCategory       EventCategory `json:"top_category"`
	CriticalEvents    int64         `json:"critical_events"`
	SecurityIncidents int64         `json:"security_incidents"`
}

// UserActivityReport 用户活动报告
type UserActivityReport struct {
	UserID            string        `json:"user_id"`
	UserName          string        `json:"user_name"`
	UserType          ActorType     `json:"user_type"`
	TotalActions      int64         `json:"total_actions"`
	SuccessRate       float64       `json:"success_rate"`
	LastActivity      time.Time     `json:"last_activity"`
	TopActions        []ActionStats `json:"top_actions"`
	ResourcesAccessed []string      `json:"resources_accessed"`
	RiskScore         float64       `json:"risk_score"`
}

// ResourceAccessReport 资源访问报告
type ResourceAccessReport struct {
	ResourceID    string        `json:"resource_id"`
	ResourceName  string        `json:"resource_name"`
	ResourceType  ResourceType  `json:"resource_type"`
	AccessCount   int64         `json:"access_count"`
	UniqueUsers   int64         `json:"unique_users"`
	LastAccessed  time.Time     `json:"last_accessed"`
	TopUsers      []ActorStats  `json:"top_users"`
	TopActions    []ActionStats `json:"top_actions"`
	AccessPattern string        `json:"access_pattern"` // normal/suspicious/anomalous
}

// ActionBreakdownReport 操作细分报告
type ActionBreakdownReport struct {
	Action       string          `json:"action"`
	Count        int64           `json:"count"`
	SuccessCount int64           `json:"success_count"`
	FailureCount int64           `json:"failure_count"`
	SuccessRate  float64         `json:"success_rate"`
	AvgDuration  time.Duration   `json:"avg_duration"`
	TopUsers     []ActorStats    `json:"top_users"`
	TopResources []ResourceStats `json:"top_resources"`
}

// TimelinePoint 时间线点
type TimelinePoint struct {
	Timestamp    time.Time               `json:"timestamp"`
	EventCount   int64                   `json:"event_count"`
	SuccessCount int64                   `json:"success_count"`
	FailureCount int64                   `json:"failure_count"`
	EventTypes   map[EventType]int64     `json:"event_types"`
	Categories   map[EventCategory]int64 `json:"categories"`
}

// TrendRequest 趋势分析请求
type TrendRequest struct {
	Filter     *AuditFilter `json:"filter"`
	Metric     string       `json:"metric"`     // count/success_rate/avg_duration
	Interval   string       `json:"interval"`   // hour/day/week/month
	Comparison bool         `json:"comparison"` // 是否与前一周期比较
}

// TrendAnalysis 趋势分析
type TrendAnalysis struct {
	Metric     string            `json:"metric"`
	Interval   string            `json:"interval"`
	DataPoints []*TrendDataPoint `json:"data_points"`
	Summary    *TrendSummary     `json:"summary"`
	Forecast   []*ForecastPoint  `json:"forecast,omitempty"`
}

// TrendDataPoint 趋势数据点
type TrendDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Count     int64     `json:"count"`
}

// TrendSummary 趋势摘要
type TrendSummary struct {
	OverallTrend  string    `json:"overall_trend"` // increasing/decreasing/stable
	ChangePercent float64   `json:"change_percent"`
	Average       float64   `json:"average"`
	Peak          float64   `json:"peak"`
	PeakTime      time.Time `json:"peak_time"`
	Trough        float64   `json:"trough"`
	TroughTime    time.Time `json:"trough_time"`
}

// ForecastPoint 预测点
type ForecastPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Lower     float64   `json:"lower"`
	Upper     float64   `json:"upper"`
}

// AnomalyRequest 异常检测请求
type AnomalyRequest struct {
	Filter      *AuditFilter `json:"filter"`
	Sensitivity float64      `json:"sensitivity"` // 0-1
	Methods     []string     `json:"methods"`     // statistical/ml/rule_based
}

// Anomaly 异常
type Anomaly struct {
	ID          string                 `json:"id"`
	Type        AnomalyType            `json:"type"`
	Severity    Severity               `json:"severity"`
	DetectedAt  time.Time              `json:"detected_at"`
	Description string                 `json:"description"`
	Entries     []*AuditEntry          `json:"entries"`
	Score       float64                `json:"score"`
	Indicators  []string               `json:"indicators"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// AnomalyType 异常类型
type AnomalyType string

const (
	AnomalyUnusualVolume       AnomalyType = "unusual_volume"
	AnomalyUnusualTime         AnomalyType = "unusual_time"
	AnomalyUnusualLocation     AnomalyType = "unusual_location"
	AnomalyUnusualAccess       AnomalyType = "unusual_access"
	AnomalyMultipleFailures    AnomalyType = "multiple_failures"
	AnomalyPrivilegeEscalation AnomalyType = "privilege_escalation"
	AnomalySuspiciousPattern   AnomalyType = "suspicious_pattern"
)

// Recommendation 建议
type Recommendation struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Priority    string   `json:"priority"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Actions     []string `json:"actions"`
	Impact      string   `json:"impact"`
}

// ChartData 图表数据
type ChartData struct {
	Type    string                 `json:"type"` // bar/line/pie/scatter
	Title   string                 `json:"title"`
	Labels  []string               `json:"labels"`
	Data    []interface{}          `json:"data"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// auditQueryService 审计查询服务实现
type auditQueryService struct {
	auditLogger      AuditLogger
	logger           logging.Logger
	metricsCollector *metrics.MetricsCollector
	tracer           trace.Tracer

	cache    sync.Map
	cacheTTL time.Duration
}

// NewAuditQueryService 创建审计查询服务
func NewAuditQueryService(
	auditLogger AuditLogger,
	logger logging.Logger,
	metricsCollector metrics.MetricsCollector,
	tracer trace.Tracer,
	cacheTTL time.Duration,
) AuditQueryService {
	return &auditQueryService{
		auditLogger:      auditLogger,
		logger:           logger,
		metricsCollector: metricsCollector,
		tracer:           tracer,
		cacheTTL:         cacheTTL,
	}
}

// Query 查询审计日志
func (s *auditQueryService) Query(ctx context.Context, query *QueryRequest) (*QueryResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuditQueryService.Query")
	defer span.End()

	s.logger.WithContext(ctx).Debug("Querying audit logs", logging.Any("filter", query.Filter))

	// 查询日志
	entries, err := s.auditLogger.Query(ctx, query.Filter)
	if err != nil {
		return nil, err
	}

	// 应用排序
	if query.Options != nil && query.Options.SortBy != "" {
		s.sortEntries(entries, query.Options.SortBy, query.Options.SortOrder)
	}

	// 应用分页
	total := int64(len(entries))
	if query.Options != nil && query.Options.PageSize > 0 {
		entries = s.paginateEntries(entries, query.Options.Page, query.Options.PageSize)
	}

	response := &QueryResponse{
		Entries:  entries,
		Total:    total,
		Page:     1,
		PageSize: len(entries),
	}

	if query.Options != nil {
		response.Page = query.Options.Page
		response.PageSize = query.Options.PageSize

		// 包含统计信息
		if query.Options.IncludeStats {
			allEntries, _ := s.auditLogger.Query(ctx, query.Filter)
			stats := s.calculateStatistics(allEntries)
			response.Statistics = stats
		}
	}

	// 处理聚合
	if query.Aggregation != nil {
		agg := s.performAggregation(entries, query.Aggregation)
		response.Aggregations = agg
	}

	return response, nil
}

// QueryByUser 按用户查询
func (s *auditQueryService) QueryByUser(ctx context.Context, userID string, options *QueryOptions) ([]*AuditEntry, error) {
	ctx, span := s.tracer.Start(ctx, "AuditQueryService.QueryByUser")
	defer span.End()

	filter := &AuditFilter{
		ActorIDs: []string{userID},
	}

	if options != nil {
		filter.Limit = options.PageSize
		filter.Offset = (options.Page - 1) * options.PageSize
		filter.SortBy = options.SortBy
		filter.SortOrder = options.SortOrder
	}

	return s.auditLogger.Query(ctx, filter)
}

// QueryByResource 按资源查询
func (s *auditQueryService) QueryByResource(ctx context.Context, resourceID string, options *QueryOptions) ([]*AuditEntry, error) {
	ctx, span := s.tracer.Start(ctx, "AuditQueryService.QueryByResource")
	defer span.End()

	filter := &AuditFilter{
		ResourceIDs: []string{resourceID},
	}

	if options != nil {
		filter.Limit = options.PageSize
		filter.Offset = (options.Page - 1) * options.PageSize
	}

	return s.auditLogger.Query(ctx, filter)
}

// QueryByTimeRange 按时间范围查询
func (s *auditQueryService) QueryByTimeRange(ctx context.Context, startTime, endTime time.Time, options *QueryOptions) ([]*AuditEntry, error) {
	ctx, span := s.tracer.Start(ctx, "AuditQueryService.QueryByTimeRange")
	defer span.End()

	filter := &AuditFilter{
		StartTime: startTime,
		EndTime:   endTime,
	}

	if options != nil {
		filter.Limit = options.PageSize
		filter.Offset = (options.Page - 1) * options.PageSize
	}

	return s.auditLogger.Query(ctx, filter)
}

// QueryByAction 按操作类型查询
func (s *auditQueryService) QueryByAction(ctx context.Context, action string, options *QueryOptions) ([]*AuditEntry, error) {
	ctx, span := s.tracer.Start(ctx, "AuditQueryService.QueryByAction")
	defer span.End()

	filter := &AuditFilter{
		Actions: []string{action},
	}

	if options != nil {
		filter.Limit = options.PageSize
		filter.Offset = (options.Page - 1) * options.PageSize
	}

	return s.auditLogger.Query(ctx, filter)
}

// AdvancedQuery 高级查询
func (s *auditQueryService) AdvancedQuery(ctx context.Context, criteria *QueryCriteria) (*QueryResponse, error) {
	ctx, span := s.tracer.Start(ctx, "AuditQueryService.AdvancedQuery")
	defer span.End()

	s.logger.WithContext(ctx).Debug("Executing advanced query", logging.Any("criteria", criteria))

	// 转换条件为过滤器
	filter := s.criteriaToFilter(criteria)

	entries, err := s.auditLogger.Query(ctx, filter)
	if err != nil {
		return nil, err
	}

	// 应用高级过滤
	entries = s.applyConditions(entries, criteria.Conditions, criteria.Logic)

	// 分组
	if len(criteria.GroupBy) > 0 {
		// 简化实现：不做实际分组
	}

	// 排序
	if len(criteria.OrderBy) > 0 {
		for _, order := range criteria.OrderBy {
			s.sortEntries(entries, order.Field, order.Order)
		}
	}

	// 分页
	total := int64(len(entries))
	if criteria.Limit > 0 {
		start := criteria.Offset
		end := start + criteria.Limit
		if start < len(entries) {
			if end > len(entries) {
				end = len(entries)
			}
			entries = entries[start:end]
		}
	}

	return &QueryResponse{
		Entries:  entries,
		Total:    total,
		Page:     criteria.Offset/criteria.Limit + 1,
		PageSize: len(entries),
	}, nil
}

// GenerateReport 生成审计报告
func (s *auditQueryService) GenerateReport(ctx context.Context, request *ReportRequest) (*AuditReport, error) {
	ctx, span := s.tracer.Start(ctx, "AuditQueryService.GenerateReport")
	defer span.End()

	s.logger.WithContext(ctx).Info("Generating audit report", logging.Any("type", request.ReportType))

	// 查询审计日志
	entries, err := s.auditLogger.Query(ctx, request.Filter)
	if err != nil {
		return nil, err
	}

	report := &AuditReport{
		ID:          fmt.Sprintf("report_%d", time.Now().UnixNano()),
		Title:       request.Title,
		Description: request.Description,
		ReportType:  request.ReportType,
		GeneratedAt: time.Now(),
		Metadata:    request.Metadata,
	}

	// 计算时间范围
	if len(entries) > 0 {
		report.TimeRange.StartTime = entries[0].Timestamp
		report.TimeRange.EndTime = entries[len(entries)-1].Timestamp
		for _, entry := range entries {
			if entry.Timestamp.Before(report.TimeRange.StartTime) {
				report.TimeRange.StartTime = entry.Timestamp
			}
			if entry.Timestamp.After(report.TimeRange.EndTime) {
				report.TimeRange.EndTime = entry.Timestamp
			}
		}
		report.TimeRange.Duration = report.TimeRange.EndTime.Sub(report.TimeRange.StartTime)
	}

	// 生成各个节
	for _, section := range request.Sections {
		switch section {
		case SectionOverview:
			report.Summary = s.generateSummary(entries)

		case SectionByUser:
			report.UserActivity = s.generateUserActivity(entries)

		case SectionByResource:
			report.ResourceAccess = s.generateResourceAccess(entries)

		case SectionByAction:
			report.ActionBreakdown = s.generateActionBreakdown(entries)

		case SectionTimeline:
			report.Timeline = s.generateTimeline(entries, request.TimeGrain)

		case SectionAnomalies:
			anomalies, _ := s.detectAnomalies(ctx, entries)
			report.Anomalies = anomalies

		case SectionRecommendations:
			report.Recommendations = s.generateRecommendations(entries, report.Summary)
		}
	}

	// 生成图表
	if request.IncludeCharts {
		report.Charts = s.generateCharts(entries, request.Sections)
	}

	s.logger.WithContext(ctx).Info("Audit report generated", logging.Any("report_id", report.ID))

	return report, nil
}

// ExportReport 导出审计报告
func (s *auditQueryService) ExportReport(ctx context.Context, report *AuditReport, format ExportFormat, writer io.Writer) error {
	ctx, span := s.tracer.Start(ctx, "AuditQueryService.ExportReport")
	defer span.End()

	s.logger.WithContext(ctx).Info("Exporting audit report", logging.Any("format", format))

	switch format {
	case FormatJSON:
		return s.exportJSON(report, writer)
	case FormatCSV:
		return s.exportCSV(report, writer)
	case FormatHTML:
		return s.exportHTML(report, writer)
	case FormatPDF:
		return s.exportPDF(report, writer)
	default:
		return errors.ValidationError(
			fmt.Sprintf("unsupported export format: %s", format))
	}
}

// GetTrends 获取趋势分析
func (s *auditQueryService) GetTrends(ctx context.Context, request *TrendRequest) (*TrendAnalysis, error) {
	ctx, span := s.tracer.Start(ctx, "AuditQueryService.GetTrends")
	defer span.End()

	s.logger.WithContext(ctx).Debug("Analyzing trends", logging.Any("metric", request.Metric))

	entries, err := s.auditLogger.Query(ctx, request.Filter)
	if err != nil {
		return nil, err
	}

	// 按时间间隔分组
	dataPoints := s.groupByInterval(entries, request.Interval, request.Metric)

	// 计算趋势摘要
	summary := s.calculateTrendSummary(dataPoints)

	analysis := &TrendAnalysis{
		Metric:     request.Metric,
		Interval:   request.Interval,
		DataPoints: dataPoints,
		Summary:    summary,
	}

	return analysis, nil
}

// GetAnomalies 获取异常检测结果
func (s *auditQueryService) GetAnomalies(ctx context.Context, request *AnomalyRequest) ([]*Anomaly, error) {
	ctx, span := s.tracer.Start(ctx, "AuditQueryService.GetAnomalies")
	defer span.End()

	s.logger.WithContext(ctx).Debug("Detecting anomalies", logging.Any("sensitivity", request.Sensitivity))

	entries, err := s.auditLogger.Query(ctx, request.Filter)
	if err != nil {
		return nil, err
	}

	return s.detectAnomalies(ctx, entries)
}

// Helper methods

func (s *auditQueryService) sortEntries(entries []*AuditEntry, sortBy, sortOrder string) {
	if sortBy == "" {
		sortBy = "timestamp"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}

	sort.Slice(entries, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "timestamp":
			less = entries[i].Timestamp.Before(entries[j].Timestamp)
		case "duration":
			less = entries[i].Duration < entries[j].Duration
		case "severity":
			less = entries[i].Severity < entries[j].Severity
		default:
			less = entries[i].Timestamp.Before(entries[j].Timestamp)
		}

		if sortOrder == "desc" {
			return !less
		}
		return less
	})
}

func (s *auditQueryService) paginateEntries(entries []*AuditEntry, page, pageSize int) []*AuditEntry {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= len(entries) {
		return []*AuditEntry{}
	}
	if end > len(entries) {
		end = len(entries)
	}

	return entries[start:end]
}

func (s *auditQueryService) calculateStatistics(entries []*AuditEntry) *AuditStatistics {
	stats := &AuditStatistics{
		TotalEvents:      int64(len(entries)),
		EventsByType:     make(map[EventType]int64),
		EventsByCategory: make(map[EventCategory]int64),
		EventsByResult:   make(map[EventResult]int64),
		EventsBySeverity: make(map[Severity]int64),
	}

	actorCounts := make(map[string]int64)
	resourceCounts := make(map[string]int64)
	actionCounts := make(map[string]int64)
	var totalDuration time.Duration
	var failureCount int64

	for _, entry := range entries {
		stats.EventsByType[entry.EventType]++
		stats.EventsByCategory[entry.Category]++
		stats.EventsByResult[entry.Result]++
		stats.EventsBySeverity[entry.Severity]++

		if entry.Result == ResultFailure || entry.Result == ResultError {
			failureCount++
		}

		actorCounts[entry.ActorID]++
		resourceCounts[entry.ResourceID]++
		actionCounts[entry.Action]++
		totalDuration += entry.Duration

		if stats.TimeRange.StartTime.IsZero() || entry.Timestamp.Before(stats.TimeRange.StartTime) {
			stats.TimeRange.StartTime = entry.Timestamp
		}
		if entry.Timestamp.After(stats.TimeRange.EndTime) {
			stats.TimeRange.EndTime = entry.Timestamp
		}
	}

	if len(entries) > 0 {
		stats.AverageDuration = totalDuration / time.Duration(len(entries))
		stats.FailureRate = float64(failureCount) / float64(len(entries))
	}

	stats.TimeRange.Duration = stats.TimeRange.EndTime.Sub(stats.TimeRange.StartTime)

	stats.TopActors = topN(actorCounts, 10, func(id string, count int64) ActorStats {
		return ActorStats{ActorID: id, EventCount: count}
	})

	stats.TopResources = topN(resourceCounts, 10, func(id string, count int64) ResourceStats {
		return ResourceStats{ResourceID: id, EventCount: count}
	})

	stats.TopActions = topN(actionCounts, 10, func(action string, count int64) ActionStats {
		return ActionStats{Action: action, EventCount: count}
	})

	return stats
}

func (s *auditQueryService) performAggregation(entries []*AuditEntry, agg *Aggregation) map[string]interface{} {
	result := make(map[string]interface{})

	switch agg.Type {
	case AggCount:
		result["count"] = len(entries)

	case AggTerms:
		terms := make(map[string]int64)
		for _, entry := range entries {
			// 简化：根据字段获取值
			key := s.getFieldValue(entry, agg.Field)
			terms[key]++
		}
		result["terms"] = terms

	case AggDateHist:
		histogram := make(map[string]int64)
		for _, entry := range entries {
			bucket := s.getBucket(entry.Timestamp, agg.Bucket.Interval)
			histogram[bucket]++
		}
		result["histogram"] = histogram
	}

	return result
}

func (s *auditQueryService) getFieldValue(entry *AuditEntry, field string) string {
	switch field {
	case "event_type":
		return string(entry.EventType)
	case "category":
		return string(entry.Category)
	case "actor_id":
		return entry.ActorID
	case "resource_id":
		return entry.ResourceID
	case "action":
		return entry.Action
	default:
		return ""
	}
}

func (s *auditQueryService) getBucket(t time.Time, interval string) string {
	// 简化实现
	switch interval {
	case "1h":
		return t.Format("2006-01-02 15:00")
	case "1d":
		return t.Format("2006-01-02")
	case "1w":
		year, week := t.ISOWeek()
		return fmt.Sprintf("%d-W%02d", year, week)
	case "1M":
		return t.Format("2006-01")
	default:
		return t.Format("2006-01-02")
	}
}

func (s *auditQueryService) criteriaToFilter(criteria *QueryCriteria) *AuditFilter {
	filter := &AuditFilter{
		Limit:  criteria.Limit,
		Offset: criteria.Offset,
	}

	// 简化：转换条件
	for _, cond := range criteria.Conditions {
		switch cond.Field {
		case "event_type":
			if eventType, ok := cond.Value.(EventType); ok {
				filter.EventTypes = append(filter.EventTypes, eventType)
			}
		case "actor_id":
			if actorID, ok := cond.Value.(string); ok {
				filter.ActorIDs = append(filter.ActorIDs, actorID)
			}
		}
	}

	return filter
}

func (s *auditQueryService) applyConditions(entries []*AuditEntry, conditions []*Condition, logic LogicOperator) []*AuditEntry {
	if len(conditions) == 0 {
		return entries
	}

	filtered := []*AuditEntry{}

	for _, entry := range entries {
		match := logic == LogicAND

		for _, cond := range conditions {
			condMatch := s.evaluateCondition(entry, cond)

			if logic == LogicAND {
				match = match && condMatch
			} else {
				match = match || condMatch
			}
		}

		if match {
			filtered = append(filtered, entry)
		}
	}

	return filtered
}

func (s *auditQueryService) evaluateCondition(entry *AuditEntry, cond *Condition) bool {
	fieldValue := s.getFieldValue(entry, cond.Field)

	switch cond.Operator {
	case OpEquals:
		return fieldValue == fmt.Sprintf("%v", cond.Value)
	case OpNotEquals:
		return fieldValue != fmt.Sprintf("%v", cond.Value)
	case OpContains:
		return contains(fieldValue, fmt.Sprintf("%v", cond.Value))
	case OpIn:
		for _, v := range cond.Values {
			if fieldValue == fmt.Sprintf("%v", v) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (s *auditQueryService) generateSummary(entries []*AuditEntry) *ReportSummary {
	summary := &ReportSummary{}

	if len(entries) == 0 {
		return summary
	}

	summary.TotalEvents = int64(len(entries))

	users := make(map[string]bool)
	resources := make(map[string]bool)
	successCount := int64(0)
	failureCount := int64(0)
	criticalCount := int64(0)
	securityCount := int64(0)
	var totalDuration time.Duration

	eventTypeCounts := make(map[EventType]int64)
	categoryCounts := make(map[EventCategory]int64)

	for _, entry := range entries {
		users[entry.ActorID] = true
		resources[entry.ResourceID] = true

		if entry.Result == ResultSuccess {
			successCount++
		} else {
			failureCount++
		}

		if entry.Severity == SeverityCritical {
			criticalCount++
		}

		if entry.Category == CategorySecurity {
			securityCount++
		}

		totalDuration += entry.Duration

		eventTypeCounts[entry.EventType]++
		categoryCounts[entry.Category]++
	}

	summary.UniqueUsers = int64(len(users))
	summary.UniqueResources = int64(len(resources))
	summary.SuccessRate = float64(successCount) / float64(len(entries))
	summary.FailureRate = float64(failureCount) / float64(len(entries))
	summary.AverageDuration = totalDuration / time.Duration(len(entries))
	summary.CriticalEvents = criticalCount
	summary.SecurityIncidents = securityCount

	// 找出最常见的事件类型和分类
	var maxTypeCount int64
	var maxCategoryCount int64
	for eventType, count := range eventTypeCounts {
		if count > maxTypeCount {
			maxTypeCount = count
			summary.TopEventType = eventType
		}
	}
	for category, count := range categoryCounts {
		if count > maxCategoryCount {
			maxCategoryCount = count
			summary.TopCategory = category
		}
	}

	return summary
}

func (s *auditQueryService) generateUserActivity(entries []*AuditEntry) []*UserActivityReport {
	userMap := make(map[string]*UserActivityReport)

	for _, entry := range entries {
		report, exists := userMap[entry.ActorID]
		if !exists {
			report = &UserActivityReport{
				UserID:            entry.ActorID,
				UserName:          entry.ActorName,
				UserType:          entry.ActorType,
				TopActions:        []ActionStats{},
				ResourcesAccessed: []string{},
			}
			userMap[entry.ActorID] = report
		}

		report.TotalActions++

		if entry.Result == ResultSuccess {
			report.SuccessRate = (report.SuccessRate*float64(report.TotalActions-1) + 1.0) / float64(report.TotalActions)
		} else {
			report.SuccessRate = (report.SuccessRate * float64(report.TotalActions-1)) / float64(report.TotalActions)
		}

		if entry.Timestamp.After(report.LastActivity) {
			report.LastActivity = entry.Timestamp
		}

		// 添加访问的资源
		if !contains(report.ResourcesAccessed, entry.ResourceID) {
			report.ResourcesAccessed = append(report.ResourcesAccessed, entry.ResourceID)
		}
	}

	reports := make([]*UserActivityReport, 0, len(userMap))
	for _, report := range userMap {
		reports = append(reports, report)
	}

	// 排序
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].TotalActions > reports[j].TotalActions
	})

	return reports
}

func (s *auditQueryService) generateResourceAccess(entries []*AuditEntry) []*ResourceAccessReport {
	resourceMap := make(map[string]*ResourceAccessReport)

	for _, entry := range entries {
		report, exists := resourceMap[entry.ResourceID]
		if !exists {
			report = &ResourceAccessReport{
				ResourceID:   entry.ResourceID,
				ResourceName: entry.ResourceName,
				ResourceType: entry.ResourceType,
				TopUsers:     []ActorStats{},
				TopActions:   []ActionStats{},
			}
			resourceMap[entry.ResourceID] = report
		}

		report.AccessCount++

		if entry.Timestamp.After(report.LastAccessed) {
			report.LastAccessed = entry.Timestamp
		}
	}

	reports := make([]*ResourceAccessReport, 0, len(resourceMap))
	for _, report := range resourceMap {
		reports = append(reports, report)
	}

	sort.Slice(reports, func(i, j int) bool {
		return reports[i].AccessCount > reports[j].AccessCount
	})

	return reports
}

func (s *auditQueryService) generateActionBreakdown(entries []*AuditEntry) []*ActionBreakdownReport {
	actionMap := make(map[string]*ActionBreakdownReport)

	for _, entry := range entries {
		report, exists := actionMap[entry.Action]
		if !exists {
			report = &ActionBreakdownReport{
				Action:       entry.Action,
				TopUsers:     []ActorStats{},
				TopResources: []ResourceStats{},
			}
			actionMap[entry.Action] = report
		}

		report.Count++

		if entry.Result == ResultSuccess {
			report.SuccessCount++
		} else {
			report.FailureCount++
		}

		report.AvgDuration = (report.AvgDuration*time.Duration(report.Count-1) + entry.Duration) / time.Duration(report.Count)
	}

	reports := make([]*ActionBreakdownReport, 0, len(actionMap))
	for _, report := range actionMap {
		if report.Count > 0 {
			report.SuccessRate = float64(report.SuccessCount) / float64(report.Count)
		}
		reports = append(reports, report)
	}

	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Count > reports[j].Count
	})

	return reports
}

func (s *auditQueryService) generateTimeline(entries []*AuditEntry, timeGrain string) []*TimelinePoint {
	if timeGrain == "" {
		timeGrain = "hour"
	}

	buckets := make(map[string]*TimelinePoint)

	for _, entry := range entries {
		bucket := s.getBucket(entry.Timestamp, getIntervalFromGrain(timeGrain))

		point, exists := buckets[bucket]
		if !exists {
			point = &TimelinePoint{
				Timestamp:  entry.Timestamp,
				EventTypes: make(map[EventType]int64),
				Categories: make(map[EventCategory]int64),
			}
			buckets[bucket] = point
		}

		point.EventCount++
		point.EventTypes[entry.EventType]++
		point.Categories[entry.Category]++

		if entry.Result == ResultSuccess {
			point.SuccessCount++
		} else {
			point.FailureCount++
		}
	}

	timeline := make([]*TimelinePoint, 0, len(buckets))
	for _, point := range buckets {
		timeline = append(timeline, point)
	}

	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].Timestamp.Before(timeline[j].Timestamp)
	})

	return timeline
}

func (s *auditQueryService) detectAnomalies(ctx context.Context, entries []*AuditEntry) ([]*Anomaly, error) {
	anomalies := []*Anomaly{}

	// 检测异常卷
	if anomaly := s.detectUnusualVolume(entries); anomaly != nil {
		anomalies = append(anomalies, anomaly)
	}

	// 检测多次失败
	if anomaly := s.detectMultipleFailures(entries); anomaly != nil {
		anomalies = append(anomalies, anomaly)
	}

	// 检测异常时间
	if anomaly := s.detectUnusualTime(entries); anomaly != nil {
		anomalies = append(anomalies, anomaly)
	}

	return anomalies, nil
}

func (s *auditQueryService) detectUnusualVolume(entries []*AuditEntry) *Anomaly {
	// 简化实现：检测事件量突增
	if len(entries) > 1000 {
		return &Anomaly{
			ID:          fmt.Sprintf("anomaly_%d", time.Now().UnixNano()),
			Type:        AnomalyUnusualVolume,
			Severity:    SeverityMedium,
			DetectedAt:  time.Now(),
			Description: "Unusual high volume of audit events detected",
			Score:       0.8,
			Indicators:  []string{"high_event_count"},
		}
	}
	return nil
}

func (s *auditQueryService) detectMultipleFailures(entries []*AuditEntry) *Anomaly {
	failureCount := 0
	for _, entry := range entries {
		if entry.Result == ResultFailure || entry.Result == ResultError {
			failureCount++
		}
	}

	if len(entries) > 0 && float64(failureCount)/float64(len(entries)) > 0.5 {
		return &Anomaly{
			ID:          fmt.Sprintf("anomaly_%d", time.Now().UnixNano()),
			Type:        AnomalyMultipleFailures,
			Severity:    SeverityHigh,
			DetectedAt:  time.Now(),
			Description: "High failure rate detected",
			Score:       0.9,
			Indicators:  []string{"high_failure_rate"},
		}
	}
	return nil
}

func (s *auditQueryService) detectUnusualTime(entries []*AuditEntry) *Anomaly {
	// 简化：检测非工作时间的活动
	for _, entry := range entries {
		hour := entry.Timestamp.Hour()
		if hour < 6 || hour > 22 {
			return &Anomaly{
				ID:          fmt.Sprintf("anomaly_%d", time.Now().UnixNano()),
				Type:        AnomalyUnusualTime,
				Severity:    SeverityMedium,
				DetectedAt:  time.Now(),
				Description: "Activity detected outside normal hours",
				Score:       0.6,
				Indicators:  []string{"off_hours_activity"},
			}
		}
	}
	return nil
}

func (s *auditQueryService) generateRecommendations(entries []*AuditEntry, summary *ReportSummary) []*Recommendation {
	recommendations := []*Recommendation{}

	if summary.FailureRate > 0.3 {
		recommendations = append(recommendations, &Recommendation{
			ID:          "rec_1",
			Type:        "security",
			Priority:    "high",
			Title:       "High Failure Rate Detected",
			Description: "The failure rate exceeds 30%. Review authentication and authorization policies.",
			Actions:     []string{"Review access policies", "Check for brute force attempts"},
			Impact:      "Reduce security risks and improve system access",
		})
	}

	if summary.SecurityIncidents > 0 {
		recommendations = append(recommendations, &Recommendation{
			ID:          "rec_2",
			Type:        "security",
			Priority:    "critical",
			Title:       "Security Incidents Detected",
			Description: "Security-related events require immediate attention.",
			Actions:     []string{"Review security logs", "Investigate incidents", "Update security policies"},
			Impact:      "Mitigate security threats",
		})
	}

	return recommendations
}

func (s *auditQueryService) generateCharts(entries []*AuditEntry, sections []ReportSection) []*ChartData {
	charts := []*ChartData{}

	for _, section := range sections {
		switch section {
		case SectionByAction:
			charts = append(charts, s.generateActionChart(entries))
		case SectionTimeline:
			charts = append(charts, s.generateTimelineChart(entries))
		}
	}

	return charts
}

func (s *auditQueryService) generateActionChart(entries []*AuditEntry) *ChartData {
	actionCounts := make(map[string]int64)
	for _, entry := range entries {
		actionCounts[entry.Action]++
	}

	labels := []string{}
	data := []interface{}{}

	for action, count := range actionCounts {
		labels = append(labels, action)
		data = append(data, count)
	}

	return &ChartData{
		Type:   "bar",
		Title:  "Actions Distribution",
		Labels: labels,
		Data:   data,
	}
}

func (s *auditQueryService) generateTimelineChart(entries []*AuditEntry) *ChartData {
	return &ChartData{
		Type:  "line",
		Title: "Activity Timeline",
	}
}

func (s *auditQueryService) groupByInterval(entries []*AuditEntry, interval, metric string) []*TrendDataPoint {
	buckets := make(map[string]*TrendDataPoint)

	for _, entry := range entries {
		bucket := s.getBucket(entry.Timestamp, interval)

		point, exists := buckets[bucket]
		if !exists {
			point = &TrendDataPoint{
				Timestamp: entry.Timestamp,
			}
			buckets[bucket] = point
		}

		point.Count++

		// 根据指标计算值
		switch metric {
		case "count":
			point.Value = float64(point.Count)
		case "success_rate":
			if entry.Result == ResultSuccess {
				point.Value = (point.Value*float64(point.Count-1) + 1.0) / float64(point.Count)
			} else {
				point.Value = (point.Value * float64(point.Count-1)) / float64(point.Count)
			}
		case "avg_duration":
			point.Value = (point.Value*float64(point.Count-1) + float64(entry.Duration.Milliseconds())) / float64(point.Count)
		}
	}

	dataPoints := make([]*TrendDataPoint, 0, len(buckets))
	for _, point := range buckets {
		dataPoints = append(dataPoints, point)
	}

	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].Timestamp.Before(dataPoints[j].Timestamp)
	})

	return dataPoints
}

func (s *auditQueryService) calculateTrendSummary(dataPoints []*TrendDataPoint) *TrendSummary {
	if len(dataPoints) == 0 {
		return &TrendSummary{}
	}

	summary := &TrendSummary{}

	var totalValue float64
	for i, point := range dataPoints {
		totalValue += point.Value

		if i == 0 || point.Value > summary.Peak {
			summary.Peak = point.Value
			summary.PeakTime = point.Timestamp
		}

		if i == 0 || point.Value < summary.Trough {
			summary.Trough = point.Value
			summary.TroughTime = point.Timestamp
		}
	}

	summary.Average = totalValue / float64(len(dataPoints))

	// 简单趋势判断
	if len(dataPoints) >= 2 {
		firstHalf := dataPoints[:len(dataPoints)/2]
		secondHalf := dataPoints[len(dataPoints)/2:]

		var firstAvg, secondAvg float64
		for _, p := range firstHalf {
			firstAvg += p.Value
		}
		for _, p := range secondHalf {
			secondAvg += p.Value
		}

		firstAvg /= float64(len(firstHalf))
		secondAvg /= float64(len(secondHalf))

		summary.ChangePercent = (secondAvg - firstAvg) / firstAvg * 100

		if summary.ChangePercent > 10 {
			summary.OverallTrend = "increasing"
		} else if summary.ChangePercent < -10 {
			summary.OverallTrend = "decreasing"
		} else {
			summary.OverallTrend = "stable"
		}
	}

	return summary
}

func (s *auditQueryService) exportJSON(report *AuditReport, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func (s *auditQueryService) exportCSV(report *AuditReport, writer io.Writer) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// 写入标题
	csvWriter.Write([]string{"Report ID", report.ID})
	csvWriter.Write([]string{"Title", report.Title})
	csvWriter.Write([]string{"Generated At", report.GeneratedAt.Format(time.RFC3339)})
	csvWriter.Write([]string{})

	// 写入摘要
	if report.Summary != nil {
		csvWriter.Write([]string{"Summary"})
		csvWriter.Write([]string{"Total Events", fmt.Sprintf("%d", report.Summary.TotalEvents)})
		csvWriter.Write([]string{"Success Rate", fmt.Sprintf("%.2f%%", report.Summary.SuccessRate*100)})
		csvWriter.Write([]string{})
	}

	return nil
}

func (s *auditQueryService) exportHTML(report *AuditReport, writer io.Writer) error {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1 { color: #333; }
        table { border-collapse: collapse; width: 100%%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #4CAF50; color: white; }
    </style>
</head>
<body>
    <h1>%s</h1>
    <p>%s</p>
    <p><strong>Generated:</strong> %s</p>
</body>
</html>`, report.Title, report.Title, report.Description, report.GeneratedAt.Format(time.RFC3339))

	_, err := writer.Write([]byte(html))
	return err
}

func (s *auditQueryService) exportPDF(report *AuditReport, writer io.Writer) error {
	// 简化：PDF 生成需要第三方库
	return errors.InternalError( "PDF export not implemented")
}

// Utility functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getIntervalFromGrain(grain string) string {
	switch grain {
	case "hour":
		return "1h"
	case "day":
		return "1d"
	case "week":
		return "1w"
	case "month":
		return "1M"
	default:
		return "1d"
	}
}

//Personal.AI order the ending
