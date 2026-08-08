package main

import (
	"os/exec"
	"strings"
	"testing"
)

// TestEveryToolIrgoRunsIsLocated — doctor is how a developer finds out where
// everything on their machine is. A tool the CLI runs but doctor never names
// is one nobody can locate when a build picks up the wrong copy.
func TestEveryToolIrgoRunsIsLocated(t *testing.T) {
	located := map[string]bool{}
	for _, r := range toolLocators() {
		located[r.name] = true
	}

	// Named differently by doctor than by the binary, on purpose: "jdk" is
	// clearer than "java" next to the Android rows, and tailwindcss is
	// installed under a versioned filename.
	alias := map[string]string{"java": "jdk", "keytool": "jdk"}

	for _, name := range toolsIrgoRuns() {
		if a, ok := alias[name]; ok {
			name = a
		}
		if !located[name] {
			t.Errorf("irgo runs %s but doctor never says where it is", name)
		}
	}
}

// TestLocatedToolsResolveOrReportAbsent — a row must either carry a real path
// or admit it has none. A path that does not exist is worse than a blank,
// because it sends someone to look at a file that is not there.
func TestLocatedToolsResolveOrReportAbsent(t *testing.T) {
	for _, r := range toolLocators() {
		if r.path == "" {
			continue
		}
		if !strings.HasPrefix(r.path, "/") && !strings.Contains(r.path, ":\\") {
			t.Errorf("%s has a relative path %q — doctor should report where it actually is", r.name, r.path)
		}
	}
}

// toolsIrgoRuns reads the binaries this package actually invokes, so the check
// above compares doctor against the code rather than against another list.
func toolsIrgoRuns() []string {
	out, err := exec.Command("sh", "-c",
		`grep -rhoE '(exec\.Command|LookPath)\("[a-zA-Z0-9_.-]+"' *.go | sed -E 's/.*\("//;s/"//' | sort -u`).Output()
	if err != nil {
		return nil
	}
	// Only the ones irgo is responsible for having. Host utilities like sudo,
	// open or apt-get are not toolchain, and naming them would be noise.
	want := map[string]bool{
		"templ": true, "air": true, "gomobile": true, "gobind": true,
		"node": true, "java": true, "keytool": true, "adb": true,
		"sdkmanager": true, "avdmanager": true, "go": true, "git": true,
		"sops": true, "xcrun": true, "xcodebuild": true, "codesign": true,
		"security": true, "clang": true,
	}
	var names []string
	for _, n := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if want[n] {
			names = append(names, n)
		}
	}
	return names
}
