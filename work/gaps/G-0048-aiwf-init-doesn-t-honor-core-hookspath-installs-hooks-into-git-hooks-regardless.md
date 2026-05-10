---
id: G-0048
title: '`aiwf init` doesn''t honor `core.hooksPath` — installs hooks into `.git/hooks/` regardless'
status: addressed
addressed_by_commit:
  - 6432f0f
---

`aiwf init` writes hooks into `<gitDir>/hooks/<name>` (via `gitops.GitDir(root)`), which is git's default location. A consumer who has set `core.hooksPath` to a tracked directory (e.g. `scripts/git-hooks/`) ends up with aiwf's hooks at `.git/hooks/<name>` where git won't invoke them — `core.hooksPath` overrides the default. The chokepoint silently fails to fire, but `aiwf doctor`'s hook row still reports `ok` because the file exists at the path it knows to check.

Surfaced by G38 dogfooding the kernel: this repo had `core.hooksPath = scripts/git-hooks` and a tracked `scripts/git-hooks/pre-commit` (the "policy lint" hook installed by `make install-hooks`). The G38 mitigation was to drop `core.hooksPath` and migrate the existing hook content to `.git/hooks/pre-commit.local` (the G45 chain target), then run `aiwf init`. Treating the kernel like any consumer: aiwf owns `.git/hooks/<name>`, the consumer's tracked logic lives at `<name>.local`, the chain composes correctly. The friction is that the consumer has to manually drop `core.hooksPath` *first* — there's no auto-detection.

**Resolution shape (proposed):**

1. **`aiwf init` reads `core.hooksPath` via `git config --get core.hooksPath`.** If unset, default behavior (write to `<gitDir>/hooks/`); if set, write to the configured directory instead.
2. **The G45 auto-migration runs against the *configured* directory.** A non-marker hook there gets moved to `<configured>/pre-push.local` etc. — same semantics as today, just rooted at a different parent.
3. **`aiwf doctor` reads `core.hooksPath` and reports against it.** The hook row's "ok / missing / stale path" judgement applies to whatever path is actually in use, not the hardcoded `.git/hooks/`.
4. **A new `aiwf doctor` advisory** for the rare case where the consumer changes `core.hooksPath` between init and doctor (e.g., they ran init with the default, then later set `core.hooksPath` and forgot to re-run init): "hook path changed from `.git/hooks/` to `scripts/git-hooks/`; run `aiwf update` to migrate."

Severity: **Medium**. Not common (most consumers leave `core.hooksPath` unset), but silently undermines the chokepoint when it does fire. The dogfood workaround is documented as the "drop core.hooksPath, use the chain" pattern; G48 makes the pattern automatic.

---

<a id="g47"></a>
