package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// irgo package setup — walk a dev through filling in the store config.
//
// Three modes, so it works identically on a laptop and in CI:
//
//  1. Interactive (TTY, not CI): for each missing required value, print where
//     to get it, OPEN the exact console page, prompt for the value, and write
//     it into irgo.package.toml. Auto-derives what it can (iOS team from
//     Xcode, macOS identity from Keychain, package name from the module path).
//  2. CI (no TTY or CI=1): never prompts, never hangs. Fails with a precise
//     list of every missing value, the env var (GitHub secret) to set, and
//     the URL to get it from.
//  3. Env vars: every value has an IRGO_* env var, so CI supplies secrets as
//     env vars (or --flags) and never needs the gitignored irgo.package.toml.
//
// Resolution order everywhere: flag > IRGO_* env > irgo.package.toml > derived
// > built-in default.
// ---------------------------------------------------------------------------

type configValue struct {
	tomlSection string // toml section, e.g. "ios"
	tomlKey     string // toml key, e.g. "team"
	env         string // env var, e.g. "IRGO_IOS_TEAM"
	flag        string // CLI flag, e.g. "--team"
	display     string // human label, e.g. "Apple Team ID"
	url         string // console page to open
	how         string // click path / what it looks like
	required    bool   // blocks packaging when unset
	secret      bool   // never echo back (passwords/keys)
}

func appleDevURL(path string) string { return "https://developer.apple.com" + path }

var (
	urlASCMembership = appleDevURL("/account")
	urlASCCerts      = appleDevURL("/account/resources/certificates/list")
	urlASCAPIKeys    = "https://appstoreconnect.apple.com/access/api"
	urlAppleIDPass   = "https://appleid.apple.com/account/manage"
	urlPlayConsole   = "https://play.google.com/console/"
	urlPartnerCenter = "https://partner.microsoft.com/dashboard"
	urlWinSDK        = "https://developer.microsoft.com/en-us/windows/downloads/windows-sdk/"
)

// storeConfigValues returns every config value a store can consume, with the
// metadata the wizard/checker needs. Optional values (defaults exist) are
// marked required=false so packaging never blocks on them.
func storeConfigValues(store string) []configValue {
	switch store {
	case "ios":
		return []configValue{
			{tomlSection: "ios", tomlKey: "team", env: "IRGO_IOS_TEAM", flag: "--team",
				display: "Apple Team ID", url: urlASCMembership,
				how: "Membership → Team ID (10-character ID)", required: true},
		}
	case "android":
		return []configValue{
			{tomlSection: "android", tomlKey: "keystore", env: "IRGO_ANDROID_KEYSTORE", flag: "--keystore",
				display: "Signing keystore", url: urlPlayConsole,
				how: "Debug keystore works by default (validation only). For release: keytool -genkey (see `irgo package setup`); keep it safe — it can't change after first upload.", required: false},
		}
	case "windows":
		return []configValue{
			{tomlSection: "windows", tomlKey: "publisher", env: "IRGO_WINDOWS_PUBLISHER", flag: "--publisher",
				display: "Publisher (CN=...)", url: urlPartnerCenter,
				how: "Partner Center → your app → Product management → Product identity → Publisher display name. Empty = test-signed.", required: false},
			{tomlSection: "windows", tomlKey: "cert", env: "IRGO_WINDOWS_CERT", flag: "--cert",
				display: "Code-signing cert (PFX)", url: urlWinSDK,
				how: "Optional real cert; empty = self-signed test cert (Store re-signs on upload).", required: false, secret: true},
		}
	case "macos":
		return []configValue{
			{tomlSection: "macos", tomlKey: "identity", env: "IRGO_MACOS_IDENTITY", flag: "--identity",
				display: "Developer ID Application identity", url: urlASCCerts,
				how: "Certificates → Developer ID Application → download + import to Keychain. Auto-picked from your Keychain if empty.", required: false},
			{tomlSection: "macos", tomlKey: "notarize", env: "IRGO_MACOS_NOTARIZE", flag: "--notarize",
				display: "Notarize?", url: "",
				how: "Requires Apple ID + Team ID + an app-specific password (below).", required: false},
			{tomlSection: "macos", tomlKey: "apple_id", env: "IRGO_APPLE_ID", flag: "--apple-id",
				display: "Apple ID", url: urlAppleIDPass,
				how: "Your Apple ID (for notarization).", required: false},
			{tomlSection: "macos", tomlKey: "password", env: "IRGO_APPLE_PASSWORD", flag: "--password",
				display: "App-specific password", url: urlAppleIDPass,
				how: "appleid.apple.com → Sign-In & Security → App-Specific Passwords → generate one.", required: false, secret: true},
		}
	case "reviews-apple-ios", "reviews-apple-mac": // App Store review monitoring
		appIDKey := "ios_app_id"
		appIDEnv := "IRGO_IOS_APP_ID"
		appIDDisplay := "iOS app numeric ID"
		if store == "reviews-apple-mac" {
			appIDKey = "mac_app_id"
			appIDEnv = "IRGO_MAC_APP_ID"
			appIDDisplay = "Mac app numeric ID"
		}
		return []configValue{
			{tomlSection: "reviews", tomlKey: appIDKey, env: appIDEnv, flag: "",
				display: appIDDisplay, url: "",
				how: "From the app's App Store URL: apps.apple.com/app/id<THIS NUMBER>", required: true},
			{tomlSection: "reviews", tomlKey: "ios_key_id", env: "IRGO_ASC_KEY_ID", flag: "",
				display: "App Store Connect API Key ID", url: urlASCAPIKeys,
				how: "App Store Connect → Users and Access → Integrations → API keys → generate (key needs Customer Reviews access). Key ID is 10 chars.", required: true},
			{tomlSection: "reviews", tomlKey: "ios_issuer_id", env: "IRGO_ASC_ISSUER_ID", flag: "",
				display: "App Store Connect API Issuer ID", url: urlASCAPIKeys,
				how: "Same page — Issuer ID is a UUID.", required: true},
			{tomlSection: "reviews", tomlKey: "ios_private_key", env: "IRGO_ASC_PRIVATE_KEY", flag: "",
				display: "App Store Connect API private key (.p8)", url: urlASCAPIKeys,
				how: "Download the .p8 when you create the key (downloadable once).", required: true, secret: true},
		}
	case "reviews-android":
		return []configValue{
			{tomlSection: "reviews", tomlKey: "android_package", env: "IRGO_ANDROID_PACKAGE", flag: "",
				display: "Play package name", url: urlPlayConsole,
				how: "Your Play package name, e.g. com.example.myapp", required: true},
			{tomlSection: "reviews", tomlKey: "android_service_account", env: "IRGO_PLAY_SERVICE_ACCOUNT", flag: "",
				display: "Play service-account JSON", url: urlPlayConsole,
				how: "Google Play Console → Setup → API access → create service account → grant 'View app information'/'Reply to reviews' → download JSON.", required: true, secret: true},
		}
	}
	return nil
}

// valueFromEnv returns the value of an IRGO_* env var (the key itself).
func valueFromEnv(cv configValue) string {
	if cv.env == "" {
		return ""
	}
	return os.Getenv(cv.env)
}

// valueFromConfig returns the value from irgo.package.toml ("" if absent).
func valueFromConfig(cv configValue) string {
	cfg := parsePackageConfig()
	switch cv.tomlSection + "." + cv.tomlKey {
	case "ios.team":
		return cfg.IOSTeam
	case "android.keystore":
		return cfg.AndroidKeystore
	case "windows.publisher":
		return cfg.WindowsPublisher
	case "windows.cert":
		return cfg.WindowsCert
	case "macos.identity":
		return cfg.MacIdentity
	case "macos.apple_id":
		return cfg.MacAppleID
	case "macos.password":
		return cfg.MacPassword
	case "reviews.ios_app_id":
		return cfg.ReviewsIOSAppID
	case "reviews.mac_app_id":
		return cfg.ReviewsMacAppID
	case "reviews.ios_key_id":
		return cfg.ReviewsIOSKeyID
	case "reviews.ios_issuer_id":
		return cfg.ReviewsIOSIssuerID
	case "reviews.ios_private_key":
		return cfg.ReviewsIOSPrivateKey
	case "reviews.android_package":
		return cfg.ReviewsAndroidPackage
	case "reviews.android_service_account":
		return cfg.ReviewsAndroidServiceAcc
	}
	return ""
}

// missingStoreConfig returns the required values that are neither in the toml
// nor in an env var, and can't be auto-derived.
func missingStoreConfig(store string) []configValue {
	var missing []configValue
	for _, cv := range storeConfigValues(store) {
		if !cv.required {
			continue
		}
		if valueFromEnv(cv) != "" || valueFromConfig(cv) != "" {
			continue
		}
		if derived := deriveConfigValue(cv); derived != "" {
			continue
		}
		missing = append(missing, cv)
	}
	return missing
}

// deriveConfigValue returns an auto-derived value for a config value (or "").
// iOS team → from the local Xcode project; package names → from the module
// path; macOS identity → from Keychain.
func deriveConfigValue(cv configValue) string {
	switch cv.tomlSection + "." + cv.tomlKey {
	case "ios.team":
		return deriveIOSTeamFromXcode()
	case "macos.identity":
		ids := macIdentities()
		if len(ids) == 1 {
			return ids[0]
		}
	case "reviews.android_package", "reviews.ios_app_id":
		// fall through to module-path derivation below
	default:
		return ""
	}
	if cv.tomlSection == "reviews" && cv.tomlKey == "android_package" {
		if mp, err := getModulePath(); err == nil {
			return bundleIDFromModulePath(mp)
		}
	}
	return ""
}

// ensureStoreConfig is called at the top of every package/reviews command. It
// guarantees the store's required config is present: interactively (dev) or
// with a precise fail-fast error (CI).
func ensureStoreConfig(store string) error {
	if err := writeDefaultPackageConfig(); err != nil {
		return err
	}
	missing := missingStoreConfig(store)
	if len(missing) == 0 {
		return nil
	}
	if interactive() {
		fmt.Printf("irgo needs a few things before it can %s. I'll open the\n", describeTarget(store))
		fmt.Println("right pages and write what you give me into irgo.package.toml.")
		fmt.Println()
		return runSetupWizard(store, missing)
	}
	return ciConfigError(store, missing)
}

func describeTarget(store string) string {
	switch store {
	case "ios":
		return "package the iOS app (.ipa)"
	case "android":
		return "package the Android app (.aab)"
	case "windows":
		return "package the Windows app (.msix)"
	case "macos":
		return "package the macOS app (.app/.dmg)"
	case "reviews-apple":
		return "monitor Apple App Store reviews"
	case "reviews-android":
		return "monitor Google Play reviews"
	}
	return store
}

// ciConfigError lists every missing value with its env var + URL, so CI logs
// are actionable and nothing ever hangs on a prompt.
func ciConfigError(store string, missing []configValue) error {
	var b strings.Builder
	fmt.Fprintf(&b, "%s is missing required config (irgo.package.toml is gitignored — set these as CI secrets / env vars, or pass --flags):\n", describeTarget(store))
	for _, cv := range missing {
		fmt.Fprintf(&b, "  - %s\n", cv.display)
		if cv.env != "" {
			fmt.Fprintf(&b, "      env:  %s\n", cv.env)
		}
		if cv.flag != "" {
			fmt.Fprintf(&b, "      flag: %s <value>\n", cv.flag)
		}
		if cv.url != "" {
			fmt.Fprintf(&b, "      get it at: %s (%s)\n", cv.url, cv.how)
		} else if cv.how != "" {
			fmt.Fprintf(&b, "      %s\n", cv.how)
		}
	}
	return fmt.Errorf("%s", b.String())
}

// runSetupWizard interactively fills the missing values, opening each page
// and writing answers into irgo.package.toml.
func runSetupWizard(store string, missing []configValue) error {
	scanner := bufio.NewScanner(os.Stdin)
	for _, cv := range missing {
		fmt.Printf("── %s ──\n", cv.display)
		if cv.how != "" {
			fmt.Printf("  %s\n", cv.how)
		}
		if cv.url != "" {
			fmt.Printf("  Opening %s\n", cv.url)
			openURL(cv.url)
		}
		fmt.Printf("  %s", cv.display)
		if cv.secret {
			fmt.Print(" (input hidden? if not, fine — it goes into the gitignored toml)")
		}
		fmt.Print(": ")
		if !scanner.Scan() {
			return fmt.Errorf("no input (interactive setup needs a terminal; in CI set %s)", cv.env)
		}
		val := strings.TrimSpace(scanner.Text())
		if val == "" {
			fmt.Printf("  (skipping %s)\n", cv.display)
			continue
		}
		if err := writeConfigValue(cv.tomlSection, cv.tomlKey, val); err != nil {
			return err
		}
		fmt.Printf("  ✓ wrote %s.%s\n", cv.tomlSection, cv.tomlKey)
	}
	fmt.Println()
	fmt.Println("Config updated. Run the command again and it should proceed.")
	return nil
}

// writeConfigValue sets a key in irgo.package.toml, preserving comments and
// creating the section if needed.
func writeConfigValue(section, key, value string) error {
	lines, err := os.ReadFile(packageConfigFile)
	if err != nil {
		return err
	}
	text := strings.Split(string(lines), "\n")
	curSection := ""
	wrote := false
	for i := range text {
		line := strings.TrimSpace(text[i])
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			curSection = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		if curSection != section {
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}
		k := strings.TrimSpace(line[:eq])
		if k != key {
			continue
		}
		text[i] = fmt.Sprintf("%s = %q", key, value)
		wrote = true
		break
	}
	if !wrote {
		if curSection != section {
			text = append(text, "", fmt.Sprintf("[%s]", section))
		}
		text = append(text, fmt.Sprintf("%s = %q", key, value))
	}
	return os.WriteFile(packageConfigFile, []byte(strings.Join(text, "\n")), 0o644)
}

// ---------------------------------------------------------------------------
// helpers: TTY, URL opening, derivation
// ---------------------------------------------------------------------------

// interactive reports whether we should prompt (a real terminal, not CI).
func interactive() bool {
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		return false
	}
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// openURL opens a URL in the default browser, best-effort.
func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

// deriveIOSTeamFromXcode reads DEVELOPMENT_TEAM from the Example Xcode project
// (only on macOS, only when the project exists).
func deriveIOSTeamFromXcode() string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	proj := filepath.Join("ios", "Example", "Example.xcodeproj")
	if !isDir(proj) {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "xcodebuild",
		"-project", proj, "-showBuildSettings", "-json").Output()
	if err != nil {
		return ""
	}
	var settings []struct {
		BuildSettings struct {
			DevelopmentTeam string `json:"DEVELOPMENT_TEAM"`
		} `json:"buildSettings"`
	}
	if err := json.Unmarshal(out, &settings); err != nil {
		return ""
	}
	for _, s := range settings {
		if s.BuildSettings.DevelopmentTeam != "" {
			return s.BuildSettings.DevelopmentTeam
		}
	}
	return ""
}

// macIdentities lists installed Developer ID Application identities from the
// Keychain ("Name (TEAMID)"), for auto-derivation and the wizard.
func macIdentities() []string {
	if runtime.GOOS != "darwin" {
		return nil
	}
	out, err := exec.Command("security", "find-identity", "-p", "codesigning", "-v").Output()
	if err != nil {
		return nil
	}
	var ids []string
	for _, line := range strings.Split(string(out), "\n") {
		i := strings.Index(line, `"`)
		j := strings.LastIndex(line, `"`)
		if i >= 0 && j > i {
			ids = append(ids, line[i+1:j])
		}
	}
	return ids
}

// checkStoreConfig prints a per-value status report with URLs + env vars.
func checkStoreConfig(store string) {
	fmt.Printf("── %s ──\n", describeTarget(store))
	for _, cv := range storeConfigValues(store) {
		val, src := resolveConfigValue(cv)

		label := val
		switch {
		case val == "":
			label = "—"
		case cv.secret:
			label = "•••••• (set)"
		}

		status := "✓"
		if val == "" {
			status = "✗"
			if !cv.required {
				status = "·" // optional: a default covers it
			}
		}
		fmt.Printf("  %s %-30s %-24s [%s]\n", status, cv.display, label, src)

		if val != "" {
			continue
		}
		// Say exactly how to supply it — in CI that means the env var name,
		// which is what a repository secret has to be called.
		if cv.env != "" {
			fmt.Printf("        set it:    %s=...   (CI secret, or your shell)\n", cv.env)
		}
		if cv.flag != "" {
			fmt.Printf("        or flag:   irgo package %s %s ...\n", store, cv.flag)
		}
		if cv.secret {
			fmt.Printf("        or file:   %s   (gitignored)\n", packageLocalFile)
		} else if cv.tomlSection != "" {
			fmt.Printf("        or file:   %s   [%s] %s\n", packageConfigFile, cv.tomlSection, cv.tomlKey)
		}
		if cv.url != "" {
			fmt.Printf("        get it at: %s\n", cv.url)
			if cv.how != "" {
				fmt.Printf("                   %s\n", cv.how)
			}
		}
	}
}

// resolveConfigValue applies the documented precedence and reports which layer
// answered, so a surprising value can be traced to its source.
func resolveConfigValue(cv configValue) (val, src string) {
	if v := valueFromEnv(cv); v != "" {
		return v, "env " + cv.env
	}
	if v := valueFromConfig(cv); v != "" {
		// parsePackageConfig already merged the local overlay; report the file
		// that actually carries the key.
		return v, configSourceFile(cv)
	}
	if v := deriveConfigValue(cv); v != "" {
		return v, "auto-derived"
	}
	if !cv.required {
		return "", "default"
	}
	return "", "missing"
}

// configSourceFile reports which of the two files a key came from.
func configSourceFile(cv configValue) string {
	if fileHasKey(packageLocalFile, cv.tomlSection, cv.tomlKey) {
		return packageLocalFile
	}
	return packageConfigFile
}

// fileHasKey reports whether a toml file sets section.key. Deliberately as
// simple as parsePackageConfigFile — this is our own file format.
func fileHasKey(path, section, key string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
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
		if eq < 0 || cur != section {
			continue
		}
		if strings.TrimSpace(line[:eq]) != key {
			continue
		}
		return strings.Trim(strings.TrimSpace(line[eq+1:]), `"'`) != ""
	}
	return false
}
