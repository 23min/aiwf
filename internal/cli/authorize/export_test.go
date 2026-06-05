package authorize

// RitualLocalBranchesForTest exposes the unexported ritualLocalBranches
// helper to the package's external tests (authorize_test package). The
// helper is the load-bearing piece of M-0102/AC-6's --branch completion:
// covering its branches in isolation lets the cobra-adapter stay a
// trivial wrapper.
var RitualLocalBranchesForTest = ritualLocalBranches

// CurrentBranchForTest exposes the unexported currentBranch helper.
// M-0103 / AC-1 + AC-3: the helper is the input to the verb-layer
// preflight's implicit-ritual-context signal. Covered in isolation so
// the CLI's RunE plumbing can stay a thin wrapper.
var CurrentBranchForTest = currentBranch

// BranchExistsForTest exposes the unexported branchExists helper.
// M-0103 / AC-2 + AC-4: distinguishes "no --branch passed" from
// "--branch <name> refers to a missing branch" in the verb's preflight.
var BranchExistsForTest = branchExists
