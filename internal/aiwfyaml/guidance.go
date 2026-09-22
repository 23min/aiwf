package aiwfyaml

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// SetGuidanceSelection updates only selected and ignored ids. An absent selection
// remains absent until a pack is selected; ignoring suggestions is not adoption.
// Editing an existing block preserves bytes outside guidance. Adding a block or
// editing a flow-style root re-encodes the document, retaining fields and comments.
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
	key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "guidance"}
	value := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if index >= 0 {
		key, value = top.Content[index], top.Content[index+1]
		if err := rejectAnchorsAndAliases(value); err != nil {
			return fmt.Errorf("expand guidance anchors and aliases before selecting packs: %w", err)
		}
		if value.Tag == "!!null" {
			key.LineComment = value.LineComment
			value = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", HeadComment: value.HeadComment, FootComment: value.FootComment}
			top.Content[index+1] = value
		}
		if value.Kind != yaml.MappingNode {
			return fmt.Errorf("guidance must be a mapping before selecting packs")
		}
	} else {
		top.Content = append(top.Content, key, value)
	}
	if selected != nil {
		setGuidanceIDs(value, "packs", *selected)
	}
	setGuidanceIDs(value, "ignored", ignored)
	var out []byte
	if index < 0 || top.Style&yaml.FlowStyle != 0 {
		encoded, err := yaml.Marshal(top)
		if err != nil { //coverage:ignore parsed nodes and constructed string sequences are YAML-encodable
			return fmt.Errorf("encoding guidance configuration: %w", err)
		}
		if len(document.Content) == 0 {
			out = append(out, d.raw...)
			if len(out) > 0 && out[len(out)-1] != '\n' {
				out = append(out, '\n')
			}
		}
		out = append(out, encoded...)
	} else {
		// The key's leading comment is outside the replaced byte range.
		blockKey := *key
		blockKey.HeadComment = ""
		block := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{&blockKey, value}}
		var encoded bytes.Buffer
		encoder := yaml.NewEncoder(&encoded)
		encoder.SetIndent(2)
		if err := encoder.Encode(block); err != nil { //coverage:ignore valid parsed nodes encoded into an infallible bytes.Buffer
			return fmt.Errorf("encoding guidance block: %w", err)
		}
		if err := encoder.Close(); err != nil { //coverage:ignore bytes.Buffer cannot fail on encoder flush
			return fmt.Errorf("closing guidance encoder: %w", err)
		}
		start, end, err := blockByteRange(d.raw, top, key, index)
		if err != nil { //coverage:ignore blockByteRange uses lineToByteOffset, which has no error-producing path
			return err
		}
		out = append(out, d.raw[:start]...)
		out = append(out, encoded.Bytes()...)
		out = append(out, d.raw[end:]...)
	}
	updated, _, err := ReadBytes(out)
	if err != nil { //coverage:ignore only guidance ids changed; emitted YAML and previously validated contracts remain valid
		return err
	}
	*d = *updated
	return nil
}

func setGuidanceIDs(mapping *yaml.Node, key string, ids []string) {
	sequence := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
	for _, id := range ids {
		sequence.Content = append(sequence.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: id})
	}
	if index := findMappingKey(mapping, key); index >= 0 {
		previous := mapping.Content[index+1]
		sequence.HeadComment, sequence.LineComment, sequence.FootComment = previous.HeadComment, previous.LineComment, previous.FootComment
		mapping.Content[index+1] = sequence
	} else {
		mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, sequence)
	}
}
