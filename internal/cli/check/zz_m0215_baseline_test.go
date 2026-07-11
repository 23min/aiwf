package check

import (
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkAiwfCheckBaseline profiles one full `aiwf check` over the live
// kernel tree, for the M-0215 wall-time baseline (E-0053). It shells the
// same hundreds of git subprocesses a real pre-push check does, so it is
// skipped under -short. Run:
//
//	go test -run=^$ -bench=BenchmarkAiwfCheckBaseline -benchtime=1x \
//	  -cpuprofile=/tmp/check-cpu.prof ./internal/cli/check/
//
// A CPU profile that shows low total CPU against an ~85s wall time means
// the check is subprocess-wait bound (the 683 merge-base fan-out); high CPU
// in a Go function means there is an in-process hot path to attack too.
// This harness is M-0215-scoped scaffolding and is removed at wrap.
func BenchmarkAiwfCheckBaseline(b *testing.B) {
	if testing.Short() {
		b.Skip("M-0215 baseline profile shells many git subprocesses")
	}
	root := benchRepoRoot(b)

	// Silence the JSON findings dump so it neither floods output nor shows
	// up as terminal-write cost in the profile.
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = devnull.Close() }()
	saved := os.Stdout
	os.Stdout = devnull
	defer func() { os.Stdout = saved }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Run(root, "json", false, "", false, false, false, nil, "")
	}
}

func benchRepoRoot(b *testing.B) string {
	b.Helper()
	dir, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			b.Fatal("go.mod not found above test cwd")
		}
		dir = parent
	}
}
