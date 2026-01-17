// Package inference provides the inference gateway and related components
// for managing LLM inference requests with caching, routing, and privacy protection.
package inference

import (
	"context"
	"fmt"
	"time"

	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/internal/platform/inference/cache"
	"github.com/openeeap/openeeap/internal/platform/inference/privacy"
	"github.com/openeeap/openeeap/pkg/errors"
	"github.com/openeeap/openeeap/pkg/types"
)

// InferenceRequest represents a request for LLM inference
type InferenceRequest struct {
	Model       string                 `json:"model" validate:"required"`
	Messages    []Message              `json:"messages" validate:"required,min=1"`
	Temperature float64                `json:"temperature,omitempty" validate:"gte=0,lte=2"`
	MaxTokens   int                    `json:"max_tokens,omitempty" validate:"gte=1,lte=4096"`
	Stream      bool                   `json:"stream,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role" validate:"required,oneof=system user assistant"`
	Content string `json:"content" validate:"required"`
}

// InferenceResponse represents the response from LLM inference
type InferenceResponse struct {
	ID         string                 `json:"id"`
	Model      string                 `json:"model"`
	Content    string                 `json:"content"`
	Usage      Usage                  `json:"usage"`
	Cached     bool                   `json:"cached"`
	Latency    time.Duration          `json:"latency"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	FinishTime time.Time              `json:"finish_time"`
}

// Usage represents token usage statistics
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk represents a streaming response chunk
type StreamChunk struct {
	ID      string `json:"id"`
	Delta   string `json:"delta"`
	Done    bool   `json:"done"`
	Usage   *Usage `json:"usage,omitempty"`
	Error   error  `json:"error,omitempty"`
	Latency time.Duration `json:"latency"`
}

// InferenceEngine defines the interface for LLM inference backends
type InferenceEngine interface {
	// Infer performs synchronous inference
	Infer(ctx context.Context, req *InferenceRequest) (*InferenceResponse, error)

	// InferStream performs streaming inference
	InferStream(ctx context.Context, req *InferenceRequest) (<-chan *StreamChunk, error)

	// HealthCheck checks if the engine is healthy
	HealthCheck(ctx context.Context) error

	// Name returns the engine name
	Name() string
}

// Gateway is the unified inference gateway
type Gateway struct {
	logger          logging.Logger
	tracer          trace.Tracer
	metricsCollector *metrics.MetricsCollector
	cacheManager    *cache.CacheManager
	privacyGateway  *privacy.PrivacyGateway
	router          *Router
	rateLimiter     RateLimiter
	authenticator   Authenticator
}

// RateLimiter defines rate limiting interface
type RateLimiter interface {
	Allow(ctx context.Context, userID string) (bool, error)
	GetQuota(ctx context.Context, userID string) (int, error)
}

// Authenticator defines authentication interface
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*AuthContext, error)
}

// AuthContext contains authentication information
type AuthContext struct {
	UserID      string
	TenantID    string
	Roles       []string
	Permissions []string
}

// GatewayConfig contains configuration for the inference gateway
type GatewayConfig struct {
	EnableCache       bool
	EnablePrivacy     bool
	EnableRateLimit   bool
	DefaultTimeout    time.Duration
	MaxConcurrency    int
	MetricsEnabled    bool
	TracingEnabled    bool
}

// NewGateway creates a new inference gateway
func NewGateway(
	logger logging.Logger,
	tracer trace.Tracer,
	metricsCollector metrics.MetricsCollector,
	cacheManager *cache.CacheManager,
	privacyGateway *privacy.PrivacyGateway,
	router *Router,
	rateLimiter RateLimiter,
	authenticator Authenticator,
	config *GatewayConfig,
) *Gateway {
	return &Gateway{
		logger:           logger,
		tracer:           tracer,
		metricsCollector: metricsCollector,
		cacheManager:     cacheManager,
		privacyGateway:   privacyGateway,
		router:           router,
		rateLimiter:      rateLimiter,
		authenticator:    authenticator,
	}
}

// Infer performs synchronous inference through the gateway
func (g *Gateway) Infer(ctx context.Context, req *InferenceRequest) (*InferenceResponse, error) {
	startTime := time.Now()

	// Start tracing
	ctx, span := g.tracer.Start(ctx, "Gateway.Infer")
	defer span.End()

	span.SetAttribute("model", req.Model)
	span.SetAttribute("stream", req.Stream)

	// Authenticate request
	authCtx, err := g.authenticateRequest(ctx)
	if err != nil {
		g.recordMetrics("infer", "auth_failed", time.Since(startTime))
		return nil, errors.Wrap(err, errors.CodeUnauthorized, "authentication failed")
	}

	span.SetAttribute("user_id", authCtx.UserID)
	span.SetAttribute("tenant_id", authCtx.TenantID)

	// Rate limiting
	if g.rateLimiter != nil {
		allowed, err := g.rateLimiter.Allow(ctx, authCtx.UserID)
		if err != nil {
			g.logger.WithContext(ctx).Error("rate limit check failed", logging.Error(err))
		}
		if !allowed {
			g.recordMetrics("infer", "rate_limited", time.Since(startTime))
			return nil, errors.New(errors.CodeRateLimitExceeded, "rate limit exceeded")
		}
	}

	// Privacy filtering (PII detection and masking)
	if g.privacyGateway != nil {
		filteredReq, err := g.privacyGateway.FilterRequest(ctx, req)
		if err != nil {
			g.logger.WithContext(ctx).Error("privacy filtering failed", logging.Error(err))
			g.recordMetrics("infer", "privacy_failed", time.Since(startTime))
			return nil, errors.Wrap(err, errors.CodeInternalError, "privacy filtering failed")
		}
		req = filteredReq
	}

	// Try cache lookup (L1 -> L2 -> L3)
	if g.cacheManager != nil {
		cachedResp, err := g.cacheManager.Get(ctx, req)
		if err == nil && cachedResp != nil {
			g.logger.WithContext(ctx).Info("cache hit", logging.Any("model", req.Model))
			g.recordMetrics("infer", "cache_hit", time.Since(startTime))
			span.SetAttribute("cached", true)

			cachedResp.Cached = true
			cachedResp.Latency = time.Since(startTime)
			return cachedResp, nil
		}

		if err != nil {
			g.logger.WithContext(ctx).Warn("cache lookup failed", logging.Error(err))
		}
	}

	span.SetAttribute("cached", false)
	g.recordMetrics("infer", "cache_miss", time.Since(startTime))

	// Route to appropriate inference engine
	engine, err := g.router.Route(ctx, req)
	if err != nil {
		g.recordMetrics("infer", "routing_failed", time.Since(startTime))
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to route request")
	}

	span.SetAttribute("engine", engine.Name())

	// Perform inference
	resp, err := engine.Infer(ctx, req)
	if err != nil {
		g.logger.WithContext(ctx).Error("inference failed", logging.Error(err), logging.Any("engine", engine.Name())
		g.recordMetrics("infer", "inference_failed", time.Since(startTime))
		return nil, errors.Wrap(err, errors.CodeInternalError, "inference failed")
	}

	// Store in cache
	if g.cacheManager != nil && resp != nil {
		if err := g.cacheManager.Set(ctx, req, resp); err != nil {
			g.logger.WithContext(ctx).Warn("failed to cache response", logging.Error(err))
		}
	}

	// Privacy restoration (if needed)
	if g.privacyGateway != nil {
		restoredResp, err := g.privacyGateway.RestoreResponse(ctx, resp)
		if err != nil {
			g.logger.WithContext(ctx).Warn("privacy restoration failed", logging.Error(err))
		} else {
			resp = restoredResp
		}
	}

	resp.Latency = time.Since(startTime)
	resp.FinishTime = time.Now()

	// Record metrics
	g.recordMetrics("infer", "success", resp.Latency)
	g.metricsCollector.RecordHistogram("inference_latency_ms", float64(resp.Latency.Milliseconds()),
		map[string]string{
			"model":  req.Model,
			"cached": fmt.Sprintf("%v", resp.Cached),
			"engine": engine.Name(),
		})

	g.metricsCollector.IncrementCounter("inference_tokens_total", float64(resp.Usage.TotalTokens),
		map[string]string{
			"model": req.Model,
			"type":  "total",
		})

 g.logger.WithContext(ctx).Info("inference completed", logging.Any("model", req.Model), logging.Any("latency_ms", resp.Latency.Milliseconds())

	return resp, nil
}

// InferStream performs streaming inference through the gateway
func (g *Gateway) InferStream(ctx context.Context, req *InferenceRequest) (<-chan *StreamChunk, error) {
	startTime := time.Now()

	// Start tracing
	ctx, span := g.tracer.Start(ctx, "Gateway.InferStream")
	defer span.End()

	span.SetAttribute("model", req.Model)
	span.SetAttribute("stream", true)

	// Authenticate request
	authCtx, err := g.authenticateRequest(ctx)
	if err != nil {
		g.recordMetrics("infer_stream", "auth_failed", time.Since(startTime))
		return nil, errors.Wrap(err, errors.CodeUnauthorized, "authentication failed")
	}

	span.SetAttribute("user_id", authCtx.UserID)

	// Rate limiting
	if g.rateLimiter != nil {
		allowed, err := g.rateLimiter.Allow(ctx, authCtx.UserID)
		if err != nil {
			g.logger.WithContext(ctx).Error("rate limit check failed", logging.Error(err))
		}
		if !allowed {
			g.recordMetrics("infer_stream", "rate_limited", time.Since(startTime))
			return nil, errors.New(errors.CodeRateLimitExceeded, "rate limit exceeded")
		}
	}

	// Privacy filtering
	if g.privacyGateway != nil {
		filteredReq, err := g.privacyGateway.FilterRequest(ctx, req)
		if err != nil {
			g.logger.WithContext(ctx).Error("privacy filtering failed", logging.Error(err))
			g.recordMetrics("infer_stream", "privacy_failed", time.Since(startTime))
			return nil, errors.Wrap(err, errors.CodeInternalError, "privacy filtering failed")
		}
		req = filteredReq
	}

	// Route to appropriate inference engine
	engine, err := g.router.Route(ctx, req)
	if err != nil {
		g.recordMetrics("infer_stream", "routing_failed", time.Since(startTime))
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to route request")
	}

	span.SetAttribute("engine", engine.Name())

	// Perform streaming inference
	chunkChan, err := engine.InferStream(ctx, req)
	if err != nil {
		g.logger.WithContext(ctx).Error("streaming inference failed", logging.Error(err), logging.Any("engine", engine.Name())
		g.recordMetrics("infer_stream", "inference_failed", time.Since(startTime))
		return nil, errors.Wrap(err, errors.CodeInternalError, "streaming inference failed")
	}

	// Wrap channel to add metrics and privacy restoration
	wrappedChan := make(chan *StreamChunk)

	go func() {
		defer close(wrappedChan)

		totalTokens := 0
		firstChunkReceived := false

		for chunk := range chunkChan {
			if !firstChunkReceived {
				g.recordMetrics("infer_stream", "first_chunk", time.Since(startTime))
				firstChunkReceived = true
			}

			// Privacy restoration for chunk if needed
			if g.privacyGateway != nil && chunk.Delta != "" {
				restoredDelta, err := g.privacyGateway.RestoreText(ctx, chunk.Delta)
				if err != nil {
					g.logger.WithContext(ctx).Warn("chunk privacy restoration failed", logging.Error(err))
				} else {
					chunk.Delta = restoredDelta
				}
			}

			wrappedChan <- chunk

			if chunk.Done && chunk.Usage != nil {
				totalTokens = chunk.Usage.TotalTokens
			}
		}

		totalLatency := time.Since(startTime)

		g.recordMetrics("infer_stream", "completed", totalLatency)
		g.metricsCollector.RecordHistogram("inference_stream_latency_ms", float64(totalLatency.Milliseconds()),
			map[string]string{
				"model":  req.Model,
				"engine": engine.Name(),
			})

		if totalTokens > 0 {
			g.metricsCollector.IncrementCounter("inference_tokens_total", float64(totalTokens),
				map[string]string{
					"model": req.Model,
					"type":  "stream",
				})
		}

  g.logger.WithContext(ctx).Info("streaming inference completed", logging.Any("model", req.Model), logging.Any("latency_ms", totalLatency.Milliseconds())
	}()

	return wrappedChan, nil
}

// authenticateRequest authenticates the request and returns auth context
func (g *Gateway) authenticateRequest(ctx context.Context) (*AuthContext, error) {
	if g.authenticator == nil {
		// No authentication configured, return default context
		return &AuthContext{
			UserID:   "anonymous",
			TenantID: "default",
		}, nil
	}

	// Extract token from context (assume it's set by middleware)
	token, ok := ctx.Value(types.ContextKeyAuthToken).(string)
	if !ok || token == "" {
		return nil, errors.New(errors.CodeUnauthorized, "missing authentication token")
	}

	authCtx, err := g.authenticator.Authenticate(ctx, token)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeUnauthorized, "invalid authentication token")
	}

	return authCtx, nil
}

// recordMetrics records metrics for gateway operations
func (g *Gateway) recordMetrics(operation, status string, latency time.Duration) {
	if g.metricsCollector == nil {
		return
	}

	g.metricsCollector.IncrementCounter("gateway_requests_total", 1,
		map[string]string{
			"operation": operation,
			"status":    status,
		})

	g.metricsCollector.RecordHistogram("gateway_operation_latency_ms", float64(latency.Milliseconds()),
		map[string]string{
			"operation": operation,
			"status":    status,
		})
}

// HealthCheck checks the health of all inference engines
func (g *Gateway) HealthCheck(ctx context.Context) error {
	engines := g.router.GetAllEngines()

	for _, engine := range engines {
		if err := engine.HealthCheck(ctx); err != nil {
			g.logger.WithContext(ctx).Error("engine health check failed", logging.Any("engine", engine.Name()), logging.Error(err))
			return errors.Wrap(err, errors.CodeInternalError, fmt.Sprintf("engine %s is unhealthy", engine.Name()))
		}
	}

	g.logger.WithContext(ctx).Info("all engines healthy", logging.Any("count", len(engines))
	return nil
}

//Personal.AI order the ending
