package policies

import "testing"

func TestPolicy_EmbeddedRitualsNoRetiredTrackingDoc(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyEmbeddedRitualsNoRetiredTrackingDoc)
}
