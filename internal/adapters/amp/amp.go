package amp

import (
	"context"
	"time"

	"github.com/fxckcode/baton/internal/agent"
	"github.com/fxckcode/baton/internal/detect"
	"github.com/fxckcode/baton/internal/schema"
)

// Adapter runs `amp usage` and parses the text report.
type Adapter struct{}

func (Adapter) ID() string { return "amp" }

func (Adapter) Command() string { return "amp" }

func (Adapter) Detect(ctx context.Context, env agent.Env) agent.Detection {
	spec, _ := detect.ByID("amp")
	return detect.Detect(ctx, spec, env)
}

func (Adapter) Status(ctx context.Context, env agent.Env, det agent.Detection) schema.Agent {
	return agent.ProbeCommand(ctx, env, det, agent.Probe{
		ID:      "amp",
		Source:  sourceTag,
		Timeout: 15 * time.Second,
		Args:    []string{"usage"},
	}, Parse)
}
