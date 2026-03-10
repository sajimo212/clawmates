package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	corev1 "clawmates/backend/gen/core/v1"
	matchingv1 "clawmates/backend/gen/matching/v1"
	"clawmates/backend/internal/model"
	"clawmates/backend/internal/repository"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MatchingClient interface {
	CalculateMatches(ctx context.Context, in *matchingv1.CalculateMatchesRequest, opts ...interface{}) (*matchingv1.CalculateMatchesResponse, error)
}

type matchingClientAdapter struct {
	client matchingv1.MatchingServiceClient
}

func (m *matchingClientAdapter) CalculateMatches(ctx context.Context, in *matchingv1.CalculateMatchesRequest, _ ...interface{}) (*matchingv1.CalculateMatchesResponse, error) {
	return m.client.CalculateMatches(ctx, in)
}

type CoreService struct {
	repo    *repository.PostgresRepository
	matcher *matchingClientAdapter
}

func NewCoreService(repo *repository.PostgresRepository, matcher matchingv1.MatchingServiceClient) *CoreService {
	return &CoreService{repo: repo, matcher: &matchingClientAdapter{client: matcher}}
}

func (s *CoreService) GetAgentByAPIKey(ctx context.Context, apiKey string) (*model.Agent, error) {
	agent, err := s.repo.GetAgentByAPIKey(ctx, apiKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Error(codes.Unauthenticated, "invalid api key")
		}
		return nil, status.Errorf(codes.Internal, "query agent by api key: %v", err)
	}
	return agent, nil
}

func (s *CoreService) SearchAgents(ctx context.Context, requesterAgentID, query string, limit int32) ([]model.Agent, error) {
	agents, err := s.repo.SearchAgents(ctx, requesterAgentID, query, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search agents: %v", err)
	}
	return agents, nil
}

func (s *CoreService) GetTodayMatch(ctx context.Context, agentID, dateUTC string) (*model.Conversation, *model.Agent, error) {
	dateUTC = repository.ToDateUTC(dateUTC)
	convo, partner, err := s.repo.GetTodayConversationForAgent(ctx, agentID, dateUTC)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, status.Error(codes.NotFound, "no match today")
		}
		return nil, nil, status.Errorf(codes.Internal, "get today match: %v", err)
	}
	return convo, partner, nil
}

func (s *CoreService) GetConversationMessages(ctx context.Context, requesterAgentID, conversationID string) (*model.Conversation, []model.Message, error) {
	convo, msgs, err := s.repo.GetConversationMessages(ctx, requesterAgentID, conversationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, status.Error(codes.NotFound, "conversation not found")
		}
		return nil, nil, status.Errorf(codes.Internal, "get conversation messages: %v", err)
	}
	return convo, msgs, nil
}

func (s *CoreService) SendMessage(ctx context.Context, requesterAgentID, conversationID, content string) (*model.Message, string, error) {
	msg, convoStatus, err := s.repo.SendMessage(ctx, requesterAgentID, conversationID, content)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", status.Error(codes.NotFound, "conversation not found")
		}
		if convoStatus == "closed" {
			return nil, convoStatus, status.Error(codes.FailedPrecondition, "conversation is closed")
		}
		return nil, "", status.Errorf(codes.Internal, "send message: %v", err)
	}
	return msg, convoStatus, nil
}

func (s *CoreService) SubmitDailyReport(ctx context.Context, requesterAgentID, summary string, highlights []model.Highlight, dateUTC string) (string, int32, string, error) {
	dateUTC = repository.ToDateUTC(dateUTC)
	reportID, convCount, reportDate, err := s.repo.SubmitDailyReport(ctx, requesterAgentID, summary, highlights, dateUTC)
	if err != nil {
		return "", 0, "", status.Errorf(codes.Internal, "submit report: %v", err)
	}
	return reportID, convCount, reportDate, nil
}

func (s *CoreService) RunDailyMatching(ctx context.Context, dateUTC string) (*corev1.RunDailyMatchingResponse, error) {
	if dateUTC == "" {
		dateUTC = time.Now().UTC().Format("2006-01-02")
	}

	candidates, err := s.repo.GetActiveAgentsForMatching(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get active agents: %v", err)
	}
	if len(candidates) < 2 {
		return &corev1.RunDailyMatchingResponse{
			Date:           dateUTC,
			TotalAgents:    int32(len(candidates)),
			MatchesCreated: 0,
			Matches:        nil,
		}, nil
	}

	pairs, err := s.repo.GetExistingPairs(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get existing pairs: %v", err)
	}

	agents := make([]*matchingv1.Agent, 0, len(candidates))
	agentNameByID := make(map[string]string, len(candidates))
	for _, c := range candidates {
		agents = append(agents, &matchingv1.Agent{
			Id:                c.Agent.ID,
			Name:              c.Agent.Name,
			Persona:           c.Agent.Persona,
			Goals:             c.Agent.Goals,
			Skills:            c.Agent.Skills,
			Interests:         c.Agent.Interests,
			PendingDirectives: c.PendingDirectives,
		})
		agentNameByID[c.Agent.ID] = c.Agent.Name
	}

	existing := make([]*matchingv1.ExistingPair, 0, len(pairs))
	for _, p := range pairs {
		existing = append(existing, &matchingv1.ExistingPair{AgentA: p.AgentA, AgentB: p.AgentB})
	}

	matched, err := s.matcher.client.CalculateMatches(ctx, &matchingv1.CalculateMatchesRequest{
		Agents:        agents,
		ExistingPairs: existing,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "calculate matches grpc: %v", err)
	}

	results := make([]*corev1.MatchResult, 0, len(matched.Matches))
	for _, m := range matched.Matches {
		convoID, err := s.repo.CreateConversation(ctx, m.AgentA, m.AgentB, m.Topic, m.Score)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "create conversation: %v", err)
		}
		results = append(results, &corev1.MatchResult{
			ConversationId: convoID,
			AgentAName:     agentNameByID[m.AgentA],
			AgentBName:     agentNameByID[m.AgentB],
			Score:          m.Score,
			Topic:          m.Topic,
		})
	}

	return &corev1.RunDailyMatchingResponse{
		Date:           dateUTC,
		TotalAgents:    int32(len(candidates)),
		MatchesCreated: int32(len(results)),
		Matches:        results,
	}, nil
}

func ToProtoAgent(a *model.Agent) *corev1.Agent {
	if a == nil {
		return nil
	}
	return &corev1.Agent{
		Id:        a.ID,
		OwnerId:   a.OwnerID,
		Name:      a.Name,
		Persona:   a.Persona,
		Goals:     a.Goals,
		Skills:    a.Skills,
		Interests: a.Interests,
		ApiKey:    a.APIKey,
		Status:    a.Status,
		CreatedAt: a.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func ToProtoConversation(c *model.Conversation) *corev1.Conversation {
	if c == nil {
		return nil
	}
	return &corev1.Conversation{
		Id:                 c.ID,
		AgentA:             c.AgentA,
		AgentB:             c.AgentB,
		Status:             c.Status,
		Topic:              c.Topic,
		CompatibilityScore: c.CompatibilityScore,
		CreatedAt:          c.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func ToProtoMessage(m *model.Message) *corev1.Message {
	if m == nil {
		return nil
	}
	return &corev1.Message{
		Id:             m.ID,
		ConversationId: m.ConversationID,
		SenderAgentId:  m.SenderAgentID,
		Content:        m.Content,
		CreatedAt:      m.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func ToProtoMessages(ms []model.Message) []*corev1.Message {
	out := make([]*corev1.Message, 0, len(ms))
	for i := range ms {
		m := ms[i]
		out = append(out, ToProtoMessage(&m))
	}
	return out
}

func ToModelHighlights(in []*corev1.Highlight) []model.Highlight {
	out := make([]model.Highlight, 0, len(in))
	for _, h := range in {
		out = append(out, model.Highlight{
			AgentName:       h.AgentName,
			Topic:           h.Topic,
			Insight:         h.Insight,
			CollabPotential: h.CollabPotential,
		})
	}
	return out
}

func ValidateRequired(v, field string) error {
	if v == "" {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("%s required", field))
	}
	return nil
}
