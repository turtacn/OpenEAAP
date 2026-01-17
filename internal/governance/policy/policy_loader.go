// internal/governance/policy/policy_loader.go
package policy

import (
"github.com/prometheus/client_golang/prometheus"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/pkg/errors"
)

// PolicyLoader 策略加载器接口
type PolicyLoader interface {
	// Load 加载策略
	Load(ctx context.Context) ([]*Policy, error)

	// LoadFromFile 从文件加载策略
	LoadFromFile(ctx context.Context, path string) (*Policy, error)

	// LoadFromDirectory 从目录加载所有策略
	LoadFromDirectory(ctx context.Context, dir string) ([]*Policy, error)

	// LoadFromDatabase 从数据库加载策略
	LoadFromDatabase(ctx context.Context, filter *PolicyFilter) ([]*Policy, error)

	// LoadFromConfigCenter 从配置中心加载策略
	LoadFromConfigCenter(ctx context.Context, key string) ([]*Policy, error)

	// Reload 重新加载策略
	Reload(ctx context.Context) error

	// Watch 监听策略变化
	Watch(ctx context.Context) (<-chan *PolicyChangeEvent, error)

	// StartAutoReload 启动自动重载
	StartAutoReload(ctx context.Context, interval time.Duration) error

	// StopAutoReload 停止自动重载
	StopAutoReload() error

	// Validate 验证策略文件
	Validate(ctx context.Context, path string) error
}

// PolicyChangeEvent 策略变更事件
type PolicyChangeEvent struct {
	Type      ChangeType             `json:"type"` // Added/Updated/Deleted
	PolicyID  string                 `json:"policy_id"`
	Policy    *Policy                `json:"policy"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"` // file/database/config_center
	Metadata  map[string]interface{} `json:"metadata"`
}

// ChangeType 变更类型
type ChangeType string

const (
	ChangeTypeAdded   ChangeType = "Added"
	ChangeTypeUpdated ChangeType = "Updated"
	ChangeTypeDeleted ChangeType = "Deleted"
)

// PolicySource 策略源接口
type PolicySource interface {
	// Fetch 获取策略
	Fetch(ctx context.Context, key string) (*Policy, error)

	// FetchAll 获取所有策略
	FetchAll(ctx context.Context) ([]*Policy, error)

	// Watch 监听变化
	Watch(ctx context.Context) (<-chan *PolicyChangeEvent, error)

	// Close 关闭源
	Close() error
}

// ConfigCenter 配置中心接口
type ConfigCenter interface {
	// Get 获取配置
	Get(ctx context.Context, key string) (string, error)

	// GetAll 获取所有配置
	GetAll(ctx context.Context, prefix string) (map[string]string, error)

	// Watch 监听配置变化
	Watch(ctx context.Context, key string) (<-chan string, error)

	// Close 关闭连接
	Close() error
}

// policyLoader 策略加载器实现
type policyLoader struct {
	pdp              PolicyDecisionPoint
	policyRepo       PolicyRepository
	configCenter     ConfigCenter
	logger           logging.Logger
	metricsCollector metrics.MetricsCollector
	tracer           trace.Tracer

	config           *PolicyLoaderConfig
	sources          []PolicySource
	loadedPolicies   sync.Map
	fileWatchers     map[string]*FileWatcher
	autoReloadTicker *time.Ticker
	autoReloadStop   chan struct{}
	changeListeners  []chan *PolicyChangeEvent
	mu               sync.RWMutex
}

// PolicyLoaderConfig 策略加载器配置
type PolicyLoaderConfig struct {
	FilePaths             []string      `yaml:"file_paths"`
	DirectoryPaths        []string      `yaml:"directory_paths"`
	DatabaseEnabled       bool          `yaml:"database_enabled"`
	ConfigCenterEnabled   bool          `yaml:"config_center_enabled"`
	ConfigCenterPrefix    string        `yaml:"config_center_prefix"`
	AutoReload            bool          `yaml:"auto_reload"`
	AutoReloadInterval    time.Duration `yaml:"auto_reload_interval"`
	WatchFiles            bool          `yaml:"watch_files"`
	ValidateOnLoad        bool          `yaml:"validate_on_load"`
	FailOnValidationError bool          `yaml:"fail_on_validation_error"`
	MaxConcurrentLoads    int           `yaml:"max_concurrent_loads"`
	LoadTimeout           time.Duration `yaml:"load_timeout"`
	CacheEnabled          bool          `yaml:"cache_enabled"`
}

// FileWatcher 文件监听器
type FileWatcher struct {
	path         string
	lastModified time.Time
	stopChan     chan struct{}
	mu           sync.RWMutex
}

// NewPolicyLoader 创建策略加载器
func NewPolicyLoader(
	pdp PolicyDecisionPoint,
	policyRepo PolicyRepository,
	configCenter ConfigCenter,
	logger logging.Logger,
	metricsCollector metrics.MetricsCollector,
	tracer trace.Tracer,
	config *PolicyLoaderConfig,
) PolicyLoader {
	return &policyLoader{
		pdp:              pdp,
		policyRepo:       policyRepo,
		configCenter:     configCenter,
		logger:           logger,
		metricsCollector: metricsCollector,
		tracer:           tracer,
		config:           config,
		sources:          []PolicySource{},
		fileWatchers:     make(map[string]*FileWatcher),
		changeListeners:  []chan *PolicyChangeEvent{},
	}
}

// Load 加载策略
func (l *policyLoader) Load(ctx context.Context) ([]*Policy, error) {
	ctx, span := l.tracer.Start(ctx, "PolicyLoader.Load")
	defer span.End()

	startTime := time.Now()
	defer func() {
		l.metricsCollector.ObserveDuration("policy_load_duration_ms",
			startTime,
			prometheus.Labels{"source": "all"})
	}()

	l.logger.WithContext(ctx).Info("Loading policies from all sources")

	allPolicies := []*Policy{}

	// 从文件加载
	for _, path := range l.config.FilePaths {
		policy, err := l.LoadFromFile(ctx, path)
		if err != nil {
   l.logger.WithContext(ctx).Warn("Failed to load policy from file", logging.Any("path", path), logging.Error(err))
			if l.config.FailOnValidationError {
				return nil, err
			}
			continue
		}
		allPolicies = append(allPolicies, policy)
	}

	// 从目录加载
	for _, dir := range l.config.DirectoryPaths {
		policies, err := l.LoadFromDirectory(ctx, dir)
		if err != nil {
   l.logger.WithContext(ctx).Warn("Failed to load policies from directory", logging.Any("dir", dir), logging.Error(err))
			if l.config.FailOnValidationError {
				return nil, err
			}
			continue
		}
		allPolicies = append(allPolicies, policies...)
	}

	// 从数据库加载
	if l.config.DatabaseEnabled && l.policyRepo != nil {
		policies, err := l.LoadFromDatabase(ctx, nil)
		if err != nil {
			l.logger.WithContext(ctx).Warn("Failed to load policies from database", logging.Error(err))
			if l.config.FailOnValidationError {
				return nil, err
			}
		} else {
			allPolicies = append(allPolicies, policies...)
		}
	}

	// 从配置中心加载
	if l.config.ConfigCenterEnabled && l.configCenter != nil {
		policies, err := l.loadFromConfigCenter(ctx)
		if err != nil {
			l.logger.WithContext(ctx).Warn("Failed to load policies from config center", logging.Error(err))
			if l.config.FailOnValidationError {
				return nil, err
			}
		} else {
			allPolicies = append(allPolicies, policies...)
		}
	}

	// 缓存加载的策略
	if l.config.CacheEnabled {
		for _, policy := range allPolicies {
			l.loadedPolicies.Store(policy.ID, policy)
		}
	}

	l.logger.WithContext(ctx).Info("Policies loaded successfully", logging.Any("count", len(allPolicies)))

	l.metricsCollector.Gauge("loaded_policies_total",
		float64(len(allPolicies)),
		map[string]string{})

	return allPolicies, nil
}

// LoadFromFile 从文件加载策略
func (l *policyLoader) LoadFromFile(ctx context.Context, path string) (*Policy, error) {
	ctx, span := l.tracer.Start(ctx, "PolicyLoader.LoadFromFile")
	defer span.End()

	l.logger.WithContext(ctx).Debug("Loading policy from file", logging.Any("path", path))

	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, errors.Wrap(err, errors.CodeNotFound,
			fmt.Sprintf("policy file not found: %s", path))
	}

	// 读取文件内容
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL",
			fmt.Sprintf("failed to read policy file: %s", path))
	}

	// 解析 YAML
	var policy Policy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidArgument,
			fmt.Sprintf("failed to parse policy file: %s", path))
	}

	// 验证策略
	if l.config.ValidateOnLoad {
		if err := l.validatePolicy(&policy); err != nil {
			return nil, errors.Wrap(err, errors.CodeInvalidArgument,
				fmt.Sprintf("policy validation failed: %s", path))
		}
	}

	// 设置元数据
	if policy.Metadata == nil {
		policy.Metadata = make(map[string]interface{})
	}
	policy.Metadata["source_file"] = path
	policy.Metadata["loaded_at"] = time.Now()

 l.logger.WithContext(ctx).Info("Policy loaded from file", logging.Any("path", path), logging.Any("policy_id", policy.ID))

	return &policy, nil
}

// LoadFromDirectory 从目录加载所有策略
func (l *policyLoader) LoadFromDirectory(ctx context.Context, dir string) ([]*Policy, error) {
	ctx, span := l.tracer.Start(ctx, "PolicyLoader.LoadFromDirectory")
	defer span.End()

	l.logger.WithContext(ctx).Debug("Loading policies from directory", logging.Any("dir", dir))

	// 检查目录是否存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, errors.Wrap(err, errors.CodeNotFound,
			fmt.Sprintf("policy directory not found: %s", dir))
	}

	policies := []*Policy{}

	// 遍历目录
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 只处理 YAML 文件
		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		policy, err := l.LoadFromFile(ctx, path)
		if err != nil {
   l.logger.WithContext(ctx).Warn("Failed to load policy file", logging.Any("path", path), logging.Error(err))
			if l.config.FailOnValidationError {
				return err
			}
			return nil
		}

		policies = append(policies, policy)
		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL",
			fmt.Sprintf("failed to load policies from directory: %s", dir))
	}

 l.logger.WithContext(ctx).Info("Policies loaded from directory", logging.Any("dir", dir), logging.Any("count", len(policies)))

	return policies, nil
}

// LoadFromDatabase 从数据库加载策略
func (l *policyLoader) LoadFromDatabase(ctx context.Context, filter *PolicyFilter) ([]*Policy, error) {
	ctx, span := l.tracer.Start(ctx, "PolicyLoader.LoadFromDatabase")
	defer span.End()

	l.logger.WithContext(ctx).Debug("Loading policies from database")

	if l.policyRepo == nil {
		return nil, errors.InternalError("policy repository not configured")
	}

	policies, err := l.policyRepo.List(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL",
			"failed to load policies from database")
	}

	// 设置元数据
	for _, policy := range policies {
		if policy.Metadata == nil {
			policy.Metadata = make(map[string]interface{})
		}
		policy.Metadata["source"] = "database"
		policy.Metadata["loaded_at"] = time.Now()
	}

	l.logger.WithContext(ctx).Info("Policies loaded from database", logging.Any("count", len(policies)))

	return policies, nil
}

// LoadFromConfigCenter 从配置中心加载策略
func (l *policyLoader) LoadFromConfigCenter(ctx context.Context, key string) ([]*Policy, error) {
	ctx, span := l.tracer.Start(ctx, "PolicyLoader.LoadFromConfigCenter")
	defer span.End()

	l.logger.WithContext(ctx).Debug("Loading policy from config center", logging.Any("key", key))

	if l.configCenter == nil {
		return nil, errors.InternalError("config center not configured")
	}

	// 获取配置内容
	content, err := l.configCenter.Get(ctx, key)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL",
			fmt.Sprintf("failed to get policy from config center: %s", key))
	}

	// 解析 YAML
	var policy Policy
	if err := yaml.Unmarshal([]byte(content), &policy); err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidArgument,
			fmt.Sprintf("failed to parse policy from config center: %s", key))
	}

	// 验证策略
	if l.config.ValidateOnLoad {
		if err := l.validatePolicy(&policy); err != nil {
			return nil, errors.Wrap(err, errors.CodeInvalidArgument,
				fmt.Sprintf("policy validation failed: %s", key))
		}
	}

	// 设置元数据
	if policy.Metadata == nil {
		policy.Metadata = make(map[string]interface{})
	}
	policy.Metadata["source"] = "config_center"
	policy.Metadata["config_key"] = key
	policy.Metadata["loaded_at"] = time.Now()

 l.logger.WithContext(ctx).Info("Policy loaded from config center", logging.Any("key", key), logging.Any("policy_id", policy.ID))

	return []*Policy{&policy}, nil
}

// Reload 重新加载策略
func (l *policyLoader) Reload(ctx context.Context) error {
	ctx, span := l.tracer.Start(ctx, "PolicyLoader.Reload")
	defer span.End()

	l.logger.WithContext(ctx).Info("Reloading policies")

	// 加载新策略
	newPolicies, err := l.Load(ctx)
	if err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to reload policies")
	}

	// 更新 PDP 中的策略
	if l.pdp != nil {
		// 先移除旧策略
		l.loadedPolicies.Range(func(key, value interface{}) bool {
			policyID := key.(string)

			// 检查新策略中是否还存在
			found := false
			for _, newPolicy := range newPolicies {
				if newPolicy.ID == policyID {
					found = true
					break
				}
			}

			// 如果不存在，则移除
			if !found {
				if err := l.pdp.RemovePolicy(ctx, policyID); err != nil {
     l.logger.WithContext(ctx).Warn("Failed to remove policy", logging.Any("policy_id", policyID), logging.Error(err))
				} else {
					l.notifyChange(&PolicyChangeEvent{
						Type:      ChangeTypeDeleted,
						PolicyID:  policyID,
						Timestamp: time.Now(),
						Source:    "reload",
					})
				}
			}

			return true
		})

		// 添加或更新新策略
		for _, policy := range newPolicies {
			// 检查是否已存在
			if _, exists := l.loadedPolicies.Load(policy.ID); exists {
				// 更新
				if err := l.pdp.UpdatePolicy(ctx, policy); err != nil {
     l.logger.WithContext(ctx).Warn("Failed to update policy", logging.Any("policy_id", policy.ID), logging.Error(err))
				} else {
					l.notifyChange(&PolicyChangeEvent{
						Type:      ChangeTypeUpdated,
						PolicyID:  policy.ID,
						Policy:    policy,
						Timestamp: time.Now(),
						Source:    "reload",
					})
				}
			} else {
				// 添加
				if err := l.pdp.AddPolicy(ctx, policy); err != nil {
     l.logger.WithContext(ctx).Warn("Failed to add policy", logging.Any("policy_id", policy.ID), logging.Error(err))
				} else {
					l.notifyChange(&PolicyChangeEvent{
						Type:      ChangeTypeAdded,
						PolicyID:  policy.ID,
						Policy:    policy,
						Timestamp: time.Now(),
						Source:    "reload",
					})
				}
			}
		}
	}

	l.logger.WithContext(ctx).Info("Policies reloaded successfully", logging.Any("count", len(newPolicies)))

	l.metricsCollector.IncrementCounter("policy_reload_total",
		map[string]string{"status": "success"})

	return nil
}

// Watch 监听策略变化
func (l *policyLoader) Watch(ctx context.Context) (<-chan *PolicyChangeEvent, error) {
	ctx, span := l.tracer.Start(ctx, "PolicyLoader.Watch")
	defer span.End()

	l.logger.WithContext(ctx).Debug("Starting policy watch")

	eventChan := make(chan *PolicyChangeEvent, 100)

	l.mu.Lock()
	l.changeListeners = append(l.changeListeners, eventChan)
	l.mu.Unlock()

	// 启动文件监听
	if l.config.WatchFiles {
		l.startFileWatchers(ctx)
	}

	// 启动配置中心监听
	if l.config.ConfigCenterEnabled && l.configCenter != nil {
		go l.watchConfigCenter(ctx, eventChan)
	}

	return eventChan, nil
}

// StartAutoReload 启动自动重载
func (l *policyLoader) StartAutoReload(ctx context.Context, interval time.Duration) error {
	ctx, span := l.tracer.Start(ctx, "PolicyLoader.StartAutoReload")
	defer span.End()

	l.logger.WithContext(ctx).Info("Starting auto-reload", logging.Any("interval", interval))

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.autoReloadTicker != nil {
		return errors.New(errors.ConflictError, "auto-reload already started")
	}

	l.autoReloadTicker = time.NewTicker(interval)
	l.autoReloadStop = make(chan struct{})

	go func() {
		for {
			select {
			case <-l.autoReloadTicker.C:
				if err := l.Reload(context.Background()); err != nil {
					l.logger.Error("Auto-reload failed", "error", err)
				}
			case <-l.autoReloadStop:
				l.logger.Info("Auto-reload stopped")
				return
			}
		}
	}()

	return nil
}

// StopAutoReload 停止自动重载
func (l *policyLoader) StopAutoReload() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.autoReloadTicker == nil {
		return errors.New(errors.CodeNotFound, "auto-reload not started")
	}

	l.autoReloadTicker.Stop()
	close(l.autoReloadStop)

	l.autoReloadTicker = nil
	l.autoReloadStop = nil

	l.logger.Info("Auto-reload stopped")

	return nil
}

// Validate 验证策略文件
func (l *policyLoader) Validate(ctx context.Context, path string) error {
	ctx, span := l.tracer.Start(ctx, "PolicyLoader.Validate")
	defer span.End()

	l.logger.WithContext(ctx).Debug("Validating policy file", logging.Any("path", path))

	policy, err := l.LoadFromFile(ctx, path)
	if err != nil {
		return err
	}

	return l.validatePolicy(policy)
}

// Helper methods

func (l *policyLoader) loadFromConfigCenter(ctx context.Context) ([]*Policy, error) {
	allConfigs, err := l.configCenter.GetAll(ctx, l.config.ConfigCenterPrefix)
	if err != nil {
		return nil, err
	}

	policies := []*Policy{}

	for key, content := range allConfigs {
		var policy Policy
		if err := yaml.Unmarshal([]byte(content), &policy); err != nil {
   l.logger.WithContext(ctx).Warn("Failed to parse policy from config center", logging.Any("key", key), logging.Error(err))
			continue
		}

		if l.config.ValidateOnLoad {
			if err := l.validatePolicy(&policy); err != nil {
    l.logger.WithContext(ctx).Warn("Policy validation failed", logging.Any("key", key), logging.Error(err))
				if l.config.FailOnValidationError {
					return nil, err
				}
				continue
			}
		}

		if policy.Metadata == nil {
			policy.Metadata = make(map[string]interface{})
		}
		policy.Metadata["source"] = "config_center"
		policy.Metadata["config_key"] = key
		policy.Metadata["loaded_at"] = time.Now()

		policies = append(policies, &policy)
	}

	return policies, nil
}

func (l *policyLoader) validatePolicy(policy *Policy) error {
	if policy.ID == "" {
		return errors.ValidationError( "policy ID is required")
	}
	if policy.Name == "" {
		return errors.ValidationError( "policy name is required")
	}
	if policy.Effect != EffectPermit && policy.Effect != EffectDeny {
		return errors.ValidationError(
			fmt.Sprintf("invalid policy effect: %s", policy.Effect))
	}
	if policy.Type != PolicyTypeRBAC && policy.Type != PolicyTypeABAC {
		return errors.ValidationError(
			fmt.Sprintf("invalid policy type: %s", policy.Type))
	}

	return nil
}

func (l *policyLoader) startFileWatchers(ctx context.Context) {
	// 监听单个文件
	for _, path := range l.config.FilePaths {
		l.watchFile(ctx, path)
	}

	// 监听目录
	for _, dir := range l.config.DirectoryPaths {
		l.watchDirectory(ctx, dir)
	}
}

func (l *policyLoader) watchFile(ctx context.Context, path string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.fileWatchers[path]; exists {
		return
	}

	watcher := &FileWatcher{
		path:     path,
		stopChan: make(chan struct{}),
	}

	// 获取初始修改时间
	if info, err := os.Stat(path); err == nil {
		watcher.lastModified = info.ModTime()
	}

	l.fileWatchers[path] = watcher

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				info, err := os.Stat(path)
				if err != nil {
					l.logger.WithContext(ctx).Warn("Failed to stat file", logging.Any("path", path), logging.Error(err))
					continue
				}

				watcher.mu.RLock()
				lastModified := watcher.lastModified
				watcher.mu.RUnlock()

				if info.ModTime().After(lastModified) {
					l.logger.WithContext(ctx).Info("Policy file changed, reloading", logging.Any("path", path))

					watcher.mu.Lock()
					watcher.lastModified = info.ModTime()
					watcher.mu.Unlock()

					if err := l.Reload(context.Background()); err != nil {
						l.logger.WithContext(ctx).Error("Failed to reload policies", logging.Error(err))
					}
				}

			case <-watcher.stopChan:
				return
			}
		}
	}()
}

func (l *policyLoader) watchDirectory(ctx context.Context, dir string) {
	// 简化实现：监听整个目录
	l.watchFile(ctx, dir)
}

func (l *policyLoader) watchConfigCenter(ctx context.Context, eventChan chan *PolicyChangeEvent) {
	// 监听配置中心的所有策略配置
	allConfigs, err := l.configCenter.GetAll(ctx, l.config.ConfigCenterPrefix)
	if err != nil {
		l.logger.WithContext(ctx).Error("Failed to get configs from config center", logging.Error(err))
		return
	}

	for key := range allConfigs {
		go func(configKey string) {
			watchChan, err := l.configCenter.Watch(ctx, configKey)
			if err != nil {
				l.logger.WithContext(ctx).Error("Failed to watch config", logging.Any("key", configKey), logging.Error(err))
				return
			}

			for content := range watchChan {
				var policy Policy
				if err := yaml.Unmarshal([]byte(content), &policy); err != nil {
     l.logger.WithContext(ctx).Warn("Failed to parse policy update", logging.Any("key", configKey), logging.Error(err))
					continue
				}

				event := &PolicyChangeEvent{
					Type:      ChangeTypeUpdated,
					PolicyID:  policy.ID,
					Policy:    &policy,
					Timestamp: time.Now(),
					Source:    "config_center",
					Metadata: map[string]interface{}{
						"config_key": configKey,
					},
				}

				eventChan <- event

				// 更新 PDP
				if l.pdp != nil {
					if err := l.pdp.UpdatePolicy(ctx, &policy); err != nil {
      l.logger.WithContext(ctx).Error("Failed to update policy in PDP", logging.Any("policy_id", policy.ID), logging.Error(err))
					}
				}
			}
		}(key)
	}
}

func (l *policyLoader) notifyChange(event *PolicyChangeEvent) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, listener := range l.changeListeners {
		select {
		case listener <- event:
		default:
			l.logger.Warn("Failed to send change event, channel full")
		}
	}
}

// PolicyExporter 策略导出器
type PolicyExporter struct {
	logger logging.Logger
}

// NewPolicyExporter 创建策略导出器
func NewPolicyExporter(logger logging.Logger) *PolicyExporter {
	return &PolicyExporter{logger: logger}
}

// ExportToFile 导出策略到文件
func (e *PolicyExporter) ExportToFile(ctx context.Context, policy *Policy, path string) error {
	data, err := yaml.Marshal(policy)
	if err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to marshal policy")
	}

	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL",
			fmt.Sprintf("failed to write policy file: %s", path))
	}

 e.logger.WithContext(ctx).Info("Policy exported to file", logging.Any("policy_id", policy.ID), logging.Any("path", path))

	return nil
}

// ExportToDirectory 导出多个策略到目录
func (e *PolicyExporter) ExportToDirectory(ctx context.Context, policies []*Policy, dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL",
			fmt.Sprintf("failed to create directory: %s", dir))
	}

	for _, policy := range policies {
		filename := fmt.Sprintf("%s.yaml", policy.ID)
		path := filepath.Join(dir, filename)

		if err := e.ExportToFile(ctx, policy, path); err != nil {
			return err
		}
	}

 e.logger.WithContext(ctx).Info("Policies exported to directory", logging.Any("count", len(policies)))

	return nil
}

// MockConfigCenter 模拟配置中心实现
type MockConfigCenter struct {
	configs  sync.Map
	watchers sync.Map
	mu       sync.RWMutex
}

// NewMockConfigCenter 创建模拟配置中心
func NewMockConfigCenter() ConfigCenter {
	return &MockConfigCenter{}
}

func (m *MockConfigCenter) Get(ctx context.Context, key string) (string, error) {
	if value, ok := m.configs.Load(key); ok {
		return value.(string), nil
	}
	return "", errors.New(errors.CodeNotFound, fmt.Sprintf("config not found: %s", key))
}

func (m *MockConfigCenter) GetAll(ctx context.Context, prefix string) (map[string]string, error) {
	result := make(map[string]string)

	m.configs.Range(func(key, value interface{}) bool {
		keyStr := key.(string)
		if prefix == "" || len(keyStr) >= len(prefix) && keyStr[:len(prefix)] == prefix {
			result[keyStr] = value.(string)
		}
		return true
	})

	return result, nil
}

func (m *MockConfigCenter) Watch(ctx context.Context, key string) (<-chan string, error) {
	watchChan := make(chan string, 10)
	m.watchers.Store(key, watchChan)
	return watchChan, nil
}

func (m *MockConfigCenter) Set(key, value string) {
	m.configs.Store(key, value)

	// 通知监听器
	if watcher, ok := m.watchers.Load(key); ok {
		select {
		case watcher.(chan string) <- value:
		default:
		}
	}
}

func (m *MockConfigCenter) Delete(key string) {
	m.configs.Delete(key)
}

func (m *MockConfigCenter) Close() error {
	// 关闭所有监听通道
	m.watchers.Range(func(key, value interface{}) bool {
		close(value.(chan string))
		return true
	})
	return nil
}

// PolicyTemplate 策略模板
type PolicyTemplate struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Type        PolicyType             `yaml:"type"`
	Variables   map[string]string      `yaml:"variables"`
	Template    string                 `yaml:"template"`
	Metadata    map[string]interface{} `yaml:"metadata"`
}

// TemplateEngine 模板引擎
type TemplateEngine struct {
	templates sync.Map
	logger    logging.Logger
}

// NewTemplateEngine 创建模板引擎
func NewTemplateEngine(logger logging.Logger) *TemplateEngine {
	return &TemplateEngine{
		logger: logger,
	}
}

// LoadTemplate 加载模板
func (t *TemplateEngine) LoadTemplate(ctx context.Context, path string) (*PolicyTemplate, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL",
			fmt.Sprintf("failed to read template file: %s", path))
	}

	var template PolicyTemplate
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidArgument,
			fmt.Sprintf("failed to parse template file: %s", path))
	}

	t.templates.Store(template.Name, &template)

 t.logger.WithContext(ctx).Info("Policy template loaded", logging.Any("name", template.Name), logging.Any("path", path))

	return &template, nil
}

// GeneratePolicy 从模板生成策略
func (t *TemplateEngine) GeneratePolicy(ctx context.Context, templateName string, variables map[string]interface{}) (*Policy, error) {
	templateValue, ok := t.templates.Load(templateName)
	if !ok {
		return nil, errors.New(errors.CodeNotFound,
			fmt.Sprintf("template not found: %s", templateName))
	}

	template := templateValue.(*PolicyTemplate)

	// 简化的变量替换实现
	policyContent := template.Template
	for key, value := range variables {
		placeholder := fmt.Sprintf("${%s}", key)
		policyContent = replaceAll(policyContent, placeholder, fmt.Sprintf("%v", value))
	}

	// 解析生成的策略
	var policy Policy
	if err := yaml.Unmarshal([]byte(policyContent), &policy); err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidArgument,
			"failed to parse generated policy")
	}

	// 添加模板元数据
	if policy.Metadata == nil {
		policy.Metadata = make(map[string]interface{})
	}
	policy.Metadata["generated_from_template"] = templateName
	policy.Metadata["generated_at"] = time.Now()
	policy.Metadata["template_variables"] = variables

 t.logger.WithContext(ctx).Info("Policy generated from template", logging.Any("template", templateName), logging.Any("policy_id", policy.ID))

	return &policy, nil
}

// PolicyConverter 策略转换器
type PolicyConverter struct {
	logger logging.Logger
}

// NewPolicyConverter 创建策略转换器
func NewPolicyConverter(logger logging.Logger) *PolicyConverter {
	return &PolicyConverter{logger: logger}
}

// ConvertToXACML 转换为 XACML 格式
func (c *PolicyConverter) ConvertToXACML(ctx context.Context, policy *Policy) (string, error) {
	// 简化的 XACML 转换实现
	xacml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Policy xmlns="urn:oasis:names:tc:xacml:3.0:core:schema:wd-17"
        PolicyId="%s"
        RuleCombiningAlgId="urn:oasis:names:tc:xacml:3.0:rule-combining-algorithm:deny-overrides"
        Version="%s">
    <Description>%s</Description>
    <Target>
        <!-- Target definition -->
    </Target>
    <Rule RuleId="%s_rule" Effect="%s">
        <!-- Rule definition -->
    </Rule>
</Policy>`, policy.ID, policy.Version, policy.Description, policy.ID, policy.Effect)

	c.logger.WithContext(ctx).Info("Policy converted to XACML", logging.Any("policy_id", policy.ID))

	return xacml, nil
}

// ConvertFromXACML 从 XACML 格式转换
func (c *PolicyConverter) ConvertFromXACML(ctx context.Context, xacml string) (*Policy, error) {
	// 简化实现：实际应该解析 XML
	policy := &Policy{
		ID:          fmt.Sprintf("policy_%d", time.Now().UnixNano()),
		Name:        "Converted from XACML",
		Description: "Policy converted from XACML format",
		Type:        PolicyTypeABAC,
		Effect:      EffectPermit,
		Enabled:     true,
		Version:     "1.0",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"converted_from": "xacml",
		},
	}

	c.logger.WithContext(ctx).Info("Policy converted from XACML", logging.Any("policy_id", policy.ID))

	return policy, nil
}

// PolicyMigrator 策略迁移器
type PolicyMigrator struct {
	sourcePDP PolicyDecisionPoint
	targetPDP PolicyDecisionPoint
	logger    logging.Logger
}

// NewPolicyMigrator 创建策略迁移器
func NewPolicyMigrator(
	sourcePDP PolicyDecisionPoint,
	targetPDP PolicyDecisionPoint,
	logger logging.Logger,
) *PolicyMigrator {
	return &PolicyMigrator{
		sourcePDP: sourcePDP,
		targetPDP: targetPDP,
		logger:    logger,
	}
}

// Migrate 迁移策略
func (m *PolicyMigrator) Migrate(ctx context.Context, filter *PolicyFilter) error {
	m.logger.WithContext(ctx).Info("Starting policy migration")

	// 从源 PDP 获取策略
	policies, err := m.sourcePDP.ListPolicies(ctx, filter)
	if err != nil {
		return errors.Wrap(err, "ERR_INTERNAL",
			"failed to list policies from source")
	}

	// 迁移到目标 PDP
	migratedCount := 0
	for _, policy := range policies {
		if err := m.targetPDP.AddPolicy(ctx, policy); err != nil {
   m.logger.WithContext(ctx).Warn("Failed to migrate policy", logging.Any("policy_id", policy.ID), logging.Error(err))
			continue
		}
		migratedCount++
	}

 m.logger.WithContext(ctx).Info("Policy migration completed", logging.Any("total", len(policies)))

	return nil
}

// PolicyBackup 策略备份
type PolicyBackup struct {
	Timestamp time.Time              `yaml:"timestamp"`
	Version   string                 `yaml:"version"`
	Policies  []*Policy              `yaml:"policies"`
	Metadata  map[string]interface{} `yaml:"metadata"`
}

// PolicyBackupManager 策略备份管理器
type PolicyBackupManager struct {
	backupDir string
	logger    logging.Logger
}

// NewPolicyBackupManager 创建策略备份管理器
func NewPolicyBackupManager(backupDir string, logger logging.Logger) *PolicyBackupManager {
	return &PolicyBackupManager{
		backupDir: backupDir,
		logger:    logger,
	}
}

// Backup 备份策略
func (b *PolicyBackupManager) Backup(ctx context.Context, policies []*Policy) (string, error) {
	if err := os.MkdirAll(b.backupDir, 0755); err != nil {
		return "", errors.Wrap(err, "ERR_INTERNAL",
			"failed to create backup directory")
	}

	backup := &PolicyBackup{
		Timestamp: time.Now(),
		Version:   "1.0",
		Policies:  policies,
		Metadata: map[string]interface{}{
			"count": len(policies),
		},
	}

	data, err := yaml.Marshal(backup)
	if err != nil {
		return "", errors.Wrap(err, "ERR_INTERNAL",
			"failed to marshal backup")
	}

	filename := fmt.Sprintf("policy_backup_%s.yaml",
		time.Now().Format("20060102_150405"))
	path := filepath.Join(b.backupDir, filename)

	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return "", errors.Wrap(err, "ERR_INTERNAL",
			"failed to write backup file")
	}

 b.logger.WithContext(ctx).Info("Policy backup created", logging.Any("path", path), logging.Any("count", len(policies)))

	return path, nil
}

// Restore 恢复策略
func (b *PolicyBackupManager) Restore(ctx context.Context, backupPath string) ([]*Policy, error) {
	data, err := ioutil.ReadFile(backupPath)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL",
			"failed to read backup file")
	}

	var backup PolicyBackup
	if err := yaml.Unmarshal(data, &backup); err != nil {
		return nil, errors.Wrap(err, errors.CodeInvalidArgument,
			"failed to parse backup file")
	}

 b.logger.WithContext(ctx).Info("Policy backup restored", logging.Any("path", backupPath), logging.Any("count", len(backup.Policies)))

	return backup.Policies, nil
}

// ListBackups 列出备份
func (b *PolicyBackupManager) ListBackups(ctx context.Context) ([]string, error) {
	files, err := ioutil.ReadDir(b.backupDir)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL",
			"failed to read backup directory")
	}

	backups := []string{}
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".yaml" {
			backups = append(backups, filepath.Join(b.backupDir, file.Name()))
		}
	}

	return backups, nil
}

// Utility functions

func replaceAll(s, old, new string) string {
	result := s
	for {
		replaced := replace(result, old, new)
		if replaced == result {
			break
		}
		result = replaced
	}
	return result
}

func replace(s, old, new string) string {
	if len(old) == 0 {
		return s
	}

	index := indexOf(s, old)
	if index == -1 {
		return s
	}

	return s[:index] + new + s[index+len(old):]
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

//Personal.AI order the ending
