# Decisions on the scope machine's audit

The maintainer's answers to the questions raised by lining up `docs.md` and `history.md`
(verifier-kit guide, step 3, prompt B3). Each answer gives its date, its reason, the sources it
makes wrong and the claims it settles. Claim ids C-n are rows of `docs.md`, H-n rows of
`history.md`.

## Q1. `ended` or `closed`, then `rebase`

- **Date:** 2026-11-21.
- **Answer:** a rebase leaves the scope as it was. `ended, rebase` stays `ended`, and
  `closed, rebase` stays `closed`.
- **Reason:** ADR-0056, accepted 2026-11-01, decides it. Its Decision has every
  `aiwf-scope-ends` hold the scope id ("The two pointers keep their keys and hold the scope
  id"), which a rebase keeps, and its Validation holds while the G-0752 reproduction, re-run on a
  scope it opens, does "keep the ended scope ended and refuse the agent's verb after the rebase".
  The provenance model (§"Scope states", "`ended` is terminal", 2026-05-02), ADR-0047
  (§"Nothing undoes an end") and M-0325 ("nothing undoes an end", 2026-08-29) say the same.
- **Applies to:** a scope opened from ADR-0056's release on. A scope opened before it "stays
  breakable until it ends" (ADR-0056, Consequences). Whether the machine carries that difference
  is question 16.
- **Sources this makes wrong:** none of the docs. Today's binary differs: G-0752 (2026-10-28)
  and history runs rb2 and rb3 (H77, H78, 2026-11-21) measure an ended or closed scope reading
  `active` after a rebase, because an end matches its scope by the opening commit's exact SHA
  (H58). G-0752 records that divergence, so no new item is drafted.
- **Claims settled:** C4, C100, C103–C107, C119, C120; H4, H48, H49, H58, H77, H78.

## Q2. `active` or `paused`, then `rebase`, then `check`

- **Date:** 2026-11-21.
- **Answer:** a rebase that keeps the branch's commits changes nothing about the scope.
  `active, rebase` stays `active`, `paused, rebase` stays `paused`, and a later `check` passes
  whether or not the agent committed before the rebase.
- **Reason:** a scope records that the human allowed this agent to work on this milestone, and
  which commits the agent made under that permission. A rebase replays those commits on a newer
  base with their content, messages and authors kept; only their hashes change, so neither fact
  changes. Today's error comes from naming a scope by its opening commit's hash. ADR-0056 names
  it by an id a rebase keeps ("A rebase no longer breaks a scope opened from this release on",
  Consequences), and its Validation requires the G-0751 reproduction, re-run on such a scope, to
  "report no `provenance-authorization-missing`". A warning after a rebase, such as
  `isolation-escape-orphaned-ai-commit` (H62, H75), does not make `check` fail.
- **Outside the `rebase` event:**
  - A rebase that drops the opening commit, or a squash that strips its trailers, does change
    the branch's history, and an error is then right. Squash is among the events question 15
    takes up.
  - A `main` that carries a change to the milestone, such as a cancel, brings it onto the branch
    with the rebase, and the scope should end because that change arrived. That is two clients
    acting at once, which E-0100 leaves out; the chapter names it as a limit. Today's behaviour
    in that case is not measured.
- **Applies to:** a scope opened from ADR-0056's release on (question 16).
- **Measured 2026-11-21**, for the two cases no source covers: binary built from cc0d64cff with
  `go build -o <scratch>/aiwf ./cmd/aiwf`, Go 1.25.12 linux/amd64, Linux devcontainer. Fixture as
  in `history.md` (epic E-0001 active, milestone M-0001 with two ACs), then on branch
  `milestone/M-0001-first-milestone`: `aiwf authorize M-0001 --to ai/claude --reason go`, a
  commit on `main`, `git rebase main`, `aiwf show M-0001`, `aiwf check`.
  - With no agent act: expected no error. Observed scope `state: active`, `check` exit 0,
    `7 findings (0 errors, 7 warnings)`.
  - With `aiwf add ac M-0001 --title three --actor ai/claude --principal human/t` and
    `aiwf authorize M-0001 --pause hold` before the rebase: expected the G-0751 error. Observed
    scope `state: paused`, `check` exit 1,
    `error provenance-authorization-missing: commit a0e92cc: aiwf-authorized-by: f296e95 does not resolve to an authorize/opened commit`.
- **Sources this makes wrong:** the provenance model, §"Scope id" (2026-05-02): "A scope is
  addressed by the SHA of its `authorize` commit. There is no separate "scope id" namespace."
  ADR-0056 already makes it wrong; question 12 takes it up. Today's binary differs where the
  agent committed before the rebase (G-0751, H75, and the paused measurement above); G-0751
  records that divergence, so no new item is drafted.
- **Claims settled:** C94–C102; H21, H62, H75, H79; not covered: `paused, rebase`.

## Q3. `copy`, in every state

- **Date:** 2026-11-21; `none`, 2026-11-26.
- **Answer:** a second copy of the opening commit changes nothing. `active`, `paused`, `ended`
  and `closed` each stay as they are, and `none`, where there is no opening commit to copy,
  stays `none`; answers.md Q2 says what copy does there.
- **Reason:** the reason given for question 2: a scope is the human's permission, and a second
  copy of the commit that recorded it does not grant it again. ADR-0056 decides the same: "**One
  id is one scope.** Every opening commit carrying an id opens the same scope. The first in
  history order is its opener, and a later copy changes nothing." (Decision), and "a copy of such
  a scope's opening commit no longer opens a second scope" (Consequences).
- **Applies to:** a scope opened from ADR-0056's release on (question 16).
- **Measured 2026-11-21**, for the three states no source covers: the binary of question 2. Fixture
  as in question 2, with AC bodies filled (`aiwf edit-body`) and `aiwf promote M-0001
  in_progress` on `epic/E-0001-engine`, then on `milestone/M-0001-first-milestone`: authorize,
  `git branch side/sibling`, a commit on `main`, `git rebase main`, the state's own step,
  `git merge --no-ff --no-edit side/sibling` (the copy), `aiwf show M-0001`,
  `aiwf retitle M-0001 "Renamed by agent" --actor ai/claude --principal human/t`, `aiwf check`.
  Expected under today's code: a second, active scope.
  - `paused` (`aiwf authorize M-0001 --pause hold`): observed one scope `state: active` beside
    the `state: paused` one; the agent's retitle exit 0; `check` exit 0, no error.
  - `ended` (`aiwf authorize M-0001 --end --reason over`): observed the ended scope and a new one
    `state: active`; the agent's retitle exit 0; `check` exit 0, no error.
  - `closed` (both ACs met, `aiwf promote M-0001 done`): the merge conflicted on the milestone's
    `status:` line and was resolved keeping `done` (`git checkout --ours`). Observed the
    milestone `status: done`, the ended scope and a new one `state: active`; the agent's retitle
    exit 0; `check` exit 0, no error.
- **Sources this makes wrong:** none of the docs. Today's binary differs in every state with a
  scope. G-0755 (2026-11-01) records the `active` case and a copy that reopens a paused, ended
  or closed scope.
- **Claims settled:** C102, C108–C113; H76; not covered: `none, copy`, `paused, copy`,
  `ended, copy`, `closed, copy`.

## Q4. A second `authorize` while the scope is `active` or `paused`

- **Date:** 2026-11-21.
- **Answer:** from `active`, a repeat `authorize` changes nothing: exit 0, no commit. From
  `paused`, it is refused, and the refusal points to `--resume`.
- **Reason:** the machine is one agent's permission on one milestone. While the scope is active
  the permission is already there, so the request is already satisfied, which CLAUDE.md item 7
  makes a NoOp ("a request already satisfied is a NoOp at exit 0 with no commit"). While it is
  paused the request is not satisfied, and lifting a pause stays one deliberate step,
  `--resume`. The `Authorize` entry of `noopExemptVerbs`, which exempts it from the same-state
  check, gives as its reason that "a silent NoOp would mask the two-active-scopes defect"
  (`internal/policies/verb_result_noop_invariant.go:54`); that holds while more than one scope
  per agent and milestone might be legal.
  Questions 2–4 leave one, so a no-op hides no second scope.
- **Outside the machine's names:** a repeat that names another branch, and a grant to another
  agent, which is another machine (question 16).
- **Applies to:** every scope. An `authorize` writes a new opening commit whatever release opened
  the scope it repeats.
- **Sources this makes wrong:**
  - Code comment, `internal/cli/cliutil/scopes.go`: "Multiple parallel scopes on the same entity
    are supported". It changes with the code.
  - The `Authorize` entry of `noopExemptVerbs`
    (`internal/policies/verb_result_noop_invariant.go:54`), whose reason is that "a silent NoOp
    would mask the two-active-scopes defect". It comes out when the code changes.
  - Not wrong: design-decisions.md ("Multiple parallel scopes are supported") and the provenance
    model §"Multiple parallel scopes", whose example is one agent on two entities; the
    start-milestone skill's "one scope per entity"; G-0460 (2026-07-27), whose §"Resolution
    shape" cites this answer for a milestone and leaves the question open for an epic.
- **Today's binary differs:** a repeat opens a second scope from either state (G-0460; H8, H50,
  H66, H68). G-0460 records it, so no new item is drafted.
- **Claims settled:** C2, C17–C20, C31 (for one milestone); H8, H50, H66, H68; not covered:
  `paused, authorize`.

## Q5. `paused`, then `close`

- **Date:** 2026-11-21.
- **Answer:** `paused, close` moves to `closed`.
- **Reason:** nothing acts on a done milestone again, so a pause left behind could never be
  lifted (provenance model §"Ending covers paused scopes": "once the entity is terminal no verb
  will act on it again, so a paused scope there can never be resumed"). Today's code does this
  from 00012965f (2026-08-29, M-0325/AC-4; CHANGELOG 0.34.0), and run c2 measured it (H72).
- **Sources this makes wrong:** none live. `docs/archive/pocv3/provenance-model-plan.md` (C86)
  and M-0323's §"Decisions made during implementation" (C87) describe the behaviour before
  00012965f; both are archived.
- **Claims settled:** C71, C81–C87; H3, H14, H47, H55, H72.

## Q6. `active` or `paused`, then `end`

- **Date:** 2026-11-21.
- **Answer:** `active, end` and `paused, end` move to `ended`; the milestone's status does not
  change.
- **Reason:** the operator's end, `aiwf authorize <id> --end`, exists from 9dd1a2d7e (2026-08-29,
  M-0325/AC-1; CHANGELOG 0.34.0), and ADR-0047, the provenance model §"Scope states",
  design-decisions.md §"Provenance model (added in I2.5)" and the `aiwf-authorize` skill all
  describe it (H45).
- **Sources this makes wrong:** none live. Not corrected: G-0282 (open), whose §"Evidence (audit,
  discovered during M-0183 design)" says "`--pause` only parks a scope; reaching `ended` needs a
  terminal-promote of the scope-entity or `--force`", a dated audit record whose §Status
  (2026-07-21) already folds that finding into G-0022, now `wontfix`;
  `docs/audits/entity-truth-audit.md` (C76, dated 2026-08-19) and
  `docs/audits/gap-truth-audit.md` (C77, dated 2026-08-23), dated audits from before the end
  mode; `docs/archive/pocv3/gaps-pre-migration.md` (C79) and E-0028's §Constraints (C80),
  archived.
- **Claims settled:** C67–C80; H45, H52; H15, H20, H39 (history conflict 8, the window before
  the operator's end existed; G-0225 is archived as wontfix).

## Q7. `ended` or `closed`, then `act`

- **Date:** 2026-11-21.
- **Answer:** the verb refuses (`provenance-no-active-scope`) and writes no commit, so a later
  `check` passes.
- **Reason:** an ended scope grants nothing. Today's code refuses at the verb (H12; G-0752 and
  research §8 E8 measured it, C43, C44, C49), and every function that writes
  `aiwf-authorized-by` must run the allow rule, which
  `internal/policies/authorized_by_via_allow.go` enforces. A commit made by hand under an ended
  scope is `check`'s to report, as `provenance-authorization-ended` at error (C115, measured by
  G-0752); the machine's `act` is a verb, not a hand-made commit.
- **Sources this makes wrong:** filed as G-0782:
  - the wrap-milestone skill, §10 and anti-patterns: "A verb-driven commit produced after that
    point risks an ended-scope `aiwf-authorized-by:` trailer on push" (C46);
  - the wrap-epic skill, §7 "Why promote is last among verb-driven commits": "Any commit
    produced *after* this that goes through a kernel verb … " (C47), and the same claim in its
    §Constraints and §Anti-patterns;
  - the provenance model, §"Worked examples", Example 5: "Subsequent agent verbs referencing it
    as `aiwf-authorized-by: 4b13a0f` produce `provenance-authorization-ended`", which reads as
    the verb committing (C45).
  Not corrected: G-0119 (C48), addressed and archived, whose post-close commits were the
  ritual's own.
- **Outside the machine:** `aiwf import` applies no scope gating (question 17).
- **Claims settled:** C43–C49; H12, H49.

## Q8. `none`, then `act`

- **Date:** 2026-11-21.
- **Answer:** an agent's verb with no scope is refused and writes no commit, whether or not it
  names a principal: with `--principal` as `provenance-no-active-scope`, without it as
  "principal required for non-human actor". Measured 2026-11-22 with the `aiwf` `conform.md`
  names, on the adapter's starting repository: `aiwf add ac M-0001 --title x --actor ai/claude`,
  exit 1, HEAD unchanged; with `--principal human/t`, exit 1 with
  `provenance-no-active-scope`. H83 records the first refusal under an active scope.
- **Reason:** CLAUDE.md item 9: "An LLM the human directs in conversation is the human's tool,
  not a co-actor; `aiwf authorize` opens a scope an agent works within." A commit made in
  conversation has the human as its actor; an `ai/` actor is an agent working on its own, which
  needs a scope. Today's code refuses at the verb (G-0706, measured) and `check` reports such a
  commit at error (H82). A `bot/` actor is refused at the verb too, but `check` reports nothing
  for its commit (H84; G-0778).
- **Sources this makes wrong:** the passages G-0706 (open, 2026-09-24) lists.
- **Claims settled:** C29, C32–C40; H84 (the verb's refusal).

## Q9. Repeated and targetless requests: `pause`, `resume`, `end`, `close`

- **Date:** 2026-11-21; the answer for `end`, 2026-11-22.
- **Answer:**
  - A request that already holds does nothing, at exit 0 with no commit: `paused, pause`,
    `active, resume` and `closed, close`.
  - A request with nothing to act on is refused: `pause` from `none`, `ended` or `closed`;
    `resume` from `none`, `ended` or `closed`.
  - The machine's `end` is `--end` without `--scope`, which ends the milestone's one scope that
    is not ended. From `none`, `ended` and `closed` it is refused: no such scope is left.
  - Outside the machine: `--end --scope` naming a scope already ended, which does nothing at
    exit 0 with no commit.
  - Which code these refusals carry (C53, C54, C56, C57, C62) is question 18's.
- **Reason:** CLAUDE.md item 7, "a request already satisfied is a NoOp at exit 0 with no
  commit", and the verb-design guidance §"Same-state convergence — resolve, then converge": a
  request must name something real (R1) and converges when its effect already holds (R2). The
  same rule gives question 4's answer for a repeat `authorize`. `closed, close` converges by
  ADR-0036 (H80). For `end`, ADR-0047 §"An operator ends a scope by naming it, and `--end`
  defaults to the only candidate": "With no `--scope` and no candidate, the verb refuses rather
  than converging", and "Naming an already-ended scope with `--scope` does converge". `closed`
  is reached with the scope ended or with none ever opened (question 10). Played with `--scope`,
  `closed, end` would do nothing after the first and have no scope to name after the second,
  two answers for one case; without `--scope` both refuse.
- **Measured 2026-11-22**, the binary of question 2, fixture of question 3: from each of
  `none`, `active`, `paused`, `ended`, `closed` with no scope ever opened and `closed` after
  `active`, `aiwf authorize M-0001 --end --reason again`, then, where a scope exists,
  `aiwf authorize M-0001 --end --scope <its opening commit> --reason again`; on `closed` with no
  scope, `--scope 1234567`. Expected: the bare `--end` ends the scope
  from `active` and `paused` and is refused elsewhere; the named one does nothing on an ended
  scope. Observed: from `active` and `paused`, exit 0 with a commit; from `none`, `ended` and
  both `closed`, exit 2, no commit, "no non-ended scope on M-0001 to end"; the named `--end` on
  an ended scope, exit 0, no commit, "scope … on M-0001 is already ended; nothing to end"; the
  unknown id, exit 2, "no scope on M-0001 matches --scope "1234567"".
- **Today's binary differs:** `paused, pause` and `active, resume` are refused (H64). M-0364/AC-4
  classifies it as a defect in the code, G-0786.
- **Sources this makes wrong:** none while the code refuses a repeat; these describe that
  refusal, in words that read as covering a repeat:
  - the provenance model, §"Verb behavior": "If none active, refuses with
    `provenance-no-active-scope-to-pause`" and "If none paused, refuses with
    `provenance-no-paused-scope-to-resume`" (C53, C62);
  - `docs/design/legal-workflows-first-principles.md` R-FP-0135 and R-FP-0136 (C54);
  - the `aiwf-authorize` skill, §"Common refusals": "`--pause` with no active scope on the
    entity: nothing to pause; the verb refuses." and "`--resume` with no paused scope on the
    entity: nothing to resume." (C55, C63).
  G-0786, the defect M-0364/AC-4 filed for the code, lists each of these. Not corrected in any
  case: `docs/archive/pocv3/health-scorecard-2026-06-16.md` (C58), archived.
- **Claims settled:** C53–C55, C58, C59, C62, C63, C70, C72, C73; H6, H64, H80 (the repeat); not
  covered: `closed, close`, `ended, pause`, `closed, pause`, `ended, resume`, `closed, end`.

## Q10. `none` or `ended`, then `close`; what `closed` means

- **Date:** 2026-11-21.
- **Answer:** `closed` means the milestone is done or cancelled, and any scope on it is ended.
  `none, close` and `ended, close` both move to `closed`. The step-2 description of `closed`
  reads "the milestone reached done or cancelled; any scope on it is ended".
- **Reason:** after a close, the milestone takes no new scope whatever state its scope was in:
  `authorize` refuses a terminal entity (`internal/verb/authorize.go`; provenance model §"Verb
  behavior", C8). Keeping `none` or `ended` after a close would let the machine's `authorize`
  through where aiwf refuses it, unless the milestone's status became a variable.
- **Measured 2026-11-21**, the binary of question 2, fixture of question 3, script steps: with no
  scope, and with `aiwf authorize M-0001 --to ai/claude --reason go` then
  `aiwf authorize M-0001 --end --reason over`; then both ACs met, `aiwf promote M-0001 done`,
  `git log -1 --format=%B`, `aiwf show M-0001`, `aiwf authorize M-0001 --to ai/claude --reason
  again`. Expected: no `aiwf-scope-ends` on the closing commit, and the new authorize refused.
  Observed in both: no `aiwf-scope-ends`; no scope listed, or the one scope `state: ended`;
  the authorize exit 2, "M-0001 is at terminal status "done"; pass --force --reason "..." to
  authorize work on a terminal entity".
- **Sources this makes wrong:** none; no source covers these two cases.
- **Claims settled:** C81 (the ended scope gets no end); H55, H80; not covered: `none, close`.

## Q11. `closed`, then `authorize`

- **Date:** 2026-11-21.
- **Answer:** `closed, authorize` is refused. `aiwf authorize --force --reason` on a closed
  milestone is a human override outside the machine, listed with its other variables
  (question 16).
- **Reason:** the machine models the ordinary verb, which refuses a terminal entity (H9;
  measured for question 10). The forced path opens a new scope on the done milestone, under which
  pause, resume and an agent's verb succeed (H73); the provenance model §"`--force` on authorize"
  and the `aiwf-authorize` skill §"Common refusals" document it as the way to reopen work.
- **Sources this makes wrong:** none. The provenance model §"Ending covers paused scopes" ("once
  the entity is terminal no verb will act on it again") and the doc comment in
  `internal/cli/cliutil/provenance.go` speak of ordinary verbs; their conclusion, that a paused
  scope on a closed milestone can never be resumed, holds by question 5.
- **Claims settled:** C8, C14–C16; H9, H49, H73.

## Q12. A scope's name, and when the docs that call it a SHA are corrected

- **Date:** 2026-11-21.
- **Answer:** the chapter names a scope by ADR-0056's random id, carried by its opening commit
  as `aiwf-scope-id`. The docs that name it by the opening commit's SHA are corrected in the
  epic that builds ADR-0056, not after this spine is ratified; until that build they describe
  aiwf correctly. The same timing holds for every doc that questions 1–3 make wrong.
- **Reason:** questions 1–3 rest on a name that a rebase or a copy keeps, which is ADR-0056's
  (accepted 2026-11-01). ADR-0056 is the foundation of the fewer-planning-commits path: the
  research's working answer builds amend "on a foundation every write-side option needs: a
  scope id that survives a rewrite" (`docs/research/fewer-planning-commits.md`), and ADR-0058
  writes no amended commit "unless it also writes ADR-0056's scope ids". The machine has no
  outputs, so the name reaches the spine only through what rebase and copy do. A doc corrected
  before the build would describe behaviour aiwf does not have.
- **Sources this makes wrong, once ADR-0056 is built:** deferred to the epic that builds
  ADR-0056 (M-0364, §Deferrals): every passage of the docs and skills that names a scope by its
  opening commit's SHA, or gives `aiwf-authorized-by` or `aiwf-scope-ends` a SHA's shape.
  ADR-0056 names the provenance model's (§Consequences, "What changes"). This search finds them
  and the others, leaving out the ADRs and the dated records under `docs/research`,
  `docs/archive` and `docs/audits`, which are not corrected. On `main` on 2026-11-22 it found 50
  lines in nine files, and every one was such a passage; a search cannot be shown complete, so
  the build epic runs it again and reads what it finds:

  ```sh
  LC_ALL=C git grep -n -i -E \
    -e 'auth[- ]?sha|auth-short|addressed by the sha|sha of (its|the) .?authorize|sha is the scope id' \
    -e '7.{1,3}40 hex|authoriz[a-z]* shas?\b|(authorized-by|scope-ends):?`? shas?\b|git sha of the' \
    -e 'shas of authorize|sha-points-to|authorize[- ]commit sha|(authorized-by|scope-ends): [0-9a-f]{7}' \
    -e 'original sha|scope `[0-9a-f]{7}|its sha to' \
    -- docs internal/skills/embedded internal/skills/embedded-rituals README.md \
    ':!docs/research' ':!docs/adr' ':!docs/archive' ':!docs/audits'
  ```

  Among them:
  - the `aiwf-authorize` skill, §Lifecycle: "The commit's SHA is the scope id; subsequent verbs
    inside the scope reference it via `aiwf-authorized-by:`." (C9);
  - the provenance model, §"Scope id": "A scope is addressed by the SHA of its `authorize`
    commit. There is no separate "scope id" namespace." (C11; question 2);
  - `docs/design/legal-workflows-first-principles.md` §6d, R-FP-0137: "A scope is addressed by
    the SHA of its `authorize` commit. No separate scope id namespace." (C12).
  No gap is filed now: until the build these docs describe aiwf correctly, and a gap would record
  a defect that does not exist yet. This list, with question 16's unit test, reaches the build
  epic through this file, since that epic starts from this spec.
- **Claims settled:** C9, C11–C13.

## Q13. What `rebase` and `copy` do

- **Date:** 2026-11-21; the branch's other commits, and a rebase or copy after a copy, 2026-11-26.
- **Answer:** each event is played the way that breaks today's aiwf.
  - `rebase`: the scope is opened on a branch, `main` gains one commit that touches nothing of
    the scope, and the branch is rebased onto `main`, so each of the branch's own commits gets a
    new SHA, the opening commit among them.
  - `copy`: the branch as it stood before that rebase is kept and merged back after it, so each
    original lands beside its rebased twin, the opening commit among them.
  - After a copy, `rebase` and `copy` both rebase with `git rebase --rebase-merges`, which keeps
    the copy's merge: the commits it merged back in, the original opening commit among them, keep
    their SHAs, and the rest get new ones. A second copy so lands each commit given a new SHA
    beside the one it was rebased from, and the original opening commit stays once.
- **Reason:** under questions 1–3 no rebase or copy changes the scope, so the events are worth
  playing only in the form that tells ADR-0056's behaviour from today's. A rebase that leaves
  the opening commit alone, such as an epic scope opened on `main` (G-0751, C97), breaks nothing
  today; merging `main` in instead keeps every SHA (C98). G-0755's route for a copy is the one
  that happens by accident; a cherry-pick of the opening commit alone (ADR-0056, Consequences)
  is narrower. These are the steps measured for questions 2 and 3, and those of history runs rb,
  rb2, rb3 and pr. A plain `git rebase` after a copy drops the merge and replays the rebased
  commits and then the originals, so after a close the original promote to `in_progress` meets
  the `done` the rebased close wrote, and the rebase stops.
- **Measured 2026-11-26**, with the `aiwf` `conform.md` names and git 2.51.1, through the
  adapter's own steps from its starting repository: `authorize`, `act`, `rebase`; `authorize`,
  `copy`; `authorize`, `copy`, `rebase`; `authorize`, `copy`, `copy`; `authorize`, `close`,
  `copy`, `rebase`; then the last and `authorize`, `pause`, `copy`, `rebase` with
  `--rebase-merges` taken out of the adapter. Expected: with no earlier copy, every branch
  commit gets a new SHA; with the flag, the commits a copy merged back in keep their SHAs and the
  rest get new ones; without it, the rebase after `close` stops and the one after `pause` runs.
  Observed: the rebase after `act` kept none of the branch's commits, and the first copy left
  each original beside its rebased twin. With the flag, both rebases after a copy kept the SHA
  of every commit the copy merged back in, the original opening commit among them, and gave
  every other commit and the merge a new one; after the second copy the branch held three
  `authorize` commits, the original once. Without it, the rebase after `close` replayed the
  rebased commits, the close among them, then the originals, and stopped at the original
  `aiwf promote M-0001 draft -> in_progress`, its `status: in_progress` in conflict with
  `status: done`; the one after `pause` ran clean.
- **Sources this makes wrong:** none.
- **Claims settled:** C96–C98 (which rebase); H75–H79 (the steps).

## Q14. Names: one `ended` or two, several scopes, and `cancel`

- **Date:** 2026-11-21.
- **Answer:**
  - `ended` and `closed` stay two states, as question 10 defines them; the adapter tells them
    apart by the milestone's status, since today's code has one ended state.
  - One scope per agent per milestone (questions 3 and 4). Another agent's scope is another
    instance of the machine; a scope on the parent epic is question 16.
  - `cancel` stays out of the machine. `close` is the promote to `done`, and the chapter states
    that `aiwf cancel` ends the scope the same way.
- **Reason:** a milestone's `done` and `cancelled` are both terminal (`internal/entity/
  transition.go`), and for the scope they are the same: one code path ends the scope for both
  (H13, H55), and `authorize` refuses both because it asks whether the status is terminal. They
  differ only in a milestone rule: a promote to `done` converges on a `done` milestone and is
  refused on a `cancelled` one (M-0318, C92). A `cancel` leading to `closed` would give
  `closed, close` two answers, and a state of its own would add only that milestone rule, which
  belongs to the milestone's lifecycle.
- **Sources this makes wrong:** none.
- **Claims settled:** C1, C3, C92 (for the scope); H13, H55, H63, H86; the names `ended` for both
  routes, several scopes per milestone, `cancel`, and `done`/`cancelled` in both tables.

## Q15. Events the sources name that are not among the nine

- **Date:** 2026-11-21.
- **Answer:** the machine adds a tenth event, `reallocate` (`aiwf reallocate <milestone>`, which
  gives the milestone a new id), and it changes nothing about the scope in any state. Every other
  event both tables name stays outside the machine, and the chapter lists each with its reason:
  - decided earlier or an old name: `cancel` (question 14), `authorize --force` (question 11),
    cherry-pick of the opening commit and a merge of `main` in place of a rebase (question 13),
    `revoke` and `authorize-end` (proposed names for `--end`), un-cancel (no such transition);
  - changes what `check` reports or what the branch rules say, not the scope: push (the hook
    runs `aiwf check`), wrap commits after a close (H22), `aiwf acknowledge illegal`, a rename of
    the bound branch, an agent's code commit that is not a verb, `git cherry-pick -x` of an
    agent commit, a merge of a ritual branch into `main` (judged by first-parent ancestry),
    force-push and `git reset --hard`, a rebase in progress;
  - acts on the milestone without changing its status: `aiwf promote --audit-only`, which
    records a status the file already has (asked for `done` on an `in_progress` milestone, it
    refuses: "audit-only records what's already true"), and `aiwf archive`, which moves a
    terminal entity's file. Measured 2026-11-24 with the `aiwf` `conform.md` names, on the
    adapter's starting repository: with a scope active, the ACs met and the file's status set to
    `done` by a hand edit committed with `git commit`,
    `aiwf promote M-0001 done --audit-only --reason record` exited 0, its commit carried
    `aiwf-scope-ends`, the scope read `ended` and the agent's retitle was refused with
    `provenance-no-active-scope`, as a close leaves it (H55). With the edit left uncommitted,
    the retitle is refused for the uncommitted file instead, which says nothing of the scope.
    After a close, `aiwf archive --apply` printed "no terminal-status entities awaiting sweep
    (tree is converged)" and made no commit, since a milestone moves with its epic, which the
    chapter keeps active. C121 says a scope still active when its milestone is swept survives
    the sweep;
  - proposed and never built: scope rebind, an agent authorizing another, time-, verb- and
    pattern-limited scopes;
  - the verb commit output machine's (M-0366): amend, squash and the host's merge methods;
  - `aiwf import`, which skips the scope check (question 17).
- **Reason:** a reallocate can come after work under a scope: an id collision surfaces when the
  branches meet, which G-0118 records happening at an epic's wrap. It changes the milestone's
  number, not the human's permission, so by question 2's reason it changes nothing about the
  scope. Today it does, measured below; ADR-0056 changes what the pointers hold, not how a scope
  is matched to its milestone, so the defect outlives it. An event left out of the machine is one
  conform cannot check.
- **Measured 2026-11-21**, the binary of question 2, fixture of question 3, on
  `milestone/M-0001-first-milestone`: `aiwf authorize M-0001 --to ai/claude --reason go`,
  `aiwf add ac M-0001 --title three --actor ai/claude --principal human/t`,
  `aiwf reallocate M-0001` (printed `aiwf reallocate M-0001 -> M-0002`), then on M-0002: `aiwf
  show`, an agent `aiwf add ac`, `--pause hold`, `--resume go`, every AC met, `aiwf promote M-0002
  done`, `git log -1 --format=%B`, `aiwf show`, an agent `aiwf retitle`, `aiwf check`. Expected,
  by this answer: the scope listed and active on M-0002, pause and resume accepted, the close
  ending it. Observed:
  - `aiwf show M-0002` listed no scope; the agent's `add ac` exit 0, with `aiwf-authorized-by`
    naming the opening commit;
  - `--pause` exit 2, "no active scope on M-0002 to pause"; `--resume` exit 2, "no paused scope
    on M-0002 to resume";
  - the closing commit carried no `aiwf-scope-ends`; `aiwf show M-0002` then listed the scope
    `state: active` on the `done` milestone, and the agent's retitle exit 0;
  - `aiwf check` reported no provenance finding (its two errors were `acs-empty-body` on the
    agent's title-only ACs).
  - With `aiwf authorize M-0001 --end --reason over` before the reallocate, the agent's `add ac`
    on M-0002 was refused, `provenance-no-active-scope`.
- **Sources this makes wrong:** none. The defect is filed as G-0776, which locates the code
  that pause, resume, `--end` and the close's end use to find a scope (G-0118, addressed,
  covered `aiwf check`'s side).
- **Claims settled:** the event names in `docs.md` §4 Events and `history.md` §5 Events.

## Q16. Variables the sources depend on

- **Date:** 2026-11-21.
- **Answer:** the machine has no variables, and covers a scope opened from ADR-0056's release on.
  The epic that builds ADR-0056 adds a unit test with one scope written in today's format, made
  by hand (an opening commit with no `aiwf-scope-id`, an agent commit and an end naming it by
  SHA), asserting that the scope reads `ended` and that `aiwf check` reports no `provenance-`
  finding on the agent commit. A scope open across the switch to that release is outside the
  machine.
- **Reason:** what old scopes need from the build is that its readers still accept a SHA
  pointer (ADR-0056, Decision: "Readers accept both shapes"). The machine cannot see that, since
  every scope its adapter opens carries an id, and a scope written in today's format rebased or
  copied behaves in ways five states cannot express and ADR-0056 leaves broken ("A scope opened
  before this release stays breakable until it ends"). A unit test reaches the reader directly.
  Scopes in this repository's history are not the test's input: six were ever opened, all on
  epics and all ended, and only this project is in scope until aiwf is released to others.
- **Measured 2026-11-21**, the binary of question 2, in this repository: the commits
  `git log --grep='^aiwf-scope: opened'` lists open scopes on E-0014 (two), E-0029, E-0032,
  E-0034 and E-0056, and `aiwf show` reads each `state: ended`. Without the `^` it also lists
  03ceea8b3 and f8315c0e1, code commits whose messages quote the trailer.
  `aiwf check --since f0aaa3e30` (the root commit) reported
  `79 findings (57 errors, 22 warnings)`, every error `trailer-verb-unknown` and no
  `provenance-` or `isolation-escape` finding.
- **The rest of the variables both tables list:** the actor's kind and `--principal`
  (question 8; `bot/` is question 17; a scope delegated to a human is not an agent's); `--force`
  and the `--reason` it takes (question 11); the entity's kind (a milestone; question 17, item 4,
  for the kinds `authorize` refuses); `--branch`, the checkout and the rungs (branch rules,
  outside E-0100); `--scope` (the machine's `end` takes none, question 9); the number of scopes
  (one, questions 3 and 4); the milestone's status (`closed`, questions 10 and 14); whether the
  rebase reaches the opening commit (always, question 13; after a copy, question 13 says which
  commits keep their SHAs); squash and the host's merge methods (M-0366, question 15); a scope
  on the parent epic (the adapter opens none; one lets the agent act on the milestone, D-0007).
- **Sources this makes wrong:** none.
- **Claims settled:** C102, C107, C113 (the release boundary); the variables in `docs.md` §4
  Variables and values, and the values in `history.md` §5 States and values.

## Q17. Defects in today's code that the machine does not reach

- **Date:** 2026-11-21.
- **Answer:** five are filed as gaps, each with its reproduction:
  1. `aiwf check` has no rule for pause state, so an agent commit made by hand while its scope
     is paused passes (H59; run p, H74), where the verb refuses the same act (H12, H64). G-0777.
  2. `aiwf check`'s no-active-scope rule tests only the `ai/` prefix, so a hand-made `bot/`
     commit with no scope passes (H61; run bt, H84), where the verb refuses it (H11). G-0778.
  3. `isolation-escape` keys openers by exact `aiwf-entity`, so an agent commit naming
     `M-…/AC-N` on the wrong branch is never reported (H60; run ie, H85), where M-0106 says it
     is (H36). G-0779, which also records an agent commit on a milestone made under its
     epic's scope.
  4. D-0007 says `--force` overrides the refusal to authorize a gap, decision, contract or ADR
     (H26); the code refuses those kinds before reading `--force` (H25; run g). G-0780.
  5. `scope.LoadScope` and `scope.IsLegalScopeTransition` have no production caller (H57), and
     `LoadScope` refuses a history that the replay production uses skips silently (H5, H56).
     G-0781.
- **Not filed:** `aiwf import` applies no scope gating (H16). Import is deprecated (G-0667,
  open, which records the surfaces that still present it as current).
- **Sources this makes wrong:** M-0106, for item 3, whose claim G-0779 records. For item 4,
  D-0007 or the code: G-0780 leaves which one changes to whoever takes it up. Items 1, 2 and 5
  are defects in the code that no doc describes otherwise.
- **Reason:** each is a defect the audit found in code this machine does not reach (hand-made
  commits, other actors, branch rules, another entity kind, dead code), so conform will not
  report it and nothing else tracks it. CLAUDE.md's engineering principle makes `aiwf check`
  the authority, which is why items 1 and 2 are defects, not gaps in a convenience.
- **Claims settled:** H5, H11, H16, H18, H25, H26, H36, H56, H57, H59–H61, H74, H84, H85.

## Q18. Smaller conflicts among the docs

- **Date:** 2026-11-21.
- **Answer:** the docs are corrected to match the code.
  - The refusal codes `provenance-no-active-scope-to-pause` and
    `provenance-no-paused-scope-to-resume` come out of the docs that name them (the provenance
    model §"Verb behavior", C53, C62; R-FP-0135 and R-FP-0136, C54); the code's refusals carry
    no code (H6; `docs/audits/normative-docs-drift-audit.md`, 2026-08-06, C56;
    `docs/audits/entity-truth-audit.md`, 2026-08-19, C57). Filed as G-0783. G-0786, from
    question 9, lists the same sentences.
  - Filed as G-0784: the start-milestone ritual's §"Anti-patterns",
    `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/
    SKILL.md`: "The kernel's preflight refuses authorize-on-milestone-branch with
    `branch-context-required` at the verb layer; the `isolation-escape` kernel finding catches
    the same shape post-hoc at `aiwf check`" (C27), both halves false.
  - Filed as G-0785: which paused scope a resume picks. The verb picks the most recently opened
    paused scope (`mostRecentScopeInState`, `internal/verb/authorize.go`; measured 2026-11-21
    with two paused scopes); the provenance model §"Verb behavior" (C61), R-FP-0136,
    R-AUDIT-0219, R-RULE-072, the `aiwf-authorize` skill and the verb's help text say "the
    most-recently-paused scope", and R-AUDIT-0123 says "the most-recently-opened active scope"
    (C65). With one scope per agent per milestone the machine does not depend on it.
  - Not corrected: ADR-0047 §"Every edge into a terminal epic status is a sovereign act", "An
    epic is the unit a scope is opened on and whose closure ends delegation; a milestone is work
    inside a scope someone already holds" (C22), an accepted ADR, beside D-0007 and the
    start-milestone ritual, which open scopes on milestones (C10, C21); M-0103's AC-2 title,
    which names the `branch-not-found` that D-0018 retired (C24, C26), archived.
- **Sources this makes wrong:** the docs each bullet of the answer names, listed by G-0783,
  G-0784 and G-0785; those its last bullet names stay as they are.
- **Reason:** the machine has no outputs, so the codes are not its to decide, and the smallest
  change makes the docs true. Codes on those refusals would be a new feature, with a decision of
  its own.
- **Measured 2026-11-21**, for C27: every run of questions 2, 3, 10 and 15 ran `aiwf authorize
  M-0001 --to ai/claude --reason go` with `milestone/M-0001-first-milestone` checked out and no
  `--branch`, and each printed `authorize exit 0`, as M-0103's AC-3 title has it (C28).
- **Claims settled:** C10, C19, C21–C28, C53, C54, C56, C57, C61, C62, C65.

## Q19. Which agent verb the machine's act is

- **Date:** 2026-11-24.
- **Answer:** the machine's `act` is the agent's `aiwf retitle` of the milestone, the verb the
  adapter plays, and the machine leaves every other agent verb on the milestone and its ACs
  out. The milestone is `tdd: none`, and no step changes that (chapter §5).
- **Reason:** an `act` wider than the verb conform plays claims what no run checks, and one verb
  inside the wider reading breaks a rule: after the agent sets the milestone's TDD policy to
  `required`, which leaves no error, close is refused from active, paused and ended, where rule 13
  moves it to closed (measured below). Two other verbs stay out for reasons of their own. The docs
  allow the agent's own close by design ("The agent can close its own scope by design", C90; C89),
  and its closing commit ends the scope with `aiwf-scope-ends`, the side effect the human's close
  has (H13), so as an event of its own it would differ from `close` only in who runs it and in
  being refused where the agent has no active scope. A verb that leaves an error, such as
  `aiwf add ac` with no body (`acs-empty-body`), fails `check` for a reason that is not the
  scope's.
- **Measured 2026-11-24**, with the `aiwf` `conform.md` names, through the adapter's own steps
  from its starting repository: `authorize`, then the agent's
  `aiwf milestone tdd M-0001 --policy required --actor ai/claude --principal human/t`, then no
  step, `pause` or `end`, then `close`. Expected, were the policy change an `act`: close moves
  to closed (rule 13). Observed: the agent's verb exit 0; `aiwf check` exit 0 in active, paused
  and ended; close rejected in each, the state unchanged, and `aiwf promote M-0001/AC-1 met`
  refused with `acs-tdd-audit`: "status: met under tdd: required but tdd_phase is (absent)".
- **Measured 2026-11-22**, with the `aiwf` `conform.md` names, on the adapter's starting
  repository: `aiwf authorize M-0001 --to ai/claude --reason go`, then the agent's
  `aiwf promote M-0001/AC-1 met`, `AC-2 met` and `aiwf promote M-0001 done`, each with
  `--actor ai/claude --principal human/t`. Expected, by C90: the close allowed and the scope
  ended. Observed: exit 0, status `done`, the scope `ended`, the closing commit carrying
  `aiwf-actor: ai/claude`, `aiwf-authorized-by` and `aiwf-scope-ends`; `aiwf check --since main`
  0 errors. With the agent's `aiwf cancel` of both ACs and then of `M-0001` instead, expected by
  question 14 to end the scope as close does: exit 0, status `cancelled`, the scope `ended`,
  `aiwf-scope-ends` on the agent's commit, 0 errors.
- **Sources this makes wrong:** none.
- **Claims settled:** C89, C90 (for the machine).

## Q20. Other words for the states, and what the steps write

- **Date:** 2026-11-24.
- **Answer:** each of these names, as the sources use them, is another word for a state or an
  event the machine has, a case no step here reaches, or something a step writes; none adds a
  state, an event or a variable.
  - `opened`, `paused` and `resumed`, the values of the `aiwf-scope` trailer, mark the commits
    authorize, pause and resume write; the machine names those steps, not the marks.
  - `open` and `live` are other words for active, or for a scope that is not ended, and
    `non-ended` is active or paused.
  - `stranded`, a paused scope on a closed milestone, does not arise: a close ends a paused scope
    (question 5).
  - `ended-success`, `ended-failure` and `ended-discarded` end the sub-scopes of subagent cycles
    in archived design notes; an agent authorizing another is proposed and never built
    (question 15).
  - The code has no `none`: no scope is the absence of an opening commit, and the adapter
    observes it as an `in_progress` milestone that lists no scope.
  - A terminal-promote is the human's promote to `done` without `--audit-only`, the machine's
    `close`, or one the machine leaves out: the agent's own (question 19), one with
    `--audit-only` (question 15) and one to `cancelled` (question 14). The auto-end is the end
    of the scope that each writes as `aiwf-scope-ends` (question 5; H55).
  - The trailers, `aiwf-scope-ends` and those an act carries among them, and the finding and
    refusal codes are what a step writes or reports, and `aiwf history`'s chips render the
    trailers. The machine has no outputs (chapter §1; question 18 for the codes).
  - `bot/` actors, `human/` as the `--to` target, a scope on the parent epic, `--scope`, and
    `--branch`, the checkout and the rungs, the future-branch carve-out among them, are question
    16's variables; `ended` for both ends, several scopes per milestone and `done`/`cancelled`
    are question 14's.
- **Reason:** each name renames a state or an event the machine has, names a case no step here
  reaches, or is something a step writes, which the chapter leaves out (§7).
- **Sources this makes wrong:** none.
- **Claims settled:** the names in `docs.md` §4 States, `docs.md` §4 Outputs and `history.md` §5
  States and values.

## Cases no source covers, answered from the answers above

Put to the maintainer on 2026-11-21 with no objection.

- `none, rebase` stays `none`: with no scope there is nothing for a rebase to change
  (question 2's reason). No source covers it, so it makes none wrong.
- `check` passes in every state. An agent's verb commits only under an active scope (questions
  7 and 8), and `rebase`, `copy` and `reallocate` change nothing about the scope (questions 2, 3
  and 15). Measured 2026-11-21 with the binary of question 2: `aiwf check` exit 0 with no scope
  (fixture before any `authorize`) and on a closed milestone (question 10's run with the ended
  scope); `aiwf reallocate M-0001` on that `done` milestone exit 0, so `closed, reallocate`
  stays `closed`.
