package content

// Code samples that appear outside the documentation pages.
//
// They live here rather than inline in the templates so that one thing walks
// every sample on the site: the highlighting generator reads Pages and Samples,
// and a sample added to a template without being added here would render
// uncoloured with nothing to say so.

// Sample is a standalone code sample.
type Sample struct {
	Lang string
	File string
	Text string
}

// Samples are keyed by where they appear.
var Samples = map[string]Sample{
	"home-handler": {
		Lang: "go",
		File: "handlers/todos.go",
		Text: `func CreateTodo(ctx *router.Context) error {
    var signals struct {
        Title string ` + "`" + `json:"title"` + "`" + `
    }
    ctx.ReadSignals(&signals)

    if signals.Title == "" {
        return ctx.SSE().PatchTempl(
            templates.Error("Title required"))
    }

    todo := db.CreateTodo(signals.Title)

    sse := ctx.SSE()
    sse.PatchTempl(templates.TodoItem(todo))
    sse.PatchSignals(map[string]any{"title": ""})
    return nil
}`,
	},

	"home-template": {
		Lang: "templ",
		File: "templates/todos.templ",
		Text: `templ TodoItem(todo Todo) {
    <div id={ "todo-" + todo.ID } class="todo-item">
        <input
            type="checkbox"
            checked?={ todo.Done }
            data-on:click={ "@patch('/todos/" + todo.ID + "')" }
        />
        <span class={ templ.KV("done", todo.Done) }>
            { todo.Title }
        </span>
        <button
            data-on:click={ "@delete('/todos/" + todo.ID + "')" }
        >x</button>
    </div>
}`,
	},

	// The demo page's counter is the real handler behind it, not an
	// illustration of one — see handlers/demo.go.
	"demo-handler": {
		Lang: "go",
		Text: `func Increment(ctx *router.Context) error {
    var signals struct {
        Count int ` + "`" + `json:"count"` + "`" + `
    }
    ctx.ReadSignals(&signals)

    count := signals.Count + 1

    sse := ctx.SSE()
    sse.PatchTempl(templates.DemoCounter(count))
    return sse.PatchSignals(map[string]any{"count": count})
}`,
	},

	"demo-template": {
		Lang: "templ",
		Text: `templ DemoCounter(count int) {
    <div id="counter-value">
        <span class="counter-demo__number">
            { strconv.Itoa(count) }
        </span>
    </div>
}`,
	},

	"demo-signals": {
		Lang: "templ",
		Text: `<div data-signals="{name: '', greeting: ''}">
    <input data-bind:name />
    <button data-on:click="@post('/greet')">
        Say Hello
    </button>
    <p data-text="$greeting"></p>
</div>`,
	},

	"demo-realtime": {
		Lang: "go",
		Text: `// Push to every client watching this URL
hub.BroadcastToURL("/dashboard",
    websocket.HTMLEnvelope("#live-stats", html))

// Or to one session
hub.SendHTML(sessionID, "#live-stats", html)`,
	},
}
