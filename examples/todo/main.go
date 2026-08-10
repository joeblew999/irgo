//go:build !desktop && !js

package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/stukennedy/irgo/pkg/livereload"
	"todo/app"
	"todo/templates"
)

func main() {
	// Check if running as desktop dev server
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		runDevServer()
		return
	}

	// Default: show usage
	fmt.Println("todo - built with irgo")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run . serve       Start development server")
	fmt.Println("  irgo server dev             Start dev server with hot reload")
	fmt.Println("  irgo app run desktop     Run as desktop app")
	fmt.Println("  irgo app run ios         Build and run on iOS Simulator")
	fmt.Println("  irgo app run android     Build and run on Android Emulator")
}

// runDevServer starts an HTTP server for development with live reload
func runDevServer() {
	// Enable dev mode for templates (enables live reload script)
	templates.DevMode = true

	r := app.NewRouter()
	lr := livereload.New()

	// Set up mux with live reload endpoint
	handler := r.Handler()
	mux := http.NewServeMux()
	mux.HandleFunc("/dev/livereload", lr.Handler())
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/", handler)

	// 8080 unless PORT says otherwise, and deliberately not a free port chosen
	// at random: `irgo app run android --dev` builds a shell that reaches this
	// machine at 10.0.2.2:8080, and the iOS one uses the LAN address on the
	// same port. A server that quietly moved would leave those looking at a
	// blank WebView with nothing to explain it.
	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		if !strings.HasPrefix(p, ":") {
			p = ":" + p
		}
		addr = p
	}

	// Listen first, so the port is reported after it is actually held rather
	// than before. net.Listen fails the same way on every platform; matching on
	// the error number would not, because Windows raises WSAEADDRINUSE where
	// Unix raises EADDRINUSE.
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("cannot serve on %s: %v\n\n"+
			"Something else is already using it — another irgo project, most\n"+
			"likely. Pick a different one:\n\n"+
			"  PORT=8081 go run . serve          (macOS, Linux)\n"+
			"  $env:PORT=8081; go run . serve    (PowerShell)\n"+
			"  set PORT=8081 && go run . serve   (cmd.exe)\n\n"+
			"Mobile dev expects 8080: `irgo app run android|ios --dev` builds the\n"+
			"shell to reach this machine on that port, so free it up before those.",
			addr, err)
	}

	fmt.Printf("Starting dev server at http://localhost%s\n", addr)
	fmt.Printf("Live reload enabled (build time: %d)\n", lr.BuildTime())
	log.Fatal(http.Serve(ln, mux))
}
