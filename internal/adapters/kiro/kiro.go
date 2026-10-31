package kiro

import (
	"context"
	"time"

	"github.com/fxckcode/baton/internal/agent"
	"github.com/fxckcode/baton/internal/detect"
	"github.com/fxckcode/baton/internal/schema"
)

// Adapter runs `kiro-cli chat --no-interactive "/usage"`.
// That command may leave a session file; Baton does not delete it.
type Adapter struct{}

func (Adapter) ID() string { return "kiro" }

func (Adapter) Command() string { return "kiro-cli" }

func (Adapter) Detect(ctx context.Context, env agent.Env) agent.Detection {
	spec, _ := detect.ByID("kiro")
	return detect.Detect(ctx, spec, env)
}

func (Adapter) Status(ctx context.Context, env agent.Env, det agent.Detection) schema.Agent {
	return agent.ProbeCommand(ctx, env, det, agent.Probe{
		ID:      "kiro",
		Source:  sourceTag,
		Timeout: 20 * time.Second,
		Args:    []string{"chat", "--no-interactive", "/usage"},
	}, Parse)
}
