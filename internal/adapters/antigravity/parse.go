package antigravity

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/fxckcode/baton/internal/schema"
)

const sourceTag = "cli:agy-usage"

// Parse reads `agy -p "/usage" --output-format json`.
// Fractions outside [0, 1] are ignored. ok is false when data is not JSON.
func Parse(data []byte, now time.Time) (schema.Agent, bool) {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return schema.Agent{}, false
	}
	status := fieldString(root, "status")
	if status != "" && !strings.EqualFold(status, "SUCCESS") && !strings.EqualFold(status, "OK") {
		return schema.Unknown("antigravity", true, sourceTag, "output was not a usage payload"), true
	}
	if cmd, ok := asMap(root["command"]); ok {
		name := fieldString(cmd, "name")
		if name != "" && !isUsageName(name) {
			return schema.Unknown("antigravity", true, sourceTag, "output was not a usage payload"), true
		}
	}
	observed := now.UTC()
	agent := schema.Agent{
		Confidence: schema.ConfidenceExact,
		Source:     sourceTag,
		Windows:    []schema.Window{},
		ObservedAt: &observed,
	}
	for _, rawGroup := range groups(root) {
		group, ok := asMap(rawGroup)
		if !ok {
			continue
		}
		gname := fieldString(group, "name", "displayName", "display_name")
		buckets, _ := group["buckets"].([]any)
		for _, rawBucket := range buckets {
			bucket, ok := asMap(rawBucket)
			if !ok || disabled(bucket) {
				continue
			}
			frac, ok := fraction(bucket)
			if !ok || frac < 0 || frac > 1 {
				continue
			}
			label := fieldString(bucket, "window", "name", "displayName", "display_name", "id", "bucketId")
			name := strings.TrimSpace(strings.TrimSpace(gname) + " " + strings.TrimSpace(label))
			if name == "" {
				name = "quota"
			}
			item := schema.Window{Name: name, RemainingPct: schema.Float(frac * 100)}
			if reset := parseReset(fieldString(bucket, "reset_time", "resetTime")); reset != nil {
				item.ResetAt = reset
			}
			agent.Windows = append(agent.Windows, item)
		}
	}
	if len(agent.Windows) == 0 {
		return schema.Unknown("antigravity", true, sourceTag, "output was not a usage payload"), true
	}
	agent.ApplyRemaining()
	return agent, true
}

func isUsageName(name string) bool {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "/")
	return strings.EqualFold(name, "usage")
}

func groups(root map[string]any) []any {
	if cmd, ok := asMap(root["command"]); ok {
		if data, ok := asMap(cmd["data"]); ok {
			if g, ok := data["groups"].([]any); ok {
				return g
			}
		}
	}
	if data, ok := asMap(root["data"]); ok {
		if g, ok := data["groups"].([]any); ok {
			return g
		}
	}
	if g, ok := root["groups"].([]any); ok {
		return g
	}
	return nil
}

func fraction(bucket map[string]any) (float64, bool) {
	if f, ok := fieldFloat(bucket, "remaining_fraction", "remainingFraction"); ok {
		return f, true
	}
	child, ok := asMap(bucket["remaining"])
	if !ok {
		return 0, false
	}
	if kind := fieldString(child, "case"); kind != "" && !strings.EqualFold(kind, "remainingFraction") && !strings.EqualFold(kind, "remaining_fraction") {
		return 0, false
	}
	return fieldFloat(child, "remaining_fraction", "remainingFraction", "value")
}

func disabled(bucket map[string]any) bool {
	flag, ok := bucket["disabled"].(bool)
	return ok && flag
}

func parseReset(value string) *time.Time {
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

func asMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func fieldString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		s, ok := m[key].(string)
		if ok && s != "" {
			return s
		}
	}
	return ""
}

func fieldFloat(m map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		switch n := m[key].(type) {
		case float64:
			return n, true
		case json.Number:
			f, err := n.Float64()
			if err == nil {
				return f, true
			}
		}
	}
	return 0, false
}
