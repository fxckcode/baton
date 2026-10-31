// Package next chooses the first eligible agent in a priority chain.
package next

import (
	"strconv"

	"github.com/fxckcode/baton/internal/schema"
)

// Policy is the handover rule. Zero values skip estimated and stale rows;
// config defaults turn AllowStale on before calling Select.
type Policy struct {
	Chain          []string
	Threshold      float64
	AllowEstimated bool
	AllowStale     bool
}

// Decision is the result of Select. OK is false when no agent qualifies.
type Decision struct {
	Agent  schema.Agent
	OK     bool
	Reason string
}

// Select walks policy.Chain and returns the first eligible agent.
// A later agent with a higher remaining percent does not win.
func Select(agents []schema.Agent, policy Policy) Decision {
	byID := make(map[string]schema.Agent, len(agents))
	for _, a := range agents {
		if _, seen := byID[a.Agent]; seen {
			continue
		}
		byID[a.Agent] = a
	}
	limit := strconv.FormatFloat(policy.Threshold, 'f', -1, 64)
	for _, id := range policy.Chain {
		a, ok := byID[id]
		if !ok || !eligible(a, policy) {
			continue
		}
		return Decision{
			Agent:  a,
			OK:     true,
			Reason: "first in chain with remaining_pct >= " + limit,
		}
	}
	return Decision{Reason: "no agent at or above threshold"}
}

func eligible(a schema.Agent, policy Policy) bool {
	if !a.Installed || a.RemainingPct == nil {
		return false
	}
	switch a.Confidence {
	case schema.ConfidenceExact:
	case schema.ConfidenceEstimated:
		if !policy.AllowEstimated {
			return false
		}
	default:
		return false
	}
	if a.Stale && !policy.AllowStale {
		return false
	}
	return *a.RemainingPct >= policy.Threshold
}
