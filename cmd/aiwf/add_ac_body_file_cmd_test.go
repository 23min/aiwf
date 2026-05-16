package main

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

// TestAddAC_BodyFile_BinaryEndToEnd is the M-067/AC-1 closure: drive
// the dispatcher seam against a real binary and assert that
// `aiwf add ac M-NNN --title "..." --body-file ./body.md` produces an
// AC whose body section under `### AC-N — <title>` contains the file's
// content, in the same atomic commit as the AC creation.
func TestAddAC_BodyFile_BinaryEndToEnd(t *testing.T) {
	t.Parallel()
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Body epic"); err != nil {
		t.Fatalf("add epic: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Body milestone"); err != nil {
		t.Fatalf("add milestone: %v\n%s", err, out)
	}

	bodyText := "Concrete pass criteria: the verb populates the body in the same commit.\n\nEdge case: an empty file produces an empty body section.\n"
	bodyPath := filepath.Join(root, "ac-body.md")
	if err := os.WriteFile(bodyPath, []byte(bodyText), 0o644); err != nil {
		t.Fatalf("write body file: %v", err)
	}

	// Head commit count before add-ac, so atomicity can be asserted.
	headBefore, err := runGit(root, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatalf("rev-list before: %v\n%s", err, headBefore)
	}

	out, err := runBin(t, root, binDir, nil,
		"add", "ac", "M-0001",
		"--title", "First AC",
		"--body-file", bodyPath)
	if err != nil {
		t.Fatalf("aiwf add ac --body-file: %v\n%s", err, out)
	}

	// Atomicity: exactly one commit was added.
	headAfter, err := runGit(root, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatalf("rev-list after: %v\n%s", err, headAfter)
	}
	before, err := strconv.Atoi(strings.TrimSpace(headBefore))
	if err != nil {
		t.Fatalf("parse rev-list before: %v", err)
	}
	after, err := strconv.Atoi(strings.TrimSpace(headAfter))
	if err != nil {
		t.Fatalf("parse rev-list after: %v", err)
	}
	if after != before+1 {
		t.Errorf("commit count after add-ac = %d, want %d (one new commit)", after, before+1)
	}

	// Trailer carries the AC composite id.
	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	var sawEntity bool
	for _, trailer := range tr {
		if trailer.Key == "aiwf-entity" && trailer.Value == "M-0001/AC-1" {
			sawEntity = true
		}
	}
	if !sawEntity {
		t.Errorf("HEAD missing aiwf-entity: M-001/AC-1 trailer; got %v", tr)
	}

	// Milestone file contains the AC heading and body content beneath it.
	matches, err := filepath.Glob(filepath.Join(root, "work", "epics", "E-0001-*", "M-0001-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob milestone: matches=%v err=%v", matches, err)
	}
	got, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	gotStr := string(got)

	headingIdx := strings.Index(gotStr, "### AC-1 — First AC")
	if headingIdx < 0 {
		t.Fatalf("milestone missing AC-1 heading:\n%s", gotStr)
	}
	bodyIdx := strings.Index(gotStr, "Concrete pass criteria")
	if bodyIdx < 0 {
		t.Fatalf("milestone missing AC-1 body content from --body-file:\n%s", gotStr)
	}
	if bodyIdx < headingIdx {
		t.Errorf("body content appeared before AC-1 heading (offset %d vs %d):\n%s", bodyIdx, headingIdx, gotStr)
	}
}

// TestAddAC_BodyFile_MultiAC_PositionalPairing is the M-067/AC-2
// closure: drive the dispatcher seam against a real binary and
// assert that
//
//	aiwf add ac M-NNN --title T1 --body-file b1.md \
//	                  --title T2 --body-file b2.md
//
// produces two ACs in one atomic commit, with each AC's body taken
// from the positionally-matching --body-file. This pins the
// "Nth --body-file populates the Nth AC" contract that the AC-1
// wiring already supports through the AddACBatch loop, so a future
// refactor cannot silently break the pairing for batches > 1.
//
// Order across flag types is left as the typical operator
// invocation ("--title T1 --body-file b1.md --title T2 --body-file
// b2.md"); pflag's StringArrayVar preserves per-flag argv order, so
// the pairing falls out from the loop's index. AC-3 covers the
// mismatched-counts refusal in a later cycle.
func TestAddAC_BodyFile_MultiAC_PositionalPairing(t *testing.T) {
	t.Parallel()
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Body epic"); err != nil {
		t.Fatalf("add epic: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Body milestone"); err != nil {
		t.Fatalf("add milestone: %v\n%s", err, out)
	}

	// Distinct, recognizable per-AC body content so a swap or merge
	// is detectable from the produced milestone file alone.
	body1Text := "First AC body. Specific to AC-1.\n"
	body2Text := "Second AC body. Specific to AC-2.\n"
	body1Path := filepath.Join(root, "ac1-body.md")
	body2Path := filepath.Join(root, "ac2-body.md")
	for _, w := range []struct {
		path, content string
	}{
		{body1Path, body1Text},
		{body2Path, body2Text},
	} {
		if err := os.WriteFile(w.path, []byte(w.content), 0o644); err != nil {
			t.Fatalf("write %s: %v", w.path, err)
		}
	}

	headBefore, err := runGit(root, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatalf("rev-list before: %v\n%s", err, headBefore)
	}

	out, err := runBin(t, root, binDir, nil,
		"add", "ac", "M-0001",
		"--title", "First AC",
		"--body-file", body1Path,
		"--title", "Second AC",
		"--body-file", body2Path)
	if err != nil {
		t.Fatalf("aiwf add ac multi --body-file: %v\n%s", err, out)
	}

	// Atomicity: two ACs land in exactly one new commit.
	headAfter, err := runGit(root, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatalf("rev-list after: %v\n%s", err, headAfter)
	}
	before, err := strconv.Atoi(strings.TrimSpace(headBefore))
	if err != nil {
		t.Fatalf("parse rev-list before: %v", err)
	}
	after, err := strconv.Atoi(strings.TrimSpace(headAfter))
	if err != nil {
		t.Fatalf("parse rev-list after: %v", err)
	}
	if after != before+1 {
		t.Errorf("commit count after multi add-ac = %d, want %d (one new commit)", after, before+1)
	}

	// The single new commit carries one aiwf-entity trailer per AC,
	// in allocation order.
	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	var entityTrailers []string
	for _, trailer := range tr {
		if trailer.Key == "aiwf-entity" {
			entityTrailers = append(entityTrailers, trailer.Value)
		}
	}
	wantEntities := []string{"M-0001/AC-1", "M-0001/AC-2"}
	if len(entityTrailers) != len(wantEntities) {
		t.Fatalf("aiwf-entity trailers = %v, want %v", entityTrailers, wantEntities)
	}
	for i, want := range wantEntities {
		if entityTrailers[i] != want {
			t.Errorf("aiwf-entity[%d] = %q, want %q", i, entityTrailers[i], want)
		}
	}

	// The milestone file shows two AC headings, each followed by the
	// body content from the matching --body-file. Crucially: AC-1's
	// section contains body1Text and not body2Text (and vice versa),
	// so a swap of the pairing would be caught.
	matches, err := filepath.Glob(filepath.Join(root, "work", "epics", "E-0001-*", "M-0001-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob milestone: matches=%v err=%v", matches, err)
	}
	got, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	gotStr := string(got)

	ac1Heading := "### AC-1 — First AC"
	ac2Heading := "### AC-2 — Second AC"
	ac1Idx := strings.Index(gotStr, ac1Heading)
	ac2Idx := strings.Index(gotStr, ac2Heading)
	if ac1Idx < 0 || ac2Idx < 0 {
		t.Fatalf("missing AC headings (ac1Idx=%d ac2Idx=%d):\n%s", ac1Idx, ac2Idx, gotStr)
	}
	if ac1Idx >= ac2Idx {
		t.Fatalf("AC-1 heading must precede AC-2 heading; got ac1Idx=%d ac2Idx=%d", ac1Idx, ac2Idx)
	}
	// AC-1's section is the slice from its heading up to AC-2's
	// heading; AC-2's section runs from its heading to EOF.
	ac1Section := gotStr[ac1Idx:ac2Idx]
	ac2Section := gotStr[ac2Idx:]
	body1Marker := "Specific to AC-1"
	body2Marker := "Specific to AC-2"
	if !strings.Contains(ac1Section, body1Marker) {
		t.Errorf("AC-1 section missing body1 marker %q:\n%s", body1Marker, ac1Section)
	}
	if strings.Contains(ac1Section, body2Marker) {
		t.Errorf("AC-1 section unexpectedly contains body2 marker %q (pairing swapped?):\n%s", body2Marker, ac1Section)
	}
	if !strings.Contains(ac2Section, body2Marker) {
		t.Errorf("AC-2 section missing body2 marker %q:\n%s", body2Marker, ac2Section)
	}
	if strings.Contains(ac2Section, body1Marker) {
		t.Errorf("AC-2 section unexpectedly contains body1 marker %q (pairing swapped?):\n%s", body1Marker, ac2Section)
	}
}

// TestAddAC_BodyFile_CountMismatch_RefusesPreAllocation is the
// M-067/AC-3 closure: when --body-file is provided but the
// per-flag counts of --title and --body-file differ, the verb
// exits with usage code 2 before any id allocation, lock
// acquisition, or disk write. The error message must let the
// operator self-correct: observed counts, the positional-pairing
// rule, and the note that omitting --body-file entirely is also
// valid (the AC-6 path).
//
// The test pins both shapes of the mismatch (more titles than
// bodies, more bodies than titles) so a future refactor can't
// accept one direction silently.
func TestAddAC_BodyFile_CountMismatch_RefusesPreAllocation(t *testing.T) {
	t.Parallel()
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Body epic"); err != nil {
		t.Fatalf("add epic: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Body milestone"); err != nil {
		t.Fatalf("add milestone: %v\n%s", err, out)
	}

	// Two real body files; the test invocations vary the counts
	// without ever providing a missing path (we want to isolate
	// the count-check, not the file-read error path).
	body1 := filepath.Join(root, "b1.md")
	body2 := filepath.Join(root, "b2.md")
	for _, p := range []string{body1, body2} {
		if err := os.WriteFile(p, []byte("body for "+filepath.Base(p)+"\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}

	headBefore, err := runGit(root, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatalf("rev-list before: %v\n%s", err, headBefore)
	}

	cases := []struct {
		name       string
		args       []string
		wantTitles string
		wantBodies string
	}{
		{
			name: "more titles than body files",
			args: []string{
				"add", "ac", "M-0001",
				"--title", "T1", "--title", "T2", "--title", "T3",
				"--body-file", body1, "--body-file", body2,
			},
			wantTitles: "3 titles",
			wantBodies: "2 body files",
		},
		{
			name: "more body files than titles",
			args: []string{
				"add", "ac", "M-0001",
				"--title", "T1", "--title", "T2",
				"--body-file", body1, "--body-file", body2, "--body-file", body1,
			},
			wantTitles: "2 titles",
			wantBodies: "3 body files",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, runErr := runBin(t, root, binDir, nil, tc.args...)
			if runErr == nil {
				t.Fatalf("expected count-mismatch refusal; got success:\n%s", out)
			}
			if !strings.Contains(out, tc.wantTitles) {
				t.Errorf("error missing observed title count %q:\n%s", tc.wantTitles, out)
			}
			if !strings.Contains(out, tc.wantBodies) {
				t.Errorf("error missing observed body-file count %q:\n%s", tc.wantBodies, out)
			}
			// Pairing rule is stated so the operator knows the
			// ordering contract. "positional" is the canonical
			// keyword from the AC-2 contract.
			if !strings.Contains(out, "positional") {
				t.Errorf("error missing pairing rule (expected the word 'positional'):\n%s", out)
			}
			// AC-6 hint: bodyless invocations are still valid.
			if !strings.Contains(out, "omit") {
				t.Errorf("error missing 'omit --body-file is valid' hint:\n%s", out)
			}
		})
	}

	// Pre-allocation check: nothing was committed across the
	// whole run, and the milestone still has no ACs.
	headAfter, err := runGit(root, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatalf("rev-list after: %v\n%s", err, headAfter)
	}
	if strings.TrimSpace(headBefore) != strings.TrimSpace(headAfter) {
		t.Errorf("commit count changed across refusal cases (%s -> %s); refusal must precede commit",
			strings.TrimSpace(headBefore), strings.TrimSpace(headAfter))
	}
	matches, err := filepath.Glob(filepath.Join(root, "work", "epics", "E-0001-*", "M-0001-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob milestone: matches=%v err=%v", matches, err)
	}
	got, _ := os.ReadFile(matches[0])
	if strings.Contains(string(got), "### AC-1") {
		t.Errorf("AC was created despite count mismatch:\n%s", got)
	}
}

// TestAddAC_BodyFile_LeadingFrontmatter_Refused is the M-067/AC-4
// closure: a body file whose first non-blank line is `---` exits
// the verb with code 2 (usage error). The error message names the
// offending file path and the rule, so the operator can fix the
// file without re-reading the help.
//
// The two subcases cover the bare leading `---\n` and the
// leading-whitespace-then-`---` case (the validator trims leading
// whitespace before the prefix check, mirroring
// internal/verb/common.go:validateUserBodyBytes).
func TestAddAC_BodyFile_LeadingFrontmatter_Refused(t *testing.T) {
	t.Parallel()
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Body epic"); err != nil {
		t.Fatalf("add epic: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Body milestone"); err != nil {
		t.Fatalf("add milestone: %v\n%s", err, out)
	}

	headBefore, err := runGit(root, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatalf("rev-list before: %v\n%s", err, headBefore)
	}

	cases := []struct {
		name    string
		content string
	}{
		{
			name:    "leading triple-dash",
			content: "---\nid: AC-9\ntitle: stowaway\n---\n\nfake body\n",
		},
		{
			name:    "leading blank lines then triple-dash",
			content: "\n\n  \n---\nrogue: yes\n---\n\nfake body\n",
		},
		{
			// CRLF arm of the prefix check — Windows-edited files
			// would otherwise slip past a Unix-only `---\n` match.
			name:    "leading triple-dash with CRLF",
			content: "---\r\nid: AC-9\r\n---\r\n\r\nfake body\r\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bodyPath := filepath.Join(root, "fm-"+strings.ReplaceAll(tc.name, " ", "_")+".md")
			if writeErr := os.WriteFile(bodyPath, []byte(tc.content), 0o644); writeErr != nil {
				t.Fatalf("write %s: %v", bodyPath, writeErr)
			}
			out, runErr := runBin(t, root, binDir, nil,
				"add", "ac", "M-0001",
				"--title", "T",
				"--body-file", bodyPath)
			if runErr == nil {
				t.Fatalf("expected leading-frontmatter refusal; got success:\n%s", out)
			}
			// Error must name the offending path so the operator
			// can fix the right file.
			if !strings.Contains(out, bodyPath) {
				t.Errorf("error missing offending path %q:\n%s", bodyPath, out)
			}
			// Error must name the rule. "frontmatter" is the
			// canonical keyword from the existing
			// validateUserBodyBytes message.
			if !strings.Contains(out, "frontmatter") {
				t.Errorf("error missing 'frontmatter' rule keyword:\n%s", out)
			}
		})
	}

	// Pre-allocation check: no commit added across the run, no AC
	// in the milestone.
	headAfter, err := runGit(root, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatalf("rev-list after: %v\n%s", err, headAfter)
	}
	if strings.TrimSpace(headBefore) != strings.TrimSpace(headAfter) {
		t.Errorf("commit count changed across refusal cases (%s -> %s); refusal must precede commit",
			strings.TrimSpace(headBefore), strings.TrimSpace(headAfter))
	}
	matches, err := filepath.Glob(filepath.Join(root, "work", "epics", "E-0001-*", "M-0001-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob milestone: matches=%v err=%v", matches, err)
	}
	got, _ := os.ReadFile(matches[0])
	if strings.Contains(string(got), "### AC-1") {
		t.Errorf("AC was created despite leading-frontmatter body file:\n%s", got)
	}
}

// TestAddAC_BodyFile_Stdin_SingleTitle_Succeeds covers the happy
// path of M-067/AC-5: `aiwf add ac M-NNN --title T --body-file -`
// reads body content from stdin and lands it under the AC heading,
// consistent with the existing whole-entity --body-file - shorthand.
func TestAddAC_BodyFile_Stdin_SingleTitle_Succeeds(t *testing.T) {
	t.Parallel()
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Body epic"); err != nil {
		t.Fatalf("add epic: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Body milestone"); err != nil {
		t.Fatalf("add milestone: %v\n%s", err, out)
	}

	stdinText := "Body piped from stdin.\n\nSpecific to AC-1.\n"

	out, err := runBinStdin(t, root, binDir, strings.NewReader(stdinText),
		"add", "ac", "M-0001",
		"--title", "Stdin AC",
		"--body-file", "-")
	if err != nil {
		t.Fatalf("aiwf add ac --body-file -: %v\n%s", err, out)
	}

	matches, err := filepath.Glob(filepath.Join(root, "work", "epics", "E-0001-*", "M-0001-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob milestone: matches=%v err=%v", matches, err)
	}
	got, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	gotStr := string(got)
	if !strings.Contains(gotStr, "### AC-1 — Stdin AC") {
		t.Fatalf("milestone missing AC-1 heading:\n%s", gotStr)
	}
	if !strings.Contains(gotStr, "Specific to AC-1") {
		t.Errorf("milestone missing stdin body content marker:\n%s", gotStr)
	}
}

// TestAddAC_BodyFile_Stdin_MultiTitle_Refused covers the refusal
// path of M-067/AC-5: when more than one --title is provided and
// any --body-file value is `-`, the verb exits with code 2 before
// stdin is consumed. Stdin is one stream — silently routing it to
// "the first AC" would surprise the operator. The check must fire
// pre-read so a piped operator doesn't lose their input on a
// doomed invocation.
//
// The two subcases pin both shapes: stdin used for every AC
// (--body-file - --body-file -) and stdin mixed with a real file
// (--body-file - --body-file file.md). Either form is forbidden.
func TestAddAC_BodyFile_Stdin_MultiTitle_Refused(t *testing.T) {
	t.Parallel()
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Body epic"); err != nil {
		t.Fatalf("add epic: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Body milestone"); err != nil {
		t.Fatalf("add milestone: %v\n%s", err, out)
	}

	realFile := filepath.Join(root, "real.md")
	if err := os.WriteFile(realFile, []byte("real body\n"), 0o644); err != nil {
		t.Fatalf("write real file: %v", err)
	}

	headBefore, err := runGit(root, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatalf("rev-list before: %v\n%s", err, headBefore)
	}

	cases := []struct {
		name string
		args []string
	}{
		{
			name: "stdin used for every AC",
			args: []string{
				"add", "ac", "M-0001",
				"--title", "T1", "--title", "T2",
				"--body-file", "-", "--body-file", "-",
			},
		},
		{
			// realFile listed first so the loop walks past a
			// non-stdin entry before hitting `-` — exercises the
			// "skip non-stdin path" arm of the inner check too.
			name: "stdin mixed with a real file (stdin second)",
			args: []string{
				"add", "ac", "M-0001",
				"--title", "T1", "--title", "T2",
				"--body-file", realFile, "--body-file", "-",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Provide a non-empty stdin so a green-on-empty bug
			// can't masquerade as the refusal we want.
			out, runErr := runBinStdin(t, root, binDir,
				strings.NewReader("would-be body content from stdin\n"),
				tc.args...)
			if runErr == nil {
				t.Fatalf("expected stdin+multi-title refusal; got success:\n%s", out)
			}
			// Error names the constraint.
			if !strings.Contains(out, "stdin") && !strings.Contains(out, "--body-file -") {
				t.Errorf("error missing stdin/`--body-file -` mention:\n%s", out)
			}
			if !strings.Contains(out, "single --title") {
				t.Errorf("error missing 'single --title' constraint phrasing:\n%s", out)
			}
		})
	}

	// Pre-read check: nothing was committed across the run.
	headAfter, err := runGit(root, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatalf("rev-list after: %v\n%s", err, headAfter)
	}
	if strings.TrimSpace(headBefore) != strings.TrimSpace(headAfter) {
		t.Errorf("commit count changed across refusal cases (%s -> %s); refusal must precede commit",
			strings.TrimSpace(headBefore), strings.TrimSpace(headAfter))
	}
	matches, err := filepath.Glob(filepath.Join(root, "work", "epics", "E-0001-*", "M-0001-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob milestone: matches=%v err=%v", matches, err)
	}
	got, _ := os.ReadFile(matches[0])
	if strings.Contains(string(got), "### AC-1") {
		t.Errorf("AC was created despite stdin+multi-title refusal:\n%s", got)
	}
}

// TestAddAC_NoBodyFile_LeavesBodyEmpty pins M-067/AC-6: when
// --body-file is omitted entirely, the verb's pre-AC-1 behavior
// holds — the AC frontmatter is allocated and the bare `### AC-N
// — <title>` heading is scaffolded with no body content under it.
// The friction-reducing flag stays opt-in; the multi-AC
// quick-scaffold flow (operator still figuring out what each AC
// means) keeps working. M-066's entity-body-empty check is the
// downstream chokepoint; this AC pins that the verb itself does
// not pre-emptively force a body.
//
// The two subcases pin both single- and multi-title invocations.
// "Empty body" here means: between this AC's heading and either
// the next `### ` heading or EOF, nothing but whitespace appears.
func TestAddAC_NoBodyFile_LeavesBodyEmpty(t *testing.T) {
	t.Parallel()
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	cases := []struct {
		name    string
		args    []string
		wantACs []string // AC heading suffixes ("AC-N — <title>") to verify exist with empty bodies
	}{
		{
			name:    "single AC, no --body-file",
			args:    []string{"add", "ac", "M-0001", "--title", "Quick scaffold"},
			wantACs: []string{"AC-1 — Quick scaffold"},
		},
		{
			name: "multi AC, no --body-file",
			args: []string{
				"add", "ac", "M-0001",
				"--title", "First quick AC",
				"--title", "Second quick AC",
				"--title", "Third quick AC",
			},
			wantACs: []string{
				"AC-1 — First quick AC",
				"AC-2 — Second quick AC",
				"AC-3 — Third quick AC",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if out, err := runGit(root, "init", "-q"); err != nil {
				t.Fatalf("git init: %v\n%s", err, out)
			}
			for _, args := range [][]string{
				{"config", "user.email", "peter@example.com"},
				{"config", "user.name", "Peter Test"},
			} {
				if out, err := runGit(root, args...); err != nil {
					t.Fatalf("git %v: %v\n%s", args, err, out)
				}
			}
			if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
				t.Fatalf("aiwf init: %v\n%s", err, out)
			}
			if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Body epic"); err != nil {
				t.Fatalf("add epic: %v\n%s", err, out)
			}
			if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Body milestone"); err != nil {
				t.Fatalf("add milestone: %v\n%s", err, out)
			}

			out, err := runBin(t, root, binDir, nil, tc.args...)
			if err != nil {
				t.Fatalf("aiwf add ac (no --body-file): %v\n%s", err, out)
			}

			matches, err := filepath.Glob(filepath.Join(root, "work", "epics", "E-0001-*", "M-0001-*.md"))
			if err != nil || len(matches) != 1 {
				t.Fatalf("glob milestone: matches=%v err=%v", matches, err)
			}
			got, err := os.ReadFile(matches[0])
			if err != nil {
				t.Fatalf("read milestone: %v", err)
			}
			gotStr := string(got)

			// Slice each AC's section: from its heading line to
			// either the next `### ` heading or EOF. Assert that
			// after stripping the heading line, only whitespace
			// remains.
			for i, want := range tc.wantACs {
				headingLine := "### " + want
				start := strings.Index(gotStr, headingLine)
				if start < 0 {
					t.Errorf("milestone missing heading %q:\n%s", headingLine, gotStr)
					continue
				}
				// Section starts just after the heading line.
				afterHeading := start + len(headingLine)
				if afterHeading > len(gotStr) && gotStr[afterHeading] == '\n' {
					afterHeading++
				}
				rest := gotStr[afterHeading:]
				// End of section: next "### " or EOF.
				end := strings.Index(rest, "\n### ")
				var section string
				if end < 0 {
					section = rest
				} else {
					section = rest[:end]
				}
				if strings.TrimSpace(section) != "" {
					t.Errorf("AC[%d] %q: body section non-empty (expected only whitespace):\n%q",
						i, want, section)
				}
			}
		})
	}
}

// TestAddAC_BodyFile_MissingFile_ExitsUsage covers the defensive
// branch in runAddACCmd's body-file loop: when the path does not
// resolve, the verb exits with the usage code (2) and creates no AC.
// This pins the error-path coverage that the AC-1 happy path leaves
// unexercised.
func TestAddAC_BodyFile_MissingFile_ExitsUsage(t *testing.T) {
	t.Parallel()
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Body epic"); err != nil {
		t.Fatalf("add epic: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Body milestone"); err != nil {
		t.Fatalf("add milestone: %v\n%s", err, out)
	}

	out, err := runBin(t, root, binDir, nil,
		"add", "ac", "M-0001",
		"--title", "First AC",
		"--body-file", filepath.Join(root, "definitely-not-a-file.md"))
	if err == nil {
		t.Fatalf("expected error on missing body file; got:\n%s", out)
	}
	// Output should name the offending path so the operator knows
	// which --body-file failed to resolve.
	if !strings.Contains(out, "definitely-not-a-file.md") {
		t.Errorf("expected error to name the missing path; got:\n%s", out)
	}

	// No AC was added — milestone should still have len(acs) == 0.
	matches, _ := filepath.Glob(filepath.Join(root, "work", "epics", "E-0001-*", "M-0001-*.md"))
	if len(matches) != 1 {
		t.Fatalf("milestone glob: %v", matches)
	}
	got, _ := os.ReadFile(matches[0])
	if strings.Contains(string(got), "### AC-1") {
		t.Errorf("AC was created despite missing body file:\n%s", got)
	}
}
