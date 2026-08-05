package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// packageIOS produces a signed IPA ready for App Store Connect:
//
//  1. build the Irgo.xcframework (the Example app links against it)
//  2. xcodebuild -archive (Release, generic/platform=iOS)
//  3. xcodebuild -exportArchive with an ExportOptions.plist (app-store)
//
// Automatic signing (-allowProvisioningUpdates) needs a Development Team
// (--team or IRGO_IOS_TEAM); the signing certificate and provisioning
// profile come from the keychain, exactly like any Xcode build. This is the
// gomobile/xcodebuild equivalent of gogio's iosbuild.go, but rides the
// project's real Xcode scheme instead of hand-assembling the .app.
func packageIOS(team, exportMethod, out string) error {
	if err := checkTool("xcodebuild", "Install Xcode from the App Store"); err != nil {
		return err
	}
	if err := checkTool("xcrun", "Install Xcode Command Line Tools: xcode-select --install"); err != nil {
		return err
	}
	if exportMethod == "" {
		exportMethod = "app-store"
	}
	if team == "" {
		team = os.Getenv("IRGO_IOS_TEAM")
	}
	if team == "" {
		return fmt.Errorf("a Development Team is required for automatic signing\n" +
			"  pass --team <TEAM_ID> or set IRGO_IOS_TEAM (find it in the Apple Developer portal)")
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

	archivePath := filepath.Join("build", "ios", "Example.xcarchive")
	os.RemoveAll(archivePath)

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

	// ExportOptions.plist describing the export (method + team).
	optsPath := filepath.Join("build", "ios", "ExportOptions.plist")
	if err := os.MkdirAll(filepath.Dir(optsPath), 0o755); err != nil {
		return err
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

	exportDir := filepath.Join("build", "ios", "export")
	os.RemoveAll(exportDir)
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

	// Locate the .ipa and place it at the requested (or default) path.
	ipas, err := filepath.Glob(filepath.Join(exportDir, "*.ipa"))
	if err != nil || len(ipas) == 0 {
		return fmt.Errorf("no .ipa produced in %s", exportDir)
	}
	if out == "" {
		out = filepath.Join("build", "ios", "Example.ipa")
	}
	if err := copyFile(out, ipas[0]); err != nil {
		return fmt.Errorf("copying ipa: %w", err)
	}
	fmt.Printf("iOS package built: %s\n", out)
	return nil
}
