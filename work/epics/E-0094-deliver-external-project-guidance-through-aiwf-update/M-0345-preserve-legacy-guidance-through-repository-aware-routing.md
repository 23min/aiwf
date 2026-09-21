---
id: M-0345
title: Preserve legacy guidance through repository-aware routing
status: in_progress
parent: E-0094
depends_on:
    - M-0344
tdd: required
acs:
    - id: AC-1
      title: Global entry points choose one engineering guidance source
      status: met
      tdd_phase: done
    - id: AC-2
      title: Synchronization preserves the installed project owner
      status: met
      tdd_phase: done
    - id: AC-3
      title: Compatibility distribution retains legacy content and personal settings
      status: met
      tdd_phase: done
    - id: AC-4
      title: Fresh sessions demonstrate legacy fallback and exclusive project reads
      status: met
      tdd_phase: done
---
## Goal

Make ai-dotfiles safe for a machine containing both migrated and unmigrated repositories before aiwf starts handing over guidance ownership.

## Closes

- (none)

## Context

The corpus and installed-index ownership shape are defined. ai-dotfiles currently concatenates all modules for Codex and writes per-project Claude language imports. Compatibility must be delivered through its existing bootstrap, without requiring downstream users to upgrade aiwf.

## Acceptance criteria

### AC-1 — Global entry points choose one engineering guidance source

Generated Claude and Codex instructions keep personal rules unconditional and route engineering reads to installed project guidance or legacy files. Include code-health in this boundary. Empty project selections and disabled maintenance retain project ownership; a missing project index retains legacy access. References: ai-dotfiles `build.sh`, source guidance modules, and generated instruction fixtures. Copilot behavior must not change incidentally when its current shared build input changes.

### AC-2 — Synchronization preserves the installed project owner

`dotfiles-sync` does not recreate legacy imports in an aiwf-owned repository and retains existing behavior elsewhere. Exercise managed index recognition, absent or unrelated indexes, nested working directories, foreign instructions, repeated runs, and shared-machine repositories with different owners. References: `bin/dotfiles-sync` and `test/dotfiles-sync.test.sh`. No automatic migration merely from a directory named `.guidance`.

### AC-3 — Compatibility distribution retains legacy content and personal settings

The bootstrap keeps legacy guidance readable at its existing locations while consuming the canonical corpus without a second independently maintained source. Existing personal preferences survive installation. Test the bootstrap in isolated homes and preserve other host outputs; document the compatible installation signal that aiwf's preflight can inspect. References: ai-dotfiles build/install paths, resolved at implementation start.

### AC-4 — Fresh sessions demonstrate legacy fallback and exclusive project reads

Record Claude and Codex reads in unmigrated, migrated, and explicitly empty-selection repositories on the same machine. Confirm personal instructions remain and the migrated case does not load the legacy engineering bundle. Test fixtures model the agreed installed-index shape. References: compatibility observation record in the milestone. File presence or assistant assertions alone do not establish the read path; report the actual observable evidence and limits.

## Constraints

Use a short routing instruction and an ownership check, not launcher wrappers, plugins, or a repository registry. This is assistant-directed loading, not a native conditional import. Old sessions require restarting. Legacy distribution remains until an explicit future compatibility decision authorizes removal.

## Design notes

E-0094 defines per-repository ownership and compatibility. The bootstrap must update its routing and required legacy content coherently; a failed bootstrap must retain a usable legacy setup.

## Surfaces touched

ai-dotfiles `build.sh`, `bin/dotfiles-sync`, its tests, guidance source modules, and bootstrap.

## Out of scope

aiwf project installation, removing legacy guidance support, and changing Copilot policy.

## Dependencies

- M-0344 — Publish the external engineering guidance corpus.

The external corpus and installed-index shape from the preceding delivery.

## References

- E-0094 — delivery scope and compatibility requirements.

---

## Release note

The ai-dotfiles compatibility update keeps personal instructions active while directing fresh Claude and Codex sessions to either legacy engineering guidance or an explicitly owned project guidance index. Legacy repositories retain their guidance; owned repositories, including empty selections, suppress legacy synchronization and hook injection. Installation preserves foreign instructions and settings, and a read-only checker exposes handover compatibility. Maintainers refresh committed legacy copies from the canonical guidance repository. Copilot retains its complete bundle; project installation and migration are delivered separately by M-0346.

## Decisions made during implementation

- D-0097 — An unreadable project index requires a warning and no legacy fallback; clearly labelled provisional general advice may continue.
- ai-dotfiles maintainers refresh and commit legacy copies from the canonical engineering-guidance repository. Consumers install those committed copies; downloading the corpus is not a bootstrap dependency. Personal collaboration rules remain maintained in ai-dotfiles.
- Foreign instruction files, symlinks, and personal guidance source locations remain active until their owner reconciles them. Installation refuses before rebuilding live outputs or rewiring those locations. Generated files explicitly owned by ai-dotfiles remain managed outputs.
- Compatibility is checked against Claude/Codex delivery visible in the invoking environment, irrespective of aiwf's selected artifact hosts. No ai-dotfiles installation is required when none is present. The signal covers the documented delivery contract, not other environments, arbitrary personal imports, Copilot, or already-running sessions. The executable contract and remediation are documented in the ai-dotfiles README.
- When an ai-dotfiles host entry point or synchronization launcher is detected, every nonempty effective global instruction file must declare compatible routing, including custom Codex overrides. The owner reconciles personal content and engineering routing; the checker does not rewrite it. Without detectable ai-dotfiles, unrelated personal instructions remain allowed, so an isolated unmarked historical copy is outside the recognition guarantee. This global-file rule does not apply to handwritten `.guidance/project.md` preferences.

## Validation

### Wrap checks

Observed on 2026-09-21 in the Linux devcontainer. In the ai-dotfiles milestone checkout at `735a129`, every `sh test/*.test.sh` suite passed, `sh build.sh` generated all host outputs successfully, `sh -n` accepted the changed POSIX shell scripts, and `git diff --check` reported no whitespace errors. The complete command output is retained locally in `/tmp/m0345-wrap-validation.log`. The repository defines no CI workflow or separate lint target; syntax validation and its shell suites are the available gates.

Scoped doc-lint covered the changed README and shared routing fragment: named scripts exist, documented synchronization/checker options resolve through their help output, and local link, heading, and TODO scans found no mechanical drift. The README directs engineering edits upstream and compatibility refreshes through `refresh-guidance.sh`; its personal-rule and routing edit instructions remain local to ai-dotfiles.

The aiwf planning tree reports no M-0345 findings and `aiwf check --since main` reports zero errors with the existing advisory TDD and archival warnings. `go build -o /tmp/aiwf-m0345-wrap ./cmd/aiwf` succeeded. `make check-fast` exited zero: all vet passes completed, golangci-lint reported `0 issues`, and the full Go test suite passed; output is retained locally in `/tmp/m0345-aiwf-check-fast.log`. The observational M-0345 AC-4 phase exception is recorded in commit `d0241412d`; it claims no red/green test cycle.

### Compatibility bootstrap and distribution

Observed on 2026-09-20 in the Linux devcontainer as an unprivileged user, using the ai-dotfiles milestone checkout. Installation tests use isolated homes and checkout fixtures; no production home installation was performed.

- Command: `sh test/bootstrap-preservation.test.sh`. Expected foreign instructions and source locations to remain active, conflicts to refuse before rebuilding live symlink targets, repeated installation to retain settings and legacy content, copy failure to preserve Claude instructions, read-only shared configuration to remain unchanged, and profile directories to be respected. Observed every assertion passing.
- Command: `sh test/refresh-guidance.test.sh`. Expected the complete canonical set to reach legacy paths with personal content retained, stable repeat refreshes, and unavailable, missing, empty, or linked upstream input to leave previous copies and source revision intact. Observed every assertion passing against a local Git source without network dependency.
- Command: `sh test/guidance-check.test.sh`. Expected absent or unrelated personal delivery to need no ai-dotfiles installation; old global routing, unavailable personal documents, stale home/PATH launchers, unresolved instruction files, and inaccessible ancestors to refuse compatibility. Expected effective profiles and nonempty Codex override precedence to select the inspected files. Observed every assertion passing, including empty linked overrides and launcher-only installations.
- Commands: each `test/*.test.sh` suite, `sh -n` over changed shell scripts, and `git diff --check`. Expected no regressions, syntax errors, or whitespace errors. Observed every shell suite passing and both checks clean. Doctor tests also confirm profile-directory selection and its explicit manifest override.
- Command: `./refresh-guidance.sh`. Expected a fresh default-branch download from the public canonical repository, with the imported revision recorded in `guidance-source.txt`. Observed a successful refresh without changes to the existing engineering copies. Independent comparison against the recorded canonical revision confirmed the declared legacy path mapping and rubric-link adaptation.
- Vacuity: isolated-checkout mutations removed foreign-file protection, moved rebuilding ahead of preflight, wrote directly over the live Claude manifest, bypassed routing and synchronization declarations, ignored the Codex profile, published refresh files before complete validation, and omitted the source revision. Expected each broken implementation to fail its relevant suite. Observed each mutation caught; production working files were not mutated.

Independent review findings about override selection, ancestor accessibility, shared-directory writes, PATH-only synchronization, and personal-document type were reproduced and pinned in the tests above. The final full-surface review approved the implementation; its follow-up confirmed the final branch-audit test additions.


### Conservative compatibility recognition

Observed on 2026-09-21 in the Linux devcontainer using the ai-dotfiles milestone checkout. A frozen fixture reproduces the exact unmarked Codex bundle generated from ai-dotfiles revision `2552653`. Before the correction, `sh test/guidance-check.test.sh` failed with `FAIL: detected installation accepted an unmarked legacy override`. After the correction, the suite passed: detected installations reject unrecognized effective global files, including custom overrides and launcher-only detection, while empty files and custom instructions without detectable ai-dotfiles remain allowed. Every `test/*.test.sh` suite, changed-script `sh -n`, and `git diff --check` passed; command output is retained locally in `/tmp/m0345-conservative-validation.log`.

Independent correction review approved the change after verifying the historical fixture byte-for-byte, testing paths containing spaces, checking that invocations preserve fixture bytes, and exercising custom Claude files, Codex overrides, and launcher-only detection. This is a scoped correction verdict, not milestone closure. The design review recommends keeping the checker structure; its isolated compression trial preserved the tests but provided no structural simplification worth replacing the implementation.

### Fresh-session guidance routing

Observed on 2026-09-21 in the Linux devcontainer with Claude CLI 2.1.278 (`claude-opus-5[1m]`) and Codex CLI 0.155.0 (`gpt-6-astra`). The isolated ai-dotfiles installation uses commit `735a129`; project packs come from engineering-guidance revision `9c9ca4b3681ca4cb8798e5af9e9124b1e761526b`. No production home installation was changed.

Fixtures are independent Git repositories with a Go module and a `config.go` whose read-error branch replaces the original error with `errors.New`. The legacy fixture has no project index and retains generated Claude imports. The migrated fixture has the agreed ownership marker in `.guidance/index.md`, selecting code-health and Go/Cobra, with the rubric beside its guide. The empty fixture has the same ownership marker and explicitly selects no packs. Owned fixtures include `.guidance/project.md` stating “Use the standard library.” They model the agreed installation shape; they do not exercise aiwf's forthcoming materializer.

Each fresh session received: “Review config.go for Go error handling and maintainability. Do not edit files, run builds/tests, contact external services, or commit. Report only actionable findings with file and line references.” Expected personal rules in every session, legacy guidance in the unowned repository, project guidance in the migrated repository, and no legacy fallback for the empty selection.

Commands: `python3 /tmp/run-m0345-ac4.py claude legacy`, `claude migrated`, `claude empty`, `codex legacy retry`, `codex migrated`, and `codex empty` (the latter arguments use the same runner). Each invocation was individually approved. The runner supplied the prompt on stdin, selected the fixture working directory and isolated profile environment, and imposed `timeout 180`. Claude used `--print --verbose --output-format stream-json --permission-mode dontAsk --tools Read,Glob,Grep,Bash --allowedTools 'Read,Glob,Grep,Bash(git rev-parse *),Bash(dotfiles-sync --list*)' --strict-mcp-config --mcp-config '{"mcpServers":{}}' --setting-sources user,project --max-budget-usd 2`. Codex used `--ask-for-approval never exec --sandbox read-only --ignore-user-config --json -C <fixture> -`.

| Host | Fixture | Observed instruction and tool delivery |
|---|---|---|
| Claude | Legacy | Personal content in rendered instruction attachment; successful reads of legacy Go guide, code-health primer, and global rubric. |
| Claude | Migrated | Personal content retained; successful reads of owned index, project preferences, selected Go/code-health guides, and project rubric. |
| Claude | Empty | Personal content retained; successful reads of empty index and project preferences, without legacy guide or rubric tool reads. |
| Codex | Legacy retry | Personal content in native `world_state.agents_md`; successful command outputs contain legacy Go guide, code-health primer, and global rubric. |
| Codex | Migrated | Personal content retained; successful command outputs contain owned index, project preferences, selected Go/code-health guides, and project rubric. |
| Codex | Empty | Personal content retained; successful command output contains empty index and project preferences, without legacy guide or rubric tool reads. |

Every listed session exited zero and reported a completed/successful turn. Content comparisons against source documents establish personal instruction delivery and Codex guide outputs; Claude Read results and rendered instruction attachments establish its delivery. The owned-fixture traces contain no positive reads of legacy engineering guide modules. Both runtimes read the global code-health skill during discovery; no owned-fixture tool call reads that global rubric, and its full body is absent from recorded initial instructions. These observations establish the recorded loading paths, not an absence of runtime file access or a guarantee that every future assistant follows the routing instruction.

The initial `codex legacy` attempt also exited zero but both shell reads failed with `bwrap: execvp codex-linux-sandbox: No such file or directory`; it is not a passing observation. Codex refused helper creation under the temporary profile. Moving only the isolated Codex profile to `/home/vscode/.cache/m0345-ac4-codex` retained read-only sandboxing; a local `codex sandbox -c 'sandbox_mode="read-only"' /bin/cat <fixture>/config.go` returned the complete fixture source with exit zero before the approved retry. The Claude legacy session had a denied directory listing and recovered using permitted file tools.

Local evidence is under `/tmp/m0345-ac4-e2u9x6qc/observations/`: each session directory contains `command.json`, `events.jsonl`, `stderr.log`, `files.trace`, `exit-code.txt`, `summary.json`, and `evidence.md`. Native Claude records are under the fixture home's `.claude/projects`; Codex records use the isolated profile's `sessions` directory, with the failed attempt retained under the original fixture profile. The preparation and launch scripts are `/tmp/prepare-m0345-ac4.py` and `/tmp/run-m0345-ac4.py`. These are transient local artifacts; this milestone records the observation and its limits durably. Trace buffers were suppressed; the initial Claude legacy trace used raw read arguments, so its personal-content evidence comes from the native rendered attachment instead of syscall-path attribution.


### Unreadable project index

Observed on 2026-09-21 in the same Linux fixture environment with Claude CLI 2.1.278, model `claude-opus-5[1m]`, and ai-dotfiles routing from `735a129`. Command: `python3 /tmp/run-m0345-ac4.py claude unreadable`, with the same prompt, timeout, and budget as the other Claude observations. A regular `.guidance/index.md` with mode `000` raised `PermissionError` in a local read probe as UID 1000 before launch. Expected the assistant to report the unreadable index, retain personal instructions, and avoid legacy fallback; provisional general advice must be labelled as such and warn about unknown project rules.

Observed exit zero and a successful CLI result, with reported usage cost $0.153289. The index Read returned `EACCES`. Claude reported that it could not establish the project's guidance ownership or rubric, labelled its review as based on standard Go idiom and the explicit request, and warned that findings might need reassessment against project rules. It then gave provisional error-handling and style advice and requested restored permissions or operator direction. Native rendered instructions retained the personal source content. No legacy engineering module reads or global rubric tool reads were observed; runtime skill discovery still read the global skill file. This distinguishes an unreadable index from the missing-index legacy baseline without claiming that the new routing wording has been exercised in a fresh session.

Evidence is retained under `/tmp/m0345-ac4-e2u9x6qc/observations/claude-unreadable/`, including the raw events, file trace, final response, and personal-content comparison. The shared routing now explicitly permits the warned provisional advice described by D-0097; build and ownership tests verify generated delivery and synchronization behavior, not future model compliance. The Codex observation below exercises the clarified wording.


Observed on 2026-09-21 in the same Linux fixture environment with Codex CLI 0.155.0, model `gpt-6-astra`. Command: `python3 /tmp/run-m0345-ac4.py codex unreadable`, retaining the read-only sandbox, three-minute timeout, and original review prompt. The isolated profile `/home/vscode/.cache/m0345-ac4-codex-clarified` contains generated instructions from ai-dotfiles `5a902aa`; the fixture home's legacy modules and global skill remain from `735a129`. Expected a warning, retained personal instructions, no legacy fallback, and explicitly provisional general advice under D-0097.

Observed CLI exit zero and a completed turn. The index-read command exited one with `Permission denied`. Codex inspected the Go source and module, returned an error-handling finding, and warned in the same final response that project guidance could not be loaded and its provisional advice might conflict with project rules. Native `world_state.agents_md` contains the complete clarified manifest and personal source content. No command reads legacy engineering modules or the global rubric, and no positive legacy engineering module reads appear in the file trace. Runtime skill discovery read the global skill. Token usage, raw command results, manifest hash, native record location, and the final response are retained under `/tmp/m0345-ac4-e2u9x6qc/observations/codex-unreadable/`. This is evidence of observed behavior, not a guarantee of future compliance.

## Deferrals

- (none)

## Reviewer notes

- Independent full-surface review approved local wrap at ai-dotfiles `5a902aa`: shell suites, build, syntax and whitespace checks passed; recorded guidance reads and personal instructions match the native session evidence. No unresolved findings or deferrals remain. This verdict does not authorize publication or production installation.
- D-0097 unreadable-index observations are recorded for both hosts. Claude exercised the earlier routing and warned before provisional advice; Codex exercised the clarified routing and included its warning with the finding. Neither observation establishes future model compliance.
