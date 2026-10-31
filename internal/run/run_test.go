package run

import (
	"bytes"
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunCapturesStdout(t *testing.T) {
	var r Runner
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, _, err := r.Run(ctx, "go", "env", "GOVERSION")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte("go1.")) {
		t.Fatalf("stdout %q", out)
	}
}

func TestRunRespectsDeadline(t *testing.T) {
	var r Runner
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	name, args := slowCommand()
	start := time.Now()
	_, _, err := r.Run(ctx, name, args...)
	if err == nil {
		t.Fatal("expected error")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("deadline took %s", time.Since(start))
	}
}

func TestStdinIsClosed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	var r Runner
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, _, err := r.Run(ctx, "sh", "-c", "read x || echo CLOSED")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "CLOSED") {
		t.Fatalf("stdout %q", out)
	}
}

func slowCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "ping", []string{"-n", "6", "127.0.0.1"}
	}
	return "sleep", []string{"5"}
}
