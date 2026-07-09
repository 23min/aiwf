---
id: G-0389
title: aiwf show's not-found path ignores --format=json
status: open
discovered_in: M-0241
---
## What's missing

\`aiwf show <id> --format=json\` does not honor \`--format=json\` on its
not-found path: \`internal/cli/show/show.go\` line ~135 calls
\`cliutil.Errorf("aiwf show: %s not found\n", id)\` unconditionally,
emitting a plain-text message to stderr with empty stdout, rather than
the standard JSON envelope (\`{"status":"error","error":{"message":...}}\`)
every other verb's not-found path emits — confirmed directly for
\`promote\` and \`history\`, both of which correctly return a JSON
envelope for a missing id regardless of \`--format\`.

## Why it matters

A JSON consumer scripting against \`aiwf show --format=json\` (the
documented, supported machine-readable path) gets an empty stdout and
a non-JSON stderr message on a not-found id, instead of a parseable
error envelope — inconsistent with every other verb's contract and
liable to break naive JSON-parsing callers. Discovered while building
M-0241/AC-5's cross-worktree reachability-isolation scenario, which
deliberately probes \`aiwf show\` for an entity unreachable from the
current worktree — the harness had to special-case this verb's
not-found path (checking exit code only, not parsing JSON) to work
around it.