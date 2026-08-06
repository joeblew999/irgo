// irgo — one Go codebase for web, desktop, iOS and Android.
//
// Files are named after what a developer types, so a command in the help and
// the code behind it are findable from each other:
//
// The CLI has two grammars, and the filenames follow whichever applies:
//
//	noun → verb     app install, tools remove, tools doctor, ios team
//	verb → target   build ios, run android, package macos
//
//	main.go                 dispatch only
//	help.go                 the help text for every command
//	cmd_<noun>.go           a noun and its verbs: app, tools, ios
//	cmd_<noun>_<verb>.go    one verb, when it is big enough to stand alone
//	cmd_<verb>.go           a verb with no noun: new, build, dev, clean, ...
//	<platform>_build.go     how a platform is built and run
//	<platform>_package.go   how it is packaged for its store
//	<thing>_toolchain.go    provisioning: android, tailwind
//	host_packages.go        brew/apt/pacman dependencies
//	util.go, icons.go       shared helpers
//
// So cmd_* is what a developer types, and everything else is how it works.
//
// A filename must never end in a GOOS name — build_ios.go and build_android.go
// compile only for GOOS=ios and GOOS=android under Go's implicit build
// constraints, which silently excluded them from every build until it was
// noticed. Hence ios_build.go, not build_ios.go.
package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"
)

// version is what the CLI reports and what artifact stamps are compared
// against. It is resolved from the build rather than hardcoded: `go tool irgo`
// builds whatever go.mod pins, so a constant here would describe a release
// nobody is necessarily running — as it did, claiming 0.4.0 for every fork
// build. See cliVersion.
var version = cliVersion()

// fallbackVersion is used only when a binary carries no build information,
// which happens if it was assembled outside the module system.
const fallbackVersion = "0.4.0"

// cliVersion reports the module version this binary was built from, falling
// back to the VCS revision for a local checkout — which is exactly the case
// `irgo pin --local` creates, and worth naming so a surprising result is
// traceable to a working tree rather than a release.
func cliVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return fallbackVersion
	}
	if v := bi.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	var rev string
	dirty := ""
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-modified"
			}
		}
	}
	if rev != "" {
		if len(rev) > 12 {
			rev = rev[:12]
		}
		return "devel-" + rev + dirty
	}
	return fallbackVersion + "-devel"
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "new":
		if len(os.Args) < 3 {
			fmt.Println("Usage: irgo new <project-name>")
			os.Exit(1)
		}
		err = newProject(os.Args[2])

	case "dev":
		err = runDev()

	case "serve":
		err = runServe()

	case "build":
		if len(os.Args) < 3 {
			fmt.Println("Usage: irgo build <ios|android|desktop|all>")
			os.Exit(1)
		}
		target := os.Args[2]
		if target == "desktop" {
			platform := ""
			if len(os.Args) > 3 {
				platform = os.Args[3]
			}
			err = buildDesktop(platform)
		} else {
			team := ""
			for i := 3; i < len(os.Args)-1; i++ {
				if os.Args[i] == "--team" {
					team = os.Args[i+1]
				}
			}
			err = runBuild(target,
				hasFlag(os.Args[3:], "--sim", "-s"),
				hasFlag(os.Args[3:], "--device", "-D"),
				team)
		}

	case "run":
		if len(os.Args) < 3 {
			fmt.Println("Usage: irgo run <ios|android|desktop> [--dev] [--avd NAME] [--no-window]")
			os.Exit(1)
		}
		platform := os.Args[2]
		devMode := hasFlag(os.Args[3:], "--dev", "-d")
		headless := hasFlag(os.Args[3:], "--no-window", "-nw")
		avd := "irgo"
		for i := 3; i < len(os.Args); i++ {
			if os.Args[i] == "--avd" && i+1 < len(os.Args) {
				avd = os.Args[i+1]
			}
		}

		team := ""
		for i := 3; i < len(os.Args)-1; i++ {
			if os.Args[i] == "--team" {
				team = os.Args[i+1]
			}
		}
		if platform == "desktop" {
			err = runDesktop(devMode, hasFlag(os.Args[3:], "--built", "-b"))
		} else if platform == "ios" && hasFlag(os.Args[3:], "--device", "-D") {
			err = runIOSDevice(team)
		} else {
			err = runMobile(platform, devMode, avd, headless)
		}

	case "package":
		if len(os.Args) < 3 {
			fmt.Println("Usage: irgo package <ios|android|macos|windows> [flags]")
			os.Exit(1)
		}
		target := os.Args[2]
		team, exportMethod, out := "", "app-store", ""
		keystore, keystorePass, keyAlias, keyPass := "", "", "", ""
		version, publisher, icon, cert, certPass := "", "", "", "", ""
		identity, appleID, password := "", "", ""
		notarize, dmg := false, false
		// setup has its own args (store / --check) — don't parse package flags.
		if target != "setup" {
			for i := 3; i < len(os.Args); i++ {
				next := func() string {
					if i+1 < len(os.Args) {
						i++
						return os.Args[i]
					}
					return ""
				}
				switch os.Args[i] {
				case "--team":
					team = next()
				case "--export-method":
					exportMethod = next()
				case "--keystore":
					keystore = next()
				case "--keystore-pass":
					keystorePass = next()
				case "--key-alias":
					keyAlias = next()
				case "--key-pass":
					keyPass = next()
				case "--version":
					version = next()
				case "--publisher":
					publisher = next()
				case "--icon":
					icon = next()
				case "--cert":
					cert = next()
				case "--cert-pass":
					certPass = next()
				case "--identity":
					identity = next()
				case "--apple-id":
					appleID = next()
				case "--password":
					password = next()
				case "--notarize":
					notarize = true
				case "--dmg":
					dmg = true
				case "-o", "--output":
					out = next()
				default:
					fmt.Printf("Unknown flag: %s\n", os.Args[i])
					os.Exit(1)
				}
			}
		}
		switch target {
		case "setup":
			// irgo package setup              → static guide (where to get every value)
			// irgo package setup <store>      → interactive wizard for that store
			// irgo package setup --check      → status report for every store
			// irgo package setup --check <s>  → status report for one store
			check := false
			store := ""
			for _, a := range os.Args[3:] {
				switch a {
				case "--check", "-c":
					check = true
				default:
					store = a
				}
			}
			if check {
				stores := []string{"ios", "android", "windows", "macos", "reviews-apple-ios", "reviews-apple-mac", "reviews-android"}
				if store != "" {
					stores = []string{store}
				}
				for _, s := range stores {
					checkStoreConfig(s)
				}
			} else if store != "" {
				if storeConfigValues(store) == nil {
					fmt.Printf("unknown setup target: %s (use ios, android, windows, macos, or reviews-*)\n", store)
					os.Exit(1)
				}
				missing := missingStoreConfig(store)
				if len(missing) == 0 {
					fmt.Printf("Nothing missing for %s — defaults cover it. Run `irgo package setup` for the full guide.\n", store)
				} else if wErr := runSetupWizard(store, missing); wErr != nil {
					err = wErr
				}
			} else {
				packageSetupGuide()
			}
		case "ios":
			err = packageIOS(team, exportMethod, out)
		case "android":
			err = packageAndroid(keystore, keystorePass, keyAlias, keyPass, version, icon, out)
		case "macos":
			err = packageMacOS(identity, notarize, appleID, team, password, dmg, icon, out)
		case "windows":
			err = packageWindows(publisher, version, icon, cert, certPass, out)
		default:
			err = fmt.Errorf("unknown package target: %s (use ios, android, macos, windows, or setup)", target)
		}

	case "ios":
		if len(os.Args) > 2 && os.Args[2] == "team" {
			err = runIOSTeamCmd(os.Args[3:])
		} else {
			err = fmt.Errorf("usage: irgo ios team [TEAM_ID]")
		}

	case "clean":
		err = runClean(hasFlag(os.Args[2:], "--all", "-a"))

	case "app":
		// Objects group their verbs, so "what does this act on" is answerable
		// from the command itself. `uninstall` and `uninstall-tools` used to
		// sit one hyphen apart while removing entirely different things.
		if len(os.Args) < 3 {
			err = fmt.Errorf("usage: irgo app <install|remove> [ios|android|desktop|all]")
			break
		}
		target := "all"
		if len(os.Args) > 3 {
			target = os.Args[3]
		}
		switch os.Args[2] {
		case "install":
			err = runAppInstall(target)
		case "remove", "uninstall":
			err = runAppUninstall(target)
		default:
			err = fmt.Errorf("unknown app command: %s (use: install, remove)", os.Args[2])
		}

	case "uninstall":
		deprecated("uninstall", "app remove")
		target := "all"
		if len(os.Args) > 2 {
			target = os.Args[2]
		}
		err = runAppUninstall(target)

	case "templ":
		// Named after the tool rather than the job, and a strict subset of
		// assets — which also builds the stylesheet, so using templ alone
		// produced a half-generated project.
		deprecated("templ", "assets")
		err = ensureAssets()

	case "assets":
		err = ensureAssets()

	case "ci":
		err = runCI(hasFlag(os.Args[2:], "--force", "-f"))

	case "pin":
		err = runPin(os.Args[2:])

	case "upgrade":
		err = runUpgrade(hasFlag(os.Args[2:], "--force"), hasFlag(os.Args[2:], "--diff"))

	case "test":
		err = runTest()

	case "tools":
		if len(os.Args) < 3 {
			err = fmt.Errorf("usage: irgo tools <install|remove|doctor> [android] [flags]")
			break
		}
		rest := os.Args[3:]
		android := len(rest) > 0 && rest[0] == "android"
		if android {
			rest = rest[1:]
		}
		switch os.Args[2] {
		case "install":
			if android {
				avd := "irgo"
				for i := 0; i < len(rest)-1; i++ {
					if rest[i] == "--avd" {
						avd = rest[i+1]
					}
				}
				err = installAndroidTools(hasFlag(rest, "--emulator", "-e"), avd)
			} else {
				err = installTools()
			}
		case "remove":
			scope := ""
			if android {
				scope = "android"
			}
			// --remove-jdk used to be how you opted IN to deleting the JDK,
			// back when removal skipped it by default. Removal now means
			// everything irgo installed, so it is accepted and ignored, and
			// --keep-jdk is the opt-out.
			err = uninstallTools(scope,
				hasFlag(rest, "--all"),
				hasFlag(rest, "--yes", "-y"),
				hasFlag(rest, "--keep-jdk"))
		case "doctor":
			switch {
			case android:
				err = doctorAndroid()
			case hasFlag(rest, "--fix"):
				err = runDoctorFix()
			default:
				err = doctorHost(hasFlag(rest, "--strict"))
			}
		default:
			err = fmt.Errorf("unknown tools command: %s (use: install, remove, doctor)", os.Args[2])
		}

	case "install-tools":
		deprecated("install-tools", "tools install")
		if len(os.Args) > 2 && os.Args[2] == "android" {
			avd := "irgo"
			for i := 3; i < len(os.Args); i++ {
				if os.Args[i] == "--avd" && i+1 < len(os.Args) {
					avd = os.Args[i+1]
				}
			}
			err = installAndroidTools(hasFlag(os.Args[3:], "--emulator", "-e"), avd)
		} else {
			err = installTools()
		}

	case "uninstall-tools":
		deprecated("uninstall-tools", "tools remove")
		scope := ""
		if len(os.Args) > 2 && os.Args[2] == "android" {
			scope = "android"
		}
		err = uninstallTools(scope,
			hasFlag(os.Args[2:], "--all"),
			hasFlag(os.Args[2:], "--yes", "-y"),
			hasFlag(os.Args[2:], "--keep-jdk"))

	case "doctor":
		deprecated("doctor", "tools doctor")
		// Bare `irgo doctor` answers the first question a dev has on a new
		// machine: what can I actually build here?
		if len(os.Args) > 2 && os.Args[2] == "android" {
			err = doctorAndroid()
		} else if len(os.Args) > 2 && !strings.HasPrefix(os.Args[2], "-") {
			err = fmt.Errorf("unknown doctor target: %s (use `irgo doctor` or `irgo doctor android`)", os.Args[2])
		} else if hasFlag(os.Args[2:], "--fix") {
			err = runDoctorFix()
		} else {
			err = doctorHost(hasFlag(os.Args[2:], "--strict"))
		}

	case "reviews":
		err = reviewsCommand(os.Args[2:])

	case "version", "-v", "--version":
		fmt.Printf("irgo %s\n", version)
		// Build info reports the *required* module version, so a project
		// tracking a fork or a checkout would be told v0.4.0 while running
		// something else entirely. go.mod has the answer.
		if r := projectReplacement(); r != "" {
			fmt.Printf("  running: %s\n", r)
		}

	case "help", "-h", "--help":
		if len(os.Args) > 2 {
			printCommandHelp(os.Args[2])
		} else {
			printUsage()
		}

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func deprecated(old, replacement string) {
	fmt.Fprintf(os.Stderr, "note: `irgo %s` is now `irgo %s` — the old name still works\n", old, replacement)
}
