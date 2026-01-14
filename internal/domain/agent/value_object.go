// Package agent provides value objects for agent domain.
// It defines immutable value objects including configurations, constraints,
// and runtime settings with validation and equality comparison methods.
package agent

import (
	"encoding/json"
	"fmt"
	"time"
)

// ============================================================================
// Runtime Configuration Value Object
// ============================================================================

// RuntimeConfig represents immutable runtime configuration
type RuntimeConfig struct {
	// Runtime environment (development, staging, production)
	environment string

	// Runtime version
	version string

	// Resource limits
	resources ResourceLimits

	// Network configuration
	network NetworkConfig

	// Security settings
	security SecurityConfig

	// Timeout settings
	timeouts TimeoutConfig
}

// NewRuntimeConfig creates a new runtime configuration
func NewRuntimeConfig(
	environment string,
	version string,
	resources ResourceLimits,
	network NetworkConfig,
	security SecurityConfig,
	timeouts TimeoutConfig,
) (RuntimeConfig, error) {
	rc := RuntimeConfig{
		environment: environment,
		version:     version,
		resources:   resources,
		network:     network,
		security:    security,
		timeouts:    timeouts,
	}

	if err := rc.Validate(); err != nil {
		return RuntimeConfig{}, fmt.Errorf("invalid runtime config: %w", err)
	}

	return rc, nil
}

// Environment returns the environment
func (rc RuntimeConfig) Environment() string {
	return rc.environment
}

// Version returns the version
func (rc RuntimeConfig) Version() string {
	return rc.version
}

// Resources returns the resource limits
func (rc RuntimeConfig) Resources() ResourceLimits {
	return rc.resources
}

// Network returns the network configuration
func (rc RuntimeConfig) Network() NetworkConfig {
	return rc.network
}

// Security returns the security configuration
func (rc RuntimeConfig) Security() SecurityConfig {
	return rc.security
}

// Timeouts returns the timeout configuration
func (rc RuntimeConfig) Timeouts() TimeoutConfig {
	return rc.timeouts
}

// Validate validates the runtime configuration
func (rc RuntimeConfig) Validate() error {
	validEnvironments := []string{"development", "staging", "production"}
	valid := false
	for _, env := range validEnvironments {
		if rc.environment == env {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid environment: %s", rc.environment)
	}

	if rc.version == "" {
		return fmt.Errorf("version is required")
	}

	if err := rc.resources.Validate(); err != nil {
		return fmt.Errorf("invalid resources: %w", err)
	}

	if err := rc.network.Validate(); err != nil {
		return fmt.Errorf("invalid network config: %w", err)
	}

	if err := rc.security.Validate(); err != nil {
		return fmt.Errorf("invalid security config: %w", err)
	}

	if err := rc.timeouts.Validate(); err != nil {
		return fmt.Errorf("invalid timeout config: %w", err)
	}

	return nil
}

// Equals checks equality with another RuntimeConfig
func (rc RuntimeConfig) Equals(other RuntimeConfig) bool {
	return rc.environment == other.environment &&
		rc.version == other.version &&
		rc.resources.Equals(other.resources) &&
		rc.network.Equals(other.network) &&
		rc.security.Equals(other.security) &&
		rc.timeouts.Equals(other.timeouts)
}

// ============================================================================
// Resource Limits Value Object
// ============================================================================

// ResourceLimits represents immutable resource constraints
type ResourceLimits struct {
	// CPU limit in cores
	cpuCores float64

	// Memory limit in MB
	memoryMB int

	// Disk limit in MB
	diskMB int

	// Max file descriptors
	maxFileDescriptors int

	// Max threads
	maxThreads int
}

// NewResourceLimits creates new resource limits
func NewResourceLimits(cpuCores float64, memoryMB, diskMB, maxFD, maxThreads int) (ResourceLimits, error) {
	rl := ResourceLimits{
		cpuCores:           cpuCores,
		memoryMB:           memoryMB,
		diskMB:             diskMB,
		maxFileDescriptors: maxFD,
		maxThreads:         maxThreads,
	}

	if err := rl.Validate(); err != nil {
		return ResourceLimits{}, err
	}

	return rl, nil
}

// CPUCores returns CPU cores limit
func (rl ResourceLimits) CPUCores() float64 {
	return rl.cpuCores
}

// MemoryMB returns memory limit in MB
func (rl ResourceLimits) MemoryMB() int {
	return rl.memoryMB
}

// DiskMB returns disk limit in MB
func (rl ResourceLimits) DiskMB() int {
	return rl.diskMB
}

// MaxFileDescriptors returns max file descriptors
func (rl ResourceLimits) MaxFileDescriptors() int {
	return rl.maxFileDescriptors
}

// MaxThreads returns max threads
func (rl ResourceLimits) MaxThreads() int {
	return rl.maxThreads
}

// Validate validates resource limits
func (rl ResourceLimits) Validate() error {
	if rl.cpuCores <= 0 {
		return fmt.Errorf("CPU cores must be positive")
	}

	if rl.memoryMB <= 0 {
		return fmt.Errorf("memory must be positive")
	}

	if rl.diskMB < 0 {
		return fmt.Errorf("disk cannot be negative")
	}

	if rl.maxFileDescriptors <= 0 {
		return fmt.Errorf("max file descriptors must be positive")
	}

	if rl.maxThreads <= 0 {
		return fmt.Errorf("max threads must be positive")
	}

	return nil
}

// Equals checks equality with another ResourceLimits
func (rl ResourceLimits) Equals(other ResourceLimits) bool {
	return rl.cpuCores == other.cpuCores &&
		rl.memoryMB == other.memoryMB &&
		rl.diskMB == other.diskMB &&
		rl.maxFileDescriptors == other.maxFileDescriptors &&
		rl.maxThreads == other.maxThreads
}

// ============================================================================
// Network Configuration Value Object
// ============================================================================

// NetworkConfig represents immutable network configuration
type NetworkConfig struct {
	// Enable outbound network access
	enableOutbound bool

	// Allowed domains
	allowedDomains []string

	// Blocked domains
	blockedDomains []string

	// Max connections
	maxConnections int

	// Connection timeout in seconds
	connectionTimeout int

	// Request timeout in seconds
	requestTimeout int

	// Proxy settings
	proxyURL string
}

// NewNetworkConfig creates new network configuration
func NewNetworkConfig(
	enableOutbound bool,
	allowedDomains, blockedDomains []string,
	maxConnections, connectionTimeout, requestTimeout int,
	proxyURL string,
) (NetworkConfig, error) {
	nc := NetworkConfig{
		enableOutbound:    enableOutbound,
		allowedDomains:    copyStringSlice(allowedDomains),
		blockedDomains:    copyStringSlice(blockedDomains),
		maxConnections:    maxConnections,
		connectionTimeout: connectionTimeout,
		requestTimeout:    requestTimeout,
		proxyURL:          proxyURL,
	}

	if err := nc.Validate(); err != nil {
		return NetworkConfig{}, err
	}

	return nc, nil
}

// EnableOutbound returns whether outbound is enabled
func (nc NetworkConfig) EnableOutbound() bool {
	return nc.enableOutbound
}

// AllowedDomains returns allowed domains
func (nc NetworkConfig) AllowedDomains() []string {
	return copyStringSlice(nc.allowedDomains)
}

// BlockedDomains returns blocked domains
func (nc NetworkConfig) BlockedDomains() []string {
	return copyStringSlice(nc.blockedDomains)
}

// MaxConnections returns max connections
func (nc NetworkConfig) MaxConnections() int {
	return nc.maxConnections
}

// ConnectionTimeout returns connection timeout
func (nc NetworkConfig) ConnectionTimeout() int {
	return nc.connectionTimeout
}

// RequestTimeout returns request timeout
func (nc NetworkConfig) RequestTimeout() int {
	return nc.requestTimeout
}

// ProxyURL returns proxy URL
func (nc NetworkConfig) ProxyURL() string {
	return nc.proxyURL
}

// Validate validates network configuration
func (nc NetworkConfig) Validate() error {
	if nc.maxConnections <= 0 {
		return fmt.Errorf("max connections must be positive")
	}

	if nc.connectionTimeout <= 0 {
		return fmt.Errorf("connection timeout must be positive")
	}

	if nc.requestTimeout <= 0 {
		return fmt.Errorf("request timeout must be positive")
	}

	return nil
}

// Equals checks equality with another NetworkConfig
func (nc NetworkConfig) Equals(other NetworkConfig) bool {
	return nc.enableOutbound == other.enableOutbound &&
		stringSliceEquals(nc.allowedDomains, other.allowedDomains) &&
		stringSliceEquals(nc.blockedDomains, other.blockedDomains) &&
		nc.maxConnections == other.maxConnections &&
		nc.connectionTimeout == other.connectionTimeout &&
		nc.requestTimeout == other.requestTimeout &&
		nc.proxyURL == other.proxyURL
}

// ============================================================================
// Security Configuration Value Object
// ============================================================================

// SecurityConfig represents immutable security settings
type SecurityConfig struct {
	// Enable sandboxing
	enableSandbox bool

	// Enable code signing verification
	enableCodeSigning bool

	// Allowed system calls
	allowedSyscalls []string

	// Enable secrets encryption
	enableSecretsEncryption bool

	// Secrets encryption key ID
	secretsKeyID string

	// Enable audit logging
	enableAuditLog bool

	// Max privilege level
	maxPrivilegeLevel PrivilegeLevel
}

// PrivilegeLevel represents security privilege level
type PrivilegeLevel string

const (
	// PrivilegeLevelMinimal for minimal privileges
	PrivilegeLevelMinimal PrivilegeLevel = "minimal"

	// PrivilegeLevelStandard for standard privileges
	PrivilegeLevelStandard PrivilegeLevel = "standard"

	// PrivilegeLevelElevated for elevated privileges
	PrivilegeLevelElevated PrivilegeLevel = "elevated"
)

// NewSecurityConfig creates new security configuration
func NewSecurityConfig(
	enableSandbox, enableCodeSigning bool,
	allowedSyscalls []string,
	enableSecretsEncryption bool,
	secretsKeyID string,
	enableAuditLog bool,
	maxPrivilegeLevel PrivilegeLevel,
) (SecurityConfig, error) {
	sc := SecurityConfig{
		enableSandbox:           enableSandbox,
		enableCodeSigning:       enableCodeSigning,
		allowedSyscalls:         copyStringSlice(allowedSyscalls),
		enableSecretsEncryption: enableSecretsEncryption,
		secretsKeyID:            secretsKeyID,
		enableAuditLog:          enableAuditLog,
		maxPrivilegeLevel:       maxPrivilegeLevel,
	}

	if err := sc.Validate(); err != nil {
		return SecurityConfig{}, err
	}

	return sc, nil
}

// EnableSandbox returns whether sandbox is enabled
func (sc SecurityConfig) EnableSandbox() bool {
	return sc.enableSandbox
}

// EnableCodeSigning returns whether code signing is enabled
func (sc SecurityConfig) EnableCodeSigning() bool {
	return sc.enableCodeSigning
}

// AllowedSyscalls returns allowed system calls
func (sc SecurityConfig) AllowedSyscalls() []string {
	return copyStringSlice(sc.allowedSyscalls)
}

// EnableSecretsEncryption returns whether secrets encryption is enabled
func (sc SecurityConfig) EnableSecretsEncryption() bool {
	return sc.enableSecretsEncryption
}

// SecretsKeyID returns secrets encryption key ID
func (sc SecurityConfig) SecretsKeyID() string {
	return sc.secretsKeyID
}

// EnableAuditLog returns whether audit logging is enabled
func (sc SecurityConfig) EnableAuditLog() bool {
	return sc.enableAuditLog
}

// MaxPrivilegeLevel returns max privilege level
func (sc SecurityConfig) MaxPrivilegeLevel() PrivilegeLevel {
	return sc.maxPrivilegeLevel
}

// Validate validates security configuration
func (sc SecurityConfig) Validate() error {
	if sc.enableSecretsEncryption && sc.secretsKeyID == "" {
		return fmt.Errorf("secrets key ID required when encryption is enabled")
	}

	validLevels := []PrivilegeLevel{
		PrivilegeLevelMinimal,
		PrivilegeLevelStandard,
		PrivilegeLevelElevated,
	}

	valid := false
	for _, level := range validLevels {
		if sc.maxPrivilegeLevel == level {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid privilege level: %s", sc.maxPrivilegeLevel)
	}

	return nil
}

// Equals checks equality with another SecurityConfig
func (sc SecurityConfig) Equals(other SecurityConfig) bool {
	return sc.enableSandbox == other.enableSandbox &&
		sc.enableCodeSigning == other.enableCodeSigning &&
		stringSliceEquals(sc.allowedSyscalls, other.allowedSyscalls) &&
		sc.enableSecretsEncryption == other.enableSecretsEncryption &&
		sc.secretsKeyID == other.secretsKeyID &&
		sc.enableAuditLog == other.enableAuditLog &&
		sc.maxPrivilegeLevel == other.maxPrivilegeLevel
}

// ============================================================================
// Timeout Configuration Value Object
// ============================================================================

// TimeoutConfig represents immutable timeout settings
type TimeoutConfig struct {
	// Execution timeout
	execution time.Duration

	// Initialization timeout
	initialization time.Duration

	// Idle timeout
	idle time.Duration

	// Cleanup timeout
	cleanup time.Duration

	// Graceful shutdown timeout
	gracefulShutdown time.Duration
}

// NewTimeoutConfig creates new timeout configuration
func NewTimeoutConfig(
	execution, initialization, idle, cleanup, gracefulShutdown time.Duration,
) (TimeoutConfig, error) {
	tc := TimeoutConfig{
		execution:        execution,
		initialization:   initialization,
		idle:             idle,
		cleanup:          cleanup,
		gracefulShutdown: gracefulShutdown,
	}

	if err := tc.Validate(); err != nil {
		return TimeoutConfig{}, err
	}

	return tc, nil
}

// Execution returns execution timeout
func (tc TimeoutConfig) Execution() time.Duration {
	return tc.execution
}

// Initialization returns initialization timeout
func (tc TimeoutConfig) Initialization() time.Duration {
	return tc.initialization
}

// Idle returns idle timeout
func (tc TimeoutConfig) Idle() time.Duration {
	return tc.idle
}

// Cleanup returns cleanup timeout
func (tc TimeoutConfig) Cleanup() time.Duration {
	return tc.cleanup
}

// GracefulShutdown returns graceful shutdown timeout
func (tc TimeoutConfig) GracefulShutdown() time.Duration {
	return tc.gracefulShutdown
}

// Validate validates timeout configuration
func (tc TimeoutConfig) Validate() error {
	if tc.execution <= 0 {
		return fmt.Errorf("execution timeout must be positive")
	}

	if tc.initialization <= 0 {
		return fmt.Errorf("initialization timeout must be positive")
	}

	if tc.idle <= 0 {
		return fmt.Errorf("idle timeout must be positive")
	}

	if tc.cleanup <= 0 {
		return fmt.Errorf("cleanup timeout must be positive")
	}

	if tc.gracefulShutdown <= 0 {
		return fmt.Errorf("graceful shutdown timeout must be positive")
	}

	return nil
}

// Equals checks equality with another TimeoutConfig
func (tc TimeoutConfig) Equals(other TimeoutConfig) bool {
	return tc.execution == other.execution &&
		tc.initialization == other.initialization &&
		tc.idle == other.idle &&
		tc.cleanup == other.cleanup &&
		tc.gracefulShutdown == other.gracefulShutdown
}

// ============================================================================
// Execution Constraints Value Object
// ============================================================================

// ExecutionConstraints represents immutable execution constraints
type ExecutionConstraints struct {
	// Max iterations
	maxIterations int

	// Max tokens per request
	maxTokensPerRequest int

	// Max total tokens
	maxTotalTokens int

	// Max concurrent executions
	maxConcurrentExecutions int

	// Rate limit per minute
	rateLimitPerMinute int

	// Cost limit per execution (in cents)
	costLimitCents int

	// Enable cost tracking
	enableCostTracking bool
}

// NewExecutionConstraints creates new execution constraints
func NewExecutionConstraints(
	maxIterations, maxTokensPerRequest, maxTotalTokens,
	maxConcurrentExecutions, rateLimitPerMinute, costLimitCents int,
	enableCostTracking bool,
) (ExecutionConstraints, error) {
	ec := ExecutionConstraints{
		maxIterations:           maxIterations,
		maxTokensPerRequest:     maxTokensPerRequest,
		maxTotalTokens:          maxTotalTokens,
		maxConcurrentExecutions: maxConcurrentExecutions,
		rateLimitPerMinute:      rateLimitPerMinute,
		costLimitCents:          costLimitCents,
		enableCostTracking:      enableCostTracking,
	}

	if err := ec.Validate(); err != nil {
		return ExecutionConstraints{}, err
	}

	return ec, nil
}

// MaxIterations returns max iterations
func (ec ExecutionConstraints) MaxIterations() int {
	return ec.maxIterations
}

// MaxTokensPerRequest returns max tokens per request
func (ec ExecutionConstraints) MaxTokensPerRequest() int {
	return ec.maxTokensPerRequest
}

// MaxTotalTokens returns max total tokens
func (ec ExecutionConstraints) MaxTotalTokens() int {
	return ec.maxTotalTokens
}

// MaxConcurrentExecutions returns max concurrent executions
func (ec ExecutionConstraints) MaxConcurrentExecutions() int {
	return ec.maxConcurrentExecutions
}

// RateLimitPerMinute returns rate limit per minute
func (ec ExecutionConstraints) RateLimitPerMinute() int {
	return ec.rateLimitPerMinute
}

// CostLimitCents returns cost limit in cents
func (ec ExecutionConstraints) CostLimitCents() int {
	return ec.costLimitCents
}

// EnableCostTracking returns whether cost tracking is enabled
func (ec ExecutionConstraints) EnableCostTracking() bool {
	return ec.enableCostTracking
}

// Validate validates execution constraints
func (ec ExecutionConstraints) Validate() error {
	if ec.maxIterations <= 0 {
		return fmt.Errorf("max iterations must be positive")
	}

	if ec.maxTokensPerRequest <= 0 {
		return fmt.Errorf("max tokens per request must be positive")
	}

	if ec.maxTotalTokens <= 0 {
		return fmt.Errorf("max total tokens must be positive")
	}

	if ec.maxConcurrentExecutions <= 0 {
		return fmt.Errorf("max concurrent executions must be positive")
	}

	if ec.rateLimitPerMinute <= 0 {
		return fmt.Errorf("rate limit must be positive")
	}

	if ec.costLimitCents < 0 {
		return fmt.Errorf("cost limit cannot be negative")
	}

	return nil
}

// Equals checks equality with another ExecutionConstraints
func (ec ExecutionConstraints) Equals(other ExecutionConstraints) bool {
	return ec.maxIterations == other.maxIterations &&
		ec.maxTokensPerRequest == other.maxTokensPerRequest &&
		ec.maxTotalTokens == other.maxTotalTokens &&
		ec.maxConcurrentExecutions == other.maxConcurrentExecutions &&
		ec.rateLimitPerMinute == other.rateLimitPerMinute &&
		ec.costLimitCents == other.costLimitCents &&
		ec.enableCostTracking == other.enableCostTracking
}

// ============================================================================
// Utility Functions
// ============================================================================

// copyStringSlice creates a copy of a string slice
func copyStringSlice(src []string) []string {
	if src == nil {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

// stringSliceEquals checks if two string slices are equal
func stringSliceEquals(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ============================================================================
// JSON Serialization
// ============================================================================

// MarshalJSON implements json.Marshaler for RuntimeConfig
func (rc RuntimeConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"environment": rc.environment,
		"version":     rc.version,
		"resources":   rc.resources,
		"network":     rc.network,
		"security":    rc.security,
		"timeouts":    rc.timeouts,
	})
}

// MarshalJSON implements json.Marshaler for TimeoutConfig
func (tc TimeoutConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"execution":          tc.execution.Seconds(),
		"initialization":     tc.initialization.Seconds(),
		"idle":               tc.idle.Seconds(),
		"cleanup":            tc.cleanup.Seconds(),
		"graceful_shutdown":  tc.gracefulShutdown.Seconds(),
	})
}

//Personal.AI order the ending
