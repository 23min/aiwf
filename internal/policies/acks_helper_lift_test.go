package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// G-0239 sabotage-fixture tests for the WalkAcknowledgedSHAEntities
// extension of PolicyAcksHelperLift. The live-tree runPolicy test
// (TestPolicy_AcksHelperLift) only proves the happy path; these prove
// each new violation class actually FIRES on the regression it names —
// which is the whole point of the policy per G-0239 ("framework
// correctness must not depend on the LLM's behavior").

// acksTreeOpts configures the synthetic tree buildSyntheticAcksTree
// writes. The SHA-walker (WalkAcknowledgedSHAs) is always declared and
// called once so its own classes are not the thing under test; the
// entities walker is varied per case. Other SHA-walker classes (the
// four-consumer wiring) will fire on these minimal trees; the tests
// filter to G-0239-tagged violations via g0239Violations.
type acksTreeOpts struct {
	declareEntitiesWalker bool // acks.go declares WalkAcknowledgedSHAEntities
	gatherCalls           int  // # of check.WalkAcknowledgedSHAEntities calls in the cli gather file
	internalRecompute     bool // a non-acks internal/check rule file calls the walker (bare)
}

func buildSyntheticAcksTree(t *testing.T, opts acksTreeOpts) string {
	t.Helper()
	root := t.TempDir()

	checkDir := filepath.Join(root, "internal", "check")
	if err := os.MkdirAll(checkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	acks := "package check\n\nfunc WalkAcknowledgedSHAs(root string) map[string]bool { return nil }\n"
	if opts.declareEntitiesWalker {
		acks += "\nfunc WalkAcknowledgedSHAEntities(root string) map[string]map[string]bool { return nil }\n"
	}
	if err := os.WriteFile(filepath.Join(checkDir, "acks.go"), []byte(acks), 0o644); err != nil {
		t.Fatal(err)
	}
	if opts.internalRecompute {
		// Bare-identifier (same-package) call — the rule-internal
		// recompute shape E3 forbids.
		rule := "package check\n\nfunc someRule(root string) {\n\t_ = WalkAcknowledgedSHAEntities(root)\n}\n"
		if err := os.WriteFile(filepath.Join(checkDir, "somerule.go"), []byte(rule), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cliDir := filepath.Join(root, "internal", "cli", "check")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// `check.WalkAcknowledged...` parses as a SelectorExpr with X=Ident
	// "check"; ParseFile does not resolve imports, so no import line is
	// needed for the policy's pkg.Name == "check" match.
	cli := "package check\n\nfunc Run(root string) {\n\t_ = check.WalkAcknowledgedSHAs(root)\n" +
		strings.Repeat("\t_ = check.WalkAcknowledgedSHAEntities(root)\n", opts.gatherCalls) +
		"}\n"
	if err := os.WriteFile(filepath.Join(cliDir, "check.go"), []byte(cli), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// g0239Violations filters policy output to the entities-walker
// violations (their Detail carries the G-0239 tag, unambiguously
// distinct from the SHA-walker classes' M-0159/M-0160 messages).
func g0239Violations(vs []Violation) []Violation {
	var out []Violation
	for _, v := range vs {
		if strings.Contains(v.Detail, "G-0239") {
			out = append(out, v)
		}
	}
	return out
}

func TestPolicyAcksHelperLift_EntitiesWalker_HappyPath(t *testing.T) {
	t.Parallel()
	root := buildSyntheticAcksTree(t, acksTreeOpts{declareEntitiesWalker: true, gatherCalls: 1})
	vs, err := PolicyAcksHelperLift(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if g := g0239Violations(vs); len(g) != 0 {
		t.Errorf("expected no G-0239 violations on a clean entities-walker tree; got %+v", g)
	}
}

func TestPolicyAcksHelperLift_EntitiesWalker_FiresOnMissingDeclaration(t *testing.T) {
	t.Parallel()
	root := buildSyntheticAcksTree(t, acksTreeOpts{declareEntitiesWalker: false, gatherCalls: 1})
	vs, err := PolicyAcksHelperLift(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	g := g0239Violations(vs)
	found := false
	for _, v := range g {
		if v.File == "internal/check/acks.go" && strings.Contains(v.Detail, "declare WalkAcknowledgedSHAEntities") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a G-0239 acks.go declaration violation; got %+v", g)
	}
}

func TestPolicyAcksHelperLift_EntitiesWalker_FiresOnZeroGatherCalls(t *testing.T) {
	t.Parallel()
	root := buildSyntheticAcksTree(t, acksTreeOpts{declareEntitiesWalker: true, gatherCalls: 0})
	vs, err := PolicyAcksHelperLift(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	g := g0239Violations(vs)
	found := false
	for _, v := range g {
		if strings.Contains(v.Detail, "found zero call sites") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a G-0239 zero-gather-call violation; got %+v", g)
	}
}

func TestPolicyAcksHelperLift_EntitiesWalker_FiresOnMultipleGatherCalls(t *testing.T) {
	t.Parallel()
	root := buildSyntheticAcksTree(t, acksTreeOpts{declareEntitiesWalker: true, gatherCalls: 2})
	vs, err := PolicyAcksHelperLift(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	g := g0239Violations(vs)
	count := 0
	for _, v := range g {
		if strings.Contains(v.Detail, "one of multiple call sites") {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 G-0239 multiple-call-site violations (one per call); got %d in %+v", count, g)
	}
}

func TestPolicyAcksHelperLift_EntitiesWalker_FiresOnInternalRecompute(t *testing.T) {
	t.Parallel()
	root := buildSyntheticAcksTree(t, acksTreeOpts{declareEntitiesWalker: true, gatherCalls: 1, internalRecompute: true})
	vs, err := PolicyAcksHelperLift(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	g := g0239Violations(vs)
	found := false
	for _, v := range g {
		if v.File == "internal/check/somerule.go" && strings.Contains(v.Detail, "rule-internal recompute") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a G-0239 rule-internal-recompute violation at somerule.go; got %+v", g)
	}
}

// Class-4f fixtures. The consumer set is stated twice — ackedSHAsConsumers
// here and WalkAcknowledgedSHAs' doc comment in internal/check/acks.go —
// and 4f is what keeps them from drifting apart. Direction 1 catches a
// policed consumer the doc forgets to name; direction 2 catches a rule
// that starts reading the map without being policed at all, which is the
// direction that makes the set self-maintaining.

// class4fViolations filters to the 4f-tagged output so the minimal
// synthetic trees' other wiring violations don't mask the assertion.
func class4fViolations(vs []Violation) []Violation {
	var out []Violation
	for _, v := range vs {
		if strings.Contains(v.Detail, "class 4f") {
			out = append(out, v)
		}
	}
	return out
}

// buildClass4fTree writes a tree whose acks.go declares the walker with a
// doc comment naming every consumer in docNames, plus one internal/check
// rule file per entry in extraReaders declaring a function of that name
// that indexes ackedSHAs.
func buildClass4fTree(t *testing.T, docNames, extraReaders []string) string {
	t.Helper()
	root := t.TempDir()

	checkDir := filepath.Join(root, "internal", "check")
	if err := os.MkdirAll(checkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := "// WalkAcknowledgedSHAs walks HEAD for acknowledgments.\n//\n// Consumers: " +
		strings.Join(docNames, ", ") + ".\n"
	acks := "package check\n\n" + doc +
		"func WalkAcknowledgedSHAs(root string) map[string]bool { return nil }\n\n" +
		"func WalkAcknowledgedSHAEntities(root string) map[string]map[string]bool { return nil }\n"
	if err := os.WriteFile(filepath.Join(checkDir, "acks.go"), []byte(acks), 0o644); err != nil {
		t.Fatal(err)
	}
	for i, name := range extraReaders {
		src := "package check\n\nfunc " + name +
			"(ackedSHAs map[string]bool, sha string) bool { return ackedSHAs[sha] }\n"
		fname := filepath.Join(checkDir, "reader"+string(rune('a'+i))+".go")
		if err := os.WriteFile(fname, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cliDir := filepath.Join(root, "internal", "cli", "check")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cli := "package check\n\nfunc Run(root string) {\n\t_ = check.WalkAcknowledgedSHAs(root)\n" +
		"\t_ = check.WalkAcknowledgedSHAEntities(root)\n}\n"
	if err := os.WriteFile(filepath.Join(cliDir, "check.go"), []byte(cli), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestPolicyAcksHelperLift_Class4f_QuietWhenDocNamesEveryConsumer(t *testing.T) {
	t.Parallel()
	all := append(append([]string{}, ackedSHAsConsumers...), ackedSHAsBodyConsumers...)
	root := buildClass4fTree(t, all, nil)
	vs, err := PolicyAcksHelperLift(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if c := class4fViolations(vs); len(c) != 0 {
		t.Errorf("class 4f fired on a tree whose doc names every consumer and whose only ackedSHAs readers are policed; got %+v", c)
	}
}

func TestPolicyAcksHelperLift_Class4f_FiresWhenDocOmitsAConsumer(t *testing.T) {
	t.Parallel()
	// Name every consumer except RunPromoteOnWrongBranch — the shape of a
	// consumer added to the policy but never written into the prose.
	var docNames []string
	for _, n := range append(append([]string{}, ackedSHAsConsumers...), ackedSHAsBodyConsumers...) {
		if n != "RunPromoteOnWrongBranch" {
			docNames = append(docNames, n)
		}
	}
	root := buildClass4fTree(t, docNames, nil)
	vs, err := PolicyAcksHelperLift(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	found := false
	for _, v := range class4fViolations(vs) {
		if v.File == "internal/check/acks.go" && strings.Contains(v.Detail, "RunPromoteOnWrongBranch") {
			found = true
		}
	}
	if !found {
		t.Errorf("class 4f did not fire for a consumer the doc comment omits; got %+v", class4fViolations(vs))
	}
}

func TestPolicyAcksHelperLift_Class4f_FiresOnAnUnpolicedAckedSHAsReader(t *testing.T) {
	t.Parallel()
	// A new rule reads ackedSHAs but appears in neither consumer list —
	// exactly how RunOrphanedAICommits and RunPromoteOnWrongBranch sat
	// unpoliced behind a doc comment describing a smaller set.
	all := append(append([]string{}, ackedSHAsConsumers...), ackedSHAsBodyConsumers...)
	root := buildClass4fTree(t, all, []string{"RunSomeBrandNewRule"})
	vs, err := PolicyAcksHelperLift(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	found := false
	for _, v := range class4fViolations(vs) {
		if strings.Contains(v.Detail, "RunSomeBrandNewRule") && strings.Contains(v.Detail, "ackedSHAsBodyConsumers") {
			found = true
		}
	}
	if !found {
		t.Errorf("class 4f did not fire for an unpoliced ackedSHAs reader; got %+v", class4fViolations(vs))
	}
}
