// Package types provides enumeration type definitions for OpenEAAP.
// All enums implement String(), Valid(), and FromString() methods
// for type-safe conversions and validation across the platform.
package types

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

// ============================================================================
// Training Type Enumerations
// ============================================================================

// TrainingType represents the type of training algorithm
type TrainingType string

const (
	// TrainingTypeRLHF represents Reinforcement Learning from Human Feedback
	TrainingTypeRLHF TrainingType = "rlhf"

	// TrainingTypeDPO represents Direct Preference Optimization
	TrainingTypeDPO TrainingType = "dpo"

	// TrainingTypeSFT represents Supervised Fine-Tuning
	TrainingTypeSFT TrainingType = "sft"

	// TrainingTypePPO represents Proximal Policy Optimization
	TrainingTypePPO TrainingType = "ppo"

	// TrainingTypeLoRA represents Low-Rank Adaptation
	TrainingTypeLoRA TrainingType = "lora"
)

// String returns the string representation
func (tt TrainingType) String() string {
	return string(tt)
}

// Valid checks if the training type is valid
func (tt TrainingType) Valid() bool {
	switch tt {
	case TrainingTypeRLHF, TrainingTypeDPO, TrainingTypeSFT, TrainingTypePPO, TrainingTypeLoRA:
		return true
	default:
		return false
	}
}

// FromStringTrainingType converts string to TrainingType
func FromStringTrainingType(s string) (TrainingType, error) {
	tt := TrainingType(strings.ToLower(s))
	if !tt.Valid() {
		return "", fmt.Errorf("invalid training type: %s", s)
	}
	return tt, nil
}

// Value implements driver.Valuer for database storage
func (tt TrainingType) Value() (driver.Value, error) {
	return string(tt), nil
}

// Scan implements sql.Scanner for database retrieval
func (tt *TrainingType) Scan(value interface{}) error {
	if value == nil {
		*tt = ""
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan type %T into TrainingType", value)
	}

	parsed, err := FromStringTrainingType(str)
	if err != nil {
		return err
	}

	*tt = parsed
	return nil
}

// ============================================================================
// Runtime Type Enumerations
// ============================================================================

// RuntimeType represents the execution runtime for agents
type RuntimeType string

const (
	// RuntimeTypeNative represents native OpenEAAP runtime
	RuntimeTypeNative RuntimeType = "native"

	// RuntimeTypeLangChain represents LangChain runtime adapter
	RuntimeTypeLangChain RuntimeType = "langchain"

	// RuntimeTypeAutoGPT represents AutoGPT runtime adapter
	RuntimeTypeAutoGPT RuntimeType = "autogpt"

	// RuntimeTypeCustom represents custom plugin runtime
	RuntimeTypeCustom RuntimeType = "custom"
)

// String returns the string representation
func (rt RuntimeType) String() string {
	return string(rt)
}

// Valid checks if the runtime type is valid
func (rt RuntimeType) Valid() bool {
	switch rt {
	case RuntimeTypeNative, RuntimeTypeLangChain, RuntimeTypeAutoGPT, RuntimeTypeCustom:
		return true
	default:
		return false
	}
}

// FromStringRuntimeType converts string to RuntimeType
func FromStringRuntimeType(s string) (RuntimeType, error) {
	rt := RuntimeType(strings.ToLower(s))
	if !rt.Valid() {
		return "", fmt.Errorf("invalid runtime type: %s", s)
	}
	return rt, nil
}

// Value implements driver.Valuer for database storage
func (rt RuntimeType) Value() (driver.Value, error) {
	return string(rt), nil
}

// Scan implements sql.Scanner for database retrieval
func (rt *RuntimeType) Scan(value interface{}) error {
	if value == nil {
		*rt = ""
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan type %T into RuntimeType", value)
	}

	parsed, err := FromStringRuntimeType(str)
	if err != nil {
		return err
	}

	*rt = parsed
	return nil
}

// ============================================================================
// Feedback Type Enumerations
// ============================================================================

// FeedbackType represents the type of user feedback
type FeedbackType string

const (
	// FeedbackTypeThumbsUp represents positive feedback
	FeedbackTypeThumbsUp FeedbackType = "thumbs_up"

	// FeedbackTypeThumbsDown represents negative feedback
	FeedbackTypeThumbsDown FeedbackType = "thumbs_down"

	// FeedbackTypeCorrection represents user correction
	FeedbackTypeCorrection FeedbackType = "correction"

	// FeedbackTypeRating represents numeric rating (1-5)
	FeedbackTypeRating FeedbackType = "rating"

	// FeedbackTypeComment represents textual comment
	FeedbackTypeComment FeedbackType = "comment"
)

// String returns the string representation
func (ft FeedbackType) String() string {
	return string(ft)
}

// Valid checks if the feedback type is valid
func (ft FeedbackType) Valid() bool {
	switch ft {
	case FeedbackTypeThumbsUp, FeedbackTypeThumbsDown, FeedbackTypeCorrection, FeedbackTypeRating, FeedbackTypeComment:
		return true
	default:
		return false
	}
}

// FromStringFeedbackType converts string to FeedbackType
func FromStringFeedbackType(s string) (FeedbackType, error) {
	ft := FeedbackType(strings.ToLower(s))
	if !ft.Valid() {
		return "", fmt.Errorf("invalid feedback type: %s", s)
	}
	return ft, nil
}

// Value implements driver.Valuer for database storage
func (ft FeedbackType) Value() (driver.Value, error) {
	return string(ft), nil
}

// Scan implements sql.Scanner for database retrieval
func (ft *FeedbackType) Scan(value interface{}) error {
	if value == nil {
		*ft = ""
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan type %T into FeedbackType", value)
	}

	parsed, err := FromStringFeedbackType(str)
	if err != nil {
		return err
	}

	*ft = parsed
	return nil
}

// ============================================================================
// Sensitivity Level Enumerations
// ============================================================================

// SensitivityLevel represents data sensitivity classification
type SensitivityLevel string

const (
	// SensitivityPublic represents publicly available data
	SensitivityPublic SensitivityLevel = "public"

	// SensitivityInternal represents internal-use data
	SensitivityInternal SensitivityLevel = "internal"

	// SensitivityConfidential represents confidential data
	SensitivityConfidential SensitivityLevel = "confidential"

	// SensitivityRestricted represents highly sensitive data
	SensitivityRestricted SensitivityLevel = "restricted"

	// SensitivityPII represents personally identifiable information
	SensitivityPII SensitivityLevel = "pii"
)

// String returns the string representation
func (sl SensitivityLevel) String() string {
	return string(sl)
}

// Valid checks if the sensitivity level is valid
func (sl SensitivityLevel) Valid() bool {
	switch sl {
	case SensitivityPublic, SensitivityInternal, SensitivityConfidential, SensitivityRestricted, SensitivityPII:
		return true
	default:
		return false
	}
}

// FromStringSensitivityLevel converts string to SensitivityLevel
func FromStringSensitivityLevel(s string) (SensitivityLevel, error) {
	sl := SensitivityLevel(strings.ToLower(s))
	if !sl.Valid() {
		return "", fmt.Errorf("invalid sensitivity level: %s", s)
	}
	return sl, nil
}

// Rank returns the numeric rank of sensitivity (higher = more sensitive)
func (sl SensitivityLevel) Rank() int {
	switch sl {
	case SensitivityPublic:
		return 1
	case SensitivityInternal:
		return 2
	case SensitivityConfidential:
		return 3
	case SensitivityRestricted:
		return 4
	case SensitivityPII:
		return 5
	default:
		return 0
	}
}

// IsMoreSensitiveThan checks if this level is more sensitive than another
func (sl SensitivityLevel) IsMoreSensitiveThan(other SensitivityLevel) bool {
	return sl.Rank() > other.Rank()
}

// Value implements driver.Valuer for database storage
func (sl SensitivityLevel) Value() (driver.Value, error) {
	return string(sl), nil
}

// Scan implements sql.Scanner for database retrieval
func (sl *SensitivityLevel) Scan(value interface{}) error {
	if value == nil {
		*sl = ""
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan type %T into SensitivityLevel", value)
	}

	parsed, err := FromStringSensitivityLevel(str)
	if err != nil {
		return err
	}

	*sl = parsed
	return nil
}

// ============================================================================
// Decision Type Enumerations
// ============================================================================

// DecisionType represents policy decision outcomes
type DecisionType string

const (
	// DecisionPermit allows the operation
	DecisionPermit DecisionType = "permit"

	// DecisionDeny denies the operation
	DecisionDeny DecisionType = "deny"

	// DecisionNotApplicable indicates policy doesn't apply
	DecisionNotApplicable DecisionType = "not_applicable"

	// DecisionIndeterminate indicates decision cannot be made
	DecisionIndeterminate DecisionType = "indeterminate"
)

// String returns the string representation
func (dt DecisionType) String() string {
	return string(dt)
}

// Valid checks if the decision type is valid
func (dt DecisionType) Valid() bool {
	switch dt {
	case DecisionPermit, DecisionDeny, DecisionNotApplicable, DecisionIndeterminate:
		return true
	default:
		return false
	}
}

// FromStringDecisionType converts string to DecisionType
func FromStringDecisionType(s string) (DecisionType, error) {
	dt := DecisionType(strings.ToLower(s))
	if !dt.Valid() {
		return "", fmt.Errorf("invalid decision type: %s", s)
	}
	return dt, nil
}

// IsPermit checks if decision is permit
func (dt DecisionType) IsPermit() bool {
	return dt == DecisionPermit
}

// IsDeny checks if decision is deny
func (dt DecisionType) IsDeny() bool {
	return dt == DecisionDeny
}

// Value implements driver.Valuer for database storage
func (dt DecisionType) Value() (driver.Value, error) {
	return string(dt), nil
}

// Scan implements sql.Scanner for database retrieval
func (dt *DecisionType) Scan(value interface{}) error {
	if value == nil {
		*dt = ""
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan type %T into DecisionType", value)
	}

	parsed, err := FromStringDecisionType(str)
	if err != nil {
		return err
	}

	*dt = parsed
	return nil
}

// ============================================================================
// Execution Status Enumerations
// ============================================================================

// ExecutionStatus represents task/workflow execution status
type ExecutionStatus string

const (
	// ExecutionStatusPending indicates execution is pending
	ExecutionStatusPending ExecutionStatus = "pending"

	// ExecutionStatusRunning indicates execution is in progress
	ExecutionStatusRunning ExecutionStatus = "running"

	// ExecutionStatusCompleted indicates execution completed successfully
	ExecutionStatusCompleted ExecutionStatus = "completed"

	// ExecutionStatusFailed indicates execution failed
	ExecutionStatusFailed ExecutionStatus = "failed"

	// ExecutionStatusCancelled indicates execution was cancelled
	ExecutionStatusCancelled ExecutionStatus = "cancelled"

	// ExecutionStatusPaused indicates execution is paused
	ExecutionStatusPaused ExecutionStatus = "paused"

	// ExecutionStatusTimeout indicates execution timed out
	ExecutionStatusTimeout ExecutionStatus = "timeout"
)

// String returns the string representation
func (es ExecutionStatus) String() string {
	return string(es)
}

// Valid checks if the execution status is valid
func (es ExecutionStatus) Valid() bool {
	switch es {
	case ExecutionStatusPending, ExecutionStatusRunning, ExecutionStatusCompleted,
		ExecutionStatusFailed, ExecutionStatusCancelled, ExecutionStatusPaused, ExecutionStatusTimeout:
		return true
	default:
		return false
	}
}

// FromStringExecutionStatus converts string to ExecutionStatus
func FromStringExecutionStatus(s string) (ExecutionStatus, error) {
	es := ExecutionStatus(strings.ToLower(s))
	if !es.Valid() {
		return "", fmt.Errorf("invalid execution status: %s", s)
	}
	return es, nil
}

// IsTerminal checks if status is terminal (no further transitions)
func (es ExecutionStatus) IsTerminal() bool {
	return es == ExecutionStatusCompleted || es == ExecutionStatusFailed ||
		es == ExecutionStatusCancelled || es == ExecutionStatusTimeout
}

// IsActive checks if execution is active
func (es ExecutionStatus) IsActive() bool {
	return es == ExecutionStatusRunning || es == ExecutionStatusPending
}

// Value implements driver.Valuer for database storage
func (es ExecutionStatus) Value() (driver.Value, error) {
	return string(es), nil
}

// Scan implements sql.Scanner for database retrieval
func (es *ExecutionStatus) Scan(value interface{}) error {
	if value == nil {
		*es = ""
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan type %T into ExecutionStatus", value)
	}

	parsed, err := FromStringExecutionStatus(str)
	if err != nil {
		return err
	}

	*es = parsed
	return nil
}

// ============================================================================
// Model Type Enumerations
// ============================================================================

// ModelType represents the type/category of AI model
type ModelType string

const (
	// ModelTypeLLM represents large language models
	ModelTypeLLM ModelType = "llm"

	// ModelTypeEmbedding represents embedding models
	ModelTypeEmbedding ModelType = "embedding"

	// ModelTypeReranker represents reranking models
	ModelTypeReranker ModelType = "reranker"

	// ModelTypeClassifier represents classification models
	ModelTypeClassifier ModelType = "classifier"

	// ModelTypeVision represents vision models
	ModelTypeVision ModelType = "vision"

	// ModelTypeMultimodal represents multimodal models
	ModelTypeMultimodal ModelType = "multimodal"
)

// String returns the string representation
func (mt ModelType) String() string {
	return string(mt)
}

// Valid checks if the model type is valid
func (mt ModelType) Valid() bool {
	switch mt {
	case ModelTypeLLM, ModelTypeEmbedding, ModelTypeReranker,
		ModelTypeClassifier, ModelTypeVision, ModelTypeMultimodal:
		return true
	default:
		return false
	}
}

// FromStringModelType converts string to ModelType
func FromStringModelType(s string) (ModelType, error) {
	mt := ModelType(strings.ToLower(s))
	if !mt.Valid() {
		return "", fmt.Errorf("invalid model type: %s", s)
	}
	return mt, nil
}

// Value implements driver.Valuer for database storage
func (mt ModelType) Value() (driver.Value, error) {
	return string(mt), nil
}

// Scan implements sql.Scanner for database retrieval
func (mt *ModelType) Scan(value interface{}) error {
	if value == nil {
		*mt = ""
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan type %T into ModelType", value)
	}

	parsed, err := FromStringModelType(str)
	if err != nil {
		return err
	}

	*mt = parsed
	return nil
}

// ============================================================================
// Cache Strategy Enumerations
// ============================================================================

// CacheStrategy represents caching strategy
type CacheStrategy string

const (
	// CacheStrategyNone disables caching
	CacheStrategyNone CacheStrategy = "none"

	// CacheStrategyL1 uses only L1 (local) cache
	CacheStrategyL1 CacheStrategy = "l1"

	// CacheStrategyL2 uses only L2 (Redis) cache
	CacheStrategyL2 CacheStrategy = "l2"

	// CacheStrategyL3 uses only L3 (vector similarity) cache
	CacheStrategyL3 CacheStrategy = "l3"

	// CacheStrategyAll uses all cache levels
	CacheStrategyAll CacheStrategy = "all"
)

// String returns the string representation
func (cs CacheStrategy) String() string {
	return string(cs)
}

// Valid checks if the cache strategy is valid
func (cs CacheStrategy) Valid() bool {
	switch cs {
	case CacheStrategyNone, CacheStrategyL1, CacheStrategyL2, CacheStrategyL3, CacheStrategyAll:
		return true
	default:
		return false
	}
}

// FromStringCacheStrategy converts string to CacheStrategy
func FromStringCacheStrategy(s string) (CacheStrategy, error) {
	cs := CacheStrategy(strings.ToLower(s))
	if !cs.Valid() {
		return "", fmt.Errorf("invalid cache strategy: %s", s)
	}
	return cs, nil
}

//Personal.AI order the ending
