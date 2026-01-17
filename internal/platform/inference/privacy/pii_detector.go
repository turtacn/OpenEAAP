// Package privacy provides PII (Personally Identifiable Information) detection
// capabilities using regex patterns and ML models.
package privacy

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/pkg/errors"
)

// PIIType represents the type of PII detected
type PIIType string

const (
	// PIITypeName represents a person's name
	PIITypeName PIIType = "name"

	// PIITypeEmail represents an email address
	PIITypeEmail PIIType = "email"

	// PIITypePhone represents a phone number
	PIITypePhone PIIType = "phone"

	// PIITypeSSN represents a social security number
	PIITypeSSN PIIType = "ssn"

	// PIITypeIDCard represents an ID card number
	PIITypeIDCard PIIType = "id_card"

	// PIITypeCreditCard represents a credit card number
	PIITypeCreditCard PIIType = "credit_card"

	// PIITypeAddress represents a physical address
	PIITypeAddress PIIType = "address"

	// PIITypeDateOfBirth represents a date of birth
	PIITypeDateOfBirth PIIType = "date_of_birth"

	// PIITypePassport represents a passport number
	PIITypePassport PIIType = "passport"

	// PIITypeIPAddress represents an IP address
	PIITypeIPAddress PIIType = "ip_address"
)

// PIIEntity represents a detected PII entity
type PIIEntity struct {
	Type       PIIType `json:"type"`
	Value      string  `json:"value"`
	Start      int     `json:"start"`
	End        int     `json:"end"`
	Confidence float64 `json:"confidence"`
	Context    string  `json:"context,omitempty"`
}

// PIIDetectionResult represents the result of PII detection
type PIIDetectionResult struct {
	Text          string       `json:"text"`
	Entities      []*PIIEntity `json:"entities"`
	HasPII        bool         `json:"has_pii"`
	RiskScore     float64      `json:"risk_score"`
	RedactedText  string       `json:"redacted_text,omitempty"`
}

// DetectorConfig contains configuration for PII detector
type DetectorConfig struct {
	EnableNameDetection      bool
	EnableEmailDetection     bool
	EnablePhoneDetection     bool
	EnableSSNDetection       bool
	EnableIDCardDetection    bool
	EnableCreditCardDetection bool
	EnableAddressDetection   bool
	EnableDOBDetection       bool
	EnablePassportDetection  bool
	EnableIPDetection        bool

	MinConfidence            float64
	EnableMLModel            bool
	ModelPath                string

	CustomPatterns           map[PIIType]*regexp.Regexp
}

// PIIDetector detects PII in text
type PIIDetector struct {
	logger  logging.Logger
	config  *DetectorConfig

	patterns map[PIIType]*regexp.Regexp
	mu       sync.RWMutex

	// ML model (placeholder for actual model)
	mlModel interface{}
}

// NewPIIDetector creates a new PII detector
func NewPIIDetector(logger logging.Logger, config *DetectorConfig) (*PIIDetector, error) {
	if config == nil {
		config = defaultConfig()
	}

	detector := &PIIDetector{
		logger:   logger,
		config:   config,
		patterns: make(map[PIIType]*regexp.Regexp),
	}

	// Initialize regex patterns
	if err := detector.initializePatterns(); err != nil {
		return nil, errors.WrapInternalError(err, "ERR_INTERNAL", "failed to initialize patterns")
	}

	// Load ML model if enabled
	if config.EnableMLModel && config.ModelPath != "" {
		if err := detector.loadMLModel(config.ModelPath); err != nil {
			logger.WithContext(context.Background()).Warn("failed to load ML model", logging.Error(err))
		}
	}

	logger.WithContext(context.Background()).Info("PII detector initialized")

	return detector, nil
}

// Detect detects PII in the given text
func (d *PIIDetector) Detect(ctx context.Context, text string) (*PIIDetectionResult, error) {
	if text == "" {
		return &PIIDetectionResult{
			Text:     text,
			Entities: []*PIIEntity{},
			HasPII:   false,
		}, nil
	}

	entities := make([]*PIIEntity, 0)

	// Run regex-based detection
	regexEntities := d.detectWithRegex(ctx, text)
	entities = append(entities, regexEntities...)

	// Run ML-based detection if enabled
	if d.config.EnableMLModel && d.mlModel != nil {
		mlEntities := d.detectWithML(ctx, text)
		entities = append(entities, mlEntities...)
	}

	// Merge overlapping entities
	entities = d.mergeEntities(entities)

	// Filter by confidence threshold
	filteredEntities := make([]*PIIEntity, 0)
	for _, entity := range entities {
		if entity.Confidence >= d.config.MinConfidence {
			filteredEntities = append(filteredEntities, entity)
		}
	}

	// Calculate risk score
	riskScore := d.calculateRiskScore(filteredEntities)

	// Generate redacted text
	redactedText := d.redactText(text, filteredEntities)

	result := &PIIDetectionResult{
		Text:         text,
		Entities:     filteredEntities,
		HasPII:       len(filteredEntities) > 0,
		RiskScore:    riskScore,
		RedactedText: redactedText,
	}

	d.logger.WithContext(ctx).Debug("PII detection completed",
		logging.Int("entity_count", len(filteredEntities)),
		logging.Bool("has_pii", result.HasPII),
		logging.Float64("risk_score", riskScore),
	)

	return result, nil
}

// DetectBatch detects PII in multiple texts
func (d *PIIDetector) DetectBatch(ctx context.Context, texts []string) ([]*PIIDetectionResult, error) {
	results := make([]*PIIDetectionResult, len(texts))

	for i, text := range texts {
		result, err := d.Detect(ctx, text)
		if err != nil {
			return nil, errors.WrapInternalError(err, "ERR_INTERNAL", "batch detection failed")
		}
		results[i] = result
	}

	return results, nil
}

// detectWithRegex detects PII using regex patterns
func (d *PIIDetector) detectWithRegex(ctx context.Context, text string) []*PIIEntity {
	d.mu.RLock()
	defer d.mu.RUnlock()

	entities := make([]*PIIEntity, 0)

	for piiType, pattern := range d.patterns {
		matches := pattern.FindAllStringIndex(text, -1)

		for _, match := range matches {
			start, end := match[0], match[1]
			value := text[start:end]

			// Calculate confidence based on pattern match and context
			confidence := d.calculateRegexConfidence(piiType, value, text, start, end)

			if confidence >= d.config.MinConfidence {
				entity := &PIIEntity{
					Type:       piiType,
					Value:      value,
					Start:      start,
					End:        end,
					Confidence: confidence,
					Context:    d.extractContext(text, start, end),
				}

				entities = append(entities, entity)
			}
		}
	}

	return entities
}

// detectWithML detects PII using ML model (placeholder)
func (d *PIIDetector) detectWithML(ctx context.Context, text string) []*PIIEntity {
	// Placeholder for ML-based detection
	// In production, this would use a trained NER model
	entities := make([]*PIIEntity, 0)

	// Example: Named Entity Recognition using ML model
	// tokens := tokenize(text)
	// predictions := d.mlModel.Predict(tokens)
	// entities = convertPredictionsToEntities(predictions)

	return entities
}

// initializePatterns initializes regex patterns for different PII types
func (d *PIIDetector) initializePatterns() error {
	// Email pattern
	if d.config.EnableEmailDetection {
		d.patterns[PIITypeEmail] = regexp.MustCompile(
			`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
		)
	}

	// Phone number patterns (US format)
	if d.config.EnablePhoneDetection {
		d.patterns[PIITypePhone] = regexp.MustCompile(
			`\b(?:\+?1[-.\s]?)?\(?([0-9]{3})\)?[-.\s]?([0-9]{3})[-.\s]?([0-9]{4})\b`,
		)
	}

	// SSN pattern (US)
	if d.config.EnableSSNDetection {
		d.patterns[PIITypeSSN] = regexp.MustCompile(
			`\b(?!000|666|9\d{2})\d{3}-(?!00)\d{2}-(?!0000)\d{4}\b`,
		)
	}

	// Credit card pattern
	if d.config.EnableCreditCardDetection {
		d.patterns[PIITypeCreditCard] = regexp.MustCompile(
			`\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|6(?:011|5[0-9]{2})[0-9]{12}|(?:2131|1800|35\d{3})\d{11})\b`,
		)
	}

	// Chinese ID card pattern
	if d.config.EnableIDCardDetection {
		d.patterns[PIITypeIDCard] = regexp.MustCompile(
			`\b[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]\b`,
		)
	}

	// Date of birth pattern
	if d.config.EnableDOBDetection {
		d.patterns[PIITypeDateOfBirth] = regexp.MustCompile(
			`\b(0[1-9]|1[0-2])/(0[1-9]|[12][0-9]|3[01])/(19|20)\d{2}\b`,
		)
	}

	// IP address pattern
	if d.config.EnableIPDetection {
		d.patterns[PIITypeIPAddress] = regexp.MustCompile(
			`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`,
		)
	}

	// Passport pattern (simplified)
	if d.config.EnablePassportDetection {
		d.patterns[PIITypePassport] = regexp.MustCompile(
			`\b[A-Z]{1,2}[0-9]{6,9}\b`,
		)
	}

	// Add custom patterns
	if d.config.CustomPatterns != nil {
		for piiType, pattern := range d.config.CustomPatterns {
			d.patterns[piiType] = pattern
		}
	}

	return nil
}

// calculateRegexConfidence calculates confidence score for regex match
func (d *PIIDetector) calculateRegexConfidence(piiType PIIType, value string, text string, start, end int) float64 {
	baseConfidence := 0.8

	// Adjust confidence based on PII type
	switch piiType {
	case PIITypeEmail:
		// Validate email format more strictly
		if strings.Count(value, "@") == 1 && strings.Contains(value, ".") {
			baseConfidence = 0.95
		}

	case PIITypePhone:
		// Check if phone number has consistent formatting
		if strings.Contains(value, "-") || strings.Contains(value, "(") {
			baseConfidence = 0.9
		}

	case PIITypeSSN:
		// SSN has strict format, high confidence
		baseConfidence = 0.95

	case PIITypeCreditCard:
		// Validate using Luhn algorithm
		if d.luhnCheck(value) {
			baseConfidence = 0.95
		} else {
			baseConfidence = 0.6
		}

	case PIITypeIDCard:
		// Validate Chinese ID card checksum
		if d.validateChineseIDCard(value) {
			baseConfidence = 0.95
		} else {
			baseConfidence = 0.7
		}
	}

	// Adjust based on context
	context := d.extractContext(text, start, end)
	if d.hasRelevantContext(piiType, context) {
		baseConfidence += 0.05
	}

	// Cap at 1.0
	if baseConfidence > 1.0 {
		baseConfidence = 1.0
	}

	return baseConfidence
}

// luhnCheck validates a number using the Luhn algorithm (for credit cards)
func (d *PIIDetector) luhnCheck(number string) bool {
	// Remove non-digits
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, number)

	if len(cleaned) < 13 || len(cleaned) > 19 {
		return false
	}

	sum := 0
	alternate := false

	for i := len(cleaned) - 1; i >= 0; i-- {
		digit := int(cleaned[i] - '0')

		if alternate {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		alternate = !alternate
	}

	return sum%10 == 0
}

// validateChineseIDCard validates Chinese ID card checksum
func (d *PIIDetector) validateChineseIDCard(idCard string) bool {
	if len(idCard) != 18 {
		return false
	}

	// Weights for checksum calculation
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	checkCodes := []byte{'1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'}

	sum := 0
	for i := 0; i < 17; i++ {
		digit := int(idCard[i] - '0')
		sum += digit * weights[i]
	}

	checkCode := checkCodes[sum%11]
	lastChar := idCard[17]

	return lastChar == checkCode || (checkCode == 'X' && lastChar == 'x')
}

// hasRelevantContext checks if the context contains relevant keywords
func (d *PIIDetector) hasRelevantContext(piiType PIIType, context string) bool {
	contextLower := strings.ToLower(context)

	keywords := map[PIIType][]string{
		PIITypeEmail:      {"email", "e-mail", "contact", "address"},
		PIITypePhone:      {"phone", "tel", "mobile", "call", "contact"},
		PIITypeSSN:        {"ssn", "social security", "tax id"},
		PIITypeCreditCard: {"card", "credit", "payment", "visa", "mastercard"},
		PIITypeIDCard:     {"id", "identification", "identity"},
		PIITypeName:       {"name", "mr", "mrs", "ms", "dr"},
	}

	if words, ok := keywords[piiType]; ok {
		for _, word := range words {
			if strings.Contains(contextLower, word) {
				return true
			}
		}
	}

	return false
}

// extractContext extracts surrounding context for an entity
func (d *PIIDetector) extractContext(text string, start, end int) string {
	contextSize := 30

	contextStart := start - contextSize
	if contextStart < 0 {
		contextStart = 0
	}

	contextEnd := end + contextSize
	if contextEnd > len(text) {
		contextEnd = len(text)
	}

	return text[contextStart:contextEnd]
}

// mergeEntities merges overlapping entities
func (d *PIIDetector) mergeEntities(entities []*PIIEntity) []*PIIEntity {
	if len(entities) <= 1 {
		return entities
	}

	// Sort by start position
	for i := 0; i < len(entities)-1; i++ {
		for j := i + 1; j < len(entities); j++ {
			if entities[i].Start > entities[j].Start {
				entities[i], entities[j] = entities[j], entities[i]
			}
		}
	}

	merged := make([]*PIIEntity, 0)
	current := entities[0]

	for i := 1; i < len(entities); i++ {
		next := entities[i]

		// Check for overlap
		if next.Start <= current.End {
			// Keep entity with higher confidence
			if next.Confidence > current.Confidence {
				current = next
			} else if next.End > current.End {
				current.End = next.End
			}
		} else {
			merged = append(merged, current)
			current = next
		}
	}

	merged = append(merged, current)

	return merged
}

// calculateRiskScore calculates overall risk score based on detected entities
func (d *PIIDetector) calculateRiskScore(entities []*PIIEntity) float64 {
	if len(entities) == 0 {
		return 0.0
	}

	// Risk weights for different PII types
	riskWeights := map[PIIType]float64{
		PIITypeSSN:            1.0,
		PIITypeCreditCard:     1.0,
		PIITypeIDCard:         0.9,
		PIITypePassport:       0.9,
		PIITypeDateOfBirth:    0.7,
		PIITypePhone:          0.6,
		PIITypeEmail:          0.5,
		PIITypeAddress:        0.6,
		PIITypeName:           0.4,
		PIITypeIPAddress:      0.3,
	}

	totalRisk := 0.0

	for _, entity := range entities {
		weight := riskWeights[entity.Type]
		if weight == 0 {
			weight = 0.5 // Default weight
		}

		totalRisk += weight * entity.Confidence
	}

	// Normalize to 0-1 range
	riskScore := totalRisk / float64(len(entities))

	if riskScore > 1.0 {
		riskScore = 1.0
	}

	return riskScore
}

// redactText redacts PII entities from text
func (d *PIIDetector) redactText(text string, entities []*PIIEntity) string {
	if len(entities) == 0 {
		return text
	}

	// Sort entities by start position (descending) to avoid offset issues
	sortedEntities := make([]*PIIEntity, len(entities))
	copy(sortedEntities, entities)

	for i := 0; i < len(sortedEntities)-1; i++ {
		for j := i + 1; j < len(sortedEntities); j++ {
			if sortedEntities[i].Start < sortedEntities[j].Start {
				sortedEntities[i], sortedEntities[j] = sortedEntities[j], sortedEntities[i]
			}
		}
	}

	redacted := text

	for _, entity := range sortedEntities {
		replacement := fmt.Sprintf("[%s]", strings.ToUpper(string(entity.Type)))
		redacted = redacted[:entity.Start] + replacement + redacted[entity.End:]
	}

	return redacted
}

// loadMLModel loads the ML model for PII detection (placeholder)
func (d *PIIDetector) loadMLModel(modelPath string) error {
	// Placeholder for loading an actual ML model
	// In production, this would load a trained NER model
	d.logger.Info("ML model loading not implemented", logging.String("path", modelPath))
	return nil
}

// defaultConfig returns default detector configuration
func defaultConfig() *DetectorConfig {
	return &DetectorConfig{
		EnableNameDetection:       true,
		EnableEmailDetection:      true,
		EnablePhoneDetection:      true,
		EnableSSNDetection:        true,
		EnableIDCardDetection:     true,
		EnableCreditCardDetection: true,
		EnableAddressDetection:    true,
		EnableDOBDetection:        true,
		EnablePassportDetection:   true,
		EnableIPDetection:         true,
		MinConfidence:             0.7,
		EnableMLModel:             false,
	}
}

//Personal.AI order the ending
