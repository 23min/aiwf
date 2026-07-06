package doctor

import (
	"fmt"
	"os"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/skills"
)

// checkRitualsResult is the pure computation behind `aiwf doctor
// --check-rituals`: whether every ritual artifact (skills/agents/
// templates) is materialized under rootDir, reusing the same
// skills.MaterializedRituals data appendMaterializedRitualsReport
// already surfaces advisory-only. message is empty when ok is true;
// otherwise it names the count and points at `aiwf update`, for a
// caller (RunCheckRituals, and transitively the M-0236 worktree hook
// script) to print verbatim.
func checkRitualsResult(rootDir string) (ok bool, message string, err error) {
	present, missing, err := skills.MaterializedRituals(rootDir, skills.ClaudeTarget)
	if err != nil { //coverage:ignore MaterializedRituals errors only when the compiled-in embed FS walk fails; unreachable at runtime, so tempdir tests cannot reach this arm
		return false, "", err
	}
	if len(missing) == 0 {
		return true, "", nil
	}
	return false, fmt.Sprintf("%d of %d ritual artifacts not materialized under %s — run `aiwf update` to refresh",
		len(missing), len(present)+len(missing), rootDir), nil
}

// RunCheckRituals is the entry point for `aiwf doctor --check-rituals`:
// a terse, exit-code-meaningful check for automation (the M-0236
// worktree-materialization hook script), distinct from the full
// `aiwf doctor` report where a missing ritual is advisory-only and
// never affects the exit code. Silent and ExitOK when every ritual
// artifact is present; a single actionable stderr line and
// ExitFindings otherwise.
func RunCheckRituals(root string) int {
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil { //coverage:ignore ResolveRoot(--root) resolves via filepath.Abs and cannot fail here; defensive parity with Run
		fmt.Fprintf(os.Stderr, "aiwf doctor --check-rituals: %v\n", err)
		return cliutil.ExitUsage
	}
	ok, message, err := checkRitualsResult(rootDir)
	if err != nil { //coverage:ignore MaterializedRituals errors only when the compiled-in embed FS walk fails; unreachable at runtime, so tempdir tests cannot reach this arm
		fmt.Fprintf(os.Stderr, "aiwf doctor --check-rituals: %v\n", err)
		return cliutil.ExitInternal
	}
	if ok {
		return cliutil.ExitOK
	}
	fmt.Fprintln(os.Stderr, "aiwf doctor --check-rituals: "+message)
	return cliutil.ExitFindings
}
