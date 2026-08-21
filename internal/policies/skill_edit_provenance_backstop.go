package policies

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// PolicySkillEditProvenanceBackstop is the diff-scoped backstop over
// shipped ritual content (G-0220 / D-0071). A `SKILL.md` under
// internal/skills/embedded-rituals/** is materialized into consumer
// repos by `aiwf init` / `aiwf update`, so an edit to one reaches
// consumers directly. This policy requires that such an edit ride a
// commit naming the entity it belongs to: an `aiwf-entity:` trailer
// whose value resolves to a real entity in the tree.
//
// What it does NOT require is a policy test referencing the edited
// path. That was the predicate M-0196 shipped, and D-0071 retired it:
// G-0220's complaint was that nothing owned the edit, and a content
// mandate answers a different question — one satisfiable by a test that
// checks nothing in particular, and one that charges per edit rather
// than once.
//
// `aiwf-entity` is the whole of the requirement. `aiwf-verb` is not
// asked for because a ritual SKILL.md is source, not an entity file,
// and no aiwf verb commits it — the closed set trailer-verb-unknown
// enforces carries no value meaning "I edited a shipped surface", so
// requiring one would mandate a fabricated trailer. `aiwf-actor` is not
// asked for because "who ran the verb" is undefined where no verb ran,
// and git's own author field already carries it.
//
// The entity's status is not consulted, only its existence. Two of the
// three invocations below resolve their base to the merge-base with
// trunk, so every skill edit on a branch is re-audited on every run for
// that branch's whole life; keying on status would turn a landed green
// commit red the moment its owning milestone reached a terminal status,
// with nothing about the commit having changed. Whether an attribution
// is apt, rather than merely real, is held at review.
//
// It is a Go policy test (CI tier), not an `aiwf check` finding,
// because the property is an aiwf-repo development invariant —
// meaningless in a consumer tree, where rituals are materialized rather
// than authored.
//
// Like PolicyBranchCoverageAudit, it is diff-scoped and reads its base
// ref from the environment so it keeps the uniform `func(root)
// ([]Violation, error)` shape the runPolicy harness drives:
//
//   - AIWF_COVERAGE_BASE — the git ref to audit forward from. An empty
//     or all-zero value (the default in the broad `go test ./...` job,
//     and a brand-new branch's github.event.before) means "no
//     comparison point" and the audit no-ops. The authoritative
//     invocations are the dedicated CI coverage-gate step and
//     `make coverage-gate`, both of which set it.
func PolicySkillEditProvenanceBackstop(root string) ([]Violation, error) {
	base := strings.TrimSpace(os.Getenv("AIWF_COVERAGE_BASE"))
	return skillEditProvenanceViolations(root, base)
}

// skillRitualsDir is the embedded-ritual authoring tree whose SKILL.md
// edits this policy backstops. The verb-skill tree (embedded/) is out of
// scope — G-0220 is about rituals.
const skillRitualsDir = "internal/skills/embedded-rituals"

// skillEditPolicyID is the violation's policy id, named for the question
// the predicate asks.
const skillEditPolicyID = "skill-edit-provenance-backstop"

// skillEdit is one (commit, watched path) pair: a commit in the audited
// range that added or modified a watched shipped surface, together with
// the `aiwf-entity` trailer value that commit carried (empty when it
// carried none).
type skillEdit struct {
	SHA    string
	Path   string
	Entity string
}

// skillEditProvenanceViolations is the testable IO core: it resolves the
// watched SKILL.md edits committed between baseRef and HEAD along with
// each commit's aiwf-entity trailer, loads the entity tree to decide
// resolution, and delegates the per-edit verdict to
// detectUnownedSkillEdits.
//
// The scope is commits, not the working tree. Provenance is a property a
// commit carries, so an uncommitted edit has none, and firing on one
// would state a fault the operator cannot clear without committing.
func skillEditProvenanceViolations(root, baseRef string) ([]Violation, error) {
	baseRef = strings.TrimSpace(baseRef)
	if baseRef == "" || baseRef == zeroSHA {
		return nil, nil
	}
	edits, err := skillEditsInRange(root, baseRef)
	if err != nil {
		return nil, err
	}
	if len(edits) == 0 {
		return nil, nil
	}
	resolves, err := entityResolver(root)
	if err != nil {
		return nil, err
	}
	return detectUnownedSkillEdits(edits, resolves), nil
}

// detectUnownedSkillEdits is the pure core. An edit is unowned when its
// commit carried no `aiwf-entity` trailer, or carried one naming an id
// that resolves to nothing. resolves reports whether an id names a real
// entity, and is handed an id already rolled up and canonicalized.
//
// A composite id is owned by its parent, so an edit attributed to
// `M-0001/AC-2` is owned by M-0001; ids compare canonicalized, since a
// narrower legacy width names the same entity (`E-01` is `E-0001`).
// Both belong here rather than in the resolver: what counts as owning is
// a question about the rule, decidable without loading a tree.
//
// The two arms carry different Details because they name different
// repairs: one is a missing trailer, the other a wrong value.
func detectUnownedSkillEdits(edits []skillEdit, resolves func(string) bool) []Violation {
	var out []Violation
	for _, e := range edits {
		id := strings.TrimSpace(e.Entity)
		owner := entity.Canonicalize(entity.CompositeRoot(id))
		switch {
		case id == "":
			out = append(out, Violation{
				Policy: skillEditPolicyID,
				File:   e.Path,
				Detail: fmt.Sprintf("commit %s edits this shipped ritual SKILL.md but carries no aiwf-entity: trailer, so nothing records which entity the edit belongs to; re-commit naming the epic, milestone, gap or decision that owns it (`--trailer \"aiwf-entity: <id>\"`) (G-0220).", shortSkillSHA(e.SHA)),
			})
		case id != "" && !resolves(owner):
			out = append(out, Violation{
				Policy: skillEditPolicyID,
				File:   e.Path,
				Detail: fmt.Sprintf("commit %s edits this shipped ritual SKILL.md under aiwf-entity: %s, which resolves to no entity in the tree; provenance that points nowhere is not provenance, so name an entity that exists or create one first.", shortSkillSHA(e.SHA), id),
			})
		}
	}
	return out
}

// entityResolver loads the tree once and returns the id-resolution
// predicate. `prior_ids` are consulted so an entity renumbered by
// `aiwf reallocate` keeps its older commit trailers resolving — the verb
// rewrites references in the tree, but it cannot rewrite a commit
// message.
//
// Per-entity load errors are ignored: a malformed entity elsewhere in
// the tree is a finding for `aiwf check` to report, not a reason this
// gate cannot answer its own question.
func entityResolver(root string) (func(string) bool, error) {
	t, _, err := tree.Load(context.Background(), root)
	if err != nil {
		return nil, fmt.Errorf("loading entity tree at %s: %w", root, err)
	}
	return func(id string) bool {
		return t.ByID(id) != nil || t.ByPriorID(id) != nil
	}, nil
}

// Record and field separators for the `git log` scan. Both are ASCII
// control characters that cannot occur in a path or a trailer value, so
// the parse needs no quoting rules.
const (
	skillEditRecSep = "\x1e"
	skillEditFldSep = "\x1f"
)

// skillEditsInRange returns one skillEdit per (commit, watched path)
// pair added or modified between baseRef and HEAD, sorted by path for
// deterministic output.
//
// Merge commits contribute nothing: `git log` emits no diff for one
// without an explicit --diff-merges, and the commits that actually
// carry the edit are in the range on their own.
func skillEditsInRange(root, baseRef string) ([]skillEdit, error) {
	format := skillEditRecSep + "%H" + skillEditFldSep +
		"%(trailers:key=" + gitops.TrailerEntity + ",valueonly,separator=" + skillEditFldSep + ")"
	cmd := exec.Command("git", "log",
		"--format="+format, "--name-only", "--diff-filter=AM",
		baseRef+"..HEAD", "--", skillRitualsDir)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		// The whole argv, not just the subcommand: an unresolvable base
		// ref is the realistic failure and it is only visible there.
		return nil, fmt.Errorf("git log %s..HEAD in %s: %w\n%s", baseRef, root, err, out)
	}
	return parseSkillEditLog(string(out)), nil
}

// parseSkillEditLog turns the `git log` scan's output into skillEdits.
// Each record is a header line (`<sha><FS><entity-trailer...>`) followed
// by the paths that commit touched.
func parseSkillEditLog(out string) []skillEdit {
	var edits []skillEdit
	for _, rec := range strings.Split(out, skillEditRecSep) {
		lines := strings.Split(strings.TrimSpace(rec), "\n")
		if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
			continue
		}
		header := strings.Split(lines[0], skillEditFldSep)
		sha := strings.TrimSpace(header[0])
		// A commit may in principle carry several aiwf-entity trailers;
		// the first is the owner, and a second would not change the
		// verdict for the first.
		var ent string
		if len(header) > 1 {
			ent = strings.TrimSpace(header[1])
		}
		for _, line := range lines[1:] {
			line = strings.TrimSpace(line)
			if line == "" || !strings.HasSuffix(line, "/SKILL.md") {
				continue
			}
			edits = append(edits, skillEdit{
				SHA:    sha,
				Path:   filepath.ToSlash(line),
				Entity: ent,
			})
		}
	}
	sort.Slice(edits, func(i, j int) bool {
		if edits[i].Path != edits[j].Path {
			return edits[i].Path < edits[j].Path
		}
		return edits[i].SHA < edits[j].SHA
	})
	return edits
}

// shortSkillSHA truncates a SHA for human-readable Details, matching
// git's own 7-character short form.
func shortSkillSHA(sha string) string {
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}
