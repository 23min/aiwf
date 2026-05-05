package version

import (
	"strings"
	"testing"
)

// FuzzParse drives Parse with arbitrary version strings and checks
// invariants the upgrade-flow consumers rely on. Filed under G44 item 1.
//
// Cross-package regressions caught by these properties are exactly the
// G29 class (pseudo-version regex example-driven, missed two of three
// canonical forms plus the `+dirty` suffix); having Parse fuzzed here
// makes the spec-driven enumeration mechanical.
func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		"",
		"devel",
		"v0.0.0",
		"v0.1.0",
		"v1.2.3",
		"v0.0.0-20251201123456-abcdef012345",
		"v0.1.1-0.20251201123456-abcdef012345",
		"v0.1.0-pre.0.20251201123456-abcdef012345",
		"v1.2.3+dirty",
		"v1.2.3-rc.1",
		"v1.2.3-rc.1+dirty",
		"vMAJOR.MINOR.PATCH",
		"not a version",
		"v0.0.0-yyyymmddhhmmss-abcdefabcdef",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, v string) {
		info := Parse(v)

		// Property 1: any string ending in "+dirty" must report Tagged=false.
		// This is the load-bearing invariant the +dirty suffix exists for.
		if strings.HasSuffix(v, "+dirty") && info.Tagged {
			t.Fatalf("Parse(%q).Tagged=true; +dirty must mark untagged", v)
		}
		// Property 2: empty string and the literal "devel" sentinel
		// always normalize to DevelVersion with Tagged=false.
		if v == "" || v == DevelVersion {
			if info.Version != DevelVersion {
				t.Fatalf("Parse(%q).Version=%q, want %q", v, info.Version, DevelVersion)
			}
			if info.Tagged {
				t.Fatalf("Parse(%q).Tagged=true, want false", v)
			}
			return
		}
		// Property 3: pseudo-version-shaped suffix (14-digit timestamp
		// + 12-hex SHA) must mark untagged. Re-derived against the
		// production regex so a regression in the regex still allows
		// the assertion to fail on an obvious counterexample.
		if pseudoVersionRE.MatchString(v) && info.Tagged {
			t.Fatalf("Parse(%q).Tagged=true; pseudo-version must mark untagged", v)
		}
		// Property 4: when Tagged is true, the value must match the
		// clean-tagged regex. (The reverse direction is allowed to
		// fail — pseudo-version inputs match taggedSemverRE's prefix
		// but are correctly demoted by the pseudo-version check.)
		if info.Tagged && !taggedSemverRE.MatchString(v) {
			t.Fatalf("Parse(%q).Tagged=true but does not match taggedSemverRE", v)
		}
		// Property 5: the Version field is byte-equal to the input
		// when not the empty/devel case (no trimming, no normalization).
		if info.Version != v {
			t.Fatalf("Parse(%q).Version=%q; expected verbatim", v, info.Version)
		}
	})
}
