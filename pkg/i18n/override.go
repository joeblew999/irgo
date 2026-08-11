package i18n

import (
	"net/http"

	"golang.org/x/text/language"
)

// OverrideParam and OverrideCookie are how a request asks for a language
// explicitly, rather than being negotiated for.
//
// Two reasons this exists, and the second is the one that made it necessary.
//
// A user whose browser says English but who wants German has no other way to
// say so. Every app that ships more than one language grows a switcher, and
// building it from the platform's preferences alone is impossible.
//
// And a developer needs to SEE their translations. Every platform-level way of
// doing that is different and most are awful: adb setprop plus a runtime
// restart on Android, simctl launch arguments on iOS, a browser flag and a
// throwaway profile for the service worker, an environment variable on
// desktop, a header on the server. Five mechanisms, several requiring a reboot
// of something, and none of them available while simply clicking around.
//
// A query parameter is one mechanism that works on all six targets, because it
// never touches the platform.
const (
	OverrideParam  = "lang"
	OverrideCookie = "irgo_lang"
)

// FromOverride is the language this request explicitly asked for.
//
// Safe against anything: language.Parse rejects what is not a well-formed tag,
// and a well-formed tag with no catalog loses to the confidence check in
// Reader. So the value cannot reach <html lang> unless it is both valid BCP 47
// and a language the app actually has — which matters, because it arrives from
// the URL.
func FromOverride(r *http.Request) []language.Tag {
	if r == nil {
		return nil
	}
	if v := r.URL.Query().Get(OverrideParam); v != "" {
		if t, err := language.Parse(v); err == nil && t != language.Und {
			return []language.Tag{t}
		}
	}
	if c, err := r.Cookie(OverrideCookie); err == nil && c.Value != "" {
		if t, err := language.Parse(c.Value); err == nil && t != language.Und {
			return []language.Tag{t}
		}
	}
	return nil
}

// Remember persists an explicit ?lang= choice, so it survives the next click.
//
// Without it the parameter lasts exactly one request: the first navigation
// drops it, and on mobile — where the WebView owns every link — a developer
// cannot get it back without rebuilding. The cookie is what makes the override
// usable rather than a party trick.
//
// Only writes when the parameter is present, so an ordinary request never gets
// a cookie it did not ask for, and a user who has chosen a language keeps it
// until they choose another.
func Remember(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query().Get(OverrideParam)
	if v == "" {
		return
	}
	t, err := language.Parse(v)
	if err != nil || t == language.Und {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:  OverrideCookie,
		Value: t.String(),
		Path:  "/",
		// A year: a language choice is not a session. SameSite=Lax because it
		// must survive a link from anywhere; HttpOnly is deliberately off, so
		// a client-side switcher can read what is currently set.
		MaxAge:   365 * 24 * 60 * 60,
		SameSite: http.SameSiteLaxMode,
	})
}
