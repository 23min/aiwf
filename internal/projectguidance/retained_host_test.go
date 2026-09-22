package projectguidance

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestInstall_EditedUnselectedRouteDoesNotBlockRefreshOrLoseConflict(t *testing.T) {
	t.Parallel()
	for _, host := range []string{"AGENTS.md", "CLAUDE.md"} {
		t.Run(host, func(t *testing.T) {
			t.Parallel()
			snapshot, opts := installationFixture(t)
			root := t.TempDir()
			if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
				t.Fatal(err)
			}
			original := readTree(t, root)
			edited := strings.Replace(original[host], "Handwritten project overrides", "Local project overrides", 1)
			write(t, root, host, edited)
			opts.HostFiles = nil
			for _, other := range []string{"AGENTS.md", "CLAUDE.md"} {
				if other != host {
					opts.HostFiles = append(opts.HostFiles, other)
				}
			}
			snapshot.Documents["packs/go/cobra/guide.md"] = []byte("# Updated policy\n")
			if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
				t.Fatalf("refresh with disabled edited host: %v", err)
			}
			updated := readTree(t, root)
			if updated[host] != edited || updated[".guidance/packs/go/cobra/guide.md"] != "# Updated policy\n" {
				t.Fatal("refresh lost edit or failed to update shared policy")
			}
			opts.HostFiles = append(opts.HostFiles, host)
			if _, err := Install(t.Context(), root, snapshot, opts); !errors.Is(err, ErrInstallConflict) {
				t.Fatalf("reenabling edited host must require reconciliation: %v", err)
			}
			if diff := cmp.Diff(updated, readTree(t, root)); diff != "" {
				t.Fatal(diff)
			}
			write(t, root, host, original[host])
			if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
				t.Fatalf("restored generated route should be accepted: %v", err)
			}
		})
	}
}

func TestInstall_TrackedGuidanceTravelsWithClonesAndWorktrees(t *testing.T) {
	t.Parallel()
	snapshot, opts := installationFixture(t)
	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "main")
	write(t, root, ".guidance/project.md", "# Our overrides\n")
	if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "fixture")
	for _, kind := range []string{"clone", "worktree"} {
		destination := filepath.Join(t.TempDir(), kind)
		if kind == "clone" {
			git(t, root, "clone", "-q", root, destination)
		} else {
			git(t, root, "worktree", "add", "--detach", destination)
		}
		report, err := Inspect(t.Context(), destination)
		if err != nil || report.Revision != snapshot.Commit || len(report.Issues) != 0 {
			t.Fatalf("%s: %+v, %v", kind, report, err)
		}
		for name, content := range readTree(t, root) {
			if strings.HasPrefix(name, ".git/") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(destination, name))
			if err != nil || string(data) != content {
				t.Fatalf("%s lost %s: %v", kind, name, err)
			}
		}
	}
}

func TestInstall_DisabledHostRecoveryDoesNotAcceptUnrecognizedBytes(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"published", "old", "edited", "malformed", "absent", "no target"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			snapshot, opts := installationFixture(t)
			root := t.TempDir()
			if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
				t.Fatal(err)
			}
			generated := readTree(t, root)["AGENTS.md"]
			block, err := ownedBytes("AGENTS.md", []byte(generated))
			if err != nil {
				t.Fatal(err)
			}
			old := strings.Replace(string(block), "Handwritten project overrides", "Earlier project overrides", 1)
			entry := receipt{Before: digest([]byte(old)), After: digest(block)}
			owned, _, err := readInstallRecords(t.Context(), root)
			if err != nil {
				t.Fatal(err)
			}
			delete(owned, "AGENTS.md")
			if err = writeInstallRecord(root, ownedFile, owned); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "old":
				write(t, root, "AGENTS.md", old)
			case "edited":
				write(t, root, "AGENTS.md", strings.Replace(generated, "Handwritten project overrides", "Custom project overrides", 1))
			case "malformed":
				write(t, root, "AGENTS.md", routeStart)
			case "absent":
				if err = os.Remove(filepath.Join(root, "AGENTS.md")); err != nil {
					t.Fatal(err)
				}
			case "no target":
				entry.After = ""
				write(t, root, "AGENTS.md", old)
			}
			if err = writeInstallRecord(root, pendingFile, map[string]receipt{"AGENTS.md": entry}); err != nil {
				t.Fatal(err)
			}
			opts.HostFiles = []string{"CLAUDE.md"}
			if _, err = Install(t.Context(), root, snapshot, opts); err != nil {
				t.Fatal(err)
			}
			report, inspectErr := Inspect(t.Context(), root)
			if inspectErr != nil {
				t.Fatal(inspectErr)
			}
			wantIssue := kind == "edited" || kind == "malformed" || kind == "absent"
			if wantIssue {
				if len(report.Issues) != 1 || report.Issues[0].Path != "AGENTS.md" {
					t.Fatalf("lost retained-host diagnostic: %+v", report)
				}
			} else if len(report.Issues) != 0 {
				t.Fatalf("unexpected retained-host diagnostic: %+v", report)
			}
			opts.HostFiles = append(opts.HostFiles, "AGENTS.md")
			_, err = Install(t.Context(), root, snapshot, opts)
			if kind == "edited" || kind == "malformed" {
				if err == nil {
					t.Fatal("accepted unrecognized bytes")
				}
			} else if err != nil {
				t.Fatalf("recognized recovery refused: %v", err)
			}
		})
	}
}
