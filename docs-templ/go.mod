module docs-templ

go 1.25.0

require (
	github.com/a-h/templ v0.3.977
	github.com/alecthomas/chroma/v2 v2.27.0
	github.com/gost-dom/browser v0.12.0
	github.com/gost-dom/browser/scripting/sobekengine v0.0.0-20260621145258-98adcf9f33a5
	github.com/stukennedy/irgo v0.4.0
	github.com/syumai/workers v0.33.0
)

require (
	github.com/CAFxX/httpcompression v0.0.9 // indirect
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/dlclark/regexp2/v2 v2.2.1 // indirect
	github.com/go-chi/chi/v5 v5.2.4 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/google/pprof v0.0.0-20251114195745-4902fdda35c8 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/gost-dom/css v0.1.0 // indirect
	github.com/grafana/sobek v0.0.0-20251113105955-976a34df9c09 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/starfederation/datastar-go v1.1.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/webview/webview_go v0.0.0-20240831120633-6173450d4dd6 // indirect
	golang.org/x/mobile v0.0.0-20260803200217-62cee1672c8e // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/net v0.57.0 // indirect
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

replace github.com/stukennedy/irgo => ../
