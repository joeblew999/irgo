// Package static embeds the showcase's assets, for the builds that carry
// their own filesystem — mobile, desktop, the Worker.
package static

import (
	"embed"
	"io/fs"
)

// One glob, rather than a list of the directories and files that happen to be
// there today.
//
// The assets are copied out of the Go module by `go generate` and are not
// committed, so a clean checkout has none of them — and //go:embed is a
// compile error when a pattern matches nothing. Naming the showcase's
// top-level files therefore stopped this package compiling before the step
// that creates them could run, and the list needed editing every time upstream
// added an asset. Which it will: that is the point of importing the showcase
// rather than copying it.
//
// `all:*` cannot match nothing — at worst it matches this file. all: because
// the tree has directories Go would otherwise skip.
//
//go:embed all:*
var Files embed.FS

// Ready reports whether the assets have actually been copied in.
//
// The embed above always succeeds, which is what makes it robust and also what
// makes this necessary: nothing else tells a binary built after `go generate`
// from one built before it. Without this the symptom is an unstyled page and a
// screen of 404s, three layers away from the cause.
func Ready() bool {
	// A file the showcase's own pages link, so its absence is exactly the
	// failure this reports rather than a proxy for it.
	_, err := fs.Stat(Files, "min/morpheus.css")
	return err == nil
}
