package codex

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fxckcode/baton/internal/agent"
	"github.com/fxckcode/baton/internal/schema"
)

func TestParseLatestWindows(t *testing.T) {
	data, err := os.ReadFile("testdata/rollout.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := Parse(data)
	if !ok {
		t.Fatal("expected a snapshot")
	}
	if got.Confidence != schema.ConfidenceExact || got.Source != sourceTag {
		t.Fatalf("%+v", got)
	}
	if got.Plan == nil || *got.Plan != "plus" {
		t.Fatalf("plan %+v", got.Plan)
	}
	if got.RemainingPct == nil || *got.RemainingPct != 19 {
		t.Fatalf("remaining %v", got.RemainingPct)
	}
	if len(got.Windows) != 2 || got.Windows[0].Name != "5h" || *got.Windows[0].RemainingPct != 38 {
		t.Fatalf("windows %+v", got.Windows)
	}
	if got.Windows[1].Name != "weekly" || *got.Windows[1].RemainingPct != 19 {
		t.Fatalf("weekly %+v", got.Windows[1])
	}
	raw, err := schema.Marshal(schema.Report{GeneratedAt: time.Unix(0, 0).UTC(), Agents: []schema.Agent{got}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if strings.Contains(text, "sk-ant-test-secret") || strings.Contains(text, "Bearer test-token") {
		t.Fatalf("leaked transcript: %s", text)
	}
}

func TestParseLegacyReset(t *testing.T) {
	data, err := os.ReadFile("testdata/legacy.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := Parse(data)
	if !ok || got.RemainingPct == nil || *got.RemainingPct != 75 {
		t.Fatalf("%+v", got)
	}
	want := time.Date(2026, 10, 30, 23, 0, 0, 0, time.UTC)
	if got.Windows[0].ResetAt == nil || !got.Windows[0].ResetAt.Equal(want) {
		t.Fatalf("reset %v", got.Windows[0].ResetAt)
	}
}

func TestParseGarbage(t *testing.T) {
	if _, ok := Parse([]byte("{]\nnot json\n")); ok {
		t.Fatal("garbage parsed as quota")
	}
}

func TestStatusSkipsNewestFileWithoutLimits(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "sessions", "2026", "10", "30")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	good, err := os.ReadFile("testdata/rollout.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	older := filepath.Join(dir, "rollout-old.jsonl")
	newer := filepath.Join(dir, "rollout-new.jsonl")
	if err := os.WriteFile(older, good, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newer, []byte("{\"type\":\"response_item\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(older, past, past); err != nil {
		t.Fatal(err)
	}
	var ad Adapter
	got := ad.Status(context.Background(), agent.Env{Home: home, Environ: []string{}}, agent.Detection{
		Installed: true, ConfigPath: home,
	})
	if got.RemainingPct == nil || *got.RemainingPct != 19 {
		t.Fatalf("%+v", got)
	}
}

func TestStatusMissingSessions(t *testing.T) {
	var ad Adapter
	got := ad.Status(context.Background(), agent.Env{Home: t.TempDir(), Environ: []string{}}, agent.Detection{Installed: true})
	if got.Confidence != schema.ConfidenceUnknown || got.RemainingPct != nil {
		t.Fatalf("%+v", got)
	}
	if strings.Contains(got.Note, "sk-") {
		t.Fatal(got.Note)
	}
}
