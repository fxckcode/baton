// Package run executes agent CLIs with a closed stdin and a hard timeout.
package run

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Runner is the process-backed agent.Runner.
type Runner struct{}

// LookPath resolves file on PATH.
func (Runner) LookPath(file string) (string, error) {
	path, err := exec.LookPath(file)
	if err != nil {
		return "", fmt.Errorf("look up %s: %w", file, err)
	}
	return path, nil
}

// Run executes name with args. Stdin is the null device. Stdout and stderr
// are captured and never logged by this package.
func (Runner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = nil
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	prepare(cmd)
	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("run %s: %w", name, err)
	}
	return stdout.Bytes(), stderr.Bytes(), nil
}
