// internal/api/http/middleware/cors.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/pkg/config"
)

// CORSConfig holds CORS middleware configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		},
		AllowedHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Request-ID",
			"X-Trace-ID",
			"X-API-Key",
		},
		ExposedHeaders: []string{
			"Content-Length",
			"X-Request-ID",
			"X-Trace-ID",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}
}

// CORSMiddleware creates a new CORS middleware with the given configuration
func CORSMiddleware(cfg *config.Config, logger logging.Logger) gin.HandlerFunc {
	corsConfig := buildCORSConfig(cfg)

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Handle preflight requests
		if c.Request.Method == http.MethodOptions {
			handlePreflightRequest(c, origin, corsConfig, logger)
			return
		}

		// Handle actual requests
		handleActualRequest(c, origin, corsConfig, logger)
		c.Next()
	}
}

// buildCORSConfig builds CORS configuration from application config
func buildCORSConfig(cfg *config.Config) *CORSConfig {
	corsConfig := DefaultCORSConfig()

	if cfg.Server.CORS.Enabled {
		if len(cfg.Server.CORS.AllowedOrigins) > 0 {
			corsConfig.AllowedOrigins = cfg.Server.CORS.AllowedOrigins
		}
		if len(cfg.Server.CORS.AllowedMethods) > 0 {
			corsConfig.AllowedMethods = cfg.Server.CORS.AllowedMethods
		}
		if len(cfg.Server.CORS.AllowedHeaders) > 0 {
			corsConfig.AllowedHeaders = cfg.Server.CORS.AllowedHeaders
		}
		if len(cfg.Server.CORS.ExposedHeaders) > 0 {
			corsConfig.ExposedHeaders = cfg.Server.CORS.ExposedHeaders
		}
		corsConfig.AllowCredentials = cfg.Server.CORS.AllowCredentials
		if cfg.Server.CORS.MaxAge > 0 {
			corsConfig.MaxAge = cfg.Server.CORS.MaxAge
		}
	}

	return corsConfig
}

// handlePreflightRequest handles CORS preflight (OPTIONS) requests
func handlePreflightRequest(c *gin.Context, origin string, cfg *CORSConfig, logger logging.Logger) {
	if !isOriginAllowed(origin, cfg.AllowedOrigins) {
		logger.Debug(c.Request.Context(), "CORS preflight rejected: origin not allowed",
			"origin", origin,
			"allowed_origins", cfg.AllowedOrigins,
		)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	requestMethod := c.Request.Header.Get("Access-Control-Request-Method")
	if !isMethodAllowed(requestMethod, cfg.AllowedMethods) {
		logger.Debug(c.Request.Context(), "CORS preflight rejected: method not allowed",
			"method", requestMethod,
			"allowed_methods", cfg.AllowedMethods,
		)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	requestHeaders := parseHeaderList(c.Request.Header.Get("Access-Control-Request-Headers"))
	if !areHeadersAllowed(requestHeaders, cfg.AllowedHeaders) {
		logger.Debug(c.Request.Context(), "CORS preflight rejected: headers not allowed",
			"headers", requestHeaders,
			"allowed_headers", cfg.AllowedHeaders,
		)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	// Set CORS headers for preflight response
	c.Header("Access-Control-Allow-Origin", origin)
	c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
	c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))

	if cfg.AllowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}

	if cfg.MaxAge > 0 {
		c.Header("Access-Control-Max-Age", string(rune(cfg.MaxAge)))
	}

	logger.Debug(c.Request.Context(), "CORS preflight accepted",
		"origin", origin,
		"method", requestMethod,
	)

	c.AbortWithStatus(http.StatusNoContent)
}

// handleActualRequest handles actual CORS requests
func handleActualRequest(c *gin.Context, origin string, cfg *CORSConfig, logger logging.Logger) {
	if origin == "" {
		// Not a CORS request
		return
	}

	if !isOriginAllowed(origin, cfg.AllowedOrigins) {
		logger.Debug(c.Request.Context(), "CORS request rejected: origin not allowed",
			"origin", origin,
			"allowed_origins", cfg.AllowedOrigins,
		)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	// Set CORS headers for actual response
	c.Header("Access-Control-Allow-Origin", origin)

	if len(cfg.ExposedHeaders) > 0 {
		c.Header("Access-Control-Expose-Headers", strings.Join(cfg.ExposedHeaders, ", "))
	}

	if cfg.AllowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}

	logger.Debug(c.Request.Context(), "CORS request accepted",
		"origin", origin,
		"method", c.Request.Method,
	)
}

// isOriginAllowed checks if the origin is allowed
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if len(allowedOrigins) == 0 {
		return false
	}

	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
		// Support wildcard subdomains (e.g., *.example.com)
		if strings.HasPrefix(allowed, "*.") {
			domain := allowed[2:]
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}

	return false
}

// isMethodAllowed checks if the HTTP method is allowed
func isMethodAllowed(method string, allowedMethods []string) bool {
	if len(allowedMethods) == 0 {
		return false
	}

	method = strings.ToUpper(method)
	for _, allowed := range allowedMethods {
		if strings.ToUpper(allowed) == method {
			return true
		}
	}

	return false
}

// areHeadersAllowed checks if all requested headers are allowed
func areHeadersAllowed(requestedHeaders, allowedHeaders []string) bool {
	if len(requestedHeaders) == 0 {
		return true
	}

	allowedMap := make(map[string]bool, len(allowedHeaders))
	for _, header := range allowedHeaders {
		allowedMap[strings.ToLower(header)] = true
	}

	// Check if wildcard is allowed
	if allowedMap["*"] {
		return true
	}

	for _, header := range requestedHeaders {
		if !allowedMap[strings.ToLower(header)] {
			return false
		}
	}

	return true
}

// parseHeaderList parses a comma-separated header list
func parseHeaderList(headerValue string) []string {
	if headerValue == "" {
		return []string{}
	}

	headers := strings.Split(headerValue, ",")
	result := make([]string, 0, len(headers))

	for _, header := range headers {
		trimmed := strings.TrimSpace(header)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// CORSWithConfig creates a CORS middleware with custom configuration
func CORSWithConfig(corsConfig *CORSConfig, logger logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if c.Request.Method == http.MethodOptions {
			handlePreflightRequest(c, origin, corsConfig, logger)
			return
		}

		handleActualRequest(c, origin, corsConfig, logger)
		c.Next()
	}
}

//Personal.AI order the ending
