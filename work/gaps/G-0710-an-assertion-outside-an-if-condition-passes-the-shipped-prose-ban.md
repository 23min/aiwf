---
id: G-0710
title: An assertion outside an if condition passes the shipped-prose ban
status: open
discovered_in: M-0333
---
## What's missing

The shipped-prose engine in `internal/policies/shipped_prose_assertion.go` treats a call as an assertion only when it stands in the condition of an `if` whose arm fails the test (`decidingCalls`, used by `assertions`). A call in any other deciding position is taken for narrowing, not an assertion, and is reported only when the same test also asserts in an `if`. Three positions pass `shipped-prose-assertion`:

- an assertion helper called as a statement, `mustContain(t, doc, "…")`;
- a `switch` case, `switch { case !strings.Contains(doc, "…"): t.Error(…) }`;
- a verdict bound in an `if`'s init statement, `if ok := strings.Contains(doc, "…"); !ok { t.Error(…) }`.

D-0072 records the engine as recognizing "an assertion helper where the document is not the first argument"; it does so only when the helper call sits in an `if` condition.

Measured in the devcontainer (`go1.25.11 linux/amd64`) at commit `525af4bd2`: a throwaway test in `internal/policies` fed each shape, over a shipped skill read through the fixture reader, to `detectProseAssertions` through `parseSyntheticPackage`, with `has` a helper returning the verdict and `mustContain` one asserting it. Expected one finding per shape; observed:

```
control-if       findings=1
helper-in-if     findings=1
helper-statement findings=0
switch-case      findings=0
if-init          findings=0
```

## Why it matters

The statement helper is the form a repeated assertion takes once it is extracted, as H1 asks, and the engine's own comment calls `mustContain(t, body, "…")` "the dominant assertion-helper signature". A new pin over a shipped surface written that way passes the blocking ban, among the spellings D-0072 counts as blocked.
