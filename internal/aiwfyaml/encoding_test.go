package aiwfyaml

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadBytes_RejectsNonUTF8BeforeCreatingEditor(t *testing.T) {
	t.Parallel()
	for _, raw := range [][]byte{
		{0xff, 0xfe, '{', 0, '}', 0, '\n', 0},
		{0xfe, 0xff, 0, '{', 0, '}', 0, '\n'},
	} {
		original := bytes.Clone(raw)
		doc, contracts, err := ReadBytes(raw)
		if err == nil || !strings.Contains(err.Error(), "save aiwf.yaml as UTF-8") {
			t.Errorf("expected actionable encoding refusal, got %v", err)
		}
		if doc != nil || contracts != nil {
			t.Error("unsupported encoding produced an editable document")
		}
		if !bytes.Equal(raw, original) {
			t.Error("refusal changed input bytes")
		}
	}
}

func TestReadBytes_AcceptsUTF8UnicodeWithOptionalBOM(t *testing.T) {
	t.Parallel()
	for _, prefix := range []string{"", "\ufeff"} {
		raw := []byte(prefix + "custom: café 世界\n")
		doc, _, err := ReadBytes(raw)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(doc.Bytes(), raw) {
			t.Error("reading changed UTF-8 input bytes")
		}
		if err := doc.SetGuidanceSelection(nil, []string{"ignored"}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(doc.Bytes()), "café 世界") {
			t.Fatalf("edit lost Unicode content: %s", doc.Bytes())
		}
	}
}
