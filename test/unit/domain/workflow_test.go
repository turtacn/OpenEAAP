package domain_test

import (
	"testing"

	"github.com/openeeap/openeeap/internal/domain/workflow"
	"github.com/stretchr/testify/assert"
)

// TestWorkflowStatistics tests the WorkflowStatistics type
func TestWorkflowStatistics(t *testing.T) {
	t.Run("Create WorkflowStatistics with all fields", func(t *testing.T) {
		stats := &workflow.WorkflowStatistics{
			TotalExecutions:      1000,
			SuccessfulExecutions: 950,
			FailedExecutions:     50,
			AvgExecutionTimeMs:   5000,
			SuccessRate:          0.95,
		}

		assert.Equal(t, int64(1000), stats.TotalExecutions)
		assert.Equal(t, int64(950), stats.SuccessfulExecutions)
		assert.Equal(t, int64(50), stats.FailedExecutions)
		assert.Equal(t, int64(5000), stats.AvgExecutionTimeMs)
		assert.InDelta(t, 0.95, stats.SuccessRate, 0.001)
	})

	t.Run("Empty WorkflowStatistics", func(t *testing.T) {
		stats := &workflow.WorkflowStatistics{}

		assert.Equal(t, int64(0), stats.TotalExecutions)
		assert.Equal(t, int64(0), stats.SuccessfulExecutions)
		assert.Equal(t, int64(0), stats.FailedExecutions)
		assert.Equal(t, float64(0), stats.SuccessRate)
	})

	t.Run("Calculate success rate", func(t *testing.T) {
		stats := &workflow.WorkflowStatistics{
			TotalExecutions:      100,
			SuccessfulExecutions: 95,
			FailedExecutions:     5,
		}

		calculatedRate := float64(stats.SuccessfulExecutions) / float64(stats.TotalExecutions)
		assert.InDelta(t, 0.95, calculatedRate, 0.01)
	})

	t.Run("Verify execution counts sum to total", func(t *testing.T) {
		stats := &workflow.WorkflowStatistics{
			TotalExecutions:      100,
			SuccessfulExecutions: 75,
			FailedExecutions:     25,
		}

		sum := stats.SuccessfulExecutions + stats.FailedExecutions
		assert.Equal(t, stats.TotalExecutions, sum)
	})
}

// TestExecutionStatus tests the ExecutionStatus type
func TestExecutionStatus(t *testing.T) {
	t.Run("All execution statuses defined", func(t *testing.T) {
		statuses := []workflow.ExecutionStatus{
			workflow.ExecutionStatusPending,
			workflow.ExecutionStatusRunning,
			workflow.ExecutionStatusCompleted,
			workflow.ExecutionStatusFailed,
			workflow.ExecutionStatusCancelled,
		}

		assert.Len(t, statuses, 5)
		assert.NotEmpty(t, workflow.ExecutionStatusPending)
		assert.NotEmpty(t, workflow.ExecutionStatusRunning)
		assert.NotEmpty(t, workflow.ExecutionStatusCompleted)
		assert.NotEmpty(t, workflow.ExecutionStatusFailed)
		assert.NotEmpty(t, workflow.ExecutionStatusCancelled)
	})

	t.Run("Status values are unique", func(t *testing.T) {
		statusMap := make(map[workflow.ExecutionStatus]bool)

		statuses := []workflow.ExecutionStatus{
			workflow.ExecutionStatusPending,
			workflow.ExecutionStatusRunning,
			workflow.ExecutionStatusCompleted,
			workflow.ExecutionStatusFailed,
			workflow.ExecutionStatusCancelled,
		}

		for _, status := range statuses {
			assert.False(t, statusMap[status], "Status should be unique: %s", status)
			statusMap[status] = true
		}

		assert.Len(t, statusMap, 5)
	})

	t.Run("Status string values", func(t *testing.T) {
		assert.Equal(t, workflow.ExecutionStatus("pending"), workflow.ExecutionStatusPending)
		assert.Equal(t, workflow.ExecutionStatus("running"), workflow.ExecutionStatusRunning)
		assert.Equal(t, workflow.ExecutionStatus("completed"), workflow.ExecutionStatusCompleted)
		assert.Equal(t, workflow.ExecutionStatus("failed"), workflow.ExecutionStatusFailed)
		assert.Equal(t, workflow.ExecutionStatus("cancelled"), workflow.ExecutionStatusCancelled)
	})
}

// TestExecutionStatusTransitions tests valid status transitions
func TestExecutionStatusTransitions(t *testing.T) {
	t.Run("Valid transition: Pending to Running", func(t *testing.T) {
		status := workflow.ExecutionStatusPending
		newStatus := workflow.ExecutionStatusRunning

		assert.NotEqual(t, status, newStatus)
		assert.True(t, isValidTransition(status, newStatus))
	})

	t.Run("Valid transition: Running to Completed", func(t *testing.T) {
		status := workflow.ExecutionStatusRunning
		newStatus := workflow.ExecutionStatusCompleted

		assert.True(t, isValidTransition(status, newStatus))
	})

	t.Run("Valid transition: Running to Failed", func(t *testing.T) {
		status := workflow.ExecutionStatusRunning
		newStatus := workflow.ExecutionStatusFailed

		assert.True(t, isValidTransition(status, newStatus))
	})

	t.Run("Valid transition: Running to Cancelled", func(t *testing.T) {
		status := workflow.ExecutionStatusRunning
		newStatus := workflow.ExecutionStatusCancelled

		assert.True(t, isValidTransition(status, newStatus))
	})

	t.Run("Invalid transition: Completed to Running", func(t *testing.T) {
		status := workflow.ExecutionStatusCompleted
		newStatus := workflow.ExecutionStatusRunning

		assert.False(t, isValidTransition(status, newStatus))
	})
}

// Helper function to validate status transitions
func isValidTransition(from, to workflow.ExecutionStatus) bool {
	// Define valid transition rules
	validTransitions := map[workflow.ExecutionStatus][]workflow.ExecutionStatus{
		workflow.ExecutionStatusPending: {
			workflow.ExecutionStatusRunning,
			workflow.ExecutionStatusCancelled,
		},
		workflow.ExecutionStatusRunning: {
			workflow.ExecutionStatusCompleted,
			workflow.ExecutionStatusFailed,
			workflow.ExecutionStatusCancelled,
		},
		workflow.ExecutionStatusCompleted: {},  // Terminal state
		workflow.ExecutionStatusFailed:    {},  // Terminal state
		workflow.ExecutionStatusCancelled: {},  // Terminal state
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, status := range allowed {
		if status == to {
			return true
		}
	}

	return false
}

// TestWorkflowStatisticsEdgeCases tests edge cases
func TestWorkflowStatisticsEdgeCases(t *testing.T) {
	t.Run("Zero executions", func(t *testing.T) {
		stats := &workflow.WorkflowStatistics{
			TotalExecutions: 0,
		}

		assert.Equal(t, int64(0), stats.TotalExecutions)
		// Avoid division by zero when calculating success rate
		if stats.TotalExecutions > 0 {
			_ = float64(stats.SuccessfulExecutions) / float64(stats.TotalExecutions)
		}
	})

	t.Run("Perfect success rate", func(t *testing.T) {
		stats := &workflow.WorkflowStatistics{
			TotalExecutions:      100,
			SuccessfulExecutions: 100,
			FailedExecutions:     0,
			SuccessRate:          1.0,
		}

		assert.Equal(t, 1.0, stats.SuccessRate)
	})

	t.Run("Zero success rate", func(t *testing.T) {
		stats := &workflow.WorkflowStatistics{
			TotalExecutions:      100,
			SuccessfulExecutions: 0,
			FailedExecutions:     100,
			SuccessRate:          0.0,
		}

		assert.Equal(t, 0.0, stats.SuccessRate)
	})

	t.Run("Very long execution time", func(t *testing.T) {
		stats := &workflow.WorkflowStatistics{
			AvgExecutionTimeMs: 3600000, // 1 hour in ms
		}

		assert.True(t, stats.AvgExecutionTimeMs > 60000, "Long running workflow")
	})
}

// BenchmarkWorkflowStatisticsCreation benchmarks statistics creation
func BenchmarkWorkflowStatisticsCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = &workflow.WorkflowStatistics{
			TotalExecutions:      1000,
			SuccessfulExecutions: 950,
			FailedExecutions:     50,
			AvgExecutionTimeMs:   5000,
			SuccessRate:          0.95,
		}
	}
}
