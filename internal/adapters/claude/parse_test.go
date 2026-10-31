package claude

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fxckcode/baton/internal/agent"
	"github.com/fxckcode/baton/internal/schema"
)

func TestParseStatusline(t *testing.T) {
	data, err := os.ReadFile("testdata/statusline.json")
	if err != nil {
		t.Fatal(err)
	}
	observed := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	got, ok := Parse(data, observed)
	if !ok {
		t.Fatal("expected windows")
	}
	if got.RemainingPct == nil || *got.RemainingPct != 37.2 {
		t.Fatalf("min remaining %v", got.RemainingPct)
	}
	if len(got.Windows) != 3 || got.Windows[0].Name != "5h" || *got.Windows[0].RemainingPct != 76.5 {
		t.Fatalf("5h %+v", got.Windows)
	}
	if got.Windows[1].Name != "7d" || *got.Windows[1].RemainingPct != 58.8 {
		t.Fatalf("7d %+v", got.Windows[1])
	}
	if got.Windows[2].Name != "spend" || *got.Windows[2].RemainingPct != 37.2 {
		t.Fatalf("spend %+v", got.Windows[2])
	}
	if got.Plan != nil {
		t.Fatal("plan should be unknown")
	}
	if got.ObservedAt == nil || !got.ObservedAt.Equal(observed) {
		t.Fatal(got.ObservedAt)
	}
}

func TestParseWithoutRateLimits(t *testing.T) {
	data, err := os.ReadFile("testdata/no-limits.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := Parse(data, time.Now()); ok {
		t.Fatal("context window usage is not quota")
	}
	if _, ok := Parse([]byte("{"), time.Now()); ok {
		t.Fatal("broken json")
	}
}

func TestStatusReadsSnapshotMtime(t *testing.T) {
	cache := t.TempDir()
	data, err := os.ReadFile("testdata/statusline.json")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cache, "claude-statusline.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	var ad Adapter
	got := ad.Status(context.Background(), agent.Env{CacheDir: cache}, agent.Detection{Installed: true, Version: "2.1.90"})
	if got.Confidence != schema.ConfidenceExact || got.ObservedAt == nil {
		t.Fatalf("%+v", got)
	}
	if !got.ObservedAt.Equal(info.ModTime().UTC()) && !got.ObservedAt.Equal(info.ModTime()) {
		t.Fatalf("observed %s mtime %s", got.ObservedAt, info.ModTime())
	}
	missing := ad.Status(context.Background(), agent.Env{CacheDir: t.TempDir()}, agent.Detection{Installed: true})
	if missing.Confidence != schema.ConfidenceUnknown || missing.RemainingPct != nil {
		t.Fatalf("%+v", missing)
	}
	if missing.Note != "Claude statusline snapshot is missing" {
		t.Fatalf("note %q", missing.Note)
	}
}
