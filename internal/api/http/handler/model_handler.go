// internal/api/http/handler/model_handler.go
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

// ModelHandler handles HTTP requests for Model operations
type ModelHandler struct {
	modelService service.ModelService
	logger       logging.Logger
	tracer       trace.Tracer
}

// NewModelHandler creates a new ModelHandler instance
func NewModelHandler(
	modelService service.ModelService,
	logger logging.Logger,
	tracer trace.Tracer,
) *ModelHandler {
	return &ModelHandler{
		modelService: modelService,
		logger:       logger,
		tracer:       tracer,
	}
}

// RegisterModel handles POST /api/v1/models
// @Summary Register a new model
// @Description Register a new model with the provided configuration
// @Tags models
// @Accept json
// @Produce json
// @Param request body dto.RegisterModelRequest true "Model registration request"
// @Success 201 {object} dto.ModelResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/models [post]
func (h *ModelHandler) RegisterModel(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "ModelHandler.RegisterModel")
	defer span.End()

	var req dto.RegisterModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("Failed to bind request", logging.Error(err))
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Invalid request body: "+err.Error(),
		))
		return
	}

	h.logger.WithContext(ctx).Info("Registering model", logging.Any("name", req.Name), logging.Any("type", req.Type))

	model, err := h.modelService.RegisterModel(ctx, &req)
	if err != nil {
		h.logger.WithContext(ctx).Error("Failed to register model", logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.WithContext(ctx).Info("Model registered successfully", logging.Any("model_id", model.ID))
	span.SetAttribute("model.id", model.ID)
	span.SetAttribute("model.name", model.Name)

	c.JSON(http.StatusCreated, model)
}

// GetModel handles GET /api/v1/models/:id
// @Summary Get model by ID
// @Description Retrieve detailed information about a specific model
// @Tags models
// @Produce json
// @Param id path string true "Model ID"
// @Success 200 {object} dto.ModelResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/models/{id} [get]
func (h *ModelHandler) GetModel(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "ModelHandler.GetModel")
	defer span.End()

	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Model ID is required",
		))
		return
	}

	span.SetAttribute("model.id", modelID)
	h.logger.WithContext(ctx).Debug("Fetching model", logging.Any("model_id", modelID))

	model, err := h.modelService.GetModel(ctx, modelID)
	if err != nil {
		h.logger.WithContext(ctx).Error("Failed to get model", logging.Any("model_id", modelID), logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, model)
}

// ListModels handles GET /api/v1/models
// @Summary List models
// @Description Retrieve a paginated list of models with optional filtering
// @Tags models
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param type query string false "Filter by model type"
// @Param status query string false "Filter by status"
// @Param search query string false "Search by name or description"
// @Success 200 {object} dto.ModelListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/models [get]
func (h *ModelHandler) ListModels(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "ModelHandler.ListModels")
	defer span.End()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	filterReq := &dto.ListModelsRequest{
		Page:      page,
		PageSize:  pageSize,
		ModelType: c.Query("type"),
		Status:    c.Query("status"),
		Search:    c.Query("search"),
	}

	span.SetAttribute("pagination.page", page)
	span.SetAttribute("pagination.page_size", pageSize)

	h.logger.WithContext(ctx).Debug("Listing models", logging.Any("page", page), logging.Any("page_size", pageSize))

	models, err := h.modelService.ListModels(ctx, filterReq)
	if err != nil {
		h.logger.WithContext(ctx).Error("Failed to list models", logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models)
}

// UpdateModel handles PUT /api/v1/models/:id
// @Summary Update model
// @Description Update an existing model's configuration
// @Tags models
// @Accept json
// @Produce json
// @Param id path string true "Model ID"
// @Param request body dto.UpdateModelRequest true "Model update request"
// @Success 200 {object} dto.ModelResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/models/{id} [put]
func (h *ModelHandler) UpdateModel(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "ModelHandler.UpdateModel")
	defer span.End()

	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Model ID is required",
		))
		return
	}

	var req dto.UpdateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("Failed to bind request", logging.Error(err))
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Invalid request body: "+err.Error(),
		))
		return
	}

	span.SetAttribute("model.id", modelID)
	h.logger.WithContext(ctx).Info("Updating model", logging.Any("model_id", modelID))

	model, err := h.modelService.UpdateModel(ctx, modelID, &req)
	if err != nil {
		h.logger.WithContext(ctx).Error("Failed to update model", logging.Any("model_id", modelID), logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.WithContext(ctx).Info("Model updated successfully", logging.Any("model_id", modelID))
	c.JSON(http.StatusOK, model)
}

// DeleteModel handles DELETE /api/v1/models/:id
// @Summary Delete model
// @Description Soft delete a model by ID
// @Tags models
// @Produce json
// @Param id path string true "Model ID"
// @Success 204 "No Content"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/models/{id} [delete]
func (h *ModelHandler) DeleteModel(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "ModelHandler.DeleteModel")
	defer span.End()

	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Model ID is required",
		))
		return
	}

	span.SetAttribute("model.id", modelID)
	h.logger.WithContext(ctx).Info("Deleting model", logging.Any("model_id", modelID))

	if err := h.modelService.DeleteModel(ctx, modelID); err != nil {
		h.logger.WithContext(ctx).Error("Failed to delete model", logging.Any("model_id", modelID), logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.WithContext(ctx).Info("Model deleted successfully", logging.Any("model_id", modelID))
	c.Status(http.StatusNoContent)
}

// DeployModel handles POST /api/v1/models/:id/deploy
// @Summary Deploy model
// @Description Deploy a model to inference infrastructure
// @Tags models
// @Accept json
// @Produce json
// @Param id path string true "Model ID"
// @Param request body dto.DeployModelRequest true "Model deployment request"
// @Success 200 {object} dto.DeploymentResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/models/{id}/deploy [post]
func (h *ModelHandler) DeployModel(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "ModelHandler.DeployModel")
	defer span.End()

	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Model ID is required",
		))
		return
	}

	var req dto.DeployModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("Failed to bind request", logging.Error(err))
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Invalid request body: "+err.Error(),
		))
		return
	}

	span.SetAttribute("model.id", modelID)
	span.SetAttribute("deployment.replicas", req.Replicas)
	h.logger.WithContext(ctx).Info("Deploying model", logging.Any("model_id", modelID), logging.Any("replicas", req.Replicas))

	result, err := h.modelService.DeployModel(ctx, modelID, &req)
	if err != nil {
		h.logger.WithContext(ctx).Error("Failed to deploy model", logging.Any("model_id", modelID), logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.WithContext(ctx).Info("Model deployed successfully", logging.Any("model_id", modelID), logging.Any("deployment_id", result.DeploymentID))
	span.SetAttribute("deployment.id", result.DeploymentID)

	c.JSON(http.StatusOK, result)
}

// UndeployModel handles POST /api/v1/models/:id/undeploy
// @Summary Undeploy model
// @Description Remove model from inference infrastructure
// @Tags models
// @Produce json
// @Param id path string true "Model ID"
// @Success 200 {object} dto.UndeploymentResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/models/{id}/undeploy [post]
func (h *ModelHandler) UndeployModel(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "ModelHandler.UndeployModel")
	defer span.End()

	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Model ID is required",
		))
		return
	}

	span.SetAttribute("model.id", modelID)
	h.logger.WithContext(ctx).Info("Undeploying model", logging.Any("model_id", modelID))

	result, err := h.modelService.UndeployModel(ctx, modelID)
	if err != nil {
		h.logger.WithContext(ctx).Error("Failed to undeploy model", logging.Any("model_id", modelID), logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.WithContext(ctx).Info("Model undeployed successfully", logging.Any("model_id", modelID))
	c.JSON(http.StatusOK, result)
}

// GetModelMetrics handles GET /api/v1/models/:id/metrics
// @Summary Get model metrics
// @Description Retrieve performance and usage metrics for a model
// @Tags models
// @Produce json
// @Param id path string true "Model ID"
// @Param start_time query string false "Start time (RFC3339)"
// @Param end_time query string false "End time (RFC3339)"
// @Param interval query string false "Metrics interval (1m, 5m, 1h)" default(5m)
// @Success 200 {object} dto.ModelMetricsResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/models/{id}/metrics [get]
func (h *ModelHandler) GetModelMetrics(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "ModelHandler.GetModelMetrics")
	defer span.End()

	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Model ID is required",
		))
		return
	}

	metricsReq := &dto.ModelMetricsRequest{
		ModelID:   modelID,
		StartTime: c.Query("start_time"),
		EndTime:   c.Query("end_time"),
		Interval:  c.DefaultQuery("interval", "5m"),
	}

	span.SetAttribute("model.id", modelID)
	span.SetAttribute("metrics.interval", metricsReq.Interval)
	h.logger.WithContext(ctx).Debug("Fetching model metrics", logging.Any("model_id", modelID), logging.Any("interval", metricsReq.Interval))

	metrics, err := h.modelService.GetModelMetrics(ctx, metricsReq)
	if err != nil {
		h.logger.WithContext(ctx).Error("Failed to get model metrics", logging.Any("model_id", modelID), logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetModelHealth handles GET /api/v1/models/:id/health
// @Summary Get model health status
// @Description Check the health status of a deployed model
// @Tags models
// @Produce json
// @Param id path string true "Model ID"
// @Success 200 {object} dto.ModelHealthResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/models/{id}/health [get]
func (h *ModelHandler) GetModelHealth(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "ModelHandler.GetModelHealth")
	defer span.End()

	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Model ID is required",
		))
		return
	}

	span.SetAttribute("model.id", modelID)
	h.logger.WithContext(ctx).Debug("Checking model health", logging.Any("model_id", modelID))

	health, err := h.modelService.GetModelHealth(ctx, modelID)
	if err != nil {
		h.logger.WithContext(ctx).Error("Failed to get model health", logging.Any("model_id", modelID), logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, health)
}

// ScaleModel handles POST /api/v1/models/:id/scale
// @Summary Scale model deployment
// @Description Scale the number of replicas for a deployed model
// @Tags models
// @Accept json
// @Produce json
// @Param id path string true "Model ID"
// @Param request body dto.ScaleModelRequest true "Model scaling request"
// @Success 200 {object} dto.ScaleResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/models/{id}/scale [post]
func (h *ModelHandler) ScaleModel(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "ModelHandler.ScaleModel")
	defer span.End()

	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Model ID is required",
		))
		return
	}

	var req dto.ScaleModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("Failed to bind request", logging.Error(err))
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Invalid request body: "+err.Error(),
		))
		return
	}

	span.SetAttribute("model.id", modelID)
	span.SetAttribute("scale.replicas", req.Replicas)
	h.logger.WithContext(ctx).Info("Scaling model", logging.Any("model_id", modelID), logging.Any("replicas", req.Replicas))

	result, err := h.modelService.ScaleModel(ctx, modelID, &req)
	if err != nil {
		h.logger.WithContext(ctx).Error("Failed to scale model", logging.Any("model_id", modelID), logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.WithContext(ctx).Info("Model scaled successfully", logging.Any("model_id", modelID), logging.Any("replicas", req.Replicas))
	c.JSON(http.StatusOK, result)
}

// InferenceModel handles POST /api/v1/models/:id/infer
// @Summary Run model inference
// @Description Execute inference using a deployed model
// @Tags models
// @Accept json
// @Produce json
// @Param id path string true "Model ID"
// @Param request body dto.InferenceRequest true "Inference request"
// @Success 200 {object} dto.InferenceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/models/{id}/infer [post]
func (h *ModelHandler) InferenceModel(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "ModelHandler.InferenceModel")
	defer span.End()

	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Model ID is required",
		))
		return
	}

	var req dto.InferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("Failed to bind request", logging.Error(err))
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Invalid request body: "+err.Error(),
		))
		return
	}

	span.SetAttribute("model.id", modelID)
	span.SetAttribute("inference.stream", req.Stream)
	h.logger.WithContext(ctx).Info("Running inference", logging.Any("model_id", modelID), logging.Any("stream", req.Stream))

	result, err := h.modelService.RunInference(ctx, modelID, &req)
	if err != nil {
		h.logger.WithContext(ctx).Error("Failed to run inference", logging.Any("model_id", modelID), logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	h.logger.WithContext(ctx).Info("Inference completed successfully", logging.Any("model_id", modelID))
	c.JSON(http.StatusOK, result)
}

// ListModelVersions handles GET /api/v1/models/:id/versions
// @Summary List model versions
// @Description Retrieve all versions of a model
// @Tags models
// @Produce json
// @Param id path string true "Model ID"
// @Success 200 {object} dto.ModelVersionsResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/models/{id}/versions [get]
func (h *ModelHandler) ListModelVersions(c *gin.Context) {
	ctx, span := h.tracer.StartSpan(c.Request.Context(), "ModelHandler.ListModelVersions")
	defer span.End()

	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, NewErrorResponse(
			errors.ErrInvalidRequest,
			"Model ID is required",
		))
		return
	}

	span.SetAttribute("model.id", modelID)
	h.logger.WithContext(ctx).Debug("Listing model versions", logging.Any("model_id", modelID))

	versions, err := h.modelService.ListModelVersions(ctx, modelID)
	if err != nil {
		h.logger.WithContext(ctx).Error("Failed to list model versions", logging.Any("model_id", modelID), logging.Error(err))
		span.RecordError(err)
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, versions)
}

//Personal.AI order the ending
