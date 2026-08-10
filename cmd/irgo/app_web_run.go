// Serving the browser bundle locally.
//
// A plain static file server, with one thing it must get right: correct
// content types. A service worker registration is rejected outright if sw.js
// is served as text/plain, and wasm instantiation refuses anything but
// application/wasm — both fail with messages about MIME types rather than
// about the app, which is a long way from the cause.
//
// http rather than https on purpose. Service workers need a secure context and
// browsers treat localhost as one, so development needs no certificate and no
// warning to click through. Deployment is the case that needs real https.
package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func runWeb(args []string) error {
	if _, err := os.Stat(filepath.Join(webOutDir, "app.wasm")); err != nil {
		fmt.Println("No browser build yet — building it first.")
		if err := buildWeb(nil); err != nil {
			return err
		}
		fmt.Println()
	}

	addr := devPort()
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("cannot serve on %s: %w\n"+
			"  Something else is using it. PORT=8081 irgo app run web", addr, err)
	}

	fmt.Printf("Serving %s at http://localhost%s\n", webOutDir, addr)
	fmt.Println()
	fmt.Println("The first load registers a service worker and reloads once.")
	fmt.Println("After that the page, its styles and its data all come from the")
	fmt.Println("wasm binary — you can stop this server and it keeps working.")

	return http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveWebFile(w, r)
	}))
}

// serveWebFile serves the bundle with the content types the browser demands.
func serveWebFile(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" {
		name = "index.html"
	}
	path := filepath.Join(webOutDir, filepath.Clean("/"+name))

	if st, err := os.Stat(path); err != nil || st.IsDir() {
		// Not a real file. Everything that is not part of the bundle belongs to
		// the service worker, and the only way the request reaches here is if
		// no worker is controlling yet — so hand back the page that registers
		// one rather than a 404 the user cannot act on.
		http.ServeFile(w, r, filepath.Join(webOutDir, "index.html"))
		return
	}

	switch filepath.Ext(path) {
	case ".wasm":
		// Required for instantiateStreaming, which refuses any other type.
		w.Header().Set("Content-Type", "application/wasm")
	case ".js":
		w.Header().Set("Content-Type", "text/javascript")
	case ".webmanifest":
		w.Header().Set("Content-Type", "application/manifest+json")
	}

	// The worker script must not be cached, or a rebuilt app keeps being
	// served by the previous worker until the cache expires.
	if strings.HasSuffix(path, "sw.js") {
		w.Header().Set("Cache-Control", "no-cache")
	}

	http.ServeFile(w, r, path)
}
