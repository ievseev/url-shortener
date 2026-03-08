package migrations

import "embed"

// FS stores embedded SQL migrations for automatic schema setup.
//
//go:embed *.sql
var FS embed.FS
