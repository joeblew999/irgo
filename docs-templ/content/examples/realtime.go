// Package examples holds the code the documentation shows.
//
// The realtime page used to document a datastar.NewHub with HandleSSE,
// BroadcastToRoom and SendToSession. None of it existed. It read perfectly,
// and every reader who tried it hit a compile error on the first line.
//
// So the samples live here, in a package that `go build ./...` compiles, and
// the pages read them back out by name — see content/examples.go. A sample
// that stops compiling is a build failure, not a support question.
//
// The regions between doc:start and doc:end markers are what the pages show.
// Everything outside them is scaffolding a real app would already have.
package examples

import (
	"time"

	"github.com/stukennedy/irgo/pkg/router"
	"github.com/stukennedy/irgo/pkg/websocket"
)

// Scaffolding: in your app these come from your own packages.
var (
	hub      = websocket.NewHub()
	renderer = struct {
		Render func(string) (string, error)
	}{Render: func(s string) (string, error) { return s, nil }}
)

func renderMessage(user, message string, at time.Time) (string, error) {
	// Your templ component: renderer.Render(templates.ChatMessage(...))
	return renderer.Render(user + ": " + message + " at " + at.Format("3:04 PM"))
}

func renderNotification(message string, at time.Time) (string, error) {
	return renderer.Render(message)
}

func currentUser(ctx *router.Context) string { return ctx.Header("X-User") }

// doc:start hub-setup
// The hub holds the open connections. Register a handler per URL pattern to
// receive what clients send.
func NewHub() *websocket.Hub {
	hub := websocket.NewHub()

	hub.HandleFunc("/chat/{roomID}",
		func(s *websocket.Session, req *websocket.Request) (*websocket.Envelope, error) {
			// Values carries the form data the client sent.
			html, err := renderMessage(
				req.GetStringValue("user"),
				req.GetStringValue("message"),
				time.Now(),
			)
			if err != nil {
				return nil, err
			}
			return websocket.HTMLEnvelope("#messages", html), nil
		})

	return hub
}

// doc:end

// doc:start hub-push
// Push from anywhere in your code — the hub is safe to call from any
// goroutine.
func pushExamples(statsHTML, msgHTML, noticeHTML, sessionID string) error {
	// Everyone connected
	hub.BroadcastHTML("#live-stats", statsHTML)

	// Everyone on one URL
	hub.BroadcastToURL("/chat/general",
		websocket.HTMLEnvelope("#messages", msgHTML))

	// One session
	return hub.SendHTML(sessionID, "#notification-list", noticeHTML)
}

// doc:end

// doc:start chat-send
// SendMessage broadcasts to the room, then clears this client's input. The
// message reaches every other client over the hub, not in this response.
func SendMessage(ctx *router.Context) error {
	roomID := ctx.Param("roomID")

	var signals struct {
		Message string `json:"message"`
	}
	ctx.ReadSignals(&signals)

	html, err := renderMessage(currentUser(ctx), signals.Message, time.Now())
	if err != nil {
		return err
	}

	hub.BroadcastToURL("/chat/"+roomID,
		websocket.HTMLEnvelope("#messages", html))

	return ctx.SSE().PatchSignals(map[string]any{"message": ""})
}

// doc:end

// doc:start notify
// NotifyUser pushes to one session, from anywhere in your code.
func NotifyUser(sessionID string, message string) error {
	html, err := renderNotification(message, time.Now())
	if err != nil {
		return err
	}
	return hub.SendHTML(sessionID, "#notification-list", html)
}

// doc:end
