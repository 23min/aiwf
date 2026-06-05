//go:build testpins

package branchtest

import (
	"reflect"
	"testing"
)

// TestPin_API_TwoStringSignature pins M-0162/AC-2's API-shape claim
// for Pin: it accepts two strings (cellID, testName) and returns
// nothing. A regression that adds a third arg or changes return
// type fires this test.
//
// Sabotage-verifiable: change Pin's signature and this test no
// longer compiles (the assignment to a typed function-value below
// catches signature drift at compile time, then the runtime
// reflect assertion double-checks).
func TestPin_API_TwoStringSignature(t *testing.T) {
	t.Parallel()

	var f func(string, string) = Pin
	if reflect.ValueOf(f).IsNil() {
		t.Fatalf("Pin is nil-valued (function-value assertion impossible)")
	}
}

// TestPins_API_MapReturn pins M-0162/AC-2's API-shape claim for
// Pins: it takes no args and returns map[string][]string. A
// regression that changes the return shape fires this test.
//
// Sabotage-verifiable: change Pins' return type and this test
// no longer compiles.
func TestPins_API_MapReturn(t *testing.T) {
	t.Parallel()

	var f func() map[string][]string = Pins
	got := f()
	if got == nil {
		t.Fatalf("Pins() returned nil; want empty map (snapshot semantics require non-nil empty)")
	}
}
