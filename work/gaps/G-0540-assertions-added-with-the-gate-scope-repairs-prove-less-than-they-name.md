---
id: G-0540
title: Assertions added with the gate-scope repairs prove less than they name
status: open
---
## What's missing

The gate-scope repairs replaced three vacuous policies with assertions that fire.
Each was sabotage-verified against the defect it was written for, and each was
also reviewed as narrower than its name. The narrowings were accepted at the
commit gate to keep those changes patch-sized, and recorded nowhere.

- **The banner-source test pins a declaration site, not the banner.**
  `TestDiscoverabilityHaystack_BannerSourceDeclaresPrintHelp` asserts that the
  file the haystack reads is the file declaring `printHelp`. The channel the
  policy actually needs is the banner *text*. Today they coincide, because the
  text is a raw string literal inside `printHelp`. Move that literal to a const
  in a sibling file — the natural refactor once the banner grows — and the
  haystack silently loses the channel while the test stays green. That is the
  same silent-loss shape the test was written to prevent, one level up.

- **The dispatcher anti-orphan test is satisfied by a pair.**
  `TestPolicyApplyCallersAcquireLock_ScopeIsNotOrphaned` requires one dispatcher
  of each naming shape under the scanned prefix. A relocation that moved all but
  two dispatchers out would leave it green while the policy examined almost
  nothing. The stronger assertion is available and cheap: the mutating-verb set
  is already enumerated as the complement of `readOnlyVerbs`
  (`internal/policies/read_only.go`), so "every mutating verb package contributes
  an examined dispatcher" is one loop away.

- **Two switch arms are reachable but unpinned.** The zero-declarations and
  multiple-declarations arms of the banner-source test's switch produce distinct
  messages and can be reached, but no test drives them, and the diff-scoped
  coverage gate cannot see them: they live in a `_test.go`, and test files are
  absent from the coverage profile the gate intersects.

- **A test shape is now duplicated.** `_UnreadableRootErrors` appears in near
  identical form in `apply_callers_lock_test.go` and `sovereign_test.go` — body
  and doc comment alike. H1 makes the second copy the extract trigger; a table
  over `(name, policy func)` holds both, and the next policy to gain an
  anti-orphan test will want it.

## Why it matters

Three of the four are assertions about assertions: a test that cannot fail for
the case it names is the defect the repairs existed to remove, and shipping a
narrower one re-creates it at a smaller scale. The first is the sharpest, because
the refactor that defeats it — hoisting a growing string literal to a const — is
one a reader would make without ever considering the policy.

The accepted-at-the-gate reasoning holds: each narrowing kept a patch patch-sized,
and the alternative was scope that belonged in a milestone. What does not hold is
leaving them unwritten. They lived only in the review transcript, which no check
reads and no future session sees.

## Options

1. **Fix all four in one patch.** They sit in two adjacent files, three are
   mechanical, and the duplication extract is what makes the third cheap. The
   banner-text assertion is the only one needing thought — it wants a literal the
   haystack must contain, or a structural read of `printHelp`'s body.
2. **Fix the banner one, defer the rest.** It is the only one where a plausible
   refactor silently defeats the guarantee; the other three degrade only under a
   relocation that would be noticed for other reasons.
3. **Fold into E-0079.** That epic already re-aims the sovereign policy and adds a
   bijection check, so it touches the same tests. Risks widening an epic whose
   scope was just settled.

Option 1 is the lean. The set is small, bounded, and entirely within two files;
splitting it costs more in tracking than in fixing.

## Scope

Surfaced by the independent reviews of the two gate-scope patches, accepted as
deliberate narrowings at each commit gate, and filed rather than left in the
transcript. The repairs themselves are sound: every one was verified by
reproducing the defect it targets.
