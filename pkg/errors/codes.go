// Package errors defines error code constants for OpenEAAP.
// Each error code includes a unique identifier, HTTP status code,
// and message template for consistent error handling across the platform.
package errors

import "net/http"

// ErrorCode represents a structured error code definition
type ErrorCode struct {
	Code       string
	Type       ErrorType
	HTTPStatus int
	Message    string
}

// Standard error codes organized by category

// ============================================================================
// Authentication & Authorization Errors (AUTH_xxx)
// ============================================================================

var (
	// ErrAuthInvalidToken indicates an invalid or expired authentication token
	ErrAuthInvalidToken = ErrorCode{
		Code:       "AUTH_001",
		Type:       ErrorTypeAuthentication,
		HTTPStatus: http.StatusUnauthorized,
		Message:    "Invalid or expired authentication token",
	}

	// ErrAuthMissingToken indicates missing authentication token
	ErrAuthMissingToken = ErrorCode{
		Code:       "AUTH_002",
		Type:       ErrorTypeAuthentication,
		HTTPStatus: http.StatusUnauthorized,
		Message:    "Authentication token is required",
	}

	// ErrAuthInvalidCredentials indicates invalid username or password
	ErrAuthInvalidCredentials = ErrorCode{
		Code:       "AUTH_003",
		Type:       ErrorTypeAuthentication,
		HTTPStatus: http.StatusUnauthorized,
		Message:    "Invalid credentials",
	}

	// ErrAuthAccountLocked indicates account is locked due to security reasons
	ErrAuthAccountLocked = ErrorCode{
		Code:       "AUTH_004",
		Type:       ErrorTypeAuthentication,
		HTTPStatus: http.StatusForbidden,
		Message:    "Account is locked",
	}

	// ErrAuthInsufficientPermissions indicates insufficient permissions
	ErrAuthInsufficientPermissions = ErrorCode{
		Code:       "AUTH_005",
		Type:       ErrorTypeAuthorization,
		HTTPStatus: http.StatusForbidden,
		Message:    "Insufficient permissions to perform this operation",
	}

	// ErrAuthResourceAccessDenied indicates access to specific resource is denied
	ErrAuthResourceAccessDenied = ErrorCode{
		Code:       "AUTH_006",
		Type:       ErrorTypeAuthorization,
		HTTPStatus: http.StatusForbidden,
		Message:    "Access to this resource is denied",
	}

	// ErrAuthPolicyViolation indicates policy engine denied the request
	ErrAuthPolicyViolation = ErrorCode{
		Code:       "AUTH_007",
		Type:       ErrorTypeAuthorization,
		HTTPStatus: http.StatusForbidden,
		Message:    "Request violates security policy",
	}
)

// ============================================================================
// Validation Errors (VAL_xxx)
// ============================================================================

var (
	// ErrValInvalidInput indicates general invalid input
	ErrValInvalidInput = ErrorCode{
		Code:       "VAL_001",
		Type:       ErrorTypeValidation,
		HTTPStatus: http.StatusBadRequest,
		Message:    "Invalid input parameters",
	}

	// ErrValMissingRequiredField indicates required field is missing
	ErrValMissingRequiredField = ErrorCode{
		Code:       "VAL_002",
		Type:       ErrorTypeValidation,
		HTTPStatus: http.StatusBadRequest,
		Message:    "Required field is missing",
	}

	// ErrValInvalidFormat indicates field format is invalid
	ErrValInvalidFormat = ErrorCode{
		Code:       "VAL_003",
		Type:       ErrorTypeValidation,
		HTTPStatus: http.StatusBadRequest,
		Message:    "Field format is invalid",
	}

	// ErrValOutOfRange indicates value is out of acceptable range
	ErrValOutOfRange = ErrorCode{
		Code:       "VAL_004",
		Type:       ErrorTypeValidation,
		HTTPStatus: http.StatusBadRequest,
		Message:    "Value is out of acceptable range",
	}

	// ErrValInvalidJSON indicates malformed JSON
	ErrValInvalidJSON = ErrorCode{
		Code:       "VAL_005",
		Type:       ErrorTypeValidation,
		HTTPStatus: http.StatusBadRequest,
		Message:    "Invalid JSON format",
	}

	// ErrValInvalidYAML indicates malformed YAML
	ErrValInvalidYAML = ErrorCode{
		Code:       "VAL_006",
		Type:       ErrorTypeValidation,
		HTTPStatus: http.StatusBadRequest,
		Message:    "Invalid YAML format",
	}
)

// ============================================================================
// Agent Errors (AGENT_xxx)
// ============================================================================

var (
	// ErrAgentNotFound indicates agent does not exist
	ErrAgentNotFound = ErrorCode{
		Code:       "AGENT_001",
		Type:       ErrorTypeNotFound,
		HTTPStatus: http.StatusNotFound,
		Message:    "Agent not found",
	}

	// ErrAgentAlreadyExists indicates agent with same name already exists
	ErrAgentAlreadyExists = ErrorCode{
		Code:       "AGENT_002",
		Type:       ErrorTypeConflict,
		HTTPStatus: http.StatusConflict,
		Message:    "Agent with this name already exists",
	}

	// ErrAgentInvalidConfig indicates agent configuration is invalid
	ErrAgentInvalidConfig = ErrorCode{
		Code:       "AGENT_003",
		Type:       ErrorTypeValidation,
		HTTPStatus: http.StatusBadRequest,
		Message:    "Invalid agent configuration",
	}

	// ErrAgentExecutionFailed indicates agent execution failed
	ErrAgentExecutionFailed = ErrorCode{
		Code:       "AGENT_004",
		Type:       ErrorTypeBusiness,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "Agent execution failed",
	}

	// ErrAgentNotActive indicates agent is not in active state
	ErrAgentNotActive = ErrorCode{
		Code:       "AGENT_005",
		Type:       ErrorTypeBusiness,
		HTTPStatus: http.StatusConflict,
		Message:    "Agent is not active",
	}

	// ErrAgentDeploymentFailed indicates agent deployment failed
	ErrAgentDeploymentFailed = ErrorCode{
		Code:       "AGENT_006",
		Type:       ErrorTypeBusiness,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "Agent deployment failed",
	}
)

// ============================================================================
// Workflow Errors (WF_xxx)
// ============================================================================

var (
	// ErrWorkflowNotFound indicates workflow does not exist
	ErrWorkflowNotFound = ErrorCode{
		Code:       "WF_001",
		Type:       ErrorTypeNotFound,
		HTTPStatus: http.StatusNotFound,
		Message:    "Workflow not found",
	}

	// ErrWorkflowInvalidDefinition indicates workflow definition is invalid
	ErrWorkflowInvalidDefinition = ErrorCode{
		Code:       "WF_002",
		Type:       ErrorTypeValidation,
		HTTPStatus: http.StatusBadRequest,
		Message:    "Invalid workflow definition",
	}

	// ErrWorkflowExecutionFailed indicates workflow execution failed
	ErrWorkflowExecutionFailed = ErrorCode{
		Code:       "WF_003",
		Type:       ErrorTypeBusiness,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "Workflow execution failed",
	}

	// ErrWorkflowStepFailed indicates a workflow step failed
	ErrWorkflowStepFailed = ErrorCode{
		Code:       "WF_004",
		Type:       ErrorTypeBusiness,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "Workflow step failed",
	}

	// ErrWorkflowInvalidState indicates workflow state transition is invalid
	ErrWorkflowInvalidState = ErrorCode{
		Code:       "WF_005",
		Type:       ErrorTypeBusiness,
		HTTPStatus: http.StatusConflict,
		Message:    "Invalid workflow state transition",
	}

	// ErrWorkflowTimeout indicates workflow execution timed out
	ErrWorkflowTimeout = ErrorCode{
		Code:       "WF_006",
		Type:       ErrorTypeTimeout,
		HTTPStatus: http.StatusRequestTimeout,
		Message:    "Workflow execution timed out",
	}
)

// ============================================================================
// Model Errors (MODEL_xxx)
// ============================================================================

var (
	// ErrModelNotFound indicates model does not exist
	ErrModelNotFound = ErrorCode{
		Code:       "MODEL_001",
		Type:       ErrorTypeNotFound,
		HTTPStatus: http.StatusNotFound,
		Message:    "Model not found",
	}

	// ErrModelNotAvailable indicates model is not available for inference
	ErrModelNotAvailable = ErrorCode{
		Code:       "MODEL_002",
		Type:       ErrorTypeBusiness,
		HTTPStatus: http.StatusServiceUnavailable,
		Message:    "Model is not available",
	}

	// ErrModelInferenceFailed indicates model inference failed
	ErrModelInferenceFailed = ErrorCode{
		Code:       "MODEL_003",
		Type:       ErrorTypeBusiness,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "Model inference failed",
	}

	// ErrModelLoadFailed indicates model loading failed
	ErrModelLoadFailed = ErrorCode{
		Code:       "MODEL_004",
		Type:       ErrorTypeInfrastructure,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "Failed to load model",
	}

	// ErrModelInvalidVersion indicates model version is invalid
	ErrModelInvalidVersion = ErrorCode{
		Code:       "MODEL_005",
		Type:       ErrorTypeValidation,
		HTTPStatus: http.StatusBadRequest,
		Message:    "Invalid model version",
	}
)

// ============================================================================
// Knowledge/Document Errors (DOC_xxx)
// ============================================================================

var (
	// ErrDocNotFound indicates document does not exist
	ErrDocNotFound = ErrorCode{
		Code:       "DOC_001",
		Type:       ErrorTypeNotFound,
		HTTPStatus: http.StatusNotFound,
		Message:    "Document not found",
	}

	// ErrDocUploadFailed indicates document upload failed
	ErrDocUploadFailed = ErrorCode{
		Code:       "DOC_002",
		Type:       ErrorTypeInfrastructure,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "Document upload failed",
	}

	// ErrDocParseFailed indicates document parsing failed
	ErrDocParseFailed = ErrorCode{
		Code:       "DOC_003",
		Type:       ErrorTypeBusiness,
		HTTPStatus: http.StatusUnprocessableEntity,
		Message:    "Failed to parse document",
	}

	// ErrDocIndexingFailed indicates document indexing failed
	ErrDocIndexingFailed = ErrorCode{
		Code:       "DOC_004",
		Type:       ErrorTypeInfrastructure,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "Document indexing failed",
	}

	// ErrDocContainsPII indicates document contains sensitive PII
	ErrDocContainsPII = ErrorCode{
		Code:       "DOC_005",
		Type:       ErrorTypeBusiness,
		HTTPStatus: http.StatusUnprocessableEntity,
		Message:    "Document contains sensitive personal information",
	}
)

// ============================================================================
// RAG Errors (RAG_xxx)
// ============================================================================

var (
	// ErrRAGRetrievalFailed indicates retrieval operation failed
	ErrRAGRetrievalFailed = ErrorCode{
		Code:       "RAG_001",
		Type:       ErrorTypeInfrastructure,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "Document retrieval failed",
	}

	// ErrRAGNoRelevantDocs indicates no relevant documents found
	ErrRAGNoRelevantDocs = ErrorCode{
		Code:       "RAG_002",
		Type:       ErrorTypeBusiness,
		HTTPStatus: http.StatusNotFound,
		Message:    "No relevant documents found",
	}

	// ErrRAGGenerationFailed indicates answer generation failed
	ErrRAGGenerationFailed = ErrorCode{
		Code:       "RAG_003",
		Type:       ErrorTypeBusiness,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "Answer generation failed",
	}
)

// ============================================================================
// Cache Errors (CACHE_xxx)
// ============================================================================

var (
	// ErrCacheReadFailed indicates cache read operation failed
	ErrCacheReadFailed = ErrorCode{
		Code:       "CACHE_001",
		Type:       ErrorTypeInfrastructure,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "Cache read failed",
	}

	// ErrCacheWriteFailed indicates cache write operation failed
	ErrCacheWriteFailed = ErrorCode{
		Code:       "CACHE_002",
		Type:       ErrorTypeInfrastructure,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "Cache write failed",
	}

	// ErrCacheInvalidationFailed indicates cache invalidation failed
	ErrCacheInvalidationFailed = ErrorCode{
		Code:       "CACHE_003",
		Type:       ErrorTypeInfrastructure,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "Cache invalidation failed",
	}
)

// ============================================================================
// Database Errors (DB_xxx)
// ============================================================================

var (
	// ErrDBConnectionFailed indicates database connection failed
	ErrDBConnectionFailed = ErrorCode{
		Code:       "DB_001",
		Type:       ErrorTypeInfrastructure,
		HTTPStatus: http.StatusServiceUnavailable,
		Message:    "Database connection failed",
	}

	// ErrDBQueryFailed indicates database query failed
	ErrDBQueryFailed = ErrorCode{
		Code:       "DB_002",
		Type:       ErrorTypeInfrastructure,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "Database query failed",
	}

	// ErrDBTransactionFailed indicates database transaction failed
	ErrDBTransactionFailed = ErrorCode{
		Code:       "DB_003",
		Type:       ErrorTypeInfrastructure,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "Database transaction failed",
	}

	// ErrDBConstraintViolation indicates database constraint violation
	ErrDBConstraintViolation = ErrorCode{
		Code:       "DB_004",
		Type:       ErrorTypeConflict,
		HTTPStatus: http.StatusConflict,
		Message:    "Database constraint violation",
	}
)

// ============================================================================
// Vector Database Errors (VEC_xxx)
// ============================================================================

var (
	// ErrVectorSearchFailed indicates vector search operation failed
	ErrVectorSearchFailed = ErrorCode{
		Code:       "VEC_001",
		Type:       ErrorTypeInfrastructure,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "Vector search failed",
	}

	// ErrVectorInsertFailed indicates vector insertion failed
	ErrVectorInsertFailed = ErrorCode{
		Code:       "VEC_002",
		Type:       ErrorTypeInfrastructure,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "Vector insertion failed",
	}

	// ErrVectorCollectionNotFound indicates vector collection does not exist
	ErrVectorCollectionNotFound = ErrorCode{
		Code:       "VEC_003",
		Type:       ErrorTypeNotFound,
		HTTPStatus: http.StatusNotFound,
		Message:    "Vector collection not found",
	}
)

// ============================================================================
// Storage Errors (STOR_xxx)
// ============================================================================

var (
	// ErrStorageUploadFailed indicates file upload failed
	ErrStorageUploadFailed = ErrorCode{
		Code:       "STOR_001",
		Type:       ErrorTypeInfrastructure,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "File upload failed",
	}

	// ErrStorageDownloadFailed indicates file download failed
	ErrStorageDownloadFailed = ErrorCode{
		Code:       "STOR_002",
		Type:       ErrorTypeInfrastructure,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "File download failed",
	}

	// ErrStorageFileNotFound indicates file does not exist
	ErrStorageFileNotFound = ErrorCode{
		Code:       "STOR_003",
		Type:       ErrorTypeNotFound,
		HTTPStatus: http.StatusNotFound,
		Message:    "File not found in storage",
	}
)

// ============================================================================
// Rate Limiting Errors (RATE_xxx)
// ============================================================================

var (
	// ErrRateLimitExceeded indicates rate limit exceeded
	ErrRateLimitExceeded = ErrorCode{
		Code:       "RATE_001",
		Type:       ErrorTypeRateLimited,
		HTTPStatus: http.StatusTooManyRequests,
		Message:    "Rate limit exceeded",
	}

	// ErrQuotaExceeded indicates usage quota exceeded
	ErrQuotaExceeded = ErrorCode{
		Code:       "RATE_002",
		Type:       ErrorTypeRateLimited,
		HTTPStatus: http.StatusTooManyRequests,
		Message:    "Usage quota exceeded",
	}
)

// ============================================================================
// Internal System Errors (SYS_xxx)
// ============================================================================

var (
	// ErrSysInternalError indicates unexpected internal error
	ErrSysInternalError = ErrorCode{
		Code:       "SYS_001",
		Type:       ErrorTypeInternal,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "Internal server error",
	}

	// ErrSysServiceUnavailable indicates service is temporarily unavailable
	ErrSysServiceUnavailable = ErrorCode{
		Code:       "SYS_002",
		Type:       ErrorTypeInfrastructure,
		HTTPStatus: http.StatusServiceUnavailable,
		Message:    "Service temporarily unavailable",
	}

	// ErrSysTimeout indicates operation timed out
	ErrSysTimeout = ErrorCode{
		Code:       "SYS_003",
		Type:       ErrorTypeTimeout,
		HTTPStatus: http.StatusRequestTimeout,
		Message:    "Operation timed out",
	}

	// ErrSysConfigurationError indicates system configuration error
	ErrSysConfigurationError = ErrorCode{
		Code:       "SYS_004",
		Type:       ErrorTypeInternal,
		HTTPStatus: http.StatusInternalServerError,
		Message:    "System configuration error",
	}
)

// NewFromCode creates an AppError from an ErrorCode
func NewFromCode(ec ErrorCode) *AppError {
	return New(ec.Code, ec.Type, ec.Message, ec.HTTPStatus)
}

// NewFromCodef creates an AppError from an ErrorCode with formatted message
func NewFromCodef(ec ErrorCode, args ...interface{}) *AppError {
	return Newf(ec.Code, ec.Type, ec.HTTPStatus, ec.Message, args...)
}

//Personal.AI order the ending
