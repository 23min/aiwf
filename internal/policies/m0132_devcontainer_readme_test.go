package policies

import "testing"

func TestPolicy_M0132DevcontainerReadme(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyM0132DevcontainerReadme)
}
