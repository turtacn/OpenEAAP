// Package vllm provides a client for interacting with vLLM inference engine.
// It supports synchronous, streaming, and batch inference with advanced features
// like KV-Cache sharing and speculative decoding.
package vllm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/pkg/errors"
)

// VLLMClient provides a client for vLLM inference engine
type VLLMClient struct {
	logger           logging.Logger
	metricsCollector metrics.Collector
	baseURL          string
	httpClient       *http.Client
	apiKey           string
	timeout          time.Duration
	maxRetries       int
}

// VLLMConfig contains configuration for vLLM client
type VLLMConfig struct {
	BaseURL    string
	APIKey     string
	Timeout    time.Duration
	MaxRetries int
}

// CompletionRequest represents a request to vLLM completion API
type CompletionRequest struct {
	Model             string                 `json:"model"`
	Prompt            interface{}            `json:"prompt"` // string or []string
	MaxTokens         int                    `json:"max_tokens,omitempty"`
	Temperature       float64                `json:"temperature,omitempty"`
	TopP              float64                `json:"top_p,omitempty"`
	N                 int                    `json:"n,omitempty"`
	Stream            bool                   `json:"stream,omitempty"`
	LogProbs          *int                   `json:"logprobs,omitempty"`
	Echo              bool                   `json:"echo,omitempty"`
	Stop              []string               `json:"stop,omitempty"`
	PresencePenalty   float64                `json:"presence_penalty,omitempty"`
	FrequencyPenalty  float64                `json:"frequency_penalty,omitempty"`
	BestOf            int                    `json:"best_of,omitempty"`
	UseBeamSearch     bool                   `json:"use_beam_search,omitempty"`

	// Advanced vLLM features
	UseKVCache        bool                   `json:"use_kv_cache,omitempty"`
	KVCacheSessionID  string                 `json:"kv_cache_session_id,omitempty"`
	EnableSpeculative bool                   `json:"enable_speculative,omitempty"`
	SpeculativeModel  string                 `json:"speculative_model,omitempty"`
	NumSpeculativeTokens int                 `json:"num_speculative_tokens,omitempty"`

	// Custom parameters
	ExtraBody         map[string]interface{} `json:"extra_body,omitempty"`
}

// CompletionResponse represents a response from vLLM completion API
type CompletionResponse struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Created int64               `json:"created"`
	Model   string              `json:"model"`
	Choices []CompletionChoice  `json:"choices"`
	Usage   CompletionUsage     `json:"usage"`
}

// CompletionChoice represents a completion choice
type CompletionChoice struct {
	Index        int                    `json:"index"`
	Text         string                 `json:"text"`
	LogProbs     *LogProbResult         `json:"logprobs,omitempty"`
	FinishReason string                 `json:"finish_reason"`
}

// LogProbResult contains log probability information
type LogProbResult struct {
	Tokens        []string             `json:"tokens"`
	TokenLogProbs []float64            `json:"token_logprobs"`
	TopLogProbs   []map[string]float64 `json:"top_logprobs,omitempty"`
	TextOffset    []int                `json:"text_offset"`
}

// CompletionUsage represents token usage statistics
type CompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk represents a streaming response chunk
type StreamChunk struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Created int64               `json:"created"`
	Model   string              `json:"model"`
	Choices []StreamChoice      `json:"choices"`
}

// StreamChoice represents a streaming choice
type StreamChoice struct {
	Index        int    `json:"index"`
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// BatchRequest represents a batch inference request
type BatchRequest struct {
	Requests []CompletionRequest `json:"requests"`
}

// BatchResponse represents a batch inference response
type BatchResponse struct {
	Responses []CompletionResponse `json:"responses"`
	Errors    []string             `json:"errors,omitempty"`
}

// NewVLLMClient creates a new vLLM client
func NewVLLMClient(logger logging.Logger, metricsCollector metrics.Collector, config *VLLMConfig) *VLLMClient {
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	return &VLLMClient{
		logger:           logger,
		metricsCollector: metricsCollector,
		baseURL:          strings.TrimSuffix(config.BaseURL, "/"),
		apiKey:           config.APIKey,
		timeout:          config.Timeout,
		maxRetries:       config.MaxRetries,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Complete performs synchronous completion inference
func (c *VLLMClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	startTime := time.Now()

	req.Stream = false

	endpoint := fmt.Sprintf("%s/v1/completions", c.baseURL)

	var resp *CompletionResponse
	var err error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Info(ctx, "retrying vLLM request", "attempt", attempt)
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		resp, err = c.doComplete(ctx, endpoint, req)
		if err == nil {
			break
		}

		if !c.isRetryableError(err) {
			break
		}
	}

	if err != nil {
		c.recordMetrics("complete", "error", time.Since(startTime))
		return nil, errors.Wrap(err, errors.CodeInternalError, "vLLM completion failed")
	}

	c.recordMetrics("complete", "success", time.Since(startTime))

	c.logger.Info(ctx, "vLLM completion completed",
		"model", req.Model,
		"tokens", resp.Usage.TotalTokens,
		"latency_ms", time.Since(startTime).Milliseconds(),
	)

	return resp, nil
}

// CompleteStream performs streaming completion inference
func (c *VLLMClient) CompleteStream(ctx context.Context, req *CompletionRequest) (<-chan *StreamChunk, error) {
	req.Stream = true

	endpoint := fmt.Sprintf("%s/v1/completions", c.baseURL)

	httpReq, err := c.buildRequest(ctx, endpoint, req)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to build request")
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "HTTP request failed")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, errors.New(errors.CodeInternalError,
			fmt.Sprintf("vLLM returned status %d: %s", resp.StatusCode, string(body)))
	}

	chunkChan := make(chan *StreamChunk, 10)

	go func() {
		defer resp.Body.Close()
		defer close(chunkChan)

		reader := bufio.NewReader(resp.Body)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					c.logger.Error(ctx, "stream read error", "error", err)
				}
				return
			}

			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue
			}

			// SSE format: "data: {...}"
			if bytes.HasPrefix(line, []byte("data: ")) {
				line = line[6:]
			}

			// Check for stream end marker
			if string(line) == "[DONE]" {
				return
			}

			var chunk StreamChunk
			if err := json.Unmarshal(line, &chunk); err != nil {
				c.logger.Warn(ctx, "failed to parse stream chunk", "error", err)
				continue
			}

			select {
			case chunkChan <- &chunk:
			case <-ctx.Done():
				return
			}
		}
	}()

	return chunkChan, nil
}

// CompleteBatch performs batch inference
func (c *VLLMClient) CompleteBatch(ctx context.Context, requests []CompletionRequest) (*BatchResponse, error) {
	startTime := time.Now()

	batchReq := &BatchRequest{
		Requests: requests,
	}

	endpoint := fmt.Sprintf("%s/v1/batch", c.baseURL)

	httpReq, err := c.buildRequest(ctx, endpoint, batchReq)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to build batch request")
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.recordMetrics("batch", "error", time.Since(startTime))
		return nil, errors.Wrap(err, errors.CodeInternalError, "HTTP request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.recordMetrics("batch", "error", time.Since(startTime))
		return nil, errors.New(errors.CodeInternalError,
			fmt.Sprintf("vLLM returned status %d: %s", resp.StatusCode, string(body)))
	}

	var batchResp BatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		c.recordMetrics("batch", "error", time.Since(startTime))
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to decode response")
	}

	c.recordMetrics("batch", "success", time.Since(startTime))

	c.logger.Info(ctx, "vLLM batch completed",
		"request_count", len(requests),
		"latency_ms", time.Since(startTime).Milliseconds(),
	)

	return &batchResp, nil
}

// CreateKVCacheSession creates a new KV-Cache session
func (c *VLLMClient) CreateKVCacheSession(ctx context.Context, sessionID string, prompt string) error {
	endpoint := fmt.Sprintf("%s/v1/kv_cache/sessions", c.baseURL)

	reqBody := map[string]interface{}{
		"session_id": sessionID,
		"prompt":     prompt,
	}

	httpReq, err := c.buildRequest(ctx, endpoint, reqBody)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to build request")
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "HTTP request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return errors.New(errors.CodeInternalError,
			fmt.Sprintf("failed to create KV-Cache session: %s", string(body)))
	}

	c.logger.Info(ctx, "KV-Cache session created", "session_id", sessionID)

	return nil
}

// DeleteKVCacheSession deletes a KV-Cache session
func (c *VLLMClient) DeleteKVCacheSession(ctx context.Context, sessionID string) error {
	endpoint := fmt.Sprintf("%s/v1/kv_cache/sessions/%s", c.baseURL, sessionID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to create request")
	}

	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "HTTP request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return errors.New(errors.CodeInternalError,
			fmt.Sprintf("failed to delete KV-Cache session: %s", string(body)))
	}

	c.logger.Info(ctx, "KV-Cache session deleted", "session_id", sessionID)

	return nil
}

// HealthCheck checks if vLLM is healthy
func (c *VLLMClient) HealthCheck(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/health", c.baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "failed to create request")
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return errors.Wrap(err, errors.CodeInternalError, "health check failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New(errors.CodeInternalError,
			fmt.Sprintf("vLLM unhealthy: status %d", resp.StatusCode))
	}

	return nil
}

// GetModels retrieves available models
func (c *VLLMClient) GetModels(ctx context.Context) ([]string, error) {
	endpoint := fmt.Sprintf("%s/v1/models", c.baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to create request")
	}

	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "HTTP request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.New(errors.CodeInternalError,
			fmt.Sprintf("failed to get models: %s", string(body)))
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, errors.CodeInternalError, "failed to decode response")
	}

	models := make([]string, len(result.Data))
	for i, model := range result.Data {
		models[i] = model.ID
	}

	return models, nil
}

// doComplete performs the actual completion request
func (c *VLLMClient) doComplete(ctx context.Context, endpoint string, req *CompletionRequest) (*CompletionResponse, error) {
	httpReq, err := c.buildRequest(ctx, endpoint, req)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vLLM returned status %d: %s", resp.StatusCode, string(body))
	}

	var completionResp CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completionResp); err != nil {
		return nil, err
	}

	return &completionResp, nil
}

// buildRequest builds an HTTP request with proper headers and body
func (c *VLLMClient) buildRequest(ctx context.Context, endpoint string, body interface{}) (*http.Request, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	return req, nil
}

// isRetryableError determines if an error is retryable
func (c *VLLMClient) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Retry on timeout, connection errors, and 5xx status codes
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "status 5")
}

// recordMetrics records operation metrics
func (c *VLLMClient) recordMetrics(operation, status string, latency time.Duration) {
	if c.metricsCollector == nil {
		return
	}

	c.metricsCollector.IncrementCounter("vllm_requests_total", 1,
		map[string]string{
			"operation": operation,
			"status":    status,
		})

	c.metricsCollector.RecordHistogram("vllm_request_latency_ms", float64(latency.Milliseconds()),
		map[string]string{
			"operation": operation,
		})
}

//Personal.AI order the ending
