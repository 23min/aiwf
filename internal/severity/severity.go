// Package severity applies the aiwf.yaml-configured severity
// escalations to findings a check pass has already produced.
//
// The check rules stay config-agnostic: each emits at its default
// severity, and `internal/check` carries one small Apply* function per
// knob that raises a named code set to error. This package is the
// single place those passes are composed, so every surface that reads
// findings agrees with `aiwf check` about how severe they are.
//
// Composing them anywhere else holds nothing in agreement: a pass added
// outside this seam reaches only the call sites its author edits, and
// each surface then reports its own severity for the same finding on
// the same tree. PolicySeverityPassComposition is the mechanical half
// of the guarantee; Apply is the seam it holds every call site to.
//
// The package sits beside internal/check rather than inside it because
// the escalation is config-driven and internal/check deliberately
// carries no aiwf.yaml type (the M-0171/AC-4 boundary that keeps the
// rules themselves free of configuration).
package severity

import (
	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/tree"
)

// Policy is the set of aiwf.yaml knobs that raise a finding's severity,
// projected off config.Config so callers pass one value instead of
// re-deriving four arguments each.
type Policy struct {
	// TDDStrict is `tdd.strict`: raises entity-body-empty and
	// milestone-tdd-undeclared.
	TDDStrict bool
	// AreasRequired is `areas.required`: raises the area-axis findings.
	AreasRequired bool
	// DocsStrict is `docs.strict`: raises the doc id rules.
	DocsStrict bool
	// ArchiveThreshold and ArchiveThresholdSet are
	// `archive.sweep_threshold`: the aggregate archive-sweep-pending
	// finding blocks once the pending count strictly exceeds the
	// ceiling. Unset (the default) leaves it advisory at any count,
	// which is why the bool travels with the int.
	ArchiveThreshold    int
	ArchiveThresholdSet bool
}

// From projects an already-loaded config onto a Policy. Callers that
// hold a *config.Config for other reasons use this; those that don't
// use Load. A nil config is the zero Policy, matching the way every
// consumer of config.Load already treats an absent aiwf.yaml.
func From(cfg *config.Config) Policy {
	if cfg == nil {
		return Policy{}
	}
	threshold, set := cfg.ArchiveSweepThreshold()
	return Policy{
		TDDStrict:           cfg.TDD.Strict,
		AreasRequired:       cfg.Areas.Required,
		DocsStrict:          cfg.DocsStrict(),
		ArchiveThreshold:    threshold,
		ArchiveThresholdSet: set,
	}
}

// Load reads root's aiwf.yaml and returns the severity policy it
// declares. An absent or unparseable aiwf.yaml yields the zero Policy —
// escalating nothing — because every caller here is a surface that must
// still report findings when the configuration is missing or broken,
// and the malformed-config diagnosis is `aiwf doctor`'s to give. It is
// the same stance `aiwf check` itself takes on a config it cannot read,
// which is what keeps the surfaces in agreement when one is broken.
//
// An empty root is the zero Policy rather than a read: config.Load
// resolves a relative "aiwf.yaml" against the process working
// directory, so an unrooted tree would otherwise escalate its findings
// against whatever repo the operator happens to be standing in.
func Load(root string) Policy {
	if root == "" {
		return Policy{}
	}
	cfg, err := config.Load(root)
	if err != nil {
		return Policy{}
	}
	return From(cfg)
}

// Apply raises the severity of every finding p escalates, in place.
//
// t supplies the pending-sweep count the archive ceiling compares
// against; passing the tree rather than the count is what keeps a
// caller from escalating one tree's findings against another's count.
//
// Callers run Apply after the findings are assembled — including any a
// surface composes outside check.Run — so a pass covering a
// CLI-composed rule reaches it too.
func Apply(findings []check.Finding, p Policy, t *tree.Tree) {
	check.ApplyTDDStrict(findings, p.TDDStrict)
	check.ApplyAreaRequiredStrict(findings, p.AreasRequired)
	check.ApplyDocsStrict(findings, p.DocsStrict)
	check.ApplyArchiveSweepThreshold(findings, p.ArchiveThreshold, p.ArchiveThresholdSet, check.CountPendingSweep(t))
}
