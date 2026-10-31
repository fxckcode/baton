package codex

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fxckcode/baton/internal/agent"
	"github.com/fxckcode/baton/internal/detect"
	"github.com/fxckcode/baton/internal/schema"
)

const (
	maxFullRead = 4 << 20
	tailRead    = 1 << 20
	maxFiles    = 8
)

// Adapter reads Codex rollout files. It does not open auth.json.
type Adapter struct{}

func (Adapter) ID() string { return "codex" }

func (Adapter) Command() string { return "codex" }

func (Adapter) Detect(ctx context.Context, env agent.Env) agent.Detection {
	spec, _ := detect.ByID("codex")
	return detect.Detect(ctx, spec, env)
}

func (Adapter) Status(_ context.Context, env agent.Env, det agent.Detection) schema.Agent {
	if !det.Installed {
		return schema.Unknown("codex", false, "none", "not installed")
	}
	root := det.ConfigPath
	if root == "" {
		if v := env.Getenv("CODEX_HOME"); v != "" {
			root = v
		} else {
			root = filepath.Join(env.Home, ".codex")
		}
	}
	files, err := newestRollouts(filepath.Join(root, "sessions"), maxFiles)
	if err != nil || len(files) == 0 {
		note := "no Codex rollouts"
		if err != nil {
			note = "no Codex session directory"
		}
		out := schema.Unknown("codex", true, sourceTag, note)
		out.Binary = det.Binary
		out.Version = det.Version
		return out
	}
	for _, path := range files {
		data, err := readTail(path)
		if err != nil {
			continue
		}
		parsed, ok := Parse(data)
		if !ok {
			continue
		}
		parsed.Agent = "codex"
		parsed.Installed = true
		parsed.Binary = det.Binary
		parsed.Version = det.Version
		parsed.ConfigPath = det.ConfigPath
		return parsed
	}
	out := schema.Unknown("codex", true, sourceTag, "no Codex rate limit in recent rollouts")
	out.Binary = det.Binary
	out.Version = det.Version
	return out
}

func newestRollouts(dir string, n int) ([]string, error) {
	entries := make([]struct {
		path string
		mod  int64
	}, 0)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == dir {
				return walkErr
			}
			return nil
		}
		if d == nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasPrefix(name, "rollout-") || !strings.HasSuffix(name, ".jsonl") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		entries = append(entries, struct {
			path string
			mod  int64
		}{path, info.ModTime().UnixNano()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].mod > entries[j].mod })
	if len(entries) > n {
		entries = entries[:n]
	}
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.path
	}
	return out, nil
}

func readTail(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() <= maxFullRead {
		return io.ReadAll(f)
	}
	if _, err := f.Seek(-tailRead, io.SeekEnd); err != nil {
		return nil, err
	}
	buf, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	if i := bytes.IndexByte(buf, '\n'); i >= 0 && i < len(buf)-1 {
		buf = buf[i+1:]
	}
	return buf, nil
}
