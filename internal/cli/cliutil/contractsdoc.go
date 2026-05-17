package cliutil

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/config"
)

// LoadContractsBlock reads aiwf.yaml from rootDir and returns the
// contracts: block (nil if absent or if the file itself is absent).
// A malformed contracts: block is an internal error — the caller
// (any verb that needs trustworthy bindings) can't proceed.
func LoadContractsBlock(rootDir string) (*aiwfyaml.Contracts, error) {
	cfgPath := filepath.Join(rootDir, config.FileName)
	if _, err := os.Stat(cfgPath); err != nil {
		return nil, nil
	}
	_, contracts, err := aiwfyaml.Read(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("reading aiwf.yaml: %w", err)
	}
	return contracts, nil
}

// LoadContractsDoc reads aiwf.yaml and returns both the editable
// Doc and the parsed contracts block. Used by mutating verbs that
// need to splice the block back into the source.
func LoadContractsDoc(rootDir string) (*aiwfyaml.Doc, *aiwfyaml.Contracts, error) {
	cfgPath := filepath.Join(rootDir, config.FileName)
	if _, err := os.Stat(cfgPath); err != nil {
		return nil, nil, fmt.Errorf("aiwf.yaml not found at %s; run 'aiwf init' first", cfgPath)
	}
	doc, contracts, err := aiwfyaml.Read(cfgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading aiwf.yaml: %w", err)
	}
	return doc, contracts, nil
}
