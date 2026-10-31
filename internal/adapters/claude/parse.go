package claude

import (
	"encoding/json"
	"time"

	"github.com/fxckcode/baton/internal/schema"
)

const sourceTag = "local:claude-statusline"

type limitWindow struct {
	UsedPercentage *float64 `json:"used_percentage"`
	ResetsAt       *int64   `json:"resets_at"`
}

type snapshot struct {
	RateLimits *struct {
		FiveHour   *limitWindow `json:"five_hour"`
		SevenDay   *limitWindow `json:"seven_day"`
		SpendLimit *limitWindow `json:"spend_limit"`
	} `json:"rate_limits"`
}

// Parse reads a Claude Code status-line JSON snapshot.
// Context-window percentages are ignored. ok is false when no rate-limit window is usable.
func Parse(data []byte, observed time.Time) (schema.Agent, bool) {
	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil || snap.RateLimits == nil {
		return schema.Agent{}, false
	}
	obs := observed.UTC()
	agent := schema.Agent{
		Confidence: schema.ConfidenceExact,
		Source:     sourceTag,
		Windows:    make([]schema.Window, 0, 3),
		ObservedAt: &obs,
	}
	add := func(name string, raw *limitWindow) {
		w, ok := toWindow(name, raw)
		if ok {
			agent.Windows = append(agent.Windows, w)
		}
	}
	add("5h", snap.RateLimits.FiveHour)
	add("7d", snap.RateLimits.SevenDay)
	add("spend", snap.RateLimits.SpendLimit)
	if len(agent.Windows) == 0 {
		return schema.Agent{}, false
	}
	agent.ApplyRemaining()
	return agent, true
}

func toWindow(name string, raw *limitWindow) (schema.Window, bool) {
	if raw == nil || raw.UsedPercentage == nil {
		return schema.Window{}, false
	}
	rem, ok := schema.RemainingFromUsed(*raw.UsedPercentage)
	if !ok {
		return schema.Window{}, false
	}
	w := schema.Window{Name: name, RemainingPct: rem}
	if raw.ResetsAt != nil {
		reset := time.Unix(*raw.ResetsAt, 0).UTC()
		w.ResetAt = &reset
	}
	return w, true
}
