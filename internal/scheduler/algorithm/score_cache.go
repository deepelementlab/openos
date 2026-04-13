package algorithm

import (
	"context"
	"fmt"
	"sync"

	"github.com/agentos/aos/internal/scheduler"
)

// AgentFingerprint is a stable key for hot-path agent classes (same shape → reuse scores).
type AgentFingerprint struct {
	CPU    int
	Mem    int64
	Disk   int64
	TypeID string // optional TaskType / agent class
}

func fingerprintFromSpec(a scheduler.AgentSpec, typeID string) AgentFingerprint {
	return AgentFingerprint{
		CPU:    a.CPURequest,
		Mem:    a.MemoryRequest,
		Disk:   a.DiskRequest,
		TypeID: typeID,
	}
}

func (f AgentFingerprint) String() string {
	return fmt.Sprintf("%s:%d:%d:%d", f.TypeID, f.CPU, f.Mem, f.Disk)
}

// ScoreCache caches per-node scores for a given agent fingerprint to avoid recomputation
// when scheduling many agents of the same class in a tight loop.
type ScoreCache struct {
	mu    sync.RWMutex
	store map[string]map[string]int // fingerprint -> nodeID -> score
}

// NewScoreCache creates an empty score cache.
func NewScoreCache() *ScoreCache {
	return &ScoreCache{
		store: make(map[string]map[string]int),
	}
}

// Get returns cached score for (fingerprint, nodeID).
func (s *ScoreCache) Get(fp AgentFingerprint, nodeID string) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := fp.String()
	m, ok := s.store[key]
	if !ok {
		return 0, false
	}
	score, ok := m[nodeID]
	return score, ok
}

// Put stores a score for (fingerprint, nodeID).
func (s *ScoreCache) Put(fp AgentFingerprint, nodeID string, score int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := fp.String()
	if s.store[key] == nil {
		s.store[key] = make(map[string]int)
	}
	s.store[key][nodeID] = score
}

// InvalidateFingerprint drops all scores for an agent class (node capacity changed materially).
func (s *ScoreCache) InvalidateFingerprint(fp AgentFingerprint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.store, fp.String())
}

// InvalidateNode drops all scores involving a node (node updated / removed).
func (s *ScoreCache) InvalidateNode(nodeID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, m := range s.store {
		delete(m, nodeID)
		if len(m) == 0 {
			delete(s.store, k)
		}
	}
}

// Clear removes all entries.
func (s *ScoreCache) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store = make(map[string]map[string]int)
}

func scoreNodesDirect(alg scheduler.SchedulingAlgorithm, ctx context.Context, nodes []scheduler.NodeState, agent scheduler.AgentSpec) (map[string]int, error) {
	scores := make(map[string]int, len(nodes))
	for _, n := range nodes {
		score, err := alg.ScoreNode(ctx, n, agent)
		if err != nil {
			return nil, err
		}
		scores[n.NodeID] = score
	}
	return scores, nil
}

// ScoreNodesWithCache scores nodes using alg.ScoreNode, consulting the cache first.
func ScoreNodesWithCache(
	alg scheduler.SchedulingAlgorithm,
	ctx context.Context,
	nodes []scheduler.NodeState,
	agent scheduler.AgentSpec,
	taskType string,
	cache *ScoreCache,
) (map[string]int, error) {
	if cache == nil {
		return scoreNodesDirect(alg, ctx, nodes, agent)
	}
	fp := fingerprintFromSpec(agent, taskType)
	scores := make(map[string]int, len(nodes))
	for _, n := range nodes {
		if sc, ok := cache.Get(fp, n.NodeID); ok {
			scores[n.NodeID] = sc
			continue
		}
		score, err := alg.ScoreNode(ctx, n, agent)
		if err != nil {
			return nil, err
		}
		cache.Put(fp, n.NodeID, score)
		scores[n.NodeID] = score
	}
	return scores, nil
}
