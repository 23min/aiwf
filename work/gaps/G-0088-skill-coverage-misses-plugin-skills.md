---
id: G-0088
title: Skill-coverage policy doesn't police plugin skills under aiwf-extensions/
status: open
discovered_in: M-0079
---
`internal/policies/skill_coverage.go`'s `PolicySkillCoverageMatchesVerbs` (added in M-0074) walks `internal/skills/embedded/` exclusively. Plugin-side skills under `aiwf-extensions/skills/aiwfx-*` are not policed by the kernel — the policy doesn't see them, and any drift in a plugin skill (name doesn't match directory, description empty, broken `aiwf <verb>` references in body prose) goes undetected by `go test ./internal/policies/`.

M-0079's AC-7 hit this directly. The acceptance criterion read *"M-0074 skill-coverage policy or plugin equivalent accepts the skill"* — the spec anticipated the gap and offered an escape valve: if M-0074's policy is kernel-only, satisfy the AC by re-applying the equivalent invariants test-side rather than expanding M-0074's scope mid-milestone. The test that was added (`TestAiwfxWhiteboard_AC7_SkillCoveragePolicyEquivalent` in `internal/policies/aiwfx_whiteboard_test.go`) re-implements the same invariants the kernel policy enforces: name matches directory, description non-empty, every backticked `aiwf <verb>` mention resolves to a registered top-level Cobra verb. Red was verified by mutating the fixture to include `aiwf bogus-verb`.

## What's missing

**Per-plugin re-application of skill-coverage invariants.** Each new plugin skill author must add their own equivalent-invariants test (or accept that the kernel's check doesn't see their skill). M-0079 paid this cost once for `aiwfx-whiteboard`. Future plugin skills under `aiwf-extensions` (and any third-party rituals plugins consumers might author) face the same per-skill duplication.

The failure mode this leaves uncovered: a plugin skill ships with a broken `aiwf <verb>` mention, no equivalent-invariants test catches it, the operator hits the broken reference at invocation time. CI's drift-prevention story for kernel verbs and embedded skills (the completion-drift test in `cmd/aiwf/completion_drift_test.go` and M-0074's policy) does not extend to plugin skills.

## Why it matters

CLAUDE.md's load-bearing principle is *"the framework's correctness must not depend on the LLM's behavior"*, with the corollary *"kernel functionality must be AI-discoverable"* — both invariants enforced by mechanical chokepoints. The skill-coverage policy is one of those chokepoints. Today it covers only half the skill surface.

Three failure modes the per-plugin re-application leaves exposed:

1. **Drift between kernel rules and plugin re-implementations.** If M-0074's policy gains a new invariant (e.g., *"frontmatter `description` must contain at least one verb name in backticks"*), the kernel-side test gains it automatically; every plugin skill's hand-rolled test-side re-implementation is silently behind until each is updated.
2. **Forgotten re-application.** A plugin skill that ships without an equivalent-invariants test has zero coverage. Reviewers spotting the omission requires noticing the absence; presence-of-absence is a weaker signal than presence-of-finding. The chokepoint discipline this gap captures is *"every plugin skill, like every kernel verb, has a kernel-policed coverage rule."*
3. **Unbounded scope of "kernel-equivalent" wording.** The test naming convention (`TestAiwfxWhiteboard_AC7_SkillCoveragePolicyEquivalent`) communicates intent but isn't structurally tied to M-0074's policy. If M-0074 evolves, plugin tests don't track unless someone manually re-checks each one.

## Fix shape

Two paths, not exclusive:

1. **Extend `PolicySkillCoverageMatchesVerbs` to walk plugin skill directories.** The policy gains a list of plugin-skill roots (resolvable at runtime via `aiwf doctor`'s plugin-dir discovery, or declared in `aiwf.yaml.doctor.recommended_plugins`). The same invariants apply uniformly across kernel and plugin skills; plugin authors don't write boilerplate equivalent-invariants tests.
2. **Codify the per-plugin re-application as a documented helper.** A `policies.SkillCoverageInvariantsFor(skillPath string)` helper exposes the same rule set as a callable from plugin tests; plugin authors call the helper rather than re-implementing. CLAUDE.md gains a §"Plugin skill coverage" rule that names the helper as the canonical test surface.

(1) is the kernel-pure fix that earns the *"framework correctness must not depend on consumer discipline"* standard. (2) is the lighter-weight fix that keeps the plugin/kernel layer separation but documents the obligation. Either one resolves this gap; ideally both ship — (1) for embedded plugins (`aiwf-extensions`, `wf-rituals`), (2) for third-party rituals plugins outside aiwf's release cadence.

## References

- `internal/policies/skill_coverage.go` — `PolicySkillCoverageMatchesVerbs`, the policy whose scope this gap names.
- **M-0074** — milestone that introduced the policy.
- **M-0079** work log AC-7 — the cycle where the limitation surfaced; the test-side equivalent it produced (`TestAiwfxWhiteboard_AC7_SkillCoveragePolicyEquivalent`) is the prototype the helper in fix-shape (2) would generalize.
- CLAUDE.md *"What aiwf commits to"* §5 — framework correctness must not depend on LLM/consumer discipline.
- CLAUDE.md *"Engineering principles"* §"Kernel functionality must be AI-discoverable" — the principle this gap's load-bearing edge violates.
- **ADR-0006** — Skills policy: per-verb default; topical multi-verb when concept-shaped; no skill when `--help` suffices. Defines the kernel-side coverage expectation that this gap extends to the plugin side.
