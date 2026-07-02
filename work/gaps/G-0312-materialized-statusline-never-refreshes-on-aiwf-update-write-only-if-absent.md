---
id: G-0312
title: materialized statusline never refreshes on aiwf update (write-only-if-absent)
status: addressed
addressed_by_commit:
    - 8ad9ba0d
---
## What's missing

A way for `aiwf update` to **refresh** the materialized statusline in place. Every
other materialized artifact — the verb/ritual skills, the `aiwf-guidance.md`
fragment, the git hooks — is version-stamped and byte-refreshed on every `aiwf
update`. The statusline is the lone exception: `--statusline` is
write-only-if-absent (its `--help` says "writes only if absent; never clobbers an
existing copy"), so once scaffolded it goes stale and the only way to update it is
a manual file copy. A consumer who scaffolded the statusline weeks ago is stranded
on that version — missing every statusline improvement shipped since (health
glyph, CI aggregate, session HUD, usage dots), with no `aiwf`-native way forward.

## Why it matters

Install-once-never-update is precisely the drift `aiwf` exists to prevent, and the
asymmetry is surprising — every other artifact self-updates, so an operator
reasonably expects `aiwf upgrade` to carry the latest statusline too, and it
silently does not. The manual-copy escape is fragile, undiscoverable, and easy to
forget across releases.

## Fix direction

Treat the statusline like the `aiwf-guidance.md` fragment: stamp a version/marker
header into the embedded script; on `aiwf update`, refresh the materialized copy in
place when it carries the `aiwf` stamp (managed and unmodified), for both
`--scope project` and `--scope user`; keep the non-clobber guard only for a
stamp-broken (genuinely customized) copy. Open design point — **adopting an existing
stamp-less copy** (every install predating the stamp, including current ones): either
a one-time `--statusline --force` to overwrite-and-adopt, or recognize known past
shipped versions by content hash and upgrade them automatically. The script is
path-independent (no absolute paths), so a `--scope user` refresh is safe even when
`~/.claude` is a shared host/container bind mount. Customization, if ever needed,
moves to an `aiwf.yaml` knob rather than hand-editing the generated file — the same
pattern the guidance fragment uses.
