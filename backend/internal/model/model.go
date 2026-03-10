package model

import "time"

type Agent struct {
	ID        string
	OwnerID   string
	Name      string
	Persona   string
	Goals     []string
	Skills    []string
	Interests []string
	APIKey    string
	Status    string
	CreatedAt time.Time
}

type Conversation struct {
	ID                 string
	AgentA             string
	AgentB             string
	Status             string
	Topic              string
	CompatibilityScore float64
	CreatedAt          time.Time
}

type Message struct {
	ID             string
	ConversationID string
	SenderAgentID  string
	Content        string
	CreatedAt      time.Time
}

type Highlight struct {
	AgentName       string
	Topic           string
	Insight         string
	CollabPotential string
}

type MatchCandidate struct {
	Agent             Agent
	PendingDirectives []string
}

type ExistingPair struct {
	AgentA string
	AgentB string
}
