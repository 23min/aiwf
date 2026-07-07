---
id: G-0385
title: aiwf upgrade silently under-reports latest right after a fresh tag
status: open
discovered_in: E-0061
---
## What's missing

`aiwf upgrade`'s version check (`internal/version/version.go`'s `latestFor`) resolves "latest" by querying the Go module proxy's `/@v/list` endpoint and picking the highest tagged semver from the returned list — deliberately preferring it over `/@latest`, per the function's own comment, "to avoid a known proxy quirk where `/@latest` can be cached with a pre-tag pseudo-version answer and not refresh after the first tag lands." `/@latest` is consulted only as a fallback when `/@v/list` returns zero tagged versions (the no-tags-yet bootstrap case).

This trades one staleness failure mode for the opposite one: `/@v/list` and `/@latest` are cached independently by the proxy and do not converge at the same speed after a *new* tag lands. Reproduced directly against `proxy.golang.org` in the minutes after pushing the `v0.26.2` tag on this repo:

- `curl https://proxy.golang.org/github.com/23min/aiwf/@v/list` — returned versions only up to `v0.26.1`; `v0.26.2` absent.
- `curl https://proxy.golang.org/github.com/23min/aiwf/@latest` (same moment) — correctly returned `{"Version":"v0.26.2", ...}`, though the request itself took ~8s (a cold cache-miss against GitHub for the fresh tag).
- `aiwf upgrade --check` run twice in that window both reported `target: v0.26.1 (tagged)` — silently one version behind the real latest, with no timeout, no error, and no indication anything was stale.
- Re-running `aiwf upgrade` a few minutes later resolved correctly to `v0.26.2` once `/@v/list`'s cache caught up.

## Why it matters

This is a different failure shape from G-0181 (proxy lookup *timing out*, which is visible — the operator sees an error and knows to retry). Here the lookup *succeeds* and prints a normal, confident, wrong answer: "target: vPREVIOUS (tagged)" with no signal that a newer version already exists and is reachable. An operator upgrading in the minutes right after a release — arguably the single most likely moment someone runs `aiwf upgrade` — can reasonably conclude they're already current, or that the new release isn't published yet, when it is.

## Possible directions (not decided)

- Fall back to `/@latest` (or race both endpoints and take the higher result) whenever `/@v/list`'s highest version is *older* than what `/@latest` reports, not only when `/@v/list` is empty — narrowing the deliberate quirk-avoidance to the case it was actually meant to cover (no tags at all) rather than every case where the two endpoints disagree.
- At minimum, surface the ambiguity: if a caller can detect the two endpoints disagree (e.g. via an opt-in double-check), note it rather than silently trusting `/@v/list`.
- Related to G-0181 (proxy lookup timeout, no retry/fallback) but distinct: that gap is about the request failing outright; this one is about a successful request returning a stale-but-plausible answer.
