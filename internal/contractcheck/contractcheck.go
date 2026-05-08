// Package contractcheck validates the structural correspondence
// between the consumer repo's aiwf.yaml `contracts:` block, the
// contract entities in the tree, and the on-disk schema/fixtures
// paths. It produces `contract-config` findings for the cases listed
// in §10 of docs/pocv3/plans/contracts-plan.md:
//
//   - entries[].id has no matching contract entity in the tree;
//   - entries[].schema does not exist as a regular file;
//   - entries[].fixtures does not exist as a directory;
//   - a contract entity has no entries[] binding (advisory).
//
// The validator-name reference rule (entries[].validator must name a
// validator in the validators map) is enforced earlier by
// aiwfyaml.Validate and never reaches this package.
//
// This package is the disk-touching counterpart to the entity-tree
// checks in internal/check/. It returns check.Finding so its
// output composes with the rest of the validation envelope.
package contractcheck

import (
	"fmt"
	"os"

	"github.com/23min/ai-workflow-v2/internal/aiwfyaml"
	"github.com/23min/ai-workflow-v2/internal/check"
	"github.com/23min/ai-workflow-v2/internal/contractconfig"
	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/tree"
)

// Run produces contract-config findings against the configured
// bindings. A nil Contracts argument is treated as "no bindings" and
// returns nil.
//
// Errors are findings: a missing binding doesn't crash the run; the
// caller's overall validation envelope simply gets a contract-config
// entry to surface.
func Run(t *tree.Tree, contracts *aiwfyaml.Contracts, repoRoot string) []check.Finding {
	if contracts == nil {
		return nil
	}
	var findings []check.Finding

	idToEntity := make(map[string]*entity.Entity)
	for _, e := range t.ByKind(entity.KindContract) {
		idToEntity[e.ID] = e
	}
	boundIDs := make(map[string]bool, len(contracts.Entries))

	resolved, escapeFindings := contractconfig.Resolve(repoRoot, contracts.Entries)
	findings = append(findings, escapeFindings...)

	for i, e := range contracts.Entries {
		boundIDs[e.ID] = true

		ent, ok := idToEntity[e.ID]
		if !ok {
			findings = append(findings, check.Finding{
				Code:     "contract-config",
				Severity: check.SeverityError,
				Subcode:  "missing-entity",
				EntityID: e.ID,
				Path:     "aiwf.yaml",
				Message:  fmt.Sprintf("contracts.entries[%d]: id %q has no matching contract entity in work/contracts/", i, e.ID),
			})
		}

		// Skip existence checks for entries whose paths escaped the
		// repo root; the path-escape finding is the more informative
		// one and double-reporting just adds noise.
		if !resolved[i].Skip {
			if !isRegularFile(resolved[i].SchemaPath) {
				findings = append(findings, check.Finding{
					Code:     "contract-config",
					Severity: check.SeverityError,
					Subcode:  "missing-schema",
					EntityID: e.ID,
					Path:     "aiwf.yaml",
					Message:  fmt.Sprintf("contracts.entries[%d] (id=%s): schema path %q does not exist or is not a regular file", i, e.ID, e.Schema),
				})
			}
			if !isDirectory(resolved[i].FixturesPath) {
				findings = append(findings, check.Finding{
					Code:     "contract-config",
					Severity: check.SeverityError,
					Subcode:  "missing-fixtures",
					EntityID: e.ID,
					Path:     "aiwf.yaml",
					Message:  fmt.Sprintf("contracts.entries[%d] (id=%s): fixtures path %q does not exist or is not a directory", i, e.ID, e.Fixtures),
				})
			}
		}

		// Skip entity-status checks for terminal-state contracts;
		// callers exclude them from verification anyway.
		if ent != nil && (ent.Status == entity.StatusRejected || ent.Status == entity.StatusRetired) {
			continue
		}
	}

	// Advisory finding: a contract entity exists with no binding. Not
	// always wrong — a registry-only contract is valid — so this is a
	// warning rather than an error.
	for id, ent := range idToEntity {
		if boundIDs[id] {
			continue
		}
		if ent.Status == entity.StatusRejected || ent.Status == entity.StatusRetired {
			continue
		}
		findings = append(findings, check.Finding{
			Code:     "contract-config",
			Severity: check.SeverityWarning,
			Subcode:  "no-binding",
			EntityID: id,
			Path:     ent.Path,
			Message:  fmt.Sprintf("contract %s has no binding in aiwf.yaml.contracts.entries[]; verification will skip it", id),
		})
	}

	return findings
}

// isRegularFile reports whether path exists and is a regular file
// (not a directory, symlink-loop, or device).
func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// isDirectory reports whether path exists and is a directory.
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
