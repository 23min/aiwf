package gitops_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/gitops"
)

func TestBulkRevwalk_PreservesQuotedPaths(t *testing.T) {
	t.Parallel()
	paths := []struct {
		name string
		path string
	}{
		{"plain", "plain.md"},
		{"spaces", " space name .md"},
		{"unicode", "héllo-日本.md"},
		{"quote", "has\"quote.md"},
		{"backslash", "has\\slash.md"},
		{"tab", "has\ttab.md"},
		{"newline", "has\n===AIWF-REC===\n===AIWF-PATHS===\n.md"},
		{"controls", "has\a\b\v\f\r\x1e\x1f\x7f.md"},
		{"non-UTF8", "has\xff.md"},
		{"non-UTF8 with tab", "has\xff\t.md"},
	}
	for _, quotePath := range []string{"true", "false"} {
		t.Run("quotePath="+quotePath, func(t *testing.T) {
			t.Parallel()
			for _, tc := range paths {
				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()
					ctx := context.Background()
					root := t.TempDir()
					if err := gitops.Init(ctx, root); err != nil {
						t.Fatal(err)
					}
					if err := runGit(ctx, root, "config", "core.quotePath", quotePath); err != nil {
						t.Fatal(err)
					}
					writeAndCommit(t, ctx, root, tc.path, "first\n", "add", nil)
					addSHA := headSHA(t, ctx, root)
					firstBlob := blobID(t, ctx, root, addSHA, tc.path)
					writeAndCommit(t, ctx, root, tc.path, "second\n", "modify", nil)
					modifySHA := headSHA(t, ctx, root)
					secondBlob := blobID(t, ctx, root, modifySHA, tc.path)
					newPath := "renamed-" + tc.path
					if err := gitops.Mv(ctx, root, tc.path, newPath); err != nil {
						t.Fatal(err)
					}
					if err := gitops.Commit(ctx, root, "rename", "", nil); err != nil {
						t.Fatal(err)
					}
					renameSHA := headSHA(t, ctx, root)
					if err := os.Remove(filepath.Join(root, newPath)); err != nil {
						t.Fatal(err)
					}
					if err := gitops.Add(ctx, root, newPath); err != nil {
						t.Fatal(err)
					}
					if err := gitops.Commit(ctx, root, "delete", "", nil); err != nil {
						t.Fatal(err)
					}
					deleteSHA := headSHA(t, ctx, root)
					got := map[string][]gitops.PathTouch{}
					for _, rec := range collectRecords(t, ctx, root) {
						got[rec.Commit] = rec.Paths
					}
					const zero = "0000000000000000000000000000000000000000"
					want := map[string][]gitops.PathTouch{
						addSHA:    {{Status: "A", Path: tc.path, PreSHA: zero, PostSHA: firstBlob}},
						modifySHA: {{Status: "M", Path: tc.path, PreSHA: firstBlob, PostSHA: secondBlob}},
						renameSHA: {{Status: "R", SrcPath: tc.path, Path: newPath, PreSHA: secondBlob, PostSHA: secondBlob}},
						deleteSHA: {{Status: "D", Path: newPath, PreSHA: secondBlob, PostSHA: zero}},
					}
					if diff := cmp.Diff(want, got); diff != "" {
						t.Errorf("history paths (-want +got):\n%s", diff)
					}
				})
			}
		})
	}
}
