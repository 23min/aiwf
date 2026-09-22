package aiwfyaml

import (
	"strings"
	"testing"
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
