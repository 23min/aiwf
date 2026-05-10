package main

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/gitops"
)

// AC-4 in M-081: pre-existing narrow-width trailers (`aiwf-entity:
// E-22`) match canonical-id queries (`aiwf history E-0022`) and vice
// versa. The kernel never rewrites old commit history; the read
// path is the chokepoint for backward compatibility.
//
// The fixture builds a real git repo by hand, lands a commit with a
// known-shape trailer, and queries via the same readHistoryChain
// the verb consumes.

// commitWithTrailer fabricates one commit on the repo at root whose
// message body carries `aiwf-entity: <id>` (and a verb / actor pair
// so the readHistoryChain trailer-pair filter doesn't drop the row
// as a wrapped-prose false positive — see admin_cmd.go's "skip
// prose-mention false-positives" branch).
func commitWithTrailer(t *testing.T, root, id, verb string) {
	t.Helper()
	msg := "test commit\n\n" +
		"aiwf-verb: " + verb + "\n" +
		"aiwf-entity: " + id + "\n" +
		"aiwf-actor: human/test\n"
	// Empty allow because we just need any tree change.
	for _, args := range [][]string{
		{"commit", "--allow-empty", "-m", msg},
	} {
		cmd := exec.Command("git", args...) //nolint:gosec // test-only
		cmd.Dir = root
		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=aiwf-test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=aiwf-test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

// initTrailerRepo bootstraps a fresh git repo with one initial commit.
func initTrailerRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := gitops.Init(context.Background(), root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	t.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	commitWithTrailer(t, root, "bootstrap", "init")
	return root
}

// TestHistory_NarrowTrailerMatchesCanonicalQuery is the AC-4
// load-bearing assertion. A commit lands with an `aiwf-entity: E-22`
// trailer; queries for both `E-22` and `E-0022` return the same
// commit. Mirrored across the entity kinds the kernel allocates:
// per-kind grammar floors come from internal/entity/entity.go::idPatterns.
func TestHistory_NarrowTrailerMatchesCanonicalQuery(t *testing.T) {
	tests := []struct {
		name    string
		stored  string
		queries []string // both queries must return the commit
	}{
		{"epic-narrow-stored", "E-22", []string{"E-22", "E-0022"}},
		{"epic-canonical-stored", "E-0022", []string{"E-22", "E-0022"}},
		{"milestone-narrow-stored", "M-007", []string{"M-007", "M-0007"}},
		{"milestone-canonical-stored", "M-0007", []string{"M-007", "M-0007"}},
		{"gap-narrow-stored", "G-093", []string{"G-093", "G-0093"}},
		{"decision-narrow-stored", "D-005", []string{"D-005", "D-0005"}},
		{"contract-narrow-stored", "C-009", []string{"C-009", "C-0009"}},
		// ADR is exempt: its grammar floor (\d{4,} per idPatterns) was
		// always the canonical width, so there is no narrow legacy
		// shape to test. ADR-0001 → ADR-0001 only.
		{"adr-canonical", "ADR-0001", []string{"ADR-0001"}},
		{"composite-narrow", "M-007/AC-1", []string{"M-007/AC-1", "M-0007/AC-1"}},
		{"composite-canonical", "M-0007/AC-1", []string{"M-007/AC-1", "M-0007/AC-1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := initTrailerRepo(t)
			commitWithTrailer(t, root, tt.stored, "promote")
			for _, q := range tt.queries {
				events, err := readHistoryChain(context.Background(), root, []string{q})
				if err != nil {
					t.Fatalf("readHistoryChain(%q): %v", q, err)
				}
				if len(events) == 0 {
					t.Errorf("query %q against trailer %q returned no events", q, tt.stored)
					continue
				}
				// At least one event must reflect the test commit
				// (verb=promote). The bootstrap commit emits
				// verb=init.
				found := false
				for _, e := range events {
					if e.Verb == "promote" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("query %q matched events but none with verb=promote: %+v", q, events)
				}
			}
		})
	}
}

// TestHistory_NewVerbsEmitCanonicalTrailers complements AC-4: the
// kernel never writes narrow-width trailers in new commits. AC-1's
// allocator change combined with verb-level emission means a fresh
// `add` writes a canonical id; this test exercises the verb-level
// commit path end-to-end and inspects HEAD's trailers.
func TestHistory_NewVerbsEmitCanonicalTrailers(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)

	trailers, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	var entityVal string
	for _, tr := range trailers {
		if tr.Key == gitops.TrailerEntity {
			entityVal = tr.Value
			break
		}
	}
	if entityVal == "" {
		t.Fatalf("no aiwf-entity trailer on HEAD: %+v", trailers)
	}
	// Canonical width for epic is E-NNNN (4 digits).
	if !strings.HasPrefix(entityVal, "E-") {
		t.Fatalf("trailer entity = %q, want E- prefix", entityVal)
	}
	digits := entityVal[len("E-"):]
	if len(digits) < 4 {
		t.Errorf("entity trailer %q is narrow-width; want canonical (>=4 digits) per AC-1", entityVal)
	}
}
