package policies

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestGocriticFilepathJoinConfigured is the mechanical evidence behind
// M-0167/AC-2: the bespoke `filepath-join-segment-by-segment` policy was
// deleted because gocritic's `filepathJoin` checker is a clean superset
// (D-0025). This test fails if the repo's golangci-lint config stops
// keeping that checker active — so the delete cannot silently lose the
// coverage the bespoke policy provided.
//
// Why these three assertions are sufficient (verified empirically with
// golangci-lint v2.12.2, the toolchain this repo pins): `filepathJoin`
// fires only when (a) gocritic is enabled, (b) the `diagnostic` tag is in
// gocritic's enabled-tags — the checker is dark under gocritic's default
// check set, and the `diagnostic` tag is what lights it (the `experimental`
// tag also does, but the repo does not enable it) — and (c) the checker is
// not listed in disabled-checks. Assert all three structurally over the
// parsed YAML, not as substring matches over the raw bytes.
//
// Residual gap (closed by M-0170): this is a *config* guard, not an
// *execution* guard. If a future gocritic re-tags `filepathJoin` out of
// `diagnostic`, this test still passes while the checker silently stops
// firing — the structural-vs-firing gap the G-0264 epic is about. M-0170
// adds the golangci-lint *execution* firing test (a fixture run through
// golangci-lint) that closes it; per E-0042 planning this test is the
// structural first step and M-0170 generalizes execution across gocritic
// and the dormant forbidigo rules.
func TestGocriticFilepathJoinConfigured(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, ".golangci.yml"))
	if err != nil {
		t.Fatalf("read .golangci.yml: %v", err)
	}

	var cfg struct {
		Linters struct {
			Enable   []string `yaml:"enable"`
			Settings struct {
				Gocritic struct {
					EnabledTags    []string `yaml:"enabled-tags"`
					DisabledChecks []string `yaml:"disabled-checks"`
				} `yaml:"gocritic"`
			} `yaml:"settings"`
		} `yaml:"linters"`
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parse .golangci.yml: %v", err)
	}

	gc := cfg.Linters.Settings.Gocritic
	if !slices.Contains(cfg.Linters.Enable, "gocritic") {
		t.Errorf("gocritic is not in linters.enable — filepathJoin cannot fire, and the deleted filepath-join-segment-by-segment policy no longer covers embedded-separator filepath.Join args (D-0025/M-0167)")
	}
	if !slices.Contains(gc.EnabledTags, "diagnostic") {
		t.Errorf("gocritic enabled-tags %v omits %q — filepathJoin is dark under gocritic's default checks and needs the diagnostic tag; the deleted bespoke policy no longer backstops it (D-0025/M-0167)", gc.EnabledTags, "diagnostic")
	}
	if slices.Contains(gc.DisabledChecks, "filepathJoin") {
		t.Errorf("gocritic disabled-checks lists filepathJoin — the checker the filepath-join-segment-by-segment delete relies on is turned off (D-0025/M-0167)")
	}
}
