// What the <html> element has to say about language, and why it is not
// optional.
//
// A page served in German with lang="en" is not a cosmetic problem:
//
//   - a screen reader pronounces German words with English phonetics, which is
//     the difference between usable and not
//   - the browser offers to translate a page already in the reader's language,
//     and does not offer when it should
//   - hyphenation and line-breaking use the wrong rules
//   - search engines index the content as English
//   - :lang() selectors and form spellcheck both target the wrong language
//
// And dir is worse, because its absence is silent until someone speaks Arabic,
// Hebrew, Persian or Urdu — at which point the layout is simply backwards.
// Nothing in a test suite written in English ever notices.
//
// This is wrong even without translations: irgo scaffolds lang="en" into every
// project, so a monolingual German app claims to be English too.
package i18n

import (
	"context"

	"golang.org/x/text/language"
)

// ctxKey carries the locale actually rendered.
//
// The one that was matched, not the one that was asked for. A visitor who
// requested French and got English must receive lang="en" — saying "fr" over
// English text is worse than saying nothing, because assistive technology
// believes it.
type ctxKey struct{}

// WithTag returns a context carrying the locale a response is rendered in.
func WithTag(ctx context.Context, tag language.Tag) context.Context {
	return context.WithValue(ctx, ctxKey{}, tag)
}

// TagFrom is the locale in this context, and whether one was set.
func TagFrom(ctx context.Context) (language.Tag, bool) {
	t, ok := ctx.Value(ctxKey{}).(language.Tag)
	return t, ok
}

// Lang is the value for the lang attribute: "de-AT", "en".
//
// Falls back to "en" when nothing set a locale, which is what irgo's templates
// hardcoded before this existed — so a project that never adopts i18n renders
// exactly as it did.
func Lang(ctx context.Context) string {
	t, ok := TagFrom(ctx)
	if !ok || t == language.Und {
		return "en"
	}
	return t.String()
}

// Dir is the value for the dir attribute: "rtl" or "ltr".
func Dir(ctx context.Context) string {
	t, ok := TagFrom(ctx)
	if !ok {
		return "ltr"
	}
	return DirOf(t)
}

// DirOf is the writing direction of a tag.
//
// By script rather than by language, because the same language can be written
// either way and the script is what actually decides. Azerbaijani, Kurdish,
// Punjabi and Uzbek all have Arabic-script and Latin-script forms, so a list of
// language codes gets those wrong in one direction or the other.
//
// x/text does not expose direction, so the scripts are listed here. It is a
// short and famously stable list — writing systems are not added often.
func DirOf(t language.Tag) string {
	if t == language.Und {
		return "ltr"
	}
	// Script() infers from the language and region when the tag omits it, so
	// "fa" resolves to Arab without the caller having written "fa-Arab".
	script, _ := t.Script()
	switch script.String() {
	case "Arab", // Arabic, Persian, Urdu, Pashto, Sindhi, Uyghur, Kashmiri
		"Hebr",                         // Hebrew, Yiddish, Ladino
		"Thaa",                         // Dhivehi
		"Nkoo",                         // N'Ko
		"Syrc",                         // Syriac
		"Samr",                         // Samaritan
		"Mand",                         // Mandaic
		"Adlm",                         // Adlam (Fulani)
		"Rohg",                         // Hanifi Rohingya
		"Yezi",                         // Yezidi
		"Mend",                         // Mende Kikakui
		"Cprt",                         // Cypriot
		"Khar",                         // Kharoshthi
		"Phnx",                         // Phoenician
		"Armi", "Prti", "Phli", "Avst", // Imperial Aramaic, Parthian, Pahlavi, Avestan
		"Sarb", "Narb", // Old South/North Arabian
		"Palm", "Nbat", "Hatr", // Palmyrene, Nabataean, Hatran
		"Mani", "Sogd", "Sogo", "Chrs", // Manichaean, Sogdian, Chorasmian
		"Elym", "Orkh", "Hung": // Elymaic, Old Turkic, Old Hungarian
		return "rtl"
	}
	return "ltr"
}
