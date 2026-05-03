## `aiwf upgrade` plan — release tagging, upgrade verb, skew detection

**Status:** proposal · 0/N items. · **Audience:** PoC continuation. Will touch `tools/cmd/aiwf/` (new `upgrade` subcommand, extend `doctor`), `tools/internal/version/` (new package), `docs/pocv3/design/design-decisions.md` (release-and-upgrade row), `README.md` (install/upgrade quick-start).

A small kernel-mechanics iteration that turns "upgrade aiwf in a consumer repo" from a two-step ritual the user has to remember (`go install …@latest` → `aiwf update`) into a one-command flow with offline-by-default skew detection. Prerequisite: tag a release on the kernel repo so `go install …@v0.x.y` and the Go module proxy can both resolve.

---

### 1. The release primitive: git tags

Releases are git tags on the kernel repo (this branch's eventual home — for the PoC, on `poc/aiwf-v3`). Semver-required by the Go module proxy: tags must start with `v` and be valid semver (`v0.1.0`, `v0.1.1`, `v0.2.0`).

That's the entire release pipeline. No GoReleaser, no GitHub Releases page, no CI release job (a tag-gated CI build is cheap to add later but not load-bearing). Push the tag, the proxy picks it up.

Why this works:

- **`go install` already speaks tags.** `go install github.com/23min/ai-workflow-v2/tools/cmd/aiwf@v0.1.0` resolves directly via the module proxy. So does `@latest`, which the proxy maps to the highest semver tag.
- **`runtime/debug.ReadBuildInfo()` reports the tag.** A binary built via `go install …@v0.1.0` has `Main.Version == "v0.1.0"`. Built from a working tree (`go build`), it's `(devel)`. Built from `…@main`, it's a pseudo-version (`v0.0.0-<utc>-<sha>`). The running binary always knows what it is.
- **The proxy gives a free `latest` lookup.** `GET https://proxy.golang.org/github.com/23min/ai-workflow-v2/tools/cmd/aiwf/@latest` returns JSON `{"Version":"v0.1.0", "Time":"..."}`. ~50ms typical. Cached. No auth.

### 2. Module path

`go.mod` is at the repo root with module `github.com/23min/ai-workflow-v2`. The cmd lives at `tools/cmd/aiwf`, so the install path is:

```
go install github.com/23min/ai-workflow-v2/tools/cmd/aiwf@latest
```

This is what `aiwf upgrade` will invoke. Hardcoded in the binary (the canonical module path doesn't change without a major-version bump).

### 3. The new package: `tools/internal/version`

Single point of truth for version handling. Tiny surface, three jobs:

```go
package version

// Current returns the running binary's version, source, and a bool
// indicating whether it's a tagged release (vs. devel/pseudo).
func Current() Info

// Latest fetches the latest published version from the Go module proxy.
// Honors GOPROXY (returns ErrProxyDisabled if GOPROXY=off). Times out
// at 3s. Returns the parsed version and the proxy URL it queried.
func Latest(ctx context.Context) (Info, error)

// Compare returns one of: Equal, BinaryAhead, BinaryBehind, Unknown.
// Unknown when either side is a non-semver (devel, pseudo-version).
func Compare(a, b Info) Skew

type Info struct {
    Version string  // "v0.1.0", "(devel)", "v0.0.0-20260503...-abc123"
    Source  string  // "buildinfo" | "proxy" | "config"
    Tagged  bool    // true iff Version is a clean semver tag
}
```

Reads `runtime/debug.ReadBuildInfo()` for `Current()`. Uses `net/http` (no third-party HTTP) for `Latest()`. No file IO; the package is pure logic + buildinfo + one HTTP call. ≤150 lines.

### 4. The new verb: `aiwf upgrade`

Subcommand of `aiwf`. Behavior:

1. **Resolve target.** Default `@latest`. Accept `aiwf upgrade --version v0.2.0` to pin.
2. **Compare.** If running binary already at target → print `aiwf is at v0.x.y` and exit 0.
3. **Confirm.** Print the action and prompt unless `--yes` (e.g. `will install github.com/23min/ai-workflow-v2/tools/cmd/aiwf@v0.2.0`). KISS prompt — match the codebase's existing prompt style.
4. **Install.** Exec `go install <module>@<version>`. Stream stderr through. Honor `GOBIN`/`GOPATH`. Fail clearly when `go` is not on PATH (point at install instructions).
5. **Re-exec.** After successful install, re-exec the *new* binary with `aiwf update --root <cwd>`. The current process replaces itself with `syscall.Exec`. (Avoids the "running binary is the file we just overwrote" race on Linux/macOS — the kernel keeps the old inode mapped for the running process, and re-exec picks up the new one cleanly.)
6. **`aiwf update` runs in cwd.** Materialized skills + hooks refresh against the new embedded set. Same as today's `aiwf update`.

Flags:

- `--version <semver>` — pin to a specific tag instead of `@latest`.
- `--check` — do step 1–2 only; print the comparison and exit. No install.
- `--yes` — skip the confirmation prompt.
- `--root <path>` — override the consumer repo for the post-install `update` step. Defaults to cwd.

Reversal: `aiwf upgrade --version <previous>` rolls back to a prior tag. `aiwf update` re-runs against the older binary's embedded artifacts. The framework doesn't keep a history of installed versions; the user decides which tag they want.

### 5. Skew detection in `aiwf doctor`

Three comparisons, three rows. All three are advisory (no exit-non-zero) by default; a future flag could promote any of them to a problem if friction shows.

**5a. Binary version.** Always shown. Reads `version.Current()`. One of:

```
binary version: v0.2.0 (tagged)
binary version: (devel)        — built from working tree
binary version: v0.0.0-…-abc12 — built from a non-tagged commit
```

**5b. Pin coherence.** Shown when `aiwf.yaml.aiwf_version` is set. Compares pin to running binary:

```
aiwf.yaml pin: v0.2.0           — matches binary
aiwf.yaml pin: v0.1.0 (binary newer; behavior may differ — update pin or roll back binary)
aiwf.yaml pin: v0.3.0 (binary older; run aiwf upgrade)
```

The "behavior may differ" wording is intentionally cautious — the pin's job is to declare intent, not gate execution. Today the field exists but isn't enforced beyond "non-empty"; this iteration uses it for advisory reporting only. Hardening it (refuse to run when binary < pin) is a separate, deliberate decision.

**5c. Latest published (opt-in network call).** Off by default. Enable via `aiwf doctor --check-latest` or set `GOPROXY=…` (any value other than `off`) plus pass the flag. Calls `version.Latest(ctx)` with a 3s timeout. On success:

```
latest published: v0.2.1 (binary at v0.2.0; run aiwf upgrade)
latest published: v0.2.0 (up to date)
latest published: unknown (skipping — GOPROXY=off)
```

Network errors don't fail doctor; they print `latest published: unavailable (<reason>)` and continue. The whole network-aware flow is opt-in so `aiwf doctor` stays fast and offline by default.

### 6. `aiwf doctor --self-check`

Extend with a non-network smoke for `aiwf upgrade --check`:

1. In a temp dir, run `aiwf upgrade --check`.
2. Assert it prints a current-version row and exits 0.
3. Assert it does *not* perform a network call when `GOPROXY=off` is set in the env (and prints the unavailable line).

Skip the actual `go install` invocation — that's an integration test we don't want in self-check (it would require a writable GOBIN and network).

### 7. Sequencing — one commit per logical step

1. **Plan + design-decisions.md update.** This doc + a short "Release & upgrade" row in `design-decisions.md` capturing: tags as the release primitive, `aiwf upgrade` as the user-facing verb, skew detection in doctor as advisory.
2. **`tools/internal/version` package.** `Current()` + `Compare()` + tests. No HTTP yet — proves out the buildinfo path and skew classification.
3. **`tools/internal/version`: add `Latest()` + tests.** Hit `proxy.golang.org` against a known-tagged module in a `go test -short`-skipped integration test; unit tests use a fake HTTP server.
4. **`aiwf upgrade` verb.** Subcommand wired in `cmd/aiwf`. Implements `--check` first (no install), then `--yes` and the install + re-exec flow. Round-trip tested against a fake `go install` shim (env `AIWF_GO_BIN=<path>` for tests; defaults to `go` in PATH for real runs).
5. **`aiwf doctor`: extend with version rows 5a + 5b.** Always-on, no network.
6. **`aiwf doctor --check-latest`: opt-in network row 5c.** Honors `GOPROXY=off`.
7. **`aiwf doctor --self-check`: cover `aiwf upgrade --check` + offline `--check-latest`.**
8. **README + docs.** Quick-start: `aiwf upgrade` is the upgrade story, period. The `go install` line stays in install instructions for first-time setup; everything after that is `aiwf upgrade`.
9. **Cut `v0.1.0`.** Tag `poc/aiwf-v3`, push the tag. Verify `go install github.com/23min/ai-workflow-v2/tools/cmd/aiwf@v0.1.0` works from a clean shell and the resulting binary reports `version v0.1.0`.

Each step compiles and tests on its own. The cut-the-tag step (9) is last because it's the moment the proxy starts serving — anything before that has to fall back to `@main` (pseudo-version), and `aiwf upgrade --check` needs to handle that gracefully (it does, via `Compare()` returning `Unknown` for pseudo-versions).

### 8. What stays out of scope

- **Hard-fail on pin mismatch.** Pin coherence stays advisory. A separate iteration can decide "binary < pin = refuse to run" if the friction warrants it.
- **A custom proxy / private registry story.** The standard `GOPROXY` env var covers it for free; nothing custom needed.
- **CI release automation.** A tag-on-merge workflow can land later. Manual `git tag vX.Y.Z && git push --tags` is fine for the PoC.
- **Self-update without `go install`.** Some tools fetch a binary directly (gh, rustup). Going through `go install` is simpler and matches how the kernel is distributed today; revisit if Go-toolchain-on-the-consumer-machine becomes a real constraint.
- **A separate "channel" concept (stable/beta).** YAGNI. `--version` covers pinning, `@latest` is the channel.
- **Pre-1.0 module path version dance.** v0.x.y in semver allows breaking changes. We don't need a `/v2` path-suffix until we cut `v1.0.0`. Even then, deferred until we actually do.

### 9. Validation

Standard PoC pre-commit gate (`go test -race ./tools/...`, `golangci-lint run`, `go build`) plus the extended `aiwf doctor --self-check` per step 7 plus a real-shell smoke after step 9: `go install …@v0.1.0` from a clean dir, run `aiwf doctor`, assert the `(tagged)` line.
