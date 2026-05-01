// Package entity defines the six aiwf entity kinds, their status sets,
// id formats, and the in-memory frontmatter shape every entity carries.
//
// The package is the data model. It deliberately knows nothing about the
// filesystem, git, or validation; the tree package loads entities, the
// check package validates them.
package entity

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Kind identifies one of the six aiwf entity kinds. The value is the
// canonical lowercase identifier used in path-discovery rules and
// in error messages.
type Kind string

// The six aiwf entity kinds. Hardcoded; see docs/poc-design-decisions.md.
const (
	KindEpic      Kind = "epic"
	KindMilestone Kind = "milestone"
	KindADR       Kind = "adr"
	KindGap       Kind = "gap"
	KindDecision  Kind = "decision"
	KindContract  Kind = "contract"
)

// AllKinds returns the closed set of kinds in canonical order. Useful
// for iteration in checks that walk every kind.
func AllKinds() []Kind {
	return []Kind{KindEpic, KindMilestone, KindADR, KindGap, KindDecision, KindContract}
}

// AllowedStatuses returns the closed status set for the kind. Statuses
// outside this set are reported by the status-valid check.
func AllowedStatuses(k Kind) []string {
	switch k {
	case KindEpic:
		return []string{"proposed", "active", "done", "cancelled"}
	case KindMilestone:
		return []string{"draft", "in_progress", "done", "cancelled"}
	case KindADR:
		return []string{"proposed", "accepted", "superseded", "rejected"}
	case KindGap:
		return []string{"open", "addressed", "wontfix"}
	case KindDecision:
		return []string{"proposed", "accepted", "superseded", "rejected"}
	case KindContract:
		return []string{"proposed", "accepted", "deprecated", "retired", "rejected"}
	}
	return nil
}

// IsAllowedStatus reports whether status is in the kind's allowed set.
func IsAllowedStatus(k Kind, status string) bool {
	for _, s := range AllowedStatuses(k) {
		if s == status {
			return true
		}
	}
	return false
}

// IDFormat returns a human-readable description of the kind's id shape.
// Used in error messages produced by the frontmatter-shape check.
func IDFormat(k Kind) string {
	switch k {
	case KindEpic:
		return "E-NN"
	case KindMilestone:
		return "M-NNN"
	case KindADR:
		return "ADR-NNNN"
	case KindGap:
		return "G-NNN"
	case KindDecision:
		return "D-NNN"
	case KindContract:
		return "C-NNN"
	}
	return string(k)
}

// idPatterns maps each kind to the regex that matches its id format.
// The PoC requires at least the canonical pad width but accepts more
// digits (so growth past M-999 doesn't require a regex change).
var idPatterns = map[Kind]*regexp.Regexp{
	KindEpic:      regexp.MustCompile(`^E-\d{2,}$`),
	KindMilestone: regexp.MustCompile(`^M-\d{3,}$`),
	KindADR:       regexp.MustCompile(`^ADR-\d{4,}$`),
	KindGap:       regexp.MustCompile(`^G-\d{3,}$`),
	KindDecision:  regexp.MustCompile(`^D-\d{3,}$`),
	KindContract:  regexp.MustCompile(`^C-\d{3,}$`),
}

// ValidateID returns nil if id matches the kind's format, or an error
// describing the mismatch. Used by the frontmatter-shape check.
func ValidateID(k Kind, id string) error {
	re, ok := idPatterns[k]
	if !ok {
		return fmt.Errorf("unknown kind %q", k)
	}
	if !re.MatchString(id) {
		return fmt.Errorf("id %q does not match %s format", id, IDFormat(k))
	}
	return nil
}

// KindFromID returns the kind matching the id's format. The second
// return is false if the id matches no kind's format. Useful for
// reverse-lookup when validating cross-kind references.
func KindFromID(id string) (Kind, bool) {
	for _, k := range AllKinds() {
		if idPatterns[k].MatchString(id) {
			return k, true
		}
	}
	return "", false
}

// Entity is the in-memory representation of a single aiwf entity, loaded
// from a markdown file's YAML frontmatter. The body prose is not parsed.
//
// The struct is the union of all six kinds' frontmatter fields. Per-kind
// shape rules (which fields are required, which references point to which
// kinds) live in the check package; this struct is the data model.
type Entity struct {
	// Common — present on every kind.
	ID     string `yaml:"id"`
	Title  string `yaml:"title"`
	Status string `yaml:"status"`

	// Milestone references.
	Parent    string   `yaml:"parent,omitempty"`
	DependsOn []string `yaml:"depends_on,omitempty"`

	// ADR chain.
	Supersedes   []string `yaml:"supersedes,omitempty"`
	SupersededBy string   `yaml:"superseded_by,omitempty"`

	// Gap.
	DiscoveredIn string   `yaml:"discovered_in,omitempty"`
	AddressedBy  []string `yaml:"addressed_by,omitempty"`

	// Decision.
	RelatesTo []string `yaml:"relates_to,omitempty"`

	// Contract.
	LinkedADRs []string `yaml:"linked_adrs,omitempty"`

	// Loader-set metadata, not part of YAML.
	Kind Kind   `yaml:"-"`
	Path string `yaml:"-"`
}

// Filename patterns for recognizing entity files during the directory walk.
// Slugs after the id are tolerated but not parsed.
var (
	milestoneFile = regexp.MustCompile(`^M-\d+(-.*)?\.md$`)
	gapFile       = regexp.MustCompile(`^G-\d+(-.*)?\.md$`)
	decisionFile  = regexp.MustCompile(`^D-\d+(-.*)?\.md$`)
	adrFile       = regexp.MustCompile(`^ADR-\d+(-.*)?\.md$`)
)

// idLeadingPattern matches the "<kind>-<digits>" prefix at the start of
// a directory or file basename. ADR is listed first so RE2's leftmost
// alternation does not match D against the leading A of ADR.
var idLeadingPattern = regexp.MustCompile(`^(?:ADR|[EMGDC])-\d+`)

// IDFromPath extracts the entity id encoded in an entity-bearing path,
// for the given kind. The id is the leading "<kind>-<digits>" portion
// of the relevant path component (the parent directory for epic and
// contract; the filename for milestone, gap, decision, and adr); any
// trailing slug is ignored.
//
// Returns false if the path does not match the kind's expected shape
// or the extracted id does not validate. Used by the tree loader to
// register stub entities for files that fail to parse.
func IDFromPath(relPath string, k Kind) (string, bool) {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	var basename string
	switch k {
	case KindEpic:
		if len(parts) != 4 || parts[3] != "epic.md" {
			return "", false
		}
		basename = parts[2]
	case KindContract:
		if len(parts) != 4 || parts[3] != "contract.md" {
			return "", false
		}
		basename = parts[2]
	case KindMilestone, KindGap, KindDecision, KindADR:
		if len(parts) == 0 {
			return "", false
		}
		basename = strings.TrimSuffix(parts[len(parts)-1], ".md")
	default:
		return "", false
	}
	id := idLeadingPattern.FindString(basename)
	if id == "" {
		return "", false
	}
	if err := ValidateID(k, id); err != nil {
		return "", false
	}
	return id, true
}

// PathKind returns the kind implied by a file's path, relative to the
// consumer repo root. The second return is false if the path doesn't
// match any entity-bearing pattern; such files are skipped by the loader.
//
// Recognized patterns:
//
//	work/epics/<dir>/epic.md            -> epic
//	work/epics/<dir>/M-*.md             -> milestone
//	work/gaps/G-*.md                    -> gap
//	work/decisions/D-*.md               -> decision
//	work/contracts/<dir>/contract.md    -> contract
//	docs/adr/ADR-*.md                   -> adr
func PathKind(relPath string) (Kind, bool) {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	switch {
	case len(parts) == 4 && parts[0] == "work" && parts[1] == "epics" && parts[3] == "epic.md":
		return KindEpic, true
	case len(parts) == 4 && parts[0] == "work" && parts[1] == "epics" && milestoneFile.MatchString(parts[3]):
		return KindMilestone, true
	case len(parts) == 3 && parts[0] == "work" && parts[1] == "gaps" && gapFile.MatchString(parts[2]):
		return KindGap, true
	case len(parts) == 3 && parts[0] == "work" && parts[1] == "decisions" && decisionFile.MatchString(parts[2]):
		return KindDecision, true
	case len(parts) == 4 && parts[0] == "work" && parts[1] == "contracts" && parts[3] == "contract.md":
		return KindContract, true
	case len(parts) == 3 && parts[0] == "docs" && parts[1] == "adr" && adrFile.MatchString(parts[2]):
		return KindADR, true
	}
	return "", false
}
