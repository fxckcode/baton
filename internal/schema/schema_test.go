package schema

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"
)

func TestMarshalGolden(t *testing.T) {
	plan := "plus"
	rem := 38.0
	reset := time.Date(2026, 10, 31, 1, 12, 0, 0, time.UTC)
	obs := time.Date(2026, 10, 30, 22, 41, 10, 0, time.UTC)
	report := Report{
		Schema:      Version,
		GeneratedAt: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC),
		Agents: []Agent{{
			Agent:     "codex",
			Installed: true,
			Version:   "0.80.0",
			Plan:      &plan,
			Windows: []Window{{
				Name:         "5h",
				RemainingPct: &rem,
				ResetAt:      &reset,
			}},
			RemainingPct: &rem,
			Confidence:   ConfidenceExact,
			Source:       "local:codex-rollout",
			ObservedAt:   &obs,
		}},
		Recommendation: &Recommendation{
			Agent:   "codex",
			Command: "codex",
			Reason:  "first in chain with remaining_pct >= 15",
		},
	}
	got, err := Marshal(report)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want, err := os.ReadFile("testdata/report.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("golden mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestUnknownOmitsPercentAndKeepsEmptyWindows(t *testing.T) {
	raw, err := Marshal(Report{
		GeneratedAt: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC),
		Agents: []Agent{{
			Agent:      "cursor",
			Installed:  true,
			Confidence: ConfidenceUnknown,
			Source:     "none",
			Note:       "quota is not read in v0.1",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["schema"] != Version {
		t.Fatalf("schema: %v", doc["schema"])
	}
	agents := doc["agents"].([]any)
	agent := agents[0].(map[string]any)
	if agent["remaining_pct"] != nil || agent["plan"] != nil || agent["observed_at"] != nil {
		t.Fatalf("unknown agent invented a value: %#v", agent)
	}
	windows, ok := agent["windows"].([]any)
	if !ok || len(windows) != 0 {
		t.Fatalf("windows: %#v", agent["windows"])
	}
	if _, ok := doc["recommendation"]; ok {
		t.Fatal("recommendation should be omitted")
	}
	if agent["note"] != "quota is not read in v0.1" {
		t.Fatalf("note: %v", agent["note"])
	}
}

func TestMinRemaining(t *testing.T) {
	hi := 38.0
	lo := 19.0
	neg := -53.12
	got := MinRemaining([]Window{
		{Name: "5h", RemainingPct: &hi},
		{Name: "weekly"},
		{Name: "over", RemainingPct: &neg},
		{Name: "weekly2", RemainingPct: &lo},
	})
	if got == nil || *got != neg {
		t.Fatalf("min = %v", got)
	}
	if MinRemaining(nil) != nil {
		t.Fatal("expected nil")
	}
}

func TestRemainingFromUsed(t *testing.T) {
	cases := []struct {
		used float64
		want float64
		ok   bool
	}{
		{62, 38, true},
		{23.5, 76.5, true},
		{100, 0, true},
		{153.1232, -53.12, true},
		{0, 100, true},
		{-1, 0, false},
		{math.NaN(), 0, false},
		{math.Inf(1), 0, false},
	}
	for _, tc := range cases {
		got, ok := RemainingFromUsed(tc.used)
		if ok != tc.ok {
			t.Fatalf("used %v ok=%v", tc.used, ok)
		}
		if !ok {
			continue
		}
		if got == nil || *got != tc.want {
			t.Fatalf("used %v got %v want %v", tc.used, got, tc.want)
		}
	}
}

func TestApplyStale(t *testing.T) {
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	old := now.Add(-7 * time.Hour)
	fresh := now.Add(-time.Hour)
	exact := now.Add(-6 * time.Hour)

	stale := Agent{ObservedAt: &old}
	stale.ApplyStale(now, 6*time.Hour)
	if !stale.Stale {
		t.Fatal("expected stale")
	}
	freshA := Agent{ObservedAt: &fresh}
	freshA.ApplyStale(now, 6*time.Hour)
	if freshA.Stale {
		t.Fatal("fresh marked stale")
	}
	edge := Agent{ObservedAt: &exact}
	edge.ApplyStale(now, 6*time.Hour)
	if edge.Stale {
		t.Fatal("age equal to max_age is not stale")
	}
	missing := Agent{}
	missing.ApplyStale(now, 6*time.Hour)
	if missing.Stale {
		t.Fatal("missing observation is not stale")
	}
}

func TestConfidenceValues(t *testing.T) {
	for _, c := range []Confidence{ConfidenceExact, ConfidenceEstimated, ConfidenceUnknown} {
		if !c.Valid() {
			t.Fatalf("%s should be valid", c)
		}
	}
	if Confidence("guess").Valid() {
		t.Fatal("guess is not a confidence")
	}
}
