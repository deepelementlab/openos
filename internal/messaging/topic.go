package messaging

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// TopicNamingConvention defines the standard topic naming convention.
// Format: aos.{domain}.{entity}.{action}.{version}
// Example: aos.agent.lifecycle.created.v1
type TopicNamingConvention struct {
	Prefix  string
	Domain  string
	Entity  string
	Action  string
	Version string
}

// TopicBuilder provides a fluent API for building topic names.
type TopicBuilder struct {
	prefix  string
	domain  string
	entity  string
	action  string
	version string
}

// NewTopicBuilder creates a new topic builder with default prefix.
func NewTopicBuilder() *TopicBuilder {
	return &TopicBuilder{
		prefix:  "aos",
		version: "v1",
	}
}

// Prefix sets a custom prefix (default: "aos").
func (b *TopicBuilder) Prefix(prefix string) *TopicBuilder {
	b.prefix = prefix
	return b
}

// Domain sets the domain (e.g., "agent", "scheduler", "workflow", "resource").
func (b *TopicBuilder) Domain(domain string) *TopicBuilder {
	b.domain = domain
	return b
}

// Entity sets the entity type (e.g., "lifecycle", "task", "job", "node").
func (b *TopicBuilder) Entity(entity string) *TopicBuilder {
	b.entity = entity
	return b
}

// Action sets the action (e.g., "created", "updated", "deleted", "started").
func (b *TopicBuilder) Action(action string) *TopicBuilder {
	b.action = action
	return b
}

// Version sets the version (default: "v1").
func (b *TopicBuilder) Version(version string) *TopicBuilder {
	b.version = version
	return b
}

// Build constructs the topic string.
func (b *TopicBuilder) Build() (string, error) {
	// Validate required fields
	if b.domain == "" {
		return "", fmt.Errorf("domain is required")
	}
	if b.entity == "" {
		return "", fmt.Errorf("entity is required")
	}
	if b.action == "" {
		return "", fmt.Errorf("action is required")
	}

	return fmt.Sprintf("%s.%s.%s.%s.%s",
		b.prefix,
		b.domain,
		b.entity,
		b.action,
		b.version,
	), nil
}

// MustBuild constructs the topic string or panics.
func (b *TopicBuilder) MustBuild() string {
	topic, err := b.Build()
	if err != nil {
		panic(err)
	}
	return topic
}

// TopicParser parses topic names according to the convention.
type TopicParser struct {
	pattern *regexp.Regexp
}

// NewTopicParser creates a new topic parser.
func NewTopicParser() *TopicParser {
	// Pattern: prefix.domain.entity.action.version
	// Example: aos.agent.lifecycle.created.v1
	pattern := regexp.MustCompile(`^([a-z]+)\.([a-z_]+)\.([a-z_]+)\.([a-z_]+)\.(v\d+)$`)
	return &TopicParser{pattern: pattern}
}

// Parse parses a topic string into components.
func (p *TopicParser) Parse(topic string) (*TopicNamingConvention, error) {
	matches := p.pattern.FindStringSubmatch(topic)
	if matches == nil {
		return nil, fmt.Errorf("invalid topic format: %s", topic)
	}

	return &TopicNamingConvention{
		Prefix:  matches[1],
		Domain:  matches[2],
		Entity:  matches[3],
		Action:  matches[4],
		Version: matches[5],
	}, nil
}

// IsValid checks if a topic follows the naming convention.
func (p *TopicParser) IsValid(topic string) bool {
	return p.pattern.MatchString(topic)
}

// TopicValidator validates topic names.
type TopicValidator struct {
	parser    *TopicParser
	maxLength int
}

// NewTopicValidator creates a new topic validator.
func NewTopicValidator() *TopicValidator {
	return &TopicValidator{
		parser:    NewTopicParser(),
		maxLength: 256,
	}
}

// Validate validates a topic name.
func (v *TopicValidator) Validate(topic string) error {
	// Check length
	if len(topic) > v.maxLength {
		return fmt.Errorf("topic too long: %d > %d", len(topic), v.maxLength)
	}

	// Check format
	if !v.parser.IsValid(topic) {
		return fmt.Errorf("invalid topic format: %s", topic)
	}

	return nil
}

// TopicMatcher provides pattern matching for topics.
type TopicMatcher struct {
	parser *TopicParser
}

// NewTopicMatcher creates a new topic matcher.
func NewTopicMatcher() *TopicMatcher {
	return &TopicMatcher{
		parser: NewTopicParser(),
	}
}

// Match checks if a topic matches a pattern.
// Pattern supports wildcards: * matches any single segment, # matches zero or more segments.
func (m *TopicMatcher) Match(topic string, pattern string) bool {
	// Simple implementation: convert pattern to regex
	// * -> ([^.]+)
	// # -> (.*)

	// Escape special regex characters except * and #
	regexPattern := regexp.QuoteMeta(pattern)
	regexPattern = strings.ReplaceAll(regexPattern, `\*`, `([^.]+)`)
	regexPattern = strings.ReplaceAll(regexPattern, `\#`, `(.*)`)
	regexPattern = "^" + regexPattern + "$"

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return false
	}

	return re.MatchString(topic)
}

// MatchesDomain checks if a topic belongs to a specific domain.
func (m *TopicMatcher) MatchesDomain(topic string, domain string) bool {
	parsed, err := m.parser.Parse(topic)
	if err != nil {
		return false
	}
	return parsed.Domain == domain
}

// MatchesEntity checks if a topic is for a specific entity type.
func (m *TopicMatcher) MatchesEntity(topic string, entity string) bool {
	parsed, err := m.parser.Parse(topic)
	if err != nil {
		return false
	}
	return parsed.Entity == entity
}

// TopicRegistry manages topic registrations.
type TopicRegistry struct {
	topics map[string]*TopicMetadata
	parser *TopicParser
	mu     sync.RWMutex
}

// TopicMetadata contains metadata about a topic.
type TopicMetadata struct {
	Name        string
	Description string
	Domain      string
	Entity      string
	Action      string
	Version     string
	SchemaRef   string // Reference to schema definition
}

// NewTopicRegistry creates a new topic registry.
func NewTopicRegistry() *TopicRegistry {
	return &TopicRegistry{
		topics: make(map[string]*TopicMetadata),
		parser: NewTopicParser(),
	}
}

// Register registers a topic with its metadata.
func (r *TopicRegistry) Register(metadata *TopicMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate topic format
	if !r.parser.IsValid(metadata.Name) {
		return fmt.Errorf("invalid topic name: %s", metadata.Name)
	}

	// Check for duplicates
	if _, exists := r.topics[metadata.Name]; exists {
		return fmt.Errorf("topic already registered: %s", metadata.Name)
	}

	r.topics[metadata.Name] = metadata
	return nil
}

// Get retrieves metadata for a topic.
func (r *TopicRegistry) Get(topic string) (*TopicMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.topics[topic]
	if !exists {
		return nil, fmt.Errorf("topic not found: %s", topic)
	}

	return metadata, nil
}

// List returns all registered topics.
func (r *TopicRegistry) List() []*TopicMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*TopicMetadata, 0, len(r.topics))
	for _, metadata := range r.topics {
		list = append(list, metadata)
	}

	return list
}

// ListByDomain returns topics for a specific domain.
func (r *TopicRegistry) ListByDomain(domain string) []*TopicMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*TopicMetadata
	for _, metadata := range r.topics {
		if metadata.Domain == domain {
			result = append(result, metadata)
		}
	}

	return result
}

// ListByEntity returns topics for a specific entity type.
func (r *TopicRegistry) ListByEntity(entity string) []*TopicMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*TopicMetadata
	for _, metadata := range r.topics {
		if metadata.Entity == entity {
			result = append(result, metadata)
		}
	}

	return result
}

// Standard domains.
const (
	DomainAgent      = "agent"
	DomainScheduler  = "scheduler"
	DomainWorkflow   = "workflow"
	DomainSaga       = "saga"
	DomainResource   = "resource"
	DomainTask       = "task"
	DomainSecurity   = "security"
	DomainSystem     = "system"
)

// Standard entities.
const (
	EntityLifecycle = "lifecycle"
	EntityTask      = "task"
	EntityJob       = "job"
	EntityNode      = "node"
	EntityStep      = "step"
	EntityResource  = "resource"
	EntityAlert     = "alert"
	EntityPolicy    = "policy"
)

// Standard actions.
const (
	ActionCreated   = "created"
	ActionUpdated   = "updated"
	ActionDeleted   = "deleted"
	ActionStarted   = "started"
	ActionStopped   = "stopped"
	ActionFailed    = "failed"
	ActionCompleted = "completed"
	ActionScheduled = "scheduled"
	ActionAssigned  = "assigned"
	ActionAllocated = "allocated"
	ActionReleased  = "released"
)

// Standard topic helpers.
func AgentLifecycleTopic(action string) string {
	return NewTopicBuilder().
		Domain(DomainAgent).
		Entity(EntityLifecycle).
		Action(action).
		MustBuild()
}

func SchedulerTaskTopic(action string) string {
	return NewTopicBuilder().
		Domain(DomainScheduler).
		Entity(EntityTask).
		Action(action).
		MustBuild()
}

func WorkflowStepTopic(action string) string {
	return NewTopicBuilder().
		Domain(DomainWorkflow).
		Entity(EntityStep).
		Action(action).
		MustBuild()
}

func ResourceAllocationTopic(action string) string {
	return NewTopicBuilder().
		Domain(DomainResource).
		Entity(EntityResource).
		Action(action).
		MustBuild()
}
