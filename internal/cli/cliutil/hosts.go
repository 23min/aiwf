package cliutil

import (
	"strings"

	"github.com/23min/aiwf/internal/config"
)

// PrintHostSelection reports the effective host set and its selection source.
func PrintHostSelection(selection config.HostSelection) {
	names := make([]string, len(selection.Hosts))
	for i, host := range selection.Hosts {
		names[i] = string(host)
	}
	value := strings.Join(names, ", ")
	if value == "" {
		value = "none"
	}
	Printf("Hosts (%s): %s\n", selection.Source, value)
}
