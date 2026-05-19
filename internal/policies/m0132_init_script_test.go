package policies

import "testing"

func TestPolicy_M0132InitScript(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyM0132InitScript)
}
