// Package datastarjs embeds the Datastar client and serves it at
// GET /_irgo/datastar.js.
//
// Datastar has two halves that speak a protocol to each other: the Go library
// that writes SSE events, and the browser client that applies them. They were
// two unrelated dependencies here — the Go side from go.mod, the client
// downloaded from a CDN when a project was scaffolded and then vendored into
// that project forever. Nothing connected them, and datastar-go publishes no
// version constant and embeds no bundle, so nothing could.
//
// The result was a release candidate of the client running against a stable
// v1.1.0 of the library in every generated project, plus a third copy in the
// framework's own static directory, plus a script tag elsewhere pointing at an
// unversioned CDN URL — whatever upstream had published that morning. They
// agreed by luck: the wire format happens not to have changed.
//
// Embedding it here makes the pairing structural. Both halves upgrade together
// with the framework, a project downloads nothing when it is created, and a
// project whose client and library disagree stops being expressible.
package datastarjs

import (
	_ "embed"
	"net/http"
	"strconv"
)

// Version is the Datastar client vendored here.
//
// It is a matched pair with the github.com/starfederation/datastar-go
// requirement in go.mod. Bump them together, and check the wire format when
// you do: the event names the library writes and the client listens for are
// the contract, and nothing but this comment enforces it.
const Version = "v1.0.2"

//go:embed datastar.js
var script []byte

// Script returns the Datastar client source, for builds that embed it rather
// than serve it — the mobile shells read it through the same bridge as
// everything else.
func Script() []byte { return script }

// Handler serves the client.
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(script)))
	// Versioned with the framework, so it can be cached hard — unlike the
	// bridge, which is small and changes with every irgo release.
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write(script)
}
