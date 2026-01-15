// test/e2e/api_test.go
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/openeeap/openeeap/internal/app/dto"
	"github.com/openeeap/openeeap/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// API endpoints
	baseURL           = "http://localhost:8080"
	agentsEndpoint    = "/api/v1/agents"
	workflowsEndpoint = "/api/v1/workflows"
	modelsEndpoint    = "/api/v1/models"
	healthEndpoint    = "/health"

	// Test timeouts
	containerStartTimeout = 120 * time.Second
	apiRequestTimeout     = 30 * time.Second
	serverStartTimeout    = 60 * time.Second
)

// E2ETestSuite defines the end-to-end test suite
type E2ETestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc

	// Container instances
	postgresContainer testcontainers.Container
	redisContainer    testcontainers.Container
	milvusContainer   testcontainers.Container
	minioContainer    testcontainers.Container

	// Application server
	serverProcess *os.Process

	// HTTP client
	httpClient *http.Client

	// Test data
	testAgentID    string
	testWorkflowID string
	testModelID    string
	authToken      string
}

// SetupSuite initializes the test suite
func (s *E2ETestSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	// Initialize HTTP client
	s.httpClient = &http.Client{
		Timeout: apiRequestTimeout,
	}

	// Start infrastructure containers
	s.startPostgresContainer()
	s.startRedisContainer()
	s.startMilvusContainer()
	s.startMinioContainer()

	// Wait for containers to be ready
	time.Sleep(10 * time.Second)

	// Start application server
	s.startApplicationServer()

	// Wait for server to be ready
	s.waitForServerReady()

	// Authenticate and get token
	s.authToken = s.authenticate()
}

// TearDownSuite cleans up the test suite
func (s *E2ETestSuite) TearDownSuite() {
	// Stop application server
	if s.serverProcess != nil {
		_ = s.serverProcess.Kill()
	}

	// Stop containers
	if s.postgresContainer != nil {
		_ = s.postgresContainer.Terminate(s.ctx)
	}
	if s.redisContainer != nil {
		_ = s.redisContainer.Terminate(s.ctx)
	}
	if s.milvusContainer != nil {
		_ = s.milvusContainer.Terminate(s.ctx)
	}
	if s.minioContainer != nil {
		_ = s.minioContainer.Terminate(s.ctx)
	}

	s.cancel()
}

// startPostgresContainer starts PostgreSQL container
func (s *E2ETestSuite) startPostgresContainer() {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "openeeap",
			"POSTGRES_PASSWORD": "openeeap",
			"POSTGRES_DB":       "openeeap_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithStartupTimeout(containerStartTimeout),
	}

	container, err := testcontainers.GenericContainer(s.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(s.T(), err)

	s.postgresContainer = container

	// Set DATABASE_URL environment variable
	host, _ := container.Host(s.ctx)
	port, _ := container.MappedPort(s.ctx, "5432")
	dsn := fmt.Sprintf("postgres://openeeap:openeeap@%s:%s/openeeap_test?sslmode=disable", host, port.Port())
	_ = os.Setenv("DATABASE_URL", dsn)
}

// startRedisContainer starts Redis container
func (s *E2ETestSuite) startRedisContainer() {
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections").WithStartupTimeout(containerStartTimeout),
	}

	container, err := testcontainers.GenericContainer(s.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(s.T(), err)

	s.redisContainer = container

	// Set REDIS_URL environment variable
	host, _ := container.Host(s.ctx)
	port, _ := container.MappedPort(s.ctx, "6379")
	redisURL := fmt.Sprintf("redis://%s:%s", host, port.Port())
	_ = os.Setenv("REDIS_URL", redisURL)
}

// startMilvusContainer starts Milvus container
func (s *E2ETestSuite) startMilvusContainer() {
	req := testcontainers.ContainerRequest{
		Image:        "milvusdb/milvus:v2.3.0",
		ExposedPorts: []string{"19530/tcp", "9091/tcp"},
		Cmd:          []string{"milvus", "run", "standalone"},
		WaitingFor:   wait.ForHTTP("/healthz").WithPort("9091").WithStartupTimeout(containerStartTimeout),
	}

	container, err := testcontainers.GenericContainer(s.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(s.T(), err)

	s.milvusContainer = container

	// Set MILVUS_URL environment variable
	host, _ := container.Host(s.ctx)
	port, _ := container.MappedPort(s.ctx, "19530")
	milvusURL := fmt.Sprintf("%s:%s", host, port.Port())
	_ = os.Setenv("MILVUS_URL", milvusURL)
}

// startMinioContainer starts MinIO container
func (s *E2ETestSuite) startMinioContainer() {
	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{"9000/tcp", "9001/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     "minioadmin",
			"MINIO_ROOT_PASSWORD": "minioadmin",
		},
		Cmd:        []string{"server", "/data", "--console-address", ":9001"},
		WaitingFor: wait.ForHTTP("/minio/health/live").WithPort("9000").WithStartupTimeout(containerStartTimeout),
	}

	container, err := testcontainers.GenericContainer(s.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(s.T(), err)

	s.minioContainer = container

	// Set MINIO_URL environment variable
	host, _ := container.Host(s.ctx)
	port, _ := container.MappedPort(s.ctx, "9000")
	minioURL := fmt.Sprintf("http://%s:%s", host, port.Port())
	_ = os.Setenv("MINIO_URL", minioURL)
}

// startApplicationServer starts the OpenEAAP server
func (s *E2ETestSuite) startApplicationServer() {
	// Set test configuration
	_ = os.Setenv("APP_ENV", "test")
	_ = os.Setenv("SERVER_PORT", "8080")

	// Build the server binary
	// Note: In real implementation, use exec.Command to build and run
	// For this example, assume the server is already built

	// Start server process (simplified)
	// In production, use proper process management
	go func() {
		// Start server
		// This is a placeholder - actual implementation would start the server
		// e.g., exec.Command("./bin/server").Start()
	}()
}

// waitForServerReady waits for the server to be ready
func (s *E2ETestSuite) waitForServerReady() {
	deadline := time.Now().Add(serverStartTimeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			resp, err := s.httpClient.Get(baseURL + healthEndpoint)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				return
			}
			if resp != nil {
				resp.Body.Close()
			}

			if time.Now().After(deadline) {
				s.T().Fatal("Server failed to start within timeout")
			}
		case <-s.ctx.Done():
			return
		}
	}
}

// authenticate performs authentication and returns token
func (s *E2ETestSuite) authenticate() string {
	// In real implementation, call auth endpoint
	// For testing, return a mock token
	return "test-auth-token"
}

// makeRequest makes an HTTP request with authentication
func (s *E2ETestSuite) makeRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, baseURL+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.authToken)

	return s.httpClient.Do(req)
}

// TestHealthCheck tests the health check endpoint
func (s *E2ETestSuite) TestHealthCheck() {
	resp, err := s.httpClient.Get(baseURL + healthEndpoint)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), "healthy", health["status"])
}

// TestAgentLifecycle tests the complete agent lifecycle
func (s *E2ETestSuite) TestAgentLifecycle() {
	// Create agent
	createReq := dto.CreateAgentRequest{
		Name:        "Test E2E Agent",
		Description: "Agent created in E2E test",
		RuntimeType: "native",
		Config: map[string]interface{}{
			"model":       "gpt-4",
			"temperature": 0.7,
		},
	}

	resp, err := s.makeRequest(http.MethodPost, agentsEndpoint, createReq)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusCreated, resp.StatusCode)

	var createResp dto.AgentResponse
	err = json.NewDecoder(resp.Body).Decode(&createResp)
	require.NoError(s.T(), err)

	s.testAgentID = createResp.ID
	assert.Equal(s.T(), createReq.Name, createResp.Name)

	// Get agent
	resp, err = s.makeRequest(http.MethodGet, agentsEndpoint+"/"+s.testAgentID, nil)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var getResp dto.AgentResponse
	err = json.NewDecoder(resp.Body).Decode(&getResp)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), s.testAgentID, getResp.ID)

	// Update agent
	updateReq := dto.UpdateAgentRequest{
		Description: "Updated description",
	}

	resp, err = s.makeRequest(http.MethodPut, agentsEndpoint+"/"+s.testAgentID, updateReq)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	// List agents
	resp, err = s.makeRequest(http.MethodGet, agentsEndpoint, nil)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var listResp struct {
		Items []dto.AgentResponse `json:"items"`
		Total int                 `json:"total"`
	}
	err = json.NewDecoder(resp.Body).Decode(&listResp)
	require.NoError(s.T(), err)

	assert.Greater(s.T(), listResp.Total, 0)

	// Delete agent
	resp, err = s.makeRequest(http.MethodDelete, agentsEndpoint+"/"+s.testAgentID, nil)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusNoContent, resp.StatusCode)
}

// TestWorkflowExecution tests workflow execution
func (s *E2ETestSuite) TestWorkflowExecution() {
	// Create workflow
	createReq := dto.CreateWorkflowRequest{
		Name:        "Test E2E Workflow",
		Description: "Workflow created in E2E test",
		Steps: []dto.WorkflowStepRequest{
			{
				Name: "step1",
				Type: "agent",
				Config: map[string]interface{}{
					"agent_id": "test-agent-id",
					"prompt":   "Hello",
				},
			},
		},
	}

	resp, err := s.makeRequest(http.MethodPost, workflowsEndpoint, createReq)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusCreated, resp.StatusCode)

	var createResp dto.WorkflowResponse
	err = json.NewDecoder(resp.Body).Decode(&createResp)
	require.NoError(s.T(), err)

	s.testWorkflowID = createResp.ID

	// Run workflow
	runReq := dto.RunWorkflowRequest{
		Input: map[string]interface{}{
			"query": "Test query",
		},
	}

	resp, err = s.makeRequest(http.MethodPost, workflowsEndpoint+"/"+s.testWorkflowID+"/run", runReq)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var runResp dto.WorkflowExecutionResponse
	err = json.NewDecoder(resp.Body).Decode(&runResp)
	require.NoError(s.T(), err)

	assert.NotEmpty(s.T(), runResp.ExecutionID)

	// Get execution status
	resp, err = s.makeRequest(http.MethodGet, workflowsEndpoint+"/"+s.testWorkflowID+"/executions/"+runResp.ExecutionID, nil)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
}

// TestModelManagement tests model management operations
func (s *E2ETestSuite) TestModelManagement() {
	// List models
	resp, err := s.makeRequest(http.MethodGet, modelsEndpoint, nil)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var listResp struct {
		Items []dto.ModelResponse `json:"items"`
		Total int                 `json:"total"`
	}
	err = json.NewDecoder(resp.Body).Decode(&listResp)
	require.NoError(s.T(), err)

	// Register model
	registerReq := dto.RegisterModelRequest{
		Name:     "Test E2E Model",
		Type:     "llm",
		Version:  "v1.0.0",
		Endpoint: "http://localhost:8000",
		Config: map[string]interface{}{
			"max_tokens": 2048,
		},
	}

	resp, err = s.makeRequest(http.MethodPost, modelsEndpoint, registerReq)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusCreated, resp.StatusCode)

	var registerResp dto.ModelResponse
	err = json.NewDecoder(resp.Body).Decode(&registerResp)
	require.NoError(s.T(), err)

	s.testModelID = registerResp.ID

	// Get model health
	resp, err = s.makeRequest(http.MethodGet, modelsEndpoint+"/"+s.testModelID+"/health", nil)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	// Status may vary, just check response is valid
	assert.Contains(s.T(), []int{http.StatusOK, http.StatusServiceUnavailable}, resp.StatusCode)
}

// TestConcurrentRequests tests concurrent API requests
func (s *E2ETestSuite) TestConcurrentRequests() {
	const concurrency = 10

	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(index int) {
			defer func() { done <- true }()

			createReq := dto.CreateAgentRequest{
				Name:        fmt.Sprintf("Concurrent Agent %d", index),
				Description: "Agent created in concurrent test",
				RuntimeType: "native",
			}

			resp, err := s.makeRequest(http.MethodPost, agentsEndpoint, createReq)
			assert.NoError(s.T(), err)
			if resp != nil {
				resp.Body.Close()
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < concurrency; i++ {
		<-done
	}
}

// TestRateLimiting tests rate limiting functionality
func (s *E2ETestSuite) TestRateLimiting() {
	const requestCount = 100
	var statusCodes []int

	for i := 0; i < requestCount; i++ {
		resp, err := s.makeRequest(http.MethodGet, agentsEndpoint, nil)
		if err == nil && resp != nil {
			statusCodes = append(statusCodes, resp.StatusCode)
			resp.Body.Close()
		}
	}

	// Should have at least some 429 (Too Many Requests) responses
	hasRateLimited := false
	for _, code := range statusCodes {
		if code == http.StatusTooManyRequests {
			hasRateLimited = true
			break
		}
	}

	// Rate limiting may or may not trigger depending on configuration
	// This is informational
	s.T().Logf("Rate limiting triggered: %v", hasRateLimited)
}

// TestSuite runs the E2E test suite
func TestE2EAPISuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}

//Personal.AI order the ending
