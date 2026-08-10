// Datastar attributes that parse, render, and do nothing.
//
// Datastar separates an attribute from its argument with a colon —
// `data-attr:class`, `data-signals:theme`, `data-class:active`. Written with a
// hyphen instead, `data-attr-class`, the page still renders: templ emits it,
// the browser accepts it as an unknown data attribute, Datastar ignores it, and
// the binding silently never happens.
//
// Nothing catches that. The handler test passes because the HTML is correct.
// The browser console is empty because nothing errored. The element is there,
// the signal is declared, and the class simply never appears — which reads as a
// Datastar bug, or a caching problem, or anything except a hyphen.
//
// It cost a full debugging session here: `data-attr-class="'theme-' + $theme"`
// versus `data-attr:class`. What eventually found it was reading Datastar's own
// reference, not any test. So this is the test.
//
// A warning rather than an error. There is no rule that a data-* attribute
// belongs to Datastar — a project may have its own — so this reports what looks
// wrong and leaves the judgement to whoever wrote it.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// datastarHyphenated finds `data-<plugin>-<argument>=`, the shape that should
// have been `data-<plugin>:<argument>=`.
//
// Only the plugins that take an argument. `data-text`, `data-show` and
// `data-signals` are complete attributes on their own, and matching those would
// flag every correct use of them.
var datastarHyphenated = regexp.MustCompile(
	`data-(attr|class|signals|bind|on|indicator|ref|persist)-([a-zA-Z][\w-]*)\s*=`)

// datastarSuffixIsFine are the plugins whose hyphenated form is also valid, so
// a match on them is not a mistake.
//
// data-on-load and data-on-signal-patch are real Datastar attributes, not
// `data-on:` with an argument. Flagging them would train people to ignore this.
var datastarSuffixIsFine = map[string]bool{
	"load": true, "signal-patch": true, "signal-patch-filter": true,
	"interval": true, "raf": true, "resize": true,
}

func checkDatastarSyntax() error {
	root := "templates"
	if _, err := os.Stat(root); err != nil {
		return nil // not a templ project
	}

	// have and want are both built from the regexp's own groups. Deriving want
	// from have by hand does not work: the separator is the FIRST hyphen after
	// the plugin, not the last, and an argument may contain its own —
	// data-attr-aria-label wants data-attr:aria-label, not data-attr-aria:label.
	type finding struct {
		file, line, have, want string
	}
	var found []finding

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".templ") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for i, line := range strings.Split(string(body), "\n") {
			for _, m := range datastarHyphenated.FindAllStringSubmatch(line, -1) {
				plugin, arg := m[1], m[2]
				if plugin == "on" && datastarSuffixIsFine[arg] {
					continue
				}
				found = append(found, finding{
					file: fmt.Sprintf("%s:%d", path, i+1),
					line: strings.TrimSpace(line),
					have: fmt.Sprintf("data-%s-%s", plugin, arg),
					want: fmt.Sprintf("data-%s:%s", plugin, arg),
				})
			}
		}
		return nil
	})
	if err != nil || len(found) == 0 {
		return nil
	}

	fmt.Println("Note: these Datastar attributes use a hyphen where it wants a colon.")
	fmt.Println("      They render, and they bind nothing.")
	fmt.Println()
	for _, f := range found {
		fmt.Printf("  %s\n", f.file)
		fmt.Printf("      %s   ->   %s\n", f.have, f.want)
	}
	fmt.Println()
	fmt.Println("      The full reference is .claude/skills/datastar/SKILL.md.")
	return nil
}

func init() {
	// After the templates are generated, so it reads what will actually ship.
	registerAssetStep(assetOrderLate, checkDatastarSyntax)
}
