// Stopping what irgo started.
//
// irgo boots an emulator when a run needs one and then leaves it running, on
// purpose: booting takes minutes, and a developer running the app twice should
// not wait twice. What was missing is the other end. An emulator here had been
// up since Thursday at 760% CPU, headless, invisible — nobody had started it
// deliberately and nothing offered to stop it.
//
// Not automatic. Shutting down after every run would make the second run as
// slow as the first, which is the cost the reuse exists to avoid. It is a
// command because the developer knows when they are finished and irgo does
// not.
package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// runAndroidStop shuts down a running emulator.
//
// `adb emu kill` rather than killing the process: it asks the emulator to shut
// down, so a snapshot is saved and the AVD is not left needing a repair on
// next boot.
func runAndroidStop() error {
	if !adbRunning() {
		fmt.Println("No emulator or device is running.")
		return nil
	}

	// A physical device is not ours to stop, and `adb emu kill` on one is a
	// confusing no-op rather than an error.
	if !attachedIsEmulator() {
		fmt.Println("A physical device is attached, not an emulator — nothing to stop.")
		return nil
	}

	fmt.Println("Stopping the emulator...")
	out, err := exec.Command(adbBin(), "emu", "kill").CombinedOutput()
	if err != nil {
		return fmt.Errorf("stopping the emulator: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	fmt.Println("Stopped. The next `irgo app run android` will boot it again,")
	fmt.Println("which takes a minute or two.")
	return nil
}

// attachedIsEmulator reports whether what adb sees is an emulator.
func attachedIsEmulator() bool {
	out, err := exec.Command(adbBin(), "devices").Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "emulator-") {
			return true
		}
	}
	return false
}

// warnIfEmulatorIsHeadless says so when the emulator running has no window.
//
// A headless emulator accepts installs and reports itself to adb exactly like
// a windowed one, so `app run android` says "App running on Android!" and the
// developer sees nothing at all. That is the most confusing possible outcome:
// everything succeeded and there is nothing to look at.
func warnIfEmulatorIsHeadless() {
	out, err := exec.Command("ps", "ax", "-o", "command").Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "qemu-system") {
			continue
		}
		if !strings.Contains(line, "headless") && !strings.Contains(line, "-no-window") {
			continue
		}
		fmt.Println()
		fmt.Println("Note: the emulator running is headless — it has no window, so")
		fmt.Println("      there is nothing to see even though the app is installed")
		fmt.Println("      and running. To get one you can watch:")
		fmt.Println("        irgo app stop android")
		fmt.Println("        irgo app run android")
		return
	}
}
