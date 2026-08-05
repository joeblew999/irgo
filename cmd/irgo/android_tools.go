package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// --- Android toolchain management -------------------------------------------
//
// irgo install-tools android / irgo uninstall-tools android / irgo doctor android
// provision and remove the Android SDK + NDK (+ optionally the emulator/AVD)
// across macOS/Linux/Windows — the same logic used to live in per-project mise
// tasks, now available to every irgo project and CI pipeline.
//
// Pinned, known-good toolchain: gomobile's bind defaults to API 16, and NDK
// r27+ dropped support below API 21, so NDK r26 is the floor that works with
// current gomobile unless -androidapi is passed explicitly (see stukennedy/irgo#9).

const (
	pinCmdlineTools = "11076708"
	pinNDK          = "26.3.11579264"
	pinBuildTools   = "35.0.0"
	pinPlatform34   = "platforms;android-34" // example app compileSdk
	pinPlatform35   = "platforms;android-35" // gomobile platform lookup fallback
	pinSysImg       = "system-images;android-35;google_apis"
	toolchainMarker = ".irgo-toolchain"
)

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}

func defaultAndroidHome() string {
	home := homeDir()
	switch runtime.GOOS {
	case "windows":
		if la := os.Getenv("LOCALAPPDATA"); la != "" {
			return filepath.Join(la, "Android", "Sdk")
		}
		return filepath.Join(home, "Android", "Sdk")
	case "linux":
		return filepath.Join(home, "Android", "Sdk")
	default: // darwin
		return filepath.Join(home, "Library", "Android", "sdk")
	}
}

func androidHome() string {
	if h := os.Getenv("ANDROID_HOME"); h != "" {
		return h
	}
	return defaultAndroidHome()
}

// --- JDK 17 ----------------------------------------------------------------

func isJava17(bin string) bool {
	if bin == "" {
		return false
	}
	if _, err := os.Stat(bin); err != nil {
		return false
	}
	out, err := exec.Command(bin, "-version").CombinedOutput()
	if err != nil {
		return false
	}
	// Matches e.g. `openjdk version "17.0.12"` / `java version "17.0.12"`.
	ok, _ := regexp.MatchString(`version "17\.`, string(out))
	return ok
}

// Managed JDK: irgo downloads a Temurin 17 into ~/.irgo/jdks so the toolchain
// is fully self-contained and cross-platform — no brew/apt/winget anywhere.
const managedJDKRel = ".irgo/jdks/temurin-17"

func managedJDKHome() string { return filepath.Join(homeDir(), managedJDKRel) }

// findManagedJDK returns the extracted JDK home under ~/.irgo/jdks/temurin-17
// (the Adoptium archive contains a single jdk-17.x.y/ dir — on macOS with a
// .app layout: jdk-17.x.y/Contents/Home/), or "" if absent.
func findManagedJDK() string {
	patterns := []string{
		filepath.Join(managedJDKHome(), "jdk-*", "Contents", "Home", "bin", "java"), // macOS .app layout
		filepath.Join(managedJDKHome(), "jdk-*", "bin", "java"),                     // linux/windows layout
	}
	for _, pat := range patterns {
		matches, _ := filepath.Glob(pat)
		if len(matches) == 0 {
			continue
		}
		// jdkHome is the ancestor of bin/java that contains lib/.
		for dir := filepath.Dir(matches[0]); dir != managedJDKHome(); dir = filepath.Dir(dir) {
			if fi, err := os.Stat(filepath.Join(dir, "lib")); err == nil && fi.IsDir() {
				return dir
			}
		}
	}
	return ""
}

// installManagedJDK downloads a Temurin JDK 17 (Adoptium) for the current
// platform/arch and extracts it into ~/.irgo/jdks/temurin-17. Returns its home.
func installManagedJDK() (string, error) {
	// Idempotent: reuse an already-extracted JDK 17.
	if p := findManagedJDK(); p != "" && isJava17(filepath.Join(p, "bin", "java")) {
		return p, nil
	}

	osName := map[string]string{"darwin": "mac", "linux": "linux", "windows": "windows"}[runtime.GOOS]
	if osName == "" {
		return "", fmt.Errorf("unsupported OS for JDK download: %s", runtime.GOOS)
	}
	arch := "x64"
	if runtime.GOARCH == "arm64" {
		arch = "aarch64"
	}
	url := fmt.Sprintf("https://api.adoptium.net/v3/binary/latest/17/ga/%s/%s/jdk/hotspot/normal/eclipse", osName, arch)

	dest := managedJDKHome()
	if err := os.RemoveAll(dest); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp("", "irgo-jdk")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	archive := filepath.Join(tmp, "jdk.tar.gz")
	if osName == "windows" {
		archive = filepath.Join(tmp, "jdk.zip")
	}
	fmt.Printf("Downloading Temurin JDK 17 (%s/%s)...\n", osName, arch)
	if err := downloadFile(url, archive); err != nil {
		return "", fmt.Errorf("JDK download failed: %w", err)
	}
	if osName == "windows" {
		if err := unzipTo(archive, dest); err != nil {
			return "", fmt.Errorf("JDK unzip failed: %w", err)
		}
	} else {
		if err := untarGz(archive, dest); err != nil {
			return "", fmt.Errorf("JDK untar failed: %w", err)
		}
	}
	home := findManagedJDK()
	if home == "" {
		return "", fmt.Errorf("JDK extracted but bin/java not found under %s", dest)
	}
	if !isJava17(filepath.Join(home, "bin", "java")) {
		return "", fmt.Errorf("downloaded JDK is not version 17 (%s)", home)
	}
	fmt.Printf("Installed managed JDK 17 at %s\n", home)
	return home, nil
}

// detectJDK17 returns (jdkHome, ok). jdkHome is empty when a usable java 17 is
// found on PATH without JAVA_HOME. Detection order: existing JAVA_HOME (only
// accepted when it is actually JDK 17 — Gradle 8.2/AGP 8.2 fails on JDK 21+,
// e.g. CI runners that default JAVA_HOME to 21), the managed ~/.irgo/jdks
// copy, macOS java_home, Linux /usr/lib/jvm, java on PATH. With install=true,
// a missing JDK is downloaded cross-platform into ~/.irgo/jdks.
func detectJDK17(install bool) (string, bool) {
	if jh := os.Getenv("JAVA_HOME"); jh != "" {
		if isJava17(filepath.Join(jh, "bin", "java")) {
			return jh, true
		}
	}
	if p := findManagedJDK(); p != "" {
		if isJava17(filepath.Join(p, "bin", "java")) {
			return p, true
		}
	}
	if runtime.GOOS == "darwin" {
		if out, err := exec.Command("/usr/libexec/java_home", "-v", "17").Output(); err == nil {
			if p := strings.TrimSpace(string(out)); isJava17(filepath.Join(p, "bin", "java")) {
				return p, true
			}
		}
	}
	if runtime.GOOS == "linux" {
		matches, _ := filepath.Glob("/usr/lib/jvm/*")
		for _, d := range matches {
			if isJava17(filepath.Join(d, "bin", "java")) {
				return d, true
			}
		}
	}
	if p, err := exec.LookPath("java"); err == nil && isJava17(p) {
		fmt.Println("Using java from PATH (JAVA_HOME not set).")
		return "", true
	}
	if install {
		p, err := installManagedJDK()
		if err != nil {
			fmt.Printf("  ! automatic JDK install failed: %v\n", err)
			return "", false
		}
		return p, true
	}
	return "", false
}

// --- sdkmanager / avdmanager ------------------------------------------------

func locateSdkmanager(androidHome string) string {
	if p, err := exec.LookPath("sdkmanager"); err == nil {
		return p
	}
	for _, c := range []string{
		filepath.Join(androidHome, "cmdline-tools", "latest", "bin", "sdkmanager"),
		filepath.Join(androidHome, "cmdline-tools", "latest", "bin", "sdkmanager.bat"),
		"/opt/homebrew/share/android-commandlinetools/cmdline-tools/latest/bin/sdkmanager",
		"/usr/local/share/android-commandlinetools/cmdline-tools/latest/bin/sdkmanager",
	} {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c
		}
	}
	return ""
}

func locateAvdmanager(androidHome string) string {
	if p, err := exec.LookPath("avdmanager"); err == nil {
		return p
	}
	for _, c := range []string{
		filepath.Join(androidHome, "cmdline-tools", "latest", "bin", "avdmanager"),
		filepath.Join(androidHome, "cmdline-tools", "latest", "bin", "avdmanager.bat"),
		"/opt/homebrew/share/android-commandlinetools/cmdline-tools/latest/bin/avdmanager",
		"/usr/local/share/android-commandlinetools/cmdline-tools/latest/bin/avdmanager",
	} {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c
		}
	}
	return ""
}

// --- download / extract (stdlib: no curl/unzip dependency) ------------------

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func unzipTo(src, destDir string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	base := filepath.Clean(destDir) + string(os.PathSeparator)
	for _, f := range r.File {
		target := filepath.Join(destDir, f.Name)
		if !strings.HasPrefix(target, base) {
			return fmt.Errorf("zip entry escapes target dir: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
		// Preserve the archive's mode bits (executable scripts/binaries).
		if err := os.Chmod(target, f.Mode()&os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

// untarGz extracts a .tar.gz into destDir (path-traversal safe).
func untarGz(src, destDir string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	base := filepath.Clean(destDir) + string(os.PathSeparator)
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, hdr.Name)
		if !strings.HasPrefix(target, base) {
			return fmt.Errorf("tar entry escapes target dir: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
			// Preserve the archive's mode bits (executable java/sdkmanager etc.).
			if err := os.Chmod(target, os.FileMode(hdr.Mode)&os.ModePerm); err != nil {
				return err
			}
		}
	}
	return nil
}

func installCmdlineTools(androidHome string) error {
	osName := map[string]string{"darwin": "mac", "linux": "linux", "windows": "win"}[runtime.GOOS]
	if osName == "" {
		return fmt.Errorf("unsupported OS for Android cmdline-tools: %s", runtime.GOOS)
	}
	url := fmt.Sprintf("https://dl.google.com/android/repository/commandlinetools-%s-%s_latest.zip", osName, pinCmdlineTools)
	tmp, err := os.MkdirTemp("", "irgo-cmdtools")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	zipPath := filepath.Join(tmp, "cmdtools.zip")
	fmt.Printf("Downloading Android cmdline-tools (%s, r%s)...\n", osName, pinCmdlineTools)
	if err := downloadFile(url, zipPath); err != nil {
		return fmt.Errorf("cmdline-tools download failed: %w", err)
	}
	if err := unzipTo(zipPath, tmp); err != nil {
		return fmt.Errorf("cmdline-tools unzip failed: %w", err)
	}
	latest := filepath.Join(androidHome, "cmdline-tools", "latest")
	if err := os.RemoveAll(latest); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(latest), 0o755); err != nil {
		return err
	}
	if err := os.Rename(filepath.Join(tmp, "cmdline-tools"), latest); err != nil {
		return fmt.Errorf("failed to place cmdline-tools: %w", err)
	}
	fmt.Printf("Installed cmdline-tools at %s\n", latest)
	return nil
}

// yesReader feeds an endless stream of "y" — used to auto-accept sdkmanager
// license prompts without a `yes` binary dependency.
type yesReader struct{}

func (yesReader) Read(p []byte) (int, error) {
	return copy(p, "y\n"), nil
}

func runSdkmanager(sdkm, androidHome string, args ...string) error {
	full := append([]string{"--sdk_root=" + androidHome}, args...)
	return runCommand(sdkm, full...)
}

// runCommandEnv runs a command with an extra environment variable, keeping
// stdout/stderr/stdin wired to the terminal.
func runCommandEnv(key, value, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), key+"="+value)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// applyJDK17ToEnv exports JAVA_HOME and prepends its bin to PATH in the current
// process, so every subprocess (sdkmanager, avdmanager, gomobile, gradle)
// inherits the JDK. brew openjdk@17 is keg-only, so this is required for it.
func applyJDK17ToEnv(jdkHome string) {
	if jdkHome == "" {
		return
	}
	_ = os.Setenv("JAVA_HOME", jdkHome)
	_ = os.Setenv("PATH", filepath.Join(jdkHome, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// applyNDKToEnv exports ANDROID_NDK_HOME in the current process so gomobile
// builds resolve the pinned NDK.
func applyNDKToEnv(ndkHome string) {
	_ = os.Setenv("ANDROID_NDK_HOME", ndkHome)
}

// applyBestJDKToEnv resolves a JDK 17 (JAVA_HOME, the managed ~/.irgo/jdks
// copy, macOS java_home, Linux JVM dirs, java on PATH) and applies it to the
// process env, so subprocesses (javac in gomobile bind, gradle) find java
// even when the caller never exported JAVA_HOME. No-op when none is found.
func applyBestJDKToEnv() {
	if jh, ok := detectJDK17(false); ok {
		applyJDK17ToEnv(jh)
	}
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// goBin returns the go executable: one on PATH, else $GOROOT/bin/go[.exe]
// (mise and standard installs set GOROOT — this survives git-bash on Windows
// stripping the Path var, where native exec.LookPath("go") fails). Falls back
// to "go" so callers surface a normal exec error.
func goBin() string {
	if p, err := exec.LookPath("go"); err == nil {
		return p
	}
	if gr := os.Getenv("GOROOT"); gr != "" {
		p := filepath.Join(gr, "bin", "go")
		if runtime.GOOS == "windows" {
			p += ".exe"
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "go"
}

// avdHomeDir returns where AVDs live — respects ANDROID_AVD_HOME (matching
// avdmanager/emulator), else ~/.android/avd.
func avdHomeDir() string {
	if h := os.Getenv("ANDROID_AVD_HOME"); h != "" {
		return h
	}
	return filepath.Join(homeDir(), ".android", "avd")
}

// templVersionFromGoMod returns the version of github.com/a-h/templ required
// by the project's go.mod, or "" when absent/unparseable. install-tools uses
// it to pin the templ generator to the templ library version — @latest drift
// breaks generated code when the two get out of sync.
func templVersionFromGoMod() string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return ""
	}
	// Handles both `require github.com/a-h/templ vX` and block form
	// `require ( ... github.com/a-h/templ vX ... )`.
	re := regexp.MustCompile(`(?m)^\s*(?:require\s+)?github\.com/a-h/templ\s+v([0-9][^\s]*)`)
	if m := re.FindSubmatch(data); m != nil {
		return string(m[1])
	}
	return ""
}

// adbBin returns the adb executable: one on PATH, else the SDK's
// platform-tools (installed by install-tools android). Falls back to "adb"
// so callers surface a normal exec error.
func adbBin() string {
	if p, err := exec.LookPath("adb"); err == nil {
		return p
	}
	p := filepath.Join(androidHome(), "platform-tools", "adb")
	if runtime.GOOS == "windows" {
		p += ".exe"
	}
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return "adb"
}

// haveAdb reports whether a usable adb is resolvable (PATH or the SDK).
func haveAdb() bool {
	if p, err := exec.LookPath("adb"); err == nil {
		_ = p
		return true
	}
	return adbBin() != "adb"
}

// adbRunning reports whether a device/emulator is already connected.
func adbRunning() bool {
	out, err := exec.Command(adbBin(), "get-state").CombinedOutput()
	return err == nil && strings.TrimSpace(string(out)) == "device"
}

// ensureEmulatorRunning boots the AVD (created by install-tools android
// --emulator) when no device is connected, then waits for boot to complete.
// Safe to call repeatedly: it is a no-op while an emulator/device is up.
// headless=true runs without a window (CI/scripting); otherwise headed unless
// there is no display available (non-macOS without DISPLAY).
func ensureEmulatorRunning(avdName string, headless bool) error {
	if adbRunning() {
		return nil
	}
	emu := filepath.Join(androidHome(), "emulator", "emulator")
	if runtime.GOOS == "windows" {
		emu += ".exe"
	}
	if !isDir(filepath.Dir(emu)) {
		return fmt.Errorf("emulator not found at %s (run 'irgo install-tools android --emulator' first)", emu)
	}
	// AVD must exist (created by installEmulator). Check the candidate homes —
	// avdmanager/emulator can disagree on the AVD home depending on env vars
	// (ANDROID_AVD_HOME vs ~/.android/avd vs the deprecated ANDROID_SDK_HOME).
	avdDir := findAvdDir(avdName)
	if avdDir == "" {
		var lines []string
		for _, base := range []string{avdHomeDir(), filepath.Join(homeDir(), ".android", "avd")} {
			entries, err := os.ReadDir(base)
			if err != nil {
				lines = append(lines, fmt.Sprintf("%s: <unreadable: %v>", base, err))
				continue
			}
			names := make([]string, 0, len(entries))
			for _, e := range entries {
				names = append(names, e.Name())
			}
			lines = append(lines, fmt.Sprintf("%s: [%s]", base, strings.Join(names, ", ")))
		}
		// Ask the emulator itself where it thinks AVDs live — the tool's own
		// view is the ground truth for where it will look at boot time.
		if out, err := exec.Command(emu, "-list-avds").CombinedOutput(); err == nil {
			lines = append(lines, fmt.Sprintf("emulator -list-avds: %q", strings.TrimSpace(string(out))))
		}
		return fmt.Errorf("AVD %q not found — ANDROID_AVD_HOME=%q, ANDROID_SDK_HOME=%q; checked:\n  %s\n(run 'irgo install-tools android --emulator --avd %s')", avdName, os.Getenv("ANDROID_AVD_HOME"), os.Getenv("ANDROID_SDK_HOME"), strings.Join(lines, "\n  "), avdName)
	}
	fmt.Printf("Using AVD at %s\n", avdDir)
	// Pin ANDROID_AVD_HOME so the emulator resolves the exact AVD we verified —
	// create and boot must agree even when the host env disagrees with us.
	_ = os.Setenv("ANDROID_AVD_HOME", filepath.Dir(avdDir))
	fmt.Printf("Booting emulator (AVD: %s)...\n", avdName)
	args := []string{"-avd", avdName, "-no-snapshot", "-no-audio", "-no-boot-anim"}
	if headless || (runtime.GOOS != "darwin" && os.Getenv("DISPLAY") == "") {
		args = append(args, "-no-window", "-gpu", "swiftshader_indirect")
		fmt.Println("Headless mode (no window).")
	}
	if runtime.GOOS == "linux" && !hasKVM() {
		// GitHub-hosted runners have no /dev/kvm — x86_64 refuses to start and
		// arm64 needs explicit software emulation (TCG). Slow boot, hence the
		// generous (configurable) deadline below.
		args = append(args, "-accel", "off")
		fmt.Println("No KVM available — using software emulation (boot will be slow).")
	}
	cmd := exec.Command(emu, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start emulator: %w", err)
	}
	// Intentionally not cmd.Wait() — the emulator runs until killed.
	fmt.Println("Waiting for emulator...")
	// adb wait-for-device blocks forever if the emulator never appears, so poll
	// adb devices against a deadline instead (mirrors the boot_completed poll).
	boot := bootTimeout()
	deadline := time.Now().Add(boot)
	for {
		out, err := exec.Command(adbBin(), "devices").CombinedOutput()
		if err == nil && strings.Contains(string(out), "emulator-") {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("emulator never appeared on adb within %s", boot)
		}
		time.Sleep(2 * time.Second)
	}
	deadline = time.Now().Add(boot)
	for time.Now().Before(deadline) {
		out, err := exec.Command(adbBin(), "shell", "getprop", "sys.boot_completed").CombinedOutput()
		if err == nil && strings.TrimSpace(string(out)) == "1" {
			fmt.Println("Emulator booted.")
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("emulator did not finish booting within %s", boot)
}

// hasKVM reports whether hardware acceleration is available on Linux (the
// Android emulator refuses to run x86_64 without it; arm64 needs -accel off).
func hasKVM() bool {
	if runtime.GOOS != "linux" {
		return true // darwin uses Hypervisor.framework; windows uses WHPX
	}
	fi, err := os.Stat("/dev/kvm")
	return err == nil && fi.Mode()&os.ModeDevice != 0
}

// bootTimeout returns the emulator boot deadline, from IRGO_BOOT_TIMEOUT_MIN
// (default 5). Software-emulated runners (no KVM) boot far more slowly.
func bootTimeout() time.Duration {
	m := 5
	if v := os.Getenv("IRGO_BOOT_TIMEOUT_MIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			m = n
		}
	}
	return time.Duration(m) * time.Minute
}

func acceptLicenses(sdkm, androidHome string) {
	cmd := exec.Command(sdkm, "--sdk_root="+androidHome, "--licenses")
	cmd.Stdin = yesReader{}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	_ = cmd.Run()
}

// --- install ----------------------------------------------------------------

// ensureAndroidToolchain provisions anything the Android toolchain is missing
// (JDK 17, cmdline-tools, platform-tools, platforms, build-tools, NDK, and —
// with withEmulator — the emulator + AVD). Idempotent: only installs what is
// absent. Called by install-tools android and, self-provisioning, by
// build android / run android so devs and CI never need a separate setup step.
func ensureAndroidToolchain(withEmulator bool, avdName string) error {
	sdk := androidHome()
	fmt.Printf("Android SDK home: %s\n", sdk)

	// Export ANDROID_HOME for every subprocess we spawn (gradle, sdkmanager,
	// aapt2...) — callers often don't have it set, and gradle refuses to build
	// without a resolvable SDK location.
	_ = os.Setenv("ANDROID_HOME", sdk)

	jdkHome, ok := detectJDK17(true)
	if !ok {
		return fmt.Errorf("No JDK 17 found and the automatic cross-platform install failed — set JAVA_HOME to a JDK 17 and re-run")
	}
	if jdkHome != "" {
		fmt.Printf("Using JDK: %s\n", jdkHome)
		applyJDK17ToEnv(jdkHome)
	}

	if err := os.MkdirAll(sdk, 0o755); err != nil {
		return err
	}

	sdkm := locateSdkmanager(sdk)
	if sdkm == "" {
		if err := installCmdlineTools(sdk); err != nil {
			return err
		}
		sdkm = locateSdkmanager(sdk)
	}
	if sdkm == "" {
		return fmt.Errorf("failed to locate sdkmanager. Install Android cmdline-tools manually or point ANDROID_HOME at an SDK that has them")
	}
	fmt.Printf("Using sdkmanager: %s\n", sdkm)

	// Fast path: when the pinned NDK + platform-tools already exist the
	// components are installed — skip the sdkmanager round-trip entirely.
	ndk := filepath.Join(sdk, "ndk", pinNDK)
	if !isDir(ndk) || !isDir(filepath.Join(sdk, "platform-tools")) {
		fmt.Println("Accepting Android SDK licenses...")
		acceptLicenses(sdkm, sdk)
		fmt.Println("Installing Android SDK components (platform-tools, platforms, build-tools, NDK)...")
		if err := runSdkmanager(sdkm, sdk, "platform-tools", pinPlatform34, pinPlatform35, "build-tools;"+pinBuildTools, "ndk;"+pinNDK); err != nil {
			return fmt.Errorf("sdkmanager install failed: %w", err)
		}
	}
	if !isDir(ndk) {
		return fmt.Errorf("NDK not found at %s — Android build will fail", ndk)
	}
	applyNDKToEnv(ndk)

	// Marker so `uninstall-tools android` knows this SDK was provisioned here
	// and never deletes an SDK irgo did not install.
	_ = os.WriteFile(filepath.Join(sdk, toolchainMarker), []byte("irgo "+version+"\n"), 0o644)

	if _, err := exec.LookPath("gomobile"); err != nil {
		fmt.Println("Installing gomobile...")
		if err := runCommand(goBin(), "install", "golang.org/x/mobile/cmd/gomobile@latest"); err != nil {
			fmt.Printf("  ! gomobile install failed: %v\n", err)
		}
		// `go install` lands in GOBIN (default $GOPATH/bin), which may not be
		// on the caller's PATH — prepend it so gomobile resolves below.
		_ = os.Setenv("PATH", gobinDir()+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	if err := runCommand("gomobile", "init"); err != nil {
		fmt.Printf("  ! gomobile init failed: %v\n", err)
	}

	if withEmulator {
		if err := installEmulator(sdk, avdName, sdkm); err != nil {
			return err
		}
	}
	return nil
}

func installAndroidTools(withEmulator bool, avdName string) error {
	if err := ensureAndroidToolchain(withEmulator, avdName); err != nil {
		return err
	}
	sdk := androidHome()
	ndk := filepath.Join(sdk, "ndk", pinNDK)
	fmt.Printf("\nAndroid toolchain ready.\n")
	fmt.Printf("  ANDROID_HOME=%s\n", sdk)
	fmt.Printf("  ANDROID_NDK_HOME=%s\n", ndk)
	fmt.Println("Add to your shell (or let your env manager export them):")
	fmt.Printf("  export ANDROID_HOME=%s\n", sdk)
	fmt.Printf("  export ANDROID_NDK_HOME=%s\n", ndk)
	return nil
}

func installEmulator(sdk, avdName, sdkm string) error {
	abi := "x86_64"
	if runtime.GOARCH == "arm64" {
		abi = "arm64-v8a" // Apple Silicon, ARM64 CI runners (e.g. GitHub ubuntu-*-arm)
	}
	sysImg := pinSysImg + ";" + abi
	fmt.Printf("Installing emulator + system image (%s) (large download)...\n", sysImg)
	if err := runSdkmanager(sdkm, sdk, "emulator", "platform-tools", sysImg); err != nil {
		return err
	}
	avdmgr := locateAvdmanager(sdk)
	if avdmgr == "" {
		return fmt.Errorf("avdmanager not found after cmdline-tools install")
	}

	// Pin the AVD home explicitly BEFORE avdmanager runs. avdmanager and the
	// emulator can resolve the AVD home differently (ANDROID_AVD_HOME vs
	// ~/.android/avd vs XDG/ANDROID_USER_HOME on Linux CI), which made "create"
	// and "boot" disagree. Forcing ANDROID_AVD_HOME makes both use the same
	// directory deterministically, cross-platform.
	avdHome := avdHomeDir()
	if err := os.MkdirAll(avdHome, 0o755); err != nil {
		return fmt.Errorf("creating AVD home %s: %w", avdHome, err)
	}
	if err := os.Setenv("ANDROID_AVD_HOME", avdHome); err != nil {
		return fmt.Errorf("setting ANDROID_AVD_HOME: %w", err)
	}
	fmt.Printf("AVD home: %s\n", avdHome)

	if avdExists(avdmgr, avdName) {
		fmt.Printf("AVD %q already exists.\n", avdName)
		return nil
	}
	fmt.Printf("Creating AVD %q (%s)...\n", avdName, sysImg)
	cmd := exec.Command(avdmgr, "create", "avd", "-n", avdName, "-k", sysImg, "--force")
	cmd.Stdin = strings.NewReader("no\n") // decline a custom hardware profile
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("avdmanager create avd failed: %w", err)
	}
	// avdmanager may write <build> placeholders for the name/id; normalize them.
	cfg := filepath.Join(avdHome, avdName+".avd", "config.ini")
	if data, err := os.ReadFile(cfg); err == nil {
		s := strings.ReplaceAll(string(data), "avd.id=<build>", "avd.id="+avdName)
		s = strings.ReplaceAll(s, "avd.name=<build>", "avd.name="+avdName)
		_ = os.WriteFile(cfg, []byte(s), 0o644)
	}

	// Verify the AVD actually landed where we will look for it (create and
	// boot must agree). Fall back to the emulator's own list when the
	// filesystem check disagrees, and fail loudly otherwise — a silently
	// missing AVD is exactly the failure mode that bit CI.
	avdDir := filepath.Join(avdHome, avdName+".avd")
	if !isDir(avdDir) {
		if found := findAvdDir(avdName); found != "" {
			avdDir = found
			// Tell the emulator where we found it so boot resolves the same AVD.
			_ = os.Setenv("ANDROID_AVD_HOME", filepath.Dir(avdDir))
		} else {
			return fmt.Errorf("avdmanager reported success but AVD %q was not created (expected %s)", avdName, avdDir)
		}
	}
	fmt.Printf("AVD %q created at %s\n", avdName, avdDir)
	return nil
}

// findAvdDir locates an existing AVD directory by name, checking the known
// homes and finally the emulator's own view (emulator -list-avds).
func findAvdDir(avdName string) string {
	for _, base := range []string{avdHomeDir(), filepath.Join(homeDir(), ".android", "avd")} {
		if d := filepath.Join(base, avdName+".avd"); isDir(d) {
			return d
		}
	}
	// Ask the emulator where it thinks the AVD lives.
	emu := filepath.Join(androidHome(), "emulator", "emulator")
	if runtime.GOOS == "windows" {
		emu += ".exe"
	}
	if out, err := exec.Command(emu, "-list-avds").CombinedOutput(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.TrimSpace(line) == avdName {
				// emulator -list-avds prints bare names; resolve the dir by
				// scanning the standard homes.
				for _, base := range []string{avdHomeDir(), filepath.Join(homeDir(), ".android", "avd")} {
					if d := filepath.Join(base, avdName+".avd"); isDir(d) {
						return d
					}
				}
			}
		}
	}
	return ""
}

func avdExists(avdmgr, avdName string) bool {
	out, err := exec.Command(avdmgr, "list", "avd").CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "Name: "+avdName)
}

// --- uninstall ---------------------------------------------------------------

func gobinDir() string {
	if out, err := exec.Command(goBin(), "env", "GOBIN").Output(); err == nil {
		if p := strings.TrimSpace(string(out)); p != "" {
			return p
		}
	}
	out, err := exec.Command(goBin(), "env", "GOPATH").Output()
	gp := strings.TrimSpace(string(out))
	if err != nil || gp == "" {
		gp = filepath.Join(homeDir(), "go")
	}
	return filepath.Join(gp, "bin")
}

func uninstallAndroidTools(removeJDK bool) error {
	home := homeDir()
	sdk := androidHome()

	// gomobile/gobind may live in the resolved GOBIN and/or the default
	// $GOPATH/bin (they differ when GOBIN was empty at install time) — check both.
	binDirs := []string{gobinDir()}
	if out, err := exec.Command(goBin(), "env", "GOPATH").Output(); err == nil {
		if gp := strings.TrimSpace(string(out)); gp != "" {
			if p := filepath.Join(gp, "bin"); p != binDirs[0] {
				binDirs = append(binDirs, p)
			}
		}
	}
	for _, dir := range binDirs {
		for _, t := range []string{"gomobile", "gobind"} {
			name := t
			if runtime.GOOS == "windows" {
				name += ".exe"
			}
			p := filepath.Join(dir, name)
			if _, err := os.Stat(p); err == nil {
				fmt.Printf("Removing %s\n", p)
				_ = os.Remove(p)
			}
		}
	}
	// Temp x/mobile clone + local go.work created by mobile builds.
	_ = os.RemoveAll(filepath.Join(os.TempDir(), "golang-mobile"))
	_ = os.Remove("go.work")
	_ = os.Remove("go.work.sum")

	if runtime.GOOS == "darwin" {
		_ = os.Remove(filepath.Join(home, "Library", "Preferences", "com.android.Emulator.plist"))
		_ = os.RemoveAll(filepath.Join(home, "Library", "Caches", "TemporaryItems", "avd"))
	}
	if removeJDK {
		fmt.Println("Removing managed JDK (~/.irgo/jdks)...")
		_ = os.RemoveAll(filepath.Join(home, ".irgo", "jdks"))
		_ = os.Remove(filepath.Join(home, ".irgo")) // empty parent
	}

	fmt.Println("Removing ~/.android (AVDs, adb keys, emulator state)...")
	_ = os.RemoveAll(filepath.Join(home, ".android"))
	fmt.Println("Removing ~/.gradle (caches from Android builds)...")
	_ = os.RemoveAll(filepath.Join(home, ".gradle"))

	// Only delete an SDK irgo provisioned (marker present); never a dev's own.
	if _, err := os.Stat(filepath.Join(sdk, toolchainMarker)); err == nil {
		fmt.Printf("Removing SDK directory (SDK, NDK, emulator, system images): %s\n", sdk)
		_ = os.RemoveAll(sdk)
		if runtime.GOOS == "darwin" {
			_ = os.Remove(filepath.Join(home, "Library", "Android")) // empty parent
		}
	} else {
		fmt.Printf("SDK at %s was not installed by irgo — leaving it in place.\n", sdk)
	}
	fmt.Println("Android tooling removed.")
	return nil
}

// --- doctor ------------------------------------------------------------------

func doctorAndroid() error {
	fail := false
	sdk := androidHome()
	ndk := filepath.Join(sdk, "ndk", pinNDK)
	emulator := filepath.Join(sdk, "emulator", "emulator")

	fmt.Println("Android toolchain doctor:")
	fmt.Printf("  ANDROID_HOME: %s\n", sdk)
	if fi, err := os.Stat(sdk); err == nil && fi.IsDir() {
		fmt.Println("    SDK dir: OK")
	} else {
		fmt.Println("    SDK dir: MISSING (run 'irgo install-tools android')")
		fail = true
	}

	if jdkHome, ok := detectJDK17(false); ok {
		if jdkHome != "" {
			fmt.Printf("  JDK 17: OK (%s)\n", jdkHome)
			applyJDK17ToEnv(jdkHome)
		} else {
			fmt.Println("  JDK 17: OK (java on PATH)")
		}
	} else {
		fmt.Println("  JDK 17: MISSING (run 'irgo install-tools android' or set JAVA_HOME)")
		fail = true
	}

	if sdkm := locateSdkmanager(sdk); sdkm != "" {
		fmt.Printf("  sdkmanager: OK (%s)\n", sdkm)
	} else {
		fmt.Println("  sdkmanager: MISSING")
		fail = true
	}

	if fi, err := os.Stat(ndk); err == nil && fi.IsDir() {
		fmt.Printf("  NDK (%s): OK\n", pinNDK)
	} else {
		fmt.Printf("  NDK (%s): MISSING\n", pinNDK)
		fail = true
	}

	if _, err := os.Stat(emulator); err == nil {
		fmt.Println("  emulator: OK")
	} else {
		fmt.Println("  emulator: not installed (use --emulator)")
	}

	if avdmgr := locateAvdmanager(sdk); avdmgr != "" {
		out, _ := exec.Command(avdmgr, "list", "avd").CombinedOutput()
		var names []string
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "Name: ") {
				names = append(names, strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "Name: ")))
			}
		}
		if len(names) > 0 {
			fmt.Printf("  AVDs: %s\n", strings.Join(names, ", "))
		} else {
			fmt.Println("  AVDs: none (run 'irgo install-tools android --emulator')")
		}
	}

	if fail {
		return fmt.Errorf("Android toolchain incomplete — fix the items marked MISSING")
	}
	fmt.Println("Android toolchain OK.")
	return nil
}
