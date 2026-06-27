package cliutil

import (
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/entity"
)

// UndeclaredAreaNote returns a one-line advisory note for a read verb's
// `--area` filter when the requested value is not a declared workstream
// area, or "" when no note is warranted (E-0043, M-0174/AC-5). It is the
// single source the `list`, `show`, and `status` verbs share so the
// message stays uniform across the read surface.
//
// The note is purely advisory: the verbs print it to stderr but the
// filter itself stays mechanical (effective-area == requested value), so
// an entity mis-tagged with an undeclared area via a hand-edit still
// surfaces under that value — the note just flags that the value is not
// one the operator declared (the M-0172 area-unknown check is the
// backstop for the mis-tag itself). Returns "":
//   - when area is empty (no filter requested), or
//   - when area is one of aiwf.yaml's declared areas.members.
//
// Otherwise it names the offending value and, when an areas block
// exists, the declared set; when no areas block exists at all it points
// at the missing block. Tolerant of a missing aiwf.yaml via
// ConfiguredAreaMembers.
func UndeclaredAreaNote(rootDir, area string) string {
	if area == "" {
		return ""
	}
	members := ConfiguredAreaMembers(rootDir)
	// Position A — `global` is feature-gated: with no areas block the field
	// is inert (M-0171), so EVERY value (including the reserved global
	// sentinel) is "not a declared area" and gets the advisory note. This
	// no-block branch precedes the IsValidAreaValue check so global cannot
	// slip through it (the predicate accepts global regardless of the
	// declared set). With a block declared, global and any declared member
	// return "" (no note) — the note is advisory only.
	if len(members) == 0 {
		return fmt.Sprintf("note: %q is not a declared area (no areas declared in aiwf.yaml)", area)
	}
	if entity.IsValidAreaValue(area, members) {
		return ""
	}
	return fmt.Sprintf("note: %q is not a declared area (declared: %s)", area, strings.Join(members, ", "))
}
