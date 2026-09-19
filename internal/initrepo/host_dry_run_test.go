package initrepo

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/config"
)

func TestHostDryRun_GuidanceLedgerMatchesAppliedActions(t *testing.T) {
	t.Parallel()
	for _, initialize := range []bool{false, true} {
		for _, state := range []string{"missing", "user", "current", "stale", "damaged", "symlink", "directory", "alias", "optout", "optout missing"} {
			t.Run(fmt.Sprintf("init=%v/%s", initialize, state), func(t *testing.T) {
				t.Parallel()
				root := t.TempDir()
				cfg := "hosts: [claude-code, codex]\n"
				if state == "optout" || state == "optout missing" {
					cfg += "guidance:\n  wire_claudemd: false\n  wire_agentsmd: false\n"
				}
				if err := os.WriteFile(filepath.Join(root, config.FileName), []byte(cfg), 0o600); err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
					switch state {
					case "user", "optout":
						if err := os.WriteFile(filepath.Join(root, name), []byte("user instructions\n"), 0o600); err != nil {
							t.Fatal(err)
						}
					case "directory":
						if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
							t.Fatal(err)
						}
					case "stale":
						content := "user prefix\n" + guidanceImportStartMarker + "\nstale generated instructions\n" + guidanceImportEndMarker + "\nuser suffix\n"
						if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
							t.Fatal(err)
						}
					case "damaged":
						if err := os.WriteFile(filepath.Join(root, name), []byte(guidanceImportStartMarker+"\n"), 0o600); err != nil {
							t.Fatal(err)
						}
					case "symlink":
						if err := os.Symlink("outside", filepath.Join(root, name)); err != nil {
							t.Fatal(err)
						}
					}
				}
				if state == "alias" {
					claude := filepath.Join(root, "CLAUDE.md")
					if err := os.WriteFile(claude, []byte("shared user instructions\n"), 0o600); err != nil {
						t.Fatal(err)
					}
					if err := os.Link(claude, filepath.Join(root, "AGENTS.md")); err != nil {
						t.Fatal(err)
					}
				}
				run := func(dry bool) *Result {
					t.Helper()
					var result *Result
					var err error
					if initialize {
						result, err = Init(context.Background(), root, Options{DryRun: dry, SkipHook: true})
					} else {
						result, err = RefreshArtifacts(context.Background(), root, RefreshOptions{DryRun: dry, SkipHooks: true, WireClaudeMd: state != "optout" && state != "optout missing"})
					}
					if err != nil {
						t.Fatal(err)
					}
					return result
				}
				if state == "current" {
					run(false)
				}
				before := dryRunSnapshot(t, root)
				preview := run(true)
				if diff := cmp.Diff(before, dryRunSnapshot(t, root)); diff != "" {
					t.Fatalf("preview wrote files (-before +after):\n%s", diff)
				}
				if !preview.DryRun {
					t.Fatal("preview not marked dry-run")
				}
				wantSelection := config.HostSelection{Hosts: []config.Host{config.HostClaudeCode, config.HostCodex}, Source: config.HostsConfigured}
				if diff := cmp.Diff(wantSelection, preview.HostSelection); diff != "" {
					t.Fatal(diff)
				}
				guidanceRows := 0
				for _, step := range preview.Steps {
					if step.What != "CLAUDE.md (aiwf guidance import)" && step.What != "AGENTS.md (guidance)" {
						continue
					}
					guidanceRows++
					want := ActionSkipped
					switch state {
					case "missing":
						want = ActionCreated
						if initialize && step.What == "CLAUDE.md (aiwf guidance import)" {
							want = ActionUpdated
						}
					case "user", "stale":
						want = ActionUpdated
					case "current":
						want = ActionPreserved
					}
					if step.Action != want {
						t.Errorf("%s: action=%s want=%s", step.What, step.Action, want)
					}
					if want == ActionSkipped && step.Detail == "" {
						t.Errorf("%s: skip has no reason", step.What)
					}
				}
				if guidanceRows != 2 {
					t.Fatalf("guidance ledger has %d rows, want both hosts", guidanceRows)
				}
				applied := run(false)
				actions := func(result *Result) map[string]Action {
					out := map[string]Action{}
					for _, step := range result.Steps {
						out[step.What] = step.Action
					}
					return out
				}
				if diff := cmp.Diff(actions(applied), actions(preview)); diff != "" {
					t.Fatalf("preview differs from actual ledger (-applied +preview):\n%s", diff)
				}
			})
		}
	}
}

// Snapshot all entries without following links; read access times are excluded.
func dryRunSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		name, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		var content string
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			content, err = os.Readlink(path)
		case info.Mode().IsRegular():
			var data []byte
			data, err = os.ReadFile(path)
			content = string(data)
		}
		if err != nil {
			return err
		}
		result[name] = fmt.Sprintf("%s %d %s", info.Mode(), info.ModTime().UnixNano(), content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
