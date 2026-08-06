package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Android — signed AAB for the Play Store (cross-platform)
// ---------------------------------------------------------------------------

// packageAndroid produces a signed AAB via the Example project's gradle
// bundleRelease, feeding signing + version through -Pirgo.* properties (the
// template's app/build.gradle.kts reads them). Runs on any OS (gradle + JDK +
// Android SDK are cross-platform). Defaults to the debug keystore; a real
// keystore is required for Play Store submission.
func packageAndroid(keystore, keystorePass, keyAlias, keyPass, version, iconPath, out string) error {
	if err := preparePackage("android"); err != nil {
		return err
	}
	if err := ensureStoreConfig("android"); err != nil {
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
	// IRGO_* env vars (CI secrets) beat the toml but lose to flags.
	if keystore == "" {
		keystore = os.Getenv("IRGO_ANDROID_KEYSTORE")
	}
	if keystorePass == "" {
		keystorePass = os.Getenv("IRGO_ANDROID_KEYSTORE_PASS")
	}
	if keyAlias == "" {
		keyAlias = os.Getenv("IRGO_ANDROID_KEY_ALIAS")
	}
	if keyPass == "" {
		keyPass = os.Getenv("IRGO_ANDROID_KEY_PASS")
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
	runHint(
		"an .aab is for Play, not direct install — to try it on a device:",
		"bundletool build-apks --bundle="+out+" --output=app.apks --local-testing",
		"bundletool install-apks --apks=app.apks",
		"or just: irgo run android",
	)
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
		return fmt.Errorf("keytool not found (set JAVA_HOME or run 'irgo tools install android'): %w", err)
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
