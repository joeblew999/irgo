// Core commands: the dev server, plain serve, and the test runner.
package main

import (
	"fmt"
	"os"
)

// runDev starts the development server with hot reload
func runDev() error {
	// Check for required tools
	if err := ensureGoTool("air"); err != nil {
		return err
	}
	if err := ensureGoTool("templ"); err != nil {
		return err
	}

	// dev.sh watches files with entr — provision it rather than failing later
	// with a bare "entr: command not found" from inside the script.
	if _, err := os.Stat("dev.sh"); err == nil {
		if err := ensureOSPackage("entr"); err != nil {
			return err
		}
	}

	// Check if dev.sh exists (user project) or we're in framework
	if _, err := os.Stat("dev.sh"); err == nil {
		// User project - run dev.sh
		return runCommand("./dev.sh")
	}

	// Framework development - run air directly
	fmt.Println("Starting development server...")

	// Generate templ files first
	if err := runTempl(); err != nil {
		fmt.Printf("Warning: templ generate failed: %v\n", err)
	}

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
