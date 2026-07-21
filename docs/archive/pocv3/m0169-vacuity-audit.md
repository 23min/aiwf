# Vacuity audit — kernel load-bearing units (M-0169)

Probe 2 of the G-0262 corpus work: the assertion-shape judgment `wf-vacuity`
does and mutation testing cannot — tautological assertions, over-narrowed
antecedents, and substring-not-structural checks. Directed (not whole-tree) over
the load-bearing units: the FSM and id allocator; the parsers/serializers
(frontmatter, trailers, slugs); the verb plans; the check rules; the renderers.

M-0168 (probe 1, gremlins) is the mechanical complement; its per-package
efficacy corroborates the "clean" verdicts below. This audit is LLM-judged, so
the claim is sized to that: a clean unit is "no weakness found," not "verified
correct."

## Weak assertions — found and strengthened

Each was strengthened and the strengthening confirmed: the targeted bug was
injected into the implementation, the new assertion went **red**, the bug was
reverted (tree byte-identical). The pre-strengthening form would have passed the
same injection — that gap is the finding.

### 1. `htmlrender_test.go` — AC anchor asserted by bare substring (substring-not-structural)

`TestRender_FixtureTree_FilesAndLinks` asserted `strings.Contains(mHTML,
id="ac-1")`. The milestone page is tabbed (`<section data-tab="manifest">`
wraps `<section class="ac" id="ac-1">`; the `#ac-1` fragment also appears as
`href` references in the build and tests tabs). A bare Contains passes even if a
bug renders the AC `<section>` under the *wrong* tab — exactly the CLAUDE.md
"substring assertions are not structural assertions" lesson.

**Strengthened** to scope the assertion to the manifest section via a
dependency-free `sectionByTab` helper (slice the document between
`<section data-tab="X">` boundaries), and to assert the anchor does *not* leak
into the overview/build/tests tabs.
**Injection:** rename the manifest tab to `overview` in `milestone.tmpl` → new
test red (manifest section missing); the old Contains stayed green (the `id`
still rendered, just in the wrong tab).

### 2. `verb/projection_test.go` — introduced finding asserted by `HasErrors` only (tautological)

`TestProjectionFindings_NewErrorIntroduced` introduces a milestone with an
unresolved parent and asserted only `check.HasErrors(got)` — "some error
surfaced," not *which*. A projection bug surfacing an unrelated finding, or one
scoped to the wrong entity, would pass.

**Strengthened** to pin that the introduced finding is the bad-parent ref on the
right entity: `Code == CodeRefsResolve && Subcode == "unresolved" && EntityID
== "M-0001"`.
**Injection:** blank the finding's `EntityID` at `check.go:548` → new test red;
`HasErrors` stayed green (still an error, just mis-scoped).

### 3. `entity/serialize_test.go` — modify-and-write asserted by substring (over-narrowed)

`TestSerialize_ModifyAndWrite` mutated `status` then asserted only
`Contains("status: in_progress")` + `Contains("body unchanged")`. It never
checks that the *untouched* fields (`id`, `title`, `parent`) survived the
marshal — a serializer that dropped one would pass.

**Strengthened** to round-trip the serialized bytes through `Parse`/`Split` and
pin the modified value, every untouched field, and the body.
**Injection:** drop the `parent` field from the entity's yaml tag → new test
red; the old status+body substrings stayed green (unaffected by the dropped
field).

## Clean — audited, assertions constrain behaviour

These units were probed and found well-constrained; M-0168's mutation efficacy
(probe 1) corroborates each:

- **FSM + id allocator** (`transition*`, `coded`, `allocate`, `canonicalize`):
  exhaustive enumeration over the legal state space, explicit negative and
  boundary cases, `cmp.Diff` against expected values — not existence checks.
- **Check rules** (`fsm_history_*`, `body_prose_id`, `acs`, `archive_*`,
  `isolation_escape`, …): assert the specific finding `Code`/`Subcode`/`EntityID`
  with paired positive and negative controls, not `len(findings) > 0`. (check
  efficacy 88.5%.)
- **Trailer parser** (`trailers_test`): `ToleratesUnknownFutureKeys` asserts both
  key *and* value; `CanonicalTrailerKeys` is pinned structurally to
  `trailerOrder`.

## Non-issues — candidates dispositioned, no change

Surfaced by the scan but judged adequate, with reason:

- `htmlrender` `href="E-NN.html"` link assertions — backstopped by
  `verifyLinkIntegrity` (structural link resolution over every rendered page).
- `htmlrender` markdown body (`<ul>` / `<li>...</li>` / `<pre>` / `fmt.Println`)
  — backstopped by negative assertions that raw markdown (` ```go `, `- item`)
  must *not* appear, confirming the fence was rendered.
- `htmlrender` page header `<h1>title</h1>` — a full-element match (tag + exact
  content), structural enough; location irrelevant.
- `entity` `TestSerialize_RoundTripACsAndTDD` — already a full `cmp` round-trip.
- `gitops` `ToleratesAbsentI25Keys` — the assertion's intent is absence
  tolerance; field values are covered by `RoundTripsThroughCommit`.

## Summary

3 weak assertions found and strengthened (all injection-verified red); the
load-bearing kernel is otherwise well-constrained, corroborated by M-0168's
mutation efficacy. The two probes are complementary: gremlins found boundary/
logic survivors in pure functions (M-0168); this pass found shape weaknesses a
mutation tool cannot — a substring standing in for a structural check, a
`HasErrors` standing in for a specific finding, a substring standing in for a
round-trip. Audit is LLM-judged: a clean verdict lowers risk, it does not
certify correctness.
