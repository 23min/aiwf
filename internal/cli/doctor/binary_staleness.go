package doctor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/version"
)

// binaryStaleness returns a suffix string for the doctor `binary:`
// row when the commit the running binary was built from differs from
// ref in the kernel checkout under rootDir. Returns "" (no suffix) when
// the check should be silent — Lane B per G-0176: kernel-developer-side
// only.
//
// The binary's commit comes from version.BuildCommit: a hash from a
// pseudo-version or a Makefile stamp, or a tag for a release, resolved
// in rootDir. A dirty build is judged by the commit it started from.
//
// Silent when:
//   - the version names no commit (`(devel)`, an empty string)
//   - expectedModule == "" (binary lacks build info)
//   - rootDir's go.mod module path != expectedModule (downstream
//     consumer repo; the check is kernel-developer-only)
//   - ref or the binary's tag does not resolve in rootDir (fresh clone,
//     no fetch yet, a tag not fetched); degrade silently — mirrors how
//     the latest: row handles GOPROXY unreachability.
//
// The check is N=0 strict per design: any commit mismatch triggers the
// suffix. The suffix is advisory only; the row's problem count is
// unaffected.
func binaryStaleness(ctx context.Context, rootDir, ref string, info version.Info, expectedModule string) string {
	built, hash, ok := version.BuildCommit(info.Version)
	if !ok || expectedModule == "" {
		return ""
	}
	rootModule, err := readModulePath(rootDir)
	if err != nil || rootModule != expectedModule {
		return ""
	}
	refSHA, err := gitops.ResolveCommitSHA(ctx, rootDir, ref)
	if err != nil {
		return ""
	}
	if !hash {
		if built, err = gitops.ResolveCommitSHA(ctx, rootDir, built); err != nil {
			return ""
		}
	}
	if strings.HasPrefix(refSHA, built) {
		return ""
	}
	return fmt.Sprintf(" (stale: built from %s, which differs from %s %s; run `make install` to refresh)",
		short(built), ref, short(refSHA))
}

// short abbreviates a commit hash to the 12 characters the staleness
// suffix prints.
func short(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// readModulePath parses the `module <path>` line from rootDir/go.mod.
// Returns ("", err) when go.mod is unreadable; ("", nil) when no
// `module` line is found.
func readModulePath(rootDir string) (string, error) {
	b, err := os.ReadFile(filepath.Join(rootDir, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", nil
}
