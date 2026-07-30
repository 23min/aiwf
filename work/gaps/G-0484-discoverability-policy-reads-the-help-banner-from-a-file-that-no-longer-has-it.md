---
id: G-0484
title: Discoverability policy reads the help banner from a file that no longer has it
status: open
priority: medium
---
## What's missing

`readDiscoverabilityChannels` (`internal/policies/discoverability.go`) assembles the haystack `PolicyFindingCodesAreDiscoverable` searches: two named files plus two directory walks. One of the named files is `cmd/aiwf/main.go`, read as the source of "the binary's printHelp output".

`cmd/aiwf/main.go` is 21 lines and entry-only — G-0107 moved the dispatcher into `internal/cli`, and `printHelp` now lives at `internal/cli/root.go`. The file the policy reads contains no help text at all, and nothing has reported the mismatch because a haystack that is missing a channel produces no error of its own.

So the policy searches three channels while its doc comment, and the error it emits, both name four.

## Why it matters

The harm is a false positive, not a false negative: a smaller haystack reports *more* violations, never fewer. A new kernel code documented in `aiwf --help` — which CLAUDE.md names as the first of the sanctioned discoverability channels, and which is the right home for a code that spans every mutating verb rather than belonging to one — is reported as *"not mentioned in any AI-discoverable channel"*. The finding is wrong, and its remediation text points at the aiwf-check skill, which for a non-check-layer code is the wrong place to put it.

Nothing fires today, which is the reason to fix it now rather than after: every code the policy enumerates happens to be documented somewhere else as well, so the defect is latent. It surfaces the first time someone documents a code only where the banner is, and it surfaces as a demand to duplicate the documentation.

The second cost is the guarantee itself. A chokepoint that names four channels and checks three is not the chokepoint its readers think they have, and the gap between the two is invisible from the outside.

## Resolution shape

Point the haystack at the file that carries the banner. `internal/cli/root.go` is the direct fix and matches how the two singletons are already read.

Reading the *rendered* banner instead of the source file would be closer to the claim — the policy asserts something about `aiwf --help` output, not about a Go source file — and `internal/cli/integration` already has a `captureHelpBanner` helper that drives `cli.Execute([]string{"--help"})` in-process. That is the more honest source, at the cost of the policy executing the CLI rather than reading files. Either resolves the defect; the second also removes the assumption that a string literal in a source file is what an operator sees.

Whichever is chosen, the fix wants a firing fixture: a synthetic code documented *only* in the banner must pass, and the same code documented nowhere must fail. Without both arms the correction is unpinned in exactly the way the original was.

Worth settling alongside: the policy enumerates codes from `internal/check/` and `internal/contractcheck/` only, so a kernel code declared elsewhere — `cliutil`'s repo-lock codes are the current example — is outside its reach entirely. Widening the needle set is a separate decision from fixing the haystack, and pulls in the question of which non-check codes are meant to be in scope.

## Where to fix

- `internal/policies/discoverability.go` — `readDiscoverabilityChannels`'s singleton list, and the doc comment on `PolicyFindingCodesAreDiscoverable` that names the four channels.
- `internal/policies/discoverability_test.go` — the firing fixture for the banner channel.
