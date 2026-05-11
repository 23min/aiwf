---
id: G-0113
title: rendered HTML site has no publish path; only viewable via local aiwf render
status: open
---
## What's missing

`aiwf render --format=html` produces the static governance site into a gitignored `site/` directory, but nothing in this repo carries that output anywhere a reader can open it. No CI workflow renders or deploys; no GitHub Pages target is configured; the `commit_output: true` recipe is documented but not used here. Looking at the current rendered tree requires cloning the repo, building the binary, running the verb, then opening a `file://` URL from a local directory. The `aiwf-render` skill's first follow-up — "Open `index.html` in a browser to confirm the render is what the user expected" — assumes the maintainer's laptop is the only consumer.

## Why it matters

The HTML render is one of the kernel's main human-readable surfaces — per-milestone tabs (Overview, Manifest, Build, Tests, Commits, Provenance), dependency edges, policy badges, force/audit chips — and its value compounds with how often it's looked at. As long as it's a private artifact behind a render command, the surface stays effectively invisible: reviewers don't open it, ADR readers don't open it, no one auditing "what's the current state of E-19?" sees it. Without a published render the maintainer also has no fast way to spot regressions in the rendered output between iterations, since the rendered output never leaves their machine for anyone else to look at. The cost of the missing publish path is paid silently on every governance question that gets answered from less-rich sources (raw markdown, `aiwf status`, git log) because the richer surface isn't reachable.

There is also a cadence question downstream of publishing: even if the site were published, the publish would happen only when a human remembered to push the deploy. Full automation (re-render on merge to trunk) is the natural follow-on but is distinct from the more basic "there is no publish path at all" problem this gap names.
