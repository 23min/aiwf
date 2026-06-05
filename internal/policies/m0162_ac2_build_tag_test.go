package policies

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestM0162_AC2_BuildTagExclusion pins M-0162/AC-2's load-bearing
// claim: the `branchtest` sub-package and its Pin/Pins symbols are
// excluded from production `go build` because pin.go carries
// `//go:build testpins`. The test compiles the aiwf binary
// without `-tags testpins`, then runs `go tool nm` and asserts
// no symbols from the `branchtest` import path appear.
//
// Sabotage-verifiable: remove the `//go:build testpins` header
// from pin.go and this test fires reporting the leaked symbols
// (the package would be pulled in transitively only if some other
// non-tagged file imports it, but the test asserts the absence
// directly — leaks via accidental import would fire too).
//
// The test is unconditional (no build tag) so it runs in every
// `go test` invocation including the pre-commit hook.
func TestM0162_AC2_BuildTagExclusion(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	outBinary := filepath.Join(t.TempDir(), "aiwf-no-pins")

	build := exec.Command("go", "build", "-o", outBinary, "./cmd/aiwf")
	build.Dir = root
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build (no -tags testpins) failed: %v\n%s", err, out)
	}

	nm := exec.Command("go", "tool", "nm", outBinary)
	out, err := nm.CombinedOutput()
	if err != nil {
		t.Fatalf("go tool nm: %v\n%s", err, out)
	}

	const marker = "internal/workflows/spec/branch/branchtest"
	count := strings.Count(string(out), marker)
	if count != 0 {
		// Surface a couple of example lines for diagnostic value.
		var lines []string
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, marker) {
				lines = append(lines, line)
				if len(lines) >= 5 {
					break
				}
			}
		}
		t.Errorf("M-0162/AC-2: %d symbol(s) from %q leaked into production binary\n  build tag may be missing or a non-tagged file imports branchtest\n  examples:\n    %s",
			count, marker, strings.Join(lines, "\n    "))
	}
}

// TestM0162_AC2_PackageDocPresence pins M-0162/AC-2's
// AI-discoverability claim: pin.go's package doc comment carries
// the build-tag convention next to the symbol so a reader (human
// or AI) tracing the registry's lifetime sees the convention
// without consulting a separate README. The structural assertion
// scans pin.go for the build-tag directive and a usage code-fence
// referencing `branchtest.Pin(`.
//
// Sabotage-verifiable: strip the package doc comment or the build
// tag and this test fires naming the missing string.
func TestM0162_AC2_PackageDocPresence(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	pinPath := filepath.Join(root, "internal", "workflows", "spec", "branch", "branchtest", "pin.go")
	contents, err := os.ReadFile(pinPath)
	if err != nil {
		t.Fatalf("read %s: %v", pinPath, err)
	}

	src := string(contents)
	checks := []struct {
		needle string
		why    string
	}{
		{"//go:build testpins", "build-tag header (load-bearing exclusion mechanism)"},
		{"Package branchtest", "package doc comment (AI-discoverable convention)"},
		{"branchtest.Pin(", "usage example referencing the Pin call shape"},
	}
	for _, c := range checks {
		if !strings.Contains(src, c.needle) {
			t.Errorf("M-0162/AC-2: pin.go missing %q (%s)", c.needle, c.why)
		}
	}
}
