package amp

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fxckcode/baton/internal/schema"
)

const sourceTag = "cli:amp-usage"

var (
	ampNamed = regexp.MustCompile(`(?i)^amp\s+([A-Za-z0-9][A-Za-z0-9 _-]{0,40}?)\s+subscription\s*:\s*([0-9]+(?:\.[0-9]+)?)%\s+other usage\s+and\s+([0-9]+(?:\.[0-9]+)?)%\s+orb usage remaining\b`)
	subNamed = regexp.MustCompile(`(?i)^(?:amp\s+)?subscription\s+([A-Za-z0-9][A-Za-z0-9 _-]{0,40}?)\s*:\s*([0-9]+(?:\.[0-9]+)?)%\s+other usage\s+and\s+([0-9]+(?:\.[0-9]+)?)%\s+orb usage remaining\b`)
	freePct  = regexp.MustCompile(`(?i)^amp free:\s*([0-9]+(?:\.[0-9]+)?)%\s+remaining\b`)
	renew    = regexp.MustCompile(`(?i)resets upon renewal in\s+([0-9]+)\s+days?\b`)
	dollar   = regexp.MustCompile(`(?i)(?:individual credits|workspace\b|amp free)\s*:?\s*\$`)
	ansi     = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
)

// Parse reads `amp usage` text. Dollar balances are not percents.
// ok is false when no usage line was recognized.
func Parse(data []byte, now time.Time) (schema.Agent, bool) {
	text := ansi.ReplaceAllString(string(data), "")
	observed := now.UTC()
	agent := schema.Agent{
		Confidence: schema.ConfidenceExact,
		Source:     sourceTag,
		Windows:    []schema.Window{},
		ObservedAt: &observed,
	}
	sawDollar := false
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if dollar.MatchString(line) {
			sawDollar = true
		}
		if matchSubscription(line, now, &agent) {
			continue
		}
		if m := freePct.FindStringSubmatch(line); m != nil {
			if pct, err := strconv.ParseFloat(m[1], 64); err == nil {
				agent.Windows = append(agent.Windows, schema.Window{Name: "free", RemainingPct: schema.Float(pct)})
			}
		}
	}
	if len(agent.Windows) == 0 {
		if sawDollar {
			out := schema.Unknown("amp", true, sourceTag, "Amp reported a credit balance, not a percentage")
			return out, true
		}
		return schema.Agent{}, false
	}
	agent.ApplyRemaining()
	return agent, true
}

func matchSubscription(line string, now time.Time, agent *schema.Agent) bool {
	var plan, other, orb string
	if m := ampNamed.FindStringSubmatch(line); m != nil {
		plan, other, orb = m[1], m[2], m[3]
	} else if m := subNamed.FindStringSubmatch(line); m != nil {
		plan, other, orb = m[1], m[2], m[3]
	} else {
		return false
	}
	agent.Plan = schema.String(strings.TrimSpace(plan))
	var reset *time.Time
	if m := renew.FindStringSubmatch(line); m != nil {
		days, err := strconv.Atoi(m[1])
		if err == nil {
			when := now.UTC().Add(time.Duration(days) * 24 * time.Hour)
			reset = &when
		}
	}
	agent.Windows = append(agent.Windows,
		schema.Window{Name: "other", RemainingPct: mustPct(other), ResetAt: reset},
		schema.Window{Name: "orb", RemainingPct: mustPct(orb), ResetAt: reset},
	)
	return true
}

func mustPct(raw string) *float64 {
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil
	}
	return schema.Float(v)
}
