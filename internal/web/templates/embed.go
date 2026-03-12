package templates

import "embed"

//go:embed *.html **/*.html
var Files embed.FS
