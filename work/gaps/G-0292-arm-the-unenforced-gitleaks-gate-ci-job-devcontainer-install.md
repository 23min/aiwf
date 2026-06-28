---
id: G-0292
title: 'Arm the unenforced gitleaks gate: CI job + devcontainer install'
status: addressed
addressed_by_commit:
    - 3115da4e
---
## Problem

The gitleaks secret / path-leak gate (G-0103) has never actually been
enforced in this repo. gitleaks is not installed on the devcontainer or
on PATH, and there is no CI job that runs it — so the pre-commit gitleaks
block self-skipped silently on every commit (`gitleaks not on PATH —
path-leak gate skipped`). The hook comment even claimed "CI runs gitleaks
independently," which is false: there is no `.github/workflows/gitleaks`
job (G-0103's archive note recorded the CI workflow as a deferred
follow-up that was never done).

This violates the kernel principle "framework correctness must not depend
on operator behavior": the gate's effectiveness depended on each operator
manually `go install`ing gitleaks, which no one did.

## Evidence

Surfaced 2026-06-27 while verifying G-0291 (the pre-commit -> pre-push
relocation) against real gitleaks. With gitleaks finally installed and
run:

- Full git-history scan: **67 path-leak findings**, all the
  `path-leak-darwin-home` rule (`/Users/<name>/`) — 65 are the
  maintainer's own home path and 2 are the codified test-placeholder path
  (allowlisted only under `_test.go`), committed over a ~3-week window
  (2026-05-07 .. 2026-05-27) while the gate was decorative.
- Current-files (HEAD) scan: **0 findings** — the leaks were sanitized out
  of the working tree over time but remain in history.

So the gate that was supposed to keep the count at 0 (G-0103 drove it to 0
at the time) silently let it drift to 67, because nothing ran it.

## Direction — make the gate real

1. **CI chokepoint** (`.github/workflows/gitleaks.yml`): the
   operator-independent gate. Install a pinned gitleaks, run
   `gitleaks git --config=.gitleaks.toml` on push and PR, fail on any new
   finding. This is the piece that makes the guarantee real.
2. **`.gitleaksignore`**: accept the historical fingerprints. History
   rewrite is forbidden (it would break every SHA, `aiwf history`, and
   trailers) and the leaked path is the maintainer's own already-public
   home path — so the specific past occurrences are accepted by
   fingerprint. This is NOT a pattern allowlist: a *new* `/Users/<name>/`
   leak still fails, including the maintainer's own.
3. **Devcontainer**: install the same pinned gitleaks so the local
   pre-push hook actually fires (fast feedback before CI).
4. **Pre-push relocation (G-0291)**: the local fast-feedback complement
   (range scan). The `.gitleaksignore` also unbreaks its full-history
   `scan_all` fallback, which would otherwise false-block on the historical
   findings on this repo.
5. **Docs**: flip the pre-push hook comment and the CLAUDE.md "what's
   enforced and where" row from "no CI backstop today" to CI-backed.
6. **Policy test** (`internal/policies/`): pin the wiring — the workflow
   exists and runs gitleaks with `--config=.gitleaks.toml`, the
   `.gitleaksignore` is present, and the devcontainer installs gitleaks —
   so the chokepoint cannot silently rot (mechanical evidence per the
   "what's enforced and where" discipline).

## Relationship

- Complements **G-0291** (relocate gitleaks to pre-push): G-0291 moved the
  local gate to the right trust boundary; this gap supplies the CI backstop
  and local install that make the gate actually fire. Both are closed by
  the same "arm gitleaks" change.
- Closes the "no CI backstop" hole that G-0291's own (honest) hook comment
  named, and supersedes the deferred CI-gitleaks follow-up recorded in
  G-0103's archive note.
- A related but distinct follow-up (NOT in scope here): the embedded
  `wf-doc-lint` ritual still recommends gitleaks as a *pre-commit* hook to
  consumers — stale advice now that aiwf itself uses pre-push + CI. File
  separately.

## Provenance

Discovered mid-G-0291-patch: verifying the relocated hook against real
gitleaks revealed (a) the tool was never installed, (b) CI never ran it,
and (c) 67 leaks had accumulated in history. The relocation's stated
latency rationale was also found hollow — an absent tool self-skips
instantly — reframing the work from "move a gate" to "arm a gate that was
never firing."
