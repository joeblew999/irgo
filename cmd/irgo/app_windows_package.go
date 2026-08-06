package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ---------------------------------------------------------------------------
// Windows — MSIX for the Microsoft Store (Windows-only)
// ---------------------------------------------------------------------------

// packageWindows builds a (test-signed) MSIX on a Windows host: build the exe,
// lay out the package (AppxManifest + Assets + exe + static), pack with
// MakeAppx, sign with signtool (self-signed test cert unless --cert given).
// MSIX tooling only exists on Windows, so this is gated to Windows.
func packageWindows(publisher, version, iconPath, cert, certPass, out string) error {
	if err := preparePackage("windows"); err != nil {
		return err
	}
	if err := ensureStoreConfig("windows"); err != nil {
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
	// IRGO_* env vars (CI secrets) beat the toml but lose to flags.
	if publisher == "" {
		publisher = os.Getenv("IRGO_WINDOWS_PUBLISHER")
	}
	if cert == "" {
		cert = os.Getenv("IRGO_WINDOWS_CERT")
	}
	if certPass == "" {
		certPass = os.Getenv("IRGO_WINDOWS_CERT_PASS")
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
	runHint(
		"Add-AppxPackage " + out + "   (signed, or with a trusted test cert)",
	)
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
$cert = New-SelfSignedCertificate -Type CodeSigningCert -Subject "CN=irgo project test" -CertStoreLocation Cert:\CurrentUser\My -KeyExportPolicy Exportable
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
