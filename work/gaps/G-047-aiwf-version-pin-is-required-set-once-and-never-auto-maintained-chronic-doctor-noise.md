---
id: G-047
title: '`aiwf_version` pin is required, set-once, and never auto-maintained — chronic doctor noise'
status: addressed
addressed_by_commit:
  - 25bf5ea
---

Resolved in commit `25bf5ea` (feat(aiwf): G47 — retire the aiwf_version pin field). The field is no longer required by `internal/config/config.go` (validation drops the requirement); `aiwf init` no longer writes it (`Config{}` is the default and an empty marshal becomes a comment-header so later hand-edited yaml blocks parse correctly); `aiwf update` strips it via `StripLegacyAiwfVersion` (mirror of the legacy-actor-strip pattern); doctor's `pin:` row goes away and the `config:` row drops the `(aiwf_version=…)` text. Two new helpers + an opt-in deprecation note on doctor for any pre-G47 yaml the consumer hasn't yet updated. Tests: legacy yamls load fine, the strip is idempotent, fresh init writes neither `actor:` nor `aiwf_version:`, and the doctor advisory fires for legacy yamls but doesn't increment the problem count.

`aiwf init` writes `aiwf_version: <binary-version>` to `aiwf.yaml`; the field is currently *required* by the loader (`internal/config/config.go:215`: `aiwf_version is required`). Nothing maintains the field after init. `aiwf doctor` compares the pinned value against the running binary and reports a "pin skew" row whenever they disagree. After any binary upgrade, the row becomes a chronic nag — the consumer didn't pin intentionally; the value was just whatever was current at first init.

**Concrete reproducer:** consumer init'd at v0.1.1 a year ago, ran `aiwf upgrade` to v0.4.0 today, runs `aiwf doctor`:

```
config:    ok (aiwf_version=v0.1.1)
pin:       pinned v0.1.1, binary newer (v0.4.0) — update pin or roll back binary
```

The user's reasonable reaction is "I never pinned anything, why am I being asked to update a pin?"

**Why the field exists at all** (kernel arc context):

The pin has two implicit purposes that are in tension:

1. *Audit signal* — "this consumer last ran against version X." Wants auto-bump on every update. Cheap to maintain.
2. *Intentional pin* — "this consumer wants to stay on version X." Wants manual-only updates. Doctor's skew row is the load-bearing UX.

The current shape tries to do both: the field is set automatically on init but never bumped, so an unintentional pin from year-old init looks indistinguishable from a deliberate "stay on v0.1.1" choice. Doctor can't tell which it is and nags either way.

**Resolution: remove the field entirely** (YAGNI). The pin's information is available via cheaper channels:

- *"What version am I on?"* → `aiwf version` (the binary self-reports; no yaml lookup needed).
- *"Is there a newer release?"* → `aiwf doctor --check-latest` (queries the module proxy; opt-in).
- *"What was this consumer's last init/upgrade against?"* → reachable via `git log` on `aiwf.yaml` if you really need it.

The field's only kernel-side consumer is doctor's pin row, which becomes vestigial once the field goes.

This is the same shape as the `actor` field removal in I2.5: a field that *was* stored in `aiwf.yaml`, became runtime-derivable from authoritative sources elsewhere (`git config user.email`), and was retired via an auto-strip on `aiwf update`. The legacy-actor-strip step is the migration template here.

**Resolution path for v0.5.0 (proposed):**

1. **Loader becomes tolerant.** `internal/config/config.go` drops the `aiwf_version is required` validation. Existing yamls with the field load fine; new yamls without it load fine; no error.
2. **`aiwf init` stops writing the field.** New initializations produce a yaml without `aiwf_version:`.
3. **`aiwf update` strips the field on refresh.** Same pattern as the legacy actor strip — ledger reports `preserved aiwf.yaml (legacy aiwf_version strip)` when the field is removed.
4. **`aiwf doctor` drops the pin row.** The `binary:` row stays (always shown); `pin:` and `latest:` rows merge into a single optional advisory: `latest:` (opt-in via `--check-latest`).
5. **Update CLAUDE.md / README** to remove pin-related references.

The discoverability lint will flag the missing `aiwf_version` reference if any embedded skill or doc still names it; the discoverability haystack just stops including the field name.

**Severity:** Medium. Doctor noise is a UX nag, not a correctness issue, but it surfaces every doctor run and trains consumers to ignore the row — which is the opposite of what doctor exists to do. Resolution is a small, well-scoped follow-up that mirrors a previously-shipped pattern (legacy actor strip).

---

<a id="g45"></a>
