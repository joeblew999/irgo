package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ---------------------------------------------------------------------------
// macOS — signed + notarized .app (and optional .dmg) for distribution
// ---------------------------------------------------------------------------

// packageMacOS packages the desktop .app for distribution: copies it to
// dist/macos/, codesigns it (--identity / [macos] identity, hardened runtime
// entitlements), optionally notarizes + staples (--notarize + Apple
// credentials), and optionally wraps it in a .dmg (--dmg). macOS-only.
func packageMacOS(identity string, notarize bool, appleID, team, password string, dmg bool, iconPath, out string) error {
	if err := preparePackage("macos"); err != nil {
		return err
	}
	if err := ensureStoreConfig("macos"); err != nil {
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
	// IRGO_* env vars (CI secrets) beat the toml but lose to flags.
	if identity == "" {
		identity = os.Getenv("IRGO_MACOS_IDENTITY")
	}
	if !notarize {
		notarize = os.Getenv("IRGO_MACOS_NOTARIZE") != ""
	}
	if appleID == "" {
		appleID = os.Getenv("IRGO_APPLE_ID")
	}
	if team == "" {
		team = os.Getenv("IRGO_IOS_TEAM")
	}
	if password == "" {
		password = os.Getenv("IRGO_APPLE_PASSWORD")
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
		// An irgo app runs an embedded HTTP server and points its own WebView
		// at it, so under the hardened runtime it needs to both listen and
		// connect. Without these two the app notarizes and ships, then fails
		// at runtime with nothing explaining why.
		ent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<!-- WebView JavaScript -->
	<key>com.apple.security.cs.allow-jit</key><true/>
	<key>com.apple.security.cs.allow-unsigned-executable-memory</key><true/>
	<key>com.apple.security.cs.disable-library-validation</key><true/>

	<!-- Embedded HTTP server, and the WebView connecting to it -->
	<key>com.apple.security.network.server</key><true/>
	<key>com.apple.security.network.client</key><true/>
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
	runHint("open " + dest)
	if dmg {
		fmt.Printf("  (dmg: %s)\n", filepath.Join(distPath("macos"), appName+".dmg"))
	}
	return nil
}
