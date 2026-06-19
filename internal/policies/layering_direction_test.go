package policies

import (
	"strings"
	"testing"
)

// imp renders a module-qualified import line for a fixture.
func imp(pkg string) string {
	return "\t\"github.com/23min/aiwf/" + pkg + "\"\n"
}

// TestPolicyLayeringDirection_FlagsUpwardAllowsLateral is the core
// fixture: a tier-6 package (entity) that imports upward (verb, tier 2)
// fires; the same file's downward import (codes, tier 7), a sideways
// import (entity-tier sibling), and a non-module import (fmt) do not.
func TestPolicyLayeringDirection_FlagsUpwardAllowsLateral(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// entity (tier 6) importing verb (tier 2) is upward.
	src := "package entity\n\nimport (\n\t\"fmt\"\n" +
		imp("internal/verb") + // upward 6 -> 2: FIRES
		imp("internal/codes") + // downward 6 -> 7: ok
		imp("internal/gitops") + // sideways 6 -> 6: ok
		")\n\nvar _ = fmt.Sprint\n"
	writeSrcFixture(t, root, "internal/entity/bad.go", src)

	violations, err := PolicyLayeringDirection(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly one upward violation; got %+v", violations)
	}
	v := violations[0]
	if v.Policy != "layering-direction" || v.File != "internal/entity/bad.go" {
		t.Errorf("unexpected policy/file: %+v", v)
	}
	if !strings.Contains(v.Detail, "internal/entity (tier 6) imports internal/verb (tier 2)") {
		t.Errorf("detail does not name the upward edge with tiers: %q", v.Detail)
	}
}

// TestPolicyLayeringDirection_AllowlistedSourceSkipped proves a test-only
// allowlisted package may import upward without firing (cellcoverage,
// which legitimately imports verb/cliutil for fixtures).
func TestPolicyLayeringDirection_AllowlistedSourceSkipped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src := "package cellcoverage\n\nimport (\n" + imp("internal/verb") + imp("internal/cli/cliutil") + ")\n"
	writeSrcFixture(t, root, "internal/cellcoverage/cov.go", src)

	violations, err := PolicyLayeringDirection(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("allowlisted source should not fire; got %+v", violations)
	}
}

// TestPolicyLayeringDirection_AllowlistedTargetSkipped proves an import
// of an allowlisted (test-only) package is not flagged as untiered.
func TestPolicyLayeringDirection_AllowlistedTargetSkipped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src := "package entity\n\nimport (\n" + imp("internal/testsupport") + ")\n"
	writeSrcFixture(t, root, "internal/entity/uses_testsupport.go", src)

	violations, err := PolicyLayeringDirection(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("allowlisted target should not fire; got %+v", violations)
	}
}

// TestPolicyLayeringDirection_UntieredSourceDeduped proves a package with
// no tier assignment is flagged once, not once per file.
func TestPolicyLayeringDirection_UntieredSourceDeduped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeSrcFixture(t, root, "internal/newpkg/a.go", "package newpkg\n")
	writeSrcFixture(t, root, "internal/newpkg/b.go", "package newpkg\n")

	violations, err := PolicyLayeringDirection(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly one (deduped) untiered-source finding; got %+v", violations)
	}
	if !strings.Contains(violations[0].Detail, "internal/newpkg has no layering tier") {
		t.Errorf("unexpected detail: %q", violations[0].Detail)
	}
}

// TestPolicyLayeringDirection_UntieredTargetDeduped proves importing an
// internal package with no tier is flagged once per (source, target).
func TestPolicyLayeringDirection_UntieredTargetDeduped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src := "package verb\n\nimport (\n" + imp("internal/newtarget") + ")\n"
	writeSrcFixture(t, root, "internal/verb/a.go", src)
	writeSrcFixture(t, root, "internal/verb/b.go", src)

	violations, err := PolicyLayeringDirection(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly one (deduped) untiered-target finding; got %+v", violations)
	}
	if !strings.Contains(violations[0].Detail, "import of untiered package internal/newtarget") {
		t.Errorf("unexpected detail: %q", violations[0].Detail)
	}
}

// TestPolicyLayeringDirection_SkipsUnparseableFile proves a known-tier
// package whose file does not parse is skipped, not errored on.
func TestPolicyLayeringDirection_SkipsUnparseableFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// Malformed package clause forces a parse error even in ImportsOnly mode.
	writeSrcFixture(t, root, "internal/entity/broken.go", "packag entity\n")

	violations, err := PolicyLayeringDirection(root)
	if err != nil {
		t.Fatalf("policy errored on an unparseable file: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("unparseable file should be skipped silently; got %+v", violations)
	}
}

// TestLayerTier_PrefixBandsAndUnknown covers the CLI/workflows prefix
// branches and the unknown-package return directly.
func TestLayerTier_PrefixBandsAndUnknown(t *testing.T) {
	t.Parallel()
	cases := []struct {
		pkg   string
		tier  int
		known bool
	}{
		{"internal/cli", 1, true},
		{"internal/cli/cliutil", 1, true},
		{"internal/workflows", 3, true},
		{"internal/workflows/spec/branch", 3, true},
		{"internal/nope", 0, false},
	}
	for _, c := range cases {
		t.Run(c.pkg, func(t *testing.T) {
			t.Parallel()
			got, known := layerTier(c.pkg)
			if got != c.tier || known != c.known {
				t.Errorf("layerTier(%q) = (%d,%v), want (%d,%v)", c.pkg, got, known, c.tier, c.known)
			}
		})
	}
}
