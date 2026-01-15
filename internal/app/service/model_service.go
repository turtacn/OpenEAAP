package service

import (
	"context"
	"fmt"
	"time"

	"github.com/openeeap/openeeap/internal/app/dto"
	"github.com/openeeap/openeeap/internal/domain/model"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/internal/platform/inference"
	"github.com/openeeap/openeeap/internal/platform/training"
	"github.com/openeeap/openeeap/pkg/errors"
	"github.com/openeeap/openeeap/pkg/types"
)

// ModelService 模型应用服务接口
type ModelService interface {
	// Register 注册模型
	Register(ctx context.Context, req *dto.RegisterModelRequest) (*dto.ModelResponse, error)

	// GetByID 根据 ID 获取模型
	GetByID(ctx context.Context, id string) (*dto.ModelResponse, error)

	// List 列出模型
	List(ctx context.Context, req *dto.ListModelRequest) (*dto.ModelListResponse, error)

	// Update 更新模型
	Update(ctx context.Context, id string, req *dto.UpdateModelRequest) (*dto.ModelResponse, error)

	// Delete 删除模型
	Delete(ctx context.Context, id string) error

	// Deploy 部署模型
	Deploy(ctx context.Context, id string, req *dto.DeployModelRequest) (*dto.DeploymentResponse, error)

	// Undeploy 下线模型
	Undeploy(ctx context.Context, id string) error

	// GetDeploymentStatus 获取部署状态
	GetDeploymentStatus(ctx context.Context, id string) (*dto.DeploymentStatus, error)

	// GetMetrics 获取模型指标
	GetMetrics(ctx context.Context, id string, req *dto.MetricsRequest) (*dto.ModelMetrics, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context, id string) (*dto.HealthCheckResponse, error)

	// StartTraining 启动训练任务
	StartTraining(ctx context.Context, req *dto.StartTrainingRequest) (*dto.TrainingJobResponse, error)

	// GetTrainingStatus 获取训练状态
	GetTrainingStatus(ctx context.Context, jobID string) (*dto.TrainingJobStatus, error)

	// StopTraining 停止训练任务
	StopTraining(ctx context.Context, jobID string) error

	// ListTrainingJobs 列出训练任务
	ListTrainingJobs(ctx context.Context, modelID string, req *dto.ListTrainingJobsRequest) (*dto.TrainingJobListResponse, error)
}

// modelService 模型应用服务实现
type modelService struct {
	modelRepo        model.ModelRepository
	modelDomain      model.ModelService
	inferenceGateway inference.Gateway
	trainingService  training.TrainingService
	logger           logging.Logger
	tracer           trace.Tracer
	metrics          metrics.MetricsCollector
}

// NewModelService 创建模型应用服务
func NewModelService(
	modelRepo model.ModelRepository,
	modelDomain model.ModelService,
	inferenceGateway inference.Gateway,
	trainingService training.TrainingService,
	logger logging.Logger,
	tracer trace.Tracer,
	metrics metrics.MetricsCollector,
) ModelService {
	return &modelService{
		modelRepo:        modelRepo,
		modelDomain:      modelDomain,
		inferenceGateway: inferenceGateway,
		trainingService:  trainingService,
		logger:           logger,
		tracer:           tracer,
		metrics:          metrics,
	}
}

// Register 注册模型
func (s *modelService) Register(ctx context.Context, req *dto.RegisterModelRequest) (*dto.ModelResponse, error) {
	span := s.tracer.StartSpan(ctx, "ModelService.Register")
	defer span.End()

	s.logger.InfoCtx(ctx, "Registering model", "name", req.Name, "type", req.Type)

	// 转换 DTO 为领域实体
	m := &model.Model{
		ID:           types.NewID(),
		Name:         req.Name,
		Type:         model.ModelType(req.Type),
		Version:      req.Version,
		Provider:     req.Provider,
		Endpoint:     req.Endpoint,
		Config:       s.convertModelConfig(req.Config),
		Capabilities: req.Capabilities,
		Status:       model.StatusRegistered,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 领域验证
	if err := s.modelDomain.ValidateModel(ctx, m); err != nil {
		s.logger.ErrorCtx(ctx, "Model validation failed", "error", err)
		return nil, errors.Wrap(err, errors.CodeValidationFailed, "model validation failed")
	}

	// 健康检查（可选）
	if req.PerformHealthCheck {
		if err := s.performInitialHealthCheck(ctx, m); err != nil {
			s.logger.WarnCtx(ctx, "Initial health check failed", "error", err)
			m.Status = model.StatusUnhealthy
		}
	}

	// 持久化
	if err := s.modelRepo.Create(ctx, m); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to register model", "error", err)
		s.metrics.IncrementCounter("model_registration_errors", nil)
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to register model")
	}

	s.metrics.IncrementCounter("model_registered", map[string]string{
		"name": req.Name,
		"type": req.Type,
	})
	s.logger.InfoCtx(ctx, "Model registered successfully", "id", m.ID)

	return s.toModelResponse(m), nil
}

// GetByID 根据 ID 获取模型
func (s *modelService) GetByID(ctx context.Context, id string) (*dto.ModelResponse, error) {
	span := s.tracer.StartSpan(ctx, "ModelService.GetByID")
	defer span.End()

	m, err := s.modelRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Failed to get model", "id", id, "error", err)
		return nil, errors.Wrap(err, errors.CodeNotFound, "model not found")
	}

	return s.toModelResponse(m), nil
}

// List 列出模型
func (s *modelService) List(ctx context.Context, req *dto.ListModelRequest) (*dto.ModelListResponse, error) {
	span := s.tracer.StartSpan(ctx, "ModelService.List")
	defer span.End()

	offset := (req.Page - 1) * req.PageSize
	models, total, err := s.modelRepo.List(ctx, req.Type, req.Status, req.Search, offset, req.PageSize)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Failed to list models", "error", err)
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to list models")
	}

	items := make([]*dto.ModelResponse, len(models))
	for i, m := range models {
		items[i] = s.toModelResponse(m)
	}

	return &dto.ModelListResponse{
		Items: items,
		Pagination: dto.Pagination{
			Page:     req.Page,
			PageSize: req.PageSize,
			Total:    total,
		},
	}, nil
}

// Update 更新模型
func (s *modelService) Update(ctx context.Context, id string, req *dto.UpdateModelRequest) (*dto.ModelResponse, error) {
	span := s.tracer.StartSpan(ctx, "ModelService.Update")
	defer span.End()

	s.logger.InfoCtx(ctx, "Updating model", "id", id)

	// 获取现有模型
	m, err := s.modelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "model not found")
	}

	// 检查是否可更新
	if m.Status == model.StatusDeploying {
		return nil, errors.New(errors.CodeBusinessLogicError, "cannot update deploying model")
	}

	// 更新字段
	if req.Name != nil {
		m.Name = *req.Name
	}
	if req.Endpoint != nil {
		m.Endpoint = *req.Endpoint
	}
	if req.Config != nil {
		m.Config = s.convertModelConfig(req.Config)
	}
	if req.Capabilities != nil {
		m.Capabilities = *req.Capabilities
	}
	m.UpdatedAt = time.Now()

	// 领域验证
	if err := s.modelDomain.ValidateModel(ctx, m); err != nil {
		return nil, errors.Wrap(err, errors.CodeValidationFailed, "model validation failed")
	}

	// 持久化
	if err := s.modelRepo.Update(ctx, m); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to update model", "error", err)
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to update model")
	}

	s.logger.InfoCtx(ctx, "Model updated successfully", "id", id)
	return s.toModelResponse(m), nil
}

// Delete 删除模型
func (s *modelService) Delete(ctx context.Context, id string) error {
	span := s.tracer.StartSpan(ctx, "ModelService.Delete")
	defer span.End()

	s.logger.InfoCtx(ctx, "Deleting model", "id", id)

	// 获取模型检查状态
	m, err := s.modelRepo.GetByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, errors.CodeNotFound, "model not found")
	}

	// 如果模型已部署，先下线
	if m.Status == model.StatusDeployed || m.Status == model.StatusDeploying {
		s.logger.InfoCtx(ctx, "Undeploying model before deletion", "id", id)
		if err := s.Undeploy(ctx, id); err != nil {
			return errors.Wrap(err, errors.CodeBusinessLogicError, "failed to undeploy model before deletion")
		}
	}

	if err := s.modelRepo.Delete(ctx, id); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to delete model", "error", err)
		return errors.Wrap(err, errors.CodeDatabaseError, "failed to delete model")
	}

	s.metrics.IncrementCounter("model_deleted", nil)
	s.logger.InfoCtx(ctx, "Model deleted successfully", "id", id)
	return nil
}

// Deploy 部署模型
func (s *modelService) Deploy(ctx context.Context, id string, req *dto.DeployModelRequest) (*dto.DeploymentResponse, error) {
	span := s.tracer.StartSpan(ctx, "ModelService.Deploy")
	defer span.End()

	startTime := time.Now()
	s.logger.InfoCtx(ctx, "Deploying model", "id", id)

	// 获取模型
	m, err := s.modelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "model not found")
	}

	// 检查模型状态
	if !s.modelDomain.CanDeploy(ctx, m) {
		return nil, errors.New(errors.CodeBusinessLogicError, "model cannot be deployed")
	}

	// 更新状态为部署中
	m.Status = model.StatusDeploying
	m.UpdatedAt = time.Now()
	if err := s.modelRepo.Update(ctx, m); err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to update model status")
	}

	// 调用推理网关部署模型
	deployConfig := &inference.DeployConfig{
		ModelID:      id,
		Replicas:     req.Replicas,
		GPUMemory:    req.GPUMemory,
		MaxBatchSize: req.MaxBatchSize,
		Timeout:      req.Timeout,
		Autoscaling:  req.Autoscaling,
	}

	deploymentID, err := s.inferenceGateway.DeployModel(ctx, m, deployConfig)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Model deployment failed", "id", id, "error", err)
		m.Status = model.StatusFailed
		s.modelRepo.Update(ctx, m)
		s.metrics.IncrementCounter("model_deployment_errors", map[string]string{"model_id": id})
		return nil, errors.Wrap(err, errors.CodeExecutionError, "model deployment failed")
	}

	// 更新状态为已部署
	m.Status = model.StatusDeployed
	m.DeploymentID = deploymentID
	m.UpdatedAt = time.Now()
	if err := s.modelRepo.Update(ctx, m); err != nil {
		s.logger.WarnCtx(ctx, "Failed to update model status after deployment", "error", err)
	}

	duration := time.Since(startTime)
	s.metrics.RecordHistogram("model_deployment_duration_seconds", duration.Seconds(), map[string]string{
		"model_id": id,
	})

	s.logger.InfoCtx(ctx, "Model deployed successfully",
		"id", id,
		"deployment_id", deploymentID,
		"duration_ms", duration.Milliseconds(),
	)

	return &dto.DeploymentResponse{
		DeploymentID: deploymentID,
		ModelID:      id,
		Status:       "deployed",
		Endpoint:     m.Endpoint,
		Replicas:     req.Replicas,
		DeployedAt:   time.Now(),
	}, nil
}

// Undeploy 下线模型
func (s *modelService) Undeploy(ctx context.Context, id string) error {
	span := s.tracer.StartSpan(ctx, "ModelService.Undeploy")
	defer span.End()

	s.logger.InfoCtx(ctx, "Undeploying model", "id", id)

	// 获取模型
	m, err := s.modelRepo.GetByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, errors.CodeNotFound, "model not found")
	}

	if m.Status != model.StatusDeployed {
		return errors.New(errors.CodeBusinessLogicError, "model is not deployed")
	}

	// 调用推理网关下线模型
	if err := s.inferenceGateway.UndeployModel(ctx, m.DeploymentID); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to undeploy model", "id", id, "error", err)
		return errors.Wrap(err, errors.CodeExecutionError, "failed to undeploy model")
	}

	// 更新状态
	m.Status = model.StatusRegistered
	m.DeploymentID = ""
	m.UpdatedAt = time.Now()
	if err := s.modelRepo.Update(ctx, m); err != nil {
		s.logger.WarnCtx(ctx, "Failed to update model status after undeployment", "error", err)
	}

	s.metrics.IncrementCounter("model_undeployed", nil)
	s.logger.InfoCtx(ctx, "Model undeployed successfully", "id", id)
	return nil
}

// GetDeploymentStatus 获取部署状态
func (s *modelService) GetDeploymentStatus(ctx context.Context, id string) (*dto.DeploymentStatus, error) {
	span := s.tracer.StartSpan(ctx, "ModelService.GetDeploymentStatus")
	defer span.End()

	m, err := s.modelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "model not found")
	}

	if m.Status != model.StatusDeployed {
		return &dto.DeploymentStatus{
			ModelID: id,
			Status:  string(m.Status),
		}, nil
	}

	// 从推理网关获取详细部署状态
	status, err := s.inferenceGateway.GetDeploymentStatus(ctx, m.DeploymentID)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Failed to get deployment status", "id", id, "error", err)
		return nil, errors.Wrap(err, errors.CodeExecutionError, "failed to get deployment status")
	}

	return &dto.DeploymentStatus{
		ModelID:           id,
		DeploymentID:      m.DeploymentID,
		Status:            status.Status,
		Replicas:          status.Replicas,
		AvailableReplicas: status.AvailableReplicas,
		Endpoint:          m.Endpoint,
		LastUpdated:       status.LastUpdated,
	}, nil
}

// GetMetrics 获取模型指标
func (s *modelService) GetMetrics(ctx context.Context, id string, req *dto.MetricsRequest) (*dto.ModelMetrics, error) {
	span := s.tracer.StartSpan(ctx, "ModelService.GetMetrics")
	defer span.End()

	m, err := s.modelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "model not found")
	}

	if m.Status != model.StatusDeployed {
		return nil, errors.New(errors.CodeBusinessLogicError, "model is not deployed")
	}

	// 从推理网关获取指标
	metricsData, err := s.inferenceGateway.GetModelMetrics(ctx, m.DeploymentID, req.StartTime, req.EndTime)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Failed to get model metrics", "id", id, "error", err)
		return nil, errors.Wrap(err, errors.CodeExecutionError, "failed to get model metrics")
	}

	return &dto.ModelMetrics{
		ModelID:         id,
		RequestCount:    metricsData.RequestCount,
		ErrorCount:      metricsData.ErrorCount,
		AvgLatency:      metricsData.AvgLatency,
		P50Latency:      metricsData.P50Latency,
		P95Latency:      metricsData.P95Latency,
		P99Latency:      metricsData.P99Latency,
		TokensProcessed: metricsData.TokensProcessed,
		CacheHitRate:    metricsData.CacheHitRate,
		GPUUtilization:  metricsData.GPUUtilization,
		ThroughputTPS:   metricsData.ThroughputTPS,
		StartTime:       req.StartTime,
		EndTime:         req.EndTime,
	}, nil
}

// HealthCheck 健康检查
func (s *modelService) HealthCheck(ctx context.Context, id string) (*dto.HealthCheckResponse, error) {
	span := s.tracer.StartSpan(ctx, "ModelService.HealthCheck")
	defer span.End()

	m, err := s.modelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "model not found")
	}

	startTime := time.Now()
	healthy, err := s.inferenceGateway.HealthCheck(ctx, m.Endpoint)
	latency := time.Since(startTime)

	status := "healthy"
	message := "Model is responding normally"

	if err != nil {
		status = "unhealthy"
		message = fmt.Sprintf("Health check failed: %v", err)
		s.logger.WarnCtx(ctx, "Model health check failed", "id", id, "error", err)

		// 更新模型状态
		m.Status = model.StatusUnhealthy
		s.modelRepo.Update(ctx, m)
	} else if !healthy {
		status = "degraded"
		message = "Model is responding but with degraded performance"
	}

	return &dto.HealthCheckResponse{
		ModelID:   id,
		Status:    status,
		Message:   message,
		Latency:   latency.Milliseconds(),
		Timestamp: time.Now(),
	}, nil
}

// StartTraining 启动训练任务
func (s *modelService) StartTraining(ctx context.Context, req *dto.StartTrainingRequest) (*dto.TrainingJobResponse, error) {
	span := s.tracer.StartSpan(ctx, "ModelService.StartTraining")
	defer span.End()

	s.logger.InfoCtx(ctx, "Starting training job", "model_id", req.ModelID, "type", req.TrainingType)

	// 获取基础模型
	m, err := s.modelRepo.GetByID(ctx, req.ModelID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "model not found")
	}

	// 创建训练任务配置
	trainingConfig := &training.TrainingConfig{
		ModelID:      req.ModelID,
		TrainingType: training.TrainingType(req.TrainingType),
		DatasetPath:  req.DatasetPath,
		Hyperparams:  req.Hyperparameters,
		Resources: training.ResourceConfig{
			GPUCount:  req.GPUCount,
			GPUMemory: req.GPUMemory,
			CPUCores:  req.CPUCores,
			Memory:    req.Memory,
		},
		OutputPath:   req.OutputPath,
		MaxEpochs:    req.MaxEpochs,
		BatchSize:    req.BatchSize,
		LearningRate: req.LearningRate,
	}

	// 调用训练服务启动任务
	jobID, err := s.trainingService.StartTraining(ctx, m, trainingConfig)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Failed to start training job", "error", err)
		s.metrics.IncrementCounter("training_start_errors", map[string]string{"model_id": req.ModelID})
		return nil, errors.Wrap(err, errors.CodeExecutionError, "failed to start training job")
	}

	s.metrics.IncrementCounter("training_jobs_started", map[string]string{
		"model_id": req.ModelID,
		"type":     req.TrainingType,
	})

	s.logger.InfoCtx(ctx, "Training job started successfully", "job_id", jobID)

	return &dto.TrainingJobResponse{
		JobID:        jobID,
		ModelID:      req.ModelID,
		TrainingType: req.TrainingType,
		Status:       "pending",
		CreatedAt:    time.Now(),
	}, nil
}

// GetTrainingStatus 获取训练状态
func (s *modelService) GetTrainingStatus(ctx context.Context, jobID string) (*dto.TrainingJobStatus, error) {
	span := s.tracer.StartSpan(ctx, "ModelService.GetTrainingStatus")
	defer span.End()

	status, err := s.trainingService.GetTrainingStatus(ctx, jobID)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Failed to get training status", "job_id", jobID, "error", err)
		return nil, errors.Wrap(err, errors.CodeNotFound, "training job not found")
	}

	return &dto.TrainingJobStatus{
		JobID:        status.JobID,
		ModelID:      status.ModelID,
		Status:       status.Status,
		Progress:     status.Progress,
		CurrentEpoch: status.CurrentEpoch,
		TotalEpochs:  status.TotalEpochs,
		Loss:         status.Loss,
		Metrics:      status.Metrics,
		StartedAt:    status.StartedAt,
		UpdatedAt:    status.UpdatedAt,
		CompletedAt:  status.CompletedAt,
		Error:        status.Error,
	}, nil
}

// StopTraining 停止训练任务
func (s *modelService) StopTraining(ctx context.Context, jobID string) error {
	span := s.tracer.StartSpan(ctx, "ModelService.StopTraining")
	defer span.End()

	s.logger.InfoCtx(ctx, "Stopping training job", "job_id", jobID)

	if err := s.trainingService.StopTraining(ctx, jobID); err != nil {
		s.logger.ErrorCtx(ctx, "Failed to stop training job", "job_id", jobID, "error", err)
		return errors.Wrap(err, errors.CodeExecutionError, "failed to stop training job")
	}

	s.metrics.IncrementCounter("training_jobs_stopped", nil)
	s.logger.InfoCtx(ctx, "Training job stopped successfully", "job_id", jobID)
	return nil
}

// ListTrainingJobs 列出训练任务
func (s *modelService) ListTrainingJobs(ctx context.Context, modelID string, req *dto.ListTrainingJobsRequest) (*dto.TrainingJobListResponse, error) {
	span := s.tracer.StartSpan(ctx, "ModelService.ListTrainingJobs")
	defer span.End()

	offset := (req.Page - 1) * req.PageSize
	jobs, total, err := s.trainingService.ListTrainingJobs(ctx, modelID, offset, req.PageSize)
	if err != nil {
		s.logger.ErrorCtx(ctx, "Failed to list training jobs", "model_id", modelID, "error", err)
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to list training jobs")
	}

	items := make([]*dto.TrainingJobStatus, len(jobs))
	for i, job := range jobs {
		items[i] = &dto.TrainingJobStatus{
			JobID:        job.JobID,
			ModelID:      job.ModelID,
			Status:       job.Status,
			Progress:     job.Progress,
			CurrentEpoch: job.CurrentEpoch,
			TotalEpochs:  job.TotalEpochs,
			StartedAt:    job.StartedAt,
			UpdatedAt:    job.UpdatedAt,
			CompletedAt:  job.CompletedAt,
		}
	}

	return &dto.TrainingJobListResponse{
		Items: items,
		Pagination: dto.Pagination{
			Page:     req.Page,
			PageSize: req.PageSize,
			Total:    total,
		},
	}, nil
}

// 辅助方法：执行初始健康检查
func (s *modelService) performInitialHealthCheck(ctx context.Context, m *model.Model) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	healthy, err := s.inferenceGateway.HealthCheck(ctx, m.Endpoint)
	if err != nil {
		return err
	}
	if !healthy {
		return fmt.Errorf("model health check returned unhealthy status")
	}
	return nil
}

// 辅助方法：转换模型配置
func (s *modelService) convertModelConfig(dtoConfig *dto.ModelConfig) *model.Config {
	if dtoConfig == nil {
		return &model.Config{}
	}
	return &model.Config{
		MaxTokens:        dtoConfig.MaxTokens,
		Temperature:      dtoConfig.Temperature,
		TopP:             dtoConfig.TopP,
		FrequencyPenalty: dtoConfig.FrequencyPenalty,
		PresencePenalty:  dtoConfig.PresencePenalty,
		StopSequences:    dtoConfig.StopSequences,
		Timeout:          dtoConfig.Timeout,
		RetryPolicy:      s.convertRetryPolicy(dtoConfig.RetryPolicy),
	}
}

// 辅助方法：转换重试策略
func (s *modelService) convertRetryPolicy(dtoPolicy *dto.RetryPolicy) *model.RetryPolicy {
	if dtoPolicy == nil {
		return nil
	}
	return &model.RetryPolicy{
		MaxRetries:   dtoPolicy.MaxRetries,
		BackoffType:  model.BackoffType(dtoPolicy.BackoffType),
		InitialDelay: dtoPolicy.InitialDelay,
		MaxDelay:     dtoPolicy.MaxDelay,
	}
}

// 辅助方法：转换为响应 DTO
func (s *modelService) toModelResponse(m *model.Model) *dto.ModelResponse {
	return &dto.ModelResponse{
		ID:           m.ID,
		Name:         m.Name,
		Type:         string(m.Type),
		Version:      m.Version,
		Provider:     m.Provider,
		Endpoint:     m.Endpoint,
		Config:       s.convertConfigToDTO(m.Config),
		Capabilities: m.Capabilities,
		Status:       string(m.Status),
		DeploymentID: m.DeploymentID,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// 辅助方法：转换配置为 DTO
func (s *modelService) convertConfigToDTO(config *model.Config) *dto.ModelConfig {
	if config == nil {
		return nil
	}
	return &dto.ModelConfig{
		MaxTokens:        config.MaxTokens,
		Temperature:      config.Temperature,
		TopP:             config.TopP,
		FrequencyPenalty: config.FrequencyPenalty,
		PresencePenalty:  config.PresencePenalty,
		StopSequences:    config.StopSequences,
		Timeout:          config.Timeout,
		RetryPolicy:      s.convertRetryPolicyToDTO(config.RetryPolicy),
	}
}

// 辅助方法：转换重试策略为 DTO
func (s *modelService) convertRetryPolicyToDTO(policy *model.RetryPolicy) *dto.RetryPolicy {
	if policy == nil {
		return nil
	}
	return &dto.RetryPolicy{
		MaxRetries:   policy.MaxRetries,
		BackoffType:  string(policy.BackoffType),
		InitialDelay: policy.InitialDelay,
		MaxDelay:     policy.MaxDelay,
	}
}
