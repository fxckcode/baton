package hermes

import (
	"context"
	"time"

	"github.com/fxckcode/baton/internal/agent"
	"github.com/fxckcode/baton/internal/detect"
	"github.com/fxckcode/baton/internal/schema"
)

// Adapter runs `hermes usage --json` and does not read Hermes credentials.
type Adapter struct{}

func (Adapter) ID() string { return "hermes" }

func (Adapter) Command() string { return "hermes" }

func (Adapter) Detect(ctx context.Context, env agent.Env) agent.Detection {
	spec, _ := detect.ByID("hermes")
	return detect.Detect(ctx, spec, env)
}

func (Adapter) Status(ctx context.Context, env agent.Env, det agent.Detection) schema.Agent {
	return agent.ProbeCommand(ctx, env, det, agent.Probe{
		ID:      "hermes",
		Source:  sourceTag,
		Timeout: 15 * time.Second,
		Args:    []string{"usage", "--json"},
	}, Parse)
}
