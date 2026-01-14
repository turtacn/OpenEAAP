package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openeeap/openeeap/internal/domain/model"
	"github.com/openeeap/openeeap/pkg/errors"
	"github.com/openeeap/openeeap/pkg/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ModelModel Model 数据库模型
type ModelModel struct {
	ID          string         `gorm:"primaryKey;type:varchar(64)"`
	Name        string         `gorm:"type:varchar(255);not null;index"`
	Type        string         `gorm:"type:varchar(50);not null;index"`
	Provider    string         `gorm:"type:varchar(100);not null;index"`
	Version     string         `gorm:"type:varchar(50);not null"`
	Endpoint    string         `gorm:"type:varchar(500)"`
	Config      string         `gorm:"type:jsonb;not null"` // JSON 格式存储配置
	Capabilities string        `gorm:"type:jsonb"`          // JSON 格式存储能力
	Limits      string         `gorm:"type:jsonb"`          // JSON 格式存储限制
	Pricing     string         `gorm:"type:jsonb"`          // JSON 格式存储定价
	Status      string         `gorm:"type:varchar(50);not null;index;default:'inactive'"`
	IsDefault   bool           `gorm:"default:false;index"`
	Priority    int            `gorm:"default:0;index"`
	Tags        string         `gorm:"type:jsonb"` // JSON 数组存储标签
	Metadata    string         `gorm:"type:jsonb"` // JSON 格式存储元数据
	CreatedBy   string         `gorm:"type:varchar(64);not null;index"`
	UpdatedBy   string         `gorm:"type:varchar(64)"`
	CreatedAt   time.Time      `gorm:"not null;index"`
	UpdatedAt   time.Time      `gorm:"not null"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名
func (ModelModel) TableName() string {
	return "models"
}

// ModelVersionModel 模型版本数据库模型
type ModelVersionModel struct {
	ID              string    `gorm:"primaryKey;type:varchar(64)"`
	ModelID         string    `gorm:"type:varchar(64);not null;index"`
	Version         string    `gorm:"type:varchar(50);not null;index"`
	Changelog       string    `gorm:"type:text"`
	Config          string    `gorm:"type:jsonb"`
	PerformanceMetrics string `gorm:"type:jsonb"` // 性能指标
	IsActive        bool      `gorm:"default:false;index"`
	ReleasedAt      time.Time `gorm:"index"`
	CreatedAt       time.Time `gorm:"not null"`
	UpdatedAt       time.Time `gorm:"not null"`
}

// TableName 指定表名
func (ModelVersionModel) TableName() string {
	return "model_versions"
}

// ModelMetricsModel 模型指标数据库模型
type ModelMetricsModel struct {
	ID              string    `gorm:"primaryKey;type:varchar(64)"`
	ModelID         string    `gorm:"type:varchar(64);not null;index"`
	MetricType      string    `gorm:"type:varchar(50);not null;index"`
	Value           float64   `gorm:"not null"`
	Unit            string    `gorm:"type:varchar(20)"`
	Timestamp       time.Time `gorm:"not null;index"`
	Tags            string    `gorm:"type:jsonb"`
	CreatedAt       time.Time `gorm:"not null"`
}

// TableName 指定表名
func (ModelMetricsModel) TableName() string {
	return "model_metrics"
}

// modelRepo Model PostgreSQL 仓储实现
type modelRepo struct {
	db *gorm.DB
}

// NewModelRepository 创建 Model 仓储
func NewModelRepository(db *gorm.DB) (model.ModelRepository, error) {
	if db == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "database connection cannot be nil")
	}

	// 自动迁移表结构
	if err := db.AutoMigrate(
		&ModelModel{},
		&ModelVersionModel{},
		&ModelMetricsModel{},
	); err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to migrate model tables")
	}

	return &modelRepo{db: db}, nil
}

// Create 创建 Model
func (r *modelRepo) Create(ctx context.Context, mdl *model.Model) error {
	if mdl == nil {
		return errors.New(errors.CodeInvalidParameter, "model cannot be nil")
	}

	// 验证 Model
	if err := mdl.Validate(); err != nil {
		return errors.Wrap(err, errors.CodeInvalidParameter, "invalid model")
	}

	// 转换为数据库模型
	dbModel, err := r.toModel(mdl)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to convert model to db model")
	}

	// 执行创建
	if err := r.db.WithContext(ctx).Create(dbModel).Error; err != nil {
		if isDuplicateKeyError(err) {
			return errors.Wrap(err, errors.CodeAlreadyExists, "model already exists")
		}
		return errors.Wrap(err, errors.CodeDatabaseError, "failed to create model")
	}

	// 更新实体的时间戳
	mdl.CreatedAt = dbModel.CreatedAt
	mdl.UpdatedAt = dbModel.UpdatedAt

	return nil
}

// GetByID 根据 ID 获取 Model
func (r *modelRepo) GetByID(ctx context.Context, id string) (*model.Model, error) {
	if id == "" {
		return nil, errors.New(errors.CodeInvalidParameter, "model ID cannot be empty")
	}

	var dbModel ModelModel
	if err := r.db.WithContext(ctx).First(&dbModel, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "model not found")
		}
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get model")
	}

	return r.toEntity(&dbModel)
}

// GetByName 根据名称获取 Model
func (r *modelRepo) GetByName(ctx context.Context, name string) (*model.Model, error) {
	if name == "" {
		return nil, errors.New(errors.CodeInvalidParameter, "model name cannot be empty")
	}

	var dbModel ModelModel
	if err := r.db.WithContext(ctx).First(&dbModel, "name = ?", name).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "model not found")
		}
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get model by name")
	}

	return r.toEntity(&dbModel)
}

// Update 更新 Model
func (r *modelRepo) Update(ctx context.Context, mdl *model.Model) error {
	if mdl == nil {
		return errors.New(errors.CodeInvalidParameter, "model cannot be nil")
	}
	if mdl.ID == "" {
		return errors.New(errors.CodeInvalidParameter, "model ID cannot be empty")
	}

	// 验证 Model
	if err := mdl.Validate(); err != nil {
		return errors.Wrap(err, errors.CodeInvalidParameter, "invalid model")
	}

	// 转换为数据库模型
	dbModel, err := r.toModel(mdl)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to convert model to db model")
	}

	// 更新
	result := r.db.WithContext(ctx).
		Model(&ModelModel{}).
		Where("id = ?", dbModel.ID).
		Updates(map[string]interface{}{
			"name":         dbModel.Name,
			"type":         dbModel.Type,
			"provider":     dbModel.Provider,
			"version":      dbModel.Version,
			"endpoint":     dbModel.Endpoint,
			"config":       dbModel.Config,
			"capabilities": dbModel.Capabilities,
			"limits":       dbModel.Limits,
			"pricing":      dbModel.Pricing,
			"status":       dbModel.Status,
			"is_default":   dbModel.IsDefault,
			"priority":     dbModel.Priority,
			"tags":         dbModel.Tags,
			"metadata":     dbModel.Metadata,
			"updated_by":   dbModel.UpdatedBy,
			"updated_at":   time.Now(),
		})

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.CodeDatabaseError, "failed to update model")
	}

	if result.RowsAffected == 0 {
		return errors.New(errors.CodeNotFound, "model not found")
	}

	// 更新实体的时间戳
	mdl.UpdatedAt = time.Now()

	return nil
}

// Delete 删除 Model（软删除）
func (r *modelRepo) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New(errors.CodeInvalidParameter, "model ID cannot be empty")
	}

	result := r.db.WithContext(ctx).Delete(&ModelModel{}, "id = ?", id)
	if result.Error != nil {
		return errors.Wrap(result.Error, errors.CodeDatabaseError, "failed to delete model")
	}

	if result.RowsAffected == 0 {
		return errors.New(errors.CodeNotFound, "model not found")
	}

	return nil
}

// List 列出 Model（分页）
func (r *modelRepo) List(ctx context.Context, filter *model.ModelFilter) ([]*model.Model, int64, error) {
	if filter == nil {
		filter = &model.ModelFilter{}
	}

	// 应用默认值
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	// 构建查询
	query := r.db.WithContext(ctx).Model(&ModelModel{})

	// 应用过滤条件
	if filter.Name != "" {
		query = query.Where("name ILIKE ?", "%"+filter.Name+"%")
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Provider != "" {
		query = query.Where("provider = ?", filter.Provider)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.IsDefault != nil {
		query = query.Where("is_default = ?", *filter.IsDefault)
	}
	if len(filter.Tags) > 0 {
		tagsJSON, _ := json.Marshal(filter.Tags)
		query = query.Where("tags @> ?", string(tagsJSON))
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.Wrap(err, errors.CodeDatabaseError, "failed to count models")
	}

	// 应用排序
	orderBy := "priority DESC, created_at DESC"
	if filter.OrderBy != "" {
		orderBy = filter.OrderBy
		if filter.OrderDesc {
			orderBy += " DESC"
		} else {
			orderBy += " ASC"
		}
	}
	query = query.Order(orderBy)

	// 应用分页
	offset := (filter.Page - 1) * filter.PageSize
	query = query.Offset(offset).Limit(filter.PageSize)

	// 查询数据
	var dbModels []ModelModel
	if err := query.Find(&dbModels).Error; err != nil {
		return nil, 0, errors.Wrap(err, errors.CodeDatabaseError, "failed to list models")
	}

	// 转换为实体
	models := make([]*model.Model, 0, len(dbModels))
	for i := range dbModels {
		mdl, err := r.toEntity(&dbModels[i])
		if err != nil {
			return nil, 0, errors.Wrap(err, errors.CodeInternalError, "failed to convert db model to entity")
		}
		models = append(models, mdl)
	}

	return models, total, nil
}

// UpdateStatus 更新 Model 状态
func (r *modelRepo) UpdateStatus(ctx context.Context, id string, status model.ModelStatus) error {
	if id == "" {
		return errors.New(errors.CodeInvalidParameter, "model ID cannot be empty")
	}
	if !status.Valid() {
		return errors.New(errors.CodeInvalidParameter, "invalid model status")
	}

	result := r.db.WithContext(ctx).
		Model(&ModelModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status.String(),
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.CodeDatabaseError, "failed to update model status")
	}

	if result.RowsAffected == 0 {
		return errors.New(errors.CodeNotFound, "model not found")
	}

	return nil
}

// SetDefault 设置默认模型
func (r *modelRepo) SetDefault(ctx context.Context, id string, modelType model.ModelType) error {
	if id == "" {
		return errors.New(errors.CodeInvalidParameter, "model ID cannot be empty")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 清除同类型的其他默认模型
		if err := tx.Model(&ModelModel{}).
			Where("type = ? AND is_default = ?", modelType.String(), true).
			Update("is_default", false).Error; err != nil {
			return errors.Wrap(err, errors.CodeDatabaseError, "failed to clear default models")
		}

		// 设置新的默认模型
		result := tx.Model(&ModelModel{}).
			Where("id = ?", id).
			Update("is_default", true)

		if result.Error != nil {
			return errors.Wrap(result.Error, errors.CodeDatabaseError, "failed to set default model")
		}

		if result.RowsAffected == 0 {
			return errors.New(errors.CodeNotFound, "model not found")
		}

		return nil
	})
}

// GetDefault 获取默认模型
func (r *modelRepo) GetDefault(ctx context.Context, modelType model.ModelType) (*model.Model, error) {
	var dbModel ModelModel
	if err := r.db.WithContext(ctx).
		Where("type = ? AND is_default = ?", modelType.String(), true).
		First(&dbModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "default model not found")
		}
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get default model")
	}

	return r.toEntity(&dbModel)
}

// CreateVersion 创建模型版本
func (r *modelRepo) CreateVersion(ctx context.Context, version *model.ModelVersion) error {
	if version == nil {
		return errors.New(errors.CodeInvalidParameter, "model version cannot be nil")
	}

	versionModel, err := r.versionToModel(version)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to convert version to model")
	}

	if err := r.db.WithContext(ctx).Create(versionModel).Error; err != nil {
		if isDuplicateKeyError(err) {
			return errors.Wrap(err, errors.CodeAlreadyExists, "model version already exists")
		}
		return errors.Wrap(err, errors.CodeDatabaseError, "failed to create model version")
	}

	version.CreatedAt = versionModel.CreatedAt
	version.UpdatedAt = versionModel.UpdatedAt

	return nil
}

// GetVersionsByModelID 获取模型的所有版本
func (r *modelRepo) GetVersionsByModelID(ctx context.Context, modelID string) ([]*model.ModelVersion, error) {
	if modelID == "" {
		return nil, errors.New(errors.CodeInvalidParameter, "model ID cannot be empty")
	}

	var versionModels []ModelVersionModel
	if err := r.db.WithContext(ctx).
		Where("model_id = ?", modelID).
		Order("released_at DESC").
		Find(&versionModels).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get model versions")
	}

	versions := make([]*model.ModelVersion, 0, len(versionModels))
	for i := range versionModels {
		version, err := r.versionToEntity(&versionModels[i])
		if err != nil {
			return nil, errors.Wrap(err, errors.CodeInternalError, "failed to convert version to entity")
		}
		versions = append(versions, version)
	}

	return versions, nil
}

// RecordMetrics 记录模型指标
func (r *modelRepo) RecordMetrics(ctx context.Context, metrics *model.ModelMetrics) error {
	if metrics == nil {
		return errors.New(errors.CodeInvalidParameter, "metrics cannot be nil")
	}

	metricsModel := r.metricsToModel(metrics)

	if err := r.db.WithContext(ctx).Create(metricsModel).Error; err != nil {
		return errors.Wrap(err, errors.CodeDatabaseError, "failed to record metrics")
	}

	return nil
}

// GetMetrics 获取模型指标
func (r *modelRepo) GetMetrics(ctx context.Context, modelID string, metricType string, start, end time.Time) ([]*model.ModelMetrics, error) {
	if modelID == "" {
		return nil, errors.New(errors.CodeInvalidParameter, "model ID cannot be empty")
	}

	query := r.db.WithContext(ctx).
		Where("model_id = ?", modelID).
		Where("timestamp >= ? AND timestamp <= ?", start, end)

	if metricType != "" {
		query = query.Where("metric_type = ?", metricType)
	}

	var metricsModels []ModelMetricsModel
	if err := query.Order("timestamp DESC").Find(&metricsModels).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get metrics")
	}

	metrics := make([]*model.ModelMetrics, 0, len(metricsModels))
	for i := range metricsModels {
		metric := r.metricsToEntity(&metricsModels[i])
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// GetStatistics 获取 Model 统计信息
func (r *modelRepo) GetStatistics(ctx context.Context) (*model.ModelStatistics, error) {
	var stats model.ModelStatistics

	// 总数
	if err := r.db.WithContext(ctx).
		Model(&ModelModel{}).
		Count(&stats.Total).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get total count")
	}

	// 按类型统计
	type TypeCount struct {
		Type  string
		Count int64
	}
	var typeCounts []TypeCount
	if err := r.db.WithContext(ctx).
		Model(&ModelModel{}).
		Select("type, COUNT(*) as count").
		Group("type").
		Find(&typeCounts).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get type counts")
	}

	stats.ByType = make(map[string]int64)
	for _, tc := range typeCounts {
		stats.ByType[tc.Type] = tc.Count
	}

	// 按状态统计
	type StatusCount struct {
		Status string
		Count  int64
	}
	var statusCounts []StatusCount
	if err := r.db.WithContext(ctx).
		Model(&ModelModel{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&statusCounts).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get status counts")
	}

	stats.ByStatus = make(map[string]int64)
	for _, sc := range statusCounts {
		stats.ByStatus[sc.Status] = sc.Count
	}

	// 按提供商统计
	type ProviderCount struct {
		Provider string
		Count    int64
	}
	var providerCounts []ProviderCount
	if err := r.db.WithContext(ctx).
		Model(&ModelModel{}).
		Select("provider, COUNT(*) as count").
		Group("provider").
		Find(&providerCounts).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get provider counts")
	}

	stats.ByProvider = make(map[string]int64)
	for _, pc := range providerCounts {
		stats.ByProvider[pc.Provider] = pc.Count
	}

	return &stats, nil
}

// toModel 将 Model 实体转换为数据库模型
func (r *modelRepo) toModel(mdl *model.Model) (*ModelModel, error) {
	configJSON, err := json.Marshal(mdl.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	capabilitiesJSON, err := json.Marshal(mdl.Capabilities)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal capabilities: %w", err)
	}

	limitsJSON, err := json.Marshal(mdl.Limits)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal limits: %w", err)
	}

	pricingJSON, err := json.Marshal(mdl.Pricing)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pricing: %w", err)
	}

	tagsJSON, err := json.Marshal(mdl.Tags)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tags: %w", err)
	}

	metadataJSON, err := json.Marshal(mdl.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return &ModelModel{
		ID:           mdl.ID,
		Name:         mdl.Name,
		Type:         mdl.Type.String(),
		Provider:     mdl.Provider,
		Version:      mdl.Version,
		Endpoint:     mdl.Endpoint,
		Config:       string(configJSON),
		Capabilities: string(capabilitiesJSON),
		Limits:       string(limitsJSON),
		Pricing:      string(pricingJSON),
		Status:       mdl.Status.String(),
		IsDefault:    mdl.IsDefault,
		Priority:     mdl.Priority,
		Tags:         string(tagsJSON),
		Metadata:     string(metadataJSON),
		CreatedBy:    mdl.CreatedBy,
		UpdatedBy:    mdl.UpdatedBy,
		CreatedAt:    mdl.CreatedAt,
		UpdatedAt:    mdl.UpdatedAt,
	}, nil
}

// toEntity 将数据库模型转换为 Model 实体
func (r *modelRepo) toEntity(dbModel *ModelModel) (*model.Model, error) {
	var config model.ModelConfig
	if err := json.Unmarshal([]byte(dbModel.Config), &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	var capabilities model.ModelCapabilities
	if err := json.Unmarshal([]byte(dbModel.Capabilities), &capabilities); err != nil {
		return nil, fmt.Errorf("failed to unmarshal capabilities: %w", err)
	}

	var limits model.ModelLimits
	if err := json.Unmarshal([]byte(dbModel.Limits), &limits); err != nil {
		return nil, fmt.Errorf("failed to unmarshal limits: %w", err)
	}

	var pricing model.ModelPricing
	if err := json.Unmarshal([]byte(dbModel.Pricing), &pricing); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pricing: %w", err)
	}

	var tags []string
	if err := json.Unmarshal([]byte(dbModel.Tags), &tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(dbModel.Metadata), &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	modelType, err := model.ModelTypeFromString(dbModel.Type)
	if err != nil {
		return nil, fmt.Errorf("invalid model type: %w", err)
	}

	status, err := model.ModelStatusFromString(dbModel.Status)
	if err != nil {
		return nil, fmt.Errorf("invalid model status: %w", err)
	}

	return &model.Model{
		ID:           dbModel.ID,
		Name:         dbModel.Name,
		Type:         modelType,
		Provider:     dbModel.Provider,
		Version:      dbModel.Version,
		Endpoint:     dbModel.Endpoint,
		Config:       config,
		Capabilities: capabilities,
		Limits:       limits,
		Pricing:      pricing,
		Status:       status,
		IsDefault:    dbModel.IsDefault,
		Priority:     dbModel.Priority,
		Tags:         tags,
		Metadata:     metadata,
		CreatedBy:    dbModel.CreatedBy,
		UpdatedBy:    dbModel.UpdatedBy,
		CreatedAt:    dbModel.CreatedAt,
		UpdatedAt:    dbModel.UpdatedAt,
	}, nil
}

// versionToModel 将版本实体转换为数据库模型
func (r *modelRepo) versionToModel(version *model.ModelVersion) (*ModelVersionModel, error) {
	configJSON, _ := json.Marshal(version.Config)
	metricsJSON, _ := json.Marshal(version.PerformanceMetrics)

	return &ModelVersionModel{
		ID:                 version.ID,
		ModelID:            version.ModelID,
		Version:            version.Version,
		Changelog:          version.Changelog,
		Config:             string(configJSON),
		PerformanceMetrics: string(metricsJSON),
		IsActive:           version.IsActive,
		ReleasedAt:         version.ReleasedAt,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}, nil
}

// versionToEntity 将数据库模型转换为版本实体
func (r *modelRepo) versionToEntity(versionModel *ModelVersionModel) (*model.ModelVersion, error) {
	var config model.ModelConfig
	json.Unmarshal([]byte(versionModel.Config), &config)

	var metrics map[string]interface{}
	json.Unmarshal([]byte(versionModel.PerformanceMetrics), &metrics)

	return &model.ModelVersion{
		ID:                 versionModel.ID,
		ModelID:            versionModel.ModelID,
		Version:            versionModel.Version,
		Changelog:          versionModel.Changelog,
		Config:             config,
		PerformanceMetrics: metrics,
		IsActive:           versionModel.IsActive,
		ReleasedAt:         versionModel.ReleasedAt,
		CreatedAt:          versionModel.CreatedAt,
		UpdatedAt:          versionModel.UpdatedAt,
	}, nil
}

// metricsToModel 将指标实体转换为数据库模型
func (r *modelRepo) metricsToModel(metrics *model.ModelMetrics) *ModelMetricsModel {
	tagsJSON, _ := json.Marshal(metrics.Tags)

	return &ModelMetricsModel{
		ID:         metrics.ID,
		ModelID:    metrics.ModelID,
		MetricType: metrics.MetricType,
		Value:      metrics.Value,
		Unit:       metrics.Unit,
		Timestamp:  metrics.Timestamp,
		Tags:       string(tagsJSON),
		CreatedAt:  time.Now(),
	}
}

// metricsToEntity 将数据库模型转换为指标实体
func (r *modelRepo) metricsToEntity(metricsModel *ModelMetricsModel) *model.ModelMetrics {
	var tags map[string]string
	json.Unmarshal([]byte(metricsModel.Tags), &tags)

	return &model.ModelMetrics{
		ID:         metricsModel.ID,
		ModelID:    metricsModel.ModelID,
		MetricType: metricsModel.MetricType,
		Value:      metricsModel.Value,
		Unit:       metricsModel.Unit,
		Timestamp:  metricsModel.Timestamp,
		Tags:       tags,
	}
}

//Personal.AI order the ending
