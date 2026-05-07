package main

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/gitops"
)

// TestAddAC_BodyFile_BinaryEndToEnd is the M-067/AC-1 closure: drive
// the dispatcher seam against a real binary and assert that
// `aiwf add ac M-NNN --title "..." --body-file ./body.md` produces an
// AC whose body section under `### AC-N — <title>` contains the file's
// content, in the same atomic commit as the AC creation.
func TestAddAC_BodyFile_BinaryEndToEnd(t *testing.T) {
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
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--epic", "E-01", "--title", "Body milestone"); err != nil {
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
		"add", "ac", "M-001",
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
		if trailer.Key == "aiwf-entity" && trailer.Value == "M-001/AC-1" {
			sawEntity = true
		}
	}
	if !sawEntity {
		t.Errorf("HEAD missing aiwf-entity: M-001/AC-1 trailer; got %v", tr)
	}

	// Milestone file contains the AC heading and body content beneath it.
	matches, err := filepath.Glob(filepath.Join(root, "work", "epics", "E-01-*", "M-001-*.md"))
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
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--epic", "E-01", "--title", "Body milestone"); err != nil {
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
		"add", "ac", "M-001",
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
	wantEntities := []string{"M-001/AC-1", "M-001/AC-2"}
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
	matches, err := filepath.Glob(filepath.Join(root, "work", "epics", "E-01-*", "M-001-*.md"))
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
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--epic", "E-01", "--title", "Body milestone"); err != nil {
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
				"add", "ac", "M-001",
				"--title", "T1", "--title", "T2", "--title", "T3",
				"--body-file", body1, "--body-file", body2,
			},
			wantTitles: "3 titles",
			wantBodies: "2 body files",
		},
		{
			name: "more body files than titles",
			args: []string{
				"add", "ac", "M-001",
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
	matches, err := filepath.Glob(filepath.Join(root, "work", "epics", "E-01-*", "M-001-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob milestone: matches=%v err=%v", matches, err)
	}
	got, _ := os.ReadFile(matches[0])
	if strings.Contains(string(got), "### AC-1") {
		t.Errorf("AC was created despite count mismatch:\n%s", got)
	}
}

// TestAddAC_BodyFile_MissingFile_ExitsUsage covers the defensive
// branch in runAddACCmd's body-file loop: when the path does not
// resolve, the verb exits with the usage code (2) and creates no AC.
// This pins the error-path coverage that the AC-1 happy path leaves
// unexercised.
func TestAddAC_BodyFile_MissingFile_ExitsUsage(t *testing.T) {
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
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--epic", "E-01", "--title", "Body milestone"); err != nil {
		t.Fatalf("add milestone: %v\n%s", err, out)
	}

	out, err := runBin(t, root, binDir, nil,
		"add", "ac", "M-001",
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
	matches, _ := filepath.Glob(filepath.Join(root, "work", "epics", "E-01-*", "M-001-*.md"))
	if len(matches) != 1 {
		t.Fatalf("milestone glob: %v", matches)
	}
	got, _ := os.ReadFile(matches[0])
	if strings.Contains(string(got), "### AC-1") {
		t.Errorf("AC was created despite missing body file:\n%s", got)
	}
}
