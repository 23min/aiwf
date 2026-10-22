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

// binaryStaleness returns why the running binary is stale when the
// commit it was built from differs from ref in the kernel checkout under
// rootDir. Returns "" when the check should be silent — Lane B per
// G-0176: kernel-developer-side only.
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
// The check is N=0 strict per design: any commit mismatch is stale. The
// result is advisory only: the doctor row's problem count is unaffected,
// and the pre-verb warning blocks nothing.
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
	return fmt.Sprintf("built from %s, which differs from %s %s; run `make install` to refresh",
		short(built), ref, short(refSHA))
}

// staleRowSuffix renders binaryStaleness's reason as the suffix appended
// to doctor's `binary:` row, or "" when there is none.
func staleRowSuffix(why string) string {
	if why == "" {
		return ""
	}
	return " (stale: " + why + ")"
}

// StaleBinaryWarning is the line every verb prints to stderr before it
// runs when the binary was built from another commit than rootDir's HEAD
// in the aiwf repository, or "" when it is current or the check is silent.
// Inside a linked worktree a Go pseudo-version is not judged: Go stamps a
// build there with the enclosing main checkout's commit, so the version
// would name the wrong source.
func StaleBinaryWarning(ctx context.Context, rootDir string, info version.Info, expectedModule string) string {
	if _, pseudo := version.PseudoSHA(strings.TrimSuffix(info.Version, "+dirty")); pseudo && inLinkedWorktree(rootDir) {
		return ""
	}
	if why := binaryStaleness(ctx, rootDir, "HEAD", info, expectedModule); why != "" {
		return "aiwf: warning: stale binary — " + why
	}
	return ""
}

// inLinkedWorktree reports whether rootDir is a linked worktree, whose
// `.git` is a file pointing at the main checkout's repository rather than
// a directory.
func inLinkedWorktree(rootDir string) bool {
	fi, err := os.Stat(filepath.Join(rootDir, ".git"))
	return err == nil && !fi.IsDir()
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
