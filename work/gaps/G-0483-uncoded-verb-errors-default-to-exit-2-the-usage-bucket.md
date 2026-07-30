---
id: G-0483
title: Uncoded verb errors default to exit 2, the usage bucket
status: open
priority: medium
---
## What's missing

`FinishVerbOutcome` (`internal/cli/cliutil/apply.go`) routes every verb error that is neither `Coded` nor `ErrInternal`-wrapped to `ExitUsage` — exit 2 — with an error envelope whose `code` field is empty. Nothing argues for that default; it is what the verbs happened to return before `FinishVerb` existed, carried through the migration unchanged.

Exit 2 is the usage bucket. `aiwf --help` states the contract plainly: `0` = no errors, `1` = errors found, `2` = usage error, `3` = internal error. A git subprocess that died, a tree that could not be written, a config that failed to parse mid-verb — each exits 2 today, so a caller reading the exit code alone concludes it typed the command wrong.

The three sibling `emitErrorEnvelope(label, "", …)` sites in the same file — `no result returned`, `validation passed but no plan produced`, and a `verb.Apply` failure — do exit 3, but carry no code either, so a consumer cannot tell them apart from each other.

## Why it matters

`--format=json` is the documented machine-consumable contract, and the exit code is the part of it a shell caller reads first. Both halves currently conflate "you passed bad arguments" with "something inside the verb broke", which are the two outcomes a caller most needs to separate: the first is fixed by editing the command, the second by investigating the machine.

The blast radius is what makes this a decision rather than a patch. Every mutating verb funnels through `FinishVerbOutcome`, so moving uncoded errors from exit 2 to exit 3 changes the observed exit code of every failure mode that has no code today — for every consumer, script, and CI lane. That is a wire-contract change, and it wants a recorded decision with a stated migration story, not a quiet edit.

G-0467 settled the adjacent half on the identity axis: the repo-lock refusals now carry `repo-lock-busy` / `repo-lock-acquire-failed` in the envelope's `code` field, which is why a caller can express "retry on contention, fail on everything else" without matching message text. It deliberately left the exit codes those refusals report alone, and left this question — what an *uncoded* error should exit with, and whether it should carry a generic code — open.

## Resolution shape

Three candidate answers, not mutually exclusive:

- **A distinct exit code.** Route uncoded, non-`ErrInternal` errors to `ExitInternal` and reserve `ExitUsage` for errors raised before the verb body runs (flag parsing, argument validation). Cleanest semantically; the widest observable change.
- **A generic code string.** Keep the exit code and give the envelope a code meaning "no more specific identity available", so a JSON consumer can at least distinguish "uncoded" from "the code field was omitted". Cheap, but leaves the exit-code conflation in place.
- **Both**, with the exit-code move recorded as the decision and the generic code as its envelope-side companion.

Settling it means auditing where usage errors are actually raised — the Cobra layer already exits 2 for flag and argument problems without reaching `FinishVerbOutcome` at all, which is evidence the verb-body default is the odd one out.

Whatever is chosen, the exit-code contract is a kernel surface: `aiwf --help`'s exit-code line, the CLI-conventions section of the kernel's `CLAUDE.md`, and the design docs all state it and all have to move together.

## Where to fix

- `internal/cli/cliutil/apply.go` — `FinishVerbOutcome`'s default arm and its three uncoded `emitErrorEnvelope` sites.
- `internal/cli/cliutil/exit.go` — the exit-code constants and their doc comments.
- `internal/cli/root.go` — `printHelp`'s exit-code line.
- `internal/policies/` — a policy pinning whichever contract is chosen, so the default cannot drift back.
