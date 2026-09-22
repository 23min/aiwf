package projectguidance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestInspect_ReportsLocalInstallationAndDamageWithoutWriting(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"current", "missing", "modified", "pending", "malformed route", "invalid record", "directory", "modified index", "missing index", "absent"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			snapshot, opts := installationFixture(t)
			if kind != "absent" {
				if _, err := Install(t.Context(), root, snapshot, opts); err != nil {
					t.Fatal(err)
				}
			}
			damaged := ".guidance/packs/go/cobra/guide.md"
			switch kind {
			case "missing":
				if err := os.Remove(filepath.Join(root, damaged)); err != nil {
					t.Fatal(err)
				}
			case "modified":
				write(t, root, damaged, "local edit")
			case "pending":
				write(t, root, pendingFile, "{}")
			case "malformed route":
				damaged = "AGENTS.md"
				write(t, root, damaged, routeStart)
			case "directory":
				if err := os.Remove(filepath.Join(root, damaged)); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Join(root, damaged), 0o755); err != nil {
					t.Fatal(err)
				}
			case "modified index":
				damaged = indexFile
				write(t, root, damaged, "edited index")
			case "missing index":
				damaged = indexFile
				if err := os.Remove(filepath.Join(root, damaged)); err != nil {
					t.Fatal(err)
				}
			case "invalid record":
				write(t, root, ownedFile, "invalid")
			}
			before := readTree(t, root)
			report, err := Inspect(t.Context(), root)
			if kind == "invalid record" {
				if err == nil {
					t.Fatal("corrupt ownership accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "absent":
				if report.Revision != "" || len(report.Files) != 0 {
					t.Fatalf("invented installation: %+v", report)
				}
			case "modified index", "missing index":
				if report.Revision != "" {
					t.Fatalf("untrusted revision: %q", report.Revision)
				}
			default:
				if report.Revision != snapshot.Commit {
					t.Fatalf("revision=%q, want %q", report.Revision, snapshot.Commit)
				}
			}
			if report.Pending != (kind == "pending") {
				t.Fatalf("pending: %+v", report)
			}
			if kind == "missing" || kind == "modified" || kind == "malformed route" || kind == "directory" || kind == "modified index" || kind == "missing index" {
				if len(report.Issues) != 1 || report.Issues[0].Path != damaged {
					t.Fatalf("wrong damage: %+v", report)
				}
			} else if len(report.Issues) != 0 {
				t.Fatalf("unexpected damage: %+v", report)
			}
			if diff := cmp.Diff(before, readTree(t, root)); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
