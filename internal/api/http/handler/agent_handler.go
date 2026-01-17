// internal/api/http/handler/agent_handler.go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/openeeap/openeeap/internal/app/dto"
	"github.com/openeeap/openeeap/internal/app/service"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/pkg/errors"
)

// AgentHandler handles HTTP requests for Agent operations
type AgentHandler struct {
	agentService service.AgentService
	logger       logging.Logger
	tracer       trace.Tracer
}

// NewAgentHandler creates a new AgentHandler instance
func NewAgentHandler(
	agentService service.AgentService,
	logger logging.Logger,
	tracer trace.Tracer,
) *AgentHandler {
	return &AgentHandler{
		agentService: agentService,
		logger:       logger,
		tracer:       tracer,
	}
}

// CreateAgent handles POST /api/v1/agents
// @Summary Create a new agent
// @Description Create a new agent with the provided configuration
// @Tags agents
// @Accept json
// @Produce json
// @Param request body dto.CreateAgentRequest true "Agent creation request"
// @Success 201 {object} dto.AgentResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/agents [post]
func (h *AgentHandler) CreateAgent(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "AgentHandler.CreateAgent")
	defer span.End()

	var req dto.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("Failed to bind request", logging.Error(err))
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Invalid request body: "+err.Error(),
		))
		return
	}

	h.logger.WithContext(ctx).Info("Creating agent", logging.Any("name", req.Name), logging.Any("runtime_type", req.RuntimeType))

	agent, err := h.agentService.CreateAgent(ctx, &req)
	if err != nil {
		h.logger.WithContext(ctx).Error("Failed to create agent", logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.WithContext(ctx).Info("Agent created successfully", logging.Any("agent_id", agent.ID))
	span.SetAttribute("agent.id", agent.ID)
	span.SetAttribute("agent.name", agent.Name)

	c.JSON(http.StatusCreated, agent)
}

// GetAgent handles GET /api/v1/agents/:id
// @Summary Get agent by ID
// @Description Retrieve detailed information about a specific agent
// @Tags agents
// @Produce json
// @Param id path string true "Agent ID"
// @Success 200 {object} dto.AgentResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/agents/{id} [get]
func (h *AgentHandler) GetAgent(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "AgentHandler.GetAgent")
	defer span.End()

	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Agent ID is required",
		))
		return
	}

	span.SetAttribute("agent.id", agentID)
	h.logger.WithContext(ctx).Debug("Fetching agent", logging.Any("agent_id", agentID))

	agent, err := h.agentService.GetAgent(ctx, agentID)
	if err != nil {
		h.logger.WithContext(ctx).Error("Failed to get agent", logging.Any("agent_id", agentID), logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, agent)
}

// ListAgents handles GET /api/v1/agents
// @Summary List agents
// @Description Retrieve a paginated list of agents with optional filtering
// @Tags agents
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param runtime_type query string false "Filter by runtime type"
// @Param status query string false "Filter by status"
// @Param search query string false "Search by name or description"
// @Success 200 {object} dto.AgentListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/agents [get]
func (h *AgentHandler) ListAgents(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "AgentHandler.ListAgents")
	defer span.End()

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Build filter request
	filterReq := &dto.ListAgentsRequest{
		Page:        page,
		PageSize:    pageSize,
		RuntimeType: c.Query("runtime_type"),
		Status:      c.Query("status"),
		Search:      c.Query("search"),
	}

	span.SetAttribute("pagination.page", page)
	span.SetAttribute("pagination.page_size", pageSize)

	h.logger.WithContext(ctx).Debug("Listing agents", logging.Any("page", page), logging.Any("page_size", pageSize))

	agents, err := h.agentService.ListAgents(ctx, filterReq)
	if err != nil {
		h.logger.WithContext(ctx).Error("Failed to list agents", logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, agents)
}

// UpdateAgent handles PUT /api/v1/agents/:id
// @Summary Update agent
// @Description Update an existing agent's configuration
// @Tags agents
// @Accept json
// @Produce json
// @Param id path string true "Agent ID"
// @Param request body dto.UpdateAgentRequest true "Agent update request"
// @Success 200 {object} dto.AgentResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/agents/{id} [put]
func (h *AgentHandler) UpdateAgent(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "AgentHandler.UpdateAgent")
	defer span.End()

	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Agent ID is required",
		))
		return
	}

	var req dto.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("Failed to bind request", logging.Error(err))
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Invalid request body: "+err.Error(),
		))
		return
	}

	span.SetAttribute("agent.id", agentID)
	h.logger.WithContext(ctx).Info("Updating agent", logging.Any("agent_id", agentID))

	agent, err := h.agentService.UpdateAgent(ctx, agentID, &req)
	if err != nil {
		h.logger.WithContext(ctx).Error("Failed to update agent", logging.Any("agent_id", agentID), logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.WithContext(ctx).Info("Agent updated successfully", logging.Any("agent_id", agentID))
	c.JSON(http.StatusOK, agent)
}

// DeleteAgent handles DELETE /api/v1/agents/:id
// @Summary Delete agent
// @Description Soft delete an agent by ID
// @Tags agents
// @Produce json
// @Param id path string true "Agent ID"
// @Success 204 "No Content"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/agents/{id} [delete]
func (h *AgentHandler) DeleteAgent(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "AgentHandler.DeleteAgent")
	defer span.End()

	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Agent ID is required",
		))
		return
	}

	span.SetAttribute("agent.id", agentID)
	h.logger.WithContext(ctx).Info("Deleting agent", logging.Any("agent_id", agentID))

	if err := h.agentService.DeleteAgent(ctx, agentID); err != nil {
		h.logger.WithContext(ctx).Error("Failed to delete agent", logging.Any("agent_id", agentID), logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.WithContext(ctx).Info("Agent deleted successfully", logging.Any("agent_id", agentID))
	c.Status(http.StatusNoContent)
}

// ExecuteAgent handles POST /api/v1/agents/:id/execute
// @Summary Execute agent
// @Description Execute an agent with the provided input
// @Tags agents
// @Accept json
// @Produce json
// @Param id path string true "Agent ID"
// @Param request body dto.ExecuteAgentRequest true "Agent execution request"
// @Success 200 {object} dto.ExecutionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/agents/{id}/execute [post]
func (h *AgentHandler) ExecuteAgent(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "AgentHandler.ExecuteAgent")
	defer span.End()

	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Agent ID is required",
		))
		return
	}

	var req dto.ExecuteAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("Failed to bind request", logging.Error(err))
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Invalid request body: "+err.Error(),
		))
		return
	}

	span.SetAttribute("agent.id", agentID)
	span.SetAttribute("execution.stream", req.Stream)
	h.logger.WithContext(ctx).Info("Executing agent", logging.Any("agent_id", agentID), logging.Any("stream", req.Stream))

	result, err := h.agentService.ExecuteAgent(ctx, agentID, &req)
	if err != nil {
		h.logger.WithContext(ctx).Error("Failed to execute agent", logging.Any("agent_id", agentID), logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.WithContext(ctx).Info("Agent executed successfully", logging.Any("agent_id", agentID), logging.Any("execution_id", result.ExecutionID))
	span.SetAttribute("execution.id", result.ExecutionID)

	c.JSON(http.StatusOK, result)
}

// TestAgent handles POST /api/v1/agents/:id/test
// @Summary Test agent
// @Description Test an agent configuration without persisting execution
// @Tags agents
// @Accept json
// @Produce json
// @Param id path string true "Agent ID"
// @Param request body dto.TestAgentRequest true "Agent test request"
// @Success 200 {object} dto.TestResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/agents/{id}/test [post]
func (h *AgentHandler) TestAgent(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "AgentHandler.TestAgent")
	defer span.End()

	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Agent ID is required",
		))
		return
	}

	var req dto.TestAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("Failed to bind request", logging.Error(err))
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Invalid request body: "+err.Error(),
		))
		return
	}

	span.SetAttribute("agent.id", agentID)
	h.logger.WithContext(ctx).Info("Testing agent", logging.Any("agent_id", agentID))

	result, err := h.agentService.TestAgent(ctx, agentID, &req)
	if err != nil {
		h.logger.WithContext(ctx).Error("Failed to test agent", logging.Any("agent_id", agentID), logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.WithContext(ctx).Info("Agent tested successfully", logging.Any("agent_id", agentID))
	c.JSON(http.StatusOK, result)
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(code, message string) *ErrorResponse {
	return &ErrorResponse{
		Code:    code,
		Message: message,
	}
}

// handleServiceError maps service errors to HTTP responses
func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.IsNotFoundError(err):
		c.JSON(http.StatusNotFound, NewErrorResponse(
			errors.ErrNotFound,
			err.Error(),
		))
	case errors.IsValidationError(err):
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrValidationFailed,
			err.Error(),
		))
	case errors.IsConflictError(err):
		c.JSON(http.StatusConflict, NewErrorResponse(
			errors.ErrConflict,
			err.Error(),
		))
	case errors.IsUnauthorizedError(err):
		c.JSON(http.StatusUnauthorized, NewErrorResponse(
			errors.UnauthorizedError,
			err.Error(),
		))
	case errors.IsForbiddenError(err):
		c.JSON(http.StatusForbidden, NewErrorResponse(
			errors.ErrForbidden,
			err.Error(),
		))
	default:
		c.JSON(http.StatusInternalServerError, NewErrorResponse(
			errors.ErrInternalServer,
			"An internal error occurred",
		))
	}
}

//Personal.AI order the ending
