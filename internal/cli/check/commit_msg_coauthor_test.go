package check

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestCheckRefusedCoauthor_Seam drives the IO shell against a real config
// file, so the wiring from aiwf.yaml to the verdict is covered and not just
// the decision above.
func TestCheckRefusedCoauthor_Seam(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		config   string
		block    string
		want     int
		wantMsg  bool
		wantAddr string
	}{
		{
			name:     "a listed address is refused",
			config:   "provenance:\n  refuse_coauthors:\n    - noreply@example.invalid\n",
			block:    "Co-Authored-By: Someone <noreply@example.invalid>\n",
			want:     cliutil.ExitFindings,
			wantMsg:  true,
			wantAddr: "noreply@example.invalid",
		},
		{
			name:   "an unlisted address passes",
			config: "provenance:\n  refuse_coauthors:\n    - noreply@example.invalid\n",
			block:  "Co-Authored-By: A Person <person@example.invalid>\n",
			want:   cliutil.ExitOK,
		},
		{
			name:   "a repo naming no address refuses none",
			config: "tree:\n  allow_paths: []\n",
			block:  "Co-Authored-By: Someone <noreply@example.invalid>\n",
			want:   cliutil.ExitOK,
		},
		{
			name:   "a repo with no config at all refuses none",
			config: "",
			block:  "Co-Authored-By: Someone <noreply@example.invalid>\n",
			want:   cliutil.ExitOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if tc.config != "" {
				if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(tc.config), 0o644); err != nil {
					t.Fatalf("writing config: %v", err)
				}
			}
			var stderr bytes.Buffer
			got := checkRefusedCoauthor(root, []byte(tc.block), &stderr)
			if got != tc.want {
				t.Errorf("checkRefusedCoauthor() = %d, want %d (stderr: %s)", got, tc.want, stderr.String())
			}
			if tc.wantMsg && !bytes.Contains(stderr.Bytes(), []byte(tc.wantAddr)) {
				t.Errorf("refusal does not name the address %q; got: %s", tc.wantAddr, stderr.String())
			}
			if !tc.wantMsg && stderr.Len() != 0 {
				t.Errorf("expected no output, got: %s", stderr.String())
			}
		})
	}
}

// TestRunCommitMsg_RefusesCoauthorSeam drives the hook entry point itself,
// so the ground is covered where it is wired rather than only where it is
// defined. A ground absent from runCommitMsg passes every test above.
func TestRunCommitMsg_RefusesCoauthorSeam(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := "provenance:\n  refuse_coauthors:\n    - " + seamRefusedAddr + "\n"
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	verbs := map[string]struct{}{"promote": {}}

	cases := []struct {
		name string
		msg  string
		want int
	}{
		{
			name: "a refused co-author is refused through the hook",
			msg:  "feat(x): a thing\n\nCo-Authored-By: Agent <" + seamRefusedAddr + ">\n",
			want: cliutil.ExitFindings,
		},
		{
			// The refusal must not swallow a message it has no quarrel with,
			// including one carrying a legitimate human co-author.
			name: "a human co-author rides through the hook",
			msg:  "feat(x): a thing\n\nCo-Authored-By: A Person <person@example.invalid>\naiwf-verb: promote\n",
			want: cliutil.ExitOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			code := runCommitMsg(writeMsg(t, tc.msg), root, verbs, &buf)
			if code != tc.want {
				t.Errorf("runCommitMsg() = %d, want %d; stderr = %q", code, tc.want, buf.String())
			}
			if tc.want == cliutil.ExitFindings && !bytes.Contains(buf.Bytes(), []byte(seamRefusedAddr)) {
				t.Errorf("refusal does not name the address; stderr = %q", buf.String())
			}
		})
	}
}

// seamRefusedAddr is an obviously-fictional address for the seam fixtures.
const seamRefusedAddr = "noreply@example.invalid"
