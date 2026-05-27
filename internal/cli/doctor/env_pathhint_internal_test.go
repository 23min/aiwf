package doctor

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestForeignHomePrefix pins both arms of the GOOS adapter and that the
// wrapper resolves through runtime.GOOS.
//
// Closes G-0174.
func TestForeignHomePrefix(t *testing.T) {
	t.Parallel()
	if got := foreignHomePrefixFor("linux"); got != "/Users/" {
		t.Errorf("foreignHomePrefixFor(linux) = %q, want \"/Users/\"", got)
	}
	if got := foreignHomePrefixFor("darwin"); got != "" {
		t.Errorf("foreignHomePrefixFor(darwin) = %q, want \"\"", got)
	}
	// Wrapper resolves through the live GOOS; assert it agrees with the
	// adapter for this platform.
	if got, want := foreignHomePrefix(), foreignHomePrefixFor(runtime.GOOS); got != want {
		t.Errorf("foreignHomePrefix() = %q, want %q for GOOS=%s", got, want, runtime.GOOS)
	}
}

// seedIndexFile writes name under <home>/.claude/plugins/ with the
// given JSON content, creating the directory tree first.
func seedIndexFile(t *testing.T, home, name, content string) {
	t.Helper()
	dir := filepath.Join(home, ".claude", "plugins")
	mustMkdirAll(t, dir)
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("seed %s: %v", name, err)
	}
}

const macForeignPrefix = "/Users/"

// TestForeignPluginPaths covers the detection seam: a macOS-pathed
// entry in each index shape (the marketplace map and the nested
// installed-plugins arrays), a clean Linux index, absent files, the
// empty-prefix no-op (non-Linux host), and malformed JSON.
//
// Closes G-0174.
func TestForeignPluginPaths(t *testing.T) {
	t.Parallel()

	t.Run("macOS path in known_marketplaces", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		seedIndexFile(t, home, "known_marketplaces.json", `{
		  "ai-workflow-rituals": {
		    "installLocation": "/Users/peterbru/.claude/plugins/marketplaces/ai-workflow-rituals",
		    "autoUpdate": true
		  }
		}`)
		sample, found := foreignPluginPaths(home, macForeignPrefix)
		if !found {
			t.Fatal("found = false, want true (macOS installLocation must be detected)")
		}
		if !strings.HasPrefix(sample, macForeignPrefix) {
			t.Errorf("sample = %q, want a value with prefix %q", sample, macForeignPrefix)
		}
	})

	t.Run("macOS path in installed_plugins nested arrays", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		// Exercises the []any and default (number/bool) branches of
		// firstForeignPathLeaf alongside the string match.
		seedIndexFile(t, home, "installed_plugins.json", `{
		  "version": 2,
		  "plugins": {
		    "aiwf-extensions@ai-workflow-rituals": [
		      {
		        "scope": "project",
		        "enabled": true,
		        "installPath": "/Users/peterbru/.claude/plugins/cache/ai-workflow-rituals/aiwf-extensions/abc123"
		      }
		    ]
		  }
		}`)
		sample, found := foreignPluginPaths(home, macForeignPrefix)
		if !found {
			t.Fatal("found = false, want true (macOS installPath in nested array must be detected)")
		}
		if !strings.HasPrefix(sample, macForeignPrefix) {
			t.Errorf("sample = %q, want a value with prefix %q", sample, macForeignPrefix)
		}
	})

	t.Run("clean Linux index", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		seedIndexFile(t, home, "known_marketplaces.json", `{
		  "ai-workflow-rituals": {
		    "installLocation": "/home/vscode/.claude/plugins/marketplaces/ai-workflow-rituals"
		  }
		}`)
		seedIndexFile(t, home, "installed_plugins.json", `{
		  "version": 2,
		  "plugins": {
		    "aiwf-extensions@ai-workflow-rituals": [
		      { "installPath": "/home/vscode/.claude/plugins/cache/ai-workflow-rituals/aiwf-extensions/abc123" }
		    ]
		  }
		}`)
		if sample, found := foreignPluginPaths(home, macForeignPrefix); found {
			t.Errorf("found = true (sample %q), want false for a clean Linux index", sample)
		}
	})

	t.Run("no index files", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		if _, found := foreignPluginPaths(home, macForeignPrefix); found {
			t.Error("found = true, want false when no index files exist")
		}
	})

	t.Run("empty prefix short-circuits (non-Linux host)", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		seedIndexFile(t, home, "known_marketplaces.json", `{
		  "m": { "installLocation": "/Users/peterbru/.claude/plugins/marketplaces/m" }
		}`)
		if _, found := foreignPluginPaths(home, ""); found {
			t.Error("found = true, want false: an empty foreignPrefix must be a no-op")
		}
	})

	t.Run("malformed JSON is skipped silently", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		seedIndexFile(t, home, "known_marketplaces.json", `{ this is not "/Users/" valid json `)
		if _, found := foreignPluginPaths(home, macForeignPrefix); found {
			t.Error("found = true, want false: malformed JSON must be skipped, not raw-substring matched")
		}
	})
}

// TestFirstForeignPathLeaf pins each branch of the walker directly,
// including the default arm (numbers, bools, nil leaves) that must not
// match.
func TestFirstForeignPathLeaf(t *testing.T) {
	t.Parallel()

	t.Run("string leaf matches", func(t *testing.T) {
		t.Parallel()
		got, ok := firstForeignPathLeaf("/Users/x/y", macForeignPrefix)
		if !ok || got != "/Users/x/y" {
			t.Errorf("got %q, %v; want \"/Users/x/y\", true", got, ok)
		}
	})

	t.Run("non-matching scalars never match", func(t *testing.T) {
		t.Parallel()
		doc := map[string]any{
			"n":    float64(2),
			"b":    true,
			"z":    nil,
			"path": "/home/vscode/ok",
			"arr":  []any{float64(1), "also-ok"},
		}
		if got, ok := firstForeignPathLeaf(doc, macForeignPrefix); ok {
			t.Errorf("got %q, true; want \"\", false (no foreign path present)", got)
		}
	})
}

// TestRenderPluginPathHintLine pins the advisory render: the
// `plugin-paths:` label, the offending sample, the issue reference, and
// both remediation pointers.
//
// Closes G-0174.
func TestRenderPluginPathHintLine(t *testing.T) {
	t.Parallel()
	sample := "/Users/peterbru/.claude/plugins/marketplaces/ai-workflow-rituals"
	got := renderPluginPathHintLine(sample)
	for _, want := range []string{
		"plugin-paths: ",
		sample,
		"claude-code#31388",
		".devcontainer/initialize.sh",
		"claude plugin marketplace remove",
		"project scope",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("render missing %q\n got: %s", want, got)
		}
	}
}
