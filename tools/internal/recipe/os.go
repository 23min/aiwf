package recipe

import (
	"io/fs"
	"os"
)

// openOSFile wraps os.Open to satisfy fs.File so the package's
// public ParseFile can read from disk through the same fs.FS plumbing
// it uses for the embed.
func openOSFile(name string) (fs.File, error) {
	return os.Open(name)
}
