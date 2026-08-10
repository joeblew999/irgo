package templates

import (
	"github.com/a-h/templ"

	"docs-templ/content"
)

// highlight renders one code sample.
//
// The markup is precomputed — see content/highlight.go for why. Anything
// without an entry (a diagram, a language we do not colour, a sample edited
// since the last regeneration) falls back to escaped plain text, so a stale
// generated file costs colour rather than the example itself.
func highlight(lang, src string) templ.Component {
	if html, ok := content.Highlighted(lang, src); ok {
		// chroma escapes the source it was given, so this is markup this
		// module produced from text that is already escaped.
		return templ.Raw(html)
	}
	return templ.Raw(templ.EscapeString(src))
}
