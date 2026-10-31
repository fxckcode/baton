package detect

import (
	"bytes"
	"context"
	"os"
	"strings"
	"time"

	"github.com/fxckcode/baton/internal/agent"
)

const versionBudget = 2 * time.Second

// Detect reports whether spec's binary or config path is present.
// A failed version probe does not clear Installed. Credential files are not opened.
func Detect(ctx context.Context, spec Spec, env agent.Env) agent.Detection {
	var det agent.Detection
	if env.Runner != nil {
		for _, name := range spec.Binaries {
			path, err := env.Runner.LookPath(name)
			if err != nil || path == "" {
				continue
			}
			det.Binary = path
			det.Installed = true
			break
		}
	}
	if spec.dirs != nil {
		det.ConfigPath = firstExisting(spec.dirs(pathsOf(env)))
		if det.ConfigPath != "" {
			det.Installed = true
		}
	}
	if det.Binary != "" && env.Runner != nil {
		det.Version = probeVersion(ctx, env.Runner, det.Binary)
	}
	return det
}

func pathsOf(env agent.Env) Paths {
	return Paths{
		GOOS:    env.OS(),
		Home:    env.Home,
		AppData: env.Getenv("APPDATA"),
		Getenv:  env.Getenv,
	}
}

func firstExisting(paths []string) string {
	for _, path := range paths {
		if path == "" {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.IsDir() || info.Mode().IsRegular() {
			return path
		}
	}
	return ""
}

func probeVersion(ctx context.Context, runner agent.Runner, binary string) string {
	ctx, cancel := context.WithTimeout(ctx, versionBudget)
	defer cancel()
	out, _, err := runner.Run(ctx, binary, "--version")
	if err != nil && len(bytes.TrimSpace(out)) == 0 {
		return ""
	}
	return shortLine(out)
}

func shortLine(out []byte) string {
	line := strings.TrimSpace(string(out))
	if i := strings.IndexAny(line, "\r\n"); i >= 0 {
		line = strings.TrimSpace(line[:i])
	}
	if len(line) > 80 {
		line = line[:80]
	}
	return line
}
