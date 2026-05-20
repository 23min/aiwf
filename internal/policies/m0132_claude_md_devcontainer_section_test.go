package policies

import "testing"

func TestPolicy_M0132ClaudeMdDevcontainerSection(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyM0132ClaudeMdDevcontainerSection)
}
