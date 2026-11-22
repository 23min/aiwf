# Answers to the scope readings' questions

Decided by human/peter on 2026-11-22, in conversation, from `spine diff` of reading-a against
reading-b and `spine check` of each. The chapter carries each answer at the rule it settles.

- **Q1. Is closed final?** No. Nothing leaves closed, but every step there is decided as the
  chapter's §2 says, so `spine conform` checks what aiwf does after a close: the refused
  authorize (audit Q11), and the refused pause and resume and the close that is a no-op (audit
  Q9).
  Reading A made closed final; reading B did not. (Chapter S3.)
- **Q2. What does copy do where no scope was opened?** In none, and in closed when no scope was
  opened, copy is allowed and leaves the state as it is: its git steps run as anywhere else,
  with no opening commit among the commits they copy. No aiwf verb runs, so nothing refuses, and
  copy asks for nothing that already holds, so it is not a no-op. Both readings left none/copy
  undecided, and reading B closed/copy too. (Chapter rule 17, and the glossary's rebase and
  copy, which audit Q13 defines by their git steps.)
- **Q3. Which kind is each state?** none, active and paused are policy; ended and closed are
  fact. Each of the two records an act aiwf never reverses: "The inverse of an end is a fresh
  grant" (ADR-0047, §"Nothing undoes an end"), and a `done` milestone has no transition out
  (`internal/entity/transition.go`), so changing either needs a new decision. The readings
  differed only on ended, which reading A made policy; `spine diff` does not compare kinds.
  (Chapter §1, States.)
