package projectguidance

import (
	"context"

	"github.com/23min/aiwf/internal/config"
)

// Selector inspects available policy, reporting suggestions or recording choices
// after an explicit, completed interaction. Reporting and failures must leave the
// caller's configuration unchanged.
type Selector func(context.Context, string, Catalogue, *config.Config) error
