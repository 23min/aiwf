package verb

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// configClaimRepo builds a real repo carrying a committed aiwf.yaml with
// an areas block and one tagged epic, and returns the tree plus the parsed
// doc. The three verbs whose claims rest on aiwf.yaml need a repo rather
// than a hand-built tree, because their guard compares that file against
// HEAD.
func configClaimRepo(t *testing.T) (*tree.Tree, *aiwfyaml.Doc) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, config.FileName), []byte(areaAiwfYAML), 0o600); err != nil {
		t.Fatalf("writing %s: %v", config.FileName, err)
	}
	ents := []*entity.Entity{
		{
			ID: "E-0001", Kind: entity.KindEpic, Title: "Platform", Status: "proposed", Area: "platform",
			Path: filepath.ToSlash(filepath.Join("work", "epics", "E-0001-platform", "epic.md")),
		},
		{
			ID: "C-0001", Kind: entity.KindContract, Title: "Envelope", Status: "proposed",
			Path: filepath.ToSlash(filepath.Join("work", "contracts", "C-0001-envelope", "contract.md")),
		},
	}
	for _, e := range ents {
		content, serr := entity.Serialize(e, []byte("\n## Goal\n\nFixture prose.\n"))
		if serr != nil {
			t.Fatalf("serialize %s: %v", e.ID, serr)
		}
		full := filepath.Join(root, e.Path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, content, 0o600); err != nil {
			t.Fatalf("writing %s: %v", e.ID, err)
		}
	}
	if err := gitops.Add(ctx, root, "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := gitops.Commit(ctx, root, "fixture: seed the config repo", "", nil); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	doc, _, err := aiwfyaml.ReadBytes([]byte(areaAiwfYAML))
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	return &tree.Tree{Root: root, Entities: ents}, doc
}

// settledContracts returns a contracts block already carrying exactly what
// the two contract verbs under test ask for, so each reaches its
// same-state NoOp rather than a plan — which is the site AC-2 scopes.
func settledContracts() *aiwfyaml.Contracts {
	return &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"fake": {Command: "true"}},
		Entries:    []aiwfyaml.Entry{{ID: "C-0001", Validator: "fake", Schema: "s", Fixtures: "f"}},
	}
}

// dirtyConfig appends a comment to the repo's aiwf.yaml without
// committing, the shape of an operator part-way through a hand-edit.
func dirtyConfig(t *testing.T, root string) string {
	t.Helper()
	p := filepath.Join(root, config.FileName)
	raw, err := os.ReadFile(p) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading %s: %v", config.FileName, err)
	}
	if err := os.WriteFile(p, append(raw, []byte("\n# UNCOMMITTED EDIT\n")...), 0o600); err != nil {
		t.Fatalf("writing %s: %v", config.FileName, err)
	}
	return config.FileName
}

// TestConfigScopedVerbs_DivergentConfig_Refuse pins the second of the four
// claim scopes. These three verbs splice a working-copy aiwf.yaml and
// decide "unchanged" from what they read there, so their claim rests on
// that file rather than on any entity — scoping them to a target entity
// would leave the guard inert at exactly the sites that need it.
func TestConfigScopedVerbs_DivergentConfig_Refuse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		call func(tr *tree.Tree, doc *aiwfyaml.Doc) (*Result, error)
	}{
		{
			name: "rename-area",
			call: func(tr *tree.Tree, doc *aiwfyaml.Doc) (*Result, error) {
				return RenameArea(context.Background(), tr, doc,
					[]config.Member{{Name: "platform"}, {Name: "billing"}},
					"platform", "platform", "human/test")
			},
		},
		{
			name: "contract bind",
			call: func(tr *tree.Tree, doc *aiwfyaml.Doc) (*Result, error) {
				return ContractBind(context.Background(), tr, doc, settledContracts(),
					"C-0001", "human/test", tr.Root, ContractBindOptions{
						Validator: "fake", Schema: "s", Fixtures: "f",
					})
			},
		},
		{
			name: "recipe install",
			call: func(tr *tree.Tree, doc *aiwfyaml.Doc) (*Result, error) {
				return RecipeInstall(context.Background(), tr, doc, settledContracts(),
					"fake", aiwfyaml.Validator{Command: "true"}, "human/test", tr.Root,
					RecipeInstallOptions{})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tr, doc := configClaimRepo(t)
			path := dirtyConfig(t, tr.Root)

			res, err := tc.call(tr, doc)
			if err == nil {
				t.Fatalf("verb returned res=%+v over a divergent aiwf.yaml, want a refusal", res)
			}
			var claimErr *ClaimDivergenceError
			if !errors.As(err, &claimErr) {
				t.Fatalf("error is not a *ClaimDivergenceError: %v", err)
			}
			var named bool
			for _, p := range claimErr.Paths() {
				if p == path {
					named = true
				}
			}
			if !named {
				t.Errorf("refusal does not name %q; names %v", path, claimErr.Paths())
			}
		})
	}
}

// TestRenameArea_CleanConfig_StillConverges is the negative control: the
// guard must not turn every same-name rename into an error.
func TestRenameArea_CleanConfig_StillConverges(t *testing.T) {
	t.Parallel()
	tr, doc := configClaimRepo(t)

	res, err := RenameArea(context.Background(), tr, doc,
		[]config.Member{{Name: "platform"}, {Name: "billing"}},
		"platform", "platform", "human/test")
	if err != nil {
		t.Fatalf("RenameArea on a clean tree: %v", err)
	}
	if !res.NoOp {
		t.Errorf("same-name rename on a clean tree did not converge: %+v", res)
	}
}

// TestGuardClaimVariants_DifferOnlyOnAPathAbsentFromHEAD pins the split
// between the two guard variants, which is the whole of their
// difference.
//
// The entity variant refuses a path HEAD does not record, because a
// claim is about an entity and an entity's file can move without any
// verb — after a plain `mv` the working copy is the only thing saying
// where the id lives, which is where the record most needs consulting.
//
// The config variant exempts it, because `aiwf init` leaves aiwf.yaml
// uncommitted by design and every verb that rewrites it would otherwise
// be unreachable until someone committed it. aiwf.yaml is not an entity
// and cannot move out from under an id, so the entity reasoning does not
// reach it.
func TestGuardClaimVariants_DifferOnlyOnAPathAbsentFromHEAD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "committed.md"), []byte("recorded\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, "committed.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(ctx, root, "seed", "", nil); err != nil {
		t.Fatal(err)
	}
	// Present on disk, absent from the record.
	if err := os.WriteFile(filepath.Join(root, "unrecorded.md"), []byte("never committed\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := guardClaim(ctx, root, "subject", "unrecorded.md"); err == nil {
		t.Error("the entity variant accepted a path the record does not hold")
	} else {
		var claimErr *ClaimDivergenceError
		if !errors.As(err, &claimErr) {
			t.Errorf("entity variant returned %v, want a *ClaimDivergenceError", err)
		}
	}

	if err := guardClaimConfig(ctx, root, "subject", "unrecorded.md"); err != nil {
		t.Errorf("the config variant refused an uncommitted aiwf.yaml: %v", err)
	}

	// Neither variant exempts a path whose recorded content disagrees.
	if err := os.WriteFile(filepath.Join(root, "committed.md"), []byte("hand-edited\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, guard := range map[string]func(context.Context, string, string, ...string) error{
		"entity": guardClaim,
		"config": guardClaimConfig,
	} {
		if err := guard(ctx, root, "subject", "committed.md"); err == nil {
			t.Errorf("%s variant accepted a hand-edited tracked path", name)
		}
	}
}

// TestClaimDivergenceError_RemedyMatchesTheKind pins that each blocking
// kind gets a diagnosis that fits it. The entity guard refuses a path
// with no version at HEAD, so "uncommitted changes" would be the wrong
// words for the one case it now refuses most often — nothing is
// uncommitted, the file is somewhere the record does not know about, and
// re-running changes nothing.
func TestClaimDivergenceError_RemedyMatchesTheKind(t *testing.T) {
	t.Parallel()

	unrecorded := (&ClaimDivergenceError{
		Subject:  "G-0001",
		Diverged: []gitops.Divergence{{Path: "work/gaps/G-0001-moved.md", Kind: gitops.DivergenceAbsentFromHEAD}},
	}).Error()
	if strings.Contains(unrecorded, "uncommitted changes at") {
		t.Errorf("a path with no version at HEAD is not an uncommitted change:\n%s", unrecorded)
	}
	// `aiwf rename` carries this same guard and refuses identically, so
	// offering it sends the operator round a loop.
	if strings.Contains(unrecorded, "aiwf rename") {
		t.Errorf("the advice offers a verb this same guard refuses:\n%s", unrecorded)
	}
	for _, want := range []string{"holds no version", "same id lands at two", "never been committed"} {
		if !strings.Contains(unrecorded, want) {
			t.Errorf("diagnosis does not mention %q:\n%s", want, unrecorded)
		}
	}

	edited := (&ClaimDivergenceError{
		Subject:  "G-0001",
		Diverged: []gitops.Divergence{{Path: "work/gaps/G-0001-x.md", Kind: gitops.DivergenceModified}},
	}).Error()
	if !strings.Contains(edited, "uncommitted changes at") {
		t.Errorf("an edited path should still read as an uncommitted change:\n%s", edited)
	}
	if strings.Contains(edited, "holds no version") {
		t.Errorf("an edited path picked up the unrecorded diagnosis:\n%s", edited)
	}
}

// TestClaimDivergenceError_MixedKinds pins the heading when a refusal
// carries more than one kind at once — reachable at `promote
// --superseded-by`, whose claim spans the target and the superseding
// entity.
//
// The heading is chosen by whether *every* diverging path lacks a
// record. A mixed refusal has at least one genuinely edited path, so
// "uncommitted changes" is the accurate opening, and the unrecorded path
// still gets its own paragraph. Both mutants of that condition produce a
// message that contradicts one of its own halves.
func TestClaimDivergenceError_MixedKinds(t *testing.T) {
	t.Parallel()
	msg := (&ClaimDivergenceError{
		Subject: "ADR-0001",
		Diverged: []gitops.Divergence{
			{Path: "docs/adr/ADR-0001-a.md", Kind: gitops.DivergenceModified},
			{Path: "docs/adr/ADR-0002-b.md", Kind: gitops.DivergenceAbsentFromHEAD},
		},
	}).Error()

	if !strings.Contains(msg, "uncommitted changes at") {
		t.Errorf("a refusal carrying an edited path should open with it:\n%s", msg)
	}
	if strings.Contains(msg, "the record holds no version of") {
		t.Errorf("the no-record heading claims every path lacks one; here only some do:\n%s", msg)
	}
	// Each kind still gets the paragraph that fits it.
	if !strings.Contains(msg, "docs/adr/ADR-0002-b.md is at a path the record does not hold") {
		t.Errorf("the unrecorded path lost its own diagnosis:\n%s", msg)
	}
	if !strings.Contains(msg, "aiwf edit-body") {
		t.Errorf("the edited path lost its remedy:\n%s", msg)
	}
}
