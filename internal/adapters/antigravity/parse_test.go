package antigravity

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fxckcode/baton/internal/agent"
	"github.com/fxckcode/baton/internal/run"
	"github.com/fxckcode/baton/internal/schema"
)

func TestParseSyntheticUsage(t *testing.T) {
	data, err := os.ReadFile("testdata/usage.json")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	got, ok := Parse(data, now)
	if !ok || got.Confidence != schema.ConfidenceExact {
		t.Fatalf("%+v", got)
	}
	if got.RemainingPct == nil || *got.RemainingPct != 40 {
		t.Fatalf("remaining %v windows %+v", got.RemainingPct, got.Windows)
	}
	if len(got.Windows) != 2 {
		t.Fatalf("windows %+v", got.Windows)
	}
	if got.Windows[0].Name != "Gemini Models 5h" || *got.Windows[0].RemainingPct != 98 {
		t.Fatalf("5h %+v", got.Windows[0])
	}
	if got.Windows[0].ResetAt == nil || !got.Windows[0].ResetAt.Equal(time.Date(2026, 10, 31, 19, 53, 0, 0, time.UTC)) {
		t.Fatalf("reset %v", got.Windows[0].ResetAt)
	}
}

func TestParseCamelCaseAndErrorStatus(t *testing.T) {
	now := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	payload := []byte(`{"status":"OK","command":{"name":"usage","data":{"groups":[{"name":"Claude","buckets":[{"window":"weekly","remainingFraction":0.5}]}]}}}`)
	got, ok := Parse(payload, now)
	if !ok || got.RemainingPct == nil || *got.RemainingPct != 50 {
		t.Fatalf("%+v", got)
	}
	bad, ok := Parse([]byte(`{"status":"ERROR","error":"nope"}`), now)
	if !ok || bad.Confidence != schema.ConfidenceUnknown || bad.RemainingPct != nil {
		t.Fatalf("error status %+v ok %v", bad, ok)
	}
	if _, ok := Parse([]byte("not-json"), now); ok {
		t.Fatal("non-json")
	}
}

func TestStatusArgv(t *testing.T) {
	data, err := os.ReadFile("testdata/usage.json")
	if err != nil {
		t.Fatal(err)
	}
	runner := &run.MapRunner{Scripts: map[string]run.Script{
		"/bin/agy -p /usage --output-format json": {Stdout: string(data)},
	}}
	var ad Adapter
	got := ad.Status(context.Background(), agent.Env{Runner: runner}, agent.Detection{Installed: true, Binary: "/bin/agy"})
	if got.RemainingPct == nil || *got.RemainingPct != 40 {
		t.Fatalf("%+v", got)
	}
	if len(runner.Calls) != 1 || runner.Calls[0] != "/bin/agy -p /usage --output-format json" {
		t.Fatalf("calls %v", runner.Calls)
	}
}
