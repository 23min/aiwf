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

When M-0238 migrates the named bare-stderr call sites to the bound
logger, a site that logs a path-shaped value outside the three named
fields will silently bypass scrubbing — reintroducing the exact
home-directory leak this milestone exists to prevent, just under a
different field name. M-0238 should decide deliberately: export
scrubHomePaths for direct reuse, add handler-level scrubbing
middleware, or establish a per-call-site review discipline — not
default into the gap by omission.