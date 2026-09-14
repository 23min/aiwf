package check

import (
	"context"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	codespkg "github.com/23min/aiwf/internal/codes"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

// CodeEntityBodySectionDropped is the typed kernel-code descriptor for the
// push-seam membership gate.
//
// The write seams hold `aiwf add` and `aiwf edit-body` to the declared section
// set, but a body can reach a commit without passing either: the wrap-milestone
// ritual commits a milestone spec with plain `git commit`. That commit carries
// aiwf trailers, so the untrailered audit beside this rule passes it — which is
// why membership is judged from the committed bytes rather than inferred from
// trailers.
//
// A finding is a required section the entity carried at its starting point and
// does not carry at HEAD. An entity present when the pushed range starts starts
// from that body. One the range creates starts from the body its creating
// commit wrote when that commit came from `aiwf add` or `aiwf import`, and from
// nothing otherwise, so a create that passed no verb is held to the whole set.
var CodeEntityBodySectionDropped = codespkg.Code{
	ID:    "entity-body-section-dropped",
	Class: codespkg.ClassStructural,
}

// DroppedBodySection describes one required section an entity lacks at HEAD
// that it carried at its starting point.
//
// Pre-computed by WalkDroppedBodySections; consumed by
// RunEntityBodySectionDropped. SHA is the commit that left the section out, so
// `aiwf acknowledge illegal` can exempt it; Path is the entity's path at HEAD.
type DroppedBodySection struct {
	SHA      string
	Path     string
	EntityID string
	Section  string
}

// RunEntityBodySectionDropped emits one error finding per record in dropped,
// minus those whose SHA appears in ackedSHAs.
//
// The rule is pure: WalkDroppedBodySections does the git work and the
// comparison; this maps records to findings, the same split
// RunIDRenameUntrailered uses.
func RunEntityBodySectionDropped(dropped []DroppedBodySection, ackedSHAs map[string]bool) []Finding {
	var out []Finding
	for _, d := range dropped {
		if ackedSHAs[d.SHA] {
			continue
		}
		out = append(out, Finding{
			Code:     CodeEntityBodySectionDropped.ID,
			Severity: SeverityError,
			Message: fmt.Sprintf(
				"commit %s leaves required section %q out of %s's body",
				shortHash(d.SHA), "## "+d.Section, d.EntityID,
			),
			Path:     d.Path,
			EntityID: d.EntityID,
		})
	}
	return out
}

// rangeCommit is one commit in the pushed range, as the range log reports it.
type rangeCommit struct {
	sha     string
	parents []string
	verb    string   // the commit's aiwf-verb trailer, empty when it carries none
	forced  bool     // the commit carries aiwf-force
	adds    []string // entity paths the commit added
	touches []string // canonical ids of the entities the commit wrote
}

// treeEntry is an entity file as a tree holds it.
type treeEntry struct{ path, blob string }

// WalkDroppedBodySections judges every entity that differs between base and
// HEAD, by id, and credits each required section it lost to the commit that
// removed it. trunk, when it names a ref, exempts any section the entity already
// lacks there: a removal already on trunk is not the pushing author's debt.
//
// Comparing the two ends makes the answer a property of the push rather than of
// its commits. A drop reaching HEAD through a merge is found, a later rename,
// archive or reallocation does not carry it away, and a section added and
// removed again inside the push leaves nothing to report.
//
// An entity present at base starts from that body, matched through its prior ids
// after a reallocation. One the range creates starts from the body its creating
// commit wrote when that commit came from `aiwf import`, or from `aiwf add` with
// `aiwf-force` — an unforced `aiwf add` refuses a body missing a section, so an
// incomplete create claiming it passed no verb — and from nothing otherwise.
//
// Returns nil when base resolves to no commit: the provenance audit reports an
// unresolvable range itself.
func WalkDroppedBodySections(ctx context.Context, root, base, trunk string) []DroppedBodySection {
	resolved, _ := gitLines(ctx, root, "rev-parse", base+"^{commit}", "HEAD^{commit}")
	if len(resolved) != 2 {
		return nil
	}
	baseSHA, headSHA := resolved[0], resolved[1]
	br, err := gitops.NewBlobReader(ctx, root)
	if err != nil { //coverage:ignore root resolved a commit a moment ago, so it is a repository; NewBlobReader then fails only when git itself cannot start
		return nil
	}
	defer func() { _ = br.Close() }()

	commits, paths := rangeHistory(ctx, root, baseSHA)
	atBase := entitiesAt(ctx, root, baseSHA, paths)
	atHEAD := entitiesAt(ctx, root, headSHA, paths)
	atTrunk := map[string]treeEntry{}
	if trunk != "" {
		atTrunk = entitiesAt(ctx, root, trunk, paths)
	}
	g := &gateReader{br: br, paths: paths, state: map[string]*sectionState{}}

	var out []DroppedBodySection
	for _, id := range sortedIDs(atHEAD) {
		head := atHEAD[id]
		raw, headBody, ok := entityBodyAt(br, headSHA, head.path)
		if !ok {
			continue
		}
		ids := lineage(id, head.path, raw)
		kind, _ := entity.PathKind(head.path)
		start, atStart := firstOf(atBase, ids)
		if atStart && start.blob == head.blob {
			continue
		}
		exempt := startingOmissions(br, kind, baseSHA, start, atStart, ids, commits)
		if t, onTrunk := firstOf(atTrunk, ids); onTrunk {
			if _, body, ok := entityBodyAt(br, trunk, t.path); ok {
				exempt = append(exempt, AbsentRequiredSections(kind, body)...)
			}
		}
		for _, section := range AbsentRequiredSections(kind, headBody) {
			if slices.Contains(exempt, section) {
				continue
			}
			out = append(out, DroppedBodySection{
				SHA:      g.removalOf(ids, kind, section, commits, headSHA),
				Path:     head.path,
				EntityID: id,
				Section:  section,
			})
		}
	}
	return out
}

// startingOmissions returns the required sections the entity already lacked at
// its starting point, which no push is asked to restore. An entity absent at
// base starts from the body its creating commit wrote only when that commit came
// from a verb that may write an incomplete body; otherwise, including when no
// commit in the range created it, it starts from nothing.
func startingOmissions(br *gitops.BlobReader, kind entity.Kind, baseSHA string, start treeEntry, atStart bool, ids []string, commits []rangeCommit) []string {
	rev, path := baseSHA, start.path
	if !atStart {
		c, created, found := creatingCommit(ids, commits)
		if !found || (c.verb != "import" && (c.verb != "add" || !c.forced)) {
			return nil
		}
		rev, path = c.sha, created
	}
	if _, body, ok := entityBodyAt(br, rev, path); ok {
		return AbsentRequiredSections(kind, body)
	}
	return nil
}

// creatingCommit returns the oldest commit in the range that added a path of
// any of ids, and that path.
func creatingCommit(ids []string, commits []rangeCommit) (rangeCommit, string, bool) {
	for i := len(commits) - 1; i >= 0; i-- {
		for _, path := range commits[i].adds {
			if id, _ := entityIDFromPath(path); slices.Contains(ids, entity.Canonicalize(id)) {
				return commits[i], path, true
			}
		}
	}
	return rangeCommit{}, "", false
}

// sectionState is whether an entity exists at a revision, and the required
// sections it lacks there.
type sectionState struct {
	exists bool
	absent []string
}

// gateReader reads an entity's state at any revision in the range, trying every
// path its ids have held, and remembers what it read.
type gateReader struct {
	br    *gitops.BlobReader
	paths map[string][]string
	state map[string]*sectionState
}

func (g *gateReader) at(rev string, ids []string, kind entity.Kind) *sectionState {
	key := rev + "\x00" + strings.Join(ids, ",")
	if st, ok := g.state[key]; ok {
		return st
	}
	st := &sectionState{}
	for _, id := range ids {
		for _, path := range g.paths[id] {
			if _, body, ok := entityBodyAt(g.br, rev, path); ok {
				st = &sectionState{exists: true, absent: AbsentRequiredSections(kind, body)}
				g.state[key] = st
				return st
			}
		}
	}
	g.state[key] = st
	return st
}

// removalOf returns the commit that left section out of the entity. It prefers
// the newest commit whose version lacks it where every parent's version carried
// it or held no such entity — a true removal. A merge adopting a parent's copy
// that lacks it has one parent that carried it, and is credited when no true
// removal exists. HEAD is the last resort, so a finding always names a commit.
func (g *gateReader) removalOf(ids []string, kind entity.Kind, section string, commits []rangeCommit, headSHA string) string {
	lacks := func(rev string) (exists, lacking bool) {
		st := g.at(rev, ids, kind)
		return st.exists, st.exists && slices.Contains(st.absent, section)
	}
	var candidates []rangeCommit
	for _, c := range commits {
		if len(c.parents) > 1 || slices.ContainsFunc(c.touches, func(id string) bool { return slices.Contains(ids, id) }) {
			candidates = append(candidates, c)
		}
	}
	for _, every := range []bool{true, false} {
		for _, c := range candidates {
			if _, lacking := lacks(c.sha); !lacking {
				continue
			}
			carried := 0
			for _, p := range c.parents {
				if exists, lacking := lacks(p); !exists || !lacking {
					carried++
				}
			}
			if (every && carried == len(c.parents)) || (!every && carried > 0) {
				return c.sha
			}
		}
	}
	return headSHA
}

// rangeHistory reads base..HEAD newest-first and returns its commits, plus every
// path each canonical entity id has held in the range.
func rangeHistory(ctx context.Context, root, baseSHA string) (commits []rangeCommit, paths map[string][]string) {
	const recSep, fieldSep = "\x1e", "\x1f"
	lines, ok := gitLines(ctx, root, "log", "--topo-order", "--cc", "--no-renames", "--name-status",
		"--format="+recSep+"%H"+fieldSep+"%P"+fieldSep+"%(trailers:only=true,unfold=true)"+fieldSep, baseSHA+"..HEAD")
	paths = map[string][]string{}
	if !ok { //coverage:ignore base and HEAD both resolved to commits, so the range is always a valid git log argument
		return nil, paths
	}
	for _, rec := range strings.Split(strings.Join(lines, "\n"), recSep)[1:] {
		fields := strings.SplitN(rec, fieldSep, 4)
		c := rangeCommit{sha: strings.TrimSpace(fields[0]), parents: strings.Fields(fields[1])}
		for _, tr := range gitops.ParseTrailers(fields[2]) {
			switch tr.Key {
			case gitops.TrailerVerb:
				c.verb = strings.TrimSpace(tr.Value)
			case gitops.TrailerForce:
				c.forced = true
			}
		}
		for _, line := range strings.Split(fields[3], "\n") {
			status, path, _ := strings.Cut(strings.TrimSpace(line), "\t")
			id, isEntity := entityIDFromPath(path)
			if !isEntity {
				continue
			}
			id = entity.Canonicalize(id)
			addPath(paths, id, path)
			if !slices.Contains(c.touches, id) {
				c.touches = append(c.touches, id)
			}
			if strings.Contains(status, "A") {
				c.adds = append(c.adds, path)
			}
		}
		commits = append(commits, c)
	}
	return commits, paths
}

// entitiesAt returns every entity file at rev, keyed by canonical id, and
// records each path in paths.
func entitiesAt(ctx context.Context, root, rev string, paths map[string][]string) map[string]treeEntry {
	out := map[string]treeEntry{}
	lines, _ := gitLines(ctx, root, "ls-tree", "-r", rev, "--", "work", "docs/adr")
	for _, line := range lines {
		meta, path, _ := strings.Cut(line, "\t")
		id, ok := entityIDFromPath(path)
		if !ok {
			continue
		}
		id = entity.Canonicalize(id)
		fields := strings.Fields(meta)
		out[id] = treeEntry{path: path, blob: fields[len(fields)-1]}
		addPath(paths, id, path)
	}
	return out
}

func addPath(paths map[string][]string, id, path string) {
	if !slices.Contains(paths[id], path) {
		paths[id] = append(paths[id], path)
	}
}

// lineage returns id followed by the canonical ids the entity carried before a
// reallocation, read from its frontmatter at HEAD.
func lineage(id, headPath string, headRaw []byte) []string {
	ids := []string{id}
	if e, err := entity.Parse(headPath, headRaw); err == nil {
		for _, prior := range e.PriorIDs {
			ids = append(ids, entity.Canonicalize(prior))
		}
	}
	return ids
}

// firstOf returns the entry for the first of ids that entries holds.
func firstOf(entries map[string]treeEntry, ids []string) (treeEntry, bool) {
	for _, id := range ids {
		if e, ok := entries[id]; ok {
			return e, true
		}
	}
	return treeEntry{}, false
}

func sortedIDs(entries map[string]treeEntry) []string {
	ids := make([]string, 0, len(entries))
	for id := range entries {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

// entityBodyAt returns the raw bytes and the post-frontmatter body of relPath at
// rev. It reports false when the path holds no blob there, or when what it holds
// carries no frontmatter and so is not an entity file.
func entityBodyAt(br *gitops.BlobReader, rev, relPath string) (raw, body []byte, ok bool) {
	raw, err := br.Read(rev, relPath)
	if err != nil {
		return nil, nil, false
	}
	_, body, ok = entity.Split(raw)
	return raw, body, ok
}

// gitLines runs git in root with quoted paths disabled — a path carrying
// non-ASCII bytes otherwise comes back quoted and matches no entity shape — and
// returns its output lines.
func gitLines(ctx context.Context, root string, args ...string) ([]string, bool) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-c", "core.quotePath=false"}, args...)...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, false
	}
	return strings.Split(strings.TrimRight(string(out), "\n"), "\n"), true
}
