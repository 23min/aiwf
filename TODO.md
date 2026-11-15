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

- any more tdd related?
- Area skill perhaps too big?
- Do we have anything in CLAUDE.md that really should belong in shipped surfaces?
- E-0019 and ADR-0001 both still `proposed` — deferred, no forcing function yet
- `readStatusAt` and the walker's path-resolving fallback arm are dead in
  production — reachable only through conditions no BulkRevwalk flag produces.
  Annotated rather than deleted, deliberately; nothing tracks the choice

## Next, in order (2026-11-15)

Clusters 1–9, 11 and 13 cover the *defect* surface; cluster 12 covers the machine
the gates run on. Cluster 10 is the exception: absences in what the kernel can
express, which are not defects and each need a decision first.
Feature and enhancement gaps fit none and are not listed, at any priority — the
filter is what the gap is, not how urgent it is. Each gap sits in exactly one
cluster.

In flight: nothing.

Planned, not started: **E-0096** — one repo-only gate for everything aiwf ships to
consumers.

Cluster 1 is split three ways below, by the three shapes its own thesis already
named.

### 1. Lying gates

A gate that reports green while proving nothing is worse than an absent one:
every commit landing before the fix gets a false pass, and a reader auditing
coverage finds a guard and stops looking. The largest cluster, and the members
fall in three shapes: a scan with a hole in its reach, a gate that fires on the
wrong thing, and no gate at the point one would still be cheap.

#### 1a. A scan with a hole in its reach

The gate exists and runs; what it looks at stops short of where the defect is.

- **G-0744** *(high)* — the untrailered-entity audit skips any commit carrying an
  `aiwf-verb` trailer, so a hand edit folded into a verb commit by a squash or
  fixup passes it and drops out of `aiwf history`
- **G-0765** — a `--no-ff` merge hides the merged branch's commits from the
  untrailered audit; `wf-patch` merges that way and never pushes the branch.
  Pairs with G-0539 and G-0679
- **G-0302** *(low)* — `check --fast` skips in-memory contract-config validation
- **G-0307** *(low)* — the top-level `aiwf.yaml` decode stays non-strict; the areas
  and `docs:` blocks reject an unknown key, the top level does not
- **G-0535** — three more policies scope their AST walk to `cmd/aiwf`, one file with
  no tests; two inspect nothing and the third is degenerate rather than vacuous
- **G-0550** — a commit carrying `aiwf-force` and no actor at all is refused by
  neither rule set: both test the actor for being non-human, which absent is not
- **G-0576** — `acs-tdd-tests-missing` reads an AC's whole history, so one cycle's
  trailer satisfies it for that criterion's entire life
- **G-0625** — lychee's `exclude_path` entries are read as regexes, dropping eight
  Normative paths out of the link gate's reach
- **G-0623** — outbound link repair covers single-file moves only, so a move that
  relocates a whole directory leaves ADR-0046's commitment unmet. E-0088's residue
- **G-0671** — the changelog audit does not see a delta in a named kernel surface
- **G-0705** *(low)* — `BulkRevwalk` ends the walk silently at a `git log` line of
  1 MiB or more and returns the records before it as the whole history
- **G-0726** — the `go` workflow's path filters omit files the policy suite reads, so
  a push changing only such a file runs no policy that judges it

#### 1b. A gate that fires on the wrong thing

The gate runs and returns a verdict about a subject that is not the one claimed.

- **G-0751 / G-0752 / G-0755** *(high)* — authorize scopes are keyed by the opening
  commit's SHA, so a rebase turns every scoped commit into an error (G-0751),
  reopens an ended scope (G-0752), and a copy of the opening commit opens a second
  scope (G-0755). ADR-0056 decides the fix, a scope id of its own; not built
- **G-0766** — `aiwf add` judges an `ai/` create by the flags it was given and
  `aiwf check` by what the entity stores, so a create admitted by a `relates_to` or
  `depends_on` link commits and then fails the push
- **G-0629** — nothing fails when an m0127 allowlist entry stops matching the file
  it exempts; the exemption outlives its subject and the scan still reports green
- **G-0573** — **already addressed.** `29eb2a94c` composed all four severity passes
  into the guard, and ADR-0042 rules the behavioural residue moot rather than
  pending. Promote `addressed --by-commit 29eb2a94c`
- **G-0679** — a branch started from a local ref is never judged at its first push
- **G-0731** — in a shallow clone, a commit whose parent is past the boundary reads as
  adding every file it carries, to both the instruction fence and the untrailered audit

#### 1c. No gate where one would still be cheap

Nothing fires at the point the defect is cheapest to catch.

- **G-0539** — a plain commit staging entity files passes pre-commit, and the finding
  it produces never surfaces in the default flow at all — not at the push, and not
  after, once the upstream is set
- **G-0767** — `aiwf add` accepts `--relates-to`, `--epic` or `--discovered-in` on a
  kind that does not store it, exits 0 and drops the link
- **G-0640** — nothing surfaces an existing gap at filing time, so duplicates land.
  The one entry in this file that reduces how many gaps get filed rather than how
  many are wrong; G-0562 and G-0578, filed five days apart against one call site
  with neither referencing the other, are what it would have caught
- **G-0638** — forcing past the epic branch guard leaves no `aiwf-force:` record and
  meets no human-only check; the reason `--force` demands is discarded
- **G-0658** — nothing pins that `--root` reaches the commit-msg guard that reads the
  index
- **G-0661** — no check caps an in-function comment at the length D-0084 sets
- **G-0663** — the confirmation round after a review finding has no chokepoint
- **G-0683** — no automated gate catches a deleted CHANGELOG entry before the release
  tag

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
  `docs/audits/normative-docs-drift-audit.md`
- **G-0595** *(high)* — every open gap, live ADR, live decision and Normative doc
  measured against the kernel; two thirds carry at least one finding. Dated inventory
  in `docs/audits/entity-truth-audit.md`; this gap tracks absorbing it. The
  2026-08-23 gap audit re-derives the entity half at a higher evidence bar
- **G-0590** *(high)* — `show`'s text renderer is the canonical per-entity view and
  drops the terminal reason that `history` renders; `--format=json` carries it
- **G-0591** — a sentence saying a concern is deferred to some entity makes a promise;
  nothing reports it when that entity is later cancelled
- **G-0589** — the doc authority tiering classifies every `docs/` subtree and leaves
  root and derived documents unassigned
- **G-0385** — `upgrade` resolves latest from the proxy list endpoint and under-reports
  right after a fresh tag
- **G-0421** *(low)* — `show --area` reports a milestone found on another branch as
  untagged, though its parent epic is in the area
- **G-0643** — `aiwfx-whiteboard` calls an operator ordering file uncommitted; this
  one is not
- **G-0667** — `aiwf import` is deprecated but every surface presents it as current
- **G-0701** — `README.md` lists the hooks `init` and `update` install and omits
  `commit-msg`
- **G-0706** — the provenance docs and skills give two rules for an unscoped
  non-human actor carrying a principal; the kernel refuses the combination
- **G-0689** — `aiwf --help`, the `init`-written `CLAUDE.md` and check text spell
  `aiwf add <kind> --title` alone, which `add` refuses for every kind but an epic
- **G-0729** — `design-decisions.md` and ANTI-0008 describe a `doctor.recommended_plugins`
  setting no code reads
- **G-0733 / G-0734** — the `aiwf-add` skill and the design docs state a narrower
  allocation and `body-prose-id` scope than the code runs. One claim in two places
- **G-0757** *(low)* — the docs say `adr-supersession-mutual` flags a missing
  successor; it never does. Same function as G-0756
- **G-0758** *(low)* — `edit-body --help` says any dropped section is refused; only
  required ones are


### 3. Oracles — what tells the builder it is wrong *(design first)*

Cluster 1 is the acute form of this: a gate that passes bad state. These are the
milder forms — a gate shallower than its name, a gate that judges its own author,
a gate whose red says nothing about the change under test, and one absence. The
apparatus decides *shape* well and decides nothing about whether the spec going
in is any good. Vocabulary in `docs/design/oracles.md`; initiative context in
`quality-signal-and-cadence.md` Q2 and Q6.

- **G-0768** — four actor tests isolate the global git config but not the checkout
  they run in, so a `user.email` set in the checkout fails them, saying nothing about
  the change under test
- **G-0720** — the `initcmd` and `update` in-process tests run against the real HOME and
  PATH, so an installed `dotfiles-guidance-check` or personal instruction file decides
  their verdict. G-0768's shape, for HOME and PATH
- **G-0282** *(high)* — "what verb undoes this?" has no chokepoint; code review is
  the whole enforcement, and it never audits verbs that already shipped. The
  filed form of the agent-performed audit two entries below
- **G-0565** — an archive-migration test's verdict turns on refs its fixture never
  stages. G-0468's family: an oracle answering about something other than its
  subject
- **G-0533** — `dupl` is off across the test corpus, the larger and faster-growing
  half. The body's first move is the *threshold raise*, not diff-scoping, which it
  prices at a config file, a Make target, hook and CI wiring, and a host CI lacks.
  Unblocked — G-0462 repaired the instrument this was waiting on
- **G-0121** *(high)* — legal workflows and verb composition are not pinned
  mechanically. Listed again: E-0080 was the mechanized half and is cancelled, so
  nothing closes this at anyone's wrap
- **G-0583** — the milestone preflight asks the reader to confirm every criterion is
  concrete and testable and supplies no method for deciding that. The filed form of
  the spec-adequacy absence below, which has since been attempted and withdrawn
- **G-0585** — shipped skills grant a verdict on the basis of reading; the appendix
  listing the sites is itself a reading rather than a census
- **G-0574** — finding identity concatenates the message and several rules interpolate
  live state into it, so a verb self-refuses while making progress. The root cause of
  the refusals in G-0572 and G-0575
- **G-0577** — `PolicyFSMInvariants` guards terminal preservation and acyclicity with
  one mechanism
- **G-0400** — the stress catalog reaches under half the leaf verbs; the body's own
  Notes already correct the counts its headline carries
- **G-0537** *(low)* — aiwf declares no contracts, so `verify` and `evolve` never run
  against this repo
- **The branch-coverage audit is agent-performed** — `wf-tdd-cycle` is explicit
  that no tool enforces it, so the agent that wrote the code certifies its own
  coverage. Independence, not depth *(unfiled)*
- **Wiring `make mutate-diff` into the AC's green→done** is the compensating
  control for the one above — an independent verdict at the moment an AC is
  claimed done. Unblocked; the target already exists *(unfiled)*
- **Spec adequacy has no oracle** — nothing decides whether an AC states an
  observable condition. Attempted once: `wf-measure-spec` shipped 2026-08-14 and was
  withdrawn the next day as contradicting the spec it was built from; E-0085 and
  M-0309 are cancelled. Reconcile with E-0019, D-0066,
  `tdd-cycle-subagent-boundaries.md` and
  `milestone-preflight-as-independent-review.md` before reopening *(unfiled)*
- **G-0697** — `wf-vacuity` has no probe for an expectation computed by the code under
  test, which moves with the implementation and cannot fail


### 4. Cheap fixes, batchable now

Small and design-free down to the rule.

- **G-0696** — three AC progress rollups count a `deferred` AC in scope forever — the
  rollup half of what G-0464 fixed for the check predicates
- **G-0502** — an *uninitialized* submodule under a moved directory is stranded; a
  checked-out one refuses at exit 3 instead
- **G-0477** *(low)* — a boundary guard that can never be false, under a comment
  describing the rejection the guard above it already performs
- **G-0497** *(low)* — the sites the `test-executable-write` detector reports
  whole-tree, across 13 packages, remain outside `WriteExecutable`. Most of
  `internal/initrepo`'s share runs `sh <path>` rather than `execve`, so that part is
  uniformity debt rather than race exposure
- **G-0762** *(low)* — the HTML render code still fills an all-set page it no longer
  writes, and every index page is headed "All …" above an Active/All toggle

### 5. Write scope — what a verb may commit *(spec first)*

A verb commits bytes it did not compute. The guard shipped; these are the routes it
cannot see, plus the missing verbs that are the upstream cause. Where no verb
reaches a field, hand-editing is the usual route and the guard then refuses the
result — though `import --on-collision update` reaches some of them with full
trailers.

- **G-0168** — three set-at-create fields have no mutation verb; `tdd:`, the only one
  with a real friction report, got `aiwf milestone tdd` in 2026-07. D-0048 defers the
  rest as consistency gaps rather than friction
- **G-0500** *(high)* — `edit-body` over a hand-moved file lands a duplicate id the local check misses
- **G-0442** — `addressed_by` / `superseded_by` can be backfilled and force-re-pointed
  but never *cleared*
- **G-0756** — a forced re-point of `superseded_by` leaves a stale reciprocal that
  `adr-supersession-mutual` structurally cannot see. G-0757 is the same function's doc
- **G-0486** — a directory move drops the exec bit, and `rename-area` drops it on
  `aiwf.yaml` with no move at all and no permanent-`M` signal. The symlink half now
  refuses at exit 2
- **G-0572** *(high)* — two verbs, each legal where it ran, compose across a branch
  boundary into an AC state the kernel reports at error severity whose only
  criterion-preserving repair is sovereign. Sits with G-0121's composition axis
  rather than this cluster
- **G-0544** *(high)* — four contract verbs reach the commit seam without passing
  through the provenance-decoration layer
- **G-0575** — a clean merge of two legal branches closes a milestone while adding
  criteria to it
- **G-0498** *(low)* — verb commits write blobs directly, so git's clean filter never runs
- **G-0739** — `retitle`, `rename` and `add ac` rewrite every same-numbered
  `### AC-N` heading in the body, so a per-criterion `## Validation` subsection changes
  silently
- **G-0713** — the `aiwf.yaml` append writers break a flow-style or indented root;
  `add` and the contract verbs commit the result

### 6. E-0074 — same-state convergence *(parallel any time)*

Finish what the convergence milestone started, so the convention the kernel
advertises holds everywhere it claims to. The allowlist's OPEN entries are the
work list for three of the four, but not a done-condition: nothing reads their
reason strings, and G-0461 has no entry and correctly never will.

- **G-0460** *(high)* — a repeat `authorize` leaves two active scopes and no finding
- **G-0459** — five event-shaped verbs append a duplicate record on an identical re-run
- **G-0461** — a composite `--for-entity` ack suppresses nothing

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
  checkout, and deciding whether it runs on planning-only pushes. Size it with
  G-0679 and G-0731: such a job judges what G-0679 leaves unjudged, and meets G-0731
  under a shallow checkout
- **G-0754** *(high)* — worktrees made by `aiwf worktree add` sit under
  `.claude/worktrees` with no managed ignore pattern, so `git add -A` commits a broken
  gitlink in a consumer repo
- **G-0743** *(high)* — in a linked worktree the repo lock lands in the project tree,
  where no ignore rule covers it, so `git worktree remove` refuses and `git add -A`
  stages it
- **G-0761** *(high)* — the hook installers overwrite a user hook that merely mentions
  an aiwf marker, with no copy kept
- **G-0370** *(high)* — ADR-0028 decided the always-on fragment should name
  dispatch triggers for all four role agents. The decision landed; the content
  never did, so the one surface in context every turn stays silent on it
- **G-0530** — the milestone template carries four sections that largely duplicate
  structured data. Nothing in the kernel mandates them — `RequiredSections` is Goal
  and Acceptance criteria — so this is template sprawl, and its `growth.md` half is
  already fixed
- **G-0600** *(high)* — `update` materializes guidance and skills from the running
  binary without consulting disk, so a stale binary moves them backwards and reports
  success. E-0099 cancelled a tagged-release guard, since no recorded incident was
  between two tags; inside this repo the pre-verb stale warning now fires, a consumer
  sees nothing
- **G-0586** — the handoff block carries settled conclusions and has no slot for an
  open question
- **G-0538** — aiwf-internal ids reach a consumer through Go string literals in
  operator-facing output; the persisted-artifact half is already fixed, so only the
  printed surface still leaks
- **G-0445** *(low)* — the diff-shape gate hardcodes a `docs/` exclusion that is wrong
  for some consumer repos
- **G-0235** — down to two unbuilt policy tests, `cited_entity_ids_resolve` and
  `cache_invalidation_documented`. Every `CLAUDE.md` item it was filed for has landed,
  been dropped, or been satisfied elsewhere; the body still reads as a doc sweep
- **G-0634** — `aiwf worktree` ships `add` and nothing that undoes it, and no
  surface says what does: not the skill, not ADR-0023, not any design doc. The
  repo's own verb-design rule requires the answer
- **G-0648** — wrap-epic's gap-closure precondition has no route for the epic's own
  spec
- **G-0688** — `aiwfx-wrap-epic` step 5 never fast-forwards local main when a worktree
  holds it
- **G-0698** — `aiwfx-plan-epic` and `aiwfx-plan-milestones` still route planning
  through a ritual branch and merge, contrary to D-0073
- **G-0727** — `doctor --self-check` runs `init` and `update` without `--no-prompt`, so
  at a terminal it waits on a question it never shows and `make ci` hangs
- **G-0738** — in Go tasks Claude reads `go-conventions.md` and rarely the Go pack, so
  a rule only the pack states goes unread. Measures E-0092's routing here
- **G-0724** — aiwf's managed `.gitignore` block omits `.claude/health.aiwf.json`, which
  aiwf writes. Same fix site as G-0743 and G-0754: the managed ignore patterns
- **G-0723** — `wf-trim` step 2 leaves step 6 unstated for a change with neither logic
  nor tests
- **G-0714** — guidance recovery advice names no fix for a corrupt ownership record.
  Reproduced only by injecting the fault; left until one is met in use


### 8. Error contract *(parallel any time)*

Everything a machine caller reads when a verb refuses. G-0483's exit-2/exit-3
boundary is the one D-0044 already ratified a three-class contract for.

- **G-0483** — uncoded verb errors exit 2, the usage bucket *(absorbed G-0494's
  prose-flattened refusal and split exit code)*
- **G-0568** — `status` rows omit `subcode`, so they cannot be matched against the
  findings envelope. Carry it when someone next touches the struct; the body declines
  severity outright and says to close this if no comparator materialises
- **G-0687** — `acknowledge illegal --for-entity` refuses a merge commit
- **G-0759** — verb-time `body-prose-id` findings give a body line number against the
  file's path


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
- **G-0454** *(low)* — the id-shape parser's prefix strip is still written three times;
  G-0633 closed the one disagreement between them. The body prescribes rather than
  asks, and the prescription is measurably not behaviour-preserving
- **G-0604** — five sites walk entities and stubs together, but only one resolves
  ids; the shared walk the body describes does not exist, and extracting to its
  description would widen four rules, one of which blocks pushes
- **G-0545** — two policies assert the same property about the same seam
- **G-0448** — the check rule list is split across four dispatch surfaces, and the
  split is a declared layering boundary rather than an accident. What is missing is a
  wiring chokepoint; `Where to fix` omits the surface carrying seven rules
- **G-0672** — two policies duplicate the git range scan over a commit range


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
  since epics carry no `depends_on`. `docs/initiatives/` is the in-tree workaround.
  The body's sibling-project evidence is external and unverifiable from here
- **G-0246** — the ADR relation schema expresses only supersession, and only ADR→ADR
- **G-0396** *(low)* — `addressed_by_commit` denormalizes a fact git owns, and the copy
  is rewrite-fragile: a merge preserves the SHA, a rebase orphans it. A
  single-source-of-truth issue rather than a vocabulary absence

### 11. Doc drift *(whenever)*

Mostly citations that no longer resolve — cheap and isolated. Two members are not
that: G-0519 and G-0548 are missing chokepoints rather than stale text.

- **G-0414** — stale test naming in a real-binary test
- **G-0444** — comments and test messages name history-verb code that no longer
  exists: `readHistory`, `runHistory`, `readHistoryChain`, `admin_cmd.go`
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
  declines. E-0088 scoped the `docs/` half out deliberately, and M-0317's review
  re-framed the body
- **G-0579** — D-0015's consequences cite a drift guard that no longer exists
- **G-0651** — backticked prose in a flag usage string renders as its value
  placeholder
- **G-0716** — the quality-signal initiative describes G-0110 by the new-file premise
  G-0110 now records as false
- **G-0717** — the same initiative credits `oracles.md` with *depth*, *reach* and
  *specificity*, which that file has never defined


### 12. Development environment — the machine the gates run on

The first three are one story: a full disk surfaces as failures in whichever tests
write most, which is the binary-building and repo-creating ones. That signature
reads as a flaky suite, so the response it invites — re-run, then hunt a race — is
aimed a layer below the cause. None of these is a kernel concern; all cost
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
- **G-0372** — the history-dependent check rules walk all reachable history from
  scratch on every push
- **G-0730** — `aiwf check` runs a whole-repository `git fsck` once per hinted history
  finding. Same family as G-0372
- **G-0763 / G-0764** — the `flake-hunt` workflow runs the suite without `aiwf` on
  PATH, and a run on any branch but `main` lacks `origin/main`; both fail packages for
  setup reasons rather than flakiness. One workflow file


### 13. Id collisions and their repair *(after ADR-0001)*

Two clones can mint the same id. Around a collision, the detectors and the repair
path misplace references, history or the entity itself, and most of it passes every
check. ADR-0001, still proposed, would mint ids at trunk integration and shrink the
whole class; until it is decided, each member stands on its own.

- **G-0745** *(high)* — when `aiwf reallocate` runs on a tree holding both colliding
  entities, it rewrites every reference to the old id, so the winner's referrers end
  up pointing at the renumbered loser; every check passes
- **G-0747** *(high)* — two same-id entities at the same path merge into one file with
  no finding
- **G-0746** — once a duplicate id is on the trunk, renumbering one copy is refused at
  pre-push, the hint's own repair included
- **G-0748** — after a collision is reallocated, `aiwf history` lists the other
  entity's commits
- **G-0753** — `aiwf reallocate` does not rewrite a contract's `linked_adrs`
- **G-0434** — `resolveViaPriorIDs` prefers a stale `prior_ids` match over a reused
  live id; G-0748's reader-side twin
