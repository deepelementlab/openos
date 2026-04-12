package scheduler

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/agentos/aos/internal/config"
)

type PriorityQueue struct {
	mu       sync.Mutex
	items    []*QueueItem
	lessFunc func(a, b *QueueItem) bool
}

type QueueItem struct {
	Task       TaskRequest
	EnqueuedAt time.Time
	Index      int
}

func NewPriorityQueue(less func(a, b *QueueItem) bool) *PriorityQueue {
	pq := &PriorityQueue{
		items:    make([]*QueueItem, 0),
		lessFunc: less,
	}
	heap.Init(pq)
	return pq
}

func DefaultLess(a, b *QueueItem) bool {
	if a.Task.Priority != b.Task.Priority {
		return a.Task.Priority > b.Task.Priority
	}
	return a.EnqueuedAt.Before(b.EnqueuedAt)
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*QueueItem)
	item.Index = len(pq.items)
	pq.items = append(pq.items, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := pq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	pq.items = old[:n-1]
	return item
}

func (pq *PriorityQueue) Len() int {
	return len(pq.items)
}

func (pq *PriorityQueue) Less(i, j int) bool {
	return pq.lessFunc(pq.items[i], pq.items[j])
}

func (pq *PriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].Index = i
	pq.items[j].Index = j
}

func (pq *PriorityQueue) Enqueue(task TaskRequest) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	heap.Push(pq, &QueueItem{
		Task:       task,
		EnqueuedAt: time.Now(),
	})
}

func (pq *PriorityQueue) Dequeue() (*QueueItem, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if pq.Len() == 0 {
		return nil, false
	}
	item := heap.Pop(pq).(*QueueItem)
	return item, true
}

func (pq *PriorityQueue) Size() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return pq.Len()
}

type FailoverScheduler struct {
	mu             sync.RWMutex
	nodes          map[string]*nodeAlloc
	allocations    map[string]string
	primaryNodeID  string
	queue          *PriorityQueue
	cfg            *config.Config
	totalScheduled int64
	totalFailed    int64
	totalFailovers int64
}

func NewFailoverScheduler() *FailoverScheduler {
	return &FailoverScheduler{
		nodes:       make(map[string]*nodeAlloc),
		allocations: make(map[string]string),
		queue:       NewPriorityQueue(DefaultLess),
	}
}

func (f *FailoverScheduler) Initialize(_ context.Context, cfg *config.Config) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cfg = cfg
	return nil
}

func (f *FailoverScheduler) Schedule(ctx context.Context, req TaskRequest) (*ScheduleResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	eligible := f.filterHealthy(req)
	if len(eligible) == 0 {
		f.totalFailed++
		return &ScheduleResult{
			Success:     false,
			TaskID:      req.TaskID,
			Reason:      "no_eligible_node",
			Message:     "no healthy node has enough resources",
			ScheduledAt: time.Now(),
		}, nil
	}

	selected := f.selectPrimary(eligible)
	if selected == nil {
		selected = eligible[0]
	}

	selected.allocatedCPU += req.CPURequest
	selected.allocatedMem += req.MemoryRequest
	selected.agentCount++
	selected.lastUpdate = time.Now()
	f.allocations[req.TaskID] = selected.state.NodeID
	f.totalScheduled++

	return &ScheduleResult{
		Success:     true,
		NodeID:      selected.state.NodeID,
		TaskID:      req.TaskID,
		Message:     fmt.Sprintf("scheduled on %s", selected.state.NodeID),
		Score:       1.0,
		ScheduledAt: time.Now(),
	}, nil
}

func (f *FailoverScheduler) ScheduleBatch(ctx context.Context, reqs []TaskRequest) ([]ScheduleResult, error) {
	results := make([]ScheduleResult, 0, len(reqs))
	for _, req := range reqs {
		res, err := f.Schedule(ctx, req)
		if err != nil {
			return nil, err
		}
		results = append(results, *res)
	}
	return results, nil
}

func (f *FailoverScheduler) Reschedule(ctx context.Context, taskID string, reason RescheduleReason) (*ScheduleResult, error) {
	f.mu.Lock()
	if oldNodeID, ok := f.allocations[taskID]; ok {
		if na, ok := f.nodes[oldNodeID]; ok {
			na.agentCount--
			if na.agentCount < 0 {
				na.agentCount = 0
			}
			na.lastUpdate = time.Now()
		}
		delete(f.allocations, taskID)
	}
	f.mu.Unlock()

	if reason == RescheduleReasonNodeFailure {
		f.mu.Lock()
		f.totalFailovers++
		f.mu.Unlock()
	}

	return f.Schedule(ctx, TaskRequest{TaskID: taskID})
}

func (f *FailoverScheduler) CancelTask(_ context.Context, taskID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	nodeID, ok := f.allocations[taskID]
	if !ok {
		return nil
	}
	if na, ok := f.nodes[nodeID]; ok {
		na.agentCount--
		if na.agentCount < 0 {
			na.agentCount = 0
		}
		na.lastUpdate = time.Now()
	}
	delete(f.allocations, taskID)
	return nil
}

func (f *FailoverScheduler) AddNode(_ context.Context, node NodeState) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if node.NodeID == "" {
		return fmt.Errorf("node ID is required")
	}
	if f.primaryNodeID == "" {
		f.primaryNodeID = node.NodeID
	}
	f.nodes[node.NodeID] = &nodeAlloc{
		state:      node,
		lastUpdate: time.Now(),
	}
	return nil
}

func (f *FailoverScheduler) RemoveNode(_ context.Context, nodeID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.nodes[nodeID]; !ok {
		return fmt.Errorf("node %s not found", nodeID)
	}
	if f.primaryNodeID == nodeID {
		f.primaryNodeID = ""
		for id := range f.nodes {
			if id != nodeID {
				f.primaryNodeID = id
				break
			}
		}
	}
	for taskID, nid := range f.allocations {
		if nid == nodeID {
			delete(f.allocations, taskID)
		}
	}
	delete(f.nodes, nodeID)
	return nil
}

func (f *FailoverScheduler) UpdateNode(_ context.Context, node NodeState) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	na, ok := f.nodes[node.NodeID]
	if !ok {
		return fmt.Errorf("node %s not found", node.NodeID)
	}
	na.state = node
	na.lastUpdate = time.Now()
	return nil
}

func (f *FailoverScheduler) ListNodes(_ context.Context) ([]NodeState, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	nodes := make([]NodeState, 0, len(f.nodes))
	for _, na := range f.nodes {
		nodes = append(nodes, na.toNodeState())
	}
	return nodes, nil
}

func (f *FailoverScheduler) HealthCheck(_ context.Context) error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	healthyCount := 0
	for _, na := range f.nodes {
		if na.state.Health == "healthy" || na.state.Health == "" {
			healthyCount++
		}
	}
	if healthyCount == 0 && len(f.nodes) > 0 {
		return fmt.Errorf("no healthy nodes available")
	}
	return nil
}

func (f *FailoverScheduler) GetStats(_ context.Context) (*SchedulerMetrics, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return &SchedulerMetrics{
		TotalScheduled: f.totalScheduled,
		TotalFailed:    f.totalFailed,
		NodeCount:      len(f.nodes),
		QueueSize:      f.queue.Size(),
		AlgorithmStats: map[string]int64{
			"failovers": f.totalFailovers,
		},
	}, nil
}

func (f *FailoverScheduler) MarkNodeFailed(ctx context.Context, nodeID string) (int, error) {
	f.mu.Lock()
	na, ok := f.nodes[nodeID]
	if !ok {
		f.mu.Unlock()
		return 0, fmt.Errorf("node %s not found", nodeID)
	}
	na.state.Health = "unhealthy"
	na.lastUpdate = time.Now()
	f.mu.Unlock()

	var migrated int
	var tasksToReschedule []string
	f.mu.RLock()
	for taskID, nid := range f.allocations {
		if nid == nodeID {
			tasksToReschedule = append(tasksToReschedule, taskID)
		}
	}
	f.mu.RUnlock()

	for _, taskID := range tasksToReschedule {
		_, err := f.Reschedule(ctx, taskID, RescheduleReasonNodeFailure)
		if err == nil {
			migrated++
		}
	}
	return migrated, nil
}

func (f *FailoverScheduler) filterHealthy(req TaskRequest) []*nodeAlloc {
	var eligible []*nodeAlloc
	for _, na := range f.nodes {
		if na.state.Health != "healthy" && na.state.Health != "" {
			continue
		}
		totalCPU := float64(na.state.CPUCores) * 1000
		freeCPU := totalCPU - na.allocatedCPU - (na.state.CPUUsage / 100 * totalCPU)
		if req.CPURequest > 0 && freeCPU < req.CPURequest {
			continue
		}
		freeMem := float64(na.state.MemoryBytes) - float64(na.allocatedMem) -
			(float64(na.state.MemoryBytes) * na.state.MemoryUsage / 100)
		if req.MemoryRequest > 0 && freeMem < float64(req.MemoryRequest) {
			continue
		}
		eligible = append(eligible, na)
	}
	return eligible
}

func (f *FailoverScheduler) selectPrimary(eligible []*nodeAlloc) *nodeAlloc {
	for _, na := range eligible {
		if na.state.NodeID == f.primaryNodeID {
			return na
		}
	}
	return nil
}

func (f *FailoverScheduler) EnqueueTask(req TaskRequest) {
	f.queue.Enqueue(req)
}

func (f *FailoverScheduler) ProcessQueue(ctx context.Context) ([]ScheduleResult, error) {
	var results []ScheduleResult
	for {
		item, ok := f.queue.Dequeue()
		if !ok {
			break
		}
		res, err := f.Schedule(ctx, item.Task)
		if err != nil {
			return results, err
		}
		results = append(results, *res)
	}
	return results, nil
}

var _ Scheduler = (*FailoverScheduler)(nil)
