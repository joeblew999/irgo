//go:build js && wasm

// Running the whole app in the browser tab, with no server.
//
// irgo already answers requests without a network on iOS and Android: the
// handler runs in-process and a bridge patches fetch() so the WebView's
// requests reach it. A browser tab is the same shape — a JavaScript runtime
// that speaks fetch — so the same idea needs no new architecture, only the
// glue for this particular host.
//
// What makes it work rather than nearly work is pkg/adapter's ResponseStream:
// chunks are delivered as the handler flushes rather than buffered until it
// returns. Datastar is server-sent events, so a buffered bridge would leave
// every page loading forever, waiting for a response that only completes when
// the connection closes. Streaming maps onto a ReadableStream exactly.
//
// The result is an app with no origin to be offline from: the handler, the
// templates and the embedded static files are all in the wasm binary.
package browser

import (
	"context"
	"net/http"
	"strings"
	"syscall/js"

	"github.com/stukennedy/irgo/pkg/adapter"
	"github.com/stukennedy/irgo/pkg/core"
)

// Serve patches window.fetch so every request this page makes is answered by
// handler, in this tab, and blocks forever.
//
// Blocking is required rather than tidy: a wasm main that returns takes the
// Go runtime with it, and the patched fetch stops answering.
func Serve(handler http.Handler) {
	a := adapter.NewHTTPAdapter(handler)

	original := js.Global().Get("fetch")
	js.Global().Set("fetch", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			// Nothing to route. Hand it back to the real fetch, which will
			// produce the same TypeError the page would have seen anyway.
			return original.Invoke()
		}
		return newPromise(func(resolve, reject js.Value) {
			go func() {
				defer func() {
					// A panic here would otherwise kill the tab's Go runtime
					// and every later request with it.
					if r := recover(); r != nil {
						reject.Invoke(errorValue("irgo: request panicked"))
					}
				}()
				resp, err := dispatch(a, args)
				if err != nil {
					reject.Invoke(errorValue(err.Error()))
					return
				}
				resolve.Invoke(resp)
			}()
		})
	}))

	// Tell the page it can start: index.html waits for this rather than
	// racing the wasm instantiation.
	if ready := js.Global().Get("__irgoReady"); ready.Type() == js.TypeFunction {
		ready.Invoke()
	}

	select {}
}

// ServeWorker answers every request the page makes from handler, by running
// inside a service worker.
//
// This is the mode that actually gets an irgo app running with no network.
// Patching fetch in the page only covers what the page's own JavaScript asks
// for — which is Datastar's requests, and nothing else. A stylesheet in a
// <link>, a module in a <script src>, an image: the browser fetches those
// through its own loader, which no amount of patching reaches. They go to the
// network, and with no server there they 404.
//
// A service worker sits in front of all of it. The same router that answers
// the Worker, the WebView and the desktop shell answers here, so the embedded
// static files are served from the wasm binary and the app has no origin left
// to be offline from.
//
// Requires a secure context: https, or localhost, which browsers treat as
// secure so that development works without certificates.
func ServeWorker(handler http.Handler) {
	a := adapter.NewHTTPAdapter(handler)

	// Exported for sw.js to call, rather than adding a fetch listener here.
	//
	// A service worker only takes control of requests if its fetch listener is
	// registered during the initial evaluation of the worker script. Wasm
	// instantiation is asynchronous, so anything Go registers is already too
	// late — the worker installs, controls nothing, and every request goes to
	// the network as though none of this existed. So the listener is plain
	// JavaScript, registered synchronously, and it waits for this.
	js.Global().Set("__irgoFetch", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return newPromise(func(_, reject js.Value) {
				reject.Invoke(errorValue("irgo: no request"))
			})
		}
		request := args[0]
		return newPromise(func(resolve, reject js.Value) {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						reject.Invoke(errorValue("irgo: request panicked"))
					}
				}()
				resp, err := dispatch(a, []js.Value{request})
				if err != nil {
					reject.Invoke(errorValue(err.Error()))
					return
				}
				resolve.Invoke(resp)
			}()
		})
	}))

	// Read by sw.js to know the router is answering. Set last, so it is never
	// true before __irgoFetch exists.
	js.Global().Set("__irgoReady", true)

	select {}
}

// dispatch turns fetch's arguments into a Response backed by the handler.
func dispatch(a *adapter.HTTPAdapter, args []js.Value) (js.Value, error) {
	method, url, headers, body := requestFrom(args)

	// A Request object carries its body as a stream rather than a string, so
	// it has to be read before the handler runs. Without this every Datastar
	// action arrives with an empty body and the handler sees no signals.
	if len(body) == 0 && args[0].Type() != js.TypeString && methodTakesBody(method) {
		if text, err := await(args[0].Call("text")); err == nil && text.Type() == js.TypeString {
			body = []byte(text.String())
		}
	}

	req := core.NewRequest(method, url)
	req.SetHeaders(headers)
	if len(body) > 0 {
		req.Body = body
	}

	// The stream is created before the handler runs, so the Response can be
	// returned as soon as headers arrive rather than when the body finishes.
	// For SSE that difference is the whole point: the body never finishes.
	ctx, cancel := context.WithCancel(context.Background())
	s := &jsStream{
		started: make(chan struct{}),
		ready:   make(chan struct{}),
		ctx:     ctx,
		cancel:  cancel,
	}
	s.stream = js.Global().Get("ReadableStream").New(map[string]any{
		"start": js.FuncOf(func(_ js.Value, a []js.Value) any {
			s.controller = a[0]
			close(s.started)
			return nil
		}),
		"cancel": js.FuncOf(func(_ js.Value, _ []js.Value) any {
			// The page navigated away or Datastar closed the connection.
			// Cancelling the context ends the handler, which is what a real
			// disconnect does.
			s.cancel()
			return nil
		}),
	})

	go a.HandleRequestStream(s.ctx, req, s)

	<-s.ready
	if s.err != "" {
		return js.Undefined(), &dispatchError{s.err}
	}

	init := map[string]any{
		"status":  s.status,
		"headers": s.jsHeaders(),
	}
	return js.Global().Get("Response").New(s.stream, init), nil
}

type dispatchError struct{ msg string }

func (e *dispatchError) Error() string { return e.msg }

// jsStream adapts adapter.ResponseStream onto a ReadableStream controller.
type jsStream struct {
	stream     js.Value
	controller js.Value
	started    chan struct{}

	ctx    context.Context
	cancel context.CancelFunc

	status  int
	headers http.Header
	err     string

	ready     chan struct{}
	readyOnce bool
}

func (s *jsStream) OnResponse(status int, headersJSON string) {
	s.status = status
	if status == 0 {
		s.status = http.StatusOK
	}
	// core.DecodeHeaders rather than unmarshalling here: EncodeHeaders writes a
	// single-valued header as a bare string and a repeated one as an array, so
	// decoding into map[string][]string fails on the common case and silently
	// drops every header. The visible symptom was a module script rejected for
	// having no MIME type, which says nothing about headers at all.
	s.headers = core.DecodeHeaders(headersJSON)
	s.signalReady()
}

func (s *jsStream) OnChunk(chunk []byte) {
	s.signalReady()
	<-s.started

	buf := js.Global().Get("Uint8Array").New(len(chunk))
	js.CopyBytesToJS(buf, chunk)
	s.controller.Call("enqueue", buf)
}

func (s *jsStream) OnComplete(errorMessage string) {
	if errorMessage != "" && !s.readyOnce {
		s.err = errorMessage
	}
	s.signalReady()
	<-s.started
	s.controller.Call("close")
}

// signalReady releases the caller waiting to construct the Response. Called on
// whichever callback happens first: a handler that writes nothing still has to
// produce a Response.
func (s *jsStream) signalReady() {
	if s.readyOnce {
		return
	}
	s.readyOnce = true
	if s.status == 0 {
		s.status = http.StatusOK
	}
	close(s.ready)
}

func (s *jsStream) jsHeaders() map[string]any {
	out := map[string]any{}
	for k, v := range s.headers {
		out[k] = strings.Join(v, ", ")
	}
	return out
}

// requestFrom reads fetch(input, init) into its parts.
func requestFrom(args []js.Value) (method, url string, headers map[string]string, body []byte) {
	method, headers = "GET", map[string]string{}

	input := args[0]
	if input.Type() == js.TypeString {
		url = input.String()
	} else {
		// A Request object.
		url = input.Get("url").String()
		if m := input.Get("method"); m.Type() == js.TypeString {
			method = m.String()
		}
		readHeaders(input.Get("headers"), headers)
	}

	if len(args) > 1 && args[1].Type() == js.TypeObject {
		init := args[1]
		if m := init.Get("method"); m.Type() == js.TypeString {
			method = m.String()
		}
		readHeaders(init.Get("headers"), headers)
		if b := init.Get("body"); b.Type() == js.TypeString {
			body = []byte(b.String())
		}
	}
	return method, url, headers, body
}

// readHeaders copies a Headers object or a plain object into dst.
func readHeaders(h js.Value, dst map[string]string) {
	if h.IsUndefined() || h.IsNull() {
		return
	}
	// Headers has forEach; a plain object does not.
	if fe := h.Get("forEach"); fe.Type() == js.TypeFunction {
		h.Call("forEach", js.FuncOf(func(_ js.Value, a []js.Value) any {
			dst[a[1].String()] = a[0].String()
			return nil
		}))
		return
	}
	keys := js.Global().Get("Object").Call("keys", h)
	for i := 0; i < keys.Length(); i++ {
		k := keys.Index(i).String()
		dst[k] = h.Get(k).String()
	}
}

// methodTakesBody reports whether reading one is worth the round trip.
func methodTakesBody(method string) bool {
	switch strings.ToUpper(method) {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	}
	return false
}

// await blocks the calling goroutine until a JS promise settles.
//
// Safe only off the main goroutine: blocking that one deadlocks the runtime,
// because the promise can never settle while Go holds the thread. Every caller
// here is already on a goroutine spawned per request.
func await(p js.Value) (js.Value, error) {
	type result struct {
		v   js.Value
		err error
	}
	ch := make(chan result, 1)

	onOK := js.FuncOf(func(_ js.Value, a []js.Value) any {
		var v js.Value
		if len(a) > 0 {
			v = a[0]
		}
		ch <- result{v: v}
		return nil
	})
	onErr := js.FuncOf(func(_ js.Value, a []js.Value) any {
		msg := "promise rejected"
		if len(a) > 0 {
			msg = a[0].Call("toString").String()
		}
		ch <- result{err: &dispatchError{msg}}
		return nil
	})
	defer onOK.Release()
	defer onErr.Release()

	p.Call("then", onOK, onErr)
	r := <-ch
	return r.v, r.err
}

func newPromise(fn func(resolve, reject js.Value)) js.Value {
	return js.Global().Get("Promise").New(js.FuncOf(func(_ js.Value, a []js.Value) any {
		fn(a[0], a[1])
		return nil
	}))
}

func errorValue(msg string) js.Value {
	return js.Global().Get("Error").New(msg)
}
