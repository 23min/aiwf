package status

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/testsupport"
	"github.com/23min/aiwf/internal/tree"
)

func TestWorktreeStatusUsesConfiguredTrunk(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		branch        string
		trunkRef      string
		merged        bool
		unrelatedMain bool
		trunkBranch   string
		noLocalTrunk  bool
	}{
		{name: "unmerged work without main", branch: "epic/E-9001-work", trunkRef: "refs/heads/develop"},
		{name: "merged work", branch: "epic/E-9001-work", trunkRef: "refs/heads/develop", merged: true},
		{name: "unrelated main cannot hide unmerged work", branch: "epic/E-9001-work", trunkRef: "refs/heads/develop", unrelatedMain: true},
		{name: "trailers identify ordinary branch", branch: "feature/work", trunkRef: "refs/heads/develop"},
		{name: "branch name with slash", branch: "epic/E-9001-work", trunkRef: "refs/heads/release/stable", trunkBranch: "release/stable"},
		{name: "remote branch name with slash", branch: "epic/E-9001-work", trunkRef: "refs/remotes/upstream/release/stable", trunkBranch: "release/stable"},
		{name: "local merge with lagging remote", branch: "epic/E-9001-work", trunkRef: "refs/remotes/upstream/develop", merged: true},
		{name: "remote comparison without local trunk", branch: "epic/E-9001-work", trunkRef: "refs/remotes/upstream/develop", noLocalTrunk: true},
		{name: "remote trunk", branch: "epic/E-9001-work", trunkRef: "refs/remotes/upstream/develop"},
		{name: "missing configured ref", branch: "epic/E-9001-work", trunkRef: "refs/heads/missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			trunkBranch := tc.trunkBranch
			if trunkBranch == "" {
				trunkBranch = "develop"
			}
			gitDo(t, root, "init", "-q", "-b", trunkBranch)
			if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("allocate:\n  trunk: "+tc.trunkRef+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			writeEpic(t, root, "E-9001-work", "E-9001", "active")
			writeEpic(t, root, "E-9002-loose", "E-9002", "active")
			gitDo(t, root, "add", "-A")
			gitDo(t, root, "commit", "-qm", "base")
			if strings.HasPrefix(tc.trunkRef, "refs/remotes/") {
				gitDo(t, root, "update-ref", tc.trunkRef, "HEAD")
			}
			wt := filepath.Join(t.TempDir(), "work")
			gitDo(t, root, "worktree", "add", "-qb", tc.branch, wt)
			writeEpic(t, wt, "E-9001-work", "E-9001", "done")
			gitDo(t, wt, "add", "-A")
			gitDo(t, wt, "commit", "-qm", "complete\n\naiwf-verb: promote\naiwf-entity: E-9001\naiwf-to: done")
			if tc.merged {
				gitDo(t, root, "merge", "--ff-only", tc.branch)
			}
			if tc.unrelatedMain {
				gitDo(t, root, "branch", "main", tc.branch)
			}
			if tc.noLocalTrunk {
				gitDo(t, root, "branch", "-m", "parking")
			}
			tr, _, err := tree.Load(t.Context(), root)
			if err != nil {
				t.Fatal(err)
			}
			views, err := BuildWorktreeViews(t.Context(), root, tr)
			if err != nil {
				t.Fatal(err)
			}
			v := viewForBranch(t, views, tc.branch)
			if v.DriverEntityID != "E-9001" {
				t.Fatalf("driver = %q, want E-9001", v.DriverEntityID)
			}
			var out bytes.Buffer
			if err := RenderWorktreeViews(&out, []WorktreeView{*v}, false); err != nil {
				t.Fatal(err)
			}
			want := "WRAP PENDING"
			if tc.merged {
				want = "SAFE TO REMOVE"
			}
			if tc.trunkRef == "refs/heads/missing" {
				want = "MERGE STATUS UNKNOWN"
			}
			if !strings.Contains(out.String(), want) {
				t.Errorf("want %q in output:\n%s", want, out.String())
			}
			if !tc.merged && strings.Contains(out.String(), "git worktree remove") {
				t.Errorf("unverified cleanup suggestion:\n%s", out.String())
			}
			if tc.trunkRef == "refs/heads/missing" {
				data, err := json.Marshal(v)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Contains(data, []byte(`"ahead_of_trunk":null`)) {
					t.Errorf("unknown count must be null: %s", data)
				}
				return
			}
			if tc.noLocalTrunk {
				for _, view := range views {
					if view.IsTrunk {
						t.Errorf("nonexistent local trunk has a checkout: %+v", view)
					}
				}
				return
			}
			trunk := viewForBranch(t, views, trunkBranch)
			if len(trunk.OtherInFlight) == 0 {
				t.Error("trunk lost other in-flight entities")
			}
			var short strings.Builder
			renderWorktreeShortLines(&short, []WorktreeView{*trunk}, 120, false)
			if !strings.Contains(short.String(), "trunk (no in-flight scope)") {
				t.Errorf("trunk mislabeled: %s", short.String())
			}
			if !tc.merged && (v.CreatedTime.IsZero() || v.LastEntityTime.IsZero()) {
				t.Error("ahead-of-trunk activity timestamps missing")
			}
		})
	}
}

func trunkCount(n int) *int { return &n }

func TestWorktreeTrunkResolution(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		config     string
		remote     bool
		cancel     bool
		wantRef    string
		wantBranch string
		wantError  bool
	}{
		{name: "offline default", wantRef: "refs/heads/main", wantBranch: "main"},
		{name: "local trunk with available default remote", remote: true, wantRef: "refs/heads/main", wantBranch: "main"},
		{name: "explicit missing remote never falls back", config: "allocate:\n  trunk: refs/remotes/upstream/develop\n", wantRef: "refs/remotes/upstream/develop", wantBranch: "develop"},
		{name: "tag has no trunk checkout", config: "allocate:\n  trunk: refs/tags/trunk\n", wantRef: "refs/tags/trunk"},
		{name: "malformed config", config: "allocate: [", wantError: true},
		{name: "git failure", cancel: true, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			gitDo(t, root, "init", "-q", "-b", "main")
			gitDo(t, root, "commit", "--allow-empty", "-qm", "base")
			if tc.config != "" {
				if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(tc.config), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if tc.remote {
				gitDo(t, root, "update-ref", "refs/remotes/origin/main", "HEAD")
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tc.cancel {
				cancel()
			}
			ref, branch, err := worktreeTrunk(ctx, root)
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, wantError %v", err, tc.wantError)
			}
			if ref != tc.wantRef || branch != tc.wantBranch {
				t.Errorf("ref, branch = %q, %q; want %q, %q", ref, branch, tc.wantRef, tc.wantBranch)
			}
		})
	}
}

func TestWorktreeStatusRejectsMalformedTrunkConfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	gitDo(t, root, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("allocate: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildWorktreeViews(t.Context(), root, &tree.Tree{}); err == nil {
		t.Fatal("malformed config must not fall back to main")
	}
}

func TestWorktreeStatusDoesNotOverrideActiveDriverWhenComparisonFails(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	gitDo(t, root, "init", "-q", "-b", "develop")
	writeEpic(t, root, "E-9001-work", "E-9001", "active")
	gitDo(t, root, "add", "-A")
	gitDo(t, root, "commit", "-qm", "base")
	wt := filepath.Join(t.TempDir(), "work")
	gitDo(t, root, "worktree", "add", "-qb", "epic/E-9001-work", wt)
	writeEpic(t, root, "E-9001-work", "E-9001", "done")
	gitDo(t, root, "add", "-A")
	gitDo(t, root, "commit", "-qm", "complete on local trunk")
	for _, trunkRef := range []string{"refs/heads/missing", "refs/heads/develop"} {
		if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("allocate:\n  trunk: "+trunkRef+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		tr, _, err := tree.Load(t.Context(), root)
		if err != nil {
			t.Fatal(err)
		}
		views, err := BuildWorktreeViews(t.Context(), root, tr)
		if err != nil {
			t.Fatal(err)
		}
		v := viewForBranch(t, views, "epic/E-9001-work")
		wantStale := trunkRef == "refs/heads/develop"
		if v.Stale != wantStale {
			t.Errorf("trunkRef=%s: stale=%v", trunkRef, v.Stale)
		}
		if !wantStale && v.DriverStatus != "active" {
			t.Errorf("unknown comparison overrode active driver: %+v", v)
		}
	}
}

func TestStatusReportsMalformedTrunkConfiguration(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	gitDo(t, root, "init", "-q", "-b", "main")
	gitDo(t, root, "commit", "--allow-empty", "-qm", "base")
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("allocate: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Run(root, "json", "", "", false, false, true); got != cliutil.ExitInternal {
		t.Errorf("exit = %d, want %d", got, cliutil.ExitInternal)
	}
}

// Serial because PATH selects a process-boundary stand-in for Git.
func TestWorktreeStatusWithholdsCleanupWhenGitComparisonFails(t *testing.T) {
	for _, output := range []string{"exit 1", "printf 'not-a-count\\n'"} {
		t.Run(output, func(t *testing.T) {
			root := t.TempDir()
			if err := testsupport.WriteExecutable(filepath.Join(root, "git"), []byte("#!/bin/sh\n"+output+"\n")); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", root)
			count := branchAheadOfTrunkCount(t.Context(), root, "refs/heads/develop", "feature/work")
			if count != nil {
				t.Fatalf("failed comparison returned %d", *count)
			}
			var out bytes.Buffer
			v := WorktreeView{DriverEntityID: "E-9001", DriverStatus: "done", Stale: true, AheadOfTrunk: count}
			if err := renderStaleSection(&out, &v, false); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "MERGE STATUS UNKNOWN") || strings.Contains(out.String(), "git worktree remove") {
				t.Errorf("unsafe hint: %s", out.String())
			}
		})
	}
}

func TestWorktreeWithoutDriverIsNotLabeledTrunk(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	if err := renderWorktreeSection(&out, &WorktreeView{Path: "/repo/feature", Branch: "feature/work"}, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "No driver entity") || strings.Contains(out.String(), "(trunk)") {
		t.Errorf("incorrect trunk label: %s", out.String())
	}
}

// Serial because PATH selects the failing Git command at the process boundary.
func TestWorktreeStatusPreservesBranchStateOnComparisonFault(t *testing.T) {
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	gitDo(t, root, "init", "-q", "-b", "main")
	writeEpic(t, root, "E-9001-work", "E-9001", "active")
	gitDo(t, root, "add", "-A")
	gitDo(t, root, "commit", "-qm", "base")
	wt := filepath.Join(t.TempDir(), "work")
	gitDo(t, root, "worktree", "add", "-qb", "epic/E-9001-work", wt)
	writeEpic(t, root, "E-9001-work", "E-9001", "done")
	gitDo(t, root, "add", "-A")
	gitDo(t, root, "commit", "-qm", "complete on trunk")
	binDir := t.TempDir()
	script := "#!/bin/sh\nif [ \"$1\" = rev-list ]; then exit 1; fi\nexec '" + strings.ReplaceAll(realGit, "'", "'\"'\"'") + "' \"$@\"\n"
	if writeErr := testsupport.WriteExecutable(filepath.Join(binDir, "git"), []byte(script)); writeErr != nil {
		t.Fatal(writeErr)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	tr, _, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	views, err := BuildWorktreeViews(t.Context(), root, tr)
	if err != nil {
		t.Fatal(err)
	}
	v := viewForBranch(t, views, "epic/E-9001-work")
	if v.AheadOfTrunk != nil || v.Stale || v.DriverStatus != "active" {
		t.Errorf("comparison failure changed branch state: %+v", v)
	}
}
