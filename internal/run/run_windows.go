//go:build windows

package run

import (
	"os/exec"
	"time"
)

func prepare(cmd *exec.Cmd) {
	cmd.WaitDelay = 2 * time.Second
}
