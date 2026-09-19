package skills

import (
	"errors"
	"testing"
)

func TestRenderCodexGuidance_RejectsInjectedReservedTokens(t *testing.T) {
	t.Parallel()
	_, err := RenderCodexGuidance("{{aiwf:unknown}}")
	if err == nil {
		t.Fatal("accepted unresolved token")
	}
	if !errors.Is(err, ErrUnknownRenderBinding) {
		t.Fatalf("error identity lost: %v", err)
	}
}
