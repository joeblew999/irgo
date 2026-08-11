package i18n

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFromOverride(t *testing.T) {
	for _, tc := range []struct{ name, url, cookie, want string }{
		{"query wins", "/?lang=de", "", "de"},
		{"regional", "/?lang=pt-BR", "", "pt-BR"},
		{"cookie when no query", "/", "ja", "ja"},
		{"query beats cookie", "/?lang=de", "ja", "de"},
		{"nothing", "/", "", ""},
		// From the URL, so it is attacker-controlled and reaches <html lang>.
		{"garbage is refused", "/?lang=<script>", "", ""},
		{"und is not a language", "/?lang=und", "", ""},
		{"empty", "/?lang=", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tc.url, nil)
			if tc.cookie != "" {
				r.AddCookie(&http.Cookie{Name: OverrideCookie, Value: tc.cookie})
			}
			got := FromOverride(r)
			if tc.want == "" {
				if len(got) != 0 {
					t.Errorf("got %v, want nothing", got)
				}
				return
			}
			if len(got) == 0 || got[0].String() != tc.want {
				t.Errorf("got %v, want %s", got, tc.want)
			}
		})
	}
}

// TestRememberOnlyWhenAsked — an ordinary request must not acquire a cookie
// nobody set, or every visitor is pinned to whatever they first negotiated and
// changing their browser language stops working.
func TestRememberOnlyWhenAsked(t *testing.T) {
	for _, tc := range []struct {
		name, url string
		wantSet   bool
	}{
		{"explicit choice is remembered", "/?lang=de", true},
		{"ordinary request is not", "/", false},
		{"garbage is not", "/?lang=!!!", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			i18nRemember(w, httptest.NewRequest("GET", tc.url, nil))
			got := w.Result().Cookies()
			if tc.wantSet && len(got) == 0 {
				t.Error("no cookie set")
			}
			if !tc.wantSet && len(got) != 0 {
				t.Errorf("unexpected cookie %v", got)
			}
		})
	}
}

func i18nRemember(w http.ResponseWriter, r *http.Request) { Remember(w, r) }

// TestOverrideOutranksEverything is the point of the feature: a user whose
// browser says English and who chose German must get German.
func TestOverrideOutranksEverything(t *testing.T) {
	t.Cleanup(func() { SetPreferred() })
	t.Setenv("LC_ALL", "fr_FR.UTF-8")
	SetPreferredTags("ja")

	r := httptest.NewRequest("GET", "/?lang=de", nil)
	r.Header.Set("Accept-Language", "en-GB,en")
	if got := Preferred(r); len(got) == 0 || got[0].String() != "de" {
		t.Errorf("Preferred = %v, want de first", got)
	}
}
