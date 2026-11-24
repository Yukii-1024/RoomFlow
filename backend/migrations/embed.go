// Package migrations embeds the SQL migrations applied at server startup.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
