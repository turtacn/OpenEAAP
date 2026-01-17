// Package model provides domain entities and business logic for AI models.
// It defines the Model structure with core fields and domain methods
// for availability checking, cost calculation, and configuration management.
package model

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Model Entity
// ============================================================================

// Model represents an AI model entity
type Model struct {
	// Unique identifier
	ID string `json:"id"`

	// Model name
	Name string `json:"name"`

	// Model display name
	DisplayName string `json:"display_name"`

	// Model description
	Description string `json:"description"`

	// Model type (llm, embedding, image, audio, etc.)
	Type ModelType `json:"type"`

	// Model provider (openai, anthropic, google, etc.)
	Provider string `json:"provider"`

	// Model version
	Version string `json:"version"`

	// API endpoint
	Endpoint string `json:"endpoint"`

	// Model configuration
	Config ModelConfig `json:"config"`

	// Capabilities
	Capabilities ModelCapabilities `json:"capabilities"`

	// Pricing information
	Pricing PricingInfo `json:"pricing"`

	// Rate limits
	RateLimits RateLimits `json:"rate_limits"`

	// Performance metrics
	Performance PerformanceMetrics `json:"performance"`

	// Availability status
	Status ModelStatus `json:"status"`

	// Health check configuration
	HealthCheck HealthCheckConfig `json:"health_check"`

	// Tags for categorization
	Tags []string `json:"tags"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata"`

	// Statistics
	Stats ModelStats `json:"stats"`

	// Is deprecated
	Deprecated bool `json:"deprecated"`

	// Deprecation message
	DeprecationMessage string `json:"deprecation_message,omitempty"`

	// Replacement model ID
	ReplacementModelID string `json:"replacement_model_id,omitempty"`

	// Creation timestamp
	CreatedAt time.Time `json:"created_at"`

	// Last update timestamp
	UpdatedAt time.Time `json:"updated_at"`

	// Last health check timestamp
	LastHealthCheckAt *time.Time `json:"last_health_check_at,omitempty"`

	// Deletion timestamp (soft delete)
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// ============================================================================
// Model Type
// ============================================================================

// ModelType defines the type of AI model
type ModelType string

const (
	// ModelTypeLLM for large language models
	ModelTypeLLM ModelType = "llm"

	// ModelTypeEmbedding for embedding models
	ModelTypeEmbedding ModelType = "embedding"

	// ModelTypeImage for image generation models
	ModelTypeImage ModelType = "image"

	// ModelTypeAudio for audio models
	ModelTypeAudio ModelType = "audio"

	// ModelTypeVideo for video models
	ModelTypeVideo ModelType = "video"

	// ModelTypeMultimodal for multimodal models
	ModelTypeMultimodal ModelType = "multimodal"

	// ModelTypeClassification for classification models
	ModelTypeClassification ModelType = "classification"

	// ModelTypeTranslation for translation models
	ModelTypeTranslation ModelType = "translation"

	// ModelTypeSummarization for summarization models
	ModelTypeSummarization ModelType = "summarization"

	// ModelTypeCodeGeneration for code generation models
	ModelTypeCodeGeneration ModelType = "code_generation"
)

// ============================================================================
// Model Status
// ============================================================================

// ModelStatus represents the current status of a model
type ModelStatus string

const (
	// ModelStatusAvailable indicates model is available
	ModelStatusAvailable ModelStatus = "available"

	// ModelStatusUnavailable indicates model is unavailable
	ModelStatusUnavailable ModelStatus = "unavailable"

	// ModelStatusMaintenance indicates model is under maintenance
	ModelStatusMaintenance ModelStatus = "maintenance"

	// ModelStatusDegraded indicates model has degraded performance
	ModelStatusDegraded ModelStatus = "degraded"

	// ModelStatusDeprecated indicates model is deprecated
	ModelStatusDeprecated ModelStatus = "deprecated"

	// ModelStatusRetired indicates model is retired
	ModelStatusRetired ModelStatus = "retired"
)

// ============================================================================
// Model Configuration
// ============================================================================

// ModelConfig defines model-specific configuration
type ModelConfig struct {
	// API key or credentials
	APIKey string `json:"api_key,omitempty"`

	// Base URL
	BaseURL string `json:"base_url"`

	// Timeout in seconds
	TimeoutSeconds int `json:"timeout_seconds"`

	// Max retries
	MaxRetries int `json:"max_retries"`

	// Retry delay in milliseconds
	RetryDelayMs int `json:"retry_delay_ms"`

	// Connection pool size
	PoolSize int `json:"pool_size"`

	// Enable streaming
	EnableStreaming bool `json:"enable_streaming"`

	// Custom headers
	CustomHeaders map[string]string `json:"custom_headers,omitempty"`

	// Proxy URL
	ProxyURL string `json:"proxy_url,omitempty"`

	// TLS configuration
	TLSConfig TLSConfig `json:"tls_config"`

	// Default parameters
	DefaultParams map[string]interface{} `json:"default_params,omitempty"`
}

// TLSConfig defines TLS configuration
type TLSConfig struct {
	// Enable TLS
	Enabled bool `json:"enabled"`

	// Verify certificate
	VerifyCert bool `json:"verify_cert"`

	// CA certificate path
	CACertPath string `json:"ca_cert_path,omitempty"`

	// Client certificate path
	ClientCertPath string `json:"client_cert_path,omitempty"`

	// Client key path
	ClientKeyPath string `json:"client_key_path,omitempty"`
}

// ============================================================================
// Model Capabilities
// ============================================================================

// ModelCapabilities defines model capabilities
type ModelCapabilities struct {
	// Supports chat
	Chat bool `json:"chat"`

	// Supports completion
	Completion bool `json:"completion"`

	// Supports streaming
	Streaming bool `json:"streaming"`

	// Supports function calling
	FunctionCalling bool `json:"function_calling"`

	// Supports vision (image input)
	Vision bool `json:"vision"`

	// Supports audio input
	AudioInput bool `json:"audio_input"`

	// Supports audio output
	AudioOutput bool `json:"audio_output"`

	// Supports fine-tuning
	FineTuning bool `json:"fine_tuning"`

	// Supports embeddings
	Embeddings bool `json:"embeddings"`

	// Max context length (tokens)
	MaxContextLength int `json:"max_context_length"`

	// Max output length (tokens)
	MaxOutputLength int `json:"max_output_length"`

	// Supported languages
	SupportedLanguages []string `json:"supported_languages"`

	// Supported formats
	SupportedFormats []string `json:"supported_formats"`
}

// ============================================================================
// Pricing Information
// ============================================================================

// PricingInfo defines model pricing
type PricingInfo struct {
	// Currency (USD, EUR, etc.)
	Currency string `json:"currency"`

	// Input token cost (per 1000 tokens)
	InputTokenCost float64 `json:"input_token_cost"`

	// Output token cost (per 1000 tokens)
	OutputTokenCost float64 `json:"output_token_cost"`

	// Image cost (per image)
	ImageCost float64 `json:"image_cost,omitempty"`

	// Audio cost (per minute)
	AudioCost float64 `json:"audio_cost,omitempty"`

	// Video cost (per minute)
	VideoCost float64 `json:"video_cost,omitempty"`

	// Request cost (per request)
	RequestCost float64 `json:"request_cost,omitempty"`

	// Minimum charge
	MinimumCharge float64 `json:"minimum_charge,omitempty"`

	// Free tier limits
	FreeTier *FreeTierLimits `json:"free_tier,omitempty"`

	// Billing model (pay_per_use, subscription, etc.)
	BillingModel string `json:"billing_model"`
}

// FreeTierLimits defines free tier limits
type FreeTierLimits struct {
	// Free tokens per month
	FreeTokensPerMonth int64 `json:"free_tokens_per_month"`

	// Free requests per month
	FreeRequestsPerMonth int64 `json:"free_requests_per_month"`

	// Free tier expires at
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// ============================================================================
// Rate Limits
// ============================================================================

// RateLimits defines rate limiting configuration
type RateLimits struct {
	// Requests per minute
	RequestsPerMinute int `json:"requests_per_minute"`

	// Requests per day
	RequestsPerDay int `json:"requests_per_day"`

	// Tokens per minute
	TokensPerMinute int `json:"tokens_per_minute"`

	// Tokens per day
	TokensPerDay int `json:"tokens_per_day"`

	// Concurrent requests
	ConcurrentRequests int `json:"concurrent_requests"`

	// Burst limit
	BurstLimit int `json:"burst_limit"`
}

// ============================================================================
// Performance Metrics
// ============================================================================

// PerformanceMetrics tracks model performance
type PerformanceMetrics struct {
	// Average latency in milliseconds
	AvgLatencyMs int64 `json:"avg_latency_ms"`

	// P50 latency
	P50LatencyMs int64 `json:"p50_latency_ms"`

	// P95 latency
	P95LatencyMs int64 `json:"p95_latency_ms"`

	// P99 latency
	P99LatencyMs int64 `json:"p99_latency_ms"`

	// Average tokens per second
	AvgTokensPerSecond float64 `json:"avg_tokens_per_second"`

	// Success rate (0-1)
	SuccessRate float64 `json:"success_rate"`

	// Uptime percentage
	UptimePercentage float64 `json:"uptime_percentage"`

	// Last measured at
	LastMeasuredAt time.Time `json:"last_measured_at"`
}

// ============================================================================
// Health Check Configuration
// ============================================================================

// HealthCheckConfig defines health check settings
type HealthCheckConfig struct {
	// Enable health checks
	Enabled bool `json:"enabled"`

	// Check interval in seconds
	IntervalSeconds int `json:"interval_seconds"`

	// Timeout in seconds
	TimeoutSeconds int `json:"timeout_seconds"`

	// Failure threshold (consecutive failures before marking unhealthy)
	FailureThreshold int `json:"failure_threshold"`

	// Success threshold (consecutive successes before marking healthy)
	SuccessThreshold int `json:"success_threshold"`

	// Health check endpoint
	Endpoint string `json:"endpoint,omitempty"`

	// Expected response code
	ExpectedStatusCode int `json:"expected_status_code"`
}

// ============================================================================
// Model Statistics
// ============================================================================

// ModelStats tracks model usage statistics
type ModelStats struct {
	// Total requests
	TotalRequests int64 `json:"total_requests"`

	// Successful requests
	SuccessfulRequests int64 `json:"successful_requests"`

	// Failed requests
	FailedRequests int64 `json:"failed_requests"`

	// Total tokens processed
	TotalTokens int64 `json:"total_tokens"`

	// Total cost
	TotalCost float64 `json:"total_cost"`

	// Last request at
	LastRequestAt *time.Time `json:"last_request_at,omitempty"`
}

// ============================================================================
// Entity Factory
// ============================================================================

// NewModel creates a new model entity
func NewModel(name, displayName, modelType, provider, version string) *Model {
	now := time.Now()
	return &Model{
		ID:          generateModelID(),
		Name:        name,
		DisplayName: displayName,
		Type:        ModelType(modelType),
		Provider:    provider,
		Version:     version,
		Status:      ModelStatusAvailable,
		Tags:        make([]string, 0),
		Metadata:    make(map[string]interface{}),
		Config:      defaultModelConfig(),
		Capabilities: ModelCapabilities{
			SupportedLanguages: make([]string, 0),
			SupportedFormats:   make([]string, 0),
		},
		Pricing: PricingInfo{
			Currency:     "USD",
			BillingModel: "pay_per_use",
		},
		RateLimits: defaultRateLimits(),
		Performance: PerformanceMetrics{
			LastMeasuredAt: now,
		},
		HealthCheck: defaultHealthCheckConfig(),
		Stats:       ModelStats{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// generateModelID generates a unique model ID
func generateModelID() string {
	return fmt.Sprintf("model_%s", uuid.New().String())
}

// defaultModelConfig returns default model configuration
func defaultModelConfig() ModelConfig {
	return ModelConfig{
		TimeoutSeconds:  30,
		MaxRetries:      3,
		RetryDelayMs:    1000,
		PoolSize:        10,
		EnableStreaming: true,
		TLSConfig: TLSConfig{
			Enabled:    true,
			VerifyCert: true,
		},
	}
}

// defaultRateLimits returns default rate limits
func defaultRateLimits() RateLimits {
	return RateLimits{
		RequestsPerMinute:  60,
		RequestsPerDay:     10000,
		TokensPerMinute:    90000,
		TokensPerDay:       1000000,
		ConcurrentRequests: 10,
		BurstLimit:         100,
	}
}

// defaultHealthCheckConfig returns default health check configuration
func defaultHealthCheckConfig() HealthCheckConfig {
	return HealthCheckConfig{
		Enabled:            true,
		IntervalSeconds:    60,
		TimeoutSeconds:     10,
		FailureThreshold:   3,
		SuccessThreshold:   2,
		ExpectedStatusCode: 200,
	}
}

// ============================================================================
// Domain Methods - Availability
// ============================================================================

// IsAvailable checks if the model is currently available
func (m *Model) IsAvailable() bool {
	if m.IsDeleted() {
		return false
	}

	if m.Status != ModelStatusAvailable {
		return false
	}

	if m.Deprecated {
		return false
	}

	return true
}

// IsHealthy checks if the model is healthy based on recent health checks
func (m *Model) IsHealthy() bool {
	if !m.IsAvailable() {
		return false
	}

	if m.LastHealthCheckAt == nil {
		return true // No health check data yet, assume healthy
	}

	// Check if last health check was recent (within 2x interval)
	maxAge := time.Duration(m.HealthCheck.IntervalSeconds*2) * time.Second
	if time.Since(*m.LastHealthCheckAt) > maxAge {
		return false
	}

	return m.Status == ModelStatusAvailable
}

// CanHandleRequest checks if model can handle a request based on rate limits
func (m *Model) CanHandleRequest(tokenCount int) bool {
	if !m.IsAvailable() {
		return false
	}

	// Check token limits (simplified - actual implementation would track usage)
	if tokenCount > m.Capabilities.MaxContextLength {
		return false
	}

	return true
}

// SupportsCapability checks if model supports a specific capability
func (m *Model) SupportsCapability(capability string) bool {
	switch capability {
	case "chat":
		return m.Capabilities.Chat
	case "completion":
		return m.Capabilities.Completion
	case "streaming":
		return m.Capabilities.Streaming
	case "function_calling":
		return m.Capabilities.FunctionCalling
	case "vision":
		return m.Capabilities.Vision
	case "audio_input":
		return m.Capabilities.AudioInput
	case "audio_output":
		return m.Capabilities.AudioOutput
	case "fine_tuning":
		return m.Capabilities.FineTuning
	case "embeddings":
		return m.Capabilities.Embeddings
	default:
		return false
	}
}

// ============================================================================
// Domain Methods - Cost Calculation
// ============================================================================

// GetCost calculates the cost for a given usage
func (m *Model) GetCost(inputTokens, outputTokens int) float64 {
	inputCost := float64(inputTokens) / 1000.0 * m.Pricing.InputTokenCost
	outputCost := float64(outputTokens) / 1000.0 * m.Pricing.OutputTokenCost
	totalCost := inputCost + outputCost

	// Apply minimum charge if applicable
	if m.Pricing.MinimumCharge > 0 && totalCost < m.Pricing.MinimumCharge {
		totalCost = m.Pricing.MinimumCharge
	}

	return totalCost
}

// GetRequestCost calculates the cost per request
func (m *Model) GetRequestCost() float64 {
	return m.Pricing.RequestCost
}

// GetImageCost calculates the cost for image generation
func (m *Model) GetImageCost(count int) float64 {
	return float64(count) * m.Pricing.ImageCost
}

// GetAudioCost calculates the cost for audio processing
func (m *Model) GetAudioCost(durationMinutes float64) float64 {
	return durationMinutes * m.Pricing.AudioCost
}

// GetVideoCost calculates the cost for video processing
func (m *Model) GetVideoCost(durationMinutes float64) float64 {
	return durationMinutes * m.Pricing.VideoCost
}

// EstimateCost estimates cost based on content length
func (m *Model) EstimateCost(contentLength int, outputRatio float64) float64 {
	estimatedInputTokens := contentLength / 4 // Rough estimate: 4 chars per token
	estimatedOutputTokens := int(float64(estimatedInputTokens) * outputRatio)
	return m.GetCost(estimatedInputTokens, estimatedOutputTokens)
}

// ============================================================================
// Domain Methods - Status Management
// ============================================================================

// IsDeleted checks if the model is soft deleted
func (m *Model) IsDeleted() bool {
	return m.DeletedAt != nil
}

// IsDeprecated checks if the model is deprecated
func (m *Model) IsDeprecated() bool {
	return m.Deprecated || m.Status == ModelStatusDeprecated
}

// IsRetired checks if the model is retired
func (m *Model) IsRetired() bool {
	return m.Status == ModelStatusRetired
}

// MarkDeprecated marks the model as deprecated
func (m *Model) MarkDeprecated(message string, replacementModelID string) {
	m.Deprecated = true
	m.DeprecationMessage = message
	m.ReplacementModelID = replacementModelID
	m.Status = ModelStatusDeprecated
	m.UpdatedAt = time.Now()
}

// MarkRetired marks the model as retired
func (m *Model) MarkRetired() error {
	if m.Status == ModelStatusAvailable {
		return fmt.Errorf("cannot retire available model without deprecation")
	}

	m.Status = ModelStatusRetired
	m.UpdatedAt = time.Now()
	return nil
}

// SetStatus updates the model status
func (m *Model) SetStatus(status ModelStatus) {
	m.Status = status
	m.UpdatedAt = time.Now()
}

// SetMaintenance sets the model to maintenance mode
func (m *Model) SetMaintenance() {
	m.Status = ModelStatusMaintenance
	m.UpdatedAt = time.Now()
}

// RestoreFromMaintenance restores the model from maintenance
func (m *Model) RestoreFromMaintenance() {
	m.Status = ModelStatusAvailable
	m.UpdatedAt = time.Now()
}

// ============================================================================
// Domain Methods - Statistics
// ============================================================================

// RecordRequest records a request in statistics
func (m *Model) RecordRequest(success bool, tokens int, cost float64) {
	m.Stats.TotalRequests++
	if success {
		m.Stats.SuccessfulRequests++
	} else {
		m.Stats.FailedRequests++
	}

	m.Stats.TotalTokens += int64(tokens)
	m.Stats.TotalCost += cost

	now := time.Now()
	m.Stats.LastRequestAt = &now
	m.UpdatedAt = now
}

// GetSuccessRate returns the success rate
func (m *Model) GetSuccessRate() float64 {
	if m.Stats.TotalRequests == 0 {
		return 0.0
	}
	return float64(m.Stats.SuccessfulRequests) / float64(m.Stats.TotalRequests)
}

// GetAverageCostPerRequest returns average cost per request
func (m *Model) GetAverageCostPerRequest() float64 {
	if m.Stats.TotalRequests == 0 {
		return 0.0
	}
	return m.Stats.TotalCost / float64(m.Stats.TotalRequests)
}

// GetAverageTokensPerRequest returns average tokens per request
func (m *Model) GetAverageTokensPerRequest() float64 {
	if m.Stats.TotalRequests == 0 {
		return 0.0
	}
	return float64(m.Stats.TotalTokens) / float64(m.Stats.TotalRequests)
}

// ============================================================================
// Domain Methods - Health
// ============================================================================

// UpdateHealthCheck updates health check timestamp and status
func (m *Model) UpdateHealthCheck(healthy bool) {
	now := time.Now()
	m.LastHealthCheckAt = &now

	if healthy {
		if m.Status != ModelStatusAvailable {
			m.Status = ModelStatusAvailable
		}
	} else {
		if m.Status == ModelStatusAvailable {
			m.Status = ModelStatusDegraded
		}
	}

	m.UpdatedAt = now
}

// ShouldPerformHealthCheck checks if health check is due
func (m *Model) ShouldPerformHealthCheck() bool {
	if !m.HealthCheck.Enabled {
		return false
	}

	if m.LastHealthCheckAt == nil {
		return true
	}

	interval := time.Duration(m.HealthCheck.IntervalSeconds) * time.Second
	return time.Since(*m.LastHealthCheckAt) >= interval
}

// ============================================================================
// Domain Methods - Validation
// ============================================================================

// Validate validates the model entity
func (m *Model) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("model ID is required")
	}

	if m.Name == "" {
		return fmt.Errorf("model name is required")
	}

	if m.Type == "" {
		return fmt.Errorf("model type is required")
	}

	if m.Provider == "" {
		return fmt.Errorf("provider is required")
	}

	if m.Version == "" {
		return fmt.Errorf("version is required")
	}

	if m.Config.TimeoutSeconds <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if m.Capabilities.MaxContextLength <= 0 {
		return fmt.Errorf("max context length must be positive")
	}

	if m.Pricing.InputTokenCost < 0 || m.Pricing.OutputTokenCost < 0 {
		return fmt.Errorf("token costs cannot be negative")
	}

	return nil
}

// ============================================================================
// Domain Methods - Configuration
// ============================================================================

// UpdateConfig updates model configuration
func (m *Model) UpdateConfig(config ModelConfig) error {
	if config.TimeoutSeconds <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	m.Config = config
	m.UpdatedAt = time.Now()
	return nil
}

// UpdatePricing updates pricing information
func (m *Model) UpdatePricing(pricing PricingInfo) error {
	if pricing.InputTokenCost < 0 || pricing.OutputTokenCost < 0 {
		return fmt.Errorf("token costs cannot be negative")
	}

	m.Pricing = pricing
	m.UpdatedAt = time.Now()
	return nil
}

// UpdateRateLimits updates rate limits
func (m *Model) UpdateRateLimits(limits RateLimits) error {
	if limits.RequestsPerMinute <= 0 || limits.TokensPerMinute <= 0 {
		return fmt.Errorf("rate limits must be positive")
	}

	m.RateLimits = limits
	m.UpdatedAt = time.Now()
	return nil
}

// UpdatePerformance updates performance metrics
func (m *Model) UpdatePerformance(metrics PerformanceMetrics) {
	m.Performance = metrics
	m.Performance.LastMeasuredAt = time.Now()
	m.UpdatedAt = time.Now()
}

// ============================================================================
// Domain Methods - Soft Delete
// ============================================================================

// SoftDelete soft deletes the model
func (m *Model) SoftDelete() {
	now := time.Now()
	m.DeletedAt = &now
	m.Status = ModelStatusRetired
	m.UpdatedAt = now
}

// Restore restores a soft-deleted model
func (m *Model) Restore() error {
	if !m.IsDeleted() {
		return fmt.Errorf("model is not deleted")
	}

	m.DeletedAt = nil
	m.Status = ModelStatusAvailable
	m.UpdatedAt = time.Now()
	return nil
}

// ============================================================================
// Domain Methods - Utilities
// ============================================================================

// Clone creates a copy of the model
func (m *Model) Clone() *Model {
	data, _ := json.Marshal(m)
	var cloned Model
	json.Unmarshal(data, &cloned)
	return &cloned
}

// ToJSON converts model to JSON
func (m *Model) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// GetEndpoint returns the full API endpoint
func (m *Model) GetEndpoint() string {
	if m.Endpoint != "" {
		return m.Endpoint
	}
	return m.Config.BaseURL
}

// HasTag checks if model has a specific tag
func (m *Model) HasTag(tag string) bool {
	for _, t := range m.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// AddTag adds a tag to the model
func (m *Model) AddTag(tag string) {
	if !m.HasTag(tag) {
		m.Tags = append(m.Tags, tag)
		m.UpdatedAt = time.Now()
	}
}

// RemoveTag removes a tag from the model
func (m *Model) RemoveTag(tag string) {
	for i, t := range m.Tags {
		if t == tag {
			m.Tags = append(m.Tags[:i], m.Tags[i+1:]...)
			m.UpdatedAt = time.Now()
			return
		}
	}
}

//Personal.AI order the ending

// ModelMetrics represents model performance metrics
type ModelMetrics struct {
	TotalInferences  int64   `json:"total_inferences"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	ErrorRate        float64 `json:"error_rate"`
	P95LatencyMs     float64 `json:"p95_latency_ms"`
	P99LatencyMs     float64 `json:"p99_latency_ms"`
	TokensPerSecond  float64 `json:"tokens_per_second"`
}
