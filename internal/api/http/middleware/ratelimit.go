// internal/api/http/middleware/ratelimit.go
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/pkg/errors"
	"golang.org/x/time/rate"
)

// RateLimitStrategy defines the rate limiting strategy
type RateLimitStrategy string

const (
	// StrategyTokenBucket uses token bucket algorithm
	StrategyTokenBucket RateLimitStrategy = "token_bucket"
	// StrategyLeakyBucket uses leaky bucket algorithm
	StrategyLeakyBucket RateLimitStrategy = "leaky_bucket"
)

// RateLimitKey defines the key type for rate limiting
type RateLimitKey string

const (
	// KeyByUserID limits by user ID
	KeyByUserID RateLimitKey = "user_id"
	// KeyByIP limits by IP address
	KeyByIP RateLimitKey = "ip"
	// KeyByAPIKey limits by API key
	KeyByAPIKey RateLimitKey = "api_key"
	// KeyGlobal applies global rate limit
	KeyGlobal RateLimitKey = "global"
)

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	Strategy       RateLimitStrategy
	KeyType        RateLimitKey
	RequestsLimit  int           // Maximum requests allowed
	WindowDuration time.Duration // Time window for rate limiting
	BurstSize      int           // Burst size for token bucket
	Logger         logging.Logger
	Tracer         trace.Tracer
	Store          RateLimitStore // Optional Redis-backed store
}

// RateLimitStore defines interface for distributed rate limiting
type RateLimitStore interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
	GetRemaining(ctx context.Context, key string) (int, error)
	Reset(ctx context.Context, key string) error
}

// RateLimitMiddleware handles rate limiting for HTTP requests
type RateLimitMiddleware struct {
	config   RateLimitConfig
	limiters sync.Map // map[string]*rateLimiter
	logger   logging.Logger
	tracer   trace.Tracer
}

// rateLimiter wraps rate.Limiter with metadata
type rateLimiter struct {
	limiter    *rate.Limiter
	lastAccess time.Time
	mu         sync.Mutex
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware(config RateLimitConfig) *RateLimitMiddleware {
	if config.BurstSize == 0 {
		config.BurstSize = config.RequestsLimit
	}

	return &RateLimitMiddleware{
		config: config,
		logger: config.Logger,
		tracer: config.Tracer,
	}
}

// Handler returns the Gin middleware handler
func (m *RateLimitMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
// 		ctx, span := m.tracer.StartSpan(c.Request.Context(), "RateLimitMiddleware.Handler")
// 		defer span.End()

// 		// Extract rate limit key based on strategy
// 		key := m.extractKey(c)
// 		if key == "" {
// 			m.logger.WithContext(ctx).Warn("Failed to extract rate limit key", logging.Any(logging.String("key_type", m.config.KeyType)))
// 			c.Next()
// 			return
// 		}

// 		span.SetAttribute("ratelimit.key", key)
// 		span.SetAttribute("ratelimit.strategy", string(m.config.Strategy))

// 		// Check rate limit
// 		allowed, remaining, resetTime, err := m.checkRateLimit(ctx, key)
// 		if err != nil {
// 			m.logger.WithContext(ctx).Error("Rate limit check failed", logging.Error(err), logging.Any(logging.String("key", key)))
// 			span.RecordError(err)
// 			// On error, allow request to proceed (fail open)
// 			c.Next()
// 			return
// 		}

// 		// Add rate limit headers
// 		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", m.config.RequestsLimit))
// 		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
// 		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

// 		if !allowed {
// 			m.logger.WithContext(ctx).Warn("Rate limit exceeded", logging.Any(logging.String("key", key)), logging.Any(logging.String("limit", m.config.RequestsLimit)))
// 			span.SetAttribute("ratelimit.exceeded", true)

// 			retryAfter := time.Until(resetTime).Seconds()
			c.Header("Retry-After", fmt.Sprintf("%.0f", retryAfter))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":        errors.ErrRateLimitExceeded,
				"message":     "Rate limit exceeded",
				"retry_after": retryAfter,
			})
			c.Abort()
			return
		}

// 		span.SetAttribute("ratelimit.remaining", remaining)
		c.Next()
	}
// }

// extractKey extracts the rate limit key from the request
func (m *RateLimitMiddleware) extractKey(c *gin.Context) string {
	switch m.config.KeyType {
	case KeyByUserID:
		userID, exists := GetUserID(c)
		if !exists {
			// Fall back to IP if user ID not available
			return m.getClientIP(c)
		}
		return fmt.Sprintf("user:%s", userID)

	case KeyByAPIKey:
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			return m.getClientIP(c)
		}
		return fmt.Sprintf("apikey:%s", apiKey)

	case KeyByIP:
		return fmt.Sprintf("ip:%s", m.getClientIP(c))

	case KeyGlobal:
		return "global"

	default:
		return m.getClientIP(c)
	}
}

// getClientIP extracts client IP from request
func (m *RateLimitMiddleware) getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header first
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Check X-Real-IP header
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return c.ClientIP()
}

// checkRateLimit checks if request is allowed under rate limit
func (m *RateLimitMiddleware) checkRateLimit(ctx context.Context, key string) (allowed bool, remaining int, resetTime time.Time, err error) {
	// Use distributed store if available
	if m.config.Store != nil {
		return m.checkDistributedRateLimit(ctx, key)
	}

	// Use in-memory rate limiter
	return m.checkLocalRateLimit(ctx, key)
}

// checkDistributedRateLimit uses Redis-backed store for rate limiting
func (m *RateLimitMiddleware) checkDistributedRateLimit(ctx context.Context, key string) (allowed bool, remaining int, resetTime time.Time, err error) {
	allowed, err = m.config.Store.Allow(ctx, key, m.config.RequestsLimit, m.config.WindowDuration)
	if err != nil {
		return false, 0, time.Time{}, err
	}

	remaining, err = m.config.Store.GetRemaining(ctx, key)
	if err != nil {
		return allowed, 0, time.Time{}, err
	}

	resetTime = time.Now().Add(m.config.WindowDuration)
	return allowed, remaining, resetTime, nil
}

// checkLocalRateLimit uses in-memory rate limiter
func (m *RateLimitMiddleware) checkLocalRateLimit(ctx context.Context, key string) (allowed bool, remaining int, resetTime time.Time, err error) {
	limiter := m.getLimiter(key)

	// Clean up old limiters periodically
	go m.cleanupLimiters()

	// Update last access time
	limiter.mu.Lock()
	limiter.lastAccess = time.Now()
	limiter.mu.Unlock()

	// Check if request is allowed
	reservation := limiter.limiter.Reserve()
	if !reservation.OK() {
		return false, 0, time.Time{}, nil
	}

	delay := reservation.Delay()
	if delay > 0 {
		reservation.Cancel()
		resetTime = time.Now().Add(delay)
		return false, 0, resetTime, nil
	}

	// Calculate remaining tokens
	tokens := limiter.limiter.Tokens()
	remaining = int(tokens)
	if remaining < 0 {
		remaining = 0
	}

	resetTime = time.Now().Add(m.config.WindowDuration)
	return true, remaining, resetTime, nil
}

// getLimiter retrieves or creates a rate limiter for the given key
func (m *RateLimitMiddleware) getLimiter(key string) *rateLimiter {
	if val, ok := m.limiters.Load(key); ok {
		return val.(*rateLimiter)
	}

	// Create new limiter based on strategy
	var limiter *rate.Limiter
	switch m.config.Strategy {
	case StrategyTokenBucket:
		// Token bucket: requests per second, with burst
		rps := float64(m.config.RequestsLimit) / m.config.WindowDuration.Seconds()
		limiter = rate.NewLimiter(rate.Limit(rps), m.config.BurstSize)

	case StrategyLeakyBucket:
		// Leaky bucket: smooth rate with no burst
		rps := float64(m.config.RequestsLimit) / m.config.WindowDuration.Seconds()
		limiter = rate.NewLimiter(rate.Limit(rps), 1)

	default:
		// Default to token bucket
		rps := float64(m.config.RequestsLimit) / m.config.WindowDuration.Seconds()
		limiter = rate.NewLimiter(rate.Limit(rps), m.config.BurstSize)
	}

	rateLimiter := &rateLimiter{
		limiter:    limiter,
		lastAccess: time.Now(),
	}

	actual, _ := m.limiters.LoadOrStore(key, rateLimiter)
	return actual.(*rateLimiter)
}

// cleanupLimiters removes old, unused limiters to prevent memory leaks
func (m *RateLimitMiddleware) cleanupLimiters() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		m.limiters.Range(func(key, value interface{}) bool {
			limiter := value.(*rateLimiter)
			limiter.mu.Lock()
			inactive := now.Sub(limiter.lastAccess) > 10*time.Minute
			limiter.mu.Unlock()

			if inactive {
				m.limiters.Delete(key)
			}
			return true
		})
	}
}

// MultiKeyRateLimitMiddleware applies multiple rate limits with different keys
type MultiKeyRateLimitMiddleware struct {
	middlewares []*RateLimitMiddleware
	logger      logging.Logger
	tracer      trace.Tracer
}

// NewMultiKeyRateLimitMiddleware creates a middleware with multiple rate limit strategies
func NewMultiKeyRateLimitMiddleware(configs []RateLimitConfig) *MultiKeyRateLimitMiddleware {
	middlewares := make([]*RateLimitMiddleware, len(configs))
	for i, config := range configs {
		middlewares[i] = NewRateLimitMiddleware(config)
	}

	return &MultiKeyRateLimitMiddleware{
		middlewares: middlewares,
		logger:      configs[0].Logger,
		tracer:      configs[0].Tracer,
	}
}

// Handler returns the Gin middleware handler
func (m *MultiKeyRateLimitMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, middleware := range m.middlewares {
			middleware.Handler()(c)
			if c.IsAborted() {
				return
			}
		}
		c.Next()
	}
}

// AdaptiveRateLimitMiddleware dynamically adjusts rate limits based on load
type AdaptiveRateLimitMiddleware struct {
	baseConfig RateLimitConfig
	middleware *RateLimitMiddleware
	logger     logging.Logger
	tracer     trace.Tracer
	mu         sync.RWMutex
}

// NewAdaptiveRateLimitMiddleware creates an adaptive rate limiting middleware
func NewAdaptiveRateLimitMiddleware(config RateLimitConfig) *AdaptiveRateLimitMiddleware {
	return &AdaptiveRateLimitMiddleware{
		baseConfig: config,
		middleware: NewRateLimitMiddleware(config),
		logger:     config.Logger,
		tracer:     config.Tracer,
	}
}

// Handler returns the Gin middleware handler
func (m *AdaptiveRateLimitMiddleware) Handler() gin.HandlerFunc {
	return m.middleware.Handler()
}

// AdjustLimit dynamically adjusts the rate limit
func (m *AdaptiveRateLimitMiddleware) AdjustLimit(newLimit int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.baseConfig.RequestsLimit = newLimit
	m.middleware = NewRateLimitMiddleware(m.baseConfig)
	m.logger.Info("Rate limit adjusted", logging.String("new_limit", newLimit))
}

//Personal.AI order the ending
