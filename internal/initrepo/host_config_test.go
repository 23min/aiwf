package initrepo

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestArtifactEntryPoints_InvalidConfigLeavesRepositoryUntouched(t *testing.T) {
	t.Parallel()
	for _, content := range []string{"hosts: [other]\n", "hosts: [codex\n", "hosts: codex\n"} {
		for _, entry := range []string{"init", "refresh"} {
			for _, dryRun := range []bool{false, true} {
				t.Run(content+entry+map[bool]string{false: "write", true: "dry run"}[dryRun], func(t *testing.T) {
					t.Parallel()
					root := t.TempDir()
					path := filepath.Join(root, "aiwf.yaml")
					if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
						t.Fatal(err)
					}
					var err error
					if entry == "init" {
						_, err = Init(context.Background(), root, Options{ActorOverride: "human/test", SkipHook: true, DryRun: dryRun})
					} else {
						_, err = RefreshArtifacts(context.Background(), root, RefreshOptions{SkipHooks: true, DryRun: dryRun})
					}
					if err == nil {
						t.Fatal("invalid configuration accepted")
					}
					entries, err := os.ReadDir(root)
					if err != nil {
						t.Fatal(err)
					}
					var names []string
					for _, item := range entries {
						names = append(names, item.Name())
					}
					if diff := cmp.Diff([]string{"aiwf.yaml"}, names); diff != "" {
						t.Fatalf("partial setup (-want +got):\n%s", diff)
					}
					assertAgentsFile(t, path, content, 0o600)
				})
			}
		}
	}
}

func TestRefreshArtifacts_MissingConfigDoesNotBlockRefresh(t *testing.T) {
	t.Parallel()
	for _, dryRun := range []bool{false, true} {
		t.Run(map[bool]string{false: "write", true: "dry run"}[dryRun], func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			refresh, err := RefreshArtifacts(context.Background(), root, RefreshOptions{SkipHooks: true, DryRun: dryRun})
			if err != nil {
				t.Fatal(err)
			}
			if refresh.HookConflict {
				t.Fatalf("missing config blocked refresh: conflict=%v, err=%v", refresh.HookConflict, err)
			}
			found := false
			for _, step := range refresh.Steps {
				if step.What == "aiwf.example.yaml" && step.Action == ActionUpdated {
					found = true
				}
			}
			if !found {
				t.Fatalf("example refresh missing from ledger: %+v", refresh.Steps)
			}
			_, statErr := os.Stat(filepath.Join(root, "aiwf.example.yaml"))
			if dryRun && !errors.Is(statErr, fs.ErrNotExist) || !dryRun && statErr != nil {
				t.Fatalf("example after dryRun=%v: %v", dryRun, statErr)
			}
			if _, configStatErr := os.Stat(filepath.Join(root, "aiwf.yaml")); !errors.Is(configStatErr, fs.ErrNotExist) {
				t.Fatalf("refresh created config: %v", configStatErr)
			}
		})
	}
}
