---
id: G-0383
title: Decide how call sites reuse WithVerb's home-path scrub
status: open
priority: high
discovered_in: M-0237
---
## What's missing

WithVerb scrubs macOS/Linux home-directory fragments only from the
three fields it explicitly binds (verb/entity/actor) — exactly what
M-0237's AC-4 asked for. scrubHomePaths itself is unexported, reachable
only through WithVerb. ADR-0017's Consequences section also commits to
a broader guarantee: "stack traces and full file paths log only at
debug level" — implying any call site that logs a path or an os.Args
element under a different key (e.g. logger.Debug("resolved config",
"path", cfgPath)) needs the same scrubbing discipline, but has no way
to reuse it.

## Why it matters

Any call site that logs a path-shaped value outside the three named
fields silently bypasses scrubbing — reintroducing the exact
home-directory leak ADR-0017 exists to prevent, just under a different
field name. The mechanism should be decided deliberately: export
scrubHomePaths for direct reuse, add handler-level scrubbing
middleware, or establish a per-call-site review discipline — not
default into the gap by omission.

## Deferred until a real call site needs it

M-0238 and M-0239 have both landed and E-0061 is closed. Checked
against every non-test logger call site in the tree: none logs a
path-shaped value outside the three fields WithVerb already scrubs, so
the scenario above hasn't occurred — the risk is real but currently
dormant, not an active leak. ADR-0017's other stated mitigation for
this case, restricting stack traces and full file paths to `debug`
level, already applies in the meantime.

The three-way fork stays open, but deciding it now would be
speculative: exporting scrubHomePaths and handler-level scrubbing
middleware are both machinery for a consumer that doesn't exist yet.
The trigger to revisit is a real call site that needs to log a
path-shaped value outside verb/entity/actor — not a calendar date or a
milestone number.