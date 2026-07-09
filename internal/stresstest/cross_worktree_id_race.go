package stresstest

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
)

// cross_worktree_id_race.go — M-0241/AC-3: CrossWorktreeIDRaceScenario
// races two real `aiwf add` subprocesses across SIBLING WORKTREES of
// one repo (contrast ConcurrentIDAllocationScenario, M-0241/AC-2,
// which races within one working copy). Per this repo's own
// eventual-consistency id-allocation design (see AC-4: a linked
// worktree's repolock lockfile is scoped to that worktree, not
// shared with the main checkout), a duplicate id here is an
// *accepted* outcome — the property this scenario pins is that
// `aiwf check` always surfaces a real collision as `ids-unique` and
// `aiwf reallocate` always resolves it cleanly, never that the race
// window is never hit.

// actorATitle / actorBTitle are fixed, single-word titles so each
// actor's entity file slug is the title verbatim (no hyphenation
// ambiguity to reverse-engineer) — see findEntityFile.
const (
	actorATitle = "actora"
	actorBTitle = "actorb"
)

// CrossWorktreeIDRaceScenario implements Scenario.
type CrossWorktreeIDRaceScenario struct {
	aiwfBin    string
	kind       entity.Kind
	violations []Violation
	collided   bool
}

// NewCrossWorktreeIDRaceScenario builds a scenario that races one
// `aiwf add <kind>` subprocess in each of two sibling worktrees. seed
// matches RunRepeated's newScenario(seed int64) Scenario signature
// (M-0240/AC-5) but is otherwise unused — this scenario's race
// jitter comes from real OS goroutine/process scheduling, not seeded
// pseudo-randomness.
func NewCrossWorktreeIDRaceScenario(aiwfBin string, kind entity.Kind, _ int64) *CrossWorktreeIDRaceScenario {
	return &CrossWorktreeIDRaceScenario{aiwfBin: aiwfBin, kind: kind}
}

// Collided reports whether this scenario instance observed a real
// cross-worktree id collision. Exposed so a caller repeating this
// scenario (via RunRepeated) can assert that at least one attempt
// actually exercised the collision-resolution path, rather than a
// run where the race window was never hit passing vacuously.
func (s *CrossWorktreeIDRaceScenario) Collided() bool { return s.collided }

// Setup creates a main repo with a seed commit, then adds two
// sibling worktrees (actor-a, actor-b) off it — dir/main, dir/wt-a,
// dir/wt-b.
func (s *CrossWorktreeIDRaceScenario) Setup(dir string) error {
	mainDir := filepath.Join(dir, "main")
	if err := os.MkdirAll(mainDir, 0o755); err != nil { //coverage:ignore defensive: mainDir is a fresh subdirectory of RunScenario's own os.MkdirTemp result, no realistic failure mode short of filesystem sabotage
		return fmt.Errorf("creating main repo dir: %w", err)
	}
	if err := gitInitAndConfig(mainDir); err != nil { //coverage:ignore defensive: gitInitAndConfig's own internal branch already carries this rationale
		return err
	}
	if err := runGit(mainDir, "commit", "-q", "--allow-empty", "-m", "seed"); err != nil { //coverage:ignore defensive: an empty commit in a freshly-initialized repo has no realistic failure mode
		return err
	}
	if err := runGit(mainDir, "worktree", "add", "-q", "-b", "actor-a", filepath.Join(dir, "wt-a")); err != nil { //coverage:ignore defensive: adding a worktree at a fresh, never-before-used path has no realistic failure mode
		return err
	}
	if err := runGit(mainDir, "worktree", "add", "-q", "-b", "actor-b", filepath.Join(dir, "wt-b")); err != nil { //coverage:ignore defensive: see the actor-a worktree add above
		return err
	}
	return nil
}

// Run races one `aiwf add` in each sibling worktree, merges actor-b
// into actor-a's worktree to bring both into one view, and — only
// if the race actually produced a duplicate id — confirms `aiwf
// check` surfaces it and `aiwf reallocate` resolves it.
func (s *CrossWorktreeIDRaceScenario) Run(dir string) error {
	wtA := filepath.Join(dir, "wt-a")
	wtB := filepath.Join(dir, "wt-b")

	var a, b rawAddResult
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); a = launchAddIn(s.aiwfBin, wtA, s.kind, actorATitle) }()
	go func() { defer wg.Done(); b = launchAddIn(s.aiwfBin, wtB, s.kind, actorBTitle) }()
	wg.Wait()

	for _, r := range []rawAddResult{a, b} {
		var exitErr *exec.ExitError
		if r.execErr != nil && !errors.As(r.execErr, &exitErr) { //coverage:ignore defensive: same launch-failure class pinned at its source by TestCrossWorktreeIDRaceScenario_RealBinary_ErrorsWhenBinaryMissing
			return fmt.Errorf("launching aiwf add across sibling worktrees: %w", r.execErr)
		}
	}
	envA, err := parseVerbEnvelope([]string{"add", string(s.kind)}, a.out)
	if err != nil { //coverage:ignore defensive: parseVerbEnvelope's own malformed-input branch is unit-tested directly in verb_sequence_classify_test.go
		return fmt.Errorf("actor A: %w", err)
	}
	envB, err := parseVerbEnvelope([]string{"add", string(s.kind)}, b.out)
	if err != nil { //coverage:ignore defensive: see actor A above
		return fmt.Errorf("actor B: %w", err)
	}

	return s.reconcile(wtA, envA, envB)
}

// reconcile merges actor-b into actor-a's worktree and, only if the
// two adds actually collided on the same id, confirms `aiwf check`
// surfaces it and `aiwf reallocate` resolves it. Split out of Run so
// a test can drive it directly with a deterministically-non-colliding
// pair of envelopes (real concurrent racing in this environment
// collides reliably enough that the "no collision" branch is
// otherwise never exercised).
func (s *CrossWorktreeIDRaceScenario) reconcile(wtA string, envA, envB verbEnvelope) error {
	if envA.Status != "ok" || envB.Status != "ok" {
		return fmt.Errorf("actor add did not report ok: a=%+v b=%+v", envA, envB)
	}

	if err := runGit(wtA, "merge", "-q", "--no-edit", "actor-b"); err != nil { //coverage:ignore defensive: the two actors' adds touch disjoint paths (distinct fixed slugs), so this merge is always a clean fast path with no realistic conflict
		return fmt.Errorf("merging actor-b into actor-a's worktree: %w", err)
	}

	collided := envA.Metadata.EntityID == envB.Metadata.EntityID
	s.collided = s.collided || collided
	if !collided {
		s.violations = append(s.violations, classifyCrossWorktreeRace(false, nil, false, "", nil)...)
		return nil
	}

	checkEnv, err := runAiwfJSON(s.aiwfBin, wtA, "check")
	if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestCrossWorktreeIDRaceScenario_RealBinary_ErrorsWhenBinaryMissing
		return fmt.Errorf("running aiwf check after the merge: %w", err)
	}

	bPath, err := findEntityFile(wtA, envB.Metadata.EntityID, actorBTitle)
	if err != nil { //coverage:ignore defensive: findEntityFile's own not-found branch is unit-tested directly in cross_worktree_id_race_classify_test.go; a real collision always leaves actor B's file in place under its known, fixed slug
		return fmt.Errorf("locating actor B's colliding entity file: %w", err)
	}
	reallocEnv, err := runAiwfJSON(s.aiwfBin, wtA, "reallocate", bPath)
	if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestCrossWorktreeIDRaceScenario_RealBinary_ErrorsWhenBinaryMissing
		return fmt.Errorf("running aiwf reallocate: %w", err)
	}

	postCheckEnv, err := runAiwfJSON(s.aiwfBin, wtA, "check")
	if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestCrossWorktreeIDRaceScenario_RealBinary_ErrorsWhenBinaryMissing
		return fmt.Errorf("running aiwf check after reallocate: %w", err)
	}

	s.violations = append(s.violations, classifyCrossWorktreeRace(true, checkEnv.Findings, true, reallocEnv.Status, postCheckEnv.Findings)...)
	return nil
}

// Verify returns every violation Run collected.
func (s *CrossWorktreeIDRaceScenario) Verify(_ string) []Violation {
	return s.violations
}

// rawAddResult is one actor's unparsed `aiwf add` subprocess result.
type rawAddResult struct {
	execErr error
	out     []byte
}

// launchAddIn runs `aiwf add <kind> --title <title>` in dir and
// returns the raw subprocess result, unparsed. Package-level (not a
// method) since it doesn't need any scenario state beyond its
// explicit parameters — the caller supplies aiwfBin so it isn't
// tied to one scenario type.
func launchAddIn(aiwfBin, dir string, kind entity.Kind, title string) rawAddResult {
	cmd := exec.Command(aiwfBin, //nolint:gosec // aiwfBin is a path this package's own BuildBinary just produced, not attacker-controlled input
		"add", string(kind),
		"--title", title,
		"--body", "cross-worktree id-race stress actor",
		"--format=json",
	)
	cmd.Dir = dir
	out, err := cmd.Output()
	return rawAddResult{execErr: err, out: out}
}

// findEntityFile walks root for a file named exactly "<id>-<slug>.md"
// and returns its path relative to root — used to disambiguate two
// colliding entities that share the same id but different slugs.
// Root-relative because `aiwf reallocate` resolves its path argument
// against the repo root (or cwd), not an absolute filesystem path —
// passing one back verbatim gets refused as "entity not found".
func findEntityFile(root, id, slug string) (string, error) {
	want := id + "-" + slug + ".md"
	var found string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil { //coverage:ignore defensive: WalkDir only surfaces a per-entry error on a filesystem race (deleted mid-walk) or permission error, neither reproducible against a scenario-managed disposable repo
			return walkErr
		}
		if !d.IsDir() && d.Name() == want {
			found = path
		}
		return nil
	})
	if err != nil { //coverage:ignore defensive: see the walkErr callback above — WalkDir's own returned error mirrors it
		return "", fmt.Errorf("walking %s for entity file %s: %w", root, want, err)
	}
	if found == "" {
		return "", fmt.Errorf("entity file %s not found under %s", want, root)
	}
	rel, err := filepath.Rel(root, found)
	if err != nil { //coverage:ignore defensive: found is always a descendant of root (WalkDir only yields paths under root), so Rel cannot fail here
		return "", fmt.Errorf("relativizing %s against %s: %w", found, root, err)
	}
	return rel, nil
}

// classifyCrossWorktreeRace judges one attempt's outcome. When no
// collision occurred, the attempt is a benign no-op (per this
// scenario's own accepted-outcome framing) — never a violation. When
// one did occur, every one of these must hold: aiwf check surfaced
// it as CodeIDsUnique, a reallocate was actually attempted and
// reported "ok", and a follow-up check no longer carries
// CodeIDsUnique.
func classifyCrossWorktreeRace(collided bool, checkFindings []verbEnvelopeFinding, attemptedReallocate bool, reallocateStatus string, postCheckFindings []verbEnvelopeFinding) []Violation {
	if !collided {
		return nil
	}
	var violations []Violation
	if !hasFindingCode(checkFindings, check.CodeIDsUnique) {
		violations = append(violations, Violation{Message: "a real cross-worktree id collision occurred but aiwf check did not surface it as " + check.CodeIDsUnique})
	}
	if !attemptedReallocate {
		violations = append(violations, Violation{Message: "a real cross-worktree id collision occurred but no aiwf reallocate was attempted to resolve it"})
		return violations
	}
	if reallocateStatus != "ok" {
		violations = append(violations, Violation{Message: fmt.Sprintf("aiwf reallocate did not cleanly resolve the collision (status=%s)", reallocateStatus)})
	}
	if hasFindingCode(postCheckFindings, check.CodeIDsUnique) {
		violations = append(violations, Violation{Message: check.CodeIDsUnique + " finding still present after aiwf reallocate"})
	}
	return violations
}

// hasFindingCode reports whether findings contains one with the
// given code, regardless of severity.
func hasFindingCode(findings []verbEnvelopeFinding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}
