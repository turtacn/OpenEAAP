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
		ModelName: "test-security-analyst",
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
					ModelName: "threat_intel_query",
					Description: "查询威胁情报数据库",
					Enabled:     true,
				},
				{
					ModelName: "log_analyzer",
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
		Version: "1.0.0",
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
		ModelName: "minimal-agent",
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
		Version: "1.0.0",
		Status:    agent.AgentStatusInactive,
		OwnerID:   "test-user",
		Version:   "1.0.0",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// LangChainAgent 返回使用 LangChain 运行时的 Agent
func (f *AgentFixtures) LangChainAgent() *agent.Agent {
	now := time.Now()
	return &agent.Agent{
		ID:          uuid.New().String(),
		ModelName: "langchain-analyst",
		Description: "基于 LangChain 的分析 Agent",
		RuntimeType: agent.RuntimeTypeHybrid,
		Config: agent.AgentConfig{
			ModelParams: agent.ModelParams{
				Provider:    "anthropic",
				ModelName: "claude-3-opus",
				Temperature: 0.5,
				MaxTokens:   4000,
			},
			Memory: agent.MemoryConfig{
				Type:     "buffer_window",
				MaxSize: 20,
			},
			Tools: []agent.ToolConfig{
				{
					ModelName: "web_search",
					Description: "网络搜索工具",
					Enabled:     true,
					Parameters: map[string]interface{}{
						"max_results": 5,
					},
				},
			},
			Limits: agent.ExecutionLimits{
				MaxExecutionTime: 600,
				MaxIterations: 5},
		},
		Version: "1.0.0",
		Status:    agent.AgentStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		OwnerID: "test-user",
	}
}

// RAGAgent 返回启用 RAG 的 Agent
func (f *AgentFixtures) RAGAgent() *agent.Agent {
	now := time.Now()
	return &agent.Agent{
		ID:          uuid.New().String(),
		ModelName: "rag-knowledge-agent",
		Description: "启用 RAG 的知识库 Agent",
		RuntimeType: agent.RuntimeTypeLLM,
		Config: agent.AgentConfig{
			ModelParams: agent.ModelParams{
				Provider:    "openai",
				ModelName: "gpt-4-turbo",
				Temperature: 0.3,
				MaxTokens:   3000,
			},
			Memory: agent.MemoryConfig{
				Type:     "conversation",
				MaxSize: 15,
			},
			RAG: &agent.RAGConfig{
				Enabled:             true,
				VectorStore:         "milvus",
				CollectionName:      "security_docs",
				TopK:                5,
				SimilarityThreshold: 0.75,
				Reranker: agent.RerankerConfig{
					Enabled: true,
					Model:   "cross-encoder",
				},
			},
			Tools: []agent.ToolConfig{
				{
					ModelName: "document_search",
					Description: "文档检索工具",
					Enabled:     true,
				},
			},
			Limits: agent.ExecutionLimits{
				MaxExecutionTime: 450,
				MaxIterations: 3},
		},
		Version: "1.0.0",
		Status:    agent.AgentStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		OwnerID: "test-user",
	}
}

// MultiToolAgent 返回多工具 Agent
func (f *AgentFixtures) MultiToolAgent() *agent.Agent {
	now := time.Now()
	return &agent.Agent{
		ID:          uuid.New().String(),
		ModelName: "multi-tool-agent",
		Description: "集成多种工具的 Agent",
		RuntimeType: agent.RuntimeTypeLLM,
		Config: agent.AgentConfig{
			ModelParams: agent.ModelParams{
				Provider:    "openai",
				ModelName: "gpt-4",
				Temperature: 0.6,
				MaxTokens:   2500,
			},
			Memory: agent.MemoryConfig{
				Type:     "summary",
				MaxSize: 30,
			},
			Tools: []agent.ToolConfig{
				{
					ModelName: "sql_query",
					Description: "SQL 数据库查询",
					Enabled:     true,
					Parameters: map[string]interface{}{
						"database": "security_db",
						"readonly": true,
					},
				},
				{
					ModelName: "api_call",
					Description: "调用外部 API",
					Enabled:     true,
					Parameters: map[string]interface{}{
						"base_url": "https://api.example.com",
						"timeout":  30,
					},
				},
				{
					ModelName: "code_executor",
					Description: "执行 Python 代码",
					Enabled:     false, // 默认禁用
					Parameters: map[string]interface{}{
						"sandbox": true,
					},
				},
			},
			Limits: agent.ExecutionLimits{
				MaxExecutionTime: 900,
				MaxIterations: 5},
		},
		Version: "1.0.0",
		Status:    agent.AgentStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		OwnerID: "test-user",
	}
}

// InactiveAgent 返回未激活的 Agent
func (f *AgentFixtures) InactiveAgent() *agent.Agent {
	now := time.Now()
	return &agent.Agent{
		ID:          uuid.New().String(),
		ModelName: "inactive-agent",
		Description: "未激活的 Agent",
		RuntimeType: agent.RuntimeTypeLLM,
		Config: agent.AgentConfig{
			ModelParams: agent.ModelParams{
				Provider: "openai",
				ModelName: "gpt-3.5-turbo",
			},
			Limits: agent.ExecutionLimits{
				MaxExecutionTime: 120,
				MaxIterations: 2,
			},
		},
		Version: "1.0.0",
		Status:    agent.AgentStatusInactive,
		CreatedAt: now,
		UpdatedAt: now,
		OwnerID: "test-user",
	}
}

// AgentWithCustomConfig 返回自定义配置的 Agent
func (f *AgentFixtures) AgentWithCustomConfig(config agent.AgentConfig) *agent.Agent {
	now := time.Now()
	return &agent.Agent{
		ID:          uuid.New().String(),
		ModelName: "custom-agent",
		Description: "自定义配置 Agent",
		RuntimeType: agent.RuntimeTypeLLM,
		Config:      config,
		Version: "1.0.0",
		Status:      agent.AgentStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
		OwnerID: "test-user",
	}
}

// BatchAgents 返回一批测试 Agent
func (f *AgentFixtures) BatchAgents(count int) []*agent.Agent {
	agents := make([]*agent.Agent, count)
	now := time.Now()

	for i := 0; i < count; i++ {
		agents[i] = &agent.Agent{
			ID:          uuid.New().String(),
			Name:        uuid.New().String()[:8] + "-agent",
			Description: "批量测试 Agent",
			RuntimeType: agent.RuntimeTypeLLM,
			Config: agent.AgentConfig{
				ModelParams: agent.ModelParams{
					Provider: "openai",
					ModelName: "gpt-3.5-turbo",
				},
				Limits: agent.ExecutionLimits{
					MaxExecutionTime: 60,
					MaxIterations: 1,
				},
			},
			Version: "1.0.0",
		Status:    agent.AgentStatusActive,
			CreatedAt: now.Add(time.Duration(i) * time.Second),
			UpdatedAt: now.Add(time.Duration(i) * time.Second),
			OwnerID: "test-user",
		}
	}

	return agents
}

// CreateAgentRequest 返回默认的创建请求
func (f *AgentFixtures) CreateAgentRequest() map[string]interface{} {
	return map[string]interface{}{
		"name":         "new-test-agent",
		"description":  "新建测试 Agent",
		"runtime_type": "native",
		"config": map[string]interface{}{
			"model": map[string]interface{}{
				"provider":    "openai",
				"name":        "gpt-4",
				"temperature": 0.7,
				"max_tokens":  2000,
			},
			"memory": map[string]interface{}{
				"type":      "conversation",
				"max_turns": 10,
			},
			"tools": []map[string]interface{}{
				{
					"name":        "calculator",
					"description": "数学计算工具",
					"enabled":     true,
				},
			},
			"constraints": map[string]interface{}{
				"max_execution_time": 300,
				"max_retries":        3,
				"timeout_seconds":    30,
			},
		},
	}
}

// UpdateAgentRequest 返回默认的更新请求
func (f *AgentFixtures) UpdateAgentRequest() map[string]interface{} {
	return map[string]interface{}{
		"name":        "updated-agent",
		"description": "更新后的 Agent",
		"config": map[string]interface{}{
			"model": map[string]interface{}{
				"temperature": 0.5,
				"max_tokens":  3000,
			},
		},
	}
}

// ExecuteAgentRequest 返回默认的执行请求
func (f *AgentFixtures) ExecuteAgentRequest() map[string]interface{} {
	return map[string]interface{}{
		"input": "分析这个安全告警: 检测到异常登录行为",
		"context": map[string]interface{}{
			"user_id":   "user-123",
			"timestamp": time.Now().Unix(),
			"severity":  "high",
		},
		"stream": false,
	}
}

// ExecuteAgentStreamRequest 返回流式执行请求
func (f *AgentFixtures) ExecuteAgentStreamRequest() map[string]interface{} {
	return map[string]interface{}{
		"input": "详细分析这个复杂的安全事件",
		"context": map[string]interface{}{
			"event_id": "evt-456",
			"source":   "firewall",
		},
		"stream": true,
	}
}

// InvalidAgentRequest 返回无效的创建请求（用于测试验证）
func (f *AgentFixtures) InvalidAgentRequest() map[string]interface{} {
	return map[string]interface{}{
		"name": "", // 空名称，应该验证失败
		"config": map[string]interface{}{
			"model": map[string]interface{}{
				"provider": "invalid-provider", // 无效的提供商
			},
		},
	}
}

// AgentValueObjects 返回测试用值对象
func (f *AgentFixtures) AgentValueObjects() struct {
	ValidConfig   agent.AgentConfig
	InvalidConfig agent.AgentConfig
	MinimalConfig agent.AgentConfig
} {
	return struct {
		ValidConfig   agent.AgentConfig
		InvalidConfig agent.AgentConfig
		MinimalConfig agent.AgentConfig
	}{
		ValidConfig: agent.AgentConfig{
			ModelParams: agent.ModelParams{
				Provider:    "openai",
				ModelName: "gpt-4",
				Temperature: 0.7,
				MaxTokens:   2000,
			},
			Memory: agent.MemoryConfig{
				Type:     "conversation",
				MaxSize: 10,
			},
			Limits: agent.ExecutionLimits{
				MaxExecutionTime: 300,
				MaxIterations: 3},
		},
		InvalidConfig: agent.AgentConfig{
			ModelParams: agent.ModelParams{
				Provider:    "",  // 无效：空提供商
				Temperature: 2.0, // 无效：温度超出范围
			},
		},
		MinimalConfig: agent.AgentConfig{
			ModelParams: agent.ModelParams{
				Provider: "openai",
				ModelName: "gpt-3.5-turbo",
			},
			Limits: agent.ExecutionLimits{
				MaxExecutionTime: 60,
				MaxIterations: 1,
			},
		},
	}
}

// AgentExecutionResults 返回测试用执行结果
func (f *AgentFixtures) AgentExecutionResults() struct {
	Success agent.ExecutionResult
	Failure agent.ExecutionResult
	Timeout agent.ExecutionResult
} {
	now := time.Now()

	return struct {
		Success agent.ExecutionResult
		Failure agent.ExecutionResult
		Timeout agent.ExecutionResult
	}{
		Success: agent.ExecutionResult{
			ExecutionID: uuid.New().String(),
			AgentID:     uuid.New().String(),
			Status:      agent.ExecutionStatusSuccess,
			Output:      "分析完成：发现 3 个可疑活动",
			Metadata: map[string]interface{}{
				"tokens_used":    1500,
				"execution_time": 2.5,
				"tools_called":   []string{"threat_intel_query", "log_analyzer"},
			},
			StartedAt:  now.Add(-3 * time.Second),
			FinishedAt: &now,
			Error:      nil,
		},
		Failure: agent.ExecutionResult{
			ExecutionID: uuid.New().String(),
			AgentID:     uuid.New().String(),
			Status:      agent.ExecutionStatusFailed,
			Output:      "",
			Metadata: map[string]interface{}{
				"retry_count": 3,
			},
			StartedAt:  now.Add(-10 * time.Second),
			FinishedAt: &now,
			Error:      stringPtr("工具调用失败: connection timeout"),
		},
		Timeout: agent.ExecutionResult{
			ExecutionID: uuid.New().String(),
			AgentID:     uuid.New().String(),
			Status:      agent.ExecutionStatusTimeout,
			Output:      "",
			Metadata: map[string]interface{}{
				"timeout_seconds": 30,
			},
			StartedAt:  now.Add(-35 * time.Second),
			FinishedAt: &now,
			Error:      stringPtr("execution timeout after 30 seconds"),
		},
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}

//Personal.AI order the ending
