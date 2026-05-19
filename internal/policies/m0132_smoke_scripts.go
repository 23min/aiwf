package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PolicyM0132SmokeScripts asserts the two operator-run smoke scripts
// exist, are executable, and have the agreed structural shape.
// Pins M-0132/AC-7 (build smoke) and M-0132/AC-8 (ci smoke).
//
// Operator-run today; CI integration (Docker-in-Docker matrix run)
// is a sibling milestone under E-0035. The structural shape is the
// mechanical chokepoint that keeps the scripts honest until CI runs
// them: each script names its operator-run status in its header
// comment, performs the preflight check for the devcontainer CLI,
// runs the right primary command (`devcontainer build` /
// `devcontainer exec ... make ci`), and verifies its claim (Go
// version grep for AC-7; exit-code propagation for AC-8).
type smokeScriptSpec struct {
	relPath        string
	ac             string
	needles        []string
	needlesDetails map[string]string
}

func PolicyM0132SmokeScripts(root string) ([]Violation, error) {
	specs := []smokeScriptSpec{
		{
			relPath: "scripts/devcontainer-build-smoke.sh",
			ac:      "AC-7",
			needles: []string{
				"Operator-run",
				"command -v devcontainer",
				"devcontainer build",
				"go1.25",
			},
			needlesDetails: map[string]string{
				"Operator-run":            "header comment must name the operator-run status (so AC-7's deferred CI integration is discoverable)",
				"command -v devcontainer": "preflight check for `devcontainer` CLI presence (so a missing-CLI failure is one-line clear, not a build error halfway through)",
				"devcontainer build":      "must invoke `devcontainer build` against the workspace-folder",
				"go1.25":                  "must verify the built image runs go1.25 (grepping the output of `go version`) — without this the AC asserts 'a build ran' not 'the right thing was built'",
			},
		},
		{
			relPath: "scripts/devcontainer-ci-smoke.sh",
			ac:      "AC-8",
			needles: []string{
				"Operator-run",
				"command -v devcontainer",
				"docker info",
				"devcontainer up",
				"devcontainer exec",
				"make ci",
			},
			needlesDetails: map[string]string{
				"Operator-run":            "header comment must name the operator-run status (so AC-8's deferred CI integration is discoverable)",
				"command -v devcontainer": "preflight check for `devcontainer` CLI presence",
				"docker info":             "preflight check for Docker daemon reachability (so a stopped-Docker failure is one-line clear)",
				"devcontainer up":         "must bring the container up via `devcontainer up`",
				"devcontainer exec":       "must run the in-container command via `devcontainer exec`",
				"make ci":                 "must invoke `make ci` (the full vet + lint + test-race + coverage + selfcheck chain)",
			},
		},
	}

	var vs []Violation
	for _, s := range specs {
		vs = append(vs, checkSmokeScript(root, s)...)
	}
	return vs, nil
}

func checkSmokeScript(root string, s smokeScriptSpec) []Violation {
	abs := filepath.Join(root, s.relPath)
	info, err := os.Stat(abs)
	if err != nil {
		return []Violation{{
			Policy: "m0132-smoke-scripts",
			File:   s.relPath,
			Detail: fmt.Sprintf("[%s] missing or unreadable: %v", s.ac, err),
		}}
	}

	raw, err := os.ReadFile(abs)
	if err != nil {
		return []Violation{{
			Policy: "m0132-smoke-scripts",
			File:   s.relPath,
			Detail: fmt.Sprintf("[%s] ReadFile failed: %v", s.ac, err),
		}}
	}
	content := string(raw)

	var vs []Violation
	report := func(detail string) {
		vs = append(vs, Violation{
			Policy: "m0132-smoke-scripts",
			File:   s.relPath,
			Detail: fmt.Sprintf("[%s] %s", s.ac, detail),
		})
	}

	if info.Mode().Perm() != 0o755 {
		report(fmt.Sprintf("mode = %#o, want 0755 (chmod +x %s)", info.Mode().Perm(), s.relPath))
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

	// Sorted iteration for stable failure output.
	keys := make([]string, 0, len(s.needles))
	keys = append(keys, s.needles...)
	sort.Strings(keys)
	for _, n := range keys {
		if !strings.Contains(content, n) {
			report(fmt.Sprintf("missing required marker %q: %s", n, s.needlesDetails[n]))
		}
	}

	return vs
}
