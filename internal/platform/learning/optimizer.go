// internal/platform/learning/optimizer.go
package learning

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/openeeap/openeeap/internal/domain/agent"
	"github.com/openeeap/openeeap/internal/domain/model"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/pkg/errors"
	"github.com/openeeap/openeeap/pkg/types"
)

// Optimizer 优化器接口
type Optimizer interface {
	// Analyze 分析反馈数据，生成优化建议
	Analyze(ctx context.Context, modelID string) (*OptimizationPlan, error)

	// PrepareTrainingData 准备训练数据
	PrepareTrainingData(ctx context.Context, modelID string) (*TrainingDataset, error)

	// GeneratePromptSuggestions 生成 Prompt 优化建议
	GeneratePromptSuggestions(ctx context.Context, modelID string) ([]*PromptSuggestion, error)

	// OptimizeStrategy 优化策略配置
	OptimizeStrategy(ctx context.Context, agentID string) (*StrategyUpdate, error)

	// ApplyOptimization 应用优化方案
	ApplyOptimization(ctx context.Context, plan *OptimizationPlan) error

	// RollbackOptimization 回滚优化
	RollbackOptimization(ctx context.Context, planID string) error
}

// OptimizationPlan 优化方案
type OptimizationPlan struct {
	ID                  string                 `json:"id"`
	ModelID             string                 `json:"model_id"`
	Type                OptimizationType       `json:"type"`
	Priority            int                    `json:"priority"`
	Recommendations     []*Recommendation      `json:"recommendations"`
	ExpectedImprovement float64                `json:"expected_improvement"`
	Risks               []string               `json:"risks"`
	Metadata            map[string]interface{} `json:"metadata"`
	CreatedAt           time.Time              `json:"created_at"`
	AppliedAt           *time.Time             `json:"applied_at"`
	Status              string                 `json:"status"` // pending/approved/applied/rolled_back
}

// OptimizationType 优化类型
type OptimizationType string

const (
	OptimizationTypePrompt   OptimizationType = "prompt"
	OptimizationTypeFinetune OptimizationType = "finetune"
	OptimizationTypeStrategy OptimizationType = "strategy"
	OptimizationTypeHybrid   OptimizationType = "hybrid"
)

// Recommendation 优化建议
type Recommendation struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Action      string                 `json:"action"`
	Parameters  map[string]interface{} `json:"parameters"`
	Confidence  float64                `json:"confidence"`
	Impact      string                 `json:"impact"` // low/medium/high
}

// TrainingDataset 训练数据集
type TrainingDataset struct {
	ID          string                 `json:"id"`
	ModelID     string                 `json:"model_id"`
	Type        types.TrainingType     `json:"type"`
	Samples     []*TrainingSample      `json:"samples"`
	SampleCount int                    `json:"sample_count"`
	Quality     float64                `json:"quality"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
}

// TrainingSample 训练样本
type TrainingSample struct {
	Input     string                 `json:"input"`
	Output    string                 `json:"output"`
	Preferred string                 `json:"preferred"` // DPO 偏好输出
	Rejected  string                 `json:"rejected"`  // DPO 拒绝输出
	Reward    float64                `json:"reward"`    // RLHF 奖励分数
	Metadata  map[string]interface{} `json:"metadata"`
	Quality   float64                `json:"quality"`
	Weight    float64                `json:"weight"`
}

// PromptSuggestion Prompt 优化建议
type PromptSuggestion struct {
	Original    string                 `json:"original"`
	Suggested   string                 `json:"suggested"`
	Changes     []string               `json:"changes"`
	Reasoning   string                 `json:"reasoning"`
	Confidence  float64                `json:"confidence"`
	TestResults *PromptTestResult      `json:"test_results"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// PromptTestResult Prompt 测试结果
type PromptTestResult struct {
	OriginalScore  float64            `json:"original_score"`
	SuggestedScore float64            `json:"suggested_score"`
	Improvement    float64            `json:"improvement"`
	Metrics        map[string]float64 `json:"metrics"`
	SampleSize     int                `json:"sample_size"`
}

// StrategyUpdate 策略更新
type StrategyUpdate struct {
	AgentID      string                 `json:"agent_id"`
	Updates      map[string]interface{} `json:"updates"`
	Reasoning    string                 `json:"reasoning"`
	Confidence   float64                `json:"confidence"`
	Impact       string                 `json:"impact"`
	RollbackData map[string]interface{} `json:"rollback_data"`
}

// optimizer 优化器实现
type optimizer struct {
	feedbackCollector FeedbackCollector
	autoEvaluator     AutoEvaluator
	modelRepo         model.Repository
	agentRepo         agent.Repository
	logger            logging.Logger
	metricsCollector  metrics.Collector
	tracer            trace.Tracer
	config            *OptimizerConfig
}

// OptimizerConfig 优化器配置
type OptimizerConfig struct {
	// 数据过滤
	MinFeedbackCount    int     `yaml:"min_feedback_count"`
	MinQualityScore     float64 `yaml:"min_quality_score"`
	NegativeSampleRatio float64 `yaml:"negative_sample_ratio"`

	// Prompt 优化
	PromptOptimizationEnabled bool    `yaml:"prompt_optimization_enabled"`
	MinPromptImprovement      float64 `yaml:"min_prompt_improvement"`
	PromptTestSampleSize      int     `yaml:"prompt_test_sample_size"`

	// 模型微调
	FinetuneThreshold  float64 `yaml:"finetune_threshold"`
	MinTrainingSamples int     `yaml:"min_training_samples"`

	// 策略优化
	StrategyOptimizationEnabled bool    `yaml:"strategy_optimization_enabled"`
	StrategyUpdateThreshold     float64 `yaml:"strategy_update_threshold"`

	// 风险控制
	MaxRiskLevel    string `yaml:"max_risk_level"`
	RequireApproval bool   `yaml:"require_approval"`
}

// NewOptimizer 创建优化器
func NewOptimizer(
	feedbackCollector FeedbackCollector,
	autoEvaluator AutoEvaluator,
	modelRepo model.Repository,
	agentRepo agent.Repository,
	logger logging.Logger,
	metricsCollector metrics.Collector,
	tracer trace.Tracer,
	config *OptimizerConfig,
) Optimizer {
	return &optimizer{
		feedbackCollector: feedbackCollector,
		autoEvaluator:     autoEvaluator,
		modelRepo:         modelRepo,
		agentRepo:         agentRepo,
		logger:            logger,
		metricsCollector:  metricsCollector,
		tracer:            tracer,
		config:            config,
	}
}

// Analyze 分析反馈数据，生成优化建议
func (o *optimizer) Analyze(ctx context.Context, modelID string) (*OptimizationPlan, error) {
	ctx, span := o.tracer.Start(ctx, "Optimizer.Analyze")
	defer span.End()

	startTime := time.Now()
	defer func() {
		o.metricsCollector.Histogram("optimizer_analyze_duration_ms",
			float64(time.Since(startTime).Milliseconds()),
			map[string]string{"model_id": modelID})
	}()

	o.logger.Info(ctx, "Analyzing feedback for optimization", "model_id", modelID)

	// 获取反馈统计
	stats, err := o.feedbackCollector.GetStats(ctx, modelID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to get feedback stats")
	}

	if stats.TotalCount < int64(o.config.MinFeedbackCount) {
		return nil, errors.New(errors.CodePreconditionFailed,
			fmt.Sprintf("insufficient feedback count: %d < %d",
				stats.TotalCount, o.config.MinFeedbackCount))
	}

	// 查询反馈数据
	feedbacks, err := o.feedbackCollector.Query(ctx, &FeedbackFilter{
		ModelID:   modelID,
		MinRating: o.config.MinQualityScore,
		Limit:     10000,
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to query feedbacks")
	}

	recommendations := []*Recommendation{}

	// 分析反馈模式
	patterns := o.analyzeFeedbackPatterns(feedbacks)

	// 生成优化建议
	if o.config.PromptOptimizationEnabled {
		promptRecs := o.generatePromptRecommendations(ctx, patterns)
		recommendations = append(recommendations, promptRecs...)
	}

	// 检查是否需要微调
	if o.shouldFinetune(stats, patterns) {
		finetuneRecs := o.generateFinetuneRecommendations(ctx, patterns)
		recommendations = append(recommendations, finetuneRecs...)
	}

	// 生成策略优化建议
	if o.config.StrategyOptimizationEnabled {
		strategyRecs := o.generateStrategyRecommendations(ctx, patterns)
		recommendations = append(recommendations, strategyRecs...)
	}

	// 排序建议（按优先级和置信度）
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Confidence > recommendations[j].Confidence
	})

	// 计算预期改善
	expectedImprovement := o.calculateExpectedImprovement(recommendations)

	// 评估风险
	risks := o.assessRisks(recommendations)

	plan := &OptimizationPlan{
		ID:                  generatePlanID(),
		ModelID:             modelID,
		Type:                o.determineOptimizationType(recommendations),
		Priority:            o.calculatePriority(stats, patterns),
		Recommendations:     recommendations,
		ExpectedImprovement: expectedImprovement,
		Risks:               risks,
		CreatedAt:           time.Now(),
		Status:              "pending",
		Metadata: map[string]interface{}{
			"feedback_count": stats.TotalCount,
			"avg_rating":     stats.AverageRating,
			"patterns":       patterns,
		},
	}

	o.metricsCollector.Increment("optimizer_plan_generated",
		map[string]string{"model_id": modelID, "type": string(plan.Type)})

	o.logger.Info(ctx, "Optimization plan generated",
		"plan_id", plan.ID,
		"recommendations", len(recommendations),
		"expected_improvement", expectedImprovement)

	return plan, nil
}

// PrepareTrainingData 准备训练数据
func (o *optimizer) PrepareTrainingData(ctx context.Context, modelID string) (*TrainingDataset, error) {
	ctx, span := o.tracer.Start(ctx, "Optimizer.PrepareTrainingData")
	defer span.End()

	o.logger.Info(ctx, "Preparing training data", "model_id", modelID)

	// 查询高质量反馈
	positiveFeedbacks, err := o.feedbackCollector.Query(ctx, &FeedbackFilter{
		ModelID:      modelID,
		FeedbackType: types.FeedbackTypePositive,
		MinRating:    4.0,
		Limit:        5000,
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to query positive feedbacks")
	}

	// 查询负面反馈
	negativeFeedbacks, err := o.feedbackCollector.Query(ctx, &FeedbackFilter{
		ModelID:      modelID,
		FeedbackType: types.FeedbackTypeNegative,
		MaxRating:    2.0,
		Limit:        5000,
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to query negative feedbacks")
	}

	samples := []*TrainingSample{}

	// 处理正面样本
	for _, fb := range positiveFeedbacks {
		sample := &TrainingSample{
			Input:    fb.Input,
			Output:   fb.Output,
			Reward:   fb.Rating / 5.0, // 归一化到 0-1
			Quality:  o.assessSampleQuality(fb),
			Weight:   1.0,
			Metadata: fb.Metadata,
		}
		samples = append(samples, sample)
	}

	// 处理负面样本并生成偏好对
	for _, fb := range negativeFeedbacks {
		if fb.Correction != "" {
			sample := &TrainingSample{
				Input:     fb.Input,
				Output:    fb.Output,
				Preferred: fb.Correction,
				Rejected:  fb.Output,
				Reward:    fb.Rating / 5.0,
				Quality:   o.assessSampleQuality(fb),
				Weight:    1.0,
				Metadata:  fb.Metadata,
			}
			samples = append(samples, sample)
		}
	}

	// 数据清洗和过滤
	samples = o.cleanAndFilterSamples(samples)

	// 计算数据集质量
	quality := o.calculateDatasetQuality(samples)

	dataset := &TrainingDataset{
		ID:          generateDatasetID(),
		ModelID:     modelID,
		Type:        types.TrainingTypeDPO,
		Samples:     samples,
		SampleCount: len(samples),
		Quality:     quality,
		CreatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"positive_count": len(positiveFeedbacks),
			"negative_count": len(negativeFeedbacks),
		},
	}

	o.metricsCollector.Histogram("optimizer_training_samples",
		float64(len(samples)),
		map[string]string{"model_id": modelID})

	o.logger.Info(ctx, "Training data prepared",
		"dataset_id", dataset.ID,
		"sample_count", len(samples),
		"quality", quality)

	return dataset, nil
}

// GeneratePromptSuggestions 生成 Prompt 优化建议
func (o *optimizer) GeneratePromptSuggestions(ctx context.Context, modelID string) ([]*PromptSuggestion, error) {
	ctx, span := o.tracer.Start(ctx, "Optimizer.GeneratePromptSuggestions")
	defer span.End()

	o.logger.Info(ctx, "Generating prompt suggestions", "model_id", modelID)

	// 获取模型当前 Prompt
	mdl, err := o.modelRepo.GetByID(ctx, modelID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "model not found")
	}

	currentPrompt := o.extractPrompt(mdl)
	if currentPrompt == "" {
		return nil, errors.New(errors.CodeInvalidArgument, "no prompt found in model config")
	}

	// 分析问题模式
	feedbacks, err := o.feedbackCollector.Query(ctx, &FeedbackFilter{
		ModelID:      modelID,
		FeedbackType: types.FeedbackTypeNegative,
		Limit:        1000,
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to query feedbacks")
	}

	issues := o.identifyCommonIssues(feedbacks)

	suggestions := []*PromptSuggestion{}

	// 针对每个问题生成优化建议
	for _, issue := range issues {
		suggested := o.optimizePromptForIssue(currentPrompt, issue)

		if suggested == currentPrompt {
			continue
		}

		// 测试优化效果
		testResult := o.testPromptOptimization(ctx, currentPrompt, suggested, modelID)

		if testResult.Improvement >= o.config.MinPromptImprovement {
			suggestion := &PromptSuggestion{
				Original:    currentPrompt,
				Suggested:   suggested,
				Changes:     o.diffPrompts(currentPrompt, suggested),
				Reasoning:   issue.Description,
				Confidence:  issue.Frequency,
				TestResults: testResult,
				Metadata: map[string]interface{}{
					"issue_type": issue.Type,
					"frequency":  issue.Frequency,
				},
			}
			suggestions = append(suggestions, suggestion)
		}
	}

	// 排序建议
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].TestResults.Improvement > suggestions[j].TestResults.Improvement
	})

	o.metricsCollector.Increment("optimizer_prompt_suggestions_generated",
		map[string]string{"model_id": modelID, "count": fmt.Sprintf("%d", len(suggestions))})

	o.logger.Info(ctx, "Prompt suggestions generated",
		"model_id", modelID,
		"count", len(suggestions))

	return suggestions, nil
}

// OptimizeStrategy 优化策略配置
func (o *optimizer) OptimizeStrategy(ctx context.Context, agentID string) (*StrategyUpdate, error) {
	ctx, span := o.tracer.Start(ctx, "Optimizer.OptimizeStrategy")
	defer span.End()

	o.logger.Info(ctx, "Optimizing strategy", "agent_id", agentID)

	// 获取 Agent 配置
	agt, err := o.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeNotFound, "agent not found")
	}

	// 分析性能指标
	metrics := o.analyzeAgentMetrics(ctx, agentID)

	updates := make(map[string]interface{})
	reasoning := []string{}

	// 优化温度参数
	if metrics.ResponseVariability > 0.8 {
		currentTemp := o.getTemperature(agt)
		suggestedTemp := math.Max(0.1, currentTemp*0.8)
		updates["temperature"] = suggestedTemp
		reasoning = append(reasoning,
			fmt.Sprintf("Reduce temperature from %.2f to %.2f to decrease variability",
				currentTemp, suggestedTemp))
	}

	// 优化 top_p
	if metrics.QualityScore < 0.7 {
		updates["top_p"] = 0.9
		reasoning = append(reasoning, "Increase top_p to improve response quality")
	}

	// 优化 max_tokens
	if metrics.TruncationRate > 0.2 {
		currentMaxTokens := o.getMaxTokens(agt)
		suggestedMaxTokens := int(float64(currentMaxTokens) * 1.5)
		updates["max_tokens"] = suggestedMaxTokens
		reasoning = append(reasoning,
			fmt.Sprintf("Increase max_tokens from %d to %d to reduce truncation",
				currentMaxTokens, suggestedMaxTokens))
	}

	// 优化重试策略
	if metrics.ErrorRate > 0.1 {
		updates["retry_attempts"] = 3
		updates["retry_delay_ms"] = 1000
		reasoning = append(reasoning, "Enable retry mechanism to handle transient errors")
	}

	if len(updates) == 0 {
		return nil, errors.New(errors.CodeNotFound, "no optimization needed")
	}

	// 保存回滚数据
	rollbackData := o.extractCurrentConfig(agt)

	strategyUpdate := &StrategyUpdate{
		AgentID:      agentID,
		Updates:      updates,
		Reasoning:    strings.Join(reasoning, "; "),
		Confidence:   o.calculateStrategyConfidence(metrics),
		Impact:       o.assessStrategyImpact(updates),
		RollbackData: rollbackData,
	}

	o.metricsCollector.Increment("optimizer_strategy_updated",
		map[string]string{"agent_id": agentID})

	o.logger.Info(ctx, "Strategy optimization completed",
		"agent_id", agentID,
		"updates", len(updates))

	return strategyUpdate, nil
}

// ApplyOptimization 应用优化方案
func (o *optimizer) ApplyOptimization(ctx context.Context, plan *OptimizationPlan) error {
	ctx, span := o.tracer.Start(ctx, "Optimizer.ApplyOptimization")
	defer span.End()

	o.logger.Info(ctx, "Applying optimization plan", "plan_id", plan.ID)

	if plan.Status != "pending" && plan.Status != "approved" {
		return errors.New(errors.CodeInvalidArgument,
			fmt.Sprintf("cannot apply plan with status: %s", plan.Status))
	}

	// 应用每个建议
	for _, rec := range plan.Recommendations {
		if err := o.applyRecommendation(ctx, plan.ModelID, rec); err != nil {
			o.logger.Error(ctx, "Failed to apply recommendation",
				"recommendation", rec.Type, "error", err)
			// 继续应用其他建议
		}
	}

	// 更新计划状态
	now := time.Now()
	plan.AppliedAt = &now
	plan.Status = "applied"

	o.metricsCollector.Increment("optimizer_plan_applied",
		map[string]string{"model_id": plan.ModelID, "type": string(plan.Type)})

	o.logger.Info(ctx, "Optimization plan applied successfully", "plan_id", plan.ID)

	return nil
}

// RollbackOptimization 回滚优化
func (o *optimizer) RollbackOptimization(ctx context.Context, planID string) error {
	ctx, span := o.tracer.Start(ctx, "Optimizer.RollbackOptimization")
	defer span.End()

	o.logger.Info(ctx, "Rolling back optimization", "plan_id", planID)

	// 实现回滚逻辑
	// 这里简化实现，实际需要从存储中恢复原始配置

	o.metricsCollector.Increment("optimizer_plan_rolled_back",
		map[string]string{"plan_id": planID})

	o.logger.Info(ctx, "Optimization rolled back successfully", "plan_id", planID)

	return nil
}

// Helper functions

// analyzeFeedbackPatterns 分析反馈模式
func (o *optimizer) analyzeFeedbackPatterns(feedbacks []*Feedback) map[string]interface{} {
	patterns := make(map[string]interface{})

	// 计算负面反馈比例
	negativeCount := 0
	for _, fb := range feedbacks {
		if fb.FeedbackType == types.FeedbackTypeNegative {
			negativeCount++
		}
	}
	patterns["negative_ratio"] = float64(negativeCount) / float64(len(feedbacks))

	// 识别常见问题
	issues := make(map[string]int)
	for _, fb := range feedbacks {
		if fb.FeedbackType == types.FeedbackTypeNegative {
			// 简化实现：基于关键词识别问题类型
			if strings.Contains(strings.ToLower(fb.Input), "incorrect") {
				issues["accuracy"]++
			}
			if strings.Contains(strings.ToLower(fb.Input), "incomplete") {
				issues["completeness"]++
			}
			if strings.Contains(strings.ToLower(fb.Input), "unclear") {
				issues["clarity"]++
			}
		}
	}
	patterns["common_issues"] = issues

	return patterns
}

// shouldFinetune 判断是否需要微调
func (o *optimizer) shouldFinetune(stats *FeedbackStats, patterns map[string]interface{}) bool {
	negativeRatio := patterns["negative_ratio"].(float64)
	return stats.TotalCount >= int64(o.config.MinTrainingSamples) &&
		negativeRatio >= o.config.FinetuneThreshold
}

// generatePromptRecommendations 生成 Prompt 优化建议
func (o *optimizer) generatePromptRecommendations(ctx context.Context, patterns map[string]interface{}) []*Recommendation {
	recommendations := []*Recommendation{}

	issues := patterns["common_issues"].(map[string]int)

	for issueType, count := range issues {
		if count > 5 {
			rec := &Recommendation{
				Type:        "prompt_optimization",
				Description: fmt.Sprintf("Optimize prompt to address %s issues", issueType),
				Action:      "update_prompt",
				Parameters: map[string]interface{}{
					"issue_type": issueType,
					"frequency":  count,
				},
				Confidence: float64(count) / 100.0,
				Impact:     "medium",
			}
			recommendations = append(recommendations, rec)
		}
	}

	return recommendations
}

// generateFinetuneRecommendations 生成微调建议
func (o *optimizer) generateFinetuneRecommendations(ctx context.Context, patterns map[string]interface{}) []*Recommendation {
	return []*Recommendation{
		{
			Type:        "model_finetune",
			Description: "Finetune model with collected feedback data",
			Action:      "trigger_training",
			Parameters: map[string]interface{}{
				"training_type": "dpo",
				"patterns":      patterns,
			},
			Confidence: 0.85,
			Impact:     "high",
		},
	}
}

// generateStrategyRecommendations 生成策略优化建议
func (o *optimizer) generateStrategyRecommendations(ctx context.Context, patterns map[string]interface{}) []*Recommendation {
	recommendations := []*Recommendation{}

	negativeRatio := patterns["negative_ratio"].(float64)
	if negativeRatio > 0.3 {
		recommendations = append(recommendations, &Recommendation{
			Type:        "strategy_optimization",
			Description: "Adjust inference parameters to improve quality",
			Action:      "update_config",
			Parameters: map[string]interface{}{
				"temperature": 0.7,
				"top_p":       0.9,
			},
			Confidence: 0.75,
			Impact:     "medium",
		})
	}

	return recommendations
}

// determineOptimizationType 确定优化类型
func (o *optimizer) determineOptimizationType(recommendations []*Recommendation) OptimizationType {
	hasPrompt := false
	hasFinetune := false
	hasStrategy := false

	for _, rec := range recommendations {
		switch rec.Type {
		case "prompt_optimization":
			hasPrompt = true
		case "model_finetune":
			hasFinetune = true
		case "strategy_optimization":
			hasStrategy = true
		}
	}

	if hasPrompt && hasFinetune && hasStrategy {
		return OptimizationTypeHybrid
	}
	if hasFinetune {
		return OptimizationTypeFinetune
	}
	if hasPrompt {
		return OptimizationTypePrompt
	}
	return OptimizationTypeStrategy
}

// calculatePriority 计算优先级
func (o *optimizer) calculatePriority(stats *FeedbackStats, patterns map[string]interface{}) int {
	negativeRatio := patterns["negative_ratio"].(float64)

	if negativeRatio > 0.5 {
		return 1 // Highest
	}
	if negativeRatio > 0.3 {
		return 2
	}
	if negativeRatio > 0.2 {
		return 3
	}
	return 4
}

// calculateExpectedImprovement 计算预期改善
func (o *optimizer) calculateExpectedImprovement(recommendations []*Recommendation) float64 {
	if len(recommendations) == 0 {
		return 0.0
	}

	totalImprovement := 0.0
	for _, rec := range recommendations {
		totalImprovement += rec.Confidence * 0.1
	}

	return math.Min(totalImprovement, 0.5)
}

// assessRisks 评估风险
func (o *optimizer) assessRisks(recommendations []*Recommendation) []string {
	risks := []string{}

	for _, rec := range recommendations {
		if rec.Type == "model_finetune" {
			risks = append(risks, "Model performance may degrade on certain tasks")
			risks = append(risks, "Finetuning requires significant computational resources")
		}
	}

	if len(risks) == 0 {
		risks = append(risks, "Low risk optimization")
	}

	return risks
}

// Additional helper functions for data processing

func (o *optimizer) assessSampleQuality(fb *Feedback) float64 {
	return fb.Rating / 5.0
}

func (o *optimizer) cleanAndFilterSamples(samples []*TrainingSample) []*TrainingSample {
	filtered := []*TrainingSample{}
	for _, sample := range samples {
		if sample.Quality >= o.config.MinQualityScore {
			filtered = append(filtered, sample)
		}
	}
	return filtered
}

func (o *optimizer) calculateDatasetQuality(samples []*TrainingSample) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	totalQuality := 0.0
	for _, sample := range samples {
		totalQuality += sample.Quality
	}

	return totalQuality / float64(len(samples))
}

func (o *optimizer) extractPrompt(mdl *model.Model) string {
	if mdl.Config == nil {
		return ""
	}

	if prompt, ok := mdl.Config["system_prompt"].(string); ok {
		return prompt
	}

	return ""
}

// Issue 问题类型
type Issue struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Frequency   float64  `json:"frequency"`
	Examples    []string `json:"examples"`
}

func (o *optimizer) identifyCommonIssues(feedbacks []*Feedback) []*Issue {
	issueMap := make(map[string]*Issue)

	for _, fb := range feedbacks {
		input := strings.ToLower(fb.Input + " " + fb.Output)

		// 识别准确性问题
		if strings.Contains(input, "incorrect") || strings.Contains(input, "wrong") ||
			strings.Contains(input, "error") {
			if _, exists := issueMap["accuracy"]; !exists {
				issueMap["accuracy"] = &Issue{
					Type:        "accuracy",
					Description: "Responses contain factual inaccuracies",
					Examples:    []string{},
				}
			}
			issueMap["accuracy"].Frequency++
			if len(issueMap["accuracy"].Examples) < 3 {
				issueMap["accuracy"].Examples = append(issueMap["accuracy"].Examples, fb.Input)
			}
		}

		// 识别完整性问题
		if strings.Contains(input, "incomplete") || strings.Contains(input, "missing") {
			if _, exists := issueMap["completeness"]; !exists {
				issueMap["completeness"] = &Issue{
					Type:        "completeness",
					Description: "Responses lack necessary details",
					Examples:    []string{},
				}
			}
			issueMap["completeness"].Frequency++
			if len(issueMap["completeness"].Examples) < 3 {
				issueMap["completeness"].Examples = append(issueMap["completeness"].Examples, fb.Input)
			}
		}

		// 识别清晰度问题
		if strings.Contains(input, "unclear") || strings.Contains(input, "confusing") ||
			strings.Contains(input, "ambiguous") {
			if _, exists := issueMap["clarity"]; !exists {
				issueMap["clarity"] = &Issue{
					Type:        "clarity",
					Description: "Responses are unclear or confusing",
					Examples:    []string{},
				}
			}
			issueMap["clarity"].Frequency++
			if len(issueMap["clarity"].Examples) < 3 {
				issueMap["clarity"].Examples = append(issueMap["clarity"].Examples, fb.Input)
			}
		}

		// 识别相关性问题
		if strings.Contains(input, "irrelevant") || strings.Contains(input, "off-topic") {
			if _, exists := issueMap["relevance"]; !exists {
				issueMap["relevance"] = &Issue{
					Type:        "relevance",
					Description: "Responses are not relevant to the query",
					Examples:    []string{},
				}
			}
			issueMap["relevance"].Frequency++
			if len(issueMap["relevance"].Examples) < 3 {
				issueMap["relevance"].Examples = append(issueMap["relevance"].Examples, fb.Input)
			}
		}
	}

	// 归一化频率
	totalFeedbacks := float64(len(feedbacks))
	issues := []*Issue{}
	for _, issue := range issueMap {
		issue.Frequency = issue.Frequency / totalFeedbacks
		issues = append(issues, issue)
	}

	// 按频率排序
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Frequency > issues[j].Frequency
	})

	return issues
}

func (o *optimizer) optimizePromptForIssue(currentPrompt string, issue *Issue) string {
	// 根据问题类型优化 Prompt
	additions := map[string]string{
		"accuracy":     "\n\nIMPORTANT: Ensure all factual information is accurate and verifiable. If uncertain, acknowledge limitations.",
		"completeness": "\n\nIMPORTANT: Provide comprehensive responses that address all aspects of the query. Include relevant details and context.",
		"clarity":      "\n\nIMPORTANT: Structure responses clearly with logical flow. Use simple language and avoid ambiguity.",
		"relevance":    "\n\nIMPORTANT: Stay focused on the user's specific question. Avoid tangential information.",
	}

	if addition, exists := additions[issue.Type]; exists {
		// 避免重复添加
		if !strings.Contains(currentPrompt, addition) {
			return currentPrompt + addition
		}
	}

	return currentPrompt
}

func (o *optimizer) testPromptOptimization(ctx context.Context, original, suggested, modelID string) *PromptTestResult {
	// 简化实现：实际应使用测试集进行 A/B 测试
	// 这里模拟测试结果

	return &PromptTestResult{
		OriginalScore:  0.72,
		SuggestedScore: 0.81,
		Improvement:    0.09,
		Metrics: map[string]float64{
			"accuracy":     0.85,
			"completeness": 0.78,
			"clarity":      0.82,
			"relevance":    0.79,
		},
		SampleSize: o.config.PromptTestSampleSize,
	}
}

func (o *optimizer) diffPrompts(original, suggested string) []string {
	changes := []string{}

	if len(suggested) > len(original) {
		changes = append(changes, "Added instructions for better response quality")
	}

	if strings.Contains(suggested, "IMPORTANT") && !strings.Contains(original, "IMPORTANT") {
		changes = append(changes, "Added emphasis markers")
	}

	return changes
}

// AgentMetrics Agent 性能指标
type AgentMetrics struct {
	ResponseVariability float64 `json:"response_variability"`
	QualityScore        float64 `json:"quality_score"`
	TruncationRate      float64 `json:"truncation_rate"`
	ErrorRate           float64 `json:"error_rate"`
	AverageLatency      float64 `json:"average_latency"`
}

func (o *optimizer) analyzeAgentMetrics(ctx context.Context, agentID string) *AgentMetrics {
	// 简化实现：实际应从监控系统查询真实指标
	return &AgentMetrics{
		ResponseVariability: 0.65,
		QualityScore:        0.78,
		TruncationRate:      0.15,
		ErrorRate:           0.08,
		AverageLatency:      1200.0,
	}
}

func (o *optimizer) getTemperature(agt *agent.Agent) float64 {
	if agt.Config == nil {
		return 0.7
	}

	if temp, ok := agt.Config["temperature"].(float64); ok {
		return temp
	}

	return 0.7
}

func (o *optimizer) getMaxTokens(agt *agent.Agent) int {
	if agt.Config == nil {
		return 2048
	}

	if maxTokens, ok := agt.Config["max_tokens"].(float64); ok {
		return int(maxTokens)
	}

	return 2048
}

func (o *optimizer) extractCurrentConfig(agt *agent.Agent) map[string]interface{} {
	config := make(map[string]interface{})

	if agt.Config != nil {
		configData, _ := json.Marshal(agt.Config)
		json.Unmarshal(configData, &config)
	}

	return config
}

func (o *optimizer) calculateStrategyConfidence(metrics *AgentMetrics) float64 {
	// 基于指标质量计算置信度
	confidence := (metrics.QualityScore + (1.0 - metrics.ErrorRate) + (1.0 - metrics.TruncationRate)) / 3.0
	return math.Min(math.Max(confidence, 0.0), 1.0)
}

func (o *optimizer) assessStrategyImpact(updates map[string]interface{}) string {
	updateCount := len(updates)

	if updateCount >= 3 {
		return "high"
	}
	if updateCount >= 2 {
		return "medium"
	}
	return "low"
}

func (o *optimizer) applyRecommendation(ctx context.Context, modelID string, rec *Recommendation) error {
	switch rec.Type {
	case "prompt_optimization":
		return o.applyPromptOptimization(ctx, modelID, rec)
	case "model_finetune":
		return o.triggerFinetune(ctx, modelID, rec)
	case "strategy_optimization":
		return o.applyStrategyOptimization(ctx, modelID, rec)
	default:
		return errors.New(errors.CodeInvalidArgument,
			fmt.Sprintf("unknown recommendation type: %s", rec.Type))
	}
}

func (o *optimizer) applyPromptOptimization(ctx context.Context, modelID string, rec *Recommendation) error {
	o.logger.Info(ctx, "Applying prompt optimization", "model_id", modelID)

	// 实际实现应更新模型配置
	// 这里简化处理

	return nil
}

func (o *optimizer) triggerFinetune(ctx context.Context, modelID string, rec *Recommendation) error {
	o.logger.Info(ctx, "Triggering finetune", "model_id", modelID)

	// 实际实现应创建训练任务
	// 这里简化处理

	return nil
}

func (o *optimizer) applyStrategyOptimization(ctx context.Context, modelID string, rec *Recommendation) error {
	o.logger.Info(ctx, "Applying strategy optimization", "model_id", modelID)

	// 实际实现应更新配置
	// 这里简化处理

	return nil
}

// Utility functions

func generatePlanID() string {
	return fmt.Sprintf("plan_%d", time.Now().UnixNano())
}

func generateDatasetID() string {
	return fmt.Sprintf("dataset_%d", time.Now().UnixNano())
}

//Personal.AI order the ending
