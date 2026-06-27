package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/config"
)

// setupObjectAreaRepo initializes a repo and patches aiwf.yaml with the given
// areas-block body (lines already indented under `areas:`). Mirrors
// setupAreaRepo but lets each test supply an object-form member list.
func setupObjectAreaRepo(t *testing.T, body string) string {
	t.Helper()
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	if err := os.WriteFile(yamlPath, []byte(string(raw)+"areas:\n"+body), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	return root
}

// areaPaths returns a map from member name to its declared paths, parsed
// structurally from the on-disk aiwf.yaml via the real config loader (not a
// substring grep). Members present with nil paths map to a nil slice.
func areaPaths(t *testing.T, root string) map[string][]string {
	t.Helper()
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("config.Load after rename: %v", err)
	}
	out := make(map[string][]string, len(cfg.Areas.Members))
	for _, m := range cfg.Areas.Members {
		out[m.Name] = m.Paths
	}
	return out
}

// TestRenameArea_PreservesPaths pins AC-4 (M-0179): renaming an object-form
// member rewrites only the renamed member's name and preserves every member's
// paths through the whole writer chain. The fixture mixes two object-with-paths
// members and one bare string member; the rewritten YAML is PARSED (structural,
// not substring) and each member's paths asserted.
func TestRenameArea_PreservesPaths(t *testing.T) {
	body := "" +
		"  members:\n" +
		"    - name: app-a\n" +
		"      paths:\n" +
		"        - projects/app-a/**\n" +
		"    - name: billing\n" +
		"      paths:\n" +
		"        - svc/billing/**\n" +
		"    - plat\n"
	root := setupObjectAreaRepo(t, body)

	// An epic tagged app-a so the rename exercises the full chain (member
	// rewrite + entity retag), not just the aiwf.yaml-only path.
	mustRun(t, "add", "epic", "--title", "App A work", "--area", "app-a", "--actor", "human/test", "--root", root)

	mustRun(t, "rename-area", "app-a", "application-a", "--actor", "human/test", "--root", root)

	got := areaPaths(t, root)

	if _, ok := got["app-a"]; ok {
		t.Errorf("old member app-a still present after rename: %v", got)
	}
	if want := []string{"projects/app-a/**"}; !equalStrings(got["application-a"], want) {
		t.Errorf("application-a paths = %v, want %v", got["application-a"], want)
	}
	if want := []string{"svc/billing/**"}; !equalStrings(got["billing"], want) {
		t.Errorf("billing paths = %v, want %v (non-renamed member untouched)", got["billing"], want)
	}
	if p, ok := got["plat"]; !ok || p != nil {
		t.Errorf("plat = (%v, present=%v), want present with nil paths (stays bare)", p, ok)
	}

	// The renamed member's paths followed its new name: the entity retag
	// also landed (application-a is now the epic's area).
	fm := frontmatterOf(readOne(t, root, filepath.Join("work", "epics", "E-0001-*", "epic.md")))
	if !strings.Contains(fm, "area: application-a") {
		t.Errorf("epic not retagged to application-a:\n%s", fm)
	}
}

// TestObjectFormConfig_NameReadersWork pins AC-5 (M-0179): the name-based read
// sites behave correctly when the config is object-form, asserted against the
// ABSOLUTE expectation (not merely "object-form == string-form" — a broken
// MemberNames() returning nil would make both forms reject everything alike and
// a pure differential test would pass green).
func TestObjectFormConfig_NameReadersWork(t *testing.T) {
	body := "" +
		"  members:\n" +
		"    - name: app-a\n" +
		"      paths:\n" +
		"        - projects/app-a/**\n" +
		"    - name: billing\n" +
		"      paths:\n" +
		"        - svc/billing/**\n"

	t.Run("MemberNames returns declared names in order", func(t *testing.T) {
		root := setupObjectAreaRepo(t, body)
		cfg, err := config.Load(root)
		if err != nil {
			t.Fatalf("config.Load: %v", err)
		}
		want := []string{"app-a", "billing"}
		if got := cfg.Areas.MemberNames(); !equalStrings(got, want) {
			t.Errorf("MemberNames() = %v, want %v", got, want)
		}
	})

	t.Run("add --area accepts a declared member", func(t *testing.T) {
		root := setupObjectAreaRepo(t, body)
		mustRun(t, "add", "gap", "--title", "Known", "--area", "app-a", "--actor", "human/test", "--root", root)
		fm := frontmatterOf(readOne(t, root, "work/gaps/G-*.md"))
		if !strings.Contains(fm, "area: app-a") {
			t.Errorf("gap frontmatter missing `area: app-a`:\n%s", fm)
		}
	})

	t.Run("add --area rejects an undeclared member", func(t *testing.T) {
		root := setupObjectAreaRepo(t, body)
		rc, _, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"add", "gap", "--title", "X", "--area", "app-z", "--actor", "human/test", "--root", root})
		})
		if rc != cliutil.ExitUsage {
			t.Errorf("rc = %d, want ExitUsage (%d)", rc, cliutil.ExitUsage)
		}
		if !strings.Contains(stderr, "app-z") {
			t.Errorf("stderr %q should name the undeclared value", stderr)
		}
		if matches, _ := filepath.Glob(filepath.Join(root, "work", "gaps", "G-*.md")); len(matches) != 0 {
			t.Errorf("no gap should be created on rejection; found %v", matches)
		}
	})

	t.Run("check flags an undeclared area but not a declared one", func(t *testing.T) {
		root := setupObjectAreaRepo(t, body)
		// One gap tagged with a declared area, one with an undeclared area.
		mustRun(t, "add", "gap", "--title", "Known", "--area", "app-a", "--actor", "human/test", "--root", root)
		mustRun(t, "add", "gap", "--title", "Drift", "--actor", "human/test", "--root", root)

		driftMatches, _ := filepath.Glob(filepath.Join(root, "work", "gaps", "G-0002-*.md"))
		if len(driftMatches) != 1 {
			t.Fatalf("locate drift gap: %v", driftMatches)
		}
		raw, err := os.ReadFile(driftMatches[0])
		if err != nil {
			t.Fatalf("read drift gap: %v", err)
		}
		patched := strings.Replace(string(raw), "status: open\n", "status: open\narea: app-z\n", 1)
		if patched == string(raw) {
			t.Fatalf("failed to inject undeclared area:\n%s", raw)
		}
		if err := os.WriteFile(driftMatches[0], []byte(patched), 0o644); err != nil {
			t.Fatalf("write drift gap: %v", err)
		}

		captured := testutil.CaptureStdout(t, func() {
			_ = cli.Execute([]string{"check", "--root", root})
		})
		out := string(captured)
		if !strings.Contains(out, "area-unknown") {
			t.Errorf("expected area-unknown for the app-z-tagged gap; got:\n%s", out)
		}
		// The app-a-tagged gap (G-0001) must NOT appear in any area-unknown
		// finding — only the app-z-tagged G-0002 does. Scope the negative
		// assertion to area-unknown lines so unrelated findings (e.g.
		// entity-body-empty, which names every gap) don't false-trip it.
		for _, line := range strings.Split(out, "\n") {
			if !strings.Contains(line, "area-unknown") {
				continue
			}
			if !strings.Contains(line, "app-z") || !strings.Contains(line, "G-0002") {
				t.Errorf("area-unknown finding should name the app-z-tagged G-0002; got line:\n%s", line)
			}
			if strings.Contains(line, "G-0001") {
				t.Errorf("declared-area gap G-0001 must not be flagged area-unknown; got line:\n%s", line)
			}
		}
	})
}

// equalStrings reports whether two string slices are element-wise equal,
// treating nil and empty as distinct (so a nil-paths assertion is exact).
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
