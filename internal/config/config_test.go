package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadMissingUsesDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("BATON_CONFIG", "")
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Threshold != 15 || cfg.MaxAge != 6*time.Hour || !cfg.AllowStale || cfg.AllowEstimated {
		t.Fatalf("%+v", cfg)
	}
	want := []string{"antigravity", "cursor", "claude", "codex", "hermes", "amp", "kiro"}
	if len(cfg.Chain) != len(want) {
		t.Fatalf("chain %v", cfg.Chain)
	}
	for i := range want {
		if cfg.Chain[i] != want[i] {
			t.Fatalf("chain %v", cfg.Chain)
		}
	}
	cfg.Chain[0] = "mutated"
	again, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if again.Chain[0] != "antigravity" {
		t.Fatal("default chain was mutated")
	}
}

func TestLoadFileAndEnvOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	body := `
chain = ["claude", "codex"]
threshold = 20
max_age = "90m"
allow_estimated = true
allow_stale = false
future_key = "ignored"
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BATON_CONFIG", filepath.Join(dir, "nope.toml"))
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Threshold != 20 || cfg.MaxAge != 90*time.Minute || !cfg.AllowEstimated || cfg.AllowStale {
		t.Fatalf("%+v", cfg)
	}
	if len(cfg.Chain) != 2 || cfg.Chain[0] != "claude" || cfg.Chain[1] != "codex" {
		t.Fatalf("chain %v", cfg.Chain)
	}

	t.Setenv("BATON_CONFIG", path)
	fromEnv, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if fromEnv.Threshold != 20 {
		t.Fatalf("env override: %+v", fromEnv)
	}
}

func TestLoadInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	if err := os.WriteFile(path, []byte("chain = ["), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if _, err := Load(filepath.Join(dir, "missing.toml")); err == nil {
		t.Fatal("explicit missing path should fail")
	}
	path = filepath.Join(dir, "age.toml")
	if err := os.WriteFile(path, []byte(`max_age = "soon"`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected duration error")
	}
}

func TestLoadEmptyChainIsExplicit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("chain = []\nthreshold = 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Chain) != 0 || cfg.Threshold != 0 {
		t.Fatalf("%+v", cfg)
	}
}
