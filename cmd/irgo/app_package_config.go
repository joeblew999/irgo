package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// irgo app package setup — walk a dev through filling in the store config.
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

// A configValue is one setting this project can carry: where it sits in the
// toml, the env var and flag that override it, what a human needs in order to
// find it, which build targets need it, and — for the private ones — what it
// is for.
//
// This is the only place a setting is declared. The struct field it lands in
// is named here as an accessor, so parsing a file, reading a value, prompting
// for it, checking it, and deciding where it may be copied are all projections
// of one list.
//
// Each of those used to be its own switch statement. A setting added to three
// of them and missed in the fourth read back empty, and nothing failed to
// compile — which is how `android.key_alias` could be parsed but unreadable,
// and how a signing credential could satisfy `secrets list` while `app package`
// insisted it was missing.
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
	setup       bool   // `app package setup` walks a human through it

	// pattern is what a valid value looks like, empty when anything goes.
	// Checked before a build starts and before a value is pushed anywhere: a
	// Team ID with a typo in it is otherwise found twenty minutes into a
	// macOS runner, by codesign, in a message about provisioning profiles.
	pattern string
	// shape describes the pattern for a human, since a regexp does not.
	shape string

	// path marks a value that names a file rather than carrying its contents.
	// CI has no filesystem to point at, so these travel as <ENV>_BASE64 and
	// are written back out to a real file before a build looks for one.
	path bool

	// targets are the build targets that need this value. It is what makes
	// the config per-app-type rather than one flat bag: `app package setup`
	// asks only about the target being packaged, and `secrets list` can say
	// which platform a credential is for.
	targets []string

	// role classifies the private ones, which decides where they may be
	// copied. Only meaningful when secret.
	role secretRole

	// field is where the value lands in packageConfig. Exactly one of field
	// or boolField is set for anything the toml carries; a value that lives
	// only in the environment (a deploy token) has neither.
	field     func(*packageConfig) *string
	boolField func(*packageConfig) *bool
}

// The targets a value can belong to. Named rather than spelled inline so a
// typo is a compile error instead of a setting that quietly belongs to no
// target and is therefore never asked for.
var (
	forPackaging      = []string{"ios", "android", "windows", "macos"}
	forIOS            = []string{"ios"}
	forAndroid        = []string{"android"}
	forWindows        = []string{"windows"}
	forMacOS          = []string{"macos"}
	forCloudflare     = []string{"cloudflare"}
	forAppleReviews   = []string{"reviews-apple-ios", "reviews-apple-mac"}
	forIOSReviews     = []string{"reviews-apple-ios"}
	forMacReviews     = []string{"reviews-apple-mac"}
	forAndroidReviews = []string{"reviews-android"}
)

// configStores is every target whose values irgo knows about, in the order a
// human should see them. Its contents are checked against the registry by
// test, so a target that gains a setting and is missed here is a failure
// rather than a silently invisible store.
var configStores = []string{"ios", "android", "windows", "macos", "cloudflare",
	"reviews-apple-ios", "reviews-apple-mac", "reviews-android"}

// configRegistry is every setting irgo understands.
var configRegistry = []configValue{
	// ---- common -----------------------------------------------------------
	{tomlSection: "common", tomlKey: "version", env: "IRGO_VERSION", targets: forPackaging,
		display: "Version", how: "Version string used as the Android versionName and the Windows version.",
		field: func(c *packageConfig) *string { return &c.Version }},
	{tomlSection: "common", tomlKey: "icon", env: "IRGO_ICON", targets: forPackaging,
		display: "App icon", how: "One source icon, scaled for every store.",
		field: func(c *packageConfig) *string { return &c.Icon }},

	// ---- iOS --------------------------------------------------------------
	{tomlSection: "ios", tomlKey: "team", env: "IRGO_IOS_TEAM", flag: "--team",
		display: "Apple Team ID", url: urlASCMembership,
		how:      "Membership → Team ID (10-character ID)",
		required: true, setup: true, targets: forIOS,
		pattern: `^[A-Z0-9]{10}$`, shape: "10 characters, letters and digits",
		field: func(c *packageConfig) *string { return &c.IOSTeam }},
	{tomlSection: "ios", tomlKey: "export_method", env: "IRGO_IOS_EXPORT_METHOD", targets: forIOS,
		display: "Export method", how: "development, ad-hoc, app-store or enterprise.",
		pattern: `^(development|ad-hoc|app-store|enterprise)$`, shape: "development, ad-hoc, app-store or enterprise",
		field: func(c *packageConfig) *string { return &c.IOSExportMethod }},

	// ---- Android ----------------------------------------------------------
	{tomlSection: "android", tomlKey: "keystore", env: "IRGO_ANDROID_KEYSTORE", flag: "--keystore",
		display: "Signing keystore", url: urlPlayConsole,
		how:   "Debug keystore works by default (validation only). For release: keytool -genkey (see `irgo app package setup`); keep it safe — it can't change after first upload.",
		setup: true, role: roleSigning, targets: forAndroid,
		path:  true,
		field: func(c *packageConfig) *string { return &c.AndroidKeystore }},
	{tomlSection: "android", tomlKey: "keystore_pass", env: "IRGO_ANDROID_KEYSTORE_PASS", targets: forAndroid,
		display: "Keystore password", secret: true, role: roleSigning,
		field: func(c *packageConfig) *string { return &c.AndroidKeystorePw }},
	{tomlSection: "android", tomlKey: "key_alias", env: "IRGO_ANDROID_KEY_ALIAS", targets: forAndroid,
		display: "Key alias",
		field:   func(c *packageConfig) *string { return &c.AndroidKeyAlias }},
	{tomlSection: "android", tomlKey: "key_pass", env: "IRGO_ANDROID_KEY_PASS", targets: forAndroid,
		display: "Key password", secret: true, role: roleSigning,
		field: func(c *packageConfig) *string { return &c.AndroidKeyPass }},

	// ---- Windows ----------------------------------------------------------
	{tomlSection: "windows", tomlKey: "publisher", env: "IRGO_WINDOWS_PUBLISHER", flag: "--publisher",
		display: "Publisher (CN=...)", url: urlPartnerCenter,
		how:   "Partner Center → your app → Product management → Product identity → Publisher display name. Empty = test-signed.",
		setup: true, targets: forWindows,
		pattern: `^CN=`, shape: "starts with CN=",
		field: func(c *packageConfig) *string { return &c.WindowsPublisher }},
	{tomlSection: "windows", tomlKey: "cert", env: "IRGO_WINDOWS_CERT", flag: "--cert",
		display: "Code-signing cert (PFX)", url: urlWinSDK,
		how:   "Optional real cert; empty = self-signed test cert (Store re-signs on upload).",
		setup: true, role: roleSigning, targets: forWindows,
		path:  true,
		field: func(c *packageConfig) *string { return &c.WindowsCert }},
	{tomlSection: "windows", tomlKey: "cert_pass", env: "IRGO_WINDOWS_CERT_PASS", targets: forWindows,
		display: "Cert password", secret: true, role: roleSigning,
		field: func(c *packageConfig) *string { return &c.WindowsCertPass }},
	{tomlSection: "windows", tomlKey: "icon", env: "IRGO_WINDOWS_ICON", targets: forWindows,
		display: "Windows icon",
		field:   func(c *packageConfig) *string { return &c.WindowsIcon }},

	// ---- macOS ------------------------------------------------------------
	{tomlSection: "macos", tomlKey: "identity", env: "IRGO_MACOS_IDENTITY", flag: "--identity",
		display: "Developer ID Application identity", url: urlASCCerts,
		how:   "Certificates → Developer ID Application → download + import to Keychain. Auto-picked from your Keychain if empty.",
		setup: true, targets: forMacOS,
		field: func(c *packageConfig) *string { return &c.MacIdentity }},
	{tomlSection: "macos", tomlKey: "notarize", env: "IRGO_MACOS_NOTARIZE", flag: "--notarize",
		display: "Notarize?",
		how:     "Requires Apple ID + Team ID + an app-specific password (below).",
		setup:   true, targets: forMacOS,
		boolField: func(c *packageConfig) *bool { return &c.MacNotarize }},
	{tomlSection: "macos", tomlKey: "apple_id", env: "IRGO_APPLE_ID", flag: "--apple-id",
		display: "Apple ID", url: urlAppleIDPass,
		how:   "Your Apple ID (for notarization).",
		setup: true, targets: forMacOS,
		field: func(c *packageConfig) *string { return &c.MacAppleID }},
	{tomlSection: "macos", tomlKey: "team", env: "IRGO_MACOS_TEAM", targets: forMacOS,
		display: "Team ID (notarization)",
		field:   func(c *packageConfig) *string { return &c.MacTeam }},
	{tomlSection: "macos", tomlKey: "password", env: "IRGO_APPLE_PASSWORD", flag: "--password",
		display: "App-specific password", url: urlAppleIDPass,
		how:   "appleid.apple.com → Sign-In & Security → App-Specific Passwords → generate one.",
		setup: true, secret: true, role: roleSigning, targets: forMacOS,
		field: func(c *packageConfig) *string { return &c.MacPassword }},
	{tomlSection: "macos", tomlKey: "dmg", env: "IRGO_MACOS_DMG", targets: forMacOS,
		display:   "Build a DMG?",
		boolField: func(c *packageConfig) *bool { return &c.MacDMG }},

	// ---- Cloudflare -------------------------------------------------------
	//
	// No toml field: these authenticate irgo itself, so they only ever come
	// from the environment or the keychain. Declared here so that classifying
	// them is a lookup rather than a guess at their name — `push` used to
	// decide by string prefix, which is a heuristic standing in for a fact
	// irgo already had.
	{env: "CLOUDFLARE_API_TOKEN", display: "Cloudflare API token",
		url:    "https://dash.cloudflare.com/profile/api-tokens",
		how:    "My Profile → API Tokens → Create Token → Edit Cloudflare Workers.",
		secret: true, role: roleDeploy, targets: forCloudflare},
	{env: "CLOUDFLARE_ACCOUNT_ID", display: "Cloudflare account ID",
		url:    "https://dash.cloudflare.com",
		how:    "Workers & Pages → Account details → Account ID.",
		secret: true, role: roleDeploy, targets: forCloudflare},

	// ---- App Store review monitoring --------------------------------------
	{tomlSection: "reviews", tomlKey: "ios_app_id", env: "IRGO_IOS_APP_ID",
		display:  "iOS app numeric ID",
		how:      "From the app's App Store URL: apps.apple.com/app/id<THIS NUMBER>",
		required: true, setup: true, targets: forIOSReviews,
		pattern: `^[0-9]+$`, shape: "digits only, from the App Store URL",
		field: func(c *packageConfig) *string { return &c.ReviewsIOSAppID }},
	{tomlSection: "reviews", tomlKey: "mac_app_id", env: "IRGO_MAC_APP_ID",
		display:  "Mac app numeric ID",
		how:      "From the app's App Store URL: apps.apple.com/app/id<THIS NUMBER>",
		required: true, setup: true, targets: forMacReviews,
		pattern: `^[0-9]+$`, shape: "digits only, from the App Store URL",
		field: func(c *packageConfig) *string { return &c.ReviewsMacAppID }},
	{tomlSection: "reviews", tomlKey: "ios_key_id", env: "IRGO_ASC_KEY_ID",
		display: "App Store Connect API Key ID", url: urlASCAPIKeys,
		how:      "App Store Connect → Users and Access → Integrations → API keys → generate (key needs Customer Reviews access). Key ID is 10 chars.",
		required: true, setup: true, targets: forAppleReviews,
		pattern: `^[A-Z0-9]{10}$`, shape: "10 characters, letters and digits",
		field: func(c *packageConfig) *string { return &c.ReviewsIOSKeyID }},
	{tomlSection: "reviews", tomlKey: "ios_issuer_id", env: "IRGO_ASC_ISSUER_ID",
		display: "App Store Connect API Issuer ID", url: urlASCAPIKeys,
		how:      "Same page — Issuer ID is a UUID.",
		required: true, setup: true, targets: forAppleReviews,
		pattern: `^[0-9a-fA-F-]{36}$`, shape: "a UUID",
		field: func(c *packageConfig) *string { return &c.ReviewsIOSIssuerID }},
	{tomlSection: "reviews", tomlKey: "ios_private_key", env: "IRGO_ASC_PRIVATE_KEY",
		display: "App Store Connect API private key (.p8)", url: urlASCAPIKeys,
		how:      "Download the .p8 when you create the key (downloadable once).",
		required: true, setup: true, role: roleSigning, targets: forAppleReviews,
		path:  true,
		field: func(c *packageConfig) *string { return &c.ReviewsIOSPrivateKey }},
	{tomlSection: "reviews", tomlKey: "android_package", env: "IRGO_ANDROID_PACKAGE",
		display: "Play package name", url: urlPlayConsole,
		how:      "Your Play package name, e.g. com.example.myapp",
		required: true, setup: true, targets: forAndroidReviews,
		pattern: `^[a-z][a-z0-9_]*(\.[a-z0-9_]+)+$`, shape: "a reverse-domain name, e.g. com.example.myapp",
		field: func(c *packageConfig) *string { return &c.ReviewsAndroidPackage }},
	{tomlSection: "reviews", tomlKey: "android_service_account", env: "IRGO_PLAY_SERVICE_ACCOUNT",
		display: "Play service-account JSON", url: urlPlayConsole,
		how:      "Google Play Console → Setup → API access → create service account → grant 'View app information'/'Reply to reviews' → download JSON.",
		required: true, setup: true, role: roleSigning, targets: forAndroidReviews,
		path:  true,
		field: func(c *packageConfig) *string { return &c.ReviewsAndroidServiceAcc }},
}

// sensitive reports whether what this setting carries has to be kept private.
//
// Two different things, which is why they are two flags:
//
//	secret — the value itself is private. A password, a token. It must never
//	         be written to a committed file and must never be echoed back.
//	path   — the value is a filename. The NAME is safe to commit and belongs
//	         in the committed config; the FILE it points at is private, and
//	         when it travels to CI it travels as contents — at which point it
//	         is exactly as private as a secret.
//
// Conflating them was a real hole. android.keystore is a path and was not
// marked secret, so classifying it fell through to "the app's own runtime
// value", and `secrets push cloudflare` would have base64'd the Android
// signing key into the Worker. That key cannot be reissued: lose control of it
// and the app can never be updated on Play again.
//
// So classification, pushing and withholding ask this. Masking and which file
// a value may be written to ask `secret` alone, because hiding a keystore's
// path helps nobody and the path belongs in the committed config.
func (cv configValue) sensitive() bool { return cv.secret || cv.path }

// registryFor finds the setting a toml section and key name, nil when the file
// carries something irgo does not understand.
func registryFor(section, key string) *configValue {
	for i := range configRegistry {
		if configRegistry[i].tomlSection == section && configRegistry[i].tomlKey == key {
			return &configRegistry[i]
		}
	}
	return nil
}

// registryForEnv finds the setting an environment variable names.
func registryForEnv(env string) *configValue {
	for i := range configRegistry {
		if configRegistry[i].env == env {
			return &configRegistry[i]
		}
	}
	return nil
}

// storeConfigValues returns the values `app package setup` walks a human
// through for one target, in registry order.
func storeConfigValues(store string) []configValue {
	var out []configValue
	for _, cv := range configRegistry {
		if cv.setup && slices.Contains(cv.targets, store) {
			out = append(out, cv)
		}
	}
	return out
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
	switch {
	case cv.field != nil:
		return *cv.field(&cfg)
	case cv.boolField != nil:
		// Empty rather than "false" when unset, so an unset flag still reads
		// as "take the default" everywhere that tests a value for emptiness.
		if *cv.boolField(&cfg) {
			return "true"
		}
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
			return androidPackageFromModulePath(mp)
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
			fmt.Printf(" (goes into %s, which is gitignored)", packageLocalFile)
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
		if err := writeConfigValueFor(cv, val); err != nil {
			return err
		}
		dest := packageConfigFile
		if cv.secret {
			dest = packageLocalFile
		}
		fmt.Printf("  ✓ wrote %s.%s to %s\n", cv.tomlSection, cv.tomlKey, dest)
	}
	fmt.Println()
	fmt.Println("Config updated. Run the command again and it should proceed.")
	return nil
}

// writeConfigValue sets a key in irgo.package.toml, preserving comments and
// creating the section if needed.
func writeConfigValue(section, key, value string) error {
	return writeConfigValueTo(packageConfigFile, section, key, value)
}

// writeConfigValueFor writes a setting to whichever file may legitimately hold
// it: a secret to the gitignored overlay, everything else to the committed
// config.
//
// The wizard used to write everything to the committed file while telling you
// it was doing the opposite — "it goes into the gitignored toml" — so running
// `irgo app package setup macos` and answering the app-specific password
// prompt put that password straight into a file destined for git.
// `project config` had the same shape: it printed a note saying a secret
// belongs in the local file, then wrote it to the committed one.
func writeConfigValueFor(cv configValue, value string) error {
	if cv.secret {
		return writeConfigValueTo(packageLocalFile, cv.tomlSection, cv.tomlKey, value)
	}
	return writeConfigValueTo(packageConfigFile, cv.tomlSection, cv.tomlKey, value)
}

func writeConfigValueTo(file, section, key, value string) error {
	lines, err := os.ReadFile(file)
	if err != nil {
		// A project generated before the config existed, or one where it was
		// deleted, should still be able to record a setting rather than fail
		// with a bare "no such file".
		if !os.IsNotExist(err) {
			return err
		}
		seed := "# irgo app package configuration — see: irgo app package setup\n"
		if file == packageLocalFile {
			seed = "# Secrets for this machine. Gitignored — never commit this file.\n"
		}
		if werr := os.WriteFile(file, []byte(seed), configPerm(file)); werr != nil {
			return fmt.Errorf("creating %s: %w", file, werr)
		}
		lines = []byte(seed)
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
	return os.WriteFile(file, []byte(strings.Join(text, "\n")), configPerm(file))
}

// configPerm keeps the secret overlay unreadable by anyone else on the
// machine. The committed config holds nothing private, so it stays ordinary.
func configPerm(file string) os.FileMode {
	if file == packageLocalFile {
		return 0o600
	}
	return 0o644
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

		// A value that is set but malformed is worse than one that is
		// missing: it looks configured, and the tool that rejects it is
		// several steps away and talking about something else.
		invalid := validateConfigValue(cv, val)

		status := "✓"
		switch {
		case invalid != nil:
			status = "!"
		case val == "":
			status = "✗"
			if !cv.required {
				status = "·" // optional: a default covers it
			}
		}
		fmt.Printf("  %s %-30s %-24s [%s]\n", status, cv.display, label, src)

		if invalid != nil {
			fmt.Printf("        problem:   %v\n", invalid)
			if cv.how != "" {
				fmt.Printf("                   %s\n", cv.how)
			}
			continue
		}
		if val != "" {
			continue
		}
		// Say exactly how to supply it — in CI that means the env var name,
		// which is what a repository secret has to be called.
		if cv.env != "" {
			fmt.Printf("        set it:    %s=...   (CI secret, or your shell)\n", cv.env)
		}
		if cv.flag != "" {
			fmt.Printf("        or flag:   irgo app package %s %s ...\n", store, cv.flag)
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

// validateConfigValue checks a resolved value against what the setting says a
// valid one looks like.
//
// The failure this prevents is expensive and confusing in equal measure: a
// Team ID with a character missing is not rejected by anything until codesign
// runs on a macOS runner, twenty minutes in, and what it says is that no
// provisioning profile matches — which sends you looking at certificates.
//
// A value that is set but wrong is a different thing from one that is unset,
// so this never reports on an empty value; that is what `required` is for.
func validateConfigValue(cv configValue, value string) error {
	if value == "" {
		return nil
	}
	if cv.pattern != "" && !regexp.MustCompile(cv.pattern).MatchString(value) {
		return fmt.Errorf("%q is not %s", value, cv.shape)
	}
	if cv.path {
		// A path setting must hold a path. Key material pasted in here would
		// be written to the committed config, which is the one place it must
		// never go — and it would look like it had worked.
		if strings.Contains(value, "-----BEGIN") || strings.ContainsAny(value, "\n\r") {
			return fmt.Errorf("this is the contents of a key, not a path to one\n"+
				"  put the file somewhere gitignored and name it here, or set %s%s in CI",
				cv.env, base64Suffix)
		}
		// A path that is not there fails later as something less obvious — an
		// empty keystore, or a signing step that quietly produces nothing.
		if _, err := os.Stat(expandHome(value)); err != nil {
			return fmt.Errorf("no file at %s", value)
		}
	}
	return nil
}

// configProblems reports every setting for a target that is set but invalid.
func configProblems(store string) map[string]error {
	out := map[string]error{}
	for _, cv := range storeConfigValues(store) {
		val, _ := resolveConfigValue(cv)
		if err := validateConfigValue(cv, val); err != nil {
			out[cv.display] = err
		}
	}
	return out
}
