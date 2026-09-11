package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestRun_ShowRendersTheAbsentVerbMarker pins internal/cli/show/show.go's verb
// column. `show` renders the same events as `history` and reaches the same
// marker through RenderVerb, but no existing show fixture contains an event
// without a verb, so reverting that call site left the whole suite green.
func TestRun_ShowRendersTheAbsentVerbMarker(t *testing.T) {
	root := setupCLITestRepo(t)
	bodyDir := t.TempDir()
	epicBody := filepath.Join(bodyDir, "epic.md")
	if err := os.WriteFile(epicBody, []byte("## Goal\n\nx\n\n## Scope\n\ny\n\n## Out of scope\n\nz\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := cli.Execute([]string{"add", "epic", "--title", "Foundations", "--body-file", epicBody, "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add epic: %d", rc)
	}
	// The shape D-0071 creates: provenance is the entity trailer alone, so
	// there is no verb for the column to name.
	if out, err := testutil.RunGit(root, "commit", "--allow-empty", "-m",
		"fix(x): correct a shipped surface\n\naiwf-entity: E-0001\n"); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}

	out := string(testutil.CaptureStdout(t, func() {
		if rc := cli.Execute([]string{"show", "--root", root, "E-0001"}); rc != cliutil.ExitOK {
			t.Fatalf("show: %d", rc)
		}
	}))
	if !strings.Contains(out, "fix(x): correct a shipped surface") {
		t.Fatalf("show does not list the entity-only commit at all:\n%s", out)
	}
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "fix(x): correct a shipped surface") {
			continue
		}
		// Fields, not Contains: the date carries hyphens, so a substring
		// check passes whether the column holds the marker or nothing.
		// show prints date, verb, to, detail — verb and to both render the
		// marker for an event with neither trailer.
		f := strings.Fields(line)
		if len(f) < 4 || f[1] != "-" || f[2] != "-" {
			t.Errorf("verb column is not the absent-trailer marker; fields = %q from line %q", f, line)
		}
		return
	}
}
