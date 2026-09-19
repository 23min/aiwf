---
id: G-0699
title: Codex devcontainer rebuild persistence remains unverified
status: open
discovered_in: M-0343
---
## What's missing

Rebuild verification is absent for the Codex installation and persistent-state
mount in `.devcontainer/init.sh`, `.devcontainer/initialize.sh` and
`.devcontainer/devcontainer.json`. M-0343/AC-5 records passing isolated script
tests and a pre-rebuild baseline, but no rebuilt-container observation.

There is no completed rebuild to reproduce. Inspect M-0343's AC-5 validation
record for the original CLI versions, configuration and saved-session hashes,
login observation and outstanding checks. Its acceptance criterion retains the
checklist: npm installation despite an editor binary, repeat initialization,
configuration/session/login persistence, Claude usability and documentation
agreement. The baseline predates any future rebuild; changes made since it was
captured must be accounted for without reading or recording credential contents.

## Why it matters

The script tests cannot establish real bind-mount behavior or login persistence
across container replacement. Users could encounter a missing CLI or unavailable
state after rebuilding. Those outcomes remain unverified; no failure has been
observed. E-0093's local workflow support does not establish rebuild persistence.
