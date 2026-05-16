package main

import (
	"os"
	"testing"
)

// TestMain seeds GIT identity env vars once for the test binary's
// lifetime. os.Setenv (not t.Setenv) because t.Setenv panics under
// t.Parallel; the values are immutable for the lifetime of the test
// binary, so once-setup is correct.
//
// Per M-0092 the cmd/aiwf/ package follows the same pattern landed
// for internal/* in M-0091. Every Test* function not on the skip-list
// below calls t.Parallel() as its first statement; table-driven
// t.Run subtests nest a second t.Parallel() inside the closure where
// the iteration is independent.
//
// Serial tests in this package (cannot run under t.Parallel because
// they mutate process-level state or saturate shared resources):
//
// integration_g37_test.go — entire file (11 tests):
//   - TestIntegrationG37_AllocatorSkipsTrunkAfterFetch
//   - TestIntegrationG37_DivergedBranchesCaughtByCheckPostFetch
//   - TestIntegrationG37_NoRemoteSkipsSilently
//   - TestIntegrationG37_MixedKindsAcrossTrunk
//   - TestIntegrationG37_ReallocateUsesTrunkView
//   - TestIntegrationG37_CleanFetchAndMergeRoundTrip
//   - TestIntegrationG37_ReallocateRewritesRefsAndHistoryThreads
//   - TestIntegrationG37_ReallocateTiebreakerPicksLocalSide
//   - TestIntegrationG37_ReallocateTiebreakerAmbiguousNeitherInTrunk
//   - TestIntegrationG37_HistoryWalksLineageChain
//   - TestIntegrationG37_PrePushHookCatchesCollision
//     Rationale: dense subprocess fan-out — each test forks a bare
//     origin plus N clones plus per-clone git processes. Running 11
//     in parallel risks fd-table / process-table saturation on
//     macOS hosts. Topology-sharing across these tests is deferred
//     per the E-0025 epic.
//
// Tests that mutate os.Stdout / os.Stderr through the captureStdout,
// captureStderr, or captureRun helpers (helpers_test.go,
// contract_cmd_test.go, upgrade_cmd_test.go):
//   - TestRun_CheckShapeOnly_* (4 in check_shape_only_test.go) — all
//     use captureStdout.
//   - TestCheck_ArchiveSweepThreshold_MessageNamesThresholdAndCount
//     — captureStdout for output assertion.
//   - TestRun_ContractVerify*, TestRun_CheckIncludesContractFindings,
//     TestRun_CheckSkipsTerminalContracts,
//     TestRun_ContractRecipeInstallIsIdempotent,
//     TestRun_ContractRecipeRemoveRefusesWhenBindingExists
//     (contract_cmd_test.go) — all capture stdout/stderr.
//   - TestEnvelopeSchemaConformance_AllJSONVerbs
//     (envelope_schema_test.go) — captureStdout per verb subtest.
//   - TestRun_HistoryJSON, TestRun_HistoryTextOutputIncludesForceLine
//     (history_cmd_test.go) — captureStdout.
//   - TestRun_ImportThroughDispatcher, TestRun_ImportDryRun
//     (import_cmd_test.go) — captureStdout.
//   - TestRun_InitDryRun, TestRun_InitSkipHook,
//     TestRun_InitMigratesAlienHook (init_cmd_test.go) —
//     captureStdout.
//   - TestRun_List_CoreFlagsEndToEnd,
//     TestRun_List_JSONResultIsArrayOfSummaryObjects,
//     TestRun_List_ArchivedFlag (list_cmd_test.go) — captureStdout.
//   - TestRun_SubverbHelpDoesNotRecurse (main_test.go) —
//     captureStdout.
//   - TestRun_RenderRoadmap_Stdout, TestRun_RenderRoadmap_EmptyRepo
//     (render_cmd_test.go) — captureStdout.
//   - TestRun_DoctorReportsRenderConfig,
//     TestRun_DoctorDetectsCommitOutputDrift (render_doctor_test.go)
//     — captureStdout.
//   - TestRun_RenderHTML_* (4 in render_gitignore_warning_test.go) —
//     captureStdout.
//   - TestRun_RenderHTML_DispatchesToSite,
//     TestRun_RenderHTML_HonorsAiwfYAMLOutDir,
//     TestRun_Render_DispatcherDistinguishesSubcommandFromFormat,
//     TestRun_Render_HelpFlag (render_site_cmd_test.go) —
//     captureStdout.
//   - TestPrintRitualsSuggestion_ContainsKeyLines,
//     TestPrintRitualsSuggestion_DoesNotRecommendCLIInstallForm
//     (rituals_test.go) — captureStdout.
//   - TestRunSchema_AllKindsText, TestRunSchema_OneKindText,
//     TestRunSchema_JSONEnvelope, TestRunSchema_JSONOneKind
//     (schema_cmd_test.go) — captureStdout.
//   - Most TestRun_Show* (show_cmd_test.go) — captureStdout.
//   - TestRenderStatusText_*, TestRenderStatusMarkdown_*,
//     TestRunStatusCmd_SweepPendingSeam (status_cmd_test.go) —
//     captureStdout.
//   - TestRunTemplate_OneKindRaw, TestRunTemplate_AllKindsHasHeaders,
//     TestRunTemplate_JSONEnvelope, TestRunTemplate_JSONOneKind
//     (template_cmd_test.go) — captureStdout.
//   - TestRunCheck_TestsMetricsWarningSurfacesViaDispatcher
//     (tests_metrics_check_test.go) — captureStdout.
//   - TestRunUpgrade_* (most in upgrade_cmd_test.go) — captureRun
//     mutates os.Stdout AND os.Stderr.
//   - TestRunWhoami_FromFlag, TestRunWhoami_LegacyConfigActorIgnored,
//     TestRunWhoami_FromGitConfig, TestRunWhoami_NoActorAvailable
//     (whoami_cmd_test.go) — captureStdout plus t.Setenv(HOME, …).
//
// Tests that call t.Setenv (panics under t.Parallel):
//   - TestResolveActor_LegacyConfigActorIgnored,
//     TestResolveActor_DerivedFromGitConfig,
//     TestResolveActor_FlagOverridesGitConfig,
//     TestResolveActor_MalformedGitEmail,
//     TestResolveActor_NoConfigErrors (actor_test.go) — t.Setenv
//     HOME / XDG_CONFIG_HOME / GIT_CONFIG_NOSYSTEM to isolate
//     git env.
//   - TestDoctor_CheckLatest_ProxyDisabled,
//     TestDoctorReport_RecommendedPlugins_* (doctor_cmd_test.go) —
//     t.Setenv HOME to point at fixture plugin index.
//
// Tests that call os.Chdir (process-wide cwd mutation):
//   - TestCompleteEntityIDs_FromTree,
//     TestCompleteEntityIDs_GracefulNoOp,
//     TestCompleteEntityIDArg_RespectsPosition,
//     TestCompleteEntityIDFlag_KindFilter (completion_helpers_test.go)
//     — local chdir helper restores via t.Cleanup, but the in-flight
//     window races concurrent tests.
//   - TestRunWhoami_FromGitConfig, TestRunWhoami_NoActorAvailable
//     (whoami_cmd_test.go) — os.Chdir into t.TempDir() to ensure
//     resolveActor reads the test's gitconfig fixture.
//
// Subtests:
//   - TestActorPattern (actor_test.go),
//     TestResolveActor_ExplicitInvalid (actor_test.go),
//     TestReorderFlagsFirst (flags_test.go),
//     TestStripTrailers (strip_trailers_test.go), and
//     TestHistory_NarrowTrailerMatchesCanonicalQuery
//     (canonicalize_history_test.go) opt their table-driven t.Run
//     subtests into a second t.Parallel() because each iteration is
//     independent. The pure-function tests trivially qualify; the
//     canonicalize-history test's per-case t.Run builds its own
//     initTrailerRepo(). Other table-driven tests in this package
//     could likewise be opted in; this milestone applies it to the
//     pure-function / fixture-isolated cases that lit up clean in
//     the -race -parallel 8 -count=3 audit.
//
// Notes on cmd/aiwf/-specific helpers:
//   - The captureStdout/captureStderr/captureRun helpers mutate
//     os.Stdout / os.Stderr — package-level fds shared by every
//     goroutine. Any test calling them must stay serial; the helpers
//     themselves are NOT being refactored under this milestone
//     (no test-semantics change per the M-0092 constraint set).
//   - integration_g37_test.go subprocess fan-out + the M-0091
//     `runBin` subprocess helper continue to exec the real binary
//     for end-to-end coverage; the binary build is shared via
//     `sync.Once` in integration_test.go::aiwfBinary so the per-test
//     compile cost is paid once per test-binary lifetime.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	os.Exit(m.Run())
}
