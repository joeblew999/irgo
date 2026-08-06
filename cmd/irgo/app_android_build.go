// Android: AAR bind and launching the app on an emulator or device.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func buildAndroid(modulePath string) error {
	fmt.Println("Building Android AAR...")

	outPath := "build/android/irgo.aar"
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}

	// Remove existing AAR
	os.Remove(outPath)

	// Self-provision the toolchain (JDK/SDK/NDK/gomobile) if anything is
	// missing — devs and CI never need a separate setup step.
	if err := ensureAndroidToolchain(false, "irgo"); err != nil {
		return fmt.Errorf("Android toolchain setup failed: %w", err)
	}

	// Ensure the canonical android/Example app exists so the AAR has a
	// consumer out of the box (scaffolded from embedded templates).
	if err := scaffoldExamples(); err != nil {
		return fmt.Errorf("Android example scaffold failed: %w", err)
	}

	// Ensure go.work and gomobile setup
	if err := ensureMobileBuildSetup(); err != nil {
		return fmt.Errorf("mobile build setup failed: %w", err)
	}

	mobilePackage := modulePath + "/mobile"
	// Pin the Android API level explicitly: gomobile defaults to API 16,
	// which modern NDKs (r26/r27) reject with "unsupported API version 16".
	// API 21 is the floor every current NDK supports.
	if err := runGomobileCommand("bind", "-target", "android", "-androidapi", "21", "-o", outPath, mobilePackage); err != nil {
		return fmt.Errorf("gomobile bind failed: %w", err)
	}
	writeArtifactStamp("build/android")

	fmt.Printf("Android AAR built: %s\n", outPath)

	// Copy to Example project if it exists
	exampleLibsPath := "android/Example/app/libs/irgo.aar"
	if _, err := os.Stat("android/Example"); err == nil {
		os.MkdirAll(filepath.Dir(exampleLibsPath), 0755)
		if err := copyFile(outPath, exampleLibsPath); err != nil {
			fmt.Printf("Warning: could not copy to example project: %v\n", err)
		} else {
			fmt.Printf("Copied to: %s\n", exampleLibsPath)
		}
	}

	return nil
}

func runAndroid(devMode bool, avdName string, headless bool) error {
	// Self-provision the Android toolchain (JDK/SDK/NDK/gomobile, plus the
	// emulator + AVD when no device is connected) — devs and CI never need a
	// separate setup step. Idempotent: only installs what is missing.
	if err := ensureAndroidToolchain(!adbRunning(), avdName); err != nil {
		return err
	}

	// Ensure the canonical android/Example app exists (scaffolded from the
	// embedded templates when missing — devs and CI never hand-copy it).
	if err := scaffoldExamples(); err != nil {
		return fmt.Errorf("Android example scaffold failed: %w", err)
	}

	// Check if android/Example project exists
	androidProjectPath := "android/Example"
	if _, err := os.Stat(androidProjectPath); os.IsNotExist(err) {
		return fmt.Errorf("Android project not found at %s", androidProjectPath)
	}

	// App icon: single source icon → launcher mipmaps (if present), so the
	// installed APK shows the project's icon.
	if ic := findAppIcon(""); ic != "" {
		_ = generateAndroidIcons(ic, filepath.Join(androidProjectPath, "app", "src", "main", "res"))
	}

	// Dev server URL as seen from the emulator (10.0.2.2 is the host loopback)
	devServerURL := "http://10.0.2.2:8080"
	var devServerCmd *exec.Cmd

	if devMode {
		fmt.Println("Running in DEV MODE with hot reload...")
		fmt.Println()

		// Check for required dev tools
		if err := ensureGoTool("air"); err != nil {
			return err
		}

		// Dev mode serves the app from the dev server, so a fresh gomobile
		// build isn't needed. The Gradle project still links against
		// app/libs/irgo.aar, so build it only when missing or built by a
		// different irgo version (stale bridge API).
		aarPath := filepath.Join(androidProjectPath, "app/libs/irgo.aar")
		if !artifactUpToDate(aarPath, "build/android") {
			modulePath, err := getModulePath()
			if err != nil {
				return fmt.Errorf("could not determine module path: %w", err)
			}

			fmt.Println("Building Android AAR (missing or built by another irgo version)...")
			if err := buildAndroid(modulePath); err != nil {
				return err
			}
		} else {
			fmt.Printf("Using existing %s (delete it to force a rebuild)\n", aarPath)
		}

		// Start dev server in background
		fmt.Println("Starting dev server at http://localhost:8080...")
		devServerCmd = exec.Command("air")
		devServerCmd.Stdout = os.Stdout
		devServerCmd.Stderr = os.Stderr
		if err := devServerCmd.Start(); err != nil {
			return fmt.Errorf("failed to start dev server: %w", err)
		}

		// Give server time to start
		fmt.Println("Waiting for dev server to start...")
		exec.Command("sleep", "3").Run()
	} else {
		// Production mode: build the AAR
		modulePath, err := getModulePath()
		if err != nil {
			return fmt.Errorf("could not determine module path: %w", err)
		}

		fmt.Println("Building Android AAR...")
		if err := buildAndroid(modulePath); err != nil {
			return err
		}
	}

	killDevServer := func() {
		if devServerCmd != nil {
			devServerCmd.Process.Kill()
		}
	}

	// Build with Gradle
	gradlew := filepath.Join(androidProjectPath, "gradlew")
	if _, err := os.Stat(gradlew); os.IsNotExist(err) {
		killDevServer()
		return fmt.Errorf("gradlew not found in %s", androidProjectPath)
	}

	fmt.Println("Building Android app...")
	// Run ./gradlew relative to cmd.Dir: exec.Command with a path relative to
	// the ORIGINAL cwd would not resolve once cmd.Dir switches the working
	// directory to androidProjectPath.
	cmd := exec.Command("./gradlew", "assembleDebug")
	cmd.Dir = androidProjectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		killDevServer()
		return fmt.Errorf("gradle build failed: %w", err)
	}

	// Find the built APK
	apkPath := filepath.Join(androidProjectPath, "app/build/outputs/apk/debug/app-debug.apk")
	if _, err := os.Stat(apkPath); os.IsNotExist(err) {
		killDevServer()
		return fmt.Errorf("built APK not found at %s", apkPath)
	}

	// Boot the emulator if none is running, then install.
	if err := ensureEmulatorRunning(avdName, headless); err != nil {
		killDevServer()
		return err
	}
	fmt.Println("Installing on Android device/emulator...")
	if err := runCommand(adbBin(), "install", "-r", apkPath); err != nil {
		killDevServer()
		return fmt.Errorf("failed to install APK (is an emulator running?): %w", err)
	}

	// Launch app
	fmt.Println("Launching app...")
	packageName := "com.irgo.example"
	activityName := ".MainActivity"
	launchArgs := []string{"shell", "am", "start", "-n", packageName + "/" + packageName + activityName}
	if devMode {
		// IrgoActivity reads the irgoDevServer extra and loads that URL
		// instead of the embedded bridge.
		launchArgs = append(launchArgs, "-e", "irgoDevServer", devServerURL)
	}
	if err := runCommand(adbBin(), launchArgs...); err != nil {
		killDevServer()
		return fmt.Errorf("failed to launch app: %w", err)
	}

	if devMode {
		fmt.Println()
		fmt.Println("===========================================")
		fmt.Println("Android app running in DEV MODE with hot reload!")
		fmt.Printf("Dev server: %s (localhost:8080 on this machine)\n", devServerURL)
		fmt.Println("Edit your Go code and see changes instantly.")
		fmt.Println("Press Ctrl+C to stop.")
		fmt.Println("===========================================")
		fmt.Println()

		// Wait for dev server to exit (user presses Ctrl+C)
		devServerCmd.Wait()
	} else {
		fmt.Println("\nApp running on Android!")
	}

	return nil
}

// Helper functions
