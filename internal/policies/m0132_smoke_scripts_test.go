package policies

import "testing"

func TestPolicy_M0132SmokeScripts(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyM0132SmokeScripts)
}
