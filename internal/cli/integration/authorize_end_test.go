package integration

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// showScopes reads `aiwf show <id> --format=json` and returns the
// entity's status alongside its scope rows.
//
// It goes through the binary's own projection rather than replaying the
// trailers here, because the claim under test is about what the replay
// reports. A test that re-derived the state from the commit it just
// wrote would agree with itself no matter what ReplayScopes did with
// the trailer.
func showScopes(t *testing.T, root, binDir, id string) (status string, scopes []struct {
	AuthSHA string `json:"auth_sha"`
	State   string `json:"state"`
},
) {
	t.Helper()
	out, err := testutil.RunBin(t, root, binDir, nil, "show", id, "--format=json")
	if err != nil {
		t.Fatalf("aiwf show %s: %v\n%s", id, err, out)
	}
	var env struct {
		Result struct {
			Status string `json:"status"`
			Scopes []struct {
				AuthSHA string `json:"auth_sha"`
				State   string `json:"state"`
			} `json:"scopes"`
		} `json:"result"`
	}
	if jsonErr := json.Unmarshal([]byte(out), &env); jsonErr != nil {
		t.Fatalf("parsing aiwf show %s JSON: %v\n%s", id, jsonErr, out)
	}
	return env.Result.Status, env.Result.Scopes
}

// TestAuthorizeEnd_EndsScope_LeavesEntityStatusUnchanged is M-0325/AC-1.
//
// Both assertions carry the AC together. An end that worked only by
// moving the entity to a terminal status would be the automatic end this
// milestone exists to complement, so the unchanged status is what
// distinguishes the new capability from the one already shipped.
//
// The fixture's epic is promoted to `active` rather than left at
// `proposed`: the point is a delegation withdrawn from an entity that
// keeps living, and an epic still in planning would satisfy the
// unchanged-status assertion without exercising that case.
func TestAuthorizeEnd_EndsScope_LeavesEntityStatusUnchanged(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	root, binDir := sovereignScopedRepo(t, [][]string{{"promote", "E-0001", "active"}})

	statusBefore, before := showScopes(t, root, binDir, "E-0001")
	if len(before) != 1 {
		t.Fatalf("fixture carries %d scopes, want exactly 1 — the sole-candidate default is what "+
			"this case exercises: %+v", len(before), before)
	}
	if before[0].State != "active" {
		t.Fatalf("fixture scope is %q, want active", before[0].State)
	}

	out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--end", "--reason", "delegation withdrawn; taking it back in-loop")
	if err != nil {
		t.Fatalf("aiwf authorize --end: %v\n%s", err, out)
	}

	statusAfter, after := showScopes(t, root, binDir, "E-0001")
	if len(after) != 1 {
		t.Fatalf("scope count changed to %d; an end records a transition, it does not add or drop a "+
			"scope: %+v", len(after), after)
	}
	if after[0].State != "ended" {
		t.Errorf("scope %s replays as %q, want \"ended\"", after[0].AuthSHA[:8], after[0].State)
	}
	if statusAfter != statusBefore {
		t.Errorf("E-0001 status moved %q -> %q; an operator end must leave the entity's status alone",
			statusBefore, statusAfter)
	}
}

// TestAuthorizeEnd_WritesTheFullAuthSHA is M-0325/AC-1's silent-failure
// guard.
//
// ReplayScopes resolves `aiwf-scope-ends:` by exact lookup against the
// full 40-char SHA, while gitops.ValidateTrailer accepts 7-40 hex for
// that key. A truncated value therefore passes validation, lands a
// commit, and leaves `aiwf check` green while the scope stays open — the
// failure is invisible at every surface except the replayed state.
//
// The assertion above would catch a truncation today, but only because
// the operator supplied no `--scope`. This one pins the trailer itself
// on the path where an abbreviation actually arrives: the operator names
// the scope by the 7-char prefix `aiwf show` prints, and the emitted
// trailer must still be the full SHA.
func TestAuthorizeEnd_WritesTheFullAuthSHA(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	root, binDir := sovereignScopedRepo(t, nil)
	_, scopes := showScopes(t, root, binDir, "E-0001")
	if len(scopes) != 1 {
		t.Fatalf("fixture carries %d scopes, want 1: %+v", len(scopes), scopes)
	}
	full := scopes[0].AuthSHA

	out, err := testutil.RunBin(t, root, binDir, nil,
		"authorize", "E-0001", "--end", "--scope", full[:7], "--reason", "named by the prefix show prints")
	if err != nil {
		t.Fatalf("aiwf authorize --end --scope <prefix>: %v\n%s", err, out)
	}

	trailers, gitErr := testutil.RunGit(root, "log", "-1", "--pretty=%(trailers:key=aiwf-scope-ends,valueonly=true)")
	if gitErr != nil {
		t.Fatalf("git log: %v\n%s", gitErr, trailers)
	}
	if got := strings.TrimSpace(trailers); got != full {
		t.Errorf("aiwf-scope-ends = %q, want the full auth SHA %q — a truncated value validates and "+
			"commits, but ReplayScopes never matches it, so the scope silently stays open", got, full)
	}

	if _, after := showScopes(t, root, binDir, "E-0001"); after[0].State != "ended" {
		t.Errorf("scope replays as %q after an end naming its prefix, want \"ended\"", after[0].State)
	}
}
