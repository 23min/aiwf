package policies

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
)

// PolicyCoauthorTrailerBan is the diff-scoped backstop over the
// `Co-Authored-By:` refusal the commit-msg hook enforces at composition
// (G-0254).
//
// The hook is the timely catch and the one an operator meets, but it runs
// only where it is installed and only as the `aiwf` it shells. Three
// routes reach a landed commit past it: a clone where `aiwf init` has not
// wired the hook, a `--no-verify` commit, and a hook shelling a release
// older than the refusal. This policy judges the commits themselves, so
// all three are covered by the one check.
//
// Which addresses are refused comes from the audited repo's own
// aiwf.yaml (`provenance.refuse_coauthors`), the same source the hook
// reads — a repo that names none is untouched, and the two layers cannot
// disagree about what the policy is.
//
// It is a Go policy test (CI tier) rather than an `aiwf check` rule
// because of where each one runs. `aiwf check` judges commit ranges
// elsewhere — RunProvenanceCheck and RunUntrailedAudit both do — but it
// reaches this repo's own history only through the pre-push hook, which
// shells whatever `aiwf` is installed. A policy test is compiled from the
// tree under test, so it is the layer a stale binary cannot defeat, which
// is the failure this backstop exists for.
//
// Like PolicySkillEditProvenanceBackstop it is diff-scoped and reads its
// base ref from AIWF_COVERAGE_BASE, so it keeps the uniform
// `func(root) ([]Violation, error)` shape the runPolicy harness drives.
// An empty or all-zero base means "no comparison point" and the audit
// no-ops; the authoritative invocations are the CI coverage-gate step and
// `make coverage-gate`, both of which set it.
func PolicyCoauthorTrailerBan(root string) ([]Violation, error) {
	base := strings.TrimSpace(os.Getenv("AIWF_COVERAGE_BASE"))
	return coauthorTrailerViolations(root, base)
}

// coauthorTrailerViolations is the testable IO core: it resolves the
// refused set from the repo's config, walks the audited range, and
// delegates the per-commit verdict to detectRefusedCoauthors.
func coauthorTrailerViolations(root, baseRef string) ([]Violation, error) {
	baseRef = strings.TrimSpace(baseRef)
	if baseRef == "" || baseRef == zeroSHA {
		return nil, nil
	}
	cfg, err := config.Load(root)
	if err != nil {
		// A repo with no readable aiwf.yaml has declared nothing to
		// refuse. Config faults are reported by the verbs that need
		// config; this policy has nothing to judge without a list.
		return nil, nil
	}
	refused := cfg.RefusedCoauthors()
	if len(refused) == 0 {
		return nil, nil
	}
	commits, err := coauthorCommitsInRange(root, baseRef)
	if err != nil {
		return nil, err
	}
	return detectRefusedCoauthors(commits, refused), nil
}

// coauthorCommit is one commit in the audited range: its SHA, its subject
// for the operator to recognize it by, and the trailer block git parsed
// from its message.
type coauthorCommit struct {
	SHA     string
	Subject string
	Trailer string
}

// coauthorFldSep separates the fields of the `git log` scan. Git refuses a
// NUL byte in a commit log message, so no subject or trailer can contain
// one — which is what makes it the one separator a message cannot forge.
// A printable or control-character separator can appear in a subject, and
// a subject carrying it shifts every field after it, so the scan reads a
// trailer that is not there and reports the commit clean.
const coauthorFldSep = "\x00"

// coauthorFldSepFormat is how the separator is spelled inside git's
// --format argument. It cannot be the byte itself: an argv string is
// NUL-terminated, so passing one truncates the argument.
const coauthorFldSepFormat = "%x00"

// coauthorCommitsInRange returns every commit between baseRef and HEAD
// with the trailer block git parsed from its message.
//
// The whole block is requested rather than `%(trailers:key=...)` so the
// key comparison stays in RefusedCoauthorIn, which both this policy and
// the commit-msg hook read. Asking git to filter by key would put a
// second, differently-spelled matcher in the path, and the two would
// answer differently the day one of them changed.
//
// `unfold` is what makes that sharing real rather than nominal. A trailer
// value may be continued on an indented line, and the hook reads its block
// through `git interpret-trailers --parse`, which joins the continuation
// back onto its key. `%(trailers)` alone does not, and the shared matcher
// is line-based — so without unfolding, a wrapped address is refused at
// composition and passed here.
func coauthorCommitsInRange(root, baseRef string) ([]coauthorCommit, error) {
	format := "%H" + coauthorFldSepFormat + "%s" + coauthorFldSepFormat +
		"%(trailers:unfold=true)" + coauthorFldSepFormat
	cmd := exec.Command("git", "log", "--format="+format, baseRef+"..HEAD")
	cmd.Dir = root
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// The whole argv, not just the subcommand: an unresolvable base
		// ref is the realistic failure and it is only visible there.
		return nil, fmt.Errorf("git log %s..HEAD in %s: %w\n%s", baseRef, root, err, stderr.String())
	}
	// stderr is kept out of the parsed text: a warning git writes there
	// would otherwise be read as log records.
	return parseCoauthorLog(stdout.String())
}

// parseCoauthorLog turns the `git log` scan's output into coauthorCommits.
// Each commit contributes three separator-terminated fields — sha, subject,
// trailer block — and the block may be empty or run to several lines.
//
// A field count that is not a multiple of three means the output is not what
// the format asked for, which no commit message can cause. It is reported
// rather than skipped: skipping would drop commits from a scan whose whole
// purpose is to find one, and report the range clean.
func parseCoauthorLog(out string) ([]coauthorCommit, error) {
	fields := strings.Split(out, coauthorFldSep)
	// git writes a newline after each formatted record, so the final split
	// element holds that newline and nothing else.
	if last := len(fields) - 1; last >= 0 && strings.TrimSpace(fields[last]) == "" {
		fields = fields[:last]
	}
	if len(fields)%3 != 0 {
		return nil, fmt.Errorf("unparseable git log output: got %d fields, which is not a whole number of "+
			"(sha, subject, trailers) records", len(fields))
	}
	var commits []coauthorCommit
	for i := 0; i < len(fields); i += 3 {
		commits = append(commits, coauthorCommit{
			SHA:     strings.TrimSpace(fields[i]),
			Subject: strings.TrimSpace(fields[i+1]),
			Trailer: fields[i+2],
		})
	}
	return commits, nil
}

// detectRefusedCoauthors is the pure core. A commit violates when its
// trailer block names an address the repo refuses. Order follows the
// commits it is handed, which `git log` yields newest first — the order an
// operator reads the range in.
func detectRefusedCoauthors(commits []coauthorCommit, refused map[string]bool) []Violation {
	var out []Violation
	for _, c := range commits {
		addr := gitops.RefusedCoauthorIn(c.Trailer, refused)
		if addr == "" {
			continue
		}
		out = append(out, Violation{
			Policy: "coauthor-trailer-ban",
			File:   c.SHA,
			Detail: fmt.Sprintf(
				"commit %s (%s) carries a Co-Authored-By trailer naming %s, "+
					"which aiwf.yaml lists under provenance.refuse_coauthors; "+
					"rewrite the message to drop the line, or remove the address from that list",
				shortSHA(c.SHA), c.Subject, addr),
		})
	}
	return out
}

// shortSHA abbreviates a full SHA for the operator-facing message, and
// returns a shorter value unchanged so a fixture SHA reads as written.
func shortSHA(sha string) string {
	if len(sha) <= 9 {
		return sha
	}
	return sha[:9]
}
