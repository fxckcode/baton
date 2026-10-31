package cli

import (
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"

	"github.com/fxckcode/baton/internal/schema"
)

func writeList(w io.Writer, agents []schema.Agent) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "AGENT\tINSTALLED\tVERSION\tBINARY")
	for _, a := range agents {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", a.Agent, yesNo(a.Installed), a.Version, a.Binary)
	}
	_ = tw.Flush()
}

func writeStatus(w io.Writer, agents []schema.Agent) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "AGENT\tINSTALLED\tREMAINING\tCONFIDENCE\tSTALE\tSOURCE")
	for _, a := range agents {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", a.Agent, yesNo(a.Installed), percent(a.RemainingPct), a.Confidence, yesNo(a.Stale), a.Source)
		if a.Note != "" {
			fmt.Fprintf(tw, "note\t%s\n", a.Note)
		}
	}
	_ = tw.Flush()
}

func writeDoctor(w io.Writer, agents []schema.Agent) {
	for _, a := range agents {
		fmt.Fprintf(w, "%s installed=%s confidence=%s source=%s remaining=%s stale=%s\n",
			a.Agent, yesNo(a.Installed), a.Confidence, a.Source, percent(a.RemainingPct), yesNo(a.Stale))
		if a.Version != "" {
			fmt.Fprintf(w, "  version: %s\n", a.Version)
		}
		if a.Note != "" {
			fmt.Fprintf(w, "  note: %s\n", a.Note)
		}
		if extra := doctorExtra[a.Agent]; extra != "" {
			fmt.Fprintf(w, "  %s\n", extra)
		}
	}
}

var doctorExtra = map[string]string{
	"claude":      "quota is read only from the Claude statusline snapshot in the baton cache",
	"codex":       "quota is read from local rollout files; auth.json is not opened",
	"antigravity": "probe runs agy -p /usage --output-format json",
	"hermes":      "probe runs hermes usage --json",
	"amp":         "probe runs amp usage",
	"kiro":        "probe runs kiro-cli chat --no-interactive and may leave a session file",
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func percent(v *float64) string {
	if v == nil {
		return "-"
	}
	return strconv.FormatFloat(*v, 'f', -1, 64) + "%"
}
