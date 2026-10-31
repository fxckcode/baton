// Package agent defines the adapter surface shared by detection and quota probes.
package agent

import (
	"context"
	"runtime"
	"strings"
	"time"

	"github.com/fxckcode/baton/internal/schema"
)

// Runner executes a local binary. Implementations must not open a TTY.
type Runner interface {
	LookPath(file string) (string, error)
	Run(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error)
}

// Env is the injected filesystem, clock, and process environment.
// A nil Environ reads the real process environment. An empty slice reads nothing.
type Env struct {
	Home     string
	CacheDir string
	Runner   Runner
	Now      time.Time
	Environ  []string
	GOOS     string
}

// OS returns the target operating system, defaulting to the host.
func (e Env) OS() string {
	if e.GOOS != "" {
		return e.GOOS
	}
	return runtime.GOOS
}

// Getenv reads Environ. A nil Environ falls back to the process environment.
func (e Env) Getenv(key string) string {
	if e.Environ == nil {
		return getenvProcess(key)
	}
	prefix := key + "="
	for i := len(e.Environ) - 1; i >= 0; i-- {
		if strings.HasPrefix(e.Environ[i], prefix) {
			return e.Environ[i][len(prefix):]
		}
	}
	return ""
}

// NowOr returns the injected clock, or the real time when unset.
func (e Env) NowOr(fallback time.Time) time.Time {
	if e.Now.IsZero() {
		return fallback
	}
	return e.Now
}

// Detection is what list can say without reading quota.
type Detection struct {
	Installed  bool
	Binary     string
	Version    string
	ConfigPath string
}

// Adapter is one coding-agent CLI.
type Adapter interface {
	ID() string
	Command() string
	Detect(ctx context.Context, env Env) Detection
	Status(ctx context.Context, env Env, det Detection) schema.Agent
}
