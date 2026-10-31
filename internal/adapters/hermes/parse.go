package hermes

import (
	"encoding/json"
	"time"

	"github.com/fxckcode/baton/internal/schema"
)

const sourceTag = "cli:hermes-usage"

type usageDoc struct {
	Plan        *string `json:"plan"`
	FetchedAt   string  `json:"fetched_at"`
	Unavailable *string `json:"unavailable_reason"`
	Windows     []struct {
		Label    string   `json:"label"`
		Used     *float64 `json:"used_percent"`
		ResetsAt *string  `json:"resets_at"`
	} `json:"windows"`
}

// Parse reads `hermes usage --json`. Provider prose is not copied.
// ok is false only when data is not that document.
func Parse(data []byte, _ time.Time) (schema.Agent, bool) {
	var doc usageDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return schema.Agent{}, false
	}
	if doc.Unavailable != nil && *doc.Unavailable != "" {
		out := schema.Unknown("hermes", true, sourceTag, "provider reported usage unavailable")
		return out, true
	}
	agent := schema.Agent{
		Confidence: schema.ConfidenceExact,
		Source:     sourceTag,
		Plan:       doc.Plan,
		Windows:    make([]schema.Window, 0, len(doc.Windows)),
		ObservedAt: parseTime(doc.FetchedAt),
	}
	if agent.Plan != nil && *agent.Plan == "" {
		agent.Plan = nil
	}
	for _, w := range doc.Windows {
		if w.Used == nil {
			continue
		}
		rem, ok := schema.RemainingFromUsed(*w.Used)
		if !ok {
			continue
		}
		name := w.Label
		if name == "" {
			name = "window"
		}
		item := schema.Window{Name: name, RemainingPct: rem, ResetAt: parseTimePtr(w.ResetsAt)}
		agent.Windows = append(agent.Windows, item)
	}
	if len(agent.Windows) == 0 {
		out := schema.Unknown("hermes", true, sourceTag, "provider omitted percentages")
		out.Plan = agent.Plan
		out.ObservedAt = agent.ObservedAt
		return out, true
	}
	agent.ApplyRemaining()
	return agent, true
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

func parseTimePtr(value *string) *time.Time {
	if value == nil {
		return nil
	}
	return parseTime(*value)
}
