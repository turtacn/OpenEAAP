// internal/governance/policy/pep.go
package policy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/pkg/errors"
)

// PolicyEnforcementPoint 策略执行点接口
type PolicyEnforcementPoint interface {
	// Enforce 执行策略检查
	Enforce(ctx context.Context, request *AccessRequest) (*EnforcementResult, error)

	// EnforceAsync 异步执行策略检查
	EnforceAsync(ctx context.Context, request *AccessRequest) <-chan *EnforcementResult

	// EnforceBatch 批量执行策略检查
	EnforceBatch(ctx context.Context, requests []*AccessRequest) ([]*EnforcementResult, error)

	// PreAuthorize 预授权检查
	PreAuthorize(ctx context.Context, subjectID, resourceID, action string) error

	// PostAuthorize 后授权检查
	PostAuthorize(ctx context.Context, request *AccessRequest, result interface{}) error

	// GetEnforcementStats 获取执行统计
	GetEnforcementStats(ctx context.Context) (*EnforcementStats, error)
}

// EnforcementResult 执行结果
type EnforcementResult struct {
	RequestID      string                 `json:"request_id"`
	Allowed        bool                   `json:"allowed"`
	Decision       *Decision              `json:"decision"`
	EnforcedAt     time.Time              `json:"enforced_at"`
	Duration       time.Duration          `json:"duration"`
	ObligationsMet bool                   `json:"obligations_met"`
	AdviceApplied  []string               `json:"advice_applied"`
	AuditID        string                 `json:"audit_id"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// EnforcementStats 执行统计
type EnforcementStats struct {
	TotalRequests      int64            `json:"total_requests"`
	PermittedRequests  int64            `json:"permitted_requests"`
	DeniedRequests     int64            `json:"denied_requests"`
	AverageLatency     time.Duration    `json:"average_latency"`
	CacheHitRate       float64          `json:"cache_hit_rate"`
	ErrorRate          float64          `json:"error_rate"`
	TopDeniedActions   map[string]int64 `json:"top_denied_actions"`
	TopDeniedResources map[string]int64 `json:"top_denied_resources"`
}

// AuditLogger 审计日志接口
type AuditLogger interface {
	// LogAccess 记录访问日志
	LogAccess(ctx context.Context, entry *AuditEntry) error

	// Query 查询审计日志
	Query(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error)
}

// AuditEntry 审计日志条目
type AuditEntry struct {
	ID             string                 `json:"id"`
	RequestID      string                 `json:"request_id"`
	Timestamp      time.Time              `json:"timestamp"`
	SubjectID      string                 `json:"subject_id"`
	SubjectType    string                 `json:"subject_type"`
	ResourceID     string                 `json:"resource_id"`
	ResourceType   string                 `json:"resource_type"`
	Action         string                 `json:"action"`
	Decision       Effect                 `json:"decision"`
	Reason         string                 `json:"reason"`
	Policies       []string               `json:"policies"`
	Duration       time.Duration          `json:"duration"`
	SourceIP       string                 `json:"source_ip"`
	UserAgent      string                 `json:"user_agent"`
	SessionID      string                 `json:"session_id"`
	AdditionalInfo map[string]interface{} `json:"additional_info"`
}

// AuditFilter 审计日志过滤器
type AuditFilter struct {
	SubjectID  string    `json:"subject_id"`
	ResourceID string    `json:"resource_id"`
	Action     string    `json:"action"`
	Decision   Effect    `json:"decision"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Limit      int       `json:"limit"`
	Offset     int       `json:"offset"`
}

// ObligationHandler 义务处理器接口
type ObligationHandler interface {
	// Handle 处理义务
	Handle(ctx context.Context, obligation *Obligation) error

	// CanHandle 检查是否能处理该义务
	CanHandle(obligation *Obligation) bool
}

// AdviceHandler 建议处理器接口
type AdviceHandler interface {
	// Apply 应用建议
	Apply(ctx context.Context, advice *Advice) error

	// CanApply 检查是否能应用该建议
	CanApply(advice *Advice) bool
}

// pep 策略执行点实现
type pep struct {
	pdp              PolicyDecisionPoint
	auditLogger      AuditLogger
	logger           logging.Logger
	metricsCollector *metrics.MetricsCollector
	tracer           trace.Tracer

	config             *PEPConfig
	obligationHandlers []ObligationHandler
	adviceHandlers     []AdviceHandler

	stats *pepStats
	mu    sync.RWMutex
}

// PEPConfig PEP 配置
type PEPConfig struct {
	EnableAudit           bool          `yaml:"enable_audit"`
	EnableMetrics         bool          `yaml:"enable_metrics"`
	EnableTracing         bool          `yaml:"enable_tracing"`
	EnforceObligations    bool          `yaml:"enforce_obligations"`
	ApplyAdvice           bool          `yaml:"apply_advice"`
	FailOnObligationError bool          `yaml:"fail_on_obligation_error"`
	AsyncEnforcement      bool          `yaml:"async_enforcement"`
	CacheDecisions        bool          `yaml:"cache_decisions"`
	CacheTTL              time.Duration `yaml:"cache_ttl"`
	MaxConcurrentRequests int           `yaml:"max_concurrent_requests"`
	RequestTimeout        time.Duration `yaml:"request_timeout"`
}

// pepStats PEP 统计信息
type pepStats struct {
	totalRequests     int64
	permittedRequests int64
	deniedRequests    int64
	totalLatency      time.Duration
	cacheHits         int64
	cacheMisses       int64
	errors            int64
	deniedActions     sync.Map
	deniedResources   sync.Map
	mu                sync.RWMutex
}

// NewPolicyEnforcementPoint 创建策略执行点
func NewPolicyEnforcementPoint(
	pdp PolicyDecisionPoint,
	auditLogger AuditLogger,
	logger logging.Logger,
	metricsCollector metrics.MetricsCollector,
	tracer trace.Tracer,
	config *PEPConfig,
) PolicyEnforcementPoint {
	return &pep{
		pdp:                pdp,
		auditLogger:        auditLogger,
		logger:             logger,
		metricsCollector:   metricsCollector,
		tracer:             tracer,
		config:             config,
		obligationHandlers: []ObligationHandler{},
		adviceHandlers:     []AdviceHandler{},
		stats:              &pepStats{},
	}
}

// Enforce 执行策略检查
func (p *pep) Enforce(ctx context.Context, request *AccessRequest) (*EnforcementResult, error) {
	ctx, span := p.tracer.Start(ctx, "PEP.Enforce")
	defer span.End()

	startTime := time.Now()

	// 设置请求超时
	if p.config.RequestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.config.RequestTimeout)
		defer cancel()
	}

 p.logger.WithContext(ctx).Debug("Enforcing access policy", logging.Any("request_id", request.RequestID), logging.Any("subject", request.Subject.ID), logging.Any("resource", request.Resource.ID), logging.Any("action", request.Action))

	// 调用 PDP 评估策略
	decision, err := p.pdp.Evaluate(ctx, request)
	if err != nil {
		p.recordError()
		p.logger.WithContext(ctx).Error("Policy evaluation failed", logging.Error(err))
		return nil, errors.Wrap(err, errors.CodeInternalError, "policy evaluation failed")
	}

	duration := time.Since(startTime)

	// 构建执行结果
	result := &EnforcementResult{
		RequestID:      request.RequestID,
		Allowed:        decision.Effect == EffectPermit,
		Decision:       decision,
		EnforcedAt:     time.Now(),
		Duration:       duration,
		ObligationsMet: true,
		AdviceApplied:  []string{},
		Metadata:       make(map[string]interface{}),
	}

	// 处理义务
	if p.config.EnforceObligations && len(decision.Obligations) > 0 {
		if err := p.handleObligations(ctx, decision.Obligations); err != nil {
			result.ObligationsMet = false
			p.logger.WithContext(ctx).Warn("Failed to fulfill obligations", logging.Error(err))

			if p.config.FailOnObligationError {
				result.Allowed = false
				result.Decision.Effect = EffectDeny
				result.Decision.Reason = fmt.Sprintf("obligation not met: %v", err)
			}
		}
	}

	// 应用建议
	if p.config.ApplyAdvice && len(decision.Advice) > 0 {
		appliedAdvice := p.applyAdvice(ctx, decision.Advice)
		result.AdviceApplied = appliedAdvice
	}

	// 记录审计日志
	if p.config.EnableAudit {
		auditID, err := p.logAudit(ctx, request, decision, duration)
		if err != nil {
			p.logger.WithContext(ctx).Warn("Failed to log audit entry", logging.Error(err))
		} else {
			result.AuditID = auditID
		}
	}

	// 更新统计信息
	p.updateStats(request, decision, duration)

	// 记录指标
	if p.config.EnableMetrics {
		p.recordMetrics(request, decision, duration)
	}

 p.logger.WithContext(ctx).Info("Access policy enforced", logging.Any("request_id", request.RequestID), logging.Any("allowed", result.Allowed), logging.Any("duration_ms", duration.Milliseconds())

	return result, nil
}

// EnforceAsync 异步执行策略检查
func (p *pep) EnforceAsync(ctx context.Context, request *AccessRequest) <-chan *EnforcementResult {
	resultChan := make(chan *EnforcementResult, 1)

	go func() {
		defer close(resultChan)

		result, err := p.Enforce(ctx, request)
		if err != nil {
			p.logger.WithContext(ctx).Error("Async enforcement failed", logging.Error(err))
			resultChan <- &EnforcementResult{
				RequestID:  request.RequestID,
				Allowed:    false,
				EnforcedAt: time.Now(),
				Metadata: map[string]interface{}{
					"error": err.Error(),
				},
			}
			return
		}

		resultChan <- result
	}()

	return resultChan
}

// EnforceBatch 批量执行策略检查
func (p *pep) EnforceBatch(ctx context.Context, requests []*AccessRequest) ([]*EnforcementResult, error) {
	ctx, span := p.tracer.Start(ctx, "PEP.EnforceBatch")
	defer span.End()

	p.logger.WithContext(ctx).Debug("Enforcing batch access policies", logging.Any("count", len(requests))

	results := make([]*EnforcementResult, len(requests))

	if p.config.AsyncEnforcement {
		// 并发执行
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, p.config.MaxConcurrentRequests)

		for i, req := range requests {
			wg.Add(1)
			go func(index int, request *AccessRequest) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				result, err := p.Enforce(ctx, request)
				if err != nil {
     p.logger.WithContext(ctx).Error("Batch enforcement failed for request", logging.Any("request_id", request.RequestID), logging.Error(err))
					results[index] = &EnforcementResult{
						RequestID: request.RequestID,
						Allowed:   false,
						Metadata: map[string]interface{}{
							"error": err.Error(),
						},
					}
					return
				}
				results[index] = result
			}(i, req)
		}

		wg.Wait()
	} else {
		// 顺序执行
		for i, req := range requests {
			result, err := p.Enforce(ctx, req)
			if err != nil {
				return nil, err
			}
			results[i] = result
		}
	}

	return results, nil
}

// PreAuthorize 预授权检查
func (p *pep) PreAuthorize(ctx context.Context, subjectID, resourceID, action string) error {
	ctx, span := p.tracer.Start(ctx, "PEP.PreAuthorize")
	defer span.End()

	request := &AccessRequest{
		RequestID: fmt.Sprintf("preauth_%s_%d", resourceID, time.Now().UnixNano()),
		Subject: &Subject{
			ID:         subjectID,
			Type:       "user",
			Attributes: make(map[string]interface{}),
		},
		Resource: &Resource{
			ID:         resourceID,
			Attributes: make(map[string]interface{}),
		},
		Action:      action,
		Environment: make(map[string]interface{}),
		Context:     make(map[string]interface{}),
		Timestamp:   time.Now(),
	}

	result, err := p.Enforce(ctx, request)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "pre-authorization failed")
	}

	if !result.Allowed {
		return errors.New(errors.CodePermissionDenied,
			fmt.Sprintf("access denied: %s", result.Decision.Reason))
	}

	return nil
}

// PostAuthorize 后授权检查
func (p *pep) PostAuthorize(ctx context.Context, request *AccessRequest, result interface{}) error {
	ctx, span := p.tracer.Start(ctx, "PEP.PostAuthorize")
	defer span.End()

	// 后授权主要用于检查返回结果是否符合策略
	// 例如：检查返回的数据是否包含敏感信息

	enforcementResult, err := p.Enforce(ctx, request)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "post-authorization failed")
	}

	if !enforcementResult.Allowed {
		return errors.New(errors.CodePermissionDenied,
			fmt.Sprintf("post-authorization denied: %s", enforcementResult.Decision.Reason))
	}

	// 可以在这里添加对结果的进一步检查和过滤
	p.logger.WithContext(ctx).Debug("Post-authorization check passed", logging.Any("request_id", request.RequestID))

	return nil
}

// GetEnforcementStats 获取执行统计
func (p *pep) GetEnforcementStats(ctx context.Context) (*EnforcementStats, error) {
	ctx, span := p.tracer.Start(ctx, "PEP.GetEnforcementStats")
	defer span.End()

	p.stats.mu.RLock()
	defer p.stats.mu.RUnlock()

	var avgLatency time.Duration
	if p.stats.totalRequests > 0 {
		avgLatency = p.stats.totalLatency / time.Duration(p.stats.totalRequests)
	}

	var cacheHitRate float64
	totalCacheAccess := p.stats.cacheHits + p.stats.cacheMisses
	if totalCacheAccess > 0 {
		cacheHitRate = float64(p.stats.cacheHits) / float64(totalCacheAccess)
	}

	var errorRate float64
	if p.stats.totalRequests > 0 {
		errorRate = float64(p.stats.errors) / float64(p.stats.totalRequests)
	}

	topDeniedActions := make(map[string]int64)
	p.stats.deniedActions.Range(func(key, value interface{}) bool {
		topDeniedActions[key.(string)] = value.(int64)
		return true
	})

	topDeniedResources := make(map[string]int64)
	p.stats.deniedResources.Range(func(key, value interface{}) bool {
		topDeniedResources[key.(string)] = value.(int64)
		return true
	})

	return &EnforcementStats{
		TotalRequests:      p.stats.totalRequests,
		PermittedRequests:  p.stats.permittedRequests,
		DeniedRequests:     p.stats.deniedRequests,
		AverageLatency:     avgLatency,
		CacheHitRate:       cacheHitRate,
		ErrorRate:          errorRate,
		TopDeniedActions:   topDeniedActions,
		TopDeniedResources: topDeniedResources,
	}, nil
}

// RegisterObligationHandler 注册义务处理器
func (p *pep) RegisterObligationHandler(handler ObligationHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.obligationHandlers = append(p.obligationHandlers, handler)
}

// RegisterAdviceHandler 注册建议处理器
func (p *pep) RegisterAdviceHandler(handler AdviceHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.adviceHandlers = append(p.adviceHandlers, handler)
}

// Helper methods

func (p *pep) handleObligations(ctx context.Context, obligations []*Obligation) error {
	p.mu.RLock()
	handlers := p.obligationHandlers
	p.mu.RUnlock()

	for _, obligation := range obligations {
		handled := false

		for _, handler := range handlers {
			if handler.CanHandle(obligation) {
				if err := handler.Handle(ctx, obligation); err != nil {
					return errors.Wrap(err, errors.CodeInternalError,
						fmt.Sprintf("failed to handle obligation: %s", obligation.ID))
				}
				handled = true
				break
			}
		}

		if !handled {
   p.logger.WithContext(ctx).Warn("No handler found for obligation", logging.Any("obligation_id", obligation.ID), logging.Any("type", obligation.Type))
		}
	}

	return nil
}

func (p *pep) applyAdvice(ctx context.Context, adviceList []*Advice) []string {
	p.mu.RLock()
	handlers := p.adviceHandlers
	p.mu.RUnlock()

	applied := []string{}

	for _, advice := range adviceList {
		for _, handler := range handlers {
			if handler.CanApply(advice) {
				if err := handler.Apply(ctx, advice); err != nil {
     p.logger.WithContext(ctx).Warn("Failed to apply advice", logging.Any("advice_id", advice.ID), logging.Error(err))
				} else {
					applied = append(applied, advice.ID)
				}
				break
			}
		}
	}

	return applied
}

func (p *pep) logAudit(ctx context.Context, request *AccessRequest, decision *Decision, duration time.Duration) (string, error) {
	auditEntry := &AuditEntry{
		ID:           fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		RequestID:    request.RequestID,
		Timestamp:    time.Now(),
		SubjectID:    request.Subject.ID,
		SubjectType:  request.Subject.Type,
		ResourceID:   request.Resource.ID,
		ResourceType: request.Resource.Type,
		Action:       request.Action,
		Decision:     decision.Effect,
		Reason:       decision.Reason,
		Policies:     decision.ApplicablePolicies,
		Duration:     duration,
		AdditionalInfo: map[string]interface{}{
			"obligations_count": len(decision.Obligations),
			"advice_count":      len(decision.Advice),
		},
	}

	// 从请求上下文提取额外信息
	if sourceIP, ok := request.Context["source_ip"].(string); ok {
		auditEntry.SourceIP = sourceIP
	}
	if userAgent, ok := request.Context["user_agent"].(string); ok {
		auditEntry.UserAgent = userAgent
	}
	if sessionID, ok := request.Context["session_id"].(string); ok {
		auditEntry.SessionID = sessionID
	}

	if err := p.auditLogger.LogAccess(ctx, auditEntry); err != nil {
		return "", err
	}

	return auditEntry.ID, nil
}

func (p *pep) updateStats(request *AccessRequest, decision *Decision, duration time.Duration) {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()

	p.stats.totalRequests++
	p.stats.totalLatency += duration

	if decision.Effect == EffectPermit {
		p.stats.permittedRequests++
	} else {
		p.stats.deniedRequests++

		// 记录被拒绝的操作
		if count, ok := p.stats.deniedActions.Load(request.Action); ok {
			p.stats.deniedActions.Store(request.Action, count.(int64)+1)
		} else {
			p.stats.deniedActions.Store(request.Action, int64(1))
		}

		// 记录被拒绝的资源
		if count, ok := p.stats.deniedResources.Load(request.Resource.ID); ok {
			p.stats.deniedResources.Store(request.Resource.ID, count.(int64)+1)
		} else {
			p.stats.deniedResources.Store(request.Resource.ID, int64(1))
		}
	}
}

func (p *pep) recordError() {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()
	p.stats.errors++
}

func (p *pep) recordMetrics(request *AccessRequest, decision *Decision, duration time.Duration) {
	p.metricsCollector.Histogram("pep_enforcement_duration_ms",
		float64(duration.Milliseconds()),
		map[string]string{
			"resource_type": request.Resource.Type,
			"action":        request.Action,
		})

	p.metricsCollector.Increment("pep_enforcement_total",
		map[string]string{
			"effect":        string(decision.Effect),
			"resource_type": request.Resource.Type,
			"action":        request.Action,
		})
}

// InMemoryAuditLogger 内存审计日志实现
type InMemoryAuditLogger struct {
	entries []AuditEntry
	mu      sync.RWMutex
}

// NewInMemoryAuditLogger 创建内存审计日志
func NewInMemoryAuditLogger() AuditLogger {
	return &InMemoryAuditLogger{
		entries: []AuditEntry{},
	}
}

func (l *InMemoryAuditLogger) LogAccess(ctx context.Context, entry *AuditEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.entries = append(l.entries, *entry)

	// 限制内存中的条目数量
	if len(l.entries) > 10000 {
		l.entries = l.entries[1:]
	}

	return nil
}

func (l *InMemoryAuditLogger) Query(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := []*AuditEntry{}

	for i := range l.entries {
		entry := &l.entries[i]

		// 应用过滤器
		if filter.SubjectID != "" && entry.SubjectID != filter.SubjectID {
			continue
		}
		if filter.ResourceID != "" && entry.ResourceID != filter.ResourceID {
			continue
		}
		if filter.Action != "" && entry.Action != filter.Action {
			continue
		}
		if filter.Decision != "" && entry.Decision != filter.Decision {
			continue
		}
		if !filter.StartTime.IsZero() && entry.Timestamp.Before(filter.StartTime) {
			continue
		}
		if !filter.EndTime.IsZero() && entry.Timestamp.After(filter.EndTime) {
			continue
		}

		result = append(result, entry)
	}

	// 应用分页
	if filter.Limit > 0 {
		start := filter.Offset
		end := start + filter.Limit

		if start >= len(result) {
			return []*AuditEntry{}, nil
		}
		if end > len(result) {
			end = len(result)
		}

		result = result[start:end]
	}

	return result, nil
}

// 默认义务处理器

// LoggingObligationHandler 日志义务处理器
type LoggingObligationHandler struct {
	logger logging.Logger
}

func NewLoggingObligationHandler(logger logging.Logger) ObligationHandler {
	return &LoggingObligationHandler{logger: logger}
}

func (h *LoggingObligationHandler) Handle(ctx context.Context, obligation *Obligation) error {
 h.logger.WithContext(ctx).Info("Obligation fulfilled", logging.Any("obligation_id", obligation.ID), logging.Any("type", obligation.Type), logging.Any("description", obligation.Description))
	return nil
}

func (h *LoggingObligationHandler) CanHandle(obligation *Obligation) bool {
	return obligation.Type == "logging" || obligation.Type == "notification"
}

// NotificationObligationHandler 通知义务处理器
type NotificationObligationHandler struct {
	logger logging.Logger
}

func NewNotificationObligationHandler(logger logging.Logger) ObligationHandler {
	return &NotificationObligationHandler{logger: logger}
}

func (h *NotificationObligationHandler) Handle(ctx context.Context, obligation *Obligation) error {
	// 实际实现应该发送通知（邮件、消息等）
 h.logger.WithContext(ctx).Info("Notification sent", logging.Any("obligation_id", obligation.ID), logging.Any("description", obligation.Description))
	return nil
}

func (h *NotificationObligationHandler) CanHandle(obligation *Obligation) bool {
	return obligation.Type == "notification" || obligation.Type == "alert"
}

// 默认建议处理器

// LoggingAdviceHandler 日志建议处理器
type LoggingAdviceHandler struct {
	logger logging.Logger
}

func NewLoggingAdviceHandler(logger logging.Logger) AdviceHandler {
	return &LoggingAdviceHandler{logger: logger}
}

func (h *LoggingAdviceHandler) Apply(ctx context.Context, advice *Advice) error {
 h.logger.WithContext(ctx).Info("Advice applied", logging.Any("advice_id", advice.ID), logging.Any("type", advice.Type), logging.Any("description", advice.Description))
	return nil
}

func (h *LoggingAdviceHandler) CanApply(advice *Advice) bool {
	return true // 可以处理所有建议
}

//Personal.AI order the ending
