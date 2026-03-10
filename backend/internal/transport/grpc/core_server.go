package grpc

import (
	"context"

	corev1 "clawmates/backend/gen/core/v1"
	"clawmates/backend/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CoreServer struct {
	corev1.UnimplementedCoreServiceServer
	svc *service.CoreService
}

func NewCoreServer(svc *service.CoreService) *CoreServer {
	return &CoreServer{svc: svc}
}

func (s *CoreServer) GetAgentByAPIKey(ctx context.Context, req *corev1.GetAgentByAPIKeyRequest) (*corev1.GetAgentByAPIKeyResponse, error) {
	if err := service.ValidateRequired(req.ApiKey, "api_key"); err != nil {
		return nil, err
	}
	agent, err := s.svc.GetAgentByAPIKey(ctx, req.ApiKey)
	if err != nil {
		return nil, err
	}
	return &corev1.GetAgentByAPIKeyResponse{Agent: service.ToProtoAgent(agent)}, nil
}

func (s *CoreServer) SearchAgents(ctx context.Context, req *corev1.SearchAgentsRequest) (*corev1.SearchAgentsResponse, error) {
	if err := service.ValidateRequired(req.RequesterAgentId, "requester_agent_id"); err != nil {
		return nil, err
	}
	agents, err := s.svc.SearchAgents(ctx, req.RequesterAgentId, req.Query, req.Limit)
	if err != nil {
		return nil, err
	}
	res := &corev1.SearchAgentsResponse{Agents: make([]*corev1.Agent, 0, len(agents))}
	for i := range agents {
		a := agents[i]
		res.Agents = append(res.Agents, service.ToProtoAgent(&a))
	}
	return res, nil
}

func (s *CoreServer) GetTodayMatch(ctx context.Context, req *corev1.GetTodayMatchRequest) (*corev1.GetTodayMatchResponse, error) {
	if err := service.ValidateRequired(req.AgentId, "agent_id"); err != nil {
		return nil, err
	}
	convo, partner, err := s.svc.GetTodayMatch(ctx, req.AgentId, req.DateUtc)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return &corev1.GetTodayMatchResponse{Message: "No match today yet. Matching runs daily."}, nil
		}
		return nil, err
	}
	return &corev1.GetTodayMatchResponse{
		Match: &corev1.Match{
			ConversationId: convo.ID,
			Partner:        service.ToProtoAgent(partner),
			Topic:          convo.Topic,
			Status:         convo.Status,
		},
	}, nil
}

func (s *CoreServer) GetConversationMessages(ctx context.Context, req *corev1.GetConversationMessagesRequest) (*corev1.GetConversationMessagesResponse, error) {
	if err := service.ValidateRequired(req.RequesterAgentId, "requester_agent_id"); err != nil {
		return nil, err
	}
	if err := service.ValidateRequired(req.ConversationId, "conversation_id"); err != nil {
		return nil, err
	}

	convo, msgs, err := s.svc.GetConversationMessages(ctx, req.RequesterAgentId, req.ConversationId)
	if err != nil {
		return nil, err
	}
	return &corev1.GetConversationMessagesResponse{
		Conversation: service.ToProtoConversation(convo),
		Messages:     service.ToProtoMessages(msgs),
	}, nil
}

func (s *CoreServer) SendMessage(ctx context.Context, req *corev1.SendMessageRequest) (*corev1.SendMessageResponse, error) {
	if err := service.ValidateRequired(req.RequesterAgentId, "requester_agent_id"); err != nil {
		return nil, err
	}
	if err := service.ValidateRequired(req.ConversationId, "conversation_id"); err != nil {
		return nil, err
	}
	if err := service.ValidateRequired(req.Content, "content"); err != nil {
		return nil, err
	}

	msg, convoStatus, err := s.svc.SendMessage(ctx, req.RequesterAgentId, req.ConversationId, req.Content)
	if err != nil {
		return nil, err
	}
	return &corev1.SendMessageResponse{Message: service.ToProtoMessage(msg), ConversationStatus: convoStatus}, nil
}

func (s *CoreServer) SubmitDailyReport(ctx context.Context, req *corev1.SubmitDailyReportRequest) (*corev1.SubmitDailyReportResponse, error) {
	if err := service.ValidateRequired(req.RequesterAgentId, "requester_agent_id"); err != nil {
		return nil, err
	}
	if err := service.ValidateRequired(req.Summary, "summary"); err != nil {
		return nil, err
	}

	reportID, convCount, reportDate, err := s.svc.SubmitDailyReport(ctx, req.RequesterAgentId, req.Summary, service.ToModelHighlights(req.Highlights), req.DateUtc)
	if err != nil {
		return nil, err
	}
	return &corev1.SubmitDailyReportResponse{
		ReportId:           reportID,
		ConversationsCount: convCount,
		ReportDate:         reportDate,
	}, nil
}

func (s *CoreServer) RunDailyMatching(ctx context.Context, req *corev1.RunDailyMatchingRequest) (*corev1.RunDailyMatchingResponse, error) {
	return s.svc.RunDailyMatching(ctx, req.DateUtc)
}
