package migrations

import "embed"

// Files contains the immutable SQL migration set compiled into the API binary.
//
//go:embed *.sql
var Files embed.FS
