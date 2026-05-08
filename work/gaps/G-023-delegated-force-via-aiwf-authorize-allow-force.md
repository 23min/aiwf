---
id: G-023
title: Delegated `--force` via `aiwf authorize --allow-force`
status: open
---

Per the I2.5 provenance model: `--force` is human-only. An LLM operating in a scope cannot `--force` even when the human has authorized that scope. The path is for the LLM to prompt the human, who then invokes `aiwf <verb> --force --reason "..."` directly.

This is the right default. But occasional friction is plausible: a long-running autonomous scope where every kernel-refusal-that-needs-overriding becomes a synchronous prompt to the human. The escape hatch would be a flag on `aiwf authorize` — `--allow-force` — which extends the agent's authorization to include forced acts within the scope. Even then, the trailer would still write `aiwf-principal: human/...` (the human authorized force-permitted scope), preserving the "sovereign acts trace to a named human" rule.

YAGNI for the PoC. The honest minimum-viable path forward is to ship I2.5 without it, watch where the friction lands, and revisit. If `--allow-force` ships, it's a flag-and-finding addition (`provenance-force-disallowed-in-scope` for misuse), not an architectural change.

Severity: Low. Specific named extension worth its own audit row so it doesn't get folded into G22 and lost.

---

<a id="g46"></a>
