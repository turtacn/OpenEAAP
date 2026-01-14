// Package validator provides unified parameter validation for OpenEAAP.
// It uses validator.v10 library and supports custom validation rules
// for use across API and application service layers.
package validator

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// ============================================================================
// Validator Instance
// ============================================================================

var (
	// Global validator instance
	validate *validator.Validate
	once     sync.Once
)

// Validator wraps go-playground validator with custom rules
type Validator struct {
	validator *validator.Validate
}

// ============================================================================
// Validator Initialization
// ============================================================================

// New creates a new validator instance with custom rules
func New() *Validator {
	v := validator.New()

	// Register custom tag name function
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	// Register custom validation rules
	registerCustomValidations(v)

	return &Validator{validator: v}
}

// GetValidator returns the global validator instance
func GetValidator() *Validator {
	once.Do(func() {
		validate = validator.New()
		registerCustomValidations(validate)
	})
	return &Validator{validator: validate}
}

// ============================================================================
// Validation Methods
// ============================================================================

// Validate validates a struct based on tags
func (v *Validator) Validate(i interface{}) error {
	if err := v.validator.Struct(i); err != nil {
		return v.formatValidationError(err)
	}
	return nil
}

// ValidateVar validates a single variable
func (v *Validator) ValidateVar(field interface{}, tag string) error {
	if err := v.validator.Var(field, tag); err != nil {
		return v.formatValidationError(err)
	}
	return nil
}

// ValidateMap validates a map based on rules
func (v *Validator) ValidateMap(data map[string]interface{}, rules map[string]interface{}) error {
	if err := v.validator.ValidateMap(data, rules); err != nil {
		return v.formatValidationError(err)
	}
	return nil
}

// ============================================================================
// Custom Validation Rules
// ============================================================================

// registerCustomValidations registers all custom validation rules
func registerCustomValidations(v *validator.Validate) {
	// Username validation
	_ = v.RegisterValidation("username", validateUsername)

	// Password strength validation
	_ = v.RegisterValidation("strong_password", validateStrongPassword)

	// Phone number validation
	_ = v.RegisterValidation("phone", validatePhone)

	// Agent ID validation
	_ = v.RegisterValidation("agent_id", validateAgentID)

	// Workflow ID validation
	_ = v.RegisterValidation("workflow_id", validateWorkflowID)

	// JSON string validation
	_ = v.RegisterValidation("json_string", validateJSONString)

	// UUID validation (v4)
	_ = v.RegisterValidation("uuid4", validateUUID4)

	// Slug validation
	_ = v.RegisterValidation("slug", validateSlug)

	// Hex color validation
	_ = v.RegisterValidation("hex_color", validateHexColor)

	// Cron expression validation
	_ = v.RegisterValidation("cron", validateCron)

	// ISO8601 date validation
	_ = v.RegisterValidation("iso8601", validateISO8601)

	// Semantic version validation
	_ = v.RegisterValidation("semver", validateSemVer)

	// Base64 validation
	_ = v.RegisterValidation("base64_custom", validateBase64)

	// MongoDB ObjectID validation
	_ = v.RegisterValidation("objectid", validateObjectID)

	// Domain name validation
	_ = v.RegisterValidation("domain", validateDomain)

	// API key validation
	_ = v.RegisterValidation("api_key", validateAPIKey)
}

// validateUsername validates username format
func validateUsername(fl validator.FieldLevel) bool {
	username := fl.Field().String()
	if len(username) < 3 || len(username) > 32 {
		return false
	}
	// Only alphanumeric, underscore, hyphen
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, username)
	return matched
}

// validateStrongPassword validates password strength
func validateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	// Minimum 8 characters
	if len(password) < 8 {
		return false
	}

	// Must contain at least one uppercase letter
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)

	// Must contain at least one lowercase letter
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)

	// Must contain at least one digit
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)

	// Must contain at least one special character
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)

	return hasUpper && hasLower && hasDigit && hasSpecial
}

// validatePhone validates phone number format
func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	// International format: +[country code][number]
	matched, _ := regexp.MatchString(`^\+?[1-9]\d{1,14}$`, phone)
	return matched
}

// validateAgentID validates agent ID format
func validateAgentID(fl validator.FieldLevel) bool {
	agentID := fl.Field().String()
	// Format: agent_[UUID]
	matched, _ := regexp.MatchString(`^agent_[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`, agentID)
	return matched
}

// validateWorkflowID validates workflow ID format
func validateWorkflowID(fl validator.FieldLevel) bool {
	workflowID := fl.Field().String()
	// Format: wf_[UUID]
	matched, _ := regexp.MatchString(`^wf_[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`, workflowID)
	return matched
}

// validateJSONString validates if string is valid JSON
func validateJSONString(fl validator.FieldLevel) bool {
	jsonStr := fl.Field().String()
	if jsonStr == "" {
		return true
	}
	// Simple JSON validation
	trimmed := strings.TrimSpace(jsonStr)
	return (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"))
}

// validateUUID4 validates UUID v4 format
func validateUUID4(fl validator.FieldLevel) bool {
	uuid := fl.Field().String()
	matched, _ := regexp.MatchString(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`, uuid)
	return matched
}

// validateSlug validates URL slug format
func validateSlug(fl validator.FieldLevel) bool {
	slug := fl.Field().String()
	matched, _ := regexp.MatchString(`^[a-z0-9]+(?:-[a-z0-9]+)*$`, slug)
	return matched
}

// validateHexColor validates hex color format
func validateHexColor(fl validator.FieldLevel) bool {
	color := fl.Field().String()
	matched, _ := regexp.MatchString(`^#[0-9A-Fa-f]{6}$`, color)
	return matched
}

// validateCron validates cron expression format
func validateCron(fl validator.FieldLevel) bool {
	cron := fl.Field().String()
	// Basic cron validation: 5 or 6 fields
	parts := strings.Fields(cron)
	return len(parts) >= 5 && len(parts) <= 6
}

// validateISO8601 validates ISO8601 date format
func validateISO8601(fl validator.FieldLevel) bool {
	date := fl.Field().String()
	matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$`, date)
	return matched
}

// validateSemVer validates semantic version format
func validateSemVer(fl validator.FieldLevel) bool {
	version := fl.Field().String()
	matched, _ := regexp.MatchString(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`, version)
	return matched
}

// validateBase64 validates base64 string
func validateBase64(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	matched, _ := regexp.MatchString(`^[A-Za-z0-9+/]*={0,2}$`, str)
	return matched && len(str)%4 == 0
}

// validateObjectID validates MongoDB ObjectID format
func validateObjectID(fl validator.FieldLevel) bool {
	id := fl.Field().String()
	matched, _ := regexp.MatchString(`^[a-f0-9]{24}$`, id)
	return matched
}

// validateDomain validates domain name format
func validateDomain(fl validator.FieldLevel) bool {
	domain := fl.Field().String()
	matched, _ := regexp.MatchString(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$`, domain)
	return matched
}

// validateAPIKey validates API key format
func validateAPIKey(fl validator.FieldLevel) bool {
	apiKey := fl.Field().String()
	// Format: sk_[base64]
	matched, _ := regexp.MatchString(`^sk_[A-Za-z0-9_-]{32,}$`, apiKey)
	return matched
}

// ============================================================================
// Error Formatting
// ============================================================================

// ValidationError represents a formatted validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Tag     string `json:"tag"`
	Value   string `json:"value,omitempty"`
}

// formatValidationError formats validation errors into readable messages
func (v *Validator) formatValidationError(err error) error {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var errors []ValidationError

		for _, e := range validationErrors {
			errors = append(errors, ValidationError{
				Field:   e.Field(),
				Message: getErrorMessage(e),
				Tag:     e.Tag(),
				Value:   fmt.Sprintf("%v", e.Value()),
			})
		}

		return &FormattedValidationError{Errors: errors}
	}

	return err
}

// FormattedValidationError contains multiple validation errors
type FormattedValidationError struct {
	Errors []ValidationError `json:"errors"`
}

// Error implements error interface
func (f *FormattedValidationError) Error() string {
	var messages []string
	for _, e := range f.Errors {
		messages = append(messages, e.Message)
	}
	return strings.Join(messages, "; ")
}

// getErrorMessage returns human-readable error message for validation tag
func getErrorMessage(fe validator.FieldError) string {
	field := fe.Field()

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", field, fe.Param())
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters long", field, fe.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, fe.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, fe.Param())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, fe.Param())
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, fe.Param())
	case "alpha":
		return fmt.Sprintf("%s must contain only alphabetic characters", field)
	case "alphanum":
		return fmt.Sprintf("%s must contain only alphanumeric characters", field)
	case "numeric":
		return fmt.Sprintf("%s must be numeric", field)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "uri":
		return fmt.Sprintf("%s must be a valid URI", field)
	case "username":
		return fmt.Sprintf("%s must be 3-32 characters and contain only letters, numbers, underscores, or hyphens", field)
	case "strong_password":
		return fmt.Sprintf("%s must be at least 8 characters and contain uppercase, lowercase, number, and special character", field)
	case "phone":
		return fmt.Sprintf("%s must be a valid phone number", field)
	case "agent_id":
		return fmt.Sprintf("%s must be a valid agent ID (format: agent_[UUID])", field)
	case "workflow_id":
		return fmt.Sprintf("%s must be a valid workflow ID (format: wf_[UUID])", field)
	case "json_string":
		return fmt.Sprintf("%s must be valid JSON", field)
	case "uuid4":
		return fmt.Sprintf("%s must be a valid UUID v4", field)
	case "slug":
		return fmt.Sprintf("%s must be a valid slug", field)
	case "hex_color":
		return fmt.Sprintf("%s must be a valid hex color", field)
	case "cron":
		return fmt.Sprintf("%s must be a valid cron expression", field)
	case "iso8601":
		return fmt.Sprintf("%s must be in ISO8601 format", field)
	case "semver":
		return fmt.Sprintf("%s must be a valid semantic version", field)
	case "base64_custom":
		return fmt.Sprintf("%s must be valid base64", field)
	case "objectid":
		return fmt.Sprintf("%s must be a valid MongoDB ObjectID", field)
	case "domain":
		return fmt.Sprintf("%s must be a valid domain name", field)
	case "api_key":
		return fmt.Sprintf("%s must be a valid API key", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, fe.Param())
	case "eqfield":
		return fmt.Sprintf("%s must be equal to %s", field, fe.Param())
	case "nefield":
		return fmt.Sprintf("%s must not be equal to %s", field, fe.Param())
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}

// ============================================================================
// Utility Functions
// ============================================================================

// RegisterCustomValidation registers a custom validation function
func (v *Validator) RegisterCustomValidation(tag string, fn validator.Func) error {
	return v.validator.RegisterValidation(tag, fn)
}

// RegisterStructValidation registers a custom struct validation function
func (v *Validator) RegisterStructValidation(fn validator.StructLevelFunc, types ...interface{}) {
	v.validator.RegisterStructValidation(fn, types...)
}

// ValidateStruct is a convenience function for validating structs
func ValidateStruct(s interface{}) error {
	return GetValidator().Validate(s)
}

// ValidateField is a convenience function for validating single fields
func ValidateField(field interface{}, tag string) error {
	return GetValidator().ValidateVar(field, tag)
}

// ============================================================================
// Common Validation Helpers
// ============================================================================

// IsEmail validates email format
func IsEmail(email string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, email)
	return matched
}

// IsURL validates URL format
func IsURL(url string) bool {
	matched, _ := regexp.MatchString(`^https?://[^\s/$.?#].[^\s]*$`, url)
	return matched
}

// IsUUID validates UUID format (any version)
func IsUUID(uuid string) bool {
	matched, _ := regexp.MatchString(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`, uuid)
	return matched
}

// IsAlphanumeric checks if string is alphanumeric
func IsAlphanumeric(s string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9]+$`, s)
	return matched
}

// IsNumeric checks if string is numeric
func IsNumeric(s string) bool {
	matched, _ := regexp.MatchString(`^[0-9]+$`, s)
	return matched
}

// InRange checks if value is within range
func InRange(value, min, max int) bool {
	return value >= min && value <= max
}

// IsOneOf checks if value is in allowed list
func IsOneOf(value string, allowed []string) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	return false
}

//Personal.AI order the ending
