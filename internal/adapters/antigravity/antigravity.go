package antigravity

import (
	"context"
	"time"

	"github.com/fxckcode/baton/internal/agent"
	"github.com/fxckcode/baton/internal/detect"
	"github.com/fxckcode/baton/internal/schema"
)

// Adapter runs the read-only Antigravity usage prompt.
type Adapter struct{}

func (Adapter) ID() string { return "antigravity" }

func (Adapter) Command() string { return "agy" }

func (Adapter) Detect(ctx context.Context, env agent.Env) agent.Detection {
	spec, _ := detect.ByID("antigravity")
	return detect.Detect(ctx, spec, env)
}

func (Adapter) Status(ctx context.Context, env agent.Env, det agent.Detection) schema.Agent {
	return agent.ProbeCommand(ctx, env, det, agent.Probe{
		ID:      "antigravity",
		Source:  sourceTag,
		Timeout: 15 * time.Second,
		Args:    []string{"-p", "/usage", "--output-format", "json"},
	}, Parse)
}
