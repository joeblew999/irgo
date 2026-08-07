//go:build js && wasm

// Entry point for Cloudflare Workers.
//
// The Worker hands each fetch event to the same router the web, desktop and
// mobile targets use, so there is one app and one set of handlers.
package main

import (
	"docs-templ/app"

	"github.com/syumai/workers"
)

func main() {
	workers.Serve(app.NewRouter().Handler())
}
