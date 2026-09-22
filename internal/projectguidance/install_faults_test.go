package projectguidance

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func TestInstall_RejectsCorruptOwnershipRecords(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, path, value string }{
		{"invalid json", ownedFile, "{"},
		{"null", ownedFile, "null"},
		{"trailing value", ownedFile, "{} {}"},
		{"foreign path", ownedFile, `{"project.md":"x"}`},
		{"unsafe path", ownedFile, `{".guidance/packs/../other.md":"x"}`},
		{"invalid digest", ownedFile, `{".guidance/index.md":"abc"}`},
		{"wrong type", ownedFile, `{".guidance/index.md":3}`},
		{"empty receipt", pendingFile, `{".guidance/index.md":{"before":"","after":""}}`},
		{"bad before", pendingFile, `{".guidance/index.md":{"before":"bad","after":""}}`},
		{"bad after", pendingFile, `{".guidance/index.md":{"before":"","after":"bad"}}`},
		{"unknown receipt field", pendingFile, `{".guidance/index.md":{"extra":true}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			snapshot, opts := installationFixture(t)
			write(t, root, tc.path, tc.value)
			before := readTree(t, root)
			if _, err := Install(t.Context(), root, snapshot, opts); err == nil {
				t.Fatal("accepted corrupt record")
			}
			if diff := cmp.Diff(before, readTree(t, root)); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestInstall_RejectsUnsafeRecoveryTemporaries(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"other directory", "other basename", "bad digest", "edited temporary", "linked temporary"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			snapshot, opts := installationFixture(t)
			root := t.TempDir()
			entry := receipt{After: digest([]byte("new")), Temp: indexFile + ".aiwf-tmp-test", TempDigest: digest([]byte("new"))}
			switch kind {
			case "other directory":
				entry.Temp = "index.md.aiwf-tmp-test"
			case "other basename":
				entry.Temp = ".guidance/other.aiwf-tmp-test"
			case "bad digest":
				entry.TempDigest = "bad"
			case "edited temporary":
				write(t, root, entry.Temp, "edited")
			case "linked temporary":
				if err := os.MkdirAll(filepath.Join(root, ".guidance"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(t.TempDir(), filepath.Join(root, entry.Temp)); err != nil {
					t.Fatal(err)
				}
			}
			write(t, root, pendingFile, "{}")
			if err := writeInstallRecord(root, pendingFile, map[string]receipt{indexFile: entry}); err != nil {
				t.Fatal(err)
			}
			if _, err := Install(t.Context(), root, snapshot, opts); err == nil {
				t.Fatal("accepted unsafe temporary")
			}
		})
	}
}

func TestInstall_EmptyPendingRecordIsRecovered(t *testing.T) {
	t.Parallel()
	snapshot, opts := installationFixture(t)
	root := t.TempDir()
	if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
		t.Fatal(err)
	}
	complete := readTree(t, root)
	write(t, root, pendingFile, "{}")
	changed, err := Install(t.Context(), root, snapshot, opts)
	if err != nil || !changed {
		t.Fatalf("recovery: %v, %v", changed, err)
	}
	if diff := cmp.Diff(complete, readTree(t, root)); diff != "" {
		t.Fatal(diff)
	}
}

func TestInstall_RemovesOnlyOwnedDocumentsAndRetainsOtherHosts(t *testing.T) {
	t.Parallel()
	snapshot, opts := installationFixture(t)
	root := t.TempDir()
	if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
		t.Fatal(err)
	}
	agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	write(t, root, ".guidance/packs/handwritten.md", "mine\n")
	opts.HostFiles = []string{"CLAUDE.md"}
	opts.Selected = nil
	if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
		t.Fatal(err)
	}
	current := readTree(t, root)
	if _, ok := current[".guidance/packs/go/cobra/guide.md"]; ok {
		t.Fatal("retired owned document remains")
	}
	if current[".guidance/packs/handwritten.md"] != "mine\n" || current["AGENTS.md"] != string(agents) {
		t.Fatal("unselected content changed")
	}
	opts.Selected = []string{"go/cobra"}
	opts.HostFiles = []string{"AGENTS.md", "CLAUDE.md"}
	if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
		t.Fatalf("retained host lost ownership when reselected: %v", err)
	}
}

func TestInstall_InvalidSnapshotAndSelection(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"nil", "no commit", "unknown pack", "duplicate pack", "no entry point", "missing document", "invalid host"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			snapshot, opts := installationFixture(t)
			switch kind {
			case "nil":
				snapshot = nil
			case "no commit":
				snapshot.Commit = ""
			case "unknown pack":
				opts.Selected = []string{"absent"}
			case "duplicate pack":
				opts.Selected = []string{"go/cobra", "go/cobra"}
			case "no entry point":
				snapshot.Catalogue.Packs[0].Files = nil
			case "missing document":
				snapshot.Documents = nil
			case "invalid host":
				opts.HostFiles = []string{"foreign.md"}
			}
			root := t.TempDir()
			if _, err := Install(t.Context(), root, snapshot, opts); err == nil {
				t.Fatal("invalid selection installed")
			}
			if len(readTree(t, root)) != 0 {
				t.Fatal("invalid selection wrote files")
			}
		})
	}
}

func TestInstall_RefusesDirectoriesAndUnreadableFiles(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"directory", "unreadable", "ownership directory"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			if kind == "unreadable" && os.Geteuid() == 0 {
				t.Skip("permission denial requires non-root")
			}
			snapshot, opts := installationFixture(t)
			root := t.TempDir()
			name := indexFile
			if kind == "ownership directory" {
				name = ownedFile
			}
			dest := filepath.Join(root, name)
			if kind == "unreadable" {
				write(t, root, name, "unreadable")
				if err := os.Chmod(dest, 0o000); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(dest, 0o644) })
			} else if err := os.MkdirAll(dest, 0o755); err != nil {
				t.Fatal(err)
			}
			if _, err := Install(t.Context(), root, snapshot, opts); err == nil {
				t.Fatal("accepted inaccessible output")
			}
		})
	}
}

func TestInstall_CancellationBeforeWrites(t *testing.T) {
	t.Parallel()
	snapshot, opts := installationFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	root := t.TempDir()
	if _, err := Install(ctx, root, snapshot, opts); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if len(readTree(t, root)) != 0 {
		t.Fatal("cancelled installation wrote files")
	}
}

func TestStageInstallation_FilesystemErrors(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"blocked parent", "read-only parent", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			if kind == "read-only parent" && os.Geteuid() == 0 {
				t.Skip("permission denial requires non-root")
			}
			root := t.TempDir()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			writes := []installWrite{{path: indexFile, content: []byte("new"), mode: 0o644}}
			switch kind {
			case "blocked parent":
				write(t, root, ".guidance", "occupied")
			case "read-only parent":
				dest := filepath.Join(root, ".guidance")
				if err := os.Mkdir(dest, 0o500); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(dest, 0o755) })
			case "cancelled":
				cancel()
			}
			if err := applyInstallation(ctx, root, writes, nil, nil); err == nil {
				t.Fatal("ignored staging failure")
			}
		})
	}
}

func TestPublishInstallation_WriteFailuresRemainRecoverable(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"pending record", "rename", "ownership record"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			snapshot, opts := installationFixture(t)
			root := t.TempDir()
			desired, err := renderInstallation(snapshot, opts)
			if err != nil {
				t.Fatal(err)
			}
			writes, owned, err := planInstallation(t.Context(), root, desired, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			pending, err := stageInstallation(t.Context(), root, writes, nil)
			if err != nil {
				t.Fatal(err)
			}
			blocker := pendingFile
			if kind == "rename" {
				blocker = writes[0].path
			}
			if kind == "ownership record" {
				blocker = ownedFile
			}
			if err := os.Mkdir(filepath.Join(root, blocker), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := publishInstallation(t.Context(), root, writes, owned, pending); err == nil {
				t.Fatal("ignored publication failure")
			}
			if err := os.Remove(filepath.Join(root, blocker)); err != nil {
				t.Fatal(err)
			}
			// A failure to create the receipt preceded all destination writes. Preserve
			// the staged plan as the interrupted operation's receipt before retrying.
			if kind == "pending record" {
				if err := writeInstallRecord(root, pendingFile, pending); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
				t.Fatal(err)
			}
			clean := t.TempDir()
			if _, err := Install(t.Context(), clean, snapshot, opts); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(readTree(t, clean), readTree(t, root)); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestInstall_RetirementFailureKeepsOldIndex(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("permission denial requires non-root")
	}
	snapshot, opts := installationFixture(t)
	root := t.TempDir()
	if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
		t.Fatal(err)
	}
	before := readTree(t, root)
	blocked := filepath.Join(root, ".guidance", "packs", "go", "cobra")
	if err := os.Chmod(blocked, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o755) })
	opts.Selected = nil
	if _, err := Install(t.Context(), root, snapshot, opts); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("got %v", err)
	}
	index, err := os.ReadFile(filepath.Join(root, indexFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(index) != before[indexFile] {
		t.Fatal("failed retirement published new index")
	}
	after := readTree(t, root)
	delete(after, pendingFile)
	if diff := cmp.Diff(before, after); diff != "" {
		t.Fatalf("failed publication changed files or leaked staging: %s", diff)
	}

	if err := os.Chmod(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
		t.Fatal(err)
	}
}

func TestWriteInstallRecord_EncodingFailure(t *testing.T) {
	t.Parallel()
	err := writeInstallRecord(t.TempDir(), ownedFile, make(chan int))
	var unsupported *json.UnsupportedTypeError
	if !errors.As(err, &unsupported) {
		t.Fatalf("lost encoding error: %v", err)
	}
}

func TestRenderInstallation_EscapesDescriptionsAndLinks(t *testing.T) {
	t.Parallel()
	snapshot, opts := installationFixture(t)
	pack := &snapshot.Catalogue.Packs[0]
	pack.Description = "line\n[link](elsewhere) <script>"
	pack.Files = []string{"packs/go/cobra/a b.md"}
	snapshot.Documents[pack.Files[0]] = []byte("# Text\n")
	rendered, err := renderInstallation(snapshot, opts)
	if err != nil {
		t.Fatal(err)
	}
	index := string(rendered[indexFile])
	if !strings.Contains(index, "packs/go/cobra/a%20b.md") || strings.Contains(index, "<script>") || strings.Contains(index, "\n[link]") {
		t.Fatalf("unsafe rendering: %s", index)
	}
}

func TestRenderInstallation_EntryLinksResolveUnusualFilenames(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"a b.md", "a)b.md"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			snapshot, opts := installationFixture(t)
			file := "packs/go/cobra/" + name
			snapshot.Catalogue.Packs[0].Files = []string{file}
			snapshot.Documents[file] = []byte("# Guidance\n")
			rendered, err := renderInstallation(snapshot, opts)
			if err != nil {
				t.Fatal(err)
			}
			document := goldmark.DefaultParser().Parse(text.NewReader(rendered[indexFile]))
			found := false
			err = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
				link, ok := node.(*ast.Link)
				if entering && ok {
					decoded, decodeErr := url.PathUnescape(string(link.Destination))
					if decodeErr != nil {
						return ast.WalkStop, decodeErr
					}
					if decoded == file {
						found = true
					}
				}
				return ast.WalkContinue, nil
			})
			if err != nil || !found {
				t.Fatalf("entry link does not resolve to %q: %s (%v)", file, rendered[indexFile], err)
			}
		})
	}
}

func TestInstall_RecoveryTemporaryRemovalFailure(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("permission denial requires non-root")
	}
	snapshot, opts := installationFixture(t)
	root := t.TempDir()
	document := ".guidance/packs/go/cobra/guide.md"
	content := snapshot.Documents["packs/go/cobra/guide.md"]
	write(t, root, document, string(content))
	temporary := document + ".aiwf-tmp-recovery"
	write(t, root, temporary, string(content))
	pending := map[string]receipt{document: {After: digest(content), Temp: temporary, TempDigest: digest(content)}}
	if err := writeInstallRecord(root, pendingFile, pending); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Dir(filepath.Join(root, temporary))
	if err := os.Chmod(directory, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(directory, 0o755) })
	if _, err := Install(t.Context(), root, snapshot, opts); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("got %v", err)
	}
	if err := os.Chmod(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, temporary)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary remains: %v", err)
	}
}

func TestPlanInstallation_RejectsMalformedReplacementRouting(t *testing.T) {
	t.Parallel()
	_, _, err := planInstallation(t.Context(), t.TempDir(), map[string][]byte{"AGENTS.md": []byte(routeEnd)}, nil, nil)
	if !errors.Is(err, ErrInstallConflict) {
		t.Fatalf("got %v", err)
	}
}

func TestInstall_RejectsNoncanonicalDigests(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"aa", strings.ToUpper(digest([]byte("content")))} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			snapshot, opts := installationFixture(t)
			root := t.TempDir()
			write(t, root, ownedFile, "{}")
			if err := writeInstallRecord(root, ownedFile, map[string]string{indexFile: value}); err != nil {
				t.Fatal(err)
			}
			if _, err := Install(t.Context(), root, snapshot, opts); !errors.Is(err, ErrInstallConflict) {
				t.Fatalf("got %v", err)
			}
		})
	}
}
