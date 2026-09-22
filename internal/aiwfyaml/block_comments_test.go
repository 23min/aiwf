package aiwfyaml

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfigurationEdits_PreserveSeparatedSiblingComments(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"guidance", "hooks", "contracts"} {
		t.Run(key, func(t *testing.T) {
			t.Parallel()
			const suffix = "# first\n\n# last\n\n\nother: value\n"
			doc, _, err := ReadBytes([]byte("hosts: []\n" + key + ": {}\n\n" + suffix))
			if err != nil {
				t.Fatal(err)
			}
			switch key {
			case "guidance":
				err = doc.SetGuidanceSelection(nil, []string{"ignored"})
			case "hooks":
				doc.SetHooks(map[string]bool{"test-hook": true})
			case "contracts":
				err = doc.SetContracts(&Contracts{})
			}
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasSuffix(string(doc.Bytes()), suffix) {
				t.Fatalf("sibling comments changed: %s", doc.Bytes())
			}
		})
	}
}

// Match yaml.v3 v3.0.1's yamlprivateh.go is_break and scannerc.go skip_line:
// CRLF is one break; LF, CR, NEL, LS and PS each advance the parser's line.
func TestConfigurationEdits_UseParserLineBoundaries(t *testing.T) {
	t.Parallel()
	for _, separator := range []string{"\n", "\r\n", "\r", "\u0085", "\u2028", "\u2029"} {
		t.Run(fmt.Sprintf("%q", separator), func(t *testing.T) {
			t.Parallel()
			for _, key := range []string{"guidance", "hooks", "contracts"} {
				t.Run(key, func(t *testing.T) {
					t.Parallel()
					prefix := "hosts: []" + separator
					suffix := strings.ReplaceAll("# sibling note\n\nother: value\n", "\n", separator)
					doc, _, err := ReadBytes([]byte(prefix + key + ": {}" + separator + separator + suffix))
					if err != nil {
						t.Fatal(err)
					}
					switch key {
					case "guidance":
						err = doc.SetGuidanceSelection(nil, []string{"ignored"})
					case "hooks":
						doc.SetHooks(map[string]bool{"test-hook": true})
					case "contracts":
						err = doc.SetContracts(&Contracts{})
					}
					if err != nil {
						t.Fatal(err)
					}
					if !strings.HasPrefix(string(doc.Bytes()), prefix) || !strings.HasSuffix(string(doc.Bytes()), suffix) {
						t.Fatalf("changed outer bytes: %q", doc.Bytes())
					}
					var got struct{ Other string }
					if err := yaml.Unmarshal(doc.Bytes(), &got); err != nil || got.Other != "value" {
						t.Fatalf("invalid edited document: %q (%v)", doc.Bytes(), err)
					}
				})
			}
		})
	}
}

func TestGuidanceSelection_PreservesDocumentEndWithParserLineBreaks(t *testing.T) {
	t.Parallel()
	for _, separator := range []string{"\n", "\r\n", "\r", "\u0085", "\u2028", "\u2029"} {
		t.Run(fmt.Sprintf("%q", separator), func(t *testing.T) {
			t.Parallel()
			suffix := "..." + separator + "# after document" + separator
			doc, _, err := ReadBytes([]byte("guidance: {}" + separator + suffix))
			if err != nil {
				t.Fatal(err)
			}
			if err := doc.SetGuidanceSelection(nil, []string{"ignored"}); err != nil {
				t.Fatal(err)
			}
			if !strings.HasSuffix(string(doc.Bytes()), suffix) {
				t.Fatalf("lost document suffix: %q", doc.Bytes())
			}
			var got struct{ Guidance struct{ Ignored []string } }
			if err := yaml.Unmarshal(doc.Bytes(), &got); err != nil || len(got.Guidance.Ignored) != 1 || got.Guidance.Ignored[0] != "ignored" {
				t.Fatalf("selection outside document: %q (%v)", doc.Bytes(), err)
			}
		})
	}
}
