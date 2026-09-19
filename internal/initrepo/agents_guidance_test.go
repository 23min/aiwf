package initrepo

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/skills"
	"github.com/23min/aiwf/internal/version"
)

func TestSpliceAgentsGuidance_PreservesUnownedBytes(t *testing.T) {
	t.Parallel()
	start, end := guidanceImportStartMarker, guidanceImportEndMarker
	block := start + "\nnative body\n" + end
	quoted := start + " appears in this explanation.\n" + "Explain `" + start + "` and `" + end + "`.\n> " + start + "\n> " + end + "\n"
	for _, tc := range []struct{ name, before, want string }{
		{"empty", "", block + "\n"},
		{"no newline", "user bytes", "user bytes\n\n" + block + "\n"},
		{"newline", "user bytes\n", "user bytes\n\n" + block + "\n"},
		{"quoted prose", quoted, quoted + "\n" + block + "\n"},
		{"existing block", "prefix\n" + start + "\nold\n" + end + "\nsuffix", "prefix\n" + block + "\nsuffix"},
		{"no final newline block", start + "\nold\n" + end, block},
		{"CRLF outside block", "prefix\r\n" + start + "\r\nold\r\n" + end + "\r\nsuffix\r\n", "prefix\r\n" + block + "\r\nsuffix\r\n"},
		{"indented markers", "prefix\n  " + start + " \nold\n\t" + end + " \nsuffix", "prefix\n" + block + "\nsuffix"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := spliceAgentsGuidance(tc.before, "native body\n")
			if err != nil || got != tc.want {
				t.Fatalf("splice = %q, %v; want %q", got, err, tc.want)
			}
			again, err := spliceAgentsGuidance(got, "native body\n")
			if err != nil || again != got {
				t.Fatalf("repeat = %q, %v; want identical bytes", again, err)
			}
		})
	}
}

func TestEnsureAgentsGuidance_RefusesAmbiguousMarkers(t *testing.T) {
	t.Parallel()
	start, end := guidanceImportStartMarker, guidanceImportEndMarker
	for name, content := range map[string]string{
		"start only": start, "end only": end, "reversed": end + "\n" + start,
		"duplicate start":    start + "\n" + start + "\n" + end,
		"duplicate end":      start + "\n" + end + "\n" + end,
		"multiple blocks":    start + "\nx\n" + end + "\n" + start + "\ny\n" + end,
		"conflicting marker": "<!-- aiwf:guidance:START -->\nold\n" + end,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := filepath.Join(root, "AGENTS.md")
			writeAgentsFixture(t, path, content, 0o640)
			step, err := ensureAgentsGuidance(context.Background(), root, nil, false)
			if err != nil || step.Action != ActionSkipped || !strings.Contains(step.Detail, "repair") || !strings.Contains(step.Detail, "AGENTS.md") {
				t.Fatalf("refusal = %+v, %v", step, err)
			}
			assertAgentsFile(t, path, content, 0o640)
		})
	}
}

func TestEnsureAgentsGuidance_CreatesOrRefreshesNativeInstructions(t *testing.T) {
	t.Parallel()
	body, err := skills.RenderCodexGuidance(version.Current().Version)
	if err != nil {
		t.Fatal(err)
	}
	block := guidanceImportStartMarker + "\n" + strings.TrimSuffix(string(body), "\n") + "\n" + guidanceImportEndMarker
	for _, tc := range []struct {
		name         string
		absent       bool
		before, want string
		mode         fs.FileMode
		action       Action
	}{
		{"absent", true, "", block + "\n", 0o644, ActionCreated},
		{"empty", false, "", block + "\n", 0o600, ActionUpdated},
		{"append", false, "user\x00bytes", "user\x00bytes\n\n" + block + "\n", 0o640, ActionUpdated},
		{"refresh", false, "before\n" + guidanceImportStartMarker + "\nold\n" + guidanceImportEndMarker + "\nafter", "before\n" + block + "\nafter", 0o755, ActionUpdated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := filepath.Join(root, "AGENTS.md")
			if !tc.absent {
				writeAgentsFixture(t, path, tc.before, tc.mode)
			}
			step, err := ensureAgentsGuidance(context.Background(), root, nil, false)
			if err != nil || step.Action != tc.action {
				t.Fatalf("write = %+v, %v", step, err)
			}
			assertAgentsFile(t, path, tc.want, tc.mode)
			before, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			step, err = ensureAgentsGuidance(context.Background(), root, nil, false)
			if err != nil || step.Action != ActionPreserved {
				t.Fatalf("repeat = %+v, %v", step, err)
			}
			assertAgentsFile(t, path, tc.want, tc.mode)
			after, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if !os.SameFile(before, after) {
				t.Fatal("unchanged file was rewritten")
			}
		})
	}
}

func TestEnsureAgentsGuidance_OptOutAndDryRunDoNotWrite(t *testing.T) {
	t.Parallel()
	for _, exists := range []bool{false, true} {
		for _, optOut := range []bool{false, true} {
			t.Run(map[bool]string{false: "absent", true: "existing"}[exists]+"/"+map[bool]string{false: "dry run", true: "opt out"}[optOut], func(t *testing.T) {
				t.Parallel()
				root := t.TempDir()
				path := filepath.Join(root, "AGENTS.md")
				const content = "keep my instructions"
				if exists {
					writeAgentsFixture(t, path, content, 0o640)
				}
				cfg := &config.Config{}
				if optOut {
					disabled := false
					cfg.Guidance.WireAgentsMd = &disabled
					if err := config.Write(root, cfg); err != nil {
						t.Fatal(err)
					}
					var err error
					cfg, err = config.Load(root)
					if err != nil {
						t.Fatal(err)
					}
				}
				step, err := ensureAgentsGuidance(context.Background(), root, cfg, !optOut)
				want := ActionCreated
				if exists {
					want = ActionUpdated
				}
				if optOut {
					want = ActionSkipped
				}
				if err != nil || step.Action != want {
					t.Fatalf("result = %+v, %v; want %s", step, err, want)
				}
				if exists {
					assertAgentsFile(t, path, content, 0o640)
				} else if _, err := os.Lstat(path); !errors.Is(err, fs.ErrNotExist) {
					t.Fatalf("file created: %v", err)
				}
			})
		}
	}
}

func TestEnsureAgentsGuidance_NonregularFilesArePreserved(t *testing.T) {
	t.Parallel()
	for _, link := range []bool{false, true} {
		t.Run(map[bool]string{false: "directory", true: "symlink"}[link], func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := filepath.Join(root, "AGENTS.md")
			target := filepath.Join(root, "instructions")
			writeAgentsFixture(t, target, "target", 0o600)
			if link {
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}
			before, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			step, err := ensureAgentsGuidance(context.Background(), root, nil, false)
			if err != nil || step.Action != ActionSkipped || !strings.Contains(step.Detail, "AGENTS.md") || !strings.Contains(step.Detail, "regular file") {
				t.Fatalf("result = %+v, %v", step, err)
			}
			after, err := os.Lstat(path)
			if err != nil || !os.SameFile(before, after) {
				t.Fatalf("nonregular path changed: %v", err)
			}
			assertAgentsFile(t, target, "target", 0o600)
		})
	}
}

func TestEnsureAgentsGuidance_CanceledContextDoesNotWrite(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ensureAgentsGuidance(ctx, root, nil, false); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("files = %v, %v", entries, err)
	}
}

func TestEnsureAgentsGuidance_FilesystemErrorsPreserveContent(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	for _, operation := range []string{"inspect", "read", "write"} {
		t.Run(operation, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := filepath.Join(root, "AGENTS.md")
			writeAgentsFixture(t, path, "original", 0o640)
			t.Cleanup(func() { _ = os.Chmod(root, 0o755); _ = os.Chmod(path, 0o640) })
			restricted, mode := root, fs.FileMode(0o000)
			if operation == "read" {
				restricted = path
			}
			if operation == "write" {
				mode = 0o555
			}
			if err := os.Chmod(restricted, mode); err != nil {
				t.Fatal(err)
			}
			_, err := ensureAgentsGuidance(context.Background(), root, nil, false)
			if !errors.Is(err, fs.ErrPermission) || !strings.Contains(err.Error(), "AGENTS.md") {
				t.Fatalf("error = %v", err)
			}
			if chmodErr := os.Chmod(root, 0o755); chmodErr != nil {
				t.Fatal(chmodErr)
			}
			if chmodErr := os.Chmod(path, 0o640); chmodErr != nil {
				t.Fatal(chmodErr)
			}
			assertAgentsFile(t, path, "original", 0o640)
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 1 {
				t.Fatalf("write left debris: %v, %v", entries, err)
			}
		})
	}
}

func writeAgentsFixture(t *testing.T, path, content string, mode fs.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func assertAgentsFile(t *testing.T, path, want string, mode fs.FileMode) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s bytes differ: got %d bytes, want %d", path, len(got), len(want))
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != mode {
		t.Fatalf("mode = %o, want %o", info.Mode().Perm(), mode)
	}
}

// The version comes from the linker, so exercise a malformed build stamp in a
// separate test binary without mutating shared package state.
func TestEnsureAgentsGuidance_InvalidBuildStampPreservesFile(t *testing.T) {
	t.Parallel()
	const stamp = "{{aiwf:unknown}}"
	if version.Current().Version != stamp {
		cmd := exec.CommandContext(context.Background(), "go", "test", "-run=^TestEnsureAgentsGuidance_InvalidBuildStampPreservesFile$", "-ldflags=-X github.com/23min/aiwf/internal/version.Stamp="+stamp, ".")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("stamped test binary: %v\n%s", err, out)
		}
		return
	}
	root := t.TempDir()
	path := filepath.Join(root, "AGENTS.md")
	writeAgentsFixture(t, path, "user guidance", 0o640)
	_, err := ensureAgentsGuidance(context.Background(), root, nil, false)
	if !errors.Is(err, skills.ErrUnknownRenderBinding) || !strings.Contains(err.Error(), "AGENTS.md") {
		t.Fatalf("render error = %v", err)
	}
	assertAgentsFile(t, path, "user guidance", 0o640)
}
