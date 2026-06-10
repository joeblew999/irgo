import type { FC } from 'hono/jsx'
import { DocsLayout } from '../../components/DocsLayout'
import { CodeBlock } from '../../components/CodeBlock'
import { Callout } from '../../components/FeatureCard'

export const WebSocketsPage: FC = () => {
  return (
    <DocsLayout title="Real-Time Updates" currentPath="/docs/websockets">
      <h1>Real-Time Updates</h1>

      <p>
        irgo with Datastar provides real-time updates via Server-Sent Events (SSE).
        This enables live dashboards, notifications, and collaborative features.
      </p>

      <h2>SSE with Datastar</h2>

      <p>
        Datastar uses SSE natively for all interactions. Every Datastar request
        receives an SSE stream response, making real-time updates natural and simple.
      </p>

      <CodeBlock language="templ">
{`templ LiveDashboard() {
    <div data-init="@get('/api/dashboard')" data-signals="{stats: {}}">
        <div id="live-stats">
            Loading...
        </div>
    </div>
}`}
      </CodeBlock>

      <h2>Polling for Updates</h2>

      <p>
        Use Datastar's interval modifier to poll for updates:
      </p>

      <CodeBlock language="templ">
{`templ StatusIndicator() {
    <div
        data-init="@get('/api/status')"
        data-on:load__interval.5s="@get('/api/status')"
    >
        <div id="status">Checking...</div>
    </div>
}`}
      </CodeBlock>

      <CodeBlock language="go">
{`func GetStatus(ctx *router.Context) error {
    status := checkSystemStatus()
    return ctx.SSE().PatchTempl(templates.StatusBadge(status))
}`}
      </CodeBlock>

      <h2>Server Push Pattern</h2>

      <p>
        For true server push (where the server initiates updates), use a long-running
        SSE connection with the Datastar hub:
      </p>

      <CodeBlock language="go">
{`import "github.com/stukennedy/irgo/pkg/datastar"

// Create hub
hub := datastar.NewHub()
go hub.Run()

// Handle SSE connections
r.GET("/api/events", func(ctx *router.Context) error {
    return hub.HandleSSE(ctx.ResponseWriter, ctx.Request,
        datastar.WithRoom("dashboard"),
    )
})

// Push updates from anywhere in your code
func updateDashboard(stats Stats) {
    html := renderer.RenderToString(templates.DashboardStats(stats))
    hub.BroadcastToRoom("dashboard", datastar.PatchElements(html))
}`}
      </CodeBlock>

      <Callout type="info">
        <p>
          Datastar's SSE approach is more efficient than WebSockets for most use cases
          since it uses standard HTTP/2 connections that work through proxies and CDNs.
        </p>
      </Callout>

      <h2>Chat Example</h2>

      <CodeBlock title="templates/chat.templ" language="templ">
{`templ ChatRoom(roomID string) {
    <div
        class="chat-room"
        data-signals="{message: ''}"
        data-init={ "@get('/chat/" + roomID + "/connect')" }
    >
        <div id="messages" class="messages">
            // Messages will be inserted here
        </div>
        <form data-on:submit__prevent={ "@post('/chat/" + roomID + "/send')" }>
            <input
                type="text"
                data-bind:message
                placeholder="Type a message..."
            />
            <button type="submit">Send</button>
        </form>
    </div>
}

templ ChatMessage(user string, message string, timestamp time.Time) {
    <div id="messages" class="message">
        <strong>{ user }</strong>
        <span>{ message }</span>
        <time>{ timestamp.Format("3:04 PM") }</time>
    </div>
}`}
      </CodeBlock>

      <CodeBlock title="handlers/chat.go" language="go">
{`func ConnectChat(ctx *router.Context) error {
    roomID := ctx.Param("roomID")
    return hub.HandleSSE(ctx.ResponseWriter, ctx.Request,
        datastar.WithRoom(roomID),
    )
}

func SendMessage(ctx *router.Context) error {
    roomID := ctx.Param("roomID")

    var signals struct {
        Message string \`json:"message"\`
    }
    ctx.ReadSignals(&signals)

    // Broadcast to all clients in the room
    html := renderer.RenderToString(templates.ChatMessage(
        ctx.UserID(),
        signals.Message,
        time.Now(),
    ))
    hub.BroadcastToRoom(roomID, datastar.PatchElements(html))

    // Clear the message input
    return ctx.SSE().PatchSignals(map[string]any{"message": ""})
}`}
      </CodeBlock>

      <h2>Live Notifications</h2>

      <CodeBlock language="templ">
{`templ NotificationListener() {
    <div
        data-signals="{unreadCount: 0}"
        data-init="@get('/api/notifications/connect')"
    >
        <div id="notification-badge" data-show="$unreadCount > 0">
            <span data-text="$unreadCount"></span>
        </div>
        <div id="notification-list"></div>
    </div>
}

templ Notification(n Notification) {
    <div id="notification-list" class="notification">
        <p>{ n.Message }</p>
        <time>{ n.CreatedAt.Format("3:04 PM") }</time>
    </div>
}`}
      </CodeBlock>

      <CodeBlock language="go">
{`// Send notification from anywhere in your code
func notifyUser(userID string, message string) {
    n := Notification{
        Message:   message,
        CreatedAt: time.Now(),
    }

    html := renderer.RenderToString(templates.Notification(n))
    count := getUnreadCount(userID)

    hub.SendToSession(userID,
        datastar.PatchElements(html),
        datastar.PatchSignals(map[string]any{"unreadCount": count}),
    )
}`}
      </CodeBlock>

      <h2>Next Steps</h2>

      <ul>
        <li><a href="/docs/datastar">Datastar Integration</a> - Core Datastar patterns</li>
        <li><a href="/docs/examples">Examples</a> - Complete examples</li>
      </ul>
    </DocsLayout>
  )
}
