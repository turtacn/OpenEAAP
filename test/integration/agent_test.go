package integration

import (
	"context"
	"testing"
	"time"

	"github.com/openeeap/openeeap/internal/app/dto"
	"github.com/openeeap/openeeap/internal/app/service"
	"github.com/openeeap/openeeap/internal/domain/agent"
	"github.com/openeeap/openeeap/internal/infrastructure/repository/postgres"
	"github.com/openeeap/openeeap/internal/infrastructure/repository/redis"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/pkg/config"
	"github.com/openeeap/openeeap/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// AgentIntegrationTestSuite 定义 Agent 集成测试套件
type AgentIntegrationTestSuite struct {
	suite.Suite
	ctx context.Context

	// Testcontainers
	postgresContainer *postgres.PostgresContainer
	redisContainer    *redis.RedisContainer

	// 基础设施
	cfg           *config.Config
	logger        logging.Logger
	agentRepo     agent.AgentRepository
	cacheRepo     redis.CacheRepository
	agentService  *service.AgentService
	domainService *agent.DomainService

	// 清理函数
	cleanup []func()
}

// SetupSuite 在所有测试前执行一次
func (s *AgentIntegrationTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.cleanup = make([]func(), 0)

	// 初始化日志
	s.logger = logging.NewLogger(&config.LogConfig{
		Level:  "debug",
		Format: "json",
	})

	// 启动 PostgreSQL 容器
	s.startPostgresContainer()

	// 启动 Redis 容器
	s.startRedisContainer()

	// 初始化配置
	s.initConfig()

	// 初始化仓储和服务
	s.initRepositories()
	s.initServices()

	// 运行数据库迁移
	s.runMigrations()
}

// TearDownSuite 在所有测试后执行一次
func (s *AgentIntegrationTestSuite) TearDownSuite() {
	// 执行清理函数
	for i := len(s.cleanup) - 1; i >= 0; i-- {
		s.cleanup[i]()
	}

	// 停止容器
	if s.postgresContainer != nil {
		_ = s.postgresContainer.Terminate(s.ctx)
	}
	if s.redisContainer != nil {
		_ = s.redisContainer.Terminate(s.ctx)
	}
}

// SetupTest 在每个测试前执行
func (s *AgentIntegrationTestSuite) SetupTest() {
	// 清理测试数据
	s.cleanupTestData()
}

// startPostgresContainer 启动 PostgreSQL 测试容器
func (s *AgentIntegrationTestSuite) startPostgresContainer() {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "openeeap_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithStartupTimeout(60 * time.Second),
	}

	container, err := postgres.RunContainer(s.ctx, testcontainers.WithImage("postgres:15-alpine"))
	require.NoError(s.T(), err, "Failed to start PostgreSQL container")

	s.postgresContainer = container
	s.logger.Info("PostgreSQL container started")
}

// startRedisContainer 启动 Redis 测试容器
func (s *AgentIntegrationTestSuite) startRedisContainer() {
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor: wait.ForLog("Ready to accept connections").
			WithStartupTimeout(30 * time.Second),
	}

	container, err := redis.RunContainer(s.ctx, testcontainers.WithImage("redis:7-alpine"))
	require.NoError(s.T(), err, "Failed to start Redis container")

	s.redisContainer = container
	s.logger.Info("Redis container started")
}

// initConfig 初始化配置
func (s *AgentIntegrationTestSuite) initConfig() {
	connStr, err := s.postgresContainer.ConnectionString(s.ctx)
	require.NoError(s.T(), err)

	redisHost, err := s.redisContainer.Host(s.ctx)
	require.NoError(s.T(), err)
	redisPort, err := s.redisContainer.MappedPort(s.ctx, "6379")
	require.NoError(s.T(), err)

	s.cfg = &config.Config{
		Database: config.DatabaseConfig{
			URL:             connStr,
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Redis: config.RedisConfig{
			Addr:         redisHost + ":" + redisPort.Port(),
			Password:     "",
			DB:           0,
			PoolSize:     10,
			MinIdleConns: 2,
		},
	}
}

// initRepositories 初始化仓储
func (s *AgentIntegrationTestSuite) initRepositories() {
	var err error

	// 初始化 PostgreSQL 仓储
	s.agentRepo, err = postgres.NewAgentRepository(s.cfg.Database, s.logger)
	require.NoError(s.T(), err, "Failed to initialize agent repository")

	// 初始化 Redis 缓存
	s.cacheRepo, err = redis.NewCacheRepository(s.cfg.Redis, s.logger)
	require.NoError(s.T(), err, "Failed to initialize cache repository")
}

// initServices 初始化服务
func (s *AgentIntegrationTestSuite) initServices() {
	// 初始化领域服务
	s.domainService = agent.NewDomainService(s.agentRepo, s.logger)

	// 初始化应用服务
	s.agentService = service.NewAgentService(
		s.domainService,
		s.agentRepo,
		s.cacheRepo,
		s.logger,
	)
}

// runMigrations 运行数据库迁移
func (s *AgentIntegrationTestSuite) runMigrations() {
	// 这里应该调用实际的迁移逻辑
	// 为简化测试，这里直接创建表结构
	db := s.agentRepo.(*postgres.AgentRepository).GetDB()

	_, err := db.Exec(`
   	CREATE TABLE IF NOT EXISTS agents (
   		id VARCHAR(36) PRIMARY KEY,
   		name VARCHAR(255) NOT NULL,
   		description TEXT,
   		runtime_type VARCHAR(50) NOT NULL,
   		config JSONB NOT NULL,
   		status VARCHAR(50) NOT NULL,
   		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
   		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
   		deleted_at TIMESTAMP
   	);
   	CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
   	CREATE INDEX IF NOT EXISTS idx_agents_runtime_type ON agents(runtime_type);
   `)
	require.NoError(s.T(), err, "Failed to run migrations")
}

// cleanupTestData 清理测试数据
func (s *AgentIntegrationTestSuite) cleanupTestData() {
	db := s.agentRepo.(*postgres.AgentRepository).GetDB()
	_, _ = db.Exec("TRUNCATE TABLE agents CASCADE")

	// 清理 Redis 缓存
	_ = s.cacheRepo.FlushAll(s.ctx)
}

// TestCreateAgent 测试创建 Agent
func (s *AgentIntegrationTestSuite) TestCreateAgent() {
	req := &dto.CreateAgentRequest{
		Name:        "Test Agent",
		Description: "This is a test agent for integration testing",
		RuntimeType: types.RuntimeTypeNative,
		Config: map[string]interface{}{
			"model":       "gpt-4",
			"temperature": 0.7,
			"max_tokens":  2000,
		},
	}

	// 创建 Agent
	resp, err := s.agentService.CreateAgent(s.ctx, req)
	require.NoError(s.T(), err, "Failed to create agent")
	assert.NotEmpty(s.T(), resp.ID)
	assert.Equal(s.T(), req.Name, resp.Name)
	assert.Equal(s.T(), req.Description, resp.Description)
	assert.Equal(s.T(), types.AgentStatusActive, resp.Status)

	// 验证数据库中存在
	dbAgent, err := s.agentRepo.GetByID(s.ctx, resp.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), resp.ID, dbAgent.ID)
	assert.Equal(s.T(), req.Name, dbAgent.Name)

	// 验证缓存中存在
	cacheKey := "agent:" + resp.ID
	cached, err := s.cacheRepo.Get(s.ctx, cacheKey)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), cached)
}

// TestGetAgent 测试获取 Agent
func (s *AgentIntegrationTestSuite) TestGetAgent() {
	// 先创建一个 Agent
	createReq := &dto.CreateAgentRequest{
		Name:        "Get Test Agent",
		Description: "Agent for get operation testing",
		RuntimeType: types.RuntimeTypeLangChain,
		Config: map[string]interface{}{
			"model": "claude-sonnet-4",
		},
	}

	createResp, err := s.agentService.CreateAgent(s.ctx, createReq)
	require.NoError(s.T(), err)

	// 获取 Agent
	getResp, err := s.agentService.GetAgent(s.ctx, createResp.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), createResp.ID, getResp.ID)
	assert.Equal(s.T(), createReq.Name, getResp.Name)

	// 测试缓存命中
	cacheKey := "agent:" + createResp.ID
	cached, err := s.cacheRepo.Get(s.ctx, cacheKey)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), cached)
}

// TestUpdateAgent 测试更新 Agent
func (s *AgentIntegrationTestSuite) TestUpdateAgent() {
	// 创建 Agent
	createReq := &dto.CreateAgentRequest{
		Name:        "Update Test Agent",
		RuntimeType: types.RuntimeTypeNative,
		Config:      map[string]interface{}{"model": "gpt-4"},
	}

	createResp, err := s.agentService.CreateAgent(s.ctx, createReq)
	require.NoError(s.T(), err)

	// 更新 Agent
	updateReq := &dto.UpdateAgentRequest{
		Name:        "Updated Agent Name",
		Description: "Updated description",
		Config: map[string]interface{}{
			"model":       "gpt-4-turbo",
			"temperature": 0.9,
		},
	}

	updateResp, err := s.agentService.UpdateAgent(s.ctx, createResp.ID, updateReq)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), updateReq.Name, updateResp.Name)
	assert.Equal(s.T(), updateReq.Description, updateResp.Description)

	// 验证数据库更新
	dbAgent, err := s.agentRepo.GetByID(s.ctx, createResp.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), updateReq.Name, dbAgent.Name)

	// 验证缓存失效
	cacheKey := "agent:" + createResp.ID
	_, err = s.cacheRepo.Get(s.ctx, cacheKey)
	assert.Error(s.T(), err) // 缓存应该被清除
}

// TestDeleteAgent 测试删除 Agent
func (s *AgentIntegrationTestSuite) TestDeleteAgent() {
	// 创建 Agent
	createReq := &dto.CreateAgentRequest{
		Name:        "Delete Test Agent",
		RuntimeType: types.RuntimeTypeNative,
		Config:      map[string]interface{}{"model": "gpt-4"},
	}

	createResp, err := s.agentService.CreateAgent(s.ctx, createReq)
	require.NoError(s.T(), err)

	// 删除 Agent
	err = s.agentService.DeleteAgent(s.ctx, createResp.ID)
	require.NoError(s.T(), err)

	// 验证数据库中已删除（软删除）
	dbAgent, err := s.agentRepo.GetByID(s.ctx, createResp.ID)
	assert.Error(s.T(), err)
	assert.Nil(s.T(), dbAgent)

	// 验证缓存已清除
	cacheKey := "agent:" + createResp.ID
	_, err = s.cacheRepo.Get(s.ctx, cacheKey)
	assert.Error(s.T(), err)
}

// TestListAgents 测试列出 Agents
func (s *AgentIntegrationTestSuite) TestListAgents() {
	// 创建多个 Agents
	for i := 0; i < 5; i++ {
		createReq := &dto.CreateAgentRequest{
			Name:        "List Test Agent " + string(rune(i)),
			RuntimeType: types.RuntimeTypeNative,
			Config:      map[string]interface{}{"model": "gpt-4"},
		}
		_, err := s.agentService.CreateAgent(s.ctx, createReq)
		require.NoError(s.T(), err)
	}

	// 列出 Agents
	listReq := &dto.ListAgentsRequest{
		Page:     1,
		PageSize: 10,
	}

	listResp, err := s.agentService.ListAgents(s.ctx, listReq)
	require.NoError(s.T(), err)
	assert.GreaterOrEqual(s.T(), len(listResp.Items), 5)
	assert.GreaterOrEqual(s.T(), listResp.Total, int64(5))
}

// TestAgentExecution 测试 Agent 执行
func (s *AgentIntegrationTestSuite) TestAgentExecution() {
	// 创建 Agent
	createReq := &dto.CreateAgentRequest{
		Name:        "Execution Test Agent",
		RuntimeType: types.RuntimeTypeNative,
		Config: map[string]interface{}{
			"model":       "gpt-4",
			"temperature": 0.7,
		},
	}

	createResp, err := s.agentService.CreateAgent(s.ctx, createReq)
	require.NoError(s.T(), err)

	// 执行 Agent（这里需要 Mock 推理引擎，简化测试）
	execReq := &dto.ExecuteAgentRequest{
		AgentID: createResp.ID,
		Input:   "Hello, can you help me?",
		Context: map[string]interface{}{
			"user_id": "test-user-123",
		},
	}

	// 注意：实际执行需要推理引擎，这里仅测试接口调用
	execResp, err := s.agentService.ExecuteAgent(s.ctx, execReq)
	if err != nil {
		// 如果推理引擎未启动，跳过此测试
		s.T().Skip("Inference engine not available")
	}

	assert.NotNil(s.T(), execResp)
	assert.NotEmpty(s.T(), execResp.Output)
	assert.NotEmpty(s.T(), execResp.TraceID)
}

// TestAgentConcurrentOperations 测试并发操作
func (s *AgentIntegrationTestSuite) TestAgentConcurrentOperations() {
	const concurrency = 10

	// 创建 Agent
	createReq := &dto.CreateAgentRequest{
		Name:        "Concurrent Test Agent",
		RuntimeType: types.RuntimeTypeNative,
		Config:      map[string]interface{}{"model": "gpt-4"},
	}

	createResp, err := s.agentService.CreateAgent(s.ctx, createReq)
	require.NoError(s.T(), err)

	// 并发读取
	errChan := make(chan error, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			_, err := s.agentService.GetAgent(s.ctx, createResp.ID)
			errChan <- err
		}()
	}

	// 检查并发结果
	for i := 0; i < concurrency; i++ {
		err := <-errChan
		assert.NoError(s.T(), err)
	}
}

// TestAgentValidation 测试参数校验
func (s *AgentIntegrationTestSuite) TestAgentValidation() {
	// 测试空名称
	req := &dto.CreateAgentRequest{
		Name:        "",
		RuntimeType: types.RuntimeTypeNative,
		Config:      map[string]interface{}{},
	}

	_, err := s.agentService.CreateAgent(s.ctx, req)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "name")

	// 测试无效的运行时类型
	req2 := &dto.CreateAgentRequest{
		Name:        "Valid Name",
		RuntimeType: "invalid-runtime",
		Config:      map[string]interface{}{},
	}

	_, err = s.agentService.CreateAgent(s.ctx, req2)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "runtime")
}

// TestSuite 运行测试套件
func TestAgentIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AgentIntegrationTestSuite))
}

//Personal.AI order the ending
