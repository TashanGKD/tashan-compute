package capabilities

import "embed"

// Files contains the capability manifest compiled into server binaries.
//
//go:embed manifest.json
var Files embed.FS
