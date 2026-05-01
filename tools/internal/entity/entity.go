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

// The six aiwf entity kinds. Hardcoded; see docs/pocv3/design/design-decisions.md.
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
// outside this set are reported by the status-valid check. Delegates
// to the schemas table so there is a single source of truth.
func AllowedStatuses(k Kind) []string {
	s, ok := schemas[k]
	if !ok {
		return nil
	}
	return s.AllowedStatuses
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

// acAllowedStatuses is the closed status set for an acceptance criterion,
// ordered as the FSM reads: open → met (or deferred/cancelled), met can
// move to deferred or cancelled if scope changes after the fact. Both
// deferred and cancelled are terminal.
var acAllowedStatuses = []string{"open", "met", "deferred", "cancelled"}

// AllowedACStatuses returns the closed status set for an acceptance
// criterion. The returned slice shares memory with the package-level
// constant; callers must not mutate it.
func AllowedACStatuses() []string {
	return acAllowedStatuses
}

// IsAllowedACStatus reports whether s is a recognized AC status. Empty
// string returns false; the empty-string sentinel for "absent" is not
// itself a legal status value.
func IsAllowedACStatus(s string) bool {
	for _, want := range acAllowedStatuses {
		if want == s {
			return true
		}
	}
	return false
}

// tddPhases is the closed phase set for the TDD audit trail on an
// acceptance criterion. Linear: red → green → (refactor) → done.
// `refactor` is optional — green may go directly to done.
var tddPhases = []string{"red", "green", "refactor", "done"}

// AllowedTDDPhases returns the closed phase set for an acceptance
// criterion's `tdd_phase` field. Shares memory with the package-level
// constant; callers must not mutate it.
func AllowedTDDPhases() []string {
	return tddPhases
}

// IsAllowedTDDPhase reports whether p is a recognized TDD phase. Empty
// string returns false; phase absence (when the parent milestone is
// `tdd: none` or `tdd: advisory`) is checked separately by the caller.
func IsAllowedTDDPhase(p string) bool {
	for _, want := range tddPhases {
		if want == p {
			return true
		}
	}
	return false
}

// tddPolicies is the closed policy set for a milestone's `tdd:` field.
// `none` is the default when the field is absent on the milestone;
// `advisory` softens the audit to a warning; `required` makes it an
// error on push.
var tddPolicies = []string{"required", "advisory", "none"}

// AllowedTDDPolicies returns the closed policy set for a milestone's
// `tdd:` field. Shares memory with the package-level constant; callers
// must not mutate it.
func AllowedTDDPolicies() []string {
	return tddPolicies
}

// IsAllowedTDDPolicy reports whether p is a recognized policy value.
// Empty string returns false; the absent-field default is `none`, but
// the caller is responsible for substituting that before consulting
// this predicate (the empty string is not itself a legal policy value).
func IsAllowedTDDPolicy(p string) bool {
	for _, want := range tddPolicies {
		if want == p {
			return true
		}
	}
	return false
}

// IDFormat returns a human-readable description of the kind's id shape.
// Used in error messages produced by the frontmatter-shape check.
// Delegates to the schemas table so there is a single source of truth.
func IDFormat(k Kind) string {
	s, ok := schemas[k]
	if !ok {
		return string(k)
	}
	return s.IDFormat
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

// compositeIDPattern matches the "<entity-id>/AC-<digits>" composite-id
// form used for namespaced sub-elements (added in I2). Only milestones
// currently have ACs, so the parent portion is restricted to a milestone
// id (`M-\d{3,}`). The regex is anchored end-to-end; partial matches
// like `M-007/AC-1-extra` are rejected.
//
// AC ids inside a composite are `AC-\d+` with no minimum-digit
// requirement at the grammar layer; the position-equality rule
// (`acs-shape`, added in Step 6) is what enforces "AC-N equals
// position+1 within the parent's `acs[]`".
var compositeIDPattern = regexp.MustCompile(`^(M-\d{3,})/(AC-\d+)$`)

// IsCompositeID reports whether s matches the composite-id grammar
// `<entity-id>/AC-<digits>`. Bare ids and malformed inputs return false.
func IsCompositeID(s string) bool {
	return compositeIDPattern.MatchString(s)
}

// ParseCompositeID splits a composite id into its parent and sub
// portions. Returns ok=false (with both strings empty) when s does not
// match the composite grammar — including bare ids, malformed parents,
// missing sub-ids, and trailing garbage.
func ParseCompositeID(s string) (parent, sub string, ok bool) {
	m := compositeIDPattern.FindStringSubmatch(s)
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

// KindFromID returns the kind matching the id's format. The second
// return is false if the id matches no kind's format. Useful for
// reverse-lookup when validating cross-kind references.
//
// Composite ids (e.g. `M-007/AC-1`) resolve to their parent's kind
// (here, milestone). The sub-kind is reported separately by
// SubKindFromID.
func KindFromID(id string) (Kind, bool) {
	if parent, _, ok := ParseCompositeID(id); ok {
		return KindFromID(parent)
	}
	for _, k := range AllKinds() {
		if idPatterns[k].MatchString(id) {
			return k, true
		}
	}
	return "", false
}

// SubKindFromID returns the sub-kind label encoded in a composite id.
// Currently only `"ac"` is defined (acceptance criterion). Bare ids and
// malformed composites return ("", false).
func SubKindFromID(id string) (string, bool) {
	if !IsCompositeID(id) {
		return "", false
	}
	return "ac", true
}

// AcceptanceCriterion is a milestone sub-element addressed by composite id
// `M-NNN/AC-N`. ACs are namespaced inside their parent milestone — they
// have no global allocator and cannot move between milestones (see
// docs/pocv3/design/design-decisions.md §"Acceptance criteria and TDD").
//
// Every field is a plain string with `omitempty`; empty == absent.
// Closed-set membership (IsAllowedACStatus, IsAllowedTDDPhase) rules out
// "" as a legal value, so the empty-string sentinel is unambiguous.
//
// AC ids are position-stable: `acs[i].ID == fmt.Sprintf("AC-%d", i+1)` for
// every index. Cancelled ACs stay in the slice at their original position;
// the allocator picks `max+1` over the full list including cancelled.
type AcceptanceCriterion struct {
	ID       string `yaml:"id,omitempty"`
	Title    string `yaml:"title,omitempty"`
	Status   string `yaml:"status,omitempty"`
	TDDPhase string `yaml:"tdd_phase,omitempty"`
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

	// Milestone — acceptance criteria and TDD policy (added in I2).
	TDD string                `yaml:"tdd,omitempty"`
	ACs []AcceptanceCriterion `yaml:"acs,omitempty"`

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

// Cardinality describes whether a reference field carries a single id
// or a list of ids.
type Cardinality string

// Cardinality values used by RefField.
const (
	Single Cardinality = "single"
	Multi  Cardinality = "multi"
)

// RefField describes one reference field on an entity. AllowedKinds
// is the set of kinds the target id may resolve to; an empty slice
// means any kind is allowed (e.g. gap.addressed_by, decision.relates_to).
// Optional reports whether the field is allowed to be empty/absent.
type RefField struct {
	Name         string      `json:"name"`
	Cardinality  Cardinality `json:"cardinality"`
	AllowedKinds []Kind      `json:"allowed_kinds,omitempty"`
	Optional     bool        `json:"optional"`
}

// Schema describes the frontmatter contract for one kind. It is the
// single source of truth consulted by `aiwf schema` (the published
// surface for skill authors) and by check.refsResolve (the runtime
// enforcement). Drift between the two is pinned by a regression test
// in the check package.
type Schema struct {
	Kind            Kind       `json:"kind"`
	IDFormat        string     `json:"id_format"`
	AllowedStatuses []string   `json:"allowed_statuses"`
	RequiredFields  []string   `json:"required_fields"`
	OptionalFields  []string   `json:"optional_fields,omitempty"`
	References      []RefField `json:"references,omitempty"`
}

// commonRequired is the field set required on every kind: id and status
// hard-required (frontmatter-shape errors), title soft-required (a
// titles-nonempty warning if missing). All three are listed because the
// schema verb describes "what a coherent entity carries," not the per-
// finding severity of omission.
var commonRequired = []string{"id", "title", "status"}

// schemas is the per-kind schema table. The same data drives the
// `aiwf schema` verb output and check.refsResolve's allowed-kinds set.
// To add or change a field for a kind: edit this table, the Entity
// struct's yaml-tagged field, and (if the field is a reference) the
// matching arm in check.collectRefs. The TestSchemaMatchesCollectRefs
// regression test will catch drift between the schema table and what
// check.collectRefs actually reads.
var schemas = map[Kind]Schema{
	KindEpic: {
		Kind:            KindEpic,
		IDFormat:        "E-NN",
		AllowedStatuses: []string{"proposed", "active", "done", "cancelled"},
		RequiredFields:  commonRequired,
	},
	KindMilestone: {
		Kind:            KindMilestone,
		IDFormat:        "M-NNN",
		AllowedStatuses: []string{"draft", "in_progress", "done", "cancelled"},
		RequiredFields:  append(append([]string(nil), commonRequired...), "parent"),
		OptionalFields:  []string{"depends_on", "tdd", "acs"},
		References: []RefField{
			{Name: "parent", Cardinality: Single, AllowedKinds: []Kind{KindEpic}, Optional: false},
			{Name: "depends_on", Cardinality: Multi, AllowedKinds: []Kind{KindMilestone}, Optional: true},
		},
	},
	KindADR: {
		Kind:            KindADR,
		IDFormat:        "ADR-NNNN",
		AllowedStatuses: []string{"proposed", "accepted", "superseded", "rejected"},
		RequiredFields:  commonRequired,
		OptionalFields:  []string{"supersedes", "superseded_by"},
		References: []RefField{
			{Name: "supersedes", Cardinality: Multi, AllowedKinds: []Kind{KindADR}, Optional: true},
			{Name: "superseded_by", Cardinality: Single, AllowedKinds: []Kind{KindADR}, Optional: true},
		},
	},
	KindGap: {
		Kind:            KindGap,
		IDFormat:        "G-NNN",
		AllowedStatuses: []string{"open", "addressed", "wontfix"},
		RequiredFields:  commonRequired,
		OptionalFields:  []string{"discovered_in", "addressed_by"},
		References: []RefField{
			{Name: "discovered_in", Cardinality: Single, AllowedKinds: []Kind{KindMilestone, KindEpic}, Optional: true},
			// addressed_by accepts any kind — empty AllowedKinds.
			{Name: "addressed_by", Cardinality: Multi, Optional: true},
		},
	},
	KindDecision: {
		Kind:            KindDecision,
		IDFormat:        "D-NNN",
		AllowedStatuses: []string{"proposed", "accepted", "superseded", "rejected"},
		RequiredFields:  commonRequired,
		OptionalFields:  []string{"relates_to"},
		References: []RefField{
			// relates_to accepts any kind — empty AllowedKinds.
			{Name: "relates_to", Cardinality: Multi, Optional: true},
		},
	},
	KindContract: {
		Kind:            KindContract,
		IDFormat:        "C-NNN",
		AllowedStatuses: []string{"proposed", "accepted", "deprecated", "retired", "rejected"},
		RequiredFields:  commonRequired,
		OptionalFields:  []string{"linked_adrs"},
		References: []RefField{
			{Name: "linked_adrs", Cardinality: Multi, AllowedKinds: []Kind{KindADR}, Optional: true},
		},
	},
}

// SchemaForKind returns the Schema for k. The second return is false
// if k is not one of the six aiwf kinds. The returned Schema shares
// slice memory with the package-level table; callers must not mutate it.
func SchemaForKind(k Kind) (Schema, bool) {
	s, ok := schemas[k]
	return s, ok
}

// AllSchemas returns one Schema per kind, in AllKinds() order.
func AllSchemas() []Schema {
	out := make([]Schema, 0, len(schemas))
	for _, k := range AllKinds() {
		out = append(out, schemas[k])
	}
	return out
}

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
