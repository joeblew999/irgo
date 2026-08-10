// `irgo project config` — read and write irgo.package.toml.
//
// Every setting is reachable the same way, so a new one needs no new command.
// This replaced `irgo ios team`, which was the only command named after a
// platform: platforms are targets everywhere else (app build ios), and "team"
// is a noun rather than a verb, so it broke the grammar twice while doing
// nothing that a generic accessor cannot.
//
// Values that need discovering rather than typing — which signing teams exist —
// are reported by `irgo tools doctor`, which is where the rest of the machine's
// capabilities already live.
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// configKey is one setting a project can carry.
type configKey struct {
	key     string // dotted name: "ios.team"
	section string
	name    string
	desc    string
	secret  bool
}

// configKeys is the full set, derived from configRegistry.
//
// It used to be a second list written out by hand, and the two disagreed the
// moment anything was added to one of them: android.compile_sdk appeared in
// `irgo project config` and resolved to nothing, because display and
// resolution were reading different tables. A setting is one fact.
func configKeys() []configKey {
	out := make([]configKey, 0, len(configRegistry))
	for _, cv := range configRegistry {
		out = append(out, configKey{
			key:     cv.tomlSection + "." + cv.tomlKey,
			section: cv.tomlSection,
			name:    cv.tomlKey,
			desc:    cv.how,
			secret:  cv.secret,
		})
	}
	return out
}

func findConfigKey(name string) (configKey, bool) {
	for _, k := range configKeys() {
		if k.key == name {
			return k, true
		}
	}
	return configKey{}, false
}

// runConfig shows every setting, one setting, or writes one.
func runConfig(args []string) error {
	if _, err := os.Stat("go.mod"); err != nil {
		return fmt.Errorf("no go.mod here — run irgo project config from your project root")
	}
	switch len(args) {
	case 0:
		return showAllConfig()
	case 1:
		return showOneConfig(args[0])
	default:
		return setConfig(args[0], strings.Join(args[1:], " "))
	}
}

func showAllConfig() error {
	fmt.Printf("Settings in %s\n", packageConfigFile)
	fmt.Println()
	section := ""
	for _, k := range configKeys() {
		if k.section != section {
			section = k.section
			fmt.Printf("[%s]\n", section)
		}
		val, src := configValueOf(k)
		fmt.Printf("  %-28s %-24s %s\n", k.name, val, src)
	}
	fmt.Println()
	fmt.Println("  irgo project config <key>          show one, with where it came from")
	fmt.Println("  irgo project config <key> <value>  set it")
	fmt.Println()
	fmt.Println("Secrets belong in " + packageLocalFile + " (gitignored) or an environment")
	fmt.Println("variable — see: irgo app package setup --check")
	return nil
}

func showOneConfig(key string) error {
	k, ok := findConfigKey(key)
	if !ok {
		return unknownConfigKey(key)
	}
	val, src := configValueOf(k)
	fmt.Printf("%s\n", k.key)
	fmt.Printf("  value   %s\n", val)
	fmt.Printf("  source  %s\n", src)
	fmt.Printf("  what    %s\n", k.desc)
	if k.key == "ios.team" {
		// The one setting whose valid values are discoverable rather than
		// looked up in a portal.
		fmt.Println()
		if teams := xcodeTeams(); len(teams) > 0 {
			cur, _, _ := resolveIOSTeam("")
			fmt.Println("  teams Xcode knows:")
			fmt.Print(formatTeams(teams, cur))
		}
	}
	return nil
}

func setConfig(key, value string) error {
	k, ok := findConfigKey(key)
	if !ok {
		return unknownConfigKey(key)
	}
	// A team that Xcode does not have fails much later, inside xcodebuild, with
	// a message that never mentions the team.
	if k.key == "ios.team" {
		if err := checkTeamKnown(value); err != nil {
			return err
		}
	}
	cv := registryFor(k.section, k.name)
	if cv == nil {
		return unknownConfigKey(k.key)
	}
	if err := validateConfigValue(*cv, value); err != nil {
		return fmt.Errorf("%s: %w\n  %s", k.key, err, cv.how)
	}
	if err := writeConfigValueFor(*cv, value); err != nil {
		return err
	}
	// A secret goes to the gitignored overlay and is not echoed back: this
	// used to print the value and say it had gone somewhere it had not.
	if cv.secret {
		fmt.Printf("%s set  (%s, gitignored — not committed)\n", k.key, packageLocalFile)
		fmt.Printf("  for CI, set %s there instead\n", cv.env)
		return nil
	}
	fmt.Printf("%s = %q  (%s)\n", k.key, value, packageConfigFile)
	return nil
}

func unknownConfigKey(key string) error {
	var names []string
	for _, k := range configKeys() {
		names = append(names, k.key)
	}
	sort.Strings(names)
	// Suggest the near-misses rather than printing two dozen keys.
	var near []string
	for _, n := range names {
		if strings.Contains(n, key) || strings.Contains(key, strings.SplitN(n, ".", 2)[0]) {
			near = append(near, n)
		}
	}
	if len(near) > 0 {
		return fmt.Errorf("unknown setting: %s\n  did you mean: %s\n  all of them: irgo project config",
			key, strings.Join(near, ", "))
	}
	return fmt.Errorf("unknown setting: %s\n  list them with: irgo project config", key)
}

// configValueOf resolves a setting through the documented precedence and says
// which layer answered, so a surprising value is traceable.
func configValueOf(k configKey) (val, src string) {
	for _, cv := range storeConfigValues("ios") {
		_ = cv
		break
	}
	if v := os.Getenv(envForConfigKey(k.key)); v != "" && envForConfigKey(k.key) != "" {
		return maskIf(k, v), "env " + envForConfigKey(k.key)
	}
	if fileHasKey(packageLocalFile, k.section, k.name) {
		return maskIf(k, readConfigRaw(packageLocalFile, k.section, k.name)), packageLocalFile
	}
	if fileHasKey(packageConfigFile, k.section, k.name) {
		return maskIf(k, readConfigRaw(packageConfigFile, k.section, k.name)), packageConfigFile
	}
	if k.key == "ios.team" {
		if teams := xcodeTeams(); len(teams) > 0 {
			return teams[0].id, "Xcode (not recorded)"
		}
	}
	return "—", "unset"
}

func maskIf(k configKey, v string) string {
	if k.secret && v != "" {
		return "••••••"
	}
	return v
}

// readConfigRaw returns the literal value of section.key in a file.
func readConfigRaw(path, section, key string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	cur := ""
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			cur = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 || cur != section || strings.TrimSpace(line[:eq]) != key {
			continue
		}
		return strings.Trim(strings.TrimSpace(line[eq+1:]), `"'`)
	}
	return ""
}

// envForConfigKey maps a setting to the environment variable that overrides
// it, which is what a CI secret has to be called.
func envForConfigKey(key string) string {
	for _, store := range configStores {
		for _, cv := range storeConfigValues(store) {
			if cv.tomlSection+"."+cv.tomlKey == key {
				return cv.env
			}
		}
	}
	return ""
}

func init() {
	register(command{
		noun: "project", verb: "config", order: 70,
		summary: "Show or set a setting (signing, stores, version)",
		args:    "[<key>] [<value>]",
		usage: [][2]string{
			{"", "Every setting, its value, and where it came from"},
			{"<key>", "One setting"},
			{"<key> <value>", "Set it"},
		},
		notes: `Settings live in irgo.package.toml. Precedence: environment variable, then
irgo.package.local.toml (gitignored), then irgo.package.toml.

Secrets belong in the local file or the environment — irgo.package.toml is
committed. Values you have to discover rather than type, such as which signing
teams exist, are reported by irgo tools doctor.`,
	})
}
