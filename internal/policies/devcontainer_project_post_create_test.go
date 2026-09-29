package policies

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPolicyDevcontainerProjectPostCreate(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyDevcontainerProjectPostCreate)
}

// TestPolicyDevcontainerProjectPostCreate_Fixtures drives each rule the
// policy holds against a hook that satisfies all of them.
func TestPolicyDevcontainerProjectPostCreate_Fixtures(t *testing.T) {
	t.Parallel()
	const goYML = "      - uses: golangci/golangci-lint-action@v8\n        with:\n          version: v2.11.4\n      - run: go install golang.org/x/vuln/cmd/govulncheck@v1.6.0\n"
	good := strings.Join([]string{
		"#!/usr/bin/env bash",
		"set -euo pipefail",
		`GOLANGCI_LINT_VERSION="v2.11.4"`,
		`GOVULNCHECK_VERSION="v1.6.0"`,
		`command -v golangci-lint || install.sh | sh -s -- -b bin "${GOLANGCI_LINT_VERSION}"`,
		"command -v gofumpt || install gofumpt",
		"command -v govulncheck || go install golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION}",
		"go install ./cmd/aiwf",
		"aiwf init --no-prompt </dev/null || true",
		"make install-hooks",
		`if [[ "${AIWF_DEVCONTAINER_E2E:-false}" == "true" ]]; then npx playwright install chromium; fi`,
	}, "\n") + "\n"
	cases := []struct {
		name   string
		files  map[string]string
		firing bool
	}{
		{name: "complete-passes", firing: false, files: map[string]string{projectPostCreatePath: good, goWorkflowPath: goYML}},
		{name: "missing", firing: true, files: map[string]string{goWorkflowPath: goYML}},
		{name: "strict-mode-only-in-a-comment", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, "set -euo pipefail", "# set -euo pipefail", 1), goWorkflowPath: goYML}},
		{name: "golangci-lint-not-guarded", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, "command -v golangci-lint", "true", 1), goWorkflowPath: goYML}},
		{name: "no-gofumpt", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, "command -v gofumpt", "true", 1), goWorkflowPath: goYML}},
		{name: "govulncheck-not-guarded", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, "command -v govulncheck", "true", 1), goWorkflowPath: goYML}},
		{name: "playwright-not-gated", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, "AIWF_DEVCONTAINER_E2E:-false", "ALWAYS", 1), goWorkflowPath: goYML}},
		{name: "pin-only-in-a-comment", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, `GOLANGCI_LINT_VERSION="v2.11.4"`, "# GOLANGCI_LINT_VERSION=\"v2.11.4\"\nGOLANGCI_LINT_VERSION=\"v2.10.0\"", 1), goWorkflowPath: goYML}},
		{name: "playwright-not-installed", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, "playwright install chromium", "true", 1), goWorkflowPath: goYML}},
		{name: "no-strict-mode", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, "set -euo pipefail", "", 1), goWorkflowPath: goYML}},
		{name: "install-hooks-only-in-a-comment", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, "make install-hooks", "# make install-hooks", 1), goWorkflowPath: goYML}},
		{name: "aiwf-build-only-in-a-trailing-comment", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, "go install ./cmd/aiwf", ": # go install ./cmd/aiwf", 1), goWorkflowPath: goYML}},
		{name: "no-aiwf-build", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, "go install ./cmd/aiwf", "", 1), goWorkflowPath: goYML}},
		{name: "no-install-hooks", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, "make install-hooks", "", 1), goWorkflowPath: goYML}},
		{name: "init-reads-stdin", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, " </dev/null", "", 1), goWorkflowPath: goYML}},
		{name: "no-version-pin", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, `GOLANGCI_LINT_VERSION="v2.11.4"`, "", 1), goWorkflowPath: goYML}},
		{name: "version-drift", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, "v2.11.4", "v2.10.0", 1), goWorkflowPath: goYML}},
		{name: "golangci-lint-installed-unpinned", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, `"${GOLANGCI_LINT_VERSION}"`, "latest", 1), goWorkflowPath: goYML}},
		{name: "govulncheck-installed-unpinned", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, "govulncheck@${GOVULNCHECK_VERSION}", "govulncheck@latest", 1), goWorkflowPath: goYML}},
		{name: "govulncheck-drift", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, "v1.6.0", "v1.7.0", 1), goWorkflowPath: goYML}},
		{name: "no-govulncheck-pin", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, `GOVULNCHECK_VERSION="v1.6.0"`, "", 1), goWorkflowPath: goYML}},
		{name: "ci-has-no-govulncheck-pin", firing: true, files: map[string]string{projectPostCreatePath: good, goWorkflowPath: strings.Replace(goYML, "govulncheck@v1.6.0", "govulncheck@latest", 1)}},
		{name: "another-steps-version-is-not-the-pin", firing: true, files: map[string]string{projectPostCreatePath: good, goWorkflowPath: "      - uses: example/other-action@v1\n        with:\n          version: v2.11.4\n" + strings.Replace(goYML, "version: v2.11.4", "version: v2.12.0", 1)}},
		{name: "ci-pin-with-a-suffix", firing: true, files: map[string]string{projectPostCreatePath: good, goWorkflowPath: strings.Replace(goYML, "version: v2.11.4", "version: v2.11.4-rc1", 1)}},
		{name: "ci-has-no-version", firing: true, files: map[string]string{projectPostCreatePath: good, goWorkflowPath: "jobs: {}\n"}},
		{name: "no-ci-workflow", firing: true, files: map[string]string{projectPostCreatePath: good}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for rel, content := range tc.files {
				mustWrite(t, filepath.Join(root, rel), content)
			}
			vs, err := PolicyDevcontainerProjectPostCreate(root)
			if err != nil {
				t.Fatalf("policy returned error: %v", err)
			}
			if got := len(vs) > 0; got != tc.firing {
				t.Errorf("fired = %v, want %v; violations: %v", got, tc.firing, vs)
			}
		})
	}
}
