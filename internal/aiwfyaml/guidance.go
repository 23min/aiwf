package aiwfyaml

import (
	"bytes"
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"
)

// SetGuidanceSelection updates only selected and ignored ids. An absent selection
// remains absent until a pack is selected, and an ignored key is written only
// when ids are ignored or the key already exists. Bytes outside the guidance
// block are kept when the edited document decodes to the same values as a
// re-encoding; a missing block is then added after the existing content.
// Otherwise, as for a flow-style or indented root, the document is re-encoded,
// retaining fields and comments.
func (d *Doc) SetGuidanceSelection(selected *[]string, ignored []string) error {
	var document yaml.Node
	if err := yaml.Unmarshal(d.raw, &document); err != nil { //coverage:ignore Doc.raw is already parsed by ReadBytes; public edits emit valid YAML
		return fmt.Errorf("reading guidance configuration: %w", err)
	}
	top := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if len(document.Content) > 0 {
		top = document.Content[0]
	}
	index := findMappingKey(top, "guidance")
	if index < 0 && findMappingKey(top, "<<") >= 0 {
		var inherited struct{ Guidance yaml.Node }
		if decodeErr := top.Decode(&inherited); decodeErr != nil {
			return fmt.Errorf("reading inherited guidance configuration: %w", decodeErr)
		}
		if inherited.Guidance.Kind != 0 {
			return fmt.Errorf("expand inherited guidance into an explicit mapping so guidance packs can be recorded")
		}
	}
	documentEnd := guidanceDocumentEnd(d.raw)
	raw := d.raw[:documentEnd]
	key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "guidance"}
	value := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if index >= 0 {
		key, value = top.Content[index], top.Content[index+1]
		if err := rejectAnchorsAndAliases(value, "guidance"); err != nil {
			return fmt.Errorf("expand guidance anchors and aliases so guidance packs can be recorded: %w", err)
		}
		if value.Tag == "!!null" {
			if value.LineComment != "" {
				key.LineComment = value.LineComment
			}
			value = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", HeadComment: value.HeadComment, FootComment: value.FootComment}
			top.Content[index+1] = value
		}
		if value.Kind != yaml.MappingNode {
			return fmt.Errorf("guidance must be a mapping so guidance packs can be recorded")
		}
	} else {
		top.Content = append(top.Content, key, value)
	}
	if selected != nil {
		setGuidanceIDs(value, "packs", *selected)
	}
	if len(ignored) > 0 || findMappingKey(value, "ignored") >= 0 {
		setGuidanceIDs(value, "ignored", ignored)
	}
	// The re-encoded document is the reference result. A byte-preserving edit
	// is used only when it decodes to exactly the same values; layouts it
	// cannot express faithfully (an indented root, a trailing block scalar
	// without a final newline) fall back to re-encoding.
	document.Kind = yaml.DocumentNode
	document.Content = []*yaml.Node{top}
	reencoded, err := yaml.Marshal(&document)
	if err != nil { //coverage:ignore parsed nodes and constructed string sequences are YAML-encodable
		return fmt.Errorf("encoding guidance configuration: %w", err)
	}
	reencoded = append(reencoded, d.raw[documentEnd:]...)
	out := reencoded
	if top.Style&yaml.FlowStyle == 0 {
		var preserved []byte
		// The key's leading comment is outside a replaced byte range.
		blockKey := *key
		blockKey.HeadComment = ""
		block := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{&blockKey, value}}
		if index+2 == len(top.Content) {
			block.FootComment = document.FootComment
		}
		var encoded bytes.Buffer
		encoder := yaml.NewEncoder(&encoded)
		encoder.SetIndent(2)
		if err = encoder.Encode(block); err != nil { //coverage:ignore valid parsed nodes encoded into an infallible bytes.Buffer
			return fmt.Errorf("encoding guidance block: %w", err)
		}
		if err = encoder.Close(); err != nil { //coverage:ignore bytes.Buffer cannot fail on encoder flush
			return fmt.Errorf("closing guidance encoder: %w", err)
		}
		if index < 0 {
			// A missing block is added after the existing bytes.
			preserved = append(preserved, raw...)
			if len(preserved) > 0 && preserved[len(preserved)-1] != '\n' {
				preserved = append(preserved, '\n')
			}
			preserved = append(preserved, encoded.Bytes()...)
		} else {
			start, end, rangeErr := blockByteRange(raw, top, key, index)
			if rangeErr != nil { //coverage:ignore blockByteRange uses lineToByteOffset, which has no error-producing path
				return rangeErr
			}
			preserved = append(preserved, raw[:start]...)
			preserved = append(preserved, encoded.Bytes()...)
			preserved = append(preserved, raw[end:]...)
		}
		preserved = append(preserved, d.raw[documentEnd:]...)
		if sameYAMLValues(preserved, reencoded) {
			out = preserved
		}
	}
	updated, _, err := ReadBytes(out)
	if err != nil { //coverage:ignore only guidance ids changed; emitted YAML and previously validated contracts remain valid
		return err
	}
	*d = *updated
	return nil
}

// setGuidanceIDs writes ids under key. Existing entries keep their nodes, and
// so their comments, and the list keeps its style. A comment on an empty key
// moves onto the new list.
func setGuidanceIDs(mapping *yaml.Node, key string, ids []string) {
	index := findMappingKey(mapping, key)
	var previous *yaml.Node
	if index >= 0 {
		previous = mapping.Content[index+1]
	}
	style := yaml.FlowStyle
	existing := make(map[string]*yaml.Node)
	if previous != nil && previous.Kind == yaml.SequenceNode {
		style = previous.Style
		for _, item := range previous.Content {
			existing[item.Value] = item
		}
	}
	sequence := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: style}
	for _, id := range ids {
		item, ok := existing[id]
		if !ok {
			item = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: id}
		}
		sequence.Content = append(sequence.Content, item)
	}
	if previous == nil {
		mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, sequence)
		return
	}
	sequence.HeadComment, sequence.LineComment, sequence.FootComment = previous.HeadComment, previous.LineComment, previous.FootComment
	if keyNode := mapping.Content[index]; sequence.LineComment == "" && previous.Kind == yaml.ScalarNode {
		sequence.LineComment, keyNode.LineComment = keyNode.LineComment, ""
	}
	mapping.Content[index+1] = sequence
}

// guidanceDocumentEnd locates the raw suffix that yaml.v3 drops when encoding.
// YAML document-end markers start in column one and require a separator.
func guidanceDocumentEnd(raw []byte) int {
	offset := 0
	for _, line := range yamlLines(raw) {
		content := bytes.TrimRight(line, "\r\n\u0085\u2028\u2029")
		if bytes.HasPrefix(content, []byte("...")) && (len(content) == 3 || bytes.ContainsAny(content[3:4], " \t")) {
			return offset
		}
		offset += len(line)
	}
	return len(raw)
}

// sameYAMLValues reports whether both documents decode to equal values. A NaN
// or a collection used as a key never compares equal, so such a document is
// re-encoded rather than edited in place.
func sameYAMLValues(a, b []byte) bool {
	var left, right any
	if yaml.Unmarshal(a, &left) != nil || yaml.Unmarshal(b, &right) != nil {
		return false
	}
	return reflect.DeepEqual(left, right)
}
