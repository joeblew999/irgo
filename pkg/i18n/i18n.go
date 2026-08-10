// Which language to show, on five targets that disagree about how to ask.
//
// The generated toki bundle answers "given these preferences, which catalog?".
// It does not answer "what are this user's preferences?", because that depends
// entirely on where the app is running:
//
//	web        the Accept-Language header
//	desktop    the OS locale, via LC_ALL / LC_MESSAGES / LANG
//	browser    navigator.languages
//	mobile     the OS, handed in by the native shell
//	worker     Accept-Language again, plus Cloudflare's own geo headers
//
// An irgo app is all of these from one codebase, so this is the piece that
// would otherwise be written five times, subtly differently, and be wrong on
// whichever target the author did not have open at the time.
//
// Everything returns []language.Tag in preference order, which is what
// tokibundle.Match takes:
//
//	reader, _ := tokibundle.Match(i18n.Preferred(r)...)
//
// # Why order, and why more than one
//
// A user with `de-CH, de;q=0.9, en;q=0.5` is asking for Swiss German, then any
// German, then English. Taking only the first and falling back to the source
// language throws away the middle of that list — so a de-CH speaker on an app
// with a de catalog and no de-CH one gets English, having explicitly said they
// read German. Passing the whole list lets the matcher do what it is for.
package i18n

import (
	"net/http"
	"os"
	"sort"
	"strings"

	"golang.org/x/text/language"
)

// FromRequest reads the browser's Accept-Language header.
//
// Never an error: this is attacker-controlled input on every request, and the
// only useful response to nonsense is to have no preference. An empty list
// already means that to the caller.
//
// The salvage path is the part worth explaining. x/text parses the header as a
// unit and is all-or-nothing — `de,!!!` returns no tags at all, not `de` and a
// complaint. So a single malformed entry from a proxy, a crawler or a
// hand-rolled client discards preferences the user really did express, and
// then Preferred falls through to the machine's locale, which on a server is
// usually unset. The visible result is an app that serves its source language
// to someone who asked for German, with nothing logged anywhere.
//
// So a failed parse is retried entry by entry, because each entry parses fine
// on its own. q-values are read by x/text rather than by hand — the ordering
// rules are the reason to use a library for this at all.
func FromRequest(r *http.Request) []language.Tag {
	if r == nil {
		return nil
	}
	header := r.Header.Get("Accept-Language")
	if header == "" {
		return nil
	}
	if tags, _, err := language.ParseAcceptLanguage(header); err == nil {
		return tags
	}

	type ranked struct {
		tag language.Tag
		q   float32
	}
	var found []ranked
	for _, entry := range strings.Split(header, ",") {
		tags, qs, err := language.ParseAcceptLanguage(strings.TrimSpace(entry))
		if err != nil || len(tags) == 0 {
			continue
		}
		found = append(found, ranked{tags[0], qs[0]})
	}
	// Stable, so entries of equal quality keep the order the client sent them
	// in — which is what the header means when q is omitted.
	sort.SliceStable(found, func(i, j int) bool { return found[i].q > found[j].q })

	out := make([]language.Tag, len(found))
	for i, f := range found {
		out[i] = f.tag
	}
	return out
}

// FromEnv reads the OS locale the way POSIX defines it.
//
// The order is the standard's, not a preference: LC_ALL overrides everything,
// LC_MESSAGES covers text specifically, and LANG is the fallback. LANGUAGE is
// last and is a GNU extension holding a colon-separated list, which is the one
// place a desktop user can express more than a single choice.
//
// The values are POSIX locale names, not BCP 47 tags: `de_AT.UTF-8` rather
// than `de-AT`. The encoding suffix and the underscore both have to go, or
// every parse fails and a correctly configured desktop silently gets English.
func FromEnv() []language.Tag {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(key); v != "" {
			if t := parsePOSIX(v); t != language.Und {
				return append([]language.Tag{t}, fromLANGUAGE()...)
			}
		}
	}
	return fromLANGUAGE()
}

// fromLANGUAGE reads the GNU list, which is how a Linux desktop says "German,
// but English before French".
func fromLANGUAGE() []language.Tag {
	var out []language.Tag
	for _, part := range strings.Split(os.Getenv("LANGUAGE"), ":") {
		if t := parsePOSIX(part); t != language.Und {
			out = append(out, t)
		}
	}
	return out
}

// parsePOSIX turns de_AT.UTF-8@euro into de-AT.
func parsePOSIX(v string) language.Tag {
	v = strings.TrimSpace(v)
	// C and POSIX are "no locale", not a language. Parsing them yields a tag
	// that looks real and matches nothing.
	if v == "" || v == "C" || v == "POSIX" {
		return language.Und
	}
	if i := strings.IndexAny(v, ".@"); i >= 0 {
		v = v[:i]
	}
	t, err := language.Parse(strings.ReplaceAll(v, "_", "-"))
	if err != nil {
		return language.Und
	}
	return t
}

// Preferred is what a handler should call: what this request asked for, and
// what this machine is set to if it asked for nothing.
//
// Both, in that order, rather than one or the other. A desktop irgo app serves
// its own UI over HTTP to a local webview, and that webview may send no
// Accept-Language at all — so a request-only answer gives an English UI on a
// German desktop, and an environment-only answer ignores a real browser user's
// stated preference. Neither source is sufficient alone, which is exactly why
// this is a function and not a line in a handler.
func Preferred(r *http.Request) []language.Tag {
	return append(FromRequest(r), platformPreferred()...)
}

// Reader picks the localization for these preferences, and the bundle's
// default when none of them is a real match.
//
// The confidence is the whole point, and discarding it is the easy mistake —
// including in toki's own quick start, which writes:
//
//	reader, _ := tokibundle.Match(language.BritishEnglish)
//
// That looks idiomatic and is wrong. x/text's matcher never fails: asked for a
// language it does not have, it returns the first supported one and reports
// language.No alongside it. So an app with German and English catalogs serves
// GERMAN to a French speaker — not the source language, not a 404, just
// confidently the wrong language — and the only signal was the value the
// underscore threw away.
//
// It is invisible in development, because the languages you test with are the
// ones you have catalogs for. It appears for users whose language you have not
// translated yet, which is everyone the feature exists to reach.
//
// Generic over the reader because the bundle is generated per project: irgo
// cannot import it, and this has to work against whatever type it declares.
//
//	reader := i18n.Reader(tokibundle.Match, tokibundle.Default, i18n.Preferred(r)...)
func Reader[R any](
	match func(...language.Tag) (R, language.Confidence),
	fallback func() R,
	prefs ...language.Tag,
) R {
	r, conf := match(prefs...)
	if conf == language.No {
		return fallback()
	}
	return r
}
