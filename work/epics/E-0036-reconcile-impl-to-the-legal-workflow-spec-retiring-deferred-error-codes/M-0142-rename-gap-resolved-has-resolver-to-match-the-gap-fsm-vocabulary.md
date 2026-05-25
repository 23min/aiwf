---
id: M-0142
title: Rename gap-resolved-has-resolver to match the gap FSM vocabulary
status: in_progress
parent: E-0036
tdd: required
acs:
    - id: AC-1
      title: Decision D-0012 records the rename and downstream-consumer caveat
      status: met
      tdd_phase: done
    - id: AC-2
      title: Finding fires under the new code; old literal absent from impl/spec/hint
      status: met
      tdd_phase: done
    - id: AC-3
      title: Hint table carries an entry for the new code name
      status: met
      tdd_phase: done
---
## Goal

Author a small decision (D-0012) recording the rename and its downstream-JSON-consumer caveat, then atomically rename `gap-resolved-has-resolver` → `gap-addressed-has-resolver` across `internal/check/check.go`, `internal/check/hint.go`, `internal/workflows/spec/rules.go`, and every string-matching test / fixture / golden under `internal/` — in one commit.

## Context

The code was named when the gap FSM used `resolved` as the addressed terminal; the current FSM uses `addressed` and `wontfix`. A reader of the code or of `aiwf check` output has to mentally translate. The rename is mechanical but spans impl, spec, hints, and fixtures, and could break downstream tools that ingest the old code from `aiwf check --format=json` — hence a recorded pre-decision rather than a silent rename. (Surfaced concretely this session: the rule fired during gap-closure as `gap-resolved-has-resolver`.)

## Acceptance criteria

Each AC carries an explicit **Evidence** gate — the named test or assertion that fails if the claim breaks. "Looks right" is not evidence.

### AC-1 — Decision D-0012 records the rename and downstream-consumer caveat

D-0012 records the rename `gap-resolved-has-resolver` → `gap-addressed-has-resolver`, its rationale (FSM-vocabulary coherence), and the downstream-consumer caveat — the code string is the stable key in the `aiwf check --format=json` `findings[].code` surface, so the rename is a breaking change for any tool that pins the old literal. Status `accepted`. *Evidence:* a `internal/policies/` structural assertion that D-0012 resolves via the loader, is `accepted`, carries its named sections (`## Context`, `## Resolution`, `## Consequences`) with non-empty prose, and names both code strings plus the JSON-surface caveat in the relevant section (scoped to the section, not a flat grep).

### AC-2 — Finding fires under the new code; old literal absent from impl/spec/hint

The `gapResolvedHasResolver` rule emits `Code: "gap-addressed-has-resolver"` when a gap is `addressed` with both `addressed_by` and `addressed_by_commit` empty, and the old literal `gap-resolved-has-resolver` appears nowhere in non-archive `internal/` source (impl, spec, hint, tests, fixtures, goldens). *Evidence:* a check-rule test in `internal/check/` driving a gap-addressed-no-resolver fixture through `check.Run` and asserting the exact new code on the finding; plus a `internal/policies/` absence chokepoint walking non-archive `internal/` and asserting zero occurrences of the old literal (its needle assembled from fragments so the asserting file itself is not a match — the policy fires if any source reintroduces the old name).

### AC-3 — Hint table carries an entry for the new code name

`hint.go`'s `hintTable` carries a `gap-addressed-has-resolver` entry; the rule emission and the hint key are renamed together so every emitted code still resolves to a hint. *Evidence:* the existing `PolicyFindingCodesHaveHints` policy stays green post-rename (it fails if an emitted `Code:` literal has no hint key); load-bearingness shown by a throwaway mutation — renaming only the emission, not the hint key, drives the policy red — then reverted.

## Constraints

- Atomic — one commit across all surfaces (impl, spec, hint, fixtures), so no intermediate state has a dangling code.
- Pre-decision (D-NNNN) lands first.
- `tdd: required`.

## Out of scope

Other finding codes; the classifier (M3) — though if M3 has landed, this rename updates the classified set in the same pass.

## Dependencies

None (independent). Best executed after M3 so the classified legality set is renamed in one pass (soft). Closes G-0144.

## Work log

The decision (D-0012) and the rename landed across four commits: `aiwf add decision D-0012` + `aiwf promote D-0012 accepted` (the AC-1 deliverable), then the single feature commit `7865516d` carrying the rename across all 23 string-matching surfaces plus the three AC tests, then the per-AC phase/met promotes. Per-AC RED→GREEN was demonstrated before the feature commit (clean assertion failures, not compile errors). The rename-test file was sequenced to land *with* the feature commit (it asserts the post-rename / post-decision state, so it is red until both exist); its RED was proven against the live tree first.

### AC-1 — Decision D-0012 records the rename and downstream-consumer caveat

D-0012 (`accepted`) records the rename `gap-resolved-has-resolver` → `gap-addressed-has-resolver`, the FSM-vocabulary rationale, and the downstream-consumer caveat over the `aiwf check --format=json` `findings[].code` surface — with the verified breakage analysis (no `aiwf.yaml` knob and no committed rendered artifact references finding codes, so only a hand-written JSON-parsing script breaks; upgrade-gated per consumer; one-command per-repo `grep` confirmation). decision `2c981169`+`7c41fd11`, tests `7865516d` · test: `TestM0142_AC1_Decision` (loader-resolved, status + named sections + scoped caveat assertions).

### AC-2 — Finding fires under the new code; old literal absent from impl/spec/hint

`gapAddressedHasResolver` (renamed from `gapResolvedHasResolver`) emits `Code: "gap-addressed-has-resolver"`; the retired literal appears nowhere in non-archive `internal/`. commit `7865516d` · tests: `TestGapAddressedHasResolver` (the rule unit test, now also pinning the code) + `TestM0142_AC2_OldGapCodeFullyRenamed` (the absence chokepoint — walks all of `internal/`, skips `archive/`, needle fragment-assembled so it scans itself; RED listed 22 offender files → GREEN 0).

### AC-3 — Hint table carries an entry for the new code name

`hint.go`'s `hintTable` carries the `gap-addressed-has-resolver` key (gofumpt re-aligned the map column for the one-char-longer key). The emission and hint key renamed together, so `PolicyFindingCodesHaveHints` stays green. commit `7865516d` · tests: `PolicyFindingCodesHaveHints` (the load-bearing chokepoint) + `TestM0142_AC3_HintKeyPresent` (readable companion). Load-bearingness of the hints policy is established (renaming only one side would fire it).

## Decisions made during implementation

- **D-0012 — Rename `gap-resolved-has-resolver` → `gap-addressed-has-resolver`** (`accepted`). Clean rename, no dual-emit alias; the breaking JSON-surface change is documented in D-0012 + CHANGELOG and is upgrade-gated per consumer. The internal Go identifier and unit-test name were renamed in the same pass for full vocabulary coherence (a deliberate extension beyond the dashed-code ACs, confirmed with the operator).

## Validation

```
CGO_ENABLED=0 go build ./...            # exit 0
go test ./... -count=1 -parallel 8      # 56 packages ok · 0 failures
golangci-lint run                       # 0 issues (gofumpt re-alignment applied to hint.go)
aiwf check                              # 0 errors · 8 warnings (pre-existing: M-0102 ×5, G-0061 ×3)
```

Per-AC mechanical evidence (all green): `TestM0142_AC1_Decision` (AC-1); `TestGapAddressedHasResolver` + `TestM0142_AC2_OldGapCodeFullyRenamed` (AC-2); `PolicyFindingCodesHaveHints` + `TestM0142_AC3_HintKeyPresent` (AC-3). The cli/integration golden suite stayed green through the literal swap (the renamed code sorts into the same relative output position).

## Deferrals

No deferral-gaps; no deferred or cancelled ACs (all three `met`). The non-rewrite of historical surfaces (`work/gaps/G-0166`, the `docs/pocv3/design/legal-workflows-audit*.md` snapshots, archived entities) is a recorded scope decision in D-0012 (forget-by-default), not deferred debt.

## Reviewer notes

- **Clean rename, no alias.** D-0012 verified the breakage surface is narrow before choosing the break: no `aiwf.yaml` knob names finding codes, no committed `STATUS.md`/`ROADMAP.md` embeds them — only a hand-written `--format=json` parser could break, and the rename is upgrade-gated per consumer with a one-line `grep` confirmation.
- **Identifier rename beyond the ACs.** The ACs require only the dashed code string; the Go identifier `gapResolvedHasResolver` and its test name were renamed too (they carry no dashes, so the absence chokepoint does not enforce them) for full FSM-vocabulary coherence — the milestone's whole point.
- **Absence chokepoint is self-scanning.** `TestM0142_AC2_OldGapCodeFullyRenamed` walks every file under `internal/` (skipping `archive/`) and asserts zero occurrences of the retired literal; its needle is assembled from two fragments so the asserting file is not itself a false positive. The policy fires if any future source reintroduces the old name.
- **README was incomplete, not inverted.** On inspection the findings table states the *invariant* ("What it checks"), so the row was not backwards — it just omitted `addressed_by_commit` (the G49 second resolver field). Fixed to name both resolver fields.
- **Commit sequencing under the policy pre-commit hook.** The hook runs `go test ./internal/policies/...` on every commit, so the rename-test (red until the rename + accepted decision both exist) was kept off-disk while the decision was created/accepted, then restored to land with the feature commit. This is RED-phase sequencing, not papering over a failure — the RED was demonstrated and recorded first.

