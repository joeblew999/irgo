//go:build !(js && wasm)

package i18n

import "golang.org/x/text/language"

// platformPreferred is the OS locale everywhere except the browser, which has
// no environment and answers through navigator instead.
func platformPreferred() []language.Tag { return FromEnv() }
