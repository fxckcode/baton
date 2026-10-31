package detect

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/fxckcode/baton/internal/agent"
	"github.com/fxckcode/baton/internal/run"
)

func TestDetectBinaryConfigAndVersion(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	secret := "sk-ant-test-secret"
	if err := os.WriteFile(filepath.Join(home, ".codex", "auth.json"), []byte(secret), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &run.MapRunner{
		Paths: map[string]string{"codex": "/usr/bin/codex"},
		Scripts: map[string]run.Script{
			"/usr/bin/codex --version": {Stdout: "codex-cli 0.80.0\n"},
		},
	}
	spec, ok := ByID("codex")
	if !ok {
		t.Fatal("missing codex spec")
	}
	got := Detect(context.Background(), spec, agent.Env{Home: home, Runner: runner, GOOS: runtime.GOOS, Environ: []string{}})
	if !got.Installed || got.Binary != "/usr/bin/codex" || got.Version != "codex-cli 0.80.0" {
		t.Fatalf("%+v", got)
	}
	if got.ConfigPath == "" {
		t.Fatal("expected config path")
	}
	if containsSecret(got, secret) {
		t.Fatalf("detection leaked secret: %+v", got)
	}

	runner.Scripts["/usr/bin/codex --version"] = run.Script{Err: context.DeadlineExceeded}
	got = Detect(context.Background(), spec, agent.Env{Home: home, Runner: runner, GOOS: runtime.GOOS, Environ: []string{}})
	if !got.Installed || got.Version != "" {
		t.Fatalf("version failure cleared install: %+v", got)
	}
}

func TestDetectConfigOnlyAndMissing(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	spec, _ := ByID("claude")
	got := Detect(context.Background(), spec, agent.Env{
		Home: home, Runner: &run.MapRunner{}, GOOS: "linux", Environ: []string{},
	})
	if !got.Installed || got.Binary != "" {
		t.Fatalf("config only: %+v", got)
	}
	empty := t.TempDir()
	got = Detect(context.Background(), spec, agent.Env{
		Home: empty, Runner: &run.MapRunner{}, GOOS: "linux", Environ: []string{},
	})
	if got.Installed {
		t.Fatalf("unexpected install: %+v", got)
	}
}

func TestConfigDirsOS(t *testing.T) {
	linux := ConfigDirs("opencode", Paths{GOOS: "linux", Home: "/home/me"})
	if !has(linux, "/home/me/.local/share/opencode") {
		t.Fatalf("linux: %v", linux)
	}
	win := ConfigDirs("opencode", Paths{GOOS: "windows", Home: `C:\Users\me`, AppData: `C:\Users\me\AppData\Roaming`})
	if !has(win, filepath.Join(`C:\Users\me\AppData\Roaming`, "opencode")) {
		t.Fatalf("windows: %v", win)
	}
	mac := ConfigDirs("amp", Paths{GOOS: "darwin", Home: "/Users/me"})
	if !has(mac, "/Users/me/Library/Application Support/amp") {
		t.Fatalf("darwin: %v", mac)
	}
	codex := ConfigDirs("codex", Paths{
		GOOS: "linux", Home: "/home/me",
		Getenv: func(k string) string {
			if k == "CODEX_HOME" {
				return "/tmp/codex-home"
			}
			return ""
		},
	})
	if !has(codex, "/tmp/codex-home") {
		t.Fatalf("CODEX_HOME: %v", codex)
	}
}

func TestCatalogCoversDesignIDs(t *testing.T) {
	want := []string{
		"claude", "codex", "antigravity", "hermes", "amp", "kiro",
		"cursor", "gemini", "copilot", "opencode", "pi", "droid", "kimi",
		"qwen", "crush", "goose", "aider", "cline", "warp", "grok", "kilo",
		"zed", "auggie", "devin", "codebuff", "junie", "openclaw",
	}
	specs := Specs()
	if len(specs) != len(want) {
		t.Fatalf("got %d specs", len(specs))
	}
	for i, id := range want {
		if specs[i].ID != id {
			t.Fatalf("order: got %s want %s", specs[i].ID, id)
		}
		if specs[i].Command == "" || len(specs[i].Binaries) == 0 {
			t.Fatalf("%s missing command", id)
		}
	}
}

func TestVersionProbeTimeoutBudget(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	runner := &run.MapRunner{
		Paths: map[string]string{"claude": "/bin/claude"},
		Scripts: map[string]run.Script{
			"/bin/claude --version": {Delay: 5 * time.Second, Stdout: "late"},
		},
	}
	spec, _ := ByID("claude")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	start := time.Now()
	got := Detect(ctx, spec, agent.Env{Home: t.TempDir(), Runner: runner, GOOS: "linux", Environ: []string{}})
	if time.Since(start) > 2500*time.Millisecond {
		t.Fatalf("version probe ignored its budget, took %s", time.Since(start))
	}
	if !got.Installed || got.Version != "" {
		t.Fatalf("%+v", got)
	}
}

func has(paths []string, want string) bool {
	for _, p := range paths {
		if p == want {
			return true
		}
	}
	return false
}

func containsSecret(got agent.Detection, secret string) bool {
	return strings.Contains(got.Binary, secret) || strings.Contains(got.Version, secret) || strings.Contains(got.ConfigPath, secret)
}
