package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// envelopeMetadata runs args through cli.Execute, requires ExitOK, and
// returns the JSON envelope's metadata map. Shared by every AC-2 test
// in this file — each verb's own metadata shape differs, but the
// extraction is identical.
func envelopeMetadata(t *testing.T, args ...string) map[string]any {
	t.Helper()
	rc, stdout, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute(args)
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf %v: rc=%d stderr=%s", args, rc, stderr)
	}
	var env struct {
		Metadata map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout)
	}
	return env.Metadata
}

// TestPromoteMetadata_ReportsEntityFromToAndSHA pins M-0239/AC-2's own
// worked example: promote reports entity_id/from/to plus commit_sha,
// alongside AC-1's correlation_id (already proven separately).
func TestPromoteMetadata_ReportsEntityFromToAndSHA(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--title", "Stale probe", "--body", "## What's missing\n\nFixture prose.\n\n## Why it matters\n\nFixture prose.\n", "--actor", "human/test", "--root", root)

	md := envelopeMetadata(t, "promote", "G-0001", "wontfix", "--actor", "human/test", "--root", root, "--format=json")

	if md["entity_id"] != "G-0001" {
		t.Errorf("metadata.entity_id = %v, want %q", md["entity_id"], "G-0001")
	}
	if md["from"] != "open" {
		t.Errorf("metadata.from = %v, want %q", md["from"], "open")
	}
	if md["to"] != "wontfix" {
		t.Errorf("metadata.to = %v, want %q", md["to"], "wontfix")
	}
	sha, _ := md["commit_sha"].(string)
	if sha == "" {
		t.Error("metadata.commit_sha missing or empty")
	}
	if headSHA(t, root) != sha {
		t.Errorf("metadata.commit_sha = %q, want the actual HEAD sha %q", sha, headSHA(t, root))
	}
}

// TestPromoteMetadata_CompositeACIDReportsEntityFromTo pins the
// composite-id (M-NNN/AC-N) shape of promote — by far the most common
// invocation in this repo's own TDD-cycle workflow (--phase red/green/
// done, then a status promote to met) — alongside the plain-entity
// shape already proven above. finalizeACPlan is the shared chokepoint
// both promoteAC and PromoteACPhase route through.
func TestPromoteMetadata_CompositeACIDReportsEntityFromTo(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Home", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Parent", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "ac", "M-0001", "--title", "First criterion", "--actor", "human/test", "--root", root)

	md := envelopeMetadata(t, "promote", "M-0001/AC-1", "met", "--actor", "human/test", "--root", root, "--format=json")
	if md["entity_id"] != "M-0001/AC-1" {
		t.Errorf("metadata.entity_id = %v, want %q", md["entity_id"], "M-0001/AC-1")
	}
	if md["from"] != "open" {
		t.Errorf("metadata.from = %v, want %q", md["from"], "open")
	}
	if md["to"] != "met" {
		t.Errorf("metadata.to = %v, want %q", md["to"], "met")
	}
	if sha, _ := md["commit_sha"].(string); sha == "" {
		t.Error("metadata.commit_sha missing or empty")
	}
}

// TestWorktreeAddMetadata_ReportsBranchAndPath pins worktree add's
// AC-2 metadata (branch/path) and, since this verb builds its own
// render.Envelope directly rather than calling emitSuccess (unlike
// every other verb in this file), re-confirms AC-1's correlation_id
// actually reaches it too — the earlier AC-1 pass set
// OutputFormat.CorrelationID correctly but nothing in this verb read
// it until this AC-2 pass wired out.Metadata(...) into the envelope.
func TestWorktreeAddMetadata_ReportsBranchAndPath(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	root, bin := setupInitedRepo(t)

	stdout, stderr, code := runSplit(t, root, bin, "worktree", "add", "feature/metadata-check", filepath.Join(t.TempDir(), "wt"), "--format=json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0\nstdout=%q\nstderr=%q", code, stdout, stderr)
	}

	var env struct {
		Metadata map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout)
	}
	if env.Metadata["branch"] != "feature/metadata-check" {
		t.Errorf("metadata.branch = %v, want %q", env.Metadata["branch"], "feature/metadata-check")
	}
	if path, _ := env.Metadata["path"].(string); path == "" {
		t.Error("metadata.path missing or empty")
	}
	if id, _ := env.Metadata["correlation_id"].(string); id == "" {
		t.Error("metadata.correlation_id missing or empty — worktree add builds its own envelope and must call out.Metadata(...) explicitly")
	}
}

// TestArchiveMetadata_ReportsSweptCountAndSHA pins M-0239/AC-2's other
// named worked example: archive reports swept_count/commit_sha. This
// verb had no --format=json support at all before this AC — it wrote
// plain text unconditionally.
func TestArchiveMetadata_ReportsSweptCountAndSHA(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--title", "Stale probe", "--body", "## What's missing\n\nFixture prose.\n\n## Why it matters\n\nFixture prose.\n", "--actor", "human/test", "--root", root)
	mustRun(t, "promote", "G-0001", "wontfix", "--actor", "human/test", "--root", root)

	md := envelopeMetadata(t, "archive", "--apply", "--actor", "human/test", "--root", root, "--format=json")
	if count, ok := md["swept_count"].(float64); !ok || count != 1 {
		t.Errorf("metadata.swept_count = %v, want 1", md["swept_count"])
	}
	sha, _ := md["commit_sha"].(string)
	if sha == "" {
		t.Error("metadata.commit_sha missing or empty")
	}
	if headSHA(t, root) != sha {
		t.Errorf("metadata.commit_sha = %q, want the actual HEAD sha %q", sha, headSHA(t, root))
	}
	if id, _ := md["correlation_id"].(string); id == "" {
		t.Error("metadata.correlation_id missing or empty")
	}
}

// TestRewidthMetadata_ReportsRenamedCountAndSHA pins rewidth's AC-2
// metadata (renamed_count). Like archive, this verb had no
// --format=json support at all before this AC.
func TestRewidthMetadata_ReportsRenamedCountAndSHA(t *testing.T) {
	root := setupCLITestRepo(t)
	seedNarrowFixture(t, root)
	commitFixture(t, root, "seed narrow fixture")

	md := envelopeMetadata(t, "rewidth", "--apply", "--root", root, "--actor", "human/test", "--format=json")
	count, ok := md["renamed_count"].(float64)
	if !ok || count < 1 {
		t.Errorf("metadata.renamed_count = %v, want >= 1", md["renamed_count"])
	}
	sha, _ := md["commit_sha"].(string)
	if sha == "" {
		t.Error("metadata.commit_sha missing or empty")
	}
	if headSHA(t, root) != sha {
		t.Errorf("metadata.commit_sha = %q, want the actual HEAD sha %q", sha, headSHA(t, root))
	}
	if id, _ := md["correlation_id"].(string); id == "" {
		t.Error("metadata.correlation_id missing or empty")
	}
}

// TestImportMetadata_ReportsImportedCountEntityIDsAndSHA pins import's
// AC-2 metadata. Like archive/rewidth, this verb had no
// --format=json support at all before this AC. import produces one
// commit per plan (a deliberate exception to the one-verb-one-commit
// norm), so commit_sha here is the batch's last commit.
func TestImportMetadata_ReportsImportedCountEntityIDsAndSHA(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	manifestPath := writeManifest(t, root, `version: 1
entities:
  - kind: epic
    id: E-0001
    frontmatter: {title: "Cake", status: active}
  - kind: milestone
    id: M-0001
    frontmatter: {title: "Bake", status: draft, parent: E-0001}
`)

	md := envelopeMetadata(t, "import", "--root", root, "--actor", "human/test", "--format=json", manifestPath)
	if count, ok := md["imported_count"].(float64); !ok || count != 2 {
		t.Errorf("metadata.imported_count = %v, want 2", md["imported_count"])
	}
	ids, ok := md["entity_ids"].([]any)
	if !ok || len(ids) != 2 {
		t.Fatalf("metadata.entity_ids = %v, want a 2-element array", md["entity_ids"])
	}
	if ids[0] != "E-0001" || ids[1] != "M-0001" {
		t.Errorf("metadata.entity_ids = %v, want [E-0001 M-0001]", ids)
	}
	sha, _ := md["commit_sha"].(string)
	if sha == "" {
		t.Error("metadata.commit_sha missing or empty")
	}
	if headSHA(t, root) != sha {
		t.Errorf("metadata.commit_sha = %q, want the actual HEAD sha %q", sha, headSHA(t, root))
	}
	if id, _ := md["correlation_id"].(string); id == "" {
		t.Error("metadata.correlation_id missing or empty")
	}
}

// TestMoveMetadata_ReportsEntityFromToAndSHA pins move's AC-2
// metadata — found missing during the branch-coverage audit (move
// already had AC-1's correlation_id but no per-verb metadata yet).
func TestMoveMetadata_ReportsEntityFromToAndSHA(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Source epic", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "epic", "--title", "Target epic", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "Child", "--actor", "human/test", "--root", root)

	md := envelopeMetadata(t, "move", "M-0001", "--epic", "E-0002", "--actor", "human/test", "--root", root, "--format=json")
	if md["entity_id"] != "M-0001" {
		t.Errorf("metadata.entity_id = %v, want %q", md["entity_id"], "M-0001")
	}
	if md["from"] != "E-0001" {
		t.Errorf("metadata.from = %v, want %q", md["from"], "E-0001")
	}
	if md["to"] != "E-0002" {
		t.Errorf("metadata.to = %v, want %q", md["to"], "E-0002")
	}
	sha, _ := md["commit_sha"].(string)
	if sha == "" {
		t.Error("metadata.commit_sha missing or empty")
	}
	if headSHA(t, root) != sha {
		t.Errorf("metadata.commit_sha = %q, want the actual HEAD sha %q", sha, headSHA(t, root))
	}
}

// TestContractVerbsMetadata_CarryCorrelationIDAndOwnMetadata pins
// correlation_id + AC-2 metadata for the contract bind/unbind/recipe-
// install/recipe-remove verbs and milestone depends-on — all five
// were missed entirely during the earlier AC-1 sweep (their NewCmd
// constructors call cliutil.AddFormatFlags but weren't in the list
// checked at the time) and had no correlationID threading until this
// pass.
func TestContractVerbsMetadata_CarryCorrelationIDAndOwnMetadata(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	script := fakeValidatorCLI(t, root)
	customPath := filepath.Join(root, "fake.yaml")
	if err := os.WriteFile(customPath, []byte("name: fake\ncommand: "+script+"\nargs:\n  - \"{{fixture}}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("contract recipe install", func(t *testing.T) {
		md := envelopeMetadata(t, "contract", "recipe", "install", "--from", customPath, "--root", root, "--actor", "human/test", "--format=json")
		if md["validator"] != "fake" {
			t.Errorf("metadata.validator = %v, want %q", md["validator"], "fake")
		}
		if id, _ := md["correlation_id"].(string); id == "" {
			t.Error("metadata.correlation_id missing or empty")
		}
	})

	mustWriteFile(t, filepath.Join(root, "schema.cue"), "")
	writeFixtureFile(t, root, "fixtures/v1/valid/good.json", "PASS")
	mustRun(t, "add", "contract", "--body", "## Purpose\n\nFixture prose.\n\n## Stability\n\nFixture prose.\n", "--title", "Public API", "--root", root, "--actor", "human/test", "--validator", "fake", "--schema", "schema.cue", "--fixtures", "fixtures")

	t.Run("contract unbind", func(t *testing.T) {
		md := envelopeMetadata(t, "contract", "unbind", "--root", root, "--actor", "human/test", "--format=json", "C-0001")
		if md["entity_id"] != "C-0001" {
			t.Errorf("metadata.entity_id = %v, want %q", md["entity_id"], "C-0001")
		}
		if id, _ := md["correlation_id"].(string); id == "" {
			t.Error("metadata.correlation_id missing or empty")
		}
	})

	t.Run("contract bind", func(t *testing.T) {
		md := envelopeMetadata(t, "contract", "bind", "C-0001", "--validator", "fake", "--schema", "schema.cue", "--fixtures", "fixtures", "--root", root, "--actor", "human/test", "--format=json")
		if md["entity_id"] != "C-0001" {
			t.Errorf("metadata.entity_id = %v, want %q", md["entity_id"], "C-0001")
		}
		if md["validator"] != "fake" {
			t.Errorf("metadata.validator = %v, want %q", md["validator"], "fake")
		}
		if id, _ := md["correlation_id"].(string); id == "" {
			t.Error("metadata.correlation_id missing or empty")
		}
	})

	mustRun(t, "contract", "unbind", "--root", root, "--actor", "human/test", "C-0001")

	t.Run("contract recipe remove", func(t *testing.T) {
		md := envelopeMetadata(t, "contract", "recipe", "remove", "--root", root, "--actor", "human/test", "--format=json", "fake")
		if md["validator"] != "fake" {
			t.Errorf("metadata.validator = %v, want %q", md["validator"], "fake")
		}
		if id, _ := md["correlation_id"].(string); id == "" {
			t.Error("metadata.correlation_id missing or empty")
		}
	})

	mustRun(t, "add", "epic", "--title", "Home", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Source", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Dependency", "--actor", "human/test", "--root", root)

	t.Run("milestone depends-on", func(t *testing.T) {
		md := envelopeMetadata(t, "milestone", "depends-on", "M-0001", "--on", "M-0002", "--actor", "human/test", "--root", root, "--format=json")
		if md["entity_id"] != "M-0001" {
			t.Errorf("metadata.entity_id = %v, want %q", md["entity_id"], "M-0001")
		}
		deps, ok := md["depends_on"].([]any)
		if !ok || len(deps) != 1 || deps[0] != "M-0002" {
			t.Errorf("metadata.depends_on = %v, want [M-0002]", md["depends_on"])
		}
		if id, _ := md["correlation_id"].(string); id == "" {
			t.Error("metadata.correlation_id missing or empty")
		}
	})
}

// jsonEnvelopeError runs args (expected to fail) through cli.Execute
// and returns the decoded error envelope. Used to pin the JSON-mode
// error path of archive/rewidth/import's newly-added failArchive/
// failRewidth/failImport helpers — none of which any pre-existing
// text-mode-only test exercises in JSON mode.
func jsonEnvelopeError(t *testing.T, wantCode int, args ...string) (message string) {
	t.Helper()
	rc, stdout, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute(args)
	})
	if rc != wantCode {
		t.Fatalf("aiwf %v: rc=%d, want %d; stderr=%s", args, rc, wantCode, stderr)
	}
	var env struct {
		Status string `json:"status"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout)
	}
	if env.Status != "error" || env.Error == nil {
		t.Fatalf("expected a status:error envelope; got %s", stdout)
	}
	if stderr != "" {
		t.Errorf("JSON-mode error: stderr must be empty; got %q", stderr)
	}
	return env.Error.Message
}

// TestArchiveMetadata_JSONModeErrorEnvelope pins failArchive's JSON
// branch — every prior archive test exercised only the text-mode
// error path.
func TestArchiveMetadata_JSONModeErrorEnvelope(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	msg := jsonEnvelopeError(t, cliutil.ExitUsage, "archive", "--kind", "bogus", "--actor", "human/test", "--root", root, "--format=json")
	if !strings.Contains(msg, "bogus") {
		t.Errorf("error message = %q, want it to mention the bad --kind value", msg)
	}
}

// TestRewidthMetadata_JSONModeErrorEnvelope pins failRewidth's JSON
// branch.
func TestRewidthMetadata_JSONModeErrorEnvelope(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	msg := jsonEnvelopeError(t, cliutil.ExitUsage, "rewidth", "--actor", "ai/bot", "--root", root, "--format=json")
	if !strings.Contains(msg, "--principal") {
		t.Errorf("error message = %q, want it to mention the missing --principal", msg)
	}
}

// TestImportMetadata_JSONModeErrorEnvelope pins failImport's JSON
// branch — every prior import test exercised only success, dry-run,
// or the separate findings-envelope path (--on-collision fail), never
// a failImport-routed error at all (text or JSON).
func TestImportMetadata_JSONModeErrorEnvelope(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	msg := jsonEnvelopeError(t, cliutil.ExitUsage, "import", "--root", root, "--actor", "human/test", "--format=json", filepath.Join(root, "does-not-exist.yaml"))
	if msg == "" {
		t.Error("error message empty")
	}
}

// TestImportMetadata_TextModeErrorUnchanged pins failImport's text-
// mode branch — refactoring Run's inline cliutil.Errorf calls into
// the shared helper must not change the operator-facing message or
// stream.
func TestImportMetadata_TextModeErrorUnchanged(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	rc, stdout, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"import", "--root", root, "--actor", "human/test", filepath.Join(root, "does-not-exist.yaml")})
	})
	if rc != cliutil.ExitUsage {
		t.Fatalf("rc = %d, want %d", rc, cliutil.ExitUsage)
	}
	if stdout != "" {
		t.Errorf("text-mode error: stdout must be empty; got %q", stdout)
	}
	if !strings.HasPrefix(stderr, "aiwf import: ") {
		t.Errorf("stderr = %q, want it to start with %q", stderr, "aiwf import: ")
	}
}

// TestCancelMetadata_ReportsEntityFromToAndSHA closes a real gap: AC-1
// verified cancel's correlation_id, but no test ever asserted on
// cancel's own AC-2 metadata (entity_id/from/to) content.
func TestCancelMetadata_ReportsEntityFromToAndSHA(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--title", "Stale probe", "--body", "## What's missing\n\nFixture prose.\n\n## Why it matters\n\nFixture prose.\n", "--actor", "human/test", "--root", root)

	md := envelopeMetadata(t, "cancel", "G-0001", "--reason", "no longer needed", "--actor", "human/test", "--root", root, "--format=json")
	if md["entity_id"] != "G-0001" {
		t.Errorf("metadata.entity_id = %v, want %q", md["entity_id"], "G-0001")
	}
	if md["from"] != "open" {
		t.Errorf("metadata.from = %v, want %q", md["from"], "open")
	}
	if md["to"] != "wontfix" {
		t.Errorf("metadata.to = %v, want %q", md["to"], "wontfix")
	}
	if sha, _ := md["commit_sha"].(string); sha == "" {
		t.Error("metadata.commit_sha missing or empty")
	}
}

// TestRenameMetadata_ReportsEntityAndNewSlug closes the same class of
// gap for rename.
func TestRenameMetadata_ReportsEntityAndNewSlug(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Home", "--actor", "human/test", "--root", root)

	md := envelopeMetadata(t, "rename", "E-0001", "renamed-slug", "--actor", "human/test", "--root", root, "--format=json")
	if md["entity_id"] != "E-0001" {
		t.Errorf("metadata.entity_id = %v, want %q", md["entity_id"], "E-0001")
	}
	if md["new_slug"] != "renamed-slug" {
		t.Errorf("metadata.new_slug = %v, want %q", md["new_slug"], "renamed-slug")
	}
}

// TestRetitleMetadata_ReportsEntityOldAndNewTitle closes the gap for
// retitle.
func TestRetitleMetadata_ReportsEntityOldAndNewTitle(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Original Title", "--actor", "human/test", "--root", root)

	md := envelopeMetadata(t, "retitle", "E-0001", "New Title", "--actor", "human/test", "--root", root, "--format=json")
	if md["entity_id"] != "E-0001" {
		t.Errorf("metadata.entity_id = %v, want %q", md["entity_id"], "E-0001")
	}
	if md["old_title"] != "Original Title" {
		t.Errorf("metadata.old_title = %v, want %q", md["old_title"], "Original Title")
	}
	if md["new_title"] != "New Title" {
		t.Errorf("metadata.new_title = %v, want %q", md["new_title"], "New Title")
	}
}

// TestSetAreaMetadata_ReportsEntityAndArea closes the gap for
// set-area, including the --clear shape (area == "").
func TestSetAreaMetadata_ReportsEntityAndArea(t *testing.T) {
	root := setupAreaRepo(t)
	mustRun(t, "add", "epic", "--title", "Home", "--area", "platform", "--actor", "human/test", "--root", root)

	t.Run("set", func(t *testing.T) {
		setRoot := setupAreaRepo(t)
		mustRun(t, "add", "epic", "--title", "Home", "--area", "platform", "--actor", "human/test", "--root", setRoot)
		md := envelopeMetadata(t, "set-area", "E-0001", "billing", "--actor", "human/test", "--root", setRoot, "--format=json")
		if md["entity_id"] != "E-0001" {
			t.Errorf("metadata.entity_id = %v, want %q", md["entity_id"], "E-0001")
		}
		if md["area"] != "billing" {
			t.Errorf("metadata.area = %v, want %q", md["area"], "billing")
		}
	})

	t.Run("clear", func(t *testing.T) {
		md := envelopeMetadata(t, "set-area", "E-0001", "--clear", "--actor", "human/test", "--root", root, "--format=json")
		if md["entity_id"] != "E-0001" {
			t.Errorf("metadata.entity_id = %v, want %q", md["entity_id"], "E-0001")
		}
		if area, ok := md["area"]; !ok || area != "" {
			t.Errorf("metadata.area = %v, want empty string (cleared)", md["area"])
		}
	})
}

// TestRenameAreaMetadata_ReportsOldNewAreaAndCount closes the gap for
// rename-area.
func TestRenameAreaMetadata_ReportsOldNewAreaAndCount(t *testing.T) {
	root := setupAreaRepo(t)
	mustRun(t, "add", "epic", "--title", "Home", "--area", "platform", "--actor", "human/test", "--root", root)

	md := envelopeMetadata(t, "rename-area", "platform", "infra", "--actor", "human/test", "--root", root, "--format=json")
	if md["old_area"] != "platform" {
		t.Errorf("metadata.old_area = %v, want %q", md["old_area"], "platform")
	}
	if md["new_area"] != "infra" {
		t.Errorf("metadata.new_area = %v, want %q", md["new_area"], "infra")
	}
	if count, ok := md["entities_rewritten"].(float64); !ok || count != 1 {
		t.Errorf("metadata.entities_rewritten = %v, want 1", md["entities_rewritten"])
	}
}

// TestReallocateMetadata_ReportsOldAndNewID closes the gap for
// reallocate.
func TestReallocateMetadata_ReportsOldAndNewID(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Home", "--actor", "human/test", "--root", root)

	md := envelopeMetadata(t, "reallocate", "E-0001", "--actor", "human/test", "--root", root, "--format=json")
	if md["old_id"] != "E-0001" {
		t.Errorf("metadata.old_id = %v, want %q", md["old_id"], "E-0001")
	}
	newID, _ := md["new_id"].(string)
	if newID == "" || newID == "E-0001" {
		t.Errorf("metadata.new_id = %v, want a fresh, different E-NNNN id", md["new_id"])
	}
}

// TestEditBodyMetadata_ReportsEntityID closes the gap for edit-body's
// explicit (--body-file) path.
func TestEditBodyMetadata_ReportsEntityID(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Home", "--actor", "human/test", "--root", root)
	bodyFile := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(bodyFile, []byte("## Goal\n\nUpdated body.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	md := envelopeMetadata(t, "edit-body", "E-0001", "--body-file", bodyFile, "--actor", "human/test", "--root", root, "--format=json")
	if md["entity_id"] != "E-0001" {
		t.Errorf("metadata.entity_id = %v, want %q", md["entity_id"], "E-0001")
	}
}

// TestAuthorizeMetadata_PauseReportsEntityAndAction closes the gap for
// authorize's --pause/--resume path (the --to <agent> open path was
// spot-checked in TestCorrelationID_PresentAcrossMutatingVerbs, but
// that only asserted correlation_id, and pause/resume is a separate
// code path in verb.Authorize entirely).
func TestAuthorizeMetadata_PauseReportsEntityAndAction(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Adoption", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Schema parser", "--actor", "human/test", "--root", root)
	// The G-0269 activating-promote branch guard requires this
	// checkout before the milestone in_progress promote below, not
	// after.
	if out, err := testutil.RunGit(root, "checkout", "-b", "epic/E-0001-adoption"); err != nil {
		t.Fatalf("git checkout -b: %v\n%s", err, out)
	}
	// M-0268/AC-1: draft -> in_progress now refuses a zero-AC
	// milestone; seed one so the promote below exercises the
	// pause/resume metadata path, not the AC-completeness guard.
	// M-0268/AC-2: draft -> in_progress also refuses an empty AC
	// body; give it real prose.
	mustRun(t, "add", "ac", "M-0001", "--title", "Parses schema", "--body-file", acBodyFixturePath(t, root), "--actor", "human/test", "--root", root)
	mustRun(t, "promote", "--root", root, "--actor", "human/test", "M-0001", "in_progress")
	mustRun(t, "authorize", "--root", root, "--actor", "human/test", "M-0001", "--to", "ai/claude")

	md := envelopeMetadata(t, "authorize", "--root", root, "--actor", "human/test", "M-0001", "--pause", "pausing for review", "--format=json")
	if md["entity_id"] != "M-0001" {
		t.Errorf("metadata.entity_id = %v, want %q", md["entity_id"], "M-0001")
	}
	if md["action"] != "pause" {
		t.Errorf("metadata.action = %v, want %q", md["action"], "pause")
	}
}

// TestAcknowledgeIllegalMetadata_ReportsSHA closes the gap: the
// earlier spot-check only asserted correlation_id, never the sha
// field's actual value.
func TestAcknowledgeIllegalMetadata_ReportsSHA(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--title", "Spot-check", "--body", "## What's missing\n\nFixture prose.\n\n## Why it matters\n\nFixture prose.\n", "--actor", "human/test", "--root", root)
	sha, err := testutil.RunGit(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v\n%s", err, sha)
	}
	sha = strings.TrimSpace(sha)

	md := envelopeMetadata(t, "acknowledge", "illegal", sha, "--reason", "spot-check", "--actor", "human/test", "--root", root, "--format=json")
	if md["sha"] != sha {
		t.Errorf("metadata.sha = %v, want %q", md["sha"], sha)
	}
}

// TestAcknowledgeMistagMetadata_ReportsEntityID closes the gap: this
// verb was never even spot-checked before (only illegal was).
func TestAcknowledgeMistagMetadata_ReportsEntityID(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "areas:\n  members:\n" +
		"    - {name: app-a, paths: [projects/app-a/**]}\n" +
		"    - {name: billing, paths: [projects/billing/**]}\n"
	if err := os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}

	mustRun(t, "add", "gap", "--body", "## What's missing\n\nFixture prose.\n\n## Why it matters\n\nFixture prose.\n", "--root", root, "--actor", "human/test", "--area", "app-a", "--title", "login timeout fix")
	if err := os.MkdirAll(filepath.Join(root, "projects", "billing"), 0o755); err != nil {
		t.Fatalf("mkdir projects/billing: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "projects", "billing", "invoice.go"), []byte("package billing\n"), 0o644); err != nil {
		t.Fatalf("write invoice: %v", err)
	}
	if err := osExec(t, root, "git", "add", "projects/billing/invoice.go"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := osExec(t, root, "git", "commit", "-q", "-m", "billing work", "--trailer", "aiwf-entity: G-0001"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	md := envelopeMetadata(t, "acknowledge", "mistag", "G-0001", "--root", root, "--actor", "human/test", "--reason", "intentional cross-cutting work into billing", "--format=json")
	if md["entity_id"] != "G-0001" {
		t.Errorf("metadata.entity_id = %v, want %q", md["entity_id"], "G-0001")
	}
}

// TestAddMetadata_ReportsEntityIDAndKind closes the gap: the earlier
// spot-check on "add" only asserted correlation_id, never entity_id/
// kind.
func TestAddMetadata_ReportsEntityIDAndKind(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	md := envelopeMetadata(t, "add", "gap", "--title", "Spot-check", "--body", "## What's missing\n\nFixture prose.\n\n## Why it matters\n\nFixture prose.\n", "--actor", "human/test", "--root", root, "--format=json")
	if md["entity_id"] != "G-0001" {
		t.Errorf("metadata.entity_id = %v, want %q", md["entity_id"], "G-0001")
	}
	if md["kind"] != "gap" {
		t.Errorf("metadata.kind = %v, want %q", md["kind"], "gap")
	}
}

// TestAddACMetadata_ReportsEntityIDAndACIDs closes the gap: the
// earlier spot-check on "add ac" only asserted correlation_id, never
// entity_id/ac_ids.
func TestAddACMetadata_ReportsEntityIDAndACIDs(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Home", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Parent", "--actor", "human/test", "--root", root)

	md := envelopeMetadata(t, "add", "ac", "M-0001", "--title", "First", "--title", "Second", "--actor", "human/test", "--root", root, "--format=json")
	if md["entity_id"] != "M-0001" {
		t.Errorf("metadata.entity_id = %v, want %q", md["entity_id"], "M-0001")
	}
	ids, ok := md["ac_ids"].([]any)
	if !ok || len(ids) != 2 || ids[0] != "M-0001/AC-1" || ids[1] != "M-0001/AC-2" {
		t.Errorf("metadata.ac_ids = %v, want [M-0001/AC-1 M-0001/AC-2]", md["ac_ids"])
	}
}

// headSHA returns the repo's current HEAD commit sha.
func headSHA(t *testing.T, root string) string {
	t.Helper()
	out, err := testutil.RunGit(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v\n%s", err, out)
	}
	return strings.TrimSpace(out)
}
