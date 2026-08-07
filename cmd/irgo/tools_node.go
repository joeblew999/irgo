// Node, only because wrangler needs it.
//
// irgo has no Node in its own story — Tailwind is a standalone binary and there
// is no npm anywhere in a generated project. But deploying to Cloudflare means
// running wrangler, wrangler is a Node program, and Cloudflare states plainly
// that the bun runtime is unsupported (`wrangler dev` fails outright on it).
//
// So irgo provisions a Node the same way it provisions the JDK: downloaded into
// ~/.irgo, used by irgo alone, removed by `irgo tools remove`. Nothing is
// installed system-wide and no package manager is involved, which matters
// because a machine that manages Node through a version manager without a
// global default has a `node` on PATH that fails rather than one that works —
// exactly the failure this exists to avoid.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// pinNode is the version irgo fetches. Pinned so a deploy does not start
// failing because upstream moved.
const pinNode = "22.14.0"

func managedNodeHome() string { return filepath.Join(homeDir(), ".irgo", "node") }

// nodeBin returns a usable node, provisioning one if needed.
//
// A node already on PATH is preferred — but only if it actually runs. A shim
// that errors when no version is selected reports success to LookPath and then
// fails, so it is tested rather than trusted.
func nodeBin(install bool) (string, error) {
	if p, err := exec.LookPath("node"); err == nil && nodeWorks(p) {
		return p, nil
	}
	managed := filepath.Join(managedNodeHome(), "bin", "node")
	if runtime.GOOS == "windows" {
		managed = filepath.Join(managedNodeHome(), "node.exe")
	}
	if nodeWorks(managed) {
		return managed, nil
	}
	if !install {
		return "", fmt.Errorf("no working node found — irgo can install one into %s", managedNodeHome())
	}
	return installNode()
}

// nodeWorks reports whether this binary actually runs.
func nodeWorks(bin string) bool {
	if bin == "" {
		return false
	}
	if _, err := os.Stat(bin); err != nil {
		return false
	}
	out, err := exec.Command(bin, "--version").CombinedOutput()
	if err != nil {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(string(out)), "v")
}

// installNode downloads Node into ~/.irgo/node.
func installNode() (string, error) {
	osName := map[string]string{"darwin": "darwin", "linux": "linux", "windows": "win"}[runtime.GOOS]
	if osName == "" {
		return "", fmt.Errorf("unsupported OS for the Node download: %s", runtime.GOOS)
	}
	arch := map[string]string{"amd64": "x64", "arm64": "arm64"}[runtime.GOARCH]
	if arch == "" {
		return "", fmt.Errorf("unsupported architecture for the Node download: %s", runtime.GOARCH)
	}

	ext, name := "tar.gz", fmt.Sprintf("node-v%s-%s-%s", pinNode, osName, arch)
	if osName == "win" {
		ext = "zip"
	}
	url := fmt.Sprintf("https://nodejs.org/dist/v%s/%s.%s", pinNode, name, ext)

	dest := managedNodeHome()
	if err := os.RemoveAll(dest); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp("", "irgo-node")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	archive := filepath.Join(tmp, "node."+ext)
	fmt.Printf("Downloading Node %s (%s/%s) — wrangler needs it...\n", pinNode, osName, arch)
	if err := downloadFile(url, archive); err != nil {
		return "", fmt.Errorf("Node download failed: %w", err)
	}
	if ext == "zip" {
		err = unzipTo(archive, tmp)
	} else {
		err = untarGz(archive, tmp)
	}
	if err != nil {
		return "", fmt.Errorf("unpacking Node failed: %w", err)
	}
	// The archive holds a single node-v<version>-<os>-<arch>/ directory; move
	// its contents up so the layout does not carry the version in its path.
	if err := os.RemoveAll(dest); err != nil {
		return "", err
	}
	if err := os.Rename(filepath.Join(tmp, name), dest); err != nil {
		return "", fmt.Errorf("installing Node into %s: %w", dest, err)
	}
	markToolInstalled("node")

	bin := filepath.Join(dest, "bin", "node")
	if runtime.GOOS == "windows" {
		bin = filepath.Join(dest, "node.exe")
	}
	if !nodeWorks(bin) {
		return "", fmt.Errorf("Node unpacked to %s but does not run", dest)
	}
	fmt.Printf("Installed Node at %s\n", dest)
	return bin, nil
}

// npxCommand runs a Node CLI through npx using irgo's Node.
//
// It invokes npx-cli.js with node rather than the bin/npx wrapper, because
// that wrapper is a symlink in the official tarball and irgo's extractor keeps
// regular files only — so bin/ holds node and nothing else. Calling the script
// directly sidesteps the question entirely and works the same on Windows,
// where those wrappers are .cmd shims rather than links.
//
// irgo's node goes first on PATH so anything npx spawns finds it too.
func npxCommand(node string, args ...string) *exec.Cmd {
	npxCLI := filepath.Join(managedNodeHome(), "lib", "node_modules", "npm", "bin", "npx-cli.js")
	if !pathExists(npxCLI) {
		// A node from PATH rather than irgo's: use its npx, which is intact.
		dir := filepath.Dir(node)
		npx := filepath.Join(dir, "npx")
		if runtime.GOOS == "windows" {
			npx = filepath.Join(dir, "npx.cmd")
		}
		cmd := exec.Command(npx, args...)
		cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
		return cmd
	}
	cmd := exec.Command(node, append([]string{npxCLI}, args...)...)
	cmd.Env = append(os.Environ(),
		"PATH="+filepath.Dir(node)+string(os.PathListSeparator)+os.Getenv("PATH"))
	return cmd
}
