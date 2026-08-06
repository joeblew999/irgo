// Signing teams — `irgo ios team`.
//
// Lived in doctor_fix.go, which is about repairing an Xcode installation.
// Choosing which team signs a build is a different question, and the one a
// developer actually types a command for.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

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
			"  Choose per run:      irgo app run ios --device --team <TEAM_ID>\n"+
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
