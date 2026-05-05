---
id: G-018
title: Contract-config validation is hook-only on `contract bind` and `add contract --validator …`
status: addressed
---

Resolved in commit `202a14a` (fix(aiwf): G18 — run contractcheck on contract bind / add+bind projection). Took the proposed approach: `ContractBind` and `Add`'s atomic-bind path now run `contractcheck.Run` on the projected `aiwf.yaml.contracts` config and surface any error-level findings whose `EntityID` matches the bound id, before mutating the doc. Catches missing-schema, missing-fixtures, and path-escape (G1) at verb time instead of push time. `contractverify.Run` (the actual validator execution) remains hook-only as a defensible carve-out — documented in `architecture.md` §3. Three new tests cover the verb-side enforcement; existing tests updated to pass a `bindRepo(t)` tmpdir with the referenced schema/fixtures present.

---
