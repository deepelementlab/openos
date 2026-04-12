package workflow

import (
	"fmt"
	"time"
)

// WorkflowDefinition defines the structure of a workflow.
type WorkflowDefinition struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Steps       []*StepDefinition      `json:"steps"`
	StartStep   string                 `json:"start_step"`
	RetryPolicy *RetryPolicy           `json:"retry_policy,omitempty"`
	Timeout     time.Duration          `json:"timeout,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// StepDefinition defines a single step in a workflow.
type StepDefinition struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	StepType     StepType               `json:"step_type"`
	Handler      string                 `json:"handler"`       // Reference to step handler
	NextSteps    []string               `json:"next_steps"`    // Next step IDs (for branching)
	Conditions   map[string]string      `json:"conditions"`    // Condition -> next step mapping
	Compensation string                 `json:"compensation"`  // Compensation step ID
	RetryPolicy  *RetryPolicy           `json:"retry_policy,omitempty"`
	Timeout      time.Duration          `json:"timeout,omitempty"`
	Inputs       map[string]string      `json:"inputs"`        // Input parameter mapping
	Outputs      map[string]string      `json:"outputs"`       // Output parameter mapping
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// StepType defines the type of workflow step.
type StepType string

const (
	StepTypeTask     StepType = "task"
	StepTypeDecision StepType = "decision"
	StepTypeParallel StepType = "parallel"
	StepTypeWait     StepType = "wait"
	StepTypeEvent    StepType = "event"
	StepTypeSubFlow  StepType = "subflow"
	StepTypeEnd      StepType = "end"
)

// RetryPolicy defines retry behavior.
type RetryPolicy struct {
	MaxAttempts int           `json:"max_attempts"`
	BackoffType BackoffType   `json:"backoff_type"`
	InitialDelay time.Duration `json:"initial_delay"`
	MaxDelay    time.Duration `json:"max_delay"`
}

// BackoffType defines the backoff strategy.
type BackoffType string

const (
	BackoffFixed    BackoffType = "fixed"
	BackoffLinear   BackoffType = "linear"
	BackoffExponential BackoffType = "exponential"
)

// WorkflowRegistry manages workflow definitions.
type WorkflowRegistry struct {
	definitions map[string]*WorkflowDefinition // id -> definition
	versions    map[string][]*WorkflowDefinition // name -> versions
}

// NewWorkflowRegistry creates a new workflow registry.
func NewWorkflowRegistry() *WorkflowRegistry {
	return &WorkflowRegistry{
		definitions: make(map[string]*WorkflowDefinition),
		versions:    make(map[string][]*WorkflowDefinition),
	}
}

// Register registers a workflow definition.
func (r *WorkflowRegistry) Register(def *WorkflowDefinition) error {
	if def.ID == "" {
		return fmt.Errorf("workflow ID is required")
	}

	// Validate steps
	if err := r.validateSteps(def); err != nil {
		return fmt.Errorf("invalid workflow %s: %w", def.ID, err)
	}

	r.definitions[def.ID] = def
	r.versions[def.Name] = append(r.versions[def.Name], def)

	return nil
}

// validateSteps validates workflow step definitions.
func (r *WorkflowRegistry) validateSteps(def *WorkflowDefinition) error {
	if len(def.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	stepMap := make(map[string]*StepDefinition)
	for _, step := range def.Steps {
		if step.ID == "" {
			return fmt.Errorf("step ID is required")
		}
		if _, exists := stepMap[step.ID]; exists {
			return fmt.Errorf("duplicate step ID: %s", step.ID)
		}
		stepMap[step.ID] = step
	}

	// Validate start step
	if def.StartStep == "" {
		def.StartStep = def.Steps[0].ID
	}
	if _, exists := stepMap[def.StartStep]; !exists {
		return fmt.Errorf("start step %s not found", def.StartStep)
	}

	// Validate next steps and conditions
	for _, step := range def.Steps {
		for _, nextID := range step.NextSteps {
			if _, exists := stepMap[nextID]; !exists {
				return fmt.Errorf("next step %s not found from step %s", nextID, step.ID)
			}
		}
		for _, nextID := range step.Conditions {
			if _, exists := stepMap[nextID]; !exists {
				return fmt.Errorf("condition target step %s not found from step %s", nextID, step.ID)
			}
		}
	}

	return nil
}

// Get retrieves a workflow definition by ID.
func (r *WorkflowRegistry) Get(id string) (*WorkflowDefinition, error) {
	def, exists := r.definitions[id]
	if !exists {
		return nil, fmt.Errorf("workflow %s not found", id)
	}
	return def, nil
}

// GetLatest retrieves the latest version of a workflow by name.
func (r *WorkflowRegistry) GetLatest(name string) (*WorkflowDefinition, error) {
	versions := r.versions[name]
	if len(versions) == 0 {
		return nil, fmt.Errorf("workflow %s not found", name)
	}
	// Return the last registered version
	return versions[len(versions)-1], nil
}

// List returns all registered workflow definitions.
func (r *WorkflowRegistry) List() []*WorkflowDefinition {
	defs := make([]*WorkflowDefinition, 0, len(r.definitions))
	for _, def := range r.definitions {
		defs = append(defs, def)
	}
	return defs
}

// Standard workflow definitions.
func AgentDeployWorkflow() *WorkflowDefinition {
	return &WorkflowDefinition{
		ID:          "agent-deploy",
		Name:        "AgentDeployWorkflow",
		Description: "Deploys an agent to a node",
		Version:     "1.0",
		StartStep:   "schedule",
		Timeout:     5 * time.Minute,
		RetryPolicy: &RetryPolicy{
			MaxAttempts:  3,
			BackoffType:  BackoffExponential,
			InitialDelay: time.Second,
			MaxDelay:     time.Minute,
		},
		Steps: []*StepDefinition{
			{
				ID:          "schedule",
				Name:        "Schedule Agent",
				StepType:    StepTypeTask,
				Handler:     "schedule_agent",
				NextSteps:   []string{"allocate_resources"},
				Compensation: "deallocate_resources",
				Timeout:     30 * time.Second,
			},
			{
				ID:          "allocate_resources",
				Name:        "Allocate Resources",
				StepType:    StepTypeTask,
				Handler:     "allocate_resources",
				NextSteps:   []string{"create_runtime"},
				Compensation: "release_resources",
				Timeout:     30 * time.Second,
			},
			{
				ID:          "create_runtime",
				Name:        "Create Runtime",
				StepType:    StepTypeTask,
				Handler:     "create_runtime",
				NextSteps:   []string{"start_agent"},
				Compensation: "destroy_runtime",
				Timeout:     60 * time.Second,
			},
			{
				ID:          "start_agent",
				Name:        "Start Agent",
				StepType:    StepTypeTask,
				Handler:     "start_agent",
				NextSteps:   []string{"wait_ready"},
				Timeout:     60 * time.Second,
			},
			{
				ID:          "wait_ready",
				Name:        "Wait for Ready",
				StepType:    StepTypeWait,
				Handler:     "wait_ready",
				Conditions:  map[string]string{
					"ready": "complete",
					"timeout": "failed",
					"failed": "failed",
				},
				Timeout:     2 * time.Minute,
			},
			{
				ID:          "complete",
				Name:        "Complete",
				StepType:    StepTypeEnd,
			},
			{
				ID:          "failed",
				Name:        "Failed",
				StepType:    StepTypeEnd,
			},
		},
	}
}

func AgentStopWorkflow() *WorkflowDefinition {
	return &WorkflowDefinition{
		ID:          "agent-stop",
		Name:        "AgentStopWorkflow",
		Description: "Stops a running agent",
		Version:     "1.0",
		StartStep:   "stop_agent",
		Timeout:     1 * time.Minute,
		Steps: []*StepDefinition{
			{
				ID:          "stop_agent",
				Name:        "Stop Agent",
				StepType:    StepTypeTask,
				Handler:     "stop_agent",
				NextSteps:   []string{"wait_stopped"},
				Timeout:     30 * time.Second,
			},
			{
				ID:          "wait_stopped",
				Name:        "Wait for Stopped",
				StepType:    StepTypeWait,
				Handler:     "wait_stopped",
				Conditions:  map[string]string{
					"stopped": "complete",
					"timeout": "failed",
				},
				Timeout:     30 * time.Second,
			},
			{
				ID:          "complete",
				Name:        "Complete",
				StepType:    StepTypeEnd,
			},
			{
				ID:          "failed",
				Name:        "Failed",
				StepType:    StepTypeEnd,
			},
		},
	}
}

func AgentDeleteWorkflow() *WorkflowDefinition {
	return &WorkflowDefinition{
		ID:          "agent-delete",
		Name:        "AgentDeleteWorkflow",
		Description: "Deletes an agent and cleans up resources",
		Version:     "1.0",
		StartStep:   "stop_if_running",
		Timeout:     2 * time.Minute,
		Steps: []*StepDefinition{
			{
				ID:          "stop_if_running",
				Name:        "Stop if Running",
				StepType:    StepTypeDecision,
				Handler:     "check_running",
				Conditions:  map[string]string{
					"running": "stop_agent",
					"stopped": "destroy_runtime",
				},
			},
			{
				ID:          "stop_agent",
				Name:        "Stop Agent",
				StepType:    StepTypeTask,
				Handler:     "stop_agent",
				NextSteps:   []string{"destroy_runtime"},
				Timeout:     30 * time.Second,
			},
			{
				ID:          "destroy_runtime",
				Name:        "Destroy Runtime",
				StepType:    StepTypeTask,
				Handler:     "destroy_runtime",
				NextSteps:   []string{"release_resources"},
				Timeout:     30 * time.Second,
			},
			{
				ID:          "release_resources",
				Name:        "Release Resources",
				StepType:    StepTypeTask,
				Handler:     "release_resources",
				NextSteps:   []string{"mark_deleted"},
				Timeout:     10 * time.Second,
			},
			{
				ID:          "mark_deleted",
				Name:        "Mark Deleted",
				StepType:    StepTypeTask,
				Handler:     "mark_deleted",
				NextSteps:   []string{"complete"},
			},
			{
				ID:          "complete",
				Name:        "Complete",
				StepType:    StepTypeEnd,
			},
			{
				ID:          "failed",
				Name:        "Failed",
				StepType:    StepTypeEnd,
			},
		},
	}
}
