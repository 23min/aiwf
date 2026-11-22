# Scope lifecycle: what the docs say

Machine `scope`: one agent's delegation on one milestone. States `none`, `active`, `paused`, `ended`, `closed`. Events `authorize`, `act`, `pause`, `resume`, `end`, `close`, `rebase`, `copy`, `check`. No variables, no outputs.

Sources read: `docs/` (design, ADRs, archive, audits, research, reference, initiatives, explorations), `work/decisions/`, `work/epics/` (active and archived), `work/gaps/` (active and archived), `internal/skills/embedded/**/SKILL.md`, `internal/skills/embedded-rituals/**/SKILL.md`, `README.md`, `CHANGELOG.md`, `CLAUDE.md`, `AGENTS.md`. Excluded by instruction: `spec/`, `docs/research/fewer-planning-commits/verifier-kit-pilot/`, and the specs of E-0100, M-0364, M-0365 and M-0366.

Notation for the case column: `state, event`, then conditions in brackets. A bracketed condition names something the machine does not have (actor role, `--force`, `--branch`, number of scopes, which branch was rebased, the milestone's status); section 4 lists them. "No active scope" means any of `none`, `paused`, `ended`, `closed` where the source does not distinguish them. Sources mostly say "scope-entity" or "entity"; a claim is listed here when it is stated for scopes in general or for a milestone. Claims stated only for an epic's scope are marked `[epic]`.

Dates are the date the source states. A CHANGELOG entry carries its release's date. "undated" means the source states none.

## 1. Claims

### States and terminality

| # | Case | What the source says happens | Source and where | Date | Quote |
|---|---|---|---|---|---|
| C1 | any | Three scope states: active, paused, ended. | `docs/design/provenance-model.md` §"Scope states" | undated | "Closed set: `active`, `paused`, `ended`. One Go function for legal transitions, mirroring the existing entity-FSM pattern." |
| C2 | any | Three states; several scopes may run in parallel; two routes to the end. | `docs/design/design-decisions.md` §I2.5 provenance bullets | undated | "**Scope is a first-class FSM.** Closed states `active \| paused \| ended`. Authorization is a typed grant created by `aiwf authorize <id> --to <agent>`; pause/resume are first-class transitions. Multiple parallel scopes are supported." |
| C3 | any | Names the scope's states as none, open, paused, ended. | `docs/research/collaborative-state-and-commit-granularity.md` §9 state table | 2026-10-28 | "\| Authorization scope \| none; open; paused; ended \|" |
| C4 | ended or closed, any | Stays ended; an un-cancel of the entity does not bring the scope back; a new authorization is needed. | `docs/design/provenance-model.md` §"Scope states" | undated | "`ended` is terminal. Un-canceling a scope-entity does not resurrect a previously-ended scope; the human must issue a new authorization (Q3.5: strict end-on-terminal)." |
| C5 | closed, [un-cancel of the milestone] | Stays ended; issue a fresh `authorize`. | `internal/skills/embedded/aiwf-authorize/SKILL.md` §Lifecycle | undated | "Ended is terminal — un-cancelling a scope-entity does not resurrect a previously-ended scope. Issue a fresh `aiwf authorize` instead." |
| C6 | ended, authorize | Opens a new scope with a new opening commit; the ended scope stays ended and its SHA stays valid on commits made under it. | `docs/adr/ADR-0047-…` §"Nothing undoes an end" | undated | "The inverse of an end is a fresh grant: `aiwf authorize <id> --to <agent>` opens a new scope with a new authorize-commit SHA, and the ended scope's SHA stays valid forever as the `aiwf-authorized-by:` reference on commits made under it." |

### authorize

| # | Case | What the source says happens | Source and where | Date | Quote |
|---|---|---|---|---|---|
| C7 | none, authorize | Moves to active. Commit carries `aiwf-verb: authorize`, `aiwf-to: <agent>`, `aiwf-scope: opened`, optional `aiwf-reason`. | `docs/design/provenance-model.md` §"Scope states" diagram and §"The `aiwf authorize` verb" trailer block | undated | "authorize commit lands → state: active" |
| C8 | any with the milestone not terminal (none, active, paused, ended), authorize; closed, authorize; any, authorize [non-human actor] | Opens a new scope; refused when the entity is terminal; refused for a non-human actor. | `docs/design/provenance-model.md` §"Verb behavior" | undated | "**`--to <agent>`**: opens a new scope. Verb refuses if `<id>` is in a terminal status (you cannot authorize work on a `done` epic). Verb refuses if `<actor>` is not `human/...` (only humans authorize)." |
| C9 | none, authorize | Moves to active; the opening commit's SHA is the scope's id. | `internal/skills/embedded/aiwf-authorize/SKILL.md` §Lifecycle | undated | "**`--to`** opens a new scope in state `active`. The commit's SHA is the scope id; subsequent verbs inside the scope reference it via `aiwf-authorized-by:`." |
| C10 | none, authorize [milestone, delegated, run on the parent epic branch with `--branch milestone/…`] | Moves to active; stays active until the milestone is terminal or the operator pauses it. | `internal/skills/embedded-rituals/…/aiwfx-start-milestone/SKILL.md` §4 | undated | "This is a *separate* commit from step 3, landed on the same parent epic branch. The scope is `active` from this commit forward; the agent operates within it until the milestone reaches a terminal status or the operator pauses the scope." |
| C11 | none, authorize | The scope is named by the opening commit's SHA; no other id. | `docs/design/provenance-model.md` §"Scope id" | undated | "A scope is addressed by the SHA of its `authorize` commit. There is no separate "scope id" namespace." |
| C12 | none, authorize | Same as C11. | `docs/design/legal-workflows-first-principles.md` §6d, R-FP-0137 | undated | "A scope is addressed by the SHA of its `authorize` commit. No separate scope id namespace." |
| C13 | none, authorize | The opening commit sends `aiwf-scope-id: <scope-id>`, a random id; later pointers name that id. `docs/research/fewer-planning-commits/reports/00-options-brief.md` §"Code facts" calls this ADR "accepted, not built". | `docs/adr/ADR-0056-…` §Decision | 2026-11-01 | "**A scope has a random id.** `aiwf authorize <id> --to <agent>` generates it and writes it on the opening commit as `aiwf-scope-id: <scope-id>`." |
| C14 | closed, authorize [`--force --reason`] | Opens a new scope; the old scope stays ended. The source also says the entity is "revived"; it does not say how. | `docs/design/provenance-model.md` §"`--force` on authorize" | undated | "The override is meaningful because it lets a human resurrect work on a cancelled entity by issuing a fresh authorization (the original ended scope stays ended; this is a new scope on the now-revived entity)." |
| C15 | closed, authorize | Refused unless `--force --reason`. | `internal/skills/embedded/aiwf-authorize/SKILL.md` §"Common refusals" | undated | "**`--to` against a terminal scope-entity**: refuses unless `--force --reason "..."` is set." |
| C16 | closed, authorize | Unclear: says the verb has no from-state meaning and that authorizing a cancelled milestone is a question the verb does not raise. | `work/decisions/D-0077-…` §Decision and §Reasoning | 2026-08-24 | "It binds a scope to a branch and has no `FromState` semantics" / "eight of them would require ruling whether authorizing a `done` epic or a `cancelled` milestone is legal, which are questions the verb's own semantics do not raise." |
| C17 | active, authorize [same agent, same branch] | Exit 0, a second commit, two active scopes, no finding. | `work/gaps/G-0460-…` §"What's missing" | undated | "two `authorize` invocations with identical arguments both exit 0, each lands its own commit, and `aiwf show <id> --format=json` then reports two scope records, both `"state": "active"`, differing only by `auth_sha`." |
| C18 | active, authorize | Same as C17. | `work/epics/E-0074-…/epic.md` §"G-0460 gates the `Authorize` entry only" | undated | "A repeat `aiwf authorize <id> --to <agent>` exits 0, appends a second commit, and leaves two simultaneously-active scopes with no check finding." |
| C19 | active, authorize | Unclear: "one scope per entity" may mean each entity holds its own scope apart from its epic's, or at most one scope per entity. | `internal/skills/embedded-rituals/…/aiwfx-start-milestone/SKILL.md` §2 "Delegation prompt (Q&A)" | undated | "Kernel semantics: one scope per entity; the milestone's scope is opened, paused, and resumed on its own entity, with its own `aiwf-branch:` (the milestone branch)." |
| C20 | ended, authorize | A re-grant after an end "may be" a new event; unsettled. | `work/gaps/G-0459-…` | undated | "A second grant may be a legitimate new event (a different agent, a different branch, a re-grant after a scope ended)." |
| C21 | none, authorize [entity is a milestone] | Allowed: a milestone is a scope-entity; other kinds are refused with `authorize-kind-not-allowed`. | `work/decisions/D-0007-…` §Resolution | undated | "`aiwf authorize <id> --to <agent>` refuses if `<id>` is not an epic or milestone." / "**Milestone**: subtree is ACs (composite) + gaps via `discovered_in`; sensible scope-entity." |
| C22 | none, authorize [entity is a milestone] | Unclear: describes the epic as the entity a scope is opened on and a milestone as work inside someone's scope. | `docs/adr/ADR-0047-…` §"Every edge into a terminal epic status is a sovereign act" | undated | "An epic is the unit a scope is opened on and whose closure ends delegation; a milestone is work inside a scope someone already holds." |
| C23 | none, authorize [AI target, no ritual branch context, no `--force`] | Refused at the verb, `branch-context-required`. | `docs/reference/workflow-legality.md` §"Global restrictions" | undated | "\| authorize \| target-agent-role == ai AND ritual-branch-context-present == false AND force == false \| Illegal \| verb-time \| true \| branch-context-required \|" |
| C24 | none, authorize [AI target, `--branch` names a branch that does not exist] | Refused, `branch-not-found`. | `work/epics/archive/E-0030-…/M-0103-…` frontmatter AC-2 title | undated | "AI-actor authorize with --branch <missing> refuses (branch-not-found)" |
| C25 | none, authorize [`--branch` names a branch that does not exist yet, run from trunk or a ritual-shape checkout] | Accepted. | `internal/skills/embedded/aiwf-authorize/SKILL.md` §"What to run" | undated | "Naming a branch that does not exist yet is accepted from the trunk or from a ritual-shape checkout, which is what lets a ritual open the scope before cutting the branch." |
| C26 | none, authorize [AI target, current branch and `--branch` not a legal rung pair] | Refused, `rung-pair-illegal`; `branch-not-found` is no longer raised. | `work/decisions/D-0018-…` | undated | "The pre-AC-2 `branch-not-found` code is now dead at its emission site." |
| C27 | none, authorize [checkout is already on the milestone branch] | Refused at the verb, `branch-context-required`. | `internal/skills/embedded-rituals/…/aiwfx-start-milestone/SKILL.md` §"Anti-patterns" | undated | "The kernel's preflight refuses authorize-on-milestone-branch with `branch-context-required` at the verb layer" |
| C28 | none, authorize [AI target, checkout on a ritual-shape branch, no `--branch`] | Accepted. | `work/epics/archive/E-0030-…/M-0103-…` frontmatter AC-3 title | undated | "AI-actor authorize from ritual-shape checkout (no --branch) accepts" |

### act

| # | Case | What the source says happens | Source and where | Date | Quote |
|---|---|---|---|---|---|
| C29 | active, act; none, act | Active: the verb proceeds and sends `aiwf-on-behalf-of` and `aiwf-authorized-by`. No active scope: refused, `provenance-no-active-scope`. | `docs/design/provenance-model.md` §"Composition with entity FSMs" | undated | "If at least one active scope grants the action, the verb proceeds and writes `aiwf-on-behalf-of:` + `aiwf-authorized-by:` for the matching scope. If none grants it, the verb refuses with finding `provenance-no-active-scope` (error)." |
| C30 | active, act | Proceeds; the kernel matches the most recently opened active scope and sends the two trailers. | `internal/skills/embedded/aiwf-authorize/SKILL.md` §"Tool vs. agent" | undated | "Subsequent verbs run with `--actor ai/<id> --principal human/<id>` and the kernel matches the most-recently-opened active scope, decorating each commit with `aiwf-on-behalf-of: human/<id>` and `aiwf-authorized-by: <auth-sha>`." |
| C31 | active, act [several active scopes match] | Records the most recently opened. | `docs/design/provenance-model.md` §"Multiple parallel scopes" | undated | "If multiple active scopes match the same verb (rare but possible — overlapping reference paths), the kernel picks the *most-recently-opened* scope deterministically and records that one." |
| C32 | none, act | Refused at the verb; `check` reports any such commit. | `internal/skills/embedded/aiwf-authorize/SKILL.md` opening paragraph | undated | "Without an active scope, the kernel refuses non-human actors at verb time and `aiwf check` flags any commit they produced as `provenance-no-active-scope`." |
| C33 | none, act [promote] | Refused, `provenance-no-active-scope`. | `internal/skills/embedded/aiwf-promote/SKILL.md` §Provenance | undated | "Without an active scope, agent promotions refuse with `provenance-no-active-scope`." |
| C34 | none, act [`--actor ai/claude --principal human/peter`] | Refused, exit 1, no commit (measured). | `work/gaps/G-0706-…` §"What's missing" | undated | "aiwf add: provenance refused: actor "ai/claude" has no active scope authorizing this act (provenance-no-active-scope)" / "Observed a refusal and no commit." |
| C35 | none, act | A non-human actor needs `--principal` and an active scope that reaches the entity. | `CHANGELOG.md` [0.39.0] "Fixed — G-0674" | 2026-09-27 | "a non-human `--actor` needs it, and also needs an active `aiwf authorize` scope that reaches the entity." |
| C36 | none, act | A non-human actor reaches the apply seam only through an active scope. | `docs/adr/ADR-0040-…` §Context | undated | "A non-human actor reaches this seam only through an active scope, and a scope always carries `aiwf-on-behalf-of`" |
| C37 | none, act [`--actor ai/claude --principal human/peter`] | Succeeds; commit carries actor and principal, no scope trailers. | `docs/design/provenance-model.md` §"Worked examples", Example 2 | undated | "Trailers: `aiwf-actor: ai/claude`, `aiwf-principal: human/peter`. No scope (the conversation is turn-by-turn HITL; the LLM isn't operating autonomously)." |
| C38 | none, act [tool mode] | Succeeds with no scope. | `internal/skills/embedded/aiwf-authorize/SKILL.md` §"Tool vs. agent" | undated | "verbs run with `--actor ai/<id> --principal human/<id>` (or just `--actor human/<id>` if the assistant is invoking on the human's behalf). No scope, no `aiwf-authorized-by:` trailer." |
| C39 | none, act [non-human actor with `--principal`] | Succeeds: a principal or an active scope is enough. | `docs/design/legal-workflows-audit.md` R-AUDIT-0104 (also R-RULE-066); doc states revision 2 dated 2026-05-18 | 2026-05-18 | "When `--actor` is non-human (`ai/...`, `bot/...`), `--principal human/<id>` must be supplied OR an active authorization scope must cover the entity; otherwise the verb errors before any write" |
| C40 | none, act [add] | Requires `--principal`; scope trailers are added only if a scope covers the new entity. | `internal/skills/embedded/aiwf-add/SKILL.md` §"What aiwf does" step 5 | undated | "When the operator is non-human (`ai/<id>`, `bot/<id>`), the kernel additionally requires a `--principal human/<id>` flag and stamps `aiwf-principal:` on the commit. If an active authorization scope (see `aiwf-authorize`) covers the new entity's parent / references, `aiwf-on-behalf-of:` and `aiwf-authorized-by:` are added too." |
| C41 | paused, act | Refused, `provenance-no-active-scope`, until resumed. | `internal/skills/embedded/aiwf-authorize/SKILL.md` §Lifecycle | undated | "While paused, agent verbs targeting the scope refuse with `provenance-no-active-scope` until the scope is resumed." |
| C42 | paused, act [AI-actor commit on the bound branch], then check | `isolation-escape` stays silent; the source calls this a "legal cell". Unclear whether it says anything about a verb's refusal; it is about the branch rule only. | `work/epics/archive/E-0030-…/epic.md` corner case 6 | undated | "**AI-actor commit on `epic/E-NN-X` while scope is paused** → finding silent (legal cell — commits made under a paused scope ride the same branch they were opened against; paused does not change the binding)." |
| C43 | ended, act [after `--end`] | Refused at the verb, exit 1 (measured). | `work/gaps/G-0752-…` reproduction | 2026-10-28 | "aiwf retitle: provenance refused: actor "ai/claude" has no active scope authorizing this act (provenance-no-active-scope)" |
| C44 | ended, act | Refused (measured). | `docs/research/collaborative-state-and-commit-granularity.md` §8 E8 | 2026-10-28 | "The agent's `aiwf retitle M-0001 … --actor ai/claude --principal human/peter` was then refused with `provenance-no-active-scope`." |
| C45 | ended or closed, act | Commits that name the ended scope draw `provenance-authorization-ended`. Unclear whether "agent verbs" means the verb writes the commit or the commit is made by other means. | `docs/design/provenance-model.md` §"Worked examples", Example 5 | undated | "After this commit, scope `4b13a0f` is `ended`. Subsequent agent verbs referencing it as `aiwf-authorized-by: 4b13a0f` produce `provenance-authorization-ended` findings." |
| C46 | closed, act | A verb run after the milestone's promote to done writes a commit carrying the ended scope's `aiwf-authorized-by`, which fails on push. | `internal/skills/embedded-rituals/…/aiwfx-wrap-milestone/SKILL.md` §10 and anti-patterns | undated | "A verb-driven commit produced after that point risks an ended-scope `aiwf-authorized-by:` trailer on push" / "closing gaps after `aiwf promote M-NNNN done` risks an ended-scope `aiwf-authorized-by:` trailer on the gap-promote commit." |
| C47 | closed, act `[epic]` | Same as C46 for an epic's scope: the commit carries the ended scope's pointer and `provenance-authorization-ended` fires on push. | `internal/skills/embedded-rituals/…/aiwfx-wrap-epic/SKILL.md` §7, "Why promote is last among verb-driven commits" | undated | "Any commit produced *after* this that goes through a kernel verb — wrap artefact, reallocates, or other verb-driven wrap-bundle commits — would carry `aiwf-authorized-by:` referencing the just-ended scope and trigger the kernel's `provenance-authorization-ended` finding on push" |
| C48 | closed, act `[epic]` | Commits after the closing promote carried the ended scope's pointer; `check` errors on push (observed). Unclear whether those commits were verb-made. | `work/gaps/archive/G-0119-…` §"What's missing" | 2026-05-13 (discovery date) | "The promote terminates the active authorize scope. Subsequent commits then carry `aiwf-authorized-by:` referencing the just-ended scope plus `aiwf-actor: ai/claude` with no live authorization — both surface as `aiwf check` errors on push" |
| C49 | closed, act | Refused at the verb, exit 1 (measured, before any rebase). | `work/gaps/G-0752-…` §"What's missing" | 2026-10-28 | "With `aiwf promote M-0001 done` in place of `--end`, `aiwf show M-0001` reports the scope `ended` before the rebase and `active` after it, and the agent's retitle is refused before (exit 1) and succeeds after (exit 0) with `aiwf check` at no error." |
| C50 | active, act [commit on a branch other than the scope's `aiwf-branch`], then check | `isolation-escape`. | `internal/skills/embedded/aiwf-check/SKILL.md` findings table | undated | "An AI-actor commit landed on a branch that does not match its active scope's recorded `aiwf-branch:` (the branch-choreography model — an AI-actor commit must ride the branch its active scope names)." |

### pause

| # | Case | What the source says happens | Source and where | Date | Quote |
|---|---|---|---|---|---|
| C51 | active, pause | Moves to paused; targets the most recently opened active scope; reason required. Commit sends `aiwf-scope: paused`, `aiwf-reason`. | `docs/design/provenance-model.md` §"Scope states" and §"Verb behavior" | undated | "`active → paused` via `aiwf authorize <id> --pause "<reason>"`" / "**`--pause "<reason>"`**: pauses the *most-recently-opened active scope* for `<id>`." |
| C52 | active, pause | Moves to paused. | `internal/skills/embedded/aiwf-authorize/SKILL.md` §Lifecycle | undated | "**`--pause "<reason>"`** flips the most-recently-opened active scope on `<id>` to `paused`." |
| C53 | no active scope, pause | Refused with code `provenance-no-active-scope-to-pause`. | `docs/design/provenance-model.md` §"Verb behavior" | undated | "If none active, refuses with `provenance-no-active-scope-to-pause`. Reason required, non-empty." |
| C54 | no active scope, pause | Same as C53. | `docs/design/legal-workflows-first-principles.md` R-FP-0135 | undated | "`--pause "<reason>"` requires a most-recently-opened active scope for `<id>`. If none, the verb refuses with `provenance-no-active-scope-to-pause`." |
| C55 | no active scope, pause | Refused. | `internal/skills/embedded/aiwf-authorize/SKILL.md` §"Common refusals" | undated | "**`--pause` with no active scope on the entity**: nothing to pause; the verb refuses." |
| C56 | no active scope, pause; no paused scope, resume | Refused with no code; the two named codes do not exist. | `docs/audits/normative-docs-drift-audit.md` | 2026-08-06 | "Neither appears anywhere in source; the real refusal is uncoded prose with no `code` field in the JSON envelope." |
| C57 | same as C56 | Same as C56. | `docs/audits/entity-truth-audit.md` §"docs/design/provenance-model.md · phantom-finding-code" | 2026-08-19 | "Both refusals are bare fmt.Errorf format strings carrying no code at all." |
| C58 | paused, pause; active, resume | Refused (exit 2), not a NoOp; no commit. | `docs/archive/pocv3/health-scorecard-2026-06-16.md` | 2026-06-16 | "`internal/verb/authorize.go:505` authorize --pause/--resume on a scope not in the source state errors via missingFmt (no matching scope) rather than NoOp" |
| C59 | paused, pause; active, resume | Unclear: a general rule that a satisfied request is a NoOp; the ADR it cites limits its own scope to `promote` and `cancel`. | `CLAUDE.md` "What aiwf commits to" item 7 | undated | "a request already satisfied is a NoOp at exit 0 with no commit" |
| C60 | active [two active scopes on the milestone], pause | One scope moves to paused; the other stays active; no finding. | `work/gaps/G-0460-…` §"Why it matters" | undated | "a pause acts on one scope, so an entity can be left carrying a paused scope and a live one simultaneously, an authorization state no operator asked for and no finding reports." |

### resume

| # | Case | What the source says happens | Source and where | Date | Quote |
|---|---|---|---|---|---|
| C61 | paused, resume | Moves to active; targets the most recently paused scope; reason required. Commit sends `aiwf-scope: resumed`. | `docs/design/provenance-model.md` §"Scope states" and §"Verb behavior" | undated | "`paused → active` via `aiwf authorize <id> --resume "<reason>"`" / "**`--resume "<reason>"`**: resumes the *most-recently-paused scope* for `<id>`." |
| C62 | no paused scope, resume | Refused with code `provenance-no-paused-scope-to-resume`. Also R-FP-0136. | `docs/design/provenance-model.md` §"Verb behavior" | undated | "If none paused, refuses with `provenance-no-paused-scope-to-resume`. Reason required, non-empty." |
| C63 | no paused scope, resume | Refused. | `internal/skills/embedded/aiwf-authorize/SKILL.md` §"Common refusals" | undated | "**`--resume` with no paused scope on the entity**: nothing to resume." |
| C64 | closed [scope paused when the milestone closed], resume | Can never be resumed: no verb acts on a terminal entity. | `docs/design/provenance-model.md` §"Ending covers paused scopes" | undated | "once the entity is terminal no verb will act on it again, so a paused scope there can never be resumed" |
| C65 | paused, resume | Unclear: says pause and resume both act on the most recently opened active scope. | `docs/design/legal-workflows-audit.md` R-AUDIT-0123 | 2026-05-18 (revision 2) | "Pause/resume the most-recently-opened active scope on `<id>`; `--end` retires one named non-ended scope without touching the entity's status. Together these cycle the scope FSM (active → paused → active → ended)" |
| C66 | no active scope [a pause commit already in history]; no paused scope [a resume commit already in history] | Replay drops the commit; the scope's state does not change. | `docs/audits/structural-sweep-audit.md` §B1 | 2026-09-17 | "`cliutil.ReplayScopes` (`internal/cli/cliutil/scopes.go:62-107`) does the real work — multi-scope, and it silently drops a pause with no active scope or a resume with no paused one (`:82,87`) where `LoadScope` errors" |

### end

| # | Case | What the source says happens | Source and where | Date | Quote |
|---|---|---|---|---|---|
| C67 | active, end; paused, end | Moves to ended; the milestone's status is unchanged; sends `aiwf-scope-ends`. | `docs/design/provenance-model.md` §"Scope states" | undated | "`active → ended` and `paused → ended`, by either of two routes, both recorded by an `aiwf-scope-ends: <auth-sha>` trailer: automatically, when the scope-entity reaches a terminal status (entity FSM says `done` or `cancelled`), one trailer per non-ended scope on the closing commit; or deliberately, via `aiwf authorize <id> --end`, which ends one named scope and leaves the entity's status untouched" |
| C68 | active, end; paused, end | The commit sends `aiwf-scope-ends` and no `aiwf-scope` or `aiwf-to`; `aiwf-reason` is required. | `docs/design/provenance-model.md` §"The `aiwf authorize` verb" | undated | "`--end` writes `aiwf-scope-ends:` rather than a fourth `aiwf-scope:` value, which is the same trailer the automatic end at terminal-promote emits" |
| C69 | active or paused, end [with `--scope`; without `--scope` and one candidate; without `--scope` and several] | Ends the named scope; ends the sole candidate; refused, listing the candidates. | `docs/adr/ADR-0047-…` §"An operator ends a scope by naming it" | undated | "`--end` ends the scope named by `--scope`; with no `--scope` it ends the entity's sole non-ended scope; with more than one candidate and no `--scope` it refuses, listing them." |
| C70 | none, end [no `--scope`]; ended, end [`--scope` names it] | Refused; NoOp. | `docs/adr/ADR-0047-…` §"An operator ends a scope by naming it" | undated | "With no `--scope` and no candidate, the verb refuses rather than converging" / "Naming an already-ended scope with `--scope` does converge, because there the target resolves and its ended state is exactly the effect asked for." |
| C71 | paused, end; paused, close | Moves to ended. | `docs/adr/ADR-0047-…` §"Ending covers every non-ended scope, paused ones included" | undated | "Both the operator end and the automatic end target scopes in `active` *or* `paused` state." |
| C72 | ended or closed, end [no `--scope`]; ended or closed, end [`--scope` names the ended scope] | Refused; NoOp at exit 0, no commit. The source says "already-ended scope" without naming which route ended it. | `internal/skills/embedded/aiwf-authorize/SKILL.md` §"Common refusals" | undated | "**`--end` with no non-ended scope on the entity**: nothing to end; the verb refuses. Naming an already-ended scope with `--scope` instead converges — the target resolves and its ended state is the effect asked for — so it exits 0 and writes no commit." |
| C73 | ended, end; none, end [`--scope` that resolves to nothing] | NoOp, exit 0, no commit; refused. | `work/epics/archive/E-0090-…/M-0325-…` AC-2 | undated | "Re-running the verb against a scope already `ended` returns exit 0, writes no commit, and reports the state as already holding. Naming a scope that does not resolve — a bad sha, an entity with no scope ever opened — is refused, not converged." |
| C74 | active, end | Moves to ended; entity status unchanged; reason required; already ended converges; unknown `--scope` refused. | `CHANGELOG.md` [0.34.0] "Added — `aiwf authorize <id> --end`" | 2026-08-30 | "`--end` ends one scope and leaves the entity's status alone." / "Re-ending an already-ended scope converges at exit 0 without a commit; a `--scope` that matches nothing is refused." |
| C75 | active, end; active, close | Moves to ended by either route. | `docs/design/design-decisions.md` §I2.5 | undated | "A scope ends by either of two routes, both recorded by an `aiwf-scope-ends:` trailer: automatically, when a promote or cancel takes the scope-entity to a terminal status, covering every non-ended scope on it; or deliberately, via `aiwf authorize <id> --end`, which retires one named scope and leaves the entity's status alone (ADR-0047)." |
| C76 | active, end | No such mode exists; the scope ends only at a terminal promote. | `docs/audits/entity-truth-audit.md` §"G-0225 · stale-cli-invocation" | 2026-08-19 | "`aiwf authorize` accepts exactly one positional arg; there is no `end` subverb and no `--end` flag (the mode flags are `--to`, `--pause`, `--resume`)." / "there is no operator-driven end at all" |
| C77 | active, end | No such flag. | `docs/audits/gap-truth-audit.md` §G-0111 | 2026-08-23 | "Its concern 3 prescribes `aiwf authorize --end`, a flag that has never existed." |
| C78 | active, end | No revoke; reaching ended needs the entity's terminal promote or `--force`. | `work/gaps/G-0282-…` (open), body list | undated (later sections dated 2026-06-26 and 2026-07-21) | "`--pause` only parks a scope; reaching `ended` needs a terminal-promote of the scope-entity or `--force`." |
| C79 | active, end | No operator end; scopes end only when the entity goes terminal. | `docs/archive/pocv3/gaps-pre-migration.md` G22 item 1 | 2026-05-05 (frozen) | "Scopes today auto-end only on terminal scope-entity status; a human cannot un-authorize an in-flight scope without forcing the entity to a terminal status." |
| C80 | active, end `[epic]` | Unclear: "ends any opened scope" may mean each scope or all of them. | `work/epics/archive/E-0028-…/epic.md` §Constraints, "What undoes this?" | undated | "`aiwf authorize E-NN --end` ends any opened scope." |

### close

| # | Case | What the source says happens | Source and where | Date | Quote |
|---|---|---|---|---|---|
| C81 | active, close; paused, close; ended, close | Active and paused move to ended; the closing commit sends one `aiwf-scope-ends` per non-ended scope; an already-ended scope gets none. | `docs/design/provenance-model.md` §"Scope states" (quoted at C67) and §"Ending covers paused scopes" | undated | "Both ends — the operator's and the automatic one at terminal promote or cancel — target every scope in `active` *or* `paused` state." |
| C82 | active, close; paused, close | Moves to ended, paused included; one `aiwf-scope-ends` per ended scope. | `internal/skills/embedded/aiwf-promote/SKILL.md` §Provenance | undated | "every non-ended scope on that entity auto-ends — the commit carries one `aiwf-scope-ends: <auth-sha>` per ended scope. A paused scope is included: nothing will act on a terminal entity, so nothing could resume it." |
| C83 | active, close; paused, close | Moves to ended. | `internal/skills/embedded/aiwf-authorize/SKILL.md` §Lifecycle | undated | "A `promote <id> <terminal-state>` or `cancel <id>` on the scope-entity (e.g., `aiwf promote E-NNNN done`) **auto-ends** every non-ended scope on that entity by writing one `aiwf-scope-ends: <auth-sha>` trailer per scope." |
| C84 | paused, close | Moves to ended. The entry also says that before this release a paused scope stayed paused. | `CHANGELOG.md` [0.34.0] "Fixed — a paused scope is now ended when its entity closes" | 2026-08-30 | "Both ends now cover every non-ended scope." / "A scope left `paused` on an entity that then closed stayed paused permanently" |
| C85 | paused, close | Moves to ended. | `work/epics/archive/E-0090-…/M-0325-…` AC-4 | undated | "An entity carrying one paused scope reaches a terminal status; afterwards that scope's replayed state is `ended`." |
| C86 | active, close; paused, close | Only active scopes get `aiwf-scope-ends`; the source says nothing of paused ones. | `docs/archive/pocv3/provenance-model-plan.md` Step 6 | undated | "when `Apply` is processing a `promote` verb whose target state is terminal for the entity's kind, it queries all active scopes whose scope-entity is the verb's target, and writes one `aiwf-scope-ends: <auth-sha>` trailer per matched scope into the same commit." |
| C87 | paused, close | Stays paused permanently (stated as the behaviour at the time). | `work/epics/archive/E-0090-…/M-0323-…` §"Decisions made during implementation" | undated | "The automatic end's predicate collects only `active` today, so a paused scope survives its entity's closure permanently; ADR-0047 rules that a defect and M-0325 carries the one-predicate fix." |
| C88 | active, close | Moves to ended. | `internal/skills/embedded-rituals/…/aiwfx-wrap-milestone/SKILL.md` §10 | undated | "before the milestone's own promote-to-`done` step below, which ends a delegated milestone's authorize scope." |
| C89 | active, close [run by the agent] | Allowed: milestones are outside the sovereign (human-only) terminal edges. | `docs/adr/ADR-0047-…` §"Every edge into a terminal epic status is a sovereign act" | undated | "Milestones share the same terminal shape and are deliberately not included." |
| C90 | active, close [run by the agent] | Allowed; the commit carries both `aiwf-authorized-by` and `aiwf-scope-ends` for the same scope; the scope ends. (The example is an epic; ADR-0047 later made an epic's close human-only.) | `docs/design/provenance-model.md` §"Worked examples", Example 5 | undated | "The agent can close its own scope by design: scopes constrain *what* an agent can do, not *which step within scope* requires synchronous human ratification." |
| C91 | active, close [run by the agent], then check; closed [wrap commits after the close], check | No finding on the closing commit; wrap-epic and wrap-milestone commits after the close are forgiven. | `work/gaps/archive/G-0111-…` §"What's missing" | undated | "`internal/check/provenance.go` carries an equality carve-out for a terminal-promote commit that ends the very scope it acts under, and `isWrapBundleCommit` in the same file forgives wrap-epic and wrap-milestone commits landing after a same-entity terminal promote." |
| C92 | closed [milestone `done`], close; closed [milestone `cancelled`], close | `done → done` is a NoOp; `cancelled → done` is refused. | `work/epics/archive/E-0089-…/M-0318-…` §"Outcome measurement and row judgments" | 2026-08-24 (measured date in the spec) | "Terminal entity and AC statuses: the self-target converges; every other status in the kind's domain is refused." |
| C93 | active, close `[epic]` | Proposal: refuse the epic's promote to done while an active scope exists. ADR-0047 and E-0090 later kept the automatic end. | `work/gaps/archive/G-0063-…` sub-decision 3 | undated | "under the new rule, `aiwf promote E-NN done` should refuse while active scopes exist on the epic, requiring the operator to end scopes explicitly first." |

### rebase

| # | Case | What the source says happens | Source and where | Date | Quote |
|---|---|---|---|---|---|
| C94 | active, rebase, then check | The opening commit's SHA can go stale; `check` validates at read time and catches the rewrite. | `docs/design/provenance-model.md` §"Closed-set constraints on values" | undated | "A SHA that exists at write time can become stale later (rebase, force-push, branch-prune)." / "Read-time validation, run on every `aiwf check`, catches both the typo case and the later-rewrite case with one mechanism." |
| C95 | active, rebase, then check | SHAs hold unless force-pushed; a force-push is caught by `provenance-authorization-missing`. | `docs/design/provenance-model.md` §"Scope id" | undated | "SHAs are stable in the absence of force-pushes; force-push handling falls under the read-time stale-SHA detection (`provenance-authorization-missing`)." |
| C96 | active, rebase [opening commit in the rebased range: a milestone scope opened on the epic branch], then check | The scope stays on the branch under a new SHA; every earlier agent commit draws `provenance-authorization-missing` at error; the hooked push is refused (measured). | `work/gaps/G-0751-…` §"What's missing" | 2026-10-28 | "A `git rebase` that re-creates the `authorize` commit gives it a new SHA and leaves every later `aiwf-authorized-by` naming the old one. The scope is still on the branch, opened by the same act for the same agent, and each commit an agent made in it before the rebase becomes an error." |
| C97 | active, rebase [opening commit not in the rebased range: an epic scope opened on `main`], then check | No error; push passes (measured). | `work/gaps/G-0751-…` §"What's missing" | 2026-10-28 | "An epic scope, which `aiwfx-start-epic` opens on `main`, is not re-created by rebasing the epic branch: with it, two agent verbs on the epic branch and a rebase leave `aiwf check` at no error and the push passes." |
| C98 | active, [merge of `main` into the branch instead of a rebase], then check | No error; push passes (measured). | `work/gaps/G-0751-…` | 2026-10-28 | "Merging `origin/main` into the epic branch instead keeps every SHA, and `aiwf check` reports no error and the hooked push passes." |
| C99 | active, rebase, then check | `provenance-authorization-missing`; push refused (measured). | `docs/research/collaborative-state-and-commit-granularity.md` §8 E6 | 2026-10-28 | "`aiwf check` then reported `error provenance-authorization-missing: … does not resolve to an authorize/opened commit`, and the hooked push was refused." |
| C100 | active, act, rebase, then check; ended, rebase | Check fails; the ended scope reads active (measured by verifier-kit against a spine without pause). | `docs/research/collaborative-state-and-commit-granularity.md` §9 | 2026-10-28 | "`authorize, end, rebase` left the scope active (E8), and `authorize, act, rebase, check` failed the check (E6)." |
| C101 | active, rebase, then check | The pipeline refuses with `provenance-authorization-missing` (model run). | `docs/research/transactions/model/RESULTS.md` headline item 5 | 2026-10-30 | "Every later `aiwf-authorized-by` on the branch then names a commit that no longer exists, and the pipeline refuses with `provenance-authorization-missing` (L✗ in `auth`)." |
| C102 | active, rebase; ended, rebase; active, copy [scope opened from ADR-0056's release on] | The scope survives a rebase; a copy opens no second scope. Scopes opened before that release keep today's behaviour. | `docs/adr/ADR-0056-…` §Consequences | 2026-11-01 | "**A rebase no longer breaks a scope opened from this release on,** nor does any cherry-pick or merge that keeps commit messages, and a copy of such a scope's opening commit no longer opens a second scope." / "**A scope opened before this release stays breakable until it ends.**" |
| C103 | ended, rebase [opening commit in the rebased range] | Reads active again; the agent's verb succeeds with `aiwf-authorized-by` naming the re-created commit; `check` and the hooked push report no error (measured). | `work/gaps/G-0752-…` §"What's missing" | 2026-10-28 | "A `git rebase` re-creates the `authorize` commit under a new SHA and leaves the end commit naming the old one, so the re-created scope has no end. The verbs treat it as active, `aiwf show` reports it `active`, and the check finds no end to hold an agent's later commit against" |
| C104 | ended, rebase | Same as C103 (measured). | `docs/research/collaborative-state-and-commit-granularity.md` §8 E8 | 2026-10-28 | "The same retitle then succeeded, recording `aiwf-authorized-by` with the re-created commit's SHA, and `aiwf check` and the hooked push reported no error." |
| C105 | ended, rebase | The end names nothing after the rebase; nothing reports it (model run). | `docs/research/transactions/model/RESULTS-amend.md` headline | 2026-11-08 | "the operator's or the host's rebase re-creates the authorize commit, and the end names nothing (G-0752). Nothing reports it." |
| C106 | closed, rebase [wrap path: agent verb on milestone branch, wrap merge, promote done on the epic branch, epic branch rebased] | Reads active again while the milestone stays done; the agent's verb succeeds; `check` raises only G-0751's `provenance-authorization-missing` on the earlier commit (measured). | `work/gaps/G-0752-…` §"What's missing" | 2026-10-28 | "a plain rebase of the epic branch onto `main` turns the scope from `ended` to `active` and the agent's retitle succeeds. There `aiwf check` raises one error, the `provenance-authorization-missing` G-0751 records, on the agent's earlier commit; nothing reports the later act." |
| C107 | ended, rebase [scope opened from ADR-0056's release on] | Stays ended; the agent's verb is refused. | `docs/adr/ADR-0056-…` §Validation | 2026-11-01 | "re-run on a scope this release opens, report no `provenance-authorization-missing`, keep the ended scope ended and refuse the agent's verb after the rebase" |

### copy

| # | Case | What the source says happens | Source and where | Date | Quote |
|---|---|---|---|---|---|
| C108 | active, copy | Two active scopes (measured). | `work/gaps/G-0755-…` §"What's missing" | 2026-11-01 | "A copy of the opening commit has the same trailers and another SHA, so one grant reads as two active scopes." |
| C109 | active [after a copy], pause | One copy moves to paused, the other stays active; the agent's verb succeeds under the active one (measured). | `work/gaps/G-0755-…` §"Why it matters" and reproduction | 2026-11-01 | "`--pause` pauses one and reports nothing about the other" |
| C110 | active [after a copy], end | Without `--scope`: refused, listing both. With `--scope` naming the original: that one ends, the copy stays active, the agent's verb succeeds, `check` reports no error (measured). | `work/gaps/G-0755-…` | 2026-11-01 | "`aiwf authorize E-0001 --end --reason "delegation over"` refused, listing both SHAs as `2 non-ended scopes`; `--end --scope` naming the original ended it (exit 0) and left the copy `active`; the agent's retitle then succeeded (exit 0) with `aiwf-authorized-by` naming the copy, and `aiwf check` reported no error." |
| C111 | active [after a copy], close | Both copies move to ended (measured). | `work/gaps/G-0755-…` | 2026-11-01 | "The automatic end covers both: `aiwf promote E-0001 done` wrote an `aiwf-scope-ends` for each copy, and `aiwf show` reported both `ended`." |
| C112 | active, copy | Each copy reads as a scope of its own today. | `docs/adr/ADR-0056-…` §Context | 2026-11-01 | "Today aiwf reads each copy of an opening commit as a scope of its own" |
| C113 | active, copy [opening commit carries an `aiwf-scope-id`] | One scope; the first copy in history opens it; later copies change nothing. | `docs/adr/ADR-0056-…` §Decision | 2026-11-01 | "**One id is one scope.** Every opening commit carrying an id opens the same scope. The first in history order is its opener, and a later copy changes nothing." |

### check

| # | Case | What the source says happens | Source and where | Date | Quote |
|---|---|---|---|---|---|
| C114 | ended, check [a commit names a scope already ended at that commit] | `provenance-authorization-ended`, error. | `docs/design/provenance-model.md` §"`aiwf check` rules" | undated | "`aiwf-authorized-by:` SHA resolves to a real authorize commit whose scope has reached `ended` state at the time of the referencing commit." |
| C115 | ended, check [hand-made agent commit after `--end`, no rebase] | `provenance-authorization-ended`, error (measured). | `work/gaps/G-0752-…` | 2026-10-28 | "Without the rebase the check does catch the act: an agent commit made by hand after `--end`, naming the original `authorize` commit, draws `provenance-authorization-ended` at error severity." |
| C116 | none, check [an `ai/` commit without `aiwf-on-behalf-of`] | `provenance-no-active-scope`, error. | `docs/design/provenance-model.md` §"`aiwf check` rules" | undated | "An `ai/...` actor produced a commit with no `aiwf-on-behalf-of:` (caught at verb time; this code surfaces it on hand-edited or externally-authored commits)." |
| C117 | any, check [`aiwf-authorized-by` does not resolve to an opening commit] | `provenance-authorization-missing`, error. | `docs/design/provenance-model.md` §"`aiwf check` rules"; `internal/skills/embedded/aiwf-check/SKILL.md` | undated | "`aiwf-authorized-by:` SHA does not resolve, or resolves to a commit whose `aiwf-verb:` is not `authorize`." |
| C118 | paused, check | No rule covers pauses. | `work/gaps/G-0755-…` §"Why it matters" | 2026-11-01 | "`aiwf check` has no rule for pauses, so on that path what the copy defeats is the verb's own refusal." |
| C119 | ended or closed, check [after a rebase, `aiwf-scope-ends` names no opening commit] | No finding. | `docs/research/transactions/model/RESULTS-amend.md` §conditions | 2026-11-08 | "`aiwf check` has no rule for an `aiwf-scope-ends` that names no opening commit (the readers match it by equality, `internal/scope/scope.go:190`), so such a T6 failure is never reported." |
| C120 | ended, check [after a rebase] | Same as C119. | `work/gaps/G-0752-…` §"Relates to" | 2026-10-28 | "the end commit's `aiwf-scope-ends`, left naming a commit no longer on the branch, draws no finding." |

### Events outside the list

| # | Case | What the source says happens | Source and where | Date | Quote |
|---|---|---|---|---|---|
| C121 | active, [`aiwf archive` sweeps the milestone] | The scope survives; `aiwf show` and `--pause` still resolve it. | `work/gaps/archive/G-0393-…`; also `work/epics/archive/E-0062-…/M-0243-…` AC-3 | undated | "the milestone's own authorize scope (if genuinely still active at sweep time) survives the sweep correctly — `aiwf show`'s `scopes` array and `aiwf authorize --pause` both continue to resolve it after the archive move." |
| C122 | active, [`aiwf reallocate` renumbers the milestone] | The scope follows the renamed id. | `docs/archive/pocv3/provenance-model-plan.md` Step 6 | undated | "**Scope-entity resolution walks the `aiwf-prior-entity:` chain.**" |

## 2. Conflicts

| Case | One side | Other side | Note |
|---|---|---|---|
| none, act [`ai/` actor with `--principal`, no scope] | Refused, `provenance-no-active-scope`: C29, C32, C33, C34, C35, C36 | Succeeds with actor and principal trailers: C37, C38, C39, C40 | G-0706 records this disagreement. |
| ended, act; closed, act | Refused at the verb: C43, C44, C49 | The verb writes a commit naming the ended scope, and `check` reports `provenance-authorization-ended`: C46, C47 `[epic]`, C48 `[epic]`; C45 unclear | C45 and C48 do not say whether a verb wrote the commit. |
| paused, close | Moves to ended: C71, C81, C82, C83, C84, C85 | Stays paused: C86 (only active scopes get the trailer), C87 | C84 (2026-08-30) and C87 describe stays-paused as the behaviour before a change. |
| active, end; paused, end | An operator end exists and moves the scope to ended: C67–C75 | No operator end exists: C76 (2026-08-19), C77 (2026-08-23), C78 (open gap), C79 (2026-05-05) | C74 dates the end mode's release 2026-08-30. |
| active, end [several non-ended scopes] | Ends one, the named or the sole candidate, else refused: C69, C72 | "ends any opened scope": C80 `[epic]` | C80's wording is unclear. |
| none, authorize [how the scope is named] | By the opening commit's SHA, no separate id: C9, C11, C12 | By a random id the opening commit sends as `aiwf-scope-id`: C13 | C13 is an accepted ADR described elsewhere as not built. |
| active, authorize | A second scope opens; two active: C17, C18 | "one scope per entity": C19 | C19 is unclear (see C19). |
| closed, authorize | Refused unless `--force --reason`: C8, C14, C15 | The verb has no from-state meaning; authorizing a cancelled milestone is a question its semantics do not raise: C16 | C16 is unclear. |
| none, authorize [AI target, `--branch` names a branch that does not exist yet] | Refused, `branch-not-found`: C24 | Accepted from trunk or a ritual-shape checkout: C25; `branch-not-found` no longer raised: C26 | |
| none, authorize [AI target, checkout already on a ritual-shape (milestone) branch] | Refused, `branch-context-required`: C27 | Accepted: C28 | |
| no active scope, pause; no paused scope, resume | Refused with `provenance-no-active-scope-to-pause` / `provenance-no-paused-scope-to-resume`: C53, C54, C62 | Refused with no code: C56, C57 | Both sides agree on the refusal; they disagree on what it sends. |
| paused, pause; active, resume | Refused (exit 2 per C58): C53, C58, C62 | NoOp at exit 0, no commit: C59 | C59 is a general rule whose cited ADR covers only `promote` and `cancel`. |
| paused, resume [which scope, with several] | The most recently paused: C61 | The most recently opened active: C65 | C65 is unclear. |
| active, rebase [opening commit in the rebased range], then check | `provenance-authorization-missing`, push refused: C94, C95, C96, C99, C100, C101 | No break: C102 | C102 is limited to scopes opened from ADR-0056's release on. |
| ended, rebase | Reads active again; agent verbs succeed; no finding: C100, C103, C104, C105 | Stays ended: C4, C107 | C107 is limited to scopes opened from ADR-0056's release on. |
| closed, rebase | Reads active again while the milestone stays done: C106 | Stays ended: C4 | |
| active, copy | Two active scopes: C108, C112 | One scope; the copy changes nothing: C102, C113 | C102 and C113 are limited to opening commits that carry a scope id. |
| none, authorize [entity is a milestone] | A milestone is a scope-entity; the ritual opens a scope on it: C10, C21 | "An epic is the unit a scope is opened on": C22 | C22 is unclear. |

## 3. Not covered

Every state (`none`, `active`, `paused`, `ended`, `closed`) and every event (`authorize`, `act`, `pause`, `resume`, `end`, `close`, `rebase`, `copy`, `check`) has at least one claim.

These (state, event) pairs have no claim:

- `none`, `close`
- `none`, `rebase`
- `none`, `copy`
- `paused`, `rebase`
- `paused`, `copy`
- `ended`, `copy`
- `closed`, `copy`

These pairs are covered only by a claim stated for "no active scope", "no paused scope", "any non-terminal entity" or "an already-ended scope", not for the state by name:

- `paused`, `authorize` (C8)
- `ended`, `pause`; `closed`, `pause` (C53, C54, C55)
- `ended`, `resume` (C62, C63)
- `closed`, `end` (C70, C72)

## 4. Names that don't fit

### States

| Name | Where | Note |
|---|---|---|
| `ended` for both `ended` and `closed` | every source: `provenance-model.md` §"Scope states", the `aiwf-authorize` and `aiwf-promote` skills, ADR-0047, CHANGELOG [0.34.0] | Sources have one ended state. They tell the two routes apart only by the commit that ends the scope: `aiwf-verb: authorize` for an operator end, `promote` or `cancel` for the automatic one (`provenance-model.md` §"The `aiwf authorize` verb"; ADR-0047). |
| `opened`, `resumed` (and `paused`) | `aiwf-scope:` trailer values: `provenance-model.md` trailer table and closed-set table; `aiwf-history` skill chips | Event markers on the opening, pause and resume commits, not states in the list. |
| `open` | `docs/research/collaborative-state-and-commit-granularity.md` §9 state table | For `active`. |
| `live` | G-0460 ("a live one"); ADR-0047 §Validation ("one live scope per entity"); E-0090 epic and wrap | Used for active or non-ended. |
| `non-ended` | ADR-0047, the `aiwf-authorize` and `aiwf-promote` skills, CHANGELOG [0.34.0] | `active` or `paused`. |
| `stranded` | ADR-0047 §"Ending covers every non-ended scope"; CHANGELOG [0.34.0] heading; E-0090 | A paused scope on a closed milestone, which the list cannot express. |
| `ended-success`, `ended-failure`, `ended-discarded` | `docs/archive/pocv3/agent-orchestration.md` §4.2 and §7.4; `docs/archive/pocv3/_scratch-subagents-research.md` | Sub-scope end states for subagent cycles. |
| `done`, `cancelled` (milestone statuses) | `provenance-model.md` §"Scope states" and §"`--force` on authorize"; M-0318 | The list's `closed` merges them; sources treat them differently for `close` (C92) and speak of resurrecting "a cancelled entity" (C14). |

### Events

| Name | Where | Note |
|---|---|---|
| `cancel` (`aiwf cancel` of the milestone) | `provenance-model.md` §"Scope states"; `aiwf-authorize` skill §Lifecycle; ADR-0047; CHANGELOG [0.34.0] | Ends scopes as `close` does; the list's `close` names only the promote to `done`. |
| un-cancel of the milestone | `provenance-model.md` §"Scope states"; `aiwf-authorize` skill; R-FP-0129 | |
| `authorize --force` on a terminal milestone | `provenance-model.md` §"`--force` on authorize"; `aiwf-authorize` skill §"Common refusals" | Opens a new scope where plain `authorize` is refused. |
| merge of `main` into the branch | G-0751 | The alternative to `rebase`; keeps every SHA. |
| wrap merge (`aiwf-verb: wrap-milestone`) | G-0751, G-0752; `aiwfx-wrap-milestone` skill | |
| force-push | `provenance-model.md` §"Scope id"; G-0751 | |
| cherry-pick | G-0755; ADR-0056 | One way a `copy` happens. |
| push (pre-push hook) | G-0751, G-0752, collaborative-state §8 | The hook runs `aiwf check`; sources report the outcome as the push passing or being refused. |
| `aiwf acknowledge illegal` | G-0751; `provenance-model.md` §Ratification | Clears findings on a named commit; a later rebase brings them back (G-0751). |
| `aiwf archive` | G-0393; M-0243 | |
| `aiwf reallocate` of the milestone | `provenance-model-plan.md` Step 6; G-0118 | |
| branch rename (`git branch -m`); `aiwf scope rebind` (proposed) | G-0206; M-0161 AC-6; G-0225 | |
| `revoke` (`aiwf revoke`, `aiwf-revoked-by:`) | `gaps-pre-migration.md` G22; G-0022; ADR-0047 References | Proposed name for an operator end; never built. |
| `authorize-end` | G-0225 ("no `authorize-end` event has fired on it") | |
| amend of a verb commit | ADR-0058 (proposed); `docs/research/transactions/model/RESULTS-amend.md` | |
| squash and the host's merge methods | `RESULTS.md`; ADR-0056 §Consequences | |
| AI-actor code commit (not a verb) | E-0030 epic corner cases; M-0106 | Read by `isolation-escape`; distinct from `act`, which is a verb. |

### Variables and values

None are in the list. Sources condition outcomes on:

- actor role (`human/`, `ai/`, `bot/`) and whether `--principal` is passed (C8, C29–C40);
- `--force` / `--reason` (C14, C15, C23);
- `--branch`, the current checkout, ritual branch context and the rung pair (C23–C28, C50);
- `--scope <auth-sha>` and its four-character prefix floor (`provenance-model.md` §"Verb behavior");
- the number of non-ended scopes on the milestone: zero, one, several (C17, C60, C69, C110);
- the entity's kind (C21) and the milestone's status, `in_progress`, `done` or `cancelled` (C8, C92);
- whether the opening commit lies in the rebased range: a milestone scope opened on the epic branch or an epic scope opened on `main` (C96, C97);
- whether the scope was opened before or after ADR-0056's release (C102, C107, C113);
- the host's merge method and squash message (`RESULTS.md`).

### Outputs

None are in the list. Sources say these are sent:

- trailers: `aiwf-scope: opened | paused | resumed`, `aiwf-scope-ends`, `aiwf-authorized-by`, `aiwf-on-behalf-of`, `aiwf-principal`, `aiwf-to`, `aiwf-reason`, `aiwf-branch`, `aiwf-branch-sha`, `aiwf-scope-id` (ADR-0056), `aiwf-force`;
- refusal and finding codes: `provenance-no-active-scope`, `provenance-authorization-missing`, `provenance-authorization-ended`, `provenance-authorization-out-of-scope`, `provenance-trailer-incoherent`, `provenance-no-active-scope-to-pause`, `provenance-no-paused-scope-to-resume`, `branch-context-required`, `branch-not-found`, `rung-pair-illegal`, `authorize-kind-not-allowed`, `isolation-escape`, `isolation-escape-orphaned-ai-commit`;
- `aiwf history` chips: `[scope: opened]`, `[<scope-entity> <auth-short>]`, `[<scope-entity> ended]`, `[scope: <auth-short> ended]` (`aiwf-history` skill), and `[E-03 paused]` / `[E-03 resumed]` (`provenance-model.md` §"Render-side").
