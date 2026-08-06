// Routing: one grammar, <noun> <verb> [target].
//
// The CLI used to mix shapes — `build ios` but `app install ios`, `doctor` but
// `tools remove`. Two grammars means two things to learn and no way to guess a
// command you have not seen. Every command is now a noun and a verb.
//
// The nouns are the things irgo acts on:
//
//	project   the repository you are in
//	app       what gets built, run, shipped and installed
//	tools     the toolchains on this machine
//	server    the development server
//	ios       Apple-specific settings
//
// Every previous spelling still routes here, so nothing that worked stops
// working; the old form prints where it moved to.
package main

import (
	"fmt"
	"strings"
)

// route maps a noun and its verb to the code that runs it. Returns false when
// the pair is not a command, so the caller can report it against the noun's
// own verb list rather than a generic unknown-command message.
func route(noun, verb string, args []string) (error, bool) {
	switch noun {

	case "project":
		switch verb {
		case "new":
			if len(args) < 1 {
				return fmt.Errorf("usage: irgo project new <name>  (or \".\" for here)"), true
			}
			return newProject(args[0]), true
		case "clean":
			return runClean(hasFlag(args, "--all", "-a")), true
		case "upgrade":
			return runUpgrade(hasFlag(args, "--force"), hasFlag(args, "--diff")), true
		case "pin":
			return runPin(args), true
		case "ci":
			return runCI(hasFlag(args, "--force", "-f")), true
		case "assets":
			return ensureAssets(), true
		case "test":
			return runTest(), true
		}

	case "app":
		target := "all"
		if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
			target = args[0]
		}
		switch verb {
		case "build":
			if target == "all" && len(args) == 0 {
				return fmt.Errorf("usage: irgo app build <ios|android|desktop|all>"), true
			}
			return runAppBuild(target, args), true
		case "run":
			if len(args) == 0 {
				return fmt.Errorf("usage: irgo app run <ios|android|desktop>"), true
			}
			return runAppRun(target, args), true
		case "package":
			if len(args) == 0 {
				return fmt.Errorf("usage: irgo app package <ios|android|macos|windows>"), true
			}
			return runAppPackage(args), true
		case "install":
			return runAppInstall(target), true
		case "remove":
			return runAppUninstall(target), true
		case "reviews":
			return reviewsCommand(args), true
		}

	case "tools":
		rest := args
		android := len(rest) > 0 && rest[0] == "android"
		if android {
			rest = rest[1:]
		}
		switch verb {
		case "install":
			if android {
				avd := "irgo"
				for i := 0; i < len(rest)-1; i++ {
					if rest[i] == "--avd" {
						avd = rest[i+1]
					}
				}
				return installAndroidTools(hasFlag(rest, "--emulator", "-e"), avd), true
			}
			return installTools(), true
		case "remove":
			scope := ""
			if android {
				scope = "android"
			}
			return uninstallTools(scope, hasFlag(rest, "--all"),
				hasFlag(rest, "--yes", "-y"), hasFlag(rest, "--keep-jdk")), true
		case "doctor":
			switch {
			case android:
				return doctorAndroid(), true
			case hasFlag(rest, "--fix"):
				return runDoctorFix(), true
			default:
				return doctorHost(hasFlag(rest, "--strict")), true
			}
		}

	case "server":
		switch verb {
		case "dev":
			return runDev(), true
		case "serve", "start":
			return runServe(), true
		}

	case "ios":
		switch verb {
		case "team":
			return runIOSTeamCmd(args), true
		}
	}
	return nil, false
}

// nounVerbs lists what each noun accepts, so an unknown verb is answered with
// the alternatives rather than the whole CLI.
var nounVerbs = map[string][]string{
	"project": {"new", "clean", "upgrade", "pin", "ci", "assets", "test"},
	"app":     {"build", "run", "package", "install", "remove", "reviews"},
	"tools":   {"install", "remove", "doctor"},
	"server":  {"dev", "serve"},
	"ios":     {"team"},
}

// legacy maps every previous spelling to its noun and verb. Renaming without
// this turns a grammar cleanup into a broken script for everyone who had one.
var legacy = map[string][2]string{
	"new":     {"project", "new"},
	"clean":   {"project", "clean"},
	"upgrade": {"project", "upgrade"},
	"pin":     {"project", "pin"},
	"ci":      {"project", "ci"},
	"assets":  {"project", "assets"},
	"templ":   {"project", "assets"},
	"test":    {"project", "test"},

	"build":   {"app", "build"},
	"run":     {"app", "run"},
	"package": {"app", "package"},
	"reviews": {"app", "reviews"},

	"dev":   {"server", "dev"},
	"serve": {"server", "serve"},

	"install-tools":   {"tools", "install"},
	"uninstall-tools": {"tools", "remove"},
	"doctor":          {"tools", "doctor"},
}
