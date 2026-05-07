---
id: M-070
title: aiwf doctor warning for missing recommended plugins
status: done
parent: E-18
tdd: required
acs:
    - id: AC-1
      title: doctor.recommended_plugins config field accepts list
      status: met
      tdd_phase: done
    - id: AC-2
      title: doctor reads installed_plugins.json and matches project scope
      status: met
      tdd_phase: done
    - id: AC-3
      title: Each missing plugin emits one warning with install command
      status: met
      tdd_phase: done
    - id: AC-4
      title: Empty config list means no checks fire (kernel-neutral)
      status: met
      tdd_phase: done
    - id: AC-5
      title: Plugins installed for this project's scope produce no finding
      status: met
      tdd_phase: done
    - id: AC-6
      title: Plugins installed only for other scopes produce a finding
      status: met
      tdd_phase: done
    - id: AC-7
      title: doctor --self-check covers the new check end-to-end
      status: met
      tdd_phase: done
---

## Goal

Add a kernel detection mechanism for missing recommended plugins. After this milestone, `aiwf doctor` reads a new `aiwf.yaml: doctor.recommended_plugins` list and warns once per declared plugin not installed for the consumer's project scope. The kernel stays neutral on *which* plugins to recommend (empty default; consumer-declared); it just provides the detection.

This is the prerequisite for [M-071](M-071-install-ritual-plugins-in-kernel-repo-document-operator-setup-path.md): once the warning exists, M-071's fix (install the plugins, declare them in this repo's `aiwf.yaml`) can be validated by watching the warning go silent. Closes [G-062](../../gaps/G-062-aiwf-doctor-does-not-surface-missing-recommended-plugins-ritual-skills-aiwf-extensions-wf-rituals-can-be-silently-absent-from-a-consumer-repo-with-no-signal-to-operator-or-ai-assistant.md).

## Approach

Add a `doctor.recommended_plugins` field to `aiwf.yaml`'s schema (list of strings, each shaped `<name>@<marketplace>` — same format the user types into `claude /plugin install`). New doctor check `recommended-plugin-not-installed`:

1. Load the consumer repo root (already known to doctor via `--root` resolution).
2. Read `~/.claude/plugins/installed_plugins.json` (a JSON file Claude Code maintains; structure: top-level `plugins` map, each entry has scope arrays with `projectPath` and `installPath`).
3. For each entry in `doctor.recommended_plugins`, search `installed_plugins.json` for a `scope: "project"` entry whose `projectPath` matches the consumer root. Mismatch → emit one warning.

Severity: warning. Plugins are advisory; refusing on absence is too strong.

The check is read-only and fast. No file mutations, no network. Failure to read `installed_plugins.json` (file missing — Claude Code never run on this machine, or running outside Claude Code entirely) is treated as "no plugins installed" — every recommended plugin warns. That's the right behavior: if the operator can't read the file, they probably aren't running under Claude Code and the warnings are noise they can ignore by leaving the config list empty.

## Acceptance criteria

### AC-1 — doctor.recommended_plugins config field accepts list

`aiwf.yaml` accepts a `doctor.recommended_plugins` field whose value is a list of strings, each shaped `<name>@<marketplace>` (e.g., `aiwf-extensions@ai-workflow-rituals`). The field is optional; absence is equivalent to an empty list. Loaded via the existing config layer in `internal/config/`. Schema fixture in `testdata/` covers: empty list, single entry, multiple entries, missing field. Invalid shape (entry missing `@`, non-string, non-list) refuses load with a clear error pointing at the field path.

### AC-2 — doctor reads installed_plugins.json and matches project scope

`aiwf doctor` resolves `~/.claude/plugins/installed_plugins.json` (using `os.UserHomeDir()` + the documented relative path), loads the JSON, and matches each `doctor.recommended_plugins` entry against installed plugins by `(name@marketplace, projectPath==consumer-root, scope=="project")`. The match logic is exact-string for name and marketplace, exact-path for `projectPath` (after both are absolute-resolved via `filepath.Abs`). Test fixtures cover: file missing, file present with no matches, file with matches under a different `projectPath`, file with matches under the right `projectPath`.

### AC-3 — Each missing plugin emits one warning with install command

For each entry in `doctor.recommended_plugins` that doesn't match an installed plugin under the consumer's scope, doctor emits exactly one warning. The warning text includes the plugin id, the marketplace, and a one-line install hint formatted as `claude /plugin install <name>@<marketplace>`. JSON envelope output carries the same structured info (`finding.code: "recommended-plugin-not-installed"`, `finding.data: {plugin, marketplace, install_command}`). One warning per missing plugin — never deduped, never skipped. Tested with 0 / 1 / N missing plugins.

### AC-4 — Empty config list means no checks fire (kernel-neutral)

When `doctor.recommended_plugins` is absent or an empty list, the new check makes zero observations: it doesn't read `installed_plugins.json`, doesn't emit findings, and doesn't change doctor's exit code. The kernel makes no assumption about which plugins a consumer "should" have; the recommendation is consumer-declared. Tested with a fixture where the field is missing entirely and another where it's `[]`; both produce zero findings from this check.

### AC-5 — Plugins installed for this project's scope produce no finding

When all entries in `doctor.recommended_plugins` are matched by an installed plugin whose scope contains the consumer repo root, the check emits zero findings. Verified with a fixture `installed_plugins.json` whose structure mirrors the real Claude Code file (top-level `plugins` map with per-name arrays containing scope objects), where every recommended plugin has a `scope: "project"` entry whose `projectPath` resolves to the consumer root. Doctor exits 0 and produces no recommended-plugin warnings.

### AC-6 — Plugins installed only for other scopes produce a finding

The session-canonical case: a recommended plugin is installed in `installed_plugins.json` but only under a `projectPath` that does *not* match the consumer root (e.g., installed for another project). The check still fires the warning — installation under another scope is not equivalent to availability for this consumer. Tested with a fixture mirroring this session's actual state: `aiwf-extensions` installed for `/Users/x/Projects/proliminal.net`, recommended in this repo's `aiwf.yaml`. Warning fires; installation elsewhere does not silence it.

### AC-7 — doctor --self-check covers the new check end-to-end

`aiwf doctor --self-check` exercises the new check in its synthetic temp repo per the existing self-check pattern. The self-check fixture declares one entry in `doctor.recommended_plugins` (a synthetic name to avoid coupling to real marketplace state) and constructs a minimal `installed_plugins.json` in a fake `$HOME` for the subprocess; both the warning case (no matching install) and the silent case (matching install) are exercised. Per the CLAUDE.md "test the seam" rule, the integration is at the binary level: build the verb, drive it as a subprocess, assert stdout/stderr/exit code.

## Work log

The kernel records the per-AC red→green→done→met timeline; `aiwf history M-070/AC-<N>` is the authoritative chronology. This section captures the per-AC outcome only.

### AC-1 — config field accepts list

`Doctor` struct + `RecommendedPlugins []string` on `Config`. Shape validation runs at `Load` time via a one-pass `preCheckTypedShape` helper that catches non-list values with a field-path error before `yaml.Unmarshal` reduces it to a generic type error; `Validate()` then per-entry-checks each string against `<name>@<marketplace>`. Tests cover absence, explicit `[]`, single + multi-entry round-trips, and four malformed shapes (no `@`, empty marketplace, empty name, non-list value). Two extra falls-through tests close the `doctor:`-block-without-`recommended_plugins:` and `recommended_plugins: null` branches. Coverage: 100%.

### AC-2 — installed_plugins.json reader + matcher

New package `internal/pluginstate/`. `Load(home)` reads the canonical Claude Code index; treats `fs.ErrNotExist` as an empty index per the spec's "Claude Code never run on this machine" case; surfaces other read / parse errors with the path. `Index.HasProjectScope(plugin, projectRoot)` matches by `(name@marketplace, scope=="project", filepath.Abs equality of projectPath)` — user-scope installs are deliberately not a match. Tests cover the missing / present / corrupted file paths plus all match/no-match permutations. Coverage: 89.7%; the residual is `filepath.Abs` failure paths, which are essentially unreachable on Unix.

### AC-3 — warning emission with finding code + install command

New helper `appendRecommendedPluginsReport` in `cmd/aiwf/admin_cmd.go`. Loads `pluginstate` once per doctor run and emits two output lines per missing plugin: a `recommended-plugin-not-installed: <id>` finding line and an indented `install: claude /plugin install <id>` continuation. Soft warning — does not increment doctor's `problems` counter (per spec: "Plugins are advisory; refusing on absence is too strong"). The pre-M-070 hardcoded `aiwf-extensions` block in `doctorReport` was deleted (subsumed by the generalized check); the same `ritualsPluginInstalled` / `printRitualsSuggestion` functions stay because `aiwf init` still calls them as a one-shot setup nudge — see *Reviewer notes*. Tests cover 1 / N warnings + soft-counter invariant + corrupted-index advisory. Coverage on helper: 93.3% (residual is `os.UserHomeDir` failure path).

### AC-4 — empty/absent config is kernel-neutral

Same code path as AC-3, validated by two test fixtures: `doctor:` block absent entirely + `doctor.recommended_plugins: []`. Helper short-circuits before any filesystem read; output contains no `recommended-plugin-not-installed` lines.

### AC-5 — matching project-scope install silences the warning

Test fixture writes a synthetic `installed_plugins.json` whose only entry has `scope: "project"` and `projectPath` equal to the test's consumer root. Doctor's output contains zero `recommended-plugin-not-installed` lines. Mirrors the real Claude Code JSON shape so the same parse + match path runs.

### AC-6 — install for a different project still warns

Test fixture writes a project-scope entry whose `projectPath` is `/Users/x/Projects/some-other-repo` plus a user-scope entry. Warning still fires once for the consumer root, confirming neither cross-project nor user-scope installs short-circuit the per-project check.

### AC-7 — `aiwf doctor --self-check` end-to-end

Two new steps appended to `runSelfCheck`: (a) declare a synthetic recommended plugin in the temp-repo `aiwf.yaml` and assert the warning fires with finding code + install command; (b) write a matching `installed_plugins.json` under a fake `$HOME` and assert the warning is silent. The fake `$HOME` is set once at the top of `runSelfCheck` (mirroring the existing `GOPROXY=off` override) with a seeded `~/.gitconfig` so identity resolution still produces a valid actor — every mutating verb in the step list depends on it. The binary-level seam test `TestRun_DoctorSelfCheck_Passes` was updated to expect the two new step labels.

## Decisions made during implementation

No new ADR / D-NNN was needed during implementation; every decision was either covered by the milestone spec's *Approach* or surfaced and resolved in conversation before TDD started:

- **JSON envelope output for AC-3 deferred.** `aiwf doctor` has no `--format=json` today; introducing an envelope is a doctor-wide change touching every existing report section. Treat AC-3's JSON line as forward-looking; the structured-info contract surfaces only when doctor gains `--format=json` under a separate decision. The `recommended-plugin-not-installed` finding-code string still appears verbatim in the human-format output so a script can grep for it.
- **Old `ritualsPluginInstalled` block deleted from `doctorReport` only.** The same heuristic is also called from `aiwf init` lines 124 / 129 to print a one-shot setup nudge for fresh consumers. The init nudge still has unique value (it fires at setup time, before any `aiwf.yaml: doctor.recommended_plugins` declaration could exist) so it stays — see *Reviewer notes*.
- **`appendRecommendedPluginsReport` takes `cfg` rather than re-loading.** Existing `appendRenderReport` / `appendValidatorReport` helpers re-load `config.Load(rootDir)` themselves; the new helper accepts the already-loaded `cfg` from `doctorReport`'s top, with a nil guard for the non-`NotFound` config-load-error case where `cfg` comes back nil. Pattern divergence is intentional — re-loading three times in the same doctor run is wasteful and the nil guard is cheap.

## Validation

```
$ go build -o /tmp/aiwf ./cmd/aiwf      → exit 0
$ go test -race ./...                   → 25 packages green (cmd/aiwf 142s)
$ go vet ./...                          → clean
$ golangci-lint run --timeout=5m ./...  → 0 issues
$ /tmp/aiwf check                       → 0 errors, 4 warnings (all pre-existing,
                                          none on M-070)
$ /tmp/aiwf doctor --self-check         → self-check passed (30 steps), incl.
                                          two new M-070/AC-7 steps green
$ /tmp/aiwf show M-070                  → 7/7 ACs met, all phases done
```

Coverage on new code:

| Symbol | Coverage | Residual |
|---|---|---|
| `internal/config.preCheckTypedShape` | 100% | — |
| `internal/config.Validate` | 100% | — |
| `internal/pluginstate.Load` | 91.7% | non-`NotExist` read errors (defensive) |
| `internal/pluginstate.HasProjectScope` | 88.2% | `filepath.Abs` failures (essentially unreachable on Unix) |
| `cmd/aiwf.appendRecommendedPluginsReport` | 93.3% | `os.UserHomeDir` failure (defensive) |

Branch-coverage walk: every reachable conditional in the diff has an explicit test except the three defensive paths above. Per CLAUDE.md, these are reachable in degenerate environments only and not deterministically testable in unit-test processes.

## Deferrals

No work deferred from this milestone. Two follow-up items were *flagged* during implementation; neither blocks M-070 nor M-071, but both warrant tracking:

- **JSON envelope for `aiwf doctor`** — AC-3's spec text references an envelope structure (`finding.code` / `finding.data`) that doctor has no surface for today. A future epic adding `--format=json` to doctor would naturally land this. No gap opened — the work doesn't have a forcing function until a JSON-consuming caller appears.
- **`aiwf init`'s `printRitualsSuggestion` is hardcoded for `aiwf-extensions`** — once M-071 declares `aiwf-extensions` in this repo's `aiwf.yaml: doctor.recommended_plugins`, the init-time nudge becomes redundant *for this repo*. For other consumers (fresh `aiwf init` against a brand-new repo) the nudge is still the only signal that the rituals plugin exists. Migrating it to a config-driven shape is a directional change worth its own decision; not in M-070's scope.

## Reviewer notes

- **Why the soft-warning (no `problems++`) decision is load-bearing.** Hard-erroring on a missing recommended plugin would break CI for any consumer who hasn't yet installed everything declared in their `aiwf.yaml`. The kernel principle here: detection is the kernel's job; remediation is the operator's. Doctor's exit code is reserved for "the framework is structurally broken" (skill drift, id collisions); recommended-plugin presence doesn't fit that bar.
- **Why the helper accepts `cfg *Config` instead of re-loading.** See *Decisions* above. The existing `append*Report` helpers all re-load — the divergence here is deliberate.
- **Why the old `ritualsPluginInstalled` heuristic stays in `aiwf init` after being removed from `doctor`.** Init runs at setup time, before a consumer has had any chance to declare `doctor.recommended_plugins` in `aiwf.yaml`. The init nudge is the only discoverability signal at that moment. Doctor, by contrast, runs after the consumer is operational; if they care about ritual plugins they declare them. The two surfaces have different consumers and different temporal context; deleting one didn't imply deleting the other.
- **Why `runSelfCheck` redirects `$HOME` for the entire function rather than per-step.** Mirrors the existing `GOPROXY=off` override and keeps the per-step setup hooks focused on their own concern (writing `installed_plugins.json` under the fake home, not orchestrating env state). The seeded `~/.gitconfig` workaround is necessary because `resolveActor`'s `git config user.email` runs from the parent process's cwd and falls back to the user-scoped global config when no local repo applies — without the seed the `whoami` and other identity-dependent steps would fail mid-self-check.
- **The synthetic plugin id `aiwf-self-check@synthetic-marketplace` is deliberately fictional** so the self-check doesn't accidentally couple to a real marketplace's state.

