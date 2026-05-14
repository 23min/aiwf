package policies

import "testing"

func TestPolicy_RaceParallelCap(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyRaceParallelCap)
}
