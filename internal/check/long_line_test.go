package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/entity"
)

// longLine is past both bufio.Scanner's 64 KiB default token size and
// the 1 MiB ceiling some scanners raise it to, so a reader capped at
// either stops before it.
var longLine = strings.Repeat("x", 2*1024*1024)

// TestBodyLineReaders_ReadPastAnyLineLength pins G-0666: every line
// reader in this package reads a body to its end whatever a line's
// length, so content after a long line is seen and a section holding
// only a long line is not reported empty.
func TestBodyLineReaders_ReadPastAnyLineLength(t *testing.T) {
	t.Parallel()

	t.Run("EmptyRequiredSections", func(t *testing.T) {
		t.Parallel()
		body := []byte("## What's missing\n\n" + longLine + "\n\n## Why it matters\n\nReal prose.\n")
		if got := EmptyRequiredSections(entity.KindGap, body); got != nil {
			t.Errorf("EmptyRequiredSections = %v, want none", got)
		}
	})

	t.Run("scanACBodies", func(t *testing.T) {
		t.Parallel()
		body := []byte("### AC-1 — long\n\n" + longLine + "\n\n### AC-2 — after\n\nafter prose\n")
		got := map[string]string{}
		for id, content := range scanACBodies(body) {
			got[id] = string(content)
		}
		want := map[string]string{
			"AC-1": "\n" + longLine + "\n\n",
			"AC-2": "\nafter prose\n",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("scanACBodies mismatch (-want +got):\n%.500s", diff)
		}
	})

	t.Run("scanACHeadings", func(t *testing.T) {
		t.Parallel()
		body := []byte("### AC-1 — first\n" + longLine + "\n### AC-2 — after\n")
		want := map[string]int{"AC-1": 1, "AC-2": 1}
		if diff := cmp.Diff(want, scanACHeadings(body)); diff != "" {
			t.Errorf("scanACHeadings mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("scanFieldLines", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "entity.md")
		content := "---\ntitle: " + longLine + "\nstatus: open\n---\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		want := map[string]int{"title": 2, "status": 3}
		if diff := cmp.Diff(want, scanFieldLines(path)); diff != "" {
			t.Errorf("scanFieldLines mismatch (-want +got):\n%s", diff)
		}
	})
}

// TestBodyLineReaders_CRLFReadsAsLF pins that the AC readers strip a
// CRLF terminator: a body saved with CRLF line endings yields what its
// LF form yields. The bare `### AC-1` heading is the case that depends
// on it, since the heading patterns anchor at line end.
func TestBodyLineReaders_CRLFReadsAsLF(t *testing.T) {
	t.Parallel()
	lf := "### AC-1\n\nprose\n\n### AC-2 — titled\n\n#### sub\n\n## Tail\n\ntext\n"
	crlf := strings.ReplaceAll(lf, "\n", "\r\n")

	if diff := cmp.Diff(scanACHeadings([]byte(lf)), scanACHeadings([]byte(crlf))); diff != "" {
		t.Errorf("scanACHeadings mismatch (-lf +crlf):\n%s", diff)
	}
	if diff := cmp.Diff(scanACBodies([]byte(lf)), scanACBodies([]byte(crlf))); diff != "" {
		t.Errorf("scanACBodies mismatch (-lf +crlf):\n%s", diff)
	}
}
