package entityview

import (
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

// eventfromcommit_admit_test.go pins the rule deciding whether a grepped
// commit is an entity event at all.
//
// The query selects commits by grepping for an `aiwf-entity:` or
// `aiwf-prior-entity:` line, which matches body prose as readily as a real
// trailer. What separates the two is git's own trailer parser: it returns a
// value for a genuine trailer and nothing for a prose line that merely looks
// like one. So the admit test reads the parsed entity trailer — the mechanism
// the false positive is actually about — rather than proxying through
// verb-or-actor, which additionally discards a commit whose entity trailer is
// genuine and whose only fault is carrying nothing else.
func TestEventFromCommit_AdmitsAnEventByItsParsedEntityTrailer(t *testing.T) {
	t.Parallel()

	tr := func(k, v string) gitops.Trailer { return gitops.Trailer{Key: k, Value: v} }

	cases := []struct {
		name     string
		trailers []gitops.Trailer
		want     bool
	}{
		// The case D-0071 created: a shipped-surface edit proves its
		// provenance with the entity trailer alone, and no verb exists to
		// name what it did.
		{"entity alone is an event", []gitops.Trailer{tr(gitops.TrailerEntity, "M-0001")}, true},
		// A reallocate's lineage: querying the old id matches on
		// prior-entity, so that key admits on its own too.
		{"prior-entity alone is an event", []gitops.Trailer{tr(gitops.TrailerPriorEntity, "M-0001")}, true},
		{"the ordinary verb event", []gitops.Trailer{
			tr(gitops.TrailerVerb, "promote"),
			tr(gitops.TrailerEntity, "M-0001"),
			tr(gitops.TrailerActor, "human/peter"),
		}, true},
		// The false positive the drop exists for: --grep matched a body
		// line, so git's parser handed back no entity trailer. Verb and
		// actor are present here to show the admit test does not consult
		// them — a commit git cannot attribute to an entity is not that
		// entity's event, whatever else it carries.
		{"no parsed entity trailer is not an event", []gitops.Trailer{
			tr(gitops.TrailerVerb, "promote"),
			tr(gitops.TrailerActor, "human/peter"),
		}, false},
		{"an empty trailer block is not an event", nil, false},
		// Whitespace is what an unfolded empty trailer value arrives as;
		// treating it as present would readmit the prose match.
		{"a blank entity value is not an event", []gitops.Trailer{tr(gitops.TrailerEntity, "   ")}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, ok := EventFromCommit("abcdef1234567890", "2026-07-03T00:00:00Z", "a subject", "a subject", tc.trailers)
			if ok != tc.want {
				t.Errorf("EventFromCommit ok = %v, want %v", ok, tc.want)
			}
		})
	}
}
