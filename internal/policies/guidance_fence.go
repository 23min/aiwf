package policies

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/23min/aiwf/internal/entity"
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
// configuration; it names the entity it belongs to in an aiwf-entity
// trailer that resolves in the tree; and when it removes a handwritten
// line — a rewording included — its message records a disposition block.
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
	fenceClaudeMD    = "CLAUDE.md"
	fenceAgentsMD    = "AGENTS.md"
	fenceRouter      = ".guidance/project.md"
	fenceOwnedRecord = ".guidance/.aiwf-owned"
	fenceConfig      = "aiwf.yaml"
	fenceGuidanceSrc = "internal/skills/embedded-guidance/"
)

// fenceCommit is one commit in the audited range, reduced to what the
// fence judges: which handwritten guidance files it changed and which
// changed files are unrelated to guidance.
type fenceCommit struct {
	SHA       string
	Entity    string
	Guidance  []string
	Unrelated []string
	Removes   bool
	Body      string
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
	if !anyGuidanceCommit(commits) {
		return nil, nil
	}
	resolves, err := entityResolver(root)
	if err != nil {
		return nil, err
	}
	return detectGuidanceFence(commits, resolves), nil
}

func anyGuidanceCommit(commits []fenceCommit) bool {
	for _, c := range commits {
		if len(c.Guidance) > 0 {
			return true
		}
	}
	return false
}

// detectGuidanceFence is the pure core. A commit that changed no
// handwritten guidance is outside the fence. One that did must name the
// entity it belongs to in an aiwf-entity trailer that resolves, and is
// refused once per unrelated file it also carries. resolves is handed an
// id already rolled up to its owner and canonicalized, so a composite id
// is owned by its milestone and a narrow legacy id names the same entity.
func detectGuidanceFence(commits []fenceCommit, resolves func(string) bool) []Violation {
	var out []Violation
	for _, c := range commits {
		if len(c.Guidance) == 0 {
			continue
		}
		id := strings.TrimSpace(c.Entity)
		switch {
		case id == "":
			out = append(out, Violation{
				Policy: "guidance-fence",
				File:   c.Guidance[0],
				Detail: fmt.Sprintf("commit %s changes handwritten guidance (%s) but carries no aiwf-entity: trailer, so nothing records which entity the change belongs to; re-commit naming it (`--trailer \"aiwf-entity: <id>\"`).", shortSkillSHA(c.SHA), strings.Join(c.Guidance, ", ")),
			})
		case !resolves(entity.Canonicalize(entity.CompositeRoot(id))):
			out = append(out, Violation{
				Policy: "guidance-fence",
				File:   c.Guidance[0],
				Detail: fmt.Sprintf("commit %s changes handwritten guidance (%s) under aiwf-entity: %s, which resolves to no entity in the tree; name an entity that exists.", shortSkillSHA(c.SHA), strings.Join(c.Guidance, ", "), id),
			})
		}
		if c.Removes {
			good, bad := dispositionBlocks(c.Body)
			switch {
			case len(bad) > 0:
				out = append(out, Violation{
					Policy: "guidance-fence",
					File:   c.Guidance[0],
					Detail: fmt.Sprintf("commit %s removes handwritten guidance and carries a malformed disposition block (%s); a block is a Removed: line followed by Disposition: copy of <path>, relocated to <path>, pointer to <id>, or deleted.", shortSkillSHA(c.SHA), strings.Join(bad, "; ")),
				})
			case good == 0:
				out = append(out, Violation{
					Policy: "guidance-fence",
					File:   c.Guidance[0],
					Detail: fmt.Sprintf("commit %s removes handwritten guidance (%s) but its message carries no disposition block; add a Removed: line followed by Disposition: copy of <path>, relocated to <path>, pointer to <id>, or deleted.", shortSkillSHA(c.SHA), strings.Join(c.Guidance, ", ")),
				})
			}
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
// The log is NUL-framed: git refuses a NUL byte in a commit message, and
// -z turns off path quoting, so no message and no path can be mistaken for
// framing. Renames are detected so a moved guidance document reports as
// one pair; --root diffs a root commit against the empty tree whatever
// log.showRoot says. Every guidance path is a markdown file, so a commit
// changing none is outside the fence without reading anything; the rest
// read their content through one cat-file pump.
func fenceCommitsInRange(root, baseRef string) ([]fenceCommit, error) {
	cmd := exec.Command("git", "-c", "diff.renames=true", "log", "-z", "--root", "-M",
		"--format=%H %P%x00%(trailers:key="+gitops.TrailerEntity+",valueonly,unfold,separator=%x0a)%x00%B%x00",
		"--name-status", baseRef+"..HEAD")
	cmd.Dir = root
	// Stdout only: a warning git prints must not reach the parsed stream.
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log %s..HEAD in %s: %w\n%s", baseRef, root, err, stderrOf(err))
	}
	reader, err := gitops.NewBlobReader(context.Background(), root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	cl := &fenceClassifier{reader: reader, revisions: map[string]fenceRevision{}}
	var commits []fenceCommit
	for _, rec := range parseFenceLog(string(out)) {
		if !touchesMarkdown(rec.changes) {
			continue
		}
		c, err := cl.classify(rec.sha, rec.parent, rec.changes)
		if err != nil { //coverage:ignore propagates a cat-file read failure; every path read exists at the revision it is read at, so only a pump that dies mid-run fails one
			return nil, err
		}
		c.Entity, c.Body = rec.entity, rec.body
		commits = append(commits, c)
	}
	return commits, nil
}

// stderrOf returns what a failed command wrote to stderr.
func stderrOf(err error) []byte {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Stderr
	}
	return nil
}

func touchesMarkdown(changes []fenceChange) bool {
	for _, ch := range changes {
		if strings.HasSuffix(ch.Old, ".md") || strings.HasSuffix(ch.New, ".md") {
			return true
		}
	}
	return false
}

// fenceLogRecord is one commit read from the NUL-framed log.
type fenceLogRecord struct {
	sha, parent, entity, body string
	changes                   []fenceChange
}

var (
	fenceHeaderRE = regexp.MustCompile(`^[0-9a-f]{40,64}( [0-9a-f]{40,64})*$`)
	fenceStatusRE = regexp.MustCompile(`^[ACDMRTUXB]\d*$`)
)

// parseFenceLog reads the NUL-framed log: each commit is its header — the
// hash and parent hashes — its aiwf-entity trailer values, one per line,
// and its message, then --name-status fields. A status is followed by one
// path, or two for a rename or a copy. Fields are read by position, so a
// path or a message that looks like a header or a status is never taken
// for one.
func parseFenceLog(out string) []fenceLogRecord {
	tok := strings.Split(out, "\x00")
	field := func(i int) string {
		if i < len(tok) {
			return tok[i]
		}
		return ""
	}
	var recs []fenceLogRecord
	for i := 0; i < len(tok); {
		// A root commit's parent list is empty, leaving a trailing space.
		head := strings.TrimSpace(tok[i])
		if !fenceHeaderRE.MatchString(head) {
			i++
			continue
		}
		revs := strings.Fields(head)
		rec := fenceLogRecord{sha: revs[0], body: field(i + 2)}
		// The first parent is the side a commit's diff is taken against;
		// a root commit has none, and reads as an empty tree.
		if len(revs) > 1 {
			rec.parent = revs[1]
		}
		// A second aiwf-entity trailer would not change the verdict for
		// the first, which names the owner.
		rec.entity, _, _ = strings.Cut(field(i+1), "\n")
		rec.entity = strings.TrimSpace(rec.entity)
		i += 3
		for i < len(tok) {
			status := strings.TrimLeft(tok[i], "\n")
			if status == "" {
				i++
				continue
			}
			if !fenceStatusRE.MatchString(status) {
				break
			}
			switch status[0] {
			case 'R', 'C':
				rec.changes = append(rec.changes, fenceChange{Old: field(i + 1), New: field(i + 2)})
				i += 3
			case 'A':
				rec.changes = append(rec.changes, fenceChange{New: field(i + 1)})
				i += 2
			case 'D':
				rec.changes = append(rec.changes, fenceChange{Old: field(i + 1)})
				i += 2
			default:
				rec.changes = append(rec.changes, fenceChange{Old: field(i + 1), New: field(i + 1)})
				i += 2
			}
		}
		recs = append(recs, rec)
	}
	return recs
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
			changed, removes, err := cl.handwrittenChanged(parent, sha, ch)
			if err != nil { //coverage:ignore propagates a cat-file pump failure
				return c, err
			}
			if changed {
				c.Guidance = append(c.Guidance, fenceReportPath(ch))
				c.Removes = c.Removes || removes
				continue
			}
		}
		if guidance || relatedPair(before, after, ch) {
			continue
		}
		c.Unrelated = append(c.Unrelated, fenceReportPath(ch))
	}
	sort.Strings(c.Unrelated)
	return c, nil
}

// relatedPair reports whether a non-guidance change may ride with a
// guidance change. A rename is related only when both its sides are, so
// an owned file cannot be moved out to carry unrelated content.
func relatedPair(before, after fenceRevision, ch fenceChange) bool {
	switch {
	case ch.Old == "":
		return after.isRelated(ch.New)
	case ch.New == "":
		return before.isRelated(ch.Old)
	case ch.Old == ch.New:
		return before.isRelated(ch.Old) || after.isRelated(ch.New)
	default:
		return before.isRelated(ch.Old) && after.isRelated(ch.New)
	}
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
		// Only a path shaped like one aiwf owns counts, so a commit cannot
		// list a file of its own in the record to carry it.
		for p := range record {
			if projectguidance.ValidOwnedPath(p) {
				r.owned[p] = true
			}
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
// router links to, resolved as resolveReference describes.
func routedDocuments(router string) []string {
	var out []string
	for _, target := range markdownLinks(router) {
		if p, ok := resolveReference(fenceRouter, target); ok && strings.HasSuffix(p, ".md") {
			out = append(out, p)
		}
	}
	return out
}

// show returns a file's content at rev, or "" when it is absent there.
// The empty revision is a root commit's missing parent, where every file
// is absent, and the empty path is the missing side of an addition or a
// deletion — which must not reach cat-file, where `rev:` names the tree.
func (cl *fenceClassifier) show(rev, p string) (string, error) {
	if rev == "" || p == "" {
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
// content differs across the commit, and whether the change removes a
// line. For a host entry point the handwritten content is the file
// outside its managed blocks; any other guidance document is handwritten
// throughout, so any change to it — a rename included — counts.
func (cl *fenceClassifier) handwrittenChanged(parent, sha string, ch fenceChange) (changed, removes bool, err error) {
	before, err := cl.show(parent, ch.Old)
	if err != nil { //coverage:ignore propagates a cat-file pump failure
		return false, false, err
	}
	after, err := cl.show(sha, ch.New)
	if err != nil { //coverage:ignore propagates a cat-file pump failure
		return false, false, err
	}
	if ch.New == fenceClaudeMD || ch.New == fenceAgentsMD {
		// aiwf separates a block it inserts with a blank line outside it,
		// so blank lines are not handwritten content.
		before, after = nonBlankLines(handwrittenText(before)), nonBlankLines(handwrittenText(after))
	}
	return ch.Old != ch.New || before != after, removesLine(before, after), nil
}

// nonBlankLines drops blank lines from text.
func nonBlankLines(text string) string {
	var kept []string
	for _, l := range strings.Split(text, "\n") {
		if strings.TrimSpace(l) != "" {
			kept = append(kept, strings.TrimRight(l, "\r"))
		}
	}
	return strings.Join(kept, "\n")
}

// removesLine reports whether some non-blank line of before is missing
// from after, counting repeats. A rewording removes its old line; a line
// moved within the file removes nothing.
func removesLine(before, after string) bool {
	kept := map[string]int{}
	for _, l := range strings.Split(after, "\n") {
		kept[strings.TrimSpace(l)]++
	}
	for _, l := range strings.Split(before, "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if kept[l] == 0 {
			return true
		}
		kept[l]--
	}
	return false
}

// Disposition-block grammar. A block is a Removed: line immediately
// followed by a Disposition: line; the value is one of the closed set.
const (
	fenceRemovedKey     = "Removed:"
	fenceDispositionKey = "Disposition:"
)

var fenceDispositionForms = []string{"copy of ", "relocated to ", "pointer to "}

// dispositionBlocks scans a commit message for disposition blocks and
// returns how many are well-formed and the malformed ones. It checks
// shape only: whether a named path or id exists, and whether every
// removed passage has its block, is held at review.
func dispositionBlocks(body string) (wellFormed int, malformed []string) {
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		if !strings.HasPrefix(strings.TrimSpace(l), fenceRemovedKey) {
			continue
		}
		next := ""
		if i+1 < len(lines) {
			next = strings.TrimSpace(lines[i+1])
		}
		value, ok := strings.CutPrefix(next, fenceDispositionKey)
		switch {
		case !ok:
			malformed = append(malformed, strings.TrimSpace(l)+" (no Disposition: line follows)")
		case validDisposition(strings.TrimSpace(value)):
			wellFormed++
		default:
			malformed = append(malformed, next)
		}
	}
	return wellFormed, malformed
}

func validDisposition(v string) bool {
	if v == "deleted" {
		return true
	}
	for _, form := range fenceDispositionForms {
		if target, ok := strings.CutPrefix(v, form); ok && strings.TrimSpace(target) != "" {
			return true
		}
	}
	return false
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
