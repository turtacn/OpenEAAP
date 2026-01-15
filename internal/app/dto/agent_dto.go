package dto

import (
	"time"
)

// CreateAgentRequest 创建 Agent 请求
type CreateAgentRequest struct {
	Name         string                 `json:"name" binding:"required,min=1,max=100" example:"CustomerServiceAgent"`
	Description  string                 `json:"description" binding:"max=500" example:"AI-powered customer service agent"`
	Type         string                 `json:"type" binding:"required,oneof=conversational task_oriented autonomous" example:"conversational"`
	ModelID      string                 `json:"model_id" binding:"required" example:"model-123"`
	SystemPrompt string                 `json:"system_prompt" binding:"required,min=1" example:"You are a helpful customer service assistant."`
	Config       *AgentConfig           `json:"config,omitempty"`
	Tools        []AgentTool            `json:"tools,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateAgentRequest 更新 Agent 请求
type UpdateAgentRequest struct {
	Name         *string                 `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	Description  *string                 `json:"description,omitempty" binding:"omitempty,max=500"`
	ModelID      *string                 `json:"model_id,omitempty"`
	SystemPrompt *string                 `json:"system_prompt,omitempty" binding:"omitempty,min=1"`
	Config       *AgentConfig            `json:"config,omitempty"`
	Tools        *[]AgentTool            `json:"tools,omitempty"`
	Metadata     *map[string]interface{} `json:"metadata,omitempty"`
}

// AgentResponse Agent 响应
type AgentResponse struct {
	ID           string                 `json:"id" example:"agent-123"`
	Name         string                 `json:"name" example:"CustomerServiceAgent"`
	Description  string                 `json:"description" example:"AI-powered customer service agent"`
	Type         string                 `json:"type" example:"conversational"`
	ModelID      string                 `json:"model_id" example:"model-123"`
	SystemPrompt string                 `json:"system_prompt" example:"You are a helpful assistant."`
	Config       *AgentConfig           `json:"config,omitempty"`
	Tools        []AgentTool            `json:"tools,omitempty"`
	Status       string                 `json:"status" example:"active"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at" example:"2026-01-15T10:00:00Z"`
	UpdatedAt    time.Time              `json:"updated_at" example:"2026-01-15T10:00:00Z"`
}

// AgentConfig Agent 配置
type AgentConfig struct {
	Temperature      float64       `json:"temperature" binding:"gte=0,lte=2" example:"0.7"`
	MaxTokens        int           `json:"max_tokens" binding:"gte=1,lte=32000" example:"2048"`
	TopP             float64       `json:"top_p" binding:"gte=0,lte=1" example:"0.9"`
	FrequencyPenalty float64       `json:"frequency_penalty" binding:"gte=-2,lte=2" example:"0.0"`
	PresencePenalty  float64       `json:"presence_penalty" binding:"gte=-2,lte=2" example:"0.0"`
	StopSequences    []string      `json:"stop_sequences,omitempty" example:"[\"END\"]"`
	Timeout          int           `json:"timeout" binding:"gte=1" example:"30"`
	MaxRetries       int           `json:"max_retries" binding:"gte=0,lte=10" example:"3"`
	EnableMemory     bool          `json:"enable_memory" example:"true"`
	MemoryConfig     *MemoryConfig `json:"memory_config,omitempty"`
	EnableRAG        bool          `json:"enable_rag" example:"false"`
	RAGConfig        *RAGConfig    `json:"rag_config,omitempty"`
}

// MemoryConfig 记忆配置
type MemoryConfig struct {
	Type                 string `json:"type" binding:"required,oneof=short_term long_term hybrid" example:"hybrid"`
	MaxHistoryLength     int    `json:"max_history_length" binding:"gte=1,lte=100" example:"10"`
	SummarizationEnabled bool   `json:"summarization_enabled" example:"true"`
	PersistenceEnabled   bool   `json:"persistence_enabled" example:"true"`
}

// RAGConfig RAG 配置
type RAGConfig struct {
	CollectionID    string  `json:"collection_id" binding:"required" example:"collection-123"`
	TopK            int     `json:"top_k" binding:"gte=1,lte=20" example:"5"`
	MinScore        float64 `json:"min_score" binding:"gte=0,lte=1" example:"0.7"`
	HybridSearch    bool    `json:"hybrid_search" example:"true"`
	VectorWeight    float64 `json:"vector_weight" binding:"gte=0,lte=1" example:"0.7"`
	KeywordWeight   float64 `json:"keyword_weight" binding:"gte=0,lte=1" example:"0.3"`
	ReRankEnabled   bool    `json:"rerank_enabled" example:"true"`
	CitationEnabled bool    `json:"citation_enabled" example:"true"`
}

// AgentTool Agent 工具定义
type AgentTool struct {
	Name        string                 `json:"name" binding:"required" example:"search"`
	Description string                 `json:"description" binding:"required" example:"Search the knowledge base"`
	Type        string                 `json:"type" binding:"required,oneof=function api workflow" example:"function"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Parameters  *ToolParameters        `json:"parameters,omitempty"`
}

// ToolParameters 工具参数定义
type ToolParameters struct {
	Type       string                           `json:"type" example:"object"`
	Properties map[string]ToolParameterProperty `json:"properties"`
	Required   []string                         `json:"required,omitempty"`
}

// ToolParameterProperty 工具参数属性
type ToolParameterProperty struct {
	Type        string   `json:"type" example:"string"`
	Description string   `json:"description" example:"The search query"`
	Enum        []string `json:"enum,omitempty"`
}

// ExecuteAgentRequest 执行 Agent 请求
type ExecuteAgentRequest struct {
	AgentID   string                 `json:"agent_id" binding:"required" example:"agent-123"`
	Input     string                 `json:"input" binding:"required,min=1" example:"How can I track my order?"`
	SessionID string                 `json:"session_id,omitempty" example:"session-456"`
	Variables map[string]interface{} `json:"variables,omitempty"`
	Stream    bool                   `json:"stream,omitempty" example:"false"`
	Context   *ExecutionContext      `json:"context,omitempty"`
}

// ExecutionContext 执行上下文
type ExecutionContext struct {
	UserID         string                 `json:"user_id,omitempty" example:"user-789"`
	ConversationID string                 `json:"conversation_id,omitempty" example:"conv-101"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// AgentExecutionResponse Agent 执行响应
type AgentExecutionResponse struct {
	ExecutionID string                 `json:"execution_id" example:"exec-123"`
	AgentID     string                 `json:"agent_id" example:"agent-123"`
	Output      string                 `json:"output" example:"Your order #12345 is out for delivery."`
	ToolCalls   []ToolCall             `json:"tool_calls,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	TokensUsed  TokenUsage             `json:"tokens_used"`
	Duration    int64                  `json:"duration" example:"1500"`
	Timestamp   time.Time              `json:"timestamp" example:"2026-01-15T10:00:00Z"`
}

// ToolCall 工具调用记录
type ToolCall struct {
	ToolName string                 `json:"tool_name" example:"search"`
	Input    map[string]interface{} `json:"input"`
	Output   interface{}            `json:"output"`
	Duration int64                  `json:"duration" example:"500"`
	Success  bool                   `json:"success" example:"true"`
	Error    string                 `json:"error,omitempty"`
}

// TokenUsage Token 使用情况
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens" example:"50"`
	CompletionTokens int `json:"completion_tokens" example:"30"`
	TotalTokens      int `json:"total_tokens" example:"80"`
}

// AgentStreamEvent Agent 流式事件
type AgentStreamEvent struct {
	Type      string                 `json:"type" example:"token"`
	Data      interface{}            `json:"data"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp" example:"2026-01-15T10:00:00Z"`
}

// ListAgentsRequest 列出 Agent 请求
type ListAgentsRequest struct {
	Page     int    `form:"page" binding:"gte=1" example:"1"`
	PageSize int    `form:"page_size" binding:"gte=1,lte=100" example:"20"`
	Type     string `form:"type" binding:"omitempty,oneof=conversational task_oriented autonomous" example:"conversational"`
	Status   string `form:"status" binding:"omitempty,oneof=active inactive" example:"active"`
	Search   string `form:"search" example:"customer"`
}

// AgentListResponse Agent 列表响应
type AgentListResponse struct {
	Items      []*AgentResponse `json:"items"`
	Pagination Pagination       `json:"pagination"`
}

// GetAgentMetricsRequest 获取 Agent 指标请求
type GetAgentMetricsRequest struct {
	StartTime time.Time `form:"start_time" binding:"required" example:"2026-01-01T00:00:00Z"`
	EndTime   time.Time `form:"end_time" binding:"required" example:"2026-01-15T23:59:59Z"`
	Interval  string    `form:"interval" binding:"omitempty,oneof=hour day week" example:"hour"`
}

// AgentMetrics Agent 指标
type AgentMetrics struct {
	AgentID           string                   `json:"agent_id" example:"agent-123"`
	ExecutionCount    int64                    `json:"execution_count" example:"1000"`
	SuccessCount      int64                    `json:"success_count" example:"950"`
	ErrorCount        int64                    `json:"error_count" example:"50"`
	SuccessRate       float64                  `json:"success_rate" example:"0.95"`
	AvgDuration       float64                  `json:"avg_duration" example:"1200.5"`
	AvgTokensUsed     float64                  `json:"avg_tokens_used" example:"75.3"`
	TotalTokensUsed   int64                    `json:"total_tokens_used" example:"75300"`
	ToolCallCount     int64                    `json:"tool_call_count" example:"250"`
	UserSatisfaction  float64                  `json:"user_satisfaction,omitempty" example:"4.5"`
	MetricsTimeseries []MetricsTimeseriesPoint `json:"metrics_timeseries,omitempty"`
	StartTime         time.Time                `json:"start_time" example:"2026-01-01T00:00:00Z"`
	EndTime           time.Time                `json:"end_time" example:"2026-01-15T23:59:59Z"`
}

// MetricsTimeseriesPoint 指标时间序列点
type MetricsTimeseriesPoint struct {
	Timestamp      time.Time `json:"timestamp" example:"2026-01-15T10:00:00Z"`
	ExecutionCount int64     `json:"execution_count" example:"50"`
	AvgDuration    float64   `json:"avg_duration" example:"1150.2"`
	ErrorRate      float64   `json:"error_rate" example:"0.05"`
}

// AgentSessionRequest Agent 会话请求
type AgentSessionRequest struct {
	AgentID  string                 `json:"agent_id" binding:"required" example:"agent-123"`
	UserID   string                 `json:"user_id,omitempty" example:"user-789"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AgentSessionResponse Agent 会话响应
type AgentSessionResponse struct {
	SessionID string                 `json:"session_id" example:"session-456"`
	AgentID   string                 `json:"agent_id" example:"agent-123"`
	UserID    string                 `json:"user_id,omitempty" example:"user-789"`
	Status    string                 `json:"status" example:"active"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at" example:"2026-01-15T10:00:00Z"`
	UpdatedAt time.Time              `json:"updated_at" example:"2026-01-15T10:00:00Z"`
	ExpiresAt time.Time              `json:"expires_at" example:"2026-01-15T11:00:00Z"`
}

// GetSessionHistoryRequest 获取会话历史请求
type GetSessionHistoryRequest struct {
	Page     int `form:"page" binding:"gte=1" example:"1"`
	PageSize int `form:"page_size" binding:"gte=1,lte=100" example:"20"`
}

// SessionHistoryResponse 会话历史响应
type SessionHistoryResponse struct {
	Items      []*SessionMessage `json:"items"`
	Pagination Pagination        `json:"pagination"`
}

// SessionMessage 会话消息
type SessionMessage struct {
	ID        string                 `json:"id" example:"msg-123"`
	SessionID string                 `json:"session_id" example:"session-456"`
	Role      string                 `json:"role" example:"user"`
	Content   string                 `json:"content" example:"How can I track my order?"`
	ToolCalls []ToolCall             `json:"tool_calls,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp" example:"2026-01-15T10:00:00Z"`
}

// ValidateAgentRequest 验证 Agent 请求
type ValidateAgentRequest struct {
	Name         string       `json:"name" binding:"required"`
	Type         string       `json:"type" binding:"required"`
	ModelID      string       `json:"model_id" binding:"required"`
	SystemPrompt string       `json:"system_prompt" binding:"required"`
	Config       *AgentConfig `json:"config,omitempty"`
	Tools        []AgentTool  `json:"tools,omitempty"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid    bool              `json:"valid" example:"true"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string `json:"field" example:"system_prompt"`
	Message string `json:"message" example:"system_prompt cannot be empty"`
	Code    string `json:"code" example:"required"`
}

// CloneAgentRequest 克隆 Agent 请求
type CloneAgentRequest struct {
	SourceAgentID string `json:"source_agent_id" binding:"required" example:"agent-123"`
	NewName       string `json:"new_name" binding:"required" example:"CustomerServiceAgent-Copy"`
	Description   string `json:"description,omitempty" example:"Cloned from original agent"`
}

// ExportAgentRequest 导出 Agent 请求
type ExportAgentRequest struct {
	Format         string `form:"format" binding:"required,oneof=json yaml" example:"json"`
	IncludeMetrics bool   `form:"include_metrics" example:"false"`
}

// ImportAgentRequest 导入 Agent 请求
type ImportAgentRequest struct {
	Format string `json:"format" binding:"required,oneof=json yaml" example:"json"`
	Data   string `json:"data" binding:"required"`
}

// BatchOperationRequest 批量操作请求
type BatchOperationRequest struct {
	AgentIDs  []string `json:"agent_ids" binding:"required,min=1" example:"[\"agent-123\",\"agent-456\"]"`
	Operation string   `json:"operation" binding:"required,oneof=activate deactivate delete" example:"activate"`
}

// BatchOperationResponse 批量操作响应
type BatchOperationResponse struct {
	TotalCount   int                    `json:"total_count" example:"2"`
	SuccessCount int                    `json:"success_count" example:"2"`
	FailureCount int                    `json:"failure_count" example:"0"`
	Results      []BatchOperationResult `json:"results"`
}

// BatchOperationResult 批量操作结果
type BatchOperationResult struct {
	AgentID string `json:"agent_id" example:"agent-123"`
	Success bool   `json:"success" example:"true"`
	Error   string `json:"error,omitempty"`
}

// Pagination 分页信息
type Pagination struct {
	Page       int   `json:"page" example:"1"`
	PageSize   int   `json:"page_size" example:"20"`
	Total      int64 `json:"total" example:"100"`
	TotalPages int   `json:"total_pages" example:"5"`
}

//Personal.AI order the ending
