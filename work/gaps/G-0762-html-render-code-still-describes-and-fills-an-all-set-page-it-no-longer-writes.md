---
id: G-0762
title: HTML render code still describes and fills an all-set page it no longer writes
status: open
priority: low
---
## What's missing

M-0099/AC-1 replaced the per-kind index's pair of pages, `<kind>.html` and
`<kind>-all.html`, with one page per kind. The render loop in
`internal/htmlrender/htmlrender.go` says so: "The pre-migration active/all-pair
design (M-0087/AC-6 + AC-7) collapses to a single emit under M-0099/AC-1", and
its only call passes `includeArchived` as `true` ("Pass includeArchived=true so
the resolver returns every entry"). The view model, the resolvers and their
comments still describe and fill the pair:

- `internal/htmlrender/pagedata.go`: `KindIndexLink.AllFileName`, "all-set
  per-kind index page, e.g. "gaps-all.html""; `KindIndexData`, documented as
  "Used by both the active-default page … and the all-set page", with
  `ActiveFileName`, `AllFileName` and `IncludeArchived`, and a comment naming a
  `SiblingFileName` the struct does not have.
- `internal/htmlrender/default_resolver.go` sets `AllFileName` to
  `<kind>-all.html`; its comments describe the "all-set (`<kind>-all.html`)
  pages", a title pair "Gaps" / "All gaps" that "together make the active/all
  distinction unambiguous", and the full epic set as reachable "via
  epics-all.html".
- `internal/cli/render/resolver.go` sets `AllFileName` the same way; its comments
  describe the "active-default and all-set pages", the title pair
  `"Gaps" (active) / "All gaps" (all-set)`, and the full epic set as living
  "on epics-all.html".
- `htmlrender.go`'s `PageDataResolver.KindIndexData` says `includeArchived` "is
  false for the active-default page and true for the all-set page", and
  `kindIndexKinds` that "`epics-all.html` provides the cross-kind consistency".

Because `includeArchived` is always `true`, both resolvers' `isArchived &&
!includeArchived` filter and the non-"All" branch of their title functions
(`titleForKindIndex`, `TitleForKindIndex`) never run in a render, and every
index page is titled "All …". No template reads the three fields:
`grep -rn 'AllFileName\|ActiveFileName\|IncludeArchived' internal/htmlrender/embedded/`
prints nothing.

Reproduced 2026-11-01 with aiwf built from `main` at `c06ffdaf9` (`vcs.modified=false`), git 2.51.1, Linux, in fresh repositories set up as `aiwf init`
leaves them, the first holding one gap, the second an active gap and a gap
promoted to `wontfix` and archived:

```
$ aiwf render --format=html --out site >/dev/null
exit 0
$ ls site
G-0001.html
adrs.html
assets
contracts.html
decisions.html
epics.html
gaps.html
index.html
status.html
$ ls site | grep -- '-all.html'
exit 1
$ grep -c -- '-all.html' site/*.html
site/G-0001.html:0
site/adrs.html:0
site/contracts.html:0
site/decisions.html:0
site/epics.html:0
site/gaps.html:0
site/index.html:0
site/status.html:0
exit 1
```

```
$ aiwf promote G-0002 wontfix >/dev/null && aiwf archive --apply >/dev/null
exit 0
$ aiwf render --format=html --out site >/dev/null
exit 0
$ grep -o -E '<title>[^<]*</title>|<h1>[^<]*</h1>|<a class="chip"[^>]*>[^<]*</a>|<tr data-archived="[a-z]*"' site/gaps.html
<title>All gaps</title>
<a class="chip" href="#sidebar-active" id="sidebar-active">Active</a>
<a class="chip" href="#sidebar-all" id="sidebar-all">All</a>
<h1>All gaps</h1>
<a class="chip" href="#active" id="active">Active</a>
<a class="chip" href="#all" id="all">All</a>
<tr data-archived="false"
<tr data-archived="true"
$ grep -h -E 'tr\[data-archived="true"\]' site/assets/*.css
.kind-index tbody tr[data-archived="true"] { display: none; }
body:has(#all:target) .kind-index tbody tr[data-archived="true"] { display: table-row; }
$ grep -o -E '<title>[^<]*</title>' site/epics.html site/decisions.html site/adrs.html site/contracts.html
site/epics.html:<title>All epics</title>
site/decisions.html:<title>All decisions</title>
site/adrs.html:<title>All adrs</title>
site/contracts.html:<title>All contracts</title>
```

The render writes one page per kind, no `-all.html`, and links to none. The gaps
page is headed "All gaps" above an "Active" / "All" chip strip, and its CSS hides
the archived row unless `#all` is targeted.

## Why it matters

Every per-kind index page is headed "All gaps", "All epics" and so on, while its
default view shows active entries only and offers "All" as the other view: the
heading names the view the reader is not looking at. In the code, the fields, the
filter and the title branch describe a two-page design that no render produces,
so a maintainer reading the view model learns the wrong design from it.

## Relates to

M-0099's "Deferrals" section records the same leftovers — these fields and the
resolver's `includeArchived bool` parameter "are dead-code paths after M-0099",
"the param is always passed `true`" — and says "Not filed as a gap — it's tracked
here"; M-0099 is done, and nothing took the follow-up. The structural-sweep audit
(`docs/audits/structural-sweep-audit.md`) lists the three fields under its item
B7, with other unread view-model fields this gap does not cover, and
`pagedata.go:38` among comments "asserting a parity that does not hold".
