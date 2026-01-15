package dto

import (
	"time"
)

// CreateWorkflowRequest 创建工作流请求
type CreateWorkflowRequest struct {
	Name        string           `json:"name" binding:"required,min=1,max=100" example:"CustomerOnboarding"`
	Description string           `json:"description" binding:"max=500" example:"Automated customer onboarding workflow"`
	Steps       []WorkflowStep   `json:"steps" binding:"required,min=1"`
	Trigger     *WorkflowTrigger `json:"trigger,omitempty"`
	Config      *WorkflowConfig  `json:"config,omitempty"`
}

// UpdateWorkflowRequest 更新工作流请求
type UpdateWorkflowRequest struct {
	Name        *string          `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	Description *string          `json:"description,omitempty" binding:"omitempty,max=500"`
	Steps       *[]WorkflowStep  `json:"steps,omitempty" binding:"omitempty,min=1"`
	Trigger     *WorkflowTrigger `json:"trigger,omitempty"`
	Config      *WorkflowConfig  `json:"config,omitempty"`
}

// WorkflowResponse 工作流响应
type WorkflowResponse struct {
	ID          string           `json:"id" example:"workflow-123"`
	Name        string           `json:"name" example:"CustomerOnboarding"`
	Description string           `json:"description" example:"Automated customer onboarding workflow"`
	Steps       []WorkflowStep   `json:"steps"`
	Trigger     *WorkflowTrigger `json:"trigger,omitempty"`
	Config      *WorkflowConfig  `json:"config,omitempty"`
	Status      string           `json:"status" example:"active"`
	CreatedAt   time.Time        `json:"created_at" example:"2026-01-15T10:00:00Z"`
	UpdatedAt   time.Time        `json:"updated_at" example:"2026-01-15T10:00:00Z"`
}

// WorkflowStep 工作流步骤
type WorkflowStep struct {
	ID          string                 `json:"id" binding:"required" example:"step-1"`
	Name        string                 `json:"name" binding:"required" example:"VerifyEmail"`
	Type        string                 `json:"type" binding:"required,oneof=agent task decision loop parallel webhook" example:"agent"`
	AgentID     string                 `json:"agent_id,omitempty" example:"agent-123"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Condition   string                 `json:"condition,omitempty" example:"$.user.email_verified == true"`
	RetryPolicy *RetryPolicy           `json:"retry_policy,omitempty"`
	Timeout     int                    `json:"timeout,omitempty" binding:"gte=1" example:"60"`
	NextSteps   []string               `json:"next_steps,omitempty" example:"[\"step-2\"]"`
}

// WorkflowTrigger 工作流触发器
type WorkflowTrigger struct {
	Type     string                 `json:"type" binding:"required,oneof=manual scheduled event webhook" example:"manual"`
	Schedule string                 `json:"schedule,omitempty" example:"0 9 * * *"`
	Event    string                 `json:"event,omitempty" example:"user.created"`
	Webhook  *WebhookConfig         `json:"webhook,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	URL             string            `json:"url" binding:"required,url" example:"https://api.example.com/webhook"`
	Secret          string            `json:"secret,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	VerifySignature bool              `json:"verify_signature" example:"true"`
}

// WorkflowConfig 工作流配置
type WorkflowConfig struct {
	MaxRetries      int    `json:"max_retries" binding:"gte=0,lte=10" example:"3"`
	Timeout         int    `json:"timeout" binding:"gte=1" example:"300"`
	ConcurrentSteps int    `json:"concurrent_steps" binding:"gte=1,lte=10" example:"3"`
	ErrorHandling   string `json:"error_handling" binding:"omitempty,oneof=fail continue rollback" example:"fail"`
	EnableLogging   bool   `json:"enable_logging" example:"true"`
	EnableMetrics   bool   `json:"enable_metrics" example:"true"`
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries   int     `json:"max_retries" binding:"gte=0,lte=10" example:"3"`
	BackoffType  string  `json:"backoff_type" binding:"required,oneof=fixed exponential linear" example:"exponential"`
	InitialDelay int     `json:"initial_delay" binding:"gte=100" example:"1000"`
	MaxDelay     int     `json:"max_delay" binding:"gte=1000" example:"60000"`
	Multiplier   float64 `json:"multiplier,omitempty" binding:"gte=1" example:"2.0"`
}

// RunWorkflowRequest 运行工作流请求
type RunWorkflowRequest struct {
	WorkflowID string                 `json:"workflow_id" binding:"required" example:"workflow-123"`
	Input      map[string]interface{} `json:"input,omitempty"`
	Variables  map[string]interface{} `json:"variables,omitempty"`
	Async      bool                   `json:"async" example:"false"`
	Callback   *CallbackConfig        `json:"callback,omitempty"`
}

// CallbackConfig 回调配置
type CallbackConfig struct {
	URL     string            `json:"url" binding:"required,url" example:"https://api.example.com/callback"`
	Headers map[string]string `json:"headers,omitempty"`
	Events  []string          `json:"events,omitempty" example:"[\"completed\",\"failed\"]"`
}

// WorkflowExecutionResponse 工作流执行响应
type WorkflowExecutionResponse struct {
	ExecutionID string                 `json:"execution_id" example:"exec-123"`
	WorkflowID  string                 `json:"workflow_id" example:"workflow-123"`
	Status      string                 `json:"status" example:"completed"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   time.Time              `json:"started_at" example:"2026-01-15T10:00:00Z"`
	CompletedAt time.Time              `json:"completed_at,omitempty" example:"2026-01-15T10:05:00Z"`
	Duration    int64                  `json:"duration" example:"300000"`
	StepResults []StepResult           `json:"step_results,omitempty"`
}

// StepResult 步骤执行结果
type StepResult struct {
	StepID      string                 `json:"step_id" example:"step-1"`
	StepName    string                 `json:"step_name" example:"VerifyEmail"`
	Status      string                 `json:"status" example:"completed"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   time.Time              `json:"started_at" example:"2026-01-15T10:00:00Z"`
	CompletedAt time.Time              `json:"completed_at" example:"2026-01-15T10:00:30Z"`
	Duration    int64                  `json:"duration" example:"30000"`
	RetryCount  int                    `json:"retry_count" example:"0"`
}

// WorkflowStreamEvent 工作流流式事件
type WorkflowStreamEvent struct {
	Type      string                 `json:"type" example:"step_started"`
	StepID    string                 `json:"step_id,omitempty" example:"step-1"`
	StepName  string                 `json:"step_name,omitempty" example:"VerifyEmail"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp" example:"2026-01-15T10:00:00Z"`
}

// WorkflowExecutionStatus 工作流执行状态
type WorkflowExecutionStatus struct {
	ExecutionID    string    `json:"execution_id" example:"exec-123"`
	WorkflowID     string    `json:"workflow_id" example:"workflow-123"`
	Status         string    `json:"status" example:"running"`
	CurrentStep    string    `json:"current_step,omitempty" example:"step-2"`
	CompletedSteps int       `json:"completed_steps" example:"1"`
	TotalSteps     int       `json:"total_steps" example:"5"`
	Progress       float64   `json:"progress" example:"20.0"`
	StartedAt      time.Time `json:"started_at" example:"2026-01-15T10:00:00Z"`
	UpdatedAt      time.Time `json:"updated_at" example:"2026-01-15T10:01:00Z"`
	Error          string    `json:"error,omitempty"`
}

// ListWorkflowRequest 列出工作流请求
type ListWorkflowRequest struct {
	Page     int    `form:"page" binding:"gte=1" example:"1"`
	PageSize int    `form:"page_size" binding:"gte=1,lte=100" example:"20"`
	Status   string `form:"status" binding:"omitempty,oneof=draft active inactive" example:"active"`
	Search   string `form:"search" example:"onboarding"`
}

// WorkflowListResponse 工作流列表响应
type WorkflowListResponse struct {
	Items      []*WorkflowResponse `json:"items"`
	Pagination Pagination          `json:"pagination"`
}

// ListExecutionsRequest 列出执行历史请求
type ListExecutionsRequest struct {
	Page     int    `form:"page" binding:"gte=1" example:"1"`
	PageSize int    `form:"page_size" binding:"gte=1,lte=100" example:"20"`
	Status   string `form:"status" binding:"omitempty,oneof=pending running completed failed cancelled" example:"completed"`
}

// ExecutionListResponse 执行历史列表响应
type ExecutionListResponse struct {
	Items      []*WorkflowExecutionStatus `json:"items"`
	Pagination Pagination                 `json:"pagination"`
}

// ValidateWorkflowRequest 验证工作流请求
type ValidateWorkflowRequest struct {
	Name   string          `json:"name" binding:"required"`
	Steps  []WorkflowStep  `json:"steps" binding:"required,min=1"`
	Config *WorkflowConfig `json:"config,omitempty"`
}

// WorkflowValidationResult 工作流验证结果
type WorkflowValidationResult struct {
	Valid    bool              `json:"valid" example:"true"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
	Graph    *WorkflowGraph    `json:"graph,omitempty"`
}

// WorkflowGraph 工作流图结构
type WorkflowGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GraphNode 图节点
type GraphNode struct {
	ID       string                 `json:"id" example:"step-1"`
	Type     string                 `json:"type" example:"agent"`
	Label    string                 `json:"label" example:"VerifyEmail"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// GraphEdge 图边
type GraphEdge struct {
	Source    string `json:"source" example:"step-1"`
	Target    string `json:"target" example:"step-2"`
	Condition string `json:"condition,omitempty" example:"$.email_verified == true"`
}

// WorkflowMetricsRequest 工作流指标请求
type WorkflowMetricsRequest struct {
	StartTime time.Time `form:"start_time" binding:"required" example:"2026-01-01T00:00:00Z"`
	EndTime   time.Time `form:"end_time" binding:"required" example:"2026-01-15T23:59:59Z"`
	Interval  string    `form:"interval" binding:"omitempty,oneof=hour day week" example:"day"`
}

// WorkflowMetrics 工作流指标
type WorkflowMetrics struct {
	WorkflowID        string                   `json:"workflow_id" example:"workflow-123"`
	ExecutionCount    int64                    `json:"execution_count" example:"500"`
	SuccessCount      int64                    `json:"success_count" example:"475"`
	FailureCount      int64                    `json:"failure_count" example:"25"`
	SuccessRate       float64                  `json:"success_rate" example:"0.95"`
	AvgDuration       float64                  `json:"avg_duration" example:"180000.5"`
	P50Duration       float64                  `json:"p50_duration" example:"150000"`
	P95Duration       float64                  `json:"p95_duration" example:"300000"`
	P99Duration       float64                  `json:"p99_duration" example:"450000"`
	StepMetrics       map[string]StepMetrics   `json:"step_metrics,omitempty"`
	MetricsTimeseries []MetricsTimeseriesPoint `json:"metrics_timeseries,omitempty"`
	StartTime         time.Time                `json:"start_time" example:"2026-01-01T00:00:00Z"`
	EndTime           time.Time                `json:"end_time" example:"2026-01-15T23:59:59Z"`
}

// StepMetrics 步骤指标
type StepMetrics struct {
	StepID         string  `json:"step_id" example:"step-1"`
	StepName       string  `json:"step_name" example:"VerifyEmail"`
	ExecutionCount int64   `json:"execution_count" example:"500"`
	SuccessCount   int64   `json:"success_count" example:"490"`
	FailureCount   int64   `json:"failure_count" example:"10"`
	SuccessRate    float64 `json:"success_rate" example:"0.98"`
	AvgDuration    float64 `json:"avg_duration" example:"30000.5"`
	RetryRate      float64 `json:"retry_rate" example:"0.05"`
}

// CloneWorkflowRequest 克隆工作流请求
type CloneWorkflowRequest struct {
	SourceWorkflowID string `json:"source_workflow_id" binding:"required" example:"workflow-123"`
	NewName          string `json:"new_name" binding:"required" example:"CustomerOnboarding-Copy"`
	Description      string `json:"description,omitempty" example:"Cloned from original workflow"`
}

// ExportWorkflowRequest 导出工作流请求
type ExportWorkflowRequest struct {
	Format         string `form:"format" binding:"required,oneof=json yaml" example:"json"`
	IncludeMetrics bool   `form:"include_metrics" example:"false"`
}

// ImportWorkflowRequest 导入工作流请求
type ImportWorkflowRequest struct {
	Format string `json:"format" binding:"required,oneof=json yaml" example:"json"`
	Data   string `json:"data" binding:"required"`
}

// TestWorkflowRequest 测试工作流请求
type TestWorkflowRequest struct {
	WorkflowID string                 `json:"workflow_id" binding:"required" example:"workflow-123"`
	Input      map[string]interface{} `json:"input,omitempty"`
	Variables  map[string]interface{} `json:"variables,omitempty"`
	MockSteps  map[string]interface{} `json:"mock_steps,omitempty"`
}

// TestWorkflowResponse 测试工作流响应
type TestWorkflowResponse struct {
	Success     bool                   `json:"success" example:"true"`
	Output      map[string]interface{} `json:"output,omitempty"`
	StepResults []StepResult           `json:"step_results"`
	Duration    int64                  `json:"duration" example:"150000"`
	Errors      []string               `json:"errors,omitempty"`
}

// ScheduleWorkflowRequest 调度工作流请求
type ScheduleWorkflowRequest struct {
	WorkflowID string                 `json:"workflow_id" binding:"required" example:"workflow-123"`
	Schedule   string                 `json:"schedule" binding:"required" example:"0 9 * * *"`
	Input      map[string]interface{} `json:"input,omitempty"`
	Variables  map[string]interface{} `json:"variables,omitempty"`
	Timezone   string                 `json:"timezone" example:"UTC"`
	Enabled    bool                   `json:"enabled" example:"true"`
}

// ScheduledWorkflowResponse 调度工作流响应
type ScheduledWorkflowResponse struct {
	ScheduleID  string    `json:"schedule_id" example:"schedule-123"`
	WorkflowID  string    `json:"workflow_id" example:"workflow-123"`
	Schedule    string    `json:"schedule" example:"0 9 * * *"`
	Timezone    string    `json:"timezone" example:"UTC"`
	Enabled     bool      `json:"enabled" example:"true"`
	NextRunTime time.Time `json:"next_run_time" example:"2026-01-16T09:00:00Z"`
	LastRunTime time.Time `json:"last_run_time,omitempty" example:"2026-01-15T09:00:00Z"`
	CreatedAt   time.Time `json:"created_at" example:"2026-01-15T10:00:00Z"`
	UpdatedAt   time.Time `json:"updated_at" example:"2026-01-15T10:00:00Z"`
}

// WorkflowVersionRequest 工作流版本请求
type WorkflowVersionRequest struct {
	Version     string `json:"version" binding:"required" example:"v1.0.0"`
	Description string `json:"description,omitempty" example:"Initial release"`
}

// WorkflowVersionResponse 工作流版本响应
type WorkflowVersionResponse struct {
	ID          string            `json:"id" example:"version-123"`
	WorkflowID  string            `json:"workflow_id" example:"workflow-123"`
	Version     string            `json:"version" example:"v1.0.0"`
	Description string            `json:"description" example:"Initial release"`
	Snapshot    *WorkflowResponse `json:"snapshot"`
	CreatedAt   time.Time         `json:"created_at" example:"2026-01-15T10:00:00Z"`
	CreatedBy   string            `json:"created_by,omitempty" example:"user-123"`
}

// ListWorkflowVersionsRequest 列出工作流版本请求
type ListWorkflowVersionsRequest struct {
	Page     int `form:"page" binding:"gte=1" example:"1"`
	PageSize int `form:"page_size" binding:"gte=1,lte=100" example:"20"`
}

// WorkflowVersionListResponse 工作流版本列表响应
type WorkflowVersionListResponse struct {
	Items      []*WorkflowVersionResponse `json:"items"`
	Pagination Pagination                 `json:"pagination"`
}

// RollbackWorkflowRequest 回滚工作流请求
type RollbackWorkflowRequest struct {
	VersionID string `json:"version_id" binding:"required" example:"version-123"`
}

//Personal.AI order the ending
