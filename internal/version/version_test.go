package version

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	// Inputs sourced from the Go module spec — see
	// https://go.dev/ref/mod#pseudo-versions for the three
	// pseudo-version forms, https://semver.org for the base grammar,
	// and https://go.dev/ref/mod#vcs-stamps for the +dirty VCS-
	// stamping suffix. Per tools/CLAUDE.md "Spec-sourced inputs":
	// the table covers the full enumerated space, not just examples.
	cases := []struct {
		in       string
		wantVer  string
		wantTags bool
	}{
		// Clean semver tags — the happy path.
		{"v0.1.0", "v0.1.0", true},
		{"v0.2.3", "v0.2.3", true},
		{"v1.0.0", "v1.0.0", true},
		{"v0.1.0-rc1", "v0.1.0-rc1", true},         // pre-release suffix per semver §9
		{"v0.1.0+build.5", "v0.1.0+build.5", true}, // build suffix per semver §10

		// Pseudo-version form 1 (no prior tag), per
		// go.dev/ref/mod#pseudo-versions:
		//   v0.0.0-yyyymmddhhmmss-abcdefabcdef
		{"v0.0.0-20260503120000-abcdef123456", "v0.0.0-20260503120000-abcdef123456", false},

		// Pseudo-version form 2 (commits after vX.Y.Z), same source:
		//   vX.Y.(Z+1)-0.yyyymmddhhmmss-abcdefabcdef
		{"v0.1.1-0.20260503120000-abcdef123456", "v0.1.1-0.20260503120000-abcdef123456", false},

		// Pseudo-version form 3 (commits between vX.Y.Z-pre and
		// vX.Y.Z), same source:
		//   vX.Y.Z-pre.0.yyyymmddhhmmss-abcdefabcdef
		{"v0.1.0-pre.0.20060102150405-abcdef123456", "v0.1.0-pre.0.20060102150405-abcdef123456", false},

		// +dirty VCS-stamping suffix (Go's `cmd/go` stamps it on
		// builds with uncommitted changes); never tagged regardless
		// of the base shape.
		{"v0.1.0+dirty", "v0.1.0+dirty", false},
		{"v0.0.0-20260503120000-abcdef123456+dirty", "v0.0.0-20260503120000-abcdef123456+dirty", false},

		// debug.ReadBuildInfo's "no module info" sentinel.
		{"(devel)", DevelVersion, false},
		{"", DevelVersion, false},

		// Not semver-shaped at all (negative cases).
		{"0.1.0", "0.1.0", false}, // missing v prefix
		{"v0.1", "v0.1", false},   // missing patch segment
		{"main", "main", false},   // not a version string
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

func TestModulePath_TestBinary(t *testing.T) {
	// Under `go test`, ModulePath returns the module of the test
	// binary, which is this repo's go.mod path.
	got := ModulePath()
	const want = "github.com/23min/aiwf"
	if got != want {
		t.Errorf("ModulePath() = %q, want %q", got, want)
	}
}

// TestProxyBase enumerates the GOPROXY chain grammar per
// https://go.dev/ref/mod#environment-variables (the canonical spec
// for module-mode environment variables): comma-or-pipe-separated
// list of entries, each being a URL, the literal `direct`, or the
// literal `off`. Per tools/CLAUDE.md "Spec-sourced inputs": the
// cases cover every entry shape and chain-position the spec
// distinguishes, not just examples we've personally hit.
func TestProxyBase(t *testing.T) {
	cases := []struct {
		name      string
		goproxy   string
		setEnv    bool
		wantBase  string
		wantErrIs error
	}{
		{
			name:     "unset uses default",
			setEnv:   false,
			wantBase: DefaultProxyURL,
		},
		{
			name:     "explicit https proxy",
			setEnv:   true,
			goproxy:  "https://proxy.example.com",
			wantBase: "https://proxy.example.com",
		},
		{
			name:     "https proxy with trailing direct",
			setEnv:   true,
			goproxy:  "https://proxy.example.com,direct",
			wantBase: "https://proxy.example.com",
		},
		{
			name:     "direct skipped, https second",
			setEnv:   true,
			goproxy:  "direct,https://proxy.example.com",
			wantBase: "https://proxy.example.com",
		},
		{
			name:     "pipe-separated chain",
			setEnv:   true,
			goproxy:  "https://proxy.example.com|https://backup.example.com",
			wantBase: "https://proxy.example.com",
		},
		{
			name:      "off terminates with error",
			setEnv:    true,
			goproxy:   "off",
			wantErrIs: ErrProxyDisabled,
		},
		{
			name:      "direct only — no http entry",
			setEnv:    true,
			goproxy:   "direct",
			wantErrIs: ErrProxyDisabled,
		},
		{
			name:      "off in chain after direct",
			setEnv:    true,
			goproxy:   "direct,off,https://too-late.example.com",
			wantErrIs: ErrProxyDisabled,
		},
		{
			name:      "malformed entry falls through to disabled",
			setEnv:    true,
			goproxy:   "not-a-url",
			wantErrIs: ErrProxyDisabled,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setEnv {
				t.Setenv("GOPROXY", tc.goproxy)
			} else {
				t.Setenv("GOPROXY", "")
			}
			got, err := proxyBase()
			if tc.wantErrIs != nil {
				if !errors.Is(err, tc.wantErrIs) {
					t.Fatalf("err = %v, want errors.Is %v", err, tc.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tc.wantBase {
				t.Errorf("base = %q, want %q", got, tc.wantBase)
			}
		})
	}
}

func TestLatest_Happy(t *testing.T) {
	const modulePath = "example.com/test/module"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + modulePath + "/@v/list":
			fmt.Fprint(w, "v0.2.0\nv0.1.0\nv0.1.1\n")
		default:
			t.Errorf("unexpected path %q (Latest should resolve from @v/list)", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	t.Setenv("GOPROXY", srv.URL)

	got, err := latestFor(context.Background(), srv.Client(), modulePath)
	if err != nil {
		t.Fatalf("latestFor: %v", err)
	}
	if got.Version != "v0.2.0" {
		t.Errorf("Version = %q, want v0.2.0 (highest tagged in list)", got.Version)
	}
	if !got.Tagged {
		t.Errorf("Tagged = false, want true")
	}
}

// TestLatest_FallsBackToAtLatest covers the no-tags-yet case: the
// /@v/list endpoint returns an empty body (or only non-tagged values
// like a pre-release-only history), and Latest falls through to the
// /@latest endpoint to surface the latest commit's pseudo-version.
func TestLatest_FallsBackToAtLatest(t *testing.T) {
	const modulePath = "example.com/test/module"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + modulePath + "/@v/list":
			fmt.Fprint(w, "") // no tags
		case "/" + modulePath + "/@latest":
			fmt.Fprintln(w, `{"Version":"v0.0.0-20060102150405-abcdef123456"}`)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GOPROXY", srv.URL)

	got, err := latestFor(context.Background(), srv.Client(), modulePath)
	if err != nil {
		t.Fatalf("latestFor: %v", err)
	}
	if got.Version != "v0.0.0-20060102150405-abcdef123456" {
		t.Errorf("Version = %q, want pseudo-version fallback", got.Version)
	}
	if got.Tagged {
		t.Errorf("Tagged = true, want false (pseudo-version)")
	}
}

func TestLatest_ProxyError(t *testing.T) {
	cases := []struct {
		name      string
		listBody  string // empty → 404 on /@v/list, forces fallback to /@latest
		latestFn  http.HandlerFunc
		listFn    http.HandlerFunc // when set, overrides default 200/empty
		wantErrIn string
	}{
		{
			name: "@v/list 500",
			listFn: func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "boom", http.StatusInternalServerError)
			},
			wantErrIn: "returned 500",
		},
		{
			name:     "fallback @latest 404",
			listBody: "",
			latestFn: func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "not found", http.StatusNotFound)
			},
			wantErrIn: "returned 404",
		},
		{
			name:     "fallback @latest malformed JSON",
			listBody: "",
			latestFn: func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprint(w, "not-json")
			},
			wantErrIn: "decoding proxy response",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/@v/list") {
					if tc.listFn != nil {
						tc.listFn(w, r)
						return
					}
					fmt.Fprint(w, tc.listBody)
					return
				}
				if strings.HasSuffix(r.URL.Path, "/@latest") {
					if tc.latestFn != nil {
						tc.latestFn(w, r)
						return
					}
					http.NotFound(w, r)
					return
				}
				http.NotFound(w, r)
			}))
			t.Cleanup(srv.Close)
			t.Setenv("GOPROXY", srv.URL)
			_, err := latestFor(context.Background(), srv.Client(), "example.com/test/module")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErrIn) {
				t.Errorf("err = %q, want substring %q", err.Error(), tc.wantErrIn)
			}
		})
	}
}

func TestLatest_GoproxyOff(t *testing.T) {
	t.Setenv("GOPROXY", "off")
	_, err := latestFor(context.Background(), http.DefaultClient, "example.com/test/module")
	if !errors.Is(err, ErrProxyDisabled) {
		t.Errorf("err = %v, want errors.Is ErrProxyDisabled", err)
	}
}

func TestLatest_EmptyModulePath(t *testing.T) {
	_, err := latestFor(context.Background(), http.DefaultClient, "")
	if err == nil {
		t.Fatal("expected error on empty module path")
	}
	if !strings.Contains(err.Error(), "module path unavailable") {
		t.Errorf("err = %q, want substring 'module path unavailable'", err.Error())
	}
}

func TestLatest_ContextTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until the request context is cancelled.
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GOPROXY", srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := latestFor(ctx, srv.Client(), "example.com/test/module")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestLatest_Wrapper(t *testing.T) {
	// Exercises the public Latest(ctx) wrapper end-to-end against a
	// fake proxy. Latest reads ModulePath() from the test binary's
	// build info; we pre-position the server to respond on that path.
	modulePath := ModulePath()
	if modulePath == "" {
		t.Skip("ModulePath unavailable in this build")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + modulePath + "/@v/list":
			fmt.Fprintln(w, "v0.9.9")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GOPROXY", srv.URL)

	got, err := Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if got.Version != "v0.9.9" {
		t.Errorf("Version = %q, want v0.9.9", got.Version)
	}
}

// TestLatest_RealProxy_ContractTest is G28's durable defense. The
// previous shape of this test asserted only "Version is non-empty,"
// which would have passed even with the v0.1.0 bug (where Latest hit
// the proxy's /@latest endpoint and returned a stale pseudo-version
// instead of the highest tag). The contract being tested here is:
// Latest must return the highest published tagged version of the
// module, derived from the proxy's /@v/list endpoint — independently
// of the implementation's choice of resolution endpoint.
//
// The test fetches /@v/list directly via http.Get (not through
// version.Latest), parses the line-separated tag list, picks the
// highest semver triple it can compute, and asserts Latest returns
// that exact value. If a future refactor switches Latest's endpoint
// or sort order, this test catches it before the next user does.
//
// Skipped under -short and on env errors so offline CI is not
// blocked. The module under test is gopkg.in/yaml.v3 because aiwf
// already depends on it and it has a stable, well-tagged history.
func TestLatest_RealProxy_ContractTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test (uses network)")
	}
	const module = "gopkg.in/yaml.v3"

	// Independent fetch of /@v/list — the test's "ground truth"
	// path. Deliberately uses net/http directly instead of any
	// helper from the version package so the assertion is not
	// circular.
	t.Setenv("GOPROXY", "")
	listURL := DefaultProxyURL + "/" + module + "/@v/list"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, http.NoBody)
	if err != nil {
		t.Skipf("building /@v/list request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("real-proxy /@v/list unavailable in this env: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Skipf("real-proxy /@v/list returned %d", resp.StatusCode)
	}
	body := make([]byte, 1<<20)
	n, _ := resp.Body.Read(body)
	body = body[:n]

	expected := computeHighestSemver(string(body))
	if expected == "" {
		t.Skipf("real proxy returned no tagged versions for %s", module)
	}

	// Now ask Latest, the code under test.
	got, err := latestFor(context.Background(), http.DefaultClient, module)
	if err != nil {
		t.Skipf("real-proxy Latest lookup unavailable in this env: %v", err)
	}
	if got.Version != expected {
		t.Errorf("Latest(%s) = %q, want %q (G28 contract violation: implementation diverged from highest-tag-from-/@v/list)",
			module, got.Version, expected)
	}
	if !got.Tagged {
		t.Errorf("Latest(%s).Tagged = false, want true (got version %q)", module, got.Version)
	}
}

// TestLatest_PrereleaseExcludedFromHighestSelection pins another
// G28-class invariant offline: pre-release versions in the /@v/list
// response (e.g., v3.0.0-rc1) must not beat clean releases. The
// underlying parseTriple already filters them, but the contract is
// load-bearing enough that an explicit fixture test catches a future
// refactor that loosens the filter.
func TestLatest_PrereleaseExcludedFromHighestSelection(t *testing.T) {
	const modulePath = "example.com/test/module"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/@v/list") {
			http.NotFound(w, r)
			return
		}
		// Mix: a pre-release that's lexically "greater" than the
		// highest non-pre-release should still lose, because
		// parseTriple rejects it.
		fmt.Fprint(w, "v0.1.0\nv0.2.0\nv0.3.0-rc1\nv0.1.5\n")
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GOPROXY", srv.URL)

	got, err := latestFor(context.Background(), srv.Client(), modulePath)
	if err != nil {
		t.Fatalf("latestFor: %v", err)
	}
	if got.Version != "v0.2.0" {
		t.Errorf("Version = %q, want v0.2.0 (pre-release v0.3.0-rc1 must not win)", got.Version)
	}
}

// computeHighestSemver is the test-side reference implementation of
// "highest tag from a /@v/list body." Deliberately written here, not
// imported from the version package, so the contract test is not
// circular: a regression in the production implementation cannot be
// hidden by a matching regression in the helper.
func computeHighestSemver(listBody string) string {
	var best string
	var bestTriple [3]int
	found := false
	for _, line := range strings.Split(listBody, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip pre-release / build suffixes.
		if strings.ContainsAny(strings.TrimPrefix(line, "v"), "-+") {
			continue
		}
		var major, minor, patch int
		if _, err := fmt.Sscanf(line, "v%d.%d.%d", &major, &minor, &patch); err != nil {
			continue
		}
		triple := [3]int{major, minor, patch}
		better := !found
		if found {
			for i := 0; i < 3; i++ {
				if triple[i] != bestTriple[i] {
					better = triple[i] > bestTriple[i]
					break
				}
			}
		}
		if better {
			best = line
			bestTriple = triple
			found = true
		}
	}
	return best
}
