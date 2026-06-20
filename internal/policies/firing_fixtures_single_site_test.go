package policies

import (
	"path/filepath"
	"testing"
)

// hasPolicyViolation reports whether vs contains a violation stamped id.
func hasPolicyViolation(vs []Violation, id string) bool {
	for _, v := range vs {
		if v.Policy == id {
			return true
		}
	}
	return false
}

// discoverabilityScaffold returns the minimal tree the discoverability
// policies need to run without erroring: readDiscoverabilityChannels reads
// cmd/aiwf/main.go + CLAUDE.md and walks internal/skills/embedded +
// docs/pocv3, so all four must exist. None of them mention the crafted
// tag/code, so it stays out of the haystack and the policy fires.
func discoverabilityScaffold() map[string]string {
	return map[string]string{
		"cmd/aiwf/main.go":              "package main\n\nfunc main() {}\n",
		"CLAUDE.md":                     "# fixture\n\nnothing relevant here\n",
		"internal/skills/embedded/x.md": "nothing relevant\n",
		"docs/pocv3/x.md":               "nothing relevant\n",
	}
}

// withTrigger returns base plus one extra file (the policy trigger).
func withTrigger(base map[string]string, rel, content string) map[string]string {
	base[rel] = content
	return base
}

// TestFiringFixtures_SingleSite is the G-0262 burn-down positive control
// for the single-dark-site policies (M-0166/AC-1). Each row builds a
// synthetic temp root whose crafted file(s) trip exactly one policy, calls
// the policy, and asserts it returns >=1 Violation stamped with its id —
// covering the policy's previously-dark Violation construction line so the
// firing-fixture-presence meta-gate has evidence the policy can fire.
func TestFiringFixtures_SingleSite(t *testing.T) {
	t.Parallel()

	// internal/config struct field with a yaml tag absent from every doc
	// channel. Built with a real backtick struct tag (collectConfigYAMLTags
	// trims backticks and parses via reflect.StructTag).
	bt := "`"
	configGoSrc := "package config\n\ntype C struct {\n\tKnob string " +
		bt + "yaml:\"madeupknob\"" + bt + "\n}\n"

	cases := []struct {
		id     string
		policy func(string) ([]Violation, error)
		files  map[string]string
	}{
		{
			id:     "apply-callers-acquire-lock",
			policy: PolicyApplyCallersAcquireLock,
			files:  map[string]string{"cmd/aiwf/x.go": "package main\n\nfunc runFoo() { verb.Apply() }\n"},
		},
		{
			id:     "authorized-by-via-allow",
			policy: PolicyAuthorizedByWriteSitesUseAllow,
			files:  map[string]string{"internal/foo/a.go": "package foo\n\nfunc F() { _ = T{Key: gitops.TrailerAuthorizedBy} }\n"},
		},
		{
			id:     "claude-md-test-discipline-section",
			policy: PolicyClaudeMdTestDisciplineSection,
			files:  map[string]string{"CLAUDE.md": "# X\n\n## Go conventions\n\nno test-discipline subsection here\n"},
		},
		{
			id:     "cli-helper-locations",
			policy: PolicyCLIHelperLocations,
			files:  map[string]string{"cmd/aiwf/h.go": "package main\n\nfunc resolveRoot() {}\n"},
		},
		{
			id:     "closed-set-status-via-constants",
			policy: PolicyClosedSetStatusViaConstants,
			files:  map[string]string{"internal/foo/s.go": "package foo\n\nvar _ = T{Status: \"active\"}\n"},
		},
		{
			id:     "config-fields-discoverable",
			policy: PolicyConfigFieldsAreDiscoverable,
			files:  withTrigger(discoverabilityScaffold(), "internal/config/config.go", configGoSrc),
		},
		{
			id:     "empty-diff-commits-carry-marker",
			policy: PolicyEmptyDiffCommitsCarryMarker,
			files:  map[string]string{"internal/verb/e.go": "package verb\n\nvar _ = Plan{AllowEmpty: true}\n"},
		},
		{
			id:     "finding-codes-are-discoverable",
			policy: PolicyFindingCodesAreDiscoverable,
			files:  withTrigger(discoverabilityScaffold(), "internal/check/x.go", "package check\n\nvar _ = Finding{Code: \"zzz-madeup-code\"}\n"),
		},
		{
			id:     "finding-codes-have-hints",
			policy: PolicyFindingCodesHaveHints,
			files:  map[string]string{"internal/check/h.go": "package check\n\nvar _ = Finding{Code: \"zzz-nonexistent-code\"}\n"},
		},
		{
			id:     "integration-tests-assert-trailers",
			policy: PolicyIntegrationTestsAssertTrailers,
			files:  map[string]string{"cmd/aiwf/i_test.go": "package main\n\nfunc TestX(t *testing.T) { runBin(\"promote\", \"M-1\") }\n"},
		},
		{
			id:     "no-actor-fields-in-aiwfyaml",
			policy: PolicyNoActorFieldsInAiwfYAML,
			files:  map[string]string{"internal/aiwfyaml/c.go": "package aiwfyaml\n\ntype C struct{ Actor string }\n"},
		},
		{
			id:     "no-hardcoded-entity-paths",
			policy: PolicyNoHardcodedEntityPaths,
			files:  map[string]string{"internal/policies/fixture_x.go": "package policies\n\nimport \"path/filepath\"\n\nvar _ = filepath.Join(rootDir, \"E-0001-foo\")\n"},
		},
		{
			id:     "no-history-rewrites",
			policy: PolicyNoHistoryRewrites,
			files:  map[string]string{"internal/foo/h.go": "package foo\n\nvar _ = []string{\"rebase\"}\n"},
		},
		{
			id:     "no-retry-loops-on-git-errors",
			policy: PolicyNoRetryLoopsOnGitErrors,
			files:  map[string]string{"internal/foo/r.go": "package foo\n\nfunc f() {\n\tfor {\n\t\t_ = exec.Command(\"git\", \"status\")\n\t}\n}\n"},
		},
		{
			id:     "no-signature-bypass",
			policy: PolicyNoSignatureBypass,
			files:  map[string]string{"internal/foo/s.go": "package foo\n\nvar _ = []string{\"--no-verify\"}\n"},
		},
		{
			id:     "no-silent-fallback",
			policy: PolicyNoSilentFallbacks,
			files:  map[string]string{"internal/foo/sw.go": "package foo\n\nfunc f(e E) {\n\tswitch e.Kind {\n\tcase \"a\":\n\t\t_ = 1\n\t}\n}\n"},
		},
		{
			id:     "no-timestamp-manipulation",
			policy: PolicyNoTimestampManipulation,
			files:  map[string]string{"internal/foo/t.go": "package foo\n\nvar _ = []string{\"GIT_AUTHOR_DATE=x\"}\n"},
		},
		{
			id:     "no-trailer-string-composition",
			policy: PolicyNoTrailerStringComposition,
			files:  map[string]string{"internal/foo/tc.go": "package foo\n\nimport \"fmt\"\n\nvar _ = fmt.Sprintf(\"aiwf-verb: %s\", \"x\")\n"},
		},
		{
			id:     "principal-write-sites-guard-human",
			policy: PolicyPrincipalWriteSitesGuardHuman,
			files:  map[string]string{"internal/foo/p.go": "package foo\n\nfunc F() { _ = T{Key: gitops.TrailerPrincipal} }\n"},
		},
		{
			id:     "role-id-regex-centralized",
			policy: PolicyRoleIDRegexCentralized,
			files:  map[string]string{"internal/foo/role.go": "package foo\n\nimport \"regexp\"\n\nvar _ = regexp.MustCompile(\"^[^/]+/[^/]+/.+\")\n"},
		},
		{
			id:     "sovereign-dispatchers-guard-human-actor",
			policy: PolicySovereignDispatchersGuardHumanActor,
			files:  map[string]string{"cmd/aiwf/sv.go": "package main\n\nfunc runThing() {\n\t_ = \"force\"\n\t_ = \"reason\"\n}\n"},
		},
		{
			id:     "tests-real-clone-not-update-ref",
			policy: PolicyTestsRealCloneNotUpdateRef,
			files:  map[string]string{"cmd/aiwf/c_test.go": "package main\n\nfunc TestX() { _ = \"git update-ref refs/remotes/origin/main\" }\n"},
		},
		{
			id:     "trailer-keys-via-constants",
			policy: PolicyTrailerKeysViaConstants,
			// Empty root → no internal/gitops/trailers.go constants →
			// the defensive "no constants found" construction line fires.
			files: map[string]string{},
		},
		{
			id:     "trailer-parser-uniqueness",
			policy: PolicyTrailerParserUniqueness,
			files:  map[string]string{"internal/foo/parse.go": "package foo\n\nfunc ParseTrailers() {}\n"},
		},
		{
			id:     "verbs-validate-then-write",
			policy: PolicyVerbsValidateThenWrite,
			files:  map[string]string{"internal/verb/v.go": "package verb\n\nfunc Foo() { os.WriteFile() }\n"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for rel, content := range tc.files {
				mustWrite(t, filepath.Join(root, rel), content)
			}
			vs, err := tc.policy(root)
			if err != nil {
				t.Fatalf("%s: policy returned error: %v", tc.id, err)
			}
			if !hasPolicyViolation(vs, tc.id) {
				t.Errorf("%s: policy did not fire on its fixture; got %d violations: %+v", tc.id, len(vs), vs)
			}
		})
	}
}
