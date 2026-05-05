package policies

import (
	"path/filepath"
	"runtime"
	"testing"
)

// repoRoot resolves the absolute path to the repo root from this
// test file's location. Avoids relying on the test runner's cwd
// (which is the package dir) and keeps the policies invokable
// from anywhere via `go test ./tools/internal/policies/...`.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller returned ok=false")
	}
	// thisFile = .../tools/internal/policies/policies_test.go
	// repo root = ../../../..
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
}

// runPolicy is the shared scaffolding: invoke the policy, surface
// each violation as a single t.Errorf so a CI run reads as a punch
// list, not a single line dump.
func runPolicy(t *testing.T, fn func(string) ([]Violation, error)) {
	t.Helper()
	root := repoRoot(t)
	vs, err := fn(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	for _, v := range vs {
		switch {
		case v.File != "" && v.Line > 0:
			t.Errorf("[%s] %s:%d: %s", v.Policy, v.File, v.Line, v.Detail)
		case v.File != "":
			t.Errorf("[%s] %s: %s", v.Policy, v.File, v.Detail)
		default:
			t.Errorf("[%s] %s", v.Policy, v.Detail)
		}
	}
}

func TestPolicy_TrailerKeysViaConstants(t *testing.T) {
	runPolicy(t, PolicyTrailerKeysViaConstants)
}

func TestPolicy_SovereignDispatchersGuardHumanActor(t *testing.T) {
	runPolicy(t, PolicySovereignDispatchersGuardHumanActor)
}

func TestPolicy_EmptyDiffCommitsCarryMarker(t *testing.T) {
	runPolicy(t, PolicyEmptyDiffCommitsCarryMarker)
}

func TestPolicy_FindingCodesHaveHints(t *testing.T) {
	runPolicy(t, PolicyFindingCodesHaveHints)
}

func TestPolicy_ReadOnlyVerbsDoNotMutate(t *testing.T) {
	runPolicy(t, PolicyReadOnlyVerbsDoNotMutate)
}

func TestPolicy_FindingCodesAreDiscoverable(t *testing.T) {
	runPolicy(t, PolicyFindingCodesAreDiscoverable)
}

func TestPolicy_NoHistoryRewrites(t *testing.T) {
	runPolicy(t, PolicyNoHistoryRewrites)
}

func TestPolicy_NoTimestampManipulation(t *testing.T) {
	runPolicy(t, PolicyNoTimestampManipulation)
}

func TestPolicy_NoSignatureBypass(t *testing.T) {
	runPolicy(t, PolicyNoSignatureBypass)
}

func TestPolicy_NoTrailerStringComposition(t *testing.T) {
	runPolicy(t, PolicyNoTrailerStringComposition)
}

func TestPolicy_RoleIDRegexCentralized(t *testing.T) {
	runPolicy(t, PolicyRoleIDRegexCentralized)
}

func TestPolicy_PrincipalWriteSitesGuardHuman(t *testing.T) {
	runPolicy(t, PolicyPrincipalWriteSitesGuardHuman)
}

func TestPolicy_AuthorizedByWriteSitesUseAllow(t *testing.T) {
	runPolicy(t, PolicyAuthorizedByWriteSitesUseAllow)
}

func TestPolicy_ApplyCallersAcquireLock(t *testing.T) {
	runPolicy(t, PolicyApplyCallersAcquireLock)
}

func TestPolicy_VerbsValidateThenWrite(t *testing.T) {
	runPolicy(t, PolicyVerbsValidateThenWrite)
}

func TestPolicy_NoActorFieldsInAiwfYAML(t *testing.T) {
	runPolicy(t, PolicyNoActorFieldsInAiwfYAML)
}

func TestPolicy_ClosedSetStatusViaConstants(t *testing.T) {
	runPolicy(t, PolicyClosedSetStatusViaConstants)
}

func TestPolicy_NoSilentFallbacks(t *testing.T) {
	runPolicy(t, PolicyNoSilentFallbacks)
}

func TestPolicy_NoRetryLoopsOnGitErrors(t *testing.T) {
	runPolicy(t, PolicyNoRetryLoopsOnGitErrors)
}

func TestPolicy_FindingCodesHaveTests(t *testing.T) {
	runPolicy(t, PolicyFindingCodesHaveTests)
}

func TestPolicy_IntegrationTestsAssertTrailers(t *testing.T) {
	runPolicy(t, PolicyIntegrationTestsAssertTrailers)
}

func TestPolicy_DesignDocAnchors(t *testing.T) {
	runPolicy(t, PolicyDesignDocAnchors)
}

func TestPolicy_FilepathJoinSegmentBySegment(t *testing.T) {
	runPolicy(t, PolicyFilepathJoinSegmentBySegment)
}

func TestPolicy_TestsRealCloneNotUpdateRef(t *testing.T) {
	runPolicy(t, PolicyTestsRealCloneNotUpdateRef)
}

func TestPolicy_ConfigFieldsAreDiscoverable(t *testing.T) {
	runPolicy(t, PolicyConfigFieldsAreDiscoverable)
}

func TestPolicy_FSMInvariants(t *testing.T) {
	runPolicy(t, PolicyFSMInvariants)
}
