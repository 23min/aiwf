package policies

import "testing"

func TestPolicy_RaceParallelCap(t *testing.T) {
	runPolicy(t, PolicyRaceParallelCap)
}
