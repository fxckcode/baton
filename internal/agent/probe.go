package agent

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/fxckcode/baton/internal/schema"
)

// Probe is the identity of one quota command.
type Probe struct {
	ID      string
	Source  string
	Timeout time.Duration
	Args    []string
}

// ProbeCommand runs a quota command and parses stdout.
// Stderr is discarded. A recognized payload wins over a non-zero exit.
func ProbeCommand(ctx context.Context, env Env, det Detection, probe Probe, parse func([]byte, time.Time) (schema.Agent, bool)) schema.Agent {
	if !det.Installed {
		return schema.Unknown(probe.ID, false, "none", "not installed")
	}
	if det.Binary == "" || env.Runner == nil {
		out := schema.Unknown(probe.ID, true, probe.Source, "quota command is not installed")
		out.Version = det.Version
		return out
	}
	timeout := probe.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	stdout, _, err := env.Runner.Run(ctx, det.Binary, probe.Args...)
	now := env.Now
	if now.IsZero() {
		now = time.Now()
	}
	if parsed, ok := parse(stdout, now); ok {
		return decorate(parsed, det, probe)
	}
	note := "output was not a usage payload"
	if err != nil {
		note = commandNote(err)
	}
	out := schema.Unknown(probe.ID, true, probe.Source, note)
	out.Binary = det.Binary
	out.Version = det.Version
	return out
}

func decorate(parsed schema.Agent, det Detection, probe Probe) schema.Agent {
	parsed.Agent = probe.ID
	parsed.Installed = true
	if parsed.Binary == "" {
		parsed.Binary = det.Binary
	}
	if parsed.Version == "" {
		parsed.Version = det.Version
	}
	if parsed.Source == "" {
		parsed.Source = probe.Source
	}
	if parsed.Windows == nil {
		parsed.Windows = []schema.Window{}
	}
	if parsed.Confidence == "" {
		parsed.Confidence = schema.ConfidenceExact
	}
	return parsed
}

func commandNote(err error) string {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "command timed out"
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return fmt.Sprintf("command exited %d", exitErr.ExitCode())
	}
	return "command failed"
}
