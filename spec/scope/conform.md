# The scope spine against today's aiwf

`spine conform` played `spine.yaml` against the `aiwf` binary through
`conformance/adapter_scope.py`, and each divergence it reported is classified below as the change
ADR-0056 makes or as a defect with a gap (M-0364/AC-4).

## The run

- **Date:** 2026-11-26, started 17:16:42 UTC; run time 1063 seconds.
- **verifier-kit:** v0.3.3 at `0952e0f`, a fresh clone with `spine` built as `spec/README.md`
  says.
- **aiwf:** `v0.42.1-0.20261121183124-0c00b37c47b3+dirty`, built from a tree whose code is `main`
  at `0c00b37c4`. Go files differ at `main` `0176635c3`; a second margin-2 run the same day, with
  `aiwf` built from that commit, printed output byte-identical to this run's. That binary reads
  `(devel)`; built again from that commit in the same folder, it matched byte for byte, and a
  build of `0c00b37c4` there did not. Go writes the build folder into the binary, so only builds
  made in one folder compare.
- **Environment:** Linux devcontainer, git 2.51.1, Python 3.12.3; `aiwf` built by go1.25.12 and
  `spine` by go1.27.1.
- **Inputs (git hash-object):** `spec/scope/spine.yaml` `917b06ae67c7d9e65a818504e50ccb7686dba0cf`,
  `conformance/adapter_scope.py` `67d1b929cd83196c8254feb696c5996151cb8192`.
- **Command,** from the repository root, with `$VK` the verifier-kit checkout and its `tools/`
  and the folder holding that `aiwf` first on `PATH`, as `spec/README.md` gives it:

  ```
  "$VK/tools/bin/spine" conform spec/scope/spine.yaml --margin 2 \
    --adapter-cmd "adapter_shim.py conformance/adapter_scope.py"
  ```

- **Expected:** divergences only where the audit recorded that today's aiwf differs: a rebase or
  a copy (`audit/decisions.md` Q1–Q3), a second authorize (Q4), a repeat pause or resume (Q9) and
  a reallocate (Q15).
- **Observed:** exit 1, every one of the 50 cases tried, and 16 divergences, each in one of those
  places:

```
error   diverges           active/authorize: after authorize() -> authorize(): spine expects ok in active, code gave ok in in_progress; scopes not ended: active, active
error   diverges           active/resume: after authorize() -> resume(): spine expects ok in active, code gave rejected in active
error   diverges           active/copy: after authorize() -> copy(): spine expects ok in active, code gave ok in in_progress; scopes not ended: active, active
error   diverges           active/reallocate: after authorize() -> reallocate(): spine expects ok in active, code gave ok in none
error   diverges           paused/authorize: after authorize() -> pause() -> authorize(): spine expects rejected in paused, code gave ok in in_progress; scopes not ended: paused, active
error   diverges           paused/pause: after authorize() -> pause() -> pause(): spine expects ok in paused, code gave rejected in paused
error   diverges           paused/copy: after authorize() -> pause() -> copy(): spine expects ok in paused, code gave ok in in_progress; scopes not ended: paused, paused
error   diverges           paused/reallocate: after authorize() -> pause() -> reallocate(): spine expects ok in paused, code gave ok in none
error   diverges           ended/rebase: after authorize() -> end() -> rebase(): spine expects ok in ended, code gave ok in active
error   diverges           ended/copy: after authorize() -> end() -> copy(): spine expects ok in ended, code gave ok in active
error   diverges           ended/reallocate: after authorize() -> end() -> reallocate(): spine expects ok in ended, code gave ok in none
error   diverges           closed/rebase: after authorize() -> close() -> rebase(): spine expects ok in closed, code gave ok in done; scopes not ended: active
error   diverges           closed/copy: after authorize() -> close() -> copy(): spine expects ok in closed, code gave ok in done; scopes not ended: active
error   diverges           active/rebase: after authorize() -> end() -> authorize() -> rebase(): spine expects ok in active, code gave ok in in_progress; scopes not ended: active, active
error   diverges           active/check: after authorize() -> act() -> rebase() -> check(): spine expects ok in active, code gave rejected in active
error   diverges           paused/rebase: after authorize() -> end() -> authorize() -> pause() -> rebase(): spine expects ok in paused, code gave ok in in_progress; scopes not ended: active, paused
info    coverage           1290 sequences, margin 2: 50 of 50 state/event/variable cases exercised
-- 16 errors, 17 findings
```

## Classification

**The change ADR-0056 makes.** Its §Consequences: "A rebase no longer breaks a scope opened from
this release on, nor does any cherry-pick or merge that keeps commit messages, and a copy of such
a scope's opening commit no longer opens a second scope. That closes G-0751, G-0752 and G-0755
for those scopes."

| Case | What aiwf did | Gap |
|---|---|---|
| active/check | after an agent's verb and a rebase, `aiwf check` fails | G-0751 |
| ended/rebase, closed/rebase | the rebase brings the ended scope back | G-0752 |
| active/rebase, paused/rebase | the rebase brings the earlier, ended scope back beside the new one | G-0752 |
| active/copy, paused/copy | the copy opens a second scope | G-0755 |
| ended/copy, closed/copy | the copy's rebased twin of the opening commit is a scope the end does not name | G-0752, G-0755 |

**Defects with a gap.**

| Case | What aiwf did | Gap |
|---|---|---|
| active/reallocate, paused/reallocate, ended/reallocate | after the reallocate, `aiwf show` lists no scope | G-0776 |
| active/authorize, paused/authorize | a second authorize opens a second scope, where audit Q4 makes it a no-op from active and a refusal from paused | G-0460 |
| active/resume, paused/pause | a repeat resume or pause is refused, where audit Q9 makes it a no-op | G-0786 |

Conform stops a sequence at its first divergence, so the reallocate sequences end at `observe`.
What else G-0776 records after a reallocate, the refused pause, resume and end and the close that
leaves the scope open, is not reached.
