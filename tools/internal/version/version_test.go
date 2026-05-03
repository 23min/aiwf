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

		// +dirty suffix (Go VCS-stamping for uncommitted working
		// trees): never tagged, regardless of the base shape.
		{"v0.1.0+dirty", "v0.1.0+dirty", false},
		{"v0.0.0-20260503120000-abcdef123456+dirty", "v0.0.0-20260503120000-abcdef123456+dirty", false},

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

func TestModulePath_TestBinary(t *testing.T) {
	// Under `go test`, ModulePath returns the module of the test
	// binary, which is this repo's go.mod path.
	got := ModulePath()
	const want = "github.com/23min/ai-workflow-v2"
	if got != want {
		t.Errorf("ModulePath() = %q, want %q", got, want)
	}
}

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

func TestLatest_RealProxy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test (uses network)")
	}
	// Hit the real Go module proxy against a stable, well-known
	// module to verify the URL shape and JSON parsing match
	// production. Uses gopkg.in/yaml.v3 because aiwf already depends
	// on it and it has stable releases.
	t.Setenv("GOPROXY", "")
	got, err := latestFor(context.Background(), http.DefaultClient, "gopkg.in/yaml.v3")
	if err != nil {
		t.Skipf("real-proxy lookup unavailable in this env: %v", err)
	}
	if got.Version == "" {
		t.Errorf("got empty Version from real proxy")
	}
}
