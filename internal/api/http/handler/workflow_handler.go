// internal/api/http/handler/workflow_handler.go
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

// WorkflowHandler handles HTTP requests for Workflow operations
type WorkflowHandler struct {
	workflowService service.WorkflowService
	logger          logging.Logger
	tracer          trace.Tracer
}

// NewWorkflowHandler creates a new WorkflowHandler instance
func NewWorkflowHandler(
	workflowService service.WorkflowService,
	logger logging.Logger,
	tracer trace.Tracer,
) *WorkflowHandler {
	return &WorkflowHandler{
		workflowService: workflowService,
		logger:          logger,
		tracer:          tracer,
	}
}

// CreateWorkflow handles POST /api/v1/workflows
// @Summary Create a new workflow
// @Description Create a new workflow with the provided configuration and steps
// @Tags workflows
// @Accept json
// @Produce json
// @Param request body dto.CreateWorkflowRequest true "Workflow creation request"
// @Success 201 {object} dto.WorkflowResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/workflows [post]
func (h *WorkflowHandler) CreateWorkflow(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "WorkflowHandler.CreateWorkflow")
	defer span.End()

	var req dto.CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Failed to bind request", "error", err)
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Invalid request body: "+err.Error(),
		))
		return
	}

	h.logger.Info(ctx, "Creating workflow", "name", req.Name, "steps", len(req.Steps))

	workflow, err := h.workflowService.CreateWorkflow(ctx, &req)
	if err != nil {
		h.logger.Error(ctx, "Failed to create workflow", "error", err)
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.Info(ctx, "Workflow created successfully", "workflow_id", workflow.ID)
	span.SetAttribute("workflow.id", workflow.ID)
	span.SetAttribute("workflow.name", workflow.Name)

	c.JSON(http.StatusCreated, workflow)
}

// GetWorkflow handles GET /api/v1/workflows/:id
// @Summary Get workflow by ID
// @Description Retrieve detailed information about a specific workflow
// @Tags workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {object} dto.WorkflowResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/workflows/{id} [get]
func (h *WorkflowHandler) GetWorkflow(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "WorkflowHandler.GetWorkflow")
	defer span.End()

	workflowID := c.Param("id")
	if workflowID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Workflow ID is required",
		))
		return
	}

	span.SetAttribute("workflow.id", workflowID)
	h.logger.Debug(ctx, "Fetching workflow", "workflow_id", workflowID)

	workflow, err := h.workflowService.GetWorkflow(ctx, workflowID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get workflow", "workflow_id", workflowID, "error", err)
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, workflow)
}

// ListWorkflows handles GET /api/v1/workflows
// @Summary List workflows
// @Description Retrieve a paginated list of workflows with optional filtering
// @Tags workflows
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param status query string false "Filter by status"
// @Param search query string false "Search by name or description"
// @Success 200 {object} dto.WorkflowListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/workflows [get]
func (h *WorkflowHandler) ListWorkflows(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "WorkflowHandler.ListWorkflows")
	defer span.End()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	filterReq := &dto.ListWorkflowsRequest{
		Page:     page,
		PageSize: pageSize,
		Status:   c.Query("status"),
		Search:   c.Query("search"),
	}

	span.SetAttribute("pagination.page", page)
	span.SetAttribute("pagination.page_size", pageSize)

	h.logger.Debug(ctx, "Listing workflows", "page", page, "page_size", pageSize)

	workflows, err := h.workflowService.ListWorkflows(ctx, filterReq)
	if err != nil {
		h.logger.Error(ctx, "Failed to list workflows", "error", err)
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, workflows)
}

// UpdateWorkflow handles PUT /api/v1/workflows/:id
// @Summary Update workflow
// @Description Update an existing workflow's configuration
// @Tags workflows
// @Accept json
// @Produce json
// @Param id path string true "Workflow ID"
// @Param request body dto.UpdateWorkflowRequest true "Workflow update request"
// @Success 200 {object} dto.WorkflowResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/workflows/{id} [put]
func (h *WorkflowHandler) UpdateWorkflow(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "WorkflowHandler.UpdateWorkflow")
	defer span.End()

	workflowID := c.Param("id")
	if workflowID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Workflow ID is required",
		))
		return
	}

	var req dto.UpdateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Failed to bind request", "error", err)
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Invalid request body: "+err.Error(),
		))
		return
	}

	span.SetAttribute("workflow.id", workflowID)
	h.logger.Info(ctx, "Updating workflow", "workflow_id", workflowID)

	workflow, err := h.workflowService.UpdateWorkflow(ctx, workflowID, &req)
	if err != nil {
		h.logger.Error(ctx, "Failed to update workflow", "workflow_id", workflowID, "error", err)
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.Info(ctx, "Workflow updated successfully", "workflow_id", workflowID)
	c.JSON(http.StatusOK, workflow)
}

// DeleteWorkflow handles DELETE /api/v1/workflows/:id
// @Summary Delete workflow
// @Description Soft delete a workflow by ID
// @Tags workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 204 "No Content"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/workflows/{id} [delete]
func (h *WorkflowHandler) DeleteWorkflow(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "WorkflowHandler.DeleteWorkflow")
	defer span.End()

	workflowID := c.Param("id")
	if workflowID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Workflow ID is required",
		))
		return
	}

	span.SetAttribute("workflow.id", workflowID)
	h.logger.Info(ctx, "Deleting workflow", "workflow_id", workflowID)

	if err := h.workflowService.DeleteWorkflow(ctx, workflowID); err != nil {
		h.logger.Error(ctx, "Failed to delete workflow", "workflow_id", workflowID, "error", err)
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.Info(ctx, "Workflow deleted successfully", "workflow_id", workflowID)
	c.Status(http.StatusNoContent)
}

// RunWorkflow handles POST /api/v1/workflows/:id/run
// @Summary Run workflow
// @Description Start executing a workflow with the provided input
// @Tags workflows
// @Accept json
// @Produce json
// @Param id path string true "Workflow ID"
// @Param request body dto.RunWorkflowRequest true "Workflow run request"
// @Success 200 {object} dto.WorkflowExecutionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/workflows/{id}/run [post]
func (h *WorkflowHandler) RunWorkflow(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "WorkflowHandler.RunWorkflow")
	defer span.End()

	workflowID := c.Param("id")
	if workflowID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Workflow ID is required",
		))
		return
	}

	var req dto.RunWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Failed to bind request", "error", err)
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Invalid request body: "+err.Error(),
		))
		return
	}

	span.SetAttribute("workflow.id", workflowID)
	span.SetAttribute("execution.async", req.Async)
	h.logger.Info(ctx, "Running workflow", "workflow_id", workflowID, "async", req.Async)

	result, err := h.workflowService.RunWorkflow(ctx, workflowID, &req)
	if err != nil {
		h.logger.Error(ctx, "Failed to run workflow", "workflow_id", workflowID, "error", err)
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.Info(ctx, "Workflow execution started", "workflow_id", workflowID, "execution_id", result.ExecutionID)
	span.SetAttribute("execution.id", result.ExecutionID)

	c.JSON(http.StatusOK, result)
}

// PauseWorkflow handles POST /api/v1/workflows/:id/executions/:execution_id/pause
// @Summary Pause workflow execution
// @Description Pause a running workflow execution
// @Tags workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Param execution_id path string true "Execution ID"
// @Success 200 {object} dto.WorkflowExecutionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/workflows/{id}/executions/{execution_id}/pause [post]
func (h *WorkflowHandler) PauseWorkflow(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "WorkflowHandler.PauseWorkflow")
	defer span.End()

	workflowID := c.Param("id")
	executionID := c.Param("execution_id")

	if workflowID == "" || executionID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Workflow ID and Execution ID are required",
		))
		return
	}

	span.SetAttribute("workflow.id", workflowID)
	span.SetAttribute("execution.id", executionID)
	h.logger.Info(ctx, "Pausing workflow execution", "workflow_id", workflowID, "execution_id", executionID)

	result, err := h.workflowService.PauseWorkflow(ctx, workflowID, executionID)
	if err != nil {
		h.logger.Error(ctx, "Failed to pause workflow", "workflow_id", workflowID, "execution_id", executionID, "error", err)
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.Info(ctx, "Workflow execution paused", "workflow_id", workflowID, "execution_id", executionID)
	c.JSON(http.StatusOK, result)
}

// ResumeWorkflow handles POST /api/v1/workflows/:id/executions/:execution_id/resume
// @Summary Resume workflow execution
// @Description Resume a paused workflow execution
// @Tags workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Param execution_id path string true "Execution ID"
// @Success 200 {object} dto.WorkflowExecutionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/workflows/{id}/executions/{execution_id}/resume [post]
func (h *WorkflowHandler) ResumeWorkflow(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "WorkflowHandler.ResumeWorkflow")
	defer span.End()

	workflowID := c.Param("id")
	executionID := c.Param("execution_id")

	if workflowID == "" || executionID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Workflow ID and Execution ID are required",
		))
		return
	}

	span.SetAttribute("workflow.id", workflowID)
	span.SetAttribute("execution.id", executionID)
	h.logger.Info(ctx, "Resuming workflow execution", "workflow_id", workflowID, "execution_id", executionID)

	result, err := h.workflowService.ResumeWorkflow(ctx, workflowID, executionID)
	if err != nil {
		h.logger.Error(ctx, "Failed to resume workflow", "workflow_id", workflowID, "execution_id", executionID, "error", err)
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.Info(ctx, "Workflow execution resumed", "workflow_id", workflowID, "execution_id", executionID)
	c.JSON(http.StatusOK, result)
}

// CancelWorkflow handles POST /api/v1/workflows/:id/executions/:execution_id/cancel
// @Summary Cancel workflow execution
// @Description Cancel a running or paused workflow execution
// @Tags workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Param execution_id path string true "Execution ID"
// @Success 200 {object} dto.WorkflowExecutionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/workflows/{id}/executions/{execution_id}/cancel [post]
func (h *WorkflowHandler) CancelWorkflow(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "WorkflowHandler.CancelWorkflow")
	defer span.End()

	workflowID := c.Param("id")
	executionID := c.Param("execution_id")

	if workflowID == "" || executionID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Workflow ID and Execution ID are required",
		))
		return
	}

	span.SetAttribute("workflow.id", workflowID)
	span.SetAttribute("execution.id", executionID)
	h.logger.Info(ctx, "Cancelling workflow execution", "workflow_id", workflowID, "execution_id", executionID)

	result, err := h.workflowService.CancelWorkflow(ctx, workflowID, executionID)
	if err != nil {
		h.logger.Error(ctx, "Failed to cancel workflow", "workflow_id", workflowID, "execution_id", executionID, "error", err)
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.Info(ctx, "Workflow execution cancelled", "workflow_id", workflowID, "execution_id", executionID)
	c.JSON(http.StatusOK, result)
}

// GetExecutionStatus handles GET /api/v1/workflows/:id/executions/:execution_id
// @Summary Get workflow execution status
// @Description Retrieve detailed status of a workflow execution
// @Tags workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Param execution_id path string true "Execution ID"
// @Success 200 {object} dto.WorkflowExecutionResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/workflows/{id}/executions/{execution_id} [get]
func (h *WorkflowHandler) GetExecutionStatus(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "WorkflowHandler.GetExecutionStatus")
	defer span.End()

	workflowID := c.Param("id")
	executionID := c.Param("execution_id")

	if workflowID == "" || executionID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Workflow ID and Execution ID are required",
		))
		return
	}

	span.SetAttribute("workflow.id", workflowID)
	span.SetAttribute("execution.id", executionID)
	h.logger.Debug(ctx, "Fetching execution status", "workflow_id", workflowID, "execution_id", executionID)

	status, err := h.workflowService.GetExecutionStatus(ctx, workflowID, executionID)
	if err != nil {
		h.logger.Error(ctx, "Failed to get execution status", "workflow_id", workflowID, "execution_id", executionID, "error", err)
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, status)
}

// ListExecutions handles GET /api/v1/workflows/:id/executions
// @Summary List workflow executions
// @Description Retrieve a paginated list of executions for a workflow
// @Tags workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param status query string false "Filter by status"
// @Success 200 {object} dto.WorkflowExecutionListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/workflows/{id}/executions [get]
func (h *WorkflowHandler) ListExecutions(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "WorkflowHandler.ListExecutions")
	defer span.End()

	workflowID := c.Param("id")
	if workflowID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Workflow ID is required",
		))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	filterReq := &dto.ListExecutionsRequest{
		WorkflowID: workflowID,
		Page:       page,
		PageSize:   pageSize,
		Status:     c.Query("status"),
	}

	span.SetAttribute("workflow.id", workflowID)
	span.SetAttribute("pagination.page", page)
	span.SetAttribute("pagination.page_size", pageSize)

	h.logger.Debug(ctx, "Listing executions", "workflow_id", workflowID, "page", page)

	executions, err := h.workflowService.ListExecutions(ctx, filterReq)
	if err != nil {
		h.logger.Error(ctx, "Failed to list executions", "workflow_id", workflowID, "error", err)
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, executions)
}

// GetExecutionLogs handles GET /api/v1/workflows/:id/executions/:execution_id/logs
// @Summary Get workflow execution logs
// @Description Retrieve logs for a specific workflow execution
// @Tags workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Param execution_id path string true "Execution ID"
// @Param step query string false "Filter by step name"
// @Success 200 {object} dto.ExecutionLogsResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/workflows/{id}/executions/{execution_id}/logs [get]
func (h *WorkflowHandler) GetExecutionLogs(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "WorkflowHandler.GetExecutionLogs")
	defer span.End()

	workflowID := c.Param("id")
	executionID := c.Param("execution_id")

	if workflowID == "" || executionID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Workflow ID and Execution ID are required",
		))
		return
	}

	stepName := c.Query("step")

	span.SetAttribute("workflow.id", workflowID)
	span.SetAttribute("execution.id", executionID)
	h.logger.Debug(ctx, "Fetching execution logs", "workflow_id", workflowID, "execution_id", executionID, "step", stepName)

	logs, err := h.workflowService.GetExecutionLogs(ctx, workflowID, executionID, stepName)
	if err != nil {
		h.logger.Error(ctx, "Failed to get execution logs", "workflow_id", workflowID, "execution_id", executionID, "error", err)
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, logs)
}

//Personal.AI order the ending
