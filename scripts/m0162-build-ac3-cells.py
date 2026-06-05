#!/usr/bin/env python3
"""Build internal/workflows/spec/branch/rules_m0162_ac3.go from the full E2E pin surface."""
import re
from pathlib import Path

root = Path("/workspaces/aiwf-M-0162")
integration = root / "internal/cli/integration"

pairs = []

# Pass 1: Scenarios-framework stamped CellIDs.
for fp in sorted(integration.glob("*_test.go")):
    text = fp.read_text(encoding="utf-8")
    lines = text.split("\n")
    pending_cellid = None
    for line in lines:
        m_cellid = re.match(r'\s*CellID:\s*"([^"]+)"', line)
        m_name = re.match(r'\s*Name:\s*"([^"]+)"', line)
        if m_cellid:
            pending_cellid = m_cellid.group(1)
        elif m_name and pending_cellid:
            pairs.append((pending_cellid, m_name.group(1), fp.name))
            pending_cellid = None

# Pass 2: inline pinCell calls in non-Scenarios test files.
# Two shapes are accepted:
#   pinCell("branch-cell-...", t.Name())  → complete literal cell ID
#   pinCell("branch-cell-...-"+var, ...)  → DYNAMIC; skip — the
#                                            matrix enumeration is
#                                            handled by Pass 3.
# Skipping prefix-only literals (those followed by `+`) is load-bearing:
# emitting a bare prefix as a cell creates a dead entry that violates
# AC-4's bijection invariant #1. See the M-0162/AC-3 reviewer S11 finding.
inline_re = re.compile(r'pinCell\("([^"]+)"\s*(\+?)')
for fp in sorted(integration.glob("*_test.go")):
    text = fp.read_text(encoding="utf-8")
    for m in inline_re.finditer(text):
        cid = m.group(1)
        if m.group(2) == "+":
            continue  # matrix rows handled by Pass 3
        if "+" in cid:
            continue
        pairs.append((cid, f"inline pin in {fp.name}", fp.name))

# Pass 3: dynamic (string-concat) pinCell calls — enumerate matrices manually.
# AC-1: 4 trunk-name shapes
ac1_shapes = ["main", "github-classic-master", "operator-chosen-dev", "operator-chosen-trunk"]
for s in ac1_shapes:
    pairs.append((f"branch-cell-m0161-ac1-{s}", f"trunk shape {s}", "authorize_scenarios_test.go"))

# AC-2: 16 rung-pair cells (4 rungs × 4 rungs)
rungs = ["trunk", "epic", "milestone", "patch"]
for c in rungs:
    for ta in rungs:
        name = f"{c}_to_{ta}"
        pairs.append((f"branch-cell-m0161-ac2-{name}", f"rung-pair {name}", "authorize_scenarios_test.go"))

# Dedup by cellID (the inline-pass and dynamic-pass may double-count).
seen = {}
for cid, name, fname in pairs:
    if cid not in seen:
        seen[cid] = (name, fname)
pairs = [(cid, n, f) for cid, (n, f) in seen.items()]
pairs.sort(key=lambda p: p[0])

out = []
out.append("package branch")
out.append("")
out.append('import "github.com/23min/aiwf/internal/workflows/spec"')
out.append("")
out.append("// ac3ExpandedCells returns the M-0162/AC-3 cell-expansion entries:")
out.append(f"// {len(pairs)} cells, one per discriminating E2E subtest across the")
out.append("// M-0106 / M-0159 / M-0160 / M-0161 surfaces. Generated from the")
out.append("// CellID/Name pairs stamped into internal/cli/integration/*_test.go")
out.append("// at AC-3 RED time. Each cell is a catalog-vocabulary entry:")
out.append('// Outcome=Legal means "the test body\'s Expect assertion is the')
out.append('// behavioral pin; this cell exists for the AC-4 bijection mapping."')
out.append("//")
out.append("// The bijection invariant at AC-4 will assert:")
out.append("//   1. Every cell here has at least one Pin (every scenario has a")
out.append("//      CellID referencing it).")
out.append("//   2. Every Pin references an ID present in this list (no orphan")
out.append("//      CellIDs in Scenario literals).")
out.append("//")
out.append("// Maintenance: when a scenario is added, removed, or renamed,")
out.append("// re-stamp via scripts/m0162-stamp-cellid.sh and regenerate")
out.append("// this file via scripts/m0162-build-ac3-cells.py. The AC-3")
out.append("// cell-presence test at internal/policies/m0162_ac3_expanded_set_test.go")
out.append("// pins the CellID → branch.Rules() consistency.")
out.append("func ac3ExpandedCells() []spec.Rule {")
out.append("\treturn []spec.Rule{")

for cellid, name, fname in pairs:
    src_file = fname.replace("_test.go", "")
    comment_name = name[:80]
    if len(name) > 80:
        comment_name += "…"
    out.append(f"\t\t// {cellid} — {src_file}: {comment_name}")
    out.append("\t\t{")
    out.append(f'\t\t\tID:      "{cellid}",')
    out.append("\t\t\tOutcome: spec.OutcomeLegal,")
    out.append('\t\t\tSources: spec.RuleSource{Decision: "ADR-0010"},')
    out.append("\t\t},")

out.append("\t}")
out.append("}")
out.append("")

target = root / "internal/workflows/spec/branch/rules_m0162_ac3.go"
target.write_text("\n".join(out), encoding="utf-8")
print(f"wrote {target} with {len(pairs)} cells")
