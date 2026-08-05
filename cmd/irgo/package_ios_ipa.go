package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

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
	if err := ensureStoreConfig("ios"); err != nil {
		return err
	}
	if err := requireMacOS("iOS packaging"); err != nil {
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
