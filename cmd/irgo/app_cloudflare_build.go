// Cloudflare Workers: the same Go handler, compiled to WebAssembly.
//
// A Worker runs JavaScript or WASM, not a binary, so the web target is built
// with GOOS=js GOARCH=wasm and served through a small shim that hands the
// Worker's fetch event to the Go http.Handler. Datastar's SSE survives the
// trip: each flush arrives as its own chunk, which is the only property that
// made this worth supporting at all.
//
// TinyGo is not an option even though it would produce a smaller module: its
// net/http on wasm targets is incomplete (tinygo-org/tinygo#4420), and irgo is
// net/http from the router down. Standard Go compiles the whole demo to about
// 2.4 MB compressed, inside even the free plan's 3 MB.
//
// What does NOT survive is memory between requests: the shim instantiates a
// fresh Go runtime per fetch, so a package-level variable is reset every time.
// That is the same bug a second replica would expose anywhere else, so the
// answer is to keep shared state in a store rather than to work around the
// platform.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// workersModule is the shim irgo builds against. Pinned: the generated
// wasm_exec.js has to match what the module expects, and a floating version
// would break builds without anything in this repository changing.
const workersModule = "github.com/syumai/workers"
const workersVersion = "v0.33.0"

const workerBuildDir = "build/worker"

func buildCloudflare(modulePath string) error {
	if err := excludeJSFromWebEntrypoint(); err != nil {
		return err
	}
	if err := ensureWorkerEntrypoint(modulePath); err != nil {
		return err
	}
	if err := ensureWorkersDependency(); err != nil {
		return err
	}
	if err := os.MkdirAll(workerBuildDir, 0o755); err != nil {
		return err
	}

	// The shim: worker.mjs, runtime.mjs and a wasm_exec.js matching it. Note
	// -mode=go — the generator defaults to tinygo, and mixing the tinygo shim
	// with a standard Go build fails at instantiation with a LinkError about
	// runtime.scheduleTimeoutEvent, which says nothing about the real cause.
	fmt.Println("Generating the Worker shim...")
	gen := goCommand("run", workersModule+"/cmd/workers-assets-gen@"+workersVersion,
		"-mode=go", "-o", workerBuildDir)
	gen.Stdout = os.Stdout
	gen.Stderr = os.Stderr
	if err := gen.Run(); err != nil {
		return fmt.Errorf("generating the Worker shim failed: %w", err)
	}

	fmt.Println("Building app.wasm (GOOS=js GOARCH=wasm)...")
	build := goCommand("build", "-o", filepath.Join(workerBuildDir, "app.wasm"), ".")
	build.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("wasm build failed: %w", err)
	}

	if err := writeWranglerConfig(baseName(modulePath)); err != nil {
		return err
	}
	writeArtifactStamp(workerBuildDir)
	reportWorkerSize()

	if !deploying {
		fmt.Println()
		fmt.Println("Deploy it:")
		fmt.Println("  irgo app deploy cloudflare")
	}
	return nil
}

// deploying suppresses the build's closing advice when a deploy is about to
// happen anyway — being told to run the command that is already running reads
// as though nothing happened.
var deploying bool

// ensureWorkerEntrypoint writes the js/wasm main if the project has none.
func ensureWorkerEntrypoint(modulePath string) error {
	const path = "main_cloudflare.go"
	if pathExists(path) {
		return nil
	}
	body := fmt.Sprintf(`//go:build js && wasm

// Entry point for Cloudflare Workers.
//
// The Worker hands each fetch event to the same router the web, desktop and
// mobile targets use, so there is one app and one set of handlers.
package main

import (
	"%s/app"

	"github.com/syumai/workers"
)

func main() {
	workers.Serve(app.NewRouter().Handler())
}
`, modulePath)
	fmt.Printf("  created: %s\n", path)
	return os.WriteFile(path, []byte(body), 0o644)
}

// ensureWorkersDependency adds the shim module when the project lacks it.
func ensureWorkersDependency() error {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return err
	}
	if strings.Contains(string(data), workersModule) {
		return nil
	}
	fmt.Printf("Adding %s@%s...\n", workersModule, workersVersion)
	get := goCommand("get", workersModule+"@"+workersVersion)
	get.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	if out, gerr := get.CombinedOutput(); gerr != nil {
		return fmt.Errorf("go get %s: %s", workersModule, strings.TrimSpace(string(out)))
	}
	return nil
}

// writeWranglerConfig seeds wrangler.toml once. It carries the Worker's name
// and any bindings, so regenerating it would discard configuration.
func writeWranglerConfig(name string) error {
	const path = "wrangler.toml"
	if pathExists(path) {
		return nil
	}
	body := fmt.Sprintf(`# Cloudflare Workers configuration.
#
# Written once by `+"`irgo app build cloudflare`"+` and yours thereafter — add
# bindings (KV, D1, Durable Objects) here.
#
# Shared state belongs in one of those bindings, not in a Go variable: every
# request gets a fresh Go runtime, so a package-level counter resets each time.
# Per-connection state inside a single SSE stream is fine, because that stream
# holds one instance open for its lifetime.
name = "%s"
main = "./%s/worker.mjs"
compatibility_date = "2025-08-03"
`, name, workerBuildDir)
	fmt.Printf("  created: %s\n", path)
	return os.WriteFile(path, []byte(body), 0o644)
}

// reportWorkerSize prints what the module weighs against the plan limits,
// because exceeding them is a deploy-time failure and the number is otherwise
// invisible until then.
func reportWorkerSize() {
	wasm := filepath.Join(workerBuildDir, "app.wasm")
	fi, err := os.Stat(wasm)
	if err != nil {
		return
	}
	gz, err := gzippedSize(wasm)
	if err != nil {
		fmt.Printf("\nBuilt %s (%.1f MB)\n", wasm, float64(fi.Size())/(1<<20))
		return
	}
	mb := float64(gz) / (1 << 20)
	fmt.Printf("\nBuilt %s — %.1f MB, %.2f MB compressed\n", wasm, float64(fi.Size())/(1<<20), mb)
	switch {
	case mb > 10:
		fmt.Println("  ! over the 10 MB limit on every plan — this will not deploy")
	case mb > 3:
		fmt.Println("  ! over the 3 MB free-plan limit (10 MB on paid)")
	default:
		fmt.Println("  within the 3 MB free-plan limit")
	}
}

func gzippedSize(path string) (int64, error) {
	// Compressed size is what Cloudflare measures, and gzip -9 is close enough
	// to it to warn before a deploy fails.
	cmd := exec.Command("gzip", "-9c", path)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	return int64(len(out)), nil
}

// excludeJSFromWebEntrypoint widens main.go's build constraint so the Worker
// entry point is not a second main.
//
// main.go is yours, and irgo does not rewrite your files — but a build
// constraint is not logic, and the alternative is "main redeclared in this
// block", which names neither the cause nor the fix. Projects created after
// this get the constraint from the template.
func excludeJSFromWebEntrypoint() error {
	const path = "main.go"
	data, err := os.ReadFile(path)
	if err != nil {
		return nil // no web entry point; nothing to exclude
	}
	body := string(data)
	if !strings.HasPrefix(body, "//go:build !desktop\n") {
		return nil // already constrained, or hand-edited — leave it alone
	}
	body = strings.Replace(body, "//go:build !desktop\n", "//go:build !desktop && !js\n", 1)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return err
	}
	fmt.Println("  updated: main.go — build tag now excludes js/wasm (the Worker has its own main)")
	return nil
}
