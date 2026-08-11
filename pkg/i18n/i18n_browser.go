//go:build js && wasm

// The browser target has no Accept-Language header to read.
//
// When irgo builds for wasm the Go router runs inside a service worker and the
// fetches it answers are same-origin ones the browser generated itself — which
// do not carry the language preferences the user set. The preference is there,
// but it is on navigator, not on the request.
//
// Without this, an app that translated correctly on every other target came out
// in the source language in the browser, with nothing to say why: the header
// really was absent, so the negotiation was working exactly as written.
package i18n

import (
	"syscall/js"

	"golang.org/x/text/language"
)

// platformPreferred is navigator.languages here. There are no environment
// variables in a browser, so the POSIX path would always return nothing.
//
// A host declaration still wins if one was made. Nothing in a browser build
// makes one today, but the API is the same package on every target and an
// override that silently did nothing on one of them would be worse than not
// having it.
func platformPreferred() []language.Tag {
	if d := Declared(); len(d) > 0 {
		return append(d, FromBrowser()...)
	}
	return FromBrowser()
}

// FromBrowser reads navigator.languages, in the order the user ranked them.
func FromBrowser() []language.Tag {
	nav := js.Global().Get("navigator")
	if !nav.Truthy() {
		return nil
	}

	// navigator.languages is the full ordered list; navigator.language is the
	// single top choice. Older browsers, and some embedded webviews, have only
	// the second.
	langs := nav.Get("languages")
	if !langs.Truthy() || langs.Length() == 0 {
		if one := nav.Get("language"); one.Truthy() {
			if t, err := language.Parse(one.String()); err == nil {
				return []language.Tag{t}
			}
		}
		return nil
	}

	var out []language.Tag
	for i := 0; i < langs.Length(); i++ {
		// Already BCP 47 here — this is the browser's own list, not a POSIX
		// environment variable, so no underscore or encoding suffix to strip.
		if t, err := language.Parse(langs.Index(i).String()); err == nil {
			out = append(out, t)
		}
	}
	return out
}
