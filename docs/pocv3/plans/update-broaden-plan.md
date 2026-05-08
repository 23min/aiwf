## Broaden `aiwf update` plan

**Status:** implemented across commits `88727c6` (kernel-shift docs) → `855996a` (self-check covers the round-trip). `aiwf update` is now the upgrade verb; the pre-commit hook for STATUS.md regeneration is default-on with `status_md.auto_update: false` as the clean opt-out. · **Audience:** PoC continuation. Touched `internal/initrepo/`, `internal/config/`, `cmd/aiwf/admin_cmd.go`, `cmd/aiwf/selfcheck.go`, `docs/pocv3/design/design-decisions.md`, `docs/pocv3/design/design-lessons.md`, `CLAUDE.md`, `README.md`.

A kernel-level shift in what `aiwf update` does, plus the first new artifact it carries: a marker-managed pre-commit hook that regenerates a committed `STATUS.md` on every commit, default-on with a clean opt-out.

---

### 1. The kernel shift

Today, `design-decisions.md` says: *"Skills materialized into the consumer repo's `.claude/skills/aiwf-*` directory and gitignored. Regenerated only on explicit `aiwf init` / `aiwf update`."* That binds `aiwf update` specifically to skill re-materialization.

The new contract: **`aiwf update` is the upgrade verb. It refreshes every marker-managed framework artifact the consumer is opted into — embedded skills, embedded git hooks, and any future templated artifact the framework ships.** `aiwf init` becomes "first-time setup that includes one update pass at the end." Re-running either verb converges to the same state for a given binary version + `aiwf.yaml`.

This closes a pre-existing inconsistency: today `aiwf update` does not refresh the pre-push hook, so on a binary upgrade the user must re-run `aiwf init` to pick up hook drift (G12 detects it via `aiwf doctor`, but doesn't fix it). Generalizing fixes that for free.

The principle replaces design-decisions §"Layered location-of-truth" / row 5 with a slightly broader phrasing. The change is loaded enough to land in the design doc, not just the code.

### 2. The first new marker-managed artifact: pre-commit hook + STATUS.md

A second git hook installed by `aiwf init` / refreshed by `aiwf update`, alongside the existing pre-push hook:

- **Path:** `.git/hooks/pre-commit` (gitignored — same as pre-push; not tracked).
- **Marker:** the first content-line is `# aiwf:pre-commit` (mirror of the existing `# aiwf:pre-push` constant). The marker protects user-written pre-commit hooks: if a non-marker hook is already in place, `aiwf init/update` skips installation, prints a remediation block, and leaves the user to compose hooks manually (mirror of how `ensurePreHook` already handles this for pre-push).
- **Body:** the same KISS shape as the local `scripts/git-hooks/pre-commit` we just landed in this dev repo — `command -v aiwf` guard, run `aiwf status --root … --format=md > STATUS.md.tmp`, atomic move, `git add STATUS.md`, silently no-op on any failure. Embedded in the binary; the consumer-side file is a derivable cache.
- **Regenerable from the binary:** `aiwf doctor` byte-compares the installed pre-commit hook against the embedded template (mirror of G12's pre-push handling) and reports drift. `aiwf update` is the one-button restore.

The committed artifact is `STATUS.md` at the consumer's repo root. The hook keeps it fresh against the entity tree on every commit.

### 3. The opt-out

Default: **on.** A consumer who runs `aiwf init` against a fresh repo gets the pre-commit hook installed and STATUS.md regenerating on every commit. The committed snapshot benefits anyone browsing the repo on GitHub, Gitea, etc. — exactly the host-agnostic visibility we wanted from the markdown renderer in the first place.

Opt-out: a new `aiwf.yaml` field `status_md.auto_update: bool` (default `true`). Setting it to `false`:

- Causes `aiwf init` and `aiwf update` to *not* install the pre-commit hook.
- Causes them to *remove* an existing marker-managed pre-commit hook if one is present (so flipping the flag and running `aiwf update` cleanly opts out). A non-marker hook is left alone, same as init's pre-push behaviour.
- Does **not** delete a previously-committed `STATUS.md`. That file is the user's content once committed; if they want it gone, `git rm STATUS.md` is a one-line action they can take deliberately.

The aiwfyaml row:

| Key | Type | Default | Meaning |
|---|---|---|---|
| `status_md.auto_update` | bool | `true` | Install the pre-commit hook that keeps `STATUS.md` in sync with the entity tree. Set to `false` to opt out (and uninstall the marker-managed hook on next `aiwf init`/`update`). |

A field, not a top-level boolean, so the namespace is open for future status-related toggles (e.g., a custom path, a section filter) without churn.

### 4. The shared installer pipeline

Today, `aiwf init` runs a sequence of steps and emits a per-step ledger. `aiwf update` calls just one of those steps (`materializeSkills`). The simplest refactor that supports the new contract:

- Extract the **post-config installer pipeline** into a single function, e.g. `initrepo.RefreshArtifacts(ctx, root, cfg) []StepResult`. The steps it runs:
  1. Refresh `.claude/skills/aiwf-*` (existing).
  2. Refresh `.gitignore` patterns (existing).
  3. Refresh `.git/hooks/pre-push` (existing — newly called from `update` too).
  4. Refresh `.git/hooks/pre-commit` (new — gated on `cfg.StatusMd.AutoUpdate`).
- `aiwf init` keeps its scaffolding-and-aiwf.yaml writes, then ends with one call to `RefreshArtifacts`.
- `aiwf update` becomes a thin wrapper around `RefreshArtifacts` (load aiwf.yaml + tree, lock, refresh, print ledger).

The ledger contract stays the same: each step returns a `StepResult{What, Wrote, Skipped, Reason}` so the existing init output format extends naturally.

### 5. `aiwf doctor`

Add a row analogous to the pre-push reporting from G12:

- `pre-commit hook: <state>` where state ∈ {`ok`, `disabled by config`, `missing`, `stale`, `not aiwf-managed`, `malformed`}.
- `disabled by config` is the new state for when `status_md.auto_update: false` and there's no marker-managed hook on disk — the desired-and-actual states agree, so it's a normal report row, not a problem.
- `stale` (drift between installed body and embedded template) and `missing` (config says install, hook isn't there) increment the doctor problem count so it exits non-zero. Remediation: run `aiwf update`.

### 6. `aiwf doctor --self-check`

Extend the self-check to drive the new path end-to-end:

1. Init a temp repo (`aiwf init` with default config — pre-commit hook lands).
2. Assert pre-commit hook exists, has the marker.
3. Edit aiwf.yaml to `status_md.auto_update: false`.
4. Run `aiwf update`.
5. Assert pre-commit hook is gone (marker-managed hook removed).
6. Edit aiwf.yaml back to `true`. Run `aiwf update`. Assert the hook is back.

### 7. Reversal

Per `CLAUDE.md` verb-design rule, every mutation needs a reversal answer:

- `aiwf update`'s reversal is *another invocation of `aiwf update` with different `aiwf.yaml`*. Flip the flag and re-run — the hook installs or uninstalls, the skills converge to the binary's embedded set.
- For STATUS.md as a tracked artifact: `git rm STATUS.md` plus setting the flag false is the deliberate opt-out. The framework doesn't auto-delete user content.

### 8. What stays out of scope

- **Migrating this dev repo to be an aiwf consumer.** The repo doesn't have `work/` and shouldn't — it's the framework's own source tree. The local `scripts/git-hooks/pre-commit` + `make hooks` we just committed stays. The framework feature is for *consumer* repos.
- **Other framework artifacts beyond hooks + skills.** The pipeline is structured to take more steps later (CI workflow files, per-kind templates that materialize alongside skills) but YAGNI: only the pre-commit hook lands in this iteration.
- **Auto-uninstall of the pre-push hook on flag flip.** The pre-push hook is non-optional — without it the framework's guarantees aren't enforced. There's no flag to disable it.
- **Migrating existing consumers automatically.** A consumer who upgrades the binary and runs `aiwf update` for the first time post-change finds the new hook installed (default-on). If they don't want it, they set the flag false and re-run `aiwf update`. One-off prose in the next release notes is enough.

### 9. Sequencing

One commit per logical step:

1. **Plan + design-decisions.md update.** This doc + the row 5 / "Layered location-of-truth" rephrasing in `design-decisions.md`. No code yet.
2. **`aiwfyaml`: add `status_md.auto_update` field with round-trip + tests.** Default `true`. Doesn't drive any behavior yet.
3. **`initrepo`: embed the pre-commit hook script + `ensurePreCommitHook` installer/uninstaller** with marker, drift detection, alien-hook skip, and tests. Doesn't get called from anywhere yet.
4. **`initrepo`: extract `RefreshArtifacts`** common pipeline; wire `aiwf init` to call it; add pre-push refresh to the pipeline (free win — `update` will pick it up next).
5. **`aiwf update`: switch to `RefreshArtifacts`.** Now `update` refreshes hooks. Update `admin_cmd.go` + tests + integration test.
6. **`aiwf doctor`: report pre-commit hook state.** Add the new row + drift detection + tests.
7. **`aiwf doctor --self-check`: exercise the install/uninstall transition.** Extend the existing self-check runner.
8. **README + docs** — update the "What aiwf writes" / coexistence section to mention pre-commit; mention the opt-out.

Each step compiles and tests on its own.

### 10. Validation

Standard PoC pre-commit gate (`go test -race ./...`, `golangci-lint run`, `go build`) plus the extended `aiwf doctor --self-check` per step 7.
