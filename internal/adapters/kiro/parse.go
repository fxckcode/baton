package kiro

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fxckcode/baton/internal/schema"
)

const sourceTag = "cli:kiro-usage"

var (
	ansi     = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	credits  = regexp.MustCompile(`(?i)([0-9]+(?:\.[0-9]+)?)\s+of\s+([0-9]+(?:\.[0-9]+)?)\s+covered in plan`)
	bonus    = regexp.MustCompile(`(?i)([0-9]+(?:\.[0-9]+)?)\s*/\s*([0-9]+(?:\.[0-9]+)?)\s+credits used`)
	resetISO = regexp.MustCompile(`(?i)resets on\s+([0-9]{4}-[0-9]{2}-[0-9]{2})`)
	resetMD  = regexp.MustCompile(`(?i)resets on\s+([0-9]{1,2})/([0-9]{1,2})\b`)
	planName = regexp.MustCompile(`\|\s*(KIRO[A-Z0-9 ]+)`)
)

// Parse reads non-interactive `/usage` text.
// A bare percentage bar is not a number we know how to read.
func Parse(data []byte, now time.Time) (schema.Agent, bool) {
	text := ansi.ReplaceAllString(string(data), "")
	observed := now.UTC()
	agent := schema.Agent{
		Confidence: schema.ConfidenceExact,
		Source:     sourceTag,
		Windows:    []schema.Window{},
		ObservedAt: &observed,
	}
	if m := planName.FindStringSubmatch(text); m != nil {
		agent.Plan = schema.String(strings.TrimSpace(m[1]))
	}
	reset := findReset(text, now)
	if m := credits.FindStringSubmatch(text); m != nil {
		if rem, ok := ratio(m[1], m[2]); ok {
			agent.Windows = append(agent.Windows, schema.Window{Name: "monthly", RemainingPct: rem, ResetAt: reset})
		}
	}
	if m := bonus.FindStringSubmatch(text); m != nil {
		if rem, ok := ratio(m[1], m[2]); ok {
			agent.Windows = append(agent.Windows, schema.Window{Name: "bonus", RemainingPct: rem})
		}
	}
	if len(agent.Windows) == 0 {
		return schema.Agent{}, false
	}
	agent.ApplyRemaining()
	return agent, true
}

func ratio(usedRaw, limitRaw string) (*float64, bool) {
	used, err1 := strconv.ParseFloat(usedRaw, 64)
	limit, err2 := strconv.ParseFloat(limitRaw, 64)
	if err1 != nil || err2 != nil || limit <= 0 || used < 0 || math.IsNaN(used) || math.IsNaN(limit) {
		return nil, false
	}
	return schema.Float((limit - used) / limit * 100), true
}

func findReset(text string, now time.Time) *time.Time {
	if m := resetISO.FindStringSubmatch(text); m != nil {
		parsed, err := time.Parse("2006-01-02", m[1])
		if err != nil {
			return nil
		}
		utc := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
		return &utc
	}
	m := resetMD.FindStringSubmatch(text)
	if m == nil {
		return nil
	}
	month, err1 := strconv.Atoi(m[1])
	day, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil {
		return nil
	}
	year := now.UTC().Year()
	cand := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if cand.Month() != time.Month(month) || cand.Day() != day {
		return nil
	}
	today := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	if cand.Before(today) {
		cand = cand.AddDate(1, 0, 0)
	}
	return &cand
}
