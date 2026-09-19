package check

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

func TestFSMHistoryConsistent_QuotedPathsRetainIllegalHistory(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		slug string
	}{
		{"unicode", "héllo"},
		{"quote", "has\"quote"},
		{"backslash", "has\\slash"},
		{"tab", "has\ttab"},
		{"record-separators", "has\x1e\x1fcontrols"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newRepoFixture(t)
			r.run("git", "config", "core.quotePath", "true")
			oldPath := "work/gaps/G-0001-" + tc.slug + ".md"
			if err := os.MkdirAll(filepath.Join(r.root, "work", "gaps"), 0o755); err != nil {
				t.Fatal(err)
			}
			var illegalSHA string
			for _, status := range []string{"addressed", "open"} {
				content := "---\nid: G-0001\ntitle: Quoted path\nstatus: " + status + "\n---\n"
				if err := os.WriteFile(filepath.Join(r.root, oldPath), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
				r.gitAddAll()
				illegalSHA = r.gitCommit("write gap status " + status)
			}

			for _, rename := range []bool{false, true} {
				if rename {
					r.run("git", "mv", oldPath, "work/gaps/G-0001-renamed-"+tc.slug+".md")
					r.gitCommit("rename gap")
				}
				tr, loadErrors, err := tree.Load(context.Background(), r.root)
				if err != nil {
					t.Fatal(err)
				}
				if len(loadErrors) != 0 {
					t.Fatalf("loading entity tree: %+v", loadErrors)
				}
				got := FSMHistoryConsistent(context.Background(), r.root, tr, nil, mustHead(t, r.root))
				if len(got) != 1 {
					t.Fatalf("renamed=%v: want one illegal-transition finding, got %+v", rename, got)
				}
				f := got[0]
				if f.Code != CodeFSMHistoryConsistent || f.Subcode != "illegal-transition" ||
					f.EntityID != "G-0001" || f.Severity != SeverityError {
					t.Errorf("renamed=%v: wrong finding: %+v", rename, f)
				}
				if !strings.Contains(f.Message, "addressed → open") || !strings.Contains(f.Message, illegalSHA[:7]) {
					t.Errorf("renamed=%v: finding does not identify the illegal transition and commit: %+v", rename, f)
				}
			}
		})
	}
}
