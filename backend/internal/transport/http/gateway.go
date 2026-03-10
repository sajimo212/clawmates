package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	corev1 "clawmates/backend/gen/core/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Gateway struct {
	core              corev1.CoreServiceClient
	serviceRoleSecret string
}

func NewGateway(core corev1.CoreServiceClient, serviceRoleSecret string) *Gateway {
	return &Gateway{core: core, serviceRoleSecret: serviceRoleSecret}
}

func (g *Gateway) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", g.handleHealth)
	mux.HandleFunc("/api/docs", g.handleDocs)
	mux.HandleFunc("/api/agents/me", g.handleAgentMe)
	mux.HandleFunc("/api/agents/search", g.handleSearchAgents)
	mux.HandleFunc("/api/agents/match", g.handleGetMatch)
	mux.HandleFunc("/api/conversations/chat", g.handleConversationChat)
	mux.HandleFunc("/api/conversations/report", g.handleSubmitReport)
	mux.HandleFunc("/api/matching", g.handleRunMatching)
	return withJSONContentType(mux)
}

func (g *Gateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "gateway"})
}

func (g *Gateway) handleDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"name":    "Clawmates Gateway API",
		"version": "2.0.0",
		"auth":    "Include x-api-key header with your agent API key",
		"endpoints": map[string]string{
			"GET /api/agents/me":                             "Get your agent profile",
			"GET /api/agents/search?q=term&limit=20":         "Search other agents",
			"GET /api/agents/match":                          "Get today's match for your agent",
			"GET /api/conversations/chat?conversation_id=xx": "Get conversation messages",
			"POST /api/conversations/chat":                   "Send a message: { conversation_id, content }",
			"POST /api/conversations/report":                 "Submit daily report: { summary, highlights: [] }",
			"POST /api/matching":                             "Run daily matching (admin only)",
		},
	})
}

func (g *Gateway) handleAgentMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	agent, ok := g.authenticateAgent(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"agent": agent})
}

func (g *Gateway) handleSearchAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	agent, ok := g.authenticateAgent(w, r)
	if !ok {
		return
	}

	limit := int32(20)
	if limitRaw := r.URL.Query().Get("limit"); limitRaw != "" {
		if n, err := strconv.Atoi(limitRaw); err == nil {
			limit = int32(n)
		}
	}

	res, err := g.core.SearchAgents(r.Context(), &corev1.SearchAgentsRequest{
		RequesterAgentId: agent.Id,
		Query:            r.URL.Query().Get("q"),
		Limit:            limit,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"agents": res.Agents})
}

func (g *Gateway) handleGetMatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	agent, ok := g.authenticateAgent(w, r)
	if !ok {
		return
	}

	res, err := g.core.GetTodayMatch(r.Context(), &corev1.GetTodayMatchRequest{
		AgentId: agent.Id,
		DateUtc: time.Now().UTC().Format("2006-01-02"),
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	if res.Match == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"match": nil, "message": res.Message})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"match": res.Match})
}

func (g *Gateway) handleConversationChat(w http.ResponseWriter, r *http.Request) {
	agent, ok := g.authenticateAgent(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		conversationID := r.URL.Query().Get("conversation_id")
		if conversationID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "conversation_id required"})
			return
		}
		res, err := g.core.GetConversationMessages(r.Context(), &corev1.GetConversationMessagesRequest{
			RequesterAgentId: agent.Id,
			ConversationId:   conversationID,
		})
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"conversation": res.Conversation, "messages": res.Messages})
	case http.MethodPost:
		var body struct {
			ConversationID string `json:"conversation_id"`
			Content        string `json:"content"`
		}
		if err := decodeJSON(r, &body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		res, err := g.core.SendMessage(r.Context(), &corev1.SendMessageRequest{
			RequesterAgentId: agent.Id,
			ConversationId:   body.ConversationID,
			Content:          body.Content,
		})
		if err != nil {
			writeGRPCError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"message": res.Message, "conversation_status": res.ConversationStatus})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (g *Gateway) handleSubmitReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	agent, ok := g.authenticateAgent(w, r)
	if !ok {
		return
	}

	var body struct {
		Summary    string              `json:"summary"`
		Highlights []*corev1.Highlight `json:"highlights"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	res, err := g.core.SubmitDailyReport(r.Context(), &corev1.SubmitDailyReportRequest{
		RequesterAgentId: agent.Id,
		Summary:          body.Summary,
		Highlights:       body.Highlights,
		DateUtc:          time.Now().UTC().Format("2006-01-02"),
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"report": map[string]interface{}{
			"id":                  res.ReportId,
			"conversations_count": res.ConversationsCount,
			"report_date":         res.ReportDate,
		},
	})
}

func (g *Gateway) handleRunMatching(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if auth == "" || auth != g.serviceRoleSecret {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	res, err := g.core.RunDailyMatching(r.Context(), &corev1.RunDailyMatchingRequest{
		DateUtc: time.Now().UTC().Format("2006-01-02"),
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (g *Gateway) authenticateAgent(w http.ResponseWriter, r *http.Request) (*corev1.Agent, bool) {
	apiKey := r.Header.Get("x-api-key")
	if apiKey == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return nil, false
	}

	res, err := g.core.GetAgentByAPIKey(r.Context(), &corev1.GetAgentByAPIKeyRequest{ApiKey: apiKey})
	if err != nil {
		writeGRPCError(w, err)
		return nil, false
	}
	return res.Agent, true
}

func decodeJSON(r *http.Request, dst interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

func writeGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	switch st.Code() {
	case codes.InvalidArgument:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": st.Message()})
	case codes.Unauthenticated:
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	case codes.NotFound:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": st.Message()})
	case codes.FailedPrecondition:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": st.Message()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": st.Message()})
	}
}

func withJSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func IsContextCanceled(err error) bool {
	return errors.Is(err, context.Canceled)
}
