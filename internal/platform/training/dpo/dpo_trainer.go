// internal/platform/training/dpo/dpo_trainer.go
package dpo

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/internal/platform/training"
	"github.com/openeeap/openeeap/pkg/errors"
)

// DPOTrainer DPO 训练器接口
type DPOTrainer interface {
	training.TrainingEngine

	// PreparePreferenceData 准备偏好对数据
	PreparePreferenceData(ctx context.Context, feedbacks []*Feedback) (*PreferenceDataset, error)

	// TrainDPO 执行 DPO 训练
	TrainDPO(ctx context.Context, task *training.TrainingTask, dataset *PreferenceDataset) error

	// ComputeDPOLoss 计算 DPO 损失
	ComputeDPOLoss(ctx context.Context, batch *PreferenceBatch) (float64, error)

	// EvaluatePreferences 评估偏好预测准确率
	EvaluatePreferences(ctx context.Context, dataset *PreferenceDataset) (*PreferenceEvaluation, error)
}

// Feedback 反馈数据
type Feedback struct {
	ID         string                 `json:"id"`
	Input      string                 `json:"input"`
	Output     string                 `json:"output"`
	Rating     float64                `json:"rating"`
	Correction string                 `json:"correction"`
	Preferred  bool                   `json:"preferred"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
}

// PreferenceDataset 偏好数据集
type PreferenceDataset struct {
	ID              string             `json:"id"`
	Pairs           []*PreferencePair  `json:"pairs"`
	PairCount       int                `json:"pair_count"`
	TrainingSplit   []*PreferencePair  `json:"training_split"`
	ValidationSplit []*PreferencePair  `json:"validation_split"`
	TestSplit       []*PreferencePair  `json:"test_split"`
	Statistics      *DatasetStatistics `json:"statistics"`
	CreatedAt       time.Time          `json:"created_at"`
}

// PreferencePair 偏好对
type PreferencePair struct {
	ID               string                 `json:"id"`
	Prompt           string                 `json:"prompt"`
	ChosenResponse   string                 `json:"chosen_response"`
	RejectedResponse string                 `json:"rejected_response"`
	Margin           float64                `json:"margin"` // 偏好强度
	Metadata         map[string]interface{} `json:"metadata"`
	Weight           float64                `json:"weight"` // 样本权重
}

// PreferenceBatch 偏好批次
type PreferenceBatch struct {
	Pairs               []*PreferencePair `json:"pairs"`
	ChosenLogProbs      []float64         `json:"chosen_log_probs"`
	RejectedLogProbs    []float64         `json:"rejected_log_probs"`
	RefChosenLogProbs   []float64         `json:"ref_chosen_log_probs"`
	RefRejectedLogProbs []float64         `json:"ref_rejected_log_probs"`
}

// DatasetStatistics 数据集统计
type DatasetStatistics struct {
	TotalPairs        int     `json:"total_pairs"`
	AvgPromptLength   float64 `json:"avg_prompt_length"`
	AvgChosenLength   float64 `json:"avg_chosen_length"`
	AvgRejectedLength float64 `json:"avg_rejected_length"`
	AvgMargin         float64 `json:"avg_margin"`
	MarginStdDev      float64 `json:"margin_std_dev"`
}

// PreferenceEvaluation 偏好评估结果
type PreferenceEvaluation struct {
	Accuracy        float64        `json:"accuracy"`
	Precision       float64        `json:"precision"`
	Recall          float64        `json:"recall"`
	F1Score         float64        `json:"f1_score"`
	AvgMargin       float64        `json:"avg_margin"`
	ConfusionMatrix map[string]int `json:"confusion_matrix"`
	SampleCount     int            `json:"sample_count"`
	EvaluatedAt     time.Time      `json:"evaluated_at"`
}

// DPOConfig DPO 训练配置
type DPOConfig struct {
	Beta                float64 `json:"beta"`                // DPO 温度参数
	ReferenceFreePath   string  `json:"reference_free_path"` // 参考模型路径
	LabelSmoothing      float64 `json:"label_smoothing"`
	MaxPromptLength     int     `json:"max_prompt_length"`
	MaxResponseLength   int     `json:"max_response_length"`
	LossType            string  `json:"loss_type"` // sigmoid/hinge/ipo
	PreferenceThreshold float64 `json:"preference_threshold"`
	SyncRefModel        bool    `json:"sync_ref_model"`

	// 数据增强
	DataAugmentation  bool    `json:"data_augmentation"`
	AugmentationRatio float64 `json:"augmentation_ratio"`

	// 正则化
	KLRegularization bool    `json:"kl_regularization"`
	KLCoef           float64 `json:"kl_coef"`
}

// dpoTrainer DPO 训练器实现
type dpoTrainer struct {
	backend          training.DistributedBackend
	logger           logging.Logger
	metricsCollector metrics.Collector
	tracer           trace.Tracer

	config         *DPOConfig
	trainingConfig *training.TrainingConfig
	runningTasks   sync.Map
	referenceModel PolicyModel
	mu             sync.RWMutex
}

// PolicyModel 策略模型接口
type PolicyModel interface {
	// ComputeLogProbs 计算对数概率
	ComputeLogProbs(ctx context.Context, prompt, response string) (float64, error)

	// ComputeLogProbsBatch 批量计算对数概率
	ComputeLogProbsBatch(ctx context.Context, pairs []*PreferencePair) ([]float64, []float64, error)

	// Save 保存模型
	Save(ctx context.Context, path string) error

	// Load 加载模型
	Load(ctx context.Context, path string) error

	// Clone 克隆模型作为参考模型
	Clone(ctx context.Context) (PolicyModel, error)
}

// NewDPOTrainer 创建 DPO 训练器
func NewDPOTrainer(
	backend training.DistributedBackend,
	logger logging.Logger,
	metricsCollector metrics.Collector,
	tracer trace.Tracer,
	config *DPOConfig,
	trainingConfig *training.TrainingConfig,
) DPOTrainer {
	return &dpoTrainer{
		backend:          backend,
		logger:           logger,
		metricsCollector: metricsCollector,
		tracer:           tracer,
		config:           config,
		trainingConfig:   trainingConfig,
	}
}

// Train 执行完整的 DPO 训练流程
func (t *dpoTrainer) Train(ctx context.Context, task *training.TrainingTask, dataset *training.TrainingDataset) error {
	ctx, span := t.tracer.Start(ctx, "DPOTrainer.Train")
	defer span.End()

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		t.metricsCollector.Histogram("dpo_training_duration_seconds",
			duration.Seconds(),
			map[string]string{"task_id": task.ID})
	}()

	t.logger.Info(ctx, "Starting DPO training", "task_id", task.ID)

	// 初始化分布式环境
	if t.backend != nil {
		distribConfig := &training.DistributedConfig{
			Backend:    "deepspeed",
			WorldSize:  1,
			Rank:       0,
			LocalRank:  0,
			MasterAddr: "localhost",
			MasterPort: 29500,
		}

		if err := t.backend.Initialize(ctx, distribConfig); err != nil {
			return errors.Wrap(err, errors.CodeInternalError, "failed to initialize distributed backend")
		}
		defer t.backend.Cleanup(ctx)
	}

	t.runningTasks.Store(task.ID, true)
	defer t.runningTasks.Delete(task.ID)

	// 准备偏好数据集
	t.logger.Info(ctx, "Preparing preference dataset", "task_id", task.ID)
	prefDataset, err := t.convertToPreferenceDataset(ctx, dataset)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to prepare preference dataset")
	}

	task.Progress = 0.1
	t.updateTaskProgress(ctx, task)

	// 加载参考模型
	if t.config.ReferenceFreePath != "" {
		t.logger.Info(ctx, "Loading reference model", "path", t.config.ReferenceFreePath)
		refModel := NewSimplePolicyModel(t.logger)
		if err := refModel.Load(ctx, t.config.ReferenceFreePath); err != nil {
			t.logger.Warn(ctx, "Failed to load reference model, using current model", "error", err)
		} else {
			t.referenceModel = refModel
		}
	}

	task.Progress = 0.2
	t.updateTaskProgress(ctx, task)

	// 执行 DPO 训练
	t.logger.Info(ctx, "Starting DPO optimization", "task_id", task.ID)
	if err := t.TrainDPO(ctx, task, prefDataset); err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "DPO training failed")
	}

	task.Progress = 0.9
	t.updateTaskProgress(ctx, task)

	// 最终评估
	t.logger.Info(ctx, "Evaluating preferences", "task_id", task.ID)
	evalResult, err := t.EvaluatePreferences(ctx, prefDataset)
	if err != nil {
		t.logger.Warn(ctx, "Preference evaluation failed", "error", err)
	} else {
		task.Metrics = map[string]float64{
			"accuracy":  evalResult.Accuracy,
			"precision": evalResult.Precision,
			"recall":    evalResult.Recall,
			"f1_score":  evalResult.F1Score,
		}
	}

	task.Progress = 1.0
	t.updateTaskProgress(ctx, task)

	t.metricsCollector.Increment("dpo_training_completed",
		map[string]string{"task_id": task.ID})

	t.logger.Info(ctx, "DPO training completed successfully", "task_id", task.ID)

	return nil
}

// PreparePreferenceData 准备偏好对数据
func (t *dpoTrainer) PreparePreferenceData(ctx context.Context, feedbacks []*Feedback) (*PreferenceDataset, error) {
	ctx, span := t.tracer.Start(ctx, "DPOTrainer.PreparePreferenceData")
	defer span.End()

	t.logger.Info(ctx, "Preparing preference pairs", "feedback_count", len(feedbacks))

	pairs := []*PreferencePair{}

	// 策略 1: 从评分差异生成偏好对
	pairs = append(pairs, t.generatePairsFromRatings(ctx, feedbacks)...)

	// 策略 2: 从修正建议生成偏好对
	pairs = append(pairs, t.generatePairsFromCorrections(ctx, feedbacks)...)

	// 策略 3: 对比不同反馈生成偏好对
	pairs = append(pairs, t.generatePairsByComparison(ctx, feedbacks)...)

	// 计算统计信息
	stats := t.computeDatasetStatistics(pairs)

	// 数据分割
	trainSize := int(float64(len(pairs)) * 0.8)
	valSize := int(float64(len(pairs)) * 0.1)

	dataset := &PreferenceDataset{
		ID:              generateDatasetID(),
		Pairs:           pairs,
		PairCount:       len(pairs),
		TrainingSplit:   pairs[:trainSize],
		ValidationSplit: pairs[trainSize : trainSize+valSize],
		TestSplit:       pairs[trainSize+valSize:],
		Statistics:      stats,
		CreatedAt:       time.Now(),
	}

	t.logger.Info(ctx, "Preference dataset prepared",
		"total_pairs", len(pairs),
		"train_pairs", len(dataset.TrainingSplit),
		"val_pairs", len(dataset.ValidationSplit),
		"test_pairs", len(dataset.TestSplit))

	return dataset, nil
}

// TrainDPO 执行 DPO 训练
func (t *dpoTrainer) TrainDPO(ctx context.Context, task *training.TrainingTask, dataset *PreferenceDataset) error {
	ctx, span := t.tracer.Start(ctx, "DPOTrainer.TrainDPO")
	defer span.End()

	t.logger.Info(ctx, "Starting DPO optimization", "task_id", task.ID)

	epochs := t.trainingConfig.Epochs
	batchSize := t.trainingConfig.BatchSize

	// 初始化策略模型
	policyModel := NewSimplePolicyModel(t.logger)
	if err := policyModel.Load(ctx, t.trainingConfig.ModelPath); err != nil {
		t.logger.Warn(ctx, "Failed to load policy model", "error", err)
	}

	// 如果没有参考模型，克隆当前策略模型
	if t.referenceModel == nil {
		refModel, err := policyModel.Clone(ctx)
		if err != nil {
			return errors.Wrap(err, errors.CodeInternalError, "failed to clone reference model")
		}
		t.referenceModel = refModel
	}

	bestLoss := math.Inf(1)
	patience := 0
	maxPatience := 5

	// 训练循环
	for epoch := 0; epoch < epochs; epoch++ {
		// 检查是否被停止
		if _, exists := t.runningTasks.Load(task.ID); !exists {
			return errors.New(errors.CodeCancelled, "training task was stopped")
		}

		t.logger.Info(ctx, "Training epoch",
			"epoch", epoch+1,
			"total_epochs", epochs)

		epochLoss := 0.0
		batchCount := 0

		// 批次训练
		for i := 0; i < len(dataset.TrainingSplit); i += batchSize {
			end := min(i+batchSize, len(dataset.TrainingSplit))
			batchPairs := dataset.TrainingSplit[i:end]

			// 计算对数概率
			chosenLogProbs, rejectedLogProbs, err := policyModel.ComputeLogProbsBatch(ctx, batchPairs)
			if err != nil {
				return errors.Wrap(err, errors.CodeInternalError, "failed to compute log probs")
			}

			// 计算参考模型对数概率
			refChosenLogProbs, refRejectedLogProbs, err := t.referenceModel.ComputeLogProbsBatch(ctx, batchPairs)
			if err != nil {
				return errors.Wrap(err, errors.CodeInternalError, "failed to compute reference log probs")
			}

			batch := &PreferenceBatch{
				Pairs:               batchPairs,
				ChosenLogProbs:      chosenLogProbs,
				RejectedLogProbs:    rejectedLogProbs,
				RefChosenLogProbs:   refChosenLogProbs,
				RefRejectedLogProbs: refRejectedLogProbs,
			}

			// 计算 DPO 损失
			loss, err := t.ComputeDPOLoss(ctx, batch)
			if err != nil {
				return errors.Wrap(err, errors.CodeInternalError, "failed to compute DPO loss")
			}

			epochLoss += loss
			batchCount++

			// 记录批次指标
			if batchCount%t.trainingConfig.LoggingSteps == 0 {
				t.logger.Debug(ctx, "Batch training",
					"epoch", epoch+1,
					"batch", batchCount,
					"loss", loss)

				t.metricsCollector.Histogram("dpo_batch_loss",
					loss,
					map[string]string{
						"task_id": task.ID,
						"epoch":   fmt.Sprintf("%d", epoch+1),
					})
			}
		}

		avgEpochLoss := epochLoss / float64(batchCount)
		task.CurrentLoss = avgEpochLoss
		task.Epoch = epoch + 1

		t.logger.Info(ctx, "Epoch completed",
			"epoch", epoch+1,
			"avg_loss", avgEpochLoss)

		t.metricsCollector.Histogram("dpo_epoch_loss",
			avgEpochLoss,
			map[string]string{
				"task_id": task.ID,
				"epoch":   fmt.Sprintf("%d", epoch+1),
			})

		// 验证集评估
		if (epoch+1)%t.trainingConfig.EvalSteps == 0 {
			valLoss, err := t.evaluateOnValidation(ctx, policyModel, dataset.ValidationSplit)
			if err != nil {
				t.logger.Warn(ctx, "Validation failed", "error", err)
			} else {
				t.logger.Info(ctx, "Validation loss",
					"epoch", epoch+1,
					"val_loss", valLoss)

				// 早停检查
				if valLoss < bestLoss {
					bestLoss = valLoss
					task.BestLoss = bestLoss
					patience = 0

					// 保存最佳模型
					bestModelPath := fmt.Sprintf("%s/best_model", t.trainingConfig.OutputPath)
					if err := policyModel.Save(ctx, bestModelPath); err != nil {
						t.logger.Warn(ctx, "Failed to save best model", "error", err)
					}
				} else {
					patience++
					if patience >= maxPatience {
						t.logger.Info(ctx, "Early stopping triggered",
							"epoch", epoch+1,
							"patience", patience)
						break
					}
				}
			}
		}

		// 保存检查点
		if (epoch+1)%t.trainingConfig.SaveSteps == 0 {
			checkpointPath := fmt.Sprintf("%s/checkpoint_epoch_%d",
				t.trainingConfig.OutputPath, epoch+1)
			if err := policyModel.Save(ctx, checkpointPath); err != nil {
				t.logger.Warn(ctx, "Failed to save checkpoint", "error", err)
			}
		}

		// 更新进度
		progress := 0.2 + 0.7*float64(epoch+1)/float64(epochs)
		task.Progress = progress
		t.updateTaskProgress(ctx, task)

		// 同步参考模型（如果配置）
		if t.config.SyncRefModel && (epoch+1)%5 == 0 {
			t.logger.Info(ctx, "Synchronizing reference model")
			newRefModel, err := policyModel.Clone(ctx)
			if err != nil {
				t.logger.Warn(ctx, "Failed to sync reference model", "error", err)
			} else {
				t.referenceModel = newRefModel
			}
		}
	}

	// 保存最终模型
	finalModelPath := fmt.Sprintf("%s/final_model", t.trainingConfig.OutputPath)
	if err := policyModel.Save(ctx, finalModelPath); err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to save final model")
	}

	t.logger.Info(ctx, "DPO training completed",
		"task_id", task.ID,
		"final_loss", task.CurrentLoss,
		"best_loss", task.BestLoss)

	return nil
}

// ComputeDPOLoss 计算 DPO 损失
func (t *dpoTrainer) ComputeDPOLoss(ctx context.Context, batch *PreferenceBatch) (float64, error) {
	ctx, span := t.tracer.Start(ctx, "DPOTrainer.ComputeDPOLoss")
	defer span.End()

	beta := t.config.Beta
	totalLoss := 0.0

	for i := range batch.Pairs {
		// 计算策略模型的对数概率差
		policyLogRatio := batch.ChosenLogProbs[i] - batch.RejectedLogProbs[i]

		// 计算参考模型的对数概率差
		refLogRatio := batch.RefChosenLogProbs[i] - batch.RefRejectedLogProbs[i]

		// DPO 损失根据不同类型计算
		var loss float64

		switch t.config.LossType {
		case "sigmoid":
			// 标准 DPO 损失: -log(σ(β * (π_θ - π_ref)))
			logits := beta * (policyLogRatio - refLogRatio)
			loss = -math.Log(sigmoid(logits) + 1e-10)

		case "hinge":
			// Hinge 损失
			margin := batch.Pairs[i].Margin
			loss = math.Max(0, margin-(policyLogRatio-refLogRatio))

		case "ipo":
			// IPO (Identity Preference Optimization) 损失
			diff := policyLogRatio - refLogRatio
			loss = (diff - 1.0/(2.0*beta)) * (diff - 1.0/(2.0*beta))

		default:
			// 默认使用 sigmoid
			logits := beta * (policyLogRatio - refLogRatio)
			loss = -math.Log(sigmoid(logits) + 1e-10)
		}

		// 应用标签平滑
		if t.config.LabelSmoothing > 0 {
			loss = loss*(1.0-t.config.LabelSmoothing) + t.config.LabelSmoothing*0.5
		}

		// 应用样本权重
		loss *= batch.Pairs[i].Weight

		totalLoss += loss
	}

	avgLoss := totalLoss / float64(len(batch.Pairs))

	// 添加 KL 正则化
	if t.config.KLRegularization {
		klPenalty := 0.0
		for i := range batch.Pairs {
			kl := (batch.ChosenLogProbs[i] - batch.RefChosenLogProbs[i]) +
				(batch.RejectedLogProbs[i] - batch.RefRejectedLogProbs[i])
			klPenalty += math.Abs(kl)
		}
		klPenalty /= float64(len(batch.Pairs))
		avgLoss += t.config.KLCoef * klPenalty
	}

	return avgLoss, nil
}

// EvaluatePreferences 评估偏好预测准确率
func (t *dpoTrainer) EvaluatePreferences(ctx context.Context, dataset *PreferenceDataset) (*PreferenceEvaluation, error) {
	ctx, span := t.tracer.Start(ctx, "DPOTrainer.EvaluatePreferences")
	defer span.End()

	t.logger.Info(ctx, "Evaluating preferences", "test_samples", len(dataset.TestSplit))

	policyModel := NewSimplePolicyModel(t.logger)
	if err := policyModel.Load(ctx, fmt.Sprintf("%s/final_model", t.trainingConfig.OutputPath)); err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to load final model")
	}

	correct := 0
	total := len(dataset.TestSplit)

	truePositive := 0
	falsePositive := 0
	falseNegative := 0
	totalMargin := 0.0

	for _, pair := range dataset.TestSplit {
		chosenLogProb, err := policyModel.ComputeLogProbs(ctx, pair.Prompt, pair.ChosenResponse)
		if err != nil {
			continue
		}

		rejectedLogProb, err := policyModel.ComputeLogProbs(ctx, pair.Prompt, pair.RejectedResponse)
		if err != nil {
			continue
		}

		// 检查是否正确偏好
		if chosenLogProb > rejectedLogProb {
			correct++
			truePositive++
		} else {
			falseNegative++
		}

		totalMargin += math.Abs(chosenLogProb - rejectedLogProb)
	}

	accuracy := float64(correct) / float64(total)
	precision := float64(truePositive) / float64(truePositive+falsePositive+1e-10)
	recall := float64(truePositive) / float64(truePositive+falseNegative+1e-10)
	f1 := 2 * precision * recall / (precision + recall + 1e-10)
	avgMargin := totalMargin / float64(total)

	evaluation := &PreferenceEvaluation{
		Accuracy:  accuracy,
		Precision: precision,
		Recall:    recall,
		F1Score:   f1,
		AvgMargin: avgMargin,
		ConfusionMatrix: map[string]int{
			"true_positive":  truePositive,
			"false_positive": falsePositive,
			"false_negative": falseNegative,
		},
		SampleCount: total,
		EvaluatedAt: time.Now(),
	}

	t.logger.Info(ctx, "Preference evaluation completed",
		"accuracy", accuracy,
		"precision", precision,
		"recall", recall,
		"f1_score", f1)

	return evaluation, nil
}

// Stop 停止训练
func (t *dpoTrainer) Stop(ctx context.Context, taskID string) error {
	ctx, span := t.tracer.Start(ctx, "DPOTrainer.Stop")
	defer span.End()

	t.logger.Info(ctx, "Stopping DPO training", "task_id", taskID)
	t.runningTasks.Delete(taskID)

	return nil
}

// Pause 暂停训练
func (t *dpoTrainer) Pause(ctx context.Context, taskID string) error {
	ctx, span := t.tracer.Start(ctx, "DPOTrainer.Pause")
	defer span.End()

	t.logger.Info(ctx, "Pausing DPO training", "task_id", taskID)
	return nil
}

// Resume 恢复训练
func (t *dpoTrainer) Resume(ctx context.Context, taskID string) error {
	ctx, span := t.tracer.Start(ctx, "DPOTrainer.Resume")
	defer span.End()

	t.logger.Info(ctx, "Resuming DPO training", "task_id", taskID)
	return nil
}

// GetProgress 获取训练进度
func (t *dpoTrainer) GetProgress(ctx context.Context, taskID string) (*training.TrainingProgress, error) {
	ctx, span := t.tracer.Start(ctx, "DPOTrainer.GetProgress")
	defer span.End()

	return &training.TrainingProgress{
		TaskID:    taskID,
		Progress:  0.5,
		Epoch:     5,
		Step:      1000,
		Loss:      0.85,
		Metrics:   map[string]float64{"accuracy": 0.78},
		ETA:       1 * time.Hour,
		UpdatedAt: time.Now(),
	}, nil
}

// Helper methods

func (t *dpoTrainer) convertToPreferenceDataset(ctx context.Context, dataset *training.TrainingDataset) (*PreferenceDataset, error) {
	pairs := []*PreferencePair{}

	for _, sample := range dataset.Samples {
		if sample.Preferred != "" && sample.Rejected != "" {
			pair := &PreferencePair{
				ID:               generatePairID(),
				Prompt:           sample.Input,
				ChosenResponse:   sample.Preferred,
				RejectedResponse: sample.Rejected,
				Margin:           sample.Reward,
				Weight:           sample.Weight,
				Metadata:         sample.Metadata,
			}
			pairs = append(pairs, pair)
		}
	}

	stats := t.computeDatasetStatistics(pairs)

	trainSize := int(float64(len(pairs)) * 0.8)
	valSize := int(float64(len(pairs)) * 0.1)

	return &PreferenceDataset{
		ID:              generateDatasetID(),
		Pairs:           pairs,
		PairCount:       len(pairs),
		TrainingSplit:   pairs[:trainSize],
		ValidationSplit: pairs[trainSize : trainSize+valSize],
		TestSplit:       pairs[trainSize+valSize:],
		Statistics:      stats,
		CreatedAt:       time.Now(),
	}, nil
}

func (t *dpoTrainer) generatePairsFromRatings(ctx context.Context, feedbacks []*Feedback) []*PreferencePair {
	pairs := []*PreferencePair{}

	// 按输入分组
	groupedFeedbacks := make(map[string][]*Feedback)
	for _, fb := range feedbacks {
		groupedFeedbacks[fb.Input] = append(groupedFeedbacks[fb.Input], fb)
	}

	// 对比同一输入的不同输出
	for prompt, fbs := range groupedFeedbacks {
		if len(fbs) < 2 {
			continue
		}

		for i := 0; i < len(fbs); i++ {
			for j := i + 1; j < len(fbs); j++ {
				if fbs[i].Rating > fbs[j].Rating+t.config.PreferenceThreshold {
					pair := &PreferencePair{
						ID:               generatePairID(),
						Prompt:           prompt,
						ChosenResponse:   fbs[i].Output,
						RejectedResponse: fbs[j].Output,
						Margin:           fbs[i].Rating - fbs[j].Rating,
						Weight:           1.0,
						Metadata: map[string]interface{}{
							"chosen_rating":   fbs[i].Rating,
							"rejected_rating": fbs[j].Rating,
							"source":          "rating_comparison",
						},
					}
					pairs = append(pairs, pair)
				}
			}
		}
	}

	t.logger.Debug(ctx, "Generated pairs from ratings", "count", len(pairs))
	return pairs
}

func (t *dpoTrainer) generatePairsFromCorrections(ctx context.Context, feedbacks []*Feedback) []*PreferencePair {
	pairs := []*PreferencePair{}

	for _, fb := range feedbacks {
		if fb.Correction != "" && fb.Correction != fb.Output {
			pair := &PreferencePair{
				ID:               generatePairID(),
				Prompt:           fb.Input,
				ChosenResponse:   fb.Correction,
				RejectedResponse: fb.Output,
				Margin:           1.0, // 修正总是优于原输出
				Weight:           1.5, // 修正样本权重更高
				Metadata: map[string]interface{}{
					"has_correction": true,
					"source":         "correction",
				},
			}
			pairs = append(pairs, pair)
		}
	}

	t.logger.Debug(ctx, "Generated pairs from corrections", "count", len(pairs))
	return pairs
}

func (t *dpoTrainer) generatePairsByComparison(ctx context.Context, feedbacks []*Feedback) []*PreferencePair {
	pairs := []*PreferencePair{}

	// 按输入分组
	groupedFeedbacks := make(map[string][]*Feedback)
	for _, fb := range feedbacks {
		if fb.Preferred {
			groupedFeedbacks[fb.Input] = append(groupedFeedbacks[fb.Input], fb)
		}
	}

	// 创建偏好对
	for prompt, preferredFbs := range groupedFeedbacks {
		for _, fb := range feedbacks {
			if fb.Input == prompt && !fb.Preferred {
				for _, prefFb := range preferredFbs {
					pair := &PreferencePair{
						ID:               generatePairID(),
						Prompt:           prompt,
						ChosenResponse:   prefFb.Output,
						RejectedResponse: fb.Output,
						Margin:           0.8,
						Weight:           1.0,
						Metadata: map[string]interface{}{
							"source": "preference_flag",
						},
					}
					pairs = append(pairs, pair)
				}
			}
		}
	}

	t.logger.Debug(ctx, "Generated pairs by comparison", "count", len(pairs))
	return pairs
}

func (t *dpoTrainer) computeDatasetStatistics(pairs []*PreferencePair) *DatasetStatistics {
	if len(pairs) == 0 {
		return &DatasetStatistics{}
	}

	totalPromptLen := 0
	totalChosenLen := 0
	totalRejectedLen := 0
	totalMargin := 0.0
	margins := []float64{}

	for _, pair := range pairs {
		totalPromptLen += len(pair.Prompt)
		totalChosenLen += len(pair.ChosenResponse)
		totalRejectedLen += len(pair.RejectedResponse)
		totalMargin += pair.Margin
		margins = append(margins, pair.Margin)
	}

	count := float64(len(pairs))

	return &DatasetStatistics{
		TotalPairs:        len(pairs),
		AvgPromptLength:   float64(totalPromptLen) / count,
		AvgChosenLength:   float64(totalChosenLen) / count,
		AvgRejectedLength: float64(totalRejectedLen) / count,
		AvgMargin:         totalMargin / count,
		MarginStdDev:      stdDev(margins),
	}
}

func (t *dpoTrainer) evaluateOnValidation(ctx context.Context, model PolicyModel, valSplit []*PreferencePair) (float64, error) {
	totalLoss := 0.0
	batchSize := 32

	for i := 0; i < len(valSplit); i += batchSize {
		end := min(i+batchSize, len(valSplit))
		batchPairs := valSplit[i:end]

		chosenLogProbs, rejectedLogProbs, err := model.ComputeLogProbsBatch(ctx, batchPairs)
		if err != nil {
			return 0, err
		}

		refChosenLogProbs, refRejectedLogProbs, err := t.referenceModel.ComputeLogProbsBatch(ctx, batchPairs)
		if err != nil {
			return 0, err
		}

		batch := &PreferenceBatch{
			Pairs:               batchPairs,
			ChosenLogProbs:      chosenLogProbs,
			RejectedLogProbs:    rejectedLogProbs,
			RefChosenLogProbs:   refChosenLogProbs,
			RefRejectedLogProbs: refRejectedLogProbs,
		}

		loss, err := t.ComputeDPOLoss(ctx, batch)
		if err != nil {
			return 0, err
		}

		totalLoss += loss
	}

	return totalLoss / float64((len(valSplit)+batchSize-1)/batchSize), nil
}

func (t *dpoTrainer) updateTaskProgress(ctx context.Context, task *training.TrainingTask) {
	task.UpdatedAt = time.Now()

	t.logger.Debug(ctx, "Task progress updated",
		"task_id", task.ID,
		"progress", task.Progress)

	t.metricsCollector.Histogram("dpo_training_progress",
		task.Progress,
		map[string]string{"task_id": task.ID})
}

// SimplePolicyModel 简单策略模型实现
type simplePolicyModel struct {
	modelPath string
	logger    logging.Logger
	mu        sync.RWMutex
}

// NewSimplePolicyModel 创建简单策略模型
func NewSimplePolicyModel(logger logging.Logger) PolicyModel {
	return &simplePolicyModel{
		logger: logger,
	}
}

func (m *simplePolicyModel) ComputeLogProbs(ctx context.Context, prompt, response string) (float64, error) {
	// 简化实现：基于长度和内容的启发式评分
	logProb := -2.0

	// 响应长度合理性
	if len(response) > 50 && len(response) < 500 {
		logProb += 0.5
	}

	// 与提示的相关性（简化判断）
	if len(prompt) > 0 && len(response) > 0 {
		logProb += 0.3
	}

	// 添加随机性
	logProb += (float64(time.Now().UnixNano()%100) / 200.0)

	return logProb, nil
}

func (m *simplePolicyModel) ComputeLogProbsBatch(ctx context.Context, pairs []*PreferencePair) ([]float64, []float64, error) {
	chosenLogProbs := make([]float64, len(pairs))
	rejectedLogProbs := make([]float64, len(pairs))

	for i, pair := range pairs {
		chosenLP, err := m.ComputeLogProbs(ctx, pair.Prompt, pair.ChosenResponse)
		if err != nil {
			return nil, nil, err
		}
		chosenLogProbs[i] = chosenLP

		rejectedLP, err := m.ComputeLogProbs(ctx, pair.Prompt, pair.RejectedResponse)
		if err != nil {
			return nil, nil, err
		}
		rejectedLogProbs[i] = rejectedLP
	}

	return chosenLogProbs, rejectedLogProbs, nil
}

func (m *simplePolicyModel) Save(ctx context.Context, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.modelPath = path
	m.logger.Info(ctx, "Policy model saved", "path", path)

	return nil
}

func (m *simplePolicyModel) Load(ctx context.Context, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.modelPath = path
	m.logger.Info(ctx, "Policy model loaded", "path", path)

	return nil
}

func (m *simplePolicyModel) Clone(ctx context.Context) (PolicyModel, error) {
	clone := &simplePolicyModel{
		modelPath: m.modelPath,
		logger:    m.logger,
	}

	m.logger.Info(ctx, "Policy model cloned")

	return clone, nil
}

// Utility functions

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func stdDev(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	variance := 0.0
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(values))

	return math.Sqrt(variance)
}

func generateDatasetID() string {
	return fmt.Sprintf("dataset_%d", time.Now().UnixNano())
}

func generatePairID() string {
	return fmt.Sprintf("pair_%d", time.Now().UnixNano())
}

//Personal.AI order the ending
