package check

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	codespkg "github.com/23min/aiwf/internal/codes"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/projectguidance"
)

// The instruction-file fence judges commits that change a project's
// handwritten instruction content — the root CLAUDE.md and AGENTS.md
// outside aiwf's managed blocks, the project router, and the documents the
// router links to (ADR-0053). Such an edit is its own commit: alongside
// it a commit may carry only the other instruction files, the guidance
// outputs aiwf owns and their record, and aiwf.yaml.
//
// A change confined to a managed block is generated output and is not
// judged. A merge commit is not judged: the commits it brings in are in
// the audited range and are judged on their own — except those already on
// the trunk, which have been pushed and were judged when they reached it.

// CodeInstructionCommitUnrelatedFile is reported once per file a commit
// carries beside a handwritten instruction change that is not one of its
// allowed companions.
var CodeInstructionCommitUnrelatedFile = codespkg.Code{
	ID:    "instruction-commit-unrelated-file",
	Class: codespkg.ClassStructural,
}

// InstructionCommit is one commit in the audited range, reduced to what the
// fence judges. Pre-computed by WalkInstructionCommits; consumed by
// RunInstructionFence.
type InstructionCommit struct {
	SHA string
	// Entity is the commit's first aiwf-entity trailer value.
	Entity string
	// Body is the commit message.
	Body string
	// Changed lists the instruction files whose handwritten content the
	// commit changed.
	Changed []string
	// Unrelated lists the files the commit changed that are neither
	// instruction files nor their allowed companions.
	Unrelated []string
}

// RunInstructionFence maps the walked commits to findings. A commit that
// changed no handwritten instruction content is outside the fence.
func RunInstructionFence(commits []InstructionCommit) []Finding {
	var out []Finding
	for _, c := range commits {
		if len(c.Changed) == 0 {
			continue
		}
		for _, p := range c.Unrelated {
			out = append(out, Finding{
				Code:     CodeInstructionCommitUnrelatedFile.ID,
				Severity: SeverityError,
				Message:  fmt.Sprintf("commit %s changes handwritten instructions (%s) and also %s, which is neither an instruction file, a guidance output aiwf owns, nor %s", shortHash(c.SHA), strings.Join(c.Changed, ", "), p, config.FileName),
				Path:     p,
			})
		}
	}
	return out
}

// instructionChange is one changed path pair from `git log --name-status`.
// Old is empty for an addition and New for a deletion; both are set, and
// equal, for a modification.
type instructionChange struct {
	Old, New string
}

// instructionLogRecord is one commit read from the NUL-framed log.
type instructionLogRecord struct {
	sha, parent, entity, body string
	changes                   []instructionChange
}

// WalkInstructionCommits lists the commits in rangeArg that are not on the
// trunk ref, with their changed paths, and classifies each against the
// instruction set as it stood at the commit and at its first parent. An
// empty trunk excludes nothing.
//
// The log is NUL-framed: git refuses a NUL byte in a commit message, and -z
// turns off path quoting, so no message and no path can be mistaken for
// framing. Renames are detected so a moved document reports as one pair;
// --root diffs a root commit against the empty tree whatever log.showRoot
// says, and --no-show-signature keeps a signature check out of the stream
// whatever log.showSignature says. A merge commit shows no diff, so it
// reaches the rule with nothing to judge. Content is read through one
// cat-file pump.
func WalkInstructionCommits(ctx context.Context, root, rangeArg, trunk string) ([]InstructionCommit, error) {
	args := []string{
		"log", "-z", "--root", "-M", "--no-show-signature",
		"--format=%H %P%x00%(trailers:key=" + gitops.TrailerEntity + ",valueonly,unfold,separator=%x0a)%x00%B%x00",
		"--name-status", rangeArg,
	}
	if trunk != "" {
		args = append(args, "--not", trunk)
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	// Stdout only: a warning git prints must not reach the parsed stream.
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git log %s: %w\n%s", rangeArg, err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git log %s: %w", rangeArg, err) //coverage:ignore git was found and started for the rev-parse calls that resolved the range; only an exit status reaches here
	}
	recs, err := parseInstructionLog(string(out))
	if err != nil { //coverage:ignore the invocation fixes the format, -z framing and signature display, so a stream it cannot frame is a change in git itself, reported rather than skipped
		return nil, err
	}
	reader, err := gitops.NewBlobReader(ctx, root)
	if err != nil { //coverage:ignore git log just ran in root, so the cat-file pump starts there too
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	var commits []InstructionCommit
	for _, rec := range recs {
		c, err := classifyInstructionCommit(reader, rec)
		if err != nil { //coverage:ignore propagates a cat-file read failure: the walk reads only paths the diff reports present at that revision, and the router and owned record, whose absence reads as ErrBlobMissing, so only a pump that dies mid-run fails one
			return nil, err
		}
		commits = append(commits, c)
	}
	return commits, nil
}

var (
	instructionHeaderRE = regexp.MustCompile(`^[0-9a-f]{40,64}( [0-9a-f]{40,64})*$`)
	instructionStatusRE = regexp.MustCompile(`^[ACDMRTUXB]\d*$`)
)

// parseInstructionLog reads the NUL-framed log: each commit is its header —
// the hash and parent hashes — its aiwf-entity trailer values, one per
// line, and its message, then --name-status fields. A status is followed by
// one path, or two for a rename or a copy. Fields are read by position, so
// a path or a message that looks like a header or a status is never taken
// for one. A field where a header belongs that is not one means the stream
// is not the one asked for, and is an error rather than a skip.
func parseInstructionLog(out string) ([]instructionLogRecord, error) {
	tok := strings.Split(out, "\x00")
	field := func(i int) string {
		if i < len(tok) {
			return tok[i]
		}
		return ""
	}
	var recs []instructionLogRecord
	for i := 0; i < len(tok); {
		// A root commit's parent list is empty, leaving a trailing space.
		head := strings.TrimSpace(tok[i])
		if head == "" {
			i++
			continue
		}
		if !instructionHeaderRE.MatchString(head) {
			return nil, fmt.Errorf("git log: expected a commit header, read %q", head)
		}
		revs := strings.Fields(head)
		rec := instructionLogRecord{sha: revs[0], body: field(i + 2)}
		// The first parent is the side a commit's diff is taken against; a
		// root commit has none, and reads as an empty tree.
		if len(revs) > 1 {
			rec.parent = revs[1]
		}
		// A second aiwf-entity trailer would not change the verdict for the
		// first, which names the owner.
		rec.entity, _, _ = strings.Cut(field(i+1), "\n")
		i += 3
		for i < len(tok) {
			status := strings.TrimLeft(tok[i], "\n")
			if status == "" {
				i++
				continue
			}
			if !instructionStatusRE.MatchString(status) {
				break
			}
			switch status[0] {
			case 'R', 'C':
				rec.changes = append(rec.changes, instructionChange{Old: field(i + 1), New: field(i + 2)})
				i += 3
			case 'A':
				rec.changes = append(rec.changes, instructionChange{New: field(i + 1)})
				i += 2
			case 'D':
				rec.changes = append(rec.changes, instructionChange{Old: field(i + 1)})
				i += 2
			default:
				rec.changes = append(rec.changes, instructionChange{Old: field(i + 1), New: field(i + 1)})
				i += 2
			}
		}
		recs = append(recs, rec)
	}
	return recs, nil
}

// instructionRevision is the instruction set and the owned outputs as they
// stood at one revision.
type instructionRevision struct {
	routed map[string]bool
	owned  map[string]bool
}

// classifyInstructionCommit decides, for each changed pair, whether it is a
// handwritten instruction change, an allowed companion, or unrelated.
func classifyInstructionCommit(br *gitops.BlobReader, rec instructionLogRecord) (InstructionCommit, error) {
	c := InstructionCommit{SHA: rec.sha, Entity: rec.entity, Body: rec.body}
	before, err := instructionRevisionAt(br, rec.parent)
	if err != nil { //coverage:ignore propagates a cat-file read failure, as WalkInstructionCommits records
		return c, err
	}
	after, err := instructionRevisionAt(br, rec.sha)
	if err != nil { //coverage:ignore propagates a cat-file read failure, as WalkInstructionCommits records
		return c, err
	}
	for _, ch := range rec.changes {
		instruction := before.isInstruction(ch.Old) || after.isInstruction(ch.New)
		if instruction {
			changed, err := handwrittenChanged(br, rec.parent, rec.sha, ch)
			if err != nil { //coverage:ignore propagates a cat-file read failure, as WalkInstructionCommits records
				return c, err
			}
			if changed {
				c.Changed = append(c.Changed, instructionReportPath(ch))
			}
			continue
		}
		if !companionPair(before, after, ch) {
			c.Unrelated = append(c.Unrelated, instructionReportPath(ch))
		}
	}
	return c, nil
}

// companionPair reports whether a change that is not an instruction file
// may ride with an instruction change: every side the change has is a
// companion where it stands, so an owned file cannot be moved out to carry
// unrelated content.
func companionPair(before, after instructionRevision, ch instructionChange) bool {
	return (ch.Old == "" || before.isCompanion(ch.Old)) && (ch.New == "" || after.isCompanion(ch.New))
}

// instructionReportPath names a pair by the path that exists after the
// commit, or the removed path for a deletion.
func instructionReportPath(ch instructionChange) string {
	if ch.New != "" {
		return ch.New
	}
	return ch.Old
}

func (r instructionRevision) isInstruction(p string) bool {
	return p == projectguidance.ClaudeFile || p == projectguidance.AgentsFile || p == projectguidance.RouterFile || r.routed[p]
}

// isCompanion reports whether p may ride with an instruction change without
// being instruction content: the owned-output record, the outputs it lists,
// and aiwf.yaml.
func (r instructionRevision) isCompanion(p string) bool {
	return p == projectguidance.OwnedRecordFile || p == config.FileName || r.owned[p]
}

// instructionRevisionAt reads the router and the owned-output record at rev.
// A file absent at rev contributes nothing.
func instructionRevisionAt(br *gitops.BlobReader, rev string) (instructionRevision, error) {
	r := instructionRevision{routed: map[string]bool{}, owned: map[string]bool{}}
	rawOwned, err := blobAt(br, rev, projectguidance.OwnedRecordFile)
	if err != nil { //coverage:ignore propagates a cat-file read failure, as WalkInstructionCommits records
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
	router, err := blobAt(br, rev, projectguidance.RouterFile)
	if err != nil { //coverage:ignore propagates a cat-file read failure, as WalkInstructionCommits records
		return r, err
	}
	for _, p := range projectguidance.RoutedDocuments(router) {
		if !r.owned[p] {
			r.routed[p] = true
		}
	}
	return r, nil
}

// blobAt returns a file's content at rev, or "" when it is absent there.
// The empty revision is a root commit's missing parent, where every file is
// absent, and the empty path is the missing side of an addition or a
// deletion — which must not reach cat-file, where `rev:` names the tree.
func blobAt(br *gitops.BlobReader, rev, p string) (string, error) {
	if rev == "" || p == "" {
		return "", nil
	}
	content, err := br.Read(rev, p)
	if errors.Is(err, gitops.ErrBlobMissing) {
		return "", nil
	}
	if err != nil { //coverage:ignore propagates a cat-file read failure, as WalkInstructionCommits records
		return "", fmt.Errorf("reading %s at %s: %w", p, rev, err)
	}
	return string(content), nil
}

// handwrittenChanged reports whether an instruction pair's handwritten
// content differs across the commit. For a host instruction file the
// handwritten content is the file outside its managed blocks; any other
// instruction document is handwritten throughout, so any change to it — a
// rename included — counts.
func handwrittenChanged(br *gitops.BlobReader, parent, sha string, ch instructionChange) (bool, error) {
	before, err := blobAt(br, parent, ch.Old)
	if err != nil { //coverage:ignore propagates a cat-file read failure, as WalkInstructionCommits records
		return false, err
	}
	after, err := blobAt(br, sha, ch.New)
	if err != nil { //coverage:ignore propagates a cat-file read failure, as WalkInstructionCommits records
		return false, err
	}
	if ch.New == projectguidance.ClaudeFile || ch.New == projectguidance.AgentsFile {
		// aiwf separates a block it inserts with a blank line outside it,
		// so blank lines are not handwritten content.
		before, after = nonBlankLines(projectguidance.HandwrittenText(before)), nonBlankLines(projectguidance.HandwrittenText(after))
	}
	// A rename moves the document; an addition or a deletion is judged by
	// its content, so a host file aiwf creates holding only its blocks is
	// not a handwritten change.
	renamed := ch.Old != "" && ch.New != "" && ch.Old != ch.New
	return renamed || before != after, nil
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
