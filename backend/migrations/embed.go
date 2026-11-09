// Package migrations embebe los scripts SQL en el binario.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
