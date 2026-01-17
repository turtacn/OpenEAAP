// internal/governance/policy/pdp.go
package policy

import (
"github.com/prometheus/client_golang/prometheus"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/pkg/errors"
)

// PolicyDecisionPoint 策略决策点接口
type PolicyDecisionPoint interface {
	// Evaluate 评估访问请求
	Evaluate(ctx context.Context, request *AccessRequest) (*Decision, error)

	// EvaluateBatch 批量评估访问请求
	EvaluateBatch(ctx context.Context, requests []*AccessRequest) ([]*Decision, error)

	// AddPolicy 添加策略
	AddPolicy(ctx context.Context, policy *Policy) error

	// RemovePolicy 移除策略
	RemovePolicy(ctx context.Context, policyID string) error

	// UpdatePolicy 更新策略
	UpdatePolicy(ctx context.Context, policy *Policy) error

	// GetPolicy 获取策略
	GetPolicy(ctx context.Context, policyID string) (*Policy, error)

	// ListPolicies 列出所有策略
	ListPolicies(ctx context.Context, filter *PolicyFilter) ([]*Policy, error)
}

// AccessRequest 访问请求
type AccessRequest struct {
	RequestID   string                 `json:"request_id"`
	Subject     *Subject               `json:"subject"`     // 主体（用户/服务）
	Resource    *Resource              `json:"resource"`    // 资源
	Action      string                 `json:"action"`      // 操作
	Environment map[string]interface{} `json:"environment"` // 环境属性
	Context     map[string]interface{} `json:"context"`     // 上下文信息
	Timestamp   time.Time              `json:"timestamp"`
}

// Subject 主体信息
type Subject struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`       // user/service/agent
	Attributes map[string]interface{} `json:"attributes"` // 主体属性
	Roles      []string               `json:"roles"`      // 角色列表
	Groups     []string               `json:"groups"`     // 组列表
}

// Resource 资源信息
type Resource struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`       // model/agent/dataset/api
	Attributes map[string]interface{} `json:"attributes"` // 资源属性
	Owner      string                 `json:"owner"`      // 资源所有者
	Tags       []string               `json:"tags"`       // 资源标签
}

// Decision 决策结果
type Decision struct {
	RequestID          string                 `json:"request_id"`
	Effect             Effect                 `json:"effect"`              // Permit/Deny
	Reason             string                 `json:"reason"`              // 决策原因
	ApplicablePolicies []string               `json:"applicable_policies"` // 应用的策略
	Obligations        []*Obligation          `json:"obligations"`         // 义务
	Advice             []*Advice              `json:"advice"`              // 建议
	EvaluatedAt        time.Time              `json:"evaluated_at"`
	Duration           time.Duration          `json:"duration"` // 评估耗时
	Metadata           map[string]interface{} `json:"metadata"`
}

// Effect 决策效果
type Effect string

const (
	EffectPermit        Effect = "Permit"
	EffectDeny          Effect = "Deny"
	EffectNotApplicable Effect = "NotApplicable"
	EffectIndeterminate Effect = "Indeterminate"
)

// Obligation 义务（必须执行的动作）
type Obligation struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Advice 建议（可选执行的动作）
type Advice struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Policy 策略
type Policy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        PolicyType             `json:"type"`        // RBAC/ABAC
	Priority    int                    `json:"priority"`    // 优先级，数字越大优先级越高
	Effect      Effect                 `json:"effect"`      // 策略效果
	Target      *Target                `json:"target"`      // 策略目标
	Condition   *Condition             `json:"condition"`   // 策略条件
	Rules       []*Rule                `json:"rules"`       // 规则列表
	Obligations []*Obligation          `json:"obligations"` // 义务
	Advice      []*Advice              `json:"advice"`      // 建议
	Enabled     bool                   `json:"enabled"`     // 是否启用
	Version     string                 `json:"version"`     // 策略版本
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// PolicyType 策略类型
type PolicyType string

const (
	PolicyTypeRBAC PolicyType = "RBAC" // 基于角色的访问控制
	PolicyTypeABAC PolicyType = "ABAC" // 基于属性的访问控制
)

// Target 策略目标
type Target struct {
	Subjects  []*Matcher `json:"subjects"`  // 主体匹配器
	Resources []*Matcher `json:"resources"` // 资源匹配器
	Actions   []string   `json:"actions"`   // 操作列表
}

// Matcher 匹配器
type Matcher struct {
	Type     string      `json:"type"`     // exact/regex/contains/range
	Field    string      `json:"field"`    // 匹配字段
	Value    interface{} `json:"value"`    // 匹配值
	Operator string      `json:"operator"` // eq/ne/gt/lt/in/contains
}

// Condition 条件
type Condition struct {
	Type       string       `json:"type"`       // and/or/not
	Conditions []*Condition `json:"conditions"` // 子条件
	Expression string       `json:"expression"` // 条件表达式
	Attributes []*Attribute `json:"attributes"` // 属性条件
}

// Attribute 属性条件
type Attribute struct {
	Category string      `json:"category"` // subject/resource/environment/action
	Name     string      `json:"name"`
	Operator string      `json:"operator"` // eq/ne/gt/lt/in/contains
	Value    interface{} `json:"value"`
}

// Rule 规则
type Rule struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Effect      Effect                 `json:"effect"`
	Condition   *Condition             `json:"condition"`
	Priority    int                    `json:"priority"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// PolicyFilter 策略过滤器
type PolicyFilter struct {
	Type     PolicyType `json:"type"`
	Enabled  *bool      `json:"enabled"`
	TargetID string     `json:"target_id"`
	Limit    int        `json:"limit"`
	Offset   int        `json:"offset"`
}

// PolicyRepository 策略仓储接口
type PolicyRepository interface {
	Create(ctx context.Context, policy *Policy) error
	GetByID(ctx context.Context, id string) (*Policy, error)
	Update(ctx context.Context, policy *Policy) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter *PolicyFilter) ([]*Policy, error)
	GetApplicablePolicies(ctx context.Context, request *AccessRequest) ([]*Policy, error)
}

// pdp 策略决策点实现
type pdp struct {
	policyRepo       PolicyRepository
	logger           logging.Logger
	metricsCollector metrics.MetricsCollector
	tracer           trace.Tracer

	config          *PDPConfig
	policyCache     sync.Map // 策略缓存
	evaluationCache sync.Map // 评估结果缓存
	mu              sync.RWMutex
}

// PDPConfig PDP 配置
type PDPConfig struct {
	CacheEnabled         bool          `yaml:"cache_enabled"`
	CacheTTL             time.Duration `yaml:"cache_ttl"`
	ConcurrentEvaluation bool          `yaml:"concurrent_evaluation"`
	MaxConcurrency       int           `yaml:"max_concurrency"`
	DefaultEffect        Effect        `yaml:"default_effect"`
	CombiningAlgorithm   string        `yaml:"combining_algorithm"` // deny-overrides/permit-overrides/first-applicable
	EnableAudit          bool          `yaml:"enable_audit"`
	EnableMetrics        bool          `yaml:"enable_metrics"`
}

// NewPolicyDecisionPoint 创建策略决策点
func NewPolicyDecisionPoint(
	policyRepo PolicyRepository,
	logger logging.Logger,
	metricsCollector metrics.MetricsCollector,
	tracer trace.Tracer,
	config *PDPConfig,
) PolicyDecisionPoint {
	return &pdp{
		policyRepo:       policyRepo,
		logger:           logger,
		metricsCollector: metricsCollector,
		tracer:           tracer,
		config:           config,
	}
}

// Evaluate 评估访问请求
func (p *pdp) Evaluate(ctx context.Context, request *AccessRequest) (*Decision, error) {
	ctx, span := p.tracer.Start(ctx, "PDP.Evaluate")
	defer span.End()

	startTime := time.Now()
	defer func() {
		if p.config.EnableMetrics {
			p.metricsCollector.ObserveDuration("pdp_evaluation_duration_ms",
				startTime,
				prometheus.Labels{"resource_type": request.Resource.Type})
		}
	}()

 p.logger.WithContext(ctx).Debug("Evaluating access request", logging.Any("request_id", request.RequestID), logging.Any("subject", request.Subject.ID), logging.Any("resource", request.Resource.ID), logging.Any("action", request.Action))

	// 检查缓存
	if p.config.CacheEnabled {
		if cached, ok := p.getCachedDecision(request); ok {
			p.logger.WithContext(ctx).Debug("Decision found in cache", logging.Any("request_id", request.RequestID))
			return cached, nil
		}
	}

	// 获取适用的策略
	policies, err := p.policyRepo.GetApplicablePolicies(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to get applicable policies")
	}

	if len(policies) == 0 {
		p.logger.WithContext(ctx).Debug("No applicable policies found", logging.Any("request_id", request.RequestID))
		return p.createDecision(request, p.config.DefaultEffect, "no applicable policies", nil), nil
	}

	// 按优先级排序策略
	sortPoliciesByPriority(policies)

	// 评估策略
	decision, err := p.evaluatePolicies(ctx, request, policies)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to evaluate policies")
	}

	// 缓存结果
	if p.config.CacheEnabled {
		p.cacheDecision(request, decision)
	}

	// 审计日志
	if p.config.EnableAudit {
		p.auditDecision(ctx, request, decision)
	}

	// 记录指标
	if p.config.EnableMetrics {
		p.metricsCollector.IncrementCounter("pdp_decisions_total",
			map[string]string{
				"effect":        string(decision.Effect),
				"resource_type": request.Resource.Type,
			})
	}

 p.logger.WithContext(ctx).Info("Access request evaluated", logging.Any("request_id", request.RequestID), logging.Any("effect", decision.Effect), logging.Duration("duration_ms", time.Since(startTime)))

	return decision, nil
}

// EvaluateBatch 批量评估访问请求
func (p *pdp) EvaluateBatch(ctx context.Context, requests []*AccessRequest) ([]*Decision, error) {
	ctx, span := p.tracer.Start(ctx, "PDP.EvaluateBatch")
	defer span.End()

	p.logger.WithContext(ctx).Debug("Evaluating batch requests", logging.Any("count", len(requests)))

	decisions := make([]*Decision, len(requests))

	if p.config.ConcurrentEvaluation {
		// 并发评估
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, p.config.MaxConcurrency)
		errorsChan := make(chan error, len(requests))

		for i, req := range requests {
			wg.Add(1)
			go func(index int, request *AccessRequest) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				decision, err := p.Evaluate(ctx, request)
				if err != nil {
					errorsChan <- err
					return
				}
				decisions[index] = decision
			}(i, req)
		}

		wg.Wait()
		close(errorsChan)

		// 检查错误
		for err := range errorsChan {
			if err != nil {
				return nil, err
			}
		}
	} else {
		// 顺序评估
		for i, req := range requests {
			decision, err := p.Evaluate(ctx, req)
			if err != nil {
				return nil, err
			}
			decisions[i] = decision
		}
	}

	return decisions, nil
}

// AddPolicy 添加策略
func (p *pdp) AddPolicy(ctx context.Context, policy *Policy) error {
	ctx, span := p.tracer.Start(ctx, "PDP.AddPolicy")
	defer span.End()

	p.logger.WithContext(ctx).Info("Adding policy", logging.Any("policy_id", policy.ID), logging.Any("policy_name", policy.Name))

	// 验证策略
	if err := p.validatePolicy(policy); err != nil {
		return errors.Wrap(err, errors.CodeInvalidArgument, "invalid policy")
	}

	// 设置默认值
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = time.Now()
	}
	policy.UpdatedAt = time.Now()

	// 保存到仓储
	if err := p.policyRepo.Create(ctx, policy); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to create policy")
	}

	// 更新缓存
	p.policyCache.Store(policy.ID, policy)

	// 清空评估缓存（策略变更）
	if p.config.CacheEnabled {
		p.clearEvaluationCache()
	}

	p.logger.WithContext(ctx).Info("Policy added successfully", logging.Any("policy_id", policy.ID))

	return nil
}

// RemovePolicy 移除策略
func (p *pdp) RemovePolicy(ctx context.Context, policyID string) error {
	ctx, span := p.tracer.Start(ctx, "PDP.RemovePolicy")
	defer span.End()

	p.logger.WithContext(ctx).Info("Removing policy", logging.Any("policy_id", policyID))

	if err := p.policyRepo.Delete(ctx, policyID); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to delete policy")
	}

	// 从缓存移除
	p.policyCache.Delete(policyID)

	// 清空评估缓存
	if p.config.CacheEnabled {
		p.clearEvaluationCache()
	}

	p.logger.WithContext(ctx).Info("Policy removed successfully", logging.Any("policy_id", policyID))

	return nil
}

// UpdatePolicy 更新策略
func (p *pdp) UpdatePolicy(ctx context.Context, policy *Policy) error {
	ctx, span := p.tracer.Start(ctx, "PDP.UpdatePolicy")
	defer span.End()

	p.logger.WithContext(ctx).Info("Updating policy", logging.Any("policy_id", policy.ID))

	// 验证策略
	if err := p.validatePolicy(policy); err != nil {
		return errors.Wrap(err, errors.CodeInvalidArgument, "invalid policy")
	}

	policy.UpdatedAt = time.Now()

	if err := p.policyRepo.Update(ctx, policy); err != nil {
		return errors.Wrap(err, "ERR_INTERNAL", "failed to update policy")
	}

	// 更新缓存
	p.policyCache.Store(policy.ID, policy)

	// 清空评估缓存
	if p.config.CacheEnabled {
		p.clearEvaluationCache()
	}

	p.logger.WithContext(ctx).Info("Policy updated successfully", logging.Any("policy_id", policy.ID))

	return nil
}

// GetPolicy 获取策略
func (p *pdp) GetPolicy(ctx context.Context, policyID string) (*Policy, error) {
	ctx, span := p.tracer.Start(ctx, "PDP.GetPolicy")
	defer span.End()

	// 先检查缓存
	if cached, ok := p.policyCache.Load(policyID); ok {
		return cached.(*Policy), nil
	}

	policy, err := p.policyRepo.GetByID(ctx, policyID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "policy not found")
	}

	// 更新缓存
	p.policyCache.Store(policyID, policy)

	return policy, nil
}

// ListPolicies 列出所有策略
func (p *pdp) ListPolicies(ctx context.Context, filter *PolicyFilter) ([]*Policy, error) {
	ctx, span := p.tracer.Start(ctx, "PDP.ListPolicies")
	defer span.End()

	policies, err := p.policyRepo.List(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "ERR_INTERNAL", "failed to list policies")
	}

	return policies, nil
}

// Helper methods

func (p *pdp) evaluatePolicies(ctx context.Context, request *AccessRequest, policies []*Policy) (*Decision, error) {
	switch p.config.CombiningAlgorithm {
	case "deny-overrides":
		return p.evaluateDenyOverrides(ctx, request, policies)
	case "permit-overrides":
		return p.evaluatePermitOverrides(ctx, request, policies)
	case "first-applicable":
		return p.evaluateFirstApplicable(ctx, request, policies)
	default:
		return p.evaluateDenyOverrides(ctx, request, policies)
	}
}

func (p *pdp) evaluateDenyOverrides(ctx context.Context, request *AccessRequest, policies []*Policy) (*Decision, error) {
	applicablePolicies := []string{}
	obligations := []*Obligation{}
	advice := []*Advice{}
	permitFound := false

	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}

		// 检查目标匹配
		if !p.matchTarget(request, policy.Target) {
			continue
		}

		// 评估条件
		conditionResult, err := p.evaluateCondition(ctx, request, policy.Condition)
		if err != nil {
			p.logger.WithContext(ctx).Warn("Failed to evaluate condition", logging.Any("policy_id", policy.ID), logging.Error(err))
			continue
		}

		if !conditionResult {
			continue
		}

		applicablePolicies = append(applicablePolicies, policy.ID)

		// Deny 优先
		if policy.Effect == EffectDeny {
			obligations = append(obligations, policy.Obligations...)
			advice = append(advice, policy.Advice...)
			return p.createDecision(request, EffectDeny,
				fmt.Sprintf("denied by policy: %s", policy.Name),
				applicablePolicies, obligations, advice), nil
		}

		if policy.Effect == EffectPermit {
			permitFound = true
			obligations = append(obligations, policy.Obligations...)
			advice = append(advice, policy.Advice...)
		}
	}

	if permitFound {
		return p.createDecision(request, EffectPermit, "permitted by policies",
			applicablePolicies, obligations, advice), nil
	}

	return p.createDecision(request, p.config.DefaultEffect, "no permit found",
		applicablePolicies), nil
}

func (p *pdp) evaluatePermitOverrides(ctx context.Context, request *AccessRequest, policies []*Policy) (*Decision, error) {
	applicablePolicies := []string{}
	obligations := []*Obligation{}
	advice := []*Advice{}
	denyFound := false

	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}

		if !p.matchTarget(request, policy.Target) {
			continue
		}

		conditionResult, err := p.evaluateCondition(ctx, request, policy.Condition)
		if err != nil {
			p.logger.WithContext(ctx).Warn("Failed to evaluate condition", logging.Any("policy_id", policy.ID), logging.Error(err))
			continue
		}

		if !conditionResult {
			continue
		}

		applicablePolicies = append(applicablePolicies, policy.ID)

		// Permit 优先
		if policy.Effect == EffectPermit {
			obligations = append(obligations, policy.Obligations...)
			advice = append(advice, policy.Advice...)
			return p.createDecision(request, EffectPermit,
				fmt.Sprintf("permitted by policy: %s", policy.Name),
				applicablePolicies, obligations, advice), nil
		}

		if policy.Effect == EffectDeny {
			denyFound = true
			obligations = append(obligations, policy.Obligations...)
			advice = append(advice, policy.Advice...)
		}
	}

	if denyFound {
		return p.createDecision(request, EffectDeny, "denied by policies",
			applicablePolicies, obligations, advice), nil
	}

	return p.createDecision(request, p.config.DefaultEffect, "no deny found",
		applicablePolicies), nil
}

func (p *pdp) evaluateFirstApplicable(ctx context.Context, request *AccessRequest, policies []*Policy) (*Decision, error) {
	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}

		if !p.matchTarget(request, policy.Target) {
			continue
		}

		conditionResult, err := p.evaluateCondition(ctx, request, policy.Condition)
		if err != nil {
			p.logger.WithContext(ctx).Warn("Failed to evaluate condition", logging.Any("policy_id", policy.ID), logging.Error(err))
			continue
		}

		if !conditionResult {
			continue
		}

		// 第一个匹配的策略
		return p.createDecision(request, policy.Effect,
			fmt.Sprintf("matched policy: %s", policy.Name),
			[]string{policy.ID}, policy.Obligations, policy.Advice), nil
	}

	return p.createDecision(request, p.config.DefaultEffect, "no applicable policy", nil), nil
}

func (p *pdp) matchTarget(request *AccessRequest, target *Target) bool {
	if target == nil {
		return true
	}

	// 匹配主体
	if len(target.Subjects) > 0 {
		matched := false
		for _, matcher := range target.Subjects {
			if p.matchSubject(request.Subject, matcher) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 匹配资源
	if len(target.Resources) > 0 {
		matched := false
		for _, matcher := range target.Resources {
			if p.matchResource(request.Resource, matcher) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 匹配操作
	if len(target.Actions) > 0 {
		matched := false
		for _, action := range target.Actions {
			if action == request.Action || action == "*" {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

func (p *pdp) matchSubject(subject *Subject, matcher *Matcher) bool {
	return p.matchEntity(subject.Attributes, subject.ID, matcher)
}

func (p *pdp) matchResource(resource *Resource, matcher *Matcher) bool {
	return p.matchEntity(resource.Attributes, resource.ID, matcher)
}

func (p *pdp) matchEntity(attributes map[string]interface{}, id string, matcher *Matcher) bool {
	var fieldValue interface{}

	if matcher.Field == "id" {
		fieldValue = id
	} else {
		fieldValue = attributes[matcher.Field]
	}

	return p.matchValue(fieldValue, matcher.Value, matcher.Operator)
}

func (p *pdp) matchValue(actual, expected interface{}, operator string) bool {
	switch operator {
	case "eq", "":
		return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
	case "ne":
		return fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected)
	case "contains":
		return contains(fmt.Sprintf("%v", actual), fmt.Sprintf("%v", expected))
	case "in":
		if list, ok := expected.([]interface{}); ok {
			for _, item := range list {
				if fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", item) {
					return true
				}
			}
		}
		return false
	default:
		return false
	}
}

func (p *pdp) evaluateCondition(ctx context.Context, request *AccessRequest, condition *Condition) (bool, error) {
	if condition == nil {
		return true, nil
	}

	switch condition.Type {
	case "and":
		for _, subCond := range condition.Conditions {
			result, err := p.evaluateCondition(ctx, request, subCond)
			if err != nil || !result {
				return false, err
			}
		}
		return true, nil

	case "or":
		for _, subCond := range condition.Conditions {
			result, err := p.evaluateCondition(ctx, request, subCond)
			if err == nil && result {
				return true, nil
			}
		}
		return false, nil

	case "not":
		if len(condition.Conditions) > 0 {
			result, err := p.evaluateCondition(ctx, request, condition.Conditions[0])
			return !result, err
		}
		return false, nil

	default:
		// 评估属性条件
		return p.evaluateAttributes(request, condition.Attributes), nil
	}
}

func (p *pdp) evaluateAttributes(request *AccessRequest, attributes []*Attribute) bool {
	for _, attr := range attributes {
		var value interface{}

		switch attr.Category {
		case "subject":
			value = request.Subject.Attributes[attr.Name]
		case "resource":
			value = request.Resource.Attributes[attr.Name]
		case "environment":
			value = request.Environment[attr.Name]
		case "action":
			value = request.Action
		}

		if !p.matchValue(value, attr.Value, attr.Operator) {
			return false
		}
	}

	return true
}

func (p *pdp) createDecision(request *AccessRequest, effect Effect, reason string, applicablePolicies []string, obligationsAndAdvice ...[]*Obligation) *Decision {
	decision := &Decision{
		RequestID:          request.RequestID,
		Effect:             effect,
		Reason:             reason,
		ApplicablePolicies: applicablePolicies,
		EvaluatedAt:        time.Now(),
		Duration:           time.Since(request.Timestamp),
		Metadata:           make(map[string]interface{}),
	}

	if len(obligationsAndAdvice) > 0 {
		decision.Obligations = obligationsAndAdvice[0]
	}
	if len(obligationsAndAdvice) > 1 {
		// Type assertion needed for advice
		if advice, ok := interface{}(obligationsAndAdvice[1]).([]*Advice); ok {
			decision.Advice = advice
		}
	}

	return decision
}

func (p *pdp) validatePolicy(policy *Policy) error {
	if policy.ID == "" {
		return errors.NewInternalError(errors.CodeInvalidArgument, "policy ID is required")
	}
	if policy.Name == "" {
		return errors.NewInternalError(errors.CodeInvalidArgument, "policy name is required")
	}
	if policy.Effect != EffectPermit && policy.Effect != EffectDeny {
		return errors.NewInternalError(errors.CodeInvalidArgument, "invalid policy effect")
	}
	return nil
}

func (p *pdp) getCachedDecision(request *AccessRequest) (*Decision, bool) {
	key := p.getCacheKey(request)
	if cached, ok := p.evaluationCache.Load(key); ok {
		if decision, ok := cached.(*Decision); ok {
			if time.Since(decision.EvaluatedAt) < p.config.CacheTTL {
				return decision, true
			}
			p.evaluationCache.Delete(key)
		}
	}
	return nil, false
}

func (p *pdp) cacheDecision(request *AccessRequest, decision *Decision) {
	key := p.getCacheKey(request)
	p.evaluationCache.Store(key, decision)
}

func (p *pdp) getCacheKey(request *AccessRequest) string {
	return fmt.Sprintf("%s:%s:%s:%s",
		request.Subject.ID,
		request.Resource.ID,
		request.Action,
		request.Resource.Type)
}

func (p *pdp) clearEvaluationCache() {
	p.evaluationCache = sync.Map{}
	p.logger.Debug("Evaluation cache cleared")
}

func (p *pdp) auditDecision(ctx context.Context, request *AccessRequest, decision *Decision) {
	auditEntry := map[string]interface{}{
		"request_id":          request.RequestID,
		"subject_id":          request.Subject.ID,
		"subject_type":        request.Subject.Type,
		"resource_id":         request.Resource.ID,
		"resource_type":       request.Resource.Type,
		"action":              request.Action,
		"effect":              string(decision.Effect),
		"reason":              decision.Reason,
		"applicable_policies": decision.ApplicablePolicies,
		"evaluated_at":        decision.EvaluatedAt,
		"duration_ms":         decision.Duration.Milliseconds(),
	}

	p.logger.WithContext(ctx).Info("Access decision audit", logging.Any("audit", auditEntry))
}

func sortPoliciesByPriority(policies []*Policy) {
	// 简单的冒泡排序，按优先级降序
	for i := 0; i < len(policies); i++ {
		for j := i + 1; j < len(policies); j++ {
			if policies[i].Priority < policies[j].Priority {
				policies[i], policies[j] = policies[j], policies[i]
			}
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

// InMemoryPolicyRepository 内存策略仓储实现
type InMemoryPolicyRepository struct {
	policies sync.Map
	mu       sync.RWMutex
}

// NewInMemoryPolicyRepository 创建内存策略仓储
func NewInMemoryPolicyRepository() PolicyRepository {
	return &InMemoryPolicyRepository{}
}

func (r *InMemoryPolicyRepository) Create(ctx context.Context, policy *Policy) error {
	if _, exists := r.policies.Load(policy.ID); exists {
		return errors.ConflictError("policy already exists")
	}

	r.policies.Store(policy.ID, policy)
	return nil
}

func (r *InMemoryPolicyRepository) GetByID(ctx context.Context, id string) (*Policy, error) {
	if policy, ok := r.policies.Load(id); ok {
		return policy.(*Policy), nil
	}
	return nil, errors.NewInternalError(errors.CodeNotFound, "policy not found")
}

func (r *InMemoryPolicyRepository) Update(ctx context.Context, policy *Policy) error {
	if _, exists := r.policies.Load(policy.ID); !exists {
		return errors.NewInternalError(errors.CodeNotFound, "policy not found")
	}

	r.policies.Store(policy.ID, policy)
	return nil
}

func (r *InMemoryPolicyRepository) Delete(ctx context.Context, id string) error {
	if _, exists := r.policies.Load(id); !exists {
		return errors.NewInternalError(errors.CodeNotFound, "policy not found")
	}

	r.policies.Delete(id)
	return nil
}

func (r *InMemoryPolicyRepository) List(ctx context.Context, filter *PolicyFilter) ([]*Policy, error) {
	result := []*Policy{}

	r.policies.Range(func(key, value interface{}) bool {
		policy := value.(*Policy)

		// 应用过滤器
		if filter != nil {
			if filter.Type != "" && policy.Type != filter.Type {
				return true
			}
			if filter.Enabled != nil && policy.Enabled != *filter.Enabled {
				return true
			}
		}

		result = append(result, policy)
		return true
	})

	// 应用分页
	if filter != nil && filter.Limit > 0 {
		start := filter.Offset
		end := start + filter.Limit

		if start >= len(result) {
			return []*Policy{}, nil
		}
		if end > len(result) {
			end = len(result)
		}

		result = result[start:end]
	}

	return result, nil
}

func (r *InMemoryPolicyRepository) GetApplicablePolicies(ctx context.Context, request *AccessRequest) ([]*Policy, error) {
	applicable := []*Policy{}

	r.policies.Range(func(key, value interface{}) bool {
		policy := value.(*Policy)

		if !policy.Enabled {
			return true
		}

		// 简化的目标匹配检查
		if policy.Target != nil {
			// 检查操作匹配
			if len(policy.Target.Actions) > 0 {
				actionMatched := false
				for _, action := range policy.Target.Actions {
					if action == request.Action || action == "*" {
						actionMatched = true
						break
					}
				}
				if !actionMatched {
					return true
				}
			}
		}

		applicable = append(applicable, policy)
		return true
	})

	return applicable, nil
}

// PolicyBuilder 策略构建器
type PolicyBuilder struct {
	policy *Policy
}

// NewPolicyBuilder 创建策略构建器
func NewPolicyBuilder(id, name string) *PolicyBuilder {
	return &PolicyBuilder{
		policy: &Policy{
			ID:        id,
			Name:      name,
			Enabled:   true,
			Priority:  0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Metadata:  make(map[string]interface{}),
		},
	}
}

func (b *PolicyBuilder) WithDescription(description string) *PolicyBuilder {
	b.policy.Description = description
	return b
}

func (b *PolicyBuilder) WithType(policyType PolicyType) *PolicyBuilder {
	b.policy.Type = policyType
	return b
}

func (b *PolicyBuilder) WithPriority(priority int) *PolicyBuilder {
	b.policy.Priority = priority
	return b
}

func (b *PolicyBuilder) WithEffect(effect Effect) *PolicyBuilder {
	b.policy.Effect = effect
	return b
}

func (b *PolicyBuilder) WithTarget(target *Target) *PolicyBuilder {
	b.policy.Target = target
	return b
}

func (b *PolicyBuilder) WithCondition(condition *Condition) *PolicyBuilder {
	b.policy.Condition = condition
	return b
}

func (b *PolicyBuilder) WithRules(rules []*Rule) *PolicyBuilder {
	b.policy.Rules = rules
	return b
}

func (b *PolicyBuilder) WithObligations(obligations []*Obligation) *PolicyBuilder {
	b.policy.Obligations = obligations
	return b
}

func (b *PolicyBuilder) WithAdvice(advice []*Advice) *PolicyBuilder {
	b.policy.Advice = advice
	return b
}

func (b *PolicyBuilder) Enabled(enabled bool) *PolicyBuilder {
	b.policy.Enabled = enabled
	return b
}

func (b *PolicyBuilder) Build() *Policy {
	return b.policy
}

// RBAC 辅助函数

// CreateRBACPolicy 创建基于角色的访问控制策略
func CreateRBACPolicy(id, name string, roles []string, resources []string, actions []string, effect Effect) *Policy {
	subjectMatchers := make([]*Matcher, len(roles))
	for i, role := range roles {
		subjectMatchers[i] = &Matcher{
			Type:     "exact",
			Field:    "role",
			Value:    role,
			Operator: "in",
		}
	}

	resourceMatchers := make([]*Matcher, len(resources))
	for i, resource := range resources {
		resourceMatchers[i] = &Matcher{
			Type:     "exact",
			Field:    "type",
			Value:    resource,
			Operator: "eq",
		}
	}

	return NewPolicyBuilder(id, name).
		WithType(PolicyTypeRBAC).
		WithEffect(effect).
		WithTarget(&Target{
			Subjects:  subjectMatchers,
			Resources: resourceMatchers,
			Actions:   actions,
		}).
		Build()
}

// ABAC 辅助函数

// CreateABACPolicy 创建基于属性的访问控制策略
func CreateABACPolicy(id, name string, attributes []*Attribute, effect Effect) *Policy {
	return NewPolicyBuilder(id, name).
		WithType(PolicyTypeABAC).
		WithEffect(effect).
		WithCondition(&Condition{
			Type:       "and",
			Attributes: attributes,
		}).
		Build()
}

// CreateTimeBasedPolicy 创建基于时间的策略
func CreateTimeBasedPolicy(id, name string, startHour, endHour int, effect Effect) *Policy {
	return NewPolicyBuilder(id, name).
		WithType(PolicyTypeABAC).
		WithEffect(effect).
		WithCondition(&Condition{
			Type: "and",
			Attributes: []*Attribute{
				{
					Category: "environment",
					Name:     "hour",
					Operator: "gte",
					Value:    startHour,
				},
				{
					Category: "environment",
					Name:     "hour",
					Operator: "lte",
					Value:    endHour,
				},
			},
		}).
		Build()
}

// CreateOwnershipPolicy 创建所有权策略
func CreateOwnershipPolicy(id, name string) *Policy {
	return NewPolicyBuilder(id, name).
		WithType(PolicyTypeABAC).
		WithEffect(EffectPermit).
		WithCondition(&Condition{
			Type: "and",
			Attributes: []*Attribute{
				{
					Category: "subject",
					Name:     "id",
					Operator: "eq",
					Value:    "${resource.owner}",
				},
			},
		}).
		Build()
}

// CreateRateLimitPolicy 创建速率限制策略
func CreateRateLimitPolicy(id, name string, maxRequests int, window time.Duration) *Policy {
	return NewPolicyBuilder(id, name).
		WithType(PolicyTypeABAC).
		WithEffect(EffectDeny).
		WithCondition(&Condition{
			Type: "and",
			Attributes: []*Attribute{
				{
					Category: "environment",
					Name:     "request_count",
					Operator: "gt",
					Value:    maxRequests,
				},
			},
		}).
		WithObligations([]*Obligation{
			{
				ID:          "rate_limit_exceeded",
				Type:        "notification",
				Description: fmt.Sprintf("Rate limit exceeded: %d requests per %s", maxRequests, window),
				Parameters: map[string]interface{}{
					"max_requests": maxRequests,
					"window":       window.String(),
				},
			},
		}).
		Build()
}

// Utility functions for common access patterns

// CanRead 检查是否有读取权限
func CanRead(ctx context.Context, pdp PolicyDecisionPoint, subjectID, resourceID string) (bool, error) {
	request := &AccessRequest{
		RequestID: fmt.Sprintf("read_%s_%d", resourceID, time.Now().UnixNano()),
		Subject: &Subject{
			ID:         subjectID,
			Type:       "user",
			Attributes: make(map[string]interface{}),
		},
		Resource: &Resource{
			ID:         resourceID,
			Attributes: make(map[string]interface{}),
		},
		Action:      "read",
		Environment: make(map[string]interface{}),
		Timestamp:   time.Now(),
	}

	decision, err := pdp.Evaluate(ctx, request)
	if err != nil {
		return false, err
	}

	return decision.Effect == EffectPermit, nil
}

// CanWrite 检查是否有写入权限
func CanWrite(ctx context.Context, pdp PolicyDecisionPoint, subjectID, resourceID string) (bool, error) {
	request := &AccessRequest{
		RequestID: fmt.Sprintf("write_%s_%d", resourceID, time.Now().UnixNano()),
		Subject: &Subject{
			ID:         subjectID,
			Type:       "user",
			Attributes: make(map[string]interface{}),
		},
		Resource: &Resource{
			ID:         resourceID,
			Attributes: make(map[string]interface{}),
		},
		Action:      "write",
		Environment: make(map[string]interface{}),
		Timestamp:   time.Now(),
	}

	decision, err := pdp.Evaluate(ctx, request)
	if err != nil {
		return false, err
	}

	return decision.Effect == EffectPermit, nil
}

// CanExecute 检查是否有执行权限
func CanExecute(ctx context.Context, pdp PolicyDecisionPoint, subjectID, resourceID string) (bool, error) {
	request := &AccessRequest{
		RequestID: fmt.Sprintf("execute_%s_%d", resourceID, time.Now().UnixNano()),
		Subject: &Subject{
			ID:         subjectID,
			Type:       "user",
			Attributes: make(map[string]interface{}),
		},
		Resource: &Resource{
			ID:         resourceID,
			Attributes: make(map[string]interface{}),
		},
		Action:      "execute",
		Environment: make(map[string]interface{}),
		Timestamp:   time.Now(),
	}

	decision, err := pdp.Evaluate(ctx, request)
	if err != nil {
		return false, err
	}

	return decision.Effect == EffectPermit, nil
}

//Personal.AI order the ending
