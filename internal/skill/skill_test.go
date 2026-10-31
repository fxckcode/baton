package skill

import (
	"strings"
	"testing"
)

func TestSkillContract(t *testing.T) {
	if !strings.HasPrefix(Text, "---\n") {
		t.Fatal("missing front matter")
	}
	for _, needle := range []string{
		"name: baton",
		"baton next --json",
		"exit",
		"code 3",
		"confidence",
		"credential",
		"antigravity",
		"cursor",
		"claude",
		"codex",
		"threshold",
		"15",
		"stale",
		"unknown",
	} {
		if !strings.Contains(Text, needle) {
			t.Fatalf("skill missing %q", needle)
		}
	}
	if strings.Contains(Text, "sk-ant-") || strings.Contains(Text, "Bearer ") {
		t.Fatal("skill contains a token-shaped example")
	}
}
