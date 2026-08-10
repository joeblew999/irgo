package main

import (
	"os/exec"
	"strings"
	"testing"
)

// TestEverythingIsFormatted — so `go test ./...` means what CI means.
//
// CI checked formatting as a separate step, which made `mise run check` weaker
// than the thing it exists to predict: it could pass locally and fail on the
// first push, over whitespace. A check that does not match CI teaches people
// to push and wait rather than to run it.
//
// In a test rather than a task, because a task would need a shell pipeline to
// turn gofmt's output into a failure — and shell pipelines are what stopped
// this working on Windows the last three times.
//
// The repository root rather than a list of directories. The list said cmd,
// pkg, mobile, desktop — and examples/ and docs-templ/ were not on it, so nine
// checked-in files sat unformatted with the formatting test passing. A list of
// places to look is a thing to update every time a directory is added, which
// means it is a thing that goes stale silently. Everything, and no list.
func TestEverythingIsFormatted(t *testing.T) {
	out, err := exec.Command("gofmt", "-l", "../..").Output()
	if err != nil {
		t.Skipf("gofmt unavailable: %v", err)
	}
	if files := strings.TrimSpace(string(out)); files != "" {
		t.Errorf("these files are not gofmt'd:\n%s\n\nrun: gofmt -w .", files)
	}
}
