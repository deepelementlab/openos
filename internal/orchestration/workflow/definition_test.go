package workflow

import (
	"testing"
	"time"
)

func TestWorkflowDefinition_Fields(t *testing.T) {
	def := &WorkflowDefinition{
		ID:          "test-wf",
		Name:        "TestWorkflow",
		Description: "A test workflow",
		Version:     "1.0",
		StartStep:   "step1",
		Steps: []*StepDefinition{
			{ID: "step1", Name: "Step 1", StepType: StepTypeEnd},
		},
		Timeout: 5 * time.Minute,
	}

	if def.ID != "test-wf" {
		t.Error("ID not set")
	}
	if len(def.Steps) != 1 {
		t.Error("steps not set")
	}
}

func TestWorkflowRegistry_Register(t *testing.T) {
	r := NewWorkflowRegistry()

	def := &WorkflowDefinition{
		ID:        "wf-1",
		Name:      "Test",
		Version:   "1.0",
		StartStep: "s1",
		Steps: []*StepDefinition{
			{ID: "s1", Name: "S1", StepType: StepTypeEnd},
		},
	}

	if err := r.Register(def); err != nil {
		t.Fatalf("failed to register: %v", err)
	}
}

func TestWorkflowRegistry_RegisterNoID(t *testing.T) {
	r := NewWorkflowRegistry()

	err := r.Register(&WorkflowDefinition{Name: "Test"})
	if err == nil {
		t.Error("should reject workflow without ID")
	}
}

func TestWorkflowRegistry_RegisterNoSteps(t *testing.T) {
	r := NewWorkflowRegistry()

	err := r.Register(&WorkflowDefinition{ID: "wf-no-steps", Name: "Test"})
	if err == nil {
		t.Error("should reject workflow without steps")
	}
}

func TestWorkflowRegistry_RegisterDuplicateStepID(t *testing.T) {
	r := NewWorkflowRegistry()

	err := r.Register(&WorkflowDefinition{
		ID:   "wf-dup",
		Name: "Test",
		Steps: []*StepDefinition{
			{ID: "s1", Name: "S1", StepType: StepTypeEnd},
			{ID: "s1", Name: "S1 Dup", StepType: StepTypeEnd},
		},
	})
	if err == nil {
		t.Error("should reject duplicate step IDs")
	}
}

func TestWorkflowRegistry_RegisterInvalidStartStep(t *testing.T) {
	r := NewWorkflowRegistry()

	err := r.Register(&WorkflowDefinition{
		ID:        "wf-bad-start",
		Name:      "Test",
		StartStep: "nonexistent",
		Steps: []*StepDefinition{
			{ID: "s1", Name: "S1", StepType: StepTypeEnd},
		},
	})
	if err == nil {
		t.Error("should reject invalid start step")
	}
}

func TestWorkflowRegistry_RegisterInvalidNextStep(t *testing.T) {
	r := NewWorkflowRegistry()

	err := r.Register(&WorkflowDefinition{
		ID:        "wf-bad-next",
		Name:      "Test",
		StartStep: "s1",
		Steps: []*StepDefinition{
			{ID: "s1", Name: "S1", StepType: StepTypeTask, NextSteps: []string{"nonexistent"}},
		},
	})
	if err == nil {
		t.Error("should reject invalid next step")
	}
}

func TestWorkflowRegistry_RegisterInvalidConditionTarget(t *testing.T) {
	r := NewWorkflowRegistry()

	err := r.Register(&WorkflowDefinition{
		ID:        "wf-bad-cond",
		Name:      "Test",
		StartStep: "s1",
		Steps: []*StepDefinition{
			{
				ID:         "s1",
				Name:       "S1",
				StepType:   StepTypeDecision,
				Conditions: map[string]string{"ok": "nonexistent"},
			},
		},
	})
	if err == nil {
		t.Error("should reject invalid condition target")
	}
}

func TestWorkflowRegistry_SetsDefaultStartStep(t *testing.T) {
	r := NewWorkflowRegistry()

	def := &WorkflowDefinition{
		ID:   "wf-default-start",
		Name: "Test",
		Steps: []*StepDefinition{
			{ID: "first", Name: "First", StepType: StepTypeEnd},
		},
	}

	if err := r.Register(def); err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	if def.StartStep != "first" {
		t.Errorf("expected StartStep to be first, got %s", def.StartStep)
	}
}

func TestWorkflowRegistry_Get(t *testing.T) {
	r := NewWorkflowRegistry()
	r.Register(&WorkflowDefinition{
		ID:    "wf-get",
		Name:  "Test",
		Steps: []*StepDefinition{{ID: "s1", Name: "S1", StepType: StepTypeEnd}},
	})

	def, err := r.Get("wf-get")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if def.ID != "wf-get" {
		t.Errorf("expected wf-get, got %s", def.ID)
	}

	_, err = r.Get("nonexistent")
	if err == nil {
		t.Error("should error for nonexistent")
	}
}

func TestWorkflowRegistry_GetLatest(t *testing.T) {
	r := NewWorkflowRegistry()
	r.Register(&WorkflowDefinition{ID: "v1", Name: "TestWF", Version: "1.0", Steps: []*StepDefinition{{ID: "s1", Name: "S1", StepType: StepTypeEnd}}})
	r.Register(&WorkflowDefinition{ID: "v2", Name: "TestWF", Version: "2.0", Steps: []*StepDefinition{{ID: "s1", Name: "S1", StepType: StepTypeEnd}}})

	def, err := r.GetLatest("TestWF")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if def.Version != "2.0" {
		t.Errorf("expected latest version 2.0, got %s", def.Version)
	}

	_, err = r.GetLatest("nonexistent")
	if err == nil {
		t.Error("should error for nonexistent name")
	}
}

func TestWorkflowRegistry_List(t *testing.T) {
	r := NewWorkflowRegistry()
	r.Register(&WorkflowDefinition{ID: "a", Name: "A", Steps: []*StepDefinition{{ID: "s", Name: "S", StepType: StepTypeEnd}}})
	r.Register(&WorkflowDefinition{ID: "b", Name: "B", Steps: []*StepDefinition{{ID: "s", Name: "S", StepType: StepTypeEnd}}})

	list := r.List()
	if len(list) != 2 {
		t.Errorf("expected 2, got %d", len(list))
	}
}

func TestStepDefinition_Fields(t *testing.T) {
	s := &StepDefinition{
		ID:           "step1",
		Name:         "Test Step",
		StepType:     StepTypeTask,
		Handler:      "test_handler",
		NextSteps:    []string{"step2"},
		Conditions:   map[string]string{"ok": "step2"},
		Compensation: "comp1",
		Timeout:      30 * time.Second,
		Inputs:       map[string]string{"param": "value"},
		Outputs:      map[string]string{"result": "output"},
	}

	if s.Handler != "test_handler" {
		t.Error("Handler not set")
	}
	if len(s.NextSteps) != 1 {
		t.Error("NextSteps not set")
	}
	if s.Compensation != "comp1" {
		t.Error("Compensation not set")
	}
}

func TestStepTypeConstants(t *testing.T) {
	types := map[StepType]string{
		StepTypeTask:     "task",
		StepTypeDecision: "decision",
		StepTypeParallel: "parallel",
		StepTypeWait:     "wait",
		StepTypeEvent:    "event",
		StepTypeSubFlow:  "subflow",
		StepTypeEnd:      "end",
	}
	for st, expected := range types {
		if string(st) != expected {
			t.Errorf("expected %s, got %s", expected, string(st))
		}
	}
}

func TestBackoffTypeConstants(t *testing.T) {
	if string(BackoffFixed) != "fixed" {
		t.Errorf("expected fixed, got %s", string(BackoffFixed))
	}
	if string(BackoffLinear) != "linear" {
		t.Errorf("expected linear, got %s", string(BackoffLinear))
	}
	if string(BackoffExponential) != "exponential" {
		t.Errorf("expected exponential, got %s", string(BackoffExponential))
	}
}

func TestRetryPolicy(t *testing.T) {
	p := &RetryPolicy{
		MaxAttempts:  3,
		BackoffType:  BackoffExponential,
		InitialDelay: time.Second,
		MaxDelay:     time.Minute,
	}
	if p.MaxAttempts != 3 {
		t.Errorf("expected 3, got %d", p.MaxAttempts)
	}
}

func TestAgentDeployWorkflow(t *testing.T) {
	wf := AgentDeployWorkflow()
	if wf.ID != "agent-deploy" {
		t.Errorf("expected agent-deploy, got %s", wf.ID)
	}
	if len(wf.Steps) != 7 {
		t.Errorf("expected 7 steps, got %d", len(wf.Steps))
	}
	if wf.StartStep != "schedule" {
		t.Errorf("expected start at schedule, got %s", wf.StartStep)
	}
}

func TestAgentStopWorkflow(t *testing.T) {
	wf := AgentStopWorkflow()
	if wf.ID != "agent-stop" {
		t.Errorf("expected agent-stop, got %s", wf.ID)
	}
	if len(wf.Steps) != 4 {
		t.Errorf("expected 4 steps, got %d", len(wf.Steps))
	}
}

func TestAgentDeleteWorkflow(t *testing.T) {
	wf := AgentDeleteWorkflow()
	if wf.ID != "agent-delete" {
		t.Errorf("expected agent-delete, got %s", wf.ID)
	}
	if len(wf.Steps) != 7 {
		t.Errorf("expected 7 steps, got %d", len(wf.Steps))
	}
}
