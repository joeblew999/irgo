// The whole app as a static site that runs in the browser with no server.
//
// Every other target already runs the router without a network: the WebView
// builds dispatch in-process, and the Worker build runs the same handlers in
// wasm. A browser tab is both of those at once — a JavaScript host that speaks
// fetch, running wasm — so this target is glue rather than a new architecture.
//
// The service worker is what makes it whole. Patching fetch inside the page
// only intercepts what the page's own script asks for, which is Datastar's
// requests and nothing else; stylesheets, module scripts and images are
// fetched by the browser's loader, which no patch reaches. A service worker
// sits in front of all of them, so the embedded static files are served out of
// the wasm binary and there is no origin left to be offline from.
//
// Output is a directory of static files. Any static host will serve it, and
// the free tiers of most of them will do it over https — which is required,
// because service workers only run in a secure context.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// webOutDir is where the bundle lands.
var webOutDir = filepath.Join("build", "web")

func buildWeb(args []string) error {
	if err := ensureAssets(); err != nil {
		return err
	}
	if err := ensureBrowserEntrypoint(); err != nil {
		return err
	}
	if err := os.MkdirAll(webOutDir, 0o755); err != nil {
		return err
	}

	name, err := getModulePath()
	if err != nil {
		name = "irgo app"
	}
	name = filepath.Base(name)

	fmt.Println("Building app.wasm (GOOS=js GOARCH=wasm -tags browser)...")
	out := filepath.Join(webOutDir, "app.wasm")
	// -s -w drops the symbol table and DWARF, -trimpath keeps this machine's
	// directory names out of the binary. Worth about 1.6% compressed, which is
	// not the reason to do it — the reason is that a wasm binary is downloaded
	// by strangers, and neither debug symbols nor local paths should be.
	build := exec.Command(goBin(), "build",
		"-tags", "browser", "-ldflags", "-s -w", "-trimpath", "-o", out, ".")
	build.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("building the browser binary: %w", err)
	}

	if err := copyWasmExec(webOutDir); err != nil {
		return err
	}
	for _, f := range []struct {
		name, body string
	}{
		{"index.html", webIndexHTML(name)},
		{"sw.js", webServiceWorkerJS()},
		{"manifest.webmanifest", webManifest(name)},
	} {
		if err := os.WriteFile(filepath.Join(webOutDir, f.name), []byte(f.body), 0o644); err != nil {
			return err
		}
	}

	report := ""
	if st, err := os.Stat(out); err == nil {
		report = fmt.Sprintf(" — %.1f MB", float64(st.Size())/(1<<20))
	}
	fmt.Printf("\nBuilt %s%s\n", webOutDir, report)
	fmt.Println()
	fmt.Println("Run it:")
	fmt.Println("  irgo app run web")
	fmt.Println()
	fmt.Println("Deploy it: it is static files, so any host will do. It must be")
	fmt.Println("served over https — service workers do not run without it, and")
	fmt.Println("without the service worker there is no app, only a loading page.")
	fmt.Println("localhost is exempt, which is why the dev server needs no")
	fmt.Println("certificate.")
	return nil
}

// copyWasmExec takes the toolchain's own wasm_exec.js.
//
// From this Go installation rather than a vendored copy: it is the other half
// of the wasm binary's ABI, and a mismatch fails at instantiation with a
// message about neither.
func copyWasmExec(dir string) error {
	var root string
	if out, err := exec.Command(goBin(), "env", "GOROOT").Output(); err == nil {
		root = strings.TrimSpace(string(out))
	}
	if root == "" {
		return fmt.Errorf("could not find GOROOT, so wasm_exec.js cannot be copied")
	}
	// Moved in Go 1.24; support both rather than pinning a layout.
	for _, p := range []string{
		filepath.Join(root, "lib", "wasm", "wasm_exec.js"),
		filepath.Join(root, "misc", "wasm", "wasm_exec.js"),
	} {
		if body, err := os.ReadFile(p); err == nil {
			return os.WriteFile(filepath.Join(dir, "wasm_exec.js"), body, 0o644)
		}
	}
	return fmt.Errorf("wasm_exec.js not found under %s", root)
}

// ensureBrowserEntrypoint writes the browser main if the project has none, and
// steps the Worker's main aside.
//
// Both are js/wasm, so without a distinguishing tag they are two mains in one
// package and nothing builds. The Worker's keeps the plain tag minus this one,
// so a project that never builds for the browser is unaffected.
func ensureBrowserEntrypoint() error {
	if err := excludeBrowserFromWorkerMain(); err != nil {
		return err
	}
	const path = "main_browser.go"
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	mod, err := getModulePath()
	if err != nil {
		return fmt.Errorf("could not determine the module path: %w", err)
	}

	body := `//go:build js && wasm && browser

// The whole app, in the browser, with no server.
//
// The same router as every other target. What differs is only who answers it:
// pkg/browser runs inside a service worker, so requests are served from this
// binary rather than sent anywhere.
package main

import (
	"net/http"

	"github.com/stukennedy/irgo/pkg/browser"

	"` + mod + `/app"
	"` + mod + `/static"
)

func main() {
	mux := http.NewServeMux()

	// The same embedded files the mobile builds carry. Served from here, so a
	// stylesheet does not need a network that may not be there.
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(static.Files))))
	mux.Handle("/", app.NewRouter().Handler())

	browser.ServeWorker(mux)
}
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return err
	}
	fmt.Println("  created: main_browser.go")
	return nil
}

// excludeBrowserFromWorkerMain narrows the Worker entrypoint's build tag.
func excludeBrowserFromWorkerMain() error {
	const path = "main_cloudflare.go"
	body, err := os.ReadFile(path)
	if err != nil {
		return nil // no Worker target in this project
	}
	const old = "//go:build js && wasm\n"
	if !strings.HasPrefix(string(body), old) {
		return nil // already narrowed, or written by hand
	}
	updated := "//go:build js && wasm && !browser\n" + strings.TrimPrefix(string(body), old)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return err
	}
	fmt.Println("  updated: main_cloudflare.go — build tag now excludes the browser build")
	return nil
}

func init() {
	registerTarget("app build", "web",
		"A static site that runs in the browser, in build/web", buildWeb)
	registerTarget("app run", "web", "Serve build/web on localhost", runWeb)
}
