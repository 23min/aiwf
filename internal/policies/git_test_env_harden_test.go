package policies

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPolicy_GitTestEnvHardened is the live-repo chokepoint: every
// exec-bearing test package under internal/ must seed its TestMain with
// testsupport.HardenGitTestEnv(). Zero violations expected. See G-0250
// and G-0251.
func TestPolicy_GitTestEnvHardened(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyGitTestEnvHardened)
}

// writeGoFixture writes a Go source file under root at the
// forward-slash repo-relative path, creating parent dirs.
func writeGoFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

// TestPolicyGitTestEnvHardened_Branches drives the policy over synthetic
// internal/ trees so every branch is exercised: exec-bearing packages
// with and without the harden call, a non-exec package (skipped), the
// exec.CommandContext constructor, a missing setup_test.go, a
// setup_test.go that has no TestMain, and unparseable sources (the
// parse-error handlers). The fixtures intentionally carry non-matching
// call expressions (os.Getenv, a bare call, m.Run) so the AST walk
// traverses the non-exec / non-testsupport selector branches too.
func TestPolicyGitTestEnvHardened_Branches(t *testing.T) {
	t.Parallel()

	// An exec-bearing test file. The bare doThing() call and the
	// os.Getenv selector exercise isExecCommandCall's non-matching
	// returns; exec.Command is the positive match.
	const execTestCommand = `package foo

func TestThing(t *testing.T) {
	_ = exec.Command("git", "status")
	_ = os.Getenv("X")
	doThing()
}
`
	const execTestCommandContext = `package foo

func TestThing(t *testing.T) {
	_ = exec.CommandContext(ctx, "git", "status")
}
`
	const noExecTest = `package foo

func TestThing(t *testing.T) {
	if true {
		doThing()
	}
}
`
	const setupWithHarden = `package foo

func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	testsupport.HardenGitTestEnv()
	os.Exit(m.Run())
}
`
	const setupWithoutHarden = `package foo

func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Exit(m.Run())
}
`
	const setupNoTestMain = `package foo

func helper() {}
`
	// TestMain whose body has a bare call (Fun is *ast.Ident) and a
	// testsupport call that isn't HardenGitTestEnv — exercises
	// testMainCallsHarden's two non-matching AST branches.
	const setupOtherCalls = `package foo

func TestMain(m *testing.M) {
	doThing()
	testsupport.Other()
	os.Exit(m.Run())
}
`
	const malformedGo = `package foo

func {{{ this does not parse
`

	cases := []struct {
		name       string
		files      map[string]string // rel path -> content
		wantCount  int
		wantPolicy string
		wantErr    bool
	}{
		{
			name: "exec with harden passes",
			files: map[string]string{
				"internal/foo/foo_test.go":   execTestCommand,
				"internal/foo/setup_test.go": setupWithHarden,
			},
			wantCount: 0,
		},
		{
			name: "exec without harden violates",
			files: map[string]string{
				"internal/foo/foo_test.go":   execTestCommand,
				"internal/foo/setup_test.go": setupWithoutHarden,
			},
			wantCount:  1,
			wantPolicy: "git-test-env-harden",
		},
		{
			name: "CommandContext without harden violates",
			files: map[string]string{
				"internal/foo/foo_test.go":   execTestCommandContext,
				"internal/foo/setup_test.go": setupWithoutHarden,
			},
			wantCount:  1,
			wantPolicy: "git-test-env-harden",
		},
		{
			name: "no exec is skipped",
			files: map[string]string{
				"internal/foo/foo_test.go":   noExecTest,
				"internal/foo/setup_test.go": setupWithoutHarden,
			},
			wantCount: 0,
		},
		{
			name: "exec with missing setup_test.go violates",
			files: map[string]string{
				"internal/foo/foo_test.go": execTestCommand,
			},
			wantCount:  1,
			wantPolicy: "git-test-env-harden",
		},
		{
			name: "exec with setup lacking TestMain violates",
			files: map[string]string{
				"internal/foo/foo_test.go":   execTestCommand,
				"internal/foo/setup_test.go": setupNoTestMain,
			},
			wantCount:  1,
			wantPolicy: "git-test-env-harden",
		},
		{
			// Malformed test file fails the exec scan's parser.ParseFile
			// (dirExecsSubprocess) and the error propagates.
			name: "unparseable test file surfaces an error",
			files: map[string]string{
				"internal/foo/zz_test.go": malformedGo,
			},
			wantErr: true,
		},
		{
			// a_test.go (sorts first) carries the exec match so the scan
			// returns before reaching setup_test.go; the malformed
			// setup_test.go then fails testMainCallsHarden's parse.
			name: "unparseable setup_test.go surfaces an error",
			files: map[string]string{
				"internal/foo/a_test.go":     execTestCommand,
				"internal/foo/setup_test.go": malformedGo,
			},
			wantErr: true,
		},
		{
			// TestMain with a bare call and a non-Harden testsupport call
			// exercises testMainCallsHarden's two non-matching AST
			// branches; neither matches, so the package violates.
			name: "exec with setup calling other funcs (not Harden) violates",
			files: map[string]string{
				"internal/foo/foo_test.go":   execTestCommand,
				"internal/foo/setup_test.go": setupOtherCalls,
			},
			wantCount:  1,
			wantPolicy: "git-test-env-harden",
		},
		{
			// A testdata/ directory is skipped by the walk even though it
			// holds an exec-bearing _test.go with no setup_test.go (which
			// would otherwise violate). Exercises the testdata SkipDir.
			name: "testdata dir is skipped",
			files: map[string]string{
				"internal/testdata/skip_test.go": execTestCommand,
			},
			wantCount: 0,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for rel, content := range tc.files {
				writeGoFixture(t, root, rel, content)
			}
			vs, err := PolicyGitTestEnvHardened(root)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got nil (violations: %+v)", vs)
				}
				return
			}
			if err != nil {
				t.Fatalf("PolicyGitTestEnvHardened: %v", err)
			}
			if len(vs) != tc.wantCount {
				t.Fatalf("got %d violations, want %d: %+v", len(vs), tc.wantCount, vs)
			}
			if tc.wantCount > 0 && vs[0].Policy != tc.wantPolicy {
				t.Errorf("violation Policy = %q, want %q", vs[0].Policy, tc.wantPolicy)
			}
		})
	}
}
