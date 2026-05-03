package version

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		in       string
		wantVer  string
		wantTags bool
	}{
		{"v0.1.0", "v0.1.0", true},
		{"v0.2.3", "v0.2.3", true},
		{"v1.0.0", "v1.0.0", true},
		{"v0.1.0-rc1", "v0.1.0-rc1", true},
		{"v0.1.0+build.5", "v0.1.0+build.5", true},

		// pseudo-versions: tagged regex matches the v0.x.y prefix
		// but the timestamp+sha suffix disqualifies them.
		{"v0.0.0-20260503120000-abcdef123456", "v0.0.0-20260503120000-abcdef123456", false},
		{"v0.1.0-pre.0.20060102150405-abcdef123456", "v0.1.0-pre.0.20060102150405-abcdef123456", false},

		// devel and empty normalize to DevelVersion.
		{"(devel)", DevelVersion, false},
		{"", DevelVersion, false},

		// not semver-shaped at all.
		{"0.1.0", "0.1.0", false},
		{"v0.1", "v0.1", false},
		{"main", "main", false},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := Parse(tc.in)
			if got.Version != tc.wantVer {
				t.Errorf("Version = %q, want %q", got.Version, tc.wantVer)
			}
			if got.Tagged != tc.wantTags {
				t.Errorf("Tagged = %v, want %v", got.Tagged, tc.wantTags)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want Skew
	}{
		{"equal patch", "v0.1.0", "v0.1.0", SkewEqual},
		{"ahead patch", "v0.1.1", "v0.1.0", SkewAhead},
		{"behind patch", "v0.1.0", "v0.1.1", SkewBehind},
		{"ahead minor", "v0.2.0", "v0.1.9", SkewAhead},
		{"behind minor", "v0.1.9", "v0.2.0", SkewBehind},
		{"ahead major", "v1.0.0", "v0.99.99", SkewAhead},
		{"behind major", "v0.99.99", "v1.0.0", SkewBehind},

		// devel and pseudo on either side → Unknown.
		{"a devel", DevelVersion, "v0.1.0", SkewUnknown},
		{"b devel", "v0.1.0", DevelVersion, SkewUnknown},
		{"a pseudo", "v0.0.0-20260503120000-abcdef123456", "v0.1.0", SkewUnknown},
		{"b pseudo", "v0.1.0", "v0.0.0-20260503120000-abcdef123456", SkewUnknown},

		// pre-release / build suffix on either side → Unknown
		// (deliberate narrowing — see package doc).
		{"a rc", "v0.1.0-rc1", "v0.1.0", SkewUnknown},
		{"b rc", "v0.1.0", "v0.1.0-rc1", SkewUnknown},
		{"a build", "v0.1.0+build.5", "v0.1.0", SkewUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Compare(Parse(tc.a), Parse(tc.b))
			if got != tc.want {
				t.Errorf("Compare(%q, %q) = %s, want %s", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestSkewString(t *testing.T) {
	cases := []struct {
		s    Skew
		want string
	}{
		{SkewEqual, "equal"},
		{SkewAhead, "ahead"},
		{SkewBehind, "behind"},
		{SkewUnknown, "unknown"},
		{Skew(99), "unknown"}, // out-of-range falls through to default
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.s.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseTriple(t *testing.T) {
	// Direct white-box test of the parseTriple helper. Compare only
	// calls it on values that already passed isTagged, so the
	// defensive paths (wrong segment count, non-numeric segments)
	// are unreachable through the public API. This pins the helper's
	// contract independently so future refactors don't drift.
	cases := []struct {
		in    string
		ok    bool
		major int
	}{
		{"v0.1.0", true, 0},
		{"v1.2.3", true, 1},
		{"v0.1.0-rc1", false, 0},   // pre-release suffix → not pure triple
		{"v0.1.0+build", false, 0}, // build suffix → not pure triple
		{"v0.1", false, 0},         // wrong segment count (2)
		{"v0.1.0.4", false, 0},     // wrong segment count (4)
		{"vfoo.1.0", false, 0},     // non-numeric major
		{"v0.bar.0", false, 0},     // non-numeric minor
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, ok := parseTriple(tc.in)
			if ok != tc.ok {
				t.Fatalf("parseTriple(%q) ok = %v, want %v", tc.in, ok, tc.ok)
			}
			if ok && got[0] != tc.major {
				t.Errorf("parseTriple(%q) major = %d, want %d", tc.in, got[0], tc.major)
			}
		})
	}
}

func TestCurrent_DevelInTestBinary(t *testing.T) {
	// `go test` builds the test binary as a working-tree build, so
	// runtime/debug.ReadBuildInfo reports Main.Version == "" or
	// "(devel)" — either way Current() returns DevelVersion with
	// Tagged=false. This pins the contract that the test binary
	// exercises the devel path of Current().
	got := Current()
	if got.Version != DevelVersion {
		t.Errorf("Current().Version = %q, want %q (running under go test)", got.Version, DevelVersion)
	}
	if got.Tagged {
		t.Errorf("Current().Tagged = true, want false (running under go test)")
	}
}
