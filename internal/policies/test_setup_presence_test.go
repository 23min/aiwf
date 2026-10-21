package policies

import (
	"path/filepath"
	"testing"
)

func TestPolicy_TestSetupPresence(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyTestSetupPresence)
}

// TestPolicyTestSetupPresence_MessagesStateTheFix pins M-0337 AC-2: each
// of the policy's three findings says what to change.
func TestPolicyTestSetupPresence_MessagesStateTheFix(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for rel, content := range map[string]string{
		"internal/a/a_test.go":     "package a\n",
		"internal/b/b_test.go":     "package b\n",
		"internal/b/setup_test.go": "package b\n\nthis is not valid go @@@\n",
		"internal/c/c_test.go":     "package c\n",
		"internal/c/setup_test.go": "package c\n",
	} {
		mustWrite(t, filepath.Join(root, rel), content)
	}
	vs, err := PolicyTestSetupPresence(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"add one whose TestMain sets the four git identity variables",
		"fix the syntax error so its TestMain can be checked",
		"add one that sets the four git identity variables",
	} {
		mustHaveViolation(t, vs, want)
	}
}
