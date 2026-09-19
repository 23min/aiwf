---
id: M-0343
title: Verify host handoff parallel worktrees and container persistence
status: in_progress
parent: E-0093
depends_on:
    - M-0342
tdd: advisory
acs:
    - id: AC-1
      title: Fresh sessions discover the supported skills and instruction sources
      status: met
    - id: AC-2
      title: Claude Codex and Claude can hand off one unfinished workflow
      status: met
    - id: AC-3
      title: Concurrent host sessions keep edits and Git state in their assigned worktrees
      status: open
    - id: AC-4
      title: An independent Codex review executes through the supported ritual path
      status: open
    - id: AC-5
      title: A rebuilt devcontainer provides Codex and retains the selected state
      status: open
    - id: AC-6
      title: Implementation passes repository gates and documents its support boundary
      status: open
---
## Goal

Establish recorded evidence that the supported Claude and Codex workflows are usable in fresh local sessions and survive the intended container lifecycle.

## Closes

- (none)

## Context

Public host detection and artifact lifecycle behavior are complete. This milestone tests actual host discovery and operator workflows, which filesystem tests alone cannot establish. Existing devcontainer edits are uncommitted and must be reviewed on their own merits before they are incorporated. The implementation worktree must receive approved planning commits before feature work starts.

## Acceptance criteria

The criteria below define the observable completion contract.

### AC-1 — Fresh sessions discover the supported skills and instruction sources

Record fresh Claude and Codex runs in a disposable consumer checkout after materialization. Verify each host's exposed skill inventory and instruction-source diagnostics where available, and execute representative planning and review steps using the generated artifacts. Repeat relevant root/nested instruction precedence and size-limit scenarios for Codex. Separately verify that Codex working on aiwf receives repository development guidance through a maintained source/reference arrangement. State what was mechanically observable and what was only demonstrated by behavior.

### AC-2 — Claude Codex and Claude can hand off one unfinished workflow

Run an approved handoff from Claude to Codex and back on the same task/worktree using current aiwf records, branch/diff state, and a concise handoff note. The receiving session identifies completed versus unfinished work and continues without changing host selection or reconstructing the workflow by translating another host's artifacts. Record how uncommitted work and outstanding approvals were represented. A host account limit that prevents a leg leaves this observation incomplete.

### AC-3 — Concurrent host sessions keep edits and Git state in their assigned worktrees

Use separate terminals and distinct branches/worktrees for a bounded pair of non-overlapping tasks. Record both repository roots and branches before work, then verify each change and staged/unstaged state in the intended checkout. Each worktree must have its complete selected host artifacts. Create these checkouts with `aiwf worktree add`. Start the fresh Codex session with `codex --disable worktrees -C "<path>"`, record the CLI version and explicit disabled-feature invocation, and verify observed skill discovery in that checkout. Codex-managed worktree creation is not part of this observation. Keep integration serial and separately approved; do not run concurrent refreshes using divergent binaries against shared hook state. This demonstrates human-operated parallel sessions, not an orchestration engine.

### AC-4 — An independent Codex review executes through the supported ritual path

Exercise a review-required ritual with a fresh reviewer context that did not author the change. Record the dispatch method, reviewed diff, findings, and how the parent handled them. Confirm that the instructions resolve to capabilities available in the supported Codex setup. If delegation is unavailable, demonstrate the actionable limitation and keep the successful independent-review observation outstanding; self-review cannot satisfy this criterion.

### AC-5 — A rebuilt devcontainer provides Codex and retains the selected state

After separate rebuild approval, exercise the reviewed install and mount setup in a fresh container build. Verify that the global npm Codex binary is available even if the editor supplies another binary, and that repeating initialization does not reinstall an existing npm binary. Check host-backed Codex configuration/session state and login status before and after without exposing credentials. Confirm Claude remains usable and that the install/mount documentation matches the exercised behavior. Record any steps not run.

### AC-6 — Implementation passes repository gates and documents its support boundary

Run the required build, formatting/lint, race-test, coverage, and selfcheck gates against the final implementation checkout and record results with failures preserved. Review setup, host override, guidance opt-out, symlink diagnostic, handoff, and parallel-worktree instructions against the demonstrated workflows. State the deferred capabilities and the distinction between materialization health and observed discovery. This criterion records actual gate execution; it must not be replaced with a proxy assertion that commands appear in documentation.

## Constraints

- Observations record command or prompt, expected outcome, actual outcome, host version, checkout, and relevant permission/configuration conditions. An unavailable run is outstanding, not a pass.
- Do not use a model's claim that it read a file as a mechanical delivery guarantee; use exposed discovery evidence and observed workflow behavior, with their limits stated.
- Live external host invocations, rebuilds, pushes, merges, and other outward or irreversible steps retain their individual approval gates.
- Separate worktrees isolate files and indexes but share repository hooks and may share a globally installed aiwf binary; use consistent source-built tooling.
- Advisory TDD applies to this observation/documentation milestone. Any new logic discovered here still follows the repository's normal TDD and coverage rules.
- No independent copy of the repo's engineering guidance, no automatic chat transfer, and no claim of full hook/custom-agent/cloud parity.

## Design notes

- E-0093 and docs/initiatives/agent-host-artifact-adapters.md define the approved scope and selected host contracts.
- D-0073 keeps planning on main and implementation in an isolated worktree.

## Surfaces touched

- repository instruction entry points and referenced guidance
- .devcontainer/init.sh, initialize.sh, devcontainer.json, README.md
- consumer setup and handoff documentation
- milestone validation records

## Out of scope

- Automated parallel TDD orchestration from E-0019.
- General guidance reduction from E-0092 or guaranteed context delivery from G-0523.
- Provider transcript migration, Codex-managed worktrees, and external deployment.

## Dependencies

- Enable host detection across setup refresh and diagnosis

## References

- E-0093
- D-0073
- D-0095
- G-0523
- G-0600
- E-0092
- E-0019

## Release note



## Decisions made during implementation

- (none)

## Validation

### AC-1 — fresh-session observations

Observed on 2026-09-19 in the Linux devcontainer: Codex CLI 0.155.0,
Claude Code 2.1.278, Node v22.23.2. Consumer setup used the worktree's
`bin/aiwf-diag`, built from source `89c9445cb`; its version label retains the
earlier milestone build context. No Go/build input changed before these runs.
Each external invocation received individual approval. Sessions used existing
accounts and configured models, with no model override. Claude's startup event
reported `claude-opus-5[1m]`. These are observations of these versions and
conditions, not a universal delivery or adherence guarantee.

#### Setup and invocation conditions

Disposable consumers were independent Git repositories on unborn `main`, with
no remote or commits. Setup used an isolated home and disabled global Git
configuration, a local synthetic Git identity, and a fixture-local `aiwf`
symlink to the implementation binary. `aiwf.yaml` selected
`hosts: [claude-code, codex]`. `aiwf init --no-prompt`, `aiwf doctor`, and
`aiwf doctor --check-rituals` completed successfully after setup corrections.
Each host had 19 verb skills and 20 ritual skills matching rendered definitions.
The isolated home produced a missing plugin-mount diagnostic; the optional
Claude worktree hook remained undecided. Neither is evidence of container
rebuild success.

Codex sessions used this command shape, with the checkout, prompt, and output
paths specific to each observation:

```sh
codex --disable worktrees -a never exec --sandbox read-only --json \
  -C "$checkout" -o "$result_file" - < "$prompt_file" > "$events_file"
```

Precedence and size runs additionally passed `-c project_doc_max_bytes=32768`
before `exec`; the size control used `65536`. User configuration was not edited.
Claude ran from the consumer root using:

```sh
claude --print --verbose --output-format stream-json \
  --permission-mode dontAsk --permission-prompts none \
  --tools Read,Glob,Grep,Skill --allowedTools Read,Glob,Grep,Skill \
  --strict-mcp-config --mcp-config '{"mcpServers":{}}' \
  --settings '{"disableAllHooks":true}' --no-session-persistence \
  < "$prompt_file" > "$events_file"
```

The Claude review added `Bash` to the tools and used the allowlist
`Read Glob Grep Skill 'Bash(node *)'`. Existing user settings remained loaded;
this is not a claim that the invocation replaced every permission source.
Prompts prohibited writes, commits, installation, delegation, and additional
provider calls. The approved sessions themselves used provider traffic.

#### Repository development guidance

Checkout: the implementation worktree on
`milestone/M-0343-live-host-validation`, HEAD `28f7e1230`, with the proposed
16-line root `AGENTS.md`. The prompt requested a read-only discovery report,
full explicit reading of its canonical development reference, validation and
approval rules, and identification of missing generated artifacts.

Expected: Codex follows the reference to `CLAUDE.md` without applying Claude's
imports or tool semantics. Observed: exit 0, `turn.completed`; tool output
contains every line of `AGENTS.md` and all 376 lines of `CLAUDE.md`, verified by
exact line comparison. The response correctly describes validation cadence,
commit gates, and ritual authoring sources. It reports the absent generated
Codex artifacts; no Claude guidance import is read. `CLAUDE.md` remains the
single development-rule source. Its SHA-256 is
`c35816631e79f0734976f16e4221e3d38e43f05da70818685ab320f71f70c6da`.
The observed full reference read does not depend on automatically importing a
Markdown link or fitting that 65,909-byte source into startup instructions.

#### Consumer planning discovery

Both fresh sessions received this task: report initially exposed instructions
and skills before tool use, distinguish that report from disk enumeration,
then use the generated planning skill to scope a beginner greeting CLI with
undecided empty-name behavior. Read the README and existing planning context,
propose scope, and ask the first scope question; stop before writing or
allocating an entity. Codex used `$aiwfx-plan-epic`; Claude used
`/aiwfx-plan-epic`. The README proposed documenting empty-name behavior and
stated that no implementation existed. Planning directories were empty.

Expected: each host discovers and uses its own generated skill. Observed:
both sessions exit 0 and perform the opening planning steps. Codex enumerates
all 39 project skills before tool use; exact generated guidance and planning
skill bytes appear in its tool output. Its startup inventory is model
narration, not an independent host diagnostic. Claude's `system/init` event
lists all 39 project skills in both `skills` and `slash_commands`; an actual
`Skill` invocation loads its Claude planning skill. Claude notices the README's
scope is narrower than the tool-planning request. It reads `AGENTS.md` during
inspection but explicitly identifies it as Codex guidance and does not use it
as its operational skill. This demonstrates host separation for this run.

Claude incorrectly states that `work/` is absent: it exists with empty
subdirectories. A file-only glob did not establish directory absence. There
are no planning entities, but that directory claim is rejected as evidence.
Neither session completed an epic plan or exercised mutating planning steps.

#### Nested precedence and size budget

Precedence fixture: preserve generated root guidance and add random markers
outside its managed block. Root `AGENTS.md` and `packages/AGENTS.md` each
provide a unique marker and conflicting precedence values.
`packages/example/AGENTS.override.md` supplies the expected winning value;
the same directory's `AGENTS.md` and a sibling directory's `AGENTS.md` each
provide a marker expected to be absent. Launch at `packages/example` with a
32 KiB budget; applicable instruction files total 14,261 bytes. The prompt
asks for the five marker fields from initial context only, with null for
unavailable values, and prohibits all tools. Expected random values are stored
outside the project and omitted from the prompt.

Observed: valid result JSON exactly matches all five expectations. Root and
intermediate markers are present; the leaf override wins; same-directory
default and sibling markers are null. The event stream contains only an
agent message, with no tools.

Size fixture: root `AGENTS.md` is 36,038 bytes, with an early random marker,
the preserved generated guidance, inert padding, and a late marker at byte
offset 36,017. A 52-byte nested `AGENTS.md` contains another marker. Launch
from that nested directory twice with identical files and prompt, separately
approved at 32 KiB and 64 KiB. Ask for the three exact markers from initial
context only, prohibiting all tools. Expected: only the early marker at
32 KiB, all three at 64 KiB. Both results exactly match, with no tool use.
The paired observation establishes size-sensitive delivery in this fixture;
it does not recommend changing the user's configured budget. Marker recovery
demonstrates supplied content; null responses remain behavioral absence
evidence rather than an authoritative inventory of excluded bytes.

#### Representative review execution

Both hosts reviewed the same new-file patch in a fresh consumer, using their
generated `wf-review-code` skill. Contract: trim a string before greeting it,
reject a trimmed-empty value with `Error("Name is required")`, and preserve
internal whitespace. Non-string input is excluded. The six-line function
instead checks `name.length === 0` before returning a greeting with
`name.trim()`. Four tests cover a plain name, surrounding spaces, literal
empty input, and an internal single space. The patch adds this function and
tests against an empty implementation; there is no fabricated commit range.

The brief requires reading the full patch, actual files and contract; running
tests and independent inline Node boundary probes; distinguishing branch
coverage from input-class coverage; and producing classified findings,
file:line locations, and regression proposals. Neither host receives the
parent's oracle result or the other host's review. Expected: request changes
for whitespace-only input despite a passing existing suite.

Observed: both sessions exit 0 and request changes. Codex's tool output
contains the complete generated skill; Claude invokes its native `Skill`
tool. Both verify the patch against actual files and independently reproduce
the defect at `greet.mjs:2` and missing rejection coverage at test line 13.
Codex's nine boundary probes yield three whitespace-only mismatches, exiting
1; a direct test-file invocation explicitly reports all four existing tests
passing. Claude additionally measures that an inline variant collapsing
internal space runs survives the existing tests, while a multiple-space
assertion catches it. Parent accepts the measured findings and preserves the
deliberately flawed fixture; these are not aiwf source defects. Any later
fixture correction should retain both literal-empty and trimmed-empty cases,
rather than replacing one input partition as the Claude reviewer suggested.
This representative execution does not satisfy AC-4's separate dispatch
workflow or the milestone's eventual independent wrap review.

#### Failures, preservation, and evidence locations

Preserved failures: an initial setup attempt supplied unsupported
`init --format=json` (exit 2); an isolated-home doctor initially lacked a Git
actor (exit 1), corrected with local synthetic identity. The repository probe
has an exit-2 enumeration of missing Codex artifact directories. Claude's
planning run has one rejected Glob `offset` parameter. Codex's first review
heredoc cannot create a temporary file in the read-only sandbox; `node -e`
then runs the probe. Claude's review has three denied commands followed by
simpler permitted probes. Two parent evidence-extraction attempts reject
string-valued event messages before the extractor handles both shapes.
These failures remain in the records; successful session exit does not erase
them. Expected exit-1 defect probes are not counted as passing tests.

Before/after SHA-256 censuses confirm that each session leaves its observed
non-Git files unchanged; consumer planning/review and paired size-control
Git status comparisons are unchanged. No source commit, entity mutation,
installation, or additional assistant dispatch occurs inside these sessions.
Raw commands, prompts, events, results, snapshots and per-run observations
are retained locally at the following paths. They are temporary evidence,
not committed dependencies; the fixture descriptions, command shapes,
expected outcomes and measured results above are the durable rerun record.

- `/tmp/aiwf-M-0343-repo-discovery-*` and `/tmp/aiwf-M-0343-probe-before.json`
- `/tmp/aiwf-M-0343-discovery-9j9_465u/`
- `/tmp/aiwf-M-0343-precedence-wenv7nci/`
- `/tmp/aiwf-M-0343-size-vfsmo537/`
- `/tmp/aiwf-M-0343-review-6p73y0r8/`

The observations establish the stated discovery and representative-use
results under the recorded conditions. They do not establish universal
comprehension, hook parity, transcript transfer, default permission behavior,
parallel editing, or rebuilt-container persistence. The remaining criteria
retain their own evidence obligations.

### AC-2 — Claude to Codex to Claude handoff

Observed on 2026-09-19 with the same host and Node versions as AC-1. Three
fresh CLI sessions received individual approval. The disposable consumer at
`/tmp/aiwf-M-0343-handoff-327e30na/consumer` selects both hosts explicitly.
Its fixture-local `bin/aiwf` symlink resolves to the implementation binary;
each session names that exact executable when reading planning state.

Local setup committed a working greeting module, seven passing tests, and a
three-line usage-guide header. The module trims string ends, preserves
internal whitespace, and rejects trimmed-empty input with
`Error("Name is required")`. The task is documentation only: complete
`docs/usage.md` with executable valid-name and rejection examples. aiwf
allocated fixture epic `E-0001`, milestone `M-0001`, and criterion
`M-0001/AC-1`; these identifiers belong to the disposable consumer, not this
repository. The AC body and README state the task and approval boundaries.

The first setup attempt to promote the milestone on `main` was refused at
exit 2: activation belongs on the epic branch. After separate approval of
the corrected sequence, `aiwf worktree add` created the epic worktree, the
milestone was activated there, and another `aiwf worktree add` created
`milestone/M-0001-host-handoff` from that epic branch. Its baseline HEAD is
`7379947ccf53053223a027adacdd97d4c8b564ab`. Both hosts have 39 current skills;
doctor and ritual checks pass. Full fixture check reports zero errors and
six warnings: four empty scaffold sections, no remaining draft milestone,
and no configured upstream for the provenance/body-section audit. Setup
did not bypass the rejected transition or fabricate planning state.

All three host sessions use the same milestone worktree under the consumer's
`.claude/worktrees/` directory. Invocation forms match AC-1: Claude uses
print/stream-json with `dontAsk`, no permission prompts, hooks disabled,
strict empty MCP configuration and no session persistence. Its bounded
allowlist includes direct read-only Git/aiwf commands and Node probes. Only
the first Claude leg exposes `Edit`; the final Claude leg has read/test
tools only. Codex uses `--disable worktrees -a never exec --json`, the exact
worktree through `-C`, and `--sandbox workspace-write`. No session is
authorized to stage, commit, promote, refresh artifacts, change hosts,
install, integrate, or invoke another assistant.

| Leg | Prompt and expected continuation | Observed result |
| --- | --- | --- |
| Claude author | Read Git/aiwf state; edit only the guide's valid-name examples; run examples/tests; leave rejection examples unfinished; emit an inline `aiwfx-handoff` note. | Exit 0. Three examples execute correctly and seven tests pass. Only the guide changes, unstaged; eight-line handoff names unfinished work, exact state pointers and pending approvals. |
| Codex receiver | Receive only that short note and current state; reconcile stale claims; complete rejection examples while preserving earlier text; verify and emit a return handoff through its own generated skill. | Exit 0. Corrects a stale clean-worktree claim against Git/aiwf, adds the two rejection examples, preserves valid-name text byte-for-byte, and emits a ten-line return note. Five documented outputs match; direct test-file invocation passes seven tests. |
| Fresh Claude verifier | Receive only the return note and current state; use its own generated review skill; review the full uncommitted guide and execute all examples/tests; report done-on-disk versus pending approvals without edits. | Exit 0, approve, no blocking or deferred findings. All five outputs match and seven tests pass. Identifies the fixture AC as open and staging, commit, completion, merge and push as unapproved. No file changes. |

The five executable examples call `greet` with `"Ada"`, `"  Ada  "`,
`"\tAda   Lovelace\n"`, `""`, and `" \t\n "`. They print respectively
`Hello, Ada!`, `Hello, Ada!`, `Hello, Ada   Lovelace!`, and twice
`Error: Name is required`. Examples use `node --input-type=module -e` from
the worktree root, catching the errors to print their name and message.
`node __tests__/greet.test.mjs` exposes all seven test results directly.
Parent reruns match the guide outputs and confirm preservation of the first
leg's valid-name text. Finite examples do not guarantee every future behavior.

Before/after hashes cover all 114 non-Git files, including ignored generated
artifacts. Only `docs/usage.md` changes in the authoring legs; no file changes
in the final leg. The index stays empty and HEAD stays at the baseline SHA.
Host selection, implementation, tests, planning records, and generated
artifacts remain unchanged. The final reviewer correctly notes that Git
alone cannot establish preservation of ignored artifacts; the parent's
snapshot comparison supplies that evidence.

The first handoff's clean-worktree statement was stale after its own edit,
although it correctly described the guide as unstaged. The parent preserves
that note exactly; Codex reconciles the contradiction from current records
before continuing. Neither receiver gets the preceding provider transcript
or a translation of the other host's artifacts. Native Claude `Skill`
invocations and Codex reads of `.agents/skills/` establish use of the
respective generated guidance. Parent capture of the short emitted notes
outside the repository is experiment evidence, not a maintained handoff file
or automatic chat transfer.

Failures retained: the first Claude leg's compound aiwf command is denied
and succeeds as separate exact-path commands. Codex's nested shell probe
fails with `spawnSync /bin/sh EPERM`, and `--test-isolation=none` is unsupported;
direct commands yield the required evidence. That compound shell's final
exit 0 does not erase its earlier failures. The final Claude leg has four
denied `git -C` commands, then succeeds with plain commands from the verified
worktree. These are observations under the deliberately bounded permission
configuration, not claims about default host permissions.

The human-operated round trip is complete, with actual uncommitted work and
pending approvals preserved. The disposable guide remains uncommitted and
its fixture AC remains open by design. Nothing is pushed or integrated.
This observation does not establish automated orchestration, parallel editing,
or AC-4's required Codex reviewer-dispatch path.

Raw setup commands/results, `allocated.json`, the three `*-prompt.txt` and
`*-command.sh` files, event streams, result reports, eight-/ten-line handoff
notes, snapshots and per-leg observations remain under
`/tmp/aiwf-M-0343-handoff-327e30na/`. They are temporary supporting evidence;
the setup, prompt contracts, command forms, expected outcomes, actual results
and limits above form the durable rerun record.

## Deferrals

- (none)

## Reviewer notes

- (none)
