---
id: G-0786
title: authorize --pause on a paused scope, or --resume on an active one, is refused
status: open
discovered_in: M-0364
---
## What's missing

`aiwf authorize <id> --pause` on a scope that is already paused, and `--resume` on one that is
already active, are refused at exit 2. The verb-design guidance (`.guidance/project/verb-design.md`,
§"Same-state convergence — resolve, then converge", R2) and CLAUDE.md commitment 7 ("a request
already satisfied is a NoOp at exit 0 with no commit") make each a NoOp; M-0364's audit
(`spec/scope/audit/decisions.md`, Q9) settled the repeat that way. `authorizeTransition`
(`internal/verb/authorize.go:475`) looks for a scope in the source state, active for a pause and
paused for a resume, and when there is none refuses with "no active scope on %s to pause" or "no
paused scope on %s to resume" (`authorize.go:294` and `:298`), without asking whether a scope is
already in the state the request names.

Measured 2026-11-22 in the Linux devcontainer (Go 1.25.12, git 2.51.1) with `aiwf` built from a
tree whose code is `main` at 0c00b37c4 (`aiwf version`:
`v0.42.1-0.20261121183124-0c00b37c47b3+dirty`; no Go file differs at `main` e369a83c7), in a
disposable repository: `aiwf init` with `hosts: []`, epic `E-0001` active, milestone `M-0001` in
progress with two ACs, and `milestone/M-0001-first-milestone` checked out. Ids in the block name
entities in that repository.

```
$ aiwf authorize M-0001 --to ai/claude --reason go
aiwf authorize M-0001 --to ai/claude                                # exit 0
$ aiwf authorize M-0001 --pause hold
aiwf authorize M-0001 --pause                                       # exit 0
$ aiwf authorize M-0001 --pause again
aiwf authorize: no active scope on M-0001 to pause                  # exit 2, HEAD unchanged
$ aiwf show M-0001
    94db43e  human/t → ai/claude  state: paused   opened 2026-11-22  events: 2
$ aiwf authorize M-0001 --resume go
aiwf authorize M-0001 --resume                                      # exit 0
$ aiwf authorize M-0001 --resume again
aiwf authorize: no paused scope on M-0001 to resume                 # exit 2, HEAD unchanged
$ aiwf show M-0001
    94db43e  human/t → ai/claude  state: active   opened 2026-11-22  events: 3
```

`TestAuthorize_Resume_RefusesWithoutPausedScope` (`internal/verb/authorize_test.go:1195`) asserts
the refusal for a resume on an active scope. The mechanical check for R2,
`PolicyVerbResultNoOpInvariant`, exempts all of `Authorize` (`noopExemptVerbs`,
`internal/policies/verb_result_noop_invariant.go:54`), for a reason that covers only a repeat
open, so it does not see this case.

The docs describe the refusal in words that also cover a repeat: the provenance model §"Verb
behavior" ("If none active, refuses", "If none paused, refuses";
`docs/design/provenance-model.md:243`–`244`), `docs/design/legal-workflows-first-principles.md`
R-FP-0135 and R-FP-0136, and the `aiwf-authorize` skill §"Common refusals"
(`internal/skills/embedded/aiwf-authorize/SKILL.md:61`–`62`).

## Why it matters

A caller that repeats a pause or a resume, as a retry after a lost terminal or a script making
sure, gets exit 2 while `aiwf show` lists the scope in the state it asked for. The refusal reads
the same as the one for a milestone with no scope at all, "no active scope on M-0001 to pause",
so the caller cannot tell "already so" from "nothing there", the distinction R2 keeps.

## Relates to

G-0783, which records the refusal codes named in the same sentences of the provenance model
§"Verb behavior" and R-FP-0135 and R-FP-0136; and G-0785, which records which paused scope a
resume picks, in `authorizeTransition` and the same sentence of the provenance model.
