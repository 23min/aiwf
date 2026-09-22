package projectguidance

import (
	"context"

	"github.com/23min/aiwf/internal/config"
)

// Selector records desired policy after an explicit, completed interaction.
// A failure must leave the caller's configuration unchanged.
type Selector func(context.Context, string, Catalogue, *config.Config) error
