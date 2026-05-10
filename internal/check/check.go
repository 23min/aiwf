// Package check validates an in-memory aiwf tree and returns findings.
//
// Each check is a small pure function from the tree (and its load errors)
// to a slice of findings. Findings carry a code, a severity, a message,
// and optional context (path / entity id / subcode). Run composes all
// eight checks plus per-file load-errors-as-findings into a single slice.
//
// "Errors are findings, not parse failures": Run never returns an error.
// A load error becomes a load-error finding; a malformed entity becomes
// a frontmatter-shape finding; the tree is loaded and validated as far
// as it can go.
package check

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/tree"
)

// Severity classifies a finding. Errors block `aiwf check` (exit 1);
// warnings are surfaced but don't change the exit code unless errors
// are also present.
type Severity string

// Severity values used by every check.
const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Finding is one structured report from a check. The finder fills in
// Code, Severity, and Message; Path and EntityID provide locator
// context where they apply; Subcode distinguishes variants of the same
// finding (e.g., "unresolved" vs "wrong-kind" within refs-resolve).
//
// Line is a 1-based line number in the file at Path. It is filled in
// post-hoc by Run() based on the Field annotation each check sets;
// when the field cannot be located in the file (or no field applies),
// Line is 1 so editors still receive a clickable file:line link.
//
// Hint is a one-line suggestion for what to change to clear the
// finding. It is set by Run() from a Code+Subcode → hint table; checks
// don't populate it directly, so the wording stays consistent.
//
// Field is an internal annotation naming the YAML key the finding is
// "about" (e.g., "parent", "status"). It is not part of the JSON
// envelope; it exists so Run() can resolve a useful Line.
type Finding struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Path     string   `json:"path,omitempty"`
	Line     int      `json:"line,omitempty"`
	EntityID string   `json:"entity_id,omitempty"`
	Subcode  string   `json:"subcode,omitempty"`
	Hint     string   `json:"hint,omitempty"`
	Field    string   `json:"-"`
}

// Run executes every check against the tree and returns all findings,
// ordered first by severity (errors first), then by code, then by path.
// Per-file load errors from the tree loader are surfaced as
// load-error findings ahead of the regular checks.
//
// Run also fills in Line (1-based) and Hint on every finding. Line is
// derived from the field name the check annotated; Hint is looked up
// from a Code+Subcode → hint table.
func Run(t *tree.Tree, loadErrs []tree.LoadError) []Finding {
	var findings []Finding
	findings = append(findings, loadErrorsToFindings(loadErrs)...)
	findings = append(findings, idsUnique(t)...)
	findings = append(findings, casePaths(t)...)
	findings = append(findings, frontmatterShape(t)...)
	findings = append(findings, idPathConsistent(t)...)
	findings = append(findings, statusValid(t)...)
	findings = append(findings, refsResolve(t)...)
	findings = append(findings, noCycles(t)...)
	findings = append(findings, titlesNonempty(t)...)
	findings = append(findings, adrSupersessionMutual(t)...)
	findings = append(findings, gapResolvedHasResolver(t)...)
	// I2: AC and TDD checks.
	findings = append(findings, acsShape(t)...)
	findings = append(findings, acsBodyCoherence(t)...)
	findings = append(findings, acsTDDAudit(t)...)
	findings = append(findings, acsTitleProse(t)...)
	findings = append(findings, milestoneDoneIncompleteACs(t)...)
	findings = append(findings, entityBodyEmpty(t)...)
	// M-083 AC-1: drift-check rule for narrow-width ids in a mixed-
	// state active tree. Per ADR-0008 §"Drift control", uniform trees
	// (either all-narrow or all-canonical) are silent; only the mixed
	// state fires.
	findings = append(findings, entityIDNarrowWidth(t)...)
	resolveLines(t.Root, findings)
	applyHints(findings)
	sortFindings(findings)
	return findings
}

// SortFindings orders findings by severity (errors first), then code,
// then path. Stable so callers that pre-sort within a code group keep
// their order. Exported for callers that merge findings from multiple
// sources (e.g. the CLI's `aiwf check` after appending contract
// findings to the entity-tree slice).
func SortFindings(fs []Finding) {
	sort.SliceStable(fs, func(i, j int) bool {
		if fs[i].Severity != fs[j].Severity {
			return fs[i].Severity == SeverityError
		}
		if fs[i].Code != fs[j].Code {
			return fs[i].Code < fs[j].Code
		}
		return fs[i].Path < fs[j].Path
	})
}

// sortFindings is the internal alias used by Run. Kept as a separate
// symbol so the package's per-call sort can evolve independently of
// the exported shape if needed.
func sortFindings(fs []Finding) { SortFindings(fs) }

// HasErrors reports whether the slice contains any error-severity finding.
func HasErrors(fs []Finding) bool {
	for i := range fs {
		if fs[i].Severity == SeverityError {
			return true
		}
	}
	return false
}

func loadErrorsToFindings(loadErrs []tree.LoadError) []Finding {
	out := make([]Finding, 0, len(loadErrs))
	for _, le := range loadErrs {
		out = append(out, Finding{
			Code:     "load-error",
			Severity: SeverityError,
			Message:  le.Err.Error(),
			Path:     le.Path,
		})
	}
	return out
}

// casePaths reports any pair of entity paths that differ only in
// case. On a case-insensitive filesystem (default macOS APFS,
// Windows NTFS) two such paths refer to the same on-disk location;
// committing them from a case-sensitive Linux dev box and checking
// out on macOS would silently collapse to one entity. Catching this
// at validation time keeps the issue on a list of findings instead
// of in the user's data.
//
// Case-folding is naive ASCII (strings.ToLower on the path) — same
// behavior as macOS HFS+'s default and good enough for the entity
// path conventions, which use only ASCII letters/digits/hyphens by
// the slug grammar.
func casePaths(t *tree.Tree) []Finding {
	groups := make(map[string][]*entity.Entity)
	for _, e := range t.Entities {
		key := strings.ToLower(e.Path)
		groups[key] = append(groups[key], e)
	}
	var findings []Finding
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		// Report every pair beyond the first so multi-way collisions
		// surface each offender. Match idsUnique's "report duplicates,
		// not the canonical first" pattern.
		first := group[0]
		for _, e := range group[1:] {
			findings = append(findings, Finding{
				Code:     "case-paths",
				Severity: SeverityError,
				Message:  fmt.Sprintf("path %q differs only in case from %q; on case-insensitive filesystems they collapse to one entity", e.Path, first.Path),
				Path:     e.Path,
				EntityID: e.ID,
			})
		}
	}
	return findings
}

// idsUnique reports any id that occurs on more than one entity.
// Reports once per duplicate occurrence (the second, third, ... entity
// with the same id), so multi-way collisions surface every duplicate.
//
// When t.TrunkIDs is populated, idsUnique also flags any working-tree
// entity whose id is allocated in the configured trunk ref's tree at
// a different path — the cross-branch case G37 closes. The trunk-side
// path is included in the finding message so the operator can see
// what they're colliding with at a glance. Same-id-same-path on trunk
// is silent: that's the entity already merged to trunk, not a
// collision.
func idsUnique(t *tree.Tree) []Finding {
	seen := make(map[string]*entity.Entity)
	var findings []Finding
	check := func(e *entity.Entity) {
		if existing, ok := seen[e.ID]; ok {
			findings = append(findings, Finding{
				Code:     "ids-unique",
				Severity: SeverityError,
				Message:  fmt.Sprintf("id %q is also used by %s", e.ID, existing.Path),
				Path:     e.Path,
				EntityID: e.ID,
				Field:    "id",
			})
			return
		}
		seen[e.ID] = e
	}
	for _, e := range t.Entities {
		check(e)
	}
	// Stubs (failed-parse files with path-derived ids) participate in
	// uniqueness too — a real entity colliding with a stub, or two stubs
	// claiming the same id, is still an id collision the user wants to
	// know about. Without this, the cascade-suppression fix would trade
	// a noisy false-positive (refs-resolve cascade) for a silent
	// false-negative (missed duplicate id).
	for _, e := range t.Stubs {
		check(e)
	}
	for _, tid := range t.TrunkIDs {
		existing, ok := seen[tid.ID]
		if !ok {
			continue
		}
		if filepath.ToSlash(existing.Path) == tid.Path {
			continue
		}
		findings = append(findings, Finding{
			Code:     "ids-unique",
			Severity: SeverityError,
			Message:  fmt.Sprintf("id %q is allocated on this branch (%s) and on trunk (%s) for different entities", tid.ID, existing.Path, tid.Path),
			Path:     existing.Path,
			EntityID: tid.ID,
			Subcode:  "trunk-collision",
			Field:    "id",
		})
	}
	return findings
}

// frontmatterShape reports missing required fields and id-format
// mismatches per kind. The id format is verified against the entity's
// loader-assigned kind, which catches both bad id strings and files
// placed in the wrong directory.
func frontmatterShape(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if e.ID == "" {
			findings = append(findings, Finding{
				Code:     "frontmatter-shape",
				Severity: SeverityError,
				Message:  "missing required field: id",
				Path:     e.Path,
				Field:    "id",
			})
		} else if err := entity.ValidateID(e.Kind, e.ID); err != nil {
			findings = append(findings, Finding{
				Code:     "frontmatter-shape",
				Severity: SeverityError,
				Message:  err.Error(),
				Path:     e.Path,
				EntityID: e.ID,
				Field:    "id",
			})
		}
		if e.Status == "" {
			findings = append(findings, Finding{
				Code:     "frontmatter-shape",
				Severity: SeverityError,
				Message:  "missing required field: status",
				Path:     e.Path,
				EntityID: e.ID,
				Field:    "status",
			})
		}
		findings = append(findings, perKindRequiredFields(e)...)
	}
	return findings
}

func perKindRequiredFields(e *entity.Entity) []Finding {
	var findings []Finding
	if e.Kind == entity.KindMilestone && e.Parent == "" {
		findings = append(findings, Finding{
			Code:     "frontmatter-shape",
			Severity: SeverityError,
			Message:  "milestone missing required field: parent",
			Path:     e.Path,
			EntityID: e.ID,
			Field:    "parent",
		})
	}
	return findings
}

// idPathConsistent reports any entity whose id in frontmatter does not
// match the id encoded in its file path. The two encodings should
// always agree — `aiwf add` and `aiwf reallocate` keep them in sync,
// and the path's id-prefix is what the G14 stub mechanism falls back
// to when frontmatter is unreadable. A silent disagreement means
// someone bypassed the verbs and edited only one side, leaving every
// reference to the entity ambiguous about which id is canonical.
//
// Stubs are skipped: they are constructed *from* the path-derived id,
// so the comparison would always pass. Entities whose path was
// accepted by PathKind but for which IDFromPath returns false are
// also skipped (defensive — by construction this shouldn't happen,
// since both consult the same patterns).
func idPathConsistent(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		pathID, ok := entity.IDFromPath(e.Path, e.Kind)
		if !ok {
			continue
		}
		// Compare canonical forms so a tree mid-migration (path-name
		// at narrow legacy width while frontmatter id lives at
		// canonical width, or vice versa) is not flagged as a
		// mismatch — per AC-2 in M-081 the parser tolerates both
		// widths. M-082's `aiwf rewidth` realigns paths and ids
		// once the consumer migrates.
		if entity.Canonicalize(pathID) == entity.Canonicalize(e.ID) {
			continue
		}
		findings = append(findings, Finding{
			Code:     "id-path-consistent",
			Severity: SeverityError,
			// Render canonical ids in the user-facing message so
			// the comparison is unambiguous regardless of on-disk
			// width — AC-3 display canonicalization.
			Message: fmt.Sprintf("frontmatter id %q does not match path-encoded id %q",
				entity.Canonicalize(e.ID), entity.Canonicalize(pathID)),
			Path:     e.Path,
			EntityID: e.ID,
			Field:    "id",
		})
	}
	return findings
}

// statusValid reports any status that is not in the kind's allowed set.
// Empty status is reported by frontmatter-shape and skipped here to
// avoid double-reporting.
func statusValid(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if e.Status == "" {
			continue
		}
		if !entity.IsAllowedStatus(e.Kind, e.Status) {
			findings = append(findings, Finding{
				Code:     "status-valid",
				Severity: SeverityError,
				Message: fmt.Sprintf("status %q is not allowed for kind %s (allowed: %s)",
					e.Status, e.Kind, strings.Join(entity.AllowedStatuses(e.Kind), ", ")),
				Path:     e.Path,
				EntityID: e.ID,
				Field:    "status",
			})
		}
	}
	return findings
}

// refsResolve checks every reference field. A reference fails resolution
// if (a) no entity with that id exists ("unresolved"), or (b) the entity
// exists but is not in the allowed-kinds set for the field ("wrong-kind").
//
// Stubs (entities whose file failed to parse) are included in the lookup
// index so that one bad file does not cascade into unresolved-reference
// findings on every entity that links to it. The parse failure is
// already reported as a load-error finding; the cascade would just be
// noise on top.
//
// Composite ids (M-NNN/AC-N, added in I2) are recognized only on open-
// target fields (gap.addressed_by, decision.relates_to). The check
// resolves them by looking up the parent milestone in the index and
// then walking its acs[] for the sub-id; missing parent surfaces as
// `unresolved-milestone`, missing sub as `unresolved-ac`. On closed-
// target fields a composite id falls through to the regular `unresolved`
// path because composites aren't in the index — that's the intended
// signal that composites aren't allowed there.
func refsResolve(t *tree.Tree) []Finding {
	// Index by canonical id so a narrow legacy reference (E-22) and a
	// canonical reference (E-0022) both resolve to the same entity per
	// AC-2's parser-tolerance rule. The stored ID on the entity stays
	// as authored on disk; only the lookup key is canonicalized.
	idx := make(map[string]*entity.Entity, len(t.Entities)+len(t.Stubs))
	for _, e := range t.Entities {
		key := entity.Canonicalize(e.ID)
		if _, exists := idx[key]; exists {
			continue
		}
		idx[key] = e
	}
	for _, e := range t.Stubs {
		key := entity.Canonicalize(e.ID)
		if _, exists := idx[key]; exists {
			continue
		}
		idx[key] = e
	}

	var findings []Finding
	for _, e := range t.Entities {
		for _, ref := range entity.ForwardRefs(e) {
			// Composite-id resolution on open-target fields.
			if entity.IsCompositeID(ref.Target) && len(ref.AllowedKinds) == 0 {
				if f, ok := resolveCompositeRef(e, ref, idx); ok {
					findings = append(findings, f)
				}
				continue
			}
			target, ok := idx[entity.Canonicalize(ref.Target)]
			if !ok {
				findings = append(findings, Finding{
					Code:     "refs-resolve",
					Severity: SeverityError,
					Subcode:  "unresolved",
					Message: fmt.Sprintf("%s field %q references unknown id %q",
						e.Kind, ref.Field, ref.Target),
					Path:     e.Path,
					EntityID: e.ID,
					Field:    ref.Field,
				})
				continue
			}
			if len(ref.AllowedKinds) == 0 {
				continue
			}
			matched := false
			for _, ak := range ref.AllowedKinds {
				if target.Kind == ak {
					matched = true
					break
				}
			}
			if !matched {
				findings = append(findings, Finding{
					Code:     "refs-resolve",
					Severity: SeverityError,
					Subcode:  "wrong-kind",
					Message: fmt.Sprintf("%s field %q expects kind in [%s], but %q is %s",
						e.Kind, ref.Field, joinKinds(ref.AllowedKinds), ref.Target, target.Kind),
					Path:     e.Path,
					EntityID: e.ID,
					Field:    ref.Field,
				})
			}
		}
	}
	return findings
}

// resolveCompositeRef returns a finding (and ok=true) when a composite
// id on an open-target field fails to resolve. ok=false means the
// composite resolved cleanly (no finding needed). Caller has already
// confirmed entity.IsCompositeID(ref.Target) and len(ref.AllowedKinds) == 0.
func resolveCompositeRef(e *entity.Entity, ref entity.ForwardRef, idx map[string]*entity.Entity) (Finding, bool) {
	parent, sub, _ := entity.ParseCompositeID(ref.Target)
	parentEntity, parentOK := idx[entity.Canonicalize(parent)]
	if !parentOK {
		return Finding{
			Code:     "refs-resolve",
			Severity: SeverityError,
			Subcode:  "unresolved-milestone",
			Message: fmt.Sprintf("%s field %q references composite id %q but parent %q does not exist",
				e.Kind, ref.Field, ref.Target, parent),
			Path:     e.Path,
			EntityID: e.ID,
			Field:    ref.Field,
		}, true
	}
	for _, ac := range parentEntity.ACs {
		if ac.ID == sub {
			return Finding{}, false
		}
	}
	return Finding{
		Code:     "refs-resolve",
		Severity: SeverityError,
		Subcode:  "unresolved-ac",
		Message: fmt.Sprintf("%s field %q references %q but %s has no %s in acs[]",
			e.Kind, ref.Field, ref.Target, parent, sub),
		Path:     e.Path,
		EntityID: e.ID,
		Field:    ref.Field,
	}, true
}

func joinKinds(ks []entity.Kind) string {
	parts := make([]string, len(ks))
	for i, k := range ks {
		parts[i] = string(k)
	}
	return strings.Join(parts, ", ")
}

// noCycles detects cycles in the milestone depends_on DAG and the ADR
// supersedes/superseded_by DAG. Each detected cycle produces one
// finding per node on the cycle (so every involved entity is locatable
// in the output).
func noCycles(t *tree.Tree) []Finding {
	var findings []Finding

	// Milestone DAG: edges follow depends_on (M -> M).
	mEdges := make(map[string][]string)
	for _, e := range t.ByKind(entity.KindMilestone) {
		mEdges[e.ID] = append([]string(nil), e.DependsOn...)
	}
	for _, id := range cycleNodes(mEdges) {
		e := t.ByID(id)
		path := ""
		if e != nil {
			path = e.Path
		}
		findings = append(findings, Finding{
			Code:     "no-cycles",
			Severity: SeverityError,
			Subcode:  "depends_on",
			Message:  fmt.Sprintf("milestone %s is on a depends_on cycle", id),
			Path:     path,
			EntityID: id,
			Field:    "depends_on",
		})
	}

	// ADR DAG: edges follow superseded_by (A -> the ADR that supersedes A).
	aEdges := make(map[string][]string)
	for _, e := range t.ByKind(entity.KindADR) {
		if e.SupersededBy != "" {
			aEdges[e.ID] = []string{e.SupersededBy}
		}
	}
	for _, id := range cycleNodes(aEdges) {
		e := t.ByID(id)
		path := ""
		if e != nil {
			path = e.Path
		}
		findings = append(findings, Finding{
			Code:     "no-cycles",
			Severity: SeverityError,
			Subcode:  "supersedes",
			Message:  fmt.Sprintf("ADR %s is on a supersedes/superseded_by cycle", id),
			Path:     path,
			EntityID: id,
			Field:    "superseded_by",
		})
	}

	return findings
}

// cycleNodes returns the set of node ids that participate in any cycle
// in the directed graph. Order is sorted ascending so output is stable.
func cycleNodes(edges map[string][]string) []string {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int)
	onCycle := make(map[string]bool)
	allNodes := make([]string, 0, len(edges))
	for n := range edges {
		allNodes = append(allNodes, n)
	}
	sort.Strings(allNodes)

	var visit func(n string, stack []string)
	visit = func(n string, stack []string) {
		switch color[n] {
		case gray:
			// Cycle: every node from the first occurrence of n in
			// stack down to the current frame is on the cycle.
			for i, s := range stack {
				if s == n {
					for _, c := range stack[i:] {
						onCycle[c] = true
					}
					return
				}
			}
			return
		case black:
			return
		}
		color[n] = gray
		stack = append(stack, n)
		for _, next := range edges[n] {
			visit(next, stack)
		}
		color[n] = black
	}
	for _, n := range allNodes {
		if color[n] == white {
			visit(n, nil)
		}
	}

	out := make([]string, 0, len(onCycle))
	for n := range onCycle {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// titlesNonempty (warning) reports any entity whose title is empty or
// whitespace-only.
func titlesNonempty(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if strings.TrimSpace(e.Title) == "" {
			findings = append(findings, Finding{
				Code:     "titles-nonempty",
				Severity: SeverityWarning,
				Message:  "title is empty or whitespace-only",
				Path:     e.Path,
				EntityID: e.ID,
				Field:    "title",
			})
		}
	}
	return findings
}

// adrSupersessionMutual (warning) reports any ADR A whose
// superseded_by points to B, where B does not list A in its
// supersedes set. Mutual-link consistency is advisory; the
// ids-unique and refs-resolve checks already handle the bookkeeping
// errors that would prevent linking.
func adrSupersessionMutual(t *tree.Tree) []Finding {
	idx := make(map[string]*entity.Entity, len(t.Entities))
	for _, e := range t.Entities {
		if _, ok := idx[e.ID]; !ok {
			idx[e.ID] = e
		}
	}
	var findings []Finding
	for _, a := range t.ByKind(entity.KindADR) {
		if a.SupersededBy == "" {
			continue
		}
		b, ok := idx[a.SupersededBy]
		if !ok || b.Kind != entity.KindADR {
			continue // refs-resolve handles this
		}
		found := false
		for _, s := range b.Supersedes {
			if s == a.ID {
				found = true
				break
			}
		}
		if !found {
			findings = append(findings, Finding{
				Code:     "adr-supersession-mutual",
				Severity: SeverityWarning,
				Message: fmt.Sprintf("ADR %s claims it is superseded by %s, but %s.supersedes does not include %s",
					a.ID, b.ID, b.ID, a.ID),
				Path:     a.Path,
				EntityID: a.ID,
				Field:    "superseded_by",
			})
		}
	}
	return findings
}

// gapResolvedHasResolver (warning) reports any gap with status
// "addressed" but no resolver — neither an entity reference in
// addressed_by nor a commit SHA in addressed_by_commit. A wontfix
// gap doesn't need a resolver: it's discarded by decision, not
// addressed by work.
//
// addressed_by_commit accepts commit SHAs for gaps closed by a
// specific commit rather than a milestone. The kernel's bulk-imported
// legacy gaps (G38) use this — most were closed by a single
// post-iteration hardening commit, not as part of a planned
// milestone, and pointing at a milestone would be revisionist.
func gapResolvedHasResolver(t *tree.Tree) []Finding {
	var findings []Finding
	for _, g := range t.ByKind(entity.KindGap) {
		if g.Status == entity.StatusAddressed && len(g.AddressedBy) == 0 && len(g.AddressedByCommit) == 0 {
			findings = append(findings, Finding{
				Code:     "gap-resolved-has-resolver",
				Severity: SeverityWarning,
				Message:  "gap is marked addressed but addressed_by and addressed_by_commit are both empty",
				Path:     g.Path,
				EntityID: g.ID,
				Field:    "addressed_by",
			})
		}
	}
	return findings
}
