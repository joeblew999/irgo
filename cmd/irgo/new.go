package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Use the all: prefix so files starting with "." or "_" (e.g. .gitignore,
// .air.toml, files under hidden directories) are embedded too - plain
// go:embed patterns skip them inside matched directories.
//
//go:embed all:templates
var templateFS embed.FS

// Datastar files to download during project creation
var datastarFiles = map[string]string{
	"static/js/datastar.js": "https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.0-RC.7/bundles/datastar.js",
}

// downloadDatastar downloads Datastar files to the project's static/js directory
func downloadDatastar(projectDir string) error {
	for destPath, url := range datastarFiles {
		fullPath := filepath.Join(projectDir, destPath)

		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", destPath, err)
		}

		// Download the file
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("downloading %s: %w", url, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("downloading %s: status %d", url, resp.StatusCode)
		}

		// Read the content
		content, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading %s: %w", url, err)
		}

		// Write to file
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", destPath, err)
		}

		fmt.Printf("  downloaded: %s\n", destPath)
	}

	return nil
}

// getGoVersion returns the current Go version (e.g., "1.24.12")
func getGoVersion() string {
	out, err := exec.Command(goBin(), "version").Output()
	if err != nil {
		return "1.23"
	}
	// Parse "go version go1.24.12 darwin/arm64"
	re := regexp.MustCompile(`go(\d+\.\d+(?:\.\d+)?)`)
	match := re.FindStringSubmatch(string(out))
	if len(match) > 1 {
		return match[1]
	}
	return "1.23"
}

// replaceDirective returns the go.mod `replace` line for the generated project,
// or "" to build against the published upstream module.
//
// IRGO_REPLACE pins a *published* module version — "github.com/joeblew999/irgo
// v0.4.0-androidapi21.24" (an "@" between module and version also works). This
// is what lets a downstream repo regenerate its app entirely from the CLI: the
// fork pin lives in that repo's own config, so the generated go.mod is pure
// output that never needs a hand-edit afterwards.
//
// Otherwise fall back to a local source checkout — the path replace you want
// when hacking on irgo itself.
func replaceDirective() string {
	if mod := strings.TrimSpace(os.Getenv("IRGO_REPLACE")); mod != "" {
		// Accept "module@version" as well as go.mod's own "module version".
		return fmt.Sprintf("\nreplace github.com/stukennedy/irgo => %s\n",
			strings.Replace(mod, "@", " ", 1))
	}
	if path := getIrgoPath(); path != "" {
		return fmt.Sprintf("\nreplace github.com/stukennedy/irgo => %s\n", path)
	}
	return ""
}

// getIrgoPath returns the path to the irgo source directory if developing locally
func getIrgoPath() string {
	// Check IRGO_PATH environment variable first
	if path := os.Getenv("IRGO_PATH"); path != "" {
		return path
	}

	// Helper to check if a directory contains irgo source
	isIrgoDir := func(dir string) bool {
		modPath := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(modPath); err == nil {
			return strings.Contains(string(data), "module github.com/stukennedy/irgo")
		}
		return false
	}

	// Check if we're running from within or near the irgo source tree
	// by looking at current directory and parents
	cwd, err := os.Getwd()
	if err == nil {
		dir := cwd
		for i := 0; i < 10; i++ {
			if isIrgoDir(dir) {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// Check relative to executable location
	// If irgo binary is at /path/to/irgo/cmd/irgo/irgo, source is at /path/to/irgo
	if execPath, err := os.Executable(); err == nil {
		execPath, _ = filepath.EvalSymlinks(execPath)
		execDir := filepath.Dir(execPath)

		// Check if we're in cmd/irgo directory
		if filepath.Base(filepath.Dir(execDir)) == "cmd" {
			possibleRoot := filepath.Dir(filepath.Dir(execDir))
			if isIrgoDir(possibleRoot) {
				return possibleRoot
			}
		}

		// Check parent directories of executable
		dir := execDir
		for i := 0; i < 5; i++ {
			if isIrgoDir(dir) {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// Check common development locations
	home, _ := os.UserHomeDir()
	commonPaths := []string{
		filepath.Join(home, "Dev", "@irgo", "core"),
		filepath.Join(home, "Dev", "irgo"),
		filepath.Join(home, "dev", "irgo"),
		filepath.Join(home, "Development", "irgo"),
		filepath.Join(home, "Projects", "irgo"),
		filepath.Join(home, "go", "src", "github.com", "stukennedy", "irgo"),
	}

	for _, path := range commonPaths {
		if isIrgoDir(path) {
			return path
		}
	}

	return ""
}

// isRemoteModulePath checks if a path looks like a remote Go module path
func isRemoteModulePath(path string) bool {
	remotePrefixes := []string{
		"github.com/",
		"gitlab.com/",
		"bitbucket.org/",
		"gopkg.in/",
	}
	for _, prefix := range remotePrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	// Any path with dots before slashes is likely remote
	if strings.Contains(path, ".") && strings.Contains(path, "/") {
		dotIdx := strings.Index(path, ".")
		slashIdx := strings.Index(path, "/")
		if dotIdx < slashIdx {
			return true
		}
	}
	return false
}

func newProject(name string) error {
	// Determine project directory, project name, and module path
	var projectDir string
	var projectName string
	var modulePath string

	if name == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
		projectDir = cwd
		projectName = filepath.Base(cwd)
		modulePath = projectName
	} else if filepath.IsAbs(name) {
		// Absolute path provided
		projectDir = name
		projectName = filepath.Base(name)
		modulePath = projectName
	} else if isRemoteModulePath(name) {
		// Remote module path like "github.com/user/project"
		// Use the last part for directory, full path for module
		projectDir = filepath.Base(name)
		projectName = filepath.Base(name)
		modulePath = name
	} else {
		projectDir = name
		projectName = name
		modulePath = name
	}

	// Check if directory exists and is not empty
	if name != "." {
		if _, err := os.Stat(projectDir); err == nil {
			entries, _ := os.ReadDir(projectDir)
			if len(entries) > 0 {
				return fmt.Errorf("directory %s already exists and is not empty", projectDir)
			}
		}
	}

	fmt.Printf("Creating new irgo project: %s\n", projectName)

	// Create project structure
	dirs := []string{
		"handlers",
		"templates",
		"static/css",
		"static/js",
	}

	for _, dir := range dirs {
		path := filepath.Join(projectDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", path, err)
		}
	}

	// Copy template files
	err := fs.WalkDir(templateFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// irgo.package.toml holds settings a person chose — the signing team,
		// store IDs, the app version. The template version is only a seed, so
		// regenerating in place (irgo new .) must not overwrite an existing
		// one: that silently discards configuration and the next build fails
		// somewhere unrelated.
		if strings.HasSuffix(path, "/"+packageConfigFile) || path == "templates/"+packageConfigFile {
			if _, err := os.Stat(filepath.Join(projectDir, packageConfigFile)); err == nil {
				fmt.Printf("  kept (yours): %s\n", packageConfigFile)
				return nil
			}
		}

		// The CI workflows live under templates/github but are not part of a
		// new project — `irgo ci` scaffolds them to .github/ on request.
		// Without this they land in every project as a stray github/ folder
		// that GitHub ignores, so nothing runs and nothing says why.
		if path == "templates/github" {
			return fs.SkipDir
		}

		// Skip the root templates directory
		if path == "templates" {
			return nil
		}

		// Get relative path from templates/
		relPath := strings.TrimPrefix(path, "templates/")
		destPath := filepath.Join(projectDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// Read template file
		content, err := templateFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading template %s: %w", path, err)
		}

		// Handle .tmpl extension (remove it) - do this before checking file type
		if strings.HasSuffix(destPath, ".tmpl") {
			destPath = strings.TrimSuffix(destPath, ".tmpl")
			relPath = strings.TrimSuffix(relPath, ".tmpl")
		}

		// Replace placeholders
		contentStr := string(content)
		contentStr = strings.ReplaceAll(contentStr, "{{PROJECT_NAME}}", projectName)
		contentStr = strings.ReplaceAll(contentStr, "{{MODULE_PATH}}", modulePath)
		contentStr = strings.ReplaceAll(contentStr, "{{GO_VERSION}}", getGoVersion())

		// Pin irgo to a fork tag (IRGO_REPLACE) or a local checkout
		if strings.HasSuffix(relPath, "go.mod") {
			contentStr = strings.ReplaceAll(contentStr, "{{REPLACE_DIRECTIVE}}", replaceDirective())
		} else {
			contentStr = strings.ReplaceAll(contentStr, "{{REPLACE_DIRECTIVE}}", "")
		}

		// Shell scripts and gradle wrappers must be executable
		fileMode := os.FileMode(0644)
		base := filepath.Base(destPath)
		if strings.HasSuffix(base, ".sh") || base == "gradlew" {
			fileMode = 0755
		}

		// Write file
		if err := os.WriteFile(destPath, []byte(contentStr), fileMode); err != nil {
			return fmt.Errorf("writing %s: %w", destPath, err)
		}

		fmt.Printf("  created: %s\n", relPath)
		return nil
	})

	if err != nil {
		return fmt.Errorf("copying templates: %w", err)
	}

	// Download Datastar files
	fmt.Println("Downloading Datastar...")
	if err := downloadDatastar(projectDir); err != nil {
		return fmt.Errorf("downloading Datastar: %w", err)
	}

	// Make scripts executable
	scripts := []string{}
	for _, script := range scripts {
		path := filepath.Join(projectDir, script)
		if err := os.Chmod(path, 0755); err != nil {
			// Ignore if file doesn't exist
			continue
		}
	}

	// Every project wants CI, so scaffold it rather than making it a step
	// people have to know about. `irgo ci --force` regenerates it later.
	if err := runCIIn(projectDir, modulePath); err != nil {
		fmt.Printf("Warning: could not scaffold CI workflows: %v\n", err)
	}

	// Generate templ files BEFORE tidy. Only the generated _templ.go files
	// import templ directly; tidy run first sees no importer and records templ
	// as `// indirect`, which the first `irgo templ` then flips back to direct
	// — dirtying go.mod, a generated file, on the very first build.
	if _, err := exec.LookPath("templ"); err == nil {
		fmt.Println("Generating templ files...")
		cmd := exec.Command("templ", "generate")
		cmd.Dir = projectDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: templ generate failed: %v\n", err)
		}
	}

	// Run go mod tidy to download dependencies
	// Skip if it's a remote module path that doesn't exist yet
	if isRemoteModulePath(modulePath) {
		fmt.Println("Skipping go mod tidy (remote module path - run manually after pushing to remote)")
	} else {
		fmt.Println("Running go mod tidy...")
		tidyCmd := exec.Command(goBin(), "mod", "tidy")
		tidyCmd.Dir = projectDir
		tidyCmd.Stdout = os.Stdout
		tidyCmd.Stderr = os.Stderr
		if err := tidyCmd.Run(); err != nil {
			fmt.Printf("Warning: go mod tidy failed: %v\n", err)
		}
	}

	fmt.Println()
	fmt.Println("Project created successfully!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", projectDir)
	fmt.Println("  bun install        # or: npm install")
	fmt.Println("  irgo dev           # start development server")
	fmt.Println("  irgo run ios       # iOS Simulator (scaffolds ios/Example)")
	fmt.Println("  irgo run android   # Android Emulator (scaffolds android/Example)")
	fmt.Println("  irgo run desktop   # native desktop window")
	fmt.Println()

	return nil
}

// scaffoldExamples writes the canonical mobile example apps (ios/Example and
// android/Example) into the CURRENT directory from the embedded templates.
// It is missing-only and idempotent — an existing Example project is never
// overwritten, so build/run can call it unconditionally. This is what makes
// `irgo build/run android|ios` work on a bare project: devs and CI never
// hand-copy the native shells, and never need a separate scaffold step.
func scaffoldExamples() error {
	modulePath, err := getModulePath()
	if err != nil {
		return fmt.Errorf("could not determine module path: %w", err)
	}
	projectName := filepath.Base(modulePath)
	if projectName == "." || projectName == "" || projectName == "/" {
		wd, _ := os.Getwd()
		projectName = filepath.Base(wd)
	}
	if projectName == "." || projectName == "" {
		projectName = "irgo"
	}

	fmt.Println("Ensuring mobile example projects (ios/Example, android/Example)...")
	for _, sub := range []string{"ios/Example", "android/Example"} {
		if err := scaffoldExampleDir(sub, projectName, modulePath); err != nil {
			return err
		}
	}
	return nil
}

// scaffoldExampleDir copies one example subtree (rel, e.g. "ios/Example") from
// the embedded templates into the current directory, resolving the same
// placeholders as irgo new. Skips (without error) when the destination already
// exists so it is safe to call on every build/run.
func scaffoldExampleDir(rel, projectName, modulePath string) error {
	if _, err := os.Stat(rel); err == nil {
		fmt.Printf("  exists (skipped): %s\n", rel)
		return nil
	}

	written := 0
	srcPrefix := "templates/" + rel
	err := fs.WalkDir(templateFS, srcPrefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		destPath := strings.TrimPrefix(path, "templates/")
		if d.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}

		content, err := templateFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading template %s: %w", path, err)
		}

		// Handle .tmpl extension (remove it), like irgo new.
		if strings.HasSuffix(destPath, ".tmpl") {
			destPath = strings.TrimSuffix(destPath, ".tmpl")
		}

		// Resolve placeholders. Only {{PROJECT_NAME}} appears in the example
		// templates today (Android app_name), but resolve the full set so the
		// two scaffolders stay in sync.
		contentStr := string(content)
		contentStr = strings.ReplaceAll(contentStr, "{{PROJECT_NAME}}", projectName)
		contentStr = strings.ReplaceAll(contentStr, "{{MODULE_PATH}}", modulePath)
		contentStr = strings.ReplaceAll(contentStr, "{{GO_VERSION}}", getGoVersion())
		contentStr = strings.ReplaceAll(contentStr, "{{REPLACE_DIRECTIVE}}", "")

		// Shell scripts and gradle wrappers must be executable.
		fileMode := os.FileMode(0o644)
		base := filepath.Base(destPath)
		if strings.HasSuffix(base, ".sh") || base == "gradlew" {
			fileMode = 0o755
		}

		if err := os.WriteFile(destPath, []byte(contentStr), fileMode); err != nil {
			return fmt.Errorf("writing %s: %w", destPath, err)
		}
		written++
		fmt.Printf("  created: %s\n", destPath)
		return nil
	})
	if err != nil {
		return err
	}
	if written == 0 {
		return fmt.Errorf("no files scaffolded for %s (template missing?)", rel)
	}
	fmt.Printf("  scaffolded %d files into %s\n", written, rel)
	return nil
}
