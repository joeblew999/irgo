package content

import (
	"embed"
	"fmt"
	"strings"
)

// Samples that are read out of compiled Go rather than typed into a string.
//
// A sample in a string literal is prose: nothing compiles it, and the realtime
// page shipped four handlers built on a hub that never existed. A sample in
// content/examples is code: `go build ./...` compiles it, and the API checker
// and the compiler both have an opinion about it.
//
// Marked regions rather than whole files, because a compiling example needs
// scaffolding — a hub to call, a renderer to render with — that would only be
// noise on the page.

//go:embed examples/*.go
var exampleFiles embed.FS

// Example returns the region marked `doc:start <name>` in content/examples.
//
// It panics on a missing name. This runs at package initialisation, so the
// alternative is a page that renders with a hole in it, and the name is a
// constant in this package's own source — if it is wrong, it is wrong for
// everyone, immediately.
func Example(name string) string {
	src, err := findRegion(name)
	if err != nil {
		panic("docs: " + err.Error())
	}
	return src
}

func findRegion(name string) (string, error) {
	entries, err := exampleFiles.ReadDir("examples")
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		data, err := exampleFiles.ReadFile("examples/" + e.Name())
		if err != nil {
			return "", err
		}
		if region, ok := extract(string(data), name); ok {
			return region, nil
		}
	}
	return "", fmt.Errorf("no example marked %q in content/examples", name)
}

// extract pulls the lines between the markers, dropping the markers
// themselves.
func extract(src, name string) (string, bool) {
	start := "// doc:start " + name
	var out []string
	var in bool
	for _, line := range strings.Split(src, "\n") {
		switch {
		case strings.TrimSpace(line) == start:
			in = true
		case in && strings.TrimSpace(line) == "// doc:end":
			return strings.Join(out, "\n"), true
		case in:
			out = append(out, line)
		}
	}
	return "", false
}
