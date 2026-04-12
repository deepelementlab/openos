package grpc

import "context"

type AgentService interface {
	CreateAgent(ctx context.Context, req *CreateAgentRequest) (*AgentInfo, error)
	GetAgent(ctx context.Context, req *GetAgentRequest) (*AgentInfo, error)
	ListAgents(ctx context.Context, req *ListAgentsRequest) (*ListAgentsResponse, error)
	UpdateAgent(ctx context.Context, req *UpdateAgentRequest) (*AgentInfo, error)
	DeleteAgent(ctx context.Context, req *DeleteAgentRequest) (*Empty, error)
}
