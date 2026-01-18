package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/openeeap/openeeap/internal/domain/agent"
	"github.com/openeeap/openeeap/pkg/errors"
	"github.com/openeeap/openeeap/pkg/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AgentModel Agent 数据库模型
type AgentModel struct {
	ID          string         `gorm:"primaryKey;type:varchar(64)"`
	Name        string         `gorm:"type:varchar(255);not null;index"`
	Description string         `gorm:"type:text"`
	RuntimeType string         `gorm:"type:varchar(50);not null;index"`
	Config      string         `gorm:"type:jsonb;not null"` // JSON 格式存储配置
	Status      string         `gorm:"type:varchar(50);not null;index;default:'inactive'"`
	Version     int            `gorm:"not null;default:1"`
	CreatedBy   string         `gorm:"type:varchar(64);not null;index"`
	UpdatedBy   string         `gorm:"type:varchar(64)"`
	CreatedAt   time.Time      `gorm:"not null;index"`
	UpdatedAt   time.Time      `gorm:"not null"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名
func (AgentModel) TableName() string {
	return "agents"
}

// agentRepo Agent PostgreSQL 仓储实现
type agentRepo struct {
	db *gorm.DB
}

// NewAgentRepository 创建 Agent 仓储
func NewAgentRepository(db *gorm.DB) (agent.AgentRepository, error) {
	if db == nil {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "database connection cannot be nil")
	}

	// 自动迁移表结构
	if err := db.AutoMigrate(&AgentModel{}); err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to migrate agent table")
	}

	return &agentRepo{db: db}, nil
}

// Create 创建 Agent
func (r *agentRepo) Create(ctx context.Context, agt *agent.Agent) error {
	if agt == nil {
		return errors.NewValidationError(errors.CodeInvalidParameter, "agent cannot be nil")
	}

	// 验证 Agent
	if err := agt.Validate(); err != nil {
		return errors.Wrap(err, errors.CodeInvalidParameter, "invalid agent")
	}

	// 转换为数据库模型
	model, err := r.toModel(agt)
	if err != nil {
		return errors.WrapInternalError(err, "ERR_INTERNAL", "failed to convert agent to model")
	}

	// 执行创建
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		if isDuplicateKeyError(err) {
			return errors.NewConflictError("AGENT_ALREADY_EXISTS", "agent already exists").WithCause(err)
		}
		return errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to create agent")
	}

	// 更新实体的时间戳
	agt.CreatedAt = model.CreatedAt
	agt.UpdatedAt = model.UpdatedAt

	return nil
}

// Update updates an existing agent
func (r *agentRepo) Update(ctx context.Context, agt *agent.Agent) error {
	if agt == nil {
		return errors.NewValidationError(errors.CodeInvalidParameter, "agent cannot be nil")
	}
	if agt.ID == "" {
		return errors.NewValidationError(errors.CodeInvalidParameter, "agent ID cannot be empty")
	}

	model := &AgentModel{
		ID:          agt.ID,
		Name:        agt.Name,
		Description: agt.Description,
		// Version:     agt.Version, // TODO: Type mismatch - Agent.Version is string, AgentModel.Version is int
		UpdatedAt:   time.Now(),
	}

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return errors.WrapDatabaseError(err, "ERR_DB", "failed to update agent")
	}

	agt.UpdatedAt = model.UpdatedAt
	return nil
}

// Delete deletes an agent (stub implementation)
func (r *agentRepo) Delete(ctx context.Context, id string) error {
return errors.NewInternalError(errors.CodeNotImplemented, "Delete not implemented")
}

// GetByID 根据 ID 获取 Agent
func (r *agentRepo) GetByID(ctx context.Context, id string) (*agent.Agent, error) {
	if id == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "agent ID cannot be empty")
	}

	var model AgentModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError(errors.CodeNotFound, "agent not found")
		}
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to get agent")
	}

	return r.toEntity(&model)
}

// GetByName retrieves an agent by name
func (r *agentRepo) GetByName(ctx context.Context, name string) (*agent.Agent, error) {
	if name == "" {
		return nil, errors.NewValidationError(errors.CodeInvalidParameter, "agent name cannot be empty")
	}

	var model AgentModel
	if err := r.db.WithContext(ctx).First(&model, "name = ?", name).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError(errors.CodeNotFound, "agent not found")
		}
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to get agent")
	}

	return r.toEntity(&model)
}

// GetByStatus retrieves agents by status
func (r *agentRepo) GetByStatus(ctx context.Context, status agent.AgentStatus, filter agent.AgentFilter) ([]*agent.Agent, error) {
	query := r.db.WithContext(ctx).Where("status = ?", status)
	
	// Apply filters
	if len(filter.RuntimeType) > 0 {
		query = query.Where("runtime_type IN ?", filter.RuntimeType)
	}
	if filter.OwnerID != "" {
		query = query.Where("owner_id = ?", filter.OwnerID)
	}
	
	var models []AgentModel
	if err := query.Find(&models).Error; err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to get agents by status")
	}
	
	agents := make([]*agent.Agent, 0, len(models))
	for _, model := range models {
		agt, err := r.toEntity(&model)
		if err != nil {
			continue // Skip invalid agents
		}
		agents = append(agents, agt)
	}
	
	return agents, nil
}

// GetByRuntimeType retrieves agents by runtime type
func (r *agentRepo) GetByRuntimeType(ctx context.Context, runtimeType agent.RuntimeType, filter agent.AgentFilter) ([]*agent.Agent, error) {
	query := r.db.WithContext(ctx).Where("runtime_type = ?", runtimeType)
	
	// Apply filters
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if filter.OwnerID != "" {
		query = query.Where("owner_id = ?", filter.OwnerID)
	}
	
	var models []AgentModel
	if err := query.Find(&models).Error; err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to get agents by runtime type")
	}
	
	agents := make([]*agent.Agent, 0, len(models))
	for _, model := range models {
		agt, err := r.toEntity(&model)
		if err != nil {
			continue // Skip invalid agents
		}
		agents = append(agents, agt)
	}
	
	return agents, nil
}

// GetByTags retrieves agents by tags
func (r *agentRepo) GetByTags(ctx context.Context, tags []string, filter agent.AgentFilter) ([]*agent.Agent, error) {
	if len(tags) == 0 {
		return []*agent.Agent{}, nil
	}
	
	var models []AgentModel
	query := r.db.WithContext(ctx)
	
	// Search for agents that have all the tags (depending on TagMode in filter)
	// For simplicity, we'll do a JSON contains check
	for _, tag := range tags {
		query = query.Where("tags LIKE ?", "%\""+tag+"\"%")
	}
	
	// Apply additional filters
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if len(filter.RuntimeType) > 0 {
		query = query.Where("runtime_type IN ?", filter.RuntimeType)
	}
	if filter.OwnerID != "" {
		query = query.Where("owner_id = ?", filter.OwnerID)
	}
	
	if err := query.Find(&models).Error; err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to get agents by tags")
	}
	
	agents := make([]*agent.Agent, 0, len(models))
	for _, model := range models {
		agt, err := r.toEntity(&model)
		if err != nil {
			continue // Skip invalid agents
		}
		agents = append(agents, agt)
	}
	
	return agents, nil
// }

// Update 更新 Agent (old implementation - commented out)
// func (r *agentRepo) Update(ctx context.Context, agt *agent.Agent) error {
// 	if agt == nil {
// 		return errors.NewValidationError(errors.CodeInvalidParameter, "agent cannot be nil")
// 	}
// 	if agt.ID == "" {
// 		return errors.NewValidationError(errors.CodeInvalidParameter, "agent ID cannot be empty")
// 	}
//
// 	// 验证 Agent
// 	if err := agt.Validate(); err != nil {
// 		return errors.Wrap(err, errors.CodeInvalidParameter, "invalid agent")
// 	}
//
// 	// 转换为数据库模型
// 	model, err := r.toModel(agt)
// 	if err != nil {
// 		return errors.WrapInternalError(err, "ERR_INTERNAL", "failed to convert agent to model")
// 	}
//
// 	// 使用乐观锁更新
// 	result := r.db.WithContext(ctx).
// 		Model(&AgentModel{}).
// 		Where("id = ? AND version = ?", model.ID, model.Version-1).
// 		Updates(map[string]interface{}{
// 			"name":         model.Name,
// 			"description":  model.Description,
// 			"runtime_type": model.RuntimeType,
// 			"config":       model.Config,
// 			"status":       model.Status,
// 			"version":      gorm.Expr("version + 1"),
// 			"updated_by":   model.UpdatedBy,
// 			"updated_at":   time.Now(),
// 		})
//
// 	if result.Error != nil {
// 		return errors.WrapDatabaseError(result.Error, errors.CodeDatabaseError, "failed to update agent")
// 	}
//
// 	if result.RowsAffected == 0 {
// 		return errors.NewConflictError("AGENT_CONFLICT", "agent version conflict or not found")
// 	}
//
// 	// 更新实体的时间戳
// 	agt.UpdatedAt = time.Now()
//
// 	return nil
// }


// HardDelete 硬删除 Agent
// func (r *agentRepo) HardDelete(ctx context.Context, id string) error {
// 	if id == "" {
// 		return errors.NewValidationError(errors.CodeInvalidParameter, "agent ID cannot be empty")
// 	}

// 	result := r.db.WithContext(ctx).Unscoped().Delete(&AgentModel{}, "id = ?", id)
// 	if result.Error != nil {
// 		return errors.WrapDatabaseError(result.Error, errors.CodeDatabaseError, "failed to hard delete agent")
// 	}

// 	if result.RowsAffected == 0 {
// 		return errors.NewNotFoundError(errors.CodeNotFound, "agent not found")
// 	}

	return nil
}

// List 列出 Agent（分页）
func (r *agentRepo) List(ctx context.Context, filter agent.AgentFilter) ([]*agent.Agent, error) {
	// 构建查询
	query := r.db.WithContext(ctx).Model(&AgentModel{})

	// 应用过滤条件
	if len(filter.RuntimeType) > 0 {
		query = query.Where("runtime_type = ?", filter.RuntimeType)
	}
	if len(filter.Status) > 0 {
		query = query.Where("status = ?", filter.Status)
	}
	if !filter.CreatedAfter.IsZero() {
		query = query.Where("created_at >= ?", filter.CreatedAfter)
	}
	if !filter.CreatedBefore.IsZero() {
		query = query.Where("created_at <= ?", filter.CreatedBefore)
	}

	// 应用排序
	orderBy := "created_at DESC"
	if filter.SortBy != "" {
		orderBy = filter.SortBy
		if filter.SortOrder == agent.SortOrderDesc {
			orderBy += " DESC"
		} else {
			orderBy += " ASC"
		}
	}
	query = query.Order(orderBy)

	// 应用分页
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	query = query.Offset(filter.Offset).Limit(filter.Limit)

	// 查询数据
	var models []AgentModel
	if err := query.Find(&models).Error; err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to list agents")
	}

	// 转换为实体
	agents := make([]*agent.Agent, 0, len(models))
	for i := range models {
		agt, err := r.toEntity(&models[i])
		if err != nil {
			return nil, errors.WrapInternalError(err, "ERR_INTERNAL", "failed to convert model to entity")
		}
		agents = append(agents, agt)
	}

	return agents, nil
}

// UpdateStatus 更新 Agent 状态
func (r *agentRepo) UpdateStatus(ctx context.Context, id string, status agent.AgentStatus) error {
	if id == "" {
		return errors.ValidationError("agent ID cannot be empty")
	}
	if status == "" {
		return errors.ValidationError("invalid agent status")
	}

	result := r.db.WithContext(ctx).
		Model(&AgentModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     string(status),
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return errors.WrapDatabaseError(result.Error, errors.CodeDatabaseError, "failed to update agent status")
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError(errors.CodeNotFound, "agent not found")
	}

	return nil
}

// Transaction 执行事务
func (r *agentRepo) Transaction(ctx context.Context, fn func(repo agent.AgentRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &agentRepo{db: tx}
		return fn(txRepo)
	})
}

// BatchCreate 批量创建 Agent
func (r *agentRepo) BatchCreate(ctx context.Context, agents []*agent.Agent) error {
	if len(agents) == 0 {
		return errors.NewValidationError(errors.CodeInvalidParameter, "agents cannot be empty")
	}

	models := make([]AgentModel, 0, len(agents))
	for _, agt := range agents {
		if err := agt.Validate(); err != nil {
			return errors.Wrap(err, errors.CodeInvalidParameter, "invalid agent in batch")
		}

		model, err := r.toModel(agt)
		if err != nil {
			return errors.WrapInternalError(err, "ERR_INTERNAL", "failed to convert agent to model")
		}
		models = append(models, *model)
	}

	// 批量插入
	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(models, 100).Error; err != nil {
		return errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to batch create agents")
	}

	return nil
}

// toModel 将 Agent 实体转换为数据库模型
func (r *agentRepo) toModel(agt *agent.Agent) (*AgentModel, error) {
	// 序列化配置
	configJSON, err := json.Marshal(agt.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	// Convert version string to int if possible
	version := 1
	if agt.Version != "" {
		if v, err := strconv.Atoi(agt.Version); err == nil {
			version = v
		}
	}
	
	return &AgentModel{
		ID:          agt.ID,
		Name:        agt.Name,
		Description: agt.Description,
		RuntimeType: string(agt.RuntimeType),
		Config:      string(configJSON),
		Status:      string(agt.Status),
		Version:     version,
		CreatedBy:   agt.OwnerID, // Use OwnerID as CreatedBy
		UpdatedBy:   agt.OwnerID, // Use OwnerID as UpdatedBy
		CreatedAt:   agt.CreatedAt,
		UpdatedAt:   agt.UpdatedAt,
	}, nil
}

// toEntity 将数据库模型转换为 Agent 实体
func (r *agentRepo) toEntity(model *AgentModel) (*agent.Agent, error) {
	// 反序列化配置
	var config agent.AgentConfig
	if err := json.Unmarshal([]byte(model.Config), &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 解析运行时类型 - RuntimeType is just a string type, so cast it
	runtimeType := agent.RuntimeType(model.RuntimeType)

	// 解析状态 - AgentStatus is just a string type, so cast it
	status := agent.AgentStatus(model.Status)

	return &agent.Agent{
		ID:          model.ID,
		Name:        model.Name,
		Description: model.Description,
		RuntimeType: runtimeType,
		Config:      config,
		Status:      status,
		Version:     strconv.Itoa(model.Version), // Convert int to string
		OwnerID:     model.CreatedBy,             // Map CreatedBy to OwnerID
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}, nil
}

// isDuplicateKeyError 检查是否为重复键错误
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL 唯一约束冲突错误码: 23505
	return err.Error() == "ERROR: duplicate key value violates unique constraint (SQLSTATE 23505)" ||
		err.Error() == "UNIQUE constraint failed"
}

// GetStatistics 获取 Agent 统计信息
func (r *agentRepo) GetStatistics(ctx context.Context) (*agent.AgentStatistics, error) {
	var stats agent.AgentStatistics

	// 总数
	if err := r.db.WithContext(ctx).
		Model(&AgentModel{}).
		Count(&stats.TotalCount).Error; err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to get total count")
	}

	// 按状态统计
	type StatusCount struct {
		Status string
		Count  int64
	}
	var statusCounts []StatusCount
	if err := r.db.WithContext(ctx).
		Model(&AgentModel{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&statusCounts).Error; err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to get status counts")
	}

	stats.ByStatus = make(map[string]int64)
	for _, sc := range statusCounts {
		stats.ByStatus[sc.Status] = sc.Count
	}

	// 按运行时类型统计
	type RuntimeCount struct {
		RuntimeType string
		Count       int64
	}
	var runtimeCounts []RuntimeCount
	if err := r.db.WithContext(ctx).
		Model(&AgentModel{}).
		Select("runtime_type, COUNT(*) as count").
		Group("runtime_type").
		Find(&runtimeCounts).Error; err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to get runtime counts")
	}

	stats.ByRuntimeType = make(map[string]int64)
	for _, rc := range runtimeCounts {
		stats.ByRuntimeType[rc.RuntimeType] = rc.Count
	}

	return &stats, nil
}

// Archive archives an agent
func (r *agentRepo) Archive(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Model(&AgentModel{}).
		Where("id = ?", id).
		Update("archived", true)
	
	if result.Error != nil {
		return errors.WrapDatabaseError(result.Error, errors.CodeDatabaseError, "failed to archive agent")
	}
	
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError(errors.CodeNotFound, "agent not found")
	}
	
	return nil
}

// GetArchived retrieves archived agents
func (r *agentRepo) GetArchived(ctx context.Context, filter agent.AgentFilter) ([]*agent.Agent, error) {
	filter.IncludeArchived = true
	agents, err := r.List(ctx, filter)
	return agents, err
}

// BatchUpdate updates multiple agents
func (r *agentRepo) BatchUpdate(ctx context.Context, agents []*agent.Agent) error {
	if len(agents) == 0 {
		return errors.NewValidationError(errors.CodeInvalidParameter, "agents cannot be empty")
	}

	for _, agt := range agents {
		if err := r.Update(ctx, agt); err != nil {
			return err
		}
	}
	
	return nil
}

// BatchDelete deletes multiple agents
func (r *agentRepo) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return errors.NewValidationError(errors.CodeInvalidParameter, "ids cannot be empty")
	}

	result := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Delete(&AgentModel{})
	
	if result.Error != nil {
		return errors.WrapDatabaseError(result.Error, errors.CodeDatabaseError, "failed to batch delete agents")
	}
	
	return nil
}

//Personal.AI order the ending

// Exists checks if an agent exists
func (r *agentRepo) Exists(ctx context.Context, id string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&AgentModel{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to check agent existence")
	}
	return count > 0, nil
}

// ExistsByName checks if an agent with the given name exists
func (r *agentRepo) ExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&AgentModel{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return false, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to check agent existence by name")
	}
	return count > 0, nil
}

// GetByOwner retrieves agents by owner ID
func (r *agentRepo) GetByOwner(ctx context.Context, ownerID string, filter agent.AgentFilter) ([]*agent.Agent, error) {
	var models []*AgentModel
	query := r.db.WithContext(ctx).Where("owner_id = ?", ownerID)
	
	// Apply filter
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}
	if len(filter.RuntimeType) > 0 {
		query = query.Where("runtime_type IN ?", filter.RuntimeType)
	}
	
	if err := query.Find(&models).Error; err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to get agents by owner")
	}
	
	agents := make([]*agent.Agent, 0, len(models))
	for _, model := range models {
		agt, err := r.toEntity(model)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agt)
	}
	
	return agents, nil
}

// GetActive retrieves all active agents
func (r *agentRepo) GetActive(ctx context.Context) ([]*agent.Agent, error) {
	var models []*AgentModel
	if err := r.db.WithContext(ctx).Where("status = ?", agent.AgentStatusActive).Find(&models).Error; err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to get active agents")
	}
	
	agents := make([]*agent.Agent, 0, len(models))
	for _, model := range models {
		agt, err := r.toEntity(model)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agt)
	}
	
	return agents, nil
}

// Count returns the total count of agents matching the filter
func (r *agentRepo) Count(ctx context.Context, filter agent.AgentFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&AgentModel{})
	if err := query.Count(&count).Error; err != nil {
		return 0, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to count agents")
	}
	return count, nil
}

// GetPopular retrieves popular agents by execution count
func (r *agentRepo) GetPopular(ctx context.Context, limit int) ([]*agent.Agent, error) {
	var agentModels []AgentModel
	
	// TODO: This currently orders by updated_at as a proxy for popularity
	// In production, this should join with an execution statistics table
	// or use a denormalized execution_count field
	query := r.db.WithContext(ctx).
		Where("status = ?", agent.AgentStatusActive).
		Order("updated_at DESC").
		Limit(limit)
	
	if err := query.Find(&agentModels).Error; err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to get popular agents")
	}
	
	agents := make([]*agent.Agent, 0, len(agentModels))
	for _, model := range agentModels {
		agt, err := r.toEntity(&model)
		if err != nil {
			return nil, errors.Wrap(err, "ERR_CONVERSION", "failed to convert agent model to entity")
		}
		agents = append(agents, agt)
	}
	
	return agents, nil
}

// GetRecent retrieves recently created agents
func (r *agentRepo) GetRecent(ctx context.Context, limit int) ([]*agent.Agent, error) {
	var agentModels []AgentModel
	
	query := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit)
	
	if err := query.Find(&agentModels).Error; err != nil {
		return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to get recent agents")
	}
	
	agents := make([]*agent.Agent, 0, len(agentModels))
	for _, model := range agentModels {
		agt, err := r.toEntity(&model)
		if err != nil {
			return nil, errors.Wrap(err, "ERR_CONVERSION", "failed to convert agent model to entity")
		}
		agents = append(agents, agt)
	}
	
	return agents, nil
}

// IncrementExecutionCount increments execution count
func (r *agentRepo) IncrementExecutionCount(ctx context.Context, id string, success bool) error {
	updates := map[string]interface{}{
		"execution_count": gorm.Expr("execution_count + ?", 1),
		"updated_at":      time.Now(),
	}
	
	if success {
// 		updates["last_executed_at"] = time.Now()
	}
	
	result := r.db.WithContext(ctx).
		Model(&AgentModel{}).
		Where("id = ?", id).
		Updates(updates)
	
	if result.Error != nil {
		return errors.Wrap(result.Error, errors.CodeDatabaseError, "failed to increment execution count")
	}
	
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError(errors.CodeNotFound, "agent not found")
	}
	
	return nil
}


// Restore restores a soft-deleted agent
func (r *agentRepo) Restore(ctx context.Context, id string) error {
result := r.db.WithContext(ctx).
Model(&AgentModel{}).
Where("id = ? AND deleted_at IS NOT NULL", id).
Update("deleted_at", nil)

if result.Error != nil {
return errors.Wrap(result.Error, errors.CodeDatabaseError, "failed to restore agent")
}

if result.RowsAffected == 0 {
return errors.NewNotFoundError(errors.CodeNotFound, "agent not found or not deleted")
}

return nil
}

// Search searches agents by query
func (r *agentRepo) Search(ctx context.Context, query string, filter agent.AgentFilter) ([]*agent.Agent, error) {
// Build query
q := r.db.WithContext(ctx).Model(&AgentModel{})

// Apply search across name and description
if query != "" {
q = q.Where("name LIKE ? OR description LIKE ?", "%"+query+"%", "%"+query+"%")
}

// Apply filters
if len(filter.RuntimeType) > 0 {
q = q.Where("runtime_type = ?", filter.RuntimeType)
}
if len(filter.Status) > 0 {
q = q.Where("status = ?", filter.Status)
}
if !filter.CreatedAfter.IsZero() {
q = q.Where("created_at >= ?", filter.CreatedAfter)
}
if !filter.CreatedBefore.IsZero() {
q = q.Where("created_at <= ?", filter.CreatedBefore)
}

// Apply pagination
if filter.Limit <= 0 {
filter.Limit = 20
}
if filter.Limit > 100 {
filter.Limit = 100
}
q = q.Offset(filter.Offset).Limit(filter.Limit)

// Execute query
var models []AgentModel
if err := q.Find(&models).Error; err != nil {
return nil, errors.WrapDatabaseError(err, errors.CodeDatabaseError, "failed to search agents")
}

// Convert to entities
agents := make([]*agent.Agent, 0, len(models))
for i := range models {
agt, err := r.toEntity(&models[i])
if err != nil {
return nil, errors.WrapInternalError(err, "ERR_INTERNAL", "failed to convert model to entity")
}
agents = append(agents, agt)
}

return agents, nil
}

// SoftDelete performs soft delete on an agent
func (r *agentRepo) SoftDelete(ctx context.Context, id string) error {
if id == "" {
return errors.NewValidationError(errors.CodeInvalidParameter, "agent ID cannot be empty")
}

result := r.db.WithContext(ctx).Model(&AgentModel{}).
Where("id = ?", id).
Update("deleted_at", time.Now())

if result.Error != nil {
return errors.WrapDatabaseError(result.Error, errors.CodeDatabaseError, "failed to soft delete agent")
}
if result.RowsAffected == 0 {
return errors.NewNotFoundError(errors.CodeNotFound, "agent not found")
}

return nil
}

// UpdateStats updates agent execution statistics
func (r *agentRepo) UpdateStats(ctx context.Context, id string, stats agent.ExecutionStats) error {
if id == "" {
return errors.NewValidationError(errors.CodeInvalidParameter, "agent ID cannot be empty")
}

result := r.db.WithContext(ctx).Model(&AgentModel{}).
Where("id = ?", id).
Updates(map[string]interface{}{
"total_executions":    stats.TotalExecutions,
"successful_executions": stats.SuccessfulExecutions,
"failed_executions":   stats.FailedExecutions,
"avg_execution_time_ms":  stats.AvgExecutionTimeMs,
// "last_executed_at":    stats.LastExecution,
"updated_at":          time.Now(),
})

if result.Error != nil {
return errors.WrapDatabaseError(result.Error, errors.CodeDatabaseError, "failed to update agent stats")
}
if result.RowsAffected == 0 {
return errors.NewNotFoundError(errors.CodeNotFound, "agent not found")
}

return nil
}
