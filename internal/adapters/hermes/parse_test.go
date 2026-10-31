package hermes

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/fxckcode/baton/internal/agent"
	"github.com/fxckcode/baton/internal/run"
	"github.com/fxckcode/baton/internal/schema"
)

func TestParseUsage(t *testing.T) {
	data, err := os.ReadFile("testdata/usage.json")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := Parse(data, time.Time{})
	if !ok || got.Confidence != schema.ConfidenceExact {
		t.Fatalf("%+v", got)
	}
	if got.Plan == nil || *got.Plan != "Plus" {
		t.Fatalf("plan %+v", got.Plan)
	}
	if got.RemainingPct == nil || *got.RemainingPct != 63 {
		t.Fatalf("remaining %v", got.RemainingPct)
	}
	if len(got.Windows) != 2 || got.Windows[0].Name != "Session" || *got.Windows[0].RemainingPct != 63 {
		t.Fatalf("session %+v", got.Windows)
	}
	if got.Windows[1].Name != "Weekly" || *got.Windows[1].RemainingPct != 87.5 {
		t.Fatalf("weekly %+v", got.Windows[1])
	}
	want := time.Date(2026, 9, 19, 7, 58, 55, 0, time.UTC)
	if got.ObservedAt == nil || !got.ObservedAt.Equal(want) {
		t.Fatalf("observed %v", got.ObservedAt)
	}
}

func TestParseUnavailableDoesNotCopyReason(t *testing.T) {
	data, err := os.ReadFile("testdata/unavailable.json")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := Parse(data, time.Time{})
	if !ok || got.Confidence != schema.ConfidenceUnknown || got.RemainingPct != nil {
		t.Fatalf("%+v", got)
	}
	if strings.Contains(got.Note, "sk-ant-test-secret") || strings.Contains(got.Note, "rejected") {
		t.Fatalf("note copied provider text: %q", got.Note)
	}
	raw, err := os.ReadFile("testdata/null-window.json")
	if err != nil {
		t.Fatal(err)
	}
	got, ok = Parse(raw, time.Time{})
	if !ok || got.Confidence != schema.ConfidenceUnknown || got.RemainingPct != nil {
		t.Fatalf("null window %+v", got)
	}
	if strings.Contains(got.Note, "$12") {
		t.Fatalf("note %q", got.Note)
	}
}

func TestStatusCommand(t *testing.T) {
	payload, err := os.ReadFile("testdata/usage.json")
	if err != nil {
		t.Fatal(err)
	}
	runner := &run.MapRunner{
		Scripts: map[string]run.Script{
			"/bin/hermes usage --json": {Stdout: string(payload)},
		},
	}
	var ad Adapter
	got := ad.Status(context.Background(), agent.Env{Runner: runner, Environ: []string{}}, agent.Detection{
		Installed: true, Binary: "/bin/hermes", Version: "0.9.0",
	})
	if got.RemainingPct == nil || *got.RemainingPct != 63 || got.Version != "0.9.0" {
		t.Fatalf("%+v", got)
	}
	if len(runner.Calls) != 1 || runner.Calls[0] != "/bin/hermes usage --json" {
		t.Fatalf("calls %v", runner.Calls)
	}

	exitErr := exec.Command("go", "this-is-not-a-command").Run()
	fail := &run.MapRunner{Scripts: map[string]run.Script{
		"/bin/hermes usage --json": {Err: exitErr},
	}}
	got = ad.Status(context.Background(), agent.Env{Runner: fail}, agent.Detection{Installed: true, Binary: "/bin/hermes"})
	if got.Confidence != schema.ConfidenceUnknown || !strings.Contains(got.Note, "exited") {
		t.Fatalf("%+v", got)
	}
	got = ad.Status(context.Background(), agent.Env{Runner: &run.MapRunner{}}, agent.Detection{Installed: true})
	if got.Note != "quota command is not installed" {
		t.Fatalf("note %q", got.Note)
	}
}
