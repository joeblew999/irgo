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
func TestEverythingIsFormatted(t *testing.T) {
	out, err := exec.Command("gofmt", "-l", "../../cmd", "../../pkg",
		"../../mobile", "../../desktop").Output()
	if err != nil {
		t.Skipf("gofmt unavailable: %v", err)
	}
	if files := strings.TrimSpace(string(out)); files != "" {
		t.Errorf("these files are not gofmt'd:\n%s\n\nrun: gofmt -w .", files)
	}
}
