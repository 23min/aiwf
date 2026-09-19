package doctor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/skills"
)

// checkRitualsResult checks selected-host ritual files against the installer's
// rendered definitions. It returns each affected host and path with remediation.
func checkRitualsResult(rootDir string) (ok bool, message string, err error) {
	cfg, err := config.Load(rootDir)
	if err != nil && !errors.Is(err, config.ErrNotFound) {
		return false, "", err
	}
	selection, err := cfg.ResolveHosts(context.Background())
	if err != nil { //coverage:ignore config.Load already validates host names and context.Background cannot be canceled
		return false, "", err
	}
	var findings []string
	for _, host := range selection.Hosts {
		statuses, inspectErr := skills.InspectArtifacts(context.Background(), rootDir, hostTarget(host), configuredAgentTiers(cfg))
		if inspectErr != nil { //coverage:ignore built-in targets and immutable embedded sources render successfully; background context cannot be canceled
			return false, "", inspectErr
		}
		for _, status := range statuses {
			if status.Family != skills.FamilySkills && status.State != skills.ArtifactCurrent {
				findings = append(findings, fmt.Sprintf("%s %s %s (%s)", host, status.Family, status.Path, status.State))
			}
		}
	}
	if len(findings) == 0 {
		return true, "", nil
	}
	return false, fmt.Sprintf("%s under %s — run `aiwf update` to refresh", strings.Join(findings, "; "), rootDir), nil
}

// RunCheckRituals is the entry point for `aiwf doctor --check-rituals`:
// a terse, exit-code-meaningful check for automation (the M-0236
// worktree-materialization hook script), distinct from the full
// `aiwf doctor` report where a missing ritual is advisory-only and
// never affects the exit code. Silent and ExitOK when every ritual
// artifact matches its rendered definition; a single actionable stderr line and
// ExitFindings otherwise.
func RunCheckRituals(root string) int {
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil { //coverage:ignore ResolveRoot(--root) resolves via filepath.Abs and cannot fail here; defensive parity with Run
		cliutil.Errorf("aiwf doctor --check-rituals: %v\n", err)
		return cliutil.ExitUsage
	}
	ok, message, err := checkRitualsResult(rootDir)
	if err != nil {
		cliutil.Errorf("aiwf doctor --check-rituals: %v\n", err)
		return cliutil.ExitInternal
	}
	if ok {
		return cliutil.ExitOK
	}
	cliutil.Errorln("aiwf doctor --check-rituals: " + message)
	return cliutil.ExitFindings
}
