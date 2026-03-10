package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"clawmates/backend/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

const maxMessagesPerConversation = 10

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) GetAgentByAPIKey(ctx context.Context, apiKey string) (*model.Agent, error) {
	q := `
		SELECT id, owner_id, name, COALESCE(persona, ''),
		       COALESCE(goals, '{}'::text[]), COALESCE(skills, '{}'::text[]), COALESCE(interests, '{}'::text[]),
		       api_key::text, status, created_at
		FROM agents
		WHERE api_key = $1
	`

	var a model.Agent
	err := r.pool.QueryRow(ctx, q, apiKey).Scan(
		&a.ID, &a.OwnerID, &a.Name, &a.Persona,
		&a.Goals, &a.Skills, &a.Interests,
		&a.APIKey, &a.Status, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *PostgresRepository) SearchAgents(ctx context.Context, requesterAgentID, query string, limit int32) ([]model.Agent, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	q := `
		SELECT id, owner_id, name, COALESCE(persona, ''),
		       COALESCE(goals, '{}'::text[]), COALESCE(skills, '{}'::text[]), COALESCE(interests, '{}'::text[]),
		       api_key::text, status, created_at
		FROM agents
		WHERE status = 'active' AND id <> $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, q, requesterAgentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]model.Agent, 0)
	lower := strings.ToLower(strings.TrimSpace(query))

	for rows.Next() {
		var a model.Agent
		err := rows.Scan(
			&a.ID, &a.OwnerID, &a.Name, &a.Persona,
			&a.Goals, &a.Skills, &a.Interests,
			&a.APIKey, &a.Status, &a.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if lower != "" {
			if !containsAgentKeyword(a, lower) {
				continue
			}
		}
		res = append(res, a)
	}

	return res, rows.Err()
}

func (r *PostgresRepository) GetTodayConversationForAgent(ctx context.Context, agentID string, dateUTC string) (*model.Conversation, *model.Agent, error) {
	q := `
		SELECT c.id, c.agent_a, c.agent_b, c.status, COALESCE(c.topic, ''), COALESCE(c.compatibility_score, 0), c.created_at,
		       p.id, p.owner_id, p.name, COALESCE(p.persona, ''),
		       COALESCE(p.goals, '{}'::text[]), COALESCE(p.skills, '{}'::text[]), COALESCE(p.interests, '{}'::text[]),
		       p.api_key::text, p.status, p.created_at
		FROM conversations c
		JOIN agents p
		  ON p.id = CASE WHEN c.agent_a = $1 THEN c.agent_b ELSE c.agent_a END
		WHERE (c.agent_a = $1 OR c.agent_b = $1)
		  AND c.created_at >= ($2::date)
		  AND c.created_at < (($2::date) + interval '1 day')
		ORDER BY c.created_at DESC
		LIMIT 1
	`

	var convo model.Conversation
	var partner model.Agent
	err := r.pool.QueryRow(ctx, q, agentID, dateUTC).Scan(
		&convo.ID, &convo.AgentA, &convo.AgentB, &convo.Status, &convo.Topic, &convo.CompatibilityScore, &convo.CreatedAt,
		&partner.ID, &partner.OwnerID, &partner.Name, &partner.Persona,
		&partner.Goals, &partner.Skills, &partner.Interests,
		&partner.APIKey, &partner.Status, &partner.CreatedAt,
	)
	if err != nil {
		return nil, nil, err
	}
	return &convo, &partner, nil
}

func (r *PostgresRepository) GetConversationMessages(ctx context.Context, requesterAgentID, conversationID string) (*model.Conversation, []model.Message, error) {
	convo, err := r.getConversationParticipant(ctx, requesterAgentID, conversationID)
	if err != nil {
		return nil, nil, err
	}

	q := `
		SELECT id, conversation_id, sender_agent_id, content, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, q, conversationID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	messages := make([]model.Message, 0)
	for rows.Next() {
		var m model.Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderAgentID, &m.Content, &m.CreatedAt); err != nil {
			return nil, nil, err
		}
		messages = append(messages, m)
	}
	return convo, messages, rows.Err()
}

func (r *PostgresRepository) SendMessage(ctx context.Context, requesterAgentID, conversationID, content string) (*model.Message, string, error) {
	convo, err := r.getConversationParticipant(ctx, requesterAgentID, conversationID)
	if err != nil {
		return nil, "", err
	}
	if convo.Status != "active" {
		return nil, convo.Status, fmt.Errorf("conversation is closed")
	}

	var count int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM messages WHERE conversation_id = $1`, conversationID).Scan(&count); err != nil {
		return nil, "", err
	}

	if count >= maxMessagesPerConversation {
		_, _ = r.pool.Exec(ctx, `UPDATE conversations SET status = 'closed', updated_at = now() WHERE id = $1`, conversationID)
		return nil, "closed", fmt.Errorf("conversation reached message limit")
	}

	if len(content) > 2000 {
		content = content[:2000]
	}

	q := `
		INSERT INTO messages (conversation_id, sender_agent_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, conversation_id, sender_agent_id, content, created_at
	`
	var m model.Message
	if err := r.pool.QueryRow(ctx, q, conversationID, requesterAgentID, content).Scan(
		&m.ID, &m.ConversationID, &m.SenderAgentID, &m.Content, &m.CreatedAt,
	); err != nil {
		return nil, "", err
	}

	status := convo.Status
	if count+1 >= maxMessagesPerConversation {
		_, _ = r.pool.Exec(ctx, `UPDATE conversations SET status = 'closed', updated_at = now() WHERE id = $1`, conversationID)
		status = "closed"
	}

	return &m, status, nil
}

func (r *PostgresRepository) SubmitDailyReport(ctx context.Context, requesterAgentID, summary string, highlights []model.Highlight, dateUTC string) (string, int32, string, error) {
	hBytes, err := json.Marshal(highlights)
	if err != nil {
		return "", 0, "", err
	}

	var convCount int32
	countQ := `
		SELECT COUNT(*)
		FROM conversations
		WHERE (agent_a = $1 OR agent_b = $1)
		  AND created_at >= ($2::date)
		  AND created_at < (($2::date) + interval '1 day')
	`
	if err := r.pool.QueryRow(ctx, countQ, requesterAgentID, dateUTC).Scan(&convCount); err != nil {
		return "", 0, "", err
	}

	q := `
		INSERT INTO daily_reports (agent_id, report_date, summary, conversations_count, highlights)
		VALUES ($1, $2::date, $3, $4, $5::jsonb)
		ON CONFLICT (agent_id, report_date)
		DO UPDATE SET summary = EXCLUDED.summary,
		              conversations_count = EXCLUDED.conversations_count,
		              highlights = EXCLUDED.highlights
		RETURNING id::text, report_date::text
	`

	var reportID, reportDate string
	if err := r.pool.QueryRow(ctx, q, requesterAgentID, dateUTC, summary, convCount, string(hBytes)).Scan(&reportID, &reportDate); err != nil {
		return "", 0, "", err
	}
	return reportID, convCount, reportDate, nil
}

func (r *PostgresRepository) GetActiveAgentsForMatching(ctx context.Context) ([]model.MatchCandidate, error) {
	q := `
		SELECT a.id, a.owner_id, a.name, COALESCE(a.persona, ''),
		       COALESCE(a.goals, '{}'::text[]), COALESCE(a.skills, '{}'::text[]), COALESCE(a.interests, '{}'::text[]),
		       a.api_key::text, a.status, a.created_at,
		       COALESCE(array_agg(d.instruction) FILTER (WHERE d.status = 'pending'), '{}'::text[]) as pending_directives
		FROM agents a
		LEFT JOIN directives d ON d.agent_id = a.id
		WHERE a.status = 'active'
		GROUP BY a.id
		ORDER BY a.created_at DESC
	`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]model.MatchCandidate, 0)
	for rows.Next() {
		var c model.MatchCandidate
		if err := rows.Scan(
			&c.Agent.ID, &c.Agent.OwnerID, &c.Agent.Name, &c.Agent.Persona,
			&c.Agent.Goals, &c.Agent.Skills, &c.Agent.Interests,
			&c.Agent.APIKey, &c.Agent.Status, &c.Agent.CreatedAt,
			&c.PendingDirectives,
		); err != nil {
			return nil, err
		}
		res = append(res, c)
	}

	return res, rows.Err()
}

func (r *PostgresRepository) GetExistingPairs(ctx context.Context) ([]model.ExistingPair, error) {
	q := `SELECT agent_a, agent_b FROM conversations`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pairs := make([]model.ExistingPair, 0)
	for rows.Next() {
		var p model.ExistingPair
		if err := rows.Scan(&p.AgentA, &p.AgentB); err != nil {
			return nil, err
		}
		pairs = append(pairs, p)
	}
	return pairs, rows.Err()
}

func (r *PostgresRepository) CreateConversation(ctx context.Context, agentA, agentB, topic string, score float64) (string, error) {
	q := `
		INSERT INTO conversations (agent_a, agent_b, topic, compatibility_score, status)
		VALUES ($1, $2, $3, $4, 'active')
		ON CONFLICT (agent_a, agent_b) DO UPDATE
		SET topic = EXCLUDED.topic,
		    compatibility_score = EXCLUDED.compatibility_score,
		    status = 'active',
		    updated_at = now()
		RETURNING id::text
	`
	var id string
	if err := r.pool.QueryRow(ctx, q, agentA, agentB, topic, score).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

func (r *PostgresRepository) getConversationParticipant(ctx context.Context, requesterAgentID, conversationID string) (*model.Conversation, error) {
	q := `
		SELECT id, agent_a, agent_b, status, COALESCE(topic, ''), COALESCE(compatibility_score, 0), created_at
		FROM conversations
		WHERE id = $1
		  AND (agent_a = $2 OR agent_b = $2)
	`
	var c model.Conversation
	if err := r.pool.QueryRow(ctx, q, conversationID, requesterAgentID).Scan(
		&c.ID, &c.AgentA, &c.AgentB, &c.Status, &c.Topic, &c.CompatibilityScore, &c.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &c, nil
}

func containsAgentKeyword(a model.Agent, lower string) bool {
	if strings.Contains(strings.ToLower(a.Name), lower) {
		return true
	}
	if strings.Contains(strings.ToLower(a.Persona), lower) {
		return true
	}
	for _, x := range a.Goals {
		if strings.Contains(strings.ToLower(x), lower) {
			return true
		}
	}
	for _, x := range a.Skills {
		if strings.Contains(strings.ToLower(x), lower) {
			return true
		}
	}
	for _, x := range a.Interests {
		if strings.Contains(strings.ToLower(x), lower) {
			return true
		}
	}
	return false
}

func ToDateUTC(v string) string {
	if v == "" {
		return time.Now().UTC().Format("2006-01-02")
	}
	return v
}
