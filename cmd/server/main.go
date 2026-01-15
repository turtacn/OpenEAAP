// cmd/server/main.go
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/openeeap/openeeap/internal/api/grpc"
	httpAPI "github.com/openeeap/openeeap/internal/api/http"
	"github.com/openeeap/openeeap/internal/app/service"
	"github.com/openeeap/openeeap/internal/domain/agent"
	"github.com/openeeap/openeeap/internal/domain/knowledge"
	"github.com/openeeap/openeeap/internal/domain/model"
	"github.com/openeeap/openeeap/internal/domain/workflow"
	"github.com/openeeap/openeeap/internal/governance/audit"
	"github.com/openeeap/openeeap/internal/governance/policy"
	"github.com/openeeap/openeeap/internal/infrastructure/message/kafka"
	"github.com/openeeap/openeeap/internal/infrastructure/repository/postgres"
	"github.com/openeeap/openeeap/internal/infrastructure/repository/redis"
	"github.com/openeeap/openeeap/internal/infrastructure/storage/minio"
	"github.com/openeeap/openeeap/internal/infrastructure/vector/milvus"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/internal/platform/inference"
	"github.com/openeeap/openeeap/internal/platform/learning"
	"github.com/openeeap/openeeap/internal/platform/orchestrator"
	"github.com/openeeap/openeeap/internal/platform/rag"
	"github.com/openeeap/openeeap/internal/platform/runtime"
	"github.com/openeeap/openeeap/internal/platform/training"
	"github.com/openeeap/openeeap/pkg/config"
	"github.com/openeeap/openeeap/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	appName    = "openeeap-server"
	appVersion = "v1.0.0"
)

// Server 主服务器结构
type Server struct {
	cfg              *config.Config
	logger           logging.Logger
	tracer           trace.Tracer
	metricsCollector *metrics.MetricsCollector

	// 数据库和存储
	db           *gorm.DB
	redisClient  *redis.Client
	milvusClient *milvus.MilvusClient
	minioClient  *minio.MinioClient
	kafkaClient  *kafka.KafkaClient

	// 仓储层
	agentRepo     agent.Repository
	workflowRepo  workflow.Repository
	modelRepo     model.Repository
	knowledgeRepo knowledge.Repository

	// 平台核心
	orchestrator    *orchestrator.Orchestrator
	runtimeManager  *runtime.PluginManager
	inferenceGW     *inference.Gateway
	ragEngine       *rag.RAGEngine
	learningEngine  *learning.OnlineLearningEngine
	trainingService *training.TrainingService

	// 治理层
	policyEngine *policy.PolicyDecisionPoint
	auditLogger  *audit.AuditLogger

	// 应用服务
	agentService    *service.AgentService
	workflowService *service.WorkflowService
	modelService    *service.ModelService
	dataService     *service.DataService

	// HTTP/gRPC 服务器
	httpServer *http.Server
	grpcServer *grpc.Server

	// 后台任务
	backgroundTasks []BackgroundTask
}

// BackgroundTask 后台任务接口
type BackgroundTask interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Name() string
}

func main() {
	// 初始化服务器
	srv, err := NewServer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create server: %v\n", err)
		os.Exit(1)
	}

	// 启动服务器
	if err := srv.Start(); err != nil {
		srv.logger.Error("Failed to start server", zap.Error(err))
		os.Exit(1)
	}

	// 等待退出信号
	srv.WaitForShutdown()
}

// NewServer 创建服务器实例
func NewServer() (*Server, error) {
	srv := &Server{}

	// 1. 加载配置
	if err := srv.loadConfig(); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// 2. 初始化可观测性
	if err := srv.initObservability(); err != nil {
		return nil, fmt.Errorf("init observability: %w", err)
	}

	srv.logger.Info("Starting OpenEAAP Server",
		zap.String("version", appVersion),
		zap.String("environment", srv.cfg.Environment))

	// 3. 初始化基础设施
	if err := srv.initInfrastructure(); err != nil {
		return nil, fmt.Errorf("init infrastructure: %w", err)
	}

	// 4. 初始化仓储层
	if err := srv.initRepositories(); err != nil {
		return nil, fmt.Errorf("init repositories: %w", err)
	}

	// 5. 初始化平台核心
	if err := srv.initPlatform(); err != nil {
		return nil, fmt.Errorf("init platform: %w", err)
	}

	// 6. 初始化治理层
	if err := srv.initGovernance(); err != nil {
		return nil, fmt.Errorf("init governance: %w", err)
	}

	// 7. 初始化应用服务
	if err := srv.initApplicationServices(); err != nil {
		return nil, fmt.Errorf("init application services: %w", err)
	}

	// 8. 初始化 API 层
	if err := srv.initAPIServers(); err != nil {
		return nil, fmt.Errorf("init API servers: %w", err)
	}

	// 9. 初始化后台任务
	if err := srv.initBackgroundTasks(); err != nil {
		return nil, fmt.Errorf("init background tasks: %w", err)
	}

	return srv, nil
}

// loadConfig 加载配置
func (s *Server) loadConfig() error {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./configs/default.yaml"
	}

	loader := config.NewLoader()
	cfg, err := loader.Load(configPath)
	if err != nil {
		return errors.Wrap(err, "load config file")
	}

	// 从环境变量覆盖配置
	if err := loader.LoadFromEnv(cfg); err != nil {
		return errors.Wrap(err, "load config from env")
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return errors.Wrap(err, "validate config")
	}

	s.cfg = cfg
	return nil
}

// initObservability 初始化可观测性
func (s *Server) initObservability() error {
	// 初始化日志
	logger, err := logging.NewLogger(s.cfg.Observability.Logging)
	if err != nil {
		return errors.Wrap(err, "create logger")
	}
	s.logger = logger

	// 初始化追踪
	tracer, err := trace.NewTracer(s.cfg.Observability.Tracing, appName)
	if err != nil {
		return errors.Wrap(err, "create tracer")
	}
	s.tracer = tracer

	// 初始化指标收集
	collector, err := metrics.NewMetricsCollector(s.cfg.Observability.Metrics)
	if err != nil {
		return errors.Wrap(err, "create metrics collector")
	}
	s.metricsCollector = collector

	s.logger.Info("Observability initialized")
	return nil
}

// initInfrastructure 初始化基础设施
func (s *Server) initInfrastructure() error {
	ctx := context.Background()

	// 初始化 PostgreSQL
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		s.cfg.Database.Host,
		s.cfg.Database.Port,
		s.cfg.Database.User,
		s.cfg.Database.Password,
		s.cfg.Database.DBName,
		s.cfg.Database.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logging.NewGormLogger(s.logger),
	})
	if err != nil {
		return errors.Wrap(err, "connect to postgres")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return errors.Wrap(err, "get sql.DB")
	}

	sqlDB.SetMaxIdleConns(s.cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(s.cfg.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(s.cfg.Database.ConnMaxLifetime) * time.Minute)

	s.db = db
	s.logger.Info("PostgreSQL connected")

	// 初始化 Redis
	redisClient, err := redis.NewClient(ctx, s.cfg.Redis)
	if err != nil {
		return errors.Wrap(err, "create redis client")
	}
	s.redisClient = redisClient
	s.logger.Info("Redis connected")

	// 初始化 Milvus
	milvusClient, err := milvus.NewMilvusClient(ctx, s.cfg.Milvus)
	if err != nil {
		return errors.Wrap(err, "create milvus client")
	}
	s.milvusClient = milvusClient
	s.logger.Info("Milvus connected")

	// 初始化 MinIO
	minioClient, err := minio.NewMinioClient(ctx, s.cfg.MinIO)
	if err != nil {
		return errors.Wrap(err, "create minio client")
	}
	s.minioClient = minioClient
	s.logger.Info("MinIO connected")

	// 初始化 Kafka
	kafkaClient, err := kafka.NewKafkaClient(ctx, s.cfg.Kafka)
	if err != nil {
		return errors.Wrap(err, "create kafka client")
	}
	s.kafkaClient = kafkaClient
	s.logger.Info("Kafka connected")

	return nil
}

// initRepositories 初始化仓储层
func (s *Server) initRepositories() error {
	s.agentRepo = postgres.NewAgentRepository(s.db, s.logger)
	s.workflowRepo = postgres.NewWorkflowRepository(s.db, s.logger)
	s.modelRepo = postgres.NewModelRepository(s.db, s.logger)
	s.knowledgeRepo = postgres.NewKnowledgeRepository(s.db, s.milvusClient, s.logger)

	s.logger.Info("Repositories initialized")
	return nil
}

// initPlatform 初始化平台核心
func (s *Server) initPlatform() error {
	ctx := context.Background()

	// 初始化运行时管理器
	runtimeManager, err := runtime.NewPluginManager(s.cfg.Platform.Runtime, s.logger)
	if err != nil {
		return errors.Wrap(err, "create runtime manager")
	}
	s.runtimeManager = runtimeManager

	// 初始化推理网关
	inferenceGW, err := inference.NewGateway(ctx, s.cfg.Platform.Inference,
		s.redisClient, s.milvusClient, s.logger, s.metricsCollector)
	if err != nil {
		return errors.Wrap(err, "create inference gateway")
	}
	s.inferenceGW = inferenceGW

	// 初始化 RAG 引擎
	ragEngine, err := rag.NewRAGEngine(ctx, s.cfg.Platform.RAG,
		s.knowledgeRepo, s.milvusClient, s.inferenceGW, s.logger)
	if err != nil {
		return errors.Wrap(err, "create RAG engine")
	}
	s.ragEngine = ragEngine

	// 初始化在线学习引擎
	learningEngine, err := learning.NewOnlineLearningEngine(ctx, s.cfg.Platform.Learning,
		s.kafkaClient, s.logger)
	if err != nil {
		return errors.Wrap(err, "create learning engine")
	}
	s.learningEngine = learningEngine

	// 初始化训练服务
	trainingService, err := training.NewTrainingService(ctx, s.cfg.Platform.Training,
		s.kafkaClient, s.logger)
	if err != nil {
		return errors.Wrap(err, "create training service")
	}
	s.trainingService = trainingService

	// 初始化编排器
	orchestratorInst, err := orchestrator.NewOrchestrator(
		s.cfg.Platform.Orchestrator,
		s.runtimeManager,
		s.inferenceGW,
		s.ragEngine,
		s.logger,
		s.tracer,
	)
	if err != nil {
		return errors.Wrap(err, "create orchestrator")
	}
	s.orchestrator = orchestratorInst

	s.logger.Info("Platform core initialized")
	return nil
}

// initGovernance 初始化治理层
func (s *Server) initGovernance() error {
	ctx := context.Background()

	// 初始化策略引擎
	policyEngine, err := policy.NewPolicyDecisionPoint(ctx, s.cfg.Governance.Policy, s.logger)
	if err != nil {
		return errors.Wrap(err, "create policy engine")
	}
	s.policyEngine = policyEngine

	// 初始化审计日志
	auditLogger, err := audit.NewAuditLogger(ctx, s.cfg.Governance.Audit, s.db, s.logger)
	if err != nil {
		return errors.Wrap(err, "create audit logger")
	}
	s.auditLogger = auditLogger

	s.logger.Info("Governance layer initialized")
	return nil
}

// initApplicationServices 初始化应用服务
func (s *Server) initApplicationServices() error {
	s.agentService = service.NewAgentService(
		s.agentRepo,
		s.orchestrator,
		s.policyEngine,
		s.auditLogger,
		s.logger,
	)

	s.workflowService = service.NewWorkflowService(
		s.workflowRepo,
		s.orchestrator,
		s.policyEngine,
		s.auditLogger,
		s.logger,
	)

	s.modelService = service.NewModelService(
		s.modelRepo,
		s.inferenceGW,
		s.trainingService,
		s.policyEngine,
		s.auditLogger,
		s.logger,
	)

	s.dataService = service.NewDataService(
		s.knowledgeRepo,
		s.ragEngine,
		s.minioClient,
		s.policyEngine,
		s.auditLogger,
		s.logger,
	)

	s.logger.Info("Application services initialized")
	return nil
}

// initAPIServers 初始化 API 服务器
func (s *Server) initAPIServers() error {
	// 初始化 HTTP 服务器
	httpRouter := httpAPI.NewRouter(
		s.cfg.Server.HTTP,
		s.agentService,
		s.workflowService,
		s.modelService,
		s.dataService,
		s.policyEngine,
		s.logger,
		s.tracer,
		s.metricsCollector,
	)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.cfg.Server.HTTP.Port),
		Handler:      httpRouter,
		ReadTimeout:  time.Duration(s.cfg.Server.HTTP.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.cfg.Server.HTTP.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(s.cfg.Server.HTTP.IdleTimeout) * time.Second,
	}

	// 初始化 gRPC 服务器
	grpcSrv, err := grpc.NewServer(
		s.cfg.Server.GRPC,
		s.agentService,
		s.workflowService,
		s.modelService,
		s.logger,
		s.tracer,
	)
	if err != nil {
		return errors.Wrap(err, "create grpc server")
	}
	s.grpcServer = grpcSrv

	s.logger.Info("API servers initialized")
	return nil
}

// initBackgroundTasks 初始化后台任务
func (s *Server) initBackgroundTasks() error {
	// 添加后台任务
	s.backgroundTasks = append(s.backgroundTasks,
		s.learningEngine,  // 在线学习任务
		s.trainingService, // 训练任务
	)

	s.logger.Info("Background tasks initialized",
		zap.Int("count", len(s.backgroundTasks)))
	return nil
}

// Start 启动服务器
func (s *Server) Start() error {
	ctx := context.Background()

	// 启动后台任务
	for _, task := range s.backgroundTasks {
		if err := task.Start(ctx); err != nil {
			return errors.Wrapf(err, "start background task: %s", task.Name())
		}
		s.logger.Info("Background task started", zap.String("name", task.Name()))
	}

	// 启动 HTTP 服务器
	go func() {
		s.logger.Info("HTTP server starting",
			zap.String("address", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// 启动 gRPC 服务器
	go func() {
		addr := fmt.Sprintf(":%d", s.cfg.Server.GRPC.Port)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			s.logger.Fatal("gRPC listen failed", zap.Error(err))
		}

		s.logger.Info("gRPC server starting", zap.String("address", addr))
		if err := s.grpcServer.Serve(lis); err != nil {
			s.logger.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	s.logger.Info("OpenEAAP Server started successfully")
	return nil
}

// WaitForShutdown 等待退出信号并优雅关闭
func (s *Server) WaitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	s.logger.Info("Received shutdown signal", zap.String("signal", sig.String()))

	// 创建关闭超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭 HTTP 服务器
	s.logger.Info("Shutting down HTTP server...")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP server shutdown failed", zap.Error(err))
	}

	// 关闭 gRPC 服务器
	s.logger.Info("Shutting down gRPC server...")
	s.grpcServer.GracefulStop()

	// 停止后台任务
	for _, task := range s.backgroundTasks {
		s.logger.Info("Stopping background task", zap.String("name", task.Name()))
		if err := task.Stop(ctx); err != nil {
			s.logger.Error("Background task stop failed",
				zap.String("name", task.Name()),
				zap.Error(err))
		}
	}

	// 关闭基础设施连接
	s.logger.Info("Closing infrastructure connections...")

	if s.kafkaClient != nil {
		if err := s.kafkaClient.Close(); err != nil {
			s.logger.Error("Kafka close failed", zap.Error(err))
		}
	}

	if s.milvusClient != nil {
		if err := s.milvusClient.Close(); err != nil {
			s.logger.Error("Milvus close failed", zap.Error(err))
		}
	}

	if s.redisClient != nil {
		if err := s.redisClient.Close(); err != nil {
			s.logger.Error("Redis close failed", zap.Error(err))
		}
	}

	if s.db != nil {
		sqlDB, _ := s.db.DB()
		if sqlDB != nil {
			if err := sqlDB.Close(); err != nil {
				s.logger.Error("PostgreSQL close failed", zap.Error(err))
			}
		}
	}

	// 关闭可观测性组件
	if s.tracer != nil {
		if err := s.tracer.Shutdown(ctx); err != nil {
			s.logger.Error("Tracer shutdown failed", zap.Error(err))
		}
	}

	s.logger.Info("OpenEAAP Server shutdown complete")

	// 同步日志
	if zapLogger, ok := s.logger.(*logging.ZapLogger); ok {
		_ = zapLogger.Sync()
	}
}

//Personal.AI order the ending
