# TODO

Personal scratchpad. Three sections, two disciplines:

- **Don't forget** — append one line; delete when done. Never reordered.
- **Next, in order** — dated, and **replaced wholesale** on each refresh, never
  appended to. An ordering that can only grow stops being an index.
- **Parked** — decided not-now, one line each, so a closed question isn't reopened.

Claude may reconcile *Next, in order* against live tree state and propose a
delta — terminal entries dropped, new entities placed, moves justified — and
writes only what you accept. A cluster's thesis is the stable part; membership
churns beneath it. Each cluster leads with what it is about; each gap gets one
line saying what it is.

## Don't forget

- initatives
- any more tdd related?
- G-0073 depends_on restricted to milestone→milestone; cross-kind blocking via body prose
- Area skill perhaps too big?
- Do we have anything in CLAUDE.md that really should belong in shipped surfaces?
- E-0019 and ADR-0001 both still `proposed` — deferred, no forcing function yet
- G-0436 and G-0439 still carry no priority
- G-0375 has a live sighting: a probe wrote user.email into the repo's own
  .git/config and broke TestResolveActor_*; the failures read as environmental
- E-0079 is active and owns M-0291..M-0294, all four with empty bodies and no
  ACs. The ordering below predates it and does not account for it
- `readStatusAt` and the walker's path-resolving fallback arm are dead in
  production — reachable only through conditions no BulkRevwalk flag produces.
  Annotated rather than deleted, deliberately; nothing tracks the choice

## Next, in order (2026-08-03)

These clusters cover the *defect* surface; the low-priority feature and
enhancement gaps fit none and are not listed. Each gap sits in exactly one
cluster.

### 1. Lying gates

A gate that reports green while proving nothing is worse than an absent one:
every commit landing before the fix gets a false pass, and a reader auditing
coverage finds a guard and stops looking.

- **G-0532** — the width check reads only the filename, so a narrow `id:` under a
  canonical filename is reported by nothing at all
- **G-0518** — a body citing a real entity at a legacy width passes `body-prose-id`
- **G-0516** — the comment-attrition scan is diff-scoped, so a stale comment
  outside the changed hunk is never examined
- **G-0474** — the blank-identifier ban has no detector; one live instance *(cheap)*

### 2. Oracles — what tells the builder it is wrong *(design first; one cheap member)*

Cluster 1 is the acute form of this: a gate that passes bad state. These are the
milder forms — a gate shallower than its name, a gate that judges its own author,
and one absence. The apparatus decides *shape* well and decides nothing about
whether the spec going in is any good. Vocabulary in `docs/design/oracles.md`;
initiative context in `quality-signal-and-cadence.md` Q2 and Q6.

- **G-0533** — `dupl` is off across the test corpus, the larger and
  faster-growing half. Cheap, and the only member with a known fix: diff-scope
  it. Sequenced after G-0462, per this list's own repair-the-instrument-first rule
- **G-0110** — mutation testing's diff filter excludes new files, which is the
  code most likely to carry an untested mutant. Blocks the idea below it
- **G-0253** — the coverage gate is statement-scoped, so a defensive arm that
  never executes still reads as covered
- **The branch-coverage audit is agent-performed** — `wf-tdd-cycle` is explicit
  that no tool enforces it, so the agent that wrote the code certifies its own
  coverage. Independence, not depth; orthogonal to G-0253. *(unfiled)*
- **Wiring `make mutate-diff` into the AC's green→done** is the compensating
  control for the two above — an independent verdict at the moment an AC is
  claimed done. Needs G-0110 first *(unfiled)*
- **Spec adequacy has no oracle** — nothing decides whether an AC states an
  observable condition. Design first, and reconcile with E-0019 and
  `tdd-cycle-subagent-boundaries.md` before opening anything: three captures
  already circle this *(unfiled)*

### 3. Cheap fixes, batchable now *(two wf-patches)*

Small, no design content between them.

- **G-0464** — three check predicates treat a `deferred` AC as still in scope
- **G-0493** — `edit-body`'s two modes judge frontmatter divergence by different rules
- **G-0498** — verb commits store raw bytes, bypassing git's clean filters
- **G-0502** — a submodule under a moved directory is stranded, unseen by the guard
- **G-0510** — the `enums:ignore` escape accepts three spellings that aren't the directive
- **G-0513** — the archive sweep reports "converged" when a candidate won't parse
- **G-0479** — epic template nests out-of-scope below the level three surfaces require
- **G-0482** — milestone template omits the Approach section `entity-body-empty` requires

### 4. Write scope — what a verb may commit *(epic; spec first)*

A verb commits bytes it did not compute. The guard shipped; these are the routes
it cannot see, plus the missing verbs that are the upstream cause — where no verb
exists, hand-editing is the only route, and the guard then refuses the result.

- **G-0168** *(high)* — four set-at-create fields have no mutation verb
- **G-0276** *(high)* — verb commit isolation rests on `git stash`; leaves orphans
- **G-0471** *(high)* — nothing catches a verb run by a binary older than the source
- **G-0442** — `addressed_by` / `superseded_by` have no amend path
- **G-0486** — a directory move rewrites a symlink as a copy and drops the exec bit
- **G-0500** — `edit-body` over a hand-moved file lands a duplicate id the local check misses
- **G-0501** — `init` / `update` replace a symlinked `CLAUDE.md` with a frozen copy
- **G-0506** — the AC phase promote refuses from working-copy bytes HEAD contradicts
- **G-0512** — a directory sitting at a move's destination is invisible to the decline

### 5. E-0074 — same-state convergence *(small epic, parallel any time)*

Finish what the convergence milestone started, so the convention the kernel
advertises holds everywhere it claims to. Done-condition is mechanical: zero
OPEN entries in the NoOp allowlist.

- **G-0460** *(high)* — a repeat `authorize` leaves two active scopes and no finding
- **G-0459** — five event-shaped verbs append a duplicate record on an identical re-run
- **G-0461** — a composite `--for-entity` ack suppresses nothing
- **G-0458** — same-phase AC promote refuses where every other verb converges

### 6. What aiwf ships — does it arrive, stay current, and bind? *(unscoped)*

Each member is a shipped surface whose failure is silent: it never arrives, it
silently goes stale, or it cannot be enforced at all.

- **G-0523** *(high)* — guidance reaches an assistant through one channel that can
  fail unobserved
- **G-0504** — `doctor` byte-checks verb skills only; ritual and guidance drift read as healthy
- **G-0526** — source-discipline rules ship as prose with no seam to enforce them
- **G-0529** — CHANGELOG completeness rests on recall at epic wrap; nothing checks it
- **G-0514** — `skill-body-id` tells CLI metavariables and non-id acronyms to become placeholders
- **G-0530** — milestone specs mandate four sections that duplicate structured data
- **A shipped skill's severity claims are not tied to the code's constants** — the
  `aiwf-check` skill documented `manual-edit` as error severity where the rule
  emits a warning, so an operator reading it misjudges whether a finding blocks a
  push. Corrected by hand; nothing stops the next drift *(unfiled)*

### 7. Error contract *(epic, parallel any time)*

Everything a machine caller reads when a verb refuses. Six gaps on one surface
with no decision outstanding first.

- **G-0483** — uncoded verb errors exit 2, the usage bucket *(absorbed G-0494's
  prose-flattened refusal and split exit code)*
- **G-0456** — prelude resolution errors ignore `--format=json`
- **G-0234** — typed `Coded` coverage, allowed sets inline, remediation sentences
- **G-0169** — four verbs have no JSON envelope
- **G-0432** — `version` / `doctor` and the envelope resolve the version differently
- **G-0070** — `doctor` has no `--format=json`

### 8. E-0077 — duplication and the instrument that measures it *(epic)*

Collapse convergent duplication and put the acknowledged-duplication inventory
back under an owner. Order is forced: the instrument is repaired first.

- **G-0462** *(high)* — `ETXTBSY` and lint-cache contention make the gate red for
  reasons unrelated to the change. Fixed first — it is what measures the rest
- **G-0472** — four clone families each duplicate one job across two layers
- **G-0473** — the `dupl` exclusion list is unowned; two entries are stale
- **G-0508** — three near-copies of the `internal/verb` AST scan drift apart
- **G-0453 / G-0454 / G-0455** — the remainder, each decide-before-extracting

### 9. Doc drift *(one wf-patch, whenever)*

Citations that no longer resolve. Cheap, isolated, no thesis between them beyond
staleness.

- **G-0412** — coverage-ignore rationale text is inaccurate across several files
- **G-0414** — stale test naming in a real-binary test
- **G-0417** — dead branch-not-found code and stale spec-table rows
- **G-0436** — `CLAUDE.md` and the id-allocation doc cite relocated paths
- **G-0439** — relocation and archive sweeps skip references outside their own scope
- **G-0444** — the id-allocation doc cites renamed functions
- **G-0517** — narrow id citations remain in the design docs, overview and architecture
- **G-0519** — documentation is not reference-checked; a cited id need not resolve

## Parked

- **E-0076 cancelled** — "give three documented rules the detectors they lack".
  The audit declined two of its three members; G-0471 moved to cluster 4.
- **A documented rule may stand without a chokepoint.** Detector-shaped gaps are
  declined by default. Trigger to revisit: a second instance where the absence
  actually costs something.
- **Nine gaps closed `wontfix`** with per-gap reasons in git. The grounds are
  three, not one — cost-per-subject, detector precision, base rate.
- **Verb commits fire post-commit but not pre-commit or commit-msg.** Principled,
  not a defect: post-commit only observes, the other two can refuse or rewrite.
  Trigger: a consumer reporting their own `pre-commit.local` didn't fire.
- **The pocv3 dangling-reference policy is not a bug.** It carries an allowlist
  and skips `archive/`. Trigger: a second file legitimately needing the literal path.
- **Survey with `aiwf list`, never `aiwf status`** — only `list` resolves
  cross-branch entities, so `status` and STATUS.md undercount during an epic.
- **A mechanical chore with no decision in it doesn't get a gap.** G-0531 was
  filed and cancelled at M-0290's wrap — ~40 comments naming a retired verb, no
  consumer reach, no judgment in the work. Tracking it costs a reader's attention
  the fix would not. Trigger: a chore large enough that someone needs to schedule it.
