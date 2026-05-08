# General policies — apply to any repo of this stack and scale

> Rules that aren't FlowTime-specific. They would fit, with minor wording changes,
> any polyglot repo with a .NET engine, a TypeScript UI, tests-as-contract, and CI.
> A good portion of these are the kind of thing a "starter pack" of policies could
> ship to any repo.

Each entry: rule, source, current enforcement rung, and where it could move.

---

## A. Code style and naming (mostly rung 2 today via `.editorconfig`)

### G-1. Private fields use camelCase without leading underscore
- **Source:** `CLAUDE.md` ("Coding Conventions"); `.editorconfig` (`dotnet_naming_rule.private_fields_camelcase.severity = error`).
- **Rung today:** 2 (Roslyn analyzer, error-level on build).
- **Subject:** style.
- **Notes:** Rare example of a *deviation* from the standard .NET convention (which uses `_camelCase`). The fact that it is loud at rung 2 is good — without that, Copilot/Claude will routinely revert to `_camelCase` from training data.

### G-2. Local variables use camelCase
- **Source:** `.editorconfig`.
- **Rung:** 2.
- **Subject:** style.

### G-3. File-scoped namespaces preferred
- **Source:** `.editorconfig` (`csharp_style_namespace_declarations = file_scoped:suggestion`).
- **Rung:** 1 (suggestion only; not enforced).

### G-4. Implicit usings + nullable enabled (.NET 9 / C# 13)
- **Source:** `CLAUDE.md`; project files (`Directory.Build.props`).
- **Rung:** 3 (compiler-enforced via project settings).

### G-5. Use `var` for built-in types and where the type is apparent
- **Source:** `.editorconfig` (suggestion).
- **Rung:** 1 (IDE suggestion).

### G-6. JSON payloads and schemas use camelCase; never reintroduce snake_case fields
- **Source:** `CLAUDE.md` ("JSON payloads and schemas use camelCase — do not introduce snake_case fields"). Also surfaced in dead-code report (telemetry-manifest schema is snake_case while model schema is camelCase — flagged as a needs-judgement question).
- **Rung:** 1 (prose). Could move to rung 2 with a JSON-Schema-driven check on serialized payloads.

### G-7. Markdown table cells with inline `|` must escape it as `\|`
- **Source:** `CLAUDE.md`.
- **Rung:** 1.
- **Notes:** Trivial to make rung 2 with a markdown-lint rule.

---

## B. Build / project / dependency hygiene (mostly rung 2-3 via tooling)

### G-8. Solution-wide build via `dotnet build FlowTime.sln`
- **Source:** `CLAUDE.md`; CI `build.yml`.
- **Rung:** 3 (build itself is the gate).

### G-9. Test before handing work back
- **Source:** `CLAUDE.md` ("Build and test before handing work back").
- **Rung:** 1 (prose). Wrap-milestone rituals enforce this conversationally; CI is the rung-2 backstop.

### G-10. Roslynator analyzer severities pinned per code
- **Source:** `.editorconfig` (specific tuning of RCS1163, RCS1058, RCS1090 to `none`; RCS1213, RCS1170, IDE0051, IDE0052 to `warning`).
- **Rung:** 2 (loaded only with `/p:RoslynatorAnalyze=true`; otherwise IDE-only).
- **Notes:** Interesting — the *policy* here is "these codes are noise on idiomatic patterns we use; these codes are high-signal." The rationale lives in a comment block. Good rung-1+rung-2 stacking.

### G-11. CI builds in `--configuration Release --no-restore`
- **Source:** `.github/workflows/build.yml`.
- **Rung:** 2.

### G-12. Each test project runs separately with `--blame-hang --blame-hang-timeout 60s`
- **Source:** `.github/workflows/build.yml`.
- **Rung:** 2.
- **Notes:** Reveals hidden policy: "no individual test should hang > 60s." Implicit. Could be elevated to rung 1 with prose.

### G-13. Performance / Benchmarks tests excluded from main test job
- **Source:** `build.yml` (`--filter "FullyQualifiedName!~Performance&FullyQualifiedName!~Benchmarks"`).
- **Rung:** 2.
- **Notes:** Implies a "Performance / Benchmark tests run on a different cadence" policy that is nowhere stated as prose.

### G-14. Build matrix targets `ubuntu-latest` only
- **Source:** `build.yml`.
- **Rung:** 2 (by absence of other matrix entries).
- **Notes:** "Cross-platform parity is not a CI gate" is implicit.

### G-15. Every milestone touching new dependencies must run `dotnet restore`
- **Source:** `build.yml` (`Restore` step), `CLAUDE.md`.
- **Rung:** 2.

---

## C. Test hygiene (rung 1 prose; rung 2 only where tests exist)

### G-16. TDD by default for logic, API, and data code (red → green → refactor)
- **Source:** `CLAUDE.md` ("TDD by default for logic, API, and data code — red → green → refactor"). Project-specific framing via `tdd_phase: red|green|refactor` AC field.
- **Rung:** 1 (prose); 2 via aiwf's per-AC `tdd_phase` tracking.
- **Notes:** Phase tracking is rare. The `tdd_phase` field is itself a candidate framework-level workflow primitive — see [03-policies-workflow.md](03-policies-workflow.md).

### G-17. Branch coverage required
- **Source:** `CLAUDE.md` ("every reachable conditional branch needs a test before declaring done; perform a line-by-line audit").
- **Rung:** 1 (prose; manual audit is the enforcement).
- **Notes:** "Line-by-line audit by the human or AI" is rung 0/1. Coverlet/coverage tools could move to rung 2.

### G-18. Unit tests must be fast and deterministic, no network or filesystem
- **Source:** `CLAUDE.md`.
- **Rung:** 1.
- **Notes:** A property-based or fuzz check could elevate; in practice nobody implements a "no-FS-in-unit-tests" rung 2 because it costs more than it earns.

### G-19. API tests use `WebApplicationFactory<Program>`, prefer real deps over mocks
- **Source:** `CLAUDE.md`.
- **Rung:** 1.

### G-20. UI work must be eval'd end-to-end in a real browser (Playwright)
- **Source:** `CLAUDE.md` ("UI testing (hard rule)" section).
- **Rung:** 1 (prose, called "hard rule"). The Playwright suite at `tests/ui/` is the enforcement infrastructure.
- **Notes:** Strongest test-coverage policy in the corpus. The framing — "type checks are necessary but not sufficient" — is exactly the design-space §11 SDD lesson: test the seam, not just the layer.

### G-21. Vitest covers pure logic; Playwright covers integration
- **Source:** `CLAUDE.md`.
- **Rung:** 1 (prose).

### G-22. Specs gracefully skip when infrastructure is down (health probe → skip)
- **Source:** `CLAUDE.md` ("Graceful skip when infrastructure is down").
- **Rung:** 1 → 2 (the pattern is implemented per-spec; could be a shared helper).

### G-23. Cover the critical paths: page load, one user interaction, reset/error path, key metric correctness
- **Source:** `CLAUDE.md` (UI testing section).
- **Rung:** 1 (prose). A Playwright spec count + path-coverage check could elevate; rarely worth it.

### G-24. Use invariant culture for parsing/formatting; tests deterministic
- **Source:** `CLAUDE.md`.
- **Rung:** 1 (prose). A grep for `CultureInfo.CurrentCulture` could be rung 2; not currently in place.

### G-25. Tool failure during a soft-signal audit is a finding, not a wrap blocker
- **Source:** `dead-code-audit/SKILL.md` ("Soft-signal contract: never mutates code, never fails the build, always exits 0").
- **Rung:** 1.
- **Notes:** Important meta-policy about *what kind of policy* this is. Worth lifting into the framework's policy taxonomy: "soft-signal" is a category.

---

## D. Documentation discipline (rung 1, with one rung-2 link-check action)

### G-26. Use Mermaid for diagrams (not ASCII art)
- **Source:** `CLAUDE.md`.
- **Rung:** 1.

### G-27. Repository language is English
- **Source:** `CLAUDE.md`.
- **Rung:** 1.

### G-28. No time or effort estimates in docs or plans
- **Source:** `CLAUDE.md`.
- **Rung:** 1.
- **Notes:** A grep for "hours / days / weeks / months" in `docs/` and `work/` could move this to rung 2. Surprisingly strong rule given how few teams enforce it.

### G-29. Sibling checkouts treated as read-only references unless instructed
- **Source:** `CLAUDE.md`.
- **Rung:** 1.

### G-30. `docs/` describes what IS (shipped, code-provable); `work/` is decided-next; `docs/archive/` and `docs/releases/` are historical; `docs/notes/` is exploration only
- **Source:** `CLAUDE.md` ("Truth Discipline > Truth classes").
- **Rung:** 1 (prose). The *truth precedence* (code > decisions > epic specs > arch docs > history) is a meta-policy about how policies themselves are ranked. **This is the most policy-shaped item in the corpus** — see [06-cross-cuts.md](06-cross-cuts.md) for a longer note.

### G-31. Don't restate a canonical contract in many places from memory; point to the owning doc
- **Source:** `CLAUDE.md` ("Truth Discipline > Guards").
- **Rung:** 1.

### G-32. Don't let one file simultaneously act as current reference and historical archive
- **Source:** `CLAUDE.md`.
- **Rung:** 1.

### G-33. Don't describe a target contract in present tense unless it is live
- **Source:** `CLAUDE.md`.
- **Rung:** 1.
- **Notes:** Tense-discipline is checkable with an LLM-as-linter pass; pure rung-1 today.

### G-34. Don't keep "temporary" compatibility shims without explicit deletion criteria
- **Source:** `CLAUDE.md`.
- **Rung:** 1.

### G-35. Don't treat aspirational docs as implementation authority
- **Source:** `CLAUDE.md`.
- **Rung:** 1.

### G-36. Doc badges (`doc-health`, `doc-correctness`) rendered from a script
- **Source:** `README.md` (badges); `scripts/render-doc-badges.sh`.
- **Rung:** 2 (badges read from a JSON; presumably regenerated).
- **Notes:** Worth pulling the script — rare example of a doc-quality numeric metric in a real repo.

---

## E. Repo / tooling hygiene

### G-37. Use `rg` and `fd` for searches; prefer precise edits; avoid broad refactors without context
- **Source:** `CLAUDE.md`.
- **Rung:** 1.

### G-38. Build outputs in `bin/`, `obj/`, `node_modules/`; outputs in `out/`, `data/`, `runs/`, `test-results/` — gitignored
- **Source:** `.gitignore`.
- **Rung:** 2.

### G-39. Never commit `.claude/worktrees/` (Claude Code agent isolation worktrees)
- **Source:** `.gitignore`.
- **Rung:** 2.
- **Notes:** Specific to AI-assisted workflows; transferable.

### G-40. Generated assistant adapter surfaces (`.github/copilot-instructions.md`, `.github/skills/`, `.claude/agents/`, `.codex/`) gitignored
- **Source:** `.gitignore`.
- **Rung:** 2.
- **Notes:** Defense-in-depth so generated content doesn't drift into git. Same shape as the framework's own `.gitignore` rule for `.ai/` etc.

### G-41. "Deprecated/removed directories — DO NOT RECREATE" comment-as-policy in `.gitignore`
- **Source:** `.gitignore` (`apis/` block with literal "DO NOT RECREATE" comment).
- **Rung:** 2 (gitignored, so creating them gets noticed) + rung 1 (the comment).
- **Notes:** Beautiful natural-language policy embedded as a gitignore comment. Recurring policy shape: "this used to exist, was killed deliberately, must not come back." Same shape as `work/guards/m-E19-02-grep-guards.sh`.

### G-42. Never blindly kill all processes on a port (devcontainer port-forwarder safety)
- **Source:** `CLAUDE.md` ("Devcontainer Port Safety").
- **Rung:** 1 (prose) + 2 (the `kill-port-8081` VS Code task is the safe-by-default mechanism).
- **Notes:** Specific to devcontainer-based dev. Strong feedback-from-incident style — "we lost a session this way, here's the rule."

### G-43. SIGTERM first, wait, SIGKILL only if still alive; never `kill -9` first
- **Source:** `CLAUDE.md`.
- **Rung:** 1.

### G-44. Verify processes before killing (`lsof -ti:PORT`, `ps aux | grep`, then `pkill -f "ProcessName"`)
- **Source:** `CLAUDE.md`.
- **Rung:** 1.

### G-45. Container worktrees go in `/workspaces/worktrees/`, never container-local storage (lost on rebuild)
- **Source:** `.claude/skills/devcontainer/SKILL.md`.
- **Rung:** 1.

### G-46. After worktree creation, always run `git submodule update --init --recursive`
- **Source:** `devcontainer/SKILL.md`.
- **Rung:** 1.

### G-47. Anything that must survive a container rebuild must be on a bind mount
- **Source:** `devcontainer/SKILL.md`.
- **Rung:** 1.

---

## F. Repo-shape policies

### G-48. `tests/` mirrors project names (e.g., `tests/FlowTime.Core.Tests`, `tests/FlowTime.Sim.Tests`)
- **Source:** `CLAUDE.md` ("Project Layout").
- **Rung:** 1 (prose). Could move to rung 2 with a tree-shape check.

### G-49. UI Playwright at `tests/ui/`, config at `tests/ui/playwright.config.ts`, specs in `tests/ui/specs/`, helpers in `tests/ui/helpers/`
- **Source:** `CLAUDE.md`.
- **Rung:** 1 (the convention itself); 2 (Playwright config validates structure).

### G-50. Schema-driven contracts live under `docs/schemas/`; one schema per data shape; index in `docs/schemas/README.md`
- **Source:** `docs/schemas/README.md`.
- **Rung:** 2 (validators reference these schemas; tests assert).

---

## Cross-cut observations on the general bucket

1. **The .editorconfig + CI combo gives this corpus its rare rung-2 floor for style.** Without it, every style rule would be at rung 1. The lesson generalizes: investing in a working linter once buys decades of cheap enforcement.
2. **Test discipline is asymmetric.** Code-quality tests (linting, formatting) are well-tooled at rung 2-3. Behavior-quality tests (UI eval'd in a browser, branch coverage, no-side-effects unit tests) are mostly rung 1 with implementation-side conventions.
3. **Doc hygiene is the largest rung-1 cliff.** All 11 of the truth-discipline rules in CLAUDE.md sit at rung 1. They are the *most policy-shaped* things in the repo (have a clear "why," supersede each other, get updated). Several would move to rung 2 cheaply: tense check, archive/current separation, "no time estimates" grep.
4. **"Defense in depth" is already a recurring pattern.** The framework gitignores `.ai/`/`.ai-repo/`/`work/epics/` to prevent consumer-only paths from leaking; FlowTime gitignores `.codex/`/`.github/copilot-instructions.md` for the same reason. Same shape, different lists. A *meta-policy* candidate: "every repo declares its own consumer/producer asymmetry."
5. **Several rules encode incident-shaped knowledge** (G-42 port-killing safety, G-39 worktree gitignore). These are exactly the "include the **why** so future-you can judge edge cases" memories that the auto-memory feedback type was designed for. The framework should have a place for incident-derived policies that doesn't bury them inside CLAUDE.md prose.
