// Package privacy provides privacy protection capabilities for inference requests.
// This file implements the privacy gateway that automatically detects and redacts PII.
package privacy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/pkg/errors"
)

// PrivacyAction represents the action taken by the privacy gateway
type PrivacyAction string

const (
	// ActionRedact redacts PII from text
	ActionRedact PrivacyAction = "redact"

	// ActionEncrypt encrypts PII in text
	ActionEncrypt PrivacyAction = "encrypt"

	// ActionBlock blocks request with PII
	ActionBlock PrivacyAction = "block"

	// ActionAllow allows request without modification
	ActionAllow PrivacyAction = "allow"
)

// PrivacyPolicy defines privacy protection policy
type PrivacyPolicy struct {
	Action              PrivacyAction
	EnableRestore       bool
	MinRiskThreshold    float64
	BlockedPIITypes     []PIIType
	AllowedPIITypes     []PIIType
	EnableAuditLog      bool
	RequireConsent      bool
}

// PrivacyGateway provides PII detection and redaction for inference
type PrivacyGateway struct {
	logger           logging.Logger
	metricsCollector *metrics.MetricsCollector
	detector         *PIIDetector
	policy           *PrivacyPolicy
	auditLogger      AuditLogger
	encryptor        Encryptor
}

// AuditLogger interface for logging privacy events
type AuditLogger interface {
	LogPrivacyEvent(ctx context.Context, event *PrivacyAuditEvent) error
}

// Encryptor interface for encrypting/decrypting PII
type Encryptor interface {
	Encrypt(ctx context.Context, plaintext string) (string, error)
	Decrypt(ctx context.Context, ciphertext string) (string, error)
}

// PrivacyAuditEvent represents a privacy audit log entry
type PrivacyAuditEvent struct {
	EventID       string                 `json:"event_id"`
	Timestamp     time.Time              `json:"timestamp"`
	UserID        string                 `json:"user_id,omitempty"`
	RequestID     string                 `json:"request_id"`
	Action        PrivacyAction          `json:"action"`
	PIIDetected   bool                   `json:"pii_detected"`
	PIITypes      []PIIType              `json:"pii_types,omitempty"`
	RiskScore     float64                `json:"risk_score"`
	Redacted      bool                   `json:"redacted"`
	Blocked       bool                   `json:"blocked"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ProcessRequest represents a request to be processed
type ProcessRequest struct {
	RequestID   string
	UserID      string
	Text        string
	Metadata    map[string]interface{}
	ConsentGiven bool
}

// ProcessResponse represents the processed response
type ProcessResponse struct {
	OriginalText   string
	ProcessedText  string
	DetectionResult *PIIDetectionResult
	Action         PrivacyAction
	Blocked        bool
	RestoreToken   string // Token to restore original PII
	Metadata       map[string]interface{}
}

// RestoreRequest represents a request to restore redacted text
type RestoreRequest struct {
	ProcessedText string
	RestoreToken  string
	RequestID     string
}

// NewPrivacyGateway creates a new privacy gateway
func NewPrivacyGateway(
	logger logging.Logger,
	metricsCollector *metrics.MetricsCollector,
	detector *PIIDetector,
	policy *PrivacyPolicy,
	auditLogger AuditLogger,
	encryptor Encryptor,
) *PrivacyGateway {
	if policy == nil {
		policy = defaultPolicy()
	}

	return &PrivacyGateway{
		logger:           logger,
		metricsCollector: metricsCollector,
		detector:         detector,
		policy:           policy,
		auditLogger:      auditLogger,
		encryptor:        encryptor,
	}
}

// ProcessInbound processes inbound request before inference
func (g *PrivacyGateway) ProcessInbound(ctx context.Context, req *ProcessRequest) (*ProcessResponse, error) {
	startTime := time.Now()

	if req == nil || req.Text == "" {
		return &ProcessResponse{
			OriginalText:  "",
			ProcessedText: "",
			Action:        ActionAllow,
		}, nil
	}

	// Detect PII in the text
	detectionResult, err := g.detector.Detect(ctx, req.Text)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "PII detection failed")
	}

	// Determine action based on policy and detection result
	action := g.determineAction(ctx, detectionResult)

	response := &ProcessResponse{
		OriginalText:    req.Text,
		ProcessedText:   req.Text,
		DetectionResult: detectionResult,
		Action:          action,
		Metadata:        make(map[string]interface{}),
	}

	// Handle based on action
	switch action {
	case ActionBlock:
		response.Blocked = true
		response.ProcessedText = ""
  g.logger.WithContext(ctx).Warn("request blocked due to PII policy violation", logging.Any("request_id", req.RequestID), logging.Any("risk_score", detectionResult.RiskScore))

	case ActionRedact:
		response.ProcessedText = detectionResult.RedactedText

		// Generate restore token if restore is enabled
		if g.policy.EnableRestore {
			token, err := g.generateRestoreToken(ctx, detectionResult)
			if err != nil {
				g.logger.WithContext(ctx).Warn("failed to generate restore token", logging.Error(err))
			} else {
				response.RestoreToken = token
			}
		}

  g.logger.WithContext(ctx).Info("PII redacted from request", logging.Any("request_id", req.RequestID), logging.Any("entity_count", len(detectionResult.Entities)))

	case ActionEncrypt:
		if g.encryptor == nil {
			g.logger.WithContext(ctx).Warn("encryptor not configured, falling back to redact")
			response.ProcessedText = detectionResult.RedactedText
			response.Action = ActionRedact
		} else {
			encryptedText, err := g.encryptPII(ctx, req.Text, detectionResult)
			if err != nil {
				g.logger.WithContext(ctx).Warn("encryption failed, falling back to redact", logging.Error(err))
				response.ProcessedText = detectionResult.RedactedText
				response.Action = ActionRedact
			} else {
				response.ProcessedText = encryptedText
			}
		}

	case ActionAllow:
		// No modification needed
		response.ProcessedText = req.Text
	}

	// Log audit event
	if g.policy.EnableAuditLog && g.auditLogger != nil {
		auditEvent := &PrivacyAuditEvent{
			EventID:     g.generateEventID(),
			Timestamp:   time.Now(),
			UserID:      req.UserID,
			RequestID:   req.RequestID,
			Action:      action,
			PIIDetected: detectionResult.HasPII,
			PIITypes:    g.extractPIITypes(detectionResult),
			RiskScore:   detectionResult.RiskScore,
			Redacted:    action == ActionRedact || action == ActionEncrypt,
			Blocked:     action == ActionBlock,
			Metadata:    req.Metadata,
		}

		if err := g.auditLogger.LogPrivacyEvent(ctx, auditEvent); err != nil {
			g.logger.WithContext(ctx).Warn("failed to log audit event", logging.Error(err))
		}
	}

	// Record metrics
	g.recordMetrics("inbound", action, detectionResult.HasPII, time.Since(startTime))

	return response, nil
}

// ProcessOutbound processes outbound response after inference
func (g *PrivacyGateway) ProcessOutbound(ctx context.Context, text string, restoreToken string) (string, error) {
	startTime := time.Now()

	if text == "" {
		return text, nil
	}

	// If no restore token, return as-is
	if restoreToken == "" || !g.policy.EnableRestore {
		return text, nil
	}

	// Detect PII in response
	detectionResult, err := g.detector.Detect(ctx, text)
	if err != nil {
		g.logger.WithContext(ctx).Warn("PII detection failed for outbound", logging.Error(err))
		return text, nil
	}

	// If response contains new PII, redact it
	if detectionResult.HasPII && detectionResult.RiskScore >= g.policy.MinRiskThreshold {
		g.logger.WithContext(ctx).Warn("new PII detected in response, redacting")
		text = detectionResult.RedactedText
	}

	// Record metrics
	g.recordMetrics("outbound", ActionAllow, detectionResult.HasPII, time.Since(startTime))

	return text, nil
}

// Restore restores original PII in text using restore token
func (g *PrivacyGateway) Restore(ctx context.Context, req *RestoreRequest) (string, error) {
	if !g.policy.EnableRestore {
		return "", errors.NewInternalError("PRIV_ERR", "restore is not enabled")
	}

	if req.RestoreToken == "" {
		return req.ProcessedText, nil
	}

	// Decode restore token
	mappings, err := g.decodeRestoreToken(ctx, req.RestoreToken)
	if err != nil {
		return "", errors.Wrap(err, errors.CodeInvalidArgument, "invalid restore token")
	}

	// Restore PII in text
	restoredText := req.ProcessedText

	// Apply replacements in reverse order (from end to start) to avoid offset issues
	for i := len(mappings) - 1; i >= 0; i-- {
		mapping := mappings[i]

		// Find and replace the redacted placeholder
		placeholder := fmt.Sprintf("[%s]", mapping.Type)
		if idx := findPlaceholder(restoredText, placeholder, mapping.Start); idx >= 0 {
			restoredText = restoredText[:idx] + mapping.OriginalValue + restoredText[idx+len(placeholder):]
		}
	}

 g.logger.WithContext(ctx).Info("PII restored", logging.Any("request_id", req.RequestID), logging.Any("mapping_count", len(mappings)))

	return restoredText, nil
}

// determineAction determines what action to take based on detection result
func (g *PrivacyGateway) determineAction(ctx context.Context, result *PIIDetectionResult) PrivacyAction {
	// If no PII detected, allow
	if !result.HasPII {
		return ActionAllow
	}

	// Check if risk score exceeds threshold
	if result.RiskScore < g.policy.MinRiskThreshold {
		return ActionAllow
	}

	// Check if any blocked PII types are present
	for _, entity := range result.Entities {
		for _, blockedType := range g.policy.BlockedPIITypes {
			if entity.Type == blockedType {
				return ActionBlock
			}
		}
	}

	// Check if consent is required
	if g.policy.RequireConsent {
		// Consent checking would be implemented here
		// For now, default to blocking
		return ActionBlock
	}

	// Return configured action
	return g.policy.Action
}

// generateRestoreToken generates a token to restore redacted PII
func (g *PrivacyGateway) generateRestoreToken(ctx context.Context, result *PIIDetectionResult) (string, error) {
	if len(result.Entities) == 0 {
		return "", nil
	}

	// Create mappings for restoration
	mappings := make([]restoreMapping, len(result.Entities))

	for i, entity := range result.Entities {
		mappings[i] = restoreMapping{
			Type:          string(entity.Type),
			OriginalValue: entity.Value,
			Start:         entity.Start,
			End:           entity.End,
		}
	}

	// Serialize mappings
	data, err := json.Marshal(mappings)
	if err != nil {
		return "", err
	}

	// Encrypt if encryptor is available
	if g.encryptor != nil {
		encrypted, err := g.encryptor.Encrypt(ctx, string(data))
		if err != nil {
			g.logger.WithContext(ctx).Warn("failed to encrypt restore token", logging.Error(err))
			return string(data), nil // Return unencrypted as fallback
		}
		return encrypted, nil
	}

	return string(data), nil
}

// decodeRestoreToken decodes a restore token
func (g *PrivacyGateway) decodeRestoreToken(ctx context.Context, token string) ([]restoreMapping, error) {
	data := token

	// Decrypt if encryptor is available
	if g.encryptor != nil {
		decrypted, err := g.encryptor.Decrypt(ctx, token)
		if err != nil {
			g.logger.WithContext(ctx).Warn("failed to decrypt restore token", logging.Error(err))
			// Continue with original token
		} else {
			data = decrypted
		}
	}

	var mappings []restoreMapping
	if err := json.Unmarshal([]byte(data), &mappings); err != nil {
		return nil, err
	}

	return mappings, nil
}

// restoreMapping represents a mapping for restoration
type restoreMapping struct {
	Type          string `json:"type"`
	OriginalValue string `json:"original_value"`
	Start         int    `json:"start"`
	End           int    `json:"end"`
}

// encryptPII encrypts PII entities in text
func (g *PrivacyGateway) encryptPII(ctx context.Context, text string, result *PIIDetectionResult) (string, error) {
	if len(result.Entities) == 0 {
		return text, nil
	}

	// Sort entities by start position (descending)
	entities := make([]*PIIEntity, len(result.Entities))
	copy(entities, result.Entities)

	for i := 0; i < len(entities)-1; i++ {
		for j := i + 1; j < len(entities); j++ {
			if entities[i].Start < entities[j].Start {
				entities[i], entities[j] = entities[j], entities[i]
			}
		}
	}

	encryptedText := text

	for _, entity := range entities {
		// Encrypt the PII value
		encrypted, err := g.encryptor.Encrypt(ctx, entity.Value)
		if err != nil {
			return "", err
		}

		// Replace with encrypted value
		replacement := fmt.Sprintf("[ENC:%s:%s]", entity.Type, encrypted)
		encryptedText = encryptedText[:entity.Start] + replacement + encryptedText[entity.End:]
	}

	return encryptedText, nil
}

// extractPIITypes extracts unique PII types from detection result
func (g *PrivacyGateway) extractPIITypes(result *PIIDetectionResult) []PIIType {
	if result == nil || len(result.Entities) == 0 {
		return []PIIType{}
	}

	typeMap := make(map[PIIType]bool)
	for _, entity := range result.Entities {
		typeMap[entity.Type] = true
	}

	types := make([]PIIType, 0, len(typeMap))
	for t := range typeMap {
		types = append(types, t)
	}

	return types
}

// generateEventID generates a unique event ID
func (g *PrivacyGateway) generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

// findPlaceholder finds a placeholder in text starting from a position
func findPlaceholder(text string, placeholder string, startPos int) int {
	// Search for the placeholder near the original position
	searchStart := startPos - 50
	if searchStart < 0 {
		searchStart = 0
	}

	searchEnd := startPos + 50
	if searchEnd > len(text) {
		searchEnd = len(text)
	}

	searchText := text[searchStart:searchEnd]
	idx := findString(searchText, placeholder)

	if idx >= 0 {
		return searchStart + idx
	}

	return -1
}

// findString finds the first occurrence of substr in str
func findString(str, substr string) int {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// recordMetrics records privacy gateway metrics
func (g *PrivacyGateway) recordMetrics(direction string, action PrivacyAction, hasPII bool, latency time.Duration) {
	if g.metricsCollector == nil {
		return
	}

	g.metricsCollector.IncrementCounter("privacy_gateway_requests_total",
		prometheus.Labels{
			"direction": direction,
			"action":    string(action),
			"has_pii":   fmt.Sprintf("%t", hasPII),
		})

	g.metricsCollector.ObserveHistogram("privacy_gateway_latency_ms", float64(latency.Milliseconds()),
		map[string]string{
			"direction": direction,
		})
}

// defaultPolicy returns default privacy policy
func defaultPolicy() *PrivacyPolicy {
	return &PrivacyPolicy{
		Action:           ActionRedact,
		EnableRestore:    true,
		MinRiskThreshold: 0.7,
		BlockedPIITypes: []PIIType{
			PIITypeSSN,
			PIITypeCreditCard,
		},
		EnableAuditLog:  true,
		RequireConsent:  false,
	}
}

// ValidatePolicy validates a privacy policy configuration
func ValidatePolicy(policy *PrivacyPolicy) error {
	if policy == nil {
		return errors.NewInternalError("PRIV_ERR", "policy cannot be nil")
	}

	if policy.MinRiskThreshold < 0 || policy.MinRiskThreshold > 1 {
		return errors.NewInternalError("PRIV_ERR", "min_risk_threshold must be between 0 and 1")
	}

	validActions := map[PrivacyAction]bool{
		ActionRedact:  true,
		ActionEncrypt: true,
		ActionBlock:   true,
		ActionAllow:   true,
	}

	if !validActions[policy.Action] {
		return errors.NewInternalError("PRIV_ERR", "invalid privacy action")
	}

	return nil
}

//Personal.AI order the ending
