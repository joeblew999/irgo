package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestScaffoldedProjectBuilds generates a project and compiles it.
//
// The other tests cover the CLI's own logic; none of them check that what it
// writes is valid Go, which is the first failure a person meets. This lived in
// a CI workflow as shell for a while, which meant it could only ever run in
// CI — so it could drift from the CLI without anyone noticing until a push.
//
// Skipped by -short: it downloads modules and tools.
func TestScaffoldedProjectBuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("scaffolds a project and builds it — needs the network")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go toolchain on PATH")
	}

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err != nil {
		t.Skipf("cannot find the repository root from %s", repoRoot)
	}

	work := t.TempDir()
	irgo := filepath.Join(work, "irgo")
	run := func(dir, name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, out)
		}
	}

	run(repoRoot, goBin, "build", "-o", irgo, "./cmd/irgo")

	proj := filepath.Join(work, "src")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	run(proj, irgo, "project", "new", "myapp")

	app := filepath.Join(proj, "myapp")
	// Build against this checkout rather than a published release, so the test
	// exercises the code under test instead of whatever is tagged.
	run(app, goBin, "mod", "edit", "-replace", upstreamModule+"="+repoRoot)
	run(app, goBin, "mod", "tidy")
	run(app, irgo, "project", "assets")
	run(app, goBin, "build", "./...")
	run(app, goBin, "test", "./...")
}
