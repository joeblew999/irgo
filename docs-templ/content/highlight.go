package content

// The highlighting is generated. `go generate ./...` refreshes it; a test
// fails when it is stale.
//
//go:generate go test . -run TestHighlightingIsCurrent -update

import (
	"crypto/sha256"
	"encoding/hex"
)

// Syntax highlighting is precomputed, not done here.
//
// The obvious implementation imports chroma and highlights on the way out.
// This binary is a WebAssembly module served from a Cloudflare Worker, and
// chroma's lexers package embeds every language it knows: importing it took
// the worker from 2.44 MB compressed to 3.49 MB, past the 3 MB free-plan
// limit, to colour five languages.
//
// The samples never change between builds, so the colouring is a build-time
// artifact like clidoc/cli.json. content/highlighted_gen.go holds it, and
// chroma is a test-only dependency that produces it — see highlight_test.go,
// which regenerates with -update and otherwise fails if the file is stale.

// CodeKey identifies a sample by what it is rather than where it is, so
// reordering pages does not invalidate the generated file.
func CodeKey(lang, src string) string {
	sum := sha256.Sum256([]byte(lang + "\x00" + src))
	return hex.EncodeToString(sum[:8])
}

// Highlighted returns the precomputed markup for a sample.
//
// A miss is not fatal: the renderer falls back to escaped plain text, so a
// forgotten regeneration costs colour rather than the example itself. The
// test is what makes sure that does not go unnoticed.
func Highlighted(lang, src string) (string, bool) {
	html, ok := highlighted[CodeKey(lang, src)]
	return html, ok
}
