package stresstest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// verbenvelope.go — shared subprocess/verb-outcome helpers every
// scenario in this package depends on: the `--format=json`
// envelope-decoding machinery (CLAUDE.md "Output":
// {tool,version,status,findings,result,metadata}), first built for
// M-0241/AC-1's VerbSequenceScenario and reused by AC-2 through AC-5's
// scenarios; and processWasSignaled, first built for M-0242/AC-1's
// LockKillScenario and reused by AC-2's MidWriteKillScenario — both
// living in their own file rather than staying stranded in the
// single-AC file that first needed them, now that more than one
// scenario depends on each.

// verbEnvelope is the subset of the envelope this package reads.
// Different verbs populate different subsets of Result/Metadata;
// decoding the same shape for every verb is simpler than one type
// per verb since the unused fields just stay zero.
type verbEnvelope struct {
	Status   string                `json:"status"`
	Error    *verbEnvelopeError    `json:"error"`
	Findings []verbEnvelopeFinding `json:"findings"`
	Result   struct {
		Status string `json:"status"`
		Path   string `json:"path"`   // populated by `show` (M-0250/AC-2)
		Parent string `json:"parent"` // populated by `show` on a milestone (M-0250/AC-2)
		Scopes []struct {
			State string `json:"state"` // populated by `show`
		} `json:"scopes"`
	} `json:"result"`
	Metadata struct {
		EntityID string `json:"entity_id"`
		Entities int    `json:"entities"` // populated by `check`
		Events   int    `json:"events"`   // populated by `history`
		// CommitSHA is populated by a mutating verb that actually
		// committed, and absent from a verb that converged to a NoOp
		// (ADR-0036). It is the machine-readable difference between the
		// two, which are otherwise both status "ok" — letting an oracle
		// cross-check what actors reported against what git recorded
		// (M-0281).
		CommitSHA     string `json:"commit_sha"`
		CorrelationID string `json:"correlation_id"` // populated by every instrumented verb (M-0239/AC-1)
	} `json:"metadata"`
}

type verbEnvelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// envelopeErrorCode returns the envelope's error code. "" means the
// code is absent, which covers both an envelope carrying no error and
// an error whose code field is empty — the two are indistinguishable
// here by design, since a refusal a caller cannot name is one it
// cannot excuse either.
func envelopeErrorCode(env verbEnvelope) string {
	if env.Error == nil {
		return ""
	}
	return env.Error.Code
}

// isBusyRefusal reports whether an actor's error code is repolock's
// documented busy refusal.
//
// A contention scenario's actors race for one lock, so some of them
// being refused is the verb honoring its specification rather than
// breaching it. Telling that apart from any other failure is what
// keeps a contention oracle a statement about aiwf instead of about
// the machine's throughput, and it reads the envelope's code rather
// than its prose so a reworded message cannot silently reclassify a
// refusal.
func isBusyRefusal(errorCode string) bool {
	return errorCode == cliutil.CodeRepoLockBusy
}

type verbEnvelopeFinding struct {
	Code     string `json:"code"`
	Subcode  string `json:"subcode"`
	Severity string `json:"severity"`
	EntityID string `json:"entity_id"`
	Hint     string `json:"hint"`
}

// runAiwfJSON runs bin with args plus --format=json in dir and
// decodes the resulting envelope. A non-zero exit is expected
// traffic (an FSM refusal, a business-rule refusal) and is not
// itself an error — only a process that fails to even run, or
// output that isn't valid JSON, returns an error. Package-level
// (not a method) so every scenario in this package can point it at
// whichever directory it's driving — e.g. one of several sibling
// worktrees, not just the scenario's own single dir.
func runAiwfJSON(bin, dir string, args ...string) (verbEnvelope, error) {
	cmd := exec.Command(bin, append(args, "--format=json")...) //nolint:gosec // bin is a path this package's own BuildBinary just produced, not attacker-controlled input
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return verbEnvelope{}, fmt.Errorf("running aiwf %v: %w", args, err)
		}
	}
	return parseVerbEnvelope(args, out)
}

// parseVerbEnvelope decodes one --format=json invocation's stdout.
// Split out of runAiwfJSON so the malformed-output path is directly
// unit-testable without a real subprocess.
func parseVerbEnvelope(args []string, out []byte) (verbEnvelope, error) {
	var env verbEnvelope
	if err := json.Unmarshal(out, &env); err != nil {
		return verbEnvelope{}, fmt.Errorf("parsing aiwf %v JSON output: %w\n%s", args, err, out)
	}
	return env, nil
}

// listVerbEnvelope is `aiwf list`'s own envelope shape (M-0250/AC-3):
// unlike every other verb this package drives, list's `result` field
// is a top-level JSON array of rows, not an object — verbEnvelope's
// shared struct-typed Result can't decode it, hence this sibling type.
type listVerbEnvelope struct {
	Status   string                `json:"status"`
	Error    *verbEnvelopeError    `json:"error"`
	Findings []verbEnvelopeFinding `json:"findings"`
	Result   []listRow             `json:"result"`
}

// runAiwfListJSON runs `aiwf list --archived --format=json` in dir
// and decodes the resulting envelope. --archived widens list's
// default non-terminal-only view to every entity regardless of
// status, matching tree.Load's own unfiltered Entities set.
func runAiwfListJSON(bin, dir string) (listVerbEnvelope, error) {
	args := []string{"list", "--archived"}
	cmd := exec.Command(bin, append(args, "--format=json")...) //nolint:gosec // bin is a path this package's own BuildBinary just produced, not attacker-controlled input
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return listVerbEnvelope{}, fmt.Errorf("running aiwf %v: %w", args, err)
		}
	}
	return parseListVerbEnvelope(args, out)
}

// parseListVerbEnvelope decodes one `aiwf list --format=json`
// invocation's stdout. Split out of runAiwfListJSON so the
// malformed-output path is directly unit-testable without a real
// subprocess, mirroring parseVerbEnvelope's own split.
func parseListVerbEnvelope(args []string, out []byte) (listVerbEnvelope, error) {
	var env listVerbEnvelope
	if err := json.Unmarshal(out, &env); err != nil {
		return listVerbEnvelope{}, fmt.Errorf("parsing aiwf %v JSON output: %w\n%s", args, err, out)
	}
	return env, nil
}

// gitHeadCommitCount returns the number of commits reachable from
// HEAD in dir. A repository whose HEAD is still unborn — `git init`
// with nothing committed yet, which is the state a scenario's Setup
// leaves when it seeds no entities — counts zero rather than failing,
// since `git rev-list HEAD` rejects the unborn ref as a bad argument.
func gitHeadCommitCount(dir string) (int, error) {
	if err := runGit(dir, "rev-parse", "--verify", "--quiet", "HEAD"); err != nil {
		return 0, nil
	}
	cmd := exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil { //coverage:ignore defensive: git rev-list on a repo whose HEAD this function just verified has no realistic failure mode
		return 0, fmt.Errorf("counting commits in %s: %w", dir, err)
	}
	return parseCommitCount(out)
}

// parseCommitCount parses `git rev-list --count`'s stdout. Split out
// of gitHeadCommitCount so the malformed-output path is directly
// unit-testable without a real git subprocess.
func parseCommitCount(out []byte) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, fmt.Errorf("parsing commit count %q: %w", out, err)
	}
	return n, nil
}

// processWasSignaled reports whether waitErr represents a process
// that terminated because it received a signal (as SIGKILL does),
// rather than a normal exit.
func processWasSignaled(waitErr error) bool {
	var exitErr *exec.ExitError
	if !errors.As(waitErr, &exitErr) {
		return false
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok { //coverage:ignore defensive: syscall.WaitStatus is the concrete type exec.Cmd.ProcessState.Sys() returns on every unix platform this package targets
		return false
	}
	return status.Signaled()
}
