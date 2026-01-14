// Package errors provides a unified error handling mechanism for OpenEAAP.
// It defines a structured error system with error codes, types, and helpful
// formatting capabilities to standardize error handling across the platform.
package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
)

// ErrorType represents the category of error
type ErrorType string

const (
	// ErrorTypeValidation indicates invalid input or parameters
	ErrorTypeValidation ErrorType = "VALIDATION"

	// ErrorTypeAuthentication indicates authentication failure
	ErrorTypeAuthentication ErrorType = "AUTHENTICATION"

	// ErrorTypeAuthorization indicates authorization/permission failure
	ErrorTypeAuthorization ErrorType = "AUTHORIZATION"

	// ErrorTypeNotFound indicates resource not found
	ErrorTypeNotFound ErrorType = "NOT_FOUND"

	// ErrorTypeConflict indicates resource conflict (e.g., duplicate)
	ErrorTypeConflict ErrorType = "CONFLICT"

	// ErrorTypeBusiness indicates business logic error
	ErrorTypeBusiness ErrorType = "BUSINESS"

	// ErrorTypeInfrastructure indicates infrastructure/external service error
	ErrorTypeInfrastructure ErrorType = "INFRASTRUCTURE"

	// ErrorTypeInternal indicates unexpected internal error
	ErrorTypeInternal ErrorType = "INTERNAL"

	// ErrorTypeRateLimited indicates rate limit exceeded
	ErrorTypeRateLimited ErrorType = "RATE_LIMITED"

	// ErrorTypeTimeout indicates operation timeout
	ErrorTypeTimeout ErrorType = "TIMEOUT"
)

// AppError represents a structured application error
type AppError struct {
	// Code is the error code (e.g., "AGENT_001")
	Code string `json:"code"`

	// Type categorizes the error
	Type ErrorType `json:"type"`

	// Message is the human-readable error message
	Message string `json:"message"`

	// HTTPStatus is the corresponding HTTP status code
	HTTPStatus int `json:"-"`

	// Details contains additional error context
	Details map[string]interface{} `json:"details,omitempty"`

	// Cause is the underlying error
	Cause error `json:"-"`

	// Stack contains the stack trace (for internal errors)
	Stack string `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for error chain unwrapping
func (e *AppError) Unwrap() error {
	return e.Cause
}

// WithDetails adds additional context to the error
func (e *AppError) WithDetails(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithCause wraps an underlying error
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// ToJSON serializes the error to JSON for API responses
func (e *AppError) ToJSON() []byte {
	data, _ := json.Marshal(e)
	return data
}

// New creates a new AppError
func New(code string, errType ErrorType, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Type:       errType,
		Message:    message,
		HTTPStatus: httpStatus,
		Details:    make(map[string]interface{}),
	}
}

// Newf creates a new AppError with formatted message
func Newf(code string, errType ErrorType, httpStatus int, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       code,
		Type:       errType,
		Message:    fmt.Sprintf(format, args...),
		HTTPStatus: httpStatus,
		Details:    make(map[string]interface{}),
	}
}

// Wrap wraps an existing error with AppError context
func Wrap(err error, code string, message string) *AppError {
	if err == nil {
		return nil
	}

	// If already an AppError, add context
	if appErr, ok := err.(*AppError); ok {
		return &AppError{
			Code:       code,
			Type:       appErr.Type,
			Message:    message,
			HTTPStatus: appErr.HTTPStatus,
			Cause:      appErr,
			Details:    make(map[string]interface{}),
		}
	}

	return &AppError{
		Code:       code,
		Type:       ErrorTypeInternal,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Cause:      err,
		Details:    make(map[string]interface{}),
	}
}

// WrapWithStack wraps an error and captures stack trace
func WrapWithStack(err error, code string, message string) *AppError {
	appErr := Wrap(err, code, message)
	if appErr != nil {
		appErr.Stack = captureStack()
	}
	return appErr
}

// captureStack captures the current stack trace
func captureStack() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// Is checks if an error matches a specific code
func Is(err error, code string) bool {
	if err == nil {
		return false
	}

	appErr, ok := err.(*AppError)
	if !ok {
		return false
	}

	return appErr.Code == code
}

// IsType checks if an error matches a specific type
func IsType(err error, errType ErrorType) bool {
	if err == nil {
		return false
	}

	appErr, ok := err.(*AppError)
	if !ok {
		return false
	}

	return appErr.Type == errType
}

// GetCode extracts the error code from an error
func GetCode(err error) string {
	if err == nil {
		return ""
	}

	appErr, ok := err.(*AppError)
	if !ok {
		return "UNKNOWN"
	}

	return appErr.Code
}

// GetHTTPStatus extracts the HTTP status code from an error
func GetHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	appErr, ok := err.(*AppError)
	if !ok {
		return http.StatusInternalServerError
	}

	return appErr.HTTPStatus
}

// Common error constructors for frequent use cases

// ValidationError creates a validation error
func ValidationError(message string) *AppError {
	return New("VALIDATION_ERROR", ErrorTypeValidation, message, http.StatusBadRequest)
}

// ValidationErrorf creates a validation error with formatted message
func ValidationErrorf(format string, args ...interface{}) *AppError {
	return Newf("VALIDATION_ERROR", ErrorTypeValidation, http.StatusBadRequest, format, args...)
}

// NotFoundError creates a not found error
func NotFoundError(resource string) *AppError {
	return Newf("NOT_FOUND", ErrorTypeNotFound, http.StatusNotFound, "%s not found", resource)
}

// UnauthorizedError creates an authentication error
func UnauthorizedError(message string) *AppError {
	return New("UNAUTHORIZED", ErrorTypeAuthentication, message, http.StatusUnauthorized)
}

// ForbiddenError creates an authorization error
func ForbiddenError(message string) *AppError {
	return New("FORBIDDEN", ErrorTypeAuthorization, message, http.StatusForbidden)
}

// ConflictError creates a conflict error
func ConflictError(message string) *AppError {
	return New("CONFLICT", ErrorTypeConflict, message, http.StatusConflict)
}

// InternalError creates an internal server error
func InternalError(message string) *AppError {
	appErr := New("INTERNAL_ERROR", ErrorTypeInternal, message, http.StatusInternalServerError)
	appErr.Stack = captureStack()
	return appErr
}

// InternalErrorf creates an internal server error with formatted message
func InternalErrorf(format string, args ...interface{}) *AppError {
	appErr := Newf("INTERNAL_ERROR", ErrorTypeInternal, http.StatusInternalServerError, format, args...)
	appErr.Stack = captureStack()
	return appErr
}

// TimeoutError creates a timeout error
func TimeoutError(operation string) *AppError {
	return Newf("TIMEOUT", ErrorTypeTimeout, http.StatusRequestTimeout, "Operation '%s' timed out", operation)
}

// RateLimitError creates a rate limit error
func RateLimitError(message string) *AppError {
	return New("RATE_LIMITED", ErrorTypeRateLimited, message, http.StatusTooManyRequests)
}

// InfrastructureError creates an infrastructure error
func InfrastructureError(service string, err error) *AppError {
	return Wrap(err, "INFRASTRUCTURE_ERROR", fmt.Sprintf("Infrastructure service '%s' error", service))
}

//Personal.AI order the ending
