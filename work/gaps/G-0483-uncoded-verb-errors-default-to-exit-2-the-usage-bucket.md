---
id: G-0483
title: Uncoded verb errors default to exit 2, the usage bucket
status: open
priority: medium
---
## What's missing

A verb's refusal loses structure on the way to a machine caller, on both axes the caller reads: the exit code it reports, and the payload it carries. Three instances, one seam.

**Uncoded errors default to the usage bucket.** `FinishVerbOutcome` (`internal/cli/cliutil/apply.go`) routes every verb error that is neither `Coded` nor `ErrInternal`-wrapped to `ExitUsage` — exit 2 — with an error envelope whose `code` field is empty. Nothing argues for that default; it is what the verbs happened to return before `FinishVerb` existed, carried through the migration unchanged.

Exit 2 is the usage bucket. `aiwf --help` states the contract plainly: `0` = no errors, `1` = errors found, `2` = usage error, `3` = internal error. A git subprocess that died, a tree that could not be written, a config that failed to parse mid-verb — each exits 2 today, so a caller reading the exit code alone concludes it typed the command wrong.

The three sibling `emitErrorEnvelope(label, "", …)` sites in the same file — `no result returned`, `validation passed but no plan produced`, and a `verb.Apply` failure — do exit 3, but carry no code either, so a consumer cannot tell them apart from each other.

**A structured refusal reaches the caller as prose.** The uncommitted-change guard builds a typed error carrying the blocking paths, split by whether git tracks them. The JSON envelope flattens it: `findings` is empty and `error.message` is the whole prose blob with the paths embedded in a sentence. A caller that wants to know which paths blocked it has to parse English, and an AI assistant is a first-class caller of this tool.

**One operator condition carries two exit codes.** A path with uncommitted changes exits 2 through the typed error; the same overlap with *staged* changes exits 3, because `checkStagedConflict` returns an untyped error that the shared handler maps to the internal-failure class. Nothing distinguishes the two conditions from the operator's side: both mean "your changes overlap with what this verb would write".

## Why it matters

`--format=json` is the documented machine-consumable contract, and the exit code is the part of it a shell caller reads first. Both currently conflate "you passed bad arguments" with "something inside the verb broke", which are the two outcomes a caller most needs to separate: the first is fixed by editing the command, the second by investigating the machine. Exit 3 means "aiwf broke" everywhere else, so a wrapper that retries on 3 and reports on 2 behaves differently for two spellings of one situation.

The prose-only paths are the more limiting half. The verb layer knows exactly which paths blocked it and which of them git tracks; a caller reconstructing that from a message will re-break every time the wording improves.

The blast radius is what makes the exit-code half a decision rather than a patch. Every mutating verb funnels through `FinishVerbOutcome`, so moving uncoded errors from exit 2 to exit 3 changes the observed exit code of every failure mode that has no code today — for every consumer, script, and CI lane. That is a wire-contract change, and it wants a recorded decision with a stated migration story, not a quiet edit.

G-0467 settled the adjacent half on the identity axis: the repo-lock refusals now carry `repo-lock-busy` / `repo-lock-acquire-failed` in the envelope's `code` field, which is why a caller can express "retry on contention, fail on everything else" without matching message text. It deliberately left the exit codes those refusals report alone, and left this question — what an *uncoded* error should exit with, and whether it should carry a generic code — open.

## Scope

The envelope's representation of a refusal, the exit code its two spellings report, and the default an uncoded error takes. Out of scope: the guards' verdicts themselves, which are correct.

## Resolution shape

On the exit-code axis, three candidate answers, not mutually exclusive:

- **A distinct exit code.** Route uncoded, non-`ErrInternal` errors to `ExitInternal` and reserve `ExitUsage` for errors raised before the verb body runs (flag parsing, argument validation). Cleanest semantically; the widest observable change.
- **A generic code string.** Keep the exit code and give the envelope a code meaning "no more specific identity available", so a JSON consumer can at least distinguish "uncoded" from "the code field was omitted". Cheap, but leaves the exit-code conflation in place.
- **Both**, with the exit-code move recorded as the decision and the generic code as its envelope-side companion.

On the payload axis, typing `checkStagedConflict`'s error the same way as its uncommitted twin surfaces both refusals through one shape — retiring the exit-code split and giving both structured paths in a single change. It changes an existing exit code, so it needs a changelog note. Structured paths without the exit-code fix is the cheaper half and leaves one condition with two codes; the exit-code fix alone leaves machine callers parsing prose.

Settling the default means auditing where usage errors are actually raised — the Cobra layer already exits 2 for flag and argument problems without reaching `FinishVerbOutcome` at all, which is evidence the verb-body default is the odd one out.

Whatever is chosen, the exit-code contract is a kernel surface: `aiwf --help`'s exit-code line, the CLI-conventions section of the kernel's `CLAUDE.md`, and the design docs all state it and all have to move together.

## Where to fix

- `internal/cli/cliutil/apply.go` — `FinishVerbOutcome`'s default arm, its three uncoded `emitErrorEnvelope` sites, and the flattening of the typed uncommitted-change error.
- `internal/cli/cliutil/exit.go` — the exit-code constants and their doc comments.
- `checkStagedConflict` — the untyped error whose exit code diverges from its uncommitted twin.
- `internal/cli/root.go` — `printHelp`'s exit-code line.
- `internal/policies/` — a policy pinning whichever contract is chosen, so the default cannot drift back.

## Related

- G-0494 — the prose-flattened refusal and the split exit code, absorbed here as one surface with one decision
- G-0467 — the identity axis, settled: repo-lock refusals carry codes
