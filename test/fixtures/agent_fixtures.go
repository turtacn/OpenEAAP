// test/fixtures/agent_fixtures.go
package fixtures

import (
	"time"

	"github.com/google/uuid"
	"github.com/openeeap/openeeap/internal/domain/agent"
)

// AgentFixtures 提供 Agent 相关的测试数据
type AgentFixtures struct{}

// NewAgentFixtures 创建 Agent 测试数据提供器
func NewAgentFixtures() *AgentFixtures {
	return &AgentFixtures{}
}

// DefaultAgent 返回默认的 Agent 实体
func (f *AgentFixtures) DefaultAgent() *agent.Agent {
	now := time.Now()
	return &agent.Agent{
		ID:          uuid.New().String(),
		Name:        "test-security-analyst",
		Description: "测试用安全分析 Agent",
		RuntimeType: agent.RuntimeTypeLLM,
		Config: agent.AgentConfig{
			SystemPrompt: "You are a security analyst agent.",
			ModelParams: agent.ModelParams{
				Provider:    "openai",
				ModelName:   "gpt-4",
				Temperature: 0.7,
				MaxTokens:   2000,
			},
			Memory: agent.MemoryConfig{
				Type:       "short_term",
				MaxSize:    100,
				Persistent: false,
			},
			Tools: []agent.ToolConfig{
				{
					Name:        "threat_intel_query",
					Description: "查询威胁情报数据库",
					Enabled:     true,
				},
				{
					Name:        "log_analyzer",
					Description: "分析安全日志",
					Enabled:     true,
				},
			},
			Limits: agent.ExecutionLimits{
				MaxExecutionTime:        300,
				MaxIterations:           10,
				MaxTokensPerExecution:   5000,
				MaxConcurrentExecutions: 1,
			},
		},
		Status:    agent.AgentStatusActive,
		OwnerID:   "test-user",
		Version:   "1.0.0",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// MinimalAgent 返回最小配置的 Agent
func (f *AgentFixtures) MinimalAgent() *agent.Agent {
	now := time.Now()
	return &agent.Agent{
		ID:          uuid.New().String(),
		Name:        "minimal-agent",
		Description: "最小配置 Agent",
		RuntimeType: agent.RuntimeTypeLLM,
		Config: agent.AgentConfig{
			SystemPrompt: "You are a helpful assistant.",
			ModelParams: agent.ModelParams{
				Provider:  "openai",
				ModelName: "gpt-3.5-turbo",
			},
			Limits: agent.ExecutionLimits{
				MaxExecutionTime: 60,
				MaxIterations:    5,
			},
		},
		Status:    agent.AgentStatusInactive,
		OwnerID:   "test-user",
		Version:   "1.0.0",
		CreatedAt: now,
		UpdatedAt: now,
	}
}
