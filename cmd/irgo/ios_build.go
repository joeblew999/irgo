// iOS: framework bind, the runnable Simulator app, and launching it.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func buildIOS(modulePath string) error {
	fmt.Println("Building iOS framework...")

	outPath := "build/ios/Irgo.xcframework"
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}

	// Remove existing framework
	os.RemoveAll(outPath)

	// Ensure go.work and gomobile setup
	if err := ensureMobileBuildSetup(); err != nil {
		return fmt.Errorf("mobile build setup failed: %w", err)
	}

	mobilePackage := modulePath + "/mobile"
	if err := runGomobileCommand("bind", "-target", "ios", "-o", outPath, mobilePackage); err != nil {
		return fmt.Errorf("gomobile bind failed: %w", err)
	}
	writeArtifactStamp("build/ios")

	fmt.Printf("iOS framework built: %s\n", outPath)
	return nil
}

// buildIOSApp builds the runnable app for the simulator or a device. It
// scaffolds ios/Example when missing, builds the framework, then drives
// xcodebuild — so callers never shell out to xcodebuild or pre-scaffold.
func buildIOSApp(modulePath string, device bool, team string) error {
	if err := checkTool("xcodebuild", "Install Xcode from the App Store"); err != nil {
		return err
	}
	if err := scaffoldExamples(); err != nil {
		return fmt.Errorf("iOS example scaffold failed: %w", err)
	}
	if err := buildIOS(modulePath); err != nil {
		return err
	}

	// A previous `irgo run ios --dev` leaves a dev-server URL in Info.plist;
	// a plain build must produce a self-contained app.
	clearDevServerInPlist(filepath.Join("ios/Example", "Example/Info.plist"))

	// Single-source app icon, same as macOS/Android/Windows.
	if ic := findAppIcon(""); ic != "" {
		if err := generateIOSIcons(ic, "ios/Example"); err != nil {
			fmt.Printf("Warning: could not write iOS app icon: %v\n", err)
		} else {
			fmt.Println("  wrote iOS app icon from " + ic)
		}
	}

	appPath, err := buildXcodeApp("ios/Example", device, team)
	if err != nil {
		return err
	}
	kind := "simulator"
	if device {
		kind = "device (Release)"
	}
	fmt.Printf("iOS %s app built: %s\n", kind, appPath)
	return nil
}

// buildXcodeApp drives xcodebuild for the example project and returns the path
// to the built .app.
//
// device=false is the Debug simulator build used by `irgo build ios --sim` and
// by `irgo run ios` before it boots the simulator. device=true is the Release
// build for physical devices and the App Store, which needs a signing team —
// supplied by --team or DEVELOPMENT_TEAM/IOS_TEAM_ID in the environment.
func buildXcodeApp(iosProjectPath string, device bool, team string) (string, error) {
	config, destination, derived, productDir := "Debug", "generic/platform=iOS Simulator",
		"build/ios/DerivedData", "Debug-iphonesimulator"
	if device {
		config, destination, derived, productDir = "Release", "generic/platform=iOS",
			"build/ios/DerivedData-Release", "Release-iphoneos"
	}

	var args []string
	if _, err := os.Stat(filepath.Join(iosProjectPath, "Example.xcworkspace")); err == nil {
		args = []string{"-workspace", filepath.Join(iosProjectPath, "Example.xcworkspace")}
	} else if _, err := os.Stat(filepath.Join(iosProjectPath, "Example.xcodeproj")); err == nil {
		args = []string{"-project", filepath.Join(iosProjectPath, "Example.xcodeproj")}
	} else {
		return "", fmt.Errorf("no Xcode project found in %s", iosProjectPath)
	}
	args = append(args, "-scheme", "Example", "-configuration", config,
		"-destination", destination, "-derivedDataPath", derived)

	if device {
		if team == "" {
			team = firstNonEmpty(os.Getenv("DEVELOPMENT_TEAM"), os.Getenv("IOS_TEAM_ID"))
		}
		if team == "" {
			return "", fmt.Errorf("a device build must be signed: pass --team <TEAM_ID>, " +
				"or set DEVELOPMENT_TEAM / IOS_TEAM_ID. For an unsigned build use --sim")
		}
		args = append(args, "-DEVELOPMENT_TEAM="+team)
	}
	args = append(args, "build")

	fmt.Printf("Building iOS app (%s)...\n", config)
	if err := runCommand("xcodebuild", args...); err != nil {
		return "", fmt.Errorf("xcodebuild failed: %w", err)
	}

	appPath := filepath.Join(derived, "Build", "Products", productDir, "Example.app")
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		return "", fmt.Errorf("built app not found at %s", appPath)
	}
	return appPath, nil
}

// Artifact version stamps: dev mode reuses previously built frameworks/AARs,
// but a framework built by an older irgo exposes an older bridge API and the
// native shells fail to compile against it. The stamp forces a rebuild after
// an irgo upgrade.

func runIOS(devMode bool) error {
	// Check for Xcode
	if err := checkTool("xcodebuild", "Install Xcode from the App Store"); err != nil {
		return err
	}
	if err := checkTool("xcrun", "Install Xcode Command Line Tools: xcode-select --install"); err != nil {
		return err
	}

	// Ensure the canonical ios/Example app exists (scaffolded from the
	// embedded templates when missing — devs and CI never hand-copy it).
	if err := scaffoldExamples(); err != nil {
		return fmt.Errorf("iOS example scaffold failed: %w", err)
	}

	// Check if ios/Example project exists
	iosProjectPath := "ios/Example"
	if _, err := os.Stat(iosProjectPath); os.IsNotExist(err) {
		return fmt.Errorf("iOS project not found at %s", iosProjectPath)
	}

	// Dev server URL for simulator to connect to
	devServerURL := "http://localhost:8080"
	var devServerCmd *exec.Cmd

	if devMode {
		fmt.Println("Running in DEV MODE with hot reload...")
		fmt.Println()

		// Check for required dev tools
		if err := ensureGoTool("air"); err != nil {
			return err
		}

		// Dev mode serves the app from localhost:8080, so a fresh gomobile
		// build isn't needed. The Xcode project still links against
		// build/ios/Irgo.xcframework, so build it only when missing or built
		// by a different irgo version (stale bridge API).
		if !artifactUpToDate("build/ios/Irgo.xcframework", "build/ios") {
			modulePath, err := getModulePath()
			if err != nil {
				return fmt.Errorf("could not determine module path: %w", err)
			}

			fmt.Println("Building iOS framework (missing or built by another irgo version)...")
			if err := buildIOS(modulePath); err != nil {
				return err
			}
		} else {
			fmt.Println("Using existing build/ios/Irgo.xcframework (delete it to force a rebuild)")
		}

		// Update Info.plist to enable dev mode
		infoPlistPath := filepath.Join(iosProjectPath, "Example/Info.plist")
		if err := setDevServerInPlist(infoPlistPath, devServerURL); err != nil {
			fmt.Printf("Warning: could not set dev server in Info.plist: %v\n", err)
		}

		// Start dev server in background
		fmt.Printf("Starting dev server at %s...\n", devServerURL)
		devServerCmd = exec.Command("air")
		devServerCmd.Stdout = os.Stdout
		devServerCmd.Stderr = os.Stderr
		if err := devServerCmd.Start(); err != nil {
			return fmt.Errorf("failed to start dev server: %w", err)
		}

		// Give server time to start
		fmt.Println("Waiting for dev server to start...")
		exec.Command("sleep", "3").Run()

	} else {
		// Production mode: build the framework
		modulePath, err := getModulePath()
		if err != nil {
			return fmt.Errorf("could not determine module path: %w", err)
		}

		fmt.Println("Building iOS framework...")
		if err := buildIOS(modulePath); err != nil {
			return err
		}

		// Clear dev server from Info.plist for production builds
		infoPlistPath := filepath.Join(iosProjectPath, "Example/Info.plist")
		clearDevServerInPlist(infoPlistPath)
	}

	appPath, err := buildXcodeApp(iosProjectPath, false, "")
	if err != nil {
		if devServerCmd != nil {
			devServerCmd.Process.Kill()
		}
		return err
	}

	// Find an available iPhone simulator
	simulatorName := findAvailableIPhoneSimulator()
	if simulatorName == "" {
		simulatorName = "iPhone 15" // Fallback
	}

	// Boot simulator if needed
	fmt.Printf("Launching iOS Simulator (%s)...\n", simulatorName)
	runCommand("xcrun", "simctl", "boot", simulatorName) // Ignore error if already booted

	// Open Simulator app
	runCommand("open", "-a", "Simulator")

	// Install app
	fmt.Println("Installing app...")
	if err := runCommand("xcrun", "simctl", "install", "booted", appPath); err != nil {
		if devServerCmd != nil {
			devServerCmd.Process.Kill()
		}
		return fmt.Errorf("failed to install app: %w", err)
	}

	// Launch app
	fmt.Println("Launching app...")
	bundleID := "com.irgo.Example" // Default bundle ID
	if err := runCommand("xcrun", "simctl", "launch", "booted", bundleID); err != nil {
		if devServerCmd != nil {
			devServerCmd.Process.Kill()
		}
		return fmt.Errorf("failed to launch app: %w", err)
	}

	if devMode {
		fmt.Println()
		fmt.Println("===========================================")
		fmt.Println("iOS app running in DEV MODE with hot reload!")
		fmt.Printf("Dev server: %s\n", devServerURL)
		fmt.Println("Edit your Go code and see changes instantly.")
		fmt.Println("Press Ctrl+C to stop.")
		fmt.Println("===========================================")
		fmt.Println()

		// Wait for dev server to exit (user presses Ctrl+C)
		devServerCmd.Wait()
	} else {
		fmt.Println("\nApp running on iOS Simulator!")
	}

	return nil
}

// findAvailableIPhoneSimulator finds an available iPhone simulator
func findAvailableIPhoneSimulator() string {
	// Get list of available simulators
	out, err := exec.Command("xcrun", "simctl", "list", "devices", "available", "-j").Output()
	if err != nil {
		return ""
	}

	// Parse JSON to find an iPhone
	// Look for common iPhone names in priority order
	preferences := []string{"iPhone 15 Pro", "iPhone 15", "iPhone 17 Pro", "iPhone 17", "iPhone SE"}
	outStr := string(out)
	for _, name := range preferences {
		if strings.Contains(outStr, name) {
			return name
		}
	}

	return ""
}

// setDevServerInPlist adds IRGO_DEV_SERVER to Info.plist
func setDevServerInPlist(plistPath, devServerURL string) error {
	data, err := os.ReadFile(plistPath)
	if err != nil {
		return err
	}

	content := string(data)

	// Check if IRGO_DEV_SERVER already exists
	if strings.Contains(content, "IRGO_DEV_SERVER") {
		// Update existing value
		// Find and replace the string value after IRGO_DEV_SERVER key
		start := strings.Index(content, "<key>IRGO_DEV_SERVER</key>")
		if start != -1 {
			stringStart := strings.Index(content[start:], "<string>")
			stringEnd := strings.Index(content[start:], "</string>")
			if stringStart != -1 && stringEnd != -1 {
				newContent := content[:start+stringStart+8] + devServerURL + content[start+stringEnd:]
				return os.WriteFile(plistPath, []byte(newContent), 0644)
			}
		}
	}

	// Add new key-value pair before </dict>
	insertPoint := strings.LastIndex(content, "</dict>")
	if insertPoint == -1 {
		return fmt.Errorf("could not find </dict> in Info.plist")
	}

	newEntry := fmt.Sprintf("\t<key>IRGO_DEV_SERVER</key>\n\t<string>%s</string>\n", devServerURL)
	newContent := content[:insertPoint] + newEntry + content[insertPoint:]

	return os.WriteFile(plistPath, []byte(newContent), 0644)
}

// clearDevServerInPlist removes IRGO_DEV_SERVER from Info.plist
func clearDevServerInPlist(plistPath string) {
	data, err := os.ReadFile(plistPath)
	if err != nil {
		return
	}

	content := string(data)

	// Find and remove IRGO_DEV_SERVER entry
	keyStart := strings.Index(content, "<key>IRGO_DEV_SERVER</key>")
	if keyStart == -1 {
		return
	}

	// Find the end of the string value
	valueEnd := strings.Index(content[keyStart:], "</string>")
	if valueEnd == -1 {
		return
	}

	// Remove the entire entry including newlines
	entryEnd := keyStart + valueEnd + len("</string>")

	// Look for trailing newline
	if entryEnd < len(content) && content[entryEnd] == '\n' {
		entryEnd++
	}

	// Look for leading whitespace/newline
	entryStart := keyStart
	for entryStart > 0 && (content[entryStart-1] == '\t' || content[entryStart-1] == ' ') {
		entryStart--
	}

	newContent := content[:entryStart] + content[entryEnd:]
	os.WriteFile(plistPath, []byte(newContent), 0644)
}
