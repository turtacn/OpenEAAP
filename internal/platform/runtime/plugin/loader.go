package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"
	"time"

	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/platform/runtime"
	"github.com/openeeap/openeeap/pkg/errors"
)

// PluginLoader 插件加载器
type PluginLoader struct {
	logger        logging.Logger
	plugins       map[string]*LoadedPlugin
	pluginsMu     sync.RWMutex
	searchPaths   []string
	config        *LoaderConfig
	validators    []PluginValidator
	hooks         *PluginHooks
	shutdownChan  chan struct{}
	wg            sync.WaitGroup
}

// LoaderConfig 加载器配置
type LoaderConfig struct {
	SearchPaths         []string               // 插件搜索路径
	AutoReload          bool                   // 自动重载
	ReloadInterval      time.Duration          // 重载检查间隔
	EnableValidation    bool                   // 启用验证
	EnableSandbox       bool                   // 启用沙箱
	MaxPlugins          int                    // 最大插件数
	LoadTimeout         time.Duration          // 加载超时
	InitTimeout         time.Duration          // 初始化超时
	ShutdownTimeout     time.Duration          // 关闭超时
	AllowedPluginTypes  []runtime.PluginType   // 允许的插件类型
	Metadata            map[string]interface{} // 元数据
}

// LoadedPlugin 已加载的插件
type LoadedPlugin struct {
	Plugin          runtime.Plugin         // 插件信息
	Instance        PluginInstance         // 插件实例
	GoPlugin        *plugin.Plugin         // Go插件对象
	LoadTime        time.Time              // 加载时间
	InitTime        time.Time              // 初始化时间
	LastAccessTime  time.Time              // 最后访问时间
	AccessCount     int64                  // 访问计数
	ErrorCount      int64                  // 错误计数
	Status          runtime.PluginStatus   // 状态
	Version         string                 // 版本
	Metadata        map[string]interface{} // 元数据
	mu              sync.RWMutex
}

// PluginInstance 插件实例接口
type PluginInstance interface {
	// Initialize 初始化插件
	Initialize(ctx context.Context, config map[string]interface{}) error

	// Execute 执行插件
	Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)

	// Shutdown 关闭插件
	Shutdown(ctx context.Context) error

	// GetMetadata 获取插件元数据
	GetMetadata() *PluginMetadata

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// PluginMetadata 插件元数据
type PluginMetadata struct {
	Name         string                 // 名称
	Version      string                 // 版本
	Description  string                 // 描述
	Author       string                 // 作者
	Type         runtime.PluginType     // 类型
	Capabilities []string               // 能力列表
	Dependencies []string               // 依赖项
	Schema       *runtime.PluginSchema  // 模式
	Extra        map[string]interface{} // 额外信息
}

// PluginValidator 插件验证器接口
type PluginValidator interface {
	// Validate 验证插件
	Validate(ctx context.Context, plugin *runtime.Plugin) error
}

// PluginHooks 插件钩子
type PluginHooks struct {
	BeforeLoad    func(ctx context.Context, pluginPath string) error
	AfterLoad     func(ctx context.Context, plugin *LoadedPlugin) error
	BeforeInit    func(ctx context.Context, plugin *LoadedPlugin) error
	AfterInit     func(ctx context.Context, plugin *LoadedPlugin) error
	BeforeExecute func(ctx context.Context, plugin *LoadedPlugin, input map[string]interface{}) error
	AfterExecute  func(ctx context.Context, plugin *LoadedPlugin, output map[string]interface{}, err error) error
	BeforeShutdown func(ctx context.Context, plugin *LoadedPlugin) error
	AfterShutdown func(ctx context.Context, plugin *LoadedPlugin) error
	OnError       func(ctx context.Context, plugin *LoadedPlugin, err error)
}

// DefaultLoaderConfig 默认加载器配置
func DefaultLoaderConfig() *LoaderConfig {
	return &LoaderConfig{
		SearchPaths:        []string{"./plugins"},
		AutoReload:         false,
		ReloadInterval:     5 * time.Minute,
		EnableValidation:   true,
		EnableSandbox:      false,
		MaxPlugins:         100,
		LoadTimeout:        30 * time.Second,
		InitTimeout:        30 * time.Second,
		ShutdownTimeout:    10 * time.Second,
		AllowedPluginTypes: []runtime.PluginType{
			runtime.PluginTypeTool,
			runtime.PluginTypeMemory,
			runtime.PluginTypeRetriever,
			runtime.PluginTypeParser,
			runtime.PluginTypeFormatter,
			runtime.PluginTypeValidator,
			runtime.PluginTypeMiddleware,
			runtime.PluginTypeCustom,
		},
		Metadata: make(map[string]interface{}),
	}
}

// NewPluginLoader 创建插件加载器
func NewPluginLoader(logger logging.Logger, config *LoaderConfig) (*PluginLoader, error) {
	if logger == nil {
		return nil, errors.New("ERR_INVALID_PARAMETER", errors.ErrorTypeValidation, "logger cannot be nil", 400)
	}

	if config == nil {
		config = DefaultLoaderConfig()
	}

	loader := &PluginLoader{
		logger:       logger,
		plugins:      make(map[string]*LoadedPlugin),
		searchPaths:  config.SearchPaths,
		config:       config,
		validators:   make([]PluginValidator, 0),
		hooks:        &PluginHooks{},
		shutdownChan: make(chan struct{}),
	}

	// 创建搜索路径
	for _, path := range loader.searchPaths {
		if err := os.MkdirAll(path, 0755); err != nil {
			loader.logger.Warn("failed to create search path",
				"path", path,
				"error", err)
		}
	}

	// 启动自动重载
	if config.AutoReload {
		loader.wg.Add(1)
		go loader.autoReloadLoop()
	}

	loader.logger.Info("plugin loader initialized",
		"search_paths", loader.searchPaths,
		"max_plugins", config.MaxPlugins)

	return loader, nil
}

// LoadPlugin 加载插件
func (pl *PluginLoader) LoadPlugin(ctx context.Context, pluginPath string) (*LoadedPlugin, error) {
	// 检查插件数量限制
	pl.pluginsMu.RLock()
	pluginCount := len(pl.plugins)
	pl.pluginsMu.RUnlock()

	if pluginCount >= pl.config.MaxPlugins {
		return nil, errors.New(errors.CodeResourceExhausted, "max plugins limit reached")
	}

	// 执行加载前钩子
	if pl.hooks.BeforeLoad != nil {
		if err := pl.hooks.BeforeLoad(ctx, pluginPath); err != nil {
			return nil, errors.Wrap(err, "ERR_INTERNAL", "before load hook failed")
		}
	}

	// 创建加载上下文
	loadCtx, cancel := context.WithTimeout(ctx, pl.config.LoadTimeout)
	defer cancel()

	// 加载Go插件
	goPlugin, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to load plugin file")
	}

	// 查找插件符号
	symbol, err := goPlugin.Lookup("Plugin")
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "plugin symbol not found")
	}

	// 转换为插件实例
	instance, ok := symbol.(PluginInstance)
	if !ok {
		return nil, errors.InternalError("invalid plugin instance type")
	}

	// 获取插件元数据
	metadata := instance.GetMetadata()
	if metadata == nil {
		return nil, errors.InternalError("plugin metadata is nil")
	}

	// 创建加载的插件对象
	loadedPlugin := &LoadedPlugin{
		Plugin: runtime.Plugin{
			ID:          fmt.Sprintf("plugin_%d", time.Now().UnixNano()),
			Name:        metadata.Name,
			Version:     metadata.Version,
			Type:        metadata.Type,
			Description: metadata.Description,
			Author:      metadata.Author,
			Path:        pluginPath,
			Capabilities: metadata.Capabilities,
			Dependencies: metadata.Dependencies,
			Schema:      metadata.Schema,
			Status:      runtime.PluginStatusLoaded,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Metadata:    metadata.Extra,
		},
		Instance:       instance,
		GoPlugin:       goPlugin,
		LoadTime:       time.Now(),
		LastAccessTime: time.Now(),
		Status:         runtime.PluginStatusLoaded,
		Version:        metadata.Version,
		Metadata:       make(map[string]interface{}),
	}

	// 验证插件类型
	if !pl.isPluginTypeAllowed(loadedPlugin.Plugin.Type) {
		return nil, errors.New(errors.CodeInvalidParameter,
			fmt.Sprintf("plugin type not allowed: %s", loadedPlugin.Plugin.Type))
	}

	// 验证插件
	if pl.config.EnableValidation {
		if err := pl.validatePlugin(loadCtx, &loadedPlugin.Plugin); err != nil {
			return nil, errors.Wrap(err, errors.CodeInvalidParameter, "plugin validation failed")
		}
	}

	// 初始化插件
	if err := pl.initializePlugin(loadCtx, loadedPlugin); err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "plugin initialization failed")
	}

	// 保存插件
	pl.pluginsMu.Lock()
	pl.plugins[loadedPlugin.Plugin.ID] = loadedPlugin
	pl.pluginsMu.Unlock()

	// 执行加载后钩子
	if pl.hooks.AfterLoad != nil {
		if err := pl.hooks.AfterLoad(ctx, loadedPlugin); err != nil {
			pl.logger.Warn("after load hook failed", "error", err)
		}
	}

	pl.logger.Info("plugin loaded",
		"plugin_id", loadedPlugin.Plugin.ID,
		"plugin_name", loadedPlugin.Plugin.Name,
		"plugin_type", loadedPlugin.Plugin.Type,
		"version", loadedPlugin.Version)

	return loadedPlugin, nil
}

// LoadPluginFromConfig 从配置加载插件
func (pl *PluginLoader) LoadPluginFromConfig(ctx context.Context, config *runtime.PluginConfig) (*LoadedPlugin, error) {
	if config == nil {
		return nil, errors.New(errors.CodeInvalidParameter, "config cannot be nil")
	}

	// 查找插件文件
	pluginPath := config.Path
	if !filepath.IsAbs(pluginPath) {
		// 在搜索路径中查找
		found := false
		for _, searchPath := range pl.searchPaths {
			fullPath := filepath.Join(searchPath, pluginPath)
			if _, err := os.Stat(fullPath); err == nil {
				pluginPath = fullPath
				found = true
				break
			}
		}
		if !found {
			return nil, errors.New(errors.CodeNotFound,
				fmt.Sprintf("plugin file not found: %s", pluginPath))
		}
	}

	// 加载插件
	loadedPlugin, err := pl.LoadPlugin(ctx, pluginPath)
	if err != nil {
		return nil, err
	}

	// 应用配置
	if config.Config != nil {
		loadedPlugin.Plugin.Config = config.Config
	}
	if config.Enabled {
		loadedPlugin.Plugin.Enabled = true
		loadedPlugin.Status = runtime.PluginStatusActive
	}

	return loadedPlugin, nil
}

// UnloadPlugin 卸载插件
func (pl *PluginLoader) UnloadPlugin(ctx context.Context, pluginID string) error {
	pl.pluginsMu.Lock()
	loadedPlugin, exists := pl.plugins[pluginID]
	if !exists {
		pl.pluginsMu.Unlock()
		return errors.New(errors.CodeNotFound, "plugin not found")
	}
	delete(pl.plugins, pluginID)
	pl.pluginsMu.Unlock()

	// 执行关闭前钩子
	if pl.hooks.BeforeShutdown != nil {
		if err := pl.hooks.BeforeShutdown(ctx, loadedPlugin); err != nil {
			pl.logger.Warn("before shutdown hook failed", "error", err)
		}
	}

	// 关闭插件
	shutdownCtx, cancel := context.WithTimeout(ctx, pl.config.ShutdownTimeout)
	defer cancel()

	if err := loadedPlugin.Instance.Shutdown(shutdownCtx); err != nil {
		pl.logger.Warn("plugin shutdown failed",
			"plugin_id", pluginID,
			"error", err)
	}

	// 更新状态
	loadedPlugin.mu.Lock()
	loadedPlugin.Status = runtime.PluginStatusUnloaded
	loadedPlugin.Plugin.Status = runtime.PluginStatusUnloaded
	loadedPlugin.mu.Unlock()

	// 执行关闭后钩子
	if pl.hooks.AfterShutdown != nil {
		if err := pl.hooks.AfterShutdown(ctx, loadedPlugin); err != nil {
			pl.logger.Warn("after shutdown hook failed", "error", err)
		}
	}

	pl.logger.Info("plugin unloaded", "plugin_id", pluginID)
	return nil
}

// GetPlugin 获取插件
func (pl *PluginLoader) GetPlugin(pluginID string) (*LoadedPlugin, error) {
	pl.pluginsMu.RLock()
	defer pl.pluginsMu.RUnlock()

	loadedPlugin, exists := pl.plugins[pluginID]
	if !exists {
		return nil, errors.New(errors.CodeNotFound, "plugin not found")
	}

	return loadedPlugin, nil
}

// ListPlugins 列出所有插件
func (pl *PluginLoader) ListPlugins() []*LoadedPlugin {
	pl.pluginsMu.RLock()
	defer pl.pluginsMu.RUnlock()

	plugins := make([]*LoadedPlugin, 0, len(pl.plugins))
	for _, plugin := range pl.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// ExecutePlugin 执行插件
func (pl *PluginLoader) ExecutePlugin(ctx context.Context, pluginID string, input map[string]interface{}) (map[string]interface{}, error) {
	loadedPlugin, err := pl.GetPlugin(pluginID)
	if err != nil {
		return nil, err
	}

	// 检查插件状态
	loadedPlugin.mu.RLock()
	status := loadedPlugin.Status
	loadedPlugin.mu.RUnlock()

	if status != runtime.PluginStatusActive && status != runtime.PluginStatusLoaded {
		return nil, errors.New(errors.CodeInvalidParameter, "plugin not active")
	}

	// 更新访问信息
	loadedPlugin.mu.Lock()
	loadedPlugin.LastAccessTime = time.Now()
	loadedPlugin.AccessCount++
	loadedPlugin.mu.Unlock()

	// 执行前钩子
	if pl.hooks.BeforeExecute != nil {
		if err := pl.hooks.BeforeExecute(ctx, loadedPlugin, input); err != nil {
			return nil, errors.Wrap(err, "ERR_INTERNAL", "before execute hook failed")
		}
	}

	// 执行插件
	output, execErr := loadedPlugin.Instance.Execute(ctx, input)

	// 更新错误计数
	if execErr != nil {
		loadedPlugin.mu.Lock()
		loadedPlugin.ErrorCount++
		loadedPlugin.mu.Unlock()

		// 执行错误钩子
		if pl.hooks.OnError != nil {
			pl.hooks.OnError(ctx, loadedPlugin, execErr)
		}
	}

	// 执行后钩子
	if pl.hooks.AfterExecute != nil {
		if err := pl.hooks.AfterExecute(ctx, loadedPlugin, output, execErr); err != nil {
			pl.logger.Warn("after execute hook failed", "error", err)
		}
	}

	return output, execErr
}

// ReloadPlugin 重载插件
func (pl *PluginLoader) ReloadPlugin(ctx context.Context, pluginID string) error {
	loadedPlugin, err := pl.GetPlugin(pluginID)
	if err != nil {
		return err
	}

	pluginPath := loadedPlugin.Plugin.Path

	// 卸载旧插件
	if err := pl.UnloadPlugin(ctx, pluginID); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to unload plugin")
	}

	// 加载新插件
	_, err = pl.LoadPlugin(ctx, pluginPath)
	if err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to reload plugin")
	}

	pl.logger.Info("plugin reloaded", "plugin_id", pluginID)
	return nil
}

// DiscoverPlugins 发现插件
func (pl *PluginLoader) DiscoverPlugins(ctx context.Context) ([]*LoadedPlugin, error) {
	loadedPlugins := make([]*LoadedPlugin, 0)

	for _, searchPath := range pl.searchPaths {
		// 遍历搜索路径
		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// 只处理.so文件（Linux）或.dylib文件（macOS）
			if info.IsDir() || (!filepath.Ext(path) == ".so" && !filepath.Ext(path) == ".dylib") {
				return nil
			}

			// 加载插件
			loadedPlugin, loadErr := pl.LoadPlugin(ctx, path)
			if loadErr != nil {
				pl.logger.Warn("failed to load discovered plugin",
					"path", path,
					"error", loadErr)
				return nil
			}

			loadedPlugins = append(loadedPlugins, loadedPlugin)
			return nil
		})

		if err != nil {
			pl.logger.Warn("failed to walk search path",
				"path", searchPath,
				"error", err)
		}
	}

	pl.logger.Info("plugin discovery completed",
		"discovered_count", len(loadedPlugins))

	return loadedPlugins, nil
}

// AddValidator 添加验证器
func (pl *PluginLoader) AddValidator(validator PluginValidator) {
	if validator != nil {
		pl.validators = append(pl.validators, validator)
	}
}

// SetHooks 设置钩子
func (pl *PluginLoader) SetHooks(hooks *PluginHooks) {
	if hooks != nil {
		pl.hooks = hooks
	}
}

// Shutdown 关闭加载器
func (pl *PluginLoader) Shutdown(ctx context.Context) error {
	pl.logger.Info("shutting down plugin loader")

	// 发送关闭信号
	close(pl.shutdownChan)

	// 等待goroutine完成
	done := make(chan struct{})
	go func() {
		pl.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return errors.New(errors.DeadlineError, "shutdown timeout")
	}

	// 卸载所有插件
	pl.pluginsMu.RLock()
	pluginIDs := make([]string, 0, len(pl.plugins))
	for id := range pl.plugins {
		pluginIDs = append(pluginIDs, id)
	}
	pl.pluginsMu.RUnlock()

	for _, pluginID := range pluginIDs {
		if err := pl.UnloadPlugin(ctx, pluginID); err != nil {
			pl.logger.Warn("failed to unload plugin during shutdown",
				"plugin_id", pluginID,
				"error", err)
		}
	}

	pl.logger.Info("plugin loader shutdown completed")
	return nil
}

// 私有方法

// initializePlugin 初始化插件
func (pl *PluginLoader) initializePlugin(ctx context.Context, loadedPlugin *LoadedPlugin) error {
	// 执行初始化前钩子
	if pl.hooks.BeforeInit != nil {
		if err := pl.hooks.BeforeInit(ctx, loadedPlugin); err != nil {
			return err
		}
	}

	// 初始化插件
	initCtx, cancel := context.WithTimeout(ctx, pl.config.InitTimeout)
	defer cancel()

	if err := loadedPlugin.Instance.Initialize(initCtx, loadedPlugin.Plugin.Config); err != nil {
		return err
	}

	// 更新状态
	loadedPlugin.mu.Lock()
	loadedPlugin.InitTime = time.Now()
	loadedPlugin.Status = runtime.PluginStatusActive
	loadedPlugin.Plugin.Status = runtime.PluginStatusActive
	loadedPlugin.mu.Unlock()

	// 执行初始化后钩子
	if pl.hooks.AfterInit != nil {
		if err := pl.hooks.AfterInit(ctx, loadedPlugin); err != nil {
			pl.logger.Warn("after init hook failed", "error", err)
		}
	}

	return nil
}

// validatePlugin 验证插件
func (pl *PluginLoader) validatePlugin(ctx context.Context, plugin *runtime.Plugin) error {
	for _, validator := range pl.validators {
		if err := validator.Validate(ctx, plugin); err != nil {
			return err
		}
	}
	return nil
}

// isPluginTypeAllowed 检查插件类型是否允许
func (pl *PluginLoader) isPluginTypeAllowed(pluginType runtime.PluginType) bool {
	for _, allowedType := range pl.config.AllowedPluginTypes {
		if pluginType == allowedType {
			return true
		}
	}
	return false
}

// autoReloadLoop 自动重载循环
func (pl *PluginLoader) autoReloadLoop() {
	defer pl.wg.Done()

	ticker := time.NewTicker(pl.config.ReloadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pl.checkAndReloadPlugins()
		case <-pl.shutdownChan:
			return
		}
	}
}

// checkAndReloadPlugins 检查并重载插件
func (pl *PluginLoader) checkAndReloadPlugins() {
	pl.pluginsMu.RLock()
	pluginsToCheck := make([]*LoadedPlugin, 0, len(pl.plugins))
	for _, plugin := range pl.plugins {
		pluginsToCheck = append(pluginsToCheck, plugin)
	}
	pl.pluginsMu.RUnlock()

	for _, plugin := range pluginsToCheck {
		// 检查插件文件是否修改
		fileInfo, err := os.Stat(plugin.Plugin.Path)
		if err != nil {
			pl.logger.Warn("failed to stat plugin file",
				"plugin_id", plugin.Plugin.ID,
				"path", plugin.Plugin.Path,
				"error", err)
			continue
		}

		// 如果文件修改时间晚于加载时间，重载插件
		if fileInfo.ModTime().After(plugin.LoadTime) {
			ctx, cancel := context.WithTimeout(context.Background(), pl.config.LoadTimeout)
			if err := pl.ReloadPlugin(ctx, plugin.Plugin.ID); err != nil {
				pl.logger.Warn("failed to reload plugin",
					"plugin_id", plugin.Plugin.ID,
					"error", err)
			}
			cancel()
		}
	}
}

// GetPluginStats 获取插件统计信息
func (pl *PluginLoader) GetPluginStats(pluginID string) (map[string]interface{}, error) {
	loadedPlugin, err := pl.GetPlugin(pluginID)
	if err != nil {
		return nil, err
	}

	loadedPlugin.mu.RLock()
	defer loadedPlugin.mu.RUnlock()

	return map[string]interface{}{
		"plugin_id":        loadedPlugin.Plugin.ID,
		"name":             loadedPlugin.Plugin.Name,
		"type":             loadedPlugin.Plugin.Type,
		"version":          loadedPlugin.Version,
		"status":           loadedPlugin.Status,
		"load_time":        loadedPlugin.LoadTime,
		"init_time":        loadedPlugin.InitTime,
		"last_access_time": loadedPlugin.LastAccessTime,
		"access_count":     loadedPlugin.AccessCount,
		"error_count":      loadedPlugin.ErrorCount,
	}, nil
}

// GetLoaderStats 获取加载器统计信息
func (pl *PluginLoader) GetLoaderStats() map[string]interface{} {
	pl.pluginsMu.RLock()
	defer pl.pluginsMu.RUnlock()

	activeCount := 0
	inactiveCount := 0
	errorCount := 0

	for _, plugin := range pl.plugins {
		plugin.mu.RLock()
		switch plugin.Status {
		case runtime.PluginStatusActive:
			activeCount++
		case runtime.PluginStatusInactive:
			inactiveCount++
		case runtime.PluginStatusError:
			errorCount++
		}
		plugin.mu.RUnlock()
	}

	return map[string]interface{}{
		"total_plugins":   len(pl.plugins),
		"active_plugins":  activeCount,
		"inactive_plugins": inactiveCount,
		"error_plugins":   errorCount,
		"search_paths":    pl.searchPaths,
		"max_plugins":     pl.config.MaxPlugins,
		"auto_reload":     pl.config.AutoReload,
	}
}

//Personal.AI order the ending
