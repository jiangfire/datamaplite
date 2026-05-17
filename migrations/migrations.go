// Package migrations holds embedded SQL migration files for the PostgreSQL store.
// Bundling them into the binary lets single-binary deployments work without
// requiring the migrations/ directory to exist alongside the executable.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
