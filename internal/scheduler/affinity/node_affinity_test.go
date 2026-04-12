package affinity

import (
	"testing"
)

func TestNewEvaluator(t *testing.T) {
	e := NewEvaluator()
	if e == nil {
		t.Fatal("expected non-nil evaluator")
	}
}

func TestNodeSelectorRequirement_OpIn_Match(t *testing.T) {
	req := NodeSelectorRequirement{
		Key:      "zone",
		Operator: OpIn,
		Values:   []string{"us-east-1a", "us-west-2b"},
	}

	labels := map[string]string{"zone": "us-east-1a"}
	if !req.Match(labels) {
		t.Error("expected match")
	}
}

func TestNodeSelectorRequirement_OpIn_NoMatch(t *testing.T) {
	req := NodeSelectorRequirement{
		Key:      "zone",
		Operator: OpIn,
		Values:   []string{"us-west-2b"},
	}

	labels := map[string]string{"zone": "us-east-1a"}
	if req.Match(labels) {
		t.Error("expected no match")
	}
}

func TestNodeSelectorRequirement_OpIn_MissingKey(t *testing.T) {
	req := NodeSelectorRequirement{
		Key:      "zone",
		Operator: OpIn,
		Values:   []string{"us-east-1a"},
	}

	labels := map[string]string{}
	if req.Match(labels) {
		t.Error("expected no match for missing key")
	}
}

func TestNodeSelectorRequirement_OpNotIn_Match(t *testing.T) {
	req := NodeSelectorRequirement{
		Key:      "zone",
		Operator: OpNotIn,
		Values:   []string{"us-west-2b"},
	}

	labels := map[string]string{"zone": "us-east-1a"}
	if !req.Match(labels) {
		t.Error("expected match (value not in list)")
	}
}

func TestNodeSelectorRequirement_OpNotIn_NoMatch(t *testing.T) {
	req := NodeSelectorRequirement{
		Key:      "zone",
		Operator: OpNotIn,
		Values:   []string{"us-east-1a"},
	}

	labels := map[string]string{"zone": "us-east-1a"}
	if req.Match(labels) {
		t.Error("expected no match (value is in list)")
	}
}

func TestNodeSelectorRequirement_OpNotIn_MissingKey(t *testing.T) {
	req := NodeSelectorRequirement{
		Key:      "zone",
		Operator: OpNotIn,
		Values:   []string{"us-east-1a"},
	}

	labels := map[string]string{}
	if !req.Match(labels) {
		t.Error("missing key should match NotIn")
	}
}

func TestNodeSelectorRequirement_OpExists(t *testing.T) {
	req := NodeSelectorRequirement{
		Key:      "zone",
		Operator: OpExists,
	}

	if !req.Match(map[string]string{"zone": "us-east-1a"}) {
		t.Error("expected match for existing key")
	}
	if req.Match(map[string]string{}) {
		t.Error("expected no match for missing key")
	}
}

func TestNodeSelectorRequirement_OpDoesNotExist(t *testing.T) {
	req := NodeSelectorRequirement{
		Key:      "zone",
		Operator: OpDoesNotExist,
	}

	if !req.Match(map[string]string{}) {
		t.Error("expected match for missing key")
	}
	if req.Match(map[string]string{"zone": "us-east-1a"}) {
		t.Error("expected no match for existing key")
	}
}

func TestNodeSelectorRequirement_OpGt_Numeric(t *testing.T) {
	req := NodeSelectorRequirement{
		Key:      "priority",
		Operator: OpGt,
		Values:   []string{"5"},
	}

	if !req.Match(map[string]string{"priority": "10"}) {
		t.Error("10 > 5 should match")
	}
	if req.Match(map[string]string{"priority": "3"}) {
		t.Error("3 > 5 should not match")
	}
	if req.Match(map[string]string{"priority": "5"}) {
		t.Error("5 > 5 should not match")
	}
}

func TestNodeSelectorRequirement_OpLt_Numeric(t *testing.T) {
	req := NodeSelectorRequirement{
		Key:      "priority",
		Operator: OpLt,
		Values:   []string{"5"},
	}

	if !req.Match(map[string]string{"priority": "3"}) {
		t.Error("3 < 5 should match")
	}
	if req.Match(map[string]string{"priority": "10"}) {
		t.Error("10 < 5 should not match")
	}
}

func TestNodeSelectorRequirement_OpGt_MissingKey(t *testing.T) {
	req := NodeSelectorRequirement{
		Key:      "priority",
		Operator: OpGt,
		Values:   []string{"5"},
	}

	if req.Match(map[string]string{}) {
		t.Error("missing key should not match Gt")
	}
}

func TestNodeSelectorRequirement_OpLt_MissingKey(t *testing.T) {
	req := NodeSelectorRequirement{
		Key:      "priority",
		Operator: OpLt,
		Values:   []string{"5"},
	}

	if req.Match(map[string]string{}) {
		t.Error("missing key should not match Lt")
	}
}

func TestNodeSelectorRequirement_OpGt_StringComparison(t *testing.T) {
	req := NodeSelectorRequirement{
		Key:      "version",
		Operator: OpGt,
		Values:   []string{"aaa"},
	}

	if !req.Match(map[string]string{"version": "bbb"}) {
		t.Error("bbb > aaa (string) should match")
	}
	if req.Match(map[string]string{"version": "aaa"}) {
		t.Error("aaa > aaa should not match")
	}
}

func TestNodeSelectorRequirement_UnknownOperator(t *testing.T) {
	req := NodeSelectorRequirement{
		Key:      "zone",
		Operator: "Unknown",
		Values:   []string{"us-east-1a"},
	}

	if req.Match(map[string]string{"zone": "us-east-1a"}) {
		t.Error("unknown operator should not match")
	}
}

func TestNodeSelectorTerm_Match_AllExpressionsMatch(t *testing.T) {
	term := NodeSelectorTerm{
		MatchExpressions: []NodeSelectorRequirement{
			{Key: "zone", Operator: OpIn, Values: []string{"us-east-1a"}},
			{Key: "tier", Operator: OpIn, Values: []string{"production"}},
		},
	}

	labels := map[string]string{"zone": "us-east-1a", "tier": "production"}
	if !term.Match(labels) {
		t.Error("expected match when all expressions match")
	}
}

func TestNodeSelectorTerm_Match_OneExpressionFails(t *testing.T) {
	term := NodeSelectorTerm{
		MatchExpressions: []NodeSelectorRequirement{
			{Key: "zone", Operator: OpIn, Values: []string{"us-east-1a"}},
			{Key: "tier", Operator: OpIn, Values: []string{"production"}},
		},
	}

	labels := map[string]string{"zone": "us-east-1a", "tier": "staging"}
	if term.Match(labels) {
		t.Error("expected no match when one expression fails")
	}
}

func TestNodeSelectorTerm_Match_EmptyExpressions(t *testing.T) {
	term := NodeSelectorTerm{}
	labels := map[string]string{"zone": "us-east-1a"}

	if !term.Match(labels) {
		t.Error("empty term should match any labels")
	}
}

func TestNodeSelectorTerm_MatchFields(t *testing.T) {
	term := NodeSelectorTerm{
		MatchFields: []NodeSelectorRequirement{
			{Key: "name", Operator: OpIn, Values: []string{"node-1"}},
		},
	}

	labels := map[string]string{"name": "node-1"}
	if !term.Match(labels) {
		t.Error("expected match on match fields")
	}
}

func TestEvaluator_MatchesRequired_NilAffinity(t *testing.T) {
	e := NewEvaluator()
	match, err := e.MatchesRequired(map[string]string{"zone": "a"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !match {
		t.Error("nil affinity should match")
	}
}

func TestEvaluator_MatchesRequired_EmptyRequired(t *testing.T) {
	e := NewEvaluator()
	affinity := &NodeAffinity{}
	match, err := e.MatchesRequired(map[string]string{}, affinity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !match {
		t.Error("empty required terms should match")
	}
}

func TestEvaluator_MatchesRequired_Match(t *testing.T) {
	e := NewEvaluator()
	affinity := &NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: []NodeSelectorTerm{
			{
				MatchExpressions: []NodeSelectorRequirement{
					{Key: "zone", Operator: OpIn, Values: []string{"us-east-1a"}},
				},
			},
		},
	}

	match, err := e.MatchesRequired(map[string]string{"zone": "us-east-1a"}, affinity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !match {
		t.Error("expected match")
	}
}

func TestEvaluator_MatchesRequired_NoMatch(t *testing.T) {
	e := NewEvaluator()
	affinity := &NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: []NodeSelectorTerm{
			{
				MatchExpressions: []NodeSelectorRequirement{
					{Key: "zone", Operator: OpIn, Values: []string{"us-east-1a"}},
				},
			},
		},
	}

	match, err := e.MatchesRequired(map[string]string{"zone": "us-west-2b"}, affinity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if match {
		t.Error("expected no match")
	}
}

func TestEvaluator_ScorePreferred_NilAffinity(t *testing.T) {
	e := NewEvaluator()
	score := e.ScorePreferred(map[string]string{}, nil)
	if score != 0 {
		t.Errorf("expected score=0, got %d", score)
	}
}

func TestEvaluator_ScorePreferred_EmptyPreferred(t *testing.T) {
	e := NewEvaluator()
	score := e.ScorePreferred(map[string]string{}, &NodeAffinity{})
	if score != 0 {
		t.Errorf("expected score=0, got %d", score)
	}
}

func TestEvaluator_ScorePreferred_Match(t *testing.T) {
	e := NewEvaluator()
	affinity := &NodeAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []PreferredSchedulingTerm{
			{
				Weight: 50,
				Preference: NodeSelectorTerm{
					MatchExpressions: []NodeSelectorRequirement{
						{Key: "zone", Operator: OpIn, Values: []string{"us-east-1a"}},
					},
				},
			},
		},
	}

	score := e.ScorePreferred(map[string]string{"zone": "us-east-1a"}, affinity)
	if score != 50 {
		t.Errorf("expected score=50, got %d", score)
	}
}

func TestEvaluator_ScorePreferred_NoMatch(t *testing.T) {
	e := NewEvaluator()
	affinity := &NodeAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []PreferredSchedulingTerm{
			{
				Weight: 50,
				Preference: NodeSelectorTerm{
					MatchExpressions: []NodeSelectorRequirement{
						{Key: "zone", Operator: OpIn, Values: []string{"us-east-1a"}},
					},
				},
			},
		},
	}

	score := e.ScorePreferred(map[string]string{"zone": "us-west-2b"}, affinity)
	if score != 0 {
		t.Errorf("expected score=0 for no match, got %d", score)
	}
}

func TestEvaluator_ScorePreferred_MultipleTerms(t *testing.T) {
	e := NewEvaluator()
	affinity := &NodeAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []PreferredSchedulingTerm{
			{
				Weight: 30,
				Preference: NodeSelectorTerm{
					MatchExpressions: []NodeSelectorRequirement{
						{Key: "zone", Operator: OpIn, Values: []string{"us-east-1a"}},
					},
				},
			},
			{
				Weight: 20,
				Preference: NodeSelectorTerm{
					MatchExpressions: []NodeSelectorRequirement{
						{Key: "tier", Operator: OpIn, Values: []string{"production"}},
					},
				},
			},
			{
				Weight: 10,
				Preference: NodeSelectorTerm{
					MatchExpressions: []NodeSelectorRequirement{
						{Key: "gpu", Operator: OpExists},
					},
				},
			},
		},
	}

	labels := map[string]string{
		"zone": "us-east-1a",
		"tier": "production",
		"gpu":  "true",
	}

	score := e.ScorePreferred(labels, affinity)
	if score != 60 {
		t.Errorf("expected score=60 (30+20+10), got %d", score)
	}
}

func TestEvaluator_ScorePreferred_PartialMatch(t *testing.T) {
	e := NewEvaluator()
	affinity := &NodeAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []PreferredSchedulingTerm{
			{
				Weight: 30,
				Preference: NodeSelectorTerm{
					MatchExpressions: []NodeSelectorRequirement{
						{Key: "zone", Operator: OpIn, Values: []string{"us-east-1a"}},
					},
				},
			},
			{
				Weight: 20,
				Preference: NodeSelectorTerm{
					MatchExpressions: []NodeSelectorRequirement{
						{Key: "tier", Operator: OpIn, Values: []string{"production"}},
					},
				},
			},
		},
	}

	labels := map[string]string{"zone": "us-east-1a"}
	score := e.ScorePreferred(labels, affinity)
	if score != 30 {
		t.Errorf("expected score=30, got %d", score)
	}
}

func TestNewNodeAffinityBuilder(t *testing.T) {
	b := NewNodeAffinityBuilder()
	if b == nil {
		t.Fatal("expected non-nil builder")
	}
}

func TestNodeAffinityBuilder_RequireLabel(t *testing.T) {
	b := NewNodeAffinityBuilder()
	affinity := b.RequireLabel("zone", "us-east-1a").Build()

	if len(affinity.RequiredDuringSchedulingIgnoredDuringExecution) != 1 {
		t.Fatal("expected 1 required term")
	}
	expr := affinity.RequiredDuringSchedulingIgnoredDuringExecution[0].MatchExpressions[0]
	if expr.Key != "zone" {
		t.Errorf("expected key=zone, got %s", expr.Key)
	}
	if expr.Operator != OpIn {
		t.Errorf("expected OpIn, got %s", expr.Operator)
	}
	if len(expr.Values) != 1 || expr.Values[0] != "us-east-1a" {
		t.Errorf("expected values=[us-east-1a], got %v", expr.Values)
	}
}

func TestNodeAffinityBuilder_RequireLabel_MultipleValues(t *testing.T) {
	b := NewNodeAffinityBuilder()
	affinity := b.RequireLabel("zone", "us-east-1a", "us-west-2b").Build()

	expr := affinity.RequiredDuringSchedulingIgnoredDuringExecution[0].MatchExpressions[0]
	if len(expr.Values) != 2 {
		t.Errorf("expected 2 values, got %d", len(expr.Values))
	}
}

func TestNodeAffinityBuilder_PreferLabel(t *testing.T) {
	b := NewNodeAffinityBuilder()
	affinity := b.PreferLabel(50, "tier", "production").Build()

	if len(affinity.PreferredDuringSchedulingIgnoredDuringExecution) != 1 {
		t.Fatal("expected 1 preferred term")
	}
	term := affinity.PreferredDuringSchedulingIgnoredDuringExecution[0]
	if term.Weight != 50 {
		t.Errorf("expected weight=50, got %d", term.Weight)
	}
	expr := term.Preference.MatchExpressions[0]
	if expr.Key != "tier" {
		t.Errorf("expected key=tier, got %s", expr.Key)
	}
}

func TestNodeAffinityBuilder_Chained(t *testing.T) {
	b := NewNodeAffinityBuilder()
	affinity := b.
		RequireLabel("zone", "us-east-1a").
		PreferLabel(50, "tier", "production").
		PreferLabel(30, "gpu", "true").
		Build()

	if len(affinity.RequiredDuringSchedulingIgnoredDuringExecution) != 1 {
		t.Errorf("expected 1 required term, got %d", len(affinity.RequiredDuringSchedulingIgnoredDuringExecution))
	}
	if len(affinity.PreferredDuringSchedulingIgnoredDuringExecution) != 2 {
		t.Errorf("expected 2 preferred terms, got %d", len(affinity.PreferredDuringSchedulingIgnoredDuringExecution))
	}
}

func TestNodeAffinityBuilder_E2E(t *testing.T) {
	b := NewNodeAffinityBuilder()
	affinity := b.
		RequireLabel("zone", "us-east-1a").
		PreferLabel(100, "instance", "gpu").
		Build()

	e := NewEvaluator()

	labels := map[string]string{"zone": "us-east-1a", "instance": "gpu"}

	match, err := e.MatchesRequired(labels, affinity)
	if err != nil {
		t.Fatalf("matches required: %v", err)
	}
	if !match {
		t.Error("expected required match")
	}

	score := e.ScorePreferred(labels, affinity)
	if score != 100 {
		t.Errorf("expected score=100, got %d", score)
	}
}

func TestNodeAffinityBuilder_E2E_RequiredFails(t *testing.T) {
	b := NewNodeAffinityBuilder()
	affinity := b.RequireLabel("zone", "us-east-1a").Build()

	e := NewEvaluator()
	match, err := e.MatchesRequired(map[string]string{"zone": "us-west-2b"}, affinity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if match {
		t.Error("expected required to fail for wrong zone")
	}
}

func TestCompareValues_Numeric(t *testing.T) {
	if compareValues("10", "5") <= 0 {
		t.Error("10 > 5")
	}
	if compareValues("3", "5") >= 0 {
		t.Error("3 < 5")
	}
	if compareValues("5", "5") != 0 {
		t.Error("5 == 5")
	}
}

func TestCompareValues_String(t *testing.T) {
	if compareValues("bbb", "aaa") <= 0 {
		t.Error("bbb > aaa")
	}
	if compareValues("aaa", "bbb") >= 0 {
		t.Error("aaa < bbb")
	}
	if compareValues("same", "same") != 0 {
		t.Error("same == same")
	}
}

func TestCompareValues_Mixed(t *testing.T) {
	result := compareValues("abc", "5")
	if result <= 0 {
		t.Errorf("expected 'abc' > '5' in string comparison, got %d", result)
	}
}
