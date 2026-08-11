// Telling each platform which languages the app has.
//
// Translations are not only text. Every output irgo produces has a manifest,
// and two of them carry language metadata the platform reads before a single
// line of the app runs:
//
//	iOS      Info.plist CFBundleLocalizations — what the App Store lists the
//	         app as supporting, and what the system offers it
//	browser  manifest.webmanifest lang and dir — the install prompt and the
//	         app name, announced before the app is opened
//
// Neither is derivable from the other and neither is visible in testing: an
// app translated into four languages ships declaring one, is listed as
// English-only in every store front, and works perfectly for anyone who
// already found it.
//
// The catalogs are the single source, the way they are for everything else
// here. A list of locales in a plist and another in a manifest would be the
// two-lists bug in its third and fourth forms — and these are the versions
// nobody would notice, because being wrong costs discoverability rather than
// correctness.
//
// Synced on every build rather than written once at scaffold time. `irgo i18n
// add fr` changes the answer, and a manifest that only knows what was true on
// the day the project was created is a manifest that is wrong from the second
// language onward.
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// syncI18nManifests updates whatever manifests this project has.
//
// Silent and skipping for a project without translations, and for targets that
// have not been scaffolded — most projects have not generated every shell.
func syncI18nManifests() error {
	if !hasI18n() {
		return nil
	}
	locales := projectLocales()
	if len(locales) == 0 {
		return nil
	}
	for _, path := range []string{
		"ios/Example/Example/Info.plist",
		"ios/App/Info.plist",
	} {
		if err := setPlistLocalizations(path, locales); err != nil {
			return err
		}
	}
	return nil
}

// plistLocalizations matches the array this writes, so re-running replaces it
// rather than appending a second one — which is valid XML and undefined
// behaviour in a plist.
var plistLocalizations = regexp.MustCompile(
	`(?s)\n\t<key>CFBundleLocalizations</key>\n\t<array>.*?\n\t</array>`)

// setPlistLocalizations declares the app's languages to iOS.
//
// Without it the App Store lists the app as supporting only its development
// region. The translations still work for anyone who has the app — this is
// entirely about whether a German speaker searching the store is shown it as a
// German app, which is the difference between the translations being read and
// not.
func setPlistLocalizations(path string, locales []string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil // this shell has not been generated
	}
	text := string(body)

	var b strings.Builder
	b.WriteString("\n\t<key>CFBundleLocalizations</key>\n\t<array>")
	for _, l := range locales {
		// BCP 47 in the catalogs, and that is what iOS wants here too — it
		// accepts "pt-BR" as readily as "pt".
		fmt.Fprintf(&b, "\n\t\t<string>%s</string>", l)
	}
	b.WriteString("\n\t</array>")
	want := b.String()

	switch {
	case plistLocalizations.MatchString(text):
		if plistLocalizations.FindString(text) == want {
			return nil // already correct; do not rewrite and dirty the tree
		}
		text = plistLocalizations.ReplaceAllString(text, want)
	default:
		// After the opening <dict>, so the file stays readable. Order is not
		// significant in a plist, but a key appended after the closing tag
		// would simply be outside the dictionary and silently ignored.
		const anchor = "<dict>"
		i := strings.Index(text, anchor)
		if i < 0 {
			return nil // not a plist shape irgo recognises
		}
		at := i + len(anchor)
		text = text[:at] + want + text[at:]
	}
	return os.WriteFile(path, []byte(text), 0o644)
}

func init() {
	// After the bundle is regenerated, so the locale list is the one this
	// build actually produced rather than the one before `irgo i18n add`.
	registerAssetStep(assetOrderLate, syncI18nManifests)
}
