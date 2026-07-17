---
id: G-0414
title: Stale test naming in promote-on-wrong-branch-detection's real-binary test
status: open
priority: low
discovered_in: M-0257
---
## What's missing

`TestPromoteOnWrongBranchDetectionScenario_RealBinary_DetectsTheMisplacedActivation` (`internal/stresstest/promote_on_wrong_branch_detection_test.go`) — its name, doc comment, and failure message ("expected aiwf check to detect the misplaced activation commit") all describe verifying that `aiwf check` detects a wrong-branch activation that actually landed. G-0269's branch guard blocks that activation promote unconditionally today, so the misplaced-landing path this test's name and docs describe is never reached — the test passes because the guard's prevention holds and (as of M-0257) the broadened check-clean baseline is also satisfied, not because detection of a real misplaced landing fired. `internal/stresstest/promote_on_wrong_branch_detection.go`'s own restructuring and its `//coverage:ignore` comment on the still-gated `classifyPromoteOnWrongBranchDetection` call already document this accurately; the sibling test file's name, doc comment, and failure message don't.

## Why it matters

A reader encountering this test's name would reasonably conclude the scenario still exercises live wrong-branch detection, when today it exercises prevention plus the broadened check-clean baseline. Realigning the test's name, doc comment, and failure message to describe what it actually verifies — or restructuring it to force a genuine guard bypass so the original detection path is truly exercised — is future work for whoever next touches this scenario, deferred by M-0257 since the file itself is outside that milestone's diff.