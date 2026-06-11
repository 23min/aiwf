package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/entity"
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

	// CellID names the branch-choreography catalog cell this
	// scenario pins (per M-0162/AC-3). Must match an ID in
	// `branch.Rules()`. RunScenarios records this in the
	// branchtest Pin registry under -tags testpins so AC-4's
	// bijection meta-test can verify the 1:1 mapping. Empty
	// CellID is permitted only during incremental migration
	// and surfaces an AC-4 bijection finding (orphan subtest)
	// once the meta-test lands.
	CellID string

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

	// FindingSeverity asserts that EVERY finding with code
	// FindingPresent has this severity ("error" or "warning").
	// Pins M-0106/AC-11's severity claim and prevents a future
	// regression that flipped warning→error without an explicit
	// decision. Empty string skips the severity check.
	FindingSeverity string

	// FindingHintContainsAll asserts that at least one finding with
	// code FindingPresent has a hint that contains EVERY substring
	// in this slice. Pins M-0106/AC-12's "hint names both override
	// paths" claim — both "cherry-pick" and "force" must appear in
	// the same hint — without requiring exact-text equality. Empty
	// slice skips the hint check.
	FindingHintContainsAll []string

	// FindingCount asserts the count of findings with code
	// FindingPresent equals this value. Used by M-0106/AC-10's
	// per-commit-firing scenario, where N violating commits must
	// produce EXACTLY N findings (not aggregate, not duplicated).
	// Zero value (default) skips the count check.
	FindingCount int

	// FindingSubcode further constrains FindingPresent /
	// NoFindingWithCode by the finding's subcode field — required
	// for rules that bundle multiple subcodes under a single Code
	// (e.g., fsm-history-consistent emits illegal-transition,
	// forced-untrailered, manual-edit, and history-walk-error all
	// under one Code). Without this distinction, a scenario
	// asserting "no forced-untrailered finding" would spuriously
	// pass when illegal-transition fires under the same Code.
	//
	// When set alongside FindingPresent: at least one finding must
	// match BOTH Code and Subcode.
	// When set alongside NoFindingWithCode: no finding with the
	// Code may also have this Subcode (a different-subcode finding
	// under the same Code is allowed).
	// Empty string skips the subcode constraint (default behavior
	// matches Code only).
	FindingSubcode string
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

// RunScenarios is the table driver. Each row runs as a t.Run
// subtest with t.Parallel; the driver builds a fresh ScenarioEnv
// per row, calls Setup, then runs `aiwf check --format=json` and
// asserts Expect.
func RunScenarios(t *testing.T, scenarios []Scenario) {
	t.Helper()
	for i := range scenarios {
		sc := &scenarios[i]
		t.Run(sc.Name, func(t *testing.T) {
			t.Parallel()
			// Record the cell pin in the branchtest registry
			// (no-op without -tags testpins per pinCell's
			// build-tagged stubs). M-0162/AC-3 feeds AC-4's
			// bijection meta-test from this single seam.
			pinCell(sc.CellID, t.Name())
			env := newScenarioEnv(t)
			sc.Setup(t, env)
			assertExpectation(t, env, sc.Expect)
		})
	}
}

// newScenarioEnv builds a fresh real-git fixture for one scenario:
// temp repo + bare upstream + `aiwf init` + a normalized "main"
// branch tracking origin/main.
//
// Trunk name is hardcoded to "main" here, matching the kernel's
// current default. G-0200 covers generalizing this to
// aiwf.yaml.allocate.trunk so consumers using "master", "dev", or
// "develop" get the same coverage. M-0161/AC-1 lands the trunk-
// config rework, at which point this helper will read the configured
// trunk name instead of hardcoding "main".
//
// Upstream-tracking discipline: SetupGitRepoWithUpstream pushes
// `HEAD:main` to origin so origin/main exists; on hosts where
// `git init` defaults to `master` (the devcontainer's git default),
// HEAD's branch is `master` and tracking is set on `master`. The
// `checkout -B main` below then creates a SEPARATE local `main`
// branch with NO upstream — `git rev-parse @{u}` fails silently on
// the new branch, which makes `aiwf check` emit the
// `provenance-untrailered-scope-undefined` advisory and SKIP the
// entire untrailered audit (including the trailer-verb-unknown
// rule). Latent bug for M-0159/AC-2..AC-4 (which test
// fsm-history-consistent and isolation-escape — both called
// BEFORE the untrailered short-circuit), discovered when AC-5's
// trailer-verb-unknown scenarios surfaced it. The explicit
// `--set-upstream-to=origin/main` after the normalization closes
// the loop so `@{u}` resolves to `origin/main` consistently across
// all hosts regardless of init.defaultBranch.
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
	// Restore upstream tracking the `checkout -B` may have severed
	// (it severs when init.defaultBranch != "main" — the original
	// HEAD branch was, say, `master`, and `-B main` created a
	// brand-new local `main` ref with no tracking). The
	// remote-tracking ref origin/main exists locally because
	// SetupGitRepoWithUpstream pushed `HEAD:main` with `-u`; we
	// only need to re-bind this branch to it.
	if out, err := testutil.RunGit(root, "branch", "--set-upstream-to=origin/main", "main"); err != nil {
		t.Fatalf("set main upstream to origin/main: %v\n%s", err, out)
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
			Subcode  string `json:"subcode"`
			Hint     string `json:"hint"`
		} `json:"findings"`
	}
	if jErr := json.Unmarshal([]byte(out), &envelope); jErr != nil {
		t.Fatalf("parse check envelope: %v\nenvelope bytes:\n%s", jErr, out)
	}

	// subcodeLabel produces a human-readable form for diagnostics
	// — "<code>" or "<code>/<subcode>" — so a test failure on an
	// over-broad match reports the discriminating field clearly.
	subcodeLabel := func(code, subcode string) string {
		if subcode == "" {
			return code
		}
		return code + "/" + subcode
	}

	if expect.NoFindingWithCode != "" {
		for _, f := range envelope.Findings {
			if f.Code != expect.NoFindingWithCode {
				continue
			}
			// When FindingSubcode is set, a different-subcode
			// finding under the same Code is allowed (M-0159/AC-4
			// case: a forced-untrailered scenario asserting "no
			// forced-untrailered finding" must NOT spuriously
			// fail when an unrelated illegal-transition under
			// fsm-history-consistent fires on the same fixture).
			if expect.FindingSubcode != "" && f.Subcode != expect.FindingSubcode {
				continue
			}
			t.Errorf("expected NO finding with code %q; got %+v\nenvelope:\n%s",
				subcodeLabel(expect.NoFindingWithCode, expect.FindingSubcode), f, out)
		}
	}
	if expect.FindingPresent != "" {
		count := 0
		var firstHit *struct {
			Code     string `json:"code"`
			Severity string `json:"severity"`
			Message  string `json:"message"`
			Subcode  string `json:"subcode"`
			Hint     string `json:"hint"`
		}
		hintSeen := false
		for i := range envelope.Findings {
			f := &envelope.Findings[i]
			if f.Code != expect.FindingPresent {
				continue
			}
			// FindingSubcode (when set) constrains the match.
			if expect.FindingSubcode != "" && f.Subcode != expect.FindingSubcode {
				continue
			}
			count++
			if firstHit == nil {
				firstHit = f
			}
			if expect.FindingSeverity != "" && f.Severity != expect.FindingSeverity {
				t.Errorf("finding %q: severity = %q; want %q\nenvelope:\n%s",
					subcodeLabel(f.Code, f.Subcode), f.Severity, expect.FindingSeverity, out)
			}
			if len(expect.FindingHintContainsAll) > 0 {
				allFound := true
				for _, sub := range expect.FindingHintContainsAll {
					if !strings.Contains(f.Hint, sub) {
						allFound = false
						break
					}
				}
				if allFound {
					hintSeen = true
				}
			}
		}
		if count == 0 {
			t.Errorf("expected finding with code %q; envelope had no such finding\nenvelope:\n%s",
				subcodeLabel(expect.FindingPresent, expect.FindingSubcode), out)
		}
		if expect.FindingCount != 0 && count != expect.FindingCount {
			t.Errorf("finding %q count = %d; want %d\nenvelope:\n%s",
				subcodeLabel(expect.FindingPresent, expect.FindingSubcode), count, expect.FindingCount, out)
		}
		if len(expect.FindingHintContainsAll) > 0 && !hintSeen {
			t.Errorf("no finding with code %q has hint containing ALL of %v\nenvelope:\n%s",
				subcodeLabel(expect.FindingPresent, expect.FindingSubcode), expect.FindingHintContainsAll, out)
		}
	}
}

// Branch-choreography helpers consumed by scenarios.

// OpenBoundScope runs `aiwf authorize <entityID> --to ai/claude
// --branch <boundBranch>`, opening a scope explicitly bound to
// boundBranch. Returns the authorize commit's SHA.
//
// The boundBranch argument is the *target* ritual branch — it can
// be the current branch (the aiwfx-start-milestone pattern, where
// HEAD is already on the ritual ref) OR a future branch that
// hasn't been cut yet (the aiwfx-start-epic step-7 pattern, where
// the opener lands on main with --branch naming the future epic
// ritual; the branch is cut in a later step). Both patterns are
// covered by M-0104/AC-4 and M-0105/AC-6 carve-outs.
//
// Trailer-emission behavior (verified against verb source at
// internal/verb/authorize.go:281-347): when --branch is supplied
// (the helper always supplies it), the verb stamps the
// aiwf-branch: trailer with that value. When --branch is OMITTED
// for an ai/* target from a ritual-shape current branch, the verb
// still stamps the trailer — it promotes opts.Branch to the
// current branch's name at authorize.go:345 ("making the implicit
// binding explicit in the commit record"). So post-M-0102, ai/*-
// targeted authorize commits ALWAYS carry aiwf-branch: when the
// preflight accepts. The only post-M-0102 shape without the
// trailer is a sovereign-override `--force --reason` that bypasses
// preflight entirely.
func OpenBoundScope(t *testing.T, env *ScenarioEnv, entityID, boundBranch string) string {
	t.Helper()
	// M-0161/AC-2: many M-0106 scenarios checkout the bound branch
	// BEFORE opening the scope (e.g., on epic/E-0001-engine, then
	// OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")). That's
	// a (epic, epic) rung pair which AC-2's predicate refuses. The
	// scenarios test post-authorize behavior (isolation-escape rule
	// firing on subsequent AI commits) — not the authorize preflight
	// itself — so the test-helper uses the sovereign-override path
	// (--force --reason) to bypass the AC-2 predicate cleanly.
	// Production callers would either authorize from the parent
	// branch first per ADR-0010 or pass --force --reason explicitly;
	// the M-0106 scenario fixture chooses the latter for shape
	// orthogonality.
	env.MustRunBin("authorize", entityID,
		"--to", "ai/claude",
		"--branch", boundBranch,
		"--force",
		"--reason", "test fixture: scope-on-bound-branch (M-0106 scenario)")
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
	// --no-verify: simulators that bypass aiwf verb-time discipline
	// must also bypass commit-time hooks for fixture consistency. The
	// G-0218 commit-msg hook accepts `aiwf-verb: edit-body` today (it's
	// a real verb), but future adjacency rules at the commit-msg
	// chokepoint should not silently break this fixture's intent.
	if out, err := testutil.RunGit(env.Root, "commit", "--no-verify", "-m", msg); err != nil {
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
	entityID := strings.TrimSpace(env.MustRunGit("log", "-1",
		"--pretty=%(trailers:key=aiwf-entity,valueonly=true,unfold=true)"))

	if verb == "" {
		t.Fatalf("ForceAmendHEAD: HEAD commit has no aiwf-verb trailer; cannot construct amend message")
	}
	if entityID == "" {
		t.Fatalf("ForceAmendHEAD: HEAD commit has no aiwf-entity trailer; cannot construct amend message")
	}

	// Compose the amend message: original subject + minimal
	// kernel trailers + human/peter actor + force trailer.
	// Trailers from the original (principal, on-behalf-of,
	// authorized-by) that bind to non-human provenance are
	// intentionally omitted — the actor flip makes them
	// inappropriate.
	newMsg := fmt.Sprintf("%s\n\naiwf-verb: %s\naiwf-entity: %s\naiwf-actor: human/peter\naiwf-force: %s\n",
		subject, verb, entityID, reason)

	if out, err := testutil.RunGit(env.Root, "commit", "--amend", "-m", newMsg); err != nil {
		t.Fatalf("git commit --amend: %v\n%s", err, out)
	}
	return strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
}

// AC-2 helpers — stubs in red phase, implemented in green.

// PauseScope runs `aiwf authorize <entityID> --pause "<reason>"`,
// transitioning the most-recently-opened active scope on entityID
// to paused. Returns the pause commit SHA.
//
// The reason is required by the verb's contract and ends up in
// the commit's aiwf-reason: trailer.
//
// HONEST SCOPE — pinned against the M-0106 algorithm at
// internal/check/isolation_escape.go:104-249. The rule has NO
// paused-state code path; the pause event is structurally
// invisible to it:
//   - The opener-index build (line 144-164) requires
//     aiwf-scope == "opened"; pause's "paused" value is skipped.
//   - The ends-index build (line 130-141) requires the
//     aiwf-scope-ends: trailer; pause has none.
//   - The per-commit walk (line 180-182) skips every commit
//     whose aiwf-verb == "authorize"; pause IS an authorize verb.
//
// So PauseScope's commit exists in chronological history but is
// behaviorally a no-op for the isolation-escape rule. Scenarios
// using this helper pin "the pause event is correctly ignored"
// rather than any pause-state suppression — a future buggy
// addition like "fire on AI commits during paused scope" would
// break those scenarios because the pause's presence would
// suddenly matter.
func PauseScope(t *testing.T, env *ScenarioEnv, entityID, reason string) string {
	t.Helper()
	env.MustRunBin("authorize", entityID, "--pause", reason)
	return strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
}

// EndScope ends the most-recently-opened active scope on
// entityID by promoting its parent (or itself, if a top-level
// entity) to a terminal status. The kernel writes an
// `aiwf-scope-ends: <opener-sha>` trailer on the terminal-promote
// commit. After this, M-0106/F-3's "AI commit after scope ended
// silent" path applies: AI commits with chronoIdx after the
// scope-end commit have no active scope to bind against.
//
// Returns the terminal-promote commit SHA. The chosen terminal
// status depends on entityID's kind (epic → done; milestone →
// done; etc.); the helper picks the simplest valid terminal.
func EndScope(t *testing.T, env *ScenarioEnv, entityID string) string {
	t.Helper()
	// Today only the epic kind is supported (matches AC-2's
	// scope: F-3 scenarios target epic-bound scopes). Per the
	// findEntityBodyPath precedent, other kinds Fatal loudly
	// until the AC that needs them extends this switch.
	if !strings.HasPrefix(entityID, "E-") {
		t.Fatalf("EndScope: entity kind for %q not supported by M-0159/AC-2 scope (only epics today; further kinds as their ACs need them)", entityID)
	}
	// Epic FSM: proposed → active → done. The terminal-promote
	// to `done` is the commit that carries the aiwf-scope-ends:
	// trailer (kernel writes it automatically on terminal
	// promote — verified via internal/cli/cliutil/provenance.go).
	env.MustRunBin("promote", entityID, "active")
	env.MustRunBin("promote", entityID, "done")
	return strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
}

// HumanCommit runs `aiwf edit-body <entityID> --body-file -` with
// the default actor (derived from git config user.email; in test
// envs this is "test@example.com" → human/test). Replaces the
// entity's body with bodyText. Returns the commit SHA.
//
// Used by the M-0106/AC-1-followup scenario that pins the rule's
// actor-prefix filter specificity: a human-actor commit on the
// wrong branch is silent because the filter looks for `ai/`, not
// "anyone with a role." Distinct from AICommit (which forces
// ai/claude) — pinning the prefix specificity needs an actor that
// is structurally similar but doesn't match the filter.
//
// The verb's preflight does NOT refuse human-actor commits off the
// bound branch — that refusal is specific to ai/* targets per
// M-0103. So this helper works on any branch.
func HumanCommit(t *testing.T, env *ScenarioEnv, entityID, bodyText string) string {
	t.Helper()
	// No --actor / --principal flags: the verb derives actor from
	// git config user.email. The scenario env is initialized via
	// initRepoFor(t, "peter@example.com") so the resolved actor
	// is "human/peter". The human-actor path is not subject to
	// M-0103 preflight refusal, so this works on any branch.
	out, err := testutil.RunBinStdin(t, env.Root, env.BinDir,
		strings.NewReader(bodyText),
		"edit-body", entityID,
		"--body-file", "-",
		"--reason", "Human work commit on scoped entity (M-0159/AC-2 scenario)")
	if err != nil {
		t.Fatalf("aiwf edit-body %s (human): %v\n%s", entityID, err, out)
	}
	return strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
}

// CherryPick runs `git cherry-pick` on sourceSHA from the current
// HEAD. The four shapes the helper supports — combinations of
// "with marker" / "without marker" and "distinct committer" /
// "same committer" — are the four discriminating fixtures
// M-0159/AC-6 needs to exercise the rule's both-signals-required
// suppression contract.
//
// If withMarker is true, the cherry-pick is `cherry-pick -x` so
// git writes the canonical `(cherry picked from commit <sha>)`
// body marker. If false, plain `cherry-pick` is used (no marker
// emitted by git).
//
// If committerEmail is non-empty, the cherry-pick subprocess
// inherits GIT_COMMITTER_EMAIL / GIT_COMMITTER_NAME env-var
// overrides (NOT `-c user.email=` config — see the inline comment
// in the body for why config-via-`-c` is silently ignored here),
// so the resulting commit's committer identity is distinct from
// the source's preserved author identity (the author identity is
// the source commit's author, which `git cherry-pick` preserves
// by default). When committerEmail is empty, the cherry-pick uses
// the test process's default GIT_COMMITTER_EMAIL (set by
// setup_test.go::TestMain to "test@example.com"), which equals
// the source's author email, so committer == author and no gap
// exists.
//
// Returns the cherry-pick commit's SHA.
//
// AC-6's gather-side recognizes a commit as a cherry-pick to
// suppress (under the M-0106 isolation-escape rule) only when
// BOTH signals are present:
//
//	(a) `(cherry picked from commit <sha>)` body marker
//	(b) committer email differs from author email
//
// Both signals together are what `git cherry-pick -x` mechanically
// produces when a human re-authors an AI's commit. The two
// negative-control scenarios (marker-only and gap-only) each lack
// one signal and therefore still fire — proving neither alone is
// sufficient. The four-arm helper makes constructing those
// scenarios mechanically explicit.
func CherryPick(t *testing.T, env *ScenarioEnv, sourceSHA, committerEmail, committerName string, withMarker bool) string {
	t.Helper()
	args := []string{"cherry-pick"}
	if withMarker {
		args = append(args, "-x")
	}
	args = append(args, sourceSHA)
	var extraEnv []string
	if committerEmail != "" {
		// Per-call committer identity override. MUST go through
		// env vars (not `-c user.email=`) because the
		// integration-test setup_test.go's TestMain sets
		// GIT_COMMITTER_NAME / GIT_COMMITTER_EMAIL process-wide
		// for deterministic identity. Git evaluates GIT_*_EMAIL
		// env vars with higher precedence than `-c` config
		// overrides — `-c` is silently ignored when env vars are
		// set. Only extraEnv passed through testutil's
		// RunGitWithExtraEnv (which appends LAST so the new
		// entries win over the defaults via the last-wins
		// duplicate-key rule in exec.Command's Env handling)
		// actually flips the committer identity.
		//
		// Both NAME and EMAIL must be set together — overriding
		// just EMAIL leaves the previous NAME in effect, which
		// is harmless for the rule's gap check (only email is
		// compared) but cosmetically weird in `git log`.
		name := committerName
		if name == "" {
			name = "Cherry Picker"
		}
		extraEnv = []string{
			"GIT_COMMITTER_EMAIL=" + committerEmail,
			"GIT_COMMITTER_NAME=" + name,
		}
	}
	if out, err := testutil.RunGitWithExtraEnv(env.Root, extraEnv, args...); err != nil {
		t.Fatalf("git cherry-pick %v (extraEnv=%v): %v\n%s", args, extraEnv, err, out)
	}
	return strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
}

// SimulateForcedUntrailedActivate constructs a raw git commit
// that promotes the named epic from `proposed` to `active`
// (the kernel's canonical sovereign-act-shape transition) by
// hand-editing the frontmatter status field and committing
// with aiwf-actor: ai/claude — but WITHOUT an aiwf-force
// trailer. This is exactly the shape that fires
// fsm-history-consistent's forced-untrailered subcode: a
// sovereign-act-shape transition by a non-human actor without
// the override trailer.
//
// The verb path (aiwf promote) would refuse this shape per the
// M-0095 verb-time gate ("requireHumanActorForSovereignAct" or
// --force --reason). Only a raw-git fabrication can reach the
// rule's predicate end-to-end; this helper produces that
// fabrication for AC-4's forced-untrailered scenarios.
//
// Returns the commit SHA. The entity must already exist at
// status=proposed (call env.MustRunBin("add", "epic", ...)
// first); only epics today (matching findEntityBodyPath's
// kind support).
func SimulateForcedUntrailedActivate(t *testing.T, env *ScenarioEnv, entityID string) string {
	t.Helper()
	bodyPath := findEntityBodyPath(t, env, entityID)
	abs := filepath.Join(env.Root, bodyPath)
	content, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("read %s: %v", abs, err)
	}
	// Replace `status: proposed` with `status: active` SCOPED
	// to the YAML frontmatter block. A naive
	// strings.Replace on the full file body would silently
	// mutate the wrong line if a future scenario seeded body
	// content containing the literal "status: proposed" (e.g.
	// in a code fence or quoted example). Per M-0159/AC-4
	// refactor #74 (first/second reviewer note N2).
	//
	// Frontmatter shape per internal/entity/serialize.go: starts
	// with `---\n` (or `---\r\n`); ends at the next `\n---` line
	// boundary. Same recognition logic the kernel's
	// parseStatusFromFrontmatter uses.
	updated, err := replaceStatusInFrontmatter(content, "proposed", "active")
	if err != nil {
		t.Fatalf("SimulateForcedUntrailedActivate(%s): %v", bodyPath, err)
	}
	if err := os.WriteFile(abs, updated, 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
	if out, err := testutil.RunGit(env.Root, "add", bodyPath); err != nil {
		t.Fatalf("git add %s: %v\n%s", bodyPath, err, out)
	}
	msg := fmt.Sprintf("simulate AI promote %s without aiwf-force (M-0159/AC-4 fixture)\n\naiwf-verb: promote\naiwf-entity: %s\naiwf-actor: ai/claude\n",
		entityID, entityID)
	// --no-verify: see SimulateAIEscape — simulators that bypass
	// aiwf verb-time discipline bypass commit-time hooks too for
	// fixture consistency.
	if out, err := testutil.RunGit(env.Root, "commit", "--no-verify", "-m", msg); err != nil {
		t.Fatalf("simulate forced-untrailered activate (raw git commit): %v\n%s", err, out)
	}
	return strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
}

// replaceStatusInFrontmatter returns content with the
// `status: <prior>` line inside the YAML frontmatter
// (delimited by `---` boundaries at the file start and at the
// next `\n---` line) replaced by `status: <next>`. The
// frontmatter slice is determined first, then the replacement
// is applied inside that slice ONLY — so any literal
// `status: <prior>` appearing in body prose is left untouched.
//
// Returns an error if (a) the file doesn't start with the
// frontmatter opening marker, (b) no closing marker is found,
// or (c) the frontmatter doesn't contain a `status: <prior>`
// line. All three are fixture-shape errors at fabrication
// time, not legitimate edge cases the helper should swallow.
func replaceStatusInFrontmatter(content []byte, prior, next string) ([]byte, error) {
	const opener = "---\n"
	const openerCRLF = "---\r\n"
	const closer = "\n---"

	rest := content
	openerLen := 0
	switch {
	case bytes.HasPrefix(content, []byte(opener)):
		rest = content[len(opener):]
		openerLen = len(opener)
	case bytes.HasPrefix(content, []byte(openerCRLF)):
		rest = content[len(openerCRLF):]
		openerLen = len(openerCRLF)
	default:
		return nil, fmt.Errorf("file does not start with YAML frontmatter opener `---` (got %q...)", firstNBytes(content, 16))
	}
	end := bytes.Index(rest, []byte(closer))
	if end < 0 {
		return nil, fmt.Errorf("frontmatter opener present but no closing `\\n---` found in file")
	}
	frontmatter := rest[:end] // bytes BETWEEN the opener and closer, exclusive
	body := rest[end:]        // bytes from `\n---` onwards, inclusive

	target := []byte("status: " + prior)
	if !bytes.Contains(frontmatter, target) {
		return nil, fmt.Errorf("frontmatter does not contain %q; cannot fabricate the %s -> %s transition", string(target), prior, next)
	}
	mutated := bytes.Replace(frontmatter, target, []byte("status: "+next), 1)

	// Reassemble: original opener + mutated frontmatter + body.
	out := make([]byte, 0, openerLen+len(mutated)+len(body))
	out = append(out, content[:openerLen]...)
	out = append(out, mutated...)
	out = append(out, body...)
	return out, nil
}

// firstNBytes returns the first n bytes of b, or all of b if
// shorter. Used for diagnostic context in error messages so a
// fixture-shape error names what the file actually started with.
func firstNBytes(b []byte, n int) []byte {
	if len(b) < n {
		return b
	}
	return b[:n]
}

// SimulateStrayVerbCommit constructs a raw git commit on the
// current branch whose `aiwf-verb:` trailer carries a fabricated
// value — i.e., one not in the running binary's Cobra command
// tree nor in the embedded ritual snapshot's verb set. This is
// the canonical G-0150 shape: an LLM session synthesizing a
// trailer like `aiwf-verb: implement` on a hand-rolled
// Conventional-Commits code commit.
//
// The fabricated commit touches a fresh non-entity file at the
// repo root (`notes-<subjectText-derived>.md`) so the diff is
// real (mirroring G-0150's "hand-rolled code commit" shape, not
// `--allow-empty` synthetic emptiness) WITHOUT also tripping the
// entity-untrailered audit — that audit fires on commits that
// touch entity files without canonical trailers, and would
// otherwise add a competing finding under a different code that
// scenarios would have to filter around. Keeping the touch
// non-entity isolates the trailer-verb-unknown signal.
//
// The trailer set is intentionally minimal: only `aiwf-verb:
// <fabricated>`. No aiwf-entity, no aiwf-actor — the trailer-
// verb-unknown rule's input is the `aiwf-verb:` value alone, and
// the fixture matches the canonical G-0150 shape (a fabricated
// `aiwf-verb:` line on a hand-rolled code commit). Adding
// non-load-bearing trailers would over-fit the fixture and
// conflate this scenario with rules that fire on actor/entity
// trailers — exactly what the per-rule isolation argument in the
// AC-5 file header guards against.
//
// Returns the commit SHA. Used by AC-5's real-git E2E to convert
// the docstring promise at trailer_verb_unknown.go:25-29
// ("Promotion to error is contingent on cleaning history first
// — potentially via `aiwf acknowledge-illegal`") into mechanical
// truth: stray commit + acknowledge-illegal → check silent.
func SimulateStrayVerbCommit(t *testing.T, env *ScenarioEnv, fabricatedVerb, subjectText string) string {
	t.Helper()
	// Per-commit filename derives from the subject text so
	// multiple stray commits in one scenario don't collide on
	// identical file content (which would re-trigger "nothing to
	// commit"). A simple suffix is enough; the file is throw-
	// away fixture content.
	relPath := "notes-" + sanitizeForFilename(subjectText) + ".md"
	abs := filepath.Join(env.Root, relPath)
	body := fmt.Sprintf("# %s\n\nfabricated stray-verb fixture (M-0159/AC-5)\n", subjectText)
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
	if out, err := testutil.RunGit(env.Root, "add", relPath); err != nil {
		t.Fatalf("git add %s: %v\n%s", relPath, err, out)
	}
	msg := fmt.Sprintf("%s\n\naiwf-verb: %s\n", subjectText, fabricatedVerb)
	// --no-verify bypasses the G-0218 commit-msg hook so this fixture
	// can still author a fabricated-trailer commit — the post-hoc
	// trailer-verb-unknown rule is what we're exercising, the hook
	// is bypassed deliberately to construct the historical shape.
	if out, err := testutil.RunGit(env.Root, "commit", "--no-verify", "-m", msg); err != nil {
		t.Fatalf("simulate stray-verb commit (raw git): %v\n%s", err, out)
	}
	return strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
}

// sanitizeForFilename produces a short, filesystem-safe slug from
// a free-form subject string. Used by SimulateStrayVerbCommit to
// derive per-commit fixture filenames.
//
// Delegates the lowercase-alnum-dash-collapse alphabet to
// entity.Slugify (the kernel's canonical slug producer at
// internal/entity/serialize.go) so drift between the test-helper
// alphabet and production slug rules cannot accumulate silently.
// The fixture-specific concerns layered on top:
//
//  1. 40-char cap so fixture filenames stay short and don't push
//     scenario paths near system limits. The cap belongs at the
//     test-helper layer — entity.Slugify intentionally does not
//     cap because real-entity slugs cap at the configured
//     entities.title_max_length (default 80) via a different
//     write-time chokepoint, and pushing a 40-char cap into the
//     kernel would couple frontmatter slug derivation to
//     filesystem path-length anxiety.
//
//  2. Empty-input fallback to "stray". entity.Slugify returns
//     empty on empty input; real-entity callers reject empties
//     upstream via the title-validation chokepoint, but test
//     fixtures have no such upstream, so the fallback lives here.
//
// The trim-after-cap is explicit: slicing [:maxLen] may land on
// a trailing `-` (entity.Slugify's collapse leaves single-hyphen
// separators), and a trailing hyphen would otherwise read as
// `notes-foo-.md` which is filesystem-safe but ugly.
func sanitizeForFilename(s string) string {
	const maxLen = 40
	slug := entity.Slugify(s)
	if len(slug) > maxLen {
		slug = strings.TrimRight(slug[:maxLen], "-")
	}
	if slug == "" {
		return "stray"
	}
	return slug
}

// AcknowledgeIllegal runs `aiwf acknowledge-illegal <targetSHA>
// --reason <reason>` which produces a current-day empty commit
// carrying:
//
//	aiwf-verb: acknowledge-illegal
//	aiwf-force-for: <targetSHA>
//	aiwf-actor: human/peter (from git config)
//	aiwf-reason: <reason>
//
// The gather layer in internal/cli/check walks HEAD's reachable
// history for `aiwf-force-for` trailers (via the M-0159/AC-3
// lifted helper WalkAcknowledgedSHAs) and passes the resulting
// SHA set to all three consumer rules. The acknowledged commit's
// SHA appears in that set, so the consuming rule's per-SHA check
// silences any finding against it.
//
// Returns the acknowledgment commit's SHA. The original target
// commit's history is NOT rewritten — the original's author,
// trailers, and SHA are all preserved per M-0136's
// no-history-rewrite principle.
//
// AC-4 uses this helper to set up the real-git E2E for the
// silencing path: acknowledge an isolation-escape or
// forced-untrailered commit, then re-run `aiwf check` and
// verify the previously-firing finding is now silent.
func AcknowledgeIllegal(t *testing.T, env *ScenarioEnv, targetSHA, reason string) string {
	t.Helper()
	env.MustRunBin("acknowledge-illegal", targetSHA, "--reason", reason)
	return strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
}

// --- Frontmatter / filesystem inspection helpers ---------------
//
// Authored at M-0160/AC-1 for reallocate scenarios; lifted from
// reallocate_scenarios_test.go to this shared file at M-0160/AC-3
// once the second caller (apply_rollback_g0170_test.go) appeared,
// per the AC-1-time docstring promise to lift on the second
// caller.

// findEntityFile returns the repo-relative path of the entity
// matching `entityID` (frontmatter `id:` match), searching under
// work/. Returns "" when not found. Used to locate files whose
// slug may differ from the id (any post-rename, any post-
// retitle). Walks both the active tree AND archive subdirs so
// post-archive entities are still discoverable.
func findEntityFile(t *testing.T, env *ScenarioEnv, entityID string) string {
	t.Helper()
	root := filepath.Join(env.Root, "work")
	var result string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		// Cheap check on the first 512 bytes — frontmatter only.
		head := body
		if len(head) > 512 {
			head = head[:512]
		}
		idRE := regexp.MustCompile(`(?m)^id:\s*` + regexp.QuoteMeta(entityID) + `\s*$`)
		if idRE.Match(head) {
			rel, err := filepath.Rel(env.Root, path)
			if err == nil {
				result = filepath.ToSlash(rel)
			}
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		// SkipAll returns nil from WalkDir; anything else is a real
		// walk problem worth logging.
		t.Logf("findEntityFile walk: %v", err)
	}
	return result
}

// fileExists returns true when an entity with the given id is
// present in the active tree (or archive subdir). Distinguished
// from findEntityFile when the test wants a boolean rather than
// the path.
func fileExists(t *testing.T, env *ScenarioEnv, entityID string) bool {
	t.Helper()
	return findEntityFile(t, env, entityID) != ""
}

// readFrontmatter returns the YAML frontmatter block of an entity
// file (the content between the first two `---` delimiters),
// without the delimiters. Returns "" on read failure or malformed
// frontmatter. The test caller asserts substring presence (e.g.,
// `id: G-0002`, `G-0001` in prior_ids) against the returned
// string.
//
// Assumes LF line endings (devcontainer / CI Linux). The closer
// detection `strings.Index(rest, "\n---")` is not CRLF-aware
// — on Windows, `\r\n---` would not match and the helper would
// return "". M-0160 runs in the devcontainer per CLAUDE.md; if
// host-side Windows testing is later in scope, port the splitter
// from replaceStatusInFrontmatter above in this file.
func readFrontmatter(t *testing.T, absPath string) string {
	t.Helper()
	body, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("readFrontmatter %s: %v", absPath, err)
	}
	s := string(body)
	if !strings.HasPrefix(s, "---") {
		return ""
	}
	rest := s[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:end])
}
