package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// branch_scenarios_helpers_test.go — M-0159/AC-1 framework: types,
// driver, and branch-choreography helpers for the combinatorial
// real-git E2E test surface against the branch-choreography rule
// set (M-0102..M-0106, M-0136). Consumed by branch_scenarios_test.go
// and (in subsequent ACs) by per-rule test files that drive the
// table over their own scenario rows.

// Scenario is one row in the branch-choreography scenario table.
// Each row sets up its own fresh real-git fixture (Setup), then the
// driver runs `aiwf check --format=json` and asserts Expect against
// the resulting envelope.
//
// Setup is imperative — Go code that calls verb + git subprocesses
// via the ScenarioEnv helpers. There is no separate "Steps" slice
// or DSL: the kernel's verbs ARE the DSL, and Go's control flow
// keeps each scenario as readable as the equivalent narrative.
type Scenario struct {
	// Name is the t.Run subtest name. Should describe the
	// observable claim ("AI commit on bound branch is silent",
	// not "happy path A").
	Name string

	// Setup runs the scenario's preparation: create entities, set
	// up branches, open scopes, make commits. Mutates the env's
	// real-git repo via env.MustRunBin / env.MustRunGit, or via
	// the typed helpers (OpenBoundScope, AICommit, etc.).
	Setup func(t *testing.T, env *ScenarioEnv)

	// Expect describes the assertions the driver runs against
	// `aiwf check`'s envelope after Setup returns.
	Expect Expectation
}

// Expectation describes one or more assertions to run against
// `aiwf check --format=json`'s envelope. All set fields are
// asserted; unset fields are not checked. A scenario can both
// require a finding's presence (FindingPresent) and the absence
// of another (NoFindingWithCode) — but not the same code.
type Expectation struct {
	// NoFindingWithCode asserts no finding in the envelope has
	// this code. Used for "silent" paths (the bound-branch
	// commit, the cherry-pick, the force-amend override).
	NoFindingWithCode string

	// FindingPresent asserts at least one finding in the envelope
	// has this code. Used for "fires" paths (the escape, the
	// worktree mismatch).
	FindingPresent string
}

// ScenarioEnv is the per-scenario real-git state: a fresh temp
// repo with `aiwf init` already run, plus the directory housing
// the built aiwf binary. Constructed by the driver per scenario;
// not shared across scenarios.
type ScenarioEnv struct {
	T      *testing.T
	Root   string // working repo root
	BinDir string // directory containing aiwf binary (for PATH composition)
}

// MustRunBin invokes the aiwf binary inside the scenario's repo,
// fatal'ing the test on non-zero exit. Returns the combined
// stdout+stderr for callers that need to parse output. Wraps
// testutil.RunBin with the env's root/binDir.
func (e *ScenarioEnv) MustRunBin(args ...string) string {
	e.T.Helper()
	out, err := testutil.RunBin(e.T, e.Root, e.BinDir, nil, args...)
	if err != nil {
		e.T.Fatalf("aiwf %v: %v\n%s", args, err, out)
	}
	return out
}

// MustRunGit invokes git inside the scenario's repo, fatal'ing on
// non-zero exit. Returns stdout. Wraps testutil.RunGit.
func (e *ScenarioEnv) MustRunGit(args ...string) string {
	e.T.Helper()
	out, err := testutil.RunGit(e.Root, args...)
	if err != nil {
		e.T.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return out
}

// TryRunBin is the non-fatal sibling of MustRunBin: returns
// (combined output, error) without fatal'ing. Used by scenarios
// that assert a verb's refusal — capturing the error envelope or
// exit code is the assertion target, not a failure of the test
// fixture. Example: pinning M-0103's preflight refusal of an AI
// authorize without ritual branch context.
func (e *ScenarioEnv) TryRunBin(args ...string) (string, error) {
	e.T.Helper()
	return testutil.RunBin(e.T, e.Root, e.BinDir, nil, args...)
}

// TryRunGit is the non-fatal sibling of MustRunGit: returns
// (stdout, error). Used by scenarios that exercise git operations
// expected to fail (force-push refused on a protected branch,
// merge conflict, etc.).
func (e *ScenarioEnv) TryRunGit(args ...string) (string, error) {
	e.T.Helper()
	return testutil.RunGit(e.Root, args...)
}

// RunScenarios is the table driver. Each row runs as a t.Run
// subtest with t.Parallel; the driver builds a fresh ScenarioEnv
// per row, calls Setup, then runs `aiwf check --format=json` and
// asserts Expect.
func RunScenarios(t *testing.T, scenarios []Scenario) {
	t.Helper()
	for _, sc := range scenarios {
		t.Run(sc.Name, func(t *testing.T) {
			t.Parallel()
			env := newScenarioEnv(t)
			sc.Setup(t, env)
			assertExpectation(t, env, sc.Expect)
		})
	}
}

// newScenarioEnv builds a fresh real-git fixture for one scenario:
// temp repo + bare upstream + `aiwf init` + a normalized "main"
// branch.
//
// Trunk name is hardcoded to "main" here, matching the kernel's
// current default. G-0200 covers generalizing this to
// aiwf.yaml.allocate.trunk so consumers using "master", "dev", or
// "develop" get the same coverage. M-0161/AC-1 lands the trunk-
// config rework, at which point this helper will read the configured
// trunk name instead of hardcoding "main".
func newScenarioEnv(t *testing.T) *ScenarioEnv {
	t.Helper()
	root, binDir := initRepoFor(t, "peter@example.com")
	// Normalize: ensure the local branch tracking origin/main is
	// named "main" regardless of init.defaultBranch git config.
	// `git checkout -B main` is idempotent here — it forces the
	// "main" ref to current HEAD whether or not it already exists.
	if out, err := testutil.RunGit(root, "checkout", "-B", "main"); err != nil {
		t.Fatalf("normalize to main branch: %v\n%s", err, out)
	}
	return &ScenarioEnv{T: t, Root: root, BinDir: binDir}
}

// assertExpectation runs `aiwf check --format=json` and asserts
// the envelope against Expect. The exit-code contract per
// cmd/aiwf/main.go: 0 = ok, 1 = findings (envelope on stdout), 2 =
// usage error, 3 = internal error. We accept 0 and 1 (both produce
// a valid envelope to assert against); 2 and 3 indicate the test
// fixture or the binary is broken in a way that should fail the
// test immediately, not silently parse-and-assert against partial
// output.
func assertExpectation(t *testing.T, env *ScenarioEnv, expect Expectation) {
	t.Helper()
	out, err := testutil.RunBin(t, env.Root, env.BinDir, nil, "check", "--format=json")
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("aiwf check failed to invoke: %v\nstdout+stderr:\n%s", err, out)
		}
		// findings → exit 1 is the legitimate non-zero exit. Anything
		// else (2 usage, 3 internal, signal-kill, ...) is a fixture
		// or binary bug; surface it loudly.
		if code := exitErr.ExitCode(); code != 1 {
			t.Fatalf("aiwf check exited %d (expected 0 or 1; 2 = usage error, 3 = internal error per cmd/aiwf/main.go)\nstdout+stderr:\n%s", code, out)
		}
	}

	var envelope struct {
		Status   string `json:"status"`
		Findings []struct {
			Code     string `json:"code"`
			Severity string `json:"severity"`
			Message  string `json:"message"`
		} `json:"findings"`
	}
	if jErr := json.Unmarshal([]byte(out), &envelope); jErr != nil {
		t.Fatalf("parse check envelope: %v\nenvelope bytes:\n%s", jErr, out)
	}

	if expect.NoFindingWithCode != "" {
		for _, f := range envelope.Findings {
			if f.Code == expect.NoFindingWithCode {
				t.Errorf("expected NO finding with code %q; got %+v\nenvelope:\n%s", expect.NoFindingWithCode, f, out)
			}
		}
	}
	if expect.FindingPresent != "" {
		found := false
		for _, f := range envelope.Findings {
			if f.Code == expect.FindingPresent {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected finding with code %q; envelope had no such finding\nenvelope:\n%s", expect.FindingPresent, out)
		}
	}
}

// Branch-choreography helpers consumed by scenarios.

// OpenBoundScope runs `aiwf authorize <entityID> --to ai/claude
// --branch <boundBranch>`, opening a scope explicitly bound to
// boundBranch. Returns the authorize commit's SHA.
//
// Per M-0102/AC-3 the `aiwf-branch:` trailer is emitted ONLY when
// `--branch` is supplied — the M-0102 implicit-from-current path
// accepts the verb in preflight but does NOT stamp the trailer.
// The isolation-escape rule (M-0106) reads the bound branch off
// that trailer; absent the trailer the rule has no bound ref to
// compare against and stays silent.
//
// The boundBranch argument is the *target* ritual branch — it can
// be the current branch (the aiwfx-start-milestone pattern, where
// HEAD is already on the ritual ref) OR a future branch that
// hasn't been cut yet (the aiwfx-start-epic step-7 pattern, where
// the opener lands on main with --branch naming the future epic
// ritual; the branch is cut in a later step). Both patterns are
// covered by M-0104/AC-4 and M-0105/AC-6 carve-outs.
func OpenBoundScope(t *testing.T, env *ScenarioEnv, entityID, boundBranch string) string {
	t.Helper()
	env.MustRunBin("authorize", entityID, "--to", "ai/claude", "--branch", boundBranch)
	return strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
}

// AICommit runs `aiwf edit-body <entityID> --body-file -` with
// `--actor ai/claude --principal human/peter`, replacing the
// entity's body with bodyText. The resulting commit carries
// aiwf-actor: ai/claude and aiwf-entity: <entityID>. Returns the
// commit SHA.
//
// Used as the canonical "AI does work on the BOUND branch" shape
// in scenarios that exercise the rule's silent path. The verb's
// own provenance check refuses if the caller is off the active
// scope's bound branch — that refusal IS the M-0103-era verb-time
// enforcement. For scenarios that simulate an AI escaping the
// verb path (subagent worktree confusion, raw git from a confused
// subagent), use SimulateAIEscape instead.
//
// The entity must exist (call env.MustRunBin("add", ...) in Setup
// first) and an active scope must be open authorizing ai/claude
// on the bound branch HEAD currently points to.
func AICommit(t *testing.T, env *ScenarioEnv, entityID, bodyText string) string {
	t.Helper()
	out, err := testutil.RunBinStdin(t, env.Root, env.BinDir,
		strings.NewReader(bodyText),
		"edit-body", entityID,
		"--body-file", "-",
		"--actor", "ai/claude",
		"--principal", "human/peter",
		"--reason", "AI work commit on scoped entity (M-0159/AC-1 scenario)")
	if err != nil {
		t.Fatalf("aiwf edit-body %s: %v\n%s", entityID, err, out)
	}
	return strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
}

// SimulateAIEscape constructs a raw git commit on the current
// branch whose trailers mimic what `aiwf edit-body` would have
// produced — but it bypasses every aiwf verb-time check. The
// commit appends a marker line to the entity's body file
// (work/epics/E-NNNN-<slug>/epic.md or
// work/epics/E-NNNN-<parent>/M-NNNN-<slug>.md, etc.) so the
// commit has a real diff — same shape as a real AI escape that
// edited an entity body via raw git. Returns the commit SHA.
//
// This is the canonical real-world escape shape: an AI subagent
// confused about which branch it's on (the G-0099 founding
// incident) ran raw `git commit` after editing an entity file
// directly. The verb-time preflight (M-0102/M-0103) doesn't fire
// because no aiwf verb was invoked; the check-time
// isolation-escape rule (M-0106) is the defense-in-depth that
// catches it.
//
// The trailer set mirrors what aiwf edit-body emits: aiwf-verb,
// aiwf-entity, aiwf-actor, aiwf-principal. No aiwf-authorized-by
// (the active scope's authorize SHA) since this commit
// deliberately pretends the AI escaped the verb.
//
// The fixture's diff is real (a body marker line), not synthetic
// emptiness via --allow-empty — keeps the commit shape consistent
// with what a real escape produces, so adjacent rules that read
// touched-file paths (the untrailered-entity-commit audit, etc.)
// see the same input shape they'd see in production.
func SimulateAIEscape(t *testing.T, env *ScenarioEnv, entityID, subjectText string) string {
	t.Helper()
	bodyPath := findEntityBodyPath(t, env, entityID)
	appendEntityMarker(t, env, bodyPath, subjectText)
	if out, err := testutil.RunGit(env.Root, "add", bodyPath); err != nil {
		t.Fatalf("git add %s: %v\n%s", bodyPath, err, out)
	}
	msg := fmt.Sprintf("%s\n\naiwf-verb: edit-body\naiwf-entity: %s\naiwf-actor: ai/claude\naiwf-principal: human/peter\n",
		subjectText, entityID)
	if out, err := testutil.RunGit(env.Root, "commit", "-m", msg); err != nil {
		t.Fatalf("simulate AI escape (raw git commit): %v\n%s", err, out)
	}
	return strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
}

// findEntityBodyPath returns the repo-relative path to the
// entity's markdown body. Today only the epic kind (E-NNNN) is
// supported — M-0159/AC-1's scenarios all target epics. Other
// kinds (M-NNNN milestones, G-NNNN gaps, D-NNNN decisions, etc.)
// Fatal with a clear message naming the unsupported kind.
//
// Scope extension happens AC-by-AC: M-0159/AC-2 will exercise
// M-0106 paths against milestones (so it will extend this helper
// to the milestone kind in the same commit set). Each extension
// keeps the discrimination explicit so a typo'd entity id surfaces
// loudly instead of silently no-op'ing.
func findEntityBodyPath(t *testing.T, env *ScenarioEnv, entityID string) string {
	t.Helper()
	switch {
	case strings.HasPrefix(entityID, "E-"):
		epicsDir := filepath.Join(env.Root, "work", "epics")
		entries, err := os.ReadDir(epicsDir)
		if err != nil {
			t.Fatalf("read work/epics: %v", err)
		}
		for _, e := range entries {
			if e.IsDir() && strings.HasPrefix(e.Name(), entityID+"-") {
				return filepath.Join("work", "epics", e.Name(), "epic.md")
			}
		}
		t.Fatalf("no epic directory found for %s under work/epics/", entityID)
	default:
		t.Fatalf("findEntityBodyPath: entity kind for %q not supported by M-0159/AC-1 scope (only epics today; M-0159/AC-2 extends to milestones; further kinds as their ACs need them)", entityID)
	}
	return ""
}

// appendEntityMarker writes a marker line to the entity body file,
// creating a real diff for SimulateAIEscape's commit. The marker
// is uniquely keyed to the subject text so multiple escapes against
// the same entity don't collide on identical body content (which
// would re-trigger "nothing to commit").
func appendEntityMarker(t *testing.T, env *ScenarioEnv, repoRelPath, marker string) {
	t.Helper()
	abs := filepath.Join(env.Root, repoRelPath)
	f, err := os.OpenFile(abs, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open %s for append: %v", abs, err)
	}
	defer func() {
		if cErr := f.Close(); cErr != nil {
			t.Fatalf("close %s: %v", abs, cErr)
		}
	}()
	if _, err := fmt.Fprintf(f, "\n<!-- AI escape marker: %s -->\n", marker); err != nil {
		t.Fatalf("append marker to %s: %v", abs, err)
	}
}

// ForceAmendHEAD runs git commit --amend on the HEAD commit,
// rewriting the commit message so the aiwf-actor: trailer reads
// "human/peter" (flipped from whatever it was) and a new
// `aiwf-force: <reason>` trailer is appended. The aiwf-verb and
// aiwf-entity trailers are preserved; trailers tied to non-human
// provenance (aiwf-principal, aiwf-on-behalf-of, aiwf-authorized-by)
// are stripped since a human-actor commit has no principal or
// delegating-scope relationship. Returns the new HEAD SHA (the
// amend rewrites the SHA).
//
// Pins the legacy M-0106/AC-8 sovereign-override mechanism: the
// rule's `ai/` actor-prefix filter sees `human/...` and skips the
// commit. The aiwf-force trailer is informational only (the
// kernel's enforcement is the actor flip, not the trailer).
func ForceAmendHEAD(t *testing.T, env *ScenarioEnv, reason string) string {
	t.Helper()

	// Read HEAD's subject and the kernel trailers we need to
	// preserve. Using --pretty=%(trailers:key=X,valueonly=true)
	// extracts a single key's value cleanly; an empty result
	// means the trailer is absent.
	subject := strings.TrimSpace(env.MustRunGit("log", "-1", "--pretty=%s"))
	verb := strings.TrimSpace(env.MustRunGit("log", "-1",
		"--pretty=%(trailers:key=aiwf-verb,valueonly=true,unfold=true)"))
	entity := strings.TrimSpace(env.MustRunGit("log", "-1",
		"--pretty=%(trailers:key=aiwf-entity,valueonly=true,unfold=true)"))

	if verb == "" {
		t.Fatalf("ForceAmendHEAD: HEAD commit has no aiwf-verb trailer; cannot construct amend message")
	}
	if entity == "" {
		t.Fatalf("ForceAmendHEAD: HEAD commit has no aiwf-entity trailer; cannot construct amend message")
	}

	// Compose the amend message: original subject + minimal
	// kernel trailers + human/peter actor + force trailer.
	// Trailers from the original (principal, on-behalf-of,
	// authorized-by) that bind to non-human provenance are
	// intentionally omitted — the actor flip makes them
	// inappropriate.
	newMsg := fmt.Sprintf("%s\n\naiwf-verb: %s\naiwf-entity: %s\naiwf-actor: human/peter\naiwf-force: %s\n",
		subject, verb, entity, reason)

	if out, err := testutil.RunGit(env.Root, "commit", "--amend", "-m", newMsg); err != nil {
		t.Fatalf("git commit --amend: %v\n%s", err, out)
	}
	return strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
}
