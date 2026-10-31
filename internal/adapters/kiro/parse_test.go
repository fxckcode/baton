package kiro

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fxckcode/baton/internal/agent"
	"github.com/fxckcode/baton/internal/run"
	"github.com/fxckcode/baton/internal/schema"
)

func TestParseOverage(t *testing.T) {
	data, err := os.ReadFile("testdata/overage.txt")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	got, ok := Parse(data, now)
	if !ok || got.Confidence != schema.ConfidenceExact {
		t.Fatalf("%+v", got)
	}
	if got.RemainingPct == nil || *got.RemainingPct != -53.12 {
		t.Fatalf("remaining %v", got.RemainingPct)
	}
	if got.Plan == nil || *got.Plan != "KIRO POWER" {
		t.Fatalf("plan %+v", got.Plan)
	}
	if len(got.Windows) != 1 || got.Windows[0].Name != "monthly" {
		t.Fatalf("windows %+v", got.Windows)
	}
	want := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if got.Windows[0].ResetAt == nil || !got.Windows[0].ResetAt.Equal(want) {
		t.Fatalf("reset %v", got.Windows[0].ResetAt)
	}
}

func TestParseANSIAndBonus(t *testing.T) {
	ansi, err := os.ReadFile("testdata/ansi.txt")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := Parse(ansi, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if !ok || got.RemainingPct == nil || *got.RemainingPct != -53.12 {
		t.Fatalf("ansi %+v", got)
	}
	bonus, err := os.ReadFile("testdata/bonus.txt")
	if err != nil {
		t.Fatal(err)
	}
	got, ok = Parse(bonus, time.Now())
	if !ok || got.RemainingPct == nil || *got.RemainingPct != 75 {
		t.Fatalf("bonus %+v windows %+v", got.RemainingPct, got.Windows)
	}
	if len(got.Windows) != 2 || got.Windows[1].Name != "bonus" || *got.Windows[1].RemainingPct != 75 {
		t.Fatalf("windows %+v", got.Windows)
	}
}

func TestParseBarOnlyIsUnknown(t *testing.T) {
	data, err := os.ReadFile("testdata/bar.txt")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := Parse(data, time.Now())
	if ok && (got.RemainingPct != nil || got.Confidence == schema.ConfidenceExact) {
		t.Fatalf("bar became quota: %+v", got)
	}
}

func TestStatusArgv(t *testing.T) {
	data, err := os.ReadFile("testdata/overage.txt")
	if err != nil {
		t.Fatal(err)
	}
	runner := &run.MapRunner{Scripts: map[string]run.Script{
		"/bin/kiro-cli chat --no-interactive /usage": {Stdout: string(data)},
	}}
	var ad Adapter
	got := ad.Status(context.Background(), agent.Env{Runner: runner}, agent.Detection{Installed: true, Binary: "/bin/kiro-cli"})
	if got.RemainingPct == nil || *got.RemainingPct != -53.12 {
		t.Fatalf("%+v", got)
	}
	if len(runner.Calls) != 1 || runner.Calls[0] != "/bin/kiro-cli chat --no-interactive /usage" {
		t.Fatalf("calls %v", runner.Calls)
	}
}
