package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestOldNodeIsSkippedNotUsed — a developer's own node is preferred, but node
// 18 is still common and wrangler 4 refuses it. Taking it anyway produces a
// failure from inside npx that reads like a Cloudflare problem.
//
// irgo must skip it and look elsewhere. It must never replace it: what is on
// the machine is the developer's business.
func TestOldNodeIsSkippedNotUsed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub is POSIX")
	}
	dir := t.TempDir()
	fake := filepath.Join(dir, "node")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho v18.20.0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if major, ok := nodeMajor(fake); !ok || major != 18 {
		t.Fatalf("nodeMajor read %d, %v — want 18", major, ok)
	}
	if nodeWorks(fake) {
		t.Errorf("node 18 accepted; wrangler %d+ is required", minNodeMajor)
	}

	newer := filepath.Join(dir, "newnode")
	if err := os.WriteFile(newer, []byte("#!/bin/sh\necho v22.14.0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !nodeWorks(newer) {
		t.Error("node 22 rejected")
	}
}

// TestNodeMajorHandlesJunk — a shim that prints an error, or nothing, must not
// read as a usable node.
func TestNodeMajorHandlesJunk(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub is POSIX")
	}
	dir := t.TempDir()
	for name, body := range map[string]string{
		"empty":   "#!/bin/sh\n",
		"noV":     "#!/bin/sh\necho 22.14.0\n",
		"shimErr": "#!/bin/sh\necho 'no version set' >&2; exit 1\n",
		"garbage": "#!/bin/sh\necho vBANANA\n",
	} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, ok := nodeMajor(p); ok {
			t.Errorf("%s was read as a usable node", name)
		}
	}
	if _, ok := nodeMajor(filepath.Join(dir, "does-not-exist")); ok {
		t.Error("a missing file was read as a usable node")
	}
}
