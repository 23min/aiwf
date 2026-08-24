---
id: G-0629
title: Nothing fails when an m0127 allowlist entry stops matching the file it exempts
status: open
priority: medium
---
## What's missing

`m0127Pocv3AllowlistPaths` in `internal/policies/m0127_no_dangling_pocv3_refs_test.go:37`
exempts 24 exact repo-relative paths from the ban on `docs/pocv3` literals. Each key is a
path written into Go source, and nothing asserts that the key still names a file which
exists and still carries the literal. The consuming scan at `:133` does a map lookup and
returns early; a key matching nothing costs nothing and reports nothing.

Measured 2026-08-24 on `1d38c247c`, by reading each key and asking whether the file
exists and still contains `docs/pocv3` — **10 of the 24 keys resolve to no file**, and
none of the remaining 14 has lost the literal.

Two mechanisms produced them, both routine:

- **Nine by archiving.** ADR-0004 moves a terminal entity under a per-parent `archive/`
  subdirectory and the scan does `SkipDir` on any directory named `archive`, so the
  exempted file leaves the scan's reach and its key stops matching. E-0034's six files
  are now under `work/epics/archive/`, and G-0074, G-0075 and G-0092 under
  `work/gaps/archive/`.
- **One by retitle.** G-0439 is still open and still in scope. `aiwf retitle` re-derived
  its slug at `aad90fc40`, moving the file; a later `aiwf edit-body` at `8170cf712`
  removed the literal from the body. The entry is dead on both counts.

The same package already answers this for four other exemption ledgers, each of which
fails when an entry names something that no longer exists — the no-op claim-scope
ledger, the verb write-guard ledger, the shipped-prose exemption classes, and the
firing-fixture allowlist. The sibling allowlist in `m0290_retirement_surface_test.go`
gained the same guard alongside this gap. `m0127Pocv3AllowlistPaths` is the one left
without.

## Why it matters

An allowlist entry reads as coverage: it asserts that a named file's mention of the
banned literal was examined and judged deliberate, and it carries a rationale saying
why. When the key stops matching, that assertion is about nothing — and a reader
cannot tell a live entry from a dead one without resolving all 24 by hand, which is
what the list exists to spare them.

The list only grows. Adding an entry is one line at the moment a scan blocks a commit;
removing one requires noticing that a file moved months earlier, which nothing prompts.
So the next reader deciding whether an exemption is still warranted is reading a list
whose dead entries look exactly like its live ones.
