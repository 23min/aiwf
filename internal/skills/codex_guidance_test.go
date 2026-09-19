package skills

import (
	"bytes"
	"errors"
	"testing"
)

func TestRenderCodexGuidance_UsesCanonicalInstructionsAndSelectedPaths(t *testing.T) {
	t.Parallel()
	const ver = "test-native-version"
	source := GuidanceBytes()
	if bytes.Count(source, []byte("{{aiwf:templates_dir}}")) == 0 {
		t.Fatal("fixture has no host-sensitive references")
	}
	want := bytes.ReplaceAll(source, []byte(guidanceVersionSentinel), []byte(ver))
	want = bytes.ReplaceAll(want, []byte("{{aiwf:templates_dir}}"), []byte(CodexTarget().TemplatesDir))
	got, err := RenderCodexGuidance(ver)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("native guidance differs from resolved canonical source")
	}
}

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
