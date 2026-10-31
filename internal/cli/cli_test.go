package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fxckcode/baton/internal/run"
	"github.com/fxckcode/baton/internal/schema"
	"github.com/fxckcode/baton/internal/skill"
)

func TestVersionAndSkill(t *testing.T) {
	out, errBuf := &bytes.Buffer{}, &bytes.Buffer{}
	code := Run(Options{Args: []string{"version"}, Stdout: out, Stderr: errBuf, Version: "0.1.0", Environ: []string{}})
	if code != 0 || out.String() != "0.1.0\n" {
		t.Fatalf("version code %d out %q err %q", code, out, errBuf)
	}
	out.Reset()
	code = Run(Options{Args: []string{"--version"}, Stdout: out, Stderr: errBuf, Version: "0.1.0", Environ: []string{}})
	if code != 0 || out.String() != "0.1.0\n" {
		t.Fatalf("--version %d %q", code, out)
	}
	out.Reset()
	errBuf.Reset()
	code = Run(Options{Args: []string{"--skill"}, Stdout: out, Stderr: errBuf, Environ: []string{}})
	if code != 0 || out.String() != skill.Text || errBuf.Len() != 0 {
		t.Fatalf("skill code %d match %v err %q", code, out.String() == skill.Text, errBuf)
	}
	out.Reset()
	code = Run(Options{Args: []string{"skill"}, Stdout: out, Stderr: errBuf, Environ: []string{}})
	if code != 0 || out.String() != skill.Text {
		t.Fatal("skill subcommand")
	}
}

func TestUsageAndUnknownAgent(t *testing.T) {
	out, errBuf := &bytes.Buffer{}, &bytes.Buffer{}
	if code := Run(Options{Args: []string{"nope"}, Stdout: out, Stderr: errBuf, Environ: []string{}}); code != 2 {
		t.Fatalf("usage code %d", code)
	}
	errBuf.Reset()
	if code := Run(Options{Args: []string{"status", "--agent", "nope"}, Stdout: out, Stderr: errBuf, Environ: []string{}, Home: t.TempDir()}); code != 1 {
		t.Fatalf("unknown agent code %d stderr %s", code, errBuf)
	}
	if !strings.Contains(errBuf.String(), "nope") {
		t.Fatalf("stderr %s", errBuf)
	}
}

func TestListStatusNextAndSecrets(t *testing.T) {
	home := t.TempDir()
	cache := t.TempDir()
	secret := "sk-ant-test-secret"
	token := "Bearer test-token"
	dir := filepath.Join(home, ".codex", "sessions", "2026", "10", "31")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	rollout := `{"timestamp":"2026-10-31T00:00:00Z","type":"response_item","payload":{"type":"message","text":"` + secret + ` ` + token + `"}}
{"timestamp":"2026-10-31T00:00:00Z","type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":10,"window_minutes":300,"resets_at":1790000000},"plan_type":"plus"}}}
`
	if err := os.WriteFile(filepath.Join(dir, "rollout-test.jsonl"), []byte(rollout), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(cfg, []byte("chain = [\"codex\"]\nthreshold = 15\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := &run.MapRunner{
		Paths: map[string]string{"codex": "/bin/codex"},
		Scripts: map[string]run.Script{
			"/bin/codex --version": {Stdout: "codex 1.2.3\n"},
		},
	}
	base := Options{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		Home:       home,
		CacheDir:   cache,
		ConfigPath: cfg,
		Runner:     runner,
		Version:    "0.1.0",
		Environ:    []string{},
		Now:        time.Date(2026, 10, 31, 1, 0, 0, 0, time.UTC),
	}

	base.Args = []string{"list", "--json"}
	base.Stdout = &bytes.Buffer{}
	base.Stderr = &bytes.Buffer{}
	if code := Run(base); code != 0 {
		t.Fatalf("list code %d %s", code, base.Stderr)
	}
	report := decode(t, base.Stdout.(*bytes.Buffer).Bytes())
	codex := findAgent(t, report, "codex")
	if !codex.Installed || codex.Version != "codex 1.2.3" || codex.RemainingPct != nil {
		t.Fatalf("list codex %+v", codex)
	}
	if report.Schema != schema.Version {
		t.Fatal(report.Schema)
	}

	base.Args = []string{"status", "--json", "--agent", "codex"}
	base.Stdout = &bytes.Buffer{}
	base.Stderr = &bytes.Buffer{}
	if code := Run(base); code != 0 {
		t.Fatalf("status code %d %s", code, base.Stderr)
	}
	statusOut := base.Stdout.(*bytes.Buffer).String()
	report = decode(t, []byte(statusOut))
	if len(report.Agents) != 1 {
		t.Fatalf("agents %d", len(report.Agents))
	}
	codex = report.Agents[0]
	if codex.Confidence != schema.ConfidenceExact || codex.RemainingPct == nil || *codex.RemainingPct != 90 {
		t.Fatalf("status %+v", codex)
	}
	if strings.Contains(statusOut, secret) || strings.Contains(statusOut, token) {
		t.Fatalf("json leaked\n%s", statusOut)
	}

	base.Args = []string{"status", "--agent", "codex"}
	base.Stdout = &bytes.Buffer{}
	base.Stderr = &bytes.Buffer{}
	if code := Run(base); code != 0 {
		t.Fatalf("human status %d", code)
	}
	human := base.Stdout.(*bytes.Buffer).String()
	if strings.Contains(human, secret) || strings.Contains(human, token) || !strings.Contains(human, "90") {
		t.Fatalf("human %s", human)
	}

	base.Args = []string{"next", "--json"}
	base.Stdout = &bytes.Buffer{}
	base.Stderr = &bytes.Buffer{}
	if code := Run(base); code != 0 {
		t.Fatalf("next code %d %s", code, base.Stderr)
	}
	report = decode(t, base.Stdout.(*bytes.Buffer).Bytes())
	if report.Recommendation == nil || report.Recommendation.Agent != "codex" || report.Recommendation.Command != "codex" {
		t.Fatalf("recommendation %+v", report.Recommendation)
	}

	base.Args = []string{"next", "--json", "--threshold", "95"}
	base.Stdout = &bytes.Buffer{}
	base.Stderr = &bytes.Buffer{}
	if code := Run(base); code != 3 {
		t.Fatalf("exhausted code %d", code)
	}
	report = decode(t, base.Stdout.(*bytes.Buffer).Bytes())
	if report.Recommendation != nil || report.Schema != schema.Version {
		t.Fatalf("exhausted report %+v", report.Recommendation)
	}

	base.Args = []string{"next", "--threshold", "95"}
	base.Stdout = &bytes.Buffer{}
	base.Stderr = &bytes.Buffer{}
	if code := Run(base); code != 3 || base.Stdout.(*bytes.Buffer).Len() != 0 {
		t.Fatalf("human next stdout %q code %d", base.Stdout, code)
	}
	if !strings.Contains(base.Stderr.(*bytes.Buffer).String(), "threshold") {
		t.Fatalf("stderr %s", base.Stderr)
	}

	base.Args = []string{"doctor"}
	base.Stdout = &bytes.Buffer{}
	base.Stderr = &bytes.Buffer{}
	if code := Run(base); code != 0 {
		t.Fatalf("doctor %d", code)
	}
	doc := base.Stdout.(*bytes.Buffer).String()
	if strings.Contains(doc, secret) || strings.Contains(doc, token) {
		t.Fatal("doctor leaked")
	}
	if !strings.Contains(doc, "statusline snapshot") || !strings.Contains(doc, "session file") {
		t.Fatalf("doctor missing guidance:\n%s", doc)
	}
}

func TestBadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.toml")
	if err := os.WriteFile(path, []byte("chain = ["), 0o644); err != nil {
		t.Fatal(err)
	}
	errBuf := &bytes.Buffer{}
	code := Run(Options{
		Args: []string{"status"}, Stdout: &bytes.Buffer{}, Stderr: errBuf,
		ConfigPath: path, Environ: []string{}, Home: t.TempDir(),
	})
	if code != 1 || !strings.Contains(errBuf.String(), "config") {
		t.Fatalf("code %d err %s", code, errBuf)
	}
}

func decode(t *testing.T, raw []byte) schema.Report {
	t.Helper()
	var report schema.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("json: %v\n%s", err, raw)
	}
	return report
}

func findAgent(t *testing.T, report schema.Report, id string) schema.Agent {
	t.Helper()
	for _, a := range report.Agents {
		if a.Agent == id {
			return a
		}
	}
	t.Fatalf("missing %s", id)
	return schema.Agent{}
}
