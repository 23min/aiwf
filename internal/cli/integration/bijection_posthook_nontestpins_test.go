//go:build !testpins

package integration

// bijectionPostHook is a no-op without -tags testpins. The Pin
// registry is also a no-op without the tag (per pin_nontestpins_test.go),
// so the bijection check is meaningless — Pins() is empty.
func bijectionPostHook() (failure string, overrideExit int) {
	return "", 0
}
