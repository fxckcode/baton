package registry

import (
	"context"
	"strings"
	"testing"

	"github.com/fxckcode/baton/internal/agent"
	"github.com/fxckcode/baton/internal/detect"
	"github.com/fxckcode/baton/internal/schema"
)

func TestAllMatchesCatalogOnce(t *testing.T) {
	seen := map[string]int{}
	for _, a := range All() {
		seen[a.ID()]++
		if a.Command() == "" {
			t.Fatalf("%s missing command", a.ID())
		}
	}
	if len(seen) != len(detect.Specs()) {
		t.Fatalf("got %d want %d", len(seen), len(detect.Specs()))
	}
	for _, spec := range detect.Specs() {
		if seen[spec.ID] != 1 {
			t.Fatalf("%s count %d", spec.ID, seen[spec.ID])
		}
	}
}

func TestDetectOnlyStaysUnknown(t *testing.T) {
	var cursor agent.Adapter
	for _, a := range All() {
		if a.ID() == "cursor" {
			cursor = a
		}
	}
	if cursor == nil {
		t.Fatal("missing cursor")
	}
	got := cursor.Status(context.Background(), agent.Env{}, agent.Detection{Installed: true, Version: "1.0.0"})
	if got.Confidence != schema.ConfidenceUnknown || got.RemainingPct != nil || got.Version != "1.0.0" {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got.Note, "v0.1") {
		t.Fatalf("note %q", got.Note)
	}
	for _, id := range []string{"claude", "codex", "antigravity", "hermes", "amp", "kiro"} {
		found := false
		for _, a := range All() {
			if a.ID() == id {
				found = true
			}
		}
		if !found {
			t.Fatal(id)
		}
	}
}
