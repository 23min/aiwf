# Answers to the scope readings' questions

Decided by human/peter on 2026-11-22, in conversation, from `spine diff` of reading-a against
reading-b, `spine check` of each, and the kind each reading gives each state, which neither tool
compares. The chapter carries each answer at the rule it settles.

- **Q1. Is closed final?** No. Nothing leaves closed, but every step there is decided as the
  chapter's §2 says, so `spine conform` checks every event after a close, among them
  the refused authorize (audit Q11), the refused pause and resume, and a close that is allowed
  and leaves closed as it is (audit Q9).
  Reading A made closed final and reading B did not, so they split on closed/authorize,
  closed/act, closed/pause, closed/resume, closed/end, closed/close, closed/rebase,
  closed/reallocate and closed/check. (Chapter S3.)
- **Q2. What does copy do where no scope was opened?** In none, and in closed when no scope was
  opened, copy is allowed and leaves the state as it is: its git steps run as anywhere else,
  with no opening commit among the commits they copy. Those steps set off the repository's
  hooks, which run aiwf, and none refuses; copy asks for nothing that already holds, so it is not
  a no-op. Both readings left none/copy
  undecided, and reading B closed/copy too. (Chapter rule 17, and the glossary's rebase and
  copy, which audit Q13 defines by their git steps.)
- **Q3. Which kind is each state?** none, active and paused are policy; ended and closed are
  fact. Each of the two records an act no ordinary verb reverses: "The inverse of an end is a
  fresh grant" (ADR-0047, §"Nothing undoes an end"), and a `done` milestone has no transition
  out (`internal/entity/transition.go`); no aiwf verb but the human override
  `aiwf promote --force` moves it ("force=true relaxes the FSM transition rule so any-to-any
  moves are permitted", `internal/verb/promote.go`) without `aiwf check` reporting it. Measured
  2026-11-22 with the `aiwf` `conform.md` names, on the adapter's starting repository closed:
  `aiwf import` of a manifest giving M-0001 `status: in_progress`, with `--on-collision update`,
  exited 0 and moved it, and `aiwf check --since main` reported
  `fsm-history-consistent/illegal-transition`. Changing either rule needs a new decision. The
  readings differed only on ended, which reading A made policy; `spine diff` does not compare
  kinds. (Chapter §1, States.)
