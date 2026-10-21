---
id: G-0729
title: design-decisions.md and ANTI-0008 describe a doctor setting no code reads
status: open
discovered_in: E-0098
---
## What's missing

Two current-truth sources describe an `aiwf.yaml` setting the kernel no longer has.
`docs/design/design-decisions.md` (line 344, the `aiwf.yaml` config table) lists a `doctor`
mapping with `recommended_plugins`, and the anti-rule ANTI-0008 in
`internal/workflows/spec/antirules.go` (line 65) states that "`aiwf.yaml.doctor.recommended_plugins`
is opt-in; default empty". The recommended-plugins check was retired: the config struct has no
`doctor` key and no `RecommendedPlugins` field, and nothing reads the setting.

Measured in the devcontainer at the head of the E-0098 epic branch:

```sh
grep -rn 'recommended_plugins\|RecommendedPlugins' --include='*.go' . | grep -v _test
```

Observed: one line, the ANTI-0008 statement in `internal/workflows/spec/antirules.go`. The
retirement itself is recorded in `docs/audits/entity-truth-audit.md` (lines 721-723), which
measured the same absence.

## Why it matters

`design-decisions.md` is in the Normative tier, read as current truth. A consumer or contributor
reading it sets a `doctor.recommended_plugins` key that aiwf ignores, and expects a check that
no longer runs; the anti-rule repeats the claim inside the workflow spec.
