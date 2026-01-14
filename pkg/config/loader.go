// Package config provides configuration loading and management for OpenEAAP.
// It supports loading from YAML files, environment variables, and command-line
// arguments, with hot-reload capabilities using Viper.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// ============================================================================
// Configuration Loader
// ============================================================================

// Loader manages configuration loading and reloading
type Loader struct {
	// Viper instance
	viper *viper.Viper

	// Current configuration
	config *Config
	mu     sync.RWMutex

	// Configuration file path
	configFile string

	// Watch for changes
	watchEnabled bool

	// Reload callbacks
	reloadCallbacks []ReloadCallback

	// Logger (optional, can be set after initialization)
	logger Logger
}

// ReloadCallback is called when configuration is reloaded
type ReloadCallback func(oldConfig, newConfig *Config) error

// Logger interface for configuration loader logging
type Logger interface {
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// LoaderOptions defines options for configuration loader
type LoaderOptions struct {
	// Configuration file path
	ConfigFile string

	// Configuration file type (yaml, json, toml)
	ConfigType string

	// Enable watching for file changes
	EnableWatch bool

	// Environment variable prefix
	EnvPrefix string

	// Additional config paths to search
	ConfigPaths []string
}

// ============================================================================
// Loader Creation and Initialization
// ============================================================================

// NewLoader creates a new configuration loader
func NewLoader(opts LoaderOptions) (*Loader, error) {
	v := viper.New()

	// Set configuration file
	if opts.ConfigFile != "" {
		v.SetConfigFile(opts.ConfigFile)
	} else {
		// Set default configuration name and type
		v.SetConfigName("config")
		v.SetConfigType("yaml")

		// Add default config paths
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("/etc/openeaap")

		// Add additional config paths
		for _, path := range opts.ConfigPaths {
			v.AddConfigPath(path)
		}
	}

	// Configure environment variables
	envPrefix := opts.EnvPrefix
	if envPrefix == "" {
		envPrefix = "OPENEAAP"
	}
	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	loader := &Loader{
		viper:        v,
		configFile:   opts.ConfigFile,
		watchEnabled: opts.EnableWatch,
	}

	return loader, nil
}

// Load loads configuration from all sources
func (l *Loader) Load() (*Config, error) {
	// Read configuration file
	if err := l.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, use defaults
			l.logWarn("Configuration file not found, using defaults", "error", err)
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal configuration
	config := &Config{}
	if err := l.viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply defaults
	l.applyDefaults(config)

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Store configuration
	l.mu.Lock()
	l.config = config
	l.mu.Unlock()

	l.logInfo("Configuration loaded successfully", "file", l.viper.ConfigFileUsed())

	// Start watching if enabled
	if l.watchEnabled {
		l.startWatch()
	}

	return config, nil
}

// Get returns the current configuration
func (l *Loader) Get() *Config {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config
}

// ============================================================================
// Configuration Defaults
// ============================================================================

// applyDefaults applies default values to configuration
func (l *Loader) applyDefaults(config *Config) {
	// Server defaults
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Server.Environment == "" {
		config.Server.Environment = "development"
	}
	if config.Server.ReadTimeout == 0 {
		config.Server.ReadTimeout = 30 * time.Second
	}
	if config.Server.WriteTimeout == 0 {
		config.Server.WriteTimeout = 30 * time.Second
	}
	if config.Server.IdleTimeout == 0 {
		config.Server.IdleTimeout = 120 * time.Second
	}
	if config.Server.MaxHeaderBytes == 0 {
		config.Server.MaxHeaderBytes = 1 << 20 // 1 MB
	}
	if config.Server.ShutdownTimeout == 0 {
		config.Server.ShutdownTimeout = 30 * time.Second
	}

	// Database defaults
	if config.Database.Port == 0 {
		config.Database.Port = 5432
	}
	if config.Database.SSLMode == "" {
		config.Database.SSLMode = "disable"
	}
	if config.Database.MaxOpenConns == 0 {
		config.Database.MaxOpenConns = 25
	}
	if config.Database.MaxIdleConns == 0 {
		config.Database.MaxIdleConns = 10
	}
	if config.Database.ConnMaxLifetime == 0 {
		config.Database.ConnMaxLifetime = 5 * time.Minute
	}
	if config.Database.ConnMaxIdleTime == 0 {
		config.Database.ConnMaxIdleTime = 10 * time.Minute
	}
	if config.Database.LogMode == "" {
		config.Database.LogMode = "error"
	}

	// Redis defaults
	if config.Redis.Port == 0 {
		config.Redis.Port = 6379
	}
	if config.Redis.PoolSize == 0 {
		config.Redis.PoolSize = 10
	}
	if config.Redis.MinIdleConns == 0 {
		config.Redis.MinIdleConns = 5
	}
	if config.Redis.DialTimeout == 0 {
		config.Redis.DialTimeout = 5 * time.Second
	}
	if config.Redis.ReadTimeout == 0 {
		config.Redis.ReadTimeout = 3 * time.Second
	}
	if config.Redis.WriteTimeout == 0 {
		config.Redis.WriteTimeout = 3 * time.Second
	}
	if config.Redis.ConnMaxIdleTime == 0 {
		config.Redis.ConnMaxIdleTime = 5 * time.Minute
	}
	if config.Redis.DefaultTTL == 0 {
		config.Redis.DefaultTTL = 1 * time.Hour
	}

	// Milvus defaults
	if config.Milvus.Port == 0 {
		config.Milvus.Port = 19530
	}
	if config.Milvus.Database == "" {
		config.Milvus.Database = "default"
	}
	if config.Milvus.DefaultCollection == "" {
		config.Milvus.DefaultCollection = "embeddings"
	}
	if config.Milvus.Dimension == 0 {
		config.Milvus.Dimension = 1536
	}
	if config.Milvus.IndexType == "" {
		config.Milvus.IndexType = "HNSW"
	}
	if config.Milvus.MetricType == "" {
		config.Milvus.MetricType = "COSINE"
	}
	if config.Milvus.Timeout == 0 {
		config.Milvus.Timeout = 30 * time.Second
	}

	// Kafka defaults
	if config.Kafka.ClientID == "" {
		config.Kafka.ClientID = "openeaap"
	}
	if config.Kafka.Producer.Compression == "" {
		config.Kafka.Producer.Compression = "snappy"
	}
	if config.Kafka.Producer.MaxMessageBytes == 0 {
		config.Kafka.Producer.MaxMessageBytes = 1000000
	}
	if config.Kafka.Producer.RequiredAcks == 0 {
		config.Kafka.Producer.RequiredAcks = 1
	}
	if config.Kafka.Consumer.GroupID == "" {
		config.Kafka.Consumer.GroupID = "openeaap-consumer"
	}
	if config.Kafka.Consumer.InitialOffset == "" {
		config.Kafka.Consumer.InitialOffset = "newest"
	}
	if config.Kafka.Consumer.SessionTimeout == 0 {
		config.Kafka.Consumer.SessionTimeout = 10 * time.Second
	}

	// Topic defaults
	if config.Kafka.Topics.AgentEvents == "" {
		config.Kafka.Topics.AgentEvents = "agent.events"
	}
	if config.Kafka.Topics.WorkflowEvents == "" {
		config.Kafka.Topics.WorkflowEvents = "workflow.events"
	}
	if config.Kafka.Topics.FeedbackEvents == "" {
		config.Kafka.Topics.FeedbackEvents = "feedback.events"
	}
	if config.Kafka.Topics.AuditLogs == "" {
		config.Kafka.Topics.AuditLogs = "audit.logs"
	}

	// Observability defaults
	if config.Observability.Logging.Level == "" {
		config.Observability.Logging.Level = "info"
	}
	if config.Observability.Logging.Format == "" {
		config.Observability.Logging.Format = "json"
	}
	if config.Observability.Logging.Output == "" {
		config.Observability.Logging.Output = "stdout"
	}
	if config.Observability.Metrics.Port == 0 {
		config.Observability.Metrics.Port = 9090
	}
	if config.Observability.Metrics.Path == "" {
		config.Observability.Metrics.Path = "/metrics"
	}
	if config.Observability.Metrics.Prometheus.Namespace == "" {
		config.Observability.Metrics.Prometheus.Namespace = "openeaap"
	}
	if config.Observability.Tracing.ServiceName == "" {
		config.Observability.Tracing.ServiceName = "openeaap"
	}
	if config.Observability.Tracing.SamplingRate == 0 {
		config.Observability.Tracing.SamplingRate = 0.1
	}

	// Security defaults
	if config.Security.JWT.ExpirationTime == 0 {
		config.Security.JWT.ExpirationTime = 24 * time.Hour
	}
	if config.Security.JWT.RefreshExpirationTime == 0 {
		config.Security.JWT.RefreshExpirationTime = 7 * 24 * time.Hour
	}
	if config.Security.JWT.Issuer == "" {
		config.Security.JWT.Issuer = "openeaap"
	}
	if config.Security.RateLimit.RequestsPerSecond == 0 {
		config.Security.RateLimit.RequestsPerSecond = 100
	}
	if config.Security.RateLimit.Burst == 0 {
		config.Security.RateLimit.Burst = 200
	}
	if config.Security.RateLimit.LimitBy == "" {
		config.Security.RateLimit.LimitBy = "ip"
	}

	// Storage defaults
	if config.Storage.Provider == "" {
		config.Storage.Provider = "local"
	}
	if config.Storage.Local.BasePath == "" {
		config.Storage.Local.BasePath = "./storage"
	}
	if config.Storage.Local.MaxFileSize == 0 {
		config.Storage.Local.MaxFileSize = 100 // 100 MB
	}

	// Model defaults
	if config.Model.DefaultProvider == "" {
		config.Model.DefaultProvider = "openai"
	}
	if config.Model.OpenAI.DefaultModel == "" {
		config.Model.OpenAI.DefaultModel = "gpt-4"
	}
	if config.Model.Embedding.Provider == "" {
		config.Model.Embedding.Provider = "openai"
	}
	if config.Model.Embedding.Model == "" {
		config.Model.Embedding.Model = "text-embedding-3-small"
	}
	if config.Model.Embedding.Dimension == 0 {
		config.Model.Embedding.Dimension = 1536
	}
	if config.Model.Timeout == 0 {
		config.Model.Timeout = 60 * time.Second
	}
	if config.Model.MaxRetries == 0 {
		config.Model.MaxRetries = 3
	}
}

// ============================================================================
// Hot Reload Support
// ============================================================================

// startWatch starts watching the configuration file for changes
func (l *Loader) startWatch() {
	l.viper.WatchConfig()
	l.viper.OnConfigChange(func(e fsnotify.Event) {
		l.logInfo("Configuration file changed, reloading", "file", e.Name)

		if err := l.reload(); err != nil {
			l.logError("Failed to reload configuration", "error", err)
		}
	})
}

// reload reloads the configuration
func (l *Loader) reload() error {
	// Load old config
	l.mu.RLock()
	oldConfig := l.config
	l.mu.RUnlock()

	// Unmarshal new configuration
	newConfig := &Config{}
	if err := l.viper.Unmarshal(newConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply defaults
	l.applyDefaults(newConfig)

	// Validate new configuration
	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("new configuration validation failed: %w", err)
	}

	// Execute reload callbacks
	for _, callback := range l.reloadCallbacks {
		if err := callback(oldConfig, newConfig); err != nil {
			return fmt.Errorf("reload callback failed: %w", err)
		}
	}

	// Update configuration
	l.mu.Lock()
	l.config = newConfig
	l.mu.Unlock()

	l.logInfo("Configuration reloaded successfully")

	return nil
}

// OnReload registers a callback to be called when configuration is reloaded
func (l *Loader) OnReload(callback ReloadCallback) {
	l.reloadCallbacks = append(l.reloadCallbacks, callback)
}

// ============================================================================
// Environment-Specific Loading
// ============================================================================

// LoadFromEnvironment loads configuration for specific environment
func LoadFromEnvironment(env string) (*Config, error) {
	configFile := fmt.Sprintf("config.%s.yaml", env)

	opts := LoaderOptions{
		ConfigFile:  configFile,
		ConfigType:  "yaml",
		EnableWatch: false,
		EnvPrefix:   "OPENEAAP",
	}

	loader, err := NewLoader(opts)
	if err != nil {
		return nil, err
	}

	return loader.Load()
}

// LoadWithDefaults loads configuration with default options
func LoadWithDefaults() (*Config, error) {
	opts := LoaderOptions{
		ConfigType:  "yaml",
		EnableWatch: false,
		EnvPrefix:   "OPENEAAP",
		ConfigPaths: []string{".", "./config", "/etc/openeaap"},
	}

	loader, err := NewLoader(opts)
	if err != nil {
		return nil, err
	}

	return loader.Load()
}

// ============================================================================
// Configuration Export
// ============================================================================

// SaveToFile saves current configuration to file
func (l *Loader) SaveToFile(filepath string) error {
	l.mu.RLock()
	config := l.config
	l.mu.RUnlock()

	return l.viper.WriteConfigAs(filepath)
}

// ExportToYAML exports configuration to YAML string
func (l *Loader) ExportToYAML() (string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.viper.AllSettings(), nil
}

// ============================================================================
// Logger Methods
// ============================================================================

// SetLogger sets the logger for configuration loader
func (l *Loader) SetLogger(logger Logger) {
	l.logger = logger
}

func (l *Loader) logInfo(msg string, fields ...interface{}) {
	if l.logger != nil {
		l.logger.Info(msg, fields...)
	}
}

func (l *Loader) logWarn(msg string, fields ...interface{}) {
	if l.logger != nil {
		l.logger.Warn(msg, fields...)
	}
}

func (l *Loader) logError(msg string, fields ...interface{}) {
	if l.logger != nil {
		l.logger.Error(msg, fields...)
	}
}

// ============================================================================
// Utility Functions
// ============================================================================

// GetConfigPath returns the path to configuration file
func GetConfigPath(filename string) (string, error) {
	// Check current directory
	if _, err := os.Stat(filename); err == nil {
		return filepath.Abs(filename)
	}

	// Check ./config directory
	configPath := filepath.Join("config", filename)
	if _, err := os.Stat(configPath); err == nil {
		return filepath.Abs(configPath)
	}

	// Check /etc/openeaap directory
	etcPath := filepath.Join("/etc/openeaap", filename)
	if _, err := os.Stat(etcPath); err == nil {
		return etcPath, nil
	}

	return "", fmt.Errorf("configuration file not found: %s", filename)
}

// MustLoad loads configuration and panics on error
func MustLoad() *Config {
	config, err := LoadWithDefaults()
	if err != nil {
		panic(fmt.Sprintf("failed to load configuration: %v", err))
	}
	return config
}

//Personal.AI order the ending
