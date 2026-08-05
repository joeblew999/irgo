// Toolchain: installing, verifying and removing the tools a build needs
// (templ, air, gomobile/gobind, mingw-w64) plus the generated assets they
// produce. Every install here has a matching uninstall.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// runTempl generates templ files
func runTempl() error {
	if err := ensureGoTool("templ"); err != nil {
		return err
	}

	fmt.Println("Generating templ files...")
	return runCommand("templ", "generate")
}

// goToolPkg maps a tool name to the module to `go install`. templ is pinned to
// the project's go.mod version: the generator and the library must match, and
// @latest drift breaks generated code.
func goToolPkg(name string) string {
	switch name {
	case "templ":
		if v := templVersionFromGoMod(); v != "" {
			return "github.com/a-h/templ/cmd/templ@v" + v
		}
		return "github.com/a-h/templ/cmd/templ@latest"
	case "air":
		return "github.com/air-verse/air@latest"
	case "gomobile":
		return "golang.org/x/mobile/cmd/gomobile@latest"
	case "gobind":
		return "golang.org/x/mobile/cmd/gobind@latest"
	}
	return ""
}

// irgoToolsDir holds one marker file per tool irgo installed itself, so
// uninstall can remove exactly those and leave a developer's own copies alone.
// Without it an uninstall would either delete tools irgo never owned or, if it
// played safe and skipped them, leave a half-provisioned machine that silently
// fails to re-provision.
func irgoToolsDir() string {
	return filepath.Join(homeDir(), ".irgo", "tools")
}

func markToolInstalled(name string) {
	dir := irgoToolsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, name), []byte("irgo "+version+"\n"), 0o644)
}

func toolInstalledByIrgo(name string) bool {
	_, err := os.Stat(filepath.Join(irgoToolsDir(), name))
	return err == nil
}

// ensureGoTool installs a Go-based tool when missing rather than printing an
// install command for someone to copy. `go install` behaves the same on macOS,
// Linux and Windows, so this needs no per-OS branching.
func ensureGoTool(name string) error {
	if _, err := exec.LookPath(name); err == nil {
		return nil
	}
	pkg := goToolPkg(name)
	if pkg == "" {
		return fmt.Errorf("%s not found, and no install source is known for it", name)
	}
	fmt.Printf("%s not found — installing %s...\n", name, pkg)
	if err := runCommand(goBin(), "install", pkg); err != nil {
		return fmt.Errorf("installing %s: %w", name, err)
	}
	markToolInstalled(name)
	// `go install` lands in GOBIN (default $GOPATH/bin), which is often absent
	// from PATH — prepend it so the tool resolves for the rest of this process.
	if dir := gobinDir(); dir != "" {
		_ = os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s still not found after installing it — is %s on your PATH?", name, gobinDir())
	}
	return nil
}

// jsRunner returns the package runner to drive package.json scripts with,
// preferring bun and falling back to npm. Empty when neither is installed.
func jsRunner() string {
	for _, r := range []string{"bun", "npm"} {
		if _, err := exec.LookPath(r); err == nil {
			return r
		}
	}
	return ""
}

// runCSS rebuilds the Tailwind stylesheet. static/css/output.css is generated
// and gitignored, but it is embedded into every build — so skipping it ships an
// unstyled app. No-op for projects without a "css" script.
func runCSS() error {
	data, err := os.ReadFile("package.json")
	if err != nil || !strings.Contains(string(data), `"css"`) {
		return nil // not a Tailwind project
	}
	runner := jsRunner()
	if runner == "" {
		fmt.Println("  (skipping CSS: neither bun nor npm found)")
		return nil
	}
	if _, err := os.Stat("node_modules"); os.IsNotExist(err) {
		fmt.Printf("Installing frontend dependencies (%s install)...\n", runner)
		if err := runCommand(runner, "install"); err != nil {
			return fmt.Errorf("%s install failed: %w", runner, err)
		}
	}
	fmt.Println("Building CSS...")
	return runCommand(runner, "run", "css")
}

// ensureAssets regenerates everything that is gitignored yet embedded into a
// build: _templ.go and the Tailwind stylesheet. Every build path runs this, so
// a fresh clone builds correctly without the caller sequencing it by hand.
func ensureAssets() error {
	if err := runTempl(); err != nil {
		return err
	}
	return runCSS()
}

// installTools installs required development tools
func installTools() error {
	fmt.Println("Installing irgo development tools...")
	fmt.Println()

	// Pin the templ generator to the templ library version in the project's
	// go.mod (they must stay in sync — @latest drift breaks generated code).
	templPkg := "github.com/a-h/templ/cmd/templ@latest"
	if v := templVersionFromGoMod(); v != "" {
		templPkg = "github.com/a-h/templ/cmd/templ@v" + v
		fmt.Printf("  templ: pinning to go.mod version v%s\n", v)
	}

	tools := []struct {
		name string
		pkg  string
	}{
		{"templ", templPkg},
		{"air", "github.com/air-verse/air@latest"},
		{"gomobile", "golang.org/x/mobile/cmd/gomobile@latest"},
	}

	for _, tool := range tools {
		if _, err := exec.LookPath(tool.name); err != nil {
			fmt.Printf("Installing %s...\n", tool.name)
			cmd := exec.Command(goBin(), "install", tool.pkg)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Printf("  Warning: failed to install %s: %v\n", tool.name, err)
			} else {
				markToolInstalled(tool.name)
				fmt.Printf("  %s: installed\n", tool.name)
			}
		} else {
			fmt.Printf("  %s: already installed\n", tool.name)
		}
	}

	// Initialize gomobile
	fmt.Println()
	fmt.Println("Initializing gomobile...")
	if err := runCommand("gomobile", "init"); err != nil {
		fmt.Printf("Warning: gomobile init failed: %v\n", err)
		fmt.Println("You may need to run 'gomobile init' manually after installing Android NDK")
	}

	fmt.Println()
	fmt.Println("Tools installed! You may also want to install:")
	fmt.Println("  - Xcode: from App Store (for iOS development)")
	fmt.Println("  - Android Studio: https://developer.android.com/studio (for Android development)")

	return nil
}

// uninstallTools is the exact inverse of `irgo install-tools`: it removes the
// Go tools irgo installed, and nothing else. Every install path in the CLI has
// a matching uninstall — without one you cannot return a machine to a known
// state, and a provisioning bug hides behind whatever was left lying around
// instead of surfacing on the next run.
//
// Marker-guarded: a tool irgo did not install is reported and kept, so a
// developer's own templ/air survives. Pass all to override that.
func uninstallTools(all bool) error {
	fmt.Println("Removing irgo-installed Go tools...")

	// A tool can sit in the resolved GOBIN and/or the default $GOPATH/bin —
	// they differ when GOBIN was empty at install time. Check both.
	binDirs := []string{gobinDir()}
	if out, err := exec.Command(goBin(), "env", "GOPATH").Output(); err == nil {
		if gp := strings.TrimSpace(string(out)); gp != "" {
			if p := filepath.Join(gp, "bin"); p != binDirs[0] {
				binDirs = append(binDirs, p)
			}
		}
	}

	removed, kept, missing := 0, 0, 0
	for _, tool := range []string{"templ", "air", "gomobile", "gobind"} {
		if !all && !toolInstalledByIrgo(tool) {
			if _, err := exec.LookPath(tool); err == nil {
				fmt.Printf("  %s: kept (not installed by irgo — use --all to remove anyway)\n", tool)
				kept++
			} else {
				missing++
			}
			continue
		}
		found := false
		for _, dir := range binDirs {
			name := tool
			if runtime.GOOS == "windows" {
				name += ".exe"
			}
			p := filepath.Join(dir, name)
			if _, err := os.Stat(p); err == nil {
				if err := os.Remove(p); err != nil {
					return fmt.Errorf("removing %s: %w", p, err)
				}
				fmt.Printf("  %s: removed (%s)\n", tool, p)
				found = true
				removed++
			}
		}
		if !found {
			missing++
		}
		_ = os.Remove(filepath.Join(irgoToolsDir(), tool))
	}

	// Build residue from mobile builds: the temp x/mobile clone and the local
	// go.work that ensureMobileBuildSetup generates.
	_ = os.RemoveAll(filepath.Join(os.TempDir(), "golang-mobile"))
	_ = os.Remove("go.work")
	_ = os.Remove("go.work.sum")

	if err := uninstallMinGW(all); err != nil {
		return err
	}

	fmt.Printf("\n%d removed, %d kept, %d not present.\n", removed, kept, missing)
	fmt.Println("Android SDK/NDK/JDK are separate: irgo uninstall-tools android --remove-jdk")
	return nil
}

// uninstallMinGW removes the mingw-w64 cross-compiler, but only when irgo
// installed it — ensureMinGW brew-installs it on macOS for Windows builds.
func uninstallMinGW(all bool) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	if _, err := exec.LookPath("x86_64-w64-mingw32-gcc"); err != nil {
		return nil
	}
	if !all && !toolInstalledByIrgo("mingw-w64") {
		fmt.Println("  mingw-w64: kept (not installed by irgo — use --all to remove anyway)")
		return nil
	}
	if _, err := exec.LookPath("brew"); err != nil {
		fmt.Println("  mingw-w64: present but Homebrew is unavailable — remove it manually")
		return nil
	}
	fmt.Println("  mingw-w64: removing via brew...")
	if err := runCommand("brew", "uninstall", "--formula", "mingw-w64"); err != nil {
		fmt.Printf("  Warning: brew uninstall mingw-w64 failed: %v\n", err)
	}
	_ = os.Remove(filepath.Join(irgoToolsDir(), "mingw-w64"))
	return nil
}

func checkTool(name, installCmd string) error {
	_, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("%s not found. Install with: %s", name, installCmd)
	}
	return nil
}
