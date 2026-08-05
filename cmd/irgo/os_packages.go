// Host OS packages: the native dependencies irgo cannot install with `go
// install` — entr for hot reload, mingw-w64 for Windows cross-compilation,
// GTK3 + WebKit2GTK for Linux desktop builds.
//
// These used to be a per-project shell task that every consumer re-implemented
// for brew/apt/pacman. Installing them is the build's own business, so each is
// provisioned at the point of need and removable again: a machine that cannot
// be returned to a known state hides provisioning bugs instead of surfacing
// them on the next run.
package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// osPackage is one native dependency, named per package manager because the
// same thing is called something different on each.
type osPackage struct {
	// key is the stable name used by irgo (flags, markers, messages).
	key string
	// brew/apt/pacman hold the package name for that manager, empty when the
	// dependency does not apply there.
	brew, apt, pacman string
	// probe reports whether the dependency is already satisfied. Checking the
	// real capability rather than the package name means a dependency
	// installed by any other route is respected.
	probe func() bool
	// why is shown when the package has to be installed.
	why string
}

func osPackages() []osPackage {
	return []osPackage{
		{
			key: "entr", brew: "entr", apt: "entr", pacman: "entr",
			probe: func() bool { _, err := exec.LookPath("entr"); return err == nil },
			why:   "file watching for `irgo dev` hot reload",
		},
		{
			key: "mingw-w64", brew: "mingw-w64", pacman: "mingw-w64-x86_64-gcc",
			probe: func() bool {
				_, err := exec.LookPath("x86_64-w64-mingw32-gcc")
				return err == nil
			},
			why: "cross-compiling the Windows desktop app",
		},
		{
			key: "webkit2gtk", apt: "libwebkit2gtk-4.0-dev libgtk-3-dev",
			probe: func() bool {
				return exec.Command("pkg-config", "--exists", "webkit2gtk-4.0").Run() == nil
			},
			why: "the Linux desktop webview",
		},
	}
}

// pkgManager reports the host package manager irgo will drive, or "" when none
// is available.
func pkgManager() string {
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("brew"); err == nil {
			return "brew"
		}
	case "linux":
		if _, err := exec.LookPath("apt-get"); err == nil {
			return "apt"
		}
	case "windows":
		if _, err := exec.LookPath("pacman"); err == nil {
			return "pacman"
		}
	}
	return ""
}

// pkgNameFor returns the package name for the host's manager, or "" when the
// dependency does not apply on this platform.
func (p osPackage) pkgNameFor(mgr string) string {
	switch mgr {
	case "brew":
		return p.brew
	case "apt":
		return p.apt
	case "pacman":
		return p.pacman
	}
	return ""
}

func findOSPackage(key string) (osPackage, bool) {
	for _, p := range osPackages() {
		if p.key == key {
			return p, true
		}
	}
	return osPackage{}, false
}

// ensureOSPackage installs a native dependency when its probe fails. It returns
// an actionable error rather than installing silently-wrong things when no
// package manager is available, so the caller always learns the real command.
func ensureOSPackage(key string) error {
	p, ok := findOSPackage(key)
	if !ok {
		return fmt.Errorf("unknown OS package: %s", key)
	}
	if p.probe() {
		return nil
	}
	mgr := pkgManager()
	name := p.pkgNameFor(mgr)
	if mgr == "" || name == "" {
		return fmt.Errorf("%s is needed for %s, but irgo cannot install it on %s — install it manually",
			p.key, p.why, runtime.GOOS)
	}

	fmt.Printf("Installing %s (%s) via %s...\n", p.key, p.why, mgr)
	if err := runCommand(pkgInstallCmd(mgr, name)[0], pkgInstallCmd(mgr, name)[1:]...); err != nil {
		return fmt.Errorf("installing %s via %s failed: %w", p.key, mgr, err)
	}
	if !p.probe() {
		return fmt.Errorf("%s still not usable after installing it via %s", p.key, mgr)
	}
	markToolInstalled(p.key)
	return nil
}

// pkgInstallCmd builds the install command. apt needs sudo and non-interactive
// flags because this runs inside builds and CI, not a terminal session.
func pkgInstallCmd(mgr, name string) []string {
	fields := strings.Fields(name)
	switch mgr {
	case "brew":
		return append([]string{"brew", "install"}, fields...)
	case "apt":
		return append([]string{"sudo", "apt-get", "install", "-y"}, fields...)
	case "pacman":
		return append([]string{"pacman", "-S", "--noconfirm", "--needed"}, fields...)
	}
	return nil
}

func pkgRemoveCmd(mgr, name string) []string {
	fields := strings.Fields(name)
	switch mgr {
	case "brew":
		return append([]string{"brew", "uninstall", "--formula"}, fields...)
	case "apt":
		return append([]string{"sudo", "apt-get", "remove", "-y"}, fields...)
	case "pacman":
		return append([]string{"pacman", "-R", "--noconfirm"}, fields...)
	}
	return nil
}

// installOSPackages eagerly provisions every native dependency this host can
// use. Builds do this lazily; this is for setting a machine up in one go.
func installOSPackages() error {
	mgr := pkgManager()
	if mgr == "" {
		fmt.Printf("No supported package manager found on %s — install native deps manually.\n", runtime.GOOS)
		return nil
	}
	fmt.Printf("Installing host packages via %s...\n", mgr)
	for _, p := range osPackages() {
		if p.pkgNameFor(mgr) == "" {
			continue // not applicable on this platform
		}
		if p.probe() {
			fmt.Printf("  %s: already present\n", p.key)
			continue
		}
		if err := ensureOSPackage(p.key); err != nil {
			fmt.Printf("  Warning: %v\n", err)
		} else {
			fmt.Printf("  %s: installed\n", p.key)
		}
	}
	return nil
}

// uninstallOSPackages is the inverse of installOSPackages. Marker-guarded: a
// package irgo did not install is kept, since it may predate irgo or be
// something the developer relies on elsewhere.
func uninstallOSPackages(all bool) error {
	mgr := pkgManager()
	if mgr == "" {
		return nil
	}
	for _, p := range osPackages() {
		name := p.pkgNameFor(mgr)
		if name == "" || !p.probe() {
			continue
		}
		if !all && !toolInstalledByIrgo(p.key) {
			fmt.Printf("  %s: kept (not installed by irgo — use --all to remove anyway)\n", p.key)
			continue
		}
		fmt.Printf("  %s: removing via %s...\n", p.key, mgr)
		if err := runCommand(pkgRemoveCmd(mgr, name)[0], pkgRemoveCmd(mgr, name)[1:]...); err != nil {
			fmt.Printf("  Warning: removing %s failed: %v\n", p.key, err)
		}
	}
	return nil
}
