package aiwfyaml

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSetGuidanceSelection_PreservesUnrelatedFields(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		"# intro\nhosts: []\nguidance:\n  source: local\n  custom: kept\n  packs: [old]\n  ignored: [prior]\n# ending\nother: value\n",
		"hosts: []\nguidance: {source: local, custom: kept}\nother: value\n",
		"{hosts: [], guidance: {source: local, custom: kept}, other: value}\n",
		"hosts: []\nother: value",
		"hosts: []\nother: value\n... # document end\n",
		"",
	} {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			doc, _, err := ReadBytes([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			selected := []string{"new"}
			if err := doc.SetGuidanceSelection(&selected, []string{"ignored"}); err != nil {
				t.Fatal(err)
			}
			var got struct {
				Guidance struct {
					Source         string
					Custom         string
					Packs, Ignored []string
				}
				Other string
			}
			if err := yaml.Unmarshal(doc.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if len(got.Guidance.Packs) != 1 || got.Guidance.Packs[0] != "new" || len(got.Guidance.Ignored) != 1 || got.Guidance.Ignored[0] != "ignored" {
				t.Fatalf("selection: %s", doc.Bytes())
			}
			if strings.Contains(raw, "source: local") && (got.Guidance.Source != "local" || got.Guidance.Custom != "kept") {
				t.Fatalf("lost guidance siblings: %s", doc.Bytes())
			}
			if strings.Contains(raw, "other: value") && got.Other != "value" {
				t.Fatalf("lost root sibling: %s", doc.Bytes())
			}
			if strings.HasPrefix(raw, "# intro") && (!strings.HasPrefix(string(doc.Bytes()), "# intro\nhosts: []\n") || !strings.HasSuffix(string(doc.Bytes()), "# ending\nother: value\n")) {
				t.Fatalf("changed outside guidance: %s", doc.Bytes())
			}
		})
	}
}

func TestSetGuidanceSelection_IgnoreDoesNotAdoptAndEmptyStaysExplicit(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{"", "guidance: null\n", "guidance: {packs: []}\n"} {
		doc, _, err := ReadBytes([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if err := doc.SetGuidanceSelection(nil, []string{"ignored"}); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(doc.Bytes()), "packs:") != strings.Contains(raw, "packs:") {
			t.Fatalf("changed ownership: %s", doc.Bytes())
		}
	}
}

func TestSetGuidanceSelection_RefusesSharedValuesWithoutChangingDocument(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{"base: &g {source: local}\nguidance: *g\n", "guidance: &g {source: local}\nother: *g\n", "guidance: scalar\n"} {
		doc, _, err := ReadBytes([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if err := doc.SetGuidanceSelection(nil, []string{"ignored"}); err == nil {
			t.Fatal("unsafe guidance shape accepted")
		}
		if string(doc.Bytes()) != raw {
			t.Fatal("failed edit changed document")
		}
	}
}

func TestSetGuidanceSelection_PreservesFieldCommentsAtFirstKey(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		"guidance:\n  packs: [old] # selected policy\n  ignored: [] # suppressed suggestions\n",
		"guidance: null # selected policy\n",
	} {
		doc, _, err := ReadBytes([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		selected := []string{"new"}
		if err := doc.SetGuidanceSelection(&selected, []string{"ignored"}); err != nil {
			t.Fatal(err)
		}
		for _, comment := range []string{"# selected policy", "# suppressed suggestions"} {
			if strings.Contains(raw, comment) && !strings.Contains(string(doc.Bytes()), comment) {
				t.Fatalf("lost field comment %q: %s", comment, doc.Bytes())
			}
		}
		var got struct{ Guidance struct{ Packs []string } }
		if err := yaml.Unmarshal(doc.Bytes(), &got); err != nil || len(got.Guidance.Packs) != 1 || got.Guidance.Packs[0] != "new" {
			t.Fatalf("first-key replacement: %s (%v)", doc.Bytes(), err)
		}
	}
}

func TestSetGuidanceSelection_PreservesCommentOnlyConfig(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{"# project note\n", "# project note"} {
		doc, _, err := ReadBytes([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if err := doc.SetGuidanceSelection(nil, []string{"ignored"}); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(doc.Bytes()), "# project note\n") {
			t.Fatalf("lost notes: %s", doc.Bytes())
		}
	}
}

func TestSetGuidanceSelection_PreservesOuterBytesWhenGuidanceFirst(t *testing.T) {
	t.Parallel()
	const suffix = "custom :   [  untouched  ] # spacing\n"
	doc, _, err := ReadBytes([]byte("guidance: {packs: []}\n" + suffix))
	if err != nil {
		t.Fatal(err)
	}
	selected := []string{"new"}
	if err := doc.SetGuidanceSelection(&selected, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(doc.Bytes()), suffix) {
		t.Fatalf("reformatted unrelated fields: %s", doc.Bytes())
	}
}
