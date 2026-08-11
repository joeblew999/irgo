module morpheus

go 1.26.5

require (
	github.com/stukennedy/irgo v0.4.0
	github.com/stukennedy/irgo/pkg/browsertest v0.0.0-00010101000000-000000000000
	github.com/syumai/workers v0.33.0
)

require (
	github.com/CAFxX/httpcompression v0.0.9 // indirect
	github.com/a-h/templ v0.3.1020 // indirect
	github.com/andybalholm/brotli v1.2.2 // indirect
	github.com/deckarep/golang-set/v2 v2.8.0 // indirect
	github.com/go-chi/chi/v5 v5.2.4 // indirect
	github.com/go-jose/go-jose/v3 v3.0.5 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/mxschmitt/playwright-go v0.6100.0 // indirect
	github.com/starfederation/datastar-go v1.1.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/webview/webview_go v0.0.0-20240831120633-6173450d4dd6 // indirect
	golang.org/x/mobile v0.0.0-20260803200217-62cee1672c8e // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
)

tool (
	// The CLI is a tool dependency, so `go tool irgo` builds the exact version
	// this module requires — including through the replace below when you track a
	// fork. That makes go.mod the single pin: nothing has to be installed first,
	// put on PATH, or kept in step by hand.
	github.com/stukennedy/irgo/cmd/irgo
	golang.org/x/mobile/cmd/gobind
)

replace github.com/stukennedy/irgo => ../..

replace github.com/stukennedy/irgo/pkg/browsertest => ../../pkg/browsertest
