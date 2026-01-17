package errors

import "net/http"

// Helper functions for common error types to simplify error creation

// NewValidationError creates a validation error
func NewValidationError(code, message string) *AppError {
	return New(code, ErrorTypeValidation, message, http.StatusBadRequest)
}

// NewAuthenticationError creates an authentication error
func NewAuthenticationError(code, message string) *AppError {
	return New(code, ErrorTypeAuthentication, message, http.StatusUnauthorized)
}

// NewAuthorizationError creates an authorization error
func NewAuthorizationError(code, message string) *AppError {
	return New(code, ErrorTypeAuthorization, message, http.StatusForbidden)
}

// NewNotFoundError creates a not found error
func NewNotFoundError(code, message string) *AppError {
	return New(code, ErrorTypeNotFound, message, http.StatusNotFound)
}

// NewConflictError creates a conflict error
func NewConflictError(code, message string) *AppError {
	return New(code, ErrorTypeConflict, message, http.StatusConflict)
}

// NewDatabaseError creates a database error
func NewDatabaseError(code, message string) *AppError {
	return New(code, ErrorTypeInfrastructure, message, http.StatusInternalServerError)
}

// NewInternalError creates an internal error
func NewInternalError(code, message string) *AppError {
	return New(code, ErrorTypeInternal, message, http.StatusInternalServerError)
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(code, message string) *AppError {
	return New(code, ErrorTypeTimeout, message, http.StatusGatewayTimeout)
}

// WrapDatabaseError wraps an existing error as database error
func WrapDatabaseError(err error, code, message string) *AppError {
	return NewDatabaseError(code, message).WithCause(err)
}

// WrapInternalError wraps an existing error as internal error
func WrapInternalError(err error, code, message string) *AppError {
	return NewInternalError(code, message).WithCause(err)
}

// Common error codes as constants
const (
	CodeInvalidParameter = "INVALID_PARAMETER"
	CodeInvalidArgument  = "INVALID_ARGUMENT"
	CodeDatabaseError    = "DATABASE_ERROR"
	CodeInternalError    = "INTERNAL_ERROR"
	CodeNotFound         = "NOT_FOUND"
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeForbidden        = "FORBIDDEN"
	CodeConflict         = "CONFLICT"
	CodeTimeout          = "TIMEOUT"
)
