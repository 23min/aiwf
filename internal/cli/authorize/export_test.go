package authorize

// RitualLocalBranchesForTest exposes the unexported ritualLocalBranches
// helper to the package's external tests (authorize_test package). The
// helper is the load-bearing piece of M-0102/AC-6's --branch completion:
// covering its branches in isolation lets the cobra-adapter stay a
// trivial wrapper.
var RitualLocalBranchesForTest = ritualLocalBranches
