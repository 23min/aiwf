package gitops

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestValidateTrailer_KnownKeys(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		// aiwf-actor: any role/<id> shape.
		{"actor human ok", TrailerActor, "human/peter", false},
		{"actor ai ok", TrailerActor, "ai/claude", false},
		{"actor bot ok", TrailerActor, "bot/ci", false},
		{"actor missing slash", TrailerActor, "human", true},
		{"actor whitespace", TrailerActor, "human / peter", true},
		{"actor empty role", TrailerActor, "/peter", true},

		// aiwf-principal: must be human/.
		{"principal human ok", TrailerPrincipal, "human/peter", false},
		{"principal ai rejected", TrailerPrincipal, "ai/claude", true},
		{"principal bot rejected", TrailerPrincipal, "bot/ci", true},
		{"principal malformed", TrailerPrincipal, "no-slash", true},

		// aiwf-on-behalf-of: same rule as principal.
		{"on-behalf-of human ok", TrailerOnBehalfOf, "human/peter", false},
		{"on-behalf-of ai rejected", TrailerOnBehalfOf, "ai/claude", true},

		// aiwf-authorized-by / aiwf-scope-ends: 7-40 hex.
		{"authorized-by 7 hex", TrailerAuthorizedBy, "4b13a0f", false},
		{"authorized-by 40 hex", TrailerAuthorizedBy, strings.Repeat("a", 40), false},
		{"authorized-by 6 hex too short", TrailerAuthorizedBy, "4b13a0", true},
		{"authorized-by 41 hex too long", TrailerAuthorizedBy, strings.Repeat("a", 41), true},
		{"authorized-by uppercase rejected", TrailerAuthorizedBy, "ABCDEF7", true},
		{"authorized-by non-hex rejected", TrailerAuthorizedBy, "g123456", true},
		{"scope-ends ok", TrailerScopeEnds, "abc1234", false},

		// aiwf-scope: closed set.
		{"scope opened", TrailerScope, "opened", false},
		{"scope paused", TrailerScope, "paused", false},
		{"scope resumed", TrailerScope, "resumed", false},
		{"scope ended rejected (not in set)", TrailerScope, "ended", true},
		{"scope active rejected", TrailerScope, "active", true},
		{"scope empty rejected", TrailerScope, "", true},

		// aiwf-reason / aiwf-force / aiwf-audit-only: non-empty after trim.
		{"reason ok", TrailerReason, "blocked by E-09 fixture work", false},
		{"reason whitespace only", TrailerReason, "   ", true},
		{"reason empty", TrailerReason, "", true},
		{"force ok", TrailerForce, "scope was wrong from the start", false},
		{"force empty rejected", TrailerForce, "", true},
		{"audit-only ok", TrailerAuditOnly, "manual commit recovery", false},
		{"audit-only empty rejected", TrailerAuditOnly, "", true},

		// Loose-string trailers: any non-empty value is fine, no shape check.
		{"verb any value", TrailerVerb, "promote", false},
		{"entity any value", TrailerEntity, "M-007/AC-1", false},
		{"to any value", TrailerTo, "met", false},
		{"prior-entity any value", TrailerPriorEntity, "M-006", false},
		{"tests any value", TrailerTests, "pass=12 fail=0 skip=0", false},

		// Unknown keys: tolerated (forward compat).
		{"unknown key tolerated", "aiwf-future-key", "anything", false},
		{"unrelated trailer tolerated", "Signed-off-by", "human/peter", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTrailer(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTrailer(%q, %q) err = %v, wantErr %v", tt.key, tt.value, err, tt.wantErr)
			}
		})
	}
}

// TestSortedTrailers_CanonicalOrder pins the load-bearing property:
// trailers come out in the order documented in provenance-model.md
// regardless of the order callers assemble them in.
func TestSortedTrailers_CanonicalOrder(t *testing.T) {
	t.Parallel()
	// Construct in a deliberately-wrong order.
	in := []Trailer{
		{Key: TrailerReason, Value: "stop work on E-03"},
		{Key: TrailerScope, Value: "paused"},
		{Key: TrailerActor, Value: "human/peter"},
		{Key: TrailerVerb, Value: "authorize"},
		{Key: TrailerEntity, Value: "E-03"},
	}
	got := SortedTrailers(in)
	want := []Trailer{
		{Key: TrailerVerb, Value: "authorize"},
		{Key: TrailerEntity, Value: "E-03"},
		{Key: TrailerActor, Value: "human/peter"},
		{Key: TrailerScope, Value: "paused"},
		{Key: TrailerReason, Value: "stop work on E-03"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("SortedTrailers mismatch (-want +got):\n%s", diff)
	}
}

// TestSortedTrailers_RepeatedKeysPreserveOrder: a commit may carry
// multiple aiwf-scope-ends entries (one per ended scope on a
// terminal-promote). Their relative input order must survive the
// sort so the rendered trailer block is reproducible.
func TestSortedTrailers_RepeatedKeysPreserveOrder(t *testing.T) {
	t.Parallel()
	in := []Trailer{
		{Key: TrailerScopeEnds, Value: "abc1234"},
		{Key: TrailerScopeEnds, Value: "def5678"},
		{Key: TrailerScopeEnds, Value: "9999999"},
	}
	got := SortedTrailers(in)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	for i, want := range []string{"abc1234", "def5678", "9999999"} {
		if got[i].Value != want {
			t.Errorf("got[%d].Value = %q, want %q", i, got[i].Value, want)
		}
	}
}

// TestSortedTrailers_UnknownKeysLast: forward compatibility — a
// caller passing an unrecognized trailer must not break the sort,
// and the unknown key sorts to the end (lex order among unknowns).
func TestSortedTrailers_UnknownKeysLast(t *testing.T) {
	t.Parallel()
	in := []Trailer{
		{Key: "z-future-trailer", Value: "v1"},
		{Key: TrailerActor, Value: "human/peter"},
		{Key: "a-other-trailer", Value: "v2"},
		{Key: TrailerVerb, Value: "promote"},
	}
	got := SortedTrailers(in)
	wantKeys := []string{TrailerVerb, TrailerActor, "a-other-trailer", "z-future-trailer"}
	for i, want := range wantKeys {
		if got[i].Key != want {
			t.Errorf("got[%d].Key = %q, want %q", i, got[i].Key, want)
		}
	}
}

// TestSortedTrailers_RoundTripsThroughCommit confirms the sorted
// trailer set survives a real git commit + readback. Pins the
// load-bearing property: every new I2.5 key parses back to its
// original value through `git log --pretty=%(trailers)`.
func TestSortedTrailers_RoundTripsThroughCommit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}
	// Touch a file so git has something to commit (refuses empty
	// commits by default).
	if err := os.WriteFile(filepath.Join(root, "marker"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Add(ctx, root, "marker"); err != nil {
		t.Fatalf("add: %v", err)
	}
	// Trailers covering every I2.5 key, intentionally out-of-order
	// at construction; SortedTrailers then committed.
	trailers := SortedTrailers([]Trailer{
		{Key: TrailerReason, Value: "implement E-03"},
		{Key: TrailerAuthorizedBy, Value: "4b13a0f"},
		{Key: TrailerOnBehalfOf, Value: "human/peter"},
		{Key: TrailerPrincipal, Value: "human/peter"},
		{Key: TrailerScope, Value: "opened"},
		{Key: TrailerActor, Value: "ai/claude"},
		{Key: TrailerEntity, Value: "E-03"},
		{Key: TrailerVerb, Value: "authorize"},
		{Key: TrailerScopeEnds, Value: "deadbeef"},
	})
	if err := Commit(ctx, root, "test commit", "", trailers); err != nil {
		t.Fatalf("commit: %v", err)
	}
	got, err := HeadTrailers(ctx, root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	// Every input trailer must reappear with its original value.
	want := map[string]string{
		TrailerVerb:         "authorize",
		TrailerEntity:       "E-03",
		TrailerActor:        "ai/claude",
		TrailerPrincipal:    "human/peter",
		TrailerOnBehalfOf:   "human/peter",
		TrailerAuthorizedBy: "4b13a0f",
		TrailerScope:        "opened",
		TrailerScopeEnds:    "deadbeef",
		TrailerReason:       "implement E-03",
	}
	gotByKey := make(map[string]string, len(got))
	for _, tr := range got {
		gotByKey[tr.Key] = tr.Value
	}
	for k, v := range want {
		if gotByKey[k] != v {
			t.Errorf("trailer %q: got %q, want %q", k, gotByKey[k], v)
		}
	}
	// And the canonical order survives readback (HeadTrailers parses
	// in the order git emits them, which matches the commit body).
	wantOrder := []string{
		TrailerVerb,
		TrailerEntity,
		TrailerActor,
		TrailerPrincipal,
		TrailerOnBehalfOf,
		TrailerAuthorizedBy,
		TrailerScope,
		TrailerScopeEnds,
		TrailerReason,
	}
	for i, k := range wantOrder {
		if i >= len(got) || got[i].Key != k {
			t.Errorf("position %d: got %v, want %q", i, got, k)
		}
	}
}

// TestParseTrailers_ToleratesAbsentI25Keys: pre-I2.5 commits carry
// only the original trailer set. The reader must not surface a
// finding or error for the absence of I2.5 keys; downstream callers
// (aiwf check) read a missing principal as "no principal," not as
// a parse failure.
func TestParseTrailers_ToleratesAbsentI25Keys(t *testing.T) {
	t.Parallel()
	preI25 := "aiwf-verb: promote\naiwf-entity: M-007\naiwf-actor: human/peter\n"
	got := parseTrailers(preI25)
	if len(got) != 3 {
		t.Errorf("expected 3 trailers, got %d: %+v", len(got), got)
	}
	keys := map[string]bool{}
	for _, tr := range got {
		keys[tr.Key] = true
	}
	for _, k := range []string{TrailerVerb, TrailerEntity, TrailerActor} {
		if !keys[k] {
			t.Errorf("expected %q in parsed trailers, got %v", k, keys)
		}
	}
}

// TestParseTrailers_ToleratesUnknownFutureKeys: a commit with a
// trailer key the binary doesn't know about (a future-version
// trailer landing on a developer's machine that hasn't upgraded yet)
// must parse without failing — the parser is forward-compatible.
func TestParseTrailers_ToleratesUnknownFutureKeys(t *testing.T) {
	t.Parallel()
	future := "aiwf-verb: promote\naiwf-future-key: future-value\naiwf-actor: human/peter\n"
	got := parseTrailers(future)
	if len(got) != 3 {
		t.Errorf("expected 3 trailers (unknown should still parse), got %d: %+v", len(got), got)
	}
	found := false
	for _, tr := range got {
		if tr.Key == "aiwf-future-key" && tr.Value == "future-value" {
			found = true
		}
	}
	if !found {
		t.Error("aiwf-future-key not present in parsed output")
	}
}
