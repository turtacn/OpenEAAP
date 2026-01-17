package domain_test

import (
	"testing"

	"github.com/openeeap/openeeap/internal/domain/model"
	"github.com/stretchr/testify/assert"
)

// TestModelMetrics tests the ModelMetrics type
func TestModelMetrics(t *testing.T) {
	t.Run("Create ModelMetrics with all fields", func(t *testing.T) {
		metrics := &model.ModelMetrics{
			TotalInferences: 10000,
			AvgLatencyMs:    120.5,
			ErrorRate:       0.02,
			P95LatencyMs:    250.0,
			P99LatencyMs:    350.0,
			TokensPerSecond: 85.3,
		}

		assert.Equal(t, int64(10000), metrics.TotalInferences)
		assert.InDelta(t, 120.5, metrics.AvgLatencyMs, 0.01)
		assert.InDelta(t, 0.02, metrics.ErrorRate, 0.001)
		assert.InDelta(t, 250.0, metrics.P95LatencyMs, 0.01)
		assert.InDelta(t, 350.0, metrics.P99LatencyMs, 0.01)
		assert.InDelta(t, 85.3, metrics.TokensPerSecond, 0.01)
	})

	t.Run("Empty ModelMetrics", func(t *testing.T) {
		metrics := &model.ModelMetrics{}

		assert.Equal(t, int64(0), metrics.TotalInferences)
		assert.Equal(t, float64(0), metrics.AvgLatencyMs)
		assert.Equal(t, float64(0), metrics.ErrorRate)
	})

	t.Run("Calculate success rate from error rate", func(t *testing.T) {
		metrics := &model.ModelMetrics{
			TotalInferences: 1000,
			ErrorRate:       0.05, // 5% error rate
		}

		successRate := 1.0 - metrics.ErrorRate
		assert.InDelta(t, 0.95, successRate, 0.001) // 95% success rate
	})

	t.Run("Validate latency percentiles", func(t *testing.T) {
		metrics := &model.ModelMetrics{
			AvgLatencyMs: 100.0,
			P95LatencyMs: 200.0,
			P99LatencyMs: 300.0,
		}

		// P99 should be >= P95 should be >= Average
		assert.True(t, metrics.P99LatencyMs >= metrics.P95LatencyMs)
		assert.True(t, metrics.P95LatencyMs >= metrics.AvgLatencyMs)
	})

	t.Run("Calculate inferences per second from tokens per second", func(t *testing.T) {
		metrics := &model.ModelMetrics{
			TotalInferences: 1000,
			TokensPerSecond: 100.0,
		}

		// Assuming average of 50 tokens per inference
		avgTokensPerInference := 50.0
		expectedInferencesPerSecond := metrics.TokensPerSecond / avgTokensPerInference

		assert.InDelta(t, 2.0, expectedInferencesPerSecond, 0.01)
	})
}

// TestModelMetricsEdgeCases tests edge cases
func TestModelMetricsEdgeCases(t *testing.T) {
	t.Run("Zero error rate indicates perfect performance", func(t *testing.T) {
		metrics := &model.ModelMetrics{
			TotalInferences: 1000,
			ErrorRate:       0.0,
		}

		assert.Equal(t, 0.0, metrics.ErrorRate)
	})

	t.Run("High error rate", func(t *testing.T) {
		metrics := &model.ModelMetrics{
			TotalInferences: 100,
			ErrorRate:       0.50, // 50% error rate
		}

		assert.Equal(t, 0.50, metrics.ErrorRate)
		assert.True(t, metrics.ErrorRate > 0.1, "High error rate should be flagged")
	})

	t.Run("Very low latency", func(t *testing.T) {
		metrics := &model.ModelMetrics{
			AvgLatencyMs: 10.0,
			P95LatencyMs: 15.0,
			P99LatencyMs: 20.0,
		}

		assert.True(t, metrics.AvgLatencyMs < 100, "Low latency model")
	})

	t.Run("High throughput model", func(t *testing.T) {
		metrics := &model.ModelMetrics{
			TokensPerSecond: 500.0,
		}

		assert.True(t, metrics.TokensPerSecond > 100, "High throughput model")
	})
}

// TestModelMetricsComparison tests comparing metrics between models
func TestModelMetricsComparison(t *testing.T) {
	t.Run("Compare two models", func(t *testing.T) {
		model1 := &model.ModelMetrics{
			AvgLatencyMs: 100.0,
			ErrorRate:    0.01,
		}

		model2 := &model.ModelMetrics{
			AvgLatencyMs: 150.0,
			ErrorRate:    0.02,
		}

		// Model1 is better (lower latency and lower error rate)
		assert.True(t, model1.AvgLatencyMs < model2.AvgLatencyMs)
		assert.True(t, model1.ErrorRate < model2.ErrorRate)
	})
}

// BenchmarkModelMetricsCreation benchmarks metrics creation
func BenchmarkModelMetricsCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = &model.ModelMetrics{
			TotalInferences: 10000,
			AvgLatencyMs:    120.5,
			ErrorRate:       0.02,
			P95LatencyMs:    250.0,
			P99LatencyMs:    350.0,
			TokensPerSecond: 85.3,
		}
	}
}
