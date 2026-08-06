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

// xcodeTeam is a provisioning team Xcode knows about from a signed-in Apple ID.
type xcodeTeam struct {
	id   string
	name string
	free bool
}

// xcodeTeams reads the teams Xcode has from its preferences.
//
// This is the right signal for "can this Mac sign a device build", not the
// keychain: with automatic signing, the certificate is created during the first
// build with -allowProvisioningUpdates. Checking for an existing certificate
// reports a machine as unable to sign when it is merely one build away, and
// sends the developer off to fix something that is not broken.
func xcodeTeams() []xcodeTeam {
	if runtime.GOOS != "darwin" {
		return nil
	}
	out, err := exec.Command("defaults", "read", "com.apple.dt.Xcode", "IDEProvisioningTeams").Output()
	if err != nil {
		return nil
	}
	var teams []xcodeTeam
	var cur xcodeTeam
	for _, raw := range strings.Split(string(out), "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "teamID = "):
			cur.id = strings.Trim(strings.TrimSuffix(strings.TrimPrefix(line, "teamID = "), ";"), `"`)
		case strings.HasPrefix(line, "teamName = "):
			cur.name = strings.Trim(strings.TrimSuffix(strings.TrimPrefix(line, "teamName = "), ";"), `"`)
		case strings.HasPrefix(line, "isFreeProvisioningTeam = "):
			cur.free = strings.Contains(line, "1")
		case line == "}" || line == "},":
			if cur.id != "" {
				teams = append(teams, cur)
			}
			cur = xcodeTeam{}
		}
	}
	return teams
}

// xcodeAppleID returns the Apple ID signed into Xcode, or "".
func xcodeAppleID() string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	out, err := exec.Command("defaults", "read", "com.apple.dt.Xcode",
		"DVTDeveloperAccountManagerAppleIDLists").Output()
	if err != nil {
		return ""
	}
	for _, raw := range strings.Split(string(out), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "username = ") {
			return strings.Trim(strings.TrimSuffix(strings.TrimPrefix(line, "username = "), ";"), `"`)
		}
	}
	return ""
}

// canSignForDevice reports whether a device build can be signed — an Xcode team
// is enough, since the certificate follows on the first build.
func canSignForDevice() bool {
	return len(xcodeTeams()) > 0 || hasSigningIdentity()
}

// resolveIOSTeam decides which development team signs a device build, and
// reports where the choice came from so a surprising one can be traced.
//
// Order: explicit flag, environment, irgo.package.toml [ios] team, then the
// teams Xcode knows. With several teams and no preference recorded, it refuses
// rather than guessing — picking the wrong one produces an Apple-side error
// (a device limit, an expired membership) that looks nothing like "wrong team".
func resolveIOSTeam(flagTeam string) (team, source string, err error) {
	// Every source is validated the same way. A typo in a flag or an env var
	// fails identically to a stale config entry, and checkTeamKnown is a no-op
	// where Xcode lists no teams at all — CI, which signs from a secret.
	if flagTeam != "" {
		return flagTeam, "--team", checkTeamKnown(flagTeam)
	}
	if v := os.Getenv("IRGO_IOS_TEAM"); v != "" {
		return v, "env IRGO_IOS_TEAM", checkTeamKnown(v)
	}
	if v := os.Getenv("DEVELOPMENT_TEAM"); v != "" {
		return v, "env DEVELOPMENT_TEAM", checkTeamKnown(v)
	}
	if cfg := parsePackageConfig(); cfg.IOSTeam != "" {
		// A recorded team can go stale — the Apple ID it came from may have
		// been removed from Xcode since. Using it anyway fails deep inside
		// xcodebuild with an Apple-side message that never mentions the team,
		// so say it here instead.
		if err := checkTeamKnown(cfg.IOSTeam); err != nil {
			return "", "", err
		}
		return cfg.IOSTeam, packageConfigFile + " [ios] team", nil
	}
	if v := deriveIOSTeamFromXcode(); v != "" {
		return v, "the Xcode project", nil
	}

	teams := xcodeTeams()
	switch len(teams) {
	case 0:
		return "", "", fmt.Errorf("no development team found.\n" +
			"  Sign an Apple ID into Xcode (Settings → Apple Accounts); a free one works.\n" +
			"  Then irgo picks up the team automatically.")
	case 1:
		return teams[0].id, "the Apple ID in Xcode", nil
	default:
		return "", "", fmt.Errorf("several development teams are available — pick one:\n\n%s\n"+
			"  Choose per run:      irgo run ios --device --team <TEAM_ID>\n"+
			"  Or record it once:   irgo ios team <TEAM_ID>\n"+
			"                       (writes [ios] team to %s)",
			formatTeams(teams, ""), packageConfigFile)
	}
}

// checkTeamKnown reports when a configured team is not one Xcode has. It is a
// no-op when Xcode lists none at all: CI signs with a team from a secret and
// has no local account, which is legitimate.
func checkTeamKnown(id string) error {
	teams := xcodeTeams()
	if len(teams) == 0 {
		return nil
	}
	for _, t := range teams {
		if t.id == id {
			return nil
		}
	}
	return fmt.Errorf("team %s is not one Xcode has.\n"+
		"  Either it is a typo, or the Apple ID it came from was removed.\n\n"+
		"  Available:\n%s\n"+
		"  Pick one:  irgo ios team <TEAM_ID>   (records it in %s)",
		id, formatTeams(teams, ""), packageConfigFile)
}

// formatTeams renders the team list, marking the one in use.
func formatTeams(teams []xcodeTeam, selected string) string {
	var b strings.Builder
	for _, t := range teams {
		kind := "paid"
		if t.free {
			kind = "free"
		}
		mark := "   "
		if t.id == selected {
			mark = " → "
		}
		fmt.Fprintf(&b, "  %s%-12s %-6s %s\n", mark, t.id, kind, t.name)
	}
	return b.String()
}

// runIOSTeamCmd records the team to use, so it does not have to be passed on
// every run. Called with no argument it lists what is available.
func runIOSTeamCmd(args []string) error {
	teams := xcodeTeams()
	cur, src, _ := resolveIOSTeam("")

	if len(args) == 0 {
		if len(teams) == 0 {
			fmt.Println("No development teams. Sign an Apple ID into Xcode (Settings → Apple Accounts).")
			return nil
		}
		fmt.Println("Development teams available to Xcode:")
		fmt.Println()
		fmt.Print(formatTeams(teams, cur))
		fmt.Println()
		if cur != "" {
			fmt.Printf("In use: %s (from %s)\n", cur, src)
		} else {
			fmt.Println("None selected.")
		}
		fmt.Printf("\nSelect one:  irgo ios team <TEAM_ID>\n")
		return nil
	}

	id := args[0]
	known := len(teams) == 0 // cannot validate when Xcode reports none
	for _, t := range teams {
		if t.id == id {
			known = true
		}
	}
	if !known {
		return fmt.Errorf("team %s is not one Xcode knows about:\n\n%s\n"+
			"  Add its Apple ID first: Xcode → Settings → Apple Accounts", id, formatTeams(teams, ""))
	}
	if err := writeConfigValue("ios", "team", id); err != nil {
		return err
	}
	fmt.Printf("Team %s recorded in %s — device builds will use it.\n", id, packageConfigFile)
	return nil
}
