package cliutil

import (
	"fmt"
	"strings"
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
	for _, m := range members {
		if m == area {
			return ""
		}
	}
	if len(members) == 0 {
		return fmt.Sprintf("note: %q is not a declared area (no areas declared in aiwf.yaml)", area)
	}
	return fmt.Sprintf("note: %q is not a declared area (declared: %s)", area, strings.Join(members, ", "))
}
