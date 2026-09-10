package policies

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// changelog_audit_wiring_test.go — M-0330/AC-4.
//
// The audit only gates a release if three things line up: the release
// workflow runs a make target, that target runs the audit's entry-point
// test, and the job it runs in has the history the base resolution
// needs. Each is asserted against the artefact that carries it, so
// renaming or deleting either side of a link reports here rather than at
// the next release.

// changelogAuditTarget is the make target the release workflow invokes.
const changelogAuditTarget = "changelog-audit"

// changelogAuditEntryPoint is the test the target runs — the audit's
// release-gate entry point.
const changelogAuditEntryPoint = "TestPolicy_ChangelogCompleteness"

// releaseWorkflowRel is the workflow that fires on a release tag.
const releaseWorkflowRel = ".github/workflows/changelog-check.yml"

// ghWorkflow is the slice of GitHub Actions' schema these assertions
// read: the jobs, their steps, and each step's `run` and `with`.
type ghWorkflow struct {
	Jobs map[string]struct {
		Steps []struct {
			Name string            `yaml:"name"`
			Uses string            `yaml:"uses"`
			Run  string            `yaml:"run"`
			With map[string]any    `yaml:"with"`
			Env  map[string]string `yaml:"env"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

func loadReleaseWorkflow(t *testing.T) ghWorkflow {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), releaseWorkflowRel))
	if err != nil {
		t.Fatalf("reading %s: %v", releaseWorkflowRel, err)
	}
	var wf ghWorkflow
	if err := yaml.Unmarshal(raw, &wf); err != nil {
		t.Fatalf("parsing %s: %v", releaseWorkflowRel, err)
	}
	return wf
}

// TestChangelogAuditWiring_TargetRunsTheEntryPoint is the middle link.
// It resolves the recipe by running make rather than by reading the
// Makefile, so renaming an intermediate target keeps it green and only
// the audit genuinely dropping out turns it red.
func TestChangelogAuditWiring_TargetRunsTheEntryPoint(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	recipe := makeDryRun(t, root, changelogAuditTarget)

	if !strings.Contains(recipe, changelogBaseEnv+"=") {
		t.Errorf("`make %s` must hand the audit a base via %s; recipe was:\n%s", changelogAuditTarget, changelogBaseEnv, recipe)
	}
	if !strings.Contains(recipe, " -v ") {
		t.Errorf("`make %s` must pass -v; the unattributed half is logged rather than failed (D-0087), and a log line is invisible without it. Recipe was:\n%s",
			changelogAuditTarget, recipe)
	}

	// The pattern the recipe actually runs, taken from the recipe rather
	// than restated here.
	m := regexp.MustCompile(`-run '([^']+)'`).FindStringSubmatch(recipe)
	if m == nil {
		t.Fatalf("`make %s` must select tests with -run '<pattern>'; recipe was:\n%s", changelogAuditTarget, recipe)
	}

	// Asking go test what that pattern selects is the whole point. A
	// comparison against a string constant proves the recipe and the
	// constant agree and says nothing about whether the named test
	// exists — measured, renaming the entry point then leaves the suite
	// green and turns this target into a zero-test pass at exit 0,
	// which is the silent-gate failure AC-4 exists to close.
	list := exec.Command("go", "test", "-list", m[1], "./internal/policies/")
	list.Dir = root
	out, err := list.CombinedOutput()
	if err != nil {
		t.Fatalf("go test -list %s: %v\n%s", m[1], err, out)
	}
	if !slices.Contains(strings.Fields(string(out)), changelogAuditEntryPoint) {
		t.Errorf("`make %s` runs -run '%s', which selects no test named %s — the release gate would pass by running nothing.\ngo test -list said:\n%s",
			changelogAuditTarget, m[1], changelogAuditEntryPoint, out)
	}
}

// TestChangelogAuditWiring_ReleaseWorkflowInvokesTheTarget is the outer
// link. Without it the target is a command nobody runs, and the audit
// reports only when someone thinks to ask.
func TestChangelogAuditWiring_ReleaseWorkflowInvokesTheTarget(t *testing.T) {
	t.Parallel()
	wf := loadReleaseWorkflow(t)

	for _, job := range wf.Jobs {
		for _, step := range job.Steps {
			if strings.Contains(step.Run, "make "+changelogAuditTarget) {
				return
			}
		}
	}
	t.Errorf("%s must invoke `make %s` in some step; no step does", releaseWorkflowRel, changelogAuditTarget)
}

// TestChangelogAuditWiring_AuditJobChecksOutFullHistory pins the link
// the other two cannot see. The base is the newest tag reachable from
// HEAD, and Actions' checkout is shallow by default with no tags — so a
// job that takes the default resolves against history it does not have,
// and the audit compares the release against the wrong range while every
// other assertion here stays green.
func TestChangelogAuditWiring_AuditJobChecksOutFullHistory(t *testing.T) {
	t.Parallel()
	wf := loadReleaseWorkflow(t)

	// Every job that runs the audit is checked, not the first one a map
	// iteration happens to yield — with two such jobs the verdict would
	// otherwise depend on iteration order.
	checked := 0
	for name, job := range wf.Jobs {
		runsAudit := false
		for _, step := range job.Steps {
			if strings.Contains(step.Run, "make "+changelogAuditTarget) {
				runsAudit = true
			}
		}
		if !runsAudit {
			continue
		}
		checked++
		depth, ok := "", false
		for _, step := range job.Steps {
			if !strings.HasPrefix(step.Uses, "actions/checkout") {
				continue
			}
			if v, present := step.With["fetch-depth"]; present {
				depth, ok = strings.TrimSpace(strings.Trim(scalarString(v), `"`)), true
			}
		}
		if !ok || depth != "0" {
			t.Errorf("job %q runs the audit but checks out with fetch-depth=%q (present=%v); the base is a tag reachable from HEAD, which a shallow checkout cannot resolve",
				name, depth, ok)
		}
	}
	if checked == 0 {
		t.Fatalf("no job in %s runs `make %s`; nothing to check the checkout depth of", releaseWorkflowRel, changelogAuditTarget)
	}
}

// scalarString renders a YAML scalar however it parsed. `fetch-depth: 0`
// yields an int and `fetch-depth: "0"` a string; both mean the same
// thing to Actions, so both have to read the same here.
func scalarString(v any) string {
	return fmt.Sprintf("%v", v)
}
