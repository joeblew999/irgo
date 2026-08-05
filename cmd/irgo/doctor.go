// Host capability reporting: what this machine can build, what it cannot, and
// why. The CLI is the authority on this — not documentation, which goes stale
// and cannot see the machine it is describing.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// capability is one build target and this host's verdict on it.
type capability struct {
	target string
	// state is one of: ready (buildable now), fixable (buildable once
	// something installs — irgo does that automatically), blocked (this OS
	// cannot produce it at all).
	state string
	note  string
}

const (
	capReady   = "ready"
	capFixable = "auto-installs"
	capBlocked = "NOT ON THIS OS"
)

// hostCapabilities evaluates every target against the current host. Blocked
// entries are the ones that matter most: they can never succeed here, no
// matter what gets installed, so a dev needs to know before trying.
func hostCapabilities() []capability {
	var caps []capability
	has := func(bin string) bool { _, err := exec.LookPath(bin); return err == nil }

	caps = append(caps, capability{"web", capReady, "go build — works everywhere"})

	// Desktop: only Windows cross-compiles, and only from macOS.
	buildable := map[string]bool{}
	for _, t := range desktopTargetsForHost() {
		buildable[t] = true
	}
	switch {
	case buildable["darwin"]:
		note := "native"
		if !has("clang") {
			note = "needs Xcode Command Line Tools: xcode-select --install"
		}
		caps = append(caps, capability{"desktop (macOS)", capReady, note})
	default:
		caps = append(caps, capability{"desktop (macOS)", capBlocked,
			"requires macOS (Xcode) — cannot cross-compile from " + runtime.GOOS})
	}
	if buildable["windows"] {
		state, note := capReady, "native"
		if runtime.GOOS == "darwin" {
			note = "cross-compile via mingw-w64"
			if !has("x86_64-w64-mingw32-gcc") {
				state = capFixable
				note = "cross-compile — irgo installs mingw-w64 on first build"
			}
		}
		caps = append(caps, capability{"desktop (Windows)", state, note})
	} else {
		caps = append(caps, capability{"desktop (Windows)", capBlocked,
			"buildable from Windows, or macOS with mingw-w64 — not " + runtime.GOOS})
	}
	if buildable["linux"] {
		state, note := capReady, "native"
		if exec.Command("pkg-config", "--exists", "webkit2gtk-4.0").Run() != nil {
			state, note = capFixable, "irgo installs GTK3 + WebKit2GTK on first build"
		}
		caps = append(caps, capability{"desktop (Linux)", state, note})
	} else {
		caps = append(caps, capability{"desktop (Linux)", capBlocked,
			"requires Linux (GTK3 + WebKit2GTK) — cannot cross-compile from " + runtime.GOOS})
	}

	// iOS is macOS-only, and Xcode is the one thing irgo cannot install.
	if runtime.GOOS == "darwin" {
		state, note := capReady, "Xcode present"
		if !has("xcodebuild") {
			state, note = capBlocked, "install Xcode from the App Store (irgo cannot install it)"
		}
		caps = append(caps,
			capability{"ios (framework)", state, note},
			capability{"ios (simulator app)", state, note},
			capability{"ios (device/App Store)", state, strings.TrimSpace(note + " — needs a signing team: --team ID")},
		)
	} else {
		blocked := "requires macOS (Xcode) — cannot cross-compile from " + runtime.GOOS
		caps = append(caps,
			capability{"ios (framework)", capBlocked, blocked},
			capability{"ios (simulator app)", capBlocked, blocked},
			capability{"ios (device/App Store)", capBlocked, blocked},
		)
	}

	// Android builds anywhere; only the emulator has host limits.
	caps = append(caps, capability{"android (AAR/APK)", capFixable,
		"irgo installs the JDK, SDK and NDK on first build — Android Studio not needed"})
	if err := ensureEmulatorSupported(); err != nil {
		caps = append(caps, capability{"android (emulator)", capBlocked, firstLine(err.Error())})
	} else {
		caps = append(caps, capability{"android (emulator)", capFixable,
			"irgo installs the emulator and creates the AVD on first run"})
	}

	return caps
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

// doctorHost prints the capability report. Exit status stays zero: a target
// this OS cannot build is a fact about the machine, not a failure.
func doctorHost() error {
	fmt.Printf("irgo %s — what this machine can build\n\n", version)
	fmt.Printf("Host: %s/%s\n\n", runtime.GOOS, runtime.GOARCH)

	caps := hostCapabilities()
	w := 0
	for _, c := range caps {
		if len(c.target) > w {
			w = len(c.target)
		}
	}
	for _, c := range caps {
		fmt.Printf("  %-*s  %-15s  %s\n", w, c.target, c.state, c.note)
	}

	var blocked []string
	for _, c := range caps {
		if c.state == capBlocked {
			blocked = append(blocked, c.target)
		}
	}
	fmt.Println()
	if len(blocked) == 0 {
		fmt.Println("Every target is buildable here.")
	} else {
		fmt.Printf("Not possible on %s: %s\n", runtime.GOOS, strings.Join(blocked, ", "))
		fmt.Println("Build those on a matching host or in CI — the repo's workflow covers")
		fmt.Println("Linux, macOS and Windows. `irgo build all` skips what it cannot do.")
	}
	checkPinDrift()

	fmt.Println()
	fmt.Println("Toolchain detail: irgo doctor android")
	return nil
}

// checkPinDrift reports when the CLI pin in the environment disagrees with the
// replace directive in go.mod. Those two must match: IRGO_REPLACE selects the
// CLI, and `irgo new` writes it into go.mod. When they drift, builds silently
// run against a different CLI than the project expects — which is exactly how
// a stale pin hides for several releases.
func checkPinDrift() {
	want := strings.TrimSpace(os.Getenv("IRGO_REPLACE"))
	if want == "" {
		return
	}
	want = strings.Replace(want, "@", " ", 1)
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "replace github.com/stukennedy/irgo") {
			continue
		}
		got := strings.TrimSpace(line[strings.Index(line, "=>")+2:])
		if got != want {
			fmt.Println()
			fmt.Println("WARNING: CLI pin drift")
			fmt.Printf("  IRGO_REPLACE : %s\n", want)
			fmt.Printf("  go.mod       : %s\n", got)
			fmt.Println("  These must match. Regenerate the app (irgo new <name>) or fix IRGO_REPLACE.")
		}
		return
	}
}
