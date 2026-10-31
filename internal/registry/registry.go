// Package registry builds the v0.1 adapter list without a process-global variable.
package registry

import (
	"github.com/fxckcode/baton/internal/adapters/amp"
	"github.com/fxckcode/baton/internal/adapters/antigravity"
	"github.com/fxckcode/baton/internal/adapters/catalog"
	"github.com/fxckcode/baton/internal/adapters/claude"
	"github.com/fxckcode/baton/internal/adapters/codex"
	"github.com/fxckcode/baton/internal/adapters/hermes"
	"github.com/fxckcode/baton/internal/adapters/kiro"
	"github.com/fxckcode/baton/internal/agent"
	"github.com/fxckcode/baton/internal/detect"
)

// All returns quota adapters and detection-only adapters in catalog order.
func All() []agent.Adapter {
	quota := map[string]agent.Adapter{
		"claude":      claude.Adapter{},
		"codex":       codex.Adapter{},
		"antigravity": antigravity.Adapter{},
		"hermes":      hermes.Adapter{},
		"amp":         amp.Adapter{},
		"kiro":        kiro.Adapter{},
	}
	out := make([]agent.Adapter, 0, len(detect.Specs()))
	for _, spec := range detect.Specs() {
		if a, ok := quota[spec.ID]; ok {
			out = append(out, a)
			continue
		}
		out = append(out, catalog.New(spec))
	}
	return out
}
