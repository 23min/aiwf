package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

func TestEpicTerminalGuards_LegacyParent(t *testing.T) {
	t.Parallel()
	for _, target := range []string{"cancel", "done", "cancelled"} {
		for _, force := range []bool{false, true} {
			name := target
			if force {
				name += "/force"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				r := newRunner(t)
				r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Parent", testActor, verb.AddOptions{}))
				r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Child", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
				r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
				path := filepath.Join(r.root, r.tree().ByID("M-0001").Path)
				before, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				legacy := strings.Replace(string(before), "parent: E-0001", "parent: E-001", 1)
				if writeErr := os.WriteFile(path, []byte(legacy), 0o644); writeErr != nil {
					t.Fatal(writeErr)
				}
				commitFixture(t, r.root, "test: load a legacy parent reference")
				tr := r.tree()
				if got := tr.ByID("M-0001").Parent; got != "E-001" {
					t.Fatalf("loaded parent = %q; want E-001", got)
				}
				var res *verb.Result
				wantCode := verb.CodeEpicPromoteNonTerminalChildren.ID
				if target == "cancel" {
					wantCode = verb.CodeEpicCancelNonTerminalChildren.ID
					res, err = verb.Cancel(r.ctx, tr, "E-0001", testActor, "test", force)
				} else {
					res, err = verb.Promote(r.ctx, tr, "E-0001", entity.Status(target), testActor, "test", force, verb.PromoteOptions{})
				}
				if code, ok := entity.Code(err); !ok || code != wantCode {
					t.Fatalf("Code(err) = (%q, %v); want (%q, true); err=%v, result=%+v", code, ok, wantCode, err, res)
				}
				if res != nil || !strings.Contains(err.Error(), "M-0001") {
					t.Fatalf("want no result and offending child M-0001; result=%+v, err=%v", res, err)
				}
			})
		}
	}
}
