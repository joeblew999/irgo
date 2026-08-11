package i18n

import (
	"context"
	"testing"

	"golang.org/x/text/language"
)

func TestDirOf(t *testing.T) {
	for _, tc := range []struct{ tag, want string }{
		{"en", "ltr"}, {"de", "ltr"}, {"ja", "ltr"}, {"ru", "ltr"},

		// The scripts that made this worth writing. Each renders backwards
		// without a dir attribute, and nothing in an English test suite ever
		// notices.
		{"ar", "rtl"}, {"he", "rtl"}, {"fa", "rtl"}, {"ur", "rtl"},
		{"ps", "rtl"}, {"sd", "rtl"}, {"dv", "rtl"}, {"yi", "rtl"},
		{"ar-EG", "rtl"}, {"he-IL", "rtl"},

		// Why this goes by script and not by language code. The same language
		// is written both ways, so a list of languages is wrong for one of
		// these however it is written.
		{"az-Arab", "rtl"}, {"az-Latn", "ltr"},
		{"ku-Arab", "rtl"}, {"ku-Latn", "ltr"},
		{"pa-Arab", "rtl"}, {"pa-Guru", "ltr"},
		{"uz-Arab", "rtl"}, {"uz-Latn", "ltr"},
	} {
		t.Run(tc.tag, func(t *testing.T) {
			if got := DirOf(language.MustParse(tc.tag)); got != tc.want {
				t.Errorf("DirOf(%s) = %s, want %s", tc.tag, got, tc.want)
			}
		})
	}
}

func TestLangAndDirFromContext(t *testing.T) {
	t.Run("nothing set renders as it always did", func(t *testing.T) {
		ctx := context.Background()
		if got := Lang(ctx); got != "en" {
			t.Errorf("Lang = %q, want en — a project without i18n must not change", got)
		}
		if got := Dir(ctx); got != "ltr" {
			t.Errorf("Dir = %q, want ltr", got)
		}
	})

	t.Run("the rendered locale is reported", func(t *testing.T) {
		ctx := WithTag(context.Background(), language.MustParse("de-AT"))
		if got := Lang(ctx); got != "de-AT" {
			t.Errorf("Lang = %q, want de-AT", got)
		}
		if got := Dir(ctx); got != "ltr" {
			t.Errorf("Dir = %q, want ltr", got)
		}
	})

	t.Run("rtl", func(t *testing.T) {
		ctx := WithTag(context.Background(), language.MustParse("ar"))
		if got := Dir(ctx); got != "rtl" {
			t.Errorf("Dir = %q, want rtl", got)
		}
	})

	// Und is what an unset or unparseable locale becomes, and "und" is a real
	// BCP 47 value that assistive technology would try to honour.
	t.Run("und is not a language", func(t *testing.T) {
		ctx := WithTag(context.Background(), language.Und)
		if got := Lang(ctx); got != "en" {
			t.Errorf("Lang = %q, want en", got)
		}
	})
}
