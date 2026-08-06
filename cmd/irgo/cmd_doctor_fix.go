// Repairing what doctor reports.
//
// Most Xcode problems are one command away, but the command is obscure and the
// failure that led you there rarely names it. Those are worth doing for the
// developer. Two things are not automatable at all — an Apple ID sign-in needs
// credentials and 2FA, and Developer Mode is a toggle on the phone — so those
// are handed over with the fewest remaining steps rather than pretended away.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// runDoctorFix applies every repair irgo can make, then reports what is left.
func runDoctorFix() error {
	if runtime.GOOS != "darwin" {
		fmt.Println("Nothing to fix: the repairs here are Xcode-specific (macOS only).")
		fmt.Println("Run `irgo doctor` to see what this host can build.")
		return nil
	}

	fmt.Println("Fixing what can be fixed automatically...")
	fmt.Println()

	fixed, manual := 0, []string{}

	if did, err := fixXcodeSelect(); err != nil {
		fmt.Printf("  ! %v\n", err)
	} else if did {
		fixed++
	}
	if did, err := fixXcodeFirstLaunch(); err != nil {
		fmt.Printf("  ! %v\n", err)
	} else if did {
		fixed++
	}
	if did, err := fixSimulatorRuntimes(); err != nil {
		fmt.Printf("  ! %v\n", err)
	} else if did {
		fixed++
	}

	// Everything below needs a human.
	if !canSignForDevice() {
		manual = append(manual, "Apple ID in Xcode")
	}
	if devs, err := listIOSDevices(); err == nil {
		for _, d := range devs {
			if !d.devModeOn {
				manual = append(manual, "Developer Mode on "+d.name)
			}
		}
	}

	fmt.Println()
	if fixed == 0 {
		fmt.Println("Nothing needed fixing.")
	} else {
		fmt.Printf("%d thing(s) fixed.\n", fixed)
	}
	if len(manual) == 0 {
		fmt.Println("Nothing left to do — irgo doctor should be clean.")
		return nil
	}

	fmt.Println()
	fmt.Printf("Needs you (%s):\n", strings.Join(manual, ", "))
	printManualSteps()
	return nil
}

// fixXcodeSelect repoints xcode-select at a full Xcode when it is aimed at the
// Command Line Tools, where xcodebuild exists but cannot build an app.
func fixXcodeSelect() (bool, error) {
	out, err := exec.Command("xcode-select", "-p").Output()
	if err != nil {
		return false, nil
	}
	cur := strings.TrimSpace(string(out))
	if !strings.Contains(cur, "CommandLineTools") {
		fmt.Println("  ✓ xcode-select already points at Xcode")
		return false, nil
	}
	xcode := findXcodeApp()
	if xcode == "" {
		return false, fmt.Errorf("xcode-select points at the Command Line Tools and no Xcode.app was found — install Xcode from the App Store")
	}
	dev := xcode + "/Contents/Developer"
	fmt.Printf("  → repointing xcode-select at %s (needs sudo)\n", xcode)
	if err := runCommand("sudo", "xcode-select", "-s", dev); err != nil {
		return false, fmt.Errorf("xcode-select failed — run it yourself: sudo xcode-select -s %s", dev)
	}
	fmt.Println("  ✓ xcode-select fixed")
	return true, nil
}

// findXcodeApp locates Xcode, preferring the standard path and falling back to
// Spotlight for a non-standard install.
func findXcodeApp() string {
	if _, err := os.Stat("/Applications/Xcode.app"); err == nil {
		return "/Applications/Xcode.app"
	}
	out, err := exec.Command("mdfind",
		"kMDItemCFBundleIdentifier == 'com.apple.dt.Xcode'").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if p := strings.TrimSpace(line); strings.HasSuffix(p, ".app") {
			return p
		}
	}
	return ""
}

// fixXcodeFirstLaunch accepts the licence and installs the bundled components.
// Skipping this fails every build with a message about agreeing to terms.
func fixXcodeFirstLaunch() (bool, error) {
	if err := exec.Command("xcodebuild", "-checkFirstLaunchStatus").Run(); err == nil {
		fmt.Println("  ✓ Xcode first launch already complete")
		return false, nil
	}
	fmt.Println("  → accepting the Xcode licence and installing components (needs sudo)")
	if err := runCommand("sudo", "xcodebuild", "-runFirstLaunch"); err != nil {
		return false, fmt.Errorf("first launch failed — run it yourself: sudo xcodebuild -runFirstLaunch")
	}
	fmt.Println("  ✓ Xcode first launch complete")
	return true, nil
}

// fixSimulatorRuntimes downloads an iOS runtime when none is installed, so
// `irgo run ios` has something to boot.
func fixSimulatorRuntimes() (bool, error) {
	if countIOSRuntimes() > 0 {
		fmt.Println("  ✓ iOS simulator runtime present")
		return false, nil
	}
	fmt.Println("  → downloading the iOS simulator runtime (large, one time)")
	if err := runCommand("xcodebuild", "-downloadPlatform", "iOS"); err != nil {
		return false, fmt.Errorf("runtime download failed — install it via Xcode → Settings → Components")
	}
	if countIOSRuntimes() == 0 {
		return false, fmt.Errorf("no iOS runtime after download — install it via Xcode → Settings → Components")
	}
	fmt.Println("  ✓ iOS simulator runtime installed")
	return true, nil
}

func countIOSRuntimes() int {
	out, err := exec.Command("xcrun", "simctl", "list", "runtimes").Output()
	if err != nil {
		return 0
	}
	n := 0
	for _, l := range strings.Split(string(out), "\n") {
		if strings.Contains(l, "iOS") {
			n++
		}
	}
	return n
}

func hasSigningIdentity() bool {
	out, err := exec.Command("security", "find-identity", "-v", "-p", "codesigning").Output()
	if err != nil {
		return false
	}
	return !strings.Contains(string(out), "0 valid identities")
}

// printManualSteps spells out the two steps no tool can take. Each is written
// as the exact taps, what you should see, and what to do when the thing is not
// where it should be — a summary like "sign in" or "toggle it on" is not
// actionable when the menu entry is missing, which is the usual case.
func printManualSteps() {
	if !canSignForDevice() {
		fmt.Println()
		fmt.Println("  ── Apple ID in Xcode ──────────────────────────────────────")
		fmt.Println("  A free Apple ID works. No paid Developer Program needed.")
		fmt.Println()
		fmt.Println("   1. Xcode is opening now.")
		fmt.Println("   2. Press Cmd+, (comma)          → Settings opens")
		fmt.Println("      (or menu bar: Xcode → Settings…)")
		fmt.Println("   3. Click the 'Accounts' tab      → top of the Settings window")
		fmt.Println("   4. Click '+' at the bottom-left  → a chooser appears")
		fmt.Println("   5. Choose 'Apple ID' → Continue")
		fmt.Println("   6. Sign in (email, password, then the 2FA code on your devices)")
		fmt.Println("   7. You should now see your name on the left, and on the right a")
		fmt.Println("      team ending in '(Personal Team)'.")
		fmt.Println("   8. Close Settings. That is all — irgo reads the Team ID itself,")
		fmt.Println("      so you never have to find or type it.")
		fmt.Println()
		fmt.Println("   No signing certificate is created yet, and that is expected:")
		fmt.Println("   Xcode makes it during the first device build.")
		if _, err := exec.LookPath("open"); err == nil {
			_ = exec.Command("open", "-a", "Xcode").Start()
		}
	}

	for _, d := range devicesNeedingDeveloperMode() {
		fmt.Println()
		fmt.Println("  ── Developer Mode on " + d.name + " ──────────────────────────")
		fmt.Println("  Apple gates this on the device itself, so no tool can set it.")
		fmt.Println()
		fmt.Println("   1. On the iPhone, open the 'Settings' app")
		fmt.Println("   2. Tap 'Privacy & Security'      → about halfway down the list")
		fmt.Println("   3. Scroll to the VERY BOTTOM     → tap 'Developer Mode'")
		fmt.Println("   4. Turn the 'Developer Mode' switch ON")
		fmt.Println("   5. Tap 'Restart' when prompted   → the phone reboots")
		fmt.Println("   6. Unlock the phone after it restarts")
		fmt.Println("   7. A dialog asks 'Turn on Developer Mode?' → tap 'Turn On'")
		fmt.Println("   8. Enter your passcode")
		fmt.Println()
		fmt.Println("   If step 3 has no 'Developer Mode' entry, the phone has not yet")
		fmt.Println("   been asked for development by a Mac. Keep it plugged in and")
		fmt.Println("   unlocked, run `irgo run ios --device` once, then look again —")
		fmt.Println("   the entry appears after that attempt. (Just done for you.)")
		_ = exec.Command("xcrun", "devicectl", "device", "info", "details",
			"--device", d.identifier).Run()
	}

	fmt.Println()
	fmt.Println("  ── Then ───────────────────────────────────────────────────")
	fmt.Println("   irgo doctor --fix     re-check (should report nothing left)")
	fmt.Println("   irgo run ios --device build, install and launch on the phone")
	fmt.Println()
	fmt.Println("   First launch on the phone shows 'Untrusted Developer'. Fix on the")
	fmt.Println("   phone: Settings → General → VPN & Device Management → tap your")
	fmt.Println("   Apple ID → Trust. Then tap the app again.")
}

// devicesNeedingDeveloperMode lists attached devices with Developer Mode off.
func devicesNeedingDeveloperMode() []iosDevice {
	devs, err := listIOSDevices()
	if err != nil {
		return nil
	}
	var out []iosDevice
	for _, d := range devs {
		// A disconnected device cannot be assessed — its reported Developer
		// Mode is whatever it was when last seen.
		if d.connected && !d.devModeOn {
			out = append(out, d)
		}
	}
	return out
}
