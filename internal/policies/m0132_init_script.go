package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// PolicyM0132InitScript asserts that .devcontainer/init.sh is the
// in-container postCreateCommand hook, has mode 0755, carries the
// bash header, contains each agreed install/config section with
// idempotency guards, and the post-install banner contains the four
// canonical literal strings the operator needs to install the
// rituals plugins at PROJECT scope.
//
// Cross-file consistency: the golangci-lint version pinned in
// init.sh must match the version pinned in .github/workflows/go.yml.
// CI is the source of truth; init.sh adopts.
//
// Pins M-0132/AC-4. The banner is the chokepoint that surfaces the
// one manual step claude-code#31388 + the CLI-form's USER-scope
// default forces us into; per Q1 of the design conversation, the
// banner is the document-and-trust answer (option 1).
func PolicyM0132InitScript(root string) ([]Violation, error) {
	const relPath = ".devcontainer/init.sh"
	abs := filepath.Join(root, relPath)

	info, err := os.Stat(abs)
	if err != nil {
		return []Violation{{
			Policy: "m0132-init-script",
			File:   relPath,
			Detail: fmt.Sprintf("missing or unreadable: %v", err),
		}}, nil
	}

	raw, err := os.ReadFile(abs)
	if err != nil {
		return []Violation{{
			Policy: "m0132-init-script",
			File:   relPath,
			Detail: fmt.Sprintf("ReadFile failed: %v", err),
		}}, nil
	}
	content := string(raw)

	var vs []Violation
	report := func(detail string) {
		vs = append(vs, Violation{
			Policy: "m0132-init-script",
			File:   relPath,
			Detail: detail,
		})
	}

	if info.Mode().Perm() != 0o755 {
		report(fmt.Sprintf("mode = %#o, want 0755 (chmod +x .devcontainer/init.sh)", info.Mode().Perm()))
	}

	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "#!/usr/bin/env bash" {
		first := ""
		if len(lines) > 0 {
			first = lines[0]
		}
		report(fmt.Sprintf("first line = %q, want \"#!/usr/bin/env bash\"", first))
	}

	if !strings.Contains(content, "set -euo pipefail") {
		report("missing `set -euo pipefail` directive")
	}

	// Each named capability must appear, with at least one idempotency
	// guard (command -v) where the install is expensive enough to
	// matter on rebuild.
	type checkSpec struct {
		name    string
		needles []string // all must appear
		detail  string
	}
	checks := []checkSpec{
		{
			name:    "git config",
			needles: []string{`git config --global user.name`, `git config --global user.email`},
			detail:  "must set user.name and user.email globally (match host identity for aiwf trailer consistency)",
		},
		{
			name:    "gh credential helper",
			needles: []string{`!gh auth git-credential`},
			detail:  "gh credential helper rewrite missing (per Liminara's precedent; uses bare `gh` so $PATH resolution works)",
		},
		{
			name:    "golangci-lint",
			needles: []string{`golangci-lint`, `command -v golangci-lint`},
			detail:  "golangci-lint install missing or not idempotent (must guard with `command -v golangci-lint`)",
		},
		{
			name:    "gofumpt",
			needles: []string{`gofumpt`, `command -v gofumpt`},
			detail:  "gofumpt install missing or not idempotent (must guard with `command -v gofumpt`)",
		},
		{
			name:    "govulncheck",
			needles: []string{`govulncheck`, `command -v govulncheck`},
			detail:  "govulncheck install missing or not idempotent (must guard with `command -v govulncheck`)",
		},
		{
			name:    "Claude Code CLI",
			needles: []string{`claude.ai/install.sh`, `command -v claude`},
			detail:  "Claude Code CLI install missing or not idempotent (must curl the native install.sh and guard with `command -v claude`)",
		},
		{
			name:    "aiwf binary",
			needles: []string{`go install ./cmd/aiwf`, `aiwf init`},
			detail:  "aiwf install + init missing (go install ./cmd/aiwf followed by aiwf init to materialize framework hooks)",
		},
		{
			name:    "make install-hooks",
			needles: []string{`make install-hooks`},
			detail:  "kernel pre-commit chain not installed (make install-hooks symlinks scripts/git-hooks/pre-commit into the chain)",
		},
		{
			name:    "Playwright env-gate",
			needles: []string{`AIWF_DEVCONTAINER_E2E:-false`, `playwright install chromium`},
			detail:  "Playwright install must be env-gated on AIWF_DEVCONTAINER_E2E with default `false` so opt-in is explicit",
		},
	}
	for _, c := range checks {
		for _, n := range c.needles {
			if !strings.Contains(content, n) {
				report(fmt.Sprintf("section %q: %s (looked for literal %q)", c.name, c.detail, n))
				break
			}
		}
	}

	// Banner block — the four canonical literal strings the operator
	// needs to install the rituals plugins correctly. Each must
	// appear at least once in the file (the cat <<'BANNER' heredoc
	// is contiguous; if any is missing, the banner is wrong).
	bannerLiterals := []string{
		"23min/ai-workflow-rituals",
		"PROJECT scope",
		"aiwf-extensions",
		"wf-rituals",
	}
	var missingBanner []string
	for _, lit := range bannerLiterals {
		if !strings.Contains(content, lit) {
			missingBanner = append(missingBanner, lit)
		}
	}
	if len(missingBanner) > 0 {
		sort.Strings(missingBanner)
		report(fmt.Sprintf("banner block missing canonical literal(s): %s", strings.Join(missingBanner, ", ")))
	}

	// Cross-file consistency: golangci-lint version pinned here must
	// match the version pinned in .github/workflows/go.yml. CI is
	// the source of truth.
	verPattern := regexp.MustCompile(`GOLANGCI_LINT_VERSION="?(v\d+\.\d+\.\d+)"?`)
	m := verPattern.FindStringSubmatch(content)
	if m == nil {
		report("can't extract GOLANGCI_LINT_VERSION assignment (expected `GOLANGCI_LINT_VERSION=\"vX.Y.Z\"`)")
	} else {
		initVer := m[1]
		ciVer, ciErr := extractGolangciVersionFromCI(root)
		switch {
		case ciErr != nil:
			report(fmt.Sprintf("can't extract golangci-lint version from .github/workflows/go.yml: %v", ciErr))
		case ciVer != initVer:
			report(fmt.Sprintf("golangci-lint version drift: init.sh pins %q, .github/workflows/go.yml pins %q — CI is the source of truth, bump init.sh", initVer, ciVer))
		}
	}

	return vs, nil
}

// extractGolangciVersionFromCI scans .github/workflows/go.yml for a
// `version: vX.Y.Z` line. The go-version key uses bare numeric tags
// (1.25.10) without a `v` prefix, so a regex requiring a leading `v`
// uniquely targets the golangci-lint-action's version pin.
func extractGolangciVersionFromCI(root string) (string, error) {
	const relGoYml = ".github/workflows/go.yml"
	raw, err := os.ReadFile(filepath.Join(root, relGoYml))
	if err != nil {
		return "", err
	}
	pat := regexp.MustCompile(`(?m)^\s*version:\s*"?(v\d+\.\d+\.\d+)"?\s*$`)
	for _, line := range strings.Split(string(raw), "\n") {
		if m := pat.FindStringSubmatch(line); m != nil {
			return m[1], nil
		}
	}
	return "", fmt.Errorf("no `version: vX.Y.Z` line found in %s", relGoYml)
}
