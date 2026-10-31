package next

import (
	"testing"

	"github.com/fxckcode/baton/internal/schema"
)

func agent(id string, installed bool, conf schema.Confidence, pct *float64, stale bool) schema.Agent {
	return schema.Agent{
		Agent:        id,
		Installed:    installed,
		Confidence:   conf,
		RemainingPct: pct,
		Stale:        stale,
		Source:       "test",
	}
}

func TestSelectThresholdAndChainOrder(t *testing.T) {
	low := 10.0
	mid := 20.0
	high := 90.0
	agents := []schema.Agent{
		agent("antigravity", true, schema.ConfidenceExact, &low, false),
		agent("cursor", true, schema.ConfidenceUnknown, &high, false),
		agent("claude", true, schema.ConfidenceExact, &mid, false),
		agent("codex", true, schema.ConfidenceExact, &high, false),
	}
	policy := Policy{Chain: []string{"antigravity", "cursor", "claude", "codex"}, Threshold: 20}

	got := Select(agents, policy)
	if !got.OK || got.Agent.Agent != "claude" {
		t.Fatalf("got %+v", got)
	}
	at := Select([]schema.Agent{agent("claude", true, schema.ConfidenceExact, &mid, false)}, Policy{
		Chain: []string{"claude"}, Threshold: 20,
	})
	if !at.OK {
		t.Fatal("remaining equal to threshold must be eligible")
	}
}

func TestSelectNoneWhenAllBelowOrUnknown(t *testing.T) {
	low := 14.9
	high := 80.0
	agents := []schema.Agent{
		agent("antigravity", true, schema.ConfidenceExact, &low, false),
		agent("cursor", true, schema.ConfidenceUnknown, nil, false),
		agent("claude", false, schema.ConfidenceExact, &high, false),
		agent("codex", true, schema.ConfidenceExact, nil, false),
	}
	got := Select(agents, Policy{
		Chain:     []string{"antigravity", "cursor", "claude", "codex"},
		Threshold: 15,
	})
	if got.OK {
		t.Fatalf("expected none, got %+v", got)
	}
}

func TestSelectSkipsEstimatedAndStaleUnlessAllowed(t *testing.T) {
	pct := 50.0
	estimated := agent("amp", true, schema.ConfidenceEstimated, &pct, false)
	stale := agent("codex", true, schema.ConfidenceExact, &pct, true)
	chain := []string{"amp", "codex"}

	if Select([]schema.Agent{estimated, stale}, Policy{Chain: chain, Threshold: 15}).OK {
		t.Fatal("estimated and stale should be skipped by default")
	}
	got := Select([]schema.Agent{estimated, stale}, Policy{
		Chain: chain, Threshold: 15, AllowEstimated: true,
	})
	if !got.OK || got.Agent.Agent != "amp" {
		t.Fatalf("estimated allowed: %+v", got)
	}
	got = Select([]schema.Agent{stale}, Policy{Chain: chain, Threshold: 15, AllowStale: true})
	if !got.OK || got.Agent.Agent != "codex" {
		t.Fatalf("stale allowed: %+v", got)
	}
}

func TestSelectEmptyChainAndMissingAgent(t *testing.T) {
	pct := 100.0
	agents := []schema.Agent{agent("claude", true, schema.ConfidenceExact, &pct, false)}
	if Select(agents, Policy{Threshold: 15}).OK {
		t.Fatal("empty chain")
	}
	if Select(agents, Policy{Chain: []string{"missing"}, Threshold: 15}).OK {
		t.Fatal("missing agent")
	}
}

func TestSelectDoesNotPreferHigherLaterPercent(t *testing.T) {
	first := 15.0
	later := 99.0
	got := Select([]schema.Agent{
		agent("claude", true, schema.ConfidenceExact, &first, false),
		agent("codex", true, schema.ConfidenceExact, &later, false),
	}, Policy{Chain: []string{"claude", "codex"}, Threshold: 15})
	if got.Agent.Agent != "claude" {
		t.Fatalf("got %s", got.Agent.Agent)
	}
	if got.Reason == "" {
		t.Fatal("reason empty")
	}
}
