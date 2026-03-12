package static

import "embed"

//go:embed *.js *.css games
var Files embed.FS
