---
id: G-0494
title: Verb refusals reach machine callers as prose, and exit differently by form
status: open
discovered_in: M-0283
---
## What's missing

Two seams turn a structured refusal back into an unstructured one.

The uncommitted-change guard builds a typed error carrying the blocking paths,
split by whether git tracks them. The JSON envelope flattens it: `findings` is
empty and `error.message` is the whole prose blob with the paths embedded in a
sentence. A caller that wants to know which paths blocked it has to parse
English, and an AI assistant is a first-class caller of this tool.

And the same operator condition carries two exit codes. A path with uncommitted
changes exits 2 through the typed error; the same overlap with *staged* changes
exits 3, because `checkStagedConflict` returns an untyped error that the shared
handler maps to the internal-failure class. Nothing distinguishes the two
conditions from the operator's side: both mean 'your changes overlap with what
this verb would write'.

## Why it matters

Exit codes are the contract a script branches on, and 3 means 'aiwf broke'
everywhere else. A wrapper that retries on 3 and reports on 2 behaves
differently for two spellings of one situation.

The prose-only paths are the more limiting half. The verb layer knows exactly
which paths blocked it and which of them git tracks; a caller reconstructing
that from a message will re-break every time the wording improves.

## Scope

The envelope's representation of a refusal, and the exit code the staged twin
reports. Out of scope: the guards' verdicts, which are correct.

## Resolution options

1. Type `checkStagedConflict`'s error the same way and surface both through
   one shape. Retires the exit-code split and gives both refusals structured
   paths. Changes an existing exit code, so it needs a changelog note.
2. Structured paths only, leaving the exit codes as they are. Cheaper; leaves
   one condition with two codes.
3. Exit code only. Cheapest, and leaves machine callers parsing prose.