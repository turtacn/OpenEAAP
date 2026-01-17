// internal/api/grpc/handler/agent_grpc.go
package handler

import (
	"context"
	"fmt"
	"io"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	// TODO: generate proto files
	//"github.com/openeeap/openeeap/api/proto"
	"github.com/openeeap/openeeap/internal/app/dto"
	"github.com/openeeap/openeeap/internal/app/service"
	"github.com/openeeap/openeeap/internal/domain/agent"
	"github.com/openeeap/openeeap/internal/observability/logging"
	"github.com/openeeap/openeeap/internal/observability/trace"
	pkgerrors "github.com/openeeap/openeeap/pkg/errors"
	"github.com/openeeap/openeeap/pkg/types"
)

// AgentGRPCHandler implements the gRPC AgentService interface
type AgentGRPCHandler struct {
	proto.UnimplementedAgentServiceServer
	agentService *service.AgentService
	logger       logging.Logger
	tracer       trace.Tracer
}

// NewAgentGRPCHandler creates a new agent gRPC handler instance
func NewAgentGRPCHandler(
	agentService *service.AgentService,
	logger logging.Logger,
	tracer trace.Tracer,
) *AgentGRPCHandler {
	return &AgentGRPCHandler{
		agentService: agentService,
		logger:       logger,
		tracer:       tracer,
	}
}

// CreateAgent handles agent creation requests
func (h *AgentGRPCHandler) CreateAgent(ctx context.Context, req *proto.CreateAgentRequest) (*proto.Agent, error) {
	span := h.tracer.StartSpan(ctx, "AgentGRPCHandler.CreateAgent")
	defer span.End()

	h.logger.Info("creating agent via gRPC", map[string]interface{}{
		"name":         req.Name,
		"runtime_type": req.RuntimeType,
	})

	// Validate request
	if err := h.validateCreateAgentRequest(req); err != nil {
		span.SetStatus(// trace.StatusError, err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Convert proto to DTO
	createDTO := &dto.CreateAgentRequest{
		Name:        req.Name,
		Description: req.Description,
		RuntimeType: types.RuntimeType(req.RuntimeType),
		Config:      h.protoConfigToMap(req.Config),
		Tools:       req.Tools,
		Metadata:    req.Metadata,
	}

	// Call application service
	agentResp, err := h.agentService.CreateAgent(ctx, createDTO)
	if err != nil {
		span.SetStatus(// trace.StatusError, err.Error())
		return nil, h.handleError(err)
	}

	// Convert DTO to proto response
	protoAgent := h.agentDTOToProto(agentResp)

	h.logger.Info("agent created successfully", map[string]interface{}{
		"agent_id": protoAgent.Id,
		"name":     protoAgent.Name,
	})

	return protoAgent, nil
}

// GetAgent retrieves an agent by ID
func (h *AgentGRPCHandler) GetAgent(ctx context.Context, req *proto.GetAgentRequest) (*proto.Agent, error) {
	span := h.tracer.StartSpan(ctx, "AgentGRPCHandler.GetAgent")
	defer span.End()

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "agent ID is required")
	}

	agentResp, err := h.agentService.GetAgent(ctx, req.Id)
	if err != nil {
		span.SetStatus(// trace.StatusError, err.Error())
		return nil, h.handleError(err)
	}

	return h.agentDTOToProto(agentResp), nil
}

// ListAgents returns a paginated list of agents
func (h *AgentGRPCHandler) ListAgents(ctx context.Context, req *proto.ListAgentsRequest) (*proto.ListAgentsResponse, error) {
	span := h.tracer.StartSpan(ctx, "AgentGRPCHandler.ListAgents")
	defer span.End()

	// Default pagination
	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	listDTO := &dto.ListAgentsRequest{
		Page:        page,
		PageSize:    pageSize,
		Status:      req.Status,
		RuntimeType: req.RuntimeType,
		SortBy:      req.SortBy,
		SortOrder:   req.SortOrder,
	}

	agentsResp, total, err := h.agentService.ListAgents(ctx, listDTO)
	if err != nil {
		span.SetStatus(// trace.StatusError, err.Error())
		return nil, h.handleError(err)
	}

	protoAgents := make([]*proto.Agent, len(agentsResp))
	for i, agentDTO := range agentsResp {
		protoAgents[i] = h.agentDTOToProto(agentDTO)
	}

	return &proto.ListAgentsResponse{
		Agents: protoAgents,
		Total:  int32(total),
		Page:   int32(page),
	}, nil
}

// UpdateAgent updates an existing agent
func (h *AgentGRPCHandler) UpdateAgent(ctx context.Context, req *proto.UpdateAgentRequest) (*proto.Agent, error) {
	span := h.tracer.StartSpan(ctx, "AgentGRPCHandler.UpdateAgent")
	defer span.End()

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "agent ID is required")
	}

	updateDTO := &dto.UpdateAgentRequest{
		ID:          req.Id,
		Name:        req.Name,
		Description: req.Description,
		Config:      h.protoConfigToMap(req.Config),
		Tools:       req.Tools,
		Metadata:    req.Metadata,
	}

	agentResp, err := h.agentService.UpdateAgent(ctx, updateDTO)
	if err != nil {
		span.SetStatus(// trace.StatusError, err.Error())
		return nil, h.handleError(err)
	}

	h.logger.Info("agent updated successfully", map[string]interface{}{
		"agent_id": req.Id,
	})

	return h.agentDTOToProto(agentResp), nil
}

// DeleteAgent deletes an agent
func (h *AgentGRPCHandler) DeleteAgent(ctx context.Context, req *proto.DeleteAgentRequest) (*emptypb.Empty, error) {
	span := h.tracer.StartSpan(ctx, "AgentGRPCHandler.DeleteAgent")
	defer span.End()

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "agent ID is required")
	}

	if err := h.agentService.DeleteAgent(ctx, req.Id); err != nil {
		span.SetStatus(// trace.StatusError, err.Error())
		return nil, h.handleError(err)
	}

	h.logger.Info("agent deleted successfully", map[string]interface{}{
		"agent_id": req.Id,
	})

	return &emptypb.Empty{}, nil
}

// ExecuteAgent executes an agent with given input (unary)
func (h *AgentGRPCHandler) ExecuteAgent(ctx context.Context, req *proto.ExecuteAgentRequest) (*proto.ExecuteAgentResponse, error) {
	span := h.tracer.StartSpan(ctx, "AgentGRPCHandler.ExecuteAgent")
	defer span.End()

	if req.AgentId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent ID is required")
	}
	if req.Input == "" {
		return nil, status.Error(codes.InvalidArgument, "input is required")
	}

	executeDTO := &dto.ExecuteAgentRequest{
		AgentID:  req.AgentId,
		Input:    req.Input,
		Context:  req.Context,
		Metadata: req.Metadata,
	}

	result, err := h.agentService.ExecuteAgent(ctx, executeDTO)
	if err != nil {
		span.SetStatus(// trace.StatusError, err.Error())
		return nil, h.handleError(err)
	}

	return &proto.ExecuteAgentResponse{
		ExecutionId: result.ExecutionID,
		Output:      result.Output,
		Status:      result.Status,
		Metadata:    result.Metadata,
		ExecutedAt:  timestamppb.New(result.ExecutedAt),
	}, nil
}

// ExecuteAgentStream executes an agent with streaming response
func (h *AgentGRPCHandler) ExecuteAgentStream(req *proto.ExecuteAgentRequest, stream proto.AgentService_ExecuteAgentStreamServer) error {
	ctx := stream.Context()
	span := h.tracer.StartSpan(ctx, "AgentGRPCHandler.ExecuteAgentStream")
	defer span.End()

	if req.AgentId == "" {
		return status.Error(codes.InvalidArgument, "agent ID is required")
	}
	if req.Input == "" {
		return status.Error(codes.InvalidArgument, "input is required")
	}

	h.logger.Info("starting agent streaming execution", map[string]interface{}{
		"agent_id": req.AgentId,
	})

	executeDTO := &dto.ExecuteAgentRequest{
		AgentID:  req.AgentId,
		Input:    req.Input,
		Context:  req.Context,
		Metadata: req.Metadata,
	}

	// Create streaming channel
	streamChan := make(chan *dto.StreamChunk, 10)
	errChan := make(chan error, 1)

	// Start execution in goroutine
	go func() {
		defer close(streamChan)
		defer close(errChan)

		if err := h.agentService.ExecuteAgentStream(ctx, executeDTO, streamChan); err != nil {
			errChan <- err
		}
	}()

	// Stream responses to client
	for {
		select {
		case <-ctx.Done():
			h.logger.Warn("client disconnected from stream", nil)
			return status.Error(codes.Canceled, "client disconnected")

		case err := <-errChan:
			if err != nil {
				span.SetStatus(// trace.StatusError, err.Error())
				return h.handleError(err)
			}

		case chunk, ok := <-streamChan:
			if !ok {
				// Stream completed successfully
				h.logger.Info("agent streaming execution completed", map[string]interface{}{
					"agent_id": req.AgentId,
				})
				return nil
			}

			// Send chunk to client
			protoChunk := &proto.StreamChunk{
				Type:      chunk.Type,
				Content:   chunk.Content,
				Metadata:  chunk.Metadata,
				Timestamp: timestamppb.New(chunk.Timestamp),
			}

			if err := stream.Send(protoChunk); err != nil {
				span.SetStatus(// trace.StatusError, err.Error())
				h.logger.Error("failed to send stream chunk", err, nil)
				return status.Error(codes.Internal, "failed to send stream chunk")
			}
		}
	}
}

// TestAgent tests an agent with sample input
func (h *AgentGRPCHandler) TestAgent(ctx context.Context, req *proto.TestAgentRequest) (*proto.TestAgentResponse, error) {
	span := h.tracer.StartSpan(ctx, "AgentGRPCHandler.TestAgent")
	defer span.End()

	if req.AgentId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent ID is required")
	}

	testDTO := &dto.TestAgentRequest{
		AgentID:     req.AgentId,
		TestInput:   req.TestInput,
		TestContext: req.TestContext,
	}

	result, err := h.agentService.TestAgent(ctx, testDTO)
	if err != nil {
		span.SetStatus(// trace.StatusError, err.Error())
		return nil, h.handleError(err)
	}

	return &proto.TestAgentResponse{
		Success:      result.Success,
		Output:       result.Output,
		ErrorMessage: result.ErrorMessage,
		Duration:     result.Duration.Milliseconds(),
		Metadata:     result.Metadata,
	}, nil
}

// validateCreateAgentRequest validates the create agent request
func (h *AgentGRPCHandler) validateCreateAgentRequest(req *proto.CreateAgentRequest) error {
	if req.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if req.RuntimeType == "" {
		return fmt.Errorf("runtime type is required")
	}
	if !types.RuntimeType(req.RuntimeType).Valid() {
		return fmt.Errorf("invalid runtime type: %s", req.RuntimeType)
	}
	return nil
}

// agentDTOToProto converts agent DTO to proto message
func (h *AgentGRPCHandler) agentDTOToProto(agentDTO *dto.AgentResponse) *proto.Agent {
	return &proto.Agent{
		Id:          agentDTO.ID,
		Name:        agentDTO.Name,
		Description: agentDTO.Description,
		RuntimeType: string(agentDTO.RuntimeType),
		Config:      h.mapToProtoConfig(agentDTO.Config),
		Tools:       agentDTO.Tools,
		Status:      string(agentDTO.Status),
		Metadata:    agentDTO.Metadata,
		CreatedAt:   timestamppb.New(agentDTO.CreatedAt),
		UpdatedAt:   timestamppb.New(agentDTO.UpdatedAt),
	}
}

// protoConfigToMap converts proto config to map
func (h *AgentGRPCHandler) protoConfigToMap(config *proto.AgentConfig) map[string]interface{} {
	if config == nil {
		return nil
	}

	result := make(map[string]interface{})
	result["model"] = config.Model
	result["temperature"] = config.Temperature
	result["max_tokens"] = config.MaxTokens
	result["timeout"] = config.Timeout

	if len(config.Prompts) > 0 {
		result["prompts"] = config.Prompts
	}
	if len(config.Parameters) > 0 {
		result["parameters"] = config.Parameters
	}

	return result
}

// mapToProtoConfig converts map to proto config
func (h *AgentGRPCHandler) mapToProtoConfig(config map[string]interface{}) *proto.AgentConfig {
	if config == nil {
		return nil
	}

	protoConfig := &proto.AgentConfig{
		Prompts:    make(map[string]string),
		Parameters: make(map[string]string),
	}

	if model, ok := config["model"].(string); ok {
		protoConfig.Model = model
	}
	if temp, ok := config["temperature"].(float64); ok {
		protoConfig.Temperature = float32(temp)
	}
	if maxTokens, ok := config["max_tokens"].(int64); ok {
		protoConfig.MaxTokens = int32(maxTokens)
	}
	if timeout, ok := config["timeout"].(int64); ok {
		protoConfig.Timeout = int32(timeout)
	}

	if prompts, ok := config["prompts"].(map[string]string); ok {
		protoConfig.Prompts = prompts
	}
	if params, ok := config["parameters"].(map[string]string); ok {
		protoConfig.Parameters = params
	}

	return protoConfig
}

// handleError converts application errors to gRPC status errors
func (h *AgentGRPCHandler) handleError(err error) error {
	if err == nil {
		return nil
	}

	// Handle custom application errors
	if appErr, ok := err.(*pkgerrors.Error); ok {
		switch appErr.Code {
		case pkgerrors.CodeNotFound:
			return status.Error(codes.NotFound, appErr.Message)
		case pkgerrors.ConflictError:
			return status.Error(codes.AlreadyExists, appErr.Message)
		case pkgerrors.CodeInvalidArgument:
			return status.Error(codes.InvalidArgument, appErr.Message)
		case pkgerrors.ForbiddenError:
			return status.Error(codes.PermissionDenied, appErr.Message)
		case pkgerrors.CodeUnauthenticated:
			return status.Error(codes.Unauthenticated, appErr.Message)
		case pkgerrors.CodeResourceExhausted:
			return status.Error(codes.ResourceExhausted, appErr.Message)
		case pkgerrors.DeadlineError:
			return status.Error(codes.DeadlineExceeded, appErr.Message)
		default:
			return status.Error(codes.Internal, appErr.Message)
		}
	}

	// Handle standard errors
	switch err {
	case agent.ErrAgentNotFound:
		return status.Error(codes.NotFound, "agent not found")
	case agent.ErrAgentAlreadyExists:
		return status.Error(codes.AlreadyExists, "agent already exists")
	case context.DeadlineExceeded:
		return status.Error(codes.DeadlineExceeded, "request timeout")
	case context.Canceled:
		return status.Error(codes.Canceled, "request canceled")
	default:
		h.logger.Error("unhandled error in gRPC handler", err, nil)
		return status.Error(codes.Internal, "internal server error")
	}
}

//Personal.AI order the ending
