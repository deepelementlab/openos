// Package process implements Agent Kernel process-group and session abstractions (OS mapping).
package process

import "time"

// GroupState is the lifecycle of an agent group.
type GroupState string

const (
	GroupStateActive   GroupState = "active"
	GroupStateStopping GroupState = "stopping"
	GroupStateStopped  GroupState = "stopped"
)

// Signal is a control-plane signal (analogous to POSIX signals, simplified).
type Signal string

const (
	SignalTerminate Signal = "SIGTERM"
	SignalKill      Signal = "SIGKILL"
	SignalReload    Signal = "SIGHUP"
)

// AgentGroup models a process group (setpgid-style) of agents.
type AgentGroup struct {
	GroupID   string
	LeaderID  string
	Members   []string
	SessionID string
	State     GroupState
	CreatedAt time.Time
}

// AgentSession models a session (setsid-style) containing one or more groups.
type AgentSession struct {
	SessionID string
	LeaderID  string
	GroupIDs  []string
	CreatedAt time.Time
}

// ProcessNamespace isolates logical PIDs / routing for a set of agents (PID namespace analogue).
type ProcessNamespace struct {
	NamespaceID string
	AgentIDs    map[string]int // virtual PID map
	CreatedAt   time.Time
}
