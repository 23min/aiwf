package policies

import "testing"

func TestPolicy_TestSetupPresence(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyTestSetupPresence)
}
