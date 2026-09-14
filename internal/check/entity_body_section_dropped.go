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

// entityWrite is one commit's write to one entity, as the range log reports it.
type entityWrite struct {
	sha  string
	verb string // the commit's aiwf-verb trailer, empty when it carries none
	path string // where the commit left the entity; empty when it deleted it
}

// WalkDroppedBodySections judges every entity a commit in base..HEAD wrote, by
// id, comparing its starting point against HEAD, and credits each required
// section it lost to the commit that removed it.
//
// Comparing the two ends is what makes the answer a property of the push rather
// than of its commits. A drop reaching HEAD through a `--no-ff` merge is found,
// a later rename or archive does not carry a drop away, and a section added and
// removed again inside the push leaves nothing to report. Identity is the id, so
// a path change is not a create, and a reallocated entity is matched to the id
// it carried before through its prior_ids.
//
// The whole history of the range is walked, merges' side branches included. A
// merge contributes a write only where it resolved content of its own; that is
// what `--cc` reports.
//
// Returns nil when base resolves to no commit: the provenance audit reports an
// unresolvable range itself.
func WalkDroppedBodySections(ctx context.Context, root, base string) []DroppedBodySection {
	baseSHA, ok := gitLines(ctx, root, "rev-parse", "--verify", base+"^{commit}")
	if !ok || len(baseSHA) != 1 {
		return nil
	}
	br, err := gitops.NewBlobReader(ctx, root)
	if err != nil { //coverage:ignore root resolved a commit a moment ago, so it is a repository; NewBlobReader then fails only when git itself cannot start
		return nil
	}
	defer func() { _ = br.Close() }()

	writes, order := entityWrites(ctx, root, baseSHA[0])
	atBase := entityPathsAt(ctx, root, baseSHA[0])
	atHEAD := entityPathsAt(ctx, root, "HEAD")

	var out []DroppedBodySection
	for _, id := range order {
		headPath, ok := atHEAD[id]
		if !ok {
			continue
		}
		raw, headBody, ok := entityBodyAt(br, "HEAD", headPath)
		if !ok {
			continue
		}
		kind, _ := entity.PathKind(headPath)
		exempt := startingOmissions(br, kind, baseSHA[0], priorPath(atBase, id, raw, headPath), writes[id])
		for _, section := range AbsentRequiredSections(kind, headBody) {
			if slices.Contains(exempt, section) {
				continue
			}
			out = append(out, DroppedBodySection{
				SHA:      removalOf(br, kind, section, writes[id]),
				Path:     headPath,
				EntityID: id,
				Section:  section,
			})
		}
	}
	return out
}

// startingOmissions returns the required sections the entity already lacked at
// its starting point, which no push is asked to restore. basePath is where the
// entity sat when the range started, or empty when the range created it — and
// then its first write in the range is the one that created it.
func startingOmissions(br *gitops.BlobReader, kind entity.Kind, baseSHA, basePath string, writes []entityWrite) []string {
	start := entityWrite{sha: baseSHA, path: basePath}
	if basePath == "" {
		start = writes[0]
		if start.verb != "add" && start.verb != "import" {
			return nil
		}
	}
	if _, body, ok := entityBodyAt(br, start.sha, start.path); ok {
		return AbsentRequiredSections(kind, body)
	}
	return nil
}

// priorPath returns where the entity sat when the range started: under its own
// id, or under an id it carried before a reallocation.
func priorPath(atBase map[string]string, id string, headRaw []byte, headPath string) string {
	if p, ok := atBase[id]; ok {
		return p
	}
	if e, err := entity.Parse(headPath, headRaw); err == nil {
		for _, prior := range e.PriorIDs {
			if p, ok := atBase[entity.Canonicalize(prior)]; ok {
				return p
			}
		}
	}
	return ""
}

// removalOf returns the commit that most recently took section out of the
// entity: the newest write whose body lacks it following one that carried it,
// or following no body at all. Only a section missing at HEAD and carried at the
// starting point is asked about, so such a write exists.
func removalOf(br *gitops.BlobReader, kind entity.Kind, section string, writes []entityWrite) string {
	credited := ""
	lackedBefore := false
	for _, w := range writes {
		lacks := false
		if w.path != "" {
			if _, body, ok := entityBodyAt(br, w.sha, w.path); ok {
				lacks = slices.Contains(AbsentRequiredSections(kind, body), section)
			}
		}
		if lacks && !lackedBefore {
			credited = w.sha
		}
		lackedBefore = lacks
	}
	return credited
}

// entityWrites reads base..HEAD oldest-first and returns, per canonical entity
// id, the commits that wrote it in order, plus the ids in the order they were
// first written.
func entityWrites(ctx context.Context, root, baseSHA string) (writes map[string][]entityWrite, order []string) {
	const recSep, fieldSep = "\x1e", "\x1f"
	lines, ok := gitLines(ctx, root, "log", "--reverse", "--topo-order", "--cc", "--no-renames", "--name-status",
		"--format="+recSep+"%H"+fieldSep+"%(trailers:only=true,unfold=true)"+fieldSep, baseSHA+"..HEAD")
	if !ok { //coverage:ignore base resolved to a commit and HEAD exists, so the range is always a valid git log argument
		return nil, nil
	}
	writes = map[string][]entityWrite{}
	for _, rec := range strings.Split(strings.Join(lines, "\n"), recSep)[1:] {
		fields := strings.SplitN(rec, fieldSep, 3)
		w := entityWrite{sha: strings.TrimSpace(fields[0])}
		for _, tr := range gitops.ParseTrailers(fields[1]) {
			if tr.Key == gitops.TrailerVerb {
				w.verb = strings.TrimSpace(tr.Value)
			}
		}
		// A commit can report one entity twice: a rename appears as the old path
		// deleted and the new one added, in either order. It is one write, to
		// wherever the entity ended up.
		written := map[string]string{}
		var ids []string
		for _, line := range strings.Split(fields[2], "\n") {
			status, path, found := strings.Cut(strings.TrimSpace(line), "\t")
			id, isEntity := entityIDFromPath(path)
			if !found || !isEntity {
				continue
			}
			id = entity.Canonicalize(id)
			if _, seen := written[id]; !seen {
				written[id] = ""
				ids = append(ids, id)
			}
			if !strings.Contains(status, "D") {
				written[id] = path
			}
		}
		for _, id := range ids {
			if len(writes[id]) == 0 {
				order = append(order, id)
			}
			writes[id] = append(writes[id], entityWrite{sha: w.sha, verb: w.verb, path: written[id]})
		}
	}
	return writes, order
}

// entityPathsAt returns the path of every entity at rev, keyed by canonical id.
func entityPathsAt(ctx context.Context, root, rev string) map[string]string {
	paths := map[string]string{}
	lines, _ := gitLines(ctx, root, "ls-tree", "-r", "--name-only", rev, "--", "work", "docs/adr")
	for _, path := range lines {
		if id, ok := entityIDFromPath(path); ok {
			paths[entity.Canonicalize(id)] = path
		}
	}
	return paths
}

// entityBodyAt returns the raw bytes and the post-frontmatter body of relPath at
// rev. It reports false when the path does not exist there, or when what it
// holds carries no frontmatter and so is not an entity file.
func entityBodyAt(br *gitops.BlobReader, rev, relPath string) (raw, body []byte, ok bool) {
	raw, err := br.Read(rev, relPath)
	if err != nil { //coverage:ignore every caller names a path git has just listed at rev — ls-tree at the base and HEAD, the log's written path at a commit — so the blob exists
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
