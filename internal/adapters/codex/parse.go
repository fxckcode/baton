package codex

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fxckcode/baton/internal/schema"
)

const sourceTag = "local:codex-rollout"

type rateWindow struct {
	UsedPercent     *float64 `json:"used_percent"`
	WindowMinutes   *int     `json:"window_minutes"`
	ResetsAt        *int64   `json:"resets_at"`
	ResetsInSeconds *int64   `json:"resets_in_seconds"`
}

type rolloutLine struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Payload   struct {
		Type       string `json:"type"`
		RateLimits *struct {
			Primary   *rateWindow `json:"primary"`
			Secondary *rateWindow `json:"secondary"`
			PlanType  *string     `json:"plan_type"`
		} `json:"rate_limits"`
	} `json:"payload"`
}

// Parse returns the last token_count rate-limit snapshot in data.
// Transcript text is ignored. ok is false when no window was found.
func Parse(data []byte) (schema.Agent, bool) {
	var found *rolloutLine
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64*1024), 2*1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var ev rolloutLine
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		if ev.Type != "event_msg" || ev.Payload.Type != "token_count" || ev.Payload.RateLimits == nil {
			continue
		}
		if ev.Payload.RateLimits.Primary == nil && ev.Payload.RateLimits.Secondary == nil {
			continue
		}
		copyEv := ev
		found = &copyEv
	}
	if found == nil {
		return schema.Agent{}, false
	}
	observed := parseTime(found.Timestamp)
	agent := schema.Agent{
		Confidence: schema.ConfidenceExact,
		Source:     sourceTag,
		Windows:    make([]schema.Window, 0, 2),
		ObservedAt: observed,
	}
	if found.Payload.RateLimits.PlanType != nil {
		agent.Plan = schema.String(*found.Payload.RateLimits.PlanType)
	}
	if w, ok := toWindow("primary", found.Payload.RateLimits.Primary, observed); ok {
		agent.Windows = append(agent.Windows, w)
	}
	if w, ok := toWindow("secondary", found.Payload.RateLimits.Secondary, observed); ok {
		agent.Windows = append(agent.Windows, w)
	}
	if len(agent.Windows) == 0 {
		return schema.Agent{}, false
	}
	agent.ApplyRemaining()
	return agent, true
}

func toWindow(fallback string, raw *rateWindow, observed *time.Time) (schema.Window, bool) {
	if raw == nil || raw.UsedPercent == nil {
		return schema.Window{}, false
	}
	rem, ok := schema.RemainingFromUsed(*raw.UsedPercent)
	if !ok {
		return schema.Window{}, false
	}
	w := schema.Window{Name: windowName(raw.WindowMinutes, fallback), RemainingPct: rem}
	switch {
	case raw.ResetsAt != nil:
		reset := time.Unix(*raw.ResetsAt, 0).UTC()
		w.ResetAt = &reset
	case raw.ResetsInSeconds != nil && observed != nil:
		reset := observed.Add(time.Duration(*raw.ResetsInSeconds) * time.Second).UTC()
		w.ResetAt = &reset
	}
	return w, true
}

func windowName(minutes *int, fallback string) string {
	if minutes == nil {
		return fallback
	}
	switch *minutes {
	case 300:
		return "5h"
	case 10080:
		return "weekly"
	default:
		return fmt.Sprintf("%dm", *minutes)
	}
}

func parseTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, value)
		if err != nil {
			return nil
		}
	}
	utc := parsed.UTC()
	return &utc
}
