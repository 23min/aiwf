package cliutil

import (
	"testing"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
)

// ct builds a scope.Commit from a SHA and alternating key/value pairs.
func ct(sha string, kv ...string) scope.Commit {
	c := scope.Commit{SHA: sha}
	for i := 0; i+1 < len(kv); i += 2 {
		c.Trailers = append(c.Trailers, gitops.Trailer{Key: kv[i], Value: kv[i+1]})
	}
	return c
}

// TestOpenersFrom covers the pure authorize-opener filter over an
// UNFILTERED commit slice — the shape render's single pass hands it
// (M-0221). AuthorizeOpeners' grep pre-filters to opener commits, so the
// predicate-skip arms here (non-authorize verb, non-opened scope) are
// only ever traversed on render's path; this pins them. Canonicalization
// of the mapped entity id (E-14 → E-0014) is asserted too.
func TestOpenersFrom(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		// opener → mapped.
		ct("sha-open", gitops.TrailerVerb, "authorize", gitops.TrailerScope, "opened", gitops.TrailerEntity, "E-0002"),
		// non-authorize verb → skipped (render-only branch).
		ct("sha-promote", gitops.TrailerVerb, "promote", gitops.TrailerEntity, "E-0003"),
		// authorize but not opened → skipped (render-only branch).
		ct("sha-paused", gitops.TrailerVerb, "authorize", gitops.TrailerScope, "paused", gitops.TrailerEntity, "E-0002"),
		// opener missing aiwf-entity → skipped (blank id guard).
		ct("sha-noent", gitops.TrailerVerb, "authorize", gitops.TrailerScope, "opened"),
		// opener with blank SHA → skipped.
		ct("", gitops.TrailerVerb, "authorize", gitops.TrailerScope, "opened", gitops.TrailerEntity, "E-0009"),
		// narrow-width opener → canonicalized in the map value.
		ct("sha-narrow", gitops.TrailerVerb, "authorize", gitops.TrailerScope, "opened", gitops.TrailerEntity, "E-14"),
	}
	got := OpenersFrom(commits)
	want := map[string]string{
		"sha-open":   "E-0002",
		"sha-narrow": "E-0014", // canonicalized
	}
	if len(got) != len(want) {
		t.Fatalf("OpenersFrom = %v, want %v", got, want)
	}
	for sha, ent := range want {
		if got[sha] != ent {
			t.Errorf("OpenersFrom[%q] = %q, want %q", sha, got[sha], ent)
		}
	}
}
