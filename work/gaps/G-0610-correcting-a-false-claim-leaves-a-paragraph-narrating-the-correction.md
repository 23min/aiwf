---
id: G-0610
title: Correcting a false claim leaves a paragraph narrating the correction
status: open
priority: medium
---
## What's missing

The shipped rule "state the conclusion, not the drafting history" draws every
example from the drafting side — an earlier draft, this session, as of this
session — plus a set of code-comment smells. It names nothing for the case that
arrives whenever a record is corrected: **retracting a claim the body itself
carries**.

The shape, from a consumer-repo session where an assistant was correcting a
stale gap body:

> The 2026-06-12 owner update claiming this was already addressed was incorrect
> — the routing shortcut it names does not exist.

This is the same defect as "an earlier draft said X": it reads as current, it
rots, and the record it corrects already lives in `aiwf history` and `git
blame`. It arrives as a *claim* retraction rather than a *draft* reference, so
an author holding the rule does not map it onto any listed example and writes
the paragraph anyway. The operator caught it; the author did not.

The rule also states its exception — a past state a reader can still encounter —
and gives no way to apply it. There is no keep/drop test and no worked pair, so
the exception is a category an author either recognises or does not.

The comment half of this rule is settled and needs nothing: it is stated in the
guidance, carried in the code-review lens, and mechanically enforced by
`comment-history-attrition` at the push boundary and by its whole-tree sibling
in the policy suite. What is missing is on the entity-body and document side,
where no check runs and the prose is the whole gate.

## Why it matters

The remediation
[G-0595](G-0595-live-records-carry-claims-the-kernel-contradicts-and-no-check-sees-them.md)
tracks is a sweep of live bodies whose claims the kernel contradicts. Every
correction in that sweep is an occasion for this paragraph: the author has just
established that a sentence in the body is false, and the most natural way to
write the fix is to say so. The rule as written does not stop them, and the
sweep then trades a false claim for a narrated one.

The narrated form is worse than the false claim it replaces in one respect. A
false claim is a defect a later audit can measure against the kernel and
correct. A retraction paragraph is *true* when written, so no audit flags it; it
decays only into noise, one paragraph per correction, in the record a reader
consults to learn what is currently so.

## What the resolution is not

**Broadening a phrase grep.**
[G-0516](G-0516-comment-history-attrition-misses-stale-comments-outside-the-changed-hunk.md)
already records the limit for the sibling class: a phrase scan cannot decide
whether a factual claim is still true. `historyAttritionPhrases` is deliberately
under-inclusive for the same reason — phrasings are held out where the
historical and legitimate senses are indistinguishable without reading the
sentence. Retraction narration is stated in the present tense about a past
claim and carries no reliable marker. This class is review-held.

**Reframing the bullet.** Leading with the principle and demoting the
drafting-history items to examples is the reflex, and it buys nothing this gap
names: the principle is already stated, and the failure is a missing example
plus a missing test, not a misordered one. It also spends the most of the
resource that bounds any fix here.

## Resolution shape

The guidance fragment is always-on context in every consumer session, so every
line is paid on every turn. That cost decides the shape: a few lines appended to
the existing bullet earn their place; a rewritten bullet does not.

- Name the retraction case beside the drafting-history examples, with the
  instruction it needs: delete the false claim, state what is currently so, and
  put the provenance in `aiwf edit-body --reason`, which lands in the commit
  body and surfaces in `aiwf history`.
- Add the keep/drop test — *would a reader who never saw the earlier version
  need this sentence?* — with one keep and one drop example.
- Add a reviewer item for retraction narration in bodies and docs, beside the
  code-comment item `wf-review-code` already carries.

The reviewer item is the load-bearing one. D-0070 and D-0071 both put content
drift in shipped prose at review rather than behind a gate, and D-0072 states
what a green suite means under the resulting ban: the common spellings are
blocked, not that no such assertion can exist. A review lens naming this failure
mode is part of what that disposition is made of.

Two mechanics bind the edit itself:

- `PolicyM0211GuidanceOperatingAnchors` pins the substrings "not the drafting
  history" and "code comment" for this rule, matched verbatim within a single
  source line, so rewrapping the bullet reddens CI even when every word
  survives. Append rather than reflow, or move the anchor with the text. That
  policy is itself a prose-presence assertion over a shipped surface, the shape
  D-0070 otherwise retires; it survives because the ban scans test functions and
  this one is a production policy. The exemption is deliberate and the class is
  tracked in
  [G-0606](G-0606-prose-assertions-written-as-a-production-policy-escape-the-shipped-prose-ban.md),
  so the pin binds — it is not an oversight to route around.
- The `wf-review-code` edit needs no accompanying test.
  `PolicySkillEditProvenanceBackstop` asks only that the commit carry an
  `aiwf-entity:` trailer resolving to a real entity, which a patch branch owned
  by this gap produces. The move to avoid is writing a test that asserts the new
  sentence is present: `PolicyShippedProseAssertion` counts `embedded-guidance`
  and the ritual bodies among its shipped surfaces, so such a test is refused
  rather than merely discouraged.

## Prior threads

- [G-0516](G-0516-comment-history-attrition-misses-stale-comments-outside-the-changed-hunk.md)
  records why a phrase grep cannot close this class.
- [G-0526](G-0526-aiwf-ships-source-discipline-rules-as-prose-with-no-seam-to-enforce-them.md)
  asks whether aiwf should ship enforcement for source-discipline rules at all;
  this gap assumes the current answer — advisory prose — and improves the prose.
- [G-0595](G-0595-live-records-carry-claims-the-kernel-contradicts-and-no-check-sees-them.md)
  is the sweep during which this failure mode fires.
- E-0087 narrowed what a test may assert about shipped prose and scoped out what
  that prose may contain, which is this gap's subject; the boundary is drawn in
  its own Out-of-scope section.
