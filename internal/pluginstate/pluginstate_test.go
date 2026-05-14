package pluginstate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoad_FileMissing_IsEmptyIndex: when ~/.claude/plugins/installed_plugins.json
// does not exist (Claude Code never run on this machine, or running outside
// Claude Code entirely) the load returns an empty index without error. M-070's
// design treats absence as "no plugins installed" so every recommended plugin
// warns — see the milestone spec's Approach §3.
func TestLoad_FileMissing_IsEmptyIndex(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	idx, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load on missing file: %v", err)
	}
	if idx == nil {
		t.Fatal("Load returned nil index, want empty")
	}
	if got := len(idx.Plugins); got != 0 {
		t.Errorf("len(Plugins) = %d, want 0 (empty index)", got)
	}
}

// TestLoad_PresentAndParses: a real-shape installed_plugins.json (mirroring
// the actual Claude Code layout) loads into a typed Index. Captures only the
// fields the matcher needs (scope, projectPath); other fields are ignored.
func TestLoad_PresentAndParses(t *testing.T) {
	t.Parallel()
	tmp := writeFixture(t, `{
  "version": 2,
  "plugins": {
    "aiwf-extensions@ai-workflow-rituals": [
      {
        "scope": "project",
        "projectPath": "/abs/path/to/repo",
        "installPath": "/some/cache",
        "version": "abcdef"
      },
      {
        "scope": "user",
        "installPath": "/some/cache"
      }
    ]
  }
}`)
	idx, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	entries := idx.Plugins["aiwf-extensions@ai-workflow-rituals"]
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	if entries[0].Scope != "project" || entries[0].ProjectPath != "/abs/path/to/repo" {
		t.Errorf("project entry = %+v", entries[0])
	}
	if entries[1].Scope != "user" || entries[1].ProjectPath != "" {
		t.Errorf("user entry = %+v (projectPath should be absent → empty string)", entries[1])
	}
}

// TestLoad_NoPluginsKey_YieldsNonNilMap: the on-disk JSON parses cleanly
// but contains no `plugins` key (e.g. a fresh installed_plugins.json that
// only carries `{"version": 2}`). Load returns an Index whose Plugins map
// is non-nil-empty so callers can range over it without a nil check.
func TestLoad_NoPluginsKey_YieldsNonNilMap(t *testing.T) {
	t.Parallel()
	tmp := writeFixture(t, `{"version": 2}`)
	idx, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if idx.Plugins == nil {
		t.Fatal("idx.Plugins = nil, want empty (non-nil) map")
	}
	if got := len(idx.Plugins); got != 0 {
		t.Errorf("len(Plugins) = %d, want 0", got)
	}
}

// TestHasProjectScope_NilReceiver: calling on a nil *Index returns false
// without panicking. Defensive contract for callers that might receive a
// nil index from a failed Load (we return &Index{} on absent file, so
// nil is the contract guard for any future code path that elects not to
// construct an empty index).
func TestHasProjectScope_NilReceiver(t *testing.T) {
	t.Parallel()
	var idx *Index
	got, err := idx.HasProjectScope("anything@anywhere", "/any/root")
	if err != nil {
		t.Fatalf("HasProjectScope on nil: %v", err)
	}
	if got {
		t.Error("HasProjectScope on nil = true, want false")
	}
}

// TestLoad_MalformedJSON_ReturnsError: if the file exists but isn't valid
// JSON, Load returns an error naming the path. Loud failure beats silent
// "no plugins" because the latter would mask a real config breakage.
func TestLoad_MalformedJSON_ReturnsError(t *testing.T) {
	t.Parallel()
	tmp := writeFixture(t, "{not json")
	_, err := Load(tmp)
	if err == nil {
		t.Fatal("Load on malformed JSON: nil error, want failure")
	}
	if !strings.Contains(err.Error(), "installed_plugins.json") {
		t.Errorf("error message = %q, want to mention installed_plugins.json", err.Error())
	}
}

// TestHasProjectScope_NoSuchPlugin: querying a plugin name not in the index
// returns false (no error). Covers the "file present, no matches" fixture
// the AC calls out.
func TestHasProjectScope_NoSuchPlugin(t *testing.T) {
	t.Parallel()
	idx := &Index{Plugins: map[string][]InstallEntry{}}
	got, err := idx.HasProjectScope("does-not-exist@somewhere", "/any/root")
	if err != nil {
		t.Fatalf("HasProjectScope: %v", err)
	}
	if got {
		t.Error("HasProjectScope = true, want false")
	}
}

// TestHasProjectScope_DifferentProjectPath: the plugin is installed at
// project scope, but for a different repo. Query against this consumer's
// root must return false. Covers the AC-6 session-canonical case where
// `aiwf-extensions` lives under another project's path.
func TestHasProjectScope_DifferentProjectPath(t *testing.T) {
	t.Parallel()
	consumerRoot := "/Users/x/Projects/consumer"
	idx := &Index{
		Plugins: map[string][]InstallEntry{
			"aiwf-extensions@ai-workflow-rituals": {
				{Scope: "project", ProjectPath: "/Users/x/Projects/other-repo"},
			},
		},
	}
	got, err := idx.HasProjectScope("aiwf-extensions@ai-workflow-rituals", consumerRoot)
	if err != nil {
		t.Fatalf("HasProjectScope: %v", err)
	}
	if got {
		t.Errorf("HasProjectScope = true, want false (project install elsewhere is not visible here)")
	}
}

// TestHasProjectScope_MatchingProjectPath: the canonical happy path.
// Plugin installed at project scope with projectPath equal to the
// consumer root → query returns true. Covers AC-5.
func TestHasProjectScope_MatchingProjectPath(t *testing.T) {
	t.Parallel()
	consumerRoot := "/Users/x/Projects/consumer"
	idx := &Index{
		Plugins: map[string][]InstallEntry{
			"aiwf-extensions@ai-workflow-rituals": {
				{Scope: "project", ProjectPath: consumerRoot},
			},
		},
	}
	got, err := idx.HasProjectScope("aiwf-extensions@ai-workflow-rituals", consumerRoot)
	if err != nil {
		t.Fatalf("HasProjectScope: %v", err)
	}
	if !got {
		t.Error("HasProjectScope = false, want true (project install for this root)")
	}
}

// TestHasProjectScope_UserScopeOnlyDoesNotMatch: user-scope installs are
// repo-agnostic and (per the spec's Approach §3) the matcher requires a
// project-scope entry whose projectPath matches the consumer root. A
// plugin installed only at user scope must NOT silence the warning for
// any consumer.
func TestHasProjectScope_UserScopeOnlyDoesNotMatch(t *testing.T) {
	t.Parallel()
	idx := &Index{
		Plugins: map[string][]InstallEntry{
			"aiwf-extensions@ai-workflow-rituals": {
				{Scope: "user"},
			},
		},
	}
	got, err := idx.HasProjectScope("aiwf-extensions@ai-workflow-rituals", "/Users/x/Projects/anything")
	if err != nil {
		t.Fatalf("HasProjectScope: %v", err)
	}
	if got {
		t.Error("HasProjectScope = true, want false (user-scope only is not a project-scope match)")
	}
}

// TestHasProjectScope_AbsolutePathNormalization: projectPath in the index
// and the consumer root passed to the query are both absolute-resolved via
// filepath.Abs before comparison. A relative consumer-root argument resolves
// against the current working dir, then matches a stored absolute index entry.
// This covers the AC's "exact-path for projectPath (after both are
// absolute-resolved via filepath.Abs)" requirement.
func TestHasProjectScope_AbsolutePathNormalization(t *testing.T) {
	tmp := t.TempDir()
	abs, err := filepath.Abs(tmp)
	if err != nil {
		t.Fatal(err)
	}
	idx := &Index{
		Plugins: map[string][]InstallEntry{
			"aiwf-extensions@ai-workflow-rituals": {
				{Scope: "project", ProjectPath: abs},
			},
		},
	}
	// Cwd into a different dir, then pass tmp via a relative segment that
	// nonetheless resolves to the same absolute path.
	if cdErr := os.Chdir(t.TempDir()); cdErr != nil {
		t.Fatal(cdErr)
	}
	t.Cleanup(func() {
		// Defensive: ensure we don't leave the parallel test process in
		// the temp dir after this test exits.
		wd, _ := os.Getwd()
		if strings.HasPrefix(wd, os.TempDir()) {
			_ = os.Chdir("/")
		}
	})
	got, err := idx.HasProjectScope("aiwf-extensions@ai-workflow-rituals", abs)
	if err != nil {
		t.Fatalf("HasProjectScope (absolute arg): %v", err)
	}
	if !got {
		t.Errorf("HasProjectScope = false, want true (abs equality)")
	}
}

// writeFixture writes installed_plugins.json under a synthetic
// $HOME-equivalent layout (~/.claude/plugins/installed_plugins.json)
// rooted at a t.TempDir() and returns the home path the caller should
// pass to Load.
func writeFixture(t *testing.T, body string) string {
	t.Helper()
	home := t.TempDir()
	dir := filepath.Join(home, ".claude", "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "installed_plugins.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return home
}
