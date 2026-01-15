package http

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/openeeap/openeeap/internal/api/http/handler"
	"github.com/openeeap/openeeap/internal/api/http/middleware"
	"github.com/openeeap/openeeap/internal/app/service"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/pkg/config"
)

// Router HTTP 路由器
type Router struct {
	engine  *gin.Engine
	config  *config.Config
	logger  logging.Logger
	tracer  trace.Tracer
	metrics metrics.MetricsCollector

	// Handlers
	agentHandler    *handler.AgentHandler
	workflowHandler *handler.WorkflowHandler
	chatHandler     *handler.ChatHandler
	modelHandler    *handler.ModelHandler
	dataHandler     *handler.DataHandler
	authHandler     *handler.AuthHandler
	healthHandler   *handler.HealthHandler

	// Middleware
	authMiddleware      *middleware.AuthMiddleware
	rateLimitMiddleware *middleware.RateLimitMiddleware
	tracingMiddleware   *middleware.TracingMiddleware
	loggingMiddleware   *middleware.LoggingMiddleware
	metricsMiddleware   *middleware.MetricsMiddleware
}

// NewRouter 创建 HTTP 路由器
func NewRouter(
	cfg *config.Config,
	logger logging.Logger,
	tracer trace.Tracer,
	metricsCollector metrics.MetricsCollector,
	// Services
	agentService service.AgentService,
	workflowService service.WorkflowService,
	chatService service.ChatService,
	modelService service.ModelService,
	dataService service.DataService,
	authService service.AuthService,
	healthService service.HealthService,
) *Router {
	// 设置 Gin 模式
	if cfg.Server.Mode == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	// 创建 Handlers
	agentHandler := handler.NewAgentHandler(agentService, logger)
	workflowHandler := handler.NewWorkflowHandler(workflowService, logger)
	chatHandler := handler.NewChatHandler(chatService, logger)
	modelHandler := handler.NewModelHandler(modelService, logger)
	dataHandler := handler.NewDataHandler(dataService, logger)
	authHandler := handler.NewAuthHandler(authService, logger)
	healthHandler := handler.NewHealthHandler(healthService, logger)

	// 创建 Middleware
	authMiddleware := middleware.NewAuthMiddleware(authService, logger)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(cfg.RateLimit, logger)
	tracingMiddleware := middleware.NewTracingMiddleware(tracer, logger)
	loggingMiddleware := middleware.NewLoggingMiddleware(logger)
	metricsMiddleware := middleware.NewMetricsMiddleware(metricsCollector)

	router := &Router{
		engine:              engine,
		config:              cfg,
		logger:              logger,
		tracer:              tracer,
		metrics:             metricsCollector,
		agentHandler:        agentHandler,
		workflowHandler:     workflowHandler,
		chatHandler:         chatHandler,
		modelHandler:        modelHandler,
		dataHandler:         dataHandler,
		authHandler:         authHandler,
		healthHandler:       healthHandler,
		authMiddleware:      authMiddleware,
		rateLimitMiddleware: rateLimitMiddleware,
		tracingMiddleware:   tracingMiddleware,
		loggingMiddleware:   loggingMiddleware,
		metricsMiddleware:   metricsMiddleware,
	}

	router.setupMiddleware()
	router.setupRoutes()

	return router
}

// setupMiddleware 设置全局中间件
func (r *Router) setupMiddleware() {
	// Recovery 中间件
	r.engine.Use(gin.Recovery())

	// CORS 中间件
	r.engine.Use(cors.New(cors.Config{
		AllowOrigins:     r.config.Server.CORS.AllowOrigins,
		AllowMethods:     r.config.Server.CORS.AllowMethods,
		AllowHeaders:     r.config.Server.CORS.AllowHeaders,
		ExposeHeaders:    r.config.Server.CORS.ExposeHeaders,
		AllowCredentials: r.config.Server.CORS.AllowCredentials,
		MaxAge:           time.Duration(r.config.Server.CORS.MaxAge) * time.Second,
	}))

	// Gzip 压缩
	if r.config.Server.EnableGzip {
		r.engine.Use(gzip.Gzip(gzip.DefaultCompression))
	}

	// 全局中间件链
	r.engine.Use(
		r.tracingMiddleware.Trace(),
		r.loggingMiddleware.Log(),
		r.metricsMiddleware.Collect(),
		middleware.RequestID(),
		middleware.Timeout(time.Duration(r.config.Server.Timeout)*time.Second),
	)
}

// setupRoutes 设置路由
func (r *Router) setupRoutes() {
	// 健康检查和基础路由（无需认证）
	r.setupPublicRoutes()

	// API v1 路由
	v1 := r.engine.Group("/api/v1")
	{
		// 认证路由（无需认证）
		r.setupAuthRoutes(v1)

		// 需要认证的路由
		authenticated := v1.Group("")
		authenticated.Use(r.authMiddleware.Authenticate())
		{
			// Agent 路由
			r.setupAgentRoutes(authenticated)

			// Workflow 路由
			r.setupWorkflowRoutes(authenticated)

			// Chat 路由
			r.setupChatRoutes(authenticated)

			// Model 路由
			r.setupModelRoutes(authenticated)

			// Data 路由
			r.setupDataRoutes(authenticated)
		}
	}

	// 管理员路由
	admin := r.engine.Group("/api/v1/admin")
	admin.Use(
		r.authMiddleware.Authenticate(),
		r.authMiddleware.RequireRole("admin"),
	)
	{
		r.setupAdminRoutes(admin)
	}

	// 调试路由（仅开发模式）
	if r.config.Server.Mode != "production" {
		r.setupDebugRoutes()
	}
}

// setupPublicRoutes 设置公开路由
func (r *Router) setupPublicRoutes() {
	// 根路径
	r.engine.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"name":    "OpenEEAP API",
			"version": "1.0.0",
			"status":  "running",
		})
	})

	// 健康检查
	health := r.engine.Group("/health")
	{
		health.GET("", r.healthHandler.HealthCheck)
		health.GET("/live", r.healthHandler.LivenessProbe)
		health.GET("/ready", r.healthHandler.ReadinessProbe)
	}

	// 指标（Prometheus）
	r.engine.GET("/metrics", r.healthHandler.Metrics)

	// API 文档
	r.engine.GET("/swagger/*any", r.healthHandler.SwaggerUI)
}

// setupAuthRoutes 设置认证路由
func (r *Router) setupAuthRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", r.authHandler.Register)
		auth.POST("/login", r.authHandler.Login)
		auth.POST("/refresh", r.authHandler.RefreshToken)
		auth.POST("/logout", r.authMiddleware.Authenticate(), r.authHandler.Logout)
		auth.POST("/forgot-password", r.authHandler.ForgotPassword)
		auth.POST("/reset-password", r.authHandler.ResetPassword)
		auth.GET("/verify-email", r.authHandler.VerifyEmail)
	}
}

// setupAgentRoutes 设置 Agent 路由
func (r *Router) setupAgentRoutes(rg *gin.RouterGroup) {
	agents := rg.Group("/agents")
	agents.Use(r.rateLimitMiddleware.Limit("agent"))
	{
		// CRUD 操作
		agents.POST("", r.agentHandler.CreateAgent)
		agents.GET("", r.agentHandler.ListAgents)
		agents.GET("/:id", r.agentHandler.GetAgent)
		agents.PUT("/:id", r.agentHandler.UpdateAgent)
		agents.DELETE("/:id", r.agentHandler.DeleteAgent)

		// Agent 执行
		agents.POST("/:id/execute", r.agentHandler.ExecuteAgent)
		agents.POST("/:id/execute/stream", r.agentHandler.ExecuteAgentStream)

		// Agent 会话
		agents.POST("/:id/sessions", r.agentHandler.CreateSession)
		agents.GET("/:id/sessions/:session_id", r.agentHandler.GetSession)
		agents.DELETE("/:id/sessions/:session_id", r.agentHandler.DeleteSession)
		agents.GET("/:id/sessions/:session_id/history", r.agentHandler.GetSessionHistory)

		// Agent 工具
		agents.GET("/:id/tools", r.agentHandler.ListTools)
		agents.POST("/:id/tools", r.agentHandler.AddTool)
		agents.DELETE("/:id/tools/:tool_id", r.agentHandler.RemoveTool)

		// Agent 指标
		agents.GET("/:id/metrics", r.agentHandler.GetMetrics)

		// Agent 操作
		agents.POST("/:id/validate", r.agentHandler.ValidateAgent)
		agents.POST("/:id/clone", r.agentHandler.CloneAgent)
		agents.POST("/:id/export", r.agentHandler.ExportAgent)
		agents.POST("/import", r.agentHandler.ImportAgent)

		// 批量操作
		agents.POST("/batch", r.agentHandler.BatchOperation)
	}
}

// setupWorkflowRoutes 设置 Workflow 路由
func (r *Router) setupWorkflowRoutes(rg *gin.RouterGroup) {
	workflows := rg.Group("/workflows")
	workflows.Use(r.rateLimitMiddleware.Limit("workflow"))
	{
		// CRUD 操作
		workflows.POST("", r.workflowHandler.CreateWorkflow)
		workflows.GET("", r.workflowHandler.ListWorkflows)
		workflows.GET("/:id", r.workflowHandler.GetWorkflow)
		workflows.PUT("/:id", r.workflowHandler.UpdateWorkflow)
		workflows.DELETE("/:id", r.workflowHandler.DeleteWorkflow)

		// Workflow 执行
		workflows.POST("/:id/run", r.workflowHandler.RunWorkflow)
		workflows.POST("/:id/run/stream", r.workflowHandler.RunWorkflowStream)
		workflows.GET("/:id/executions", r.workflowHandler.ListExecutions)
		workflows.GET("/:id/executions/:execution_id", r.workflowHandler.GetExecution)
		workflows.POST("/:id/executions/:execution_id/cancel", r.workflowHandler.CancelExecution)

		// Workflow 调度
		workflows.POST("/:id/schedules", r.workflowHandler.CreateSchedule)
		workflows.GET("/:id/schedules", r.workflowHandler.ListSchedules)
		workflows.DELETE("/:id/schedules/:schedule_id", r.workflowHandler.DeleteSchedule)

		// Workflow 版本
		workflows.POST("/:id/versions", r.workflowHandler.CreateVersion)
		workflows.GET("/:id/versions", r.workflowHandler.ListVersions)
		workflows.POST("/:id/rollback", r.workflowHandler.RollbackWorkflow)

		// Workflow 指标
		workflows.GET("/:id/metrics", r.workflowHandler.GetMetrics)

		// Workflow 操作
		workflows.POST("/:id/validate", r.workflowHandler.ValidateWorkflow)
		workflows.POST("/:id/test", r.workflowHandler.TestWorkflow)
		workflows.POST("/:id/clone", r.workflowHandler.CloneWorkflow)
		workflows.POST("/:id/export", r.workflowHandler.ExportWorkflow)
		workflows.POST("/import", r.workflowHandler.ImportWorkflow)
	}
}

// setupChatRoutes 设置 Chat 路由
func (r *Router) setupChatRoutes(rg *gin.RouterGroup) {
	chat := rg.Group("/chat")
	chat.Use(r.rateLimitMiddleware.Limit("chat"))
	{
		// 会话管理
		chat.POST("/sessions", r.chatHandler.CreateSession)
		chat.GET("/sessions", r.chatHandler.ListSessions)
		chat.GET("/sessions/:id", r.chatHandler.GetSession)
		chat.DELETE("/sessions/:id", r.chatHandler.DeleteSession)

		// 消息操作
		chat.POST("/sessions/:id/messages", r.chatHandler.SendMessage)
		chat.POST("/sessions/:id/messages/stream", r.chatHandler.SendMessageStream)
		chat.GET("/sessions/:id/messages", r.chatHandler.GetMessages)
		chat.DELETE("/sessions/:id/messages/:message_id", r.chatHandler.DeleteMessage)

		// 会话操作
		chat.POST("/sessions/:id/clear", r.chatHandler.ClearSession)
		chat.POST("/sessions/:id/export", r.chatHandler.ExportSession)
		chat.GET("/sessions/:id/summary", r.chatHandler.GetSessionSummary)
	}
}

// setupModelRoutes 设置 Model 路由
func (r *Router) setupModelRoutes(rg *gin.RouterGroup) {
	models := rg.Group("/models")
	models.Use(r.rateLimitMiddleware.Limit("model"))
	{
		// CRUD 操作
		models.POST("", r.modelHandler.RegisterModel)
		models.GET("", r.modelHandler.ListModels)
		models.GET("/:id", r.modelHandler.GetModel)
		models.PUT("/:id", r.modelHandler.UpdateModel)
		models.DELETE("/:id", r.modelHandler.DeleteModel)

		// 模型部署
		models.POST("/:id/deploy", r.modelHandler.DeployModel)
		models.POST("/:id/undeploy", r.modelHandler.UndeployModel)
		models.GET("/:id/deployment", r.modelHandler.GetDeploymentStatus)

		// 模型监控
		models.GET("/:id/metrics", r.modelHandler.GetMetrics)
		models.GET("/:id/health", r.modelHandler.HealthCheck)

		// 模型训练
		models.POST("/:id/training/start", r.modelHandler.StartTraining)
		models.GET("/:id/training/jobs", r.modelHandler.ListTrainingJobs)
		models.GET("/:id/training/jobs/:job_id", r.modelHandler.GetTrainingStatus)
		models.POST("/:id/training/jobs/:job_id/stop", r.modelHandler.StopTraining)
	}
}

// setupDataRoutes 设置 Data 路由
func (r *Router) setupDataRoutes(rg *gin.RouterGroup) {
	data := rg.Group("/data")
	data.Use(r.rateLimitMiddleware.Limit("data"))
	{
		// 文档管理
		documents := data.Group("/documents")
		{
			documents.POST("", r.dataHandler.UploadDocument)
			documents.POST("/stream", r.dataHandler.UploadDocumentStream)
			documents.GET("", r.dataHandler.ListDocuments)
			documents.GET("/:id", r.dataHandler.GetDocument)
			documents.PUT("/:id", r.dataHandler.UpdateDocument)
			documents.DELETE("/:id", r.dataHandler.DeleteDocument)

			// 文档处理
			documents.POST("/:id/process", r.dataHandler.ProcessDocument)
			documents.GET("/:id/status", r.dataHandler.GetProcessingStatus)
		}

		// 集合管理
		collections := data.Group("/collections")
		{
			collections.POST("", r.dataHandler.CreateCollection)
			collections.GET("", r.dataHandler.ListCollections)
			collections.GET("/:id", r.dataHandler.GetCollection)
			collections.DELETE("/:id", r.dataHandler.DeleteCollection)
		}

		// 搜索
		search := data.Group("/search")
		{
			search.POST("/documents", r.dataHandler.SearchDocuments)
			search.POST("/semantic", r.dataHandler.SemanticSearch)
			search.POST("/hybrid", r.dataHandler.HybridSearch)
		}

		// 统计
		data.GET("/statistics", r.dataHandler.GetStatistics)
	}
}

// setupAdminRoutes 设置管理员路由
func (r *Router) setupAdminRoutes(rg *gin.RouterGroup) {
	// 用户管理
	users := rg.Group("/users")
	{
		users.GET("", r.authHandler.ListUsers)
		users.GET("/:id", r.authHandler.GetUser)
		users.PUT("/:id", r.authHandler.UpdateUser)
		users.DELETE("/:id", r.authHandler.DeleteUser)
		users.POST("/:id/roles", r.authHandler.AssignRole)
		users.DELETE("/:id/roles/:role", r.authHandler.RevokeRole)
	}

	// 系统配置
	config := rg.Group("/config")
	{
		config.GET("", r.healthHandler.GetConfig)
		config.PUT("", r.healthHandler.UpdateConfig)
	}

	// 审计日志
	rg.GET("/audit-logs", r.healthHandler.GetAuditLogs)

	// 系统统计
	rg.GET("/statistics", r.healthHandler.GetSystemStatistics)
}

// setupDebugRoutes 设置调试路由
func (r *Router) setupDebugRoutes() {
	debug := r.engine.Group("/debug")
	{
		// pprof
		pprof.Register(r.engine, "/debug/pprof")

		// 配置信息
		debug.GET("/config", func(c *gin.Context) {
			c.JSON(http.StatusOK, r.config)
		})

		// 路由列表
		debug.GET("/routes", func(c *gin.Context) {
			routes := r.engine.Routes()
			c.JSON(http.StatusOK, routes)
		})
	}
}

// Run 启动 HTTP 服务器
func (r *Router) Run() error {
	addr := r.config.Server.Host + ":" + r.config.Server.Port
	r.logger.Info("Starting HTTP server", "address", addr)

	return r.engine.Run(addr)
}

// RunTLS 启动 HTTPS 服务器
func (r *Router) RunTLS(certFile, keyFile string) error {
	addr := r.config.Server.Host + ":" + r.config.Server.Port
	r.logger.Info("Starting HTTPS server", "address", addr)

	return r.engine.RunTLS(addr, certFile, keyFile)
}

// Shutdown 优雅关闭服务器
func (r *Router) Shutdown(ctx context.Context) error {
	r.logger.Info("Shutting down HTTP server")
	// Gin 不直接支持 Shutdown，需要使用 http.Server
	// 这里返回 nil，实际关闭逻辑在 main.go 中处理
	return nil
}

// GetEngine 获取 Gin 引擎（用于测试）
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}

//Personal.AI order the ending
