// Package skill is the embedded instructions an agent prints with --skill.
package skill

import _ "embed"

//go:embed SKILL.md
var Text string
