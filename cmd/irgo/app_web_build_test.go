package main

import (
	"os"
	"strings"
	"testing"
)

// TestBrowserPackageIsConstrained — pkg/browser imports syscall/js, which only
// exists for GOOS=js. Without a build constraint the package is compiled on
// every platform and `go build ./...` fails for everyone, including projects
// that never build for the browser.
//
// The failure names syscall/js and a Go source directory, so it reads as a
// broken toolchain rather than a missing line in this repository.
func TestBrowserPackageIsConstrained(t *testing.T) {
	body, err := os.ReadFile("../../pkg/browser/browser.go")
	if err != nil {
		t.Fatal(err)
	}
	first := strings.SplitN(string(body), "\n", 2)[0]
	if !strings.HasPrefix(first, "//go:build js && wasm") {
		t.Errorf("pkg/browser/browser.go must start with a js && wasm build "+
			"constraint, got %q", first)
	}
}
