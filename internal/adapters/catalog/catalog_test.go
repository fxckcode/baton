package catalog

import (
	"context"
	"testing"

	"github.com/fxckcode/baton/internal/agent"
	"github.com/fxckcode/baton/internal/detect"
	"github.com/fxckcode/baton/internal/schema"
)

func TestNotInstalled(t *testing.T) {
	spec, ok := detect.ByID("goose")
	if !ok {
		t.Fatal("goose")
	}
	got := New(spec).Status(context.Background(), agent.Env{}, agent.Detection{})
	if got.Installed || got.Confidence != schema.ConfidenceUnknown || got.Note != "not installed" {
		t.Fatalf("%+v", got)
	}
}
