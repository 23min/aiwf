package aiwfyaml

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Hooks decodes the doc's current `hooks:` block into a decision map (name
// -> enabled). An entry present in the block but omitting `enabled:` is
// excluded from the returned map — the same "undecided" semantics as key
// absence entirely (ADR-0032, mirrors config.Hook's *bool tristate). Returns
// an empty, non-nil map when no `hooks:` block exists.
func (d *Doc) Hooks() (map[string]bool, error) {
	if !d.hasHooks {
		return map[string]bool{}, nil
	}
	var root yaml.Node
	if err := yaml.Unmarshal(d.raw, &root); err != nil { //coverage:ignore re-parse of d.raw (already parsed at ReadBytes) cannot newly fail
		return nil, fmt.Errorf("parsing aiwf.yaml: %w", err)
	}
	top := root.Content[0]
	hooksIdx := findMappingKey(top, "hooks")
	if hooksIdx < 0 { //coverage:ignore hasHooks implies the hooks key is present in the re-parse
		return map[string]bool{}, nil
	}
	return decodeHooks(top.Content[hooksIdx+1])
}

// SetHooks splices decisions into the doc's source bytes as the hooks:
// block, replacing any existing block (or appending a new one when none was
// present). decisions is the FULL desired state — SetHooks does not merge
// with what is already there; a caller wanting a merge reads the current
// state via Hooks() first, computes the union, and passes that in.
//
// Unlike SetContracts, this cannot fail — a decision map is always a valid
// hooks: block, with no Contracts-shaped Validate() to run — so, unlike
// that sibling, it returns no error.
func (d *Doc) SetHooks(decisions map[string]bool) {
	block := marshalHooksBlock(decisions)
	if d.hasHooks {
		d.replaceHooks(block)
	} else {
		d.appendHooks(block)
	}
}

// decodeHooks converts the yaml.Node value of the hooks: key into a
// decision map, with KnownFields strictness so an unrecognized key inside
// one hook's entry fails at parse time (mirrors decodeContracts).
func decodeHooks(n *yaml.Node) (map[string]bool, error) {
	type rawHook struct {
		Enabled *bool `yaml:"enabled"`
	}
	buf, err := yaml.Marshal(n)
	if err != nil { //coverage:ignore re-marshaling an already-parsed node cannot fail
		return nil, fmt.Errorf("re-marshaling hooks subtree: %w", err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(buf))
	dec.KnownFields(true)
	var raw map[string]rawHook
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding hooks: %w", err)
	}
	out := make(map[string]bool, len(raw))
	for name, h := range raw {
		if h.Enabled != nil {
			out[name] = *h.Enabled
		}
	}
	return out, nil
}

// marshalHooksBlock serializes decisions as a YAML fragment beginning with
// the line `hooks:` and ending with a trailing newline, two-space indented,
// sorted by name for determinism. An empty map still emits `hooks: {}` —
// an unambiguous empty mapping rather than a null-valued top-level,
// mirroring marshalContractsBlock's `{}`/`[]` discipline for stable
// round-trips through strict KnownFields decoding.
func marshalHooksBlock(decisions map[string]bool) []byte {
	var b strings.Builder
	if len(decisions) == 0 {
		b.WriteString("hooks: {}\n")
		return []byte(b.String())
	}
	b.WriteString("hooks:\n")
	names := make([]string, 0, len(decisions))
	for name := range decisions {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(&b, "  %s:\n", yamlScalar(name))
		fmt.Fprintf(&b, "    enabled: %t\n", decisions[name])
	}
	return []byte(b.String())
}

// replaceHooks substitutes the block bytes in d.raw at the recorded byte
// range. The range is updated so subsequent SetHooks calls see the new
// layout. Mirrors replaceContracts.
func (d *Doc) replaceHooks(block []byte) {
	out := make([]byte, 0, len(d.raw)-(d.hooksAt.end-d.hooksAt.start)+len(block))
	out = append(out, d.raw[:d.hooksAt.start]...)
	out = append(out, block...)
	out = append(out, d.raw[d.hooksAt.end:]...)
	d.raw = out
	d.hooksAt = byteRange{start: d.hooksAt.start, end: d.hooksAt.start + len(block)}
	d.hasHooks = true
}

// appendHooks adds a brand-new hooks: block at the end of the file,
// separated from prior content by a single blank line for readability.
// Mirrors appendContracts.
func (d *Doc) appendHooks(block []byte) {
	out := make([]byte, 0, len(d.raw)+len(block)+2)
	out = append(out, d.raw...)
	if len(out) > 0 && out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	if len(out) > 0 && !endsWithBlankLine(out) {
		out = append(out, '\n')
	}
	start := len(out)
	out = append(out, block...)
	d.raw = out
	d.hooksAt = byteRange{start: start, end: len(out)}
	d.hasHooks = true
}
