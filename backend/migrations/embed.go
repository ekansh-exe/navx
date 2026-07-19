// Package migrations embeds this directory's own SQL files into the Go
// binary (see internal/migrate) so the server can apply them at startup
// without a separate `migrate` CLI step — needed for automatic migrations on
// deploy (Render, or any other host). The manual `migrate -path migrations
// -database ... up` workflow (Makefile, scripts/setup.sh) still works
// unchanged: it reads these same files straight off disk and never sees this
// file, since it isn't a .sql file.
package migrations

import "embed"

//go:embed *.sql
var Files embed.FS
