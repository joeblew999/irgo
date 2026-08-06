// iOS: framework bind, the runnable Simulator app, and launching it.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// iosBundleID is the identifier the Example project ships with; it is set in
// the Xcode project rather than derived from the Go module path.
const iosBundleID = "com.irgo.Example"

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

// iosDevice is a physical iPhone/iPad attached to this Mac.
type iosDevice struct {
	name       string
	identifier string
	model      string
	// devModeOn reports Developer Mode, which a device build requires. It is
	// off by default on every iPhone and is the most common reason a device
	// run fails, so it is read up front rather than inferred from a failure.
	devModeOn bool
}

// listIOSDevices returns the connected devices via devicectl (Xcode 15+).
func listIOSDevices() ([]iosDevice, error) {
	tmp, err := os.CreateTemp("", "irgo-devices-*.json")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	tmp.Close()

	if err := exec.Command("xcrun", "devicectl", "list", "devices",
		"--json-output", tmp.Name()).Run(); err != nil {
		return nil, fmt.Errorf("devicectl failed (needs Xcode 15+): %w", err)
	}
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		return nil, err
	}
	var payload struct {
		Result struct {
			Devices []struct {
				Identifier       string `json:"identifier"`
				DeviceProperties struct {
					Name                string `json:"name"`
					DeveloperModeStatus string `json:"developerModeStatus"`
				} `json:"deviceProperties"`
				HardwareProperties struct {
					ProductType string `json:"productType"`
				} `json:"hardwareProperties"`
				ConnectionProperties struct {
					TunnelState string `json:"tunnelState"`
				} `json:"connectionProperties"`
			} `json:"devices"`
		} `json:"result"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("could not read devicectl output: %w", err)
	}
	var out []iosDevice
	for _, d := range payload.Result.Devices {
		out = append(out, iosDevice{
			name:       d.DeviceProperties.Name,
			identifier: d.Identifier,
			model:      d.HardwareProperties.ProductType,
			devModeOn:  d.DeviceProperties.DeveloperModeStatus == "enabled",
		})
	}
	return out, nil
}

// runIOSDevice builds, installs and launches on a physically attached device.
//
// Kept separate from runIOS (simulator): a device build must be signed, is
// installed through devicectl rather than simctl, and fails in ways a
// simulator never does — so the errors need to name provisioning explicitly.
func runIOSDevice(team string) error {
	if err := requireMacOS("iOS device"); err != nil {
		return err
	}
	if err := checkTool("xcodebuild", "Install Xcode from the App Store"); err != nil {
		return err
	}

	devices, err := listIOSDevices()
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		return fmt.Errorf("no iOS device found.\n" +
			"  Connect it by USB, unlock it, and tap Trust This Computer.\n" +
			"  Check what the Mac sees with: xcrun devicectl list devices")
	}
	dev := devices[0]
	fmt.Printf("Device: %s (%s)\n", dev.name, dev.model)

	// Check before building: the framework build takes minutes and this is
	// knowable in a millisecond.
	if !dev.devModeOn {
		return fmt.Errorf("Developer Mode is off on %s.\n%s", dev.name, developerModeHelp())
	}

	modulePath, err := getModulePath()
	if err != nil {
		return err
	}
	if err := ensureAssets(); err != nil {
		return err
	}
	if err := scaffoldExamples(); err != nil {
		return fmt.Errorf("iOS example scaffold failed: %w", err)
	}
	if err := buildIOS(modulePath); err != nil {
		return err
	}
	clearDevServerInPlist(filepath.Join("ios/Example", "Example/Info.plist"))
	if ic := findAppIcon(""); ic != "" {
		_ = generateIOSIcons(ic, "ios/Example")
	}

	appPath, err := buildIOSDeviceApp(dev, team)
	if err != nil {
		return err
	}

	fmt.Println("Installing on device...")
	if err := runCommand("xcrun", "devicectl", "device", "install", "app",
		"--device", dev.identifier, appPath); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	fmt.Println("Launching...")
	if err := runCommand("xcrun", "devicectl", "device", "process", "launch",
		"--device", dev.identifier, iosBundleID); err != nil {
		return fmt.Errorf("launch failed (the app is installed — you can tap it on the device): %w", err)
	}
	fmt.Printf("\nRunning on %s.\n", dev.name)
	return nil
}

// buildIOSDeviceApp builds a signed Debug build for the attached device.
//
// -allowProvisioningUpdates lets Xcode register the device and create the
// profile, which is what makes a free personal Apple ID work without any
// manual portal steps.
func buildIOSDeviceApp(dev iosDevice, team string) (string, error) {
	if team == "" {
		team = firstNonEmpty(os.Getenv("DEVELOPMENT_TEAM"), os.Getenv("IRGO_IOS_TEAM"), deriveIOSTeamFromXcode())
		// Fall back to the team from the Apple ID signed into Xcode, so a
		// developer who has done the one manual step never has to find and
		// type a Team ID.
		if team == "" {
			if teams := xcodeTeams(); len(teams) > 0 {
				team = teams[0].id
			}
		}
	}

	args := []string{"-project", "ios/Example/Example.xcodeproj",
		"-scheme", "Example", "-configuration", "Debug",
		"-destination", "id=" + dev.identifier,
		"-derivedDataPath", "build/ios/DerivedData-Device",
		"-allowProvisioningUpdates"}
	if team != "" {
		args = append(args, "-DEVELOPMENT_TEAM="+team)
		fmt.Printf("Signing with team %s\n", team)
	}
	args = append(args, "build")

	fmt.Println("Building for device...")
	out, err := runCommandCapture("xcodebuild", args...)
	if err != nil {
		return "", fmt.Errorf("%w\n\n%s", err, iosDeviceBuildHelp(out, team))
	}

	appPath := "build/ios/DerivedData-Device/Build/Products/Debug-iphoneos/Example.app"
	if _, err := os.Stat(appPath); err != nil {
		return "", fmt.Errorf("built app not found at %s", appPath)
	}
	return appPath, nil
}

// iosDeviceBuildHelp turns an xcodebuild failure into the specific thing to go
// and do. Both blockers here are physical actions on the device or in Xcode
// that no CLI can perform, so naming the exact one matters more than usual.
func iosDeviceBuildHelp(out, team string) string {
	if strings.Contains(out, "Developer Mode disabled") {
		return developerModeHelp()
	}
	if strings.Contains(out, "requires a provisioning profile") ||
		strings.Contains(out, "No signing certificate") ||
		strings.Contains(out, "no profiles for") ||
		strings.Contains(out, "Signing for") {
		return iosSigningHelp(team)
	}
	return iosSigningHelp(team)
}

// iosSigningHelp explains the one thing irgo cannot do for you. Running on a
// real device requires an Apple ID; a free one is enough, but it has to be
// added to Xcode by hand.
func iosSigningHelp(team string) string {
	var b strings.Builder
	b.WriteString("Running on a physical device requires code signing.\n")
	if team == "" {
		b.WriteString("  No development team was found.\n")
	} else {
		b.WriteString("  Team " + team + " was used.\n")
	}
	b.WriteString("  A free Apple ID is enough — no paid account needed:\n")
	b.WriteString("    1. Xcode → Settings → Accounts → + → Apple ID → sign in\n")
	b.WriteString("    2. irgo run ios --device --team <TEAM_ID>\n")
	b.WriteString("       (or set IRGO_IOS_TEAM; find the ID under Manage Certificates)\n")
	b.WriteString("  On first launch the device will refuse an untrusted developer:\n")
	b.WriteString("    Settings → General → VPN & Device Management → trust the profile\n")
	b.WriteString("  For the simulator instead, which needs no signing: irgo run ios")
	return b.String()
}

// developerModeHelp is the fix for the most common device-run blocker.
func developerModeHelp() string {
	return "  Enable it on the phone:\n" +
		"    1. Settings → Privacy & Security → Developer Mode → On\n" +
		"    2. The phone restarts; unlock it and confirm Turn On\n" +
		"    3. irgo run ios --device\n" +
		"  If the menu entry is not there, plug the phone in and run\n" +
		"  `irgo run ios --device` once — the entry appears after a Mac has\n" +
		"  tried to use the device for development."
}
