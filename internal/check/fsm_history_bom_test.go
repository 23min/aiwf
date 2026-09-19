package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

func TestFSMHistoryConsistent_BOMDoesNotHideTransitions(t *testing.T) {
	t.Parallel()
	for _, newline := range []string{"\n", "\r\n"} {
		newlineName := "LF"
		if newline == "\r\n" {
			newlineName = "CRLF"
		}
		for _, tc := range []struct {
			name        string
			beforeBOM   bool
			afterBOM    bool
			next        entity.Status
			wantIllegal bool
		}{
			{name: "plain", next: entity.StatusDone, wantIllegal: true},
			{name: "BOM in parent", beforeBOM: true, next: entity.StatusDone, wantIllegal: true},
			{name: "BOM in child", afterBOM: true, next: entity.StatusDone, wantIllegal: true},
			{name: "BOM in both", beforeBOM: true, afterBOM: true, next: entity.StatusDone, wantIllegal: true},
			{name: "BOM-only change", afterBOM: true, next: entity.StatusProposed},
		} {
			t.Run(tc.name+"/"+newlineName, func(t *testing.T) {
				t.Parallel()
				r := newRepoFixture(t)
				rel := canonicalEntityPath("E-0001", entity.KindEpic)
				path := filepath.Join(r.root, rel)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				var changedSHA string
				for _, state := range []struct {
					status entity.Status
					bom    bool
				}{
					{entity.StatusProposed, tc.beforeBOM}, {tc.next, tc.afterBOM},
				} {
					content := "---\nid: E-0001\ntitle: Example epic\nstatus: " + string(state.status) + "\n---\n## Goal\nExample.\n"
					content = strings.ReplaceAll(content, "\n", newline)
					if state.bom {
						content = "\xef\xbb\xbf" + content
					}
					if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
						t.Fatal(err)
					}
					r.gitAddAll()
					changedSHA = r.gitCommit("record state")
				}
				tr, loadErrors, err := tree.Load(t.Context(), r.root)
				if err != nil || len(loadErrors) != 0 {
					t.Fatalf("loading current entity: %v, %v", err, loadErrors)
				}
				if tr.ByID("E-0001") == nil {
					t.Fatal("current entity missing")
				}
				findings := FSMHistoryConsistent(t.Context(), r.root, tr, nil, mustHead(t, r.root))
				var illegal []Finding
				for _, f := range findings {
					if f.Code == CodeFSMHistoryConsistent && f.Subcode == "illegal-transition" {
						illegal = append(illegal, f)
					}
					if f.Code == CodeFSMHistoryConsistent && f.Subcode == "history-walk-error" {
						t.Errorf("unexpected walk error: %+v", f)
					}
				}
				if !tc.wantIllegal {
					if len(illegal) != 0 {
						t.Fatalf("BOM-only edit produced illegal transition: %+v", illegal)
					}
					return
				}
				if len(illegal) != 1 {
					t.Fatalf("want one illegal transition, got %+v", findings)
				}
				f := illegal[0]
				if f.EntityID != "E-0001" || f.Path != rel || f.Severity != SeverityError ||
					!strings.Contains(f.Message, "proposed → done") || !strings.Contains(f.Message, changedSHA[:7]) {
					t.Errorf("wrong transition attribution: %+v", f)
				}
			})
		}
	}
}
