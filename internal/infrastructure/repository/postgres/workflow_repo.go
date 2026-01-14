package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openeeap/openeeap/internal/domain/workflow"
	"github.com/openeeap/openeeap/pkg/errors"
	"github.com/openeeap/openeeap/pkg/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WorkflowModel Workflow 数据库模型
type WorkflowModel struct {
	ID          string         `gorm:"primaryKey;type:varchar(64)"`
	Name        string         `gorm:"type:varchar(255);not null;index"`
	Description string         `gorm:"type:text"`
	Trigger     string         `gorm:"type:jsonb"` // JSON 格式存储触发器配置
	Config      string         `gorm:"type:jsonb"` // JSON 格式存储全局配置
	Status      string         `gorm:"type:varchar(50);not null;index;default:'draft'"`
	Version     int            `gorm:"not null;default:1"`
	CreatedBy   string         `gorm:"type:varchar(64);not null;index"`
	UpdatedBy   string         `gorm:"type:varchar(64)"`
	CreatedAt   time.Time      `gorm:"not null;index"`
	UpdatedAt   time.Time      `gorm:"not null"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Steps       []WorkflowStepModel `gorm:"foreignKey:WorkflowID;constraint:OnDelete:CASCADE"`
}

// TableName 指定表名
func (WorkflowModel) TableName() string {
	return "workflows"
}

// WorkflowStepModel Workflow 步骤数据库模型
type WorkflowStepModel struct {
	ID           string    `gorm:"primaryKey;type:varchar(64)"`
	WorkflowID   string    `gorm:"type:varchar(64);not null;index"`
	Name         string    `gorm:"type:varchar(255);not null"`
	Type         string    `gorm:"type:varchar(50);not null;index"`
	AgentID      string    `gorm:"type:varchar(64);index"`
	Config       string    `gorm:"type:jsonb"`
	Input        string    `gorm:"type:jsonb"`
	Output       string    `gorm:"type:jsonb"`
	Dependencies string    `gorm:"type:jsonb"` // JSON 数组，存储依赖的步骤 ID
	Conditions   string    `gorm:"type:jsonb"` // JSON 格式存储执行条件
	RetryPolicy  string    `gorm:"type:jsonb"` // JSON 格式存储重试策略
	Timeout      int64     `gorm:"default:300"` // 超时时间（秒）
	Order        int       `gorm:"not null;index"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
}

// TableName 指定表名
func (WorkflowStepModel) TableName() string {
	return "workflow_steps"
}

// WorkflowExecutionModel Workflow 执行记录模型
type WorkflowExecutionModel struct {
	ID          string    `gorm:"primaryKey;type:varchar(64)"`
	WorkflowID  string    `gorm:"type:varchar(64);not null;index"`
	Status      string    `gorm:"type:varchar(50);not null;index"`
	Input       string    `gorm:"type:jsonb"`
	Output      string    `gorm:"type:jsonb"`
	Error       string    `gorm:"type:text"`
	StartedAt   time.Time `gorm:"index"`
	CompletedAt *time.Time
	Duration    int64     // 执行时长（毫秒）
	TriggeredBy string    `gorm:"type:varchar(64);index"`
	Context     string    `gorm:"type:jsonb"` // 执行上下文
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
	Steps       []WorkflowStepExecutionModel `gorm:"foreignKey:ExecutionID;constraint:OnDelete:CASCADE"`
}

// TableName 指定表名
func (WorkflowExecutionModel) TableName() string {
	return "workflow_executions"
}

// WorkflowStepExecutionModel Workflow 步骤执行记录模型
type WorkflowStepExecutionModel struct {
	ID          string    `gorm:"primaryKey;type:varchar(64)"`
	ExecutionID string    `gorm:"type:varchar(64);not null;index"`
	StepID      string    `gorm:"type:varchar(64);not null;index"`
	StepName    string    `gorm:"type:varchar(255);not null"`
	Status      string    `gorm:"type:varchar(50);not null;index"`
	Input       string    `gorm:"type:jsonb"`
	Output      string    `gorm:"type:jsonb"`
	Error       string    `gorm:"type:text"`
	RetryCount  int       `gorm:"default:0"`
	StartedAt   time.Time `gorm:"index"`
	CompletedAt *time.Time
	Duration    int64     // 执行时长（毫秒）
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

// TableName 指定表名
func (WorkflowStepExecutionModel) TableName() string {
	return "workflow_step_executions"
}

// workflowRepo Workflow PostgreSQL 仓储实现
type workflowRepo struct {
	db *gorm.DB
}

// NewWorkflowRepository 创建 Workflow 仓储
func NewWorkflowRepository(db *gorm.DB) (workflow.WorkflowRepository, error) {
	if db == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "database connection cannot be nil")
	}

	// 自动迁移表结构
	if err := db.AutoMigrate(
		&WorkflowModel{},
		&WorkflowStepModel{},
		&WorkflowExecutionModel{},
		&WorkflowStepExecutionModel{},
	); err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to migrate workflow tables")
	}

	return &workflowRepo{db: db}, nil
}

// Create 创建 Workflow
func (r *workflowRepo) Create(ctx context.Context, wf *workflow.Workflow) error {
	if wf == nil {
		return errors.New(errors.CodeInvalidParameter, "workflow cannot be nil")
	}

	// 验证 Workflow
	if err := wf.Validate(); err != nil {
		return errors.Wrap(err, errors.CodeInvalidParameter, "invalid workflow")
	}

	// 转换为数据库模型
	model, err := r.toModel(wf)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to convert workflow to model")
	}

	// 使用事务创建 Workflow 及其步骤
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建 Workflow
		if err := tx.Create(model).Error; err != nil {
			if isDuplicateKeyError(err) {
				return errors.Wrap(err, errors.CodeAlreadyExists, "workflow already exists")
			}
			return errors.Wrap(err, errors.CodeDatabaseError, "failed to create workflow")
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 更新实体的时间戳
	wf.CreatedAt = model.CreatedAt
	wf.UpdatedAt = model.UpdatedAt

	return nil
}

// GetByID 根据 ID 获取 Workflow
func (r *workflowRepo) GetByID(ctx context.Context, id string) (*workflow.Workflow, error) {
	if id == "" {
		return nil, errors.New(errors.CodeInvalidParameter, "workflow ID cannot be empty")
	}

	var model WorkflowModel
	if err := r.db.WithContext(ctx).
		Preload("Steps", func(db *gorm.DB) *gorm.DB {
			return db.Order("workflow_steps.order ASC")
		}).
		First(&model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "workflow not found")
		}
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get workflow")
	}

	return r.toEntity(&model)
}

// Update 更新 Workflow
func (r *workflowRepo) Update(ctx context.Context, wf *workflow.Workflow) error {
	if wf == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "workflow cannot be nil")
	}
	if wf.ID == "" {
		return errors.New(errors.CodeInvalidParameter, "workflow ID cannot be empty")
	}

	// 验证 Workflow
	if err := wf.Validate(); err != nil {
		return errors.Wrap(err, errors.CodeInvalidParameter, "invalid workflow")
	}

	// 转换为数据库模型
	model, err := r.toModel(wf)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to convert workflow to model")
	}

	// 使用事务更新 Workflow 及其步骤
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 使用乐观锁更新 Workflow
		result := tx.Model(&WorkflowModel{}).
			Where("id = ? AND version = ?", model.ID, model.Version-1).
			Updates(map[string]interface{}{
				"name":        model.Name,
				"description": model.Description,
				"trigger":     model.Trigger,
				"config":      model.Config,
				"status":      model.Status,
				"version":     gorm.Expr("version + 1"),
				"updated_by":  model.UpdatedBy,
				"updated_at":  time.Now(),
			})

		if result.Error != nil {
			return errors.Wrap(result.Error, errors.CodeDatabaseError, "failed to update workflow")
		}

		if result.RowsAffected == 0 {
			return errors.New(errors.CodeConflict, "workflow version conflict or not found")
		}

		// 删除旧的步骤
		if err := tx.Where("workflow_id = ?", model.ID).Delete(&WorkflowStepModel{}).Error; err != nil {
			return errors.Wrap(err, errors.CodeDatabaseError, "failed to delete old steps")
		}

		// 创建新的步骤
		if len(model.Steps) > 0 {
			if err := tx.Create(&model.Steps).Error; err != nil {
				return errors.Wrap(err, errors.CodeDatabaseError, "failed to create new steps")
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 更新实体的版本号和时间戳
	wf.Version++
	wf.UpdatedAt = time.Now()

	return nil
}

// Delete 删除 Workflow（软删除）
func (r *workflowRepo) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New(errors.CodeInvalidParameter, "workflow ID cannot be empty")
	}

	result := r.db.WithContext(ctx).Delete(&WorkflowModel{}, "id = ?", id)
	if result.Error != nil {
		return errors.Wrap(result.Error, errors.CodeDatabaseError, "failed to delete workflow")
	}

	if result.RowsAffected == 0 {
		return errors.New(errors.CodeNotFound, "workflow not found")
	}

	return nil
}

// List 列出 Workflow（分页）
func (r *workflowRepo) List(ctx context.Context, filter *workflow.WorkflowFilter) ([]*workflow.Workflow, int64, error) {
	if filter == nil {
		filter = &workflow.WorkflowFilter{}
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
	query := r.db.WithContext(ctx).Model(&WorkflowModel{})

	// 应用过滤条件
	if filter.Name != "" {
		query = query.Where("name ILIKE ?", "%"+filter.Name+"%")
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.CreatedBy != "" {
		query = query.Where("created_by = ?", filter.CreatedBy)
	}
	if !filter.CreatedAfter.IsZero() {
		query = query.Where("created_at >= ?", filter.CreatedAfter)
	}
	if !filter.CreatedBefore.IsZero() {
		query = query.Where("created_at <= ?", filter.CreatedBefore)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.Wrap(err, errors.CodeDatabaseError, "failed to count workflows")
	}

	// 应用排序
	orderBy := "created_at DESC"
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

	// 查询数据（预加载步骤）
	var models []WorkflowModel
	if err := query.
		Preload("Steps", func(db *gorm.DB) *gorm.DB {
			return db.Order("workflow_steps.order ASC")
		}).
		Find(&models).Error; err != nil {
		return nil, 0, errors.Wrap(err, errors.CodeDatabaseError, "failed to list workflows")
	}

	// 转换为实体
	workflows := make([]*workflow.Workflow, 0, len(models))
	for i := range models {
		wf, err := r.toEntity(&models[i])
		if err != nil {
			return nil, 0, errors.Wrap(err, errors.CodeInternalError, "failed to convert model to entity")
		}
		workflows = append(workflows, wf)
	}

	return workflows, total, nil
}

// UpdateStatus 更新 Workflow 状态
func (r *workflowRepo) UpdateStatus(ctx context.Context, id string, status workflow.WorkflowStatus) error {
	if id == "" {
		return errors.New(errors.CodeInvalidParameter, "workflow ID cannot be empty")
	}
	if !status.Valid() {
		return errors.New(errors.CodeInvalidParameter, "invalid workflow status")
	}

	result := r.db.WithContext(ctx).
		Model(&WorkflowModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status.String(),
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.CodeDatabaseError, "failed to update workflow status")
	}

	if result.RowsAffected == 0 {
		return errors.New(errors.CodeNotFound, "workflow not found")
	}

	return nil
}

// CreateExecution 创建 Workflow 执行记录
func (r *workflowRepo) CreateExecution(ctx context.Context, exec *workflow.WorkflowExecution) error {
	if exec == nil {
		return errors.New(errors.CodeInvalidParameter, "execution cannot be nil")
	}

	// 转换为数据库模型
	model, err := r.executionToModel(exec)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to convert execution to model")
	}

	// 使用事务创建执行记录及步骤执行记录
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(model).Error; err != nil {
			return errors.Wrap(err, errors.CodeDatabaseError, "failed to create execution")
		}
		return nil
	})

	if err != nil {
		return err
	}

	exec.CreatedAt = model.CreatedAt
	exec.UpdatedAt = model.UpdatedAt

	return nil
}

// GetExecutionByID 根据 ID 获取执行记录
func (r *workflowRepo) GetExecutionByID(ctx context.Context, id string) (*workflow.WorkflowExecution, error) {
	if id == "" {
		return nil, errors.New(errors.CodeInvalidParameter, "execution ID cannot be empty")
	}

	var model WorkflowExecutionModel
	if err := r.db.WithContext(ctx).
		Preload("Steps", func(db *gorm.DB) *gorm.DB {
			return db.Order("workflow_step_executions.created_at ASC")
		}).
		First(&model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.CodeNotFound, "execution not found")
		}
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get execution")
	}

	return r.executionToEntity(&model)
}

// UpdateExecution 更新执行记录
func (r *workflowRepo) UpdateExecution(ctx context.Context, exec *workflow.WorkflowExecution) error {
	if exec == nil {
		return errors.New(errors.CodeInvalidParameter, "execution cannot be nil")
	}

	model, err := r.executionToModel(exec)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to convert execution to model")
	}

	// 使用事务更新
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新执行记录
		if err := tx.Model(&WorkflowExecutionModel{}).
			Where("id = ?", model.ID).
			Updates(map[string]interface{}{
				"status":       model.Status,
				"output":       model.Output,
				"error":        model.Error,
				"completed_at": model.CompletedAt,
				"duration":     model.Duration,
				"context":      model.Context,
				"updated_at":   time.Now(),
			}).Error; err != nil {
			return errors.Wrap(err, errors.CodeDatabaseError, "failed to update execution")
		}

		// 更新步骤执行记录
		for _, step := range model.Steps {
			if err := tx.Model(&WorkflowStepExecutionModel{}).
				Where("id = ?", step.ID).
				Updates(map[string]interface{}{
					"status":       step.Status,
					"output":       step.Output,
					"error":        step.Error,
					"retry_count":  step.RetryCount,
					"completed_at": step.CompletedAt,
					"duration":     step.Duration,
					"updated_at":   time.Now(),
				}).Error; err != nil {
				return errors.Wrap(err, errors.CodeDatabaseError, "failed to update step execution")
			}
		}

		return nil
	})

	return err
}

// ListExecutions 列出执行记录
func (r *workflowRepo) ListExecutions(ctx context.Context, workflowID string, filter *workflow.ExecutionFilter) ([]*workflow.WorkflowExecution, int64, error) {
	if filter == nil {
		filter = &workflow.ExecutionFilter{}
	}

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	query := r.db.WithContext(ctx).Model(&WorkflowExecutionModel{})

	if workflowID != "" {
		query = query.Where("workflow_id = ?", workflowID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if !filter.StartedAfter.IsZero() {
		query = query.Where("started_at >= ?", filter.StartedAfter)
	}
	if !filter.StartedBefore.IsZero() {
		query = query.Where("started_at <= ?", filter.StartedBefore)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.Wrap(err, errors.CodeDatabaseError, "failed to count executions")
	}

	offset := (filter.Page - 1) * filter.PageSize
	var models []WorkflowExecutionModel
	if err := query.
		Preload("Steps").
		Order("started_at DESC").
		Offset(offset).
		Limit(filter.PageSize).
		Find(&models).Error; err != nil {
		return nil, 0, errors.Wrap(err, errors.CodeDatabaseError, "failed to list executions")
	}

	executions := make([]*workflow.WorkflowExecution, 0, len(models))
	for i := range models {
		exec, err := r.executionToEntity(&models[i])
		if err != nil {
			return nil, 0, errors.Wrap(err, errors.CodeInternalError, "failed to convert execution to entity")
		}
		executions = append(executions, exec)
	}

	return executions, total, nil
}

// toModel 将 Workflow 实体转换为数据库模型
func (r *workflowRepo) toModel(wf *workflow.Workflow) (*WorkflowModel, error) {
	triggerJSON, err := json.Marshal(wf.Trigger)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal trigger: %w", err)
	}

	configJSON, err := json.Marshal(wf.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	model := &WorkflowModel{
		ID:          wf.ID,
		Name:        wf.Name,
		Description: wf.Description,
		Trigger:     string(triggerJSON),
		Config:      string(configJSON),
		Status:      wf.Status.String(),
		Version:     wf.Version,
		CreatedBy:   wf.CreatedBy,
		UpdatedBy:   wf.UpdatedBy,
		CreatedAt:   wf.CreatedAt,
		UpdatedAt:   wf.UpdatedAt,
		Steps:       make([]WorkflowStepModel, 0, len(wf.Steps)),
	}

	for i, step := range wf.Steps {
		stepModel, err := r.stepToModel(step, wf.ID, i)
		if err != nil {
			return nil, err
		}
		model.Steps = append(model.Steps, *stepModel)
	}

	return model, nil
}

// stepToModel 将步骤实体转换为数据库模型
func (r *workflowRepo) stepToModel(step *workflow.WorkflowStep, workflowID string, order int) (*WorkflowStepModel, error) {
	configJSON, _ := json.Marshal(step.Config)
	inputJSON, _ := json.Marshal(step.Input)
	outputJSON, _ := json.Marshal(step.Output)
	depsJSON, _ := json.Marshal(step.Dependencies)
	condsJSON, _ := json.Marshal(step.Conditions)
	retryJSON, _ := json.Marshal(step.RetryPolicy)

	return &WorkflowStepModel{
		ID:           step.ID,
		WorkflowID:   workflowID,
		Name:         step.Name,
		Type:         step.Type.String(),
		AgentID:      step.AgentID,
		Config:       string(configJSON),
		Input:        string(inputJSON),
		Output:       string(outputJSON),
		Dependencies: string(depsJSON),
		Conditions:   string(condsJSON),
		RetryPolicy:  string(retryJSON),
		Timeout:      step.Timeout,
		Order:        order,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

// toEntity 将数据库模型转换为 Workflow 实体
func (r *workflowRepo) toEntity(model *WorkflowModel) (*workflow.Workflow, error) {
	var trigger workflow.WorkflowTrigger
	if err := json.Unmarshal([]byte(model.Trigger), &trigger); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trigger: %w", err)
	}

	var config workflow.WorkflowConfig
	if err := json.Unmarshal([]byte(model.Config), &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	status, err := workflow.WorkflowStatusFromString(model.Status)
	if err != nil {
		return nil, err
	}

	wf := &workflow.Workflow{
		ID:          model.ID,
		Name:        model.Name,
		Description: model.Description,
		Trigger:     trigger,
		Config:      config,
		Status:      status,
		Version:     model.Version,
		CreatedBy:   model.CreatedBy,
		UpdatedBy:   model.UpdatedBy,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
		Steps:       make([]*workflow.WorkflowStep, 0, len(model.Steps)),
	}

	for _, stepModel := range model.Steps {
		step, err := r.stepToEntity(&stepModel)
		if err != nil {
			return nil, err
		}
		wf.Steps = append(wf.Steps, step)
	}

	return wf, nil
}

// stepToEntity 将步骤模型转换为实体
func (r *workflowRepo) stepToEntity(model *WorkflowStepModel) (*workflow.WorkflowStep, error) {
	var config map[string]interface{}
	json.Unmarshal([]byte(model.Config), &config)

	var input map[string]interface{}
	json.Unmarshal([]byte(model.Input), &input)

	var output map[string]interface{}
	json.Unmarshal([]byte(model.Output), &output)

	var deps []string
	json.Unmarshal([]byte(model.Dependencies), &deps)

	var conds map[string]interface{}
	json.Unmarshal([]byte(model.Conditions), &conds)

	var retry workflow.RetryPolicy
	json.Unmarshal([]byte(model.RetryPolicy), &retry)

	stepType, err := workflow.StepTypeFromString(model.Type)
	if err != nil {
		return nil, err
	}

	return &workflow.WorkflowStep{
		ID:           model.ID,
		Name:         model.Name,
		Type:         stepType,
		AgentID:      model.AgentID,
		Config:       config,
		Input:        input,
		Output:       output,
		Dependencies: deps,
		Conditions:   conds,
		RetryPolicy:  retry,
		Timeout:      model.Timeout,
	}, nil
}

// executionToModel 将执行实体转换为模型
func (r *workflowRepo) executionToModel(exec *workflow.WorkflowExecution) (*WorkflowExecutionModel, error) {
	inputJSON, _ := json.Marshal(exec.Input)
	outputJSON, _ := json.Marshal(exec.Output)
	contextJSON, _ := json.Marshal(exec.Context)

	model := &WorkflowExecutionModel{
		ID:          exec.ID,
		WorkflowID:  exec.WorkflowID,
		Status:      exec.Status.String(),
		Input:       string(inputJSON),
		Output:      string(outputJSON),
		Error:       exec.Error,
		StartedAt:   exec.StartedAt,
		CompletedAt: exec.CompletedAt,
		Duration:    exec.Duration,
		TriggeredBy: exec.TriggeredBy,
		Context:     string(contextJSON),
		CreatedAt:   exec.CreatedAt,
		UpdatedAt:   exec.UpdatedAt,
		Steps:       make([]WorkflowStepExecutionModel, 0, len(exec.StepExecutions)),
	}

	for _, stepExec := range exec.StepExecutions {
		stepModel := r.stepExecutionToModel(stepExec, exec.ID)
		model.Steps = append(model.Steps, *stepModel)
	}

	return model, nil
}

// stepExecutionToModel 将步骤执行转换为模型
func (r *workflowRepo) stepExecutionToModel(exec *workflow.StepExecution, executionID string) *WorkflowStepExecutionModel {
	inputJSON, _ := json.Marshal(exec.Input)
	outputJSON, _ := json.Marshal(exec.Output)

	return &WorkflowStepExecutionModel{
		ID:          exec.ID,
		ExecutionID: executionID,
		StepID:      exec.StepID,
		StepName:    exec.StepName,
		Status:      exec.Status.String(),
		Input:       string(inputJSON),
		Output:      string(outputJSON),
		Error:       exec.Error,
		RetryCount:  exec.RetryCount,
		StartedAt:   exec.StartedAt,
		CompletedAt: exec.CompletedAt,
		Duration:    exec.Duration,
		CreatedAt:   exec.CreatedAt,
		UpdatedAt:   exec.UpdatedAt,
	}
}

// executionToEntity 将执行模型转换为实体
func (r *workflowRepo) executionToEntity(model *WorkflowExecutionModel) (*workflow.WorkflowExecution, error) {
	var input map[string]interface{}
	json.Unmarshal([]byte(model.Input), &input)

	var output map[string]interface{}
	json.Unmarshal([]byte(model.Output), &output)

	var context map[string]interface{}
	json.Unmarshal([]byte(model.Context), &context)
	status, err := workflow.ExecutionStatusFromString(model.Status)
	if err != nil {
		return nil, err
	}

	exec := &workflow.WorkflowExecution{
		ID:          model.ID,
		WorkflowID:  model.WorkflowID,
		Status:      status,
		Input:       input,
		Output:      output,
		Error:       model.Error,
		StartedAt:   model.StartedAt,
		CompletedAt: model.CompletedAt,
		Duration:    model.Duration,
		TriggeredBy: model.TriggeredBy,
		Context:     context,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
		StepExecutions: make([]*workflow.StepExecution, 0, len(model.Steps)),
	}

	for _, stepModel := range model.Steps {
		stepExec, err := r.stepExecutionToEntity(&stepModel)
		if err != nil {
			return nil, err
		}
		exec.StepExecutions = append(exec.StepExecutions, stepExec)
	}

	return exec, nil
}

// stepExecutionToEntity 将步骤执行模型转换为实体
func (r *workflowRepo) stepExecutionToEntity(model *WorkflowStepExecutionModel) (*workflow.StepExecution, error) {
	var input map[string]interface{}
	json.Unmarshal([]byte(model.Input), &input)

	var output map[string]interface{}
	json.Unmarshal([]byte(model.Output), &output)

	status, err := workflow.ExecutionStatusFromString(model.Status)
	if err != nil {
		return nil, err
	}

	return &workflow.StepExecution{
		ID:          model.ID,
		StepID:      model.StepID,
		StepName:    model.StepName,
		Status:      status,
		Input:       input,
		Output:      output,
		Error:       model.Error,
		RetryCount:  model.RetryCount,
		StartedAt:   model.StartedAt,
		CompletedAt: model.CompletedAt,
		Duration:    model.Duration,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}, nil
}

// Transaction 执行事务
func (r *workflowRepo) Transaction(ctx context.Context, fn func(repo workflow.WorkflowRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &workflowRepo{db: tx}
		return fn(txRepo)
	})
}

// GetStatistics 获取 Workflow 统计信息
func (r *workflowRepo) GetStatistics(ctx context.Context) (*workflow.WorkflowStatistics, error) {
	var stats workflow.WorkflowStatistics

	// 总数
	if err := r.db.WithContext(ctx).
		Model(&WorkflowModel{}).
		Count(&stats.Total).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get total count")
	}

	// 按状态统计
	type StatusCount struct {
		Status string
		Count  int64
	}
	var statusCounts []StatusCount
	if err := r.db.WithContext(ctx).
		Model(&WorkflowModel{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&statusCounts).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get status counts")
	}

	stats.ByStatus = make(map[string]int64)
	for _, sc := range statusCounts {
		stats.ByStatus[sc.Status] = sc.Count
	}

	// 执行统计
	if err := r.db.WithContext(ctx).
		Model(&WorkflowExecutionModel{}).
		Count(&stats.TotalExecutions).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get execution count")
	}

	// 平均执行时长
	var avgDuration sql.NullFloat64
	if err := r.db.WithContext(ctx).
		Model(&WorkflowExecutionModel{}).
		Select("AVG(duration) as avg_duration").
		Where("status = ?", "completed").
		Scan(&avgDuration).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get avg duration")
	}

	if avgDuration.Valid {
		stats.AverageDuration = int64(avgDuration.Float64)
	}

	// 成功率
	var successCount int64
	if err := r.db.WithContext(ctx).
		Model(&WorkflowExecutionModel{}).
		Where("status = ?", "completed").
		Count(&successCount).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get success count")
	}

	if stats.TotalExecutions > 0 {
		stats.SuccessRate = float64(successCount) / float64(stats.TotalExecutions) * 100
	}

	return &stats, nil
}

// GetWorkflowsByAgentID 根据 Agent ID 获取相关 Workflow
func (r *workflowRepo) GetWorkflowsByAgentID(ctx context.Context, agentID string) ([]*workflow.Workflow, error) {
	if agentID == "" {
		return nil, errors.New(errors.CodeInvalidParameter, "agent ID cannot be empty")
	}

	var models []WorkflowModel
	if err := r.db.WithContext(ctx).
		Joins("JOIN workflow_steps ON workflow_steps.workflow_id = workflows.id").
		Where("workflow_steps.agent_id = ?", agentID).
		Preload("Steps").
		Distinct().
		Find(&models).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get workflows by agent ID")
	}

	workflows := make([]*workflow.Workflow, 0, len(models))
	for i := range models {
		wf, err := r.toEntity(&models[i])
		if err != nil {
			return nil, errors.Wrap(err, errors.CodeInternalError, "failed to convert model to entity")
		}
		workflows = append(workflows, wf)
	}

	return workflows, nil
}

// DeleteExecutionsByWorkflowID 删除 Workflow 的所有执行记录
func (r *workflowRepo) DeleteExecutionsByWorkflowID(ctx context.Context, workflowID string) error {
	if workflowID == "" {
		return errors.New(errors.CodeInvalidParameter, "workflow ID cannot be empty")
	}

	// 由于外键级联删除，只需删除执行记录即可
	if err := r.db.WithContext(ctx).
		Where("workflow_id = ?", workflowID).
		Delete(&WorkflowExecutionModel{}).Error; err != nil {
		return errors.Wrap(err, errors.CodeDatabaseError, "failed to delete executions")
	}

	return nil
}

// GetRecentExecutions 获取最近的执行记录
func (r *workflowRepo) GetRecentExecutions(ctx context.Context, workflowID string, limit int) ([]*workflow.WorkflowExecution, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	query := r.db.WithContext(ctx).Model(&WorkflowExecutionModel{})

	if workflowID != "" {
		query = query.Where("workflow_id = ?", workflowID)
	}

	var models []WorkflowExecutionModel
	if err := query.
		Preload("Steps").
		Order("started_at DESC").
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get recent executions")
	}

	executions := make([]*workflow.WorkflowExecution, 0, len(models))
	for i := range models {
		exec, err := r.executionToEntity(&models[i])
		if err != nil {
			return nil, errors.Wrap(err, errors.CodeInternalError, "failed to convert execution to entity")
		}
		executions = append(executions, exec)
	}

	return executions, nil
}

// CountActiveWorkflows 统计活跃的 Workflow 数量
func (r *workflowRepo) CountActiveWorkflows(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&WorkflowModel{}).
		Where("status IN (?)", []string{"active", "running"}).
		Count(&count).Error; err != nil {
		return 0, errors.Wrap(err, errors.CodeDatabaseError, "failed to count active workflows")
	}

	return count, nil
}

// GetExecutionDurationDistribution 获取执行时长分布
func (r *workflowRepo) GetExecutionDurationDistribution(ctx context.Context, workflowID string) (map[string]int64, error) {
	type DurationBucket struct {
		Bucket string
		Count  int64
	}

	query := r.db.WithContext(ctx).Model(&WorkflowExecutionModel{}).
		Select(`
			CASE 
				WHEN duration < 1000 THEN '0-1s'
				WHEN duration < 5000 THEN '1-5s'
				WHEN duration < 10000 THEN '5-10s'
				WHEN duration < 30000 THEN '10-30s'
				WHEN duration < 60000 THEN '30-60s'
				ELSE '60s+'
			END as bucket,
			COUNT(*) as count
		`).
		Where("status = ?", "completed").
		Group("bucket")

	if workflowID != "" {
		query = query.Where("workflow_id = ?", workflowID)
	}

	var buckets []DurationBucket
	if err := query.Find(&buckets).Error; err != nil {
		return nil, errors.Wrap(err, errors.CodeDatabaseError, "failed to get duration distribution")
	}

	distribution := make(map[string]int64)
	for _, b := range buckets {
		distribution[b.Bucket] = b.Count
	}

	return distribution, nil
}

// BatchUpdateStepStatus 批量更新步骤执行状态
func (r *workflowRepo) BatchUpdateStepStatus(ctx context.Context, stepExecutionIDs []string, status workflow.ExecutionStatus) error {
	if len(stepExecutionIDs) == 0 {
		return errors.New(errors.CodeInvalidParameter, "step execution IDs cannot be empty")
	}
	if !status.Valid() {
		return errors.New(errors.CodeInvalidParameter, "invalid execution status")
	}

	result := r.db.WithContext(ctx).
		Model(&WorkflowStepExecutionModel{}).
		Where("id IN ?", stepExecutionIDs).
		Updates(map[string]interface{}{
			"status":     status.String(),
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return errors.Wrap(result.Error, errors.CodeDatabaseError, "failed to batch update step status")
	}

	return nil
}

// CleanupOldExecutions 清理旧的执行记录
func (r *workflowRepo) CleanupOldExecutions(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&WorkflowExecutionModel{})

	if result.Error != nil {
		return 0, errors.Wrap(result.Error, errors.CodeDatabaseError, "failed to cleanup old executions")
	}

	return result.RowsAffected, nil
}

//Personal.AI order the ending
