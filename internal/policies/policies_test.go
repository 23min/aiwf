package policies

import (
	"path/filepath"
	"runtime"
	"testing"
)

// repoRoot resolves the absolute path to the repo root from this
// test file's location. Avoids relying on the test runner's cwd
// (which is the package dir) and keeps the policies invokable
// from anywhere via `go test ./internal/policies/...`.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller returned ok=false")
	}
	// thisFile = .../internal/policies/policies_test.go
	// repo root = ../..
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
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
	t.Parallel()
	runPolicy(t, PolicyTrailerKeysViaConstants)
}

func TestPolicy_SovereignDispatchersGuardHumanActor(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicySovereignDispatchersGuardHumanActor)
}

func TestPolicy_EmptyDiffCommitsCarryMarker(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyEmptyDiffCommitsCarryMarker)
}

func TestPolicy_FindingCodesHaveHints(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyFindingCodesHaveHints)
}

func TestPolicy_ReadOnlyVerbsDoNotMutate(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyReadOnlyVerbsDoNotMutate)
}

func TestPolicy_FindingCodesAreDiscoverable(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyFindingCodesAreDiscoverable)
}

func TestPolicy_SkillCoverageMatchesVerbs(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicySkillCoverageMatchesVerbs)
}

func TestPolicy_NoHistoryRewrites(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyNoHistoryRewrites)
}

func TestPolicy_NoTimestampManipulation(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyNoTimestampManipulation)
}

func TestPolicy_NoSignatureBypass(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyNoSignatureBypass)
}

func TestPolicy_NoTrailerStringComposition(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyNoTrailerStringComposition)
}

func TestPolicy_RoleIDRegexCentralized(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyRoleIDRegexCentralized)
}

func TestPolicy_PrincipalWriteSitesGuardHuman(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyPrincipalWriteSitesGuardHuman)
}

func TestPolicy_AuthorizedByWriteSitesUseAllow(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyAuthorizedByWriteSitesUseAllow)
}

func TestPolicy_ApplyCallersAcquireLock(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyApplyCallersAcquireLock)
}

func TestPolicy_VerbsValidateThenWrite(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyVerbsValidateThenWrite)
}

func TestPolicy_NoActorFieldsInAiwfYAML(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyNoActorFieldsInAiwfYAML)
}

func TestPolicy_ClosedSetStatusViaConstants(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyClosedSetStatusViaConstants)
}

func TestPolicy_NoSilentFallbacks(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyNoSilentFallbacks)
}

func TestPolicy_NoRetryLoopsOnGitErrors(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyNoRetryLoopsOnGitErrors)
}

func TestPolicy_FindingCodesHaveTests(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyFindingCodesHaveTests)
}

func TestPolicy_IntegrationTestsAssertTrailers(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyIntegrationTestsAssertTrailers)
}

func TestPolicy_DesignDocAnchors(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyDesignDocAnchors)
}

func TestPolicy_FilepathJoinSegmentBySegment(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyFilepathJoinSegmentBySegment)
}

func TestPolicy_NoHardcodedEntityPaths(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyNoHardcodedEntityPaths)
}

func TestPolicy_TestsRealCloneNotUpdateRef(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyTestsRealCloneNotUpdateRef)
}

func TestPolicy_ConfigFieldsAreDiscoverable(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyConfigFieldsAreDiscoverable)
}

func TestPolicy_FSMInvariants(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyFSMInvariants)
}
