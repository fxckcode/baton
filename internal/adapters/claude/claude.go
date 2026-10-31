package claude

import (
	"context"
	"os"
	"path/filepath"

	"github.com/fxckcode/baton/internal/agent"
	"github.com/fxckcode/baton/internal/detect"
	"github.com/fxckcode/baton/internal/schema"
)

// SnapshotName is the cache file written by a Claude Code status-line hook.
const SnapshotName = "claude-statusline.json"

// Adapter reads a status-line snapshot. It does not start Claude.
type Adapter struct{}

func (Adapter) ID() string { return "claude" }

func (Adapter) Command() string { return "claude" }

func (Adapter) Detect(ctx context.Context, env agent.Env) agent.Detection {
	spec, _ := detect.ByID("claude")
	return detect.Detect(ctx, spec, env)
}

// SnapshotPath is cacheDir/claude-statusline.json.
func SnapshotPath(cacheDir string) string {
	return filepath.Join(cacheDir, SnapshotName)
}

func (Adapter) Status(_ context.Context, env agent.Env, det agent.Detection) schema.Agent {
	base := func(note string) schema.Agent {
		out := schema.Unknown("claude", det.Installed, sourceTag, note)
		out.Binary = det.Binary
		out.Version = det.Version
		if !det.Installed {
			out.Source = "none"
			out.Note = "not installed"
		}
		return out
	}
	if !det.Installed {
		return base("not installed")
	}
	if env.CacheDir == "" {
		return base("Claude statusline snapshot is missing")
	}
	path := SnapshotPath(env.CacheDir)
	data, err := os.ReadFile(path)
	if err != nil {
		return base("Claude statusline snapshot is missing")
	}
	info, err := os.Stat(path)
	if err != nil {
		return base("Claude statusline snapshot is missing")
	}
	parsed, ok := Parse(data, info.ModTime())
	if !ok {
		out := base("Claude statusline snapshot had no rate-limit windows")
		return out
	}
	parsed.Agent = "claude"
	parsed.Installed = true
	parsed.Binary = det.Binary
	parsed.Version = det.Version
	return parsed
}
