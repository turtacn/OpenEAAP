// Package agent provides domain services for agent business logic.
// It implements complex business rules including state transitions,
// configuration validation, dependency checking, and orchestration.
package agent

import (
	"context"
	"fmt"
	"time"
)

// ============================================================================
// Domain Service Interface
// ============================================================================

// AgentService defines the domain service interface for agent operations
type AgentService interface {
	// CreateAgent creates a new agent with validation
	CreateAgent(ctx context.Context, req CreateAgentRequest) (*Agent, error)

	// UpdateAgent updates an existing agent
	UpdateAgent(ctx context.Context, id string, req UpdateAgentRequest) (*Agent, error)

	// DeleteAgent deletes an agent
	DeleteAgent(ctx context.Context, id string, hard bool) error

	// GetAgent retrieves an agent by ID
	GetAgent(ctx context.Context, id string) (*Agent, error)

	// ListAgents lists agents with filtering
	ListAgents(ctx context.Context, filter AgentFilter) ([]*Agent, error)

	// ActivateAgent activates an agent
	ActivateAgent(ctx context.Context, id string) (*Agent, error)

	// DeactivateAgent deactivates an agent
	DeactivateAgent(ctx context.Context, id string) (*Agent, error)

	// ArchiveAgent archives an agent
	ArchiveAgent(ctx context.Context, id string) (*Agent, error)

	// RestoreAgent restores an archived agent
	RestoreAgent(ctx context.Context, id string) (*Agent, error)

	// ValidateAgent validates agent configuration
	ValidateAgent(ctx context.Context, agent *Agent) error

	// CheckDependencies checks agent dependencies
	CheckDependencies(ctx context.Context, agent *Agent) error

	// CloneAgent creates a copy of an agent
	CloneAgent(ctx context.Context, id string, newName string) (*Agent, error)

	// UpdateConfiguration updates agent configuration
	UpdateConfiguration(ctx context.Context, id string, config AgentConfig) (*Agent, error)

	// RecordExecution records agent execution results
	RecordExecution(ctx context.Context, id string, result ExecutionResult) error

	// GetExecutionStats retrieves execution statistics
	GetExecutionStats(ctx context.Context, id string) (*ExecutionStats, error)

	// SearchAgents searches agents by query
	SearchAgents(ctx context.Context, query string, filter AgentFilter) ([]*Agent, error)
}

// ============================================================================
// Domain Service Implementation
// ============================================================================

// agentService implements the AgentService interface
type agentService struct {
	repo      AgentRepository
	validator AgentValidator
	checker   DependencyChecker
}

// NewAgentService creates a new agent domain service
func NewAgentService(
	repo AgentRepository,
	validator AgentValidator,
	checker DependencyChecker,
) AgentService {
	return &agentService{
		repo:      repo,
		validator: validator,
		checker:   checker,
	}
}

// ============================================================================
// Request/Response Types
// ============================================================================

// CreateAgentRequest represents agent creation request
type CreateAgentRequest struct {
	Name         string
	Description  string
	RuntimeType  RuntimeType
	Config       AgentConfig
	OwnerID      string
	Tags         []string
	Capabilities []string
	Model        ModelInfo
	Metadata     map[string]interface{}
}

// UpdateAgentRequest represents agent update request
type UpdateAgentRequest struct {
	Name         *string
	Description  *string
	Config       *AgentConfig
	Tags         []string
	Capabilities []string
	Model        *ModelInfo
	Metadata     map[string]interface{}
}

// ExecutionResult represents execution outcome
type ExecutionResult struct {
	Success      bool
	TokensUsed   int64
	Duration     time.Duration
	Error        string
	OutputData   interface{}
	Metadata     map[string]interface{}
}

// ============================================================================
// Create Agent
// ============================================================================

// CreateAgent creates a new agent with full validation
func (s *agentService) CreateAgent(ctx context.Context, req CreateAgentRequest) (*Agent, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid create request: %w", err)
	}

	// Check if agent with same name exists
	exists, err := s.repo.ExistsByName(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check agent existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("agent with name '%s' already exists", req.Name)
	}

	// Create agent entity
	agent := NewAgent(req.Name, req.Description, req.RuntimeType, req.OwnerID)
	agent.Config = req.Config
	agent.Tags = req.Tags
	agent.Capabilities = req.Capabilities
	agent.Model = req.Model
	agent.Metadata = req.Metadata

	// Validate agent
	if err := s.validator.Validate(ctx, agent); err != nil {
		return nil, fmt.Errorf("agent validation failed: %w", err)
	}

	// Check dependencies
	if err := s.checker.CheckDependencies(ctx, agent); err != nil {
		return nil, fmt.Errorf("dependency check failed: %w", err)
	}

	// Validate configuration
	if err := s.validateConfiguration(agent.Config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Persist agent
	if err := s.repo.Create(ctx, agent); err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return agent, nil
}

// validateCreateRequest validates create request
func (s *agentService) validateCreateRequest(req CreateAgentRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}

	if len(req.Name) < 3 || len(req.Name) > 100 {
		return fmt.Errorf("name must be between 3 and 100 characters")
	}

	if req.RuntimeType == "" {
		return fmt.Errorf("runtime type is required")
	}

	if req.OwnerID == "" {
		return fmt.Errorf("owner ID is required")
	}

	return nil
}

// ============================================================================
// Update Agent
// ============================================================================

// UpdateAgent updates an existing agent
func (s *agentService) UpdateAgent(ctx context.Context, id string, req UpdateAgentRequest) (*Agent, error) {
	// Retrieve existing agent
	agent, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if agent.IsDeleted() {
		return nil, fmt.Errorf("cannot update deleted agent")
	}

	// Apply updates
	if req.Name != nil {
		agent.Name = *req.Name
	}

	if req.Description != nil {
		agent.Description = *req.Description
	}

	if req.Config != nil {
		// Validate new configuration
		if err := s.validateConfiguration(*req.Config); err != nil {
			return nil, fmt.Errorf("invalid configuration: %w", err)
		}
		agent.Config = *req.Config
	}

	if req.Tags != nil {
		agent.Tags = req.Tags
	}

	if req.Capabilities != nil {
		agent.Capabilities = req.Capabilities
	}

	if req.Model != nil {
		agent.Model = *req.Model
	}

	if req.Metadata != nil {
		for k, v := range req.Metadata {
			agent.Metadata[k] = v
		}
	}

	agent.UpdatedAt = time.Now()

	// Validate updated agent
	if err := s.validator.Validate(ctx, agent); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Update in repository
	if err := s.repo.Update(ctx, agent); err != nil {
		return nil, fmt.Errorf("failed to update agent: %w", err)
	}

	return agent, nil
}

// ============================================================================
// Delete Agent
// ============================================================================

// DeleteAgent deletes an agent
func (s *agentService) DeleteAgent(ctx context.Context, id string, hard bool) error {
	// Retrieve agent
	agent, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Check if agent is currently executing
	if agent.IsExecuting() {
		return fmt.Errorf("cannot delete agent while executing")
	}

	// Check dependencies before deletion
	if err := s.checker.CheckDependenciesForDeletion(ctx, agent); err != nil {
		return fmt.Errorf("cannot delete agent due to dependencies: %w", err)
	}

	if hard {
		return s.repo.Delete(ctx, id)
	}

	return s.repo.SoftDelete(ctx, id)
}

// ============================================================================
// Get Agent
// ============================================================================

// GetAgent retrieves an agent by ID
func (s *agentService) GetAgent(ctx context.Context, id string) (*Agent, error) {
	agent, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if agent.IsDeleted() {
		return nil, fmt.Errorf("agent not found")
	}

	return agent, nil
}

// ============================================================================
// List Agents
// ============================================================================

// ListAgents lists agents with filtering
func (s *agentService) ListAgents(ctx context.Context, filter AgentFilter) ([]*Agent, error) {
	agents, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	return agents, nil
}

// ============================================================================
// State Transitions
// ============================================================================

// ActivateAgent activates an agent
func (s *agentService) ActivateAgent(ctx context.Context, id string) (*Agent, error) {
	agent, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// Validate agent before activation
	if err := s.validator.Validate(ctx, agent); err != nil {
		return nil, fmt.Errorf("cannot activate invalid agent: %w", err)
	}

	// Check dependencies
	if err := s.checker.CheckDependencies(ctx, agent); err != nil {
		return nil, fmt.Errorf("cannot activate agent with missing dependencies: %w", err)
	}

	// Perform state transition
	if err := agent.Activate(); err != nil {
		return nil, fmt.Errorf("failed to activate agent: %w", err)
	}

	// Update status in repository
	if err := s.repo.UpdateStatus(ctx, id, AgentStatusActive); err != nil {
		return nil, fmt.Errorf("failed to update agent status: %w", err)
	}

	return agent, nil
}

// DeactivateAgent deactivates an agent
func (s *agentService) DeactivateAgent(ctx context.Context, id string) (*Agent, error) {
	agent, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// Check if agent is executing
	if agent.IsExecuting() {
		return nil, fmt.Errorf("cannot deactivate agent while executing")
	}

	// Perform state transition
	if err := agent.Deactivate(); err != nil {
		return nil, fmt.Errorf("failed to deactivate agent: %w", err)
	}

	// Update status
	if err := s.repo.UpdateStatus(ctx, id, AgentStatusInactive); err != nil {
		return nil, fmt.Errorf("failed to update agent status: %w", err)
	}

	return agent, nil
}

// ArchiveAgent archives an agent
func (s *agentService) ArchiveAgent(ctx context.Context, id string) (*Agent, error) {
	agent, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if err := agent.Archive(); err != nil {
		return nil, fmt.Errorf("failed to archive agent: %w", err)
	}

	if err := s.repo.Archive(ctx, id); err != nil {
		return nil, fmt.Errorf("failed to archive agent in repository: %w", err)
	}

	return agent, nil
}

// RestoreAgent restores an archived agent
func (s *agentService) RestoreAgent(ctx context.Context, id string) (*Agent, error) {
	agent, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if !agent.IsDeleted() && !agent.IsArchived() {
		return nil, fmt.Errorf("agent is not archived or deleted")
	}

	if err := s.repo.Restore(ctx, id); err != nil {
		return nil, fmt.Errorf("failed to restore agent: %w", err)
	}

	agent.DeletedAt = nil
	agent.Status = AgentStatusInactive
	agent.UpdatedAt = time.Now()

	return agent, nil
}

// ============================================================================
// Validation
// ============================================================================

// ValidateAgent validates agent configuration
func (s *agentService) ValidateAgent(ctx context.Context, agent *Agent) error {
	return s.validator.Validate(ctx, agent)
}

// validateConfiguration validates agent configuration
func (s *agentService) validateConfiguration(config AgentConfig) error {
	if config.ModelParams.Temperature < 0 || config.ModelParams.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}

	if config.ModelParams.TopP < 0 || config.ModelParams.TopP > 1 {
		return fmt.Errorf("top_p must be between 0 and 1")
	}

	if config.ModelParams.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be positive")
	}

	if config.Limits.MaxExecutionTime <= 0 {
		return fmt.Errorf("max_execution_time must be positive")
	}

	if config.Limits.MaxIterations <= 0 {
		return fmt.Errorf("max_iterations must be positive")
	}

	if config.RetryPolicy.Enabled {
		if config.RetryPolicy.MaxAttempts <= 0 {
			return fmt.Errorf("max_attempts must be positive when retry is enabled")
		}
	}

	return nil
}

// ============================================================================
// Dependencies
// ============================================================================

// CheckDependencies checks agent dependencies
func (s *agentService) CheckDependencies(ctx context.Context, agent *Agent) error {
	return s.checker.CheckDependencies(ctx, agent)
}

// ============================================================================
// Clone Agent
// ============================================================================

// CloneAgent creates a copy of an agent
func (s *agentService) CloneAgent(ctx context.Context, id string, newName string) (*Agent, error) {
	// Get original agent
	original, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get original agent: %w", err)
	}

	// Check if new name is available
	exists, err := s.repo.ExistsByName(ctx, newName)
	if err != nil {
		return nil, fmt.Errorf("failed to check name availability: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("agent with name '%s' already exists", newName)
	}

	// Clone agent
	cloned := original.Clone()
	cloned.ID = generateAgentID()
	cloned.Name = newName
	cloned.Status = AgentStatusDraft
	cloned.Stats = ExecutionStats{}
	now := time.Now()
	cloned.CreatedAt = now
	cloned.UpdatedAt = now
	cloned.LastExecutedAt = nil

	// Create cloned agent
	if err := s.repo.Create(ctx, cloned); err != nil {
		return nil, fmt.Errorf("failed to create cloned agent: %w", err)
	}

	return cloned, nil
}

// ============================================================================
// Configuration Management
// ============================================================================

// UpdateConfiguration updates agent configuration
func (s *agentService) UpdateConfiguration(ctx context.Context, id string, config AgentConfig) (*Agent, error) {
	agent, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// Validate new configuration
	if err := s.validateConfiguration(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	agent.Config = config
	agent.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, agent); err != nil {
		return nil, fmt.Errorf("failed to update agent: %w", err)
	}

	return agent, nil
}

// ============================================================================
// Execution Tracking
// ============================================================================

// RecordExecution records agent execution results
func (s *agentService) RecordExecution(ctx context.Context, id string, result ExecutionResult) error {
	agent, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Update execution stats
	agent.CompleteExecution(result.Success, result.TokensUsed, result.Duration)

	// Update stats in repository
	if err := s.repo.UpdateStats(ctx, id, agent.Stats); err != nil {
		return fmt.Errorf("failed to update stats: %w", err)
	}

	return nil
}

// GetExecutionStats retrieves execution statistics
func (s *agentService) GetExecutionStats(ctx context.Context, id string) (*ExecutionStats, error) {
	agent, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	return &agent.Stats, nil
}

// ============================================================================
// Search
// ============================================================================

// SearchAgents searches agents by query
func (s *agentService) SearchAgents(ctx context.Context, query string, filter AgentFilter) ([]*Agent, error) {
	agents, err := s.repo.Search(ctx, query, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to search agents: %w", err)
	}

	return agents, nil
}

// ============================================================================
// Validator Interface
// ============================================================================

// AgentValidator defines validation interface
type AgentValidator interface {
	Validate(ctx context.Context, agent *Agent) error
}

// ============================================================================
// Dependency Checker Interface
// ============================================================================

// DependencyChecker defines dependency checking interface
type DependencyChecker interface {
	CheckDependencies(ctx context.Context, agent *Agent) error
	CheckDependenciesForDeletion(ctx context.Context, agent *Agent) error
}

//Personal.AI order the ending
