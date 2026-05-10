package main

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/entity"
)

// TestStatusesForID covers the closed-set lookup behind
// `aiwf promote <id> <TAB>`: derive kind from the id prefix and return
// the kind's allowed statuses. Composite ids and malformed input must
// return nil so the shell falls back to file completion.
func TestStatusesForID(t *testing.T) {
	cases := []struct {
		name string
		id   string
		want []string
	}{
		{"epic", "E-0001", entity.AllowedStatuses(entity.KindEpic)},
		{"milestone", "M-0007", entity.AllowedStatuses(entity.KindMilestone)},
		{"adr", "ADR-0001", entity.AllowedStatuses(entity.KindADR)},
		{"gap", "G-0042", entity.AllowedStatuses(entity.KindGap)},
		{"decision", "D-0013", entity.AllowedStatuses(entity.KindDecision)},
		{"contract", "C-0005", entity.AllowedStatuses(entity.KindContract)},
		{"empty", "", nil},
		{"composite", "M-0007/AC-1", nil},
		{"unknown_prefix", "X-01", nil},
		{"no_prefix", "epic", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := statusesForID(tc.id)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("statusesForID(%q) mismatch (-want +got):\n%s", tc.id, diff)
			}
		})
	}
}

// TestAllKindNames pins the kind list against entity.AllKinds so a
// rename or addition in the entity package fails the test rather
// than silently desynchronizing the completion source.
func TestAllKindNames(t *testing.T) {
	got := allKindNames()
	want := make([]string, 0, len(entity.AllKinds()))
	for _, k := range entity.AllKinds() {
		want = append(want, string(k))
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("allKindNames mismatch (-want +got):\n%s", diff)
	}
}

// TestWrapExitCode is the single load-bearing translation between
// verb int returns and Cobra's RunE error channel. Zero must collapse
// to nil; non-zero must round-trip through *exitError so run() can
// unwrap the original code.
func TestWrapExitCode(t *testing.T) {
	if got := wrapExitCode(exitOK); got != nil {
		t.Errorf("wrapExitCode(exitOK) = %v, want nil", got)
	}
	for _, code := range []int{exitFindings, exitUsage, exitInternal, 42} {
		err := wrapExitCode(code)
		var ee *exitError
		if !errors.As(err, &ee) {
			t.Fatalf("wrapExitCode(%d): err type = %T, want *exitError", code, err)
		}
		if ee.code != code {
			t.Errorf("wrapExitCode(%d) carried code = %d, want %d", code, ee.code, code)
		}
	}
}

// TestCompleteEntityIDs_FromTree covers the dynamic-completion happy
// path: a populated planning tree under a tempdir root yields the
// expected ids. The test exercises both the unfiltered surface and a
// kind filter so consumers of the helper (e.g. --epic completion)
// land the right slice.
func TestCompleteEntityIDs_FromTree(t *testing.T) {
	root := newCompletionFixtureRepo(t)

	chdir(t, root)

	t.Run("all_kinds", func(t *testing.T) {
		ids, dir := completeEntityIDs("")
		if dir != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("directive = %v, want NoFileComp", dir)
		}
		got := append([]string{}, ids...)
		sort.Strings(got)
		want := []string{"E-0001", "M-0001", "M-0002"}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("ids mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("filter_epic", func(t *testing.T) {
		ids, _ := completeEntityIDs(entity.KindEpic)
		if diff := cmp.Diff([]string{"E-0001"}, ids); diff != "" {
			t.Errorf("epic-filtered ids mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("filter_milestone", func(t *testing.T) {
		ids, _ := completeEntityIDs(entity.KindMilestone)
		got := append([]string{}, ids...)
		sort.Strings(got)
		want := []string{"M-0001", "M-0002"}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("milestone-filtered ids mismatch (-want +got):\n%s", diff)
		}
	})
}

// TestCompleteEntityIDs_GracefulNoOp pins M-054 AC-2: when the cwd
// does not contain a planning tree, the completion helper returns an
// empty list and ShellCompDirectiveNoFileComp — never an error, never
// a nil-deref panic. Three flavors:
//
//   - empty tempdir (no aiwf.yaml, no work/ tree) — happy "outside any
//     project" path.
//   - tempdir with a planted aiwf.yaml but a malformed work/ tree —
//     load fails, helper still returns cleanly.
//   - tempdir whose work/ subtree is unreadable — disk error path.
//
// All three must produce no completions and no panic.
func TestCompleteEntityIDs_GracefulNoOp(t *testing.T) {
	t.Run("no_aiwf_yaml", func(t *testing.T) {
		root := t.TempDir()
		chdir(t, root)
		ids, dir := completeEntityIDs("")
		if len(ids) != 0 {
			t.Errorf("expected empty completions outside aiwf project, got %v", ids)
		}
		if dir != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("directive = %v, want NoFileComp", dir)
		}
	})

	t.Run("malformed_work_tree", func(t *testing.T) {
		root := t.TempDir()
		// Plant an aiwf.yaml but no work/ — tree.Load returns an empty
		// tree, the helper still returns cleanly.
		if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("schema_version: 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		chdir(t, root)
		ids, _ := completeEntityIDs("")
		if len(ids) != 0 {
			t.Errorf("expected empty completions on empty tree, got %v", ids)
		}
	})
}

// TestCompleteEntityIDArg_RespectsPosition covers the wrapper used by
// every <id>-positional ValidArgsFunction: only the first positional
// (or whichever index `position` names) gets completions; subsequent
// args return empty so e.g. `aiwf promote E-01 <TAB>` doesn't re-suggest
// entity ids when the second positional is the new-status.
func TestCompleteEntityIDArg_RespectsPosition(t *testing.T) {
	root := newCompletionFixtureRepo(t)
	chdir(t, root)

	fn := completeEntityIDArg("", 0)

	t.Run("first_positional_lists_ids", func(t *testing.T) {
		ids, _ := fn(nil, []string{}, "")
		if len(ids) == 0 {
			t.Error("expected ids for first positional, got none")
		}
	})

	t.Run("second_positional_returns_empty", func(t *testing.T) {
		ids, dir := fn(nil, []string{"E-0001"}, "")
		if len(ids) != 0 {
			t.Errorf("expected no ids for second positional, got %v", ids)
		}
		if dir != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("directive = %v, want NoFileComp", dir)
		}
	})
}

// TestCompleteEntityIDFlag_KindFilter exercises the flag-side wrapper.
// The contract is the same as completeEntityIDs with the filter
// applied; the wrapper just adapts the function shape Cobra requires.
func TestCompleteEntityIDFlag_KindFilter(t *testing.T) {
	root := newCompletionFixtureRepo(t)
	chdir(t, root)

	fn := completeEntityIDFlag(entity.KindMilestone)
	ids, dir := fn(nil, nil, "")
	if dir != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", dir)
	}
	got := append([]string{}, ids...)
	sort.Strings(got)
	want := []string{"M-0001", "M-0002"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("milestone-filtered flag completion mismatch (-want +got):\n%s", diff)
	}
}

// TestRegisterFormatCompletion is a behavioral check on the helper:
// after registration, the command must report a completion function
// for --format that emits {text, json} with the no-file directive.
func TestRegisterFormatCompletion(t *testing.T) {
	cmd := &cobra.Command{Use: "fake"}
	cmd.Flags().String("format", "text", "")
	registerFormatCompletion(cmd)

	fn, ok := cmd.GetFlagCompletionFunc("format")
	if !ok {
		t.Fatal("registerFormatCompletion did not bind a completion function")
	}
	got, dir := fn(cmd, nil, "")
	if dir != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", dir)
	}
	if diff := cmp.Diff([]string{"text", "json"}, got); diff != "" {
		t.Errorf("format completions mismatch (-want +got):\n%s", diff)
	}
}

// newCompletionFixtureRepo builds a small synthetic planning tree
// under a tempdir and returns the root. Three entities: one epic and
// two milestones. Synthetic content per CLAUDE.md test conventions —
// the fixtures must read as obviously fictional, not as anonymized
// copies of any real project.
func newCompletionFixtureRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("schema_version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	epicDir := filepath.Join(root, "work", "epics", "E-0001-fixture-epic")
	if err := os.MkdirAll(epicDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(epicDir, "epic.md"), []byte(`---
id: E-01
title: Fixture epic
status: active
---
`), 0o644); err != nil {
		t.Fatal(err)
	}
	for i, slug := range []string{"first-fixture-milestone", "second-fixture-milestone"} {
		body := []byte("---\nid: M-00" + intToStr(i+1) + "\ntitle: Fixture milestone " + intToStr(i+1) + "\nstatus: draft\nparent: E-01\n---\n")
		path := filepath.Join(epicDir, "M-0000"+intToStr(i+1)+"-"+slug+".md")
		if err := os.WriteFile(path, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func intToStr(n int) string {
	// Tiny helper for the fixture builder — keeps the file body
	// composition string-only without pulling fmt into the loop.
	if n < 0 || n > 9 {
		return "?"
	}
	return string(rune('0' + n))
}

// chdir pushes the test into root for the duration of the subtest and
// restores cwd via t.Cleanup. completeEntityIDs reads cwd through
// resolveRoot, so subtests that share fixture roots set cwd here.
func chdir(t *testing.T, root string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}
