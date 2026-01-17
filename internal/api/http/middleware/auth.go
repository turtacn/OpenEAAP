// internal/api/http/middleware/auth.go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/pkg/errors"
)

// ContextKey represents keys used for storing values in context
type ContextKey string

const (
	// ContextKeyUserID stores the authenticated user ID
	ContextKeyUserID ContextKey = "user_id"
	// ContextKeyUserEmail stores the authenticated user email
	ContextKeyUserEmail ContextKey = "user_email"
	// ContextKeyUserRole stores the authenticated user role
	ContextKeyUserRole ContextKey = "user_role"
	// ContextKeyAuthType stores the authentication type (jwt/apikey)
	ContextKeyAuthType ContextKey = "auth_type"
	// ContextKeyAPIKey stores the API key used for authentication
	ContextKeyAPIKey ContextKey = "api_key"
)

// AuthMiddleware handles authentication for HTTP requests
type AuthMiddleware struct {
	jwtSecret     string
	apiKeyStore   APIKeyStore
	logger        logging.Logger
	tracer        trace.Tracer
	skipPaths     map[string]bool
	optionalPaths map[string]bool
}

// APIKeyStore defines the interface for API key validation
type APIKeyStore interface {
	ValidateAPIKey(ctx context.Context, apiKey string) (*APIKeyInfo, error)
}

// APIKeyInfo represents API key metadata
type APIKeyInfo struct {
	UserID    string
	Email     string
	Role      string
	Scopes    []string
	ExpiresAt int64
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Role   string   `json:"role"`
	Scopes []string `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

// AuthConfig holds configuration for authentication middleware
type AuthConfig struct {
	JWTSecret     string
	APIKeyStore   APIKeyStore
	Logger        logging.Logger
	Tracer        trace.Tracer
	SkipPaths     []string
	OptionalPaths []string
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(config AuthConfig) *AuthMiddleware {
	skipPathsMap := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPathsMap[path] = true
	}

	optionalPathsMap := make(map[string]bool)
	for _, path := range config.OptionalPaths {
		optionalPathsMap[path] = true
	}

	return &AuthMiddleware{
		jwtSecret:     config.JWTSecret,
		apiKeyStore:   config.APIKeyStore,
		logger:        config.Logger,
		tracer:        config.Tracer,
		skipPaths:     skipPathsMap,
		optionalPaths: optionalPathsMap,
	}
}

// Handler returns the Gin middleware handler
func (m *AuthMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, span := m.tracer.StartSpan(c.Request.Context(), "AuthMiddleware.Handler")
		defer span.End()

		path := c.Request.URL.Path

		// Skip authentication for certain paths (e.g., health checks, public endpoints)
		if m.skipPaths[path] {
			m.logger.Debug(ctx, "Skipping authentication", "path", path)
			c.Next()
			return
		}

		// Check if authentication is optional for this path
		optional := m.optionalPaths[path]

		// Extract authentication token
		authHeader := c.GetHeader("Authorization")
		apiKey := c.GetHeader("X-API-Key")

		var authType string
		var userInfo *UserInfo
		var err error

		// Try JWT authentication first
		if authHeader != "" {
			authType = "jwt"
			userInfo, err = m.validateJWT(ctx, authHeader)
			if err != nil {
				m.logger.Warn(ctx, "JWT validation failed", "error", err)
				span.RecordError(err)
				if !optional {
					c.JSON(http.StatusUnauthorized, gin.H{
						"code":    errors.UnauthorizedError,
						"message": "Invalid or expired JWT token",
					})
					c.Abort()
					return
				}
			}
		}

		// Try API Key authentication if JWT failed or not provided
		if userInfo == nil && apiKey != "" {
			authType = "apikey"
			userInfo, err = m.validateAPIKey(ctx, apiKey)
			if err != nil {
				m.logger.Warn(ctx, "API key validation failed", "error", err)
				span.RecordError(err)
				if !optional {
					c.JSON(http.StatusUnauthorized, gin.H{
						"code":    errors.UnauthorizedError,
						"message": "Invalid API key",
					})
					c.Abort()
					return
				}
			}
		}

		// If no valid authentication found and authentication is required
		if userInfo == nil && !optional {
			m.logger.Warn(ctx, "No valid authentication provided", "path", path)
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    errors.UnauthorizedError,
				"message": "Authentication required",
			})
			c.Abort()
			return
		}

		// Inject user information into context if authenticated
		if userInfo != nil {
			c.Set(string(ContextKeyUserID), userInfo.UserID)
			c.Set(string(ContextKeyUserEmail), userInfo.Email)
			c.Set(string(ContextKeyUserRole), userInfo.Role)
			c.Set(string(ContextKeyAuthType), authType)

			// Update request context
			ctx = context.WithValue(ctx, ContextKeyUserID, userInfo.UserID)
			ctx = context.WithValue(ctx, ContextKeyUserEmail, userInfo.Email)
			ctx = context.WithValue(ctx, ContextKeyUserRole, userInfo.Role)
			ctx = context.WithValue(ctx, ContextKeyAuthType, authType)
			c.Request = c.Request.WithContext(ctx)

			// Add user info to span
			span.SetAttribute("user.id", userInfo.UserID)
			span.SetAttribute("user.email", userInfo.Email)
			span.SetAttribute("user.role", userInfo.Role)
			span.SetAttribute("auth.type", authType)

			m.logger.Debug(ctx, "User authenticated", "user_id", userInfo.UserID, "auth_type", authType)
		}

		c.Next()
	}
}

// UserInfo represents authenticated user information
type UserInfo struct {
	UserID string
	Email  string
	Role   string
	Scopes []string
}

// validateJWT validates a JWT token and extracts user information
func (m *AuthMiddleware) validateJWT(ctx context.Context, authHeader string) (*UserInfo, error) {
	// Extract token from "Bearer <token>" format
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil, errors.NewUnauthorizedError("Invalid authorization header format")
	}

	tokenString := parts[1]

	// Parse and validate JWT token
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.NewUnauthorizedError("Invalid signing method")
		}
		return []byte(m.jwtSecret), nil
	})

	if err != nil {
		m.logger.Error(ctx, "Failed to parse JWT token", "error", err)
		return nil, errors.NewUnauthorizedError("Invalid token: " + err.Error())
	}

	// Extract claims
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.NewUnauthorizedError("Invalid token claims")
	}

	// Validate required claims
	if claims.UserID == "" {
		return nil, errors.NewUnauthorizedError("Missing user_id in token claims")
	}

	return &UserInfo{
		UserID: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
		Scopes: claims.Scopes,
	}, nil
}

// validateAPIKey validates an API key and extracts user information
func (m *AuthMiddleware) validateAPIKey(ctx context.Context, apiKey string) (*UserInfo, error) {
	if apiKey == "" {
		return nil, errors.NewUnauthorizedError("Empty API key")
	}

	// Validate API key using the store
	keyInfo, err := m.apiKeyStore.ValidateAPIKey(ctx, apiKey)
	if err != nil {
		m.logger.Error(ctx, "API key validation failed", "error", err)
		return nil, errors.NewUnauthorizedError("Invalid API key")
	}

	// Check if API key has expired
	if keyInfo.ExpiresAt > 0 && keyInfo.ExpiresAt < currentTimestamp() {
		return nil, errors.NewUnauthorizedError("API key has expired")
	}

	return &UserInfo{
		UserID: keyInfo.UserID,
		Email:  keyInfo.Email,
		Role:   keyInfo.Role,
		Scopes: keyInfo.Scopes,
	}, nil
}

// GetUserID extracts user ID from Gin context
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get(string(ContextKeyUserID))
	if !exists {
		return "", false
	}
	userIDStr, ok := userID.(string)
	return userIDStr, ok
}

// GetUserEmail extracts user email from Gin context
func GetUserEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get(string(ContextKeyUserEmail))
	if !exists {
		return "", false
	}
	emailStr, ok := email.(string)
	return emailStr, ok
}

// GetUserRole extracts user role from Gin context
func GetUserRole(c *gin.Context) (string, bool) {
	role, exists := c.Get(string(ContextKeyUserRole))
	if !exists {
		return "", false
	}
	roleStr, ok := role.(string)
	return roleStr, ok
}

// GetAuthType extracts authentication type from Gin context
func GetAuthType(c *gin.Context) (string, bool) {
	authType, exists := c.Get(string(ContextKeyAuthType))
	if !exists {
		return "", false
	}
	authTypeStr, ok := authType.(string)
	return authTypeStr, ok
}

// RequireRole returns a middleware that checks if user has required role
func RequireRole(logger logging.Logger, requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := GetUserRole(c)
		if !exists {
			logger.Warn(c.Request.Context(), "User role not found in context")
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    errors.UnauthorizedError,
				"message": "Authentication required",
			})
			c.Abort()
			return
		}

		if role != requiredRole && role != "admin" { // Admin has access to everything
			logger.Warn(c.Request.Context(), "Insufficient permissions", "required_role", requiredRole, "user_role", role)
			c.JSON(http.StatusForbidden, gin.H{
				"code":    errors.ErrForbidden,
				"message": "Insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyRole returns a middleware that checks if user has any of the required roles
func RequireAnyRole(logger logging.Logger, requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := GetUserRole(c)
		if !exists {
			logger.Warn(c.Request.Context(), "User role not found in context")
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    errors.UnauthorizedError,
				"message": "Authentication required",
			})
			c.Abort()
			return
		}

		// Admin has access to everything
		if role == "admin" {
			c.Next()
			return
		}

		// Check if user has any of the required roles
		hasRole := false
		for _, requiredRole := range requiredRoles {
			if role == requiredRole {
				hasRole = true
				break
			}
		}

		if !hasRole {
			logger.Warn(c.Request.Context(), "Insufficient permissions", "required_roles", requiredRoles, "user_role", role)
			c.JSON(http.StatusForbidden, gin.H{
				"code":    errors.ErrForbidden,
				"message": "Insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// currentTimestamp returns current Unix timestamp
func currentTimestamp() int64 {
	return jwt.NewNumericDate(jwt.TimeFunc()).Unix()
}

//Personal.AI order the ending
