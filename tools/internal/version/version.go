// Package version is the single point of truth for the running aiwf
// binary's version and the skew classification between any two
// versions. A future step in upgrade-flow-plan.md adds a Latest(ctx)
// function that fetches the latest published version from the Go
// module proxy; this file ships only Current and Compare.
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
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
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
// canonical forms (per `golang.org/x/mod/module`):
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
func isTagged(v string) bool {
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
