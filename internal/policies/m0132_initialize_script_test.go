package policies

import "testing"

func TestPolicy_M0132InitializeScript(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyM0132InitializeScript)
}
