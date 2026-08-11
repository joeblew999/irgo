package i18n

import (
	"strings"
	"sync/atomic"

	"golang.org/x/text/language"
)

// declared is what the host told us the device is set to, if anything.
//
// An atomic value rather than a plain slice: it is written once during startup
// on the platform's main thread and read on every request from whichever
// goroutine is serving it. That is a data race in the ordinary way, and the
// kind that survives every test on a device with one user tapping slowly.
var declared atomic.Pointer[[]language.Tag]

// SetPreferred records the languages this device is set to, in order.
//
// For the hosts that have no other way to say. A web request carries
// Accept-Language and a desktop process inherits LC_ALL, but a mobile app has
// neither: gomobile inherits no shell environment, so LANG is unset, and the
// WebView's intercepted requests do not reliably carry the header either —
// Android's shouldInterceptRequest exposes a subset that does not include it.
//
// Without this, an app translated into the user's language served them the
// source language on a phone, correctly, with every catalog present and no
// error to explain it.
//
// Called once at startup rather than read on demand, because the answer lives
// behind a platform API in Swift or Kotlin and Go cannot reach it. A language
// changed while the app runs restarts the activity on Android and re-launches
// on iOS, so once is enough.
func SetPreferred(tags ...language.Tag) {
	if len(tags) == 0 {
		declared.Store(nil)
		return
	}
	t := append([]language.Tag(nil), tags...)
	declared.Store(&t)
}

// SetPreferredTags is SetPreferred for hosts that can only pass a string.
//
// gomobile binds strings, not slices of a package's types, so this is the form
// the native shells actually call. The argument is what both platforms already
// produce: a comma-separated BCP 47 list, from Locale.preferredLanguages on
// iOS and LocaleList.toLanguageTags on Android.
//
// Unparseable entries are skipped rather than rejected. This is a startup call
// from a platform API, and a device offering one tag Go cannot read is not a
// reason to have no languages at all.
func SetPreferredTags(list string) {
	var tags []language.Tag
	for _, s := range strings.Split(list, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		// Android writes und for an undetermined locale, and x/text parses it
		// happily into a tag that matches nothing.
		if t, err := language.Parse(s); err == nil && t != language.Und {
			tags = append(tags, t)
		}
	}
	SetPreferred(tags...)
}

// Declared is what SetPreferred was given, or nil.
func Declared() []language.Tag {
	if p := declared.Load(); p != nil {
		return *p
	}
	return nil
}
