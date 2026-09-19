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
// commit wrote when that commit came from `aiwf import` or a forced `aiwf add`,
// and from nothing otherwise, so a create that passed no verb is held to the
// whole set.
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
	adds    []string // entity paths the commit added
	deletes []string // entity paths the commit deleted
	touches []string // entity paths the commit wrote
}

// WalkDroppedBodySections judges every entity file at HEAD that differs from
// where the entity started, and credits each required section it lost to the
// commit that removed it. trunk, when it names a ref, exempts any section the
// entity already lacks there: a removal already on trunk is not the pushing
// author's debt.
//
// Comparing the two ends makes the answer a property of the push rather than of
// its commits. A drop reaching HEAD through a merge is found, a later rename,
// archive or reallocation does not carry it away, and a section added and
// removed again inside the push leaves nothing to report.
//
// An entity is followed by path. A retitle, an archive or a reallocation inside
// the push deletes one entity path and adds another in one commit, and the chain
// of those moves, read back from the file at HEAD, is where the entity was at
// every revision. Two files holding one id at once are two entities, judged
// apart; that state is `ids-unique`'s finding.
//
// An entity present at base starts from its body there. One the range creates
// starts from the body its creating commit wrote when that commit came from
// `aiwf import`, or from `aiwf add` with `aiwf-force` — an unforced `aiwf add`
// refuses a body missing a section, so an incomplete create claiming that verb
// without force did not pass it — and from nothing otherwise.
//
// Returns nil when base resolves to no commit: the provenance audit reports an
// unresolvable range itself.
func WalkDroppedBodySections(ctx context.Context, root, base, trunk string) []DroppedBodySection {
	// Where the branch left its base, not wherever the base has since reached: an
	// upstream that moved on after the fork carries changes this push did not
	// make, and judging against its tip charges them to the pusher. Two histories
	// with no commit in common give the push no starting point at all.
	forked, _ := gitLines(ctx, root, "merge-base", base, "HEAD")
	headRev, _ := gitLines(ctx, root, "rev-parse", "HEAD^{commit}")
	if len(forked) != 1 || len(headRev) != 1 {
		return nil
	}
	baseSHA, headSHA := forked[0], headRev[0]
	br, err := gitops.NewBlobReader(ctx, root)
	if err != nil { //coverage:ignore callers pass the repository root the check verb resolved, and git just answered two rev queries in it
		return nil
	}
	defer func() { _ = br.Close() }()

	commits := rangeHistory(ctx, root, baseSHA)
	atBase := filesAt(ctx, root, baseSHA)
	atHEAD := filesAt(ctx, root, headSHA)
	atTrunk := map[string]string{}
	if trunk != "" {
		atTrunk = filesAt(ctx, root, trunk)
	}
	g := &gateReader{br: br, state: map[string][]string{}, verbs: map[string]createTrailers{}}

	var out []DroppedBodySection
	for _, path := range sortedPaths(atHEAD) {
		raw, headBody, ok := entityBodyAt(ctx, root, br, headSHA, path)
		if !ok {
			continue
		}
		id := canonicalIDOf(path)
		chain, created := chainOf(path, lineage(id, path, raw), commits)
		// Identical bytes carry identical sections, so the common case — an
		// entity the push left alone — costs one map lookup and no read.
		startPath, atStart := firstIn(atBase, chain)
		if atStart && atBase[startPath] == atHEAD[path] {
			continue
		}
		kind, _ := entity.PathKind(path)
		exempt := g.startingOmissions(ctx, root, kind, baseSHA, atStart, chain, created, commits)
		// Trunk exempts only what this entity lacks there, under its own id: a
		// chain path carrying a prior id belongs, on trunk, to whatever entity
		// kept that id.
		for _, p := range chain {
			if _, onTrunk := atTrunk[p]; !onTrunk || canonicalIDOf(p) != id {
				continue
			}
			if _, body, ok := entityBodyAt(ctx, root, br, trunk, p); ok {
				exempt = append(exempt, AbsentRequiredSections(kind, body)...)
			}
			break
		}
		for _, section := range AbsentRequiredSections(kind, headBody) {
			if slices.Contains(exempt, section) {
				continue
			}
			out = append(out, DroppedBodySection{
				SHA:      g.removalOf(ctx, root, chain, kind, section, commits, headSHA),
				Path:     path,
				EntityID: id,
				Section:  section,
			})
		}
	}
	return out
}

// createTrailers is what a create commit's trailers say about how it was made.
type createTrailers struct {
	verb   string // the aiwf-verb value, empty when the commit carries none
	forced bool   // the commit carries aiwf-force
}

// gateReader reads entity bodies across revisions for one walk.
type gateReader struct {
	br    *gitops.BlobReader
	state map[string][]string       // absent sections, by revision and chain
	verbs map[string]createTrailers // a create commit's trailers, by commit
}

// startingOmissions returns the required sections the entity already lacked at
// its starting point, which no push is asked to restore. An entity absent at
// base starts from the body its creating commit wrote only when that commit came
// from a verb that may write an incomplete body; otherwise, including when no
// commit in the range created it, it starts from nothing. created indexes the
// commit that added the chain's oldest path, or is negative.
func (g *gateReader) startingOmissions(ctx context.Context, root string, kind entity.Kind, baseSHA string, atStart bool, chain []string, created int, commits []rangeCommit) []string {
	if atStart {
		return g.absentAt(ctx, root, baseSHA, chain, kind)
	}
	if created < 0 {
		return nil
	}
	c := commits[created]
	if t := g.createVerb(ctx, root, c.sha); t.verb != "import" && (t.verb != "add" || !t.forced) {
		return nil
	}
	return g.absentAt(ctx, root, c.sha, chain[len(chain)-1:], kind)
}

// createVerb returns the aiwf-verb trailer a commit carries, and whether it
// carries aiwf-force, reading each commit once per walk. The range log reads no
// message bytes, so the one commit whose trailers decide anything is read on
// its own.
func (g *gateReader) createVerb(ctx context.Context, root, sha string) createTrailers {
	if t, ok := g.verbs[sha]; ok {
		return t
	}
	var t createTrailers
	lines, _ := gitLines(ctx, root, "show", "-s", "--format=%(trailers:only=true,unfold=true)", sha)
	for _, tr := range gitops.ParseTrailers(strings.Join(lines, "\n")) {
		switch tr.Key {
		case gitops.TrailerVerb:
			t.verb = strings.TrimSpace(tr.Value)
		case gitops.TrailerForce:
			t.forced = true
		}
	}
	g.verbs[sha] = t
	return t
}

// chainOf returns the paths an entity has held across the range, newest first,
// from its path at HEAD, and the index in commits of the commit that added the
// oldest of them — the commit the file at HEAD descends from — or -1 when no
// commit in the range added it. The commit that added a path and, in the same
// commit, deleted one carrying an id the entity claims — its own for a retitle
// or an archive, a prior one for a reallocation — moved the entity, and the
// deleted path is where it was before. Each step looks only at commits older
// than the last move, so the walk ends.
func chainOf(headPath string, ids []string, commits []rangeCommit) (chain []string, created int) {
	chain = []string{headPath}
	path, from := headPath, 0
	for {
		i := slices.IndexFunc(commits[from:], func(c rangeCommit) bool { return slices.Contains(c.adds, path) })
		if i < 0 {
			return chain, -1
		}
		i += from
		j := slices.IndexFunc(commits[i].deletes, func(d string) bool { return slices.Contains(ids, canonicalIDOf(d)) })
		if j < 0 {
			return chain, i
		}
		path, from = commits[i].deletes[j], i+1
		chain = append(chain, path)
	}
}

// absentAt returns the required sections the entity lacks at rev, read from the
// first path of its chain that holds an entity body there; nil where none does.
func (g *gateReader) absentAt(ctx context.Context, root, rev string, chain []string, kind entity.Kind) []string {
	key := rev + "\x00" + strings.Join(chain, "\x00")
	if absent, ok := g.state[key]; ok {
		return absent
	}
	var absent []string
	for _, path := range chain {
		if _, body, ok := entityBodyAt(ctx, root, g.br, rev, path); ok {
			absent = AbsentRequiredSections(kind, body)
			break
		}
	}
	g.state[key] = absent
	return absent
}

// removalOf returns the commit that left section out of the entity. It prefers
// the newest commit whose version lacks it where every parent's version carried
// it or held no such entity — a true removal. A merge adopting a parent's copy
// that lacks it has one parent that carried it, and is credited when no true
// removal exists. The newest candidate is the last resort: an acknowledgment is
// an empty commit, never a candidate, so the commit a finding names does not
// move under one.
func (g *gateReader) removalOf(ctx context.Context, root string, chain []string, kind entity.Kind, section string, commits []rangeCommit, headSHA string) string {
	lacks := func(rev string) bool {
		return slices.Contains(g.absentAt(ctx, root, rev, chain, kind), section)
	}
	var candidates []rangeCommit
	for _, c := range commits {
		if len(c.parents) > 1 || slices.ContainsFunc(c.touches, func(p string) bool { return slices.Contains(chain, p) }) {
			candidates = append(candidates, c)
		}
	}
	for _, every := range []bool{true, false} {
		for _, c := range candidates {
			if !lacks(c.sha) {
				continue
			}
			carried := 0
			for _, p := range c.parents {
				if !lacks(p) {
					carried++
				}
			}
			if (every && carried == len(c.parents)) || (!every && carried > 0) {
				return c.sha
			}
		}
	}
	// A pass above always names a commit. The section is present at base and
	// absent at HEAD, and base is an ancestor of HEAD, so on every line of
	// ancestry from HEAD back to base some commit's version lacks it while a
	// parent's carries it; that commit changed the version, so it lists a chain
	// path or is a merge, and the second pass credits it at the latest. What
	// follows is the shape of the answer, not a path.
	if len(candidates) > 0 { //coverage:ignore unreached, as argued above
		return candidates[0].sha //coverage:ignore unreached, as argued above
	}
	return headSHA //coverage:ignore unreached, as argued above
}

// rangeHistory reads base..HEAD newest-first and returns its commits.
func rangeHistory(ctx context.Context, root, baseSHA string) []rangeCommit {
	// NUL frames both headers and status/path pairs. Consume each path as one
	// field before looking for another header: every non-NUL byte is legal in
	// a path, including any printable or control-byte header marker.
	fields, _ := gitFields(ctx, root, "\x00", "log", "-z", "--topo-order", "--no-renames", "--name-status",
		"--format=commit %H %P", baseSHA+"..HEAD")
	var commits []rangeCommit
	for i := 0; i < len(fields); i++ {
		field := strings.TrimLeft(fields[i], "\n")
		if header, ok := strings.CutPrefix(field, "commit "); ok {
			sha, parents, _ := strings.Cut(header, " ")
			commits = append(commits, rangeCommit{sha: sha, parents: strings.Fields(parents)})
			continue
		}
		if field == "" {
			continue
		}
		i++ // --no-renames gives exactly one path per status.
		path := fields[i]
		if _, isEntity := entityIDFromPath(path); !isEntity {
			continue
		}
		c := &commits[len(commits)-1]
		c.touches = append(c.touches, path)
		switch field {
		case "A":
			c.adds = append(c.adds, path)
		case "D":
			c.deletes = append(c.deletes, path)
		}
	}
	return commits
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

// filesAt returns the blob of every entity-shaped path at rev, keyed by path.
func filesAt(ctx context.Context, root, rev string) map[string]string {
	out := map[string]string{}
	lines, _ := gitFields(ctx, root, "\x00", "ls-tree", "-rz", rev, "--", "work", "docs/adr")
	for _, line := range lines {
		meta, path, _ := strings.Cut(line, "\t")
		if _, ok := entityIDFromPath(path); !ok {
			continue
		}
		fields := strings.Fields(meta)
		out[path] = fields[len(fields)-1]
	}
	return out
}

// firstIn returns the first path of chain that files holds.
func firstIn(files map[string]string, chain []string) (string, bool) {
	for _, path := range chain {
		if _, ok := files[path]; ok {
			return path, true
		}
	}
	return "", false
}

// canonicalIDOf returns the canonical id an entity-shaped path carries.
func canonicalIDOf(path string) string {
	id, _ := entityIDFromPath(path)
	return entity.Canonicalize(id)
}

// sortedPaths returns the keys of files in order, so findings come out in a
// stable order.
func sortedPaths(files map[string]string) []string {
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	return paths
}

// entityBodyAt returns the raw bytes and the post-frontmatter body of relPath at
// rev. It reports false when the path holds no blob there, or when what it holds
// carries no frontmatter and so is not an entity file.
func entityBodyAt(ctx context.Context, root string, br *gitops.BlobReader, rev, relPath string) (raw, body []byte, ok bool) {
	// cat-file --batch takes line-delimited requests. Resolve paths containing
	// LF through an argv argument and send only the object id to that stream.
	var err error
	if strings.Contains(relPath, "\n") {
		ids, resolved := gitLines(ctx, root, "rev-parse", "--verify", rev+":"+relPath)
		if !resolved {
			return nil, nil, false
		}
		raw, err = br.ReadObject(ids[0])
	} else {
		raw, err = br.Read(rev, relPath)
	}
	if err != nil {
		return nil, nil, false
	}
	_, body, ok = entity.Split(raw)
	return raw, body, ok
}

// gitLines runs git in root and returns its output lines.
func gitLines(ctx context.Context, root string, args ...string) ([]string, bool) {
	return gitFields(ctx, root, "\n", args...)
}

// gitFields preserves path bytes when the caller requests NUL-delimited output.
func gitFields(ctx context.Context, root, separator string, args ...string) ([]string, bool) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, false
	}
	return strings.Split(strings.TrimSuffix(string(out), separator), separator), true
}
