---
id: G-0625
title: lychee exclude_path entries are regexes, dropping nine docs from link-check
status: open
discovered_in: M-0317
---
## What's missing

`.lychee.toml`'s `exclude_path` entries are written as though they were directory
prefixes. lychee treats them as unanchored regular expressions matched against
the whole path, so eight Normative documents are silently absent from
link-check.

`lychee --help`: *"`--exclude-path <EXCLUDE_PATH>` Exclude paths from getting
checked. The values are treated as regular expressions."* Measured with
`lychee --config .lychee.toml --dump-inputs './**/*.md'` at `origin/main`
da34c1009, lychee 0.24.2 — the version `lycheeverse/lychee-action@v2` installs:

- `"work"` matches any path containing that substring — `docs/workflows.md`,
  `docs/working-paper.md`, `docs/design/legal-workflows-audit.md`,
  `docs/design/legal-workflows-first-principles.md`, and the accepted ADRs
  `ADR-0010`, `ADR-0011`, `ADR-0016` and `ADR-0023`, each carrying `work` in its
  slug.
- `".git"` has an unescaped dot, so it matches `<any>git` — which occurs inside
  "digits", removing the accepted `ADR-0008-canonicalize-kernel-ids-to-4-digits.md`.
  Verified causally: dropping `".git"` from the list makes that file reappear in
  `--dump-inputs`.

Nine files are dropped this way. Eight are Normative tier, five of those accepted
ADRs; the ninth, `docs/working-paper.md`, is Exploratory and so exempt on tier
grounds regardless. The entries whose intent is a directory —
`docs/explorations`, `docs/research`, `docs/archive`, `internal/check/testdata` —
happen to behave as intended because no other path contains those substrings, so
the list looks correct at a glance and is not.

`".git"` earns nothing even when it works: the workflow's `./**/*.md` glob never
descends into dot-directories, so no `.git/` path reaches lychee to be excluded.
Its only measurable effect is the ADR-0008 collateral.

## Why it matters

The exemptions the repo means to grant are tier-based and documented: the
Archival and Exploratory tiers are forget-by-default. Nothing grants one to
`docs/workflows.md` or to five accepted ADRs, which sit in the Normative tier
whose contents are supposed to be kept in lockstep with the code. Their links
are unchecked by accident of substring, and a break in any of them produces no
signal at all — not a late one, none.

None of the eight carries a `docs/`-to-`work/` link today, so no link is currently
missed. That is what makes this worth deciding now rather than after: the
exposure is a future link written into a file the author has every reason to
believe is checked.

Anchoring is the obvious fix and is not free. `'^work/'` and `'^\.git/'` restore
the eight to the checked set, and any broken link they already carry becomes a
new failure in a workflow that is red already — so the change wants to land with,
or after, the red being cleared. The alternative is to leave the entries as they
are and record that these files are outside link discipline, which at least stops
the list reading as something it is not.

Write an anchored entry as a TOML **literal** string, in single quotes. The basic
(double-quoted) form processes escapes, so `"^\.git/"` is not loadable TOML at
all — it fails with *"missing escaped value"* — and the spelling that does load,
`"^\\.git/"`, reaches lychee as `^\.git/` while any reader taking the raw bytes
sees a different regex. The literal form has no escapes to disagree about.

## Where to fix

- `.lychee.toml` — the `exclude_path` entries and whatever anchoring is chosen.
- `internal/policies/m0317_docs_link_coverage_test.go` — `firstMatch` models this
  matching so the coverage guard sees what lychee sees, and
  `parseLycheeExcludePaths` reads the entries. The parser accepts a literal
  string and refuses a backslash-bearing basic one, so the anchored spelling
  above parses and the misreadable one is rejected rather than misread — but a
  reader changing the entries should confirm that still holds.
