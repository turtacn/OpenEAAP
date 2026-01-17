package dto

import (
	"fmt"
	"time"
)

// PaginationRequest 分页请求
type PaginationRequest struct {
	Page     int `form:"page" json:"page" binding:"gte=1" example:"1"`
	PageSize int `form:"page_size" json:"page_size" binding:"gte=1,lte=100" example:"20"`
}

// GetOffset 获取偏移量
func (p *PaginationRequest) GetOffset() int {
	return (p.Page - 1) * p.PageSize
}

// GetLimit 获取限制数量
func (p *PaginationRequest) GetLimit() int {
	return p.PageSize
}

// Pagination 分页响应
type Pagination struct {
	Page       int   `json:"page" example:"1"`
	PageSize   int   `json:"page_size" example:"20"`
	Total      int64 `json:"total" example:"100"`
	TotalPages int   `json:"total_pages" example:"5"`
}

// NewPagination 创建分页响应
func NewPagination(page, pageSize int, total int64) Pagination {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	return Pagination{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}

// SortRequest 排序请求
type SortRequest struct {
	SortBy    string `form:"sort_by" json:"sort_by" example:"created_at"`
	SortOrder string `form:"sort_order" json:"sort_order" binding:"omitempty,oneof=asc desc" example:"desc"`
}

// GetSortBy 获取排序字段
func (s *SortRequest) GetSortBy() string {
	if s.SortBy == "" {
		return "created_at"
	}
	return s.SortBy
}

// GetSortOrder 获取排序顺序
func (s *SortRequest) GetSortOrder() string {
	if s.SortOrder == "" {
		return "desc"
	}
	return s.SortOrder
}

// IsAscending 是否升序
func (s *SortRequest) IsAscending() bool {
	return s.GetSortOrder() == "asc"
}

// FilterRequest 过滤请求
type FilterRequest struct {
	Filters map[string]interface{} `form:"filters" json:"filters"`
}

// GetFilter 获取指定过滤器值
func (f *FilterRequest) GetFilter(key string) (interface{}, bool) {
	if f.Filters == nil {
		return nil, false
	}
	val, ok := f.Filters[key]
	return val, ok
}

// GetStringFilter 获取字符串过滤器值
func (f *FilterRequest) GetStringFilter(key string) (string, bool) {
	val, ok := f.GetFilter(key)
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetIntFilter 获取整数过滤器值
func (f *FilterRequest) GetIntFilter(key string) (int, bool) {
	val, ok := f.GetFilter(key)
	if !ok {
		return 0, false
	}
	switch v := val.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// GetBoolFilter 获取布尔过滤器值
func (f *FilterRequest) GetBoolFilter(key string) (bool, bool) {
	val, ok := f.GetFilter(key)
	if !ok {
		return false, false
	}
	b, ok := val.(bool)
	return b, ok
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Query string `form:"query" json:"query" binding:"max=500" example:"search term"`
}

// DateRangeRequest 日期范围请求
type DateRangeRequest struct {
	StartDate time.Time `form:"start_date" json:"start_date" example:"2026-01-01T00:00:00Z"`
	EndDate   time.Time `form:"end_date" json:"end_date" example:"2026-01-15T23:59:59Z"`
}

// Validate 验证日期范围
func (d *DateRangeRequest) Validate() error {
	if !d.StartDate.IsZero() && !d.EndDate.IsZero() && d.StartDate.After(d.EndDate) {
		return ErrInvalidDateRange
	}
	return nil
}

// ListRequest 通用列表请求
type ListRequest struct {
	PaginationRequest
	SortRequest
	FilterRequest
	SearchRequest
	DateRangeRequest
}

// IDRequest ID 请求
type IDRequest struct {
	ID string `uri:"id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
}

// IDsRequest 多个 ID 请求
type IDsRequest struct {
	IDs []string `json:"ids" binding:"required,min=1" example:"[\"123e4567-e89b-12d3-a456-426614174000\"]"`
}

// Response 通用响应
type Response struct {
	Code    int         `json:"code" example:"200"`
	Message string      `json:"message" example:"success"`
	Data    interface{} `json:"data,omitempty"`
}

// NewResponse 创建通用响应
func NewResponse(code int, message string, data interface{}) *Response {
	return &Response{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// SuccessResponse 成功响应
func SuccessResponse(data interface{}) *Response {
	return NewResponse(200, "success", data)
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code      int                    `json:"code" example:"400"`
	Message   string                 `json:"message" example:"Bad request"`
	Details   string                 `json:"details,omitempty" example:"Invalid input parameter"`
	RequestID string                 `json:"request_id,omitempty" example:"req-123"`
	Timestamp time.Time              `json:"timestamp" example:"2026-01-15T10:00:00Z"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Error implements the error interface for ErrorResponse
func (e *ErrorResponse) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(code int, message string, details string) *ErrorResponse {
	return &ErrorResponse{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
	}
}

// WithRequestID 设置请求 ID
func (e *ErrorResponse) WithRequestID(requestID string) *ErrorResponse {
	e.RequestID = requestID
	return e
}

// WithMetadata 设置元数据
func (e *ErrorResponse) WithMetadata(metadata map[string]interface{}) *ErrorResponse {
	e.Metadata = metadata
	return e
}

// HealthCheckResponse 健康检查响应
type HealthCheckResponse struct {
	Status    string                 `json:"status" example:"healthy"`
	Version   string                 `json:"version" example:"1.0.0"`
	Timestamp time.Time              `json:"timestamp" example:"2026-01-15T10:00:00Z"`
	Checks    map[string]CheckStatus `json:"checks,omitempty"`
}

// CheckStatus 检查状态
type CheckStatus struct {
	Status  string                 `json:"status" example:"healthy"`
	Message string                 `json:"message,omitempty" example:"Database connection is healthy"`
	Latency int64                  `json:"latency,omitempty" example:"5"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// VersionResponse 版本信息响应
type VersionResponse struct {
	Version   string    `json:"version" example:"1.0.0"`
	GitCommit string    `json:"git_commit" example:"abc123def456"`
	BuildTime time.Time `json:"build_time" example:"2026-01-15T10:00:00Z"`
	GoVersion string    `json:"go_version" example:"1.21.0"`
}

// StatusResponse 状态响应
type StatusResponse struct {
	Status    string                 `json:"status" example:"running"`
	Uptime    int64                  `json:"uptime" example:"86400"`
	Timestamp time.Time              `json:"timestamp" example:"2026-01-15T10:00:00Z"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// BatchRequest 批量操作请求
type BatchRequest struct {
	Items []interface{} `json:"items" binding:"required,min=1,max=100"`
}

// BatchResponse 批量操作响应
type BatchResponse struct {
	TotalCount   int           `json:"total_count" example:"10"`
	SuccessCount int           `json:"success_count" example:"9"`
	FailureCount int           `json:"failure_count" example:"1"`
	Results      []BatchResult `json:"results"`
}

// BatchResult 批量操作结果
type BatchResult struct {
	Index   int         `json:"index" example:"0"`
	Success bool        `json:"success" example:"true"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// NewBatchResponse 创建批量响应
func NewBatchResponse(totalCount int) *BatchResponse {
	return &BatchResponse{
		TotalCount:   totalCount,
		SuccessCount: 0,
		FailureCount: 0,
		Results:      make([]BatchResult, 0, totalCount),
	}
}

// AddSuccess 添加成功结果
func (b *BatchResponse) AddSuccess(index int, data interface{}) {
	b.SuccessCount++
	b.Results = append(b.Results, BatchResult{
		Index:   index,
		Success: true,
		Data:    data,
	})
}

// AddFailure 添加失败结果
func (b *BatchResponse) AddFailure(index int, err error) {
	b.FailureCount++
	b.Results = append(b.Results, BatchResult{
		Index:   index,
		Success: false,
		Error:   err.Error(),
	})
}

// FileUploadRequest 文件上传请求
type FileUploadRequest struct {
	Filename    string                 `json:"filename" binding:"required" example:"document.pdf"`
	ContentType string                 `json:"content_type" binding:"required" example:"application/pdf"`
	Size        int64                  `json:"size" binding:"required,gt=0" example:"1024000"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// FileUploadResponse 文件上传响应
type FileUploadResponse struct {
	FileID      string    `json:"file_id" example:"file-123"`
	Filename    string    `json:"filename" example:"document.pdf"`
	URL         string    `json:"url" example:"https://storage.example.com/files/file-123"`
	ContentType string    `json:"content_type" example:"application/pdf"`
	Size        int64     `json:"size" example:"1024000"`
	UploadedAt  time.Time `json:"uploaded_at" example:"2026-01-15T10:00:00Z"`
}

// ExportRequest 导出请求
type ExportRequest struct {
	Format  string                 `form:"format" json:"format" binding:"required,oneof=json csv xlsx pdf" example:"json"`
	Fields  []string               `form:"fields" json:"fields,omitempty" example:"[\"id\",\"name\",\"created_at\"]"`
	Filters map[string]interface{} `form:"filters" json:"filters,omitempty"`
}

// ExportResponse 导出响应
type ExportResponse struct {
	ExportID    string    `json:"export_id" example:"export-123"`
	Format      string    `json:"format" example:"json"`
	Status      string    `json:"status" example:"completed"`
	URL         string    `json:"url,omitempty" example:"https://storage.example.com/exports/export-123.json"`
	RecordCount int       `json:"record_count" example:"100"`
	FileSize    int64     `json:"file_size,omitempty" example:"1024000"`
	CreatedAt   time.Time `json:"created_at" example:"2026-01-15T10:00:00Z"`
	ExpiresAt   time.Time `json:"expires_at" example:"2026-01-16T10:00:00Z"`
}

// ImportRequest 导入请求
type ImportRequest struct {
	Format  string                 `json:"format" binding:"required,oneof=json csv xlsx" example:"json"`
	FileURL string                 `json:"file_url,omitempty" binding:"omitempty,url" example:"https://storage.example.com/imports/data.json"`
	Data    string                 `json:"data,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
	DryRun  bool                   `json:"dry_run" example:"false"`
}

// ImportResponse 导入响应
type ImportResponse struct {
	ImportID     string    `json:"import_id" example:"import-123"`
	Status       string    `json:"status" example:"completed"`
	TotalRecords int       `json:"total_records" example:"100"`
	SuccessCount int       `json:"success_count" example:"95"`
	FailureCount int       `json:"failure_count" example:"5"`
	Errors       []string  `json:"errors,omitempty"`
	CreatedAt    time.Time `json:"created_at" example:"2026-01-15T10:00:00Z"`
	CompletedAt  time.Time `json:"completed_at,omitempty" example:"2026-01-15T10:05:00Z"`
}

// WebhookEvent Webhook 事件
type WebhookEvent struct {
	EventID   string                 `json:"event_id" example:"event-123"`
	EventType string                 `json:"event_type" example:"agent.created"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp" example:"2026-01-15T10:00:00Z"`
	Signature string                 `json:"signature,omitempty"`
}

// TaskStatus 任务状态
type TaskStatus struct {
	TaskID      string                 `json:"task_id" example:"task-123"`
	Status      string                 `json:"status" example:"running"`
	Progress    float64                `json:"progress" example:"50.5"`
	Message     string                 `json:"message,omitempty" example:"Processing..."`
	Result      interface{}            `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at" example:"2026-01-15T10:00:00Z"`
	UpdatedAt   time.Time              `json:"updated_at" example:"2026-01-15T10:05:00Z"`
	CompletedAt time.Time              `json:"completed_at,omitempty" example:"2026-01-15T10:10:00Z"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationRequest 通知请求
type NotificationRequest struct {
	Type      string                 `json:"type" binding:"required,oneof=email sms push webhook" example:"email"`
	Recipient string                 `json:"recipient" binding:"required" example:"user@example.com"`
	Subject   string                 `json:"subject,omitempty" example:"Notification Subject"`
	Body      string                 `json:"body" binding:"required" example:"Notification body content"`
	Priority  string                 `json:"priority" binding:"omitempty,oneof=low normal high urgent" example:"normal"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationResponse 通知响应
type NotificationResponse struct {
	NotificationID string    `json:"notification_id" example:"notification-123"`
	Status         string    `json:"status" example:"sent"`
	SentAt         time.Time `json:"sent_at" example:"2026-01-15T10:00:00Z"`
}

// AuditLogEntry 审计日志条目
type AuditLogEntry struct {
	ID           string                 `json:"id" example:"audit-123"`
	UserID       string                 `json:"user_id" example:"user-123"`
	Action       string                 `json:"action" example:"agent.created"`
	ResourceType string                 `json:"resource_type" example:"agent"`
	ResourceID   string                 `json:"resource_id" example:"agent-123"`
	Changes      map[string]interface{} `json:"changes,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty" example:"192.168.1.1"`
	UserAgent    string                 `json:"user_agent,omitempty" example:"Mozilla/5.0"`
	Timestamp    time.Time              `json:"timestamp" example:"2026-01-15T10:00:00Z"`
}

// StatisticsRequest 统计请求
type StatisticsRequest struct {
	StartDate time.Time              `form:"start_date" json:"start_date" example:"2026-01-01T00:00:00Z"`
	EndDate   time.Time              `form:"end_date" json:"end_date" example:"2026-01-15T23:59:59Z"`
	GroupBy   string                 `form:"group_by" json:"group_by" binding:"omitempty,oneof=hour day week month" example:"day"`
	Metrics   []string               `form:"metrics" json:"metrics,omitempty" example:"[\"count\",\"avg_duration\"]"`
	Filters   map[string]interface{} `form:"filters" json:"filters,omitempty"`
}

// StatisticsResponse 统计响应
type StatisticsResponse struct {
	Metrics    map[string]interface{} `json:"metrics"`
	Timeseries []TimeseriesPoint      `json:"timeseries,omitempty"`
	StartDate  time.Time              `json:"start_date" example:"2026-01-01T00:00:00Z"`
	EndDate    time.Time              `json:"end_date" example:"2026-01-15T23:59:59Z"`
}

// TimeseriesPoint 时间序列点
type TimeseriesPoint struct {
	Timestamp time.Time              `json:"timestamp" example:"2026-01-15T10:00:00Z"`
	Values    map[string]interface{} `json:"values"`
}

// RateLimitInfo 速率限制信息
type RateLimitInfo struct {
	Limit     int       `json:"limit" example:"100"`
	Remaining int       `json:"remaining" example:"95"`
	Reset     time.Time `json:"reset" example:"2026-01-15T11:00:00Z"`
}

// CacheControl 缓存控制
type CacheControl struct {
	MaxAge         int  `json:"max_age" example:"3600"`
	NoCache        bool `json:"no_cache" example:"false"`
	NoStore        bool `json:"no_store" example:"false"`
	MustRevalidate bool `json:"must_revalidate" example:"false"`
}

// 通用错误
var (
	ErrInvalidDateRange = NewErrorResponse(400, "Invalid date range", "Start date must be before end date")
	ErrInvalidPageSize  = NewErrorResponse(400, "Invalid page size", "Page size must be between 1 and 100")
	ErrInvalidSortField = NewErrorResponse(400, "Invalid sort field", "The specified sort field is not valid")
)

//Personal.AI order the ending
