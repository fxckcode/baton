// Package catalog is the detection-only adapter used for agents with no
// verified quota source in v0.1.
package catalog

import (
	"context"

	"github.com/fxckcode/baton/internal/agent"
	"github.com/fxckcode/baton/internal/detect"
	"github.com/fxckcode/baton/internal/schema"
)

// UnknownNote explains why a detected agent has no remaining percent.
const UnknownNote = "quota is not read in v0.1 because the only known sources are interactive UI or non-public APIs"

// Adapter detects one agent and never invents a quota.
type Adapter struct {
	Spec detect.Spec
}

// New returns a detection-only adapter.
func New(spec detect.Spec) Adapter { return Adapter{Spec: spec} }

func (a Adapter) ID() string { return a.Spec.ID }

func (a Adapter) Command() string { return a.Spec.Command }

func (a Adapter) Detect(ctx context.Context, env agent.Env) agent.Detection {
	return detect.Detect(ctx, a.Spec, env)
}

func (a Adapter) Status(_ context.Context, _ agent.Env, det agent.Detection) schema.Agent {
	note := UnknownNote
	source := "none"
	if !det.Installed {
		note = "not installed"
	}
	out := schema.Unknown(a.Spec.ID, det.Installed, source, note)
	out.Binary = det.Binary
	out.Version = det.Version
	out.ConfigPath = det.ConfigPath
	return out
}
