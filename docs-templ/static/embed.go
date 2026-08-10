// Package static provides embedded static files for mobile builds.
package static

import "embed"

//go:embed css js img llms.txt
var Files embed.FS
