package application

import "github.com/saurabh/distributed-rate-limiter/internal/domain/entity"

// RuleResolver picks the best enabled rule for a request route.
type RuleResolver struct{}

func NewRuleResolver() *RuleResolver {
	return &RuleResolver{}
}

// Resolve returns the most specific enabled rule that matches route,
// or nil when no rule applies (caller should allow the request).
func (RuleResolver) Resolve(rules []entity.Rule, route string) *entity.Rule {
	var best *entity.Rule
	bestScore := -1

	for i := range rules {
		rule := &rules[i]
		if !rule.Enabled {
			continue
		}
		score := rule.Specificity(route)
		if score <= 0 {
			continue
		}
		if score > bestScore {
			bestScore = score
			best = rule
		}
	}

	return best
}
