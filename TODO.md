# TODO

Personal scratchpad. Two sections, two disciplines:

- **Don't forget** — append one line; delete when done. Never reordered.
- **Next, in order** — dated, and **replaced wholesale** on each refresh, never
  appended to. An ordering that can only grow stops being an index. It orders and
  names; it does not size. Whether something is a patch, a milestone or an epic
  is a planning judgment that churns with membership, and belongs where the
  planning happens.

Nothing here is a record. A decision goes to a `D-NNNN` entity, a declined gap
to that gap's own disposition, an operating rule to the shipped guidance — each
queryable, diffable, and reachable by someone who never opens this file. What
belongs here is ordering and recall: what to do next, and what not to lose
before it becomes an entity.

Claude may reconcile *Next, in order* against live tree state and propose a
delta — terminal entries dropped, new entities placed, moves justified — and
writes only what you accept. A cluster's thesis is the stable part; membership
churns beneath it. Each cluster leads with what it is about; each gap gets one
line saying what it is.

## Don't forget

- initatives
- any more tdd related?
- Area skill perhaps too big?
- Do we have anything in CLAUDE.md that really should belong in shipped surfaces?
- E-0019 and ADR-0001 both still `proposed` — deferred, no forcing function yet
- G-0375 has a live sighting: a probe wrote user.email into the repo's own
  .git/config and broke TestResolveActor_*; the failures read as environmental
- `readStatusAt` and the walker's path-resolving fallback arm are dead in
  production — reachable only through conditions no BulkRevwalk flag produces.
  Annotated rather than deleted, deliberately; nothing tracks the choice
- G-0626 + T1.4 in gap-truth-audit.md — two patches, guidance rule first, then
  the status resolver

## Next, in order (2026-08-23)

Clusters 1–9 and 11 cover the *defect* surface; cluster 12 covers the machine the
gates run on. Cluster 10 is the exception: absences in what the kernel can
express, which are not defects and each need a decision first.
Feature and enhancement gaps fit none and are not listed, at any priority — the
filter is what the gap is, not how urgent it is. Each gap sits in exactly one
cluster.

In flight: **E-0088** owns M-0314..M-0317 — every path-changing verb repairs the
links it breaks. M-0314 is the entry point; M-0315 and M-0316 follow it and
M-0317 settles whether ADR-0033's `docs/` delegation fires at all. It closes no
gap listed below, so nothing is omitted on its account: the `docs/` half stays as
G-0478 and G-0439 in cluster 11.

### 1. Lying gates

A gate that reports green while proving nothing is worse than an absent one:
every commit landing before the fix gets a false pass, and a reader auditing
coverage finds a guard and stops looking. The largest cluster, and the members
fall in three shapes: a scan with a hole in its reach, a gate that fires on the
wrong thing, and no gate at the point one would still be cheap.

- **G-0518** — a body citing a real entity at a legacy width passes
  `body-prose-id`, which canonicalizes before resolving. Measured worklist is two
  bodies. G-0559 removed the generator by deriving `IDFormat` from
  `CanonicalPad`, so what remains is chat-convention leakage; the case is now
  symmetry — entity ids, docs, skills and Go literals are all enforced, entity
  body citations are the one axis of five that is not
- **G-0563** *(low)* — bare-loading verbs refuse a reference that resolves only on
  another branch. The remaining surface of the gate G-0558 fixed for read paths
- **G-0543** — the golangci firing harness asserts its isolation against the
  command constructor, not against the harness that must use it; reverting the
  call site leaves every test green. The instance form of this cluster's thesis:
  the gate built to catch a dormant rule has claims nothing checks
- **G-0516** — the comment-attrition scan is diff-scoped, so a stale comment
  outside the changed hunk is never examined. Its title-bearing shape widens the
  diff filter and leaves the phrase set alone; only the second shape asks to
  reverse the calibration argued in the source
- **G-0580** — the skill-edit backstop reaches embedded rituals only, so an agent
  card or a verb skill changes with nothing asking who owns it
- **G-0581** — the backticked-verb resolution policy walks the `aiwfx-*` skills only,
  so a retired verb would survive in the guidance source, the `wf-*` skills, the
  agent cards and the templates alike — none of them walked
- **G-0602** — `git log --name-only` emits no diff for a merge, so prose written
  into a shipped skill while resolving a conflict passes the provenance gate
- **G-0606** — the shipped-prose ban walks test functions, so the same assertion
  written as a production policy sits outside it by construction
- **G-0607** — a regexp carries its needle inside the compiled pattern, so regexp
  matching over shipped content bypasses that same ban
- **G-0618** — the provenance backstop's path scoping reaches under half the shipped
  files; duplicated by G-0580, which describes the same hole against a retired predicate
- **G-0369** *(low)* — `body-prose-id` misses an unhyphenated reference to a real entity
- **G-0302** *(low)* — `check --fast` skips in-memory contract-config validation
- **G-0307** *(low)* — the top-level `aiwf.yaml` decode stays non-strict; the areas
  and `docs:` blocks reject an unknown key, the top level does not
- **G-0535** — three more policies scope their AST walk to `cmd/aiwf`, one file with
  no tests; two inspect nothing and the third is degenerate rather than vacuous
- **G-0068** — the discoverability policy substring-matches finding codes and misses
  every composite subcode `internal/check` emits, dynamic or literal
- **G-0550** — a commit carrying `aiwf-force` and no actor at all is refused by
  neither rule set: both test the actor for being non-human, which absent is not
- **G-0573** — **already addressed.** `29eb2a94c` composed all four severity passes
  into the guard, and ADR-0042 rules the behavioural residue moot rather than
  pending. Promote `addressed --by-commit 29eb2a94c`
- **G-0576** — `acs-tdd-tests-missing` reads an AC's whole history, so one cycle's
  trailer satisfies it for that criterion's entire life
- **G-0571** — a body missing a required section outright is skipped by design;
  `entity-body-empty` reports only one that is present and empty
- **G-0233** — most of the htmlrender suite asserts structure with `strings.Contains`,
  proving a literal exists somewhere rather than in the right element, and no AST
  policy stops the pattern creeping back
- **G-0540** — the assertions that replaced three vacuous policies were each accepted
  as narrower than their names; one of the three has since been deleted outright,
  so the live worklist is the other two
- **G-0539** — a plain commit staging entity files passes pre-commit, and the finding
  it produces never surfaces in the default flow at all — not at the push, and not
  after, once the upstream is set
- **G-0603** — the provenance backstop is commit-scoped by design, so a forgotten
  entity trailer surfaces at the next gate rather than while the edit is cheap
- **G-0608** — negative regression pins over shipped surfaces fit neither class
  D-0070 names; one site was dropped rather than invent a third, and a second pin
  is invisible to the ban entirely

### 2. Surfaces that state something false *(one decision first)*

A reader acts on these and is wrong. Unlike a stale citation, a wrong claim
about behaviour reads as authoritative and stops the reader looking further —
the same failure whether the surface ships to a consumer or stays in this repo;
shipping changes the blast radius, not the thesis. Nothing mechanical covers the
claim itself: G-0542 pinned where a row sits and G-0547 pinned its shape;
neither reads what it says.

- **G-0553** *(high)* — six Meaning cells state a trigger the code contradicts, all six
  re-measured; more omit the guard that decides whether the rule fires, and the body
  leaves that population open. The Fix column is gone, so re-derive against the
  two-column tables rather than the body's counts
- **G-0549** — a documented row for a code nothing emits, and a conditional code
  with no row inheriting the wrong table. Two ends of one broken mapping
- **G-0560** — the Normative doc tree states behaviour the kernel reverses or
  refuses: epic roll-up, the `CLAUDE.md` carve-out, `render roadmap --write`
  committing, and worked examples the verbs now refuse. Dated inventory in
  `docs/initiatives/normative-docs-drift-audit.md`
- **G-0595** *(high)* — every open gap, live ADR, live decision and Normative doc
  measured against the kernel; two thirds carry at least one finding. Dated inventory
  in `docs/initiatives/entity-truth-audit.md`; this gap tracks absorbing it. The
  2026-08-23 gap audit re-derives the entity half at a higher evidence bar
- **G-0590** *(high)* — `show`'s text renderer is the canonical per-entity view and
  drops the terminal reason that `history` renders; `--format=json` carries it
- **G-0591** — a sentence saying a concern is deferred to some entity makes a promise;
  nothing reports it when that entity is later cancelled
- **G-0589** — the doc authority tiering classifies every `docs/` subtree and leaves
  root and derived documents unassigned
- **G-0217** — `status --worktrees` cannot tell a wrapped milestone merged to its epic
  from one merged nowhere; the two-state conflation the body claims does not exist,
  since the label is gated on a terminal status
- **G-0366** — the roadmap renderer is epic-only, so a gap closed by a patch is
  invisible in it. Two of its three Direction items are dead: the CHANGELOG sibling
  shipped, and wiring roadmap regen into `wf-patch` is forbidden by that ritual
- **G-0385** — `upgrade` resolves latest from the proxy list endpoint and under-reports
  right after a fresh tag
- **G-0333** — the two-tier `--force` boundary is load-bearing and stated in six
  shipped and Normative surfaces; only `CLAUDE.md` and the provenance model omit it.
  The body's "undocumented" premise is dead — it is an instance of this cluster, not
  a report of one

### 3. Oracles — what tells the builder it is wrong *(design first)*

Cluster 1 is the acute form of this: a gate that passes bad state. These are the
milder forms — a gate shallower than its name, a gate that judges its own author,
a gate whose red says nothing about the change under test, and one absence. The
apparatus decides *shape* well and decides nothing about whether the spec going
in is any good. Vocabulary in `docs/design/oracles.md`; initiative context in
`quality-signal-and-cadence.md` Q2 and Q6.

- **G-0375** *(high)* — a contributor whose global config sets `commit.gpgsign` sees
  every commit-based test in `internal/verb` and `internal/gitops` fail, saying
  nothing about their change; the count grows with the suite. The fix is per-key, not
  one blanket cut-off — actor resolution depends on inheriting global identity
- **G-0282** *(high)* — "what verb undoes this?" has no chokepoint; code review is
  the whole enforcement, and it never audits verbs that already shipped. The
  filed form of the agent-performed audit two entries below
- **G-0564** — what cancelled E-0080 left behind: no declarative enumeration of
  blessed workflows, and choreography pinned to a single host. Neither has a settled
  design question. Its body still reads as though E-0080 were live
- **G-0565** — an archive-migration test's verdict turns on refs its fixture never
  stages. G-0468's family: an oracle answering about something other than its
  subject
- **G-0533** — `dupl` is off across the test corpus, the larger and faster-growing
  half. The body's first move is the *threshold raise*, not diff-scoping, which it
  prices at a config file, a Make target, hook and CI wiring, and a host CI lacks.
  Unblocked — G-0462 repaired the instrument this was waiting on
- **G-0110** — mutation testing's diff filter is misdiagnosed: from the module root
  it includes new files; passed a subpath it excludes everything, modified files
  included. Blocks nothing — `make mutate-diff` never calls `--diff`
- **G-0253** — the coverage gate is statement-scoped, so an arm with no block of its
  own — an implicit else, a co-listed `case`, a short-circuited sub-condition — reads
  as covered. A defensive arm with a body is caught
- **G-0121** *(high)* — legal workflows and verb composition are not pinned
  mechanically. Listed again: E-0080 was the mechanized half and is cancelled, so
  nothing closes this at anyone's wrap. Its remainder is G-0564 above
- **G-0583** — the milestone preflight asks the reader to confirm every criterion is
  concrete and testable and supplies no method for deciding that. The filed form of
  the spec-adequacy absence below, which has since been attempted and withdrawn
- **G-0585** — shipped skills grant a verdict on the basis of reading; the appendix
  listing the sites is itself a reading rather than a census
- **G-0587** — the collision it names is not live: one side is a specification D-0069
  rejected, and a shipped agent card satisfies both today. A real rule-vs-rule
  collision exists elsewhere — `CLAUDE.md`'s doc-link carve-out against the
  record-decision skill's ban on exactly that
- **G-0551** — the coherence rules exist twice, verb-side and history-walking, and
  nothing checks that the two agree
- **G-0574** — finding identity concatenates the message and several rules interpolate
  live state into it, so a verb self-refuses while making progress
- **G-0577** — `PolicyFSMInvariants` guards terminal preservation and acyclicity with
  one mechanism
- **G-0400** — the stress catalog reaches under half the leaf verbs; the body's own
  Notes already correct the counts its headline carries
- **G-0537** *(low)* — aiwf declares no contracts, so `verify` and `evolve` never run
  against this repo
- **G-0328** — no byte-identity golden comparator for `check`
- **G-0161** — `AntiRules()` has no negative coverage: nothing asserts the kernel does
  *not* enforce each ANTI cell
- **G-0605** — the Go-source checks match syntax where the rules are written about
  meaning; some parse a tree, some scan raw bytes, and neither resolves a type.
  Analysis in `docs/explorations/10-type-aware-static-analysis.md`
- **The branch-coverage audit is agent-performed** — `wf-tdd-cycle` is explicit
  that no tool enforces it, so the agent that wrote the code certifies its own
  coverage. Independence, not depth; orthogonal to G-0253. *(unfiled)*
- **Wiring `make mutate-diff` into the AC's green→done** is the compensating
  control for the two above — an independent verdict at the moment an AC is
  claimed done. Unblocked; the target already exists *(unfiled)*
- **Spec adequacy has no oracle** — nothing decides whether an AC states an
  observable condition. Attempted once: `wf-measure-spec` shipped 2026-08-14 and was
  withdrawn the next day as contradicting the spec it was built from; E-0085 and
  M-0309 are cancelled. Reconcile with E-0019, D-0066,
  `tdd-cycle-subagent-boundaries.md` and
  `milestone-preflight-as-independent-review.md` before reopening *(unfiled)*

### 4. Cheap fixes, batchable now

Small and design-free down to the rule, which is where the two that pose a decision
begin.

- **G-0464** — three check predicates treat a `deferred` AC as still in scope
- **G-0502** — an *uninitialized* submodule under a moved directory is stranded; a
  checked-out one refuses at exit 3 instead
- **G-0510** — the `enums:ignore` escape accepts three spellings that aren't the
  directive. The census it defers to is empty, so the tightening is
  behaviour-preserving
- **G-0513** — the archive sweep reports "converged" when a candidate won't parse
- **G-0477** *(low)* — a boundary guard that can never be false, under a comment
  describing the rejection the guard above it already performs
- **G-0497** *(low)* — the sites the `test-executable-write` detector reports
  whole-tree, across 13 packages, remain outside `WriteExecutable`. Most of
  `internal/initrepo`'s share runs `sh <path>` rather than `execve`, so that part is
  uniformity debt rather than race exposure

Already fixed, awaiting only the promote:

- **G-0562 / G-0578** — `793b1ad97` (2026-08-19). One call site, two independently
  filed sightings; the fix commit carries no `aiwf-entity:` trailer, which is why
  neither closed. Promote both, keep one entry
- **G-0622** — `0e7500a8e`, on the E-0088 branch and not on `main` at all; it arrives
  terminal on merge. Listed so it is not re-filed

Decisions, not sweeps:

- **G-0493** — `edit-body`'s two modes judge frontmatter divergence by different
  rules. Three mutually-exclusive resolution routes; the `Apply` guard already
  answers the same question field-based for both modes
- **G-0613** — the wrap changelog's category set is narrower than Keep-a-Changelog
  and than this repo's practice, which carries `Changed (breaking)`, `Security` and
  `Internal`. `wf-patch` names the same three-category set, so a fix lands in two
  surfaces. The body asks for a D-0031 amendment

### 5. Write scope — what a verb may commit *(spec first)*

A verb commits bytes it did not compute. The guard shipped; these are the routes it
cannot see, plus the missing verbs that are the upstream cause. Where no verb
reaches a field, hand-editing is the usual route and the guard then refuses the
result — though `import --on-collision update` reaches some of them with full
trailers.

- **G-0168** — three set-at-create fields have no mutation verb; `tdd:`, the only one
  with a real friction report, got `aiwf milestone tdd` in 2026-07. D-0048 defers the
  rest as consistency gaps rather than friction
- **G-0471** *(high)* — nothing catches a verb run by a binary older than the source
- **G-0500** *(high)* — `edit-body` over a hand-moved file lands a duplicate id the local check misses
- **G-0501** *(high)* — `init` / `update` replace a symlinked `CLAUDE.md` with a frozen copy
- **G-0442** — `addressed_by` / `superseded_by` can be backfilled and force-re-pointed
  but never *cleared*, and a forced re-point leaves a stale reciprocal that
  `adr-supersession-mutual` structurally cannot see
- **G-0486** — a directory move drops the exec bit, and `rename-area` drops it on
  `aiwf.yaml` with no move at all and no permanent-`M` signal. The symlink half now
  refuses at exit 2
- **G-0506** — the AC phase promote refuses from working-copy bytes HEAD contradicts
- **G-0512** — a directory sitting at a move's destination is invisible to the decline
- **G-0572** *(high)* — two verbs, each legal where it ran, compose across a branch
  boundary into an AC state the kernel reports at error severity whose only
  criterion-preserving repair is sovereign. Sits with G-0121's composition axis
  rather than this cluster
- **G-0544** *(high)* — four contract verbs reach the commit seam without passing
  through the provenance-decoration layer
- **G-0575** — a clean merge of two legal branches closes a milestone while adding
  criteria to it
- **G-0546** *(low)* — a verb's trailer set is incomplete when the verb returns; the
  CLI layer completes it afterwards
- **G-0498** *(low)* — verb commits write blobs directly, so git's clean filter never runs
- **G-0527** *(low)* — worktree creation is a verb and teardown is plain git. Not
  silent — `aiwf worktree remove` exits 2, `aiwf status` prints the hint, two rituals
  ship the pair — so what is left is a one-sentence doc ask
- **G-0398** — `add milestone` has no precondition against a terminal parent epic, so
  it refuses accidentally rather than purposefully
- **G-0249** *(low)* — no reciprocal guard for a milestone `in_progress` under a
  non-active epic
- **G-0434** — `resolveViaPriorIDs` prefers a stale `prior_ids` match over a reused live id
- **G-0212** — the systematic data-loss audit across verb composition; the scope
  crosses the entity surface rather than sitting under one verb

### 6. E-0074 — same-state convergence *(parallel any time)*

Finish what the convergence milestone started, so the convention the kernel
advertises holds everywhere it claims to. The allowlist's OPEN entries are the
work list for three of the four, but not a done-condition: nothing reads their
reason strings, and G-0461 has no entry and correctly never will.

- **G-0460** *(high)* — a repeat `authorize` leaves two active scopes and no finding
- **G-0459** — five event-shaped verbs append a duplicate record on an identical re-run
- **G-0461** — a composite `--for-entity` ack suppresses nothing
- **G-0458** — same-phase AC promote refuses where every other verb either converges
  or silently duplicates

### 7. What aiwf ships — does it arrive, stay current, and bind?

Each member is a shipped surface whose failure is silent: it never arrives, it
silently goes stale, or it cannot be enforced at all.

- **G-0523** *(high)* — guidance reaches an assistant through one channel that can
  fail unobserved
- **G-0536** *(high)* — the same shape for the check itself. Hooks are not
  committable, so all three positions are opt-in per clone and a fresh clone carries
  no `aiwf check` at all. Every other locally-firing gate has a second position on
  push. **Unblocked** — G-0556 landed, so a CI-shaped clone reports clean; the body
  still says otherwise. What remains is picking a job, giving it a non-shallow
  checkout, and deciding whether it runs on planning-only pushes
- **G-0504** *(high)* — `doctor` byte-checks verb skills only; ritual and guidance drift read as healthy
- **G-0370** *(high)* — ADR-0028 decided the always-on fragment should name
  dispatch triggers for all four role agents. The decision landed; the content
  never did, so the one surface in context every turn stays silent on it
- **G-0526** — source-discipline rules ship as prose with no rule or envelope of
  aiwf's own to enforce them. A seam does exist: the materialized hooks chain to a
  consumer's `<hook>.local`, which is what this repo's own instance rides
- **G-0529** — CHANGELOG completeness rests on recall at epic wrap; nothing checks it
- **G-0514** — `skill-body-id` would tell a CLI metavariable or a non-id acronym to
  become a placeholder. Latent: the M-0288 sweep emptied the population, so nothing
  is being misdirected today
- **G-0530** — the milestone template carries four sections that largely duplicate
  structured data. Nothing in the kernel mandates them — `RequiredSections` is Goal
  and Acceptance criteria — so this is template sprawl, and its `growth.md` half is
  already fixed
- **G-0600** *(high)* — `update` materializes guidance and skills from the running
  binary without consulting disk, so a stale binary moves them backwards and reports
  success. The statusline already implements the guard this asks for, and refuses to
  downgrade — copy it rather than design it
- **G-0601** — `history` renders a commit carrying `aiwf-verb` or `aiwf-actor`, so a
  shipped-surface edit provable by `aiwf-entity` alone is invisible in the timeline it
  belongs to. Measured on `main`: 13 such commits, 7 of them watched skill edits
- **G-0588** — `import` ships as a live verb with no consumer and its spec archived
- **G-0586** — the handoff block carries settled conclusions and has no slot for an
  open question
- **G-0594** — two authoring rules shipped and the next spec written breached both.
  Recorded as a measurement; the remedy it invites is sharpening the wording again,
  which is what failed. Nine of its ten attributions hold; the tenth fails in the
  class it exists to record
- **G-0538** — aiwf-internal ids reach a consumer through Go string literals in
  operator-facing output; the persisted-artifact half is already fixed, so only the
  printed surface still leaks
- **G-0254** — the `Co-Authored-By` trailer convention is unstated and unenforced. Its
  proposed home is a check rule CI never runs — CI has never run `aiwf check` at all,
  which is G-0536 two entries above
- **G-0445** *(low)* — the diff-shape gate hardcodes a `docs/` exclusion that is wrong
  for some consumer repos
- **G-0235** — down to two unbuilt policy tests, `cited_entity_ids_resolve` and
  `cache_invalidation_documented`. Every `CLAUDE.md` item it was filed for has landed,
  been dropped, or been satisfied elsewhere; the body still reads as a doc sweep

### 8. Error contract *(parallel any time)*

Everything a machine caller reads when a verb refuses. One surface, and one fork to
settle before most of it: G-0483 and G-0561 propose opposite moves on the same
exit-2/exit-3 boundary, which D-0044 already ratified a three-class contract for.

- **G-0483** — uncoded verb errors exit 2, the usage bucket *(absorbed G-0494's
  prose-flattened refusal and split exit code)*
- **G-0561** — the install verbs report an operator's permission fault as an
  internal error, so the exit code sends the reader after a bug in aiwf
- **G-0456** — prelude resolution errors ignore `--format=json`
- **G-0234** — typed `Coded` coverage, allowed sets inline, remediation sentences
- **G-0169** — three subverbs have no JSON envelope: `contract recipes`,
  `contract recipe show`, `render roadmap`. `import` ships one
- **G-0070** — `doctor` has no `--format=json`
- **G-0568** — `status` rows omit `subcode`, so they cannot be matched against the
  findings envelope. Carry it when someone next touches the struct; the body declines
  severity outright and says to close this if no comparator materialises

### 9. Duplication and the instrument that measures it *(any order)*

Not one job: most of the collapses do not earn the change, and the gate that
finds them yields about one finding per 200 commits. What is left is independent
— each member decides its own case, and G-0472
now says outright that only one of its four families is worth collapsing. The
inventory question is the one that stands on its own.

- **G-0473** — the `dupl` exclusion list is unowned; two entries are stale
- **G-0472** — four near-identical function families; only one worth collapsing
- **G-0508** — eight policies carry the `internal/verb` AST scan and drift apart; the
  gap's title says three and its body four, so fixing it needs an `aiwf retitle` too
- **G-0453 / G-0454 / G-0455** *(low)* — the remainder, each needing a design or
  decision pass before extracting. G-0454's body prescribes rather than asks, and
  the prescription is measurably not behaviour-preserving
- **G-0604** — five sites walk entities and stubs together, but only one resolves
  ids; the shared walk the body describes does not exist, and extracting to its
  description would widen four rules, one of which blocks pushes
- **G-0545** — two policies assert the same property about the same seam
- **G-0448** — the check rule list is split across four dispatch surfaces, and the
  split is a declared layering boundary rather than an accident. What is missing is a
  wiring chokepoint; `Where to fix` omits the surface carrying seven rules

### 10. What the model can't express *(each gated on a decision)*

Not defects — absences in the kernel's closed-set vocabulary. The work either
gets jammed into a kind that does not fit, or lives outside the kernel and the
projections go quietly incomplete. These sat outside the ordering entirely
because every cluster above covers the defect surface and these are not defects.
Two of them justify themselves by the same kernel rule: correctness must not
depend on LLM behaviour, and a relationship the kernel cannot read is enforced
only in someone's head.

- **G-0060** *(high)* — a patch is real, common work with no entity kind, no FSM and
  no trailer. Closure *is* recordable — `promote <gap> addressed --by-commit` is the
  shipped, guarded route — but there is no patch-side queryable record, and `check`
  never validates what the SHA means. Four resolution options spanning wontfix to a
  seventh kind; the choice is open
- **G-0073** *(high)* — `depends_on` is milestone→milestone only, so every
  cross-kind blocking edge (epic on ADR, ADR on ADR, epic on epic) lives in body
  prose. `aiwf promote` cannot see it and render cannot draw it
- **G-0311** *(high)* — no tier above epic, and since areas landed an epic is an
  area-atom, so cross-cutting work splits into peer epics that cannot even be wired,
  since epics carry no `depends_on`. `docs/initiatives/` is the in-tree workaround
  and is now twelve files deep. The body's sibling-project evidence is external and
  unverifiable from here
- **G-0111** *(high)* — the wrap side of the epic ritual: scope-end bundled into
  `promote done` and no human-only rule on `done`, both measured. The closure
  mechanism shipped in 2026-07 as G-0431, so only the declarative `closes:` field is
  absent — and only that part is a vocabulary gap. Recommends its own epic
- **G-0246** — the ADR relation schema expresses only supersession, and only ADR→ADR
- **G-0569** — the AC `tdd_phase` FSM is linear and terminal, with no vocabulary for a
  second cycle after rework
- **G-0396** *(low)* — `addressed_by_commit` denormalizes a fact git owns, and the copy
  is rewrite-fragile: a merge preserves the SHA, a rebase orphans it. A
  single-source-of-truth issue rather than a vocabulary absence

### 11. Doc drift *(whenever)*

Mostly citations that no longer resolve — cheap and isolated. Three members are
not that: G-0439 is a behavioural mover gap with an ADR collision and a milestone
in flight, and G-0519 and G-0548 are missing chokepoints rather than stale text.

- **G-0412** — coverage-ignore rationale text is inaccurate across several files
- **G-0414** — stale test naming in a real-binary test
- **G-0417** — dead branch-not-found code and stale spec-table rows
- **G-0436** — `CLAUDE.md` cites two dead `cmd/aiwf/` things; the id-allocation half
  was repointed in 2026-07 when G-0443 closed
- **G-0439** — relocation and archive sweeps skip references outside their own scope.
  Live right now: an archive sweep on 2026-08-22 broke three `docs/initiatives/`
  links and `link-check` has been red on `main` ever since. Its leading resolution
  is what ADR-0033 declines
- **G-0444** — the id-allocation doc cites renamed functions
- **G-0517** — narrow ids remain in the design docs, overview and architecture — but
  none is a *citation*; they are worked-example fiction, so the fix is to
  placeholder them, not widen them. Opposite polarity to G-0518, so not one sweep.
  Blocked in an unusual way: a policy test pins the word "Widen" into this gap's own
  Resolution section, so the honest edit turns CI red
- **G-0519** — documentation is not reference-checked; a cited id need not resolve
- **G-0548** — shipped surfaces cite this repo's own paths; the id half is enforced at
  error severity, and only the `CLAUDE.md`-section shape of the path half has been
  mechanized
- **G-0478** *(high)* — an entity move breaks every path-based link naming its old
  location. `link-check` and `wf-doc-lint` do report it, so the absence is a *verb-side*
  repair, not detection — and the detection route the body prefers is what ADR-0033
  declines. E-0088 scopes the `docs/` half out deliberately; its last milestone
  re-frames or confirms this, and carries this gap's stale counts
- **G-0579** — D-0015's consequences cite a drift guard that no longer exists

### 12. Development environment — the machine the gates run on

The first three are one story: a full disk surfaces as failures in whichever tests
write most, which is the binary-building and repo-creating ones. That signature
reads as a flaky suite, so the response it invites — re-run, then hunt a race — is
aimed a layer below the cause. None of the five is a kernel concern; all cost
review cycles, and a green run on a nearly-full disk is not evidence either way.

- **G-0552** — nothing bounds the Go build cache; it reached 84 GB and filled
  the overlay. Carries the preflight idea: a gate that cannot tell a full disk
  from a broken test will mislead every time
- **G-0555** — three test helpers build into a temp dir and never remove it; several
  gigabytes a day here. The fix belongs in the code, but the body's claim that no age
  cutoff can reclaim the backlog does not hold — most of it is days old, and
  reclaimable now
- **G-0554** — the statusline CI cache mints a file per commit and deletes none.
  Same thesis as G-0555: unbounded growth with no reclaim path
- **G-0524** — the devcontainer binds the clone's parent, exposing 28 sibling repos
  as writable and putting a rival `CLAUDE.md` in reach — the capability, not an
  occupant; there is none there today. Its own fix fails `PolicyM0132DevcontainerShape`,
  which requires the binding it removes
- **G-0372** — the history-dependent check rules walk all reachable history from
  scratch on every push
