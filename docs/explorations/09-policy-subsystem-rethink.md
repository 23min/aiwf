# Policy subsystem rethink — architecture pass + empirical verify

This captures the two-round rethink of `internal/policies/` (aiwf's bespoke CI
meta-test suite) run during E-0042 planning. It is the evidence behind
**D-0025** ("Policy subsystem stays bespoke; two linter cleanups excepted").

- **Round 1 — architecture pass** (read-only, `Plan` agent): proposed migrating
  ~9 policies to `golangci-lint`, retiring the firing-fixture meta-gate, and
  several deletes.
- **Round 2 — adversarial empirical verify** (`reviewer` agent): wrote the
  candidate configs and ran `golangci-lint` against fixtures to try to *break*
  round 1's load-bearing claims. It refuted most of them.

**Outcome (D-0025):** the bespoke complexity is largely load-bearing; the
subsystem stays bespoke. Only two cleanups survived the empirical pass —
migrate `no-time-now-in-core` → `forbidigo`, and delete
`filepath-join-segment-by-segment` (gocritic covers it). The detour's value was
confirming the original burndown was sound *and* avoiding a costly wrong turn
(churning days-old G-0235 code, deleting live guarantees, retiring a meta-gate
that does per-site work a table-test cannot). A live instance of the "verify by
measuring, not reasoning" principle the subsystem itself exists to enforce.

---

## Round 1 — architecture pass (proposal)

### Corpus

59 registered policy functions: 2 env-gated meta-gates
(`firing-fixture-presence`, `branch-coverage-audit`), 1 true structure-auditor
(`fsm-invariants` — discards `root`, introspects `entity.AllKinds()`/
`AllowedTransitions`), and 56 tree-scanners (all fixture-able under `root`).
44 are in `grandfatherDark`.

**Doc-drift confirmed by code, not name:** `trailer-order-matches-constants`
and `closed-set-status-via-constants` were labelled "structure-auditors" in
CLAUDE.md/G-0259, but both `WalkGoFiles(root,…)` and are fixture-able. Only
`fsm-invariants` discards `root`.

### What `.golangci.yml` enables today

golangci-lint v2, `default: none`, enabled:
`bodyclose, errcheck, errorlint, forbidigo, gocritic, gosec, govet,
ineffassign, misspell, revive, staticcheck, thelper, unconvert, unused` +
formatters `gofumpt, goimports`. `forbidigo` is ON (with path-scoped
exclusions). `gocritic` is ON (can host `ruleguard`). `depguard` is OFF but
ships inside golangci-lint (config-only to enable, no new dependency).

### Proposed buckets (round 1)

| Bucket | ~Count | Members |
|---|---|---|
| MIGRATE to linter | ~9 | layering-direction (→depguard); no-history-rewrites, no-timestamp-manipulation, no-signature-bypass, no-time-now-in-core (→forbidigo); no-trailer-string-composition, verbs-validate-then-write, validate-check-is-never-writes, atomic-write-chokepoint (→ruleguard) |
| KEEP (bespoke) | ~33 | the cross-file/cross-language/structural set (finding-code↔doc↔test, skill-coverage, design-doc anchors, devcontainer + claude-md families, test-setup/harden, race-cap, fsm-invariants, provenance co-occurrence guards, trailer-keys/order, enum/finding-code adoption, version pair, read-only-verbs, no-retry-loops, no-hardcoded-paths, embedded-rituals, branch-coverage-audit) |
| DELETE | ~5 | filepath-join (gocritic dup), capture-stdout-singleton, cli-helper-locations, m0137-ac3-batched-walker, acks-helper-lift (frozen-snapshot) |
| MERGE | 1 | closed-set-status-via-constants → enum-literal-adoption |
| RETIRE meta | 1 | firing-fixture-presence + grandfatherDark → per-policy table-test |

### Round 1 bottom line

The maintainer's "is this over-built?" hypothesis is *largely correct but
under-counts the bespoke core*: ~9 policies genuinely reinvent linters
golangci-lint already runs, but ~33 are genuinely cross-file/cross-language with
no off-the-shelf equivalent. The dominant defect (per round 1) is a YAGNI
meta-layer: `firing-fixture-presence` + `grandfatherDark` stands in for a
missing one-line test convention. Round 1's single highest-leverage move:
replace the coverage-introspection meta-gate with a uniform per-policy fixture
table-test and re-found the burndown around it. **(Round 2 refuted this.)**

---

## Round 2 — adversarial empirical verify (refutation)

Tooling: `golangci-lint` v2.12.2 (go1.26.2), network available (ruleguard DSL
fetched). All experiments in throwaway `/tmp` modules; no tracked file modified.

### Claim A — `depguard` replaces `layering-direction`: **PARTIAL, leaning REFUTED** (empirical)

Built a non-cycling fixture (`entity` tier-6 importing `verb` tier-2 — a new
upward edge Go's cycle-ban can't catch) + a legal sideways edge + a brand-new
untiered package.

- (1) Illegal upward import FIRES. ✅
- (2) Legal sideways import does NOT fire. ✅
- (3) The "new untiered package" self-check is **LOST** — depguard stayed silent
  on `freshpkg` even importing `entity`. depguard has no "package not in my
  model" concept; a new package added without a tier silently escapes.

Two real losses: the unplaced-package self-check is gone, and depguard has no
tier abstraction, so `tgtTier < srcTier` must be flattened into an O(n²)
hand-maintained pairwise deny matrix that grows with the tree. **Recommendation:
KEEP bespoke.** "No loss" is false.

### Claim B — retire the firing-fixture meta-gate: **REFUTED** (reasoned; key quantity measured)

The proposed replacement (per-policy table-test `fixture→≥1 violation` +
presence-check) is strictly weaker on three axes:

1. **Per-construction-site darkness (decisive).** Measured against the live
   tree: **106 `Policy:"id"` construction sites across 51 policies; 19 policies
   have more than one site** (`acks-helper-lift` 16, `skill-coverage` 6,
   `fsm-invariants`/`git-test-env-harden` 7, …). The coverage gate flags *any*
   dark site; a table-test asserts only that *some* site fires, so a 16-site
   policy with 1 dark site passes the table-test but fails the coverage gate. 55
   sites whose individual darkness the table-test cannot see.
2. **"Fixture exists but doesn't exercise the real branch."** A table row whose
   fixture doesn't drive the specific firing branch is invisible to a
   presence-check and the table-test; the coverage gate keys on actual coverage
   of the construction line.
3. **`-coverpkg` fail-closed.** `hasPoliciesBlocks` errors loudly if the profile
   carries no policies blocks; a table-test would keep passing while measuring
   nothing.

**Recommendation: KEEP the meta-gate.** It is total and per-site; the table-test
is per-policy.

### Claim C — the G-0235 trio → `ruleguard`: **REFUTED** (empirical)

`validate-check-is-never-writes` is the hardest. Two faithful encodings tested:

- **Text-match `$body`**: correctly excludes `Issue` (word boundary holds) but
  **false-positives on legal read-only opens** — `HasThing` with
  `os.OpenFile(…, os.O_RDONLY, 0)` FIRED. The text filter can't inspect the flag
  argument (which the bespoke `hasOSWriteFlag` AST walk does).
- **Structured-inner**: correctly discriminated the write flag, **but only for
  one statement shape** — with three real call shapes (`x,_ := os.OpenFile`,
  nested-call-arg, `if`-init), the single pattern matched only the first and
  MISSED the other two. Faithful coverage needs a `Match` per syntactic shape ×
  per primitive — unbounded and brittle.

So ruleguard forces a choice between false positives or an unmaintainable
matrix. The bespoke policy solves it in ~150 lines with one uniform
`ast.Inspect`. These three (G-0235) landed days ago and two are already lit.
**Recommendation: KEEP bespoke.**

### Claim D — `no-time-now-in-core` → `forbidigo`: **CONFIRMED** (conditional; empirical)

forbidigo config: default-forbid `^time\.(Now|Since|Until)$` with
`analyze-types: true`, then path-exclude the edge (`cmd/aiwf/`, `internal/cli/`)
and the two allowlisted core packages (`internal/repolock/`,
`internal/htmlrender/`). Results:

- existing core (`internal/verb`) FIRES. ✅
- **brand-new unlisted core package FIRES by default — no silent escape.** ✅
- `cli`, `cmd`, `repolock`, `htmlrender` SILENT. ✅
- **Bonus:** with `analyze-types: true`, forbidigo catches an aliased
  `t "time"; t.Now()` import — which the bespoke AST policy explicitly *misses*.
  forbidigo is strictly stronger here.

The "no silent escape" invariant survives precisely because the default is
forbid-everywhere. **Recommendation: SAFE to commit — conditional** on
default-forbid + exclude-edge (not forbid-only-in-core-paths) + `analyze-types`.

### Claim E — the DELETEs

**E(1) `filepath-join-segment-by-segment` → gocritic `filepathJoin`:
CONFIRMED** (empirical). gocritic's `filepathJoin` is active; on a fixture it
fired on embedded forward-slash, backslash, and test files — the bespoke
policy's scope, and a strict superset (it also flags `arg[0]`, which the policy
skips). The repo has zero `filepath.Join` calls with embedded-slash first args,
so the stricter behavior flags nothing new. **SAFE to DELETE.**

**E(2) frozen-snapshot deletes: REFUTED** (empirical) for both:

- `m0137-ac3-batched-walker`: its companion perf test asserts only a 10-second
  wall-time budget; the policy asserts the *structural* property (uses
  `BulkRevwalk`/`BlobReader`; the per-entity helpers are deleted). The perf-test
  docstring itself says a re-introduced per-entity fan-out under budget would
  pass the perf test. Deleting drops a real guarantee. **KEEP.**
- `acks-helper-lift`: not a one-time snapshot — it polices an ongoing
  single-source-of-truth / single-compute / four-consumer-wiring invariant (10+
  violation classes, plus G-0239 extension with its own firing tests). The
  sibling tests only assert signatures compile. **KEEP.**

### Round 2 bottom-line table

| Claim | Migration | Commit to epic? |
|---|---|---|
| A | depguard ↔ layering-direction | **No** — loses unplaced-package self-check; O(n²) deny matrix. KEEP bespoke. |
| B | retire firing-fixture meta-gate | **No** — strictly weaker (per-site darkness, dead-fixture, `-coverpkg`). KEEP. |
| C | ruleguard ↔ the G-0235 trio | **No** — not faithfully expressible; zero payoff on days-old code. KEEP. |
| D | forbidigo ↔ no-time-now-in-core | **Conditional Yes** — de-risked; survives + catches an extra blind spot. |
| E1 | DELETE filepath-join | **Yes** — gocritic superset; no first-arg violations. |
| E2 | DELETE m0137 / acks-helper | **No** — guard live properties nothing else carries. KEEP. |

**Net:** only D (with the config conditions) and E1 are de-risked; A, B, C, and
the E2 deletes stay KEEP. This is D-0025.
