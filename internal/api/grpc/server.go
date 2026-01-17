// internal/api/grpc/server.go
package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	// TODO: generate proto files
	//"github.com/openeeap/openeeap/api/proto"
	"github.com/openeeap/openeeap/internal/api/grpc/handler"
	"github.com/openeeap/openeeap/internal/app/service"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/metrics"
	"github.com/openeeap/openeeap/internal/observability/trace"
	"github.com/openeeap/openeeap/pkg/config"
)

// Server gRPC 服务器
type Server struct {
	config           *config.GRPCConfig
	server           *grpc.Server
	listener         net.Listener
	logger           logging.Logger
	tracer           trace.Tracer
	metricsCollector *metrics.MetricsCollector

	// 处理器
	agentHandler    *handler.AgentGRPCHandler
	workflowHandler *handler.WorkflowGRPCHandler
	modelHandler    *handler.ModelGRPCHandler

	// 健康检查
	healthServer *health.Server
}

// NewServer 创建 gRPC 服务器
func NewServer(
	cfg *config.GRPCConfig,
	logger logging.Logger,
	tracer trace.Tracer,
	metricsCollector metrics.MetricsCollector,
	agentService service.AgentService,
	workflowService service.WorkflowService,
	modelService service.ModelService,
) (*Server, error) {
	// 创建监听器
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	// 创建服务器实例
	s := &Server{
		config:           cfg,
		listener:         listener,
		logger:           logger,
		tracer:           tracer,
		metricsCollector: metricsCollector,
		healthServer:     health.NewServer(),
	}

	// 配置服务器选项
	serverOpts, err := s.buildServerOptions()
	if err != nil {
		return nil, fmt.Errorf("failed to build server options: %w", err)
	}

	// 创建 gRPC 服务器
	s.server = grpc.NewServer(serverOpts...)

	// 创建处理器
	s.agentHandler = handler.NewAgentGRPCHandler(agentService, logger, tracer)
	s.workflowHandler = handler.NewWorkflowGRPCHandler(workflowService, logger, tracer)
	s.modelHandler = handler.NewModelGRPCHandler(modelService, logger, tracer)

	// 注册服务
	s.registerServices()

	// 注册健康检查
	grpc_health_v1.RegisterHealthServer(s.server, s.healthServer)

	// 注册反射服务（用于 grpcurl 等工具）
	if cfg.EnableReflection {
		reflection.Register(s.server)
	}

	return s, nil
}

// buildServerOptions 构建服务器选项
func (s *Server) buildServerOptions() ([]grpc.ServerOption, error) {
	opts := []grpc.ServerOption{
		// 链式拦截器
		grpc.ChainUnaryInterceptor(
			s.recoveryInterceptor(),
			s.loggingInterceptor(),
			s.tracingInterceptor(),
			s.metricsInterceptor(),
			s.authInterceptor(),
			s.validationInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			s.streamRecoveryInterceptor(),
			s.streamLoggingInterceptor(),
			s.streamTracingInterceptor(),
			s.streamMetricsInterceptor(),
			s.streamAuthInterceptor(),
		),

		// 连接参数
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     time.Duration(s.config.MaxConnectionIdle) * time.Second,
			MaxConnectionAge:      time.Duration(s.config.MaxConnectionAge) * time.Second,
			MaxConnectionAgeGrace: time.Duration(s.config.MaxConnectionAgeGrace) * time.Second,
			Time:                  time.Duration(s.config.KeepAliveTime) * time.Second,
			Timeout:               time.Duration(s.config.KeepAliveTimeout) * time.Second,
		}),

		// 强制执行策略
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             time.Duration(s.config.MinKeepAliveTime) * time.Second,
			PermitWithoutStream: s.config.PermitWithoutStream,
		}),

		// 最大消息大小
		grpc.MaxRecvMsgSize(s.config.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(s.config.MaxSendMsgSize),

		// 并发流数量限制
		grpc.MaxConcurrentStreams(uint32(s.config.MaxConcurrentStreams)),
	}

	// TLS 配置
	if s.config.TLS.Enabled {
		creds, err := s.loadTLSCredentials()
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	return opts, nil
}

// loadTLSCredentials 加载 TLS 凭证
func (s *Server) loadTLSCredentials() (credentials.TransportCredentials, error) {
	creds, err := credentials.NewServerTLSFromFile(
		s.config.TLS.CertFile,
		s.config.TLS.KeyFile,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
	}
	return creds, nil
}

// registerServices 注册所有 gRPC 服务
func (s *Server) registerServices() {
	proto.RegisterAgentServiceServer(s.server, s.agentHandler)
	proto.RegisterWorkflowServiceServer(s.server, s.workflowHandler)
	proto.RegisterModelServiceServer(s.server, s.modelHandler)

	s.logger.Info("gRPC services registered",
		"services", []string{"AgentService", "WorkflowService", "ModelService"})
}

// Start 启动 gRPC 服务器
func (s *Server) Start() error {
	// 设置所有服务为健康状态
	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	s.healthServer.SetServingStatus("AgentService", grpc_health_v1.HealthCheckResponse_SERVING)
	s.healthServer.SetServingStatus("WorkflowService", grpc_health_v1.HealthCheckResponse_SERVING)
	s.healthServer.SetServingStatus("ModelService", grpc_health_v1.HealthCheckResponse_SERVING)

	s.logger.Info("Starting gRPC server",
		"address", s.listener.Addr().String(),
		"tls_enabled", s.config.TLS.Enabled,
		"reflection_enabled", s.config.EnableReflection)

	// 启动服务器
	if err := s.server.Serve(s.listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Stop 优雅停止 gRPC 服务器
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping gRPC server")

	// 设置所有服务为非健康状态
	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	s.healthServer.SetServingStatus("AgentService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	s.healthServer.SetServingStatus("WorkflowService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	s.healthServer.SetServingStatus("ModelService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// 创建停止信号通道
	stopped := make(chan struct{})

	// 优雅停止
	go func() {
		s.server.GracefulStop()
		close(stopped)
	}()

	// 等待停止完成或超时
	select {
	case <-stopped:
		s.logger.Info("gRPC server stopped gracefully")
		return nil
	case <-ctx.Done():
		s.logger.Warn("gRPC server stop timeout, forcing stop")
		s.server.Stop()
		return ctx.Err()
	}
}

// GetPort 获取服务器监听端口
func (s *Server) GetPort() int {
	if tcpAddr, ok := s.listener.Addr().(*net.TCPAddr); ok {
		return tcpAddr.Port
	}
	return s.config.Port
}

// recoveryInterceptor panic 恢复拦截器
func (s *Server) recoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("Panic recovered in gRPC handler",
					"method", info.FullMethod,
					"panic", r)
				s.metricsCollector.IncrementCounter("grpc_panic_total", map[string]string{
					"method": info.FullMethod,
				})
				err = fmt.Errorf("internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// loggingInterceptor 日志拦截器
func (s *Server) loggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)

		s.logger.Info("gRPC request completed",
			"method", info.FullMethod,
			"duration_ms", duration.Milliseconds(),
			"error", err)

		return resp, err
	}
}

// tracingInterceptor 追踪拦截器
func (s *Server) tracingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		span, newCtx := s.tracer.StartSpan(ctx, info.FullMethod)
		defer span.End()

		span.SetTag("rpc.system", "grpc")
		span.SetTag("rpc.service", info.FullMethod)

		resp, err := handler(newCtx, req)

		if err != nil {
			span.SetStatus(// trace.StatusError, err.Error())
		} else {
			span.SetStatus(trace.StatusOK, "")
		}

		return resp, err
	}
}

// metricsInterceptor 指标拦截器
func (s *Server) metricsInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		labels := map[string]string{
			"method": info.FullMethod,
			"status": getStatusFromError(err),
		}

		s.metricsCollector.IncrementCounter("grpc_requests_total", labels)
		s.metricsCollector.ObserveDuration("grpc_request_duration_seconds", duration.Seconds(), labels)

		if err != nil {
			s.metricsCollector.IncrementCounter("grpc_errors_total", labels)
		}

		return resp, err
	}
}

// authInterceptor 认证拦截器
func (s *Server) authInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 跳过健康检查和反射服务的认证
		if info.FullMethod == "/grpc.health.v1.Health/Check" ||
			info.FullMethod == "/grpc.health.v1.Health/Watch" ||
			info.FullMethod == "/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo" {
			return handler(ctx, req)
		}

		// TODO: 实现认证逻辑，验证 JWT Token 或 API Key
		// 示例：从 metadata 中提取 token，验证后将用户信息注入 context

		return handler(ctx, req)
	}
}

// validationInterceptor 参数验证拦截器
func (s *Server) validationInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// TODO: 实现参数验证逻辑
		// 可以使用 protoc-gen-validate 生成的验证代码

		return handler(ctx, req)
	}
}

// streamRecoveryInterceptor 流式 panic 恢复拦截器
func (s *Server) streamRecoveryInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("Panic recovered in gRPC stream handler",
					"method", info.FullMethod,
					"panic", r)
				s.metricsCollector.IncrementCounter("grpc_stream_panic_total", map[string]string{
					"method": info.FullMethod,
				})
				err = fmt.Errorf("internal server error")
			}
		}()
		return handler(srv, ss)
	}
}

// streamLoggingInterceptor 流式日志拦截器
func (s *Server) streamLoggingInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		err := handler(srv, ss)

		duration := time.Since(start)

		s.logger.Info("gRPC stream request completed",
			"method", info.FullMethod,
			"duration_ms", duration.Milliseconds(),
			"error", err)

		return err
	}
}

// streamTracingInterceptor 流式追踪拦截器
func (s *Server) streamTracingInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		span, _ := s.tracer.StartSpan(ss.Context(), info.FullMethod)
		defer span.End()

		span.SetTag("rpc.system", "grpc")
		span.SetTag("rpc.service", info.FullMethod)
		span.SetTag("stream", true)

		err := handler(srv, ss)

		if err != nil {
			span.SetStatus(// trace.StatusError, err.Error())
		} else {
			span.SetStatus(trace.StatusOK, "")
		}

		return err
	}
}

// streamMetricsInterceptor 流式指标拦截器
func (s *Server) streamMetricsInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		err := handler(srv, ss)

		duration := time.Since(start)
		labels := map[string]string{
			"method": info.FullMethod,
			"status": getStatusFromError(err),
			"stream": "true",
		}

		s.metricsCollector.IncrementCounter("grpc_stream_requests_total", labels)
		s.metricsCollector.ObserveDuration("grpc_stream_duration_seconds", duration.Seconds(), labels)

		if err != nil {
			s.metricsCollector.IncrementCounter("grpc_stream_errors_total", labels)
		}

		return err
	}
}

// streamAuthInterceptor 流式认证拦截器
func (s *Server) streamAuthInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// 跳过健康检查的认证
		if info.FullMethod == "/grpc.health.v1.Health/Watch" {
			return handler(srv, ss)
		}

		// TODO: 实现认证逻辑

		return handler(srv, ss)
	}
}

// getStatusFromError 从错误获取状态码字符串
func getStatusFromError(err error) string {
	if err == nil {
		return "ok"
	}
	// TODO: 根据 gRPC status code 返回具体状态
	return "error"
}

//Personal.AI order the ending
