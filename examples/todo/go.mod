module todo

go 1.25.0

require (
	github.com/a-h/templ v0.3.977
	github.com/starfederation/datastar-go v1.1.0
	github.com/stukennedy/irgo v0.4.0
)

require (
	github.com/CAFxX/httpcompression v0.0.9 // indirect
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/go-chi/chi/v5 v5.2.4 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/webview/webview_go v0.0.0-20240831120633-6173450d4dd6 // indirect
)

// The CLI is a tool dependency, so `go tool irgo` builds the exact version
// this module requires — including through the replace below when you track a
// fork. That makes go.mod the single pin: nothing has to be installed first,
// put on PATH, or kept in step by hand.
tool github.com/stukennedy/irgo/cmd/irgo

replace github.com/stukennedy/irgo => ../..
