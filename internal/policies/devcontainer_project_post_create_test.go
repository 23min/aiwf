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
	const goYML = "      - uses: golangci/golangci-lint-action@v8\n        with:\n          version: v2.11.4\n"
	good := strings.Join([]string{
		"#!/usr/bin/env bash",
		"set -euo pipefail",
		`GOLANGCI_LINT_VERSION="v2.11.4"`,
		"command -v golangci-lint || install golangci-lint",
		"command -v gofumpt || install gofumpt",
		"command -v govulncheck || install govulncheck",
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
		{name: "no-strict-mode", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, "set -euo pipefail", "", 1), goWorkflowPath: goYML}},
		{name: "init-reads-stdin", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, " </dev/null", "", 1), goWorkflowPath: goYML}},
		{name: "no-version-pin", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, `GOLANGCI_LINT_VERSION="v2.11.4"`, "", 1), goWorkflowPath: goYML}},
		{name: "version-drift", firing: true, files: map[string]string{projectPostCreatePath: strings.Replace(good, "v2.11.4", "v2.10.0", 1), goWorkflowPath: goYML}},
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
