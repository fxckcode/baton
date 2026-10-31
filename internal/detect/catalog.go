package detect

import "path/filepath"

// Spec describes how to notice one agent without reading quota.
type Spec struct {
	ID       string
	Command  string
	Binaries []string
	dirs     func(Paths) []string
}

// Paths is the home layout used to build config candidates.
type Paths struct {
	GOOS    string
	Home    string
	AppData string
	Getenv  func(string) string
}

func (p Paths) get(key string) string {
	if p.Getenv == nil {
		return ""
	}
	return p.Getenv(key)
}

// Specs returns the catalog in stable list order.
func Specs() []Spec {
	return []Spec{
		{ID: "claude", Command: "claude", Binaries: []string{"claude"}, dirs: claudeDirs},
		{ID: "codex", Command: "codex", Binaries: []string{"codex"}, dirs: codexDirs},
		{ID: "antigravity", Command: "agy", Binaries: []string{"agy"}, dirs: func(p Paths) []string {
			return []string{filepath.Join(p.Home, ".gemini", "antigravity-cli")}
		}},
		{ID: "hermes", Command: "hermes", Binaries: []string{"hermes"}, dirs: hermesDirs},
		{ID: "amp", Command: "amp", Binaries: []string{"amp"}, dirs: func(p Paths) []string { return shareDirs(p, "amp") }},
		{ID: "kiro", Command: "kiro-cli", Binaries: []string{"kiro-cli"}, dirs: func(p Paths) []string {
			return []string{filepath.Join(p.Home, ".kiro")}
		}},
		{ID: "cursor", Command: "agent", Binaries: []string{"agent", "cursor-agent"}, dirs: func(p Paths) []string {
			return []string{filepath.Join(p.Home, ".cursor")}
		}},
		{ID: "gemini", Command: "gemini", Binaries: []string{"gemini"}, dirs: geminiDirs},
		{ID: "copilot", Command: "copilot", Binaries: []string{"copilot"}, dirs: func(p Paths) []string {
			return []string{filepath.Join(p.Home, ".copilot")}
		}},
		{ID: "opencode", Command: "opencode", Binaries: []string{"opencode"}, dirs: func(p Paths) []string { return shareDirs(p, "opencode") }},
		{ID: "pi", Command: "pi", Binaries: []string{"pi"}, dirs: func(p Paths) []string {
			return []string{filepath.Join(p.Home, ".pi", "agent")}
		}},
		{ID: "droid", Command: "droid", Binaries: []string{"droid"}, dirs: func(p Paths) []string {
			return []string{filepath.Join(p.Home, ".factory")}
		}},
		{ID: "kimi", Command: "kimi", Binaries: []string{"kimi"}, dirs: func(p Paths) []string {
			return []string{filepath.Join(p.Home, ".kimi"), filepath.Join(p.Home, ".kimi-code")}
		}},
		{ID: "qwen", Command: "qwen", Binaries: []string{"qwen"}, dirs: func(p Paths) []string {
			return []string{filepath.Join(p.Home, ".qwen")}
		}},
		{ID: "crush", Command: "crush", Binaries: []string{"crush"}, dirs: func(p Paths) []string { return shareDirs(p, "crush") }},
		{ID: "goose", Command: "goose", Binaries: []string{"goose"}, dirs: gooseDirs},
		{ID: "aider", Command: "aider", Binaries: []string{"aider"}, dirs: func(p Paths) []string {
			return []string{filepath.Join(p.Home, ".aider.conf.yml")}
		}},
		{ID: "cline", Command: "cline", Binaries: []string{"cline"}, dirs: func(p Paths) []string {
			return []string{filepath.Join(p.Home, ".cline")}
		}},
		{ID: "warp", Command: "warp", Binaries: []string{"warp", "oz"}},
		{ID: "grok", Command: "grok", Binaries: []string{"grok"}, dirs: func(p Paths) []string {
			return []string{filepath.Join(p.Home, ".grok")}
		}},
		{ID: "kilo", Command: "kilo", Binaries: []string{"kilo"}, dirs: func(p Paths) []string { return shareDirs(p, "kilo") }},
		{ID: "zed", Command: "zed", Binaries: []string{"zed"}},
		{ID: "auggie", Command: "auggie", Binaries: []string{"auggie"}},
		{ID: "devin", Command: "devin", Binaries: []string{"devin"}},
		{ID: "codebuff", Command: "codebuff", Binaries: []string{"codebuff"}},
		{ID: "junie", Command: "junie", Binaries: []string{"junie"}},
		{ID: "openclaw", Command: "openclaw", Binaries: []string{"openclaw"}},
	}
}

// ByID returns the spec for id.
func ByID(id string) (Spec, bool) {
	for _, spec := range Specs() {
		if spec.ID == id {
			return spec, true
		}
	}
	return Spec{}, false
}

// ConfigDirs returns candidate config paths for id on p.
func ConfigDirs(id string, p Paths) []string {
	spec, ok := ByID(id)
	if !ok || spec.dirs == nil {
		return nil
	}
	return spec.dirs(p)
}

func claudeDirs(p Paths) []string {
	return envThen(p, "CLAUDE_CONFIG_DIR", filepath.Join(p.Home, ".claude"))
}

func codexDirs(p Paths) []string {
	return envThen(p, "CODEX_HOME", filepath.Join(p.Home, ".codex"))
}

func hermesDirs(p Paths) []string {
	return envThen(p, "HERMES_HOME", filepath.Join(p.Home, ".hermes"))
}

func geminiDirs(p Paths) []string {
	return envThen(p, "GEMINI_CLI_HOME", filepath.Join(p.Home, ".gemini"))
}

func envThen(p Paths, key, fallback string) []string {
	if v := p.get(key); v != "" {
		return []string{v, fallback}
	}
	return []string{fallback}
}

func shareDirs(p Paths, name string) []string {
	primary := filepath.Join(shareRoot(p), name)
	legacy := filepath.Join(p.Home, ".local", "share", name)
	if primary == legacy {
		return []string{primary}
	}
	return []string{primary, legacy}
}

func shareRoot(p Paths) string {
	switch p.GOOS {
	case "windows":
		if p.AppData != "" {
			return p.AppData
		}
		return filepath.Join(p.Home, "AppData", "Roaming")
	case "darwin":
		return filepath.Join(p.Home, "Library", "Application Support")
	default:
		if v := p.get("XDG_DATA_HOME"); v != "" {
			return v
		}
		return filepath.Join(p.Home, ".local", "share")
	}
}

func gooseDirs(p Paths) []string {
	unix := filepath.Join(p.Home, ".config", "goose")
	if p.GOOS == "windows" {
		return []string{filepath.Join(shareRoot(p), "goose"), unix}
	}
	return []string{unix}
}
