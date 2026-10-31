package amp

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fxckcode/baton/internal/schema"
)

func TestParseSubscriptionDays(t *testing.T) {
	data, err := os.ReadFile("testdata/subscription.txt")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	got, ok := Parse(data, now)
	if !ok || got.Confidence != schema.ConfidenceExact {
		t.Fatalf("%+v", got)
	}
	if got.Plan == nil || *got.Plan != "Megawatt" {
		t.Fatalf("plan %+v", got.Plan)
	}
	if got.RemainingPct == nil || *got.RemainingPct != 97 {
		t.Fatalf("remaining %v", got.RemainingPct)
	}
	if len(got.Windows) != 2 || got.Windows[0].Name != "other" || *got.Windows[0].RemainingPct != 97 {
		t.Fatalf("other %+v", got.Windows)
	}
	if got.Windows[1].Name != "orb" || *got.Windows[1].RemainingPct != 100 {
		t.Fatalf("orb %+v", got.Windows[1])
	}
	want := now.Add(29 * 24 * time.Hour)
	if got.Windows[0].ResetAt == nil || !got.Windows[0].ResetAt.Equal(want) {
		t.Fatalf("reset %v", got.Windows[0].ResetAt)
	}
	assertNoEmail(t, got)
}

func TestParseMegawattMonthHasNoInventedReset(t *testing.T) {
	data, err := os.ReadFile("testdata/megawatt.txt")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	got, ok := Parse(data, now)
	if !ok || got.RemainingPct == nil || *got.RemainingPct != 100 {
		t.Fatalf("%+v", got)
	}
	if got.Plan == nil || *got.Plan != "Megawatt" {
		t.Fatalf("plan %+v", got.Plan)
	}
	for _, w := range got.Windows {
		if w.ResetAt != nil {
			t.Fatalf("invented reset %+v", w)
		}
	}
	assertNoEmail(t, got)
}

func TestParseCreditsAreNotAPercent(t *testing.T) {
	data, err := os.ReadFile("testdata/credits.txt")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := Parse(data, time.Now())
	if !ok || got.Confidence != schema.ConfidenceUnknown || got.RemainingPct != nil {
		t.Fatalf("%+v", got)
	}
	if got.Note != "Amp reported a credit balance, not a percentage" {
		t.Fatalf("note %q", got.Note)
	}
	assertNoEmail(t, got)
}

func assertNoEmail(t *testing.T, agent schema.Agent) {
	t.Helper()
	raw, err := schema.Marshal(schema.Report{GeneratedAt: time.Unix(0, 0).UTC(), Agents: []schema.Agent{agent}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "you@example.com") || strings.Contains(agent.Note, "example.com") {
		t.Fatalf("email leaked: %s", raw)
	}
}
