# Policy System UX, Mining, and Compression: A Design Study

> **Status:** exploration — design study with concrete proposals, including kernel-shaped commitments. Not a final spec; the targeted design session for `docs/design/policy-model.md` is downstream.
> **Audience:** anyone working on what the policy primitive's *operating surface* looks like — verbs, lifecycle UX, the agent-facing digest, and the question of where new policies *come from* in the steady state.
> **Hypothesis (tentative):** three commitments, taken together, make a policy system feel elegant in daily use rather than ceremonious: (a) a capture-triage-promote verb set with batch operations, (b) external mining (especially from skill repos) as a first-class source of candidate policies, and (c) a generated digest in a pipe-delimited dense format that compresses well for an LLM without sacrificing scannability for a human. The third commitment is what makes the agent-side workable; the first two are what make the policy population *grow with the project* without becoming a tax on the team.
> **Tags:** #aiwf #policies #ux #mining #compression #design-study

---

## What this is, and what it isn't

This is a design study. It picks positions on three questions the prior
explorations left open, names the verbs and formats those positions imply, and
flags the kernel-level commitments that follow. It is **not** the targeted
design session that produces `docs/design/policy-model.md`; that session has
broader scope (form, supersession semantics, governance boundary). This doc
narrows to three concrete sub-questions:

1. **Lifecycle UX** — what does it *feel like* to add a new policy, contest an
   old one, waive one, or promote a soft-signal to blocking, when this happens
   weekly rather than yearly?
2. **Mining** — where do new policies *come from* in steady state, beyond
   the in-session capture flow? Specifically: can the framework mine policies
   from public skill repos, and what does that imply about provenance?
3. **Compression** — how does the agent-facing digest fit ~80-200 active
   policies into context without bloat, given that the consumer is an LLM and
   we control the encoding?

The shape:

1. The lifecycle verb set and what makes it elegant.
2. Mining from external sources, with skill repos as the primary case.
3. The digest format — what compresses, what doesn't, and the proposal.
4. The editorial filter — which policies earn a digest entry at all.
5. Kernel commitments that follow from the above.
6. What this leaves to the targeted design session.

---

## 1. The lifecycle UX

The corpus mining surfaced a population characteristic: policies are added
*frequently*. Both Liminara and FlowTime show ongoing policy work — new ADRs
land, new gaps get filed, old conventions get renamed. Steady-state addition
rate is plausibly several per week in an active repo. **An elegant policy
system has to make per-policy operations cheap, batch-friendly, and reversible
— or it becomes a tax the team works around.**

The verb set, with proposed shapes:

### 1.1 Capture is one keystroke

```
aiwf policy capture --from-conversation
aiwf policy capture --from-finding <ref>
aiwf policy capture "rule headline here"
```

- `--from-conversation` — the agent looks at the last few turns, drafts a
  candidate policy file, populates frontmatter from context (subject from the
  files touched, severity guess from the language used, scope from path
  globs), commits with status=`proposed`. The user reviews on next triage.
- `--from-finding <ref>` — the verifier just fired a finding the user
  considers a *new policy in disguise* (the finding revealed a missing rule
  rather than violating an existing one). Capture turns the finding into a
  candidate policy entity; the finding gets a back-reference. The `<ref>`
  shape is discussed in §1.8 below — most of the time it's an ephemeral
  `<commit>:<file>:<line>` triple, occasionally a tracked `F-NN` id.
- Bare-arg form — when the user knows what they want and doesn't need the
  agent to draft it: a one-line headline, the rest defaulted, status=`proposed`.

**Capture is cheap because most captures get triaged away, and that's fine.**
The cost of a bad capture is a triage rejection; the cost of a missed capture
is an unwritten rule.

### 1.2 Triage is a batch operation

```
aiwf policy triage
```

Walks every `proposed` policy in one pass, presents them as a structured Q&A
list, and lets the user accept / reject / edit / merge / defer / supersede in
one sitting. UX shape: same as `git rebase -i` — a list, one keystroke per
entry, batch commit at the end. **Frequency drops the per-policy cost
dramatically when the operation is batch-shaped.** Triaging twelve captured
proposals in one ten-minute session is sustainable; triaging one proposal at a
time across twelve micro-interruptions is not.

The triage verb is also where *conflict surfaces eagerly* (§1.5): if a
candidate would contradict an existing accepted policy, triage refuses to
accept it without explicit reconciliation (supersedes, scope-narrows, or
declared co-existence with precedence).

### 1.3 Promote is one keystroke and *scaffolds the enforcer*

```
aiwf policy promote P-NN accepted
```

Runs the FSM check (`proposed → accepted` is a legal transition), allocates
empty rung-2 enforcer scaffolding based on the policy's declared
`enforces[]` shape (an empty test file, an empty CUE schema, an empty grep
guard — whichever the policy declares), commits with provenance trailer.

**Critically: scaffolding being *empty by default* is the load-bearing
choice.** Promotion does not require the enforcer to *exist*; it requires it
to be *named*. Filling in the enforcer body is a follow-up, not a blocker.
This decouples *ratifying that something is a rule* from *implementing the
mechanical check*, which today are conflated and pay double cost.

### 1.4 Waiver is trivial to issue, expensive to forget

```
aiwf policy waive P-NN --reason "..." --until 2026-09-01
aiwf policy waive P-NN --reason "..." --scope "src/legacy/**"
```

Writes a waiver entity, commits with provenance trailer
(`aiwf-policy-waived: P-NN`, `aiwf-policy-waiver-until: 2026-09-01`).

**The asymmetry matters.** Issuing a waiver is one verb. *Forgetting* a waiver
is loud: the policy reapplies after `--until`, and any commits taken under the
waiver re-surface as findings on the next verify pass. The verifier does *not*
silently grandfather waiver-covered violations after expiry. This is the
equivalent of OPA's bundle-version pinning, applied per policy.

Waivers can be *scoped* (path-glob, milestone-id, principal) instead of
*time-bounded* — both shapes write the same entity with different active
fields. The kernel rule: **every waiver names at least one expiry condition
(time, scope, or both); a waiver with no expiry is rejected by the verb.** No
permanent waivers, ever.

### 1.5 Conflict surfaces eagerly, not at verify time

When triage accepts a new policy, the kernel walks the existing accepted set
for conflicts. Two policies conflict when their scopes overlap and their
required-state claims disagree. Examples:

- New: `JSON casing must be camelCase` (scope: `**/*.json`).
- Existing: `Telemetry manifests use snake_case` (scope: `runs/**/manifest.json`).
- **Conflict.** The scopes overlap (`runs/**/manifest.json` ⊂ `**/*.json`),
  the required casings disagree.

Triage *refuses to accept* until reconciled. Reconciliation has three
canonical shapes: **supersedes** (new replaces old; old goes to `superseded`),
**scope-narrows** (new explicitly excludes the existing scope; old keeps its
narrower territory), or **declared co-existence with precedence** (both stay
accepted; one is named as winning where they overlap, recorded on the policy
entity).

The kernel rule: **conflicts surface at acceptance time, not at verify time.**
A finding that says "two of your accepted policies disagree about this file"
is a framework bug, not a user error.

### 1.6 The digest regenerates as a hook

Every `aiwf policy *` verb that mutates the active policy set (`promote`,
`waive`, `supersede`, `retire`) regenerates `.aiwf/policy-digest.md` as part
of the same commit. The kernel rule: **the digest never goes stale because the
consumer cannot forget to regenerate it.** A precommit check refuses commits
where the policy set has changed but the digest hash hasn't.

### 1.7 The lifecycle FSM

For completeness, the proposed status set:

```
proposed → accepted → in-effect → [waived | superseded] → retired
                ↑                        ↓
                └──── revised ───────────┘
                  (in-place edit; same id)
```

Same shape as the design-space §5 sketch, with `revised` as an in-place edit
that does not change id (used for clarifications that don't change semantics —
typo fixes, rationale expansions). Semantic changes go through supersession
(new id, old `superseded_by` chain).

### 1.8 Finding identity and the relation to gaps

Several verbs above reference findings (`capture --from-finding`, the waiver
asymmetry, the digest-coverage feedback loop). This subsection settles what a
finding *is* — and how it relates to the existing `gap` entity, which is the
question the design must not duck.

**Findings are ephemeral by default.** A finding is the verifier's per-run
output: a `(policy-id, file, line, counterexample)` tuple emitted when a
policy's enforcer fires. The next verify run produces a fresh list. There is
no per-run id allocation, no entity store, no kernel state. This keeps the
verifier cheap and the storage shape predictable.

**Reference shape for ephemeral findings is `<commit>:<file>:<line>` plus
the policy id.** Example: `(per P-2 at abc123:src/Foo.cs:42)`. Verbose, but
stable across re-runs of the same commit and unambiguous in conversation. No
fingerprint needed.

**A finding can be *promoted* to durable status by one of two paths:**

1. **`aiwf gap add --from-finding <ref>`** — the finding revealed a *structural*
   problem, not just a one-off violation. The gap inherits the existing G-NN
   identity, status set (`open` → `addressed | wontfix`), and narrative shape.
   This is the dominant path: most "the finding is worth tracking" cases turn
   out to be "we don't have a rule for this class of thing yet" or "the rule
   is right but our implementation has structural holes" — both gap-shaped.
2. **`aiwf finding track <ref>`** — the finding is a *specific instance* worth
   tracking durably (a known regression we're watching, a deferred fix with a
   named owner). Allocates an `F-NN` id with its own small status set
   (`open` → `fixed | waived | promoted-to-gap | promoted-to-policy`). Less
   common; reserve for "this exact instance, not the class."

**G-NN and F-NN are sibling kinds, not the same thing.** A G-NN is *narrative
and structural* — "tests are too weak: surveyed-output-only canaries cannot
detect drift" (a real FlowTime gap). An F-NN is *mechanical and instance-bound*
— "policy P-2 fired at this commit, this file, this line, and we are watching
it." A G-NN may *address* multiple F-NNs that all clear when the gap closes;
an F-NN may *promote to* a G-NN when its underlying issue turns out to be
structural; an F-NN may *promote to* a P-NN policy proposal when it reveals a
missing rule. Each promotion is a verb; each leaves a trailer.

**The fingerprint approach is rejected.** A content-derived id sounds elegant
but breaks under refactors (renaming a file changes the fingerprint, breaks
history), under policy evolution (scope-narrows changes the fingerprint
interpretation), and is not human-readable in conversation. The
`<commit>:<file>:<line>` reference is enough for ephemeral cases; promoted
findings get real ids. Same posture aiwf already takes for entities (the
kernel allocates; the agent never invents).

**When does a finding "surface"?** Whenever the verifier runs:
- **Pre-commit / pre-push** — the highest-leverage surface; the agent or
  human catches violations before they reach the remote.
- **In CI** — backstop for what slipped through.
- **On demand via `aiwf policy verify`** — the agent's continuous-feedback
  channel during work; see [`03-policy-corpus-mining-and-the-agent-side.md`](03-policy-corpus-mining-and-the-agent-side.md)
  §5 on autonomous runs.
- **Via `wf-policy-sweep`** — the convention skill at handoff boundaries
  (also covered in doc 03 §5).

The kernel rule that follows: **findings are tool output, not entities, until
explicitly promoted.** Promotion goes through `aiwf gap add --from-finding`
(default) or `aiwf finding track` (instance-specific). The framework grows no
new entity kind for ephemeral findings; the existing gap covers the structural
case; F-NN is a small new kind for the rare instance-tracking case.

---

## 2. Mining policies from external sources

Capture handles in-session policy creation. The other source — and the more
strategically interesting one — is *mining* policies from artifacts that
already contain them, especially **public skill repositories**.

### 2.1 The mining hypothesis

Skills today carry MUST/SHOULD/MAY claims throughout their bodies — "the
skill MUST exit 0 from the audit path," "wrap-milestone SHOULD invoke
dead-code-audit as a non-blocking step," "tests MUST be deterministic." These
are *policy claims dressed as procedural prose*, and they are exactly the
"skill straying into policy territory" pattern the design-space §8 names.

If skills are a delivery vehicle for policies, the framework should treat
them as such: **mine the policy claims out of skill bodies and turn them into
first-class policy entities**, with the skill itself becoming the *procedure*
and the extracted policies becoming the *constraints the procedure
honors*.

### 2.2 The verb shape

```
aiwf policy mine <skill-source>
aiwf policy mine .ai/skills/dead-code-audit/SKILL.md
aiwf policy mine github.com/anthropic-ai/claude-code-skills/foo
aiwf policy mine ./node_modules/some-bundle/skills/
```

For each mined skill, output one `proposed` policy file per extractable
normative claim, citing source path, source line range, and source revision
(commit hash for git sources). The claim itself is paraphrased into the policy
headline; the original sentence is preserved in the policy body for
review-time fidelity.

Triage proceeds normally: accept / reject / edit. Bulk operations on mined
batches are the common path ("accept all the dead-code-audit policies as a
group" is one keystroke).

### 2.3 Provenance gets a new actor type

The trailer family extends:

```
aiwf-policy-source: skill@anthropic-ai/claude-code-skills/dead-code-audit@sha:abc123
aiwf-policy-source-line: 42-58
aiwf-policy-mined-at: 2026-05-03T14:22:11Z
```

The principal × agent × scope model already accommodates this — the principal
is the human who triaged; the agent is the framework-binary that mined; the
scope is the bulk-mine operation. **No new provenance shape needed; the
existing one stretches.**

### 2.4 Upstream updates flow with explicit consent

```
aiwf policy resync <bundle>
```

When the upstream skill changes, `resync` re-mines and *diffs against the
local accepted set*. Three outcomes per upstream change:

1. **Pure addition** (new claim upstream that doesn't exist locally) — surfaces
   as a new `proposed` policy in the next triage.
2. **Modification** (existing local policy was sourced from a now-changed
   upstream claim) — surfaces as a `revised` proposal that supersedes the
   local policy on acceptance.
3. **Removal** (local policy was sourced from a claim the upstream removed)
   — surfaces as a `retire?` candidate; triage accepts or rejects.

The asymmetry: **upstream changes never auto-apply.** The consumer always
triages. This honors the design-space §12 cross-project portability concern
("if a policy is ratified in project X and pulled into project Y, what does
'ratified' mean in Y?") with a clear answer: **mining is sourcing; ratification
is local; updates require consent.**

### 2.5 The framework ships a default policy library mined from its own skills

At framework-release time, every `wf-*` skill in the framework is mined; the
resulting policy bundle is what `aiwf init` installs into a new consumer
repo as the *defaults*. The consumer triages on first use and customizes from
there.

This is the consolidation move: **today, every consumer repo's CLAUDE.md
re-asserts the same TDD discipline / commit format / branch coverage rules.
After this move, those rules ship as a versioned, supersedable, waivable
policy bundle, and the consumer's CLAUDE.md just references it.**

### 2.6 The cross-repo policy bundle pattern

Beyond mining, *publishing* — `aiwf policy publish <bundle>` writes the local
accepted set to a publishable artifact (a folder, a tarball, a git tree),
which other consumers can `aiwf policy subscribe` to. Updates flow downstream
with the same consent semantics as §2.4. Org-standard policies become a
shared bundle; per-team customizations are local supersession. Same shape as
ESLint configs, Rego bundles, OPA bundles.

---

## 3. The digest format — compression for an LLM consumer

### 3.1 What's actually known about LLM compression

Two findings worth being clear about because they cut against intuition:

**Aggressive abbreviation does not save tokens.** Tokenizers handle "MUST"
and "must" essentially identically; "shall not" tokenizes as cheaply as
"shall not." Inventing terse synonyms (`@P-2: JSON.cc!`) saves *characters*
but those characters tokenize *worse* than English because they fall outside
the BPE vocabulary the model trained on. Pay-per-token, not per-character.

**Per-entry uniformity matters more than per-entry brevity.** A list of 200
uniform three-line entries is faster for the model to read than 200 prose
paragraphs of equal length, because the model pattern-matches the structure
and skims entries it doesn't need. Format consistency is a compression vector
in itself.

### 3.2 What plausibly compresses better than English

**Pidgin natural language with a fixed vocabulary** — pipe-separated
key-or-value entries with a small legend. The format is structured enough that
the model parses it instantly (it has seen pipe-tables and key:value pairs
trillions of times) and dense enough that ~40-60% fewer tokens than prose-form
entries.

**Symbolic shorthand for the bindingness layer**, defined once at the digest
header. RFC 2119 keywords collapse to single sigils — `!` for MUST, `~` for
SHOULD, `?` for MAY, `⊘` for MUST NOT. Costs ~30 tokens for the legend;
saves a token per entry; reads cleanly.

**Scope and trigger as glob/regex**, not prose. "When a milestone is promoted
to done" in EARS is six tokens; `event:promote(milestone,*,done)` is roughly
five and parses unambiguously. Small per-entry savings; consistent across all
entries.

**Policy-id-only references inside model reasoning.** The digest convention
encourages the agent to write "(per P-2, P-11)" instead of restating the
rules. Saves tokens in the model's *outputs*, not just inputs; also makes
the agent's reasoning citation-traceable for the verifier.

### 3.3 What probably won't work

- **A custom DSL invented for this purpose.** Compression budget is eaten by
  the in-context teaching budget. Exception: a DSL isomorphic to a structure
  the model has seen (CUE, JSON Schema, YAML frontmatter) costs nothing.
  Inventing genuinely new syntax is a tax.
- **Binary / base64 / embedding-shaped representations.** Tokenizers handle
  these badly; designed for text.
- **Loading the digest as model-side context (fine-tune per consumer).**
  Theoretically free at runtime; practically: policy churn is weekly, fine-tune
  cycles are not, and citation-by-id breaks.

### 3.4 The proposed digest format

```
# .aiwf/policy-digest.md  (generated by aiwf policy digest; do not edit)
# Legend: ! = MUST, ~ = SHOULD, ? = MAY, ⊘ = MUST NOT
# Format: id | subject | bindingness rule | scope | severity (rung)
# Cite policies in your reasoning as (per P-NN). The verifier will check.

[engineering / style]
P-1 | naming.private | ! camelCase, no underscore | path: src/**/*.cs | block (analyzer)
P-2 | json.casing | ! camelCase | path: src/**/*.cs JSON, docs/schemas/** | block (test)
P-3 | namespace | ~ file-scoped | path: src/**/*.cs | warn (analyzer)

[engineering / test]
P-16 | tdd.phase | ! red→green→refactor for logic ACs | scope: ac.type=logic | block (aiwf check)
P-20 | ui.eval | ! Playwright in real browser | scope: milestone touches ui | block (CI)

[engineering / migration]
P-25 | migration.forward-only | ⊘ compat readers post-boundary-change | scope: schema change | warn (judgment)

[project / numeric]
P-11 | nan.tier | ! pick tier 1/2/3 per div site | path: src/**/*.cs at div sites | block (test)
P-15 | pmf.invalid | ⊘ all-zero probabilities | path: src/**/Pmf.cs | runtime throw

[meta]
M-9 | truth.precedence | code+tests > decisions/ADRs > epic > arch > history | scope: conflict resolution | judgment
```

Properties:

- **~60% denser than three-line prose entries.** For ~150 active policies after
  the editorial filter (§4), this lands at ~3000-3500 tokens — comfortably
  within session budgets.
- **Fully uniform.** Every entry has the same shape. The model skims to the
  relevant section.
- **Scannable by humans.** Pipe-tables are not pretty, but they are *readable*.
  The legend is one line. The structure does not require training to parse.
- **Tokenizer-friendly.** Everything is ASCII the model has seen ten million
  times. No novel tokens.
- **Citation-ready.** The agent writes `(per P-11)` in its reasoning. The
  verifier can check that the agent's claimed-citations match the policies
  whose scopes apply to the change.

### 3.5 On EARS

EARS earns its keep in regulated industries because it has been audited and
certified for that use. For an AI-consumer digest, it is *verbose ceremony for
no compression gain*. "When X, the system shall Y" tokenizes to roughly the
same cost as a prose claim. The pipe-format above does the same job (named
trigger, named bindingness, named requirement) without the ceremony.

**Recommendation: adopt EARS for human-facing policy bodies where regulated-
industry interop matters; do not use EARS in the digest.** The two layers
serve different audiences.

---

## 4. The editorial filter — which policies earn a digest entry

The honest realization the compression analysis surfaced: **the choice of
*what* to digest matters more than the encoding of *how*.** A well-curated
digest of 80 high-leverage policies will outperform a perfectly-encoded digest
of 400 noise-floor policies, because the agent's attention is finite and
every entry costs cognitive load.

### 4.1 The proposed filter rule

> A policy earns a digest entry if violating it is *not auto-correctable by
> the verifier* OR is *expensive to remediate*.

Two implications:

- **Style rules that the formatter auto-fixes do not need a digest entry.**
  The formatter handles them; the agent does not need to think about them.
  The digest stays small by *trusting the autofixers*.
- **Rules where the agent needs to make a decision in advance** — picking a
  NaN tier, choosing camelCase vs snake_case for a new payload, picking
  forward-only vs compat for a migration — those earn entries because the
  cost of "do it wrong, fix on finding" is much higher than the cost of "know
  in advance."

### 4.2 The split

Walking the FlowTime + Liminara corpora through this filter, roughly half of
the 540 combined entries fall out:

- **Predictive policies** (~80-100 entries): the agent needs them in advance.
  Scope-deciding rules, tier-picking rules, naming conventions for new
  artifacts, migration posture, anti-patterns where the violation is visible
  in the structure of the code rather than a single token.
- **Reactive policies** (~150-200 entries): the verifier handles them
  cheaply. Style rules with auto-fixers, schema validations that produce
  clear error messages, presence-checks the verifier can re-run.
- **Soft policies** (~80-120 entries): judgment calls, escalation playbook,
  meta-policies. These belong in the *escalation* section of CLAUDE.md, not
  the digest itself.

The digest carries the predictive set. The escalation section carries the
soft set. The reactive set lives in the policy store, available on demand,
not pre-loaded.

### 4.3 The kernel rule

**Every policy declares its own digest tier on authoring.** Three values:
`predictive` (in the digest), `reactive` (in the store, not the digest),
`soft` (in the escalation section, not the digest). The author makes the call;
triage can override; the verifier reports digest-tier mismatches as findings
("policy P-12 is tagged reactive but has fired blocking findings six times
this quarter — consider promotion to predictive").

This makes the digest size *bounded by editorial choice*, not by corpus size.
A repo with 600 policies can still have an 80-entry digest if the editorial
discipline is held.

---

## 5. Kernel commitments

The proposals above imply commitments that have to live in the kernel, not in
optional layers. Listed for the design session:

1. **The lifecycle FSM is `proposed → accepted → in-effect → {waived |
   superseded} → retired`, with `revised` as an in-place loop on `in-effect`
   that preserves id.** Kernel-enforced; no consumer override.

2. **Promotion scaffolds enforcers but does not require them to be filled.**
   Empty test files, empty CUE fragments, empty grep guards count as legitimate
   enforcer pointers; the kernel does not refuse promotion based on enforcer
   emptiness.

3. **Every waiver names at least one expiry condition (time, scope, or both).
   Permanent waivers are rejected by the verb.** This is the
   "waivers expensive to forget" half of the asymmetry.

4. **Conflicts surface at acceptance time, not at verify time.** The kernel's
   triage / promote verbs walk the existing accepted set and refuse to accept
   a conflicting policy without explicit reconciliation
   (supersedes / scope-narrows / declared-co-existence).

5. **The digest regenerates on every policy-mutating verb, in the same
   commit.** A precommit check refuses commits where the active policy set
   changed but the digest hash didn't. The digest cannot silently go stale.

6. **Mining is sourcing; ratification is local; updates require consent.**
   Upstream skill / bundle changes never auto-apply; `policy resync` produces
   triage candidates, not direct mutations. The provenance trailer family
   extends to record `aiwf-policy-source` for mined policies.

7. **Every policy declares its digest tier (`predictive` / `reactive` /
   `soft`) on authoring.** The digest is bounded by editorial choice, not by
   corpus size. Tier-mismatch findings are produced by the verifier when the
   declared tier conflicts with observed finding-frequency.

8. **The digest format is a kernel commitment, not a consumer choice.** The
   pipe-delimited shape with a fixed legend is what the kernel emits. Consumers
   may add headers and grouping; they may not change the per-entry shape.
   This is the property that makes the digest readable across repos by the
   same agent without re-learning.

9. **Citations are checkable.** When the agent writes `(per P-NN)` in its
   reasoning or commit messages, the verifier can check that P-NN exists, is
   accepted, and applies to the change in scope. Hallucinated citations are
   findings.

10. **Capture is cheap, triage is batch.** The kernel ships both verbs as
    first-class operations. Per-policy operations outside `triage` are
    discouraged but not forbidden; the cost-of-bypass is just verb-typing.

11. **Findings are tool output, not entities, until explicitly promoted.**
    Promotion goes through `aiwf gap add --from-finding` (default; the
    finding revealed a structural problem) or `aiwf finding track` (rare;
    the specific instance is worth durable tracking). The framework grows no
    new entity kind for ephemeral findings; G-NN absorbs the structural case;
    F-NN is a small new kind for instance-tracking. Reference shape for
    ephemeral findings is `<commit>:<file>:<line>` plus the policy id; no
    fingerprint scheme.

---

## 6. The shape of an elegant session, end-to-end

To make the verb set concrete, a single hypothetical session walking the full
loop:

```
# Mid-implementation, the user notices a recurring pattern they want a rule for.
> aiwf policy capture --from-conversation
captured P-67 (proposed): "JSON payload field names use camelCase"
  source: conversation 2026-05-03T14:22:11Z (last 4 turns)
  scope guess: src/**/*.cs touching JsonSerializer
  severity guess: block (test)
  digest tier guess: predictive

# Later that day, six other things have been captured. The user runs triage.
> aiwf policy triage
6 candidate policies pending review.

P-67  JSON payload field names camelCase
      source: conversation; scope: src/**/*.cs; tier: predictive
      [a]ccept  [r]eject  [e]dit  [m]erge  [d]efer  [s]upersede  [?]help
> a
  CONFLICT: P-67 overlaps with P-23 ("telemetry manifests use snake_case")
  on scope runs/**/manifest.json. Reconcile:
      [n]arrow new scope  [s]upersede P-23  [c]o-exist with precedence
> n
  enter narrowed scope for P-67: src/**/*.cs ! runs/**/manifest.json
> [accepts]

P-68  ... (next candidate)
...

triage complete: 4 accepted, 1 rejected, 1 deferred
digest regenerated; 154 entries (was 150)
1 commit landed: "policy: triage 6 candidates, accept 4"

# Two weeks later, an exception case.
> aiwf policy waive P-67 --scope "src/legacy/**" --reason "legacy serializer; tracked in #247"
waiver written; P-67 now scope-waived for src/legacy/** until issue #247 closes
1 commit landed: "policy: scope-waive P-67 for src/legacy/**"

# Three months later, an upstream skill changed.
> aiwf policy resync skill@anthropic-ai/claude-code-skills@v1.4.0
3 changes pending review:
  + P-MINED-04 (new): "Soft-signal skills must always exit 0"
  ~ P-MINED-12 (revised): finding-class names changed
  - P-MINED-19 (removed): "Recipe bodies must include blind-spot families"
> aiwf policy triage
[walks the three changes]
```

The lifecycle in §1, the mining in §2, the digest regeneration in §3, the
editorial filter in §4 — all working together, mostly invisibly, with the
user's friction concentrated at *triage* (intentionally) and amortized
elsewhere.

---

## 7. What this leaves to the targeted design session

This study takes positions on the *operating surface* of the policy system —
the verbs the user touches, the artifacts they produce, the digest the agent
consumes. It deliberately does not pick positions on:

- **The form of the policy entity itself** (single kind vs discriminator on
  contract vs new ADR-superset). The verbs in §1 work the same against any of
  these forms; the form decision is independent.
- **The substrate selection for enforcers** (CUE vs JSON Schema vs Rego vs
  pure code). The `enforces[]` pointer model accommodates all of them; the
  choice per policy is local.
- **Cross-project portability mechanics** beyond the mining + subscribe shape
  in §2. The bundle-versioning question, the registry-vs-vendoring question,
  the central-repo-vs-distributed question — these remain open.
- **Governance scope** — who has standing to ratify, how amendment of the
  policy system itself works, the boundary with the governance-design-space
  exploration.

Those are the design session's territory. The contribution here: eleven kernel
commitments concrete enough to test against the form decision, four worked
verbs that feel sustainable in daily use, a digest format that fits the
agent's context budget, and an editorial filter that holds digest size bounded
as the policy population grows.

---

*This document is a design study, not a specification. The verbs, formats,
and kernel rules above are concrete enough to argue about, not concrete enough
to ship. The next bridge is the vertical slice proposed in
[`03-policy-corpus-mining-and-the-agent-side.md`](03-policy-corpus-mining-and-the-agent-side.md)
§7 — pick one real policy (the FlowTime NaN policy is the recommended
candidate), express it through the verb set above, generate its digest entry,
and see what survives contact.*
