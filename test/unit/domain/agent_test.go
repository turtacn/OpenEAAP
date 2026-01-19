package domain_test

import (
	"testing"
	"time"

	"github.com/openeeap/openeeap/internal/domain/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentStatistics tests the AgentStatistics type
func TestAgentStatistics(t *testing.T) {
	t.Run("Create AgentStatistics", func(t *testing.T) {
		stats := &agent.AgentStatistics{
			TotalCount:    100,
			ActiveCount:   75,
			ArchivedCount: 25,
			ByStatus: map[string]int64{
				"active":   75,
				"inactive": 15,
				"archived": 10,
			},
			ByRuntimeType: map[string]int64{
				"native":    60,
				"langchain": 30,
				"custom":    10,
			},
		}

		assert.Equal(t, int64(100), stats.TotalCount)
		assert.Equal(t, int64(75), stats.ActiveCount)
		assert.Equal(t, 3, len(stats.ByStatus))
		assert.Equal(t, 3, len(stats.ByRuntimeType))
	})

	t.Run("Empty AgentStatistics", func(t *testing.T) {
		stats := &agent.AgentStatistics{}

		assert.Equal(t, int64(0), stats.TotalCount)
		assert.Nil(t, stats.ByStatus)
		assert.Nil(t, stats.ByRuntimeType)
	})

	t.Run("Calculate statistics aggregates", func(t *testing.T) {
		stats := &agent.AgentStatistics{
			TotalCount:    100,
			ActiveCount:   75,
			ArchivedCount: 10,
			ByStatus: map[string]int64{
				"active":   75,
				"inactive": 15,
				"archived": 10,
			},
		}

		// Verify totals match
		statusSum := int64(0)
		for _, count := range stats.ByStatus {
			statusSum += count
		}
		assert.Equal(t, stats.TotalCount, statusSum)
	})
}

// TestExecutionStats tests execution statistics
func TestExecutionStats(t *testing.T) {
	t.Run("Initialize empty stats", func(t *testing.T) {
		stats := &agent.ExecutionStats{}

		assert.Equal(t, int64(0), stats.TotalExecutions)
		assert.Equal(t, int64(0), stats.SuccessfulExecutions)
		assert.Equal(t, int64(0), stats.FailedExecutions)
	})

	t.Run("Calculate success rate", func(t *testing.T) {
		stats := &agent.ExecutionStats{
			TotalExecutions:      100,
			SuccessfulExecutions: 95,
			FailedExecutions:     5,
		}

		successRate := float64(stats.SuccessfulExecutions) / float64(stats.TotalExecutions)
		assert.InDelta(t, 0.95, successRate, 0.01)
	})

	t.Run("Track performance metrics", func(t *testing.T) {
		stats := &agent.ExecutionStats{
			TotalExecutions:    10,
			AvgExecutionTimeMs: 1500,
			TotalTokensUsed:    5000,
		}

		avgTokensPerExecution := stats.TotalTokensUsed / stats.TotalExecutions
		assert.Equal(t, int64(500), avgTokensPerExecution)
	})
}

// TestAgentCreation tests Agent entity creation
func TestAgentCreation(t *testing.T) {
	t.Run("Create valid agent", func(t *testing.T) {
		a := agent.NewAgent(
			"TestAgent",
			"Test agent description",
			agent.RuntimeTypeLLM,
			"user-123",
		)

		require.NotNil(t, a)
		assert.NotEmpty(t, a.ID)
		assert.Equal(t, "TestAgent", a.Name)
		assert.Equal(t, "Test agent description", a.Description)
		assert.Equal(t, agent.RuntimeTypeLLM, a.RuntimeType)
		assert.Equal(t, "user-123", a.OwnerID)
		assert.Equal(t, agent.AgentStatusDraft, a.Status)
		assert.NotZero(t, a.CreatedAt)
		assert.NotZero(t, a.UpdatedAt)
	})

	t.Run("Agent has default values", func(t *testing.T) {
		a := agent.NewAgent(
			"TestAgent",
			"Description",
			agent.RuntimeTypeLLM,
			"user-123",
		)

		assert.NotEmpty(t, a.Version)
		assert.NotNil(t, a.Tags)
		assert.NotNil(t, a.Capabilities)
	})
}

// TestAgentValidation tests Agent validation logic
func TestAgentValidation(t *testing.T) {
	t.Run("Valid agent passes validation", func(t *testing.T) {
		a := agent.NewAgent(
			"ValidAgent",
			"Valid description",
			agent.RuntimeTypeLLM,
			"user-123",
		)

		err := a.Validate()
		assert.NoError(t, err)
	})

	t.Run("Agent without name fails validation", func(t *testing.T) {
		a := agent.NewAgent(
			"",
			"Description",
			agent.RuntimeTypeLLM,
			"user-123",
		)

		err := a.Validate()
		assert.Error(t, err)
	})

	t.Run("Agent without owner fails validation", func(t *testing.T) {
		a := agent.NewAgent(
			"TestAgent",
			"Description",
			agent.RuntimeTypeLLM,
			"",
		)

		err := a.Validate()
		assert.Error(t, err)
	})
}

// TestAgentClone tests the Clone method
func TestAgentClone(t *testing.T) {
	t.Run("Clone creates independent copy", func(t *testing.T) {
		original := agent.NewAgent(
			"OriginalAgent",
			"Original description",
			agent.RuntimeTypeLLM,
			"user-123",
		)
		original.Tags = []string{"tag1", "tag2"}

		cloned := original.Clone()

		require.NotNil(t, cloned)
		assert.NotEqual(t, original.ID, cloned.ID, "Clone should have different ID")
		assert.Equal(t, original.Name, cloned.Name)
		assert.Equal(t, original.Description, cloned.Description)

		// Verify deep copy
		cloned.Tags[0] = "modified"
		assert.NotEqual(t, original.Tags[0], cloned.Tags[0])
	})
}

// BenchmarkAgentCreation benchmarks agent creation
func BenchmarkAgentCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		agent.NewAgent(
			"TestAgent",
			"Test description",
			agent.RuntimeTypeLLM,
			"user-123",
		)
	}
}

// BenchmarkAgentValidation benchmarks validation
func BenchmarkAgentValidation(b *testing.B) {
	a := agent.NewAgent(
		"TestAgent",
		"Test description",
		agent.RuntimeTypeLLM,
		"user-123",
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.Validate()
	}
}
