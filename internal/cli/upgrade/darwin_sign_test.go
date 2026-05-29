package upgrade

import (
	"runtime"
	"testing"
)

// TestSignDarwinBinary_NoOpOnNonDarwin pins the GOOS gate: on Linux
// and Windows the helper returns nil immediately without attempting
// codesign. On Darwin the helper would attempt codesign on the path —
// we skip in that case because the path here is a fixture string, not
// a real Mach-O binary, and codesign(1) would legitimately fail.
//
// Internal test — exercises the unexported helper directly.
func TestSignDarwinBinary_NoOpOnNonDarwin(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "darwin" {
		t.Skip("Linux/Windows-only no-op branch; on darwin the helper invokes codesign(1) which is exercised elsewhere")
	}
	// A nonexistent path is fine: the function returns nil at the
	// GOOS gate before any path-touching codesign attempt.
	if err := signDarwinBinary("/nonexistent/aiwf-binary-path"); err != nil {
		t.Errorf("signDarwinBinary on %s = %v, want nil (no-op outside Darwin)", runtime.GOOS, err)
	}
}
