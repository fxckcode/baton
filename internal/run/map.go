package run

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Script is a canned command result used by tests.
type Script struct {
	Stdout string
	Stderr string
	Err    error
	Delay  time.Duration
}

// MapRunner is a Runner that never starts a real process.
type MapRunner struct {
	mu      sync.Mutex
	Paths   map[string]string
	Scripts map[string]Script
	Calls   []string
}

// LookPath returns Paths[file].
func (m *MapRunner) LookPath(file string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	path, ok := m.Paths[file]
	if !ok || path == "" {
		return "", fmt.Errorf("look up %s: %w", file, exec.ErrNotFound)
	}
	return path, nil
}

// Run returns the script keyed by name and args. Delay waits unless ctx ends.
func (m *MapRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	key := name
	if len(args) > 0 {
		key += " " + strings.Join(args, " ")
	}
	m.mu.Lock()
	m.Calls = append(m.Calls, key)
	script := m.Scripts[key]
	m.mu.Unlock()
	if script.Delay > 0 {
		timer := time.NewTimer(script.Delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-timer.C:
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if script.Err != nil {
		return []byte(script.Stdout), []byte(script.Stderr), script.Err
	}
	return []byte(script.Stdout), []byte(script.Stderr), nil
}
