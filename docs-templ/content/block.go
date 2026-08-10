package content

import "strings"

// Kind is what a block of a page is.
type Kind string

const (
	KindH2    Kind = "h2"
	KindH3    Kind = "h3"
	KindPara  Kind = "p"
	KindCode  Kind = "code"
	KindList  Kind = "list"
	KindTable Kind = "table"
	KindNote  Kind = "note"
)

// Block is one piece of a page.
//
// One struct with optional fields rather than an interface per kind: a page is
// written as a list of literals, and the constructors below are what a page
// actually uses. Nothing outside this file builds a Block field by field.
type Block struct {
	Kind Kind

	// Prose and headings. Text carries inline markup — see Inline.
	Text string

	// Code.
	Lang string // language for highlighting; "" is plain text
	File string // caption above the sample — usually the file it belongs in

	// List.
	Ordered bool
	Items   []string // inline markup

	// Table.
	Head []string
	Rows [][]string // inline markup

	// Note.
	Tone  string   // "info", "tip" or "warning"
	Title string   // optional heading
	Paras []string // inline markup
}

// H2 is a section heading.
func H2(text string) Block { return Block{Kind: KindH2, Text: text} }

// H3 is a subsection heading.
func H3(text string) Block { return Block{Kind: KindH3, Text: text} }

// P is a paragraph. Its text carries inline markup.
func P(text string) Block { return Block{Kind: KindPara, Text: text} }

// Code is a code sample with no caption. Lang names the language for
// highlighting — see highlight.go for the ones that mean anything.
func Code(lang, src string) Block {
	return Block{Kind: KindCode, Lang: lang, Text: src}
}

// CodeFile is a code sample under a caption, almost always the file it belongs
// in. The old site showed that caption, and it is often the only thing telling
// a reader where the code goes.
func CodeFile(lang, file, src string) Block {
	return Block{Kind: KindCode, Lang: lang, File: file, Text: src}
}

// Bullets is an unordered list. Items carry inline markup.
func Bullets(items ...string) Block {
	return Block{Kind: KindList, Items: items}
}

// Steps is a numbered list, for sequences where the order is the point.
func Steps(items ...string) Block {
	return Block{Kind: KindList, Ordered: true, Items: items}
}

// Table is a comparison table. Cells carry inline markup.
func Table(head []string, rows ...[]string) Block {
	return Block{Kind: KindTable, Head: head, Rows: rows}
}

// Note is a called-out aside. Tone is "info", "tip" or "warning"; title may be
// empty.
func Note(tone, title string, paras ...string) Block {
	return Block{Kind: KindNote, Tone: tone, Title: title, Paras: paras}
}

// SpanKind is what a run of inline text is.
type SpanKind int

const (
	SpanText SpanKind = iota
	SpanCode
	SpanStrong
	SpanLink
)

// Span is a run of inline text with one kind of emphasis.
type Span struct {
	Kind SpanKind
	Text string
	URL  string // SpanLink only
}

// Inline splits prose into spans, understanding a deliberately small markup:
//
//	`code`          a literal
//	**strong**      emphasis
//	[label](url)    a link
//
// Small because the alternative is prose that carries HTML, and prose carrying
// HTML is prose that can inject it. Everything here becomes an element with a
// text child, so the renderer never has to trust the content.
//
// Anything unterminated is left alone rather than swallowed, so a stray
// asterisk or bracket in ordinary prose survives as itself.
func Inline(s string) []Span {
	var spans []Span
	var lit strings.Builder

	flush := func() {
		if lit.Len() > 0 {
			spans = append(spans, Span{Kind: SpanText, Text: lit.String()})
			lit.Reset()
		}
	}

	for i := 0; i < len(s); {
		switch {
		case s[i] == '`':
			if end := strings.IndexByte(s[i+1:], '`'); end > 0 {
				flush()
				spans = append(spans, Span{Kind: SpanCode, Text: s[i+1 : i+1+end]})
				i += end + 2
				continue
			}

		case strings.HasPrefix(s[i:], "**"):
			if end := strings.Index(s[i+2:], "**"); end > 0 {
				flush()
				spans = append(spans, Span{Kind: SpanStrong, Text: s[i+2 : i+2+end]})
				i += end + 4
				continue
			}

		case s[i] == '[':
			if label := strings.IndexByte(s[i:], ']'); label > 0 &&
				i+label+1 < len(s) && s[i+label+1] == '(' {
				if url := strings.IndexByte(s[i+label+1:], ')'); url > 0 {
					flush()
					spans = append(spans, Span{
						Kind: SpanLink,
						Text: s[i+1 : i+label],
						URL:  s[i+label+2 : i+label+1+url],
					})
					i += label + url + 2
					continue
				}
			}
		}

		lit.WriteByte(s[i])
		i++
	}

	flush()
	return spans
}
