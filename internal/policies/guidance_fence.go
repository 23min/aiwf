package policies

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/pathutil"
	"github.com/23min/aiwf/internal/projectguidance"
)

// PolicyGuidanceFence is the commit-seam gate over this repository's
// development guidance. A commit that changes handwritten guidance — the
// root CLAUDE.md or AGENTS.md outside their managed blocks, the project
// router .guidance/project.md, or a document that router links to — is
// its own logical change, so it may carry only related guidance files,
// the guidance source and the outputs generated from it, and
// configuration.
//
// A change confined to a managed block is generated output and is judged
// through the source that renders it, not here. Merge commits add no
// judgment of their own: `git log` emits no diff for one, and every
// commit a merge brings in is in the range and judged on its own.
//
// It is a Go policy test (CI tier) rather than an `aiwf check` finding,
// because the property is this repository's development invariant and is
// meaningless in a consumer tree. Like PolicySkillEditProvenanceBackstop
// it judges commits between AIWF_COVERAGE_BASE and HEAD, and no-ops when
// no base is set.
func PolicyGuidanceFence(root string) ([]Violation, error) {
	return guidanceFenceViolations(root, os.Getenv("AIWF_COVERAGE_BASE"))
}

// Paths the fence reads by name.
const (
	fenceClaudeMD     = "CLAUDE.md"
	fenceAgentsMD     = "AGENTS.md"
	fenceRouter       = ".guidance/project.md"
	fenceOwnedRecord  = ".guidance/.aiwf-owned"
	fenceConfig       = "aiwf.yaml"
	fenceGuidanceSrc  = "internal/skills/embedded-guidance/"
	fenceRecSep       = "\x1e"
	fenceRouterLinkRE = `\[[^\]]*\]\(([^)\s]+)\)`
)

var fenceRouterLink = regexp.MustCompile(fenceRouterLinkRE)

// fenceCommit is one commit in the audited range, reduced to what the
// fence judges: which handwritten guidance files it changed and which
// changed files are unrelated to guidance.
type fenceCommit struct {
	SHA       string
	Guidance  []string
	Unrelated []string
}

// guidanceFenceViolations is the IO core: it reduces every commit in
// baseRef..HEAD to a fenceCommit and delegates the verdict to
// detectGuidanceFence.
func guidanceFenceViolations(root, baseRef string) ([]Violation, error) {
	baseRef = strings.TrimSpace(baseRef)
	if baseRef == "" || baseRef == zeroSHA {
		return nil, nil
	}
	commits, err := fenceCommitsInRange(root, baseRef)
	if err != nil {
		return nil, err
	}
	return detectGuidanceFence(commits), nil
}

// detectGuidanceFence is the pure core. A commit that changed no
// handwritten guidance is outside the fence; one that did is refused once
// per unrelated file it also carries.
func detectGuidanceFence(commits []fenceCommit) []Violation {
	var out []Violation
	for _, c := range commits {
		if len(c.Guidance) == 0 {
			continue
		}
		for _, p := range c.Unrelated {
			out = append(out, Violation{
				Policy: "guidance-fence",
				File:   p,
				Detail: fmt.Sprintf("commit %s changes handwritten guidance (%s) and also this file, which is neither guidance, its source, its generated output nor configuration; a guidance change is its own commit, so move this file into a separate one.", shortSkillSHA(c.SHA), strings.Join(c.Guidance, ", ")),
			})
		}
	}
	return out
}

// fenceChange is one changed path pair from `git log --name-status`. Old
// is empty for an addition and New for a deletion; both are set, and
// equal, for a modification.
type fenceChange struct {
	Old, New string
}

// fenceCommitsInRange lists the commits in baseRef..HEAD with their
// changed paths and classifies each commit against the guidance set as it
// stood at the commit and at its first parent.
//
// Renames are detected so a moved guidance document reports as one pair;
// the settings are pinned on the invocation so the verdict does not vary
// with the caller's git config. Every guidance path is a markdown file,
// so a commit changing none is outside the fence without reading
// anything; the rest read their content through one cat-file pump.
func fenceCommitsInRange(root, baseRef string) ([]fenceCommit, error) {
	cmd := exec.Command("git", "-c", "core.quotePath=false", "-c", "diff.renames=true", "log",
		"--format="+fenceRecSep+"%H %P", "--name-status", baseRef+"..HEAD")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git log %s..HEAD in %s: %w\n%s", baseRef, root, err, out)
	}
	reader, err := gitops.NewBlobReader(context.Background(), root)
	if err != nil { //coverage:ignore root is a repository, or git log above would have failed
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	cl := &fenceClassifier{reader: reader, revisions: map[string]fenceRevision{}}
	var commits []fenceCommit
	for _, rec := range strings.Split(string(out), fenceRecSep) {
		lines := strings.Split(strings.TrimSpace(rec), "\n")
		revs := strings.Fields(lines[0])
		changes := parseFenceChanges(lines[1:])
		if len(revs) == 0 || !touchesMarkdown(changes) {
			continue
		}
		// The first parent is the side a commit's diff is taken against;
		// a root commit has none, and reads as an empty tree.
		parent := ""
		if len(revs) > 1 {
			parent = revs[1]
		}
		c, err := cl.classify(revs[0], parent, changes)
		if err != nil { //coverage:ignore propagates a cat-file pump failure, which needs the git subprocess to die mid-run
			return nil, err
		}
		commits = append(commits, c)
	}
	return commits, nil
}

func touchesMarkdown(changes []fenceChange) bool {
	for _, ch := range changes {
		if strings.HasSuffix(ch.Old, ".md") || strings.HasSuffix(ch.New, ".md") {
			return true
		}
	}
	return false
}

// parseFenceChanges turns `--name-status` lines into path pairs. A rename
// line carries a similarity score and two paths; every other status
// carries one.
func parseFenceChanges(lines []string) []fenceChange {
	var out []fenceChange
	for _, line := range lines {
		fields := strings.Split(strings.TrimSpace(line), "\t")
		if len(fields) < 2 {
			continue
		}
		switch {
		case strings.HasPrefix(fields[0], "R") && len(fields) == 3:
			out = append(out, fenceChange{Old: fields[1], New: fields[2]})
		case fields[0] == "A":
			out = append(out, fenceChange{New: fields[1]})
		case fields[0] == "D":
			out = append(out, fenceChange{Old: fields[1]})
		default:
			out = append(out, fenceChange{Old: fields[1], New: fields[1]})
		}
	}
	return out
}

// fenceRevision is the guidance set and its related files as they stood
// at one revision.
type fenceRevision struct {
	routed map[string]bool
	owned  map[string]bool
}

// fenceClassifier reads file content at revisions through one cat-file
// pump and caches each revision's guidance set, since a commit's parent is
// usually the previous commit judged.
type fenceClassifier struct {
	reader    *gitops.BlobReader
	revisions map[string]fenceRevision
}

// classify decides, for each changed pair, whether it is a handwritten
// guidance change, a related file, or unrelated.
func (cl *fenceClassifier) classify(sha, parent string, changes []fenceChange) (fenceCommit, error) {
	c := fenceCommit{SHA: sha}
	before, err := cl.revision(parent)
	if err != nil { //coverage:ignore propagates a cat-file pump failure
		return c, err
	}
	after, err := cl.revision(sha)
	if err != nil { //coverage:ignore propagates a cat-file pump failure
		return c, err
	}
	for _, ch := range changes {
		guidance := before.isGuidance(ch.Old) || after.isGuidance(ch.New)
		if guidance {
			changed, err := cl.handwrittenChanged(parent, sha, ch)
			if err != nil { //coverage:ignore propagates a cat-file pump failure
				return c, err
			}
			if changed {
				c.Guidance = append(c.Guidance, fenceReportPath(ch))
				continue
			}
		}
		if guidance || before.isRelated(ch.Old) || after.isRelated(ch.New) {
			continue
		}
		c.Unrelated = append(c.Unrelated, fenceReportPath(ch))
	}
	sort.Strings(c.Unrelated)
	return c, nil
}

// fenceReportPath names a pair by the path that exists after the commit,
// or the removed path for a deletion.
func fenceReportPath(ch fenceChange) string {
	if ch.New != "" {
		return ch.New
	}
	return ch.Old
}

func (r fenceRevision) isGuidance(p string) bool {
	return p == fenceClaudeMD || p == fenceAgentsMD || p == fenceRouter || r.routed[p]
}

// isRelated reports whether p may ride with a guidance change without
// being guidance itself: the owned-output record and the outputs it
// lists, the guidance source, and configuration.
func (r fenceRevision) isRelated(p string) bool {
	switch {
	case p == "":
		return false
	case p == fenceOwnedRecord, p == fenceConfig, r.owned[p]:
		return true
	default:
		return strings.HasPrefix(p, fenceGuidanceSrc) && strings.HasSuffix(p, ".md")
	}
}

// revision reads the router and the owned-output record at rev. A file
// absent at rev contributes nothing.
func (cl *fenceClassifier) revision(rev string) (fenceRevision, error) {
	if r, ok := cl.revisions[rev]; ok {
		return r, nil
	}
	r := fenceRevision{routed: map[string]bool{}, owned: map[string]bool{}}
	rawOwned, err := cl.show(rev, fenceOwnedRecord)
	if err != nil { //coverage:ignore propagates a cat-file pump failure
		return r, err
	}
	var record map[string]string
	if json.Unmarshal([]byte(rawOwned), &record) == nil {
		for p := range record {
			r.owned[p] = true
		}
	}
	router, err := cl.show(rev, fenceRouter)
	if err != nil { //coverage:ignore propagates a cat-file pump failure
		return r, err
	}
	for _, p := range routedDocuments(router) {
		if !r.owned[p] {
			r.routed[p] = true
		}
	}
	cl.revisions[rev] = r
	return r, nil
}

// routedDocuments returns the repository-relative markdown documents the
// router links to. Links resolve against the router's own directory; a
// link with a scheme, or one leaving the repository, is not a route.
func routedDocuments(router string) []string {
	var out []string
	for _, m := range fenceRouterLink.FindAllStringSubmatch(router, -1) {
		target, _, _ := strings.Cut(m[1], "#")
		if strings.Contains(target, ":") || !strings.HasSuffix(target, ".md") {
			continue
		}
		p := path.Clean(path.Join(path.Dir(fenceRouter), target))
		if strings.HasPrefix(p, "../") {
			continue
		}
		out = append(out, p)
	}
	return out
}

// show returns a file's content at rev, or "" when it is absent there.
// The empty revision is a root commit's missing parent, where every file
// is absent.
func (cl *fenceClassifier) show(rev, p string) (string, error) {
	if rev == "" {
		return "", nil
	}
	content, err := cl.reader.Read(rev, p)
	if errors.Is(err, gitops.ErrBlobMissing) {
		return "", nil
	}
	if err != nil { //coverage:ignore a live cat-file pump fails only if the git subprocess dies mid-run
		return "", fmt.Errorf("reading %s at %s: %w", p, rev, err)
	}
	return string(content), nil
}

// handwrittenChanged reports whether a guidance pair's handwritten
// content differs across the commit. For a host entry point that is the
// file outside its managed blocks; any other guidance document is
// handwritten throughout, so any change to it — a rename included —
// counts.
func (cl *fenceClassifier) handwrittenChanged(parent, sha string, ch fenceChange) (bool, error) {
	if ch.Old != ch.New {
		return true, nil
	}
	before, err := cl.show(parent, ch.Old)
	if err != nil { //coverage:ignore propagates a cat-file pump failure
		return false, err
	}
	after, err := cl.show(sha, ch.New)
	if err != nil { //coverage:ignore propagates a cat-file pump failure
		return false, err
	}
	if ch.New == fenceClaudeMD || ch.New == fenceAgentsMD {
		return handwrittenText(before) != handwrittenText(after), nil
	}
	return before != after, nil
}

// handwrittenText is a host entry point with its managed blocks removed.
// A file whose markers are malformed has no block aiwf can own, so all of
// it is judged as handwritten.
func handwrittenText(content string) string {
	for _, markers := range [][3]string{fenceMarkers(initrepo.GuidanceMarkers), fenceMarkers(projectguidance.RouteMarkers)} {
		start, end, err := pathutil.ManagedBlockSpan(content, markers[0], markers[1], markers[2])
		if err != nil || start < 0 {
			continue
		}
		content = content[:start] + content[end:]
	}
	return content
}

func fenceMarkers(f func() (string, string, string)) [3]string {
	start, end, prefix := f()
	return [3]string{start, end, prefix}
}
