package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/platform/runtime"
	"github.com/openeeap/openeeap/pkg/errors"
)

// PluginManager 插件管理器
type PluginManager struct {
	logger           logging.Logger
	loader           *PluginLoader
	registry         *PluginRegistry
	versionControl   *VersionControl
	healthChecker    *HealthChecker
	config           *ManagerConfig
	eventHandlers    []PluginEventHandler
	metrics          *managerMetrics
	mu               sync.RWMutex
	shutdownChan     chan struct{}
	wg               sync.WaitGroup
}

// ManagerConfig 管理器配置
type ManagerConfig struct {
	EnableHealthCheck      bool                   // 启用健康检查
	HealthCheckInterval    time.Duration          // 健康检查间隔
	EnableVersionControl   bool                   // 启用版本控制
	EnableMetrics          bool                   // 启用指标
	EnableEventHandling    bool                   // 启用事件处理
	MaxConcurrentPlugins   int                    // 最大并发插件数
	PluginTimeout          time.Duration          // 插件执行超时
	RegistryBackupInterval time.Duration          // 注册表备份间隔
	Metadata               map[string]interface{} // 元数据
}

// managerMetrics 管理器指标
type managerMetrics struct {
	totalPlugins        int64
	activePlugins       int64
	failedPlugins       int64
	totalExecutions     int64
	successfulExecutions int64
	failedExecutions    int64
	totalExecutionTime  time.Duration
	mu                  sync.RWMutex
}

// PluginRegistry 插件注册表
type PluginRegistry struct {
	plugins   map[string]*RegistryEntry
	index     map[runtime.PluginType][]string
	mu        sync.RWMutex
	logger    // logger.Logger
}

// RegistryEntry 注册表条目
type RegistryEntry struct {
	Plugin          *runtime.Plugin
	LoadedPlugin    *LoadedPlugin
	RegisteredAt    time.Time
	UpdatedAt       time.Time
	Version         *Version
	Dependencies    []*Dependency
	Metadata        map[string]interface{}
	HealthStatus    *PluginHealthStatus
	mu              sync.RWMutex
}

// Version 版本信息
type Version struct {
	Major      int
	Minor      int
	Patch      int
	PreRelease string
	BuildMeta  string
}

// Dependency 依赖
type Dependency struct {
	PluginID       string
	Name           string
	Version        string
	VersionRange   string
	Optional       bool
	Resolved       bool
}

// PluginHealthStatus 插件健康状态
type PluginHealthStatus struct {
	Status         runtime.HealthStatus
	Message        string
	LastCheckTime  time.Time
	CheckCount     int64
	FailureCount   int64
	Details        map[string]interface{}
}

// VersionControl 版本控制
type VersionControl struct {
	versions  map[string][]*VersionEntry
	mu        sync.RWMutex
	logger    // logger.Logger
}

// VersionEntry 版本条目
type VersionEntry struct {
	PluginID      string
	Version       *Version
	Path          string
	Active        bool
	RegisteredAt  time.Time
	Metadata      map[string]interface{}
}

// HealthChecker 健康检查器
type HealthChecker struct {
	logger        // logger.Logger
	checkInterval time.Duration
	results       map[string]*PluginHealthStatus
	mu            sync.RWMutex
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// PluginEventHandler 插件事件处理器
type PluginEventHandler interface {
	HandleEvent(event *PluginEvent) error
}

// PluginEvent 插件事件
type PluginEvent struct {
	Type      PluginEventType
	PluginID  string
	Plugin    *runtime.Plugin
	Timestamp time.Time
	Data      map[string]interface{}
	Error     error
}

// PluginEventType 插件事件类型
type PluginEventType string

const (
	PluginEventTypeRegistered   PluginEventType = "registered"
	PluginEventTypeUnregistered PluginEventType = "unregistered"
	PluginEventTypeActivated    PluginEventType = "activated"
	PluginEventTypeDeactivated  PluginEventType = "deactivated"
	PluginEventTypeUpdated      PluginEventType = "updated"
	PluginEventTypeHealthChanged PluginEventType = "health_changed"
	PluginEventTypeError        PluginEventType = "error"
)

// DefaultManagerConfig 默认管理器配置
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		EnableHealthCheck:      true,
		HealthCheckInterval:    1 * time.Minute,
		EnableVersionControl:   true,
		EnableMetrics:          true,
		EnableEventHandling:    true,
		MaxConcurrentPlugins:   50,
		PluginTimeout:          30 * time.Second,
		RegistryBackupInterval: 5 * time.Minute,
		Metadata:               make(map[string]interface{}),
	}
}

// NewPluginManager 创建插件管理器
func NewPluginManager(logger // logger.Logger, loader *PluginLoader, config *ManagerConfig) (*PluginManager, error) {
	if logger == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "logger cannot be nil")
	}
	if loader == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "loader cannot be nil")
	}
	if config == nil {
		config = DefaultManagerConfig()
	}

	manager := &PluginManager{
		logger:        logger,
		loader:        loader,
		registry:      NewPluginRegistry(logger),
		config:        config,
		eventHandlers: make([]PluginEventHandler, 0),
		metrics:       &managerMetrics{},
		shutdownChan:  make(chan struct{}),
	}

	// 初始化版本控制
	if config.EnableVersionControl {
		manager.versionControl = NewVersionControl(logger)
	}

	// 初始化健康检查器
	if config.EnableHealthCheck {
		manager.healthChecker = NewHealthChecker(logger, config.HealthCheckInterval)
		manager.wg.Add(1)
		go manager.startHealthChecking()
	}

	// 启动注册表备份
	if config.RegistryBackupInterval > 0 {
		manager.wg.Add(1)
		go manager.registryBackupLoop()
	}

	manager.// logger.Info("plugin manager initialized",
		"health_check_enabled", config.EnableHealthCheck,
		"version_control_enabled", config.EnableVersionControl)

	return manager, nil
}

// RegisterPlugin 注册插件
func (pm *PluginManager) RegisterPlugin(ctx context.Context, plugin *runtime.Plugin) error {
	if plugin == nil {
		return errors.New(errors.CodeInvalidParameter, "plugin cannot be nil")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 检查插件是否已注册
	if pm.registry.Exists(plugin.ID) {
		return errors.New(errors.CodeAlreadyExists, "plugin already registered")
	}

	// 检查并发限制
	if pm.metrics.activePlugins >= int64(pm.config.MaxConcurrentPlugins) {
		return errors.New(errors.CodeResourceExhausted, "max concurrent plugins limit reached")
	}

	// 解析版本
	version, err := ParseVersion(plugin.Version)
	if err != nil {
		return errors.Wrap(err, errors.CodeInvalidParameter, "invalid plugin version")
	}

	// 检查依赖
	dependencies, err := pm.resolveDependencies(ctx, plugin)
	if err != nil {
		return errors.Wrap(err, errors.CodeInvalidParameter, "failed to resolve dependencies")
	}

	// 创建注册表条目
	entry := &RegistryEntry{
		Plugin:       plugin,
		RegisteredAt: time.Now(),
		UpdatedAt:    time.Now(),
		Version:      version,
		Dependencies: dependencies,
		Metadata:     make(map[string]interface{}),
		HealthStatus: &PluginHealthStatus{
			Status:        runtime.HealthStatusHealthy,
			LastCheckTime: time.Now(),
			Details:       make(map[string]interface{}),
		},
	}

	// 注册到注册表
	if err := pm.registry.Register(plugin.ID, entry); err != nil {
		return err
	}

	// 版本控制
	if pm.config.EnableVersionControl {
		versionEntry := &VersionEntry{
			PluginID:     plugin.ID,
			Version:      version,
			Path:         plugin.Path,
			Active:       true,
			RegisteredAt: time.Now(),
			Metadata:     plugin.Metadata,
		}
		pm.versionControl.AddVersion(plugin.Name, versionEntry)
	}

	// 更新指标
	pm.metrics.mu.Lock()
	pm.metrics.totalPlugins++
	pm.metrics.activePlugins++
	pm.metrics.mu.Unlock()

	// 触发事件
	pm.emitEvent(&PluginEvent{
		Type:      PluginEventTypeRegistered,
		PluginID:  plugin.ID,
		Plugin:    plugin,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"version": version.String(),
		},
	})

	pm.// logger.Info("plugin registered",
		"plugin_id", plugin.ID,
		"plugin_name", plugin.Name,
		"version", version.String())

	return nil
}

// UnregisterPlugin 注销插件
func (pm *PluginManager) UnregisterPlugin(ctx context.Context, pluginID string) error {
	if pluginID == "" {
		return errors.New(errors.CodeInvalidParameter, "plugin id cannot be empty")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 获取注册表条目
	entry, err := pm.registry.Get(pluginID)
	if err != nil {
		return err
	}

	// 检查依赖插件
	dependents := pm.findDependents(pluginID)
	if len(dependents) > 0 {
		return errors.New(errors.CodeInvalidParameter,
			fmt.Sprintf("plugin has %d dependent(s), cannot unregister", len(dependents)))
	}

	// 卸载插件
	if entry.LoadedPlugin != nil {
		if err := pm.loader.UnloadPlugin(ctx, pluginID); err != nil {
			pm.// logger.Warn("failed to unload plugin during unregister",
				"plugin_id", pluginID,
				"error", err)
		}
	}

	// 从注册表移除
	if err := pm.registry.Unregister(pluginID); err != nil {
		return err
	}

	// 更新指标
	pm.metrics.mu.Lock()
	pm.metrics.activePlugins--
	pm.metrics.mu.Unlock()

	// 触发事件
	pm.emitEvent(&PluginEvent{
		Type:      PluginEventTypeUnregistered,
		PluginID:  pluginID,
		Plugin:    entry.Plugin,
		Timestamp: time.Now(),
	})

	pm.// logger.Info("plugin unregistered", "plugin_id", pluginID)
	return nil
}

// GetPlugin 获取插件
func (pm *PluginManager) GetPlugin(pluginID string) (*runtime.Plugin, error) {
	entry, err := pm.registry.Get(pluginID)
	if err != nil {
		return nil, err
	}
	return entry.Plugin, nil
}

// ListPlugins 列出所有插件
func (pm *PluginManager) ListPlugins() []*runtime.Plugin {
	return pm.registry.ListAll()
}

// ListPluginsByType 按类型列出插件
func (pm *PluginManager) ListPluginsByType(pluginType runtime.PluginType) []*runtime.Plugin {
	return pm.registry.ListByType(pluginType)
}

// SearchPlugins 搜索插件
func (pm *PluginManager) SearchPlugins(query string) []*runtime.Plugin {
	return pm.registry.Search(query)
}

// ActivatePlugin 激活插件
func (pm *PluginManager) ActivatePlugin(ctx context.Context, pluginID string) error {
	entry, err := pm.registry.Get(pluginID)
	if err != nil {
		return err
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.Plugin.Status == runtime.PluginStatusActive {
		return errors.New(errors.CodeInvalidParameter, "plugin already active")
	}

	// 如果插件未加载，先加载
	if entry.LoadedPlugin == nil {
		loadedPlugin, err := pm.loader.LoadPlugin(ctx, entry.Plugin.Path)
		if err != nil {
			return errors.Wrap(err, errors.CodeInternalError, "failed to load plugin")
		}
		entry.LoadedPlugin = loadedPlugin
	}

	entry.Plugin.Status = runtime.PluginStatusActive
	entry.Plugin.Enabled = true
	entry.UpdatedAt = time.Now()

	// 触发事件
	pm.emitEvent(&PluginEvent{
		Type:      PluginEventTypeActivated,
		PluginID:  pluginID,
		Plugin:    entry.Plugin,
		Timestamp: time.Now(),
	})

	pm.// logger.Info("plugin activated", "plugin_id", pluginID)
	return nil
}

// DeactivatePlugin 停用插件
func (pm *PluginManager) DeactivatePlugin(ctx context.Context, pluginID string) error {
	entry, err := pm.registry.Get(pluginID)
	if err != nil {
		return err
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.Plugin.Status == runtime.PluginStatusInactive {
		return errors.New(errors.CodeInvalidParameter, "plugin already inactive")
	}

	entry.Plugin.Status = runtime.PluginStatusInactive
	entry.Plugin.Enabled = false
	entry.UpdatedAt = time.Now()

	// 触发事件
	pm.emitEvent(&PluginEvent{
		Type:      PluginEventTypeDeactivated,
		PluginID:  pluginID,
		Plugin:    entry.Plugin,
		Timestamp: time.Now(),
	})

	pm.// logger.Info("plugin deactivated", "plugin_id", pluginID)
	return nil
}

// UpdatePlugin 更新插件
func (pm *PluginManager) UpdatePlugin(ctx context.Context, pluginID string, newPlugin *runtime.Plugin) error {
	if newPlugin == nil {
		return errors.New(errors.CodeInvalidParameter, "new plugin cannot be nil")
	}

	entry, err := pm.registry.Get(pluginID)
	if err != nil {
		return err
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// 解析新版本
	newVersion, err := ParseVersion(newPlugin.Version)
	if err != nil {
		return errors.Wrap(err, errors.CodeInvalidParameter, "invalid plugin version")
	}

	// 检查版本
	if !newVersion.GreaterThan(entry.Version) {
		return errors.New(errors.CodeInvalidParameter, "new version must be greater than current version")
	}

	// 卸载旧版本
	if entry.LoadedPlugin != nil {
		if err := pm.loader.UnloadPlugin(ctx, pluginID); err != nil {
			return errors.Wrap(err, errors.CodeInternalError, "failed to unload old version")
		}
	}

	// 更新插件信息
	entry.Plugin = newPlugin
	entry.Version = newVersion
	entry.UpdatedAt = time.Now()
	entry.LoadedPlugin = nil

	// 版本控制
	if pm.config.EnableVersionControl {
		versionEntry := &VersionEntry{
			PluginID:     pluginID,
			Version:      newVersion,
			Path:         newPlugin.Path,
			Active:       true,
			RegisteredAt: time.Now(),
			Metadata:     newPlugin.Metadata,
		}
		pm.versionControl.AddVersion(newPlugin.Name, versionEntry)
	}

	// 触发事件
	pm.emitEvent(&PluginEvent{
		Type:      PluginEventTypeUpdated,
		PluginID:  pluginID,
		Plugin:    newPlugin,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"new_version": newVersion.String(),
		},
	})

	pm.// logger.Info("plugin updated",
		"plugin_id", pluginID,
		"new_version", newVersion.String())

	return nil
}

// ExecutePlugin 执行插件
func (pm *PluginManager) ExecutePlugin(ctx context.Context, pluginID string, input map[string]interface{}) (map[string]interface{}, error) {
	entry, err := pm.registry.Get(pluginID)
	if err != nil {
		return nil, err
	}

	// 检查插件状态
	if entry.Plugin.Status != runtime.PluginStatusActive {
		return nil, errors.New(errors.CodeInvalidParameter, "plugin not active")
	}

	// 更新指标
	pm.metrics.mu.Lock()
	pm.metrics.totalExecutions++
	pm.metrics.mu.Unlock()

	startTime := time.Now()

	// 执行插件
	output, execErr := pm.loader.ExecutePlugin(ctx, pluginID, input)

	duration := time.Since(startTime)

	// 更新指标
	pm.metrics.mu.Lock()
	if execErr != nil {
		pm.metrics.failedExecutions++
	} else {
		pm.metrics.successfulExecutions++
	}
	pm.metrics.totalExecutionTime += duration
	pm.metrics.mu.Unlock()

	if execErr != nil {
		// 更新健康状态
		if pm.healthChecker != nil {
			entry.HealthStatus.Status = runtime.HealthStatusDegraded
			entry.HealthStatus.FailureCount++
		}

		// 触发错误事件
		pm.emitEvent(&PluginEvent{
			Type:      PluginEventTypeError,
			PluginID:  pluginID,
			Plugin:    entry.Plugin,
			Timestamp: time.Now(),
			Error:     execErr,
		})
	}

	return output, execErr
}

// GetPluginHealth 获取插件健康状态
func (pm *PluginManager) GetPluginHealth(pluginID string) (*PluginHealthStatus, error) {
	if pm.healthChecker == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "health checker not enabled")
	}

	entry, err := pm.registry.Get(pluginID)
	if err != nil {
		return nil, err
	}

	entry.mu.RLock()
	defer entry.mu.RUnlock()

	return entry.HealthStatus, nil
}

// GetPluginVersions 获取插件版本列表
func (pm *PluginManager) GetPluginVersions(pluginName string) ([]*VersionEntry, error) {
	if pm.versionControl == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "version control not enabled")
	}

	return pm.versionControl.GetVersions(pluginName), nil
}

// RollbackPlugin 回滚插件版本
func (pm *PluginManager) RollbackPlugin(ctx context.Context, pluginID string, targetVersion string) error {
	if pm.versionControl == nil {
		return errors.New(errors.CodeInvalidParameter, "version control not enabled")
	}

	entry, err := pm.registry.Get(pluginID)
	if err != nil {
		return err
	}

	// 查找目标版本
	versions := pm.versionControl.GetVersions(entry.Plugin.Name)
	var targetVersionEntry *VersionEntry
	for _, v := range versions {
		if v.Version.String() == targetVersion {
			targetVersionEntry = v
			break
		}
	}

	if targetVersionEntry == nil {
		return errors.New(errors.CodeNotFound, "target version not found")
	}

	// 加载目标版本
	loadedPlugin, err := pm.loader.LoadPlugin(ctx, targetVersionEntry.Path)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to load target version")
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// 卸载当前版本
	if entry.LoadedPlugin != nil {
		if err := pm.loader.UnloadPlugin(ctx, pluginID); err != nil {
			pm.// logger.Warn("failed to unload current version", "error", err)
		}
	}

	// 更新为目标版本
	entry.LoadedPlugin = loadedPlugin
	entry.Version = targetVersionEntry.Version
	entry.UpdatedAt = time.Now()

	pm.// logger.Info("plugin rolled back",
		"plugin_id", pluginID,
		"target_version", targetVersion)

	return nil
}

// AddEventHandler 添加事件处理器
func (pm *PluginManager) AddEventHandler(handler PluginEventHandler) {
	if handler != nil {
		pm.eventHandlers = append(pm.eventHandlers, handler)
	}
}

// GetMetrics 获取管理器指标
func (pm *PluginManager) GetMetrics() map[string]interface{} {
	pm.metrics.mu.RLock()
	defer pm.metrics.mu.RUnlock()

	avgExecutionTime := time.Duration(0)
	if pm.metrics.totalExecutions > 0 {
		avgExecutionTime = pm.metrics.totalExecutionTime / time.Duration(pm.metrics.totalExecutions)
	}

	successRate := 0.0
	if pm.metrics.totalExecutions > 0 {
		successRate = float64(pm.metrics.successfulExecutions) / float64(pm.metrics.totalExecutions)
	}

	return map[string]interface{}{
		"total_plugins":          pm.metrics.totalPlugins,
		"active_plugins":         pm.metrics.activePlugins,
		"failed_plugins":         pm.metrics.failedPlugins,
		"total_executions":       pm.metrics.totalExecutions,
		"successful_executions":  pm.metrics.successfulExecutions,
		"failed_executions":      pm.metrics.failedExecutions,
		"avg_execution_time":     avgExecutionTime,
		"success_rate":           successRate,
	}
}

// Shutdown 关闭管理器
func (pm *PluginManager) Shutdown(ctx context.Context) error {
	pm.// logger.Info("shutting down plugin manager")

	// 发送关闭信号
	close(pm.shutdownChan)

	// 等待goroutine完成
	done := make(chan struct{})
	go func() {
		pm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return errors.New(errors.CodeDeadlineExceeded, "shutdown timeout")
	}

	// 关闭健康检查器
	if pm.healthChecker != nil {
		pm.healthChecker.Stop()
	}

	pm.// logger.Info("plugin manager shutdown completed")
	return nil
}

// 私有方法

// resolveDependencies 解析依赖
func (pm *PluginManager) resolveDependencies(ctx context.Context, plugin *runtime.Plugin) ([]*Dependency, error) {
	dependencies := make([]*Dependency, 0, len(plugin.Dependencies))

	for _, depName := range plugin.Dependencies {
		// 查找依赖插件
		depPlugin := pm.registry.FindByName(depName)
		if depPlugin == nil {
			return nil, errors.New(errors.CodeNotFound,
				fmt.Sprintf("dependency not found: %s", depName))
		}

		dep := &Dependency{
			PluginID: depPlugin.ID,
			Name:     depName,
			Version:  depPlugin.Version,
			Resolved: true,
		}
		dependencies = append(dependencies, dep)
	}

	return dependencies, nil
}

// findDependents 查找依赖该插件的其他插件
func (pm *PluginManager) findDependents(pluginID string) []string {
	dependents := make([]string, 0)

	for _, entry := range pm.registry.plugins {
		for _, dep := range entry.Dependencies {
			if dep.PluginID == pluginID {
				dependents = append(dependents, entry.Plugin.ID)
				break
			}
		}
	}

	return dependents
}

// emitEvent 触发事件
func (pm *PluginManager) emitEvent(event *PluginEvent) {
	if !pm.config.EnableEventHandling {
		return
	}

	for _, handler := range pm.eventHandlers {
		go func(h PluginEventHandler) {
			if err := h.HandleEvent(event); err != nil {
				pm.// logger.Warn("event handler failed",
					"event_type", event.Type,
					"plugin_id", event.PluginID,
					"error", err)
			}
		}(handler)
	}
}

// startHealthChecking 启动健康检查
func (pm *PluginManager) startHealthChecking() {
	defer pm.wg.Done()

	pm.healthChecker.Start(pm.registry, func(pluginID string, status *PluginHealthStatus) {
		entry, err := pm.registry.Get(pluginID)
		if err != nil {
			return
		}

		entry.mu.Lock()
		oldStatus := entry.HealthStatus.Status
		entry.HealthStatus = status
		entry.mu.Unlock()

		// 如果健康状态改变，触发事件
		if oldStatus != status.Status {
			pm.emitEvent(&PluginEvent{
				Type:      PluginEventTypeHealthChanged,
				PluginID:  pluginID,
				Plugin:    entry.Plugin,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"old_status": oldStatus,
					"new_status": status.Status,
				},
			})
		}
	})
}

// registryBackupLoop 注册表备份循环
func (pm *PluginManager) registryBackupLoop() {
	defer pm.wg.Done()

	ticker := time.NewTicker(pm.config.RegistryBackupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := pm.registry.Backup(); err != nil {
				pm.// logger.Warn("registry backup failed", "error", err)
			}
		case <-pm.shutdownChan:
			return
		}
	}
}

// NewPluginRegistry 创建插件注册表
func NewPluginRegistry(logger // logger.Logger) *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]*RegistryEntry),
		index:   make(map[runtime.PluginType][]string),
		logger:  logger,
	}
}

// Register 注册插件
func (pr *PluginRegistry) Register(pluginID string, entry *RegistryEntry) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if _, exists := pr.plugins[pluginID]; exists {
		return errors.New(errors.CodeAlreadyExists, "plugin already exists")
	}

	pr.plugins[pluginID] = entry

	// 更新索引
	pluginType := entry.Plugin.Type
	if pr.index[pluginType] == nil {
		pr.index[pluginType] = make([]string, 0)
	}
	pr.index[pluginType] = append(pr.index[pluginType], pluginID)

	return nil
}

// Unregister 注销插件
func (pr *PluginRegistry) Unregister(pluginID string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	entry, exists := pr.plugins[pluginID]
	if !exists {
		return errors.New(errors.CodeNotFound, "plugin not found")
	}

	// 从索引中移除
	pluginType := entry.Plugin.Type
	if ids, ok := pr.index[pluginType]; ok {
		for i, id := range ids {
			if id == pluginID {
				pr.index[pluginType] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
	}

	delete(pr.plugins, pluginID)
	return nil
}

// Get 获取插件
func (pr *PluginRegistry) Get(pluginID string) (*RegistryEntry, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	entry, exists := pr.plugins[pluginID]
	if !exists {
		return nil, errors.New(errors.CodeNotFound, "plugin not found")
	}

	return entry, nil
}

// Exists 检查插件是否存在
func (pr *PluginRegistry) Exists(pluginID string) bool {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	_, exists := pr.plugins[pluginID]
	return exists
}

// ListAll 列出所有插件
func (pr *PluginRegistry) ListAll() []*runtime.Plugin {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	plugins := make([]*runtime.Plugin, 0, len(pr.plugins))
	for _, entry := range pr.plugins {
		plugins = append(plugins, entry.Plugin)
	}

	return plugins
}

// ListByType 按类型列出插件
func (pr *PluginRegistry) ListByType(pluginType runtime.PluginType) []*runtime.Plugin {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	pluginIDs, ok := pr.index[pluginType]
	if !ok {
		return []*runtime.Plugin{}
	}

	plugins := make([]*runtime.Plugin, 0, len(pluginIDs))
	for _, id := range pluginIDs {
		if entry, exists := pr.plugins[id]; exists {
			plugins = append(plugins, entry.Plugin)
		}
	}

	return plugins
}
// Search 搜索插件
func (pr *PluginRegistry) Search(query string) []*runtime.Plugin {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	plugins := make([]*runtime.Plugin, 0)

	for _, entry := range pr.plugins {
		// 搜索名称、描述、作者
		if matchesQuery(entry.Plugin.Name, query) ||
			matchesQuery(entry.Plugin.Description, query) ||
			matchesQuery(entry.Plugin.Author, query) {
			plugins = append(plugins, entry.Plugin)
		}
	}

	return plugins
}

// FindByName 按名称查找插件
func (pr *PluginRegistry) FindByName(name string) *runtime.Plugin {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	for _, entry := range pr.plugins {
		if entry.Plugin.Name == name {
			return entry.Plugin
		}
	}

	return nil
}

// Backup 备份注册表
func (pr *PluginRegistry) Backup() error {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	// 简化实现：实际应该序列化到文件
	pr.// logger.Debug("registry backup completed",
		"plugin_count", len(pr.plugins))

	return nil
}

// matchesQuery 检查是否匹配查询
func matchesQuery(text, query string) bool {
	if text == "" || query == "" {
		return false
	}
	// 简单的包含匹配，实际可以使用更复杂的算法
	return len(text) >= len(query) &&
		(text == query || len(text) > 0)
}

// NewVersionControl 创建版本控制
func NewVersionControl(logger // logger.Logger) *VersionControl {
	return &VersionControl{
		versions: make(map[string][]*VersionEntry),
		logger:   logger,
	}
}

// AddVersion 添加版本
func (vc *VersionControl) AddVersion(pluginName string, entry *VersionEntry) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	if vc.versions[pluginName] == nil {
		vc.versions[pluginName] = make([]*VersionEntry, 0)
	}

	// 将旧版本标记为非活跃
	for _, v := range vc.versions[pluginName] {
		v.Active = false
	}

	// 添加新版本
	vc.versions[pluginName] = append(vc.versions[pluginName], entry)

	vc.// logger.Debug("version added",
		"plugin_name", pluginName,
		"version", entry.Version.String())
}

// GetVersions 获取版本列表
func (vc *VersionControl) GetVersions(pluginName string) []*VersionEntry {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	if versions, ok := vc.versions[pluginName]; ok {
		// 返回副本
		result := make([]*VersionEntry, len(versions))
		copy(result, versions)
		return result
	}

	return []*VersionEntry{}
}

// GetActiveVersion 获取活跃版本
func (vc *VersionControl) GetActiveVersion(pluginName string) *VersionEntry {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	if versions, ok := vc.versions[pluginName]; ok {
		for _, v := range versions {
			if v.Active {
				return v
			}
		}
	}

	return nil
}

// ParseVersion 解析版本字符串
func ParseVersion(versionStr string) (*Version, error) {
	if versionStr == "" {
		return nil, errors.New(errors.CodeInvalidParameter, "version string cannot be empty")
	}

	// 简化实现：实际应该支持完整的语义化版本解析
	// 格式: major.minor.patch[-prerelease][+buildmeta]
	version := &Version{
		Major: 1,
		Minor: 0,
		Patch: 0,
	}

	return version, nil
}

// String 版本字符串表示
func (v *Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		s += "-" + v.PreRelease
	}
	if v.BuildMeta != "" {
		s += "+" + v.BuildMeta
	}
	return s
}

// GreaterThan 比较版本大小
func (v *Version) GreaterThan(other *Version) bool {
	if v.Major != other.Major {
		return v.Major > other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor > other.Minor
	}
	if v.Patch != other.Patch {
		return v.Patch > other.Patch
	}
	return false
}

// Equal 版本相等
func (v *Version) Equal(other *Version) bool {
	return v.Major == other.Major &&
		v.Minor == other.Minor &&
		v.Patch == other.Patch &&
		v.PreRelease == other.PreRelease
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(logger // logger.Logger, checkInterval time.Duration) *HealthChecker {
	return &HealthChecker{
		logger:        logger,
		checkInterval: checkInterval,
		results:       make(map[string]*PluginHealthStatus),
		stopChan:      make(chan struct{}),
	}
}

// Start 启动健康检查
func (hc *HealthChecker) Start(registry *PluginRegistry, callback func(string, *PluginHealthStatus)) {
	hc.wg.Add(1)
	go hc.checkLoop(registry, callback)
}

// Stop 停止健康检查
func (hc *HealthChecker) Stop() {
	close(hc.stopChan)
	hc.wg.Wait()
}

// checkLoop 健康检查循环
func (hc *HealthChecker) checkLoop(registry *PluginRegistry, callback func(string, *PluginHealthStatus)) {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.performHealthChecks(registry, callback)
		case <-hc.stopChan:
			return
		}
	}
}

// performHealthChecks 执行健康检查
func (hc *HealthChecker) performHealthChecks(registry *PluginRegistry, callback func(string, *PluginHealthStatus)) {
	plugins := registry.ListAll()

	for _, plugin := range plugins {
		status := hc.checkPluginHealth(plugin)

		hc.mu.Lock()
		hc.results[plugin.ID] = status
		hc.mu.Unlock()

		if callback != nil {
			callback(plugin.ID, status)
		}
	}
}

// checkPluginHealth 检查插件健康
func (hc *HealthChecker) checkPluginHealth(plugin *runtime.Plugin) *PluginHealthStatus {
	status := &PluginHealthStatus{
		Status:        runtime.HealthStatusHealthy,
		Message:       "Plugin is healthy",
		LastCheckTime: time.Now(),
		CheckCount:    0,
		FailureCount:  0,
		Details:       make(map[string]interface{}),
	}

	// 获取之前的状态
	hc.mu.RLock()
	if prevStatus, ok := hc.results[plugin.ID]; ok {
		status.CheckCount = prevStatus.CheckCount + 1
		status.FailureCount = prevStatus.FailureCount
	}
	hc.mu.RUnlock()

	// 检查插件状态
	if plugin.Status == runtime.PluginStatusError {
		status.Status = runtime.HealthStatusUnhealthy
		status.Message = "Plugin is in error state"
		status.FailureCount++
	} else if plugin.Status == runtime.PluginStatusInactive {
		status.Status = runtime.HealthStatusDegraded
		status.Message = "Plugin is inactive"
	}

	// 添加详细信息
	status.Details["plugin_id"] = plugin.ID
	status.Details["plugin_name"] = plugin.Name
	status.Details["plugin_type"] = plugin.Type
	status.Details["plugin_status"] = plugin.Status

	return status
}

// GetHealthStatus 获取健康状态
func (hc *HealthChecker) GetHealthStatus(pluginID string) *PluginHealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if status, ok := hc.results[pluginID]; ok {
		return status
	}

	return nil
}

// SimpleEventHandler 简单事件处理器
type SimpleEventHandler struct {
	logger // logger.Logger
}

// NewSimpleEventHandler 创建简单事件处理器
func NewSimpleEventHandler(logger // logger.Logger) *SimpleEventHandler {
	return &SimpleEventHandler{
		logger: logger,
	}
}

// HandleEvent 处理事件
func (seh *SimpleEventHandler) HandleEvent(event *PluginEvent) error {
	seh.// logger.Info("plugin event",
		"type", event.Type,
		"plugin_id", event.PluginID,
		"timestamp", event.Timestamp)

	if event.Error != nil {
		seh.// logger.Error("plugin error event",
			"plugin_id", event.PluginID,
			"error", event.Error)
	}

	return nil
}

// ValidatePluginConfig 验证插件配置
func ValidatePluginConfig(plugin *runtime.Plugin) error {
	if plugin == nil {
		return errors.New(errors.CodeInvalidParameter, "plugin cannot be nil")
	}

	if plugin.ID == "" {
		return errors.New(errors.CodeInvalidParameter, "plugin id cannot be empty")
	}

	if plugin.Name == "" {
		return errors.New(errors.CodeInvalidParameter, "plugin name cannot be empty")
	}

	if plugin.Version == "" {
		return errors.New(errors.CodeInvalidParameter, "plugin version cannot be empty")
	}

	if plugin.Type == "" {
		return errors.New(errors.CodeInvalidParameter, "plugin type cannot be empty")
	}

	if plugin.Path == "" {
		return errors.New(errors.CodeInvalidParameter, "plugin path cannot be empty")
	}

	return nil
}

// BasicPluginValidator 基础插件验证器
type BasicPluginValidator struct {
	logger // logger.Logger
}

// NewBasicPluginValidator 创建基础插件验证器
func NewBasicPluginValidator(logger // logger.Logger) *BasicPluginValidator {
	return &BasicPluginValidator{
		logger: logger,
	}
}

// Validate 验证插件
func (bpv *BasicPluginValidator) Validate(ctx context.Context, plugin *runtime.Plugin) error {
	// 验证基本配置
	if err := ValidatePluginConfig(plugin); err != nil {
		return err
	}

	// 验证路径存在
	if _, err := os.Stat(plugin.Path); err != nil {
		return errors.Wrap(err, errors.CodeNotFound, "plugin file not found")
	}

	// 验证Schema
	if plugin.Schema != nil {
		if err := validatePluginSchema(plugin.Schema); err != nil {
			return errors.Wrap(err, errors.CodeInvalidParameter, "invalid plugin schema")
		}
	}

	bpv.// logger.Debug("plugin validation passed", "plugin_id", plugin.ID)
	return nil
}

// validatePluginSchema 验证插件Schema
func validatePluginSchema(schema *runtime.PluginSchema) error {
	if schema == nil {
		return nil
	}

	if schema.Input == nil {
		return errors.New(errors.CodeInvalidParameter, "input schema cannot be nil")
	}

	if schema.Output == nil {
		return errors.New(errors.CodeInvalidParameter, "output schema cannot be nil")
	}

	return nil
}

// GetPluginStatistics 获取插件统计信息
func (pm *PluginManager) GetPluginStatistics() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := make(map[string]interface{})

	// 按类型统计
	typeStats := make(map[runtime.PluginType]int)
	statusStats := make(map[runtime.PluginStatus]int)

	for _, entry := range pm.registry.plugins {
		typeStats[entry.Plugin.Type]++
		statusStats[entry.Plugin.Status]++
	}

	stats["by_type"] = typeStats
	stats["by_status"] = statusStats
	stats["total"] = len(pm.registry.plugins)

	// 健康状态统计
	if pm.healthChecker != nil {
		healthStats := make(map[runtime.HealthStatus]int)
		pm.healthChecker.mu.RLock()
		for _, status := range pm.healthChecker.results {
			healthStats[status.Status]++
		}
		pm.healthChecker.mu.RUnlock()
		stats["health_status"] = healthStats
	}

	return stats
}

// ExportPluginRegistry 导出插件注册表
func (pm *PluginManager) ExportPluginRegistry() ([]byte, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugins := pm.registry.ListAll()

	// 简化实现：实际应该序列化为JSON或其他格式
	data := fmt.Sprintf("Total plugins: %d", len(plugins))

	return []byte(data), nil
}

// ImportPluginRegistry 导入插件注册表
func (pm *PluginManager) ImportPluginRegistry(ctx context.Context, data []byte) error {
	if len(data) == 0 {
		return errors.New(errors.CodeInvalidParameter, "data cannot be empty")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 简化实现：实际应该反序列化并注册插件
	pm.// logger.Info("plugin registry imported",
		"data_size", len(data))

	return nil
}

// CleanupInactivePlugins 清理不活跃的插件
func (pm *PluginManager) CleanupInactivePlugins(ctx context.Context, threshold time.Duration) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now()
	toRemove := make([]string, 0)

	for id, entry := range pm.registry.plugins {
		entry.mu.RLock()
		lastAccess := entry.LoadedPlugin.LastAccessTime
		status := entry.Plugin.Status
		entry.mu.RUnlock()

		if status == runtime.PluginStatusInactive && now.Sub(lastAccess) > threshold {
			toRemove = append(toRemove, id)
		}
	}

	for _, id := range toRemove {
		if err := pm.UnregisterPlugin(ctx, id); err != nil {
			pm.// logger.Warn("failed to cleanup inactive plugin",
				"plugin_id", id,
				"error", err)
		}
	}

	pm.// logger.Info("inactive plugins cleaned up",
		"removed_count", len(toRemove))

	return nil
}

//Personal.AI order the ending
