---
id: G-0022
title: Provenance model extension surface
status: open
priority: low
---

The I2.5 provenance model ([`docs/design/provenance-model.md`](../../docs/design/provenance-model.md)) deliberately keeps the verb surface narrow. Six known extensions are filed here for future evaluation, all YAGNI for the PoC:

1. **Explicit scope end — delivered.** A human ends an in-flight scope with `aiwf authorize <id> --end`, which retires one named scope without moving its entity's status (ADR-0047). It ships as a mode of `authorize` rather than the `aiwf revoke` verb sketched here, and writes the `aiwf-scope-ends:` trailer the replay already reads — so the reserved `aiwf-revoked-by:` slot stays unused, and nothing further is owed on this item.
2. **Time-bound scopes (`--until <date>` or `--for <duration>`).** Auto-end on a wall-clock deadline. Adds a clock dependency to the kernel; not present today.
3. **Verb-set restrictions (`--verbs add,promote`).** Constrain which verbs an agent can invoke under a scope. Real safety win in adversarial settings; significant added complexity.
4. **Pattern scopes (`--pattern "M-007/*"`).** Scope by id pattern instead of (or in addition to) reference-graph reachability. More flexible; harder to verify; the "did the agent act outside scope?" question gets fuzzier.
5. **Sub-agent delegation.** Whether an `aiwf-verb: authorize` commit may itself be inside a scope (an agent authorizing another agent). The mutually-exclusive pair `(aiwf-verb: authorize, aiwf-on-behalf-of:)` is *not* enforced in I2.5; G22 owns the policy decision when real friction shows up.
6. **Bulk-import per-entity actor attribution.** `aiwf import` today writes one collapsed `aiwf-actor:` trailer for the whole import. When the source data carries per-row author info, the importer should write per-entity `aiwf-actor:` pairs instead. Solves the migration case where authorship is recoverable only via `git blame` on the v1 source.

Severity: Low. Each item is a clear extension path; the I2.5 model leaves room for all of them without architectural retrofits.

---
