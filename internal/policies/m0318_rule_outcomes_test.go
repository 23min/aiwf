package policies

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cellcoverage"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/workflows/spec"
)

func TestM0318_AC3_TerminalStatusTargetsDeclared(t *testing.T) {
	t.Parallel()
	kinds := append(entity.AllKinds(), spec.KindAC)
	for _, kind := range kinds {
		states := entity.AllowedStatuses(kind)
		if kind == spec.KindAC {
			states = entity.AllowedACStatuses()
		}
		for _, from := range states {
			terminal := entity.IsTerminal(kind, from)
			if kind == spec.KindAC {
				terminal = entity.IsTerminalACStatus(from)
			}
			if !terminal {
				continue
			}
			for _, to := range states {
				want := spec.OutcomeIllegal
				if from == to {
					want = spec.OutcomeNoOp
				}
				found := false
				for _, r := range spec.LookupRules(kind, string(from), "promote") {
					if r.ToState != string(to) {
						continue
					}
					found = true
					if r.Outcome != want {
						t.Errorf("%s %s -> %s: outcome %v, want %v", kind, from, to, r.Outcome, want)
					}
				}
				if !found {
					t.Errorf("missing terminal cell %s %s -> %s", kind, from, to)
				}
			}
		}
	}
}

func TestM0318_AC3_DeclaredOutcomeShape(t *testing.T) {
	t.Parallel()
	for _, r := range spec.Rules() {
		if r.ToState == "" {
			t.Errorf("missing target: %+v", r)
		}
		if r.Outcome == spec.OutcomeNoOp && (r.ToState != r.FromState || r.ExpectedErrorCode != "" || r.RejectionLayer != spec.RejectionLayerNone || r.BlockingStrict) {
			t.Errorf("NoOp must converge without rejection metadata: %+v", r)
		}
	}
	if spec.OutcomeNoOp == spec.OutcomeUnspecified || spec.OutcomeNoOp == spec.OutcomeLegal || spec.OutcomeNoOp == spec.OutcomeIllegal {
		t.Fatal("NoOp must be distinct from unspecified, legal mutation, and illegal")
	}
}

func TestM0318_AC3_NoOpCells(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	for _, r := range spec.Rules() {
		if r.Outcome != spec.OutcomeNoOp {
			continue
		}
		t.Run(caseName(r, r.ToState), func(t *testing.T) {
			t.Parallel()
			f := cellcoverage.NewCellFixture(t)
			id := bringEntityForCell(t, f, r, deriveBringOpts(r))
			ctx := spec.EvalContext{}
			for _, p := range r.Preconditions {
				f.SatisfyPredicate(t, p, id, &ctx)
			}
			before := fixtureGitSnapshot(t, f.Root)
			args := buildVerbArgs(t, positiveCase{rule: r, target: r.ToState}, id, extraArgs{testMetrics: ctx.TestMetrics})
			out, err := testutil.RunBin(t, f.Root, "", nil, args...)
			if err != nil {
				t.Fatalf("NoOp exited nonzero: %v\n%s", err, out)
			}
			message := "nothing to change"
			if r.Verb == "cancel" {
				message = "nothing to cancel"
			}
			if !strings.Contains(out, message) {
				t.Errorf("NoOp message missing: %s", out)
			}
			if after := fixtureGitSnapshot(t, f.Root); !bytes.Equal(before, after) {
				t.Errorf("NoOp changed HEAD or project files")
			}
		})
	}
}

// fixtureGitSnapshot includes HEAD and tracked or untracked project file bytes,
// excluding ignored runtime artifacts. A write cannot masquerade as a NoOp
// merely by leaving HEAD unchanged.
func fixtureGitSnapshot(t *testing.T, root string) []byte {
	t.Helper()
	head, err := testutil.RunGit(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	files, err := testutil.RunGit(root, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	out.WriteString(head)
	for _, name := range strings.Split(strings.TrimSuffix(files, "\x00"), "\x00") {
		contents, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&out, "%s:%d:", name, len(contents))
		out.Write(contents)
	}
	return out.Bytes()
}

func TestM0318_AC3_NegativePromoteUsesDeclaredTarget(t *testing.T) {
	t.Parallel()
	for _, tc := range enumerateIllegalCases(t) {
		if tc.rule.Verb != "promote" {
			continue
		}
		args := buildIllegalVerbArgs(t, tc, "M-0001/AC-1", spec.EvalContext{})
		if args[len(args)-1] != tc.rule.ToState {
			t.Errorf("%s: arguments %v do not request declared target %q", tc.name, args, tc.rule.ToState)
		}
	}
}
