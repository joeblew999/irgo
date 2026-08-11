//go:build !(js && wasm)

package i18n

import "golang.org/x/text/language"

// platformPreferred is what the host said first, then the OS environment.
//
// The order matters on mobile and nowhere else. A phone has no LC_ALL, so
// FromEnv is empty there and only SetPreferred has an answer — but a desktop
// build of the same app has the environment and no host declaration. Trying
// both, in that order, is what lets one binary be right on both.
func platformPreferred() []language.Tag {
	if d := Declared(); len(d) > 0 {
		return append(d, FromEnv()...)
	}
	return FromEnv()
}
