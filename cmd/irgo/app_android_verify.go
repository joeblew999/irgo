// Whether the thing just built can actually be uploaded.
//
// A bundle that builds is not a bundle Play accepts, and the gap between those
// two is where a release goes wrong — at upload, or worse, on a user's device.
// Three things decide it, none of them visible from a successful build:
//
//   - 16 KB page alignment. Android 15 and later run on devices with 16 KB
//     memory pages, and a native library aligned for 4 KB fails to load there.
//     NDK r28 made this the default and r29 requires it, so a project on an
//     older NDK produces a bundle that installs fine and crashes on launch for
//     a growing share of devices. Nothing in the build says so.
//   - A signature. An unsigned bundle is rejected at upload, after the wait.
//   - targetSdk. Play refuses new apps and updates below its floor, which is
//     36 from August 2026.
//
// Checked here rather than left to aapt2 and zipalign, because a developer who
// has to remember three Android SDK tools to know whether their release is
// valid will find out from the Play Console instead.
package main

import (
	"archive/zip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// playTargetSdkFloor is what Play requires of new apps and updates.
const playTargetSdkFloor = 36

// verifyAndroidArtifact reports what would stop this being uploaded.
//
// Warnings rather than errors: a debug build is a legitimate thing to make,
// and irgo does not know whether this one is heading for the store. What it
// can do is say plainly what is wrong before the Play Console does.
func verifyAndroidArtifact(path string) {
	fmt.Println()
	fmt.Println("Checks:")

	if ok, detail := nativeLibsAre16KAligned(path); ok {
		fmt.Println("  16 KB alignment  OK")
	} else {
		fmt.Printf("  16 KB alignment  FAILS — %s\n", detail)
		fmt.Println("       Android 15+ devices use 16 KB memory pages, and a library")
		fmt.Println("       aligned for 4 KB will not load on them. The app installs and")
		fmt.Println("       then crashes on launch. NDK r28+ aligns by default:")
		fmt.Println("         irgo tools install android")
	}

	if signed, detail := artifactIsSigned(path); signed {
		fmt.Println("  Signature        OK")
	} else {
		fmt.Printf("  Signature        MISSING — %s\n", detail)
		fmt.Println("       Play rejects unsigned uploads. See: irgo app package setup")
	}

	if debug, ks := signedWithDebugKeystore(); debug {
		fmt.Println("  Signing key      DEBUG — Play will reject this")
		fmt.Printf("       %s is the throwaway key every Android SDK ships.\n", ks)
		fmt.Println("       A release needs your own, and it can never be changed after")
		fmt.Println("       the first upload — losing it means a new listing:")
		fmt.Println("         irgo app package setup")
	} else {
		fmt.Println("  Signing key      your own")
	}

	if v := artifactTargetSdk(path); v == 0 {
		fmt.Println("  Target SDK       could not be read")
	} else if v < playTargetSdkFloor {
		fmt.Printf("  Target SDK       %d — Play requires %d or higher\n", v, playTargetSdkFloor)
		fmt.Printf("         irgo project config android.target_sdk %d\n", playTargetSdkFloor)
	} else {
		fmt.Printf("  Target SDK       %d\n", v)
	}
}

// nativeLibsAre16KAligned reads the archive directly rather than shelling out
// to zipalign, which is not always installed and differs between build-tools
// versions. An uncompressed .so must start on a 16 KB boundary; a compressed
// one is extracted at install time and aligned by the installer.
func nativeLibsAre16KAligned(path string) (bool, string) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return false, fmt.Sprintf("cannot read %s: %v", filepath.Base(path), err)
	}
	defer r.Close()

	const pageSize = 16 * 1024
	checked := 0
	for _, f := range r.File {
		if !strings.HasSuffix(f.Name, ".so") {
			continue
		}
		if f.Method != zip.Store {
			continue // extracted and aligned at install time
		}
		off, err := f.DataOffset()
		if err != nil {
			return false, fmt.Sprintf("cannot locate %s: %v", f.Name, err)
		}
		checked++
		if off%pageSize != 0 {
			return false, fmt.Sprintf("%s starts at %d, not a multiple of 16384", f.Name, off)
		}
	}
	if checked == 0 {
		return true, "no uncompressed native libraries"
	}
	return true, ""
}

// artifactIsSigned looks for a signature block. v2/v3 signatures live in the
// archive itself; v1 leaves files under META-INF.
func artifactIsSigned(path string) (bool, string) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return false, fmt.Sprintf("cannot read %s", filepath.Base(path))
	}
	defer r.Close()
	for _, f := range r.File {
		n := strings.ToUpper(f.Name)
		if strings.HasPrefix(n, "META-INF/") &&
			(strings.HasSuffix(n, ".RSA") || strings.HasSuffix(n, ".DSA") ||
				strings.HasSuffix(n, ".EC") || strings.HasSuffix(n, ".SF")) {
			return true, ""
		}
	}
	return false, "no signature block found"
}

// artifactTargetSdk reads the level the artifact declares.
//
// An .aab keeps its manifest as protobuf, which aapt2 dump badging cannot
// read — it is an APK tool, and pointed at a bundle it simply reports nothing.
// So for a bundle the answer comes from what produced it: the setting if the
// project has one, otherwise the default written into the generated Gradle
// file. That is the same number Gradle used, read from the same place.
func artifactTargetSdk(path string) int {
	if strings.HasSuffix(path, ".aab") {
		if v := settingValue("android.target_sdk"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				return n
			}
		}
		return gradleDefaultTargetSdk()
	}
	aapt := aapt2Bin()
	if aapt == "" {
		return 0
	}
	out, err := exec.Command(aapt, "dump", "badging", path).Output()
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.HasPrefix(line, "targetSdkVersion:") {
			continue
		}
		v := strings.Trim(strings.TrimPrefix(line, "targetSdkVersion:"), "' ")
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 0
}

// aapt2Bin finds aapt2 in the pinned build-tools, or "" if it is absent.
//
// The .exe suffix matters: everything else here is pure Go and portable, and
// this was the one line that would have made the target-SDK check silently
// return zero on Windows — reporting "could not be read" on a machine where
// the tool was sitting right there.
func aapt2Bin() string {
	name := "aapt2"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	p := filepath.Join(androidHome(), "build-tools", pinBuildTools, name)
	if pathExists(p) {
		return p
	}
	return ""
}

// gradleDefaultTargetSdk reads the fallback out of the generated shell — the
// `?: 35` in `targetSdk = providers.gradleProperty(...).orNull ?: 35`.
//
// Read rather than hardcoded: the shell is regenerated from a template that
// moves, and a second copy of the number here would be a second thing to
// update and the one nobody remembers.
func gradleDefaultTargetSdk() int {
	body, err := os.ReadFile(filepath.Join("android", "Example", "app", "build.gradle.kts"))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(body), "\n") {
		if !strings.Contains(line, "targetSdk") {
			continue
		}
		if i := strings.LastIndex(line, "?:"); i >= 0 {
			if n, err := strconv.Atoi(strings.TrimSpace(line[i+2:])); err == nil {
				return n
			}
		}
	}
	return 0
}

// signedWithDebugKeystore reports whether the release would be signed with the
// keystore every Android SDK ships.
//
// It signs, it installs, and it builds — and Play rejects the upload, after the
// wait. Worth saying at build time rather than at the end of a release.
func signedWithDebugKeystore() (bool, string) {
	ks := settingValue("android.keystore")
	if ks == "" {
		ks = filepath.Join(homeDir(), ".android", "debug.keystore")
	}
	ks = expandHome(ks)
	return strings.HasSuffix(ks, "debug.keystore"), ks
}
