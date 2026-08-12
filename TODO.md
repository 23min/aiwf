# TODO

Personal scratchpad. Two sections, two disciplines:

- **Don't forget** — append one line; delete when done. Never reordered.
- **Next, in order** — dated, and **replaced wholesale** on each refresh, never
  appended to. An ordering that can only grow stops being an index.

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

## Next, in order (2026-08-06)

Clusters 1–9 cover the *defect* surface. Cluster 10 covers absences in what the
kernel can express, which are not defects and each need a decision before any
milestone. Low-priority feature and enhancement gaps fit none and are not
listed. Each gap sits in exactly one cluster.

In flight: **E-0080** owns M-0300..M-0304 — the mechanized half of G-0121, per
D-0063. M-0300 is the entry point; M-0301 and M-0302 are parallel behind it and
M-0304 is parallel throughout. G-0121 closes at that epic's wrap, so it is not
listed below; its remainder is G-0564 in cluster 3.

### 1. Lying gates

A gate that reports green while proving nothing is worse than an absent one:
every commit landing before the fix gets a false pass, and a reader auditing
coverage finds a guard and stops looking.

- **G-0518** — a body citing a real entity at a legacy width passes
  `body-prose-id`, which canonicalizes before resolving. Measured worklist is two
  citations. G-0559 removed the generator by deriving `IDFormat` from
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
  outside the changed hunk is never examined. Reframe before ranking: the
  whole-tree sibling's under-inclusiveness is a calibrated precision trade
  argued in the source, so this asks to reverse a decision, not to fill a hole

### 2. Surfaces that state something false *(one decision, then a milestone)*

A reader acts on these and is wrong. Unlike a stale citation, a wrong claim
about behaviour reads as authoritative and stops the reader looking further —
the same failure whether the surface ships to a consumer or stays in this repo;
shipping changes the blast radius, not the thesis. Nothing mechanical covers the
claim itself: G-0542 pinned where a row sits and G-0547 pinned its shape;
neither reads what it says.

- **G-0553** *(high)* — six Meaning cells state a trigger the code contradicts; ~18 more
  omit the guard that decides whether the rule fires. Larger than a patch. Re-count
  first: the table is two-column now, and eight Meaning cells absorbed text folded
  in from the deleted Fix column
- **G-0549** — a documented row for a code nothing emits, and a conditional code
  with no row inheriting the wrong table. Two ends of one broken mapping
- **G-0560** — the Normative doc tree states behaviour the kernel reverses or
  refuses: epic roll-up, the `CLAUDE.md` carve-out, `render roadmap --write`
  committing, and worked examples the verbs now refuse. Dated inventory in
  `docs/initiatives/normative-docs-drift-audit.md`

### 3. Oracles — what tells the builder it is wrong *(design first; one cheap member)*

Cluster 1 is the acute form of this: a gate that passes bad state. These are the
milder forms — a gate shallower than its name, a gate that judges its own author,
a gate whose red says nothing about the change under test, and one absence. The
apparatus decides *shape* well and decides nothing about whether the spec going
in is any good. Vocabulary in `docs/design/oracles.md`; initiative context in
`quality-signal-and-cadence.md` Q2 and Q6.

- **G-0375** *(high)* — the same shape locally: a contributor whose global config
  sets `commit.gpgsign` sees 221 failures in `internal/verb` that say nothing
  about their change. The blanket fix was tried and reverted — actor resolution
  depends on inheriting global identity, so this is per-key, not one toggle
- **G-0282** *(high)* — "what verb undoes this?" has no chokepoint; code review is
  the whole enforcement, and it never audits verbs that already shipped. The
  filed form of the agent-performed audit two entries below
- **G-0564** — what E-0080 deliberately leaves behind: no declarative enumeration
  of blessed workflows, and choreography pinned to a single host. Neither has a
  settled design question, which is why the epic excludes both — E-0033 examined
  the enumeration surface and chose advisory, so reversing that is a decision
- **G-0565** — an archive-migration test's verdict turns on refs its fixture never
  stages. G-0468's family: an oracle answering about something other than its
  subject
- **G-0533** — `dupl` is off across the test corpus, the larger and
  faster-growing half. Cheap, and it has a known fix: diff-scope it.
  Unblocked — G-0462 repaired the instrument this was waiting on
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

### 4. Cheap fixes, batchable now *(two wf-patches)*

Small, no design content between them.

- **G-0464** — three check predicates treat a `deferred` AC as still in scope
- **G-0493** — `edit-body`'s two modes judge frontmatter divergence by different rules
- **G-0502** — a submodule under a moved directory is stranded, unseen by the guard
- **G-0510** — the `enums:ignore` escape accepts three spellings that aren't the directive
- **G-0513** — the archive sweep reports "converged" when a candidate won't parse
- **G-0562** — a worktree-hook test writes its executable without `WriteExecutable`

### 5. Write scope — what a verb may commit *(epic; spec first)*

A verb commits bytes it did not compute. The guard shipped; these are the routes
it cannot see, plus the missing verbs that are the upstream cause — where no verb
exists, hand-editing is the only route, and the guard then refuses the result.

- **G-0168** *(high)* — four set-at-create fields have no mutation verb
- **G-0276** *(high)* — verb commit isolation rests on `git stash`; leaves orphans
- **G-0471** *(high)* — nothing catches a verb run by a binary older than the source
- **G-0500** *(high)* — `edit-body` over a hand-moved file lands a duplicate id the local check misses
- **G-0501** *(high)* — `init` / `update` replace a symlinked `CLAUDE.md` with a frozen copy
- **G-0442** — `addressed_by` / `superseded_by` have no amend path
- **G-0486** — a directory move rewrites a symlink as a copy and drops the exec bit
- **G-0506** — the AC phase promote refuses from working-copy bytes HEAD contradicts
- **G-0512** — a directory sitting at a move's destination is invisible to the decline

### 6. E-0074 — same-state convergence *(small epic, parallel any time)*

Finish what the convergence milestone started, so the convention the kernel
advertises holds everywhere it claims to. Done-condition is mechanical: zero
OPEN entries in the NoOp allowlist.

- **G-0460** *(high)* — a repeat `authorize` leaves two active scopes and no finding
- **G-0459** — five event-shaped verbs append a duplicate record on an identical re-run
- **G-0461** — a composite `--for-entity` ack suppresses nothing
- **G-0458** — same-phase AC promote refuses where every other verb converges

### 7. What aiwf ships — does it arrive, stay current, and bind? *(unscoped)*

Each member is a shipped surface whose failure is silent: it never arrives, it
silently goes stale, or it cannot be enforced at all.

- **G-0523** *(high)* — guidance reaches an assistant through one channel that can
  fail unobserved
- **G-0536** *(high)* — the same shape for the check itself. Hooks are not
  committable, so both positions are opt-in per clone and a fresh clone carries no
  `aiwf check` at all. Every other locally-firing gate has a second position on push.
  **Unblocked** — G-0556 landed, so the step no longer reports errors on day one;
  the `fetch-depth` question is what remains
- **G-0504** *(high)* — `doctor` byte-checks verb skills only; ritual and guidance drift read as healthy
- **G-0370** *(high)* — ADR-0028 decided the always-on fragment should name
  dispatch triggers for all four role agents. The decision landed; the content
  never did, so the one surface in context every turn stays silent on it
- **G-0526** — source-discipline rules ship as prose with no seam to enforce them
- **G-0529** — CHANGELOG completeness rests on recall at epic wrap; nothing checks it
- **G-0514** — `skill-body-id` tells CLI metavariables and non-id acronyms to become placeholders
- **G-0530** — milestone specs mandate four sections that duplicate structured data

### 8. Error contract *(epic, parallel any time)*

Everything a machine caller reads when a verb refuses. Six gaps on one surface
with no decision outstanding first.

- **G-0483** — uncoded verb errors exit 2, the usage bucket *(absorbed G-0494's
  prose-flattened refusal and split exit code)*
- **G-0561** — the install verbs report an operator's permission fault as an
  internal error, so the exit code sends the reader after a bug in aiwf
- **G-0456** — prelude resolution errors ignore `--format=json`
- **G-0234** — typed `Coded` coverage, allowed sets inline, remediation sentences
- **G-0169** — four verbs have no JSON envelope
- **G-0432** — `version` / `doctor` and the envelope resolve the version differently
- **G-0070** — `doctor` has no `--format=json`

### 9. Duplication and the instrument that measures it *(patches, any order)*

Not an epic, and not one job: most of the collapses do not earn the change, and
the gate that finds them yields about one finding per 200 commits. What is left
is patch-shaped and independent — each member decides its own case, and G-0472
now says outright that only one of its four families is worth collapsing. The
inventory question is the one that stands on its own.

- **G-0473** — the `dupl` exclusion list is unowned; two entries are stale
- **G-0472** — four near-identical function families; only one worth collapsing
- **G-0508** — three near-copies of the `internal/verb` AST scan drift apart
- **G-0453 / G-0454 / G-0455** *(low)* — the remainder, each decide-before-extracting

### 10. What the model can't express *(each gated on a decision)*

Not defects — absences in the kernel's closed-set vocabulary. The work either
gets jammed into a kind that does not fit, or lives outside the kernel and the
projections go quietly incomplete. These sat outside the ordering entirely
because every cluster above covers the defect surface and these are not defects.
Two of them justify themselves by the same kernel rule: correctness must not
depend on LLM behaviour, and a relationship the kernel cannot read is enforced
only in someone's head.

- **G-0060** *(high)* — a patch is real, common work with no entity kind, no FSM,
  no trailer, and no way to record which gap it closed. Four resolution options
  spanning wontfix to a seventh kind; the choice is open
- **G-0073** *(high)* — `depends_on` is milestone→milestone only, so every
  cross-kind blocking edge (epic on ADR, ADR on ADR, epic on epic) lives in body
  prose. `aiwf promote` cannot see it and render cannot draw it
- **G-0311** *(high)* — no tier above epic, and since areas landed an epic is an
  area-atom, so cross-cutting work splits into peer epics with nothing naming the
  capability. Two sibling projects already hand-rolled the same missing concept
- **G-0111** *(high)* — the wrap side of the epic ritual: scope-end bundled into
  `promote done`, no human-only rule on `done`, and no mechanism to land the gap
  closures the epic's own spec claims. Recommends its own epic

### 11. Doc drift *(one wf-patch, whenever)*

Citations that no longer resolve. Cheap, isolated, no thesis between them beyond
staleness.

- **G-0412** — coverage-ignore rationale text is inaccurate across several files
- **G-0414** — stale test naming in a real-binary test
- **G-0417** — dead branch-not-found code and stale spec-table rows
- **G-0436** — `CLAUDE.md` and the id-allocation doc cite relocated paths
- **G-0439** — relocation and archive sweeps skip references outside their own scope
- **G-0444** — the id-allocation doc cites renamed functions
- **G-0517** — narrow id citations remain in the design docs, overview and
  architecture. Unblocked — G-0559 landed, so the surface they were copied from
  now prints canonical width. Pairs naturally with G-0518: one sweep, two surfaces
- **G-0519** — documentation is not reference-checked; a cited id need not resolve
- **G-0548** — shipped surfaces cite this repo's own paths; the id half of that
  rule is enforced at error severity, the path half has no check

### 12. Development environment — the machine the gates run on *(unscoped)*

A full disk surfaces as failures in whichever tests write most, which is the
binary-building and repo-creating ones. That signature reads as a flaky suite,
so the response it invites — re-run, then hunt a race — is aimed a layer below
the cause. Neither member is a kernel concern; both cost review cycles, and a
green run on a nearly-full disk is not evidence either way.

- **G-0552** — nothing bounds the Go build cache; it reached 84 GB and filled
  the overlay. Carries the preflight idea: a gate that cannot tell a full disk
  from a broken test will mislead every time
- **G-0555** — two test helpers build into a temp dir and never remove it. Over
  a gigabyte in a day here, and every directory younger than any age cutoff a
  concurrent session makes safe — so no sweep can reclaim it and the fix has to
  be in the code
- **G-0554** — the statusline CI cache mints a file per commit and deletes none.
  Same thesis as G-0555: unbounded growth with no reclaim path
