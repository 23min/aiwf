package policies

import (
	"strings"
	"testing"
)

// TestPolicyNoTimeNowInCore_FiresForEachSelector proves a core-tier
// package is flagged for time.Now, time.Since, and time.Until, with the
// finding naming the package, its tier, the selector, and the call line.
func TestPolicyNoTimeNowInCore_FiresForEachSelector(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src := `package verb

func f(x, y interface{}) {
	_ = time.Now()
	_ = time.Since(x)
	_ = time.Until(y)
}
`
	writeSrcFixture(t, root, "internal/verb/clock.go", src) // verb = tier 2 (core)

	violations, err := PolicyNoTimeNowInCore(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) != 3 {
		t.Fatalf("expected 3 violations (Now/Since/Until); got %+v", violations)
	}
	wantSel := map[string]int{"time.Now": 4, "time.Since": 5, "time.Until": 6}
	for _, v := range violations {
		if v.Policy != "no-time-now-in-core" || v.File != "internal/verb/clock.go" {
			t.Errorf("unexpected policy/file: %+v", v)
		}
		if !strings.Contains(v.Detail, "internal/verb (core tier 2) calls time.") {
			t.Errorf("detail missing package/tier: %q", v.Detail)
		}
		matched := false
		for sel, line := range wantSel {
			if strings.Contains(v.Detail, sel) {
				matched = true
				if v.Line != line {
					t.Errorf("%s reported on line %d, want %d", sel, v.Line, line)
				}
			}
		}
		if !matched {
			t.Errorf("violation names no expected selector: %q", v.Detail)
		}
	}
}

// TestPolicyNoTimeNowInCore_EdgeNotScanned proves an edge package
// (tier <= 1) may read the wall clock without firing — that is where the
// clock is legitimately acquired.
func TestPolicyNoTimeNowInCore_EdgeNotScanned(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src := "package add\n\nfunc f() { _ = time.Now() }\n"
	writeSrcFixture(t, root, "internal/cli/add/x.go", src) // cli/* = tier 1 (edge)

	violations, err := PolicyNoTimeNowInCore(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("edge package should not be scanned; got %+v", violations)
	}
}

// TestPolicyNoTimeNowInCore_ExemptCoreSkipped proves an allowlisted core
// package (operational/perf time) does not fire.
func TestPolicyNoTimeNowInCore_ExemptCoreSkipped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src := "package htmlrender\n\nfunc f(s interface{}) { _ = time.Now(); _ = time.Since(s) }\n"
	writeSrcFixture(t, root, "internal/htmlrender/x.go", src) // exempt

	violations, err := PolicyNoTimeNowInCore(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("exempt core package should not fire; got %+v", violations)
	}
}

// TestPolicyNoTimeNowInCore_UntieredSkipped proves a package with no
// layering tier is not scanned (the layering policy handles untiered
// packages; this one stays silent rather than double-reporting).
func TestPolicyNoTimeNowInCore_UntieredSkipped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src := "package newpkg\n\nfunc f() { _ = time.Now() }\n"
	writeSrcFixture(t, root, "internal/newpkg/x.go", src)

	violations, err := PolicyNoTimeNowInCore(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("untiered package should not be scanned; got %+v", violations)
	}
}

// TestPolicyNoTimeNowInCore_NonMatchingCallsIgnored proves the selector
// match is precise: time.Parse/time.Sleep (not clock reads), clock.Now
// (not the time package), a bare call, and a selector-of-selector all
// pass through cleanly even inside a core package.
func TestPolicyNoTimeNowInCore_NonMatchingCallsIgnored(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src := `package check

func f(a, b, d interface{}) {
	_ = time.Parse(a, b)
	_ = time.Sleep(d)
	_ = clock.Now()
	bare()
	_ = a.b.Now()
}
`
	writeSrcFixture(t, root, "internal/check/x.go", src) // check = tier 4 (core)

	violations, err := PolicyNoTimeNowInCore(root)
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("non-clock-read calls should be ignored; got %+v", violations)
	}
}

// TestPolicyNoTimeNowInCore_SkipsUnparseableFile proves a known-core
// package whose file does not parse is skipped, not errored on.
func TestPolicyNoTimeNowInCore_SkipsUnparseableFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeSrcFixture(t, root, "internal/verb/broken.go", "packag verb\n")

	violations, err := PolicyNoTimeNowInCore(root)
	if err != nil {
		t.Fatalf("policy errored on an unparseable file: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("unparseable file should be skipped; got %+v", violations)
	}
}
