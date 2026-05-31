// Package branchparse holds the canonical ritual-shape branch grammar
// defined by ADR-0010: `epic/E-NNNN-<slug>`, `milestone/M-NNNN-<slug>`,
// `patch/g-NNNN-<slug>` (case-insensitive id segment). Lifted from
// internal/cli/status/worktrees.go in M-0102 so M-0103's preflight,
// M-0102's `aiwf authorize --branch` completion, and the existing
// `aiwf status --worktrees` correlation share one regex set — drift
// between them is structurally impossible.
package branchparse

import (
	"regexp"
	"strings"
)

// branchEntityPattern matches the conventional ritual-branch prefixes:
//
//	epic/E-NNNN-<slug>          → E-NNNN
//	milestone/M-NNNN-<slug>     → M-NNNN
//	patch/g-NNNN-<slug>         → G-NNNN (case-insensitive id segment)
//
// Other shapes (fix/*, chore/*, patch/<topic-without-id>) yield "".
// Narrow-legacy id widths (E-01, M-007) are preserved as-typed on output;
// canonicalization is a downstream concern handled by entity.Canonicalize
// at the consumer's discretion (e.g. when stamping a trailer).
var branchEntityPattern = regexp.MustCompile(`^(?:epic|milestone|patch)/([EeMmGg]-\d+)(?:-|$)`)

// ParseEntityFromBranch tries to derive an entity id from a ritual-shape
// branch name. Honors the conventional `epic/E-NNNN-...`,
// `milestone/M-NNNN-...`, `patch/g-NNNN-...` shapes. Returns "" on no
// match — the caller treats that as "not a ritual branch."
func ParseEntityFromBranch(branch string) string {
	m := branchEntityPattern.FindStringSubmatch(branch)
	if m == nil {
		return ""
	}
	return strings.ToUpper(m[1])
}
