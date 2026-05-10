package check

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
	"github.com/23min/aiwf/internal/tree"
)

// TestRunProvenance_Empty asserts that an empty commit slice produces
// no findings (a fresh repo or pre-aiwf-only history is silent).
func TestRunProvenance_Empty(t *testing.T) {
	got := RunProvenance(nil, nil)
	if len(got) != 0 {
		t.Fatalf("findings = %v, want empty", got)
	}
}

// TestRunProvenance_CleanCommits asserts the loaded examples from the
// design doc don't fire any findings — every shape rule and every
// cross-commit rule has its happy path here.
func TestRunProvenance_CleanCommits(t *testing.T) {
	tr := buildProvenanceTree(t)
	authSHA := strings.Repeat("4", 40)
	commits := []scope.Commit{
		humanCommit("aaaa111", "promote", "E-0001", "human/peter", nil),
		authorizeOpenedCommit(authSHA, "E-0001", "human/peter", "ai/claude"),
		agentCommit("bbbb222", "promote", "M-0001", "ai/claude", "human/peter", authSHA, nil),
		// terminal-promote ends the scope: same commit, scope-ends
		// trailer naming the auth SHA. Within-commit equality is the
		// "auto-end" edge case — explicitly allowed by the design.
		agentCommit("cccc333", "promote", "E-0001", "ai/claude", "human/peter", authSHA, []gitops.Trailer{
			{Key: gitops.TrailerScopeEnds, Value: authSHA},
		}),
	}
	got := RunProvenance(commits, tr)
	if len(got) != 0 {
		for i := range got {
			f := &got[i]
			t.Logf("unexpected finding: %s %s — %s", f.Severity, f.Code, f.Message)
		}
		t.Fatalf("clean fixture produced %d findings, want 0", len(got))
	}
}

// TestRunProvenance_PreAiwfCommitsSilent asserts that pre-aiwf
// commits — those without any aiwf trailer — produce no findings.
// The cmd-glue grep already filters them out, but the in-package
// rules are defensive.
func TestRunProvenance_PreAiwfCommitsSilent(t *testing.T) {
	got := RunProvenance([]scope.Commit{{SHA: "abc1234"}}, nil)
	if len(got) != 0 {
		t.Fatalf("pre-aiwf commit produced %d findings, want 0", len(got))
	}
}

// TestRunProvenance_ShapeRules covers the per-trailer shape findings:
// actor-malformed, principal-non-human, on-behalf-of-non-human,
// authorized-by-malformed, force-non-human, audit-only-non-human.
func TestRunProvenance_ShapeRules(t *testing.T) {
	tests := []struct {
		name     string
		commit   scope.Commit
		wantCode string
	}{
		{
			name: "actor missing slash",
			commit: scope.Commit{SHA: "aaa1111", Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "promote"},
				{Key: gitops.TrailerEntity, Value: "E-0001"},
				{Key: gitops.TrailerActor, Value: "human-no-slash"},
			}},
			wantCode: CodeProvenanceActorMalformed,
		},
		{
			name: "principal non-human",
			commit: scope.Commit{SHA: "bbb2222", Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "promote"},
				{Key: gitops.TrailerEntity, Value: "E-0001"},
				{Key: gitops.TrailerActor, Value: "ai/claude"},
				{Key: gitops.TrailerPrincipal, Value: "ai/sub-agent"},
			}},
			wantCode: CodeProvenancePrincipalNonHuman,
		},
		{
			name: "on-behalf-of non-human",
			commit: scope.Commit{SHA: "ccc3333", Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "promote"},
				{Key: gitops.TrailerEntity, Value: "E-0001"},
				{Key: gitops.TrailerActor, Value: "ai/claude"},
				{Key: gitops.TrailerPrincipal, Value: "human/peter"},
				{Key: gitops.TrailerOnBehalfOf, Value: "ai/claude"},
				{Key: gitops.TrailerAuthorizedBy, Value: "deadbef"},
			}},
			wantCode: CodeProvenanceOnBehalfOfNonHuman,
		},
		{
			name: "authorized-by malformed",
			commit: scope.Commit{SHA: "ddd4444", Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "promote"},
				{Key: gitops.TrailerEntity, Value: "E-0001"},
				{Key: gitops.TrailerActor, Value: "ai/claude"},
				{Key: gitops.TrailerPrincipal, Value: "human/peter"},
				{Key: gitops.TrailerOnBehalfOf, Value: "human/peter"},
				{Key: gitops.TrailerAuthorizedBy, Value: "XYZ"},
			}},
			wantCode: CodeProvenanceAuthorizedByMalformed,
		},
		{
			name: "force on non-human actor",
			commit: scope.Commit{SHA: "eee5555", Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "cancel"},
				{Key: gitops.TrailerEntity, Value: "E-0001"},
				{Key: gitops.TrailerActor, Value: "ai/claude"},
				{Key: gitops.TrailerPrincipal, Value: "human/peter"},
				{Key: gitops.TrailerForce, Value: "override"},
			}},
			wantCode: CodeProvenanceForceNonHuman,
		},
		{
			name: "audit-only on non-human actor",
			commit: scope.Commit{SHA: "fff6666", Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "cancel"},
				{Key: gitops.TrailerEntity, Value: "E-0001"},
				{Key: gitops.TrailerActor, Value: "ai/claude"},
				{Key: gitops.TrailerPrincipal, Value: "human/peter"},
				{Key: gitops.TrailerAuditOnly, Value: "manual recovery"},
			}},
			wantCode: CodeProvenanceAuditOnlyNonHuman,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RunProvenance([]scope.Commit{tt.commit}, nil)
			if !hasFinding(got, tt.wantCode) {
				t.Fatalf("findings = %v, want code %q", findingCodes(got), tt.wantCode)
			}
		})
	}
}

// TestRunProvenance_CoherenceRules covers the required-together /
// mutually-exclusive incoherent-trailer findings.
func TestRunProvenance_CoherenceRules(t *testing.T) {
	tests := []struct {
		name        string
		commit      scope.Commit
		wantContain string
	}{
		{
			name: "on-behalf-of without authorized-by",
			commit: scope.Commit{SHA: "1111111", Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "promote"},
				{Key: gitops.TrailerEntity, Value: "E-0001"},
				{Key: gitops.TrailerActor, Value: "ai/claude"},
				{Key: gitops.TrailerPrincipal, Value: "human/peter"},
				{Key: gitops.TrailerOnBehalfOf, Value: "human/peter"},
			}},
			wantContain: "aiwf-on-behalf-of: present without aiwf-authorized-by:",
		},
		{
			name: "non-human actor without principal",
			commit: scope.Commit{SHA: "2222222", Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "promote"},
				{Key: gitops.TrailerEntity, Value: "E-0001"},
				{Key: gitops.TrailerActor, Value: "ai/claude"},
			}},
			wantContain: "is non-human but aiwf-principal: is missing",
		},
		{
			name: "human actor with principal",
			commit: scope.Commit{SHA: "3333333", Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "promote"},
				{Key: gitops.TrailerEntity, Value: "E-0001"},
				{Key: gitops.TrailerActor, Value: "human/peter"},
				{Key: gitops.TrailerPrincipal, Value: "human/peter"},
			}},
			wantContain: "aiwf-principal: is forbidden when aiwf-actor: is human/",
		},
		{
			name: "force with on-behalf-of",
			commit: scope.Commit{SHA: "4444444", Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "cancel"},
				{Key: gitops.TrailerEntity, Value: "E-0001"},
				{Key: gitops.TrailerActor, Value: "human/peter"},
				{Key: gitops.TrailerForce, Value: "override"},
				{Key: gitops.TrailerOnBehalfOf, Value: "human/peter"},
				{Key: gitops.TrailerAuthorizedBy, Value: "deadbef"},
			}},
			wantContain: "aiwf-force: and aiwf-on-behalf-of: are mutually exclusive",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RunProvenance([]scope.Commit{tt.commit}, nil)
			found := false
			for i := range got {
				f := &got[i]
				if f.Code == CodeProvenanceTrailerIncoherent && strings.Contains(f.Message, tt.wantContain) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("findings = %v, want incoherent message containing %q", findingCodes(got), tt.wantContain)
			}
		})
	}
}

// TestRunProvenance_NoActiveScope covers an ai/... actor with no
// on-behalf-of: the verb-time gate would refuse, but a hand-edited
// commit in history surfaces here.
func TestRunProvenance_NoActiveScope(t *testing.T) {
	c := scope.Commit{SHA: "9999999", Trailers: []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "promote"},
		{Key: gitops.TrailerEntity, Value: "E-0001"},
		{Key: gitops.TrailerActor, Value: "ai/claude"},
		{Key: gitops.TrailerPrincipal, Value: "human/peter"},
	}}
	got := RunProvenance([]scope.Commit{c}, nil)
	if !hasFinding(got, CodeProvenanceNoActiveScope) {
		t.Fatalf("findings = %v, want %q", findingCodes(got), CodeProvenanceNoActiveScope)
	}
}

// TestRunProvenance_AuthorizationMissing fires when aiwf-authorized-by
// names a SHA that isn't an authorize/opened commit in history.
func TestRunProvenance_AuthorizationMissing(t *testing.T) {
	c := scope.Commit{SHA: "aaaaaaa", Trailers: []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "promote"},
		{Key: gitops.TrailerEntity, Value: "E-0001"},
		{Key: gitops.TrailerActor, Value: "ai/claude"},
		{Key: gitops.TrailerPrincipal, Value: "human/peter"},
		{Key: gitops.TrailerOnBehalfOf, Value: "human/peter"},
		{Key: gitops.TrailerAuthorizedBy, Value: strings.Repeat("0", 40)},
	}}
	got := RunProvenance([]scope.Commit{c}, nil)
	if !hasFinding(got, CodeProvenanceAuthorizationMissing) {
		t.Fatalf("findings = %v, want %q", findingCodes(got), CodeProvenanceAuthorizationMissing)
	}
}

// TestRunProvenance_AuthorizationEnded fires when a verb references
// a scope after a prior commit ended it.
func TestRunProvenance_AuthorizationEnded(t *testing.T) {
	tr := buildProvenanceTree(t)
	authSHA := strings.Repeat("a", 40)
	commits := []scope.Commit{
		authorizeOpenedCommit(authSHA, "E-0001", "human/peter", "ai/claude"),
		// First scoped commit — fine.
		agentCommit("bbbb222", "promote", "M-0001", "ai/claude", "human/peter", authSHA, nil),
		// Terminal-promote ends the scope (same commit). Allowed.
		agentCommit("cccc333", "promote", "E-0001", "ai/claude", "human/peter", authSHA, []gitops.Trailer{
			{Key: gitops.TrailerScopeEnds, Value: authSHA},
		}),
		// LATE commit references the now-ended scope.
		agentCommit("dddd444", "promote", "M-0002", "ai/claude", "human/peter", authSHA, nil),
	}
	got := RunProvenance(commits, tr)
	if !hasFinding(got, CodeProvenanceAuthorizationEnded) {
		t.Fatalf("findings = %v, want %q", findingCodes(got), CodeProvenanceAuthorizationEnded)
	}
}

// TestRunProvenance_AuthorizationOutOfScope fires when scope-entity
// has no reference path to the verb's target.
func TestRunProvenance_AuthorizationOutOfScope(t *testing.T) {
	tr := buildProvenanceTree(t)
	authSHA := strings.Repeat("a", 40)
	commits := []scope.Commit{
		// Scope opened against the unrelated epic E-09.
		authorizeOpenedCommit(authSHA, "E-0009", "human/peter", "ai/claude"),
		// Agent acts on M-001, which is under E-01 — out of scope.
		agentCommit("bbbb222", "promote", "M-0001", "ai/claude", "human/peter", authSHA, nil),
	}
	got := RunProvenance(commits, tr)
	if !hasFinding(got, CodeProvenanceAuthorizationOutOfScope) {
		t.Fatalf("findings = %v, want %q", findingCodes(got), CodeProvenanceAuthorizationOutOfScope)
	}
}

// TestRunProvenance_PriorEntityChainResolves fires when a scope was
// opened against an entity that has since been reallocated; the
// rename-chain walker should resolve the scope-entity to its current
// id, and the rule should NOT fire.
func TestRunProvenance_PriorEntityChainResolves(t *testing.T) {
	tr := buildProvenanceTree(t)
	authSHA := strings.Repeat("b", 40)
	// E-07 was reallocated to E-01 (the live entity in the fixture).
	// The scope was opened against E-07; the agent now operates on
	// M-001 under E-01.
	commits := []scope.Commit{
		authorizeOpenedCommit(authSHA, "E-0007", "human/peter", "ai/claude"),
		// Reallocate commit: prior=E-07, new=E-01.
		{
			SHA: "9999991",
			Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "reallocate"},
				{Key: gitops.TrailerEntity, Value: "E-0001"},
				{Key: gitops.TrailerPriorEntity, Value: "E-0007"},
				{Key: gitops.TrailerActor, Value: "human/peter"},
			},
		},
		// Agent acts on M-001 (under E-01, which used to be E-07).
		agentCommit("bbbb222", "promote", "M-0001", "ai/claude", "human/peter", authSHA, nil),
	}
	got := RunProvenance(commits, tr)
	if hasFinding(got, CodeProvenanceAuthorizationOutOfScope) {
		t.Fatalf("findings = %v, did not expect out-of-scope after rename chain", findingCodes(got))
	}
}

// TestRunProvenance_AuthorizeCommitNoActiveScopeSkipped asserts that
// an authorize commit (which itself doesn't operate inside a scope)
// is exempt from the no-active-scope rule. The kernel reserves the
// authorize+on-behalf-of question to G22.
func TestRunProvenance_AuthorizeCommitNoActiveScopeSkipped(t *testing.T) {
	c := scope.Commit{SHA: "1234567", Trailers: []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "authorize"},
		{Key: gitops.TrailerEntity, Value: "E-0001"},
		{Key: gitops.TrailerActor, Value: "human/peter"},
		{Key: gitops.TrailerTo, Value: "ai/claude"},
		{Key: gitops.TrailerScope, Value: "opened"},
	}}
	got := RunProvenance([]scope.Commit{c}, nil)
	if hasFinding(got, CodeProvenanceNoActiveScope) {
		t.Fatalf("authorize commit fired no-active-scope: %v", findingCodes(got))
	}
}

// TestRunUntrailedAudit_AuditOnlyClearsWarning covers the
// "warning clears on the next push" promise: a manual commit
// touching an entity file is followed in the same range by an
// audit-only commit on that entity. The warning is suppressed.
func TestRunUntrailedAudit_AuditOnlyClearsWarning(t *testing.T) {
	commits := []UntrailedCommit{
		{
			SHA:   "manual1",
			Paths: []string{"work/gaps/G-001-leak.md"},
		},
		{
			SHA: "audit01",
			Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "cancel"},
				{Key: gitops.TrailerEntity, Value: "G-0001"},
				{Key: gitops.TrailerAuditOnly, Value: "manual flip recovery"},
			},
		},
	}
	got := RunUntrailedAudit(commits)
	if len(got) != 0 {
		t.Errorf("warning should clear after audit-only on G-001; got %v", findingCodes(got))
	}
}

// TestRunUntrailedAudit_AuditOnlyDoesNotCoverPriorManual covers the
// ordering rule: an audit-only commit BEFORE a manual commit does
// not retroactively cover later manual commits. The warning fires.
func TestRunUntrailedAudit_AuditOnlyBeforeManualStillFires(t *testing.T) {
	commits := []UntrailedCommit{
		{
			SHA: "audit01",
			Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "cancel"},
				{Key: gitops.TrailerEntity, Value: "G-0001"},
				{Key: gitops.TrailerAuditOnly, Value: "earlier recovery"},
			},
		},
		{
			SHA:   "manual1",
			Paths: []string{"work/gaps/G-001-leak.md"},
		},
	}
	got := RunUntrailedAudit(commits)
	if !hasFinding(got, CodeProvenanceUntrailedEntityCommit) {
		t.Errorf("manual commit AFTER audit-only must still fire; got %v", findingCodes(got))
	}
}

// TestRunUntrailedAudit_AuditOnlyOnDifferentEntityDoesNotCover
// covers the per-entity matching: an audit-only on G-001 does not
// suppress a manual commit on G-002.
func TestRunUntrailedAudit_AuditOnlyOnDifferentEntityDoesNotCover(t *testing.T) {
	commits := []UntrailedCommit{
		{
			SHA:   "manual1",
			Paths: []string{"work/gaps/G-002-other.md"},
		},
		{
			SHA: "audit01",
			Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "cancel"},
				{Key: gitops.TrailerEntity, Value: "G-0001"},
				{Key: gitops.TrailerAuditOnly, Value: "different entity"},
			},
		},
	}
	got := RunUntrailedAudit(commits)
	if !hasFinding(got, CodeProvenanceUntrailedEntityCommit) {
		t.Errorf("audit-only on G-001 must not cover G-002; got %v", findingCodes(got))
	}
}

// TestRunUntrailedAudit_SquashMergeSubcode covers G31: when the
// offending untrailered commit's subject matches GitHub's default
// squash-merge pattern (ends in ` (#NNN)`), the finding fires
// with subcode `squash-merge`. A subject without that suffix
// produces the bare code only.
func TestRunUntrailedAudit_SquashMergeSubcode(t *testing.T) {
	tests := []struct {
		name        string
		subject     string
		wantSubcode string
	}{
		{"github default", "feat(api): add caching (#42)", "squash-merge"},
		{"plain hand-edit", "manual: flip G-001 wontfix", ""},
		{"prose with parens", "fix bug (the second one)", ""},
		{"single-digit PR", "chore: minor tweak (#1)", "squash-merge"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RunUntrailedAudit([]UntrailedCommit{
				{
					SHA:     "abc1234",
					Subject: tt.subject,
					Paths:   []string{"work/gaps/G-001-leak.md"},
				},
			})
			if len(got) != 1 {
				t.Fatalf("findings = %d, want 1; got %v", len(got), got)
			}
			if got[0].Code != CodeProvenanceUntrailedEntityCommit {
				t.Errorf("code = %q, want %q", got[0].Code, CodeProvenanceUntrailedEntityCommit)
			}
			if got[0].Subcode != tt.wantSubcode {
				t.Errorf("subcode = %q, want %q", got[0].Subcode, tt.wantSubcode)
			}
		})
	}
}

// TestRunUntrailedAudit_PerEntityFindings: a single manual commit
// touching three entity files emits three findings, one per
// entity. Each finding carries the entity id; messages are short
// (no embedded path list).
func TestRunUntrailedAudit_PerEntityFindings(t *testing.T) {
	commits := []UntrailedCommit{
		{
			SHA: "manual1",
			Paths: []string{
				"work/gaps/G-001-leak.md",
				"work/gaps/G-002-other.md",
				"work/decisions/D-005-yaml.md",
			},
		},
	}
	got := RunUntrailedAudit(commits)
	if len(got) != 3 {
		t.Fatalf("findings = %d, want 3 (one per entity); got %v", len(got), got)
	}
	wantIDs := map[string]bool{"G-0001": true, "G-0002": true, "D-0005": true}
	for _, f := range got {
		if f.Code != CodeProvenanceUntrailedEntityCommit {
			t.Errorf("code = %q, want %q", f.Code, CodeProvenanceUntrailedEntityCommit)
		}
		if !wantIDs[f.EntityID] {
			t.Errorf("EntityID = %q, want one of G-001/G-002/D-005", f.EntityID)
		}
		if strings.Contains(f.Message, ",") || strings.Contains(f.Message, "entity files") {
			t.Errorf("per-entity message should be short, no path list; got %q", f.Message)
		}
	}
}

// TestRunUntrailedAudit_AuditOnlyClearsPerEntity: a manual commit
// touches G-001 and G-002; a later audit-only commit covers only
// G-001. Result: one finding remaining (G-002), not zero, not two.
// This is the load-bearing fix from issue #5 sub-item 1 — the
// previous all-or-nothing suppression left BOTH warnings flagged
// in this scenario.
func TestRunUntrailedAudit_AuditOnlyClearsPerEntity(t *testing.T) {
	commits := []UntrailedCommit{
		{
			SHA: "manual1",
			Paths: []string{
				"work/gaps/G-001-leak.md",
				"work/gaps/G-002-other.md",
			},
		},
		{
			SHA: "audit01",
			Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "cancel"},
				{Key: gitops.TrailerEntity, Value: "G-0001"},
				{Key: gitops.TrailerAuditOnly, Value: "manual flip recovery"},
			},
		},
	}
	got := RunUntrailedAudit(commits)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1 (only uncovered entity); got %v", len(got), got)
	}
	if got[0].EntityID != "G-0002" {
		t.Errorf("remaining finding EntityID = %q, want G-002", got[0].EntityID)
	}
}

// TestRunUntrailedAudit_CompositeAuditCoversParentManual: an
// `aiwf <verb> --audit-only` on `M-001/AC-1` rolls up to M-001 for
// matching, so a manual mutation of the M-001 file before it is
// covered.
func TestRunUntrailedAudit_CompositeAuditCoversParentManual(t *testing.T) {
	commits := []UntrailedCommit{
		{
			SHA:   "manual1",
			Paths: []string{"work/epics/E-01-foo/M-001-cache.md"},
		},
		{
			SHA: "audit01",
			Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "promote"},
				{Key: gitops.TrailerEntity, Value: "M-0001/AC-1"},
				{Key: gitops.TrailerAuditOnly, Value: "AC backfill"},
			},
		},
	}
	got := RunUntrailedAudit(commits)
	if len(got) != 0 {
		t.Errorf("composite-id audit-only on M-001/AC-1 should cover manual on M-001; got %v", findingCodes(got))
	}
}

// TestRunUntrailedAudit covers step 7b: a commit touching entity
// files without an aiwf-verb: trailer should fire one warning. A
// commit with the trailer is silent; a commit touching only
// non-entity files is silent.
func TestRunUntrailedAudit(t *testing.T) {
	tests := []struct {
		name      string
		commits   []UntrailedCommit
		wantCount int
	}{
		{
			name: "manual commit touching milestone fires",
			commits: []UntrailedCommit{
				{
					SHA:   "abc1234",
					Paths: []string{"work/epics/E-01-foo/M-001-bar.md"},
				},
			},
			wantCount: 1,
		},
		{
			name: "trailered verb commit is silent",
			commits: []UntrailedCommit{
				{
					SHA: "def5678",
					Trailers: []gitops.Trailer{
						{Key: gitops.TrailerVerb, Value: "promote"},
						{Key: gitops.TrailerEntity, Value: "M-0001"},
					},
					Paths: []string{"work/epics/E-01-foo/M-001-bar.md"},
				},
			},
			wantCount: 0,
		},
		{
			name: "non-entity-only commit is silent",
			commits: []UntrailedCommit{
				{
					SHA:   "999aaaa",
					Paths: []string{"STATUS.md", "aiwf.yaml", ".claude/skills/x.md"},
				},
			},
			wantCount: 0,
		},
		{
			name: "mixed commit fires once for the entity path",
			commits: []UntrailedCommit{
				{
					SHA:   "555ccc1",
					Paths: []string{"STATUS.md", "work/gaps/G-001-leak.md"},
				},
			},
			wantCount: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RunUntrailedAudit(tt.commits)
			if len(got) != tt.wantCount {
				t.Fatalf("findings = %d (%v), want %d", len(got), findingCodes(got), tt.wantCount)
			}
			if tt.wantCount == 1 && got[0].Code != CodeProvenanceUntrailedEntityCommit {
				t.Errorf("finding code = %q, want %q", got[0].Code, CodeProvenanceUntrailedEntityCommit)
			}
			if tt.wantCount == 1 && got[0].Severity != SeverityWarning {
				t.Errorf("severity = %q, want %q", got[0].Severity, SeverityWarning)
			}
		})
	}
}

// TestShaOK covers the boundary cases of the SHA-shape predicate.
// 7..40 hex passes; everything else fails. Lowercase only.
func TestShaOK(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"", false},
		{"abcdef", false}, // 6, too short
		{"abcdef0", true}, // 7, the floor
		{"4b13a0fdeadbeefcafebabefeedface000000000", true},   // 40, the ceiling
		{"4b13a0fdeadbeefcafebabefeedface0000000001", false}, // 41, too long
		{"ABCDEF7", false},  // uppercase rejected
		{"4b13a0g", false},  // non-hex char
		{"4b13 a0f", false}, // whitespace
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := shaOK(tt.s); got != tt.want {
				t.Errorf("shaOK(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

// TestRoleIDOK covers the role/id shape predicate (mirrors the
// ValidateTrailer regex, exposed here for the in-package shape
// checks).
func TestRoleIDOK(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"", false},
		{"human/peter", true},
		{"ai/claude", true},
		{"bot/ci", true},
		{"human", false},        // no slash
		{"/peter", false},       // empty role
		{"human/", false},       // empty id
		{"a/b/c", false},        // two slashes
		{"human peter", false},  // no slash
		{"human /peter", false}, // whitespace
		{"human\tpeter", false}, // tab
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := roleIDOK(tt.s); got != tt.want {
				t.Errorf("roleIDOK(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

// TestWalkRenameChain covers three branches:
//   - empty input passes through;
//   - a healthy chain walks forward to the terminal id;
//   - a cycle (defensive — corrupted history) is broken by the
//     visit-once guard, returning the current position rather than
//     looping.
func TestWalkRenameChain(t *testing.T) {
	t.Run("empty id", func(t *testing.T) {
		if got := walkRenameChain("", map[string]string{"E-0001": "E-0002"}); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
	t.Run("healthy chain", func(t *testing.T) {
		chain := map[string]string{
			"E-0001": "E-0002",
			"E-0002": "E-0003",
		}
		if got := walkRenameChain("E-0001", chain); got != "E-0003" {
			t.Errorf("got %q, want E-03", got)
		}
	})
	t.Run("not in chain", func(t *testing.T) {
		if got := walkRenameChain("E-0099", map[string]string{"E-0001": "E-0002"}); got != "E-0099" {
			t.Errorf("got %q, want E-99 (unchanged)", got)
		}
	})
	t.Run("cycle broken", func(t *testing.T) {
		// E-01 → E-02 → E-01 (corrupted; should not loop forever).
		chain := map[string]string{
			"E-0001": "E-0002",
			"E-0002": "E-0001",
		}
		got := walkRenameChain("E-0001", chain)
		// The cycle guard returns the latest unvisited position
		// before the cycle closes. Either E-01 or E-02 is acceptable
		// (the implementation visits E-01 first, walks to E-02, then
		// E-02→E-01 hits the visited set and returns E-02).
		if got != "E-0002" && got != "E-0001" {
			t.Errorf("got %q, want E-01 or E-02 (cycle broken without looping)", got)
		}
	})
}

// TestResolveAuthSHA_AmbiguousPrefix: when a (rare) short SHA
// prefix matches more than one opener, the resolver returns
// (nil, false) so the standing rule fires authorization-missing
// rather than picking one silently. The full SHA path is unaffected.
func TestResolveAuthSHA_AmbiguousPrefix(t *testing.T) {
	full1 := strings.Repeat("a", 40)
	full2 := "a" + strings.Repeat("b", 39)
	authIndex := map[string]*scope.Commit{
		full1: {SHA: full1},
		full2: {SHA: full2},
	}
	t.Run("exact full SHA wins", func(t *testing.T) {
		got, ok := resolveAuthSHA(full1, authIndex)
		if !ok || got == nil || got.SHA != full1 {
			t.Errorf("exact lookup failed: ok=%v sha=%v", ok, got)
		}
	})
	t.Run("ambiguous prefix returns missing", func(t *testing.T) {
		// "a" alone matches both keys.
		got, ok := resolveAuthSHA("a", authIndex)
		if ok || got != nil {
			t.Errorf("ambiguous prefix resolved silently: ok=%v sha=%v", ok, got)
		}
	})
	t.Run("unique prefix resolves", func(t *testing.T) {
		// "ab" matches only full2.
		got, ok := resolveAuthSHA("ab", authIndex)
		if !ok || got == nil || got.SHA != full2 {
			t.Errorf("unique prefix lookup failed: ok=%v sha=%v", ok, got)
		}
	})
	t.Run("no match", func(t *testing.T) {
		got, ok := resolveAuthSHA("c", authIndex)
		if ok || got != nil {
			t.Errorf("no-match returned: ok=%v sha=%v", ok, got)
		}
	})
}

// TestRunProvenance_CompositeTargetRollsUp covers the
// out-of-scope rule's composite-id rollup: a target like
// `M-001/AC-1` rolls up to `M-001` for reachability. When the
// scope is on the parent epic and the target is a composite under a
// child milestone, the rule must NOT fire (M-001 reaches E-01
// via parent).
func TestRunProvenance_CompositeTargetRollsUp(t *testing.T) {
	tr := buildProvenanceTree(t)
	authSHA := strings.Repeat("c", 40)
	commits := []scope.Commit{
		authorizeOpenedCommit(authSHA, "E-0001", "human/peter", "ai/claude"),
		// Agent acts on M-001/AC-1 (composite). Rolls up to M-001;
		// M-001 reaches E-01 (parent). No out-of-scope finding.
		agentCommit("bbbb222", "promote", "M-0001/AC-1",
			"ai/claude", "human/peter", authSHA, nil),
	}
	got := RunProvenance(commits, tr)
	if hasFinding(got, CodeProvenanceAuthorizationOutOfScope) {
		t.Fatalf("out-of-scope fired on composite target that rolls up correctly: %v", findingCodes(got))
	}
}

// TestRunProvenance_SelfReferentialOutOfScope covers the
// short-circuit branch where target == scope-entity (after composite
// rollup). The reachability check is skipped via `from == to`.
func TestRunProvenance_SelfReferentialOutOfScope(t *testing.T) {
	tr := buildProvenanceTree(t)
	authSHA := strings.Repeat("d", 40)
	commits := []scope.Commit{
		authorizeOpenedCommit(authSHA, "E-0001", "human/peter", "ai/claude"),
		// Agent acts on E-01 itself. target == scope-entity.
		agentCommit("bbbb222", "promote", "E-0001",
			"ai/claude", "human/peter", authSHA, nil),
	}
	got := RunProvenance(commits, tr)
	if hasFinding(got, CodeProvenanceAuthorizationOutOfScope) {
		t.Fatalf("out-of-scope fired on self-referential target: %v", findingCodes(got))
	}
}

// TestRunProvenance_MultipleAuthorizedByLastWins documents the
// behavior when a commit (defensively / pathologically) carries two
// `aiwf-authorized-by:` trailers: the indexCommitTrailers helper is
// last-wins, so only the LAST trailer drives the rules. This test
// pins that behavior so a future change to the indexer surfaces
// here.
func TestRunProvenance_MultipleAuthorizedByLastWins(t *testing.T) {
	tr := buildProvenanceTree(t)
	goodSHA := strings.Repeat("a", 40)
	missingSHA := strings.Repeat("0", 40)
	commits := []scope.Commit{
		authorizeOpenedCommit(goodSHA, "E-0001", "human/peter", "ai/claude"),
		// Two authorized-by trailers: first valid, second missing.
		// Last-wins → -authorization-missing fires.
		{
			SHA: "bbbb222",
			Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "promote"},
				{Key: gitops.TrailerEntity, Value: "M-0001"},
				{Key: gitops.TrailerActor, Value: "ai/claude"},
				{Key: gitops.TrailerPrincipal, Value: "human/peter"},
				{Key: gitops.TrailerOnBehalfOf, Value: "human/peter"},
				{Key: gitops.TrailerAuthorizedBy, Value: goodSHA},
				{Key: gitops.TrailerAuthorizedBy, Value: missingSHA},
			},
		},
	}
	got := RunProvenance(commits, tr)
	if !hasFinding(got, CodeProvenanceAuthorizationMissing) {
		t.Fatalf("expected last-wins to drive -authorization-missing on the second SHA; got %v",
			findingCodes(got))
	}
}

// hasFinding reports whether any finding has the given code.
func hasFinding(fs []Finding, code string) bool {
	for i := range fs {
		if fs[i].Code == code {
			return true
		}
	}
	return false
}

func findingCodes(fs []Finding) []string {
	out := make([]string, len(fs))
	for i := range fs {
		out[i] = string(fs[i].Severity) + " " + fs[i].Code
	}
	return out
}

// buildProvenanceTree mirrors buildAllowTree from verb/allow_test.go
// but lives here so the check package can stay self-contained.
func buildProvenanceTree(t *testing.T) *tree.Tree {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"work/epics/E-01-platform/epic.md": "---\nid: E-01\ntitle: Platform\nstatus: active\n---\n",
		"work/epics/E-01-platform/M-001-cache.md": "---\nid: M-001\ntitle: Cache warmup\n" +
			"status: in_progress\nparent: E-01\n---\n",
		"work/epics/E-01-platform/M-002-evict.md": "---\nid: M-002\ntitle: Eviction policy\n" +
			"status: draft\nparent: E-01\n---\n",
		"work/epics/E-09-unrelated/epic.md": "---\nid: E-09\ntitle: Unrelated\nstatus: proposed\n---\n",
	}
	for relPath, content := range files {
		full := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	return tr
}

// humanCommit builds a direct-human commit (single actor, no
// principal/scope trailers).
func humanCommit(sha, verb, entityID, actor string, extra []gitops.Trailer) scope.Commit {
	trailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: verb},
		{Key: gitops.TrailerEntity, Value: entityID},
		{Key: gitops.TrailerActor, Value: actor},
	}
	trailers = append(trailers, extra...)
	return scope.Commit{SHA: sha, Trailers: trailers}
}

// authorizeOpenedCommit builds an authorize/opened commit.
func authorizeOpenedCommit(sha, entityID, principal, agent string) scope.Commit {
	return scope.Commit{
		SHA: sha,
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "authorize"},
			{Key: gitops.TrailerEntity, Value: entityID},
			{Key: gitops.TrailerActor, Value: principal},
			{Key: gitops.TrailerTo, Value: agent},
			{Key: gitops.TrailerScope, Value: "opened"},
		},
	}
}

// agentCommit builds an agent-acting-in-scope commit.
func agentCommit(sha, verb, entityID, actor, principal, authSHA string, extra []gitops.Trailer) scope.Commit {
	trailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: verb},
		{Key: gitops.TrailerEntity, Value: entityID},
		{Key: gitops.TrailerActor, Value: actor},
		{Key: gitops.TrailerPrincipal, Value: principal},
		{Key: gitops.TrailerOnBehalfOf, Value: principal},
		{Key: gitops.TrailerAuthorizedBy, Value: authSHA},
	}
	trailers = append(trailers, extra...)
	return scope.Commit{SHA: sha, Trailers: trailers}
}
