// Shared mobile build plumbing: target dispatch, host gating, the gomobile
// workspace, and the artifact stamps that force a rebuild after an upgrade.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// runBuild builds for mobile platforms. sim additionally builds the runnable
// iOS Simulator app (ios target only).
func runBuild(target string, sim, device bool, team string) error {
	// No gomobile check here: buildIOS/buildAndroid both run
	// ensureMobileBuildSetup, which installs gomobile + gobind when missing.
	// Gating up front would fail the build before that could happen.

	// Determine module path
	modulePath, err := getModulePath()
	if err != nil {
		return fmt.Errorf("could not determine module path: %w", err)
	}

	// _templ.go and output.css are generated and gitignored, yet the mobile
	// package imports the former and every build embeds the latter. Regenerate
	// both here rather than making callers remember the ordering.
	if err := ensureAssets(); err != nil {
		return err
	}

	// Create build directory
	if err := os.MkdirAll("build", 0755); err != nil {
		return fmt.Errorf("creating build directory: %w", err)
	}

	if sim && device {
		return fmt.Errorf("--sim and --device build different things: " +
			"--sim is an unsigned Simulator app, --device is a signed build for real hardware. Pick one")
	}

	switch target {
	case "ios":
		if err := requireMacOS("iOS"); err != nil {
			return err
		}
		if sim || device {
			return buildIOSApp(modulePath, device, team)
		}
		return buildIOS(modulePath)
	case "android":
		return buildAndroid(modulePath)
	case "all":
		// Android builds anywhere; iOS cannot leave macOS. Skip rather than
		// fail so `irgo build all` stays usable on Linux/Windows CI.
		if runtime.GOOS == "darwin" {
			if err := buildIOS(modulePath); err != nil {
				return err
			}
		} else {
			fmt.Printf("Skipping iOS framework: requires macOS (host is %s)\n", runtime.GOOS)
		}
		return buildAndroid(modulePath)
	default:
		return fmt.Errorf("unknown build target: %s (use ios, android, or all)", target)
	}
}

// requireMacOS gates Apple-only targets with an actionable error, instead of
// letting the build fail later on a missing xcodebuild that the host can never
// have in the first place.
func requireMacOS(what string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("%s builds require macOS (Xcode) — cannot cross-compile from %s.\n  Run `irgo doctor` to see everything this host can and cannot build", what, runtime.GOOS)
	}
	return nil
}

// runMobile builds and runs on mobile simulator
func runMobile(platform string, devMode bool, avdName string, headless bool) error {
	// Same as the build paths: _templ.go and output.css are generated and
	// gitignored but compiled into the app, so running without regenerating
	// them ships whatever happened to be on disk — or, after irgo clean,
	// nothing at all, which looks like a broken app rather than a missing step.
	if err := ensureAssets(); err != nil {
		return err
	}
	switch platform {
	case "ios":
		return runIOS(devMode)
	case "android":
		return runAndroid(devMode, avdName, headless)
	default:
		return fmt.Errorf("unknown platform: %s (use ios or android)", platform)
	}
}

// ensureMobileBuildSetup ensures the go.work file and x/mobile are set up correctly
func ensureMobileBuildSetup() error {
	goVersion := getGoVersion()

	// Check if go.work exists with x/mobile
	if _, err := os.Stat("go.work"); os.IsNotExist(err) {
		// Get irgo path for replacement
		irgoPath := getIrgoPath()

		// Clone x/mobile if not already present
		mobileDir := filepath.Join(os.TempDir(), "golang-mobile")
		if _, err := os.Stat(mobileDir); os.IsNotExist(err) {
			fmt.Println("Cloning golang.org/x/mobile...")
			cmd := exec.Command("git", "clone", "--depth", "1", "https://github.com/golang/mobile", mobileDir)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to clone x/mobile: %w", err)
			}
		}

		// Update go.mod in cloned repo to use current Go version
		mobileModPath := filepath.Join(mobileDir, "go.mod")
		if data, err := os.ReadFile(mobileModPath); err == nil {
			content := string(data)
			// Replace any go 1.x.x version with current version
			lines := splitLines(content)
			for i, line := range lines {
				if len(line) > 3 && line[:3] == "go " {
					lines[i] = "go " + goVersion
					break
				}
			}
			os.WriteFile(mobileModPath, []byte(strings.Join(lines, "\n")), 0644)
		}

		// Create go.work file
		workContent := fmt.Sprintf("go %s\n\nuse (\n\t.\n", goVersion)
		if irgoPath != "" {
			workContent += fmt.Sprintf("\t%s\n", irgoPath)
		}
		workContent += fmt.Sprintf("\t%s\n)\n", mobileDir)

		if err := os.WriteFile("go.work", []byte(workContent), 0644); err != nil {
			return fmt.Errorf("failed to create go.work: %w", err)
		}
		fmt.Println("Created go.work for mobile build")

		// Install gomobile and gobind from local source
		fmt.Println("Installing gomobile from source...")
		cmd := exec.Command(goBin(), "install", "./cmd/gomobile")
		cmd.Dir = mobileDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install gomobile: %w", err)
		}

		cmd = exec.Command(goBin(), "install", "./cmd/gobind")
		cmd.Dir = mobileDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install gobind: %w", err)
		}

		// `go install` lands in GOBIN (default $GOPATH/bin), which may not be on
		// the caller's PATH — prepend it so runGomobileCommand resolves gomobile.
		_ = os.Setenv("PATH", gobinDir()+string(os.PathListSeparator)+os.Getenv("PATH"))
	}

	return nil
}

// runGomobileCommand runs a gomobile command with the correct GOTOOLCHAIN
func runGomobileCommand(args ...string) error {
	goVersion := getGoVersion()

	// gomobile bind for Android shells out to javac, which needs a JDK on
	// PATH — resolve the managed ~/.irgo/jdks JDK (or an existing one) so the
	// toolchain is self-contained after `irgo install-tools android`.
	applyBestJDKToEnv()

	cmd := exec.Command("gomobile", args...)
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=go"+goVersion)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func writeArtifactStamp(dir string) {
	_ = os.WriteFile(filepath.Join(dir, ".irgo-version"), []byte(version), 0644)
}

func artifactUpToDate(artifact, stampDir string) bool {
	fi, err := os.Stat(artifact)
	if err != nil {
		return false
	}
	data, rerr := os.ReadFile(filepath.Join(stampDir, ".irgo-version"))
	if rerr != nil || strings.TrimSpace(string(data)) != version {
		return false
	}
	// The framework embeds the app's Go code and static assets, so a source
	// edit makes it stale even when the CLI version is unchanged. Without this
	// dev mode reuses a framework built before the change and the app runs
	// old code with no indication why.
	return !anySourceNewerThan(fi.ModTime())
}

// anySourceNewerThan reports whether any embedded source is newer than t.
func anySourceNewerThan(t time.Time) bool {
	for _, dir := range []string{"static", "templates", "handlers", "app", "mobile"} {
		newer := false
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if info, ierr := d.Info(); ierr == nil && info.ModTime().After(t) {
				newer = true
				return filepath.SkipAll
			}
			return nil
		})
		if newer {
			return true
		}
	}
	// main.go and go.mod sit at the root rather than in a package directory.
	for _, f := range []string{"main.go", "go.mod"} {
		if info, err := os.Stat(f); err == nil && info.ModTime().After(t) {
			return true
		}
	}
	return false
}
