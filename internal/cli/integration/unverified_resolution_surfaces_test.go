package integration

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// The overclaim a ref-less surface must not make, and the clause that
// replaces it. Grepping rendered output for these is what pins the
// downgrade per surface — the unit test covers the pass itself, but
// only driving each command proves the command applies it (G-0558).
const (
	overclaimPhrase = "no entity allocated at this id"
	neutralPhrase   = "resolves to no entity in this working tree"
)

// unverifiedFixture builds a repo whose only content finding is a prose
// reference to an unallocated id — the exact case a ref-less load
// cannot settle, and no other finding to confuse a substring assertion.
func unverifiedFixture(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	gapDir := filepath.Join(root, "work", "gaps")
	if err := os.MkdirAll(gapDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gap := "---\nid: G-0001\ntitle: ref fixture\nstatus: open\n---\nSee M-9999 for the rule.\n"
	if err := os.WriteFile(filepath.Join(gapDir, "G-0001-ref-fixture.md"), []byte(gap), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestSurfaces_RefLessReadPathsDoNotOverclaim drives each read path
// that loads without the cross-branch scan and asserts none of them
// asserts the id is unallocated — a claim about tiers they never built.
//
// One test per surface rather than one shared assertion, because the
// pass is applied per call site: a surface that drops it regresses
// alone, and the failure should name which.
func TestSurfaces_RefLessReadPathsDoNotOverclaim(t *testing.T) {
	t.Run("aiwf show", func(t *testing.T) {
		root := unverifiedFixture(t)
		rc, out, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"show", "G-0001", "--root", root})
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("show rc = %d, want %d:\n%s", rc, cliutil.ExitOK, out)
		}
		assertDoesNotOverclaim(t, "aiwf show", out)
	})

	t.Run("aiwf status", func(t *testing.T) {
		root := unverifiedFixture(t)
		rc, out, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"status", "--root", root})
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("status rc = %d, want %d:\n%s", rc, cliutil.ExitOK, out)
		}
		// status renders counts, not finding messages, so the pin is the
		// Health line: an error here contradicts the full check, and the
		// same line tells the reader to go run it.
		if strings.Contains(out, "1 errors") {
			t.Errorf("status Health counts an error the full check does not raise:\n%s", out)
		}
	})

	t.Run("aiwf render", func(t *testing.T) {
		root := unverifiedFixture(t)
		outDir := filepath.Join(t.TempDir(), "site")
		rc, cliOut, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"render", "--format", "html", "--root", root, "--out", outDir})
		})
		if rc != cliutil.ExitOK {
			t.Fatalf("render rc = %d, want %d:\n%s", rc, cliutil.ExitOK, cliOut)
		}
		// index.html's severity tally is the only thing render's own
		// findings slice feeds. Asserting on prose elsewhere in the site
		// would pin whichever surface produced that page instead — the
		// status page, for one, is built by status.BuildStatus and
		// applies the pass on its own account.
		errs, warns := renderedFindingCounts(t, filepath.Join(outDir, "index.html"))
		if errs != 0 {
			t.Errorf("render tallied %d error(s); this surface cannot substantiate `unresolved`, so it must not count one", errs)
		}
		if warns == 0 {
			t.Error("render tallied no warnings, so the fixture's finding never reached the tally and the check above proves nothing")
		}
	})
}

// findingsTallyPattern matches index.tmpl's severity line.
var findingsTallyPattern = regexp.MustCompile(`Findings: (\d+) error\(s\), (\d+) warning\(s\)`)

// renderedFindingCounts reads the error/warning tally off the rendered
// index page.
func renderedFindingCounts(t *testing.T, indexPath string) (errs, warns int) {
	t.Helper()
	b, err := os.ReadFile(indexPath) //nolint:gosec // path is a test-owned temp dir
	if err != nil {
		t.Fatalf("reading rendered index: %v", err)
	}
	m := findingsTallyPattern.FindStringSubmatch(string(b))
	if m == nil {
		t.Fatalf("no findings tally in %s; the assertions below would be vacuous", indexPath)
	}
	errs, err = strconv.Atoi(m[1])
	if err != nil {
		t.Fatal(err)
	}
	warns, err = strconv.Atoi(m[2])
	if err != nil {
		t.Fatal(err)
	}
	return errs, warns
}

// assertDoesNotOverclaim fails when output carries the strong verdict,
// and also when it carries neither phrasing — which would mean the
// finding stopped surfacing altogether and the test had gone vacuous.
func assertDoesNotOverclaim(t *testing.T, surface, out string) {
	t.Helper()
	if strings.Contains(out, overclaimPhrase) {
		t.Errorf("%s asserts %q, which it did not establish — the surface is not applying MarkUnverifiedResolution:\n%s",
			surface, overclaimPhrase, out)
	}
	if !strings.Contains(out, neutralPhrase) {
		t.Errorf("%s reports neither phrasing, so this assertion proves nothing; the fixture's finding must reach the output:\n%s",
			surface, out)
	}
}
