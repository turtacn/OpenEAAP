// Package config provides centralized configuration management for OpenEAAP.
// It defines configuration structures for all components and supports
// validation, default values, and environment-based configuration loading.
package config

import (
	"fmt"
	"time"
)

// ============================================================================
// Main Configuration Structure
// ============================================================================

// Config represents the complete application configuration
type Config struct {
	// Server configuration
	Server ServerConfig `mapstructure:"server" yaml:"server" json:"server"`

	// Database configuration
	Database DatabaseConfig `mapstructure:"database" yaml:"database" json:"database"`

	// Redis configuration
	Redis RedisConfig `mapstructure:"redis" yaml:"redis" json:"redis"`

	// Milvus vector database configuration
	Milvus MilvusConfig `mapstructure:"milvus" yaml:"milvus" json:"milvus"`

	// Kafka message queue configuration
	Kafka KafkaConfig `mapstructure:"kafka" yaml:"kafka" json:"kafka"`

	// Observability configuration
	Observability ObservabilityConfig `mapstructure:"observability" yaml:"observability" json:"observability"`

	// Security configuration
	Security SecurityConfig `mapstructure:"security" yaml:"security" json:"security"`

	// Storage configuration
	Storage StorageConfig `mapstructure:"storage" yaml:"storage" json:"storage"`

	// Model configuration
	Model ModelConfig `mapstructure:"model" yaml:"model" json:"model"`

	// Feature flags
	Features FeatureConfig `mapstructure:"features" yaml:"features" json:"features"`
}

// ============================================================================
// Server Configuration
// ============================================================================

// ServerConfig defines HTTP server configuration
type ServerConfig struct {
	// Host to bind to
	Host string `mapstructure:"host" yaml:"host" json:"host"`

	// Port to listen on
	Port int `mapstructure:"port" yaml:"port" json:"port"`

	// Environment (development, staging, production)
	Environment string `mapstructure:"environment" yaml:"environment" json:"environment"`

	// Read timeout
	ReadTimeout time.Duration `mapstructure:"read_timeout" yaml:"read_timeout" json:"read_timeout"`

	// Write timeout
	WriteTimeout time.Duration `mapstructure:"write_timeout" yaml:"write_timeout" json:"write_timeout"`

	// Idle timeout
	IdleTimeout time.Duration `mapstructure:"idle_timeout" yaml:"idle_timeout" json:"idle_timeout"`

	// Max header bytes
	MaxHeaderBytes int `mapstructure:"max_header_bytes" yaml:"max_header_bytes" json:"max_header_bytes"`

	// Enable CORS
	EnableCORS bool `mapstructure:"enable_cors" yaml:"enable_cors" json:"enable_cors"`

	// CORS allowed origins
	CORSAllowedOrigins []string `mapstructure:"cors_allowed_origins" yaml:"cors_allowed_origins" json:"cors_allowed_origins"`

	// Enable request logging
	EnableRequestLogging bool `mapstructure:"enable_request_logging" yaml:"enable_request_logging" json:"enable_request_logging"`

	// Enable metrics
	EnableMetrics bool `mapstructure:"enable_metrics" yaml:"enable_metrics" json:"enable_metrics"`

	// Graceful shutdown timeout
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" yaml:"shutdown_timeout" json:"shutdown_timeout"`

	// TLS configuration
	TLS TLSConfig `mapstructure:"tls" yaml:"tls" json:"tls"`
}

// TLSConfig defines TLS/SSL configuration
type TLSConfig struct {
	// Enable TLS
	Enabled bool `mapstructure:"enabled" yaml:"enabled" json:"enabled"`

	// Certificate file path
	CertFile string `mapstructure:"cert_file" yaml:"cert_file" json:"cert_file"`

	// Private key file path
	KeyFile string `mapstructure:"key_file" yaml:"key_file" json:"key_file"`

	// Minimum TLS version (1.2, 1.3)
	MinVersion string `mapstructure:"min_version" yaml:"min_version" json:"min_version"`
}

// ============================================================================
// Database Configuration
// ============================================================================

// DatabaseConfig defines PostgreSQL database configuration
type DatabaseConfig struct {
	// Host address
	Host string `mapstructure:"host" yaml:"host" json:"host"`

	// Port number
	Port int `mapstructure:"port" yaml:"port" json:"port"`

	// Database name
	Database string `mapstructure:"database" yaml:"database" json:"database"`

	// Username
	Username string `mapstructure:"username" yaml:"username" json:"username"`

	// Password
	Password string `mapstructure:"password" yaml:"password" json:"password"`

	// SSL mode (disable, require, verify-ca, verify-full)
	SSLMode string `mapstructure:"ssl_mode" yaml:"ssl_mode" json:"ssl_mode"`

	// Maximum open connections
	MaxOpenConns int `mapstructure:"max_open_conns" yaml:"max_open_conns" json:"max_open_conns"`

	// Maximum idle connections
	MaxIdleConns int `mapstructure:"max_idle_conns" yaml:"max_idle_conns" json:"max_idle_conns"`

	// Connection max lifetime
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" yaml:"conn_max_lifetime" json:"conn_max_lifetime"`

	// Connection max idle time
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" yaml:"conn_max_idle_time" json:"conn_max_idle_time"`

	// Enable auto migration
	AutoMigrate bool `mapstructure:"auto_migrate" yaml:"auto_migrate" json:"auto_migrate"`

	// Log mode (silent, error, warn, info)
	LogMode string `mapstructure:"log_mode" yaml:"log_mode" json:"log_mode"`
}

// ============================================================================
// Redis Configuration
// ============================================================================

// RedisConfig defines Redis cache configuration
type RedisConfig struct {
	// Host address
	Host string `mapstructure:"host" yaml:"host" json:"host"`

	// Port number
	Port int `mapstructure:"port" yaml:"port" json:"port"`

	// Password
	Password string `mapstructure:"password" yaml:"password" json:"password"`

	// Database number
	DB int `mapstructure:"db" yaml:"db" json:"db"`

	// Pool size
	PoolSize int `mapstructure:"pool_size" yaml:"pool_size" json:"pool_size"`

	// Minimum idle connections
	MinIdleConns int `mapstructure:"min_idle_conns" yaml:"min_idle_conns" json:"min_idle_conns"`

	// Dial timeout
	DialTimeout time.Duration `mapstructure:"dial_timeout" yaml:"dial_timeout" json:"dial_timeout"`

	// Read timeout
	ReadTimeout time.Duration `mapstructure:"read_timeout" yaml:"read_timeout" json:"read_timeout"`

	// Write timeout
	WriteTimeout time.Duration `mapstructure:"write_timeout" yaml:"write_timeout" json:"write_timeout"`

	// Connection max idle time
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" yaml:"conn_max_idle_time" json:"conn_max_idle_time"`

	// Enable TLS
	EnableTLS bool `mapstructure:"enable_tls" yaml:"enable_tls" json:"enable_tls"`

	// Default TTL for cache entries
	DefaultTTL time.Duration `mapstructure:"default_ttl" yaml:"default_ttl" json:"default_ttl"`
}

// ============================================================================
// Milvus Configuration
// ============================================================================

// MilvusConfig defines Milvus vector database configuration
type MilvusConfig struct {
	// Host address
	Host string `mapstructure:"host" yaml:"host" json:"host"`

	// Port number
	Port int `mapstructure:"port" yaml:"port" json:"port"`

	// Username
	Username string `mapstructure:"username" yaml:"username" json:"username"`

	// Password
	Password string `mapstructure:"password" yaml:"password" json:"password"`

	// Database name
	Database string `mapstructure:"database" yaml:"database" json:"database"`

	// Default collection name
	DefaultCollection string `mapstructure:"default_collection" yaml:"default_collection" json:"default_collection"`

	// Vector dimension
	Dimension int `mapstructure:"dimension" yaml:"dimension" json:"dimension"`

	// Index type (IVF_FLAT, IVF_SQ8, IVF_PQ, HNSW, etc.)
	IndexType string `mapstructure:"index_type" yaml:"index_type" json:"index_type"`

	// Metric type (L2, IP, COSINE)
	MetricType string `mapstructure:"metric_type" yaml:"metric_type" json:"metric_type"`

	// Search parameters
	SearchParams map[string]interface{} `mapstructure:"search_params" yaml:"search_params" json:"search_params"`

	// Connection timeout
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" json:"timeout"`
}

// ============================================================================
// Kafka Configuration
// ============================================================================

// KafkaConfig defines Kafka message queue configuration
type KafkaConfig struct {
	// Broker addresses
	Brokers []string `mapstructure:"brokers" yaml:"brokers" json:"brokers"`

	// Client ID
	ClientID string `mapstructure:"client_id" yaml:"client_id" json:"client_id"`

	// SASL configuration
	SASL SASLConfig `mapstructure:"sasl" yaml:"sasl" json:"sasl"`

	// Producer configuration
	Producer KafkaProducerConfig `mapstructure:"producer" yaml:"producer" json:"producer"`

	// Consumer configuration
	Consumer KafkaConsumerConfig `mapstructure:"consumer" yaml:"consumer" json:"consumer"`

	// Topic configuration
	Topics KafkaTopicConfig `mapstructure:"topics" yaml:"topics" json:"topics"`
}

// SASLConfig defines SASL authentication configuration
type SASLConfig struct {
	// Enable SASL
	Enabled bool `mapstructure:"enabled" yaml:"enabled" json:"enabled"`

	// Mechanism (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512)
	Mechanism string `mapstructure:"mechanism" yaml:"mechanism" json:"mechanism"`

	// Username
	Username string `mapstructure:"username" yaml:"username" json:"username"`

	// Password
	Password string `mapstructure:"password" yaml:"password" json:"password"`
}

// KafkaProducerConfig defines Kafka producer configuration
type KafkaProducerConfig struct {
	// Compression type (none, gzip, snappy, lz4, zstd)
	Compression string `mapstructure:"compression" yaml:"compression" json:"compression"`

	// Max message bytes
	MaxMessageBytes int `mapstructure:"max_message_bytes" yaml:"max_message_bytes" json:"max_message_bytes"`

	// Required acks (-1, 0, 1)
	RequiredAcks int `mapstructure:"required_acks" yaml:"required_acks" json:"required_acks"`

	// Idempotent
	Idempotent bool `mapstructure:"idempotent" yaml:"idempotent" json:"idempotent"`
}

// KafkaConsumerConfig defines Kafka consumer configuration
type KafkaConsumerConfig struct {
	// Group ID
	GroupID string `mapstructure:"group_id" yaml:"group_id" json:"group_id"`

	// Initial offset (oldest, newest)
	InitialOffset string `mapstructure:"initial_offset" yaml:"initial_offset" json:"initial_offset"`

	// Session timeout
	SessionTimeout time.Duration `mapstructure:"session_timeout" yaml:"session_timeout" json:"session_timeout"`

	// Enable auto commit
	EnableAutoCommit bool `mapstructure:"enable_auto_commit" yaml:"enable_auto_commit" json:"enable_auto_commit"`
}

// KafkaTopicConfig defines Kafka topic names
type KafkaTopicConfig struct {
	// Agent events topic
	AgentEvents string `mapstructure:"agent_events" yaml:"agent_events" json:"agent_events"`

	// Workflow events topic
	WorkflowEvents string `mapstructure:"workflow_events" yaml:"workflow_events" json:"workflow_events"`

	// Feedback events topic
	FeedbackEvents string `mapstructure:"feedback_events" yaml:"feedback_events" json:"feedback_events"`

	// Audit logs topic
	AuditLogs string `mapstructure:"audit_logs" yaml:"audit_logs" json:"audit_logs"`
}

// ============================================================================
// Observability Configuration
// ============================================================================

// ObservabilityConfig defines observability configuration
type ObservabilityConfig struct {
	// Logging configuration
	Logging LoggingConfig `mapstructure:"logging" yaml:"logging" json:"logging"`

	// Metrics configuration
	Metrics MetricsConfig `mapstructure:"metrics" yaml:"metrics" json:"metrics"`

	// Tracing configuration
	Tracing TracingConfig `mapstructure:"tracing" yaml:"tracing" json:"tracing"`
}

// LoggingConfig defines logging configuration
type LoggingConfig struct {
	// Log level (debug, info, warn, error, fatal)
	Level string `mapstructure:"level" yaml:"level" json:"level"`

	// Log format (json, text)
	Format string `mapstructure:"format" yaml:"format" json:"format"`

	// Output (stdout, stderr, file)
	Output string `mapstructure:"output" yaml:"output" json:"output"`

	// Log file path (if output is file)
	FilePath string `mapstructure:"file_path" yaml:"file_path" json:"file_path"`

	// Max file size in MB
	MaxSize int `mapstructure:"max_size" yaml:"max_size" json:"max_size"`

	// Max backup files
	MaxBackups int `mapstructure:"max_backups" yaml:"max_backups" json:"max_backups"`

	// Max age in days
	MaxAge int `mapstructure:"max_age" yaml:"max_age" json:"max_age"`

	// Enable compression
	Compress bool `mapstructure:"compress" yaml:"compress" json:"compress"`
}

// MetricsConfig defines metrics configuration
type MetricsConfig struct {
	// Enable metrics collection
	Enabled bool `mapstructure:"enabled" yaml:"enabled" json:"enabled"`

	// Metrics port
	Port int `mapstructure:"port" yaml:"port" json:"port"`

	// Metrics path
	Path string `mapstructure:"path" yaml:"path" json:"path"`

	// Prometheus configuration
	Prometheus PrometheusConfig `mapstructure:"prometheus" yaml:"prometheus" json:"prometheus"`
}

// PrometheusConfig defines Prometheus-specific configuration
type PrometheusConfig struct {
	// Namespace for metrics
	Namespace string `mapstructure:"namespace" yaml:"namespace" json:"namespace"`

	// Subsystem for metrics
	Subsystem string `mapstructure:"subsystem" yaml:"subsystem" json:"subsystem"`

	// Enable histogram metrics
	EnableHistogram bool `mapstructure:"enable_histogram" yaml:"enable_histogram" json:"enable_histogram"`
}

// TracingConfig defines distributed tracing configuration
type TracingConfig struct {
	// Enable tracing
	Enabled bool `mapstructure:"enabled" yaml:"enabled" json:"enabled"`

	// Provider (jaeger, zipkin, otlp)
	Provider string `mapstructure:"provider" yaml:"provider" json:"provider"`

	// Endpoint
	Endpoint string `mapstructure:"endpoint" yaml:"endpoint" json:"endpoint"`

	// Service name
	ServiceName string `mapstructure:"service_name" yaml:"service_name" json:"service_name"`

	// Sampling rate (0.0 - 1.0)
	SamplingRate float64 `mapstructure:"sampling_rate" yaml:"sampling_rate" json:"sampling_rate"`
}

// ============================================================================
// Security Configuration
// ============================================================================

// SecurityConfig defines security configuration
type SecurityConfig struct {
	// JWT configuration
	JWT JWTConfig `mapstructure:"jwt" yaml:"jwt" json:"jwt"`

	// Encryption configuration
	Encryption EncryptionConfig `mapstructure:"encryption" yaml:"encryption" json:"encryption"`

	// Rate limiting configuration
	RateLimit RateLimitConfig `mapstructure:"rate_limit" yaml:"rate_limit" json:"rate_limit"`

	// RBAC configuration
	RBAC RBACConfig `mapstructure:"rbac" yaml:"rbac" json:"rbac"`
}

// JWTConfig defines JWT authentication configuration
type JWTConfig struct {
	// Secret key for signing
	Secret string `mapstructure:"secret" yaml:"secret" json:"secret"`

	// Token expiration time
	ExpirationTime time.Duration `mapstructure:"expiration_time" yaml:"expiration_time" json:"expiration_time"`

	// Refresh token expiration
	RefreshExpirationTime time.Duration `mapstructure:"refresh_expiration_time" yaml:"refresh_expiration_time" json:"refresh_expiration_time"`

	// Issuer
	Issuer string `mapstructure:"issuer" yaml:"issuer" json:"issuer"`

	// Audience
	Audience string `mapstructure:"audience" yaml:"audience" json:"audience"`
}

// EncryptionConfig defines encryption configuration
type EncryptionConfig struct {
	// Master encryption key (base64-encoded, 32 bytes for AES-256)
	MasterKey string `mapstructure:"master_key" yaml:"master_key" json:"master_key"`

	// Enable field-level encryption
	EnableFieldEncryption bool `mapstructure:"enable_field_encryption" yaml:"enable_field_encryption" json:"enable_field_encryption"`

	// Fields to encrypt
	EncryptedFields []string `mapstructure:"encrypted_fields" yaml:"encrypted_fields" json:"encrypted_fields"`
}

// RateLimitConfig defines rate limiting configuration
type RateLimitConfig struct {
	// Enable rate limiting
	Enabled bool `mapstructure:"enabled" yaml:"enabled" json:"enabled"`

	// Requests per second
	RequestsPerSecond int `mapstructure:"requests_per_second" yaml:"requests_per_second" json:"requests_per_second"`

	// Burst size
	Burst int `mapstructure:"burst" yaml:"burst" json:"burst"`

	// Rate limit by (ip, user, api_key)
	LimitBy string `mapstructure:"limit_by" yaml:"limit_by" json:"limit_by"`
}

// RBACConfig defines role-based access control configuration
type RBACConfig struct {
	// Enable RBAC
	Enabled bool `mapstructure:"enabled" yaml:"enabled" json:"enabled"`

	// Policy file path
	PolicyFile string `mapstructure:"policy_file" yaml:"policy_file" json:"policy_file"`

	// Enable policy caching
	EnableCache bool `mapstructure:"enable_cache" yaml:"enable_cache" json:"enable_cache"`
}

// ============================================================================
// Storage Configuration
// ============================================================================

// StorageConfig defines object storage configuration
type StorageConfig struct {
	// Provider (s3, minio, local)
	Provider string `mapstructure:"provider" yaml:"provider" json:"provider"`

	// S3/MinIO configuration
	S3 S3Config `mapstructure:"s3" yaml:"s3" json:"s3"`

	// Local storage configuration
	Local LocalStorageConfig `mapstructure:"local" yaml:"local" json:"local"`
}

// S3Config defines S3-compatible storage configuration
type S3Config struct {
	// Endpoint (for MinIO or S3-compatible services)
	Endpoint string `mapstructure:"endpoint" yaml:"endpoint" json:"endpoint"`

	// Region
	Region string `mapstructure:"region" yaml:"region" json:"region"`

	// Bucket name
	Bucket string `mapstructure:"bucket" yaml:"bucket" json:"bucket"`

	// Access key ID
	AccessKeyID string `mapstructure:"access_key_id" yaml:"access_key_id" json:"access_key_id"`

	// Secret access key
	SecretAccessKey string `mapstructure:"secret_access_key" yaml:"secret_access_key" json:"secret_access_key"`

	// Use SSL
	UseSSL bool `mapstructure:"use_ssl" yaml:"use_ssl" json:"use_ssl"`
}

// LocalStorageConfig defines local filesystem storage configuration
type LocalStorageConfig struct {
	// Base directory
	BasePath string `mapstructure:"base_path" yaml:"base_path" json:"base_path"`

	// Max file size in MB
	MaxFileSize int `mapstructure:"max_file_size" yaml:"max_file_size" json:"max_file_size"`
}

// ============================================================================
// Model Configuration
// ============================================================================

// ModelConfig defines AI model configuration
type ModelConfig struct {
	// Default LLM provider (openai, anthropic, local)
	DefaultProvider string `mapstructure:"default_provider" yaml:"default_provider" json:"default_provider"`

	// OpenAI configuration
	OpenAI OpenAIConfig `mapstructure:"openai" yaml:"openai" json:"openai"`

	// Embedding model configuration
	Embedding EmbeddingConfig `mapstructure:"embedding" yaml:"embedding" json:"embedding"`

	// Model timeout
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" json:"timeout"`

	// Max retries
	MaxRetries int `mapstructure:"max_retries" yaml:"max_retries" json:"max_retries"`
}

// OpenAIConfig defines OpenAI API configuration
type OpenAIConfig struct {
	// API key
	APIKey string `mapstructure:"api_key" yaml:"api_key" json:"api_key"`

	// Organization ID
	OrganizationID string `mapstructure:"organization_id" yaml:"organization_id" json:"organization_id"`

	// Default model
	DefaultModel string `mapstructure:"default_model" yaml:"default_model" json:"default_model"`

	// Base URL (for custom endpoints)
	BaseURL string `mapstructure:"base_url" yaml:"base_url" json:"base_url"`
}

// EmbeddingConfig defines embedding model configuration
type EmbeddingConfig struct {
	// Provider (openai, sentence-transformers, local)
	Provider string `mapstructure:"provider" yaml:"provider" json:"provider"`

	// Model name
	Model string `mapstructure:"model" yaml:"model" json:"model"`

	// Dimension
	Dimension int `mapstructure:"dimension" yaml:"dimension" json:"dimension"`
}

// ============================================================================
// Feature Configuration
// ============================================================================

// FeatureConfig defines feature flags
type FeatureConfig struct {
	// Enable RAG
	EnableRAG bool `mapstructure:"enable_rag" yaml:"enable_rag" json:"enable_rag"`

	// Enable RLHF
	EnableRLHF bool `mapstructure:"enable_rlhf" yaml:"enable_rlhf" json:"enable_rlhf"`

	// Enable multi-level caching
	EnableCaching bool `mapstructure:"enable_caching" yaml:"enable_caching" json:"enable_caching"`

	// Enable workflow engine
	EnableWorkflow bool `mapstructure:"enable_workflow" yaml:"enable_workflow" json:"enable_workflow"`

	// Enable privacy computing
	EnablePrivacyComputing bool `mapstructure:"enable_privacy_computing" yaml:"enable_privacy_computing" json:"enable_privacy_computing"`
}

// ============================================================================
// Configuration Validation
// ============================================================================

// Validate validates the entire configuration
func (c *Config) Validate() error {
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server config: %w", err)
	}

	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database config: %w", err)
	}

	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("redis config: %w", err)
	}

	if err := c.Security.Validate(); err != nil {
		return fmt.Errorf("security config: %w", err)
	}

	return nil
}

// Validate validates server configuration
func (sc *ServerConfig) Validate() error {
	if sc.Port < 1 || sc.Port > 65535 {
		return fmt.Errorf("invalid port: %d", sc.Port)
	}

	if sc.Environment == "" {
		return fmt.Errorf("environment is required")
	}

	return nil
}

// Validate validates database configuration
func (dc *DatabaseConfig) Validate() error {
	if dc.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if dc.Database == "" {
		return fmt.Errorf("database name is required")
	}

	if dc.Username == "" {
		return fmt.Errorf("database username is required")
	}

	return nil
}

// Validate validates Redis configuration
func (rc *RedisConfig) Validate() error {
	if rc.Host == "" {
		return fmt.Errorf("redis host is required")
	}

	if rc.Port < 1 || rc.Port > 65535 {
		return fmt.Errorf("invalid redis port: %d", rc.Port)
	}

	return nil
}

// Validate validates security configuration
func (sec *SecurityConfig) Validate() error {
	if sec.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	return nil
}

//Personal.AI order the ending
