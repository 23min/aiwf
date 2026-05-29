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
// row when the running binary's source SHA differs from
// refs/remotes/origin/main in the kernel checkout under rootDir.
// Returns "" (no suffix) when the check should be silent — Lane B per
// G-0176: kernel-developer-side only.
//
// Skip-by-shape conditions (binary):
//   - DevelVersion (working-tree go-build); not deployable
//   - +dirty suffix (VCS-stamped working-tree build); same reason
//   - Tagged release; covered by the latest: row instead
//   - Not a pseudo-version (defensive — unreachable after the above)
//
// Skip-by-environment conditions:
//   - expectedModule == "" (binary lacks build info)
//   - rootDir's go.mod module path != expectedModule (downstream
//     consumer repo; the check is kernel-developer-only)
//   - refs/remotes/origin/main is absent (fresh clone, no fetch yet,
//     or detached state); degrade silently — mirrors how the latest:
//     row handles GOPROXY unreachability.
//
// The check is N=0 strict per design: any SHA mismatch triggers the
// suffix. The suffix is advisory only; the row's problem count is
// unaffected.
func binaryStaleness(ctx context.Context, rootDir string, info version.Info, expectedModule string) string {
	if info.Version == version.DevelVersion || info.Version == "" {
		return ""
	}
	if strings.HasSuffix(info.Version, "+dirty") {
		return ""
	}
	if info.Tagged {
		return ""
	}
	pseudoSHA, ok := version.PseudoSHA(info.Version)
	if !ok {
		return ""
	}
	if expectedModule == "" {
		return ""
	}
	rootModule, err := readModulePath(rootDir)
	if err != nil || rootModule != expectedModule {
		return ""
	}
	ref := "refs/remotes/origin/main"
	exists, err := gitops.HasRef(ctx, rootDir, ref)
	if err != nil || !exists {
		return ""
	}
	mainSHA, err := gitops.ShortSHA(ctx, rootDir, ref, 12)
	if err != nil {
		//coverage:ignore environmental git failure between HasRef and
		// ShortSHA — reachable only via TOCTOU on the ref between two
		// git subprocesses; not exercisable without intrusive mocking.
		return ""
	}
	if mainSHA == pseudoSHA {
		return ""
	}
	return fmt.Sprintf(" (stale: pseudo-base SHA %s differs from %s %s; run `make install` to refresh)",
		pseudoSHA, ref, mainSHA)
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
