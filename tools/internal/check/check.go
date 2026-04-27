// Package check validates an in-memory aiwf tree and returns findings.
//
// Each check is a small pure function from the tree (and its load errors)
// to a slice of findings. Findings carry a code, a severity, a message,
// and optional context (path / entity id / subcode). Run composes all
// nine checks plus per-file load-errors-as-findings into a single slice.
//
// "Errors are findings, not parse failures": Run never returns an error.
// A load error becomes a load-error finding; a malformed entity becomes
// a frontmatter-shape finding; the tree is loaded and validated as far
// as it can go.
package check

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
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
type Finding struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Path     string   `json:"path,omitempty"`
	EntityID string   `json:"entity_id,omitempty"`
	Subcode  string   `json:"subcode,omitempty"`
}

// Run executes every check against the tree and returns all findings,
// ordered first by severity (errors first), then by code, then by path.
// Per-file load errors from the tree loader are surfaced as
// load-error findings ahead of the regular checks.
func Run(t *tree.Tree, loadErrs []tree.LoadError) []Finding {
	var findings []Finding
	findings = append(findings, loadErrorsToFindings(loadErrs)...)
	findings = append(findings, idsUnique(t)...)
	findings = append(findings, frontmatterShape(t)...)
	findings = append(findings, statusValid(t)...)
	findings = append(findings, refsResolve(t)...)
	findings = append(findings, noCycles(t)...)
	findings = append(findings, contractArtifactExists(t)...)
	findings = append(findings, titlesNonempty(t)...)
	findings = append(findings, adrSupersessionMutual(t)...)
	findings = append(findings, gapResolvedHasResolver(t)...)
	sortFindings(findings)
	return findings
}

func sortFindings(fs []Finding) {
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

// HasErrors reports whether the slice contains any error-severity finding.
func HasErrors(fs []Finding) bool {
	for _, f := range fs {
		if f.Severity == SeverityError {
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

// idsUnique reports any id that occurs on more than one entity.
// Reports once per duplicate occurrence (the second, third, ... entity
// with the same id), so multi-way collisions surface every duplicate.
func idsUnique(t *tree.Tree) []Finding {
	seen := make(map[string]*entity.Entity)
	var findings []Finding
	for _, e := range t.Entities {
		if existing, ok := seen[e.ID]; ok {
			findings = append(findings, Finding{
				Code:     "ids-unique",
				Severity: SeverityError,
				Message:  fmt.Sprintf("id %q is also used by %s", e.ID, existing.Path),
				Path:     e.Path,
				EntityID: e.ID,
			})
			continue
		}
		seen[e.ID] = e
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
			})
		} else if err := entity.ValidateID(e.Kind, e.ID); err != nil {
			findings = append(findings, Finding{
				Code:     "frontmatter-shape",
				Severity: SeverityError,
				Message:  err.Error(),
				Path:     e.Path,
				EntityID: e.ID,
			})
		}
		if e.Status == "" {
			findings = append(findings, Finding{
				Code:     "frontmatter-shape",
				Severity: SeverityError,
				Message:  "missing required field: status",
				Path:     e.Path,
				EntityID: e.ID,
			})
		}
		findings = append(findings, perKindRequiredFields(e)...)
	}
	return findings
}

func perKindRequiredFields(e *entity.Entity) []Finding {
	var findings []Finding
	switch e.Kind {
	case entity.KindMilestone:
		if e.Parent == "" {
			findings = append(findings, Finding{
				Code:     "frontmatter-shape",
				Severity: SeverityError,
				Message:  "milestone missing required field: parent",
				Path:     e.Path,
				EntityID: e.ID,
			})
		}
	case entity.KindContract:
		if e.Format == "" {
			findings = append(findings, Finding{
				Code:     "frontmatter-shape",
				Severity: SeverityError,
				Message:  "contract missing required field: format",
				Path:     e.Path,
				EntityID: e.ID,
			})
		}
		if e.Artifact == "" {
			findings = append(findings, Finding{
				Code:     "frontmatter-shape",
				Severity: SeverityError,
				Message:  "contract missing required field: artifact",
				Path:     e.Path,
				EntityID: e.ID,
			})
		}
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
			})
		}
	}
	return findings
}

// refsResolve checks every reference field. A reference fails resolution
// if (a) no entity with that id exists ("unresolved"), or (b) the entity
// exists but is not in the allowed-kinds set for the field ("wrong-kind").
func refsResolve(t *tree.Tree) []Finding {
	idx := make(map[string]*entity.Entity, len(t.Entities))
	for _, e := range t.Entities {
		if _, exists := idx[e.ID]; exists {
			continue
		}
		idx[e.ID] = e
	}

	var findings []Finding
	for _, e := range t.Entities {
		for _, ref := range collectRefs(e) {
			target, ok := idx[ref.target]
			if !ok {
				findings = append(findings, Finding{
					Code:     "refs-resolve",
					Severity: SeverityError,
					Subcode:  "unresolved",
					Message: fmt.Sprintf("%s field %q references unknown id %q",
						e.Kind, ref.field, ref.target),
					Path:     e.Path,
					EntityID: e.ID,
				})
				continue
			}
			if len(ref.allowed) == 0 {
				continue
			}
			matched := false
			for _, ak := range ref.allowed {
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
						e.Kind, ref.field, joinKinds(ref.allowed), ref.target, target.Kind),
					Path:     e.Path,
					EntityID: e.ID,
				})
			}
		}
	}
	return findings
}

type ref struct {
	field   string
	target  string
	allowed []entity.Kind // empty == any kind
}

func collectRefs(e *entity.Entity) []ref {
	var refs []ref
	switch e.Kind {
	case entity.KindMilestone:
		if e.Parent != "" {
			refs = append(refs, ref{field: "parent", target: e.Parent, allowed: []entity.Kind{entity.KindEpic}})
		}
		for _, dep := range e.DependsOn {
			refs = append(refs, ref{field: "depends_on", target: dep, allowed: []entity.Kind{entity.KindMilestone}})
		}
	case entity.KindADR:
		for _, sup := range e.Supersedes {
			refs = append(refs, ref{field: "supersedes", target: sup, allowed: []entity.Kind{entity.KindADR}})
		}
		if e.SupersededBy != "" {
			refs = append(refs, ref{field: "superseded_by", target: e.SupersededBy, allowed: []entity.Kind{entity.KindADR}})
		}
	case entity.KindGap:
		if e.DiscoveredIn != "" {
			refs = append(refs, ref{field: "discovered_in", target: e.DiscoveredIn, allowed: []entity.Kind{entity.KindMilestone, entity.KindEpic}})
		}
		for _, addr := range e.AddressedBy {
			refs = append(refs, ref{field: "addressed_by", target: addr})
		}
	case entity.KindDecision:
		for _, rel := range e.RelatesTo {
			refs = append(refs, ref{field: "relates_to", target: rel})
		}
	}
	return refs
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

// contractArtifactExists verifies for every contract that the artifact
// path is relative, contains no ".." segments, and resolves to an
// existing file inside the contract directory.
func contractArtifactExists(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.ByKind(entity.KindContract) {
		if e.Artifact == "" {
			continue // covered by frontmatter-shape
		}
		clean := filepath.ToSlash(filepath.Clean(e.Artifact))
		if filepath.IsAbs(clean) {
			findings = append(findings, Finding{
				Code:     "contract-artifact-exists",
				Severity: SeverityError,
				Message:  fmt.Sprintf("artifact path %q must be relative, not absolute", e.Artifact),
				Path:     e.Path,
				EntityID: e.ID,
			})
			continue
		}
		if hasParentSegment(clean) {
			findings = append(findings, Finding{
				Code:     "contract-artifact-exists",
				Severity: SeverityError,
				Message:  fmt.Sprintf("artifact path %q must not contain '..' segments", e.Artifact),
				Path:     e.Path,
				EntityID: e.ID,
			})
			continue
		}
		// Resolve relative to the contract directory (the dir holding contract.md).
		contractDir := filepath.Dir(e.Path)
		relArtifact := filepath.ToSlash(filepath.Join(contractDir, clean))
		if t.HasPlannedFile(relArtifact) {
			continue
		}
		artifactPath := filepath.Join(t.Root, contractDir, clean)
		if !fileExists(artifactPath) {
			findings = append(findings, Finding{
				Code:     "contract-artifact-exists",
				Severity: SeverityError,
				Message:  fmt.Sprintf("artifact %q does not exist (looked in %s)", e.Artifact, contractDir),
				Path:     e.Path,
				EntityID: e.ID,
			})
		}
	}
	return findings
}

func hasParentSegment(slashPath string) bool {
	for _, seg := range strings.Split(slashPath, "/") {
		if seg == ".." {
			return true
		}
	}
	return false
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
			})
		}
	}
	return findings
}

// gapResolvedHasResolver (warning) reports any gap with status
// "addressed" but an empty addressed_by list. A wontfix gap doesn't
// need a resolver — it's discarded by decision, not addressed by work.
func gapResolvedHasResolver(t *tree.Tree) []Finding {
	var findings []Finding
	for _, g := range t.ByKind(entity.KindGap) {
		if g.Status == "addressed" && len(g.AddressedBy) == 0 {
			findings = append(findings, Finding{
				Code:     "gap-resolved-has-resolver",
				Severity: SeverityWarning,
				Message:  "gap is marked addressed but addressed_by is empty",
				Path:     g.Path,
				EntityID: g.ID,
			})
		}
	}
	return findings
}

// fileExists reports whether path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}
