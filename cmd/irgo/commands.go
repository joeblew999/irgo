// Core commands: the dev server, plain serve, and the test runner.
package main

import (
	"fmt"
	"os"
)

// runDev starts the development server with hot reload.
//
// air rebuilds on change using .air.toml, which calls `irgo assets` — so the
// generate step lives in one place rather than being restated per project.
func runDev() error {
	if err := ensureGoTool("air"); err != nil {
		return err
	}
	if err := ensureGoTool("templ"); err != nil {
		return err
	}
	// Generate once up front so the first build is correct; air regenerates on
	// every change after that.
	if err := ensureAssets(); err != nil {
		return err
	}

	if _, err := os.Stat("main.go"); err != nil {
		// Framework checkout rather than a generated project.
		if _, e := os.Stat("examples/todo/main.go"); e != nil {
			return fmt.Errorf("no main.go found - are you in an irgo project?")
		}
	}

	fmt.Println("Starting development server (http://localhost:8080)...")
	return runCommand("air")
}

// runServe starts the server without file watching
func runServe() error {
	// Check if main.go exists
	if _, err := os.Stat("main.go"); err == nil {
		// User project
		return runCommand(goBin(), "run", ".", "serve")
	}

	// Framework - run example
	if _, err := os.Stat("examples/todo/main.go"); err == nil {
		return runCommand(goBin(), "run", "./examples/todo", "serve")
	}

	return fmt.Errorf("no main.go found - are you in an irgo project?")
}

// runTest runs the test suite
func runTest() error {
	fmt.Println("Running tests...")
	return runCommand(goBin(), "test", "-v", "./...")
}
