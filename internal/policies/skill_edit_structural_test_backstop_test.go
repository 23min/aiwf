package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Synthetic skill paths for the pure-detector table. They use an
// obviously-fictional skill name so the strings here never accidentally
// "back" a real embedded-rituals skill when policyTestRefs concatenates
// this very file.
const (
	fakeSkillA = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-fictional-alpha/SKILL.md"
	fakeSkillB = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-fictional-bravo/SKILL.md"
)

// TestDetectUnbackedSkillEdits drives the pure core directly with
// hand-built inputs — AC-1 (fires on an unbacked edit) and AC-2 (silent
// when backed; discriminates per path on mixed input).
func TestDetectUnbackedSkillEdits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		changed   []string
		refs      string
		wantFiles []string
	}{
		{
			name:      "unbacked edit fires (AC-1)",
			changed:   []string{fakeSkillA},
			refs:      "package policies\n// no reference to any skill path\n",
			wantFiles: []string{fakeSkillA},
		},
		{
			name:      "backed edit is silent (AC-2)",
			changed:   []string{fakeSkillA},
			refs:      "const fixture = \"" + fakeSkillA + "\"\n",
			wantFiles: nil,
		},
		{
			name:      "mixed input fires only for the unbacked path (AC-2)",
			changed:   []string{fakeSkillA, fakeSkillB},
			refs:      "const fixture = \"" + fakeSkillB + "\"\n",
			wantFiles: []string{fakeSkillA},
		},
		{
			name:      "no changed skills is silent",
			changed:   nil,
			refs:      "",
			wantFiles: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := violationFiles(detectUnbackedSkillEdits(tt.changed, tt.refs))
			if !equalStrings(got, tt.wantFiles) {
				t.Errorf("violation files = %v, want %v", got, tt.wantFiles)
			}
			for _, v := range detectUnbackedSkillEdits(tt.changed, tt.refs) {
				if v.Policy != "skill-edit-structural-test-backstop" {
					t.Errorf("violation Policy = %q, want skill-edit-structural-test-backstop", v.Policy)
				}
			}
		})
	}
}

// violationFiles returns the File field of each violation, preserving
// order (the detector emits in the caller-supplied changed order, which
// changedSkillFiles sorts).
func violationFiles(vs []Violation) []string {
	if len(vs) == 0 {
		return nil
	}
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		out = append(out, v.File)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// seamSkillRel is the fictional embedded-rituals SKILL.md path the git
// fixtures add or modify.
const seamSkillRel = skillRitualsDir + "/plugins/aiwf-extensions/skills/aiwfx-fictional-seam/SKILL.md"

// skillFixtureBase inits a throwaway git repo with a base commit and
// returns the root, a git runner, a file writer, and the base SHA. The
// HEAD mutation is the caller's to stage and commit.
func skillFixtureBase(t *testing.T) (root string, runGit func(...string) string, writeFile func(string, string), baseSHA string) {
	t.Helper()
	root = t.TempDir()
	runGit = repoGitRunner(t, root)
	writeFile = repoFileWriter(t, root)
	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "aiwf-test")
	writeFile("go.mod", "module example.com/seam\n\ngo 1.24\n")
	writeFile("README.md", "base\n")
	runGit("add", "-A")
	runGit("commit", "-m", "base")
	baseSHA = trimLine(runGit("rev-parse", "HEAD"))
	return root, runGit, writeFile, baseSHA
}

// TestSkillEditBackstopViolations_Seam drives the full IO shell end to
// end (AC-3): a synthetic git repo whose HEAD commit edits an
// embedded-rituals SKILL.md, run through `git diff <base>` →
// changedSkillFiles → policyTestRefs → detector. It proves the resolver
// is wired, not just the pure detector layer.
func TestSkillEditBackstopViolations_Seam(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		policyTest string // contents of internal/policies/seam_test.go in HEAD
		wantFiles  []string
	}{
		{
			name:       "unbacked skill edit fires",
			policyTest: "package policies\n\n// references no skill path\n",
			wantFiles:  []string{seamSkillRel},
		},
		{
			name:       "backed skill edit is silent",
			policyTest: "package policies\n\nconst seamFixture = \"" + seamSkillRel + "\"\n",
			wantFiles:  nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, runGit, writeFile, baseSHA := skillFixtureBase(t)
			// HEAD: add the skill edit + a policies test file that either
			// does or does not reference the edited path.
			writeFile(seamSkillRel, "# fictional seam skill\n\nprescriptive content\n")
			writeFile("internal/policies/seam_test.go", tt.policyTest)
			runGit("add", "-A")
			runGit("commit", "-m", "head")

			vs, err := skillEditBackstopViolations(root, baseSHA)
			if err != nil {
				t.Fatalf("skillEditBackstopViolations: %v", err)
			}
			if got := violationFiles(vs); !equalStrings(got, tt.wantFiles) {
				t.Errorf("violation files = %v, want %v", got, tt.wantFiles)
			}
		})
	}
}

// TestSkillEditBackstopViolations_BaseUnresolvable confirms the gate
// no-ops on an empty or all-zero base ref (AC-3: inert without a base) —
// the broad `go test ./...` job and a brand-new branch's all-zero
// github.event.before both hit this path.
func TestSkillEditBackstopViolations_BaseUnresolvable(t *testing.T) {
	t.Parallel()
	root, runGit, writeFile, _ := skillFixtureBase(t)
	writeFile(seamSkillRel, "# x\n")
	runGit("add", "-A")
	runGit("commit", "-m", "head")

	for _, base := range []string{"", zeroSHA} {
		vs, err := skillEditBackstopViolations(root, base)
		if err != nil {
			t.Fatalf("base %q: unexpected error: %v", base, err)
		}
		if len(vs) != 0 {
			t.Errorf("base %q: got %d violations, want 0", base, len(vs))
		}
	}
}

// TestSkillEditBackstopViolations_Errors exercises the IO core's
// error and early-return branches: a bad base ref (git diff fails), a
// HEAD that touches no skill (the len==0 short-circuit), and a tree with
// no internal/policies/ directory (policyTestRefs read fails).
func TestSkillEditBackstopViolations_Errors(t *testing.T) {
	t.Parallel()

	t.Run("git diff error on bad base ref", func(t *testing.T) {
		t.Parallel()
		root, runGit, writeFile, _ := skillFixtureBase(t)
		writeFile(seamSkillRel, "# x\n")
		runGit("add", "-A")
		runGit("commit", "-m", "head")
		_, err := skillEditBackstopViolations(root, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
		if err == nil {
			t.Fatal("want error for nonexistent base ref, got nil")
		}
	})

	t.Run("no changed skill is silent", func(t *testing.T) {
		t.Parallel()
		root, runGit, writeFile, baseSHA := skillFixtureBase(t)
		// HEAD edits a non-skill file → changedSkillFiles returns empty →
		// the len==0 guard returns before policyTestRefs is consulted.
		writeFile("README.md", "base\nmore\n")
		runGit("add", "-A")
		runGit("commit", "-m", "head")
		vs, err := skillEditBackstopViolations(root, baseSHA)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(vs) != 0 {
			t.Errorf("want 0 violations, got %+v", vs)
		}
	})

	t.Run("policyTestRefs error when internal/policies absent", func(t *testing.T) {
		t.Parallel()
		root, runGit, writeFile, baseSHA := skillFixtureBase(t)
		// HEAD adds a skill but no internal/policies/ dir → os.ReadDir
		// fails when policyTestRefs tries to scan the test sources.
		writeFile(seamSkillRel, "# x\n")
		runGit("add", "-A")
		runGit("commit", "-m", "head")
		_, err := skillEditBackstopViolations(root, baseSHA)
		if err == nil {
			t.Fatal("want error for missing internal/policies dir, got nil")
		}
	})
}

// TestPolicyTestRefs confirms the scan concatenates only _test.go files,
// skipping production .go sources and subdirectories (the IsDir /
// non-test continue branch).
func TestPolicyTestRefs(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "policies")
	if err := os.MkdirAll(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("a_test.go", "TEST_MARKER\n")
	mustWrite("b.go", "PROD_MARKER\n") // non-test → skipped

	refs, err := policyTestRefs(root)
	if err != nil {
		t.Fatalf("policyTestRefs: %v", err)
	}
	if !strings.Contains(refs, "TEST_MARKER") {
		t.Error("want _test.go contents included")
	}
	if strings.Contains(refs, "PROD_MARKER") {
		t.Error("non-test .go source must be skipped")
	}
}

// TestPolicySkillEditStructuralTestBackstop_Env drives the env-fed entry
// point so the wrapper body is exercised during profile generation.
// Serial (t.Setenv panics under t.Parallel) and documented in
// setup_test.go's skip-list.
func TestPolicySkillEditStructuralTestBackstop_Env(t *testing.T) {
	// Unset base → no-op.
	t.Setenv("AIWF_COVERAGE_BASE", "")
	vs, err := PolicySkillEditStructuralTestBackstop(t.TempDir())
	if err != nil {
		t.Fatalf("unset base: unexpected error: %v", err)
	}
	if vs != nil {
		t.Fatalf("unset base: want nil violations, got %+v", vs)
	}

	// Set base → delegates and surfaces the unbacked skill edit.
	root, runGit, writeFile, baseSHA := skillFixtureBase(t)
	writeFile(seamSkillRel, "# fictional seam skill\n\nprescriptive content\n")
	writeFile("internal/policies/seam_test.go", "package policies\n\n// references no skill path\n")
	runGit("add", "-A")
	runGit("commit", "-m", "head")

	t.Setenv("AIWF_COVERAGE_BASE", baseSHA)
	vs, err = PolicySkillEditStructuralTestBackstop(root)
	if err != nil {
		t.Fatalf("set base: unexpected error: %v", err)
	}
	if len(vs) != 1 || vs[0].File != seamSkillRel {
		t.Fatalf("set base: want one violation for %s, got %+v", seamSkillRel, vs)
	}
}

// TestSkillEditBackstop_WiredIntoCoverageGate (AC-4) pins that the gate
// actually runs at the integration boundary: the policy test is named in
// the coverage-gate run-pattern of both the CI workflow and the Makefile
// target, alongside the other profile-driven gates. Without this a future
// edit could drop the gate from the run set and it would silently never
// fire.
func TestSkillEditBackstop_WiredIntoCoverageGate(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	const testName = "SkillEditStructuralTestBackstop"

	for _, f := range []string{".github/workflows/go.yml", "Makefile"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		line := coverageGateRunLine(t, f, string(data))
		if !strings.Contains(line, testName) {
			t.Errorf("%s: coverage-gate run-pattern does not include %s:\n  %s", f, testName, line)
		}
	}
}

// coverageGateRunLine returns the single line in content that invokes the
// profile-driven gates via `go test -run '^TestPolicy_(...)$'`. Scoping
// the assertion to that exact line (rather than a flat file-wide
// substring) keeps the check structural: it pins the run-set, not an
// incidental mention of the test name elsewhere in the file.
func coverageGateRunLine(t *testing.T, fname, content string) string {
	t.Helper()
	var found []string
	for _, ln := range strings.Split(content, "\n") {
		if strings.Contains(ln, "-run '^TestPolicy_(") {
			found = append(found, strings.TrimSpace(ln))
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s: want exactly one coverage-gate run-pattern line, found %d: %v", fname, len(found), found)
	}
	return found[0]
}

// TestSkillEditBackstop_DocumentedInClaudeMd (AC-5) pins that the
// chokepoint is documented on both CLAUDE.md surfaces it belongs to, each
// assertion scoped to its named section (not a flat file-wide grep): the
// "What's enforced and where" table names the engine file, and the
// "Ritual content authoring" section names the policy as the mechanical
// backstop that replaces operator vigilance (G-0220 tertiary item).
func TestSkillEditBackstop_DocumentedInClaudeMd(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	content := string(data)

	// markdownSection (defined in m0134_claude_md_test_running_sections.go)
	// takes the full heading line, with its `#` markers.
	enforce := markdownSection(content, "### What's enforced and where")
	if enforce == "" {
		t.Fatal(`CLAUDE.md has no "### What's enforced and where" section`)
	}
	if !strings.Contains(enforce, "skill_edit_structural_test_backstop.go") {
		t.Error(`"What's enforced and where" table must name the backstop policy's engine file`)
	}

	authoring := markdownSection(content, "## Ritual content authoring")
	if authoring == "" {
		t.Fatal(`CLAUDE.md has no "## Ritual content authoring" section`)
	}
	if !strings.Contains(authoring, "skill-edit-structural-test-backstop") {
		t.Error(`"Ritual content authoring" must name the skill-edit-structural-test-backstop chokepoint`)
	}
}

// TestPolicy_SkillEditStructuralTestBackstop is the CI gate entry point.
// It runs the diff-scoped backstop against the live tree using the base
// ref supplied via AIWF_COVERAGE_BASE. Without a base (the default in the
// broad `go test ./...` job) it skips — the authoritative invocation is
// the dedicated CI coverage-gate step and `make coverage-gate`.
func TestPolicy_SkillEditStructuralTestBackstop(t *testing.T) {
	t.Parallel()
	if os.Getenv("AIWF_COVERAGE_BASE") == "" {
		t.Skip("AIWF_COVERAGE_BASE unset; run via `make coverage-gate` or the CI coverage-gate step")
	}
	runPolicy(t, PolicySkillEditStructuralTestBackstop)
}
