package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// package configuration (irgo.package.toml)
//
// `irgo new` scaffolds a default irgo.package.toml at the project root. The
// `irgo package` commands read it for defaults; explicit CLI flags override
// the file. Missing file = built-in defaults + CLI flags.
// ---------------------------------------------------------------------------

type packageConfig struct {
	Version           string // common.version (android versionName, windows version)
	Icon              string // common.icon (single source icon for all stores)
	IOSTeam           string // ios.team
	IOSExportMethod   string // ios.export_method
	AndroidKeystore   string // android.keystore
	AndroidKeystorePw string // android.keystore_pass
	AndroidKeyAlias   string // android.key_alias
	AndroidKeyPass    string // android.key_pass
	WindowsPublisher  string // windows.publisher
	WindowsCert       string // windows.cert
	WindowsCertPass   string // windows.cert_pass
	WindowsIcon       string // windows.icon
	MacIdentity       string // macos.identity
	MacNotarize       bool   // macos.notarize
	MacAppleID        string // macos.apple_id
	MacTeam           string // macos.team
	MacPassword       string // macos.password
	MacDMG            bool   // macos.dmg
}

const packageConfigFile = "irgo.package.toml"

// parsePackageConfig reads the simple, fixed-format irgo.package.toml:
//
//	[common]
//	version = "0.1.0"
//	[ios]
//	team = ""
//	...
//
// Returns an empty config when the file is absent or unreadable (callers fall
// back to built-in defaults + flags). Only `key = "value"` (with # comments
// and [section] headers) is supported — this is our own file format.
func parsePackageConfig() packageConfig {
	var cfg packageConfig
	data, err := os.ReadFile(packageConfigFile)
	if err != nil {
		return cfg
	}
	section := ""
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.Trim(strings.TrimSpace(line[eq+1:]), `"'`)
		switch section {
		case "common":
			switch key {
			case "version":
				cfg.Version = val
			case "icon":
				cfg.Icon = val
			}
		case "ios":
			switch key {
			case "team":
				cfg.IOSTeam = val
			case "export_method":
				cfg.IOSExportMethod = val
			}
		case "android":
			switch key {
			case "keystore":
				cfg.AndroidKeystore = val
			case "keystore_pass":
				cfg.AndroidKeystorePw = val
			case "key_alias":
				cfg.AndroidKeyAlias = val
			case "key_pass":
				cfg.AndroidKeyPass = val
			}
		case "windows":
			switch key {
			case "publisher":
				cfg.WindowsPublisher = val
			case "cert":
				cfg.WindowsCert = val
			case "cert_pass":
				cfg.WindowsCertPass = val
			case "icon":
				cfg.WindowsIcon = val
			}
		case "macos":
			switch key {
			case "identity":
				cfg.MacIdentity = val
			case "notarize":
				cfg.MacNotarize = val == "true" || val == "1" || val == "yes"
			case "apple_id":
				cfg.MacAppleID = val
			case "team":
				cfg.MacTeam = val
			case "password":
				cfg.MacPassword = val
			case "dmg":
				cfg.MacDMG = val == "true" || val == "1" || val == "yes"
			}
		}
	}
	return cfg
}

// writeDefaultPackageConfig scaffolds irgo.package.toml with documented
// defaults (missing-only, like the rest of the CLI).
func writeDefaultPackageConfig() error {
	if _, err := os.Stat(packageConfigFile); err == nil {
		return nil
	}
	content := `# irgo package configuration — defaults for ` + "`irgo package <ios|android|macos|windows>`" + `
#
# CLI flags override any value here (e.g. ` + "`irgo package android --keystore ...`" + `).
# Run ` + "`irgo package setup`" + ` for a step-by-step guide to obtaining every value
# below for each store.

[common]
version = "0.1.0"

[ios]
# Apple Team ID — 10-character ID. Get it: developer.apple.com → Membership →
# "Team ID". Requires a paid Apple Developer account (US$99/year). Required
# for automatic signing.
team = ""
# app-store (default) | ad-hoc | development
export_method = "app-store"

[android]
# Debug keystore works out of the box (validation only). For the Play Store
# create a release keystore and KEEP IT SAFE — you can't change it after your
# first upload:
#   keytool -genkey -v -keystore ~/keys/release.keystore -alias myapp \
#     -keyalg RSA -keysize 2048 -validity 10000
keystore = "~/.android/debug.keystore"
keystore_pass = "android"
# Empty = auto-detected from the keystore.
key_alias = ""
key_pass = "android"

[windows]
# Publisher DN — must match your Partner Center publisher identity. Get it:
# Partner Center → your app → Product management → Product identity →
# "Publisher display name"/Publisher ID (partner.microsoft.com). A Microsoft
# Store developer account costs US$19 one-time. Empty = test-signed with
# "CN=<app>".
publisher = ""
# Optional real code-signing cert (PFX) + password; empty = self-signed test cert.
cert = ""
cert_pass = ""
icon = ""

[macos]
# Developer ID Application certificate name, e.g.
# "Developer ID Application: Your Name (TEAMID)". Get it:
# developer.apple.com → Certificates → create a Developer ID Application cert →
# download + double-click to import into Keychain. Requires a paid account.
identity = ""
# Notarization: Apple ID + Team ID + an app-specific password
# (appleid.apple.com → Sign-In & Security → App-Specific Passwords).
notarize = false
apple_id = ""
team = ""
password = ""
# Also produce a .dmg
dmg = false
`
	if err := os.WriteFile(packageConfigFile, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("  created: %s (edit for packaging defaults)\n", packageConfigFile)
	return nil
}

// ---------------------------------------------------------------------------
// OS gating — a packaging target only runs on an OS that supports its tools.
// ---------------------------------------------------------------------------

func gateOS(target string) error {
	switch target {
	case "ios":
		if runtime.GOOS != "darwin" {
			return fmt.Errorf("irgo package ios requires macOS (Xcode + codesign); this host is %s — run it on a Mac or in CI on macos-latest", runtime.GOOS)
		}
	case "windows":
		if runtime.GOOS != "windows" {
			return fmt.Errorf("irgo package windows requires Windows (MakeAppx/signtool from the Windows SDK); this host is %s — run it on Windows or in CI on windows-latest", runtime.GOOS)
		}
	case "macos":
		if runtime.GOOS != "darwin" {
			return fmt.Errorf("irgo package macos requires macOS (Xcode codesign/notarytool/hdiutil); this host is %s — run it on a Mac or in CI on macos-latest", runtime.GOOS)
		}
	case "android":
		// Cross-platform: gradle + JDK + Android SDK run on every OS.
	}
	return nil
}

// packageSetupGuide prints where to get every store config value — run with
// `irgo package setup` and fill the results into irgo.package.toml.
func packageSetupGuide() {
	fmt.Print(`irgo package — what each store needs, and where to get it

Everything below goes into irgo.package.toml (created by ` + "`irgo package`" + ` or
` + "`irgo new`" + `). CLI flags override it per invocation. One source icon
(appicon.png or [common] icon) feeds every store.

COMMON
  version ..... your app version, e.g. "1.2.3" (android versionName, MSIX version)
  icon ........ one PNG, e.g. "appicon.png" (also picked up from static/icon.png)

iOS — App Store (.ipa)
  Account ..... Apple Developer Program, US$99/year: developer.apple.com
  team ........ 10-char "Team ID": developer.apple.com → Membership → Team ID
  Automatic signing handles certificates + provisioning via Xcode.

Android — Google Play (.aab)
  Account ..... Google Play Console, US$25 one-time: play.google.com/console
  Keystore .... Debug keystore works for validation. For release:
                keytool -genkey -v -keystore ~/keys/release.keystore \
                  -alias myapp -keyalg RSA -keysize 2048 -validity 10000
                KEEP IT SAFE — it cannot be changed after your first upload.
  key_alias ... auto-detected from the keystore if left empty.

Windows — Microsoft Store (.msix)
  Account ..... Partner Center, US$19 one-time: partner.microsoft.com
  publisher ... Partner Center → your app → Product identity →
                "Publisher display name"/Publisher ID (put the CN=... form
                in [windows] publisher). Empty = test-signed "CN=<app>".
  cert ........ optional real code-signing cert (PFX); empty = self-signed
                test cert. The Store re-signs on upload.

macOS — notarized distribution (.app/.dmg)
  Account ..... Apple Developer Program (same as iOS).
  identity .... "Developer ID Application: Your Name (TEAMID)":
                developer.apple.com → Certificates → Developer ID Application →
                download + double-click to import into Keychain.
  notarize .... true + apple_id (your Apple ID), team (same Team ID),
                password = app-specific password (appleid.apple.com →
                Sign-In & Security → App-Specific Passwords).

TIPS
  - ` + "`irgo package ios --team X`" + ` etc. overrides the config file for one run.
  - iOS/macOS packaging must run on macOS; MSIX must run on Windows;
    Android runs anywhere. The CLI enforces this per target.
  - Each target outputs to its own dist/<target>/ folder, never touching
    the unpackaged build/ trees.
`)
}

// projectBaseName returns the current project's directory name (for default
// artifact/display names).
func projectBaseName() string {
	if modulePath, err := getModulePath(); err == nil {
		if b := filepath.Base(modulePath); b != "." && b != "" && b != "/" {
			return b
		}
	}
	wd, _ := os.Getwd()
	if b := filepath.Base(wd); b != "" && b != "/" {
		return b
	}
	return "irgo"
}

// distPath returns the dedicated packaging output dir for a target, keeping
// packaged artifacts separate from the unpackaged build/ trees.
func distPath(target string) string {
	return filepath.Join("dist", target)
}

// ---------------------------------------------------------------------------
// iOS — signed IPA for App Store Connect
// ---------------------------------------------------------------------------

// packageIOS produces a signed IPA ready for App Store Connect:
//
//  1. build the Irgo.xcframework (the Example app links against it)
//  2. xcodebuild -archive (Release, generic/platform=iOS)
//  3. xcodebuild -exportArchive with an ExportOptions.plist (app-store)
//
// macOS-only (Xcode + codesign). The .ipa lands in dist/ios/; all intermediate
// artifacts (xcarchive, export dir) live in a temp dir so build/ios/ keeps only
// the unpackaged framework + DerivedData.
func packageIOS(team, exportMethod, out string) error {
	if err := gateOS("ios"); err != nil {
		return err
	}
	if err := checkTool("xcodebuild", "Install Xcode from the App Store"); err != nil {
		return err
	}
	if err := checkTool("xcrun", "Install Xcode Command Line Tools: xcode-select --install"); err != nil {
		return err
	}

	cfg := parsePackageConfig()
	if team == "" {
		team = cfg.IOSTeam
	}
	if team == "" {
		team = os.Getenv("IRGO_IOS_TEAM")
	}
	if team == "" {
		return fmt.Errorf("a Development Team is required for automatic signing\n" +
			"  set it in irgo.package.toml ([ios] team = \"...\"), pass --team <TEAM_ID>, or set IRGO_IOS_TEAM")
	}
	if exportMethod == "" {
		exportMethod = cfg.IOSExportMethod
	}
	if exportMethod == "" {
		exportMethod = "app-store"
	}

	// The canonical ios/Example app — scaffolded from the embedded templates
	// when missing, so `irgo package ios` works on a bare project.
	if err := scaffoldExamples(); err != nil {
		return fmt.Errorf("iOS example scaffold failed: %w", err)
	}

	// Build the framework the Example app links against.
	modulePath, err := getModulePath()
	if err != nil {
		return fmt.Errorf("could not determine module path: %w", err)
	}
	fmt.Println("Building iOS framework...")
	if err := buildIOS(modulePath); err != nil {
		return err
	}

	tmp, err := os.MkdirTemp("", "irgo-ipa-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	archivePath := filepath.Join(tmp, "Example.xcarchive")
	exportDir := filepath.Join(tmp, "export")
	optsPath := filepath.Join(tmp, "ExportOptions.plist")

	fmt.Println("Archiving iOS app (Release, device)...")
	archive := exec.Command(
		"xcodebuild",
		"-project", filepath.Join("ios", "Example", "Example.xcodeproj"),
		"-scheme", "Example",
		"-configuration", "Release",
		"-destination", "generic/platform=iOS",
		"-archivePath", archivePath,
		"-DEVELOPMENT_TEAM", team,
		"-allowProvisioningUpdates",
		"archive",
	)
	archive.Stdout = os.Stdout
	archive.Stderr = os.Stderr
	if err := archive.Run(); err != nil {
		return fmt.Errorf("xcodebuild archive failed: %w", err)
	}

	opts := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>method</key>
	<string>%s</string>
	<key>teamID</key>
	<string>%s</string>
</dict>
</plist>
`, exportMethod, team)
	if err := os.WriteFile(optsPath, []byte(opts), 0o644); err != nil {
		return err
	}

	fmt.Printf("Exporting IPA (%s)...\n", exportMethod)
	export := exec.Command(
		"xcodebuild",
		"-exportArchive",
		"-archivePath", archivePath,
		"-exportOptionsPlist", optsPath,
		"-exportPath", exportDir,
		"-allowProvisioningUpdates",
	)
	export.Stdout = os.Stdout
	export.Stderr = os.Stderr
	if err := export.Run(); err != nil {
		return fmt.Errorf("xcodebuild export failed: %w", err)
	}

	ipas, err := filepath.Glob(filepath.Join(exportDir, "*.ipa"))
	if err != nil || len(ipas) == 0 {
		return fmt.Errorf("no .ipa produced in %s", exportDir)
	}
	if out == "" {
		out = filepath.Join(distPath("ios"), "Example.ipa")
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := copyFile(ipas[0], out); err != nil {
		return fmt.Errorf("copying ipa: %w", err)
	}
	fmt.Printf("iOS package built: %s\n", out)
	return nil
}

// ---------------------------------------------------------------------------
// Android — signed AAB for the Play Store (cross-platform)
// ---------------------------------------------------------------------------

// packageAndroid produces a signed AAB via the Example project's gradle
// bundleRelease, feeding signing + version through -Pirgo.* properties (the
// template's app/build.gradle.kts reads them). Runs on any OS (gradle + JDK +
// Android SDK are cross-platform). Defaults to the debug keystore; a real
// keystore is required for Play Store submission.
func packageAndroid(keystore, keystorePass, keyAlias, keyPass, version, iconPath, out string) error {
	if err := gateOS("android"); err != nil {
		return err
	}
	if err := ensureAndroidToolchain(false, "irgo"); err != nil {
		return err
	}
	// JDK on PATH for keytool (and gradle).
	applyBestJDKToEnv()
	if err := writeDefaultPackageConfig(); err != nil {
		return err
	}
	cfg := parsePackageConfig()
	if keystore == "" {
		keystore = cfg.AndroidKeystore
	}
	if keystorePass == "" {
		keystorePass = cfg.AndroidKeystorePw
	}
	if keyAlias == "" {
		keyAlias = cfg.AndroidKeyAlias
	}
	if keyPass == "" {
		keyPass = cfg.AndroidKeyPass
	}
	if version == "" {
		version = cfg.Version
	}

	if err := scaffoldExamples(); err != nil {
		return fmt.Errorf("Android example scaffold failed: %w", err)
	}
	modulePath, err := getModulePath()
	if err != nil {
		return fmt.Errorf("could not determine module path: %w", err)
	}
	fmt.Println("Building Android AAR...")
	if err := buildAndroid(modulePath); err != nil {
		return err
	}

	// Single source icon → launcher mipmaps (template defaults otherwise).
	if ic := findAppIcon(iconPath); ic != "" {
		resDir := filepath.Join("android", "Example", "app", "src", "main", "res")
		if err := generateAndroidIcons(ic, resDir); err != nil {
			return err
		}
	}

	// Signing key: --keystore/config, else the debug keystore (generated on
	// demand) — same default as gogio.
	if keystore == "" {
		keystore = filepath.Join(homeDir(), ".android", "debug.keystore")
		keystorePass, keyPass = "android", "android"
	}
	keystore = expandHome(keystore)
	if keystorePass == "" {
		keystorePass = "android"
	}
	if keyPass == "" {
		keyPass = "android"
	}
	// Ensure the keystore exists BEFORE resolving the alias, so detection
	// sees the generated (or pre-existing) key.
	if _, err := os.Stat(keystore); os.IsNotExist(err) {
		fmt.Printf("Generating debug keystore at %s...\n", keystore)
		if err := generateDebugKeystore(keystore); err != nil {
			return err
		}
	}
	if keyAlias == "" {
		// Default to the first alias in the keystore — the Android SDK's own
		// debug.keystore uses "androiddebugkey", gogio-style ones use
		// "android"; a real release keystore may use anything.
		keyAlias = "android"
		if a, err := firstKeystoreAlias(keystore, keystorePass); err == nil && a != "" {
			keyAlias = a
		}
	}
	fmt.Printf("  signing: %s (alias %s)\n", keystore, keyAlias)

	gradlew := filepath.Join("android", "Example", "gradlew")
	if _, err := os.Stat(gradlew); os.IsNotExist(err) {
		return fmt.Errorf("gradlew not found in android/Example")
	}
	fmt.Println("Building signed AAB (gradle bundleRelease)...")
	cmd := exec.Command("./gradlew", "bundleRelease",
		"-Pirgo.keystore="+keystore,
		"-Pirgo.keystorePass="+keystorePass,
		"-Pirgo.keyAlias="+keyAlias,
		"-Pirgo.keyPass="+keyPass,
	)
	if version != "" {
		cmd.Args = append(cmd.Args,
			"-Pirgo.versionName="+version,
			"-Pirgo.versionCode="+versionToCode(version),
		)
	}
	cmd.Dir = filepath.Join("android", "Example")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gradle bundleRelease failed: %w", err)
	}

	aab := filepath.Join("android", "Example", "app", "build", "outputs", "bundle", "release", "app-release.aab")
	if _, err := os.Stat(aab); os.IsNotExist(err) {
		return fmt.Errorf("no AAB produced at %s", aab)
	}
	if out == "" {
		out = filepath.Join(distPath("android"), "Example.aab")
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := copyFile(aab, out); err != nil {
		return fmt.Errorf("copying aab: %w", err)
	}
	fmt.Printf("Android package built: %s\n", out)
	return nil
}

// versionToCode maps "major.minor.patch" to a monotonically increasing
// Android versionCode (major*10000 + minor*100 + patch).
func versionToCode(v string) string {
	var maj, min, pat int
	fmt.Sscanf(v, "%d.%d.%d", &maj, &min, &pat)
	return strconv.Itoa(maj*10000 + min*100 + pat)
}

func generateDebugKeystore(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	keytool, err := exec.LookPath("keytool")
	if err != nil {
		return fmt.Errorf("keytool not found (set JAVA_HOME or run 'irgo install-tools android'): %w", err)
	}
	cmd := exec.Command(keytool,
		"-genkey", "-v",
		"-keystore", path,
		"-alias", "android",
		"-keyalg", "RSA", "-keysize", "2048",
		"-validity", "10000",
		"-storepass", "android", "-keypass", "android",
		"-dname", "CN=Android Debug,O=Android,C=US",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// firstKeystoreAlias returns the first alias in an existing keystore (used to
// default the signing alias when the user didn't specify one). Handles both
// JKS output ("Alias name: X") and PKCS12 output ("X, 5 Aug 2026,
// PrivateKeyEntry,").
func firstKeystoreAlias(keystore, pass string) (string, error) {
	keytool, err := exec.LookPath("keytool")
	if err != nil {
		return "", err
	}
	out, err := exec.Command(keytool, "-list", "-keystore", keystore, "-storepass", pass).CombinedOutput()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Alias name:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Alias name:")), nil
		}
		// PKCS12: "androiddebugkey, 5 Aug 2026, PrivateKeyEntry,"
		if strings.Contains(line, ",") && strings.Contains(line, "PrivateKeyEntry") {
			if a := strings.TrimSpace(strings.SplitN(line, ",", 2)[0]); a != "" {
				return a, nil
			}
		}
	}
	return "", fmt.Errorf("no alias found in %s", keystore)
}

// ---------------------------------------------------------------------------
// Windows — MSIX for the Microsoft Store (Windows-only)
// ---------------------------------------------------------------------------

// packageWindows builds a (test-signed) MSIX on a Windows host: build the exe,
// lay out the package (AppxManifest + Assets + exe + static), pack with
// MakeAppx, sign with signtool (self-signed test cert unless --cert given).
// MSIX tooling only exists on Windows, so this is gated to Windows.
func packageWindows(publisher, version, iconPath, cert, certPass, out string) error {
	if err := gateOS("windows"); err != nil {
		return err
	}
	if err := writeDefaultPackageConfig(); err != nil {
		return err
	}
	cfg := parsePackageConfig()
	if publisher == "" {
		publisher = cfg.WindowsPublisher
	}
	if version == "" {
		version = cfg.Version
	}
	if cert == "" {
		cert = cfg.WindowsCert
	}
	if certPass == "" {
		certPass = cfg.WindowsCertPass
	}
	if iconPath == "" {
		iconPath = cfg.WindowsIcon
	}

	appName := projectBaseName()
	if err := buildDesktop(""); err != nil {
		return fmt.Errorf("windows build failed: %w", err)
	}

	// Single source icon (config/flag/appicon.png) → MSIX tile assets.
	if ic := findAppIcon(iconPath); ic != "" {
		iconPath = ic
	}

	makeappx, err := findWindowsSDKTool("MakeAppx.exe")
	if err != nil {
		return err
	}
	signtool, err := findWindowsSDKTool("signtool.exe")
	if err != nil {
		return err
	}

	tmp, err := os.MkdirTemp("", "irgo-msix-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	pkgDir := filepath.Join(tmp, "pkg")
	if err := os.MkdirAll(filepath.Join(pkgDir, "Assets"), 0o755); err != nil {
		return err
	}

	// Copy the built exe + static assets into the package payload.
	exeDir := filepath.Join("build", "desktop", "windows")
	exe := filepath.Join(exeDir, appName+".exe")
	if _, err := os.Stat(exe); os.IsNotExist(err) {
		matches, _ := filepath.Glob(filepath.Join(exeDir, "*.exe"))
		if len(matches) == 0 {
			return fmt.Errorf("no .exe found in %s", exeDir)
		}
		exe = matches[0]
	}
	exeBase := filepath.Base(exe)
	if err := copyFile(exe, filepath.Join(pkgDir, exeBase)); err != nil {
		return err
	}
	if isDir(filepath.Join(exeDir, "static")) {
		if err := copyTree(filepath.Join(exeDir, "static"), filepath.Join(pkgDir, "static")); err != nil {
			return err
		}
	}

	// Identity + version (4-part for MSIX).
	msixVersion := "1.0.0.0"
	if version != "" {
		parts := strings.Split(version, ".")
		for len(parts) < 4 {
			parts = append(parts, "0")
		}
		msixVersion = strings.Join(parts[:4], ".")
	}
	pub := publisher
	if pub == "" {
		pub = "CN=" + appName
	}
	identityName := "irgo." + strings.ToLower(appName)

	manifest := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<Package xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10"
         xmlns:uap="http://schemas.microsoft.com/appx/manifest/uap/windows10"
         xmlns:rescap="http://schemas.microsoft.com/appx/manifest/foundation/windows10/restrictedcapabilities"
         IgnorableNamespaces="uap rescap">
  <Identity Name="%s" Publisher="%s" Version="%s"/>
  <Properties>
    <DisplayName>%s</DisplayName>
    <PublisherDisplayName>%s</PublisherDisplayName>
    <Logo>Assets\StoreLogo.png</Logo>
  </Properties>
  <Dependencies>
    <TargetDeviceFamily Name="Windows.Universal" MinVersion="10.0.17763.0" MaxVersionTested="10.0.22621.0"/>
  </Dependencies>
  <Resources>
    <Resource Language="en-us"/>
  </Resources>
  <Applications>
    <Application Id="App" Executable="%s" EntryPoint="Windows.FullTrustApplication">
      <uap:VisualElements DisplayName="%s" Square150x150Logo="Assets\Square150x150Logo.png"
        Square44x44Logo="Assets\Square44x44Logo.png" Description="%s"
        BackgroundColor="transparent"/>
    </Application>
  </Applications>
  <Capabilities>
    <rescap:Capability Name="runFullTrust"/>
  </Capabilities>
</Package>
`, identityName, pub, msixVersion, appName, appName, exeBase, appName, appName)
	if err := os.WriteFile(filepath.Join(pkgDir, "AppxManifest.xml"), []byte(manifest), 0o644); err != nil {
		return err
	}

	// Visual assets (solid-color placeholders unless --icon is provided).
	if err := writeMSIXAssets(filepath.Join(pkgDir, "Assets"), iconPath, appName); err != nil {
		return err
	}

	// MakeAppx pack.
	msix := filepath.Join(tmp, "app.msix")
	fmt.Println("Packing MSIX...")
	pack := exec.Command(makeappx, "pack", "/d", pkgDir, "/p", msix, "/o")
	pack.Stdout = os.Stdout
	pack.Stderr = os.Stderr
	if err := pack.Run(); err != nil {
		return fmt.Errorf("MakeAppx pack failed: %w", err)
	}

	// Sign — --cert, else a self-signed test certificate.
	if cert == "" {
		fmt.Println("Generating self-signed test certificate...")
		cert, certPass, err = generateTestCert(tmp)
		if err != nil {
			return err
		}
	}
	sign := exec.Command(signtool, "sign", "/fd", "SHA256", "/f", cert)
	if certPass != "" {
		sign.Args = append(sign.Args, "/p", certPass)
	}
	sign.Args = append(sign.Args, msix)
	sign.Stdout = os.Stdout
	sign.Stderr = os.Stderr
	if err := sign.Run(); err != nil {
		return fmt.Errorf("signtool sign failed: %w", err)
	}

	if out == "" {
		out = filepath.Join(distPath("windows"), appName+".msix")
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := copyFile(msix, out); err != nil {
		return err
	}
	fmt.Printf("Windows package built: %s\n", out)
	return nil
}

// ---------------------------------------------------------------------------
// macOS — signed + notarized .app (and optional .dmg) for distribution
// ---------------------------------------------------------------------------

// packageMacOS packages the desktop .app for distribution: copies it to
// dist/macos/, codesigns it (--identity / [macos] identity, hardened runtime
// entitlements), optionally notarizes + staples (--notarize + Apple
// credentials), and optionally wraps it in a .dmg (--dmg). macOS-only.
func packageMacOS(identity string, notarize bool, appleID, team, password string, dmg bool, iconPath, out string) error {
	if err := gateOS("macos"); err != nil {
		return err
	}
	if err := writeDefaultPackageConfig(); err != nil {
		return err
	}
	cfg := parsePackageConfig()
	if identity == "" {
		identity = cfg.MacIdentity
	}
	if !notarize {
		notarize = cfg.MacNotarize
	}
	if appleID == "" {
		appleID = cfg.MacAppleID
	}
	if team == "" {
		team = cfg.MacTeam
	}
	if password == "" {
		password = cfg.MacPassword
	}
	if !dmg {
		dmg = cfg.MacDMG
	}

	appName := projectBaseName()
	if err := buildDesktop(""); err != nil {
		return fmt.Errorf("macos build failed: %w", err)
	}
	built := filepath.Join("build", "desktop", "macos", appName+".app")
	if !isDir(built) {
		matches, _ := filepath.Glob(filepath.Join("build", "desktop", "macos", "*.app"))
		if len(matches) == 0 {
			return fmt.Errorf("no .app found in build/desktop/macos")
		}
		built = matches[0]
	}

	dest := filepath.Join(distPath("macos"), filepath.Base(built))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	os.RemoveAll(dest)
	if err := copyTree(built, dest); err != nil {
		return err
	}

	// Single source icon → .icns in the bundle (skip if none provided).
	if ic := findAppIcon(iconPath); ic != "" {
		if err := generateICNS(ic, dest); err != nil {
			return err
		}
	}

	tmp, err := os.MkdirTemp("", "irgo-macos-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	if identity != "" {
		entitlements := filepath.Join(tmp, "entitlements.plist")
		ent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>com.apple.security.cs.allow-jit</key><true/>
	<key>com.apple.security.cs.allow-unsigned-executable-memory</key><true/>
	<key>com.apple.security.cs.disable-library-validation</key><true/>
</dict>
</plist>
`
		if err := os.WriteFile(entitlements, []byte(ent), 0o644); err != nil {
			return err
		}
		fmt.Printf("Codesigning with %q...\n", identity)
		sign := exec.Command("codesign", "--force", "--deep", "--options", "runtime",
			"--entitlements", entitlements, "--sign", identity, dest)
		sign.Stdout = os.Stdout
		sign.Stderr = os.Stderr
		if err := sign.Run(); err != nil {
			return fmt.Errorf("codesign failed: %w", err)
		}
		verify := exec.Command("codesign", "--verify", "--deep", "--strict", dest)
		verify.Stdout = os.Stdout
		verify.Stderr = os.Stderr
		if err := verify.Run(); err != nil {
			return fmt.Errorf("codesign verify failed: %w", err)
		}
		fmt.Println("Codesign OK.")
	}

	if notarize {
		if appleID == "" || team == "" || password == "" {
			return fmt.Errorf("notarization requires --apple-id, --team and --password (an App Store Connect app-specific password) — or set [macos] in irgo.package.toml")
		}
		zipPath := filepath.Join(tmp, appName+".zip")
		if err := runCommand("ditto", "-c", "-k", "--keepParent", dest, zipPath); err != nil {
			return fmt.Errorf("ditto failed: %w", err)
		}
		fmt.Println("Submitting for notarization (can take a few minutes)...")
		submit := exec.Command("xcrun", "notarytool", "submit", zipPath, "--wait",
			"--apple-id", appleID, "--team-id", team, "--password", password)
		submit.Stdout = os.Stdout
		submit.Stderr = os.Stderr
		if err := submit.Run(); err != nil {
			return fmt.Errorf("notarytool submit failed: %w", err)
		}
		fmt.Println("Stapling notarization ticket...")
		staple := exec.Command("xcrun", "stapler", "staple", dest)
		staple.Stdout = os.Stdout
		staple.Stderr = os.Stderr
		if err := staple.Run(); err != nil {
			return fmt.Errorf("stapler failed: %w", err)
		}
	}

	if dmg {
		dmgOut := filepath.Join(distPath("macos"), appName+".dmg")
		if err := os.MkdirAll(filepath.Dir(dmgOut), 0o755); err != nil {
			return err
		}
		os.Remove(dmgOut)
		fmt.Printf("Creating DMG %s...\n", dmgOut)
		hdiutil := exec.Command("hdiutil", "create", "-volname", appName,
			"-srcfolder", dest, "-ov", "-format", "UDZO", dmgOut)
		hdiutil.Stdout = os.Stdout
		hdiutil.Stderr = os.Stderr
		if err := hdiutil.Run(); err != nil {
			return fmt.Errorf("hdiutil failed: %w", err)
		}
	}

	if out != "" {
		if filepath.Ext(out) == ".dmg" && dmg {
			if err := copyFile(filepath.Join(distPath("macos"), appName+".dmg"), out); err != nil {
				return err
			}
		} else {
			os.RemoveAll(out)
			if err := copyTree(dest, out); err != nil {
				return err
			}
		}
	}
	fmt.Printf("macOS package built: %s\n", dest)
	if dmg {
		fmt.Printf("  (dmg: %s)\n", filepath.Join(distPath("macos"), appName+".dmg"))
	}
	return nil
}

// findWindowsSDKTool locates a Windows SDK binary (MakeAppx.exe, signtool.exe)
// under the Windows Kits bin directory, preferring the newest version, then
// falls back to PATH.
func findWindowsSDKTool(tool string) (string, error) {
	patterns := []string{
		`C:\Program Files (x86)\Windows Kits\10\bin\*\x64\` + tool,
		`C:\Program Files (x86)\Windows Kits\10\bin\*\` + tool,
		`C:\Program Files\Windows Kits\10\bin\*\x64\` + tool,
	}
	for _, p := range patterns {
		matches, _ := filepath.Glob(p)
		if len(matches) > 0 {
			return matches[len(matches)-1], nil // string sort ≈ newest version
		}
	}
	if p, err := exec.LookPath(tool); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("%s not found — install the Windows SDK (Visual Studio Build Tools, 'Windows SDK' component)", tool)
}

// generateTestCert creates a self-signed code-signing cert and exports it as a
// PFX (pass "irgo") for test-signing MSIX packages.
func generateTestCert(dir string) (string, string, error) {
	ps := `
$cert = New-SelfSignedCertificate -Type CodeSigningCert -Subject "CN=irgo test" -CertStoreLocation Cert:\CurrentUser\My -KeyExportPolicy Exportable
$pass = ConvertTo-SecureString "irgo" -Force -AsPlainText
Export-PfxCertificate -Cert $cert -FilePath cert.pfx -Password $pass | Out-Null
`
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", ps)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("self-signed cert generation failed: %w", err)
	}
	return filepath.Join(dir, "cert.pfx"), "irgo", nil
}

// expandHome expands a leading "~" or "~/".
func expandHome(p string) string {
	if p == "~" {
		return homeDir()
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(homeDir(), p[2:])
	}
	return p
}

// copyTree recursively copies a directory tree.
func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target)
	})
}
