// Package agent provides domain entities and business logic for agents.
// It defines the Agent structure with core fields and domain methods
// for validation, status checking, and execution capabilities.
package agent

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Agent Entity
// ============================================================================

// Agent represents an intelligent agent in the system
type Agent struct {
	// Unique identifier
	ID string `json:"id"`

	// Agent name
	Name string `json:"name"`

	// Agent description
	Description string `json:"description"`

	// Runtime type (python, nodejs, go, etc.)
	RuntimeType RuntimeType `json:"runtime_type"`

	// Agent configuration
	Config AgentConfig `json:"config"`

	// Agent metadata
	Metadata map[string]interface{} `json:"metadata"`

	// Current status
	Status AgentStatus `json:"status"`

	// Owner user ID
	OwnerID string `json:"owner_id"`

	// Agent version
	Version string `json:"version"`

	// Tags for categorization
	Tags []string `json:"tags"`

	// Capabilities
	Capabilities []string `json:"capabilities"`

	// Model information
	Model ModelInfo `json:"model"`

	// Execution statistics
	Stats ExecutionStats `json:"stats"`

	// Creation timestamp
	CreatedAt time.Time `json:"created_at"`

	// Last update timestamp
	UpdatedAt time.Time `json:"updated_at"`

	// Last execution timestamp
	LastExecutedAt *time.Time `json:"last_executed_at,omitempty"`

	// Deletion timestamp (soft delete)
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// ============================================================================
// Agent Configuration
// ============================================================================

// AgentConfig defines agent configuration
type AgentConfig struct {
	// System prompt
	SystemPrompt string `json:"system_prompt"`

	// Model parameters
	ModelParams ModelParams `json:"model_params"`

	// Memory configuration
	Memory MemoryConfig `json:"memory"`

	// Tool configurations
	Tools []ToolConfig `json:"tools"`

	// Execution limits
	Limits ExecutionLimits `json:"limits"`

	// Retry policy
	RetryPolicy RetryPolicy `json:"retry_policy"`

	// Logging configuration
	Logging LoggingConfig `json:"logging"`

	// Environment variables
	Environment map[string]string `json:"environment"`
}

// ModelParams defines LLM model parameters
type ModelParams struct {
	// Temperature (0.0 - 2.0)
	Temperature float64 `json:"temperature"`

	// Top P sampling (0.0 - 1.0)
	TopP float64 `json:"top_p"`

	// Max tokens
	MaxTokens int `json:"max_tokens"`

	// Frequency penalty
	FrequencyPenalty float64 `json:"frequency_penalty"`

	// Presence penalty
	PresencePenalty float64 `json:"presence_penalty"`

	// Stop sequences
	StopSequences []string `json:"stop_sequences,omitempty"`
}

// MemoryConfig defines memory configuration
type MemoryConfig struct {
	// Memory type (short_term, long_term, episodic)
	Type string `json:"type"`

	// Max memory size
	MaxSize int `json:"max_size"`

	// Enable persistence
	Persistent bool `json:"persistent"`

	// Vector store configuration
	VectorStore VectorStoreConfig `json:"vector_store"`
}

// VectorStoreConfig defines vector store configuration
type VectorStoreConfig struct {
	// Provider (pinecone, weaviate, qdrant)
	Provider string `json:"provider"`

	// Collection name
	Collection string `json:"collection"`

	// Embedding model
	EmbeddingModel string `json:"embedding_model"`

	// Dimension
	Dimension int `json:"dimension"`
}

// ToolConfig defines tool configuration
type ToolConfig struct {
	// Tool name
	Name string `json:"name"`

	// Tool type
	Type string `json:"type"`

	// Tool description
	Description string `json:"description"`

	// Tool parameters
	Parameters map[string]interface{} `json:"parameters"`

	// Enable/disable tool
	Enabled bool `json:"enabled"`
}

// ExecutionLimits defines execution limits
type ExecutionLimits struct {
	// Max execution time in seconds
	MaxExecutionTime int `json:"max_execution_time"`

	// Max iterations
	MaxIterations int `json:"max_iterations"`

	// Max tokens per execution
	MaxTokensPerExecution int `json:"max_tokens_per_execution"`

	// Max concurrent executions
	MaxConcurrentExecutions int `json:"max_concurrent_executions"`
}

// RetryPolicy defines retry policy
type RetryPolicy struct {
	// Enable retry
	Enabled bool `json:"enabled"`

	// Max retry attempts
	MaxAttempts int `json:"max_attempts"`

	// Initial delay in seconds
	InitialDelay int `json:"initial_delay"`

	// Max delay in seconds
	MaxDelay int `json:"max_delay"`

	// Backoff multiplier
	BackoffMultiplier float64 `json:"backoff_multiplier"`
}

// LoggingConfig defines logging configuration
type LoggingConfig struct {
	// Log level
	Level string `json:"level"`

	// Enable request logging
	LogRequests bool `json:"log_requests"`

	// Enable response logging
	LogResponses bool `json:"log_responses"`

	// Enable trace logging
	LogTraces bool `json:"log_traces"`
}

// ============================================================================
// Agent Status
// ============================================================================

// AgentStatus represents the current status of an agent
type AgentStatus string

const (
	// AgentStatusDraft indicates agent is in draft state
	AgentStatusDraft AgentStatus = "draft"

	// AgentStatusActive indicates agent is active and ready
	AgentStatusActive AgentStatus = "active"

	// AgentStatusInactive indicates agent is inactive
	AgentStatusInactive AgentStatus = "inactive"

	// AgentStatusExecuting indicates agent is currently executing
	AgentStatusExecuting AgentStatus = "executing"

	// AgentStatusError indicates agent has an error
	AgentStatusError AgentStatus = "error"

	// AgentStatusArchived indicates agent is archived
	AgentStatusArchived AgentStatus = "archived"
)

// ============================================================================
// Runtime Type
// ============================================================================

// RuntimeType represents the agent runtime environment
type RuntimeType string

const (
	// RuntimeTypePython for Python agents
	RuntimeTypePython RuntimeType = "python"

	// RuntimeTypeNodeJS for Node.js agents
	RuntimeTypeNodeJS RuntimeType = "nodejs"

	// RuntimeTypeGo for Go agents
	RuntimeTypeGo RuntimeType = "go"

	// RuntimeTypeLLM for pure LLM agents
	RuntimeTypeLLM RuntimeType = "llm"

	// RuntimeTypeHybrid for hybrid agents
	RuntimeTypeHybrid RuntimeType = "hybrid"
)

// ============================================================================
// Model Information
// ============================================================================

// ModelInfo defines LLM model information
type ModelInfo struct {
	// Provider (openai, anthropic, google, etc.)
	Provider string `json:"provider"`

	// Model name
	Name string `json:"name"`

	// Model version
	Version string `json:"version"`

	// API endpoint
	Endpoint string `json:"endpoint,omitempty"`
}

// ============================================================================
// Execution Statistics
// ============================================================================

// ExecutionStats tracks agent execution statistics
type ExecutionStats struct {
	// Total executions
	TotalExecutions int64 `json:"total_executions"`

	// Successful executions
	SuccessfulExecutions int64 `json:"successful_executions"`

	// Failed executions
	FailedExecutions int64 `json:"failed_executions"`

	// Average execution time in milliseconds
	AvgExecutionTimeMs int64 `json:"avg_execution_time_ms"`

	// Total tokens used
	TotalTokensUsed int64 `json:"total_tokens_used"`

	// Last execution status
	LastExecutionStatus string `json:"last_execution_status,omitempty"`
}

// AgentStatistics represents aggregate statistics for agents
type AgentStatistics struct {
	// Total number of agents
	TotalCount int64 `json:"total_count"`

	// Agents by status
	ByStatus map[string]int64 `json:"by_status"`

	// Agents by runtime type
	ByRuntimeType map[string]int64 `json:"by_runtime_type"`

	// Active agents count
	ActiveCount int64 `json:"active_count"`

	// Archived agents count
	ArchivedCount int64 `json:"archived_count"`
}

// ============================================================================
// Entity Factory
// ============================================================================

// NewAgent creates a new agent entity
func NewAgent(name, description string, runtimeType RuntimeType, ownerID string) *Agent {
	now := time.Now()
	return &Agent{
		ID:           generateAgentID(),
		Name:         name,
		Description:  description,
		RuntimeType:  runtimeType,
		Status:       AgentStatusDraft,
		OwnerID:      ownerID,
		Version:      "1.0.0",
		Tags:         make([]string, 0),
		Capabilities: make([]string, 0),
		Metadata:     make(map[string]interface{}),
		Config:       defaultAgentConfig(),
		Stats:        ExecutionStats{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// generateAgentID generates a unique agent ID
func generateAgentID() string {
	return fmt.Sprintf("agent_%s", uuid.New().String())
}

// defaultAgentConfig returns default agent configuration
func defaultAgentConfig() AgentConfig {
	return AgentConfig{
		ModelParams: ModelParams{
			Temperature:      0.7,
			TopP:             1.0,
			MaxTokens:        2048,
			FrequencyPenalty: 0.0,
			PresencePenalty:  0.0,
		},
		Memory: MemoryConfig{
			Type:       "short_term",
			MaxSize:    100,
			Persistent: false,
		},
		Tools: make([]ToolConfig, 0),
		Limits: ExecutionLimits{
			MaxExecutionTime:        300,
			MaxIterations:           10,
			MaxTokensPerExecution:   10000,
			MaxConcurrentExecutions: 5,
		},
		RetryPolicy: RetryPolicy{
			Enabled:           true,
			MaxAttempts:       3,
			InitialDelay:      1,
			MaxDelay:          30,
			BackoffMultiplier: 2.0,
		},
		Logging: LoggingConfig{
			Level:        "info",
			LogRequests:  true,
			LogResponses: false,
			LogTraces:    true,
		},
		Environment: make(map[string]string),
	}
}

// ============================================================================
// Domain Methods - Validation
// ============================================================================

// Validate validates the agent entity
func (a *Agent) Validate() error {
	if a.ID == "" {
		return fmt.Errorf("agent ID is required")
	}

	if a.Name == "" {
		return fmt.Errorf("agent name is required")
	}

	if len(a.Name) < 3 || len(a.Name) > 100 {
		return fmt.Errorf("agent name must be between 3 and 100 characters")
	}

	if a.RuntimeType == "" {
		return fmt.Errorf("runtime type is required")
	}

	if !a.isValidRuntimeType() {
		return fmt.Errorf("invalid runtime type: %s", a.RuntimeType)
	}

	if a.OwnerID == "" {
		return fmt.Errorf("owner ID is required")
	}

	if err := a.validateConfig(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	return nil
}

// isValidRuntimeType checks if runtime type is valid
func (a *Agent) isValidRuntimeType() bool {
	validTypes := []RuntimeType{
		RuntimeTypePython,
		RuntimeTypeNodeJS,
		RuntimeTypeGo,
		RuntimeTypeLLM,
		RuntimeTypeHybrid,
	}

	for _, t := range validTypes {
		if a.RuntimeType == t {
			return true
		}
	}

	return false
}

// validateConfig validates agent configuration
func (a *Agent) validateConfig() error {
	if a.Config.ModelParams.Temperature < 0 || a.Config.ModelParams.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}

	if a.Config.ModelParams.TopP < 0 || a.Config.ModelParams.TopP > 1 {
		return fmt.Errorf("top_p must be between 0 and 1")
	}

	if a.Config.ModelParams.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be positive")
	}

	if a.Config.Limits.MaxExecutionTime <= 0 {
		return fmt.Errorf("max_execution_time must be positive")
	}

	return nil
}

// ============================================================================
// Domain Methods - Status
// ============================================================================

// IsActive checks if the agent is active
func (a *Agent) IsActive() bool {
	return a.Status == AgentStatusActive
}

// IsExecuting checks if the agent is currently executing
func (a *Agent) IsExecuting() bool {
	return a.Status == AgentStatusExecuting
}

// IsArchived checks if the agent is archived
func (a *Agent) IsArchived() bool {
	return a.Status == AgentStatusArchived
}

// IsDeleted checks if the agent is soft deleted
func (a *Agent) IsDeleted() bool {
	return a.DeletedAt != nil
}

// HasError checks if the agent has an error status
func (a *Agent) HasError() bool {
	return a.Status == AgentStatusError
}

// ============================================================================
// Domain Methods - Execution
// ============================================================================

// CanExecute checks if the agent can execute
func (a *Agent) CanExecute() bool {
	if a.IsDeleted() {
		return false
	}

	if a.IsArchived() {
		return false
	}

	if !a.IsActive() && !a.IsExecuting() {
		return false
	}

	return true
}

// CanConcurrentExecute checks if agent can handle concurrent execution
func (a *Agent) CanConcurrentExecute(currentExecutions int) bool {
	if !a.CanExecute() {
		return false
	}

	return currentExecutions < a.Config.Limits.MaxConcurrentExecutions
}

// ============================================================================
// Domain Methods - State Transitions
// ============================================================================

// Activate activates the agent
func (a *Agent) Activate() error {
	if a.IsDeleted() {
		return fmt.Errorf("cannot activate deleted agent")
	}

	if a.Status == AgentStatusActive {
		return fmt.Errorf("agent is already active")
	}

	a.Status = AgentStatusActive
	a.UpdatedAt = time.Now()
	return nil
}

// Deactivate deactivates the agent
func (a *Agent) Deactivate() error {
	if a.Status == AgentStatusInactive {
		return fmt.Errorf("agent is already inactive")
	}

	a.Status = AgentStatusInactive
	a.UpdatedAt = time.Now()
	return nil
}

// StartExecution marks agent as executing
func (a *Agent) StartExecution() error {
	if !a.CanExecute() {
		return fmt.Errorf("agent cannot execute in current state")
	}

	a.Status = AgentStatusExecuting
	now := time.Now()
	a.LastExecutedAt = &now
	a.UpdatedAt = now
	return nil
}

// CompleteExecution marks execution as complete
func (a *Agent) CompleteExecution(success bool, tokensUsed int64, duration time.Duration) {
	a.Stats.TotalExecutions++
	if success {
		a.Stats.SuccessfulExecutions++
		a.Stats.LastExecutionStatus = "success"
	} else {
		a.Stats.FailedExecutions++
		a.Stats.LastExecutionStatus = "failed"
	}

	a.Stats.TotalTokensUsed += tokensUsed

	// Update average execution time
	if a.Stats.TotalExecutions > 0 {
		totalTime := a.Stats.AvgExecutionTimeMs * (a.Stats.TotalExecutions - 1)
		totalTime += duration.Milliseconds()
		a.Stats.AvgExecutionTimeMs = totalTime / a.Stats.TotalExecutions
	}

	a.Status = AgentStatusActive
	a.UpdatedAt = time.Now()
}

// SetError sets agent to error state
func (a *Agent) SetError() {
	a.Status = AgentStatusError
	a.UpdatedAt = time.Now()
}

// Archive archives the agent
func (a *Agent) Archive() error {
	if a.IsDeleted() {
		return fmt.Errorf("cannot archive deleted agent")
	}

	a.Status = AgentStatusArchived
	a.UpdatedAt = time.Now()
	return nil
}

// SoftDelete soft deletes the agent
func (a *Agent) SoftDelete() error {
	if a.IsDeleted() {
		return fmt.Errorf("agent is already deleted")
	}

	now := time.Now()
	a.DeletedAt = &now
	a.Status = AgentStatusArchived
	a.UpdatedAt = now
	return nil
}

// ============================================================================
// Domain Methods - Utilities
// ============================================================================

// Update updates agent fields
func (a *Agent) Update(name, description string, config AgentConfig) {
	if name != "" {
		a.Name = name
	}
	if description != "" {
		a.Description = description
	}
	a.Config = config
	a.UpdatedAt = time.Now()
}

// AddTag adds a tag to the agent
func (a *Agent) AddTag(tag string) {
	for _, t := range a.Tags {
		if t == tag {
			return
		}
	}
	a.Tags = append(a.Tags, tag)
	a.UpdatedAt = time.Now()
}

// RemoveTag removes a tag from the agent
func (a *Agent) RemoveTag(tag string) {
	for i, t := range a.Tags {
		if t == tag {
			a.Tags = append(a.Tags[:i], a.Tags[i+1:]...)
			a.UpdatedAt = time.Now()
			return
		}
	}
}

// SetMetadata sets a metadata field
func (a *Agent) SetMetadata(key string, value interface{}) {
	a.Metadata[key] = value
	a.UpdatedAt = time.Now()
}

// ToJSON converts agent to JSON
func (a *Agent) ToJSON() ([]byte, error) {
	return json.Marshal(a)
}

// ============================================================================
// Domain Methods - Capabilities
// ============================================================================

// HasCapability checks if agent has a specific capability
func (a *Agent) HasCapability(capability string) bool {
	for _, c := range a.Capabilities {
		if c == capability {
			return true
		}
	}
	return false
}

// AddCapability adds a capability to the agent
func (a *Agent) AddCapability(capability string) {
	if !a.HasCapability(capability) {
		a.Capabilities = append(a.Capabilities, capability)
		a.UpdatedAt = time.Now()
	}
}

// GetSuccessRate returns the success rate of executions
func (a *Agent) GetSuccessRate() float64 {
	if a.Stats.TotalExecutions == 0 {
		return 0.0
	}
	return float64(a.Stats.SuccessfulExecutions) / float64(a.Stats.TotalExecutions)
}

// Clone creates a deep copy of the agent
func (a *Agent) Clone() *Agent {
	if a == nil {
		return nil
	}
	
	clone := &Agent{
		ID:             a.ID,
		Name:           a.Name,
		Description:    a.Description,
		RuntimeType:    a.RuntimeType,
		Config:         a.Config, // AgentConfig is value type, automatically copied
		Status:         a.Status,
		OwnerID:        a.OwnerID,
		Version:        a.Version,
		Model:          a.Model, // ModelInfo is value type, automatically copied
		Stats:          a.Stats, // ExecutionStats is value type, automatically copied
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
	
	// Deep copy slices
	if len(a.Tags) > 0 {
		clone.Tags = make([]string, len(a.Tags))
		copy(clone.Tags, a.Tags)
	}
	
	if len(a.Capabilities) > 0 {
		clone.Capabilities = make([]string, len(a.Capabilities))
		copy(clone.Capabilities, a.Capabilities)
	}
	
	// Deep copy map
	if len(a.Metadata) > 0 {
		clone.Metadata = make(map[string]interface{}, len(a.Metadata))
		for k, v := range a.Metadata {
			clone.Metadata[k] = v
		}
	}
	
	// Copy pointers to time
	if a.LastExecutedAt != nil {
		t := *a.LastExecutedAt
		clone.LastExecutedAt = &t
	}
	
	if a.DeletedAt != nil {
		t := *a.DeletedAt
		clone.DeletedAt = &t
	}
	
	return clone
}

//Personal.AI order the ending
