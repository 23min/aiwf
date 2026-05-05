package policies

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// PolicyFindingCodesAreDiscoverable asserts that every finding code
// emitted by the kernel — both named constants in
// tools/internal/check/ and inline `Code: "..."` literals in
// Finding{} composite literals across check/ and contractcheck/ —
// appears verbatim in at least one channel an AI assistant routinely
// consults: any embedded skill SKILL.md, the binary's printHelp
// output (cmd/aiwf/main.go), CLAUDE.md / tools/CLAUDE.md, or any
// markdown file under docs/pocv3/. The CLAUDE.md "kernel
// functionality must be AI-discoverable" principle: a code that
// only exists in source is, by definition, undocumented.
//
// Closes G21. The policy used to scope to provenance-* codes only;
// extending to all codes (and to the full four-channel set) is the
// substantive G21 fix.
func PolicyFindingCodesAreDiscoverable(root string) ([]Violation, error) {
	prodFiles, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	codes := allCheckCodes(prodFiles)

	haystack, err := readDiscoverabilityChannels(root)
	if err != nil {
		return nil, err
	}

	var out []Violation
	for code := range codes {
		if bytes.Contains(haystack, []byte(code)) {
			continue
		}
		out = append(out, Violation{
			Policy: "finding-codes-are-discoverable",
			File:   "tools/internal/skills/embedded/aiwf-check/SKILL.md",
			Detail: code + " is a finding code in the kernel but is not mentioned in any AI-discoverable channel (embedded skills, aiwf <verb> --help, CLAUDE.md, or docs/pocv3/**/*.md)",
		})
	}
	return out, nil
}

// allCheckCodes returns the union of finding codes from named
// constants in tools/internal/check/ and inline `Code: "..."`
// literals in Finding{} composite literals across check/ and
// contractcheck/. When a Finding{} composite also carries a
// non-empty Subcode literal, the composite "<code>/<subcode>" is
// included alongside the bare code, matching how aiwf-check
// SKILL.md writes them. Filtered to kebab-case finding-code shape
// so non-code constants (severities, etc.) are excluded.
func allCheckCodes(files []FileEntry) map[string]struct{} {
	out := map[string]struct{}{}
	for _, v := range loadCheckCodeConstants(files) {
		if looksLikeFindingCode(v) {
			out[v] = struct{}{}
		}
	}
	for v := range loadCheckCodeLiterals(files) {
		out[v] = struct{}{}
	}
	return out
}

// loadCheckCodeLiterals AST-parses production check/ and
// contractcheck/ files and returns the set of inline string literals
// assigned to a `Code` field in any composite literal. Captures the
// codes that aren't declared as named constants (most of the
// pre-I2.5 surface).
//
// When the same composite literal also has a Subcode field with a
// non-empty string literal, the composite "<code>/<subcode>" string
// is added too — that's how aiwf-check SKILL.md writes them, and a
// new subcode that's never named in the discoverability haystack is
// just as undocumented as a new code.
func loadCheckCodeLiterals(files []FileEntry) map[string]struct{} {
	out := map[string]struct{}{}
	fset := token.NewFileSet()
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "tools/internal/check/") &&
			!strings.HasPrefix(f.Path, "tools/internal/contractcheck/") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			code := stringFieldValue(cl, "Code")
			if code == "" || !looksLikeFindingCode(code) {
				return true
			}
			out[code] = struct{}{}
			if sub := stringFieldValue(cl, "Subcode"); sub != "" {
				out[code+"/"+sub] = struct{}{}
			}
			return true
		})
	}
	return out
}

// stringFieldValue returns the unquoted string-literal value of the
// named field on the composite, or "" when absent / non-string.
func stringFieldValue(cl *ast.CompositeLit, name string) string {
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != name {
			continue
		}
		lit, ok := kv.Value.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		v, err := strconv.Unquote(lit.Value)
		if err != nil {
			continue
		}
		return v
	}
	return ""
}

// readDiscoverabilityChannels concatenates the contents of every
// documentation channel an AI assistant routinely consults. The
// concatenation is matched as one big haystack — substring presence
// in any channel passes the policy.
func readDiscoverabilityChannels(root string) ([]byte, error) {
	var out []byte
	singletons := []string{
		filepath.Join(root, "tools", "cmd", "aiwf", "main.go"),
		filepath.Join(root, "CLAUDE.md"),
		filepath.Join(root, "tools", "CLAUDE.md"),
	}
	for _, p := range singletons {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		out = append(out, data...)
		out = append(out, '\n')
	}
	for _, dir := range []string{
		filepath.Join(root, "tools", "internal", "skills", "embedded"),
		filepath.Join(root, "docs", "pocv3"),
	} {
		walkErr := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(p, ".md") {
				return nil
			}
			data, rerr := os.ReadFile(p)
			if rerr != nil {
				return rerr
			}
			out = append(out, data...)
			out = append(out, '\n')
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}
	return out, nil
}
