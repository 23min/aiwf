# Verb design

## Principles

- **Errors are findings, not parse failures.** `aiwf check` loads inconsistent state and reports it; it never refuses to start. Validation is a separate axis from loading.
- **Kernel functionality must be AI-discoverable.** Every verb, flag, JSON field, body-section name, finding code, trailer key, and YAML field is reachable via `aiwf <verb> --help`, embedded skills under the selected host’s `.claude/skills/aiwf-*` or `.agents/skills/aiwf-*`, or the design docs. If an AI must grep source to learn a capability, it's undocumented — ship `--help` + skill docs alongside the implementation.
- **CLI surfaces must be auto-completion-friendly.** Every verb, subverb, flag, and closed-set value is reachable via tab-completion (Cobra `ValidArgs` / `RegisterFlagCompletionFunc`; entity ids enumerate dynamically). The drift test `internal/cli/integration/completion_drift_test.go` fails CI on a verb/flag added without completion wiring.

## Commit trailers

Every mutating verb writes trailers so `aiwf history` can render timelines:

```
aiwf-verb: promote
aiwf-entity: M-NNNN
aiwf-actor: human/peter
```

## Title length

- **`entities.title_max_length` in `aiwf.yaml`** caps title (at `add`/`retitle`/`import`) and slug (at `rename`); default 80. Hard-reject, not truncate — the error points at `--body-file`. Title and slug share the budget so filenames and frontmatter stay in sync. Pre-cap titles are grandfathered; clean up with `aiwf retitle` (re-derives the slug), which the check treats as a git rename so a retitle batch pushes cleanly. Re-derivation applies only while the slug still tracks the title — one set deliberately with `aiwf rename` is preserved, so `rename`'s effect outlives the next retitle rather than lasting until it ([the retitle ADR](../../docs/adr/ADR-0037-retitle-re-derives-the-slug-only-while-it-tracks-the-title.md)).

## Designing a new verb

The design isn't done until you can answer **"what verb undoes this?"** Acceptable: another invocation of the same verb; an explicit terminal transition (`aiwf cancel`, `aiwf reallocate`); "you can't, deliberately — here's why" (`aiwf init` is one-shot; written down); "you'd open a new entity for the inverse." Not acceptable: "we'll figure it out later." See [docs/design/design-lessons.md](../../docs/design/design-lessons.md) §"On reversal".

### Same-state convergence — resolve, then converge

A mutating verb handed input that already equals current state returns a NoOp (exit 0, no commit), not an error. Two rules decide it, **in order**:

- **R1 — resolve before you converge.** Every argument must name something real — an existing entity, a declared area member, an AC, a working-copy edit — *before* the verb asks whether the request is already satisfied. An unresolvable argument is a refusal, never a NoOp. Converging there asserts success for state that cannot exist and buries the operator's typo (`aiwf rename-area <undeclared> <undeclared>` must refuse, not report "already named").
- **R2 — converge when the verb's intended effect already holds.** "Effect" is the verb's own declared target, which is not always a single field: stored bytes for the field-mutation verbs, a *terminal disposition* for `cancel` (any terminal converges, not only the one `cancel` would have picked), an existing exemption record for `acknowledge illegal`, the committed body for `edit-body --body-file`.

Two corollaries worth stating, because getting either wrong loses work rather than merely annoying someone:

- **A NoOp claims the verb's effect already holds, so every surface that effect spans must hold.** The baseline is the **loaded tree**: `tree.Load` reads the working tree, so the loaded entity is what every verb and every check rule sees, and a structured-state verb comparing it is comparing current state rather than a stale projection. (That the loaded tree is the *baseline* does not mean `aiwf check` validates every property of it — a `parent:` disagreeing with the file's directory, a title carrying a newline, and a drifted canonical H1 all pass at 0 errors. Being the baseline and being validated are separate claims.) A verb whose effect spans more than one surface checks each: `retitle` compares the stored title *and* the body H1, so a drifted heading still gets repaired rather than converged over; `move` compares the `parent:` field *and* the file's location under the epic's directory; a `--superseded-by` re-run does not converge while the reciprocal back-link is still missing. `edit-body --body-file` additionally compares HEAD, because it is a *commit-this-content* verb whose supported input includes "the content is already on disk" — a disk match alone would not prove the content is committed. That HEAD conjunct is verb-specific reasoning rather than a rule to copy into every guard — but not because the converging path is safe. Measured, a loaded-only comparison against HEAD-divergent frontmatter fails in both directions: asking for the value already on the dirty disk reports "already set; nothing to change" while HEAD says otherwise, dropping the operator's mutation behind a message that is false about the record; and asking for HEAD's value commits a tree byte-identical to its parent. Whether a verb should run at all against uncommitted frontmatter is a **precondition** question rather than a convergence one, and it belongs at one shared seam ahead of the guards instead of duplicated into each — where that seam sits relative to the same-state comparison decides whether any conjunct is needed at all. That seam is two seams, because one placement cannot cover both cases ([the HEAD-divergence ADR](../../docs/adr/ADR-0038-refuse-verb-writes-over-head-divergent-entity-content.md)): a commit-side guard at `verb.Apply` covers every path a plan carries, including nested entities no verb named, and a claim-side guard in each verb's prelude covers the converging case, which returns before a plan exists and so never reaches `Apply`. `PolicyClaimGuardPresence` fails CI on a converging route that reaches neither, and `noOpClaimScopes` records what each verb's claim is scoped to. The multi-entity sweeps decline per candidate rather than refusing per verb, so one mid-edit file costs its own candidate and not the sweep.
- **Same-state is not the same as same-spelling.** Compare ids canonicalized, since narrower legacy widths name the same entity. Compare *content* byte-for-byte, since a write that changes stored bytes is a real change.

A verb whose input cannot be checked against reality is outside R2 entirely: `aiwf edit-body` in bless mode is handed no target — its input *is* the working copy — so it cannot distinguish "I meant to change nothing" from "my editor never saved", and refusing is the only honest answer.

**Composite ids follow the same rules at sub-entity granularity.** `promote <id>/AC-N` and `cancel <id>/AC-N` converge exactly as their entity-level counterparts do, against the AC FSM's own notion of terminal — which is not the entity FSM's: `met` has outgoing edges and is *not* terminal, so cancelling a met AC still does real work. R1 binds hardest here, because a sub-element's state can be edited onto disk without passing a verb: a status the FSM does not recognize is not terminal, so it must reach the FSM consult and be refused rather than converged or silently written.

Mechanical companion: `internal/policies/verb_result_noop_invariant.go` — note it scans *exported* entry points, so an unexported composite branch is invisible to it and needs its own test. [The same-status ADR](../../docs/adr/ADR-0036-same-status-fsm-transitions-converge-to-noop-not-refusal.md) settles the FSM-transition case (`promote`/`cancel`, at both granularities) specifically; this section is the general convention.
