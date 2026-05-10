# Provenance model (I2.5)

This is the canonical writeup of aiwf's provenance model: who acts, on whose authority, within what scope. It is the design context that the `provenance-model-plan.md` build plan implements.

The model is added in iteration **I2.5**, which sits between I2 (acceptance criteria + TDD) and I3 (governance HTML render). The I3 render's Provenance tab consumes this model directly; without it, the tab can render only single-actor history and loses the "implement E-0003" autonomous-scope use case entirely.

If a proposed change conflicts with anything below, treat it as a kernel-level decision and surface it explicitly.

---

## Why a model

Today's aiwf records a single `aiwf-actor:` trailer per commit. That works for direct human acts and breaks for everything else:

- **Human directs LLM to run a verb.** The trailer says `ai/claude` (the operator). The human (the accountability-bearer) is invisible.
- **Bulk import.** One commit, N entities, one actor. Per-entity authorship is collapsed.
- **Authorized autonomous work.** A human says "implement E-0003 end-to-end"; the LLM runs many verbs without per-verb approval. The authorization itself isn't recorded; subsequent commits can't be distinguished from interactive ones.
- **Pivots and pauses.** Mid-implementation the human pauses E-0003 to address a blocker on E-0009; later they resume. The pivot is invisible in trailers.

The model below addresses all four. It is not a generalization of "actor" into a richer field — it is an **accountability layer** layered on top of operator identity, plus a **scope FSM** with explicit authorization commits. The two components are orthogonal but compose cleanly.

The model draws on well-trodden patterns: kernel-style commit trailers (multiple roles per commit), git's author/committer split, OAuth's principal/agent/scope/expiry tokens, Kubernetes RBAC (RoleBinding × Namespace × suspend), and Linux capabilities + cgroups. The shared shape across all of them: **principal/agent identity + scope/permission lifecycle + per-action gating that composes them.** This document instantiates that shape for aiwf's domain (entities, references, verbs).

---

## Identity: where it comes from

### `aiwf-actor:` is runtime-derived, not stored

Per [`design-decisions.md`](design-decisions.md) §"Layered location-of-truth," aiwf separates project policy (in `aiwf.yaml`, committed) from per-developer identity (per-checkout). The pre-I2.5 `aiwf.yaml.actor` field violated this separation: it was committed, so every developer cloning the repo got whoever-ran-`aiwf-init`'s identity unless they remembered `--actor`.

I2.5 removes `aiwf.yaml.actor` entirely. Actor identity is derived at runtime:

| Source | Precedence | Result |
|---|---|---|
| `--actor <role>/<id>` flag on the verb | highest | overrides everything; LLM harnesses use this to set `ai/claude` |
| `git config user.email` | default | `human/<localpart>` |
| (no source) | n/a | verb refuses with usage error; `aiwf doctor` flags this as misconfiguration |

`aiwf doctor` validates that `git config user.email` is set and well-formed (`<role>/<id>` derivable). The `<role>/<id>` regex is the same as today: `^[^\s/]+/[^\s/]+$` (exactly one `/`, no whitespace, neither side empty).

This decoupling fixes the multi-clone bug for free: each developer's checkout uses their own git config; worktrees inherit the parent repo's config; CI bots use whatever identity the runner is configured with.

### Roles by convention

Roles are freeform but conventional:

- `human/<id>` — a human individual.
- `ai/<id>` — an AI agent (e.g., `ai/claude`, `ai/cursor`).
- `bot/<id>` — an unattended automated process (e.g., `bot/ci`, `bot/github-actions`).

The kernel makes one structural distinction: **`human/...` vs. everything else.** Specific rules below depend on whether the role starts with `human/`.

### When the LLM is a tool vs. an agent

A practical clarification, codified in CLAUDE.md:

> When the human directs the LLM in conversation ("add a gap that says X"), the LLM is a *tool*, not a co-author. The principal is the human; the agent is the LLM. There is no separate "co-actor."

The principal/agent split below captures this without inflating tool use into authorship.

---

## Trailer set

Five new trailers, layered on the existing set:

| Trailer | Meaning | When written |
|---|---|---|
| `aiwf-actor:` | Operator (existing) — who ran the verb. | Always. |
| `aiwf-principal:` | Accountability-bearer. The person whose judgment authorizes this. | Required when `aiwf-actor:` starts with `ai/`. Forbidden when `aiwf-actor:` starts with `human/`. |
| `aiwf-on-behalf-of:` | The principal whose authorized scope this commit is acting under. Equal to or attributable-to a principal of an `aiwf-verb: authorize` commit. | Only inside an authorized scope. Required-with `aiwf-authorized-by:`. |
| `aiwf-authorized-by:` | Git SHA of the `aiwf-verb: authorize` commit that opened the scope. | Only inside an authorized scope. Required-with `aiwf-on-behalf-of:`. |
| `aiwf-scope:` | Scope state event marker on the authorize verb itself. Closed-set: `opened \| paused \| resumed`. (No `ended` — see "Scope termination.") | Only on `aiwf-verb: authorize` commits. |
| `aiwf-scope-ends:` | Lists the SHAs of authorize commits whose scope this commit is auto-ending. Repeatable (one trailer per ended scope). | On any commit that promotes the scope-entity of one or more active scopes to a terminal status. |
| `aiwf-reason:` | Free-text rationale for verbs that require one. Non-empty after trim. | Required on `aiwf authorize --pause` and `--resume`; optional on `aiwf authorize --to`. Distinct from `aiwf-force:` (sovereign override) and `aiwf-audit-only:` (G24 backfill rationale) — each reason-bearing trailer carries its own semantic. |

The pre-I2.5 trailers (`aiwf-verb`, `aiwf-entity`, `aiwf-actor`, `aiwf-to`, `aiwf-force`, `aiwf-prior-entity`, `aiwf-tests`) keep their existing semantics. `aiwf-actor:` specifically retains its meaning: **whoever ran the verb**, consistent with current PoC behavior.

### Required-together and mutually-exclusive rules

**Required-together** (both present or both absent — verb refuses partial; `aiwf check` flags partial as `provenance-trailer-incoherent`, error):

- `aiwf-on-behalf-of:` ↔ `aiwf-authorized-by:` — both signal scope membership.
- `aiwf-principal:` ↔ a non-`human/` `aiwf-actor:` — the principal slot is defined by the agent-acts-for-principal split. A principal without a non-human actor is incoherent; a non-human actor without a principal is unaccountable.

**Mutually exclusive** (verb refuses; `aiwf check` flags as `provenance-trailer-incoherent`, error):

- `aiwf-force:` + `aiwf-on-behalf-of:` — force is human-only (see below); on-behalf-of implies an agent operator. The two cannot coexist.
- `aiwf-principal:` + `aiwf-actor: human/...` — the human is acting directly; no second-actor split applies.
- `aiwf-on-behalf-of:` + `aiwf-actor: human/...` — same reason; a direct human act has no on-behalf-of.

**Deferred to G22** (sub-agent delegation): `aiwf-verb: authorize` + `aiwf-on-behalf-of:`. Whether an authorize commit may itself be inside a scope (i.e., an agent authorizing a sub-agent) is the policy question G22 reserves; the I2.5 kernel does not enforce a rule either way.

### Closed-set constraints on values

Validated at write time by the verb (refuse on shape mismatch):

| Trailer | Shape rule |
|---|---|
| `aiwf-actor:` | `<role>/<id>` regex (existing). |
| `aiwf-principal:` | Same regex. Role must start with `human/`. |
| `aiwf-on-behalf-of:` | Same regex. Role must start with `human/`. |
| `aiwf-authorized-by:` | 7–40 hex characters (matches `git rev-parse` output). |
| `aiwf-scope:` | Closed set: `opened`, `paused`, `resumed`. |
| `aiwf-scope-ends:` | Same shape as `aiwf-authorized-by:` — 7–40 hex. |

**SHA-points-to-real-authorize-commit is verified at read time, not write time.** Reasons:

- A SHA that exists at write time can become stale later (rebase, force-push, branch-prune). Write-time validation gives only weak guarantees and costs a per-verb log walk.
- Read-time validation, run on every `aiwf check`, catches both the typo case and the later-rewrite case with one mechanism.
- The standing rule already runs in the same pass as the rest of `aiwf check`'s read-time invariants.

The standing-check codes for stale references are below.

---

## The `--force` rule

**`--force` is a sovereign act. Only humans wield it. The kernel refuses `--force` from any `aiwf-actor:` whose role does not start with `human/`.**

This is a kernel-level rule. The trailer on a forced act is the existing `aiwf-force: <reason>` (reason required, non-empty after trim, per I2). The rule is enforced in two places: by the verb (refuses with usage error if `--force` is set and actor is not human) and by `aiwf check` (standing rule; finding code `provenance-force-non-human`, error).

If an LLM operating in a scope hits a kernel refusal that should genuinely be overridden, the path is: tell the human what would be forced and why; the human invokes `aiwf <verb> --force --reason "..."` directly. The trailer then says `aiwf-actor: human/peter`, no principal, no on-behalf-of — the human is operating without delegation.

A future delegated-force flag (`aiwf authorize --allow-force`) is filed as **G23** and intentionally deferred. When/if it ships, the rule extends to: `--force` requires either `aiwf-actor: human/...` *or* an active scope with `--allow-force` set, in which case the trailer must still carry `aiwf-principal: human/...` (the human who authorized force-permitted scope).

---

## Scope as a first-class FSM

A scope is the kernel's unit of "this work is authorized." It is *not* a structural property of the entity tree (despite drawing on the reference graph for reachability — see "Scope check"); it is an **intentional grant** with its own lifecycle.

### Scope states

Closed set: `active`, `paused`, `ended`. One Go function for legal transitions, mirroring the existing entity-FSM pattern.

```
authorize commit lands → state: active
   ↓
active ──pause──→ paused ──resume──→ active ──...
   ↓                              ↓
ended ←──── (auto: scope-entity reaches terminal)
                 (or: future revoke verb — G22)
```

Legal transitions:
- `active → paused` via `aiwf authorize <id> --pause "<reason>"`
- `paused → active` via `aiwf authorize <id> --resume "<reason>"`
- `active → ended` and `paused → ended`: auto, when the scope-entity reaches a terminal status (entity FSM says `done` or `cancelled`); recorded by an `aiwf-scope-ends:` trailer on the terminal-promote commit
- `ended` is terminal. Un-canceling a scope-entity does not resurrect a previously-ended scope; the human must issue a new authorization (Q3.5: strict end-on-terminal).

Scope state at any commit is computed by walking from the authorize commit forward through history, applying transitions in commit order. The scope's "frontmatter" is its trailer set on the original authorize commit; transitions are themselves commits with trailers. This means **`aiwf history <auth-sha>` works on scopes the same way it works on entities** — no new storage primitive.

Tombstone semantics for ended scopes: same as cancelled entities. The authorize commit stays in history forever; subsequent reads can always render the scope's full lifecycle. `aiwf history`, `aiwf check`, and the Provenance tab treat ended scopes as queryable historical objects.

### Scope check (the gating function)

A scope authorizes acts that are *intentional* about its scope-entity, not acts that are merely *under* its tree. The check answers: "given an active scope opened for entity S, is verb V on entity E permitted under this scope?"

```go
func scopeAllows(scope Scope, verb Verb, target Entity) bool {
    if scope.State != "active" {
        return false
    }
    // Creation acts: check the new entity's references reach the scope-entity.
    if verb.IsCreation() {
        return referenceGraph.Reaches(verb.NewEntity.References(), scope.Entity.ID)
    }
    // Move acts (strict): both endpoints must be in scope.
    if verb.IsMove() {
        return referenceGraph.Reaches(verb.From, scope.Entity.ID) &&
               referenceGraph.Reaches(verb.To, scope.Entity.ID)
    }
    // All other acts: target must reach scope-entity.
    return referenceGraph.Reaches(target.ID, scope.Entity.ID)
}
```

Reachability uses the **reference-graph index** built at tree-load time (the I2 step 11 deliverable; also used by `aiwf show`'s `referenced_by` field). An entity E reaches scope-entity S iff there exists a chain of frontmatter references (`parent`, `depends_on`, `addressed_by`, `relates_to`, `discovered_in`, etc., as defined in the schema) from E to S. The chain is bounded by the existing kind reference grammar.

### Composition with entity FSMs

Entity FSMs and scope FSMs are **orthogonal axes**, not nested:

```go
allow(verb v on entity e by actor a) =
    legalEntityTransition(e, v.target_state)         // existing entity FSM check
    AND scopeAllows(a, v, e)                          // new scope check (only for non-human actors)
```

For a `human/...` actor with no `--principal` flag, `scopeAllows` returns `true` unconditionally — the kernel does not require humans to be inside a scope to act. Scopes constrain *agents acting on a human's behalf*; humans need no such constraint.

For an `ai/...` actor, `scopeAllows` checks the union of all currently-active scopes attached to that actor. If at least one active scope grants the action, the verb proceeds and writes `aiwf-on-behalf-of:` + `aiwf-authorized-by:` for the matching scope. If none grants it, the verb refuses with finding `provenance-no-active-scope` (error).

### Multiple parallel scopes

A human can authorize the same agent for multiple scopes simultaneously. Example: `aiwf authorize E-03 --to ai/claude` and `aiwf authorize E-09 --to ai/claude`. The agent can be working on either at any moment. Each commit picks the matching scope at write time (the first active scope whose `scopeAllows` returns true) and writes its SHA to `aiwf-authorized-by:`.

If multiple active scopes match the same verb (rare but possible — overlapping reference paths), the kernel picks the *most-recently-opened* scope deterministically and records that one. The other(s) are not touched.

---

## The `aiwf authorize` verb

```
aiwf authorize <id> --to <agent> [--reason "<text>"]
aiwf authorize <id> --pause "<reason>"
aiwf authorize <id> --resume "<reason>"
```

Read-only on file content (no entity is modified); however, the verb writes a commit, so it goes through the existing `Apply` orchestrator and lock (G4) like other mutating verbs. Trailers on the authorize commit:

```
aiwf-verb: authorize
aiwf-entity: <id>                 # the scope-entity
aiwf-actor: <role>/<id>           # always human/...; verb refuses non-human
aiwf-to: <agent role>/<id>        # the agent being authorized; reuses aiwf-to:
aiwf-scope: opened                # or paused / resumed
aiwf-reason: <text>               # required on --pause / --resume; optional on --to
aiwf-force: <reason>              # only if the human used --force to override an end
```

### Why `aiwf-to:` for the agent

The kernel already uses `aiwf-to:` to record the *target state* of a `promote` event. For `authorize`, the target state is "scope opened with this agent." Reusing `aiwf-to:` for the agent identity is consistent with the existing trailer schema (the scope is the "entity" being acted on; its target state encodes who can act under it). No new trailer key for the agent.

### Verb behavior

- **`--to <agent>`**: opens a new scope. Verb refuses if `<id>` is in a terminal status (you cannot authorize work on a `done` epic). Verb refuses if `<actor>` is not `human/...` (only humans authorize).
- **`--pause "<reason>"`**: pauses the *most-recently-opened active scope* for `<id>`. If none active, refuses with `provenance-no-active-scope-to-pause`. Reason required, non-empty.
- **`--resume "<reason>"`**: resumes the *most-recently-paused scope* for `<id>`. If none paused, refuses with `provenance-no-paused-scope-to-resume`. Reason required, non-empty.

Each invocation produces exactly one commit, preserving the one-commit-per-verb rule.

### Scope id

A scope is addressed by the SHA of its `authorize` commit. There is no separate "scope id" namespace. SHAs are stable in the absence of force-pushes; force-push handling falls under the read-time stale-SHA detection (`provenance-authorization-missing`).

### `--force` on authorize

`aiwf authorize <id> --to <agent>` against a terminal scope-entity refuses by default. The human can override with `--force --reason "..."` per the existing kernel pattern. The `aiwf-force:` trailer lands on the authorize commit. The override is meaningful because it lets a human resurrect work on a cancelled entity by issuing a fresh authorization (the original ended scope stays ended; this is a new scope on the now-revived entity).

---

## `aiwf check` rules

New finding codes added in I2.5. All error severity unless noted; all run as standing rules on every push.

| Code | Trigger |
|---|---|
| `provenance-trailer-incoherent` | Required-together pair partial, or mutually-exclusive pair both present (per "Required-together and mutually-exclusive rules"). |
| `provenance-force-non-human` | `aiwf-force:` present and `aiwf-actor:` does not start with `human/`. |
| `provenance-actor-malformed` | `aiwf-actor:` does not match `<role>/<id>` regex. |
| `provenance-principal-non-human` | `aiwf-principal:` present and its role does not start with `human/`. |
| `provenance-on-behalf-of-non-human` | `aiwf-on-behalf-of:` present and its role does not start with `human/`. |
| `provenance-authorized-by-malformed` | `aiwf-authorized-by:` does not match the SHA shape (7–40 hex). |
| `provenance-authorization-missing` | `aiwf-authorized-by:` SHA does not resolve, or resolves to a commit whose `aiwf-verb:` is not `authorize`. |
| `provenance-authorization-out-of-scope` | `aiwf-authorized-by:` SHA resolves to a real authorize commit, but the referenced scope-entity has no reference path to the verb's target entity. |
| `provenance-authorization-ended` | `aiwf-authorized-by:` SHA resolves to a real authorize commit whose scope has reached `ended` state at the time of the referencing commit. |
| `provenance-no-active-scope` | An `ai/...` actor produced a commit with no `aiwf-on-behalf-of:` (caught at verb time; this code surfaces it on hand-edited or externally-authored commits). |

The pairing of write-time-refuse + standing-rule (Q3.6a/b) means each rule is enforced twice: once as the verb's pre-commit check (the chokepoint), once as the standing audit. The standing rule catches commits that bypassed the kernel verb (hand-edited, imported, pre-I2.5 history).

### Backwards compatibility

Pre-I2.5 commits do not carry the new trailers. The standing rules treat absence as benign for pre-I2.5 commits: a commit with `aiwf-actor: human/peter` and no `aiwf-principal:` is fine (Q3.6b: principal forbidden when actor is human). A commit with `aiwf-actor: ai/claude` and no principal *would* fire `provenance-trailer-incoherent` — but in practice no pre-I2.5 commits had `ai/...` as the actor, so no false positives are expected.

---

## Render-side (governance HTML)

The HTML render's Provenance tab consumes the model directly (per [`governance-html-plan.md`](../plans/governance-html-plan.md) §3.3, decision Q3.3):

- **Top section: scopes that touched this entity.** A table listing each authorization that ever applied to this entity: scope id (auth SHA short form), authorization actor (the human), opened date, current state (`active` / `paused` / `ended`), end date if ended, event count.
- **Below: chronological event timeline.** Each row carries a scope-id chip when authorized (`[E-03]`, `[E-09]`, etc.), no chip when direct. Pause / resume / end events appear on the timeline as scope-state changes with their own chips (`[E-03 paused]`, `[E-03 resumed]`, `[E-03 ended]`).

Reading "what happened to this milestone under what authorization" is one glance at the top section; reading the chronological detail is the timeline below.

### `aiwf history` text rendering

Default text output (per Q3.4):

```
2026-04-30  promote   M-007/AC-2 → met       human/peter via ai/claude  [E-03]
2026-04-30  authorize E-03                   human/peter             [E-03 opened]
2026-05-02  authorize E-03                   human/peter             [E-03 paused]
2026-05-04  authorize E-03                   human/peter             [E-03 resumed]
```

Actor column shows `principal via agent` syntax when they differ; just `actor` when they don't. Trailing `[scope-id]` chip when the row is scope-authorized. Authorization SHA hidden by default; `--show-authorization` flag adds a column. Full trailer set always available via `--format=json`.

---

## Worked examples

**Example 1: Solo human, direct verb.**
```
$ aiwf add gap --title "validators leak temp files"
```
Trailers: `aiwf-actor: human/peter`. No principal, no scope. The simplest case.

**Example 2: Human directs LLM in a Claude Code session.**
```
> "add a gap that says validators leak temp files"
[Claude runs:]
$ aiwf add gap --actor ai/claude --principal human/peter --title "..."
```
Trailers: `aiwf-actor: ai/claude`, `aiwf-principal: human/peter`. No scope (the conversation is turn-by-turn HITL; the LLM isn't operating autonomously).

**Example 3: Authorized autonomous work.**
```
[Human:]   $ aiwf authorize E-03 --to ai/claude --reason "implement the engine"
           # commits: aiwf-verb: authorize, aiwf-actor: human/peter,
           #          aiwf-to: ai/claude, aiwf-scope: opened
           # SHA: 4b13a0f

[Claude:]  $ aiwf promote M-007 --phase green --actor ai/claude --principal human/peter
           # commits: aiwf-actor: ai/claude, aiwf-principal: human/peter,
           #          aiwf-on-behalf-of: human/peter, aiwf-authorized-by: 4b13a0f
```

**Example 4: Pivot to another epic mid-flight.**
```
[Human:]   $ aiwf authorize E-03 --pause "blocked by E-09 fixture work"
[Human:]   $ aiwf authorize E-09 --to ai/claude --reason "unblock fixtures"

[Claude operates on E-09 verbs, all carry aiwf-authorized-by: <E-09's auth SHA>]

[Human:]   $ aiwf authorize E-09 --pause "fixture work landed"
[Human:]   $ aiwf authorize E-03 --resume "back to engine work"

[Claude resumes operating on E-03, all carry aiwf-authorized-by: <E-03's original SHA>]
```

The two scopes are independent. Each pause/resume is a distinct commit, recoverable from `aiwf history`.

**Example 5: Scope ends when epic completes.**
```
[Claude:]  $ aiwf promote E-03 done --actor ai/claude --principal human/peter
           # Allowed: E-03 reaches itself trivially via the reference graph,
           # so scopeAllows returns true. Because the target state (`done`) is
           # terminal for the entity kind, the trailer-writer additionally
           # writes aiwf-scope-ends: 4b13a0f into this same commit.
           # Trailers: aiwf-actor: ai/claude, aiwf-principal: human/peter,
           #          aiwf-on-behalf-of: human/peter, aiwf-authorized-by: 4b13a0f,
           #          aiwf-scope-ends: 4b13a0f.
```
After this commit, scope `4b13a0f` is `ended`. Subsequent agent verbs referencing it as `aiwf-authorized-by: 4b13a0f` produce `provenance-authorization-ended` findings.

The agent can close its own scope by design: scopes constrain *what* an agent can do, not *which step within scope* requires synchronous human ratification. The auto-end is mechanical and recoverable from `aiwf history`; the human's ratification opportunity moves from synchronous-at-close to post-hoc via the audit trail. If the human wants a synchronous handoff at the close, the path is to authorize a narrower scope (e.g., the milestones inside E-0003 individually) and invoke the epic-level `done` directly.

**Example 6: Forced override by the human.**
```
[Human directly:]
$ aiwf cancel M-007 --force --reason "scope was wrong from the start"
# Trailers: aiwf-actor: human/peter, aiwf-force: scope was wrong from the start
# No principal, no on-behalf-of. The human is acting directly.
# --force is permitted (actor is human/...).
```

If an LLM tried the same with `--force`, the verb refuses with `provenance-force-non-human`. The LLM would need to prompt the human to invoke directly.

---

## Open extensions

Documented as known-incomplete; deferred from I2.5 by design:

- **G22 — provenance model extension surface.** Future verbs and flags: `aiwf revoke <auth-sha> --reason "..."` (explicit revocation); time-bound scopes (`--until <date>`); verb-set restrictions (`--verbs add,promote`); pattern scopes (`--pattern "M-007/*"`); sub-agent delegation (whether an agent can authorize a sub-agent — the policy question reserved by Q3.6b's deferred mutually-exclusive pair).
- **G23 — delegated `--force` via `aiwf authorize --allow-force`.** A future flag on `aiwf authorize` letting the agent invoke `--force` within scope, while still writing the human as `aiwf-principal:`. YAGNI for the PoC; revisit if real friction shows up.
- **Bulk-import per-entity attribution.** When `aiwf import` ingests data with per-row author info, the importer should write per-entity `aiwf-actor:` pairs instead of one collapsed trailer. Bundled into G22.

---

## Summary

| Axis | Pre-I2.5 | I2.5 |
|---|---|---|
| Identity source | `aiwf.yaml.actor` (committed) | `git config user.email` (per-checkout, runtime) |
| Operator | `aiwf-actor:` | `aiwf-actor:` (unchanged semantics) |
| Accountability | conflated with operator | `aiwf-principal:` (separate; human-only) |
| Authorized scope | implicit / undefined | `aiwf-on-behalf-of:` + `aiwf-authorized-by:` referencing an authorize commit |
| Authorization verb | n/a | `aiwf authorize <id> --to <agent>` (with `--pause` / `--resume`) |
| Scope lifecycle | n/a | First-class FSM: `active \| paused \| ended` |
| Composition | entity FSMs only | entity FSM × scope FSM (gating, not containment) |
| `--force` | any actor | `human/...` only |
| Standing audit | per-entity findings | + `provenance-*` finding family |
| Render | single-actor timeline | scope-as-section + chronological timeline with chips |

The model is a kernel-level commitment: every property above is enforceable mechanically, and the LLM's correctness is gated by the same checks that gate any other actor.
