// Package aiwfyaml reads and writes the `contracts:` block of the
// consumer repo's aiwf.yaml without disturbing the rest of the file.
//
// The package owns one stretch of YAML — the `contracts:` mapping —
// and leaves everything else byte-for-byte alone. Comments, blank
// lines, key ordering, and indentation outside the block survive
// every programmatic mutation; within the block, structure is
// canonicalized on write.
//
// Two responsibilities:
//
//  1. Parse aiwf.yaml, yield the typed Contracts block (or nil if
//     the block is absent), and the source bytes plus the byte range
//     occupied by `contracts:`. The structural validation rules
//     documented in docs/poc-contracts-plan.md §5 are applied here:
//     - every entries[].validator must reference a key in validators;
//     - every entries[].id must match `C-NNN`;
//     - anchors and aliases anywhere inside the contracts: subtree
//     are a hard error;
//     - unknown fields anywhere in the block are a hard error.
//     Path existence (schema, fixtures) is *not* checked here —
//     those checks happen at verify time.
//
//  2. Splice an updated Contracts block back into the source. The
//     splice is textual: the engine re-marshals only the contracts:
//     block and replaces the corresponding byte range. Bytes before
//     and after that range are untouched. This is the load-bearing
//     guarantee the verbs in §6 of the contracts plan rely on so
//     the LLM can be told "the engine never rewrites your YAML".
package aiwfyaml

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Contracts is the typed shape of the aiwf.yaml `contracts:` block.
// Validators map names (the key the user picks, e.g. "cue") to the
// invocation shape (binary plus argv template). Entries declare
// per-contract bindings: id, the validator name to invoke, the
// schema path, and the fixtures-tree root path.
//
// StrictValidators is the opt-in to "fail the push if any configured
// validator binary is not on PATH." Default false (warn only) so a
// teammate without `cue` can still push when their changes don't
// touch contracts. Teams that want to enforce validator presence on
// every machine set `strict_validators: true` in aiwf.yaml.
type Contracts struct {
	Validators       map[string]Validator
	Entries          []Entry
	StrictValidators bool
}

// Validator declares how to invoke a schema validator. Command is
// resolved via exec.LookPath at run time (absolute path, PATH-relative
// binary name, or repo-relative path all work). Args is an argv
// template that the engine substitutes the four documented variables
// into: {{schema}}, {{fixture}}, {{contract_id}}, {{version}}.
type Validator struct {
	Command string
	Args    []string
}

// Entry binds a contract id to a validator and its schema and
// fixtures locations. Schema and Fixtures are repo-relative paths
// that are not checked for existence at parse time.
type Entry struct {
	ID        string
	Validator string
	Schema    string
	Fixtures  string
}

// idPattern matches the contract id format. Mirrors entity.idPatterns
// for KindContract; duplicated here so this package has no dependency
// on the entity package (parse-time validation only — runtime checks
// against the in-memory tree happen elsewhere).
var idPattern = regexp.MustCompile(`^C-\d{3,}$`)

// Doc is a parsed aiwf.yaml plus the byte range of its `contracts:`
// block. Doc carries enough information to write an updated contracts
// block back without round-tripping the rest of the file.
type Doc struct {
	raw          []byte
	contractsAt  byteRange
	hasContracts bool
}

// byteRange names a half-open [start, end) byte interval inside Doc.raw.
// When the contracts: block is absent, byteRange points at len(raw)
// and hasContracts is false; SetContracts then appends a new block.
type byteRange struct {
	start int
	end   int
}

// Read parses aiwf.yaml at path. Returns the Doc, the parsed
// Contracts block (nil if the file has no `contracts:` key), and an
// error if the file is unreadable, malformed, or violates the
// structural rules for the contracts: block.
func Read(path string) (*Doc, *Contracts, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return ReadBytes(raw)
}

// ReadBytes is Read for an in-memory byte slice. Useful for tests
// and for callers that already have the file content.
func ReadBytes(raw []byte) (*Doc, *Contracts, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(raw, &root); err != nil {
		return nil, nil, fmt.Errorf("parsing aiwf.yaml: %w", err)
	}
	doc := &Doc{raw: raw, contractsAt: byteRange{start: len(raw), end: len(raw)}}
	if root.Kind == 0 || len(root.Content) == 0 {
		return doc, nil, nil
	}
	top := root.Content[0]
	if top.Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("aiwf.yaml: top-level must be a YAML mapping, got %s", kindName(top.Kind))
	}

	keyIdx := findMappingKey(top, "contracts")
	if keyIdx < 0 {
		return doc, nil, nil
	}
	keyNode := top.Content[keyIdx]
	valNode := top.Content[keyIdx+1]

	if err := rejectAnchorsAndAliases(valNode); err != nil {
		return nil, nil, fmt.Errorf("aiwf.yaml contracts: %w", err)
	}

	contracts, err := decodeContracts(valNode)
	if err != nil {
		return nil, nil, fmt.Errorf("aiwf.yaml contracts: %w", err)
	}
	if err = contracts.Validate(); err != nil {
		return nil, nil, fmt.Errorf("aiwf.yaml contracts: %w", err)
	}

	start, end, err := blockByteRange(raw, top, keyNode, keyIdx)
	if err != nil {
		return nil, nil, err
	}
	doc.contractsAt = byteRange{start: start, end: end}
	doc.hasContracts = true
	return doc, contracts, nil
}

// SetContracts splices c into the doc's source bytes, replacing any
// existing contracts: block (or appending one when none was present).
// A nil c removes the block. Calling SetContracts more than once on
// the same Doc is supported; each call operates on the post-splice
// bytes.
func (d *Doc) SetContracts(c *Contracts) error {
	if c == nil {
		d.removeContracts()
		return nil
	}
	if err := c.Validate(); err != nil {
		return fmt.Errorf("contracts: %w", err)
	}
	block := marshalContractsBlock(c)
	if d.hasContracts {
		d.replaceContracts(block)
	} else {
		d.appendContracts(block)
	}
	return nil
}

// Bytes returns the doc's current source bytes. The slice is shared;
// callers must not mutate it.
func (d *Doc) Bytes() []byte {
	return d.raw
}

// Write atomically writes the doc's bytes to path.
func (d *Doc) Write(path string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, d.raw, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming %s -> %s: %w", tmp, path, err)
	}
	return nil
}

// Validate enforces the contracts plan §5 structural rules: every
// entries[].validator references a name in Validators, every
// entries[].id matches the C-NNN format, every validator has a
// non-empty command, and every entry has non-empty schema/fixtures.
func (c *Contracts) Validate() error {
	for name, v := range c.Validators {
		if name == "" {
			return fmt.Errorf("validators: empty key is not allowed")
		}
		if v.Command == "" {
			return fmt.Errorf("validators[%s]: command is required", name)
		}
	}
	for i, e := range c.Entries {
		if !idPattern.MatchString(e.ID) {
			return fmt.Errorf("entries[%d]: id %q does not match C-NNN format", i, e.ID)
		}
		if e.Validator == "" {
			return fmt.Errorf("entries[%d] (id=%s): validator is required", i, e.ID)
		}
		if _, ok := c.Validators[e.Validator]; !ok {
			return fmt.Errorf("entries[%d] (id=%s): validator %q is not declared in contracts.validators", i, e.ID, e.Validator)
		}
		if e.Schema == "" {
			return fmt.Errorf("entries[%d] (id=%s): schema is required", i, e.ID)
		}
		if e.Fixtures == "" {
			return fmt.Errorf("entries[%d] (id=%s): fixtures is required", i, e.ID)
		}
	}
	return nil
}

// findMappingKey returns the index of the key node in m.Content
// whose value is name, or -1 when the key is absent. Mapping nodes
// store key/value pairs as adjacent slice entries; the value node is
// at index+1.
func findMappingKey(m *yaml.Node, name string) int {
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == name {
			return i
		}
	}
	return -1
}

// rejectAnchorsAndAliases walks the subtree and returns an error on
// the first anchor or alias node it sees. The contracts plan §5
// declares anchors/aliases inside the contracts: block unsupported;
// we fail loudly rather than risk silent semantic drift on round-trip.
func rejectAnchorsAndAliases(n *yaml.Node) error {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.AliasNode {
		return fmt.Errorf("aliases are not supported inside the contracts: block (line %d)", n.Line)
	}
	if n.Anchor != "" {
		return fmt.Errorf("anchors are not supported inside the contracts: block (anchor %q, line %d)", n.Anchor, n.Line)
	}
	for _, c := range n.Content {
		if err := rejectAnchorsAndAliases(c); err != nil {
			return err
		}
	}
	return nil
}

// decodeContracts converts the yaml.Node value of the contracts: key
// into a typed *Contracts, with KnownFields strictness so unknown
// keys anywhere in the subtree fail at parse time.
//
// The implementation re-marshals the node and decodes the bytes
// through a yaml.Decoder with KnownFields(true). Two parses for a
// tiny block; the simplicity is worth the cost.
func decodeContracts(n *yaml.Node) (*Contracts, error) {
	type rawValidator struct {
		Command string   `yaml:"command"`
		Args    []string `yaml:"args"`
	}
	type rawEntry struct {
		ID        string `yaml:"id"`
		Validator string `yaml:"validator"`
		Schema    string `yaml:"schema"`
		Fixtures  string `yaml:"fixtures"`
	}
	type rawContracts struct {
		Validators       map[string]rawValidator `yaml:"validators"`
		Entries          []rawEntry              `yaml:"entries"`
		StrictValidators bool                    `yaml:"strict_validators"`
	}

	buf, err := yaml.Marshal(n)
	if err != nil {
		return nil, fmt.Errorf("re-marshaling contracts subtree: %w", err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(buf))
	dec.KnownFields(true)
	var rc rawContracts
	if err := dec.Decode(&rc); err != nil {
		return nil, fmt.Errorf("decoding: %w", err)
	}

	out := &Contracts{
		Validators:       make(map[string]Validator, len(rc.Validators)),
		StrictValidators: rc.StrictValidators,
	}
	for name, v := range rc.Validators {
		out.Validators[name] = Validator{
			Command: v.Command,
			Args:    append([]string(nil), v.Args...),
		}
	}
	for _, e := range rc.Entries {
		out.Entries = append(out.Entries, Entry(e))
	}
	return out, nil
}

// blockByteRange returns the half-open [start, end) byte interval
// occupied by the contracts: key/value pair in raw. The interval
// covers the line containing the `contracts:` key down to (but not
// including) the next top-level key, or to EOF when contracts: is
// the last top-level key.
//
// Comments above `contracts:` (yaml.v3's HeadComment) are *outside*
// the block per the §5 contract — the splice starts at the
// `contracts:` line itself so those comments survive untouched.
// Likewise, a HeadComment on the next top-level key belongs to that
// next key, so the splice stops at the comment, not at the key line.
func blockByteRange(raw []byte, top, keyNode *yaml.Node, keyIdx int) (start, end int, err error) {
	start, err = lineToByteOffset(raw, keyNode.Line)
	if err != nil {
		return 0, 0, err
	}

	if keyIdx+2 < len(top.Content) {
		nextKey := top.Content[keyIdx+2]
		endLine := nextKey.Line
		if hc := strings.TrimSpace(nextKey.HeadComment); hc != "" {
			endLine -= countLines(nextKey.HeadComment)
			if endLine < 1 {
				endLine = 1
			}
		}
		end, err = lineToByteOffset(raw, endLine)
		if err != nil {
			return 0, 0, err
		}
	} else {
		end = len(raw)
	}
	return start, end, nil
}

// lineToByteOffset returns the byte offset of the start of the
// 1-based line in raw. Returns len(raw) when line is past EOF.
func lineToByteOffset(raw []byte, line int) (int, error) {
	if line <= 1 {
		return 0, nil
	}
	current := 1
	for i, b := range raw {
		if b == '\n' {
			current++
			if current == line {
				return i + 1, nil
			}
		}
	}
	return len(raw), nil
}

// countLines returns the number of newline-separated lines in s.
// Empty string is zero lines; a non-empty string with no newline is
// one line; a trailing newline counts the line it terminates.
func countLines(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}

// marshalContractsBlock serializes c as a YAML fragment beginning
// with the line `contracts:` and ending with a trailing newline. The
// fragment is two-space-indented, sorted-key for validators, and
// preserves entry order from the slice.
//
// validators: and entries: are always emitted (as `{}` / `[]` when
// empty) so the resulting block is unambiguously a mapping with two
// keys, not a null-valued top-level. This keeps round-trips through
// strict KnownFields decoding stable even for empty contracts blocks.
func marshalContractsBlock(c *Contracts) []byte {
	var b strings.Builder
	b.WriteString("contracts:\n")

	if c.StrictValidators {
		b.WriteString("  strict_validators: true\n")
	}

	if len(c.Validators) > 0 {
		b.WriteString("  validators:\n")
		names := sortedKeys(c.Validators)
		for _, name := range names {
			v := c.Validators[name]
			fmt.Fprintf(&b, "    %s:\n", name)
			fmt.Fprintf(&b, "      command: %s\n", yamlScalar(v.Command))
			if len(v.Args) > 0 {
				b.WriteString("      args:\n")
				for _, a := range v.Args {
					fmt.Fprintf(&b, "        - %s\n", yamlScalar(a))
				}
			} else {
				b.WriteString("      args: []\n")
			}
		}
	} else {
		b.WriteString("  validators: {}\n")
	}

	if len(c.Entries) > 0 {
		b.WriteString("  entries:\n")
		for _, e := range c.Entries {
			fmt.Fprintf(&b, "    - id: %s\n", yamlScalar(e.ID))
			fmt.Fprintf(&b, "      validator: %s\n", yamlScalar(e.Validator))
			fmt.Fprintf(&b, "      schema: %s\n", yamlScalar(e.Schema))
			fmt.Fprintf(&b, "      fixtures: %s\n", yamlScalar(e.Fixtures))
		}
	} else {
		b.WriteString("  entries: []\n")
	}

	return []byte(b.String())
}

// sortedKeys returns the keys of m in ascending order. Used to make
// validator output deterministic; iteration order of Go maps is
// randomized.
func sortedKeys(m map[string]Validator) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// yamlScalar emits a scalar in a form yaml.v3 will round-trip without
// surprises: a plain string when the value is "safe" (alphanumeric,
// dots, slashes, dashes, underscores, no leading reserved character),
// otherwise a double-quoted string with the standard escapes.
func yamlScalar(s string) string {
	if s == "" {
		return `""`
	}
	if needsQuoting(s) {
		return quoteYAML(s)
	}
	return s
}

// needsQuoting flags any string yaml.v3 might interpret as something
// other than a plain string when written unquoted. The set is
// deliberately conservative: when in doubt, quote.
func needsQuoting(s string) bool {
	switch s {
	case "true", "false", "yes", "no", "on", "off", "null", "~":
		return true
	}
	if strings.ContainsAny(s, ":#@`*&!|>'\"%\n\t") {
		return true
	}
	if strings.HasPrefix(s, "-") || strings.HasPrefix(s, "?") || strings.HasPrefix(s, "[") || strings.HasPrefix(s, "{") {
		return true
	}
	if strings.HasPrefix(s, " ") || strings.HasSuffix(s, " ") {
		return true
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	if s[0] >= '0' && s[0] <= '9' {
		return true
	}
	return false
}

// quoteYAML returns s as a double-quoted YAML scalar with the
// standard escape sequences applied. yaml.v3 accepts this form
// everywhere a scalar is allowed.
func quoteYAML(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&b, `\x%02x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// replaceContracts substitutes the block bytes in d.raw at the
// recorded byte range. The range is updated so subsequent
// SetContracts calls see the new layout.
func (d *Doc) replaceContracts(block []byte) {
	out := make([]byte, 0, len(d.raw)-(d.contractsAt.end-d.contractsAt.start)+len(block))
	out = append(out, d.raw[:d.contractsAt.start]...)
	out = append(out, block...)
	out = append(out, d.raw[d.contractsAt.end:]...)
	d.raw = out
	d.contractsAt = byteRange{start: d.contractsAt.start, end: d.contractsAt.start + len(block)}
	d.hasContracts = true
}

// appendContracts adds a brand-new contracts: block at the end of
// the file, separated from prior content by a single blank line for
// readability.
func (d *Doc) appendContracts(block []byte) {
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
	d.contractsAt = byteRange{start: start, end: len(out)}
	d.hasContracts = true
}

// removeContracts deletes the contracts: block from d.raw, also
// trimming the optional preceding blank line so the file doesn't
// accumulate blank lines on repeated set-then-clear calls.
func (d *Doc) removeContracts() {
	if !d.hasContracts {
		return
	}
	start := d.contractsAt.start
	for start >= 2 && d.raw[start-1] == '\n' && d.raw[start-2] == '\n' {
		start--
	}
	out := make([]byte, 0, len(d.raw)-(d.contractsAt.end-start))
	out = append(out, d.raw[:start]...)
	out = append(out, d.raw[d.contractsAt.end:]...)
	d.raw = out
	d.contractsAt = byteRange{start: len(out), end: len(out)}
	d.hasContracts = false
}

// endsWithBlankLine reports whether b ends in two consecutive
// newlines (i.e., the final non-empty line is followed by a blank).
func endsWithBlankLine(b []byte) bool {
	n := len(b)
	return n >= 2 && b[n-1] == '\n' && b[n-2] == '\n'
}

// kindName returns a human-readable label for a yaml.Node kind.
// Used in error messages where the underlying kind is otherwise an
// uninformative integer.
func kindName(k yaml.Kind) string {
	switch k {
	case yaml.DocumentNode:
		return "document"
	case yaml.SequenceNode:
		return "sequence"
	case yaml.MappingNode:
		return "mapping"
	case yaml.ScalarNode:
		return "scalar"
	case yaml.AliasNode:
		return "alias"
	}
	return "unknown"
}
