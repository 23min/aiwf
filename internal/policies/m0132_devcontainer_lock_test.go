package policies

import "testing"

func TestPolicy_M0132DevcontainerLock(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyM0132DevcontainerLock)
}
