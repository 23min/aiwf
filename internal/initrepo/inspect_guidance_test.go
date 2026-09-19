package initrepo

import (
	"context"
	"errors"
	"testing"

	"github.com/23min/aiwf/internal/config"
)

func TestInspectGuidance_ValidatesHostAndCancellation(t *testing.T) {
	t.Parallel()
	if _, err := InspectGuidance(context.Background(), t.TempDir(), config.Host("unsupported"), nil); !errors.Is(err, config.ErrInvalidHost) {
		t.Fatalf("host error = %v", err)
	}
	for _, host := range []config.Host{config.HostClaudeCode, config.HostCodex} {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := InspectGuidance(ctx, t.TempDir(), host, nil); !errors.Is(err, context.Canceled) {
			t.Fatalf("%s cancellation = %v", host, err)
		}
	}
}
