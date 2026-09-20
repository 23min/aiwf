package policies

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/config"
)

const testRefusedAddr = "noreply@example.invalid"

// TestDetectRefusedCoauthors pins the per-commit verdict and the report
// it produces: which commits are named, and in the order handed in.
func TestDetectRefusedCoauthors(t *testing.T) {
	t.Parallel()

	refused := map[string]bool{testRefusedAddr: true}
	offending := "Co-Authored-By: Agent <" + testRefusedAddr + ">\n"

	cases := []struct {
		name    string
		commits []coauthorCommit
		want    []string // violating SHAs, in the order expected
	}{
		{
			name:    "a refused co-author violates",
			commits: []coauthorCommit{{SHA: "a1", Subject: "feat: thing", Trailer: offending}},
			want:    []string{"a1"},
		},
		{
			// One rule covers every block naming no refused address; which
			// blocks those are is RefusedCoauthorIn's own decision.
			name:    "a commit naming no refused address passes",
			commits: []coauthorCommit{{SHA: "b2", Subject: "feat: thing", Trailer: "Co-Authored-By: A Person <person@example.invalid>\n"}},
			want:    nil,
		},
		{
			name: "every offending commit is named, in the order handed in",
			commits: []coauthorCommit{
				{SHA: "d4", Subject: "one", Trailer: offending},
				{SHA: "c3", Subject: "clean", Trailer: ""},
				{SHA: "e5", Subject: "two", Trailer: offending},
			},
			want: []string{"d4", "e5"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := detectRefusedCoauthors(tc.commits, refused)
			if len(got) != len(tc.want) {
				t.Fatalf("detectRefusedCoauthors() returned %d violations, want %d: %+v", len(got), len(tc.want), got)
			}
			for i, wantSHA := range tc.want {
				if got[i].File != wantSHA {
					t.Errorf("violation %d names commit %q, want %q", i, got[i].File, wantSHA)
				}
				if got[i].Policy != "coauthor-trailer-ban" {
					t.Errorf("violation policy = %q, want %q", got[i].Policy, "coauthor-trailer-ban")
				}
				if !strings.Contains(got[i].Detail, testRefusedAddr) {
					t.Errorf("detail does not name the address: %q", got[i].Detail)
				}
			}
		})
	}
}

// TestParseCoauthorLog pins the scan's parse, including a commit that
// carries no trailer block at all — the common case in any real range.
func TestParseCoauthorLog(t *testing.T) {
	t.Parallel()

	out := "aaa" + coauthorFldSep + "feat: with trailer" + coauthorFldSep +
		"Co-Authored-By: Agent <" + testRefusedAddr + ">\n" + coauthorFldSep + "\n" +
		"bbb" + coauthorFldSep + "chore: bare" + coauthorFldSep + "" + coauthorFldSep + "\n"

	got, err := parseCoauthorLog(out)
	if err != nil {
		t.Fatalf("parseCoauthorLog() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("parseCoauthorLog() returned %d commits, want 2: %+v", len(got), got)
	}
	if got[0].SHA != "aaa" || got[0].Subject != "feat: with trailer" {
		t.Errorf("first record = %+v", got[0])
	}
	if !strings.Contains(got[0].Trailer, testRefusedAddr) {
		t.Errorf("first record lost its trailer block: %q", got[0].Trailer)
	}
	if got[1].SHA != "bbb" || strings.TrimSpace(got[1].Trailer) != "" {
		t.Errorf("second record = %+v, want a bare commit", got[1])
	}
}

// TestParseCoauthorLog_ReportsPartialRecord pins that output which is not a
// whole number of records is reported rather than quietly truncated.
// Reporting the range clean for commits the scan never parsed is the one
// outcome this policy must never produce.
func TestParseCoauthorLog_ReportsPartialRecord(t *testing.T) {
	t.Parallel()

	out := "aaa" + coauthorFldSep + "feat: missing its trailer field" + coauthorFldSep + "\n"

	if _, err := parseCoauthorLog(out); err == nil {
		t.Fatal("expected an error for output that is not a whole number of records, got nil")
	}
}

// TestParseCoauthorLog_EmptyRange pins that an empty range yields no commits
// and no error — the ordinary case for a push that adds nothing.
func TestParseCoauthorLog_EmptyRange(t *testing.T) {
	t.Parallel()

	got, err := parseCoauthorLog("")
	if err != nil {
		t.Fatalf("parseCoauthorLog(\"\") error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("parseCoauthorLog(\"\") = %+v, want no commits", got)
	}
}

// TestPolicyCoauthorTrailerBan_Env drives the env-fed entry point so the
// wrapper body is exercised. Serial (t.Setenv panics under t.Parallel) and
// documented in setup_test.go's skip-list.
func TestPolicyCoauthorTrailerBan_Env(t *testing.T) {
	// Unset base -> no-op.
	t.Setenv("AIWF_COVERAGE_BASE", "")
	vs, err := PolicyCoauthorTrailerBan(t.TempDir())
	if err != nil {
		t.Fatalf("unset base: unexpected error: %v", err)
	}
	if vs != nil {
		t.Fatalf("unset base: want nil violations, got %+v", vs)
	}

	// Set base -> delegates and surfaces the refused co-author.
	root, base := repoWithCoauthorCommit(t,
		"provenance:\n  refuse_coauthors:\n    - "+testRefusedAddr+"\n",
		"Co-Authored-By: Agent <"+testRefusedAddr+">")
	t.Setenv("AIWF_COVERAGE_BASE", base)
	vs, err = PolicyCoauthorTrailerBan(root)
	if err != nil {
		t.Fatalf("set base: unexpected error: %v", err)
	}
	if len(vs) != 1 {
		t.Fatalf("set base: want 1 violation, got %+v", vs)
	}
}

// TestCoauthorTrailerViolations_Fires is the firing fixture: a real repo
// whose aiwf.yaml refuses an address, carrying a commit that names it.
func TestCoauthorTrailerViolations_Fires(t *testing.T) {
	t.Parallel()

	root, base := repoWithCoauthorCommit(t,
		"provenance:\n  refuse_coauthors:\n    - "+testRefusedAddr+"\n",
		"Co-Authored-By: Agent <"+testRefusedAddr+">")

	got, err := coauthorTrailerViolations(root, base)
	if err != nil {
		t.Fatalf("coauthorTrailerViolations() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d violations, want 1: %+v", len(got), got)
	}
	if got[0].Policy != "coauthor-trailer-ban" {
		t.Errorf("policy = %q", got[0].Policy)
	}
	if !strings.Contains(got[0].Detail, testRefusedAddr) {
		t.Errorf("detail does not name the address: %q", got[0].Detail)
	}
}

// TestCoauthorTrailerViolations_Silent pins the three ways this policy
// declines to judge: no comparison point, a repo refusing nothing, and a
// repo with no config at all. Each must pass the same offending commit.
func TestCoauthorTrailerViolations_Silent(t *testing.T) {
	t.Parallel()

	refusing := "provenance:\n  refuse_coauthors:\n    - " + testRefusedAddr + "\n"
	cases := []struct {
		name   string
		config string
		// base overrides the fixture's real base SHA; "-" means "leave it".
		base string
	}{
		// The three no-comparison-point spellings the sibling diff-scoped
		// policies pin. zeroSHA is what a brand-new branch's
		// github.event.before supplies, so it is the arm CI depends on.
		{name: "an empty base is no comparison point", config: refusing, base: ""},
		{name: "an all-zero base is no comparison point", config: refusing, base: zeroSHA},
		{name: "a whitespace-only base is no comparison point", config: refusing, base: "   "},
		{name: "a repo refusing nothing is untouched", config: "tree:\n  allow_paths: []\n", base: "-"},
		{name: "a repo with no config is untouched", config: "", base: "-"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, base := repoWithCoauthorCommit(t, tc.config,
				"Co-Authored-By: Agent <"+testRefusedAddr+">")
			if tc.base != "-" {
				base = tc.base
			}
			got, err := coauthorTrailerViolations(root, base)
			if err != nil {
				t.Fatalf("coauthorTrailerViolations() error: %v", err)
			}
			if len(got) != 0 {
				t.Errorf("got %d violations, want 0: %+v", len(got), got)
			}
		})
	}
}

// TestCoauthorTrailerViolations_UnresolvableBase pins that an unusable base
// ref is reported rather than silently passing — a silent pass would read
// as "no offending commits" for a range that was never walked.
func TestCoauthorTrailerViolations_UnresolvableBase(t *testing.T) {
	t.Parallel()

	root, _ := repoWithCoauthorCommit(t,
		"provenance:\n  refuse_coauthors:\n    - "+testRefusedAddr+"\n",
		"Co-Authored-By: Agent <"+testRefusedAddr+">")

	if _, err := coauthorTrailerViolations(root, "no-such-ref"); err == nil {
		t.Fatal("expected an error for an unresolvable base ref, got nil")
	}
}

// TestShortSHA pins both arms: a full SHA abbreviates, a fixture-length one
// is returned as written.
func TestShortSHA(t *testing.T) {
	t.Parallel()

	if got := shortSHA("0123456789abcdef"); got != "012345678" {
		t.Errorf("shortSHA(full) = %q, want %q", got, "012345678")
	}
	if got := shortSHA("abc"); got != "abc" {
		t.Errorf("shortSHA(short) = %q, want %q", got, "abc")
	}
}

// repoWithCoauthorCommit builds a git repo carrying cfg as aiwf.yaml (when
// non-empty), a base commit, and a second commit whose message ends in the
// given trailer paragraph. It returns the repo root and the base commit SHA.
func repoWithCoauthorCommit(t *testing.T, cfg, trailer string) (root, base string) {
	t.Helper()
	return repoWithCoauthorCommitSubject(t, cfg, "feat: the change", trailer)
}

// repoWithCoauthorCommitSubject is repoWithCoauthorCommit with the second
// commit's subject chosen by the caller.
func repoWithCoauthorCommitSubject(t *testing.T, cfg, subject, trailer string) (root, base string) {
	t.Helper()

	root = t.TempDir()
	gitInit(t, root)
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}

	if cfg != "" {
		if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(cfg), 0o644); err != nil {
			t.Fatalf("writing config: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("writing base file: %v", err)
	}
	run("add", "-A")
	run("commit", "-q", "-m", "chore: base")
	base = run("rev-parse", "HEAD")

	if err := os.WriteFile(filepath.Join(root, "work.txt"), []byte("work\n"), 0o644); err != nil {
		t.Fatalf("writing work file: %v", err)
	}
	run("add", "-A")
	run("commit", "-q", "-m", subject, "-m", trailer)
	return root, base
}

// TestPolicy_CoauthorTrailerBan is the CI gate entry point. It runs the
// diff-scoped ban against the live tree using the base ref supplied via
// AIWF_COVERAGE_BASE. Without a base (the default in the broad
// `go test ./...` job) it skips — the authoritative invocations are the
// dedicated CI coverage-gate step and `make coverage-gate`.
func TestPolicy_CoauthorTrailerBan(t *testing.T) {
	t.Parallel()
	if os.Getenv("AIWF_COVERAGE_BASE") == "" {
		t.Skip("AIWF_COVERAGE_BASE unset; run via `make coverage-gate` or the CI coverage-gate step")
	}
	runPolicy(t, PolicyCoauthorTrailerBan)
}

// TestCoauthorTrailerBan_WiredIntoCoverageGate pins this policy into the
// coverage-gate run-pattern of both the CI workflow and the Makefile
// target. Without it a future edit could drop the gate from the run set
// and it would silently never fire.
//
// Scoped to the run-pattern line via coverageGateRunLine rather than a
// file-wide substring, which would also be satisfied by an incidental
// mention of the name elsewhere in the file.
func TestCoauthorTrailerBan_WiredIntoCoverageGate(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	const testName = "CoauthorTrailerBan"

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

// TestCoauthorTrailerViolations_FoldedValue pins that a trailer whose value
// is continued on an indented line is caught here too.
//
// The commit-msg hook reads its block through `git interpret-trailers
// --parse`, which joins the continuation back onto its key; a scan that does
// not unfold leaves the address on a line of its own, where the line-based
// matcher cannot see it. Without unfolding this commit is refused at
// composition and passed here — the two layers disagreeing about one message,
// which is the divergence sharing the matcher is meant to prevent.
func TestCoauthorTrailerViolations_FoldedValue(t *testing.T) {
	t.Parallel()

	root, base := repoWithCoauthorCommit(t,
		"provenance:\n  refuse_coauthors:\n    - "+testRefusedAddr+"\n",
		"Co-Authored-By: Agent\n  <"+testRefusedAddr+">")

	got, err := coauthorTrailerViolations(root, base)
	if err != nil {
		t.Fatalf("coauthorTrailerViolations() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("a folded trailer value was not caught: got %d violations, want 1: %+v", len(got), got)
	}
}

// TestCoauthorTrailerBan_RepoConfigArmsTheBan pins that this repo's own
// aiwf.yaml still names at least one address.
//
// Both layers read that list and both refuse nothing without it, so deleting
// or mistyping the block disarms the ban completely while every other test
// stays green. CLAUDE.md states the rule in prose; this is what ties the
// prose to the configuration that makes it true.
func TestCoauthorTrailerBan_RepoConfigArmsTheBan(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(repoRoot(t))
	if err != nil {
		t.Fatalf("loading the repo's aiwf.yaml: %v", err)
	}
	if len(cfg.RefusedCoauthors()) == 0 {
		t.Error("aiwf.yaml names no provenance.refuse_coauthors address, so the " +
			"Co-Authored-By ban refuses nothing — restore the address or retire the rule in CLAUDE.md")
	}
}

// TestCoauthorTrailerViolations_SeparatorBytesInSubject pins the scan against
// a subject carrying the control bytes an ad-hoc separator would use.
//
// Git permits any byte but NUL in a commit message, so a subject can contain
// them. Where the scan separates on such a byte, the subject's remainder
// shifts into the next field and the trailer is read from the wrong place —
// the commit is refused at composition and reported clean here, which is the
// one outcome this policy must never produce.
func TestCoauthorTrailerViolations_SeparatorBytesInSubject(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		subject string
	}{
		{name: "a record-separator byte", subject: "feat: a\x1eb"},
		{name: "a field-separator byte", subject: "feat: a\x1fb"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, base := repoWithCoauthorCommitSubject(t,
				"provenance:\n  refuse_coauthors:\n    - "+testRefusedAddr+"\n",
				tc.subject,
				"Co-Authored-By: Agent <"+testRefusedAddr+">")

			got, err := coauthorTrailerViolations(root, base)
			if err != nil {
				t.Fatalf("coauthorTrailerViolations() error: %v", err)
			}
			if len(got) != 1 {
				t.Fatalf("a subject carrying %q hid the refused co-author: got %d violations, want 1: %+v",
					tc.subject, len(got), got)
			}
		})
	}
}
