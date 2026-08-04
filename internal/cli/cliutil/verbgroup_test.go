package cliutil_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// newTestGroup builds a marked verb group carrying one child, with both
// output streams captured — the shape MarkVerbGroup's callers build.
func newTestGroup(t *testing.T) (group *cobra.Command, out *bytes.Buffer) {
	t.Helper()
	group = cliutil.MarkVerbGroup(&cobra.Command{
		Use:   "group",
		Short: "A namespace over subverbs",
	})
	group.AddCommand(&cobra.Command{
		Use:  "child",
		RunE: func(*cobra.Command, []string) error { return nil },
	})
	out = &bytes.Buffer{}
	group.SetOut(out)
	group.SetErr(out)
	return group, out
}

// TestMarkVerbGroup_RejectsUnrecognizedSubverb pins the reason
// MarkVerbGroup supplies a RunE at all: Cobra short-circuits a command
// that is not Runnable to help before it validates arguments, so an Args
// constraint on a bare namespace never runs. A marked group is Runnable,
// so NoArgs rejects a name the group does not carry.
func TestMarkVerbGroup_RejectsUnrecognizedSubverb(t *testing.T) {
	t.Parallel()
	group, out := newTestGroup(t)
	group.SetArgs([]string{"no-such-subverb"})

	err := group.Execute()
	if err == nil {
		t.Fatalf("unrecognized subverb returned no error; output was:\n%s", out)
	}
	// A rejection, not an exit-code carrier: Cobra's own Args error
	// reaches the dispatcher, which maps a plain error to ExitUsage.
	var ee *cliutil.ExitError
	if errors.As(err, &ee) {
		t.Errorf("err = %v (*cliutil.ExitError), want Cobra's argument-validation error", err)
	}
	if got := err.Error(); !bytes.Contains([]byte(got), []byte("no-such-subverb")) {
		t.Errorf("err = %q, want it to name the rejected subverb", got)
	}
}

// TestMarkVerbGroup_BareInvocationPrintsHelpAndReportsUsage pins the
// no-subverb case: naming a group completes no operation, so it reports
// ExitUsage — while still printing the help that routes the operator to
// the subverbs.
func TestMarkVerbGroup_BareInvocationPrintsHelpAndReportsUsage(t *testing.T) {
	t.Parallel()
	group, out := newTestGroup(t)
	group.SetArgs([]string{})

	err := group.Execute()
	var ee *cliutil.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("err = %v (%T), want *cliutil.ExitError", err, err)
	}
	if ee.Code != cliutil.ExitUsage {
		t.Errorf("exit code = %d, want %d", ee.Code, cliutil.ExitUsage)
	}
	if s := out.String(); !bytes.Contains([]byte(s), []byte("child")) {
		t.Errorf("help output does not list the subverb, so it cannot route the operator:\n%s", s)
	}
}

// TestMarkVerbGroup_DispatchesToChild asserts the marker does not shadow
// the children it exists to namespace.
func TestMarkVerbGroup_DispatchesToChild(t *testing.T) {
	t.Parallel()
	group, out := newTestGroup(t)
	group.SetArgs([]string{"child"})

	if err := group.Execute(); err != nil {
		t.Errorf("dispatching to child: %v\n%s", err, out)
	}
}

// TestIsVerbGroup separates a marked group from every other command
// shape, and pins that marking preserves annotations already present —
// the marker adds a key rather than replacing the map.
func TestIsVerbGroup(t *testing.T) {
	t.Parallel()

	if got := cliutil.IsVerbGroup(&cobra.Command{Use: "plain"}); got {
		t.Error("IsVerbGroup(unmarked command) = true, want false")
	}
	if got := cliutil.IsVerbGroup(&cobra.Command{
		Use:         "annotated",
		Annotations: map[string]string{"unrelated": "value"},
	}); got {
		t.Error("IsVerbGroup(command with unrelated annotations) = true, want false")
	}

	preexisting := cliutil.MarkVerbGroup(&cobra.Command{
		Use:         "group",
		Annotations: map[string]string{"unrelated": "value"},
	})
	if !cliutil.IsVerbGroup(preexisting) {
		t.Error("IsVerbGroup(marked command) = false, want true")
	}
	if got := preexisting.Annotations["unrelated"]; got != "value" {
		t.Errorf("pre-existing annotation = %q, want %q", got, "value")
	}
}
