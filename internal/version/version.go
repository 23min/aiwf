// Package version is the single point of truth for the running aiwf
// binary's version, the latest published version on the Go module
// proxy, and the skew classification between any two versions.
//
// Three version shapes show up in practice:
//
//  1. Tagged release — "v0.1.0", "v0.2.3-rc1". Returned by
//     `runtime/debug.ReadBuildInfo` for binaries installed via
//     `go install <module>@v0.1.0`. Tagged is true; Compare can
//     classify against another tagged value.
//
//  2. Working-tree build — "(devel)". Returned for `go build` /
//     `go run` from a working tree. Tagged is false; Compare always
//     returns SkewUnknown when this shape is on either side.
//
//  3. Pseudo-version — "v0.0.0-20260503...-abc123" or
//     "v0.1.0-pre.0.20060102150405-abcdef123456". Returned for
//     `go install <module>@<branch-or-sha>` when the commit isn't
//     tagged. Tagged is false; Compare returns SkewUnknown.
//
// Pre-release suffixes on otherwise-clean tags ("v0.1.0-rc1") are
// recognized as tagged (the proxy serves them) but Compare returns
// SkewUnknown when either side carries any pre-release segment. This
// is a deliberate narrowing: the comparison aiwf needs is between
// concrete release versions; pre-release ordering is full-semver
// territory and lives in golang.org/x/mod/semver if we ever need it.
package version

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

// DevelVersion is the value reported by runtime/debug.ReadBuildInfo
// for binaries built from a working tree (`go build`, `go run`).
const DevelVersion = "(devel)"

// Info is a parsed version with a flag for whether it identifies a
// concrete tagged release (vs. devel or a pseudo-version).
type Info struct {
	// Version is the raw version string. Always non-empty: a binary
	// with no build info reports DevelVersion.
	Version string
	// Tagged is true when Version is a clean semver tag of the form
	// `vMAJOR.MINOR.PATCH` (optionally with a pre-release or build
	// suffix). False for DevelVersion and pseudo-versions.
	Tagged bool
}

// Skew classifies the relationship between two Info values, comparing
// the first to the second. SkewUnknown is the safe default for any
// comparison that involves a non-tagged value or a pre-release suffix.
type Skew int

// Skew values.
const (
	SkewUnknown Skew = iota
	SkewEqual
	SkewAhead  // a > b
	SkewBehind // a < b
)

// String returns the lowercase name of the skew value.
func (s Skew) String() string {
	switch s {
	case SkewEqual:
		return "equal"
	case SkewAhead:
		return "ahead"
	case SkewBehind:
		return "behind"
	default:
		return "unknown"
	}
}

// pseudoVersionRE matches Go module proxy pseudo-versions, all three
// canonical forms per https://go.dev/ref/mod#pseudo-versions
// (mirrored in golang.org/x/mod/module):
//
//	v0.0.0-yyyymmddhhmmss-abcdefabcdef          (no parent tag)
//	vX.Y.(Z+1)-0.yyyymmddhhmmss-abcdefabcdef    (commits after vX.Y.Z)
//	vX.Y.Z-pre.0.yyyymmddhhmmss-abcdefabcdef    (between vX.Y.Z-pre and vX.Y.Z)
//
// The shared tail is a 14-digit UTC timestamp followed by a
// 12-character hex prefix of the commit SHA. The character before the
// timestamp is `-` for the simple form and `.` for the two
// pre-release-style forms.
var pseudoVersionRE = regexp.MustCompile(`[-.]\d{14}-[0-9a-f]{12}$`)

// taggedSemverRE matches a clean semver tag prefix: `v` followed by
// three dot-separated unsigned integers. Trailing pre-release or
// build segments are allowed (the proxy serves them as tagged) and
// are surfaced separately by Compare via the SkewUnknown fallback.
var taggedSemverRE = regexp.MustCompile(`^v\d+\.\d+\.\d+([-+].*)?$`)

// Current returns the running binary's version Info, read from
// runtime/debug.ReadBuildInfo. Binaries built without module info
// report DevelVersion with Tagged=false.
func Current() Info {
	if bi, ok := debug.ReadBuildInfo(); ok {
		return Parse(bi.Main.Version)
	}
	//coverage:ignore ReadBuildInfo only returns ok=false in
	// degenerate builds (e.g. CGO-stripped binaries with no module
	// info embedded); not reachable from `go test` or `go install`.
	return Parse("")
}

// ModulePath returns the module path of the running aiwf binary,
// read from runtime/debug.ReadBuildInfo. Returns "" when build info
// is unavailable. Used by Latest to construct the proxy lookup URL.
//
// The module path is the value declared in go.mod
// (e.g., "github.com/23min/ai-workflow-v2"). For the go-install
// invocation that wants the cmd's package path (one level deeper),
// use PackagePath instead.
func ModulePath() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		return bi.Main.Path
	}
	//coverage:ignore Same degenerate-build path as Current.
	return ""
}

// PackagePath returns the package path of the running aiwf binary's
// main package, read from runtime/debug.ReadBuildInfo. Returns ""
// when build info is unavailable.
//
// The package path is what `go install` accepts as its module-aware
// argument: e.g., "github.com/23min/ai-workflow-v2/cmd/aiwf".
// Always equal to or below ModulePath in the import-path tree.
func PackagePath() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		return bi.Path
	}
	//coverage:ignore Same degenerate-build path as Current.
	return ""
}

// Parse classifies an arbitrary version string into Info. Useful for
// values read from sources other than the running binary (the
// aiwf.yaml pin, the proxy response).
func Parse(v string) Info {
	if v == "" || v == DevelVersion {
		return Info{Version: DevelVersion}
	}
	return Info{Version: v, Tagged: isTagged(v)}
}

// Compare classifies how a relates to b. Returns SkewUnknown when
// either side is not Tagged or when either side carries a pre-release
// or build suffix (the aiwf upgrade flow only needs to compare
// concrete vMAJOR.MINOR.PATCH releases).
func Compare(a, b Info) Skew {
	if !a.Tagged || !b.Tagged {
		return SkewUnknown
	}
	an, ok := parseTriple(a.Version)
	if !ok {
		return SkewUnknown
	}
	bn, ok := parseTriple(b.Version)
	if !ok {
		return SkewUnknown
	}
	for i := 0; i < 3; i++ {
		switch {
		case an[i] < bn[i]:
			return SkewBehind
		case an[i] > bn[i]:
			return SkewAhead
		}
	}
	return SkewEqual
}

// isTagged reports whether v is a clean tagged semver value rather
// than a pseudo-version. Pseudo-versions match the timestamp+sha
// suffix regardless of the base version they're attached to.
//
// The `+dirty` suffix appended by Go's VCS-stamped builds (when
// uncommitted changes are present in the working tree) flips Tagged
// false: the binary doesn't correspond to any committed state, so
// it is not safely comparable as a release.
func isTagged(v string) bool {
	if strings.HasSuffix(v, "+dirty") {
		return false
	}
	if !taggedSemverRE.MatchString(v) {
		return false
	}
	if pseudoVersionRE.MatchString(v) {
		return false
	}
	return true
}

// parseTriple extracts the (major, minor, patch) integers from a
// clean semver value. Returns ok=false when the value carries any
// pre-release or build suffix (Compare classifies that as Unknown).
func parseTriple(v string) ([3]int, bool) {
	core := strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(core, "-+"); i >= 0 {
		return [3]int{}, false
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var out [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}, false
		}
		out[i] = n
	}
	return out, true
}

// DefaultProxyURL is the public Go module proxy operated by Google.
// Used when GOPROXY is unset.
const DefaultProxyURL = "https://proxy.golang.org"

// LatestTimeout caps the HTTP round-trip for proxy lookups. Three
// seconds is enough for proxy.golang.org from anywhere with working
// connectivity; a slow link returns an error rather than blocking
// `aiwf doctor` indefinitely.
const LatestTimeout = 3 * time.Second

// ErrProxyDisabled is returned by Latest when GOPROXY=off (or when
// the GOPROXY chain contains no http(s) entry). Callers that want
// optional skew detection should treat this as a non-fatal "skip
// the latest-version check" signal.
var ErrProxyDisabled = errors.New("module proxy disabled")

// proxyResponse is the JSON shape returned by the Go module proxy at
// /<module>/@latest. Only Version is load-bearing for skew detection.
type proxyResponse struct {
	Version string `json:"Version"`
	Time    string `json:"Time"`
}

// Latest fetches the latest published version of the aiwf module
// from the Go module proxy. Honors GOPROXY (returns ErrProxyDisabled
// when set to `off` or to a chain with no http(s) entry). The HTTP
// round-trip is capped at LatestTimeout independently of ctx; ctx's
// cancellation also aborts the request.
//
// Returns the parsed Info from the proxy response. The Info is
// Tagged=true for clean semver tags and Tagged=false for pseudo-
// versions (the proxy may serve either depending on the module's
// tag history).
func Latest(ctx context.Context) (Info, error) {
	return latestFor(ctx, http.DefaultClient, ModulePath())
}

// latestFor is the testable seam: callers in tests pass an httptest
// server's client and a fixed module path; Latest passes the package
// defaults.
//
// The function tries `/<module>/@v/list` first to get all published
// tagged versions and picks the highest semver. This avoids a known
// proxy quirk where `/<module>/@latest` can be cached with a pre-tag
// pseudo-version answer and not refresh after the first tag lands.
// Only when the list endpoint returns no tagged versions does the
// function fall back to `/<module>/@latest`, which serves the
// pseudo-version of the latest commit on the default branch.
func latestFor(ctx context.Context, client *http.Client, modulePath string) (Info, error) {
	if modulePath == "" {
		return Info{}, errors.New("module path unavailable from build info")
	}
	base, err := proxyBase()
	if err != nil {
		return Info{}, err
	}

	// Step 1: list tagged versions; pick the highest.
	listURL := strings.TrimRight(base, "/") + "/" + modulePath + "/@v/list"
	highest, found, listErr := highestTaggedFromList(ctx, client, listURL)
	if listErr != nil {
		return Info{}, listErr
	}
	if found {
		return highest, nil
	}

	// Step 2: fall back to @latest for the no-tags-yet case.
	u := strings.TrimRight(base, "/") + "/" + modulePath + "/@latest"

	httpCtx, cancel := context.WithTimeout(ctx, LatestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return Info{}, fmt.Errorf("building proxy request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return Info{}, fmt.Errorf("querying %s: %w", u, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		// Drain a bounded amount of the body for the error message;
		// proxy error responses are short.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return Info{}, fmt.Errorf("proxy %s returned %d: %s",
			u, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload proxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Info{}, fmt.Errorf("decoding proxy response: %w", err)
	}

	return Parse(payload.Version), nil
}

// highestTaggedFromList queries the proxy's `/@v/list` endpoint for
// the published tagged versions of a module and returns the highest
// one as a tagged Info. ok=false means the endpoint returned no
// tagged versions (the module has never been tagged); err means a
// transport or status-code failure that the caller should propagate.
//
// The list endpoint returns one version per line. Pre-release and
// build-suffixed values are skipped since Compare returns Unknown
// against them (they can't anchor a "latest" comparison).
func highestTaggedFromList(ctx context.Context, client *http.Client, listURL string) (Info, bool, error) {
	httpCtx, cancel := context.WithTimeout(ctx, LatestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodGet, listURL, http.NoBody)
	if err != nil {
		return Info{}, false, fmt.Errorf("building proxy list request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return Info{}, false, fmt.Errorf("querying %s: %w", listURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return Info{}, false, fmt.Errorf("proxy %s returned %d: %s",
			listURL, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Info{}, false, fmt.Errorf("reading proxy list: %w", err)
	}

	var best Info
	var bestTriple [3]int
	found := false
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		info := Parse(line)
		if !info.Tagged {
			continue
		}
		triple, ok := parseTriple(info.Version)
		if !ok {
			continue
		}
		if !found || tripleGreater(triple, bestTriple) {
			best = info
			bestTriple = triple
			found = true
		}
	}
	return best, found, nil
}

// tripleGreater reports whether a > b in lexicographic (major,
// minor, patch) order.
func tripleGreater(a, b [3]int) bool {
	for i := 0; i < 3; i++ {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

// proxyBase resolves the module proxy URL from GOPROXY. Walks the
// comma-separated chain left-to-right and returns the first http(s)
// entry. `off` terminates the walk with ErrProxyDisabled; `direct`
// is skipped (the toolchain's "fetch from VCS" mode has no HTTP
// surface for our @latest lookup). When GOPROXY is unset, returns
// DefaultProxyURL.
//
// Per the Go toolchain spec, GOPROXY may also be separated by `|`
// (fall-through-on-any-error semantics) instead of `,`. Both
// separators get the same first-http(s) treatment here.
func proxyBase() (string, error) {
	raw := os.Getenv("GOPROXY")
	if raw == "" {
		return DefaultProxyURL, nil
	}
	for _, p := range splitProxyChain(raw) {
		switch p {
		case "":
			continue
		case "off":
			return "", ErrProxyDisabled
		case "direct":
			continue
		default:
			if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
				return p, nil
			}
			// Anything else is malformed for our purposes; fall through.
		}
	}
	return "", ErrProxyDisabled
}

// splitProxyChain splits a GOPROXY value on `,` or `|` and trims
// whitespace from each entry. Empty entries are preserved so callers
// can recognize them; proxyBase skips them.
func splitProxyChain(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '|'
	})
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
