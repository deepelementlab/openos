package containerd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/agentos/aos/pkg/runtime/interfaces"
	"github.com/agentos/aos/pkg/runtime/types"
)

const defaultNamespace = "agentos"

// ContainerdRuntime implements interfaces.Runtime using containerd.
type ContainerdRuntime struct {
	config *types.RuntimeConfig
	client *containerd.Client
}

// NewRuntime is the factory function used by pkg/runtime/factory.go.
func NewRuntime(_ context.Context, config *types.RuntimeConfig) (interfaces.Runtime, error) {
	r := &ContainerdRuntime{}
	if config != nil {
		r.config = config
	}
	return r, nil
}

// NewContainerdRuntime creates a bare instance (caller must call Initialize).
func NewContainerdRuntime() *ContainerdRuntime {
	return &ContainerdRuntime{}
}

func (r *ContainerdRuntime) Initialize(ctx context.Context, config *types.RuntimeConfig) error {
	r.config = config

	socketPath := "/run/containerd/containerd.sock"
	if config != nil && config.Options != nil {
		if sp, ok := config.Options["socket_path"].(string); ok && sp != "" {
			socketPath = sp
		}
	}

	client, err := containerd.New(socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to containerd: %w", err)
	}
	r.client = client
	return nil
}

func (r *ContainerdRuntime) GetRuntimeInfo() *types.RuntimeInfo {
	return &types.RuntimeInfo{
		Type:       types.RuntimeContainerd,
		Name:       "containerd",
		Version:    "1.0.0",
		APIVersion: "v1alpha1",
		Features: []string{
			"container-management",
			"resource-limits",
			"network-isolation",
			"image-management",
		},
		Capabilities: []string{
			"create", "start", "stop", "delete", "exec", "logs",
		},
	}
}

func (r *ContainerdRuntime) CreateAgent(ctx context.Context, spec *types.AgentSpec) (*types.Agent, error) {
	if spec == nil {
		return nil, fmt.Errorf("agent spec is required")
	}
	if spec.ID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}
	if spec.Image == "" {
		return nil, fmt.Errorf("agent image is required")
	}

	ctx = namespaces.WithNamespace(ctx, defaultNamespace)

	image, err := r.client.Pull(ctx, spec.Image, containerd.WithPullUnpack)
	if err != nil {
		return nil, fmt.Errorf("failed to pull image %s: %w", spec.Image, err)
	}

	containerSpec := &specs.Spec{
		Version: "1.0.0",
		Root:    &specs.Root{Path: "rootfs"},
		Process: &specs.Process{
			Cwd:      spec.WorkingDir,
			Args:     append(spec.Command, spec.Args...),
			Env:      spec.Env,
			Terminal: false,
		},
		Hostname: spec.Name,
	}

	if spec.Resources != nil && spec.Resources.MemoryLimit > 0 {
		limit := spec.Resources.MemoryLimit
		containerSpec.Linux = &specs.Linux{
			Resources: &specs.LinuxResources{
				Memory: &specs.LinuxMemory{Limit: &limit},
			},
		}
	}

	opts := []containerd.NewContainerOpts{
		containerd.WithImage(image),
		containerd.WithNewSnapshot(spec.ID, image),
		containerd.WithSpec(containerSpec),
	}

	_, err = r.client.NewContainer(ctx, spec.ID, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	now := time.Now()
	return &types.Agent{
		ID:        spec.ID,
		Name:      spec.Name,
		Image:     spec.Image,
		State:     types.AgentStateCreated,
		CreatedAt: now,
		Labels:    spec.Labels,
	}, nil
}

func (r *ContainerdRuntime) StartAgent(ctx context.Context, agentID string) error {
	if agentID == "" {
		return fmt.Errorf("agent ID is required")
	}
	ctx = namespaces.WithNamespace(ctx, defaultNamespace)

	container, err := r.client.LoadContainer(ctx, agentID)
	if err != nil {
		return fmt.Errorf("failed to load container %s: %w", agentID, err)
	}

	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	if err := task.Start(ctx); err != nil {
		return fmt.Errorf("failed to start task: %w", err)
	}
	return nil
}

func (r *ContainerdRuntime) StopAgent(ctx context.Context, agentID string, timeout time.Duration) error {
	if agentID == "" {
		return fmt.Errorf("agent ID is required")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx = namespaces.WithNamespace(ctx, defaultNamespace)

	container, err := r.client.LoadContainer(ctx, agentID)
	if err != nil {
		return fmt.Errorf("failed to load container %s: %w", agentID, err)
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	exitCh, err := task.Wait(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for task: %w", err)
	}

	if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to kill task: %w", err)
	}

	select {
	case <-exitCh:
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for task to exit")
	}
	_, _ = task.Delete(ctx)
	return nil
}

func (r *ContainerdRuntime) DeleteAgent(ctx context.Context, agentID string) error {
	if agentID == "" {
		return fmt.Errorf("agent ID is required")
	}
	ctx = namespaces.WithNamespace(ctx, defaultNamespace)

	container, err := r.client.LoadContainer(ctx, agentID)
	if err != nil {
		return fmt.Errorf("failed to load container %s: %w", agentID, err)
	}

	if task, taskErr := container.Task(ctx, nil); taskErr == nil {
		exitCh, _ := task.Wait(ctx)
		_ = task.Kill(ctx, syscall.SIGKILL)
		select {
		case <-exitCh:
		case <-time.After(10 * time.Second):
		}
		_, _ = task.Delete(ctx)
	}

	if err := container.Delete(ctx, containerd.WithSnapshotCleanup); err != nil {
		return fmt.Errorf("failed to delete container: %w", err)
	}
	return nil
}

func (r *ContainerdRuntime) GetAgent(ctx context.Context, agentID string) (*types.Agent, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}
	ctx = namespaces.WithNamespace(ctx, defaultNamespace)

	container, err := r.client.LoadContainer(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load container %s: %w", agentID, err)
	}

	info, err := container.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}

	state := types.AgentStateCreated
	task, taskErr := container.Task(ctx, nil)
	if taskErr == nil {
		taskStatus, err := task.Status(ctx)
		if err == nil {
			switch taskStatus.Status {
			case containerd.Running:
				state = types.AgentStateRunning
			case containerd.Created:
				state = types.AgentStateCreated
			case containerd.Stopped:
				state = types.AgentStateStopped
			default:
				state = types.AgentStateUnknown
			}
		}
	}

	return &types.Agent{
		ID:        agentID,
		Name:      info.Labels["agentos.name"],
		Image:     info.Image,
		State:     state,
		CreatedAt: info.CreatedAt,
		Labels:    info.Labels,
	}, nil
}

func (r *ContainerdRuntime) ListAgents(ctx context.Context, _ *types.AgentFilter) ([]*types.Agent, error) {
	ctx = namespaces.WithNamespace(ctx, defaultNamespace)

	ctrs, err := r.client.Containers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	agents := make([]*types.Agent, 0, len(ctrs))
	for _, c := range ctrs {
		a, err := r.GetAgent(ctx, c.ID())
		if err != nil {
			continue
		}
		agents = append(agents, a)
	}
	return agents, nil
}

func (r *ContainerdRuntime) ExecuteCommand(ctx context.Context, agentID string, cmd *types.Command) (*types.CommandResult, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}
	if cmd == nil {
		return nil, fmt.Errorf("command is required")
	}
	ctx = namespaces.WithNamespace(ctx, defaultNamespace)

	container, err := r.client.LoadContainer(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load container %s: %w", agentID, err)
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	args := append(cmd.Command, cmd.Args...)
	execID := fmt.Sprintf("exec-%d", time.Now().UnixNano())
	process, err := task.Exec(ctx, execID, &specs.Process{
		Args: args,
		Cwd:  cmd.WorkingDir,
		Env:  cmd.Env,
	}, cio.NewCreator(cio.WithStreams(nil, &stdoutBuf, &stderrBuf)))
	if err != nil {
		return nil, fmt.Errorf("failed to create exec process: %w", err)
	}

	exitCh, err := process.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for process: %w", err)
	}

	execStart := time.Now()
	if err := process.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start exec: %w", err)
	}

	status := <-exitCh
	duration := time.Since(execStart)
	code, _, err := status.Result()
	if err != nil {
		return nil, fmt.Errorf("exec error: %w", err)
	}

	return &types.CommandResult{
		ExitCode: int32(code),
		Stdout:   stdoutBuf.Bytes(),
		Stderr:   stderrBuf.Bytes(),
		Duration: duration,
	}, nil
}

func (r *ContainerdRuntime) GetAgentLogs(_ context.Context, agentID string, _ *types.LogOptions) (io.ReadCloser, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}
	return io.NopCloser(&emptyReader{}), nil
}

func (r *ContainerdRuntime) GetAgentStats(ctx context.Context, agentID string) (*types.ResourceStats, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}
	ctx = namespaces.WithNamespace(ctx, defaultNamespace)

	container, err := r.client.LoadContainer(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load container %s: %w", agentID, err)
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	_, err = task.Metrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	return &types.ResourceStats{
		Timestamp: time.Now(),
	}, nil
}

func (r *ContainerdRuntime) UpdateAgent(_ context.Context, agentID string, spec *types.AgentSpec) error {
	if agentID == "" {
		return fmt.Errorf("agent ID is required")
	}
	if spec == nil {
		return fmt.Errorf("agent spec is required")
	}
	return nil
}

func (r *ContainerdRuntime) ResizeAgentTerminal(_ context.Context, agentID string, _, _ uint) error {
	if agentID == "" {
		return fmt.Errorf("agent ID is required")
	}
	return nil
}

func (r *ContainerdRuntime) AttachAgent(_ context.Context, agentID string, _ *types.AttachOptions) (io.ReadWriteCloser, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}
	return nil, fmt.Errorf("attach not implemented yet")
}

func (r *ContainerdRuntime) HealthCheck(ctx context.Context) error {
	if r.client == nil {
		return fmt.Errorf("containerd client not initialized")
	}
	_, err := r.client.Version(ctx)
	if err != nil {
		return fmt.Errorf("containerd health check failed: %w", err)
	}
	return nil
}

func (r *ContainerdRuntime) Cleanup(_ context.Context) error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

type emptyReader struct{}

func (r *emptyReader) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

var _ interfaces.Runtime = (*ContainerdRuntime)(nil)
