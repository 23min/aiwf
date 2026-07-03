package cliutil

import (
	"testing"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
)

// ct builds a CommitTrailers from a SHA and alternating key/value pairs.
// Repeated keys (notably aiwf-scope-ends) preserve order, so the scope
// replay's multi-value path is exercised faithfully.
func ct(sha string, kv ...string) CommitTrailers {
	c := CommitTrailers{SHA: sha}
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
	commits := []CommitTrailers{
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

// TestReplayScopes covers the pure scope FSM render replays from the
// shared HEAD pass (M-0221): open, pause, resume, end (incl. repeating
// aiwf-scope-ends on one commit), and the two no-op arms (pause with no
// active scope, resume with no paused scope). LoadEntityScopes' git-based
// tests already traverse the happy paths; this pins the pure contract
// render depends on directly.
func TestReplayScopes(t *testing.T) {
	t.Parallel()

	t.Run("open sets active with metadata", func(t *testing.T) {
		t.Parallel()
		s := ReplayScopes([]CommitTrailers{
			ct("auth1", gitops.TrailerVerb, "authorize", gitops.TrailerScope, "opened",
				gitops.TrailerEntity, "E-0002", gitops.TrailerTo, "ai/claude",
				gitops.TrailerActor, "human/peter", gitops.TrailerReason, "delegate"),
		})
		if len(s) != 1 {
			t.Fatalf("len = %d, want 1", len(s))
		}
		if s[0].State != scope.StateActive || s[0].AuthSHA != "auth1" ||
			s[0].Entity != "E-0002" || s[0].Agent != "ai/claude" || s[0].Principal != "human/peter" {
			t.Errorf("scope = %+v, want active auth1/E-0002/ai-claude/human-peter", s[0])
		}
	})

	t.Run("open then pause then resume", func(t *testing.T) {
		t.Parallel()
		s := ReplayScopes([]CommitTrailers{
			ct("auth1", gitops.TrailerVerb, "authorize", gitops.TrailerScope, "opened", gitops.TrailerEntity, "E-0002"),
			ct("p1", gitops.TrailerVerb, "authorize", gitops.TrailerScope, "paused", gitops.TrailerEntity, "E-0002"),
			ct("r1", gitops.TrailerVerb, "authorize", gitops.TrailerScope, "resumed", gitops.TrailerEntity, "E-0002"),
		})
		if len(s) != 1 || s[0].State != scope.StateActive {
			t.Fatalf("scopes = %+v, want one active scope after resume", s)
		}
		if len(s[0].Events) != 3 {
			t.Errorf("events = %d, want 3 (open, pause, resume)", len(s[0].Events))
		}
	})

	t.Run("repeating scope-ends on one commit ends both openers", func(t *testing.T) {
		t.Parallel()
		s := ReplayScopes([]CommitTrailers{
			ct("authA", gitops.TrailerVerb, "authorize", gitops.TrailerScope, "opened", gitops.TrailerEntity, "E-0002"),
			ct("authB", gitops.TrailerVerb, "authorize", gitops.TrailerScope, "opened", gitops.TrailerEntity, "E-0002"),
			// one terminal commit ending both scopes — the repeating-trailer path.
			ct("endcommit", gitops.TrailerVerb, "promote", gitops.TrailerEntity, "E-0002",
				gitops.TrailerScopeEnds, "authA", gitops.TrailerScopeEnds, "authB"),
		})
		if len(s) != 2 {
			t.Fatalf("len = %d, want 2 scopes", len(s))
		}
		for _, sc := range s {
			if sc.State != scope.StateEnded {
				t.Errorf("scope %s state = %q, want ended", sc.AuthSHA, sc.State)
			}
		}
	})

	t.Run("pause with no active scope is a no-op", func(t *testing.T) {
		t.Parallel()
		s := ReplayScopes([]CommitTrailers{
			ct("p1", gitops.TrailerVerb, "authorize", gitops.TrailerScope, "paused", gitops.TrailerEntity, "E-0002"),
		})
		if len(s) != 0 {
			t.Errorf("scopes = %+v, want none (pause with nothing active)", s)
		}
	})

	t.Run("resume with no paused scope is a no-op", func(t *testing.T) {
		t.Parallel()
		s := ReplayScopes([]CommitTrailers{
			ct("auth1", gitops.TrailerVerb, "authorize", gitops.TrailerScope, "opened", gitops.TrailerEntity, "E-0002"),
			ct("r1", gitops.TrailerVerb, "authorize", gitops.TrailerScope, "resumed", gitops.TrailerEntity, "E-0002"),
		})
		if len(s) != 1 || s[0].State != scope.StateActive {
			t.Errorf("scopes = %+v, want one active scope (resume no-op leaves it active)", s)
		}
	})

	t.Run("scope-ends for unknown opener is ignored", func(t *testing.T) {
		t.Parallel()
		s := ReplayScopes([]CommitTrailers{
			ct("endcommit", gitops.TrailerVerb, "promote", gitops.TrailerEntity, "E-0002",
				gitops.TrailerScopeEnds, "nonexistent-auth"),
		})
		if len(s) != 0 {
			t.Errorf("scopes = %+v, want none (scope-ends referencing no opener)", s)
		}
	})
}
