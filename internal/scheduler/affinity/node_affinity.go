package affinity

import (
	"fmt"
	"strings"
)

// NodeAffinity defines node affinity rules.
type NodeAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []NodeSelectorTerm  `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []PreferredSchedulingTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// NodeSelectorTerm defines a node selector term.
type NodeSelectorTerm struct {
	MatchExpressions []NodeSelectorRequirement `json:"matchExpressions,omitempty"`
	MatchFields      []NodeSelectorRequirement `json:"matchFields,omitempty"`
}

// NodeSelectorRequirement defines a node selector requirement.
type NodeSelectorRequirement struct {
	Key      string   `json:"key"`
	Operator Operator `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

// Operator defines selector operators.
type Operator string

const (
	OpIn           Operator = "In"
	OpNotIn        Operator = "NotIn"
	OpExists       Operator = "Exists"
	OpDoesNotExist Operator = "DoesNotExist"
	OpGt           Operator = "Gt"
	OpLt           Operator = "Lt"
)

// PreferredSchedulingTerm defines a preferred scheduling term with weight.
type PreferredSchedulingTerm struct {
	Weight     int32            `json:"weight"`
	Preference NodeSelectorTerm `json:"preference"`
}

// Match checks if labels match the selector term.
func (t *NodeSelectorTerm) Match(labels map[string]string) bool {
	// Check match expressions
	for _, expr := range t.MatchExpressions {
		if !expr.Match(labels) {
			return false
		}
	}

	// Check match fields
	for _, field := range t.MatchFields {
		if !field.Match(labels) {
			return false
		}
	}

	return true
}

// Match checks if the requirement matches the labels.
func (r *NodeSelectorRequirement) Match(labels map[string]string) bool {
	value, exists := labels[r.Key]

	switch r.Operator {
	case OpIn:
		if !exists {
			return false
		}
		for _, v := range r.Values {
			if value == v {
				return true
			}
		}
		return false

	case OpNotIn:
		if !exists {
			return true
		}
		for _, v := range r.Values {
			if value == v {
				return false
			}
		}
		return true

	case OpExists:
		return exists

	case OpDoesNotExist:
		return !exists

	case OpGt:
		if !exists {
			return false
		}
		return compareValues(value, r.Values[0]) > 0

	case OpLt:
		if !exists {
			return false
		}
		return compareValues(value, r.Values[0]) < 0

	default:
		return false
	}
}

// compareValues compares two values (for Gt/Lt operators).
func compareValues(a, b string) int {
	// Try numeric comparison
	var numA, numB float64
	if _, err := fmt.Sscanf(a, "%f", &numA); err == nil {
		if _, err := fmt.Sscanf(b, "%f", &numB); err == nil {
			if numA < numB {
				return -1
			}
			if numA > numB {
				return 1
			}
			return 0
		}
	}

	// String comparison
	return strings.Compare(a, b)
}

// Evaluator evaluates node affinity.
type Evaluator struct{}

// NewEvaluator creates a new affinity evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// MatchesRequired checks if a node matches required affinity rules.
func (e *Evaluator) MatchesRequired(nodeLabels map[string]string, affinity *NodeAffinity) (bool, error) {
	if affinity == nil || len(affinity.RequiredDuringSchedulingIgnoredDuringExecution) == 0 {
		return true, nil // No requirements
	}

	// All required terms must match
	for _, term := range affinity.RequiredDuringSchedulingIgnoredDuringExecution {
		if !term.Match(nodeLabels) {
			return false, nil
		}
	}

	return true, nil
}

// ScorePreferred calculates a score for preferred affinity rules.
func (e *Evaluator) ScorePreferred(nodeLabels map[string]string, affinity *NodeAffinity) int {
	if affinity == nil || len(affinity.PreferredDuringSchedulingIgnoredDuringExecution) == 0 {
		return 0
	}

	score := 0
	for _, term := range affinity.PreferredDuringSchedulingIgnoredDuringExecution {
		if term.Preference.Match(nodeLabels) {
			score += int(term.Weight)
		}
	}

	return score
}

// NodeAffinityBuilder builds node affinity rules.
type NodeAffinityBuilder struct {
	affinity *NodeAffinity
}

// NewNodeAffinityBuilder creates a new node affinity builder.
func NewNodeAffinityBuilder() *NodeAffinityBuilder {
	return &NodeAffinityBuilder{
		affinity: &NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution:  make([]NodeSelectorTerm, 0),
			PreferredDuringSchedulingIgnoredDuringExecution: make([]PreferredSchedulingTerm, 0),
		},
	}
}

// RequireLabel adds a required label selector.
func (b *NodeAffinityBuilder) RequireLabel(key string, values ...string) *NodeAffinityBuilder {
	term := NodeSelectorTerm{
		MatchExpressions: []NodeSelectorRequirement{
			{
				Key:      key,
				Operator: OpIn,
				Values:   values,
			},
		},
	}
	b.affinity.RequiredDuringSchedulingIgnoredDuringExecution = append(
		b.affinity.RequiredDuringSchedulingIgnoredDuringExecution,
		term,
	)
	return b
}

// PreferLabel adds a preferred label selector.
func (b *NodeAffinityBuilder) PreferLabel(weight int32, key string, values ...string) *NodeAffinityBuilder {
	term := PreferredSchedulingTerm{
		Weight: weight,
		Preference: NodeSelectorTerm{
			MatchExpressions: []NodeSelectorRequirement{
				{
					Key:      key,
					Operator: OpIn,
					Values:   values,
				},
			},
		},
	}
	b.affinity.PreferredDuringSchedulingIgnoredDuringExecution = append(
		b.affinity.PreferredDuringSchedulingIgnoredDuringExecution,
		term,
	)
	return b
}

// Build builds the node affinity.
func (b *NodeAffinityBuilder) Build() *NodeAffinity {
	return b.affinity
}
