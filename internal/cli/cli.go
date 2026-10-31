// Package cli is the baton command line.
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/fxckcode/baton/internal/agent"
	"github.com/fxckcode/baton/internal/config"
	"github.com/fxckcode/baton/internal/next"
	"github.com/fxckcode/baton/internal/registry"
	"github.com/fxckcode/baton/internal/run"
	"github.com/fxckcode/baton/internal/schema"
	"github.com/fxckcode/baton/internal/skill"
)

// Version is the build version. Release builds may override it with -X.
var Version = "0.1.0"

// Options is the process environment for one invocation.
// A nil Environ uses the real process environment. An empty slice uses none.
type Options struct {
	Args       []string
	Stdout     io.Writer
	Stderr     io.Writer
	Now        time.Time
	Home       string
	CacheDir   string
	ConfigPath string
	Runner     agent.Runner
	Version    string
	Environ    []string
	Adapters   []agent.Adapter
}

const usageText = `baton is the quota handover for coding-agent CLIs.

Usage:
  baton list [--json]
  baton status [--json] [--agent ID]
  baton next [--json] [--threshold N]
  baton doctor
  baton skill
  baton version

  baton --skill    print agent instructions
  baton --version

Exit status 3 from next means no agent is at or above the threshold.
`

// Run executes one invocation and returns the process exit code.
func Run(opts Options) int {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.Version == "" {
		opts.Version = Version
	}
	if len(opts.Args) == 0 || opts.Args[0] == "help" || opts.Args[0] == "--help" || opts.Args[0] == "-h" {
		fmt.Fprint(opts.Stdout, usageText)
		return 0
	}
	switch opts.Args[0] {
	case "version", "--version", "-v":
		fmt.Fprintln(opts.Stdout, opts.Version)
		return 0
	case "skill", "--skill":
		fmt.Fprint(opts.Stdout, skill.Text)
		return 0
	case "list", "status", "next", "doctor":
		return runChecked(opts, opts.Args[0], opts.Args[1:])
	default:
		fmt.Fprintf(opts.Stderr, "unknown command %q\n%s", opts.Args[0], usageText)
		return 2
	}
}

type runtime struct {
	stdout   io.Writer
	stderr   io.Writer
	now      time.Time
	cfg      config.Config
	env      agent.Env
	adapters []agent.Adapter
}

func runChecked(opts Options, cmd string, args []string) int {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		fmt.Fprintf(opts.Stderr, "config: %v\n", err)
		return 1
	}
	rt, err := newRuntime(opts, cfg)
	if err != nil {
		fmt.Fprintf(opts.Stderr, "baton: %v\n", err)
		return 1
	}
	switch cmd {
	case "list":
		return runList(rt, args)
	case "status":
		return runStatus(rt, args)
	case "next":
		return runNext(rt, args)
	case "doctor":
		return runDoctor(rt, args)
	default:
		return 2
	}
}

func newRuntime(opts Options, cfg config.Config) (runtime, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	home := opts.Home
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return runtime{}, fmt.Errorf("home directory: %w", err)
		}
	}
	environ := opts.Environ
	if opts.Environ == nil {
		environ = os.Environ()
	}
	cache := opts.CacheDir
	if cache == "" {
		cache = cacheDir(environ)
	}
	runner := opts.Runner
	if runner == nil {
		runner = run.Runner{}
	}
	adapters := opts.Adapters
	if adapters == nil {
		adapters = registry.All()
	}
	return runtime{
		stdout: opts.Stdout,
		stderr: opts.Stderr,
		now:    now,
		cfg:    cfg,
		env: agent.Env{
			Home:     home,
			CacheDir: cache,
			Runner:   runner,
			Now:      now,
			Environ:  environ,
		},
		adapters: adapters,
	}, nil
}

func cacheDir(environ []string) string {
	for _, item := range environ {
		const key = "BATON_CACHE_DIR="
		if len(item) > len(key) && item[:len(key)] == key {
			return item[len(key):]
		}
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "baton")
}

func runList(rt runtime, args []string) int {
	asJSON, code, ok := parseFlags(rt, "list", args, nil)
	if !ok {
		return code
	}
	agents := make([]schema.Agent, 0, len(rt.adapters))
	for _, a := range rt.adapters {
		det := a.Detect(context.Background(), rt.env)
		agents = append(agents, detectionRow(a.ID(), det))
	}
	report := schema.Report{Schema: schema.Version, GeneratedAt: rt.now.UTC(), Agents: agents}
	if asJSON {
		return writeJSON(rt, report, 0)
	}
	writeList(rt.stdout, agents)
	return 0
}

func runStatus(rt runtime, args []string) int {
	var agentID string
	asJSON, code, ok := parseFlags(rt, "status", args, func(fs *flag.FlagSet) {
		fs.StringVar(&agentID, "agent", "", "limit the report to one agent id")
	})
	if !ok {
		return code
	}
	selected, errCode := rt.selectAdapters(agentID)
	if errCode != 0 {
		return errCode
	}
	agents := rt.probe(selected)
	report := schema.Report{Schema: schema.Version, GeneratedAt: rt.now.UTC(), Agents: agents}
	if asJSON {
		return writeJSON(rt, report, 0)
	}
	writeStatus(rt.stdout, agents)
	return 0
}

func runNext(rt runtime, args []string) int {
	threshold := rt.cfg.Threshold
	asJSON, code, ok := parseFlags(rt, "next", args, func(fs *flag.FlagSet) {
		fs.Float64Var(&threshold, "threshold", rt.cfg.Threshold, "minimum remaining percent")
	})
	if !ok {
		return code
	}
	agents := rt.probe(rt.chainAdapters())
	decision := next.Select(agents, next.Policy{
		Chain:          rt.cfg.Chain,
		Threshold:      threshold,
		AllowEstimated: rt.cfg.AllowEstimated,
		AllowStale:     rt.cfg.AllowStale,
	})
	report := schema.Report{Schema: schema.Version, GeneratedAt: rt.now.UTC(), Agents: agents}
	if decision.OK {
		report.Recommendation = &schema.Recommendation{
			Agent:   decision.Agent.Agent,
			Command: rt.command(decision.Agent.Agent),
			Reason:  decision.Reason,
		}
	}
	exit := 0
	if !decision.OK {
		exit = 3
	}
	if asJSON {
		return writeJSON(rt, report, exit)
	}
	if !decision.OK {
		fmt.Fprintln(rt.stderr, "no agent has remaining quota at or above the threshold")
		return 3
	}
	fmt.Fprintf(rt.stdout, "agent: %s\ncommand: %s\nreason: %s\n", report.Recommendation.Agent, report.Recommendation.Command, report.Recommendation.Reason)
	return 0
}

func runDoctor(rt runtime, args []string) int {
	_, code, ok := parseFlags(rt, "doctor", args, nil)
	if !ok {
		return code
	}
	writeDoctor(rt.stdout, rt.probe(rt.adapters))
	return 0
}

func (rt runtime) probe(adapters []agent.Adapter) []schema.Agent {
	out := make([]schema.Agent, 0, len(adapters))
	ctx := context.Background()
	for _, a := range adapters {
		det := a.Detect(ctx, rt.env)
		row := a.Status(ctx, rt.env, det)
		row.ApplyStale(rt.now, rt.cfg.MaxAge)
		out = append(out, row)
	}
	return out
}

func (rt runtime) selectAdapters(id string) ([]agent.Adapter, int) {
	if id == "" {
		return rt.adapters, 0
	}
	for _, a := range rt.adapters {
		if a.ID() == id {
			return []agent.Adapter{a}, 0
		}
	}
	fmt.Fprintf(rt.stderr, "unknown agent %q\n", id)
	return nil, 1
}

func (rt runtime) chainAdapters() []agent.Adapter {
	byID := make(map[string]agent.Adapter, len(rt.adapters))
	for _, a := range rt.adapters {
		byID[a.ID()] = a
	}
	out := make([]agent.Adapter, 0, len(rt.cfg.Chain))
	for _, id := range rt.cfg.Chain {
		if a, ok := byID[id]; ok {
			out = append(out, a)
		}
	}
	return out
}

func (rt runtime) command(id string) string {
	for _, a := range rt.adapters {
		if a.ID() == id {
			return a.Command()
		}
	}
	return id
}

func detectionRow(id string, det agent.Detection) schema.Agent {
	note := ""
	if !det.Installed {
		note = "not installed"
	}
	row := schema.Unknown(id, det.Installed, "detect", note)
	row.Version = det.Version
	row.Binary = det.Binary
	row.ConfigPath = det.ConfigPath
	return row
}

func parseFlags(rt runtime, name string, args []string, extra func(*flag.FlagSet)) (bool, int, bool) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(rt.stderr)
	asJSON := fs.Bool("json", false, "print baton/v1 JSON")
	if extra != nil {
		extra(fs)
	}
	if err := fs.Parse(args); err != nil {
		return false, 2, false
	}
	return *asJSON, 0, true
}

func writeJSON(rt runtime, report schema.Report, code int) int {
	buf, err := schema.Marshal(report)
	if err != nil {
		fmt.Fprintf(rt.stderr, "encode json: %v\n", err)
		return 1
	}
	if _, err := rt.stdout.Write(buf); err != nil {
		fmt.Fprintf(rt.stderr, "write json: %v\n", err)
		return 1
	}
	return code
}
