package integration

import (
	"strings"
	"testing"
)

// branch_scenarios_helpers_unit_test.go — unit-level pins on the
// branch-scenario framework's helpers that exist independently of
// the real-git scenario driver. Currently houses the test for
// replaceStatusInFrontmatter (M-0159/AC-4 refactor task #74,
// closing the first/second-reviewer note N2 fragility).

// TestReplaceStatusInFrontmatter_HappyPath pins the canonical
// case: a freshly-`aiwf add`'d epic body starts with the
// frontmatter opener, contains `status: proposed`, and ends with
// a closer + body sections. The helper finds the frontmatter
// line, replaces only inside, and returns the rest unmodified.
func TestReplaceStatusInFrontmatter_HappyPath(t *testing.T) {
	t.Parallel()
	input := []byte("---\nid: E-0001\ntitle: Engine\nstatus: proposed\n---\n\n## Goal\n\n## Scope\n")
	got, err := replaceStatusInFrontmatter(input, "proposed", "active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "---\nid: E-0001\ntitle: Engine\nstatus: active\n---\n\n## Goal\n\n## Scope\n"
	if string(got) != want {
		t.Errorf("output mismatch\n  got: %q\n want: %q", string(got), want)
	}
}

// TestReplaceStatusInFrontmatter_DoesNotMutateBodyContent pins
// the N2 sabotage closure: a future scenario whose epic body
// contains the literal `status: proposed` (e.g. in a code fence
// quoting an example FSM state) must NOT have its body line
// mutated; only the frontmatter line moves. The naive
// strings.Replace would have rewritten the body line silently —
// producing a fixture that fired DIFFERENT findings than the
// scenario intended.
func TestReplaceStatusInFrontmatter_DoesNotMutateBodyContent(t *testing.T) {
	t.Parallel()
	input := []byte("---\nid: E-0001\ntitle: Engine\nstatus: proposed\n---\n\n## Goal\n\n" +
		"Example transition in code:\n\n    status: proposed -> status: active\n\nMore prose.\n")
	got, err := replaceStatusInFrontmatter(input, "proposed", "active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gotStr := string(got)
	// Frontmatter line moved.
	if !strings.Contains(gotStr, "\nstatus: active\n---") {
		t.Errorf("frontmatter `status: active` line not found at the expected position; got:\n%s", gotStr)
	}
	// Body line preserved verbatim — both `status: proposed`
	// and `status: active` continue to appear as the original
	// prose example wrote them, untouched by the helper.
	if !strings.Contains(gotStr, "    status: proposed -> status: active\n") {
		t.Errorf("body content `status: proposed` literal was mutated; expected the exact original line `    status: proposed -> status: active` to be present unchanged; got:\n%s", gotStr)
	}
	// And exactly one `status: active` substring per the
	// frontmatter slot — the naive replace would have
	// produced two (frontmatter + body code-fence).
	if strings.Count(gotStr, "status: active") != 2 {
		// Expecting two occurrences: one in frontmatter (the
		// mutation), one in body (preserved from the prose
		// example). Three would mean the naive replace
		// touched the body line too.
		t.Errorf("expected exactly 2 occurrences of `status: active` (one frontmatter, one body prose); got %d; output:\n%s",
			strings.Count(gotStr, "status: active"), gotStr)
	}
}

// TestReplaceStatusInFrontmatter_CRLFOpener pins Windows-host
// fixture support: an entity body written with CRLF line
// endings opens with `---\r\n`, not `---\n`. The helper
// recognizes both per the kernel's frontmatter parser
// (parseStatusFromFrontmatter in internal/check/).
func TestReplaceStatusInFrontmatter_CRLFOpener(t *testing.T) {
	t.Parallel()
	input := []byte("---\r\nid: E-0001\r\nstatus: proposed\r\n---\r\n\r\nbody\r\n")
	got, err := replaceStatusInFrontmatter(input, "proposed", "active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(got), "status: active") {
		t.Errorf("CRLF input did not produce a `status: active` line; got:\n%s", string(got))
	}
	if strings.Contains(string(got), "status: proposed") {
		t.Errorf("CRLF input still contains `status: proposed`; got:\n%s", string(got))
	}
}

// TestReplaceStatusInFrontmatter_NoOpener errors loudly: a file
// that doesn't open with `---` is not a YAML-frontmatter
// document. Fabrication-time shape error, surfaced as a test
// failure rather than a silent no-op.
func TestReplaceStatusInFrontmatter_NoOpener(t *testing.T) {
	t.Parallel()
	input := []byte("just body content\nstatus: proposed\n")
	_, err := replaceStatusInFrontmatter(input, "proposed", "active")
	if err == nil {
		t.Fatal("expected error for input without frontmatter opener; got nil")
	}
	if !strings.Contains(err.Error(), "frontmatter opener") {
		t.Errorf("error message should name the opener-missing shape; got %q", err.Error())
	}
}

// TestReplaceStatusInFrontmatter_NoCloser errors loudly: an
// unterminated frontmatter is a malformed entity body. The
// helper does NOT speculatively replace inside an open-ended
// frontmatter.
func TestReplaceStatusInFrontmatter_NoCloser(t *testing.T) {
	t.Parallel()
	input := []byte("---\nid: E-0001\nstatus: proposed\nthis frontmatter has no closing marker\n")
	_, err := replaceStatusInFrontmatter(input, "proposed", "active")
	if err == nil {
		t.Fatal("expected error for input without frontmatter closer; got nil")
	}
	if !strings.Contains(err.Error(), "closing") {
		t.Errorf("error message should name the closer-missing shape; got %q", err.Error())
	}
}

// TestReplaceStatusInFrontmatter_PriorStatusAbsent errors when
// the frontmatter is well-formed but its status doesn't match
// the requested prior. Catches a fixture-setup mistake where
// the entity is already at a different status than the
// fabrication intends.
func TestReplaceStatusInFrontmatter_PriorStatusAbsent(t *testing.T) {
	t.Parallel()
	input := []byte("---\nid: E-0001\nstatus: active\n---\nbody\n")
	_, err := replaceStatusInFrontmatter(input, "proposed", "active")
	if err == nil {
		t.Fatal("expected error when frontmatter status doesn't match prior; got nil")
	}
	if !strings.Contains(err.Error(), "proposed") {
		t.Errorf("error message should name the missing prior value; got %q", err.Error())
	}
}
