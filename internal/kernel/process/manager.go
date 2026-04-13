package process

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager is the Agent Kernel process manager (groups, sessions, namespaces).
type Manager interface {
	CreateGroup(ctx context.Context, leaderID string) (*AgentGroup, error)
	AddToGroup(ctx context.Context, groupID, agentID string) error
	RemoveFromGroup(ctx context.Context, groupID, agentID string) error
	SignalGroup(ctx context.Context, groupID string, sig Signal) error
	GetGroup(ctx context.Context, groupID string) (*AgentGroup, error)

	CreateSession(ctx context.Context, leaderID string) (*AgentSession, error)
	AttachGroupToSession(ctx context.Context, sessionID, groupID string) error
	GetSession(ctx context.Context, sessionID string) (*AgentSession, error)

	CreateNamespace(ctx context.Context) (*ProcessNamespace, error)
	EnterNamespace(ctx context.Context, agentID string, ns *ProcessNamespace) error
	LeaveNamespace(ctx context.Context, agentID string, nsID string) error
}

// InMemoryManager is a reference implementation for tests and single-node control plane.
type InMemoryManager struct {
	mu         sync.RWMutex
	groups     map[string]*AgentGroup
	sessions   map[string]*AgentSession
	namespaces map[string]*ProcessNamespace
	// agentID -> namespaceID
	agentNS map[string]string
	// next virtual PID per namespace
	vpid map[string]int
}

// NewInMemoryManager creates an empty in-memory process manager.
func NewInMemoryManager() *InMemoryManager {
	return &InMemoryManager{
		groups:     make(map[string]*AgentGroup),
		sessions:   make(map[string]*AgentSession),
		namespaces: make(map[string]*ProcessNamespace),
		agentNS:    make(map[string]string),
		vpid:       make(map[string]int),
	}
}

func (m *InMemoryManager) CreateGroup(ctx context.Context, leaderID string) (*AgentGroup, error) {
	if leaderID == "" {
		return nil, fmt.Errorf("process: empty leader id")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	gid := uuid.NewString()
	g := &AgentGroup{
		GroupID:   gid,
		LeaderID:  leaderID,
		Members:   []string{leaderID},
		SessionID: "",
		State:     GroupStateActive,
		CreatedAt: time.Now().UTC(),
	}
	m.groups[gid] = g
	return cloneGroup(g), nil
}

func (m *InMemoryManager) AddToGroup(ctx context.Context, groupID, agentID string) error {
	if groupID == "" || agentID == "" {
		return fmt.Errorf("process: empty id")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.groups[groupID]
	if !ok {
		return fmt.Errorf("process: unknown group %s", groupID)
	}
	for _, x := range g.Members {
		if x == agentID {
			return nil
		}
	}
	g.Members = append(g.Members, agentID)
	return nil
}

func (m *InMemoryManager) RemoveFromGroup(ctx context.Context, groupID, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.groups[groupID]
	if !ok {
		return fmt.Errorf("process: unknown group %s", groupID)
	}
	out := g.Members[:0]
	for _, x := range g.Members {
		if x != agentID {
			out = append(out, x)
		}
	}
	g.Members = out
	if len(g.Members) == 0 {
		g.State = GroupStateStopped
	}
	return nil
}

func (m *InMemoryManager) SignalGroup(ctx context.Context, groupID string, sig Signal) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.groups[groupID]
	if !ok {
		return fmt.Errorf("process: unknown group %s", groupID)
	}
	switch sig {
	case SignalTerminate, SignalKill:
		g.State = GroupStateStopping
	default:
		// no-op beyond state for stub
	}
	return nil
}

func (m *InMemoryManager) GetGroup(ctx context.Context, groupID string) (*AgentGroup, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	g, ok := m.groups[groupID]
	if !ok {
		return nil, fmt.Errorf("process: unknown group %s", groupID)
	}
	return cloneGroup(g), nil
}

func (m *InMemoryManager) CreateSession(ctx context.Context, leaderID string) (*AgentSession, error) {
	if leaderID == "" {
		return nil, fmt.Errorf("process: empty leader id")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	sid := uuid.NewString()
	s := &AgentSession{
		SessionID: sid,
		LeaderID:  leaderID,
		GroupIDs:  nil,
		CreatedAt: time.Now().UTC(),
	}
	m.sessions[sid] = s
	return cloneSession(s), nil
}

func (m *InMemoryManager) AttachGroupToSession(ctx context.Context, sessionID, groupID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return fmt.Errorf("process: unknown session %s", sessionID)
	}
	g, ok := m.groups[groupID]
	if !ok {
		return fmt.Errorf("process: unknown group %s", groupID)
	}
	for _, x := range s.GroupIDs {
		if x == groupID {
			return nil
		}
	}
	s.GroupIDs = append(s.GroupIDs, groupID)
	g.SessionID = sessionID
	return nil
}

func (m *InMemoryManager) GetSession(ctx context.Context, sessionID string) (*AgentSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("process: unknown session %s", sessionID)
	}
	return cloneSession(s), nil
}

func (m *InMemoryManager) CreateNamespace(ctx context.Context) (*ProcessNamespace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	nid := uuid.NewString()
	ns := &ProcessNamespace{
		NamespaceID: nid,
		AgentIDs:    make(map[string]int),
		CreatedAt:   time.Now().UTC(),
	}
	m.namespaces[nid] = ns
	m.vpid[nid] = 100
	return cloneNS(ns), nil
}

func (m *InMemoryManager) EnterNamespace(ctx context.Context, agentID string, ns *ProcessNamespace) error {
	if ns == nil {
		return fmt.Errorf("process: nil namespace")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	base, ok := m.namespaces[ns.NamespaceID]
	if !ok {
		return fmt.Errorf("process: unknown namespace %s", ns.NamespaceID)
	}
	m.agentNS[agentID] = ns.NamespaceID
	m.vpid[ns.NamespaceID]++
	base.AgentIDs[agentID] = m.vpid[ns.NamespaceID]
	return nil
}

func (m *InMemoryManager) LeaveNamespace(ctx context.Context, agentID string, nsID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.agentNS, agentID)
	if ns, ok := m.namespaces[nsID]; ok {
		delete(ns.AgentIDs, agentID)
	}
	return nil
}

func cloneGroup(g *AgentGroup) *AgentGroup {
	cp := *g
	cp.Members = append([]string(nil), g.Members...)
	return &cp
}

func cloneSession(s *AgentSession) *AgentSession {
	cp := *s
	cp.GroupIDs = append([]string(nil), s.GroupIDs...)
	return &cp
}

func cloneNS(n *ProcessNamespace) *ProcessNamespace {
	cp := *n
	cp.AgentIDs = make(map[string]int, len(n.AgentIDs))
	for k, v := range n.AgentIDs {
		cp.AgentIDs[k] = v
	}
	return &cp
}
