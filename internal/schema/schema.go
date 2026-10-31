// Package schema is the stable baton/v1 JSON document.
package schema

import (
	"bytes"
	"encoding/json"
	"math"
	"time"
)

// Version is the schema identifier written into every report.
const Version = "baton/v1"

// Confidence is how a remaining percent was obtained.
type Confidence string

const (
	ConfidenceExact     Confidence = "exact"
	ConfidenceEstimated Confidence = "estimated"
	ConfidenceUnknown   Confidence = "unknown"
)

// Valid reports whether c is one of the three schema values.
func (c Confidence) Valid() bool {
	switch c {
	case ConfidenceExact, ConfidenceEstimated, ConfidenceUnknown:
		return true
	default:
		return false
	}
}

// Window is one quota window. RemainingPct and ResetAt are null when unknown.
type Window struct {
	Name         string     `json:"name"`
	RemainingPct *float64   `json:"remaining_pct"`
	ResetAt      *time.Time `json:"reset_at"`
}

// Agent is one coding-agent CLI. Null pointers marshal as JSON null.
type Agent struct {
	Agent        string     `json:"agent"`
	Installed    bool       `json:"installed"`
	Version      string     `json:"version,omitempty"`
	Plan         *string    `json:"plan"`
	Windows      []Window   `json:"windows"`
	RemainingPct *float64   `json:"remaining_pct"`
	Confidence   Confidence `json:"confidence"`
	Source       string     `json:"source"`
	ObservedAt   *time.Time `json:"observed_at"`
	Stale        bool       `json:"stale"`
	Note         string     `json:"note,omitempty"`
	Binary       string     `json:"binary,omitempty"`
	ConfigPath   string     `json:"config_path,omitempty"`
}

// Recommendation is set only when next selects an agent.
type Recommendation struct {
	Agent   string `json:"agent"`
	Command string `json:"command"`
	Reason  string `json:"reason"`
}

// Report is a baton/v1 document.
type Report struct {
	Schema         string          `json:"schema"`
	GeneratedAt    time.Time       `json:"generated_at"`
	Agents         []Agent         `json:"agents"`
	Recommendation *Recommendation `json:"recommendation,omitempty"`
}

// Float returns a rounded percent pointer.
func Float(v float64) *float64 {
	r := Round2(v)
	return &r
}

// Round2 rounds to two decimal places.
func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// RemainingFromUsed converts a consumed percent into a remaining percent.
// used must be finite and >= 0. Values above 100 produce a negative remaining
// percent so overage stays visible.
func RemainingFromUsed(used float64) (*float64, bool) {
	if math.IsNaN(used) || math.IsInf(used, 0) || used < 0 {
		return nil, false
	}
	return Float(100 - used), true
}

// MinRemaining is the smallest non-nil window percent.
func MinRemaining(windows []Window) *float64 {
	var min *float64
	for _, w := range windows {
		if w.RemainingPct == nil {
			continue
		}
		v := *w.RemainingPct
		if min == nil || v < *min {
			min = &v
		}
	}
	return min
}

// ApplyRemaining sets RemainingPct to the minimum window percent.
func (a *Agent) ApplyRemaining() {
	a.RemainingPct = MinRemaining(a.Windows)
}

// ApplyStale sets Stale when the observation is older than maxAge.
// A missing observation is not stale. Age equal to maxAge is not stale.
func (a *Agent) ApplyStale(now time.Time, maxAge time.Duration) {
	if a.ObservedAt == nil || maxAge <= 0 {
		a.Stale = false
		return
	}
	a.Stale = now.Sub(*a.ObservedAt) > maxAge
}

// Marshal renders r as indented baton/v1 JSON with a trailing newline.
// Nil agent slices and nil window slices become empty arrays.
func Marshal(r Report) ([]byte, error) {
	if r.Schema == "" {
		r.Schema = Version
	}
	if r.Agents == nil {
		r.Agents = []Agent{}
	}
	for i := range r.Agents {
		if r.Agents[i].Windows == nil {
			r.Agents[i].Windows = []Window{}
		}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unknown is an installed-or-not row with a null remaining percent.
func Unknown(id string, installed bool, source, note string) Agent {
	return Agent{
		Agent:      id,
		Installed:  installed,
		Confidence: ConfidenceUnknown,
		Source:     source,
		Windows:    []Window{},
		Note:       note,
	}
}

// String returns a pointer to s, or nil when s is empty.
func String(s string) *string {
	if s == "" {
		return nil
	}
	out := s
	return &out
}
