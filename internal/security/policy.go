package security

import (
	"context"
	"fmt"
	"sync"
)

// Action represents a security action.
type Action string

const (
	ActionAllow Action = "allow"
	ActionDeny  Action = "deny"
)

// PolicyRule defines a single authorisation rule.
type PolicyRule struct {
	ID       string   `json:"id"`
	Roles    []string `json:"roles"`
	Resource string   `json:"resource"`
	Actions  []string `json:"actions"`
	Effect   Action   `json:"effect"`
}

// EvalRequest holds the context for a policy evaluation.
type EvalRequest struct {
	UserID   string
	Roles    []string
	Resource string
	Action   string
}

// EvalResult holds the outcome of a policy evaluation.
type EvalResult struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// PolicyEvaluator evaluates access-control policies.
type PolicyEvaluator interface {
	Evaluate(ctx context.Context, req EvalRequest) (*EvalResult, error)
	AddRule(rule PolicyRule) error
	RemoveRule(ruleID string) error
	ListRules() []PolicyRule
}

// DefaultPolicyEvaluator provides an in-memory RBAC evaluator.
type DefaultPolicyEvaluator struct {
	mu    sync.RWMutex
	rules []PolicyRule
}

func NewDefaultPolicyEvaluator() *DefaultPolicyEvaluator {
	return &DefaultPolicyEvaluator{}
}

func (e *DefaultPolicyEvaluator) Evaluate(_ context.Context, req EvalRequest) (*EvalResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, rule := range e.rules {
		if !matchResource(rule.Resource, req.Resource) {
			continue
		}
		if !matchAction(rule.Actions, req.Action) {
			continue
		}
		if !matchRoles(rule.Roles, req.Roles) {
			continue
		}

		if rule.Effect == ActionDeny {
			return &EvalResult{Allowed: false, Reason: fmt.Sprintf("denied by rule %s", rule.ID)}, nil
		}
		return &EvalResult{Allowed: true, Reason: fmt.Sprintf("allowed by rule %s", rule.ID)}, nil
	}

	return &EvalResult{Allowed: false, Reason: "no matching rule found (default deny)"}, nil
}

func (e *DefaultPolicyEvaluator) AddRule(rule PolicyRule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, r := range e.rules {
		if r.ID == rule.ID {
			return fmt.Errorf("rule %s already exists", rule.ID)
		}
	}
	e.rules = append(e.rules, rule)
	return nil
}

func (e *DefaultPolicyEvaluator) RemoveRule(ruleID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, r := range e.rules {
		if r.ID == ruleID {
			e.rules = append(e.rules[:i], e.rules[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("rule %s not found", ruleID)
}

func (e *DefaultPolicyEvaluator) ListRules() []PolicyRule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	out := make([]PolicyRule, len(e.rules))
	copy(out, e.rules)
	return out
}

func matchResource(pattern, resource string) bool {
	if pattern == "*" {
		return true
	}
	return pattern == resource
}

func matchAction(allowed []string, action string) bool {
	for _, a := range allowed {
		if a == "*" || a == action {
			return true
		}
	}
	return false
}

func matchRoles(ruleRoles, userRoles []string) bool {
	if len(ruleRoles) == 0 {
		return true
	}
	roleSet := make(map[string]bool, len(userRoles))
	for _, r := range userRoles {
		roleSet[r] = true
	}
	for _, r := range ruleRoles {
		if roleSet[r] {
			return true
		}
	}
	return false
}

var _ PolicyEvaluator = (*DefaultPolicyEvaluator)(nil)
