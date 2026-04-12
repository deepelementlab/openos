package messaging

import (
	"strings"
	"testing"
)

func TestTopicBuilder_Build(t *testing.T) {
	topic, err := NewTopicBuilder().
		Domain("agent").
		Entity("lifecycle").
		Action("created").
		Build()
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if topic != "aos.agent.lifecycle.created.v1" {
		t.Errorf("expected aos.agent.lifecycle.created.v1, got %s", topic)
	}
}

func TestTopicBuilder_Build_CustomPrefix(t *testing.T) {
	topic, err := NewTopicBuilder().
		Prefix("custom").
		Domain("agent").
		Entity("lifecycle").
		Action("started").
		Version("v2").
		Build()
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if topic != "custom.agent.lifecycle.started.v2" {
		t.Errorf("expected custom.agent.lifecycle.started.v2, got %s", topic)
	}
}

func TestTopicBuilder_Build_MissingDomain(t *testing.T) {
	_, err := NewTopicBuilder().Entity("e").Action("a").Build()
	if err == nil {
		t.Error("expected error for missing domain")
	}
}

func TestTopicBuilder_Build_MissingEntity(t *testing.T) {
	_, err := NewTopicBuilder().Domain("d").Action("a").Build()
	if err == nil {
		t.Error("expected error for missing entity")
	}
}

func TestTopicBuilder_Build_MissingAction(t *testing.T) {
	_, err := NewTopicBuilder().Domain("d").Entity("e").Build()
	if err == nil {
		t.Error("expected error for missing action")
	}
}

func TestTopicBuilder_MustBuild(t *testing.T) {
	topic := NewTopicBuilder().Domain("d").Entity("e").Action("a").MustBuild()
	if topic != "aos.d.e.a.v1" {
		t.Errorf("expected aos.d.e.a.v1, got %s", topic)
	}
}

func TestTopicBuilder_MustBuild_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic from MustBuild")
		}
	}()
	NewTopicBuilder().MustBuild()
}

func TestTopicParser_Parse(t *testing.T) {
	parser := NewTopicParser()
	result, err := parser.Parse("aos.agent.lifecycle.created.v1")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if result.Prefix != "aos" {
		t.Errorf("expected aos, got %s", result.Prefix)
	}
	if result.Domain != "agent" {
		t.Errorf("expected agent, got %s", result.Domain)
	}
	if result.Entity != "lifecycle" {
		t.Errorf("expected lifecycle, got %s", result.Entity)
	}
	if result.Action != "created" {
		t.Errorf("expected created, got %s", result.Action)
	}
	if result.Version != "v1" {
		t.Errorf("expected v1, got %s", result.Version)
	}
}

func TestTopicParser_Parse_InvalidFormat(t *testing.T) {
	parser := NewTopicParser()
	_, err := parser.Parse("invalid")
	if err == nil {
		t.Error("expected error for invalid topic")
	}
}

func TestTopicParser_Parse_InvalidPrefix(t *testing.T) {
	parser := NewTopicParser()
	_, err := parser.Parse("123.agent.lifecycle.created.v1")
	if err == nil {
		t.Error("expected error for numeric prefix")
	}
}

func TestTopicParser_IsValid(t *testing.T) {
	parser := NewTopicParser()
	if !parser.IsValid("aos.agent.lifecycle.created.v1") {
		t.Error("expected valid topic")
	}
	if parser.IsValid("invalid") {
		t.Error("expected invalid topic")
	}
	if parser.IsValid("aos..lifecycle.created.v1") {
		t.Error("expected invalid for empty domain")
	}
}

func TestTopicValidator_Validate(t *testing.T) {
	validator := NewTopicValidator()
	if err := validator.Validate("aos.agent.lifecycle.created.v1"); err != nil {
		t.Errorf("expected valid: %v", err)
	}
}

func TestTopicValidator_Validate_TooLong(t *testing.T) {
	validator := NewTopicValidator()
	longTopic := "aos." + strings.Repeat("a", 300) + ".e.a.v1"
	if err := validator.Validate(longTopic); err == nil {
		t.Error("expected error for too long topic")
	}
}

func TestTopicValidator_Validate_InvalidFormat(t *testing.T) {
	validator := NewTopicValidator()
	if err := validator.Validate("bad-format"); err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestTopicMatcher_Match_Exact(t *testing.T) {
	matcher := NewTopicMatcher()
	if !matcher.Match("aos.agent.lifecycle.created.v1", "aos.agent.lifecycle.created.v1") {
		t.Error("expected exact match")
	}
	if matcher.Match("aos.agent.lifecycle.created.v1", "aos.agent.lifecycle.deleted.v1") {
		t.Error("expected no match for different action")
	}
}

func TestTopicMatcher_Match_SingleWildcard(t *testing.T) {
	matcher := NewTopicMatcher()
	if !matcher.Match("aos.agent.lifecycle.created.v1", "aos.*.lifecycle.created.v1") {
		t.Error("expected wildcard match on domain")
	}
	if !matcher.Match("aos.agent.lifecycle.created.v1", "aos.agent.*.created.v1") {
		t.Error("expected wildcard match on entity")
	}
}

func TestTopicMatcher_Match_MultipleWildcards(t *testing.T) {
	matcher := NewTopicMatcher()
	if !matcher.Match("aos.agent.lifecycle.created.v1", "aos.*.*.created.v1") {
		t.Error("expected match with multiple single wildcards")
	}
	if !matcher.Match("aos.agent.lifecycle.created.v1", "*.*.*.*.*") {
		t.Error("expected match with all wildcards")
	}
}

func TestTopicMatcher_MatchesDomain(t *testing.T) {
	matcher := NewTopicMatcher()
	if !matcher.MatchesDomain("aos.agent.lifecycle.created.v1", "agent") {
		t.Error("expected domain match")
	}
	if matcher.MatchesDomain("aos.agent.lifecycle.created.v1", "workflow") {
		t.Error("expected no domain match")
	}
}

func TestTopicMatcher_MatchesDomain_Invalid(t *testing.T) {
	matcher := NewTopicMatcher()
	if matcher.MatchesDomain("invalid", "agent") {
		t.Error("expected false for invalid topic")
	}
}

func TestTopicMatcher_MatchesEntity(t *testing.T) {
	matcher := NewTopicMatcher()
	if !matcher.MatchesEntity("aos.agent.lifecycle.created.v1", "lifecycle") {
		t.Error("expected entity match")
	}
	if matcher.MatchesEntity("aos.agent.lifecycle.created.v1", "task") {
		t.Error("expected no entity match")
	}
}

func TestTopicMatcher_MatchesEntity_Invalid(t *testing.T) {
	matcher := NewTopicMatcher()
	if matcher.MatchesEntity("invalid", "lifecycle") {
		t.Error("expected false for invalid topic")
	}
}

func TestTopicRegistry_Register(t *testing.T) {
	registry := NewTopicRegistry()
	meta := &TopicMetadata{
		Name:        "aos.agent.lifecycle.created.v1",
		Description: "Agent created event",
		Domain:      "agent",
		Entity:      "lifecycle",
		Action:      "created",
		Version:     "v1",
	}
	if err := registry.Register(meta); err != nil {
		t.Fatalf("register failed: %v", err)
	}
}

func TestTopicRegistry_Register_Duplicate(t *testing.T) {
	registry := NewTopicRegistry()
	meta := &TopicMetadata{Name: "aos.agent.lifecycle.created.v1", Domain: "agent"}
	registry.Register(meta)
	if err := registry.Register(meta); err == nil {
		t.Error("expected error for duplicate topic")
	}
}

func TestTopicRegistry_Register_InvalidName(t *testing.T) {
	registry := NewTopicRegistry()
	meta := &TopicMetadata{Name: "invalid-name", Domain: "agent"}
	if err := registry.Register(meta); err == nil {
		t.Error("expected error for invalid topic name")
	}
}

func TestTopicRegistry_Get(t *testing.T) {
	registry := NewTopicRegistry()
	meta := &TopicMetadata{Name: "aos.agent.lifecycle.created.v1", Domain: "agent", Entity: "lifecycle"}
	registry.Register(meta)

	result, err := registry.Get("aos.agent.lifecycle.created.v1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if result.Domain != "agent" {
		t.Errorf("expected domain agent, got %s", result.Domain)
	}
}

func TestTopicRegistry_Get_NotFound(t *testing.T) {
	registry := NewTopicRegistry()
	_, err := registry.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent topic")
	}
}

func TestTopicRegistry_List(t *testing.T) {
	registry := NewTopicRegistry()
	registry.Register(&TopicMetadata{Name: "aos.agent.lifecycle.created.v1", Domain: "agent"})
	registry.Register(&TopicMetadata{Name: "aos.workflow.step.started.v1", Domain: "workflow"})

	list := registry.List()
	if len(list) != 2 {
		t.Errorf("expected 2 topics, got %d", len(list))
	}
}

func TestTopicRegistry_ListByDomain(t *testing.T) {
	registry := NewTopicRegistry()
	registry.Register(&TopicMetadata{Name: "aos.agent.lifecycle.created.v1", Domain: "agent"})
	registry.Register(&TopicMetadata{Name: "aos.agent.lifecycle.deleted.v1", Domain: "agent"})
	registry.Register(&TopicMetadata{Name: "aos.workflow.step.started.v1", Domain: "workflow"})

	agentTopics := registry.ListByDomain("agent")
	if len(agentTopics) != 2 {
		t.Errorf("expected 2 agent topics, got %d", len(agentTopics))
	}
	workflowTopics := registry.ListByDomain("workflow")
	if len(workflowTopics) != 1 {
		t.Errorf("expected 1 workflow topic, got %d", len(workflowTopics))
	}
	emptyTopics := registry.ListByDomain("nonexistent")
	if len(emptyTopics) != 0 {
		t.Errorf("expected 0 topics, got %d", len(emptyTopics))
	}
}

func TestTopicRegistry_ListByEntity(t *testing.T) {
	registry := NewTopicRegistry()
	registry.Register(&TopicMetadata{Name: "aos.agent.lifecycle.created.v1", Domain: "agent", Entity: "lifecycle"})
	registry.Register(&TopicMetadata{Name: "aos.agent.lifecycle.deleted.v1", Domain: "agent", Entity: "lifecycle"})
	registry.Register(&TopicMetadata{Name: "aos.scheduler.task.scheduled.v1", Domain: "scheduler", Entity: "task"})

	lifecycleTopics := registry.ListByEntity("lifecycle")
	if len(lifecycleTopics) != 2 {
		t.Errorf("expected 2 lifecycle topics, got %d", len(lifecycleTopics))
	}
}

func TestTopicConstants(t *testing.T) {
	if DomainAgent != "agent" {
		t.Errorf("expected agent, got %s", DomainAgent)
	}
	if DomainWorkflow != "workflow" {
		t.Errorf("expected workflow, got %s", DomainWorkflow)
	}
	if EntityLifecycle != "lifecycle" {
		t.Errorf("expected lifecycle, got %s", EntityLifecycle)
	}
	if ActionCreated != "created" {
		t.Errorf("expected created, got %s", ActionCreated)
	}
}

func TestAgentLifecycleTopic(t *testing.T) {
	topic := AgentLifecycleTopic(ActionCreated)
	if topic != "aos.agent.lifecycle.created.v1" {
		t.Errorf("expected aos.agent.lifecycle.created.v1, got %s", topic)
	}
}

func TestSchedulerTaskTopic(t *testing.T) {
	topic := SchedulerTaskTopic(ActionScheduled)
	if topic != "aos.scheduler.task.scheduled.v1" {
		t.Errorf("expected aos.scheduler.task.scheduled.v1, got %s", topic)
	}
}

func TestWorkflowStepTopic(t *testing.T) {
	topic := WorkflowStepTopic(ActionStarted)
	if topic != "aos.workflow.step.started.v1" {
		t.Errorf("expected aos.workflow.step.started.v1, got %s", topic)
	}
}

func TestResourceAllocationTopic(t *testing.T) {
	topic := ResourceAllocationTopic(ActionAllocated)
	if topic != "aos.resource.resource.allocated.v1" {
		t.Errorf("expected aos.resource.resource.allocated.v1, got %s", topic)
	}
}
