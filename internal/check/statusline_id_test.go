package check

// M-0227 AC-2: white-box tests for the statusline #-comment id scan. They
// live in package check to reach the unexported shellCommentMask and the
// shared scanMaskedForRealIDs — the same machinery the production walker
// uses, so the tests cannot drift from the rule's notion of "comment".

import (
	"strings"
	"testing"
)

// TestShellCommentMask pins that a real id fires only when it sits in a
// shell COMMENT, exercising every arm of the comment-detection rule:
// full-line comments (leading whitespace allowed), whitespace-preceded
// inline comments, and the shell forms where '#' is NOT a comment
// (parameter expansion, positional '$#', the shebang, and any id inside
// shell code). A canonical placeholder in a comment stays silent.
func TestShellCommentMask(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		src      string
		wantFire bool
	}{
		{"full-line comment fires", "# See G-0001 for detail\n", true},
		{"indented full-line comment fires", "    # See G-0001\n", true},
		{"space-preceded inline comment fires", "code=1  # note G-0001\n", true},
		{"tab-preceded inline comment fires", "code=1\t# note G-0001\n", true},
		{"comment with no trailing newline fires", "# tail G-0001", true},
		{"param-expansion hash is not a comment", "x=\"${line#G-0001}\"\n", false},
		{"double-hash expansion is not a comment", "y=\"${cached##G-0001}\"\n", false},
		{"positional-count hash is not a comment", "n=$#; echo G-0001\n", false},
		{"shebang carries no id", "#!/usr/bin/env bash\n", false},
		{"real id in shell code string is exempt", "echo \"G-0001\"\n", false},
		{"line with no hash is exempt", "label=\"epic G-0001 here\"\n", false},
		{"placeholder in comment is silent", "# See G-NNNN\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			masked := shellCommentMask([]byte(tc.src))
			got := scanMaskedForRealIDs(masked, "internal/skills/embedded-statusline/x.sh")
			if tc.wantFire && len(got) == 0 {
				t.Fatalf("expected a finding, got none\nsrc=%q\nmasked=%q", tc.src, masked)
			}
			if !tc.wantFire && len(got) != 0 {
				t.Fatalf("expected no finding, got %d: %+v\nsrc=%q\nmasked=%q", len(got), got, tc.src, masked)
			}
		})
	}
}

// TestShellCommentMask_PreservesShape pins the mask contract: same length,
// newline positions preserved (so downstream line numbers stay exact),
// comment text copied through, and shell code before an inline comment
// masked out.
func TestShellCommentMask_PreservesShape(t *testing.T) {
	t.Parallel()
	src := []byte("code=1  # note G-0001\n${x#y}\n")
	masked := shellCommentMask(src)

	if len(masked) != len(src) {
		t.Fatalf("length changed: got %d, want %d", len(masked), len(src))
	}
	for i := range src {
		if (src[i] == '\n') != (masked[i] == '\n') {
			t.Fatalf("newline position mismatch at byte %d: src=%q masked=%q", i, src[i], masked[i])
		}
	}
	if !strings.Contains(masked, "# note G-0001") {
		t.Errorf("comment text not preserved:\n%q", masked)
	}
	if strings.Contains(masked, "code=1") {
		t.Errorf("shell code before the comment should be masked out:\n%q", masked)
	}
	if strings.Contains(masked, "${x#y}") {
		t.Errorf("shell code line should be masked out:\n%q", masked)
	}
}
