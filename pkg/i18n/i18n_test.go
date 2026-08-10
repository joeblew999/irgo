package i18n

import (
	"net/http"
	"testing"

	"golang.org/x/text/language"
)

func TestFromRequest(t *testing.T) {
	for _, tc := range []struct {
		name, header string
		want         []string
	}{
		{"single", "de", []string{"de"}},
		{"ordered by q", "de-CH,de;q=0.9,en;q=0.5", []string{"de-CH", "de", "en"}},
		{"q reorders", "en;q=0.5,de;q=0.9", []string{"de", "en"}},
		{"absent", "", nil},
		// Attacker-controlled on every request. The answer is "no preference",
		// not a panic and not a 500.
		{"nonsense", "!!!!", nil},
		// x/text is all-or-nothing on the whole header, so these exercise the
		// per-entry salvage rather than the library.
		{"garbage after valid", "de,!!!", []string{"de"}},
		{"garbage before valid", "!!!,de", []string{"de"}},
		{"salvage keeps q order", "en;q=0.3,!!!,de;q=0.9", []string{"de", "en"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := http.NewRequest("GET", "/", nil)
			if tc.header != "" {
				r.Header.Set("Accept-Language", tc.header)
			}
			got := FromRequest(r)
			if len(got) < len(tc.want) {
				t.Fatalf("Accept-Language %q: got %v, want at least %v", tc.header, got, tc.want)
			}
			for i, w := range tc.want {
				if got[i].String() != w {
					t.Errorf("Accept-Language %q: position %d is %s, want %s",
						tc.header, i, got[i], w)
				}
			}
		})
	}
}

// TestFromRequestNilIsNotAPanic — desktop and mobile call the same handlers
// through paths that do not always carry a request.
func TestFromRequestNilIsNotAPanic(t *testing.T) {
	if got := FromRequest(nil); got != nil {
		t.Errorf("FromRequest(nil) = %v, want nil", got)
	}
}

func TestFromEnv(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  map[string]string
		want []string
	}{
		// The POSIX shapes. Underscore and encoding suffix both have to go, or
		// a correctly configured desktop silently falls back to the source
		// language — the failure this function exists to prevent.
		{"plain", map[string]string{"LANG": "de"}, []string{"de"}},
		{"region", map[string]string{"LANG": "de_AT"}, []string{"de-AT"}},
		{"encoding", map[string]string{"LANG": "de_AT.UTF-8"}, []string{"de-AT"}},
		{"modifier", map[string]string{"LANG": "de_DE.UTF-8@euro"}, []string{"de-DE"}},

		// LC_ALL wins over everything, LC_MESSAGES over LANG. That is the
		// standard's order, not a preference.
		{"LC_ALL wins", map[string]string{
			"LC_ALL": "fr_FR.UTF-8", "LC_MESSAGES": "de_DE", "LANG": "en_US",
		}, []string{"fr-FR"}},
		{"LC_MESSAGES over LANG", map[string]string{
			"LC_MESSAGES": "de_DE", "LANG": "en_US",
		}, []string{"de-DE"}},

		// C and POSIX mean "no locale". Parsing them yields a tag that looks
		// real and matches no catalog.
		{"C is not a language", map[string]string{"LANG": "C"}, nil},
		{"POSIX is not a language", map[string]string{"LANG": "POSIX"}, nil},
		{"unset", map[string]string{}, nil},

		// The GNU extension: the one place a desktop expresses a ranked list.
		{"LANGUAGE list", map[string]string{"LANGUAGE": "de:en:fr"},
			[]string{"de", "en", "fr"}},
		{"LANG then LANGUAGE", map[string]string{"LANG": "de_AT", "LANGUAGE": "de:en"},
			[]string{"de-AT", "de", "en"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG", "LANGUAGE"} {
				t.Setenv(k, tc.env[k])
			}
			got := FromEnv()
			if len(got) != len(tc.want) {
				t.Fatalf("env %v: got %v, want %v", tc.env, got, tc.want)
			}
			for i, w := range tc.want {
				if got[i].String() != w {
					t.Errorf("env %v: position %d is %s, want %s", tc.env, i, got[i], w)
				}
			}
		})
	}
}

// TestPreferredUsesBothSources pins the reason Preferred exists.
//
// A desktop irgo app serves its own UI to a local webview, and that webview
// may send no Accept-Language at all. Request-only would give an English UI on
// a German desktop; environment-only would ignore a real browser user. Neither
// source alone is enough.
func TestPreferredUsesBothSources(t *testing.T) {
	t.Setenv("LC_ALL", "de_DE.UTF-8")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "")
	t.Setenv("LANGUAGE", "")

	t.Run("no header falls back to the machine", func(t *testing.T) {
		r, _ := http.NewRequest("GET", "/", nil)
		got := Preferred(r)
		if len(got) == 0 || got[0] != language.MustParse("de-DE") {
			t.Errorf("got %v, want de-DE first", got)
		}
	})

	t.Run("a header outranks the machine", func(t *testing.T) {
		r, _ := http.NewRequest("GET", "/", nil)
		r.Header.Set("Accept-Language", "fr")
		got := Preferred(r)
		if len(got) == 0 || got[0].String() != "fr" {
			t.Errorf("got %v, want fr first", got)
		}
		// ...but the machine's locale is still in the list, so a French
		// speaker on a German desktop with no fr catalog gets German rather
		// than the source language.
		if len(got) < 2 {
			t.Errorf("got %v, want the environment retained as a fallback", got)
		}
	})
}
