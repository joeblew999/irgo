//go:build js && wasm && browser

// The whole app, in the browser, with no server.
//
// The same router as every other target. What differs is only who answers it:
// pkg/browser runs inside a service worker, so requests are served from this
// binary rather than sent anywhere.
package main

import (
	"net/http"

	"github.com/stukennedy/irgo/pkg/browser"

	"github.com/stukennedy/irgo/app"
	"github.com/stukennedy/irgo/static"
)

func main() {
	mux := http.NewServeMux()

	// The same embedded files the mobile builds carry. Served from here, so a
	// stylesheet does not need a network that may not be there.
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(static.Files))))
	mux.Handle("/", app.NewRouter().Handler())

	browser.ServeWorker(mux)
}
