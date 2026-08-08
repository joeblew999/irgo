package handlers

import (
	"time"

	"docs-templ/templates"
	"github.com/stukennedy/irgo/pkg/router"
)

// The counter behind the demo page.
//
// The TSX site drew this with a JavaScript closure and wrote invented lines
// into a box labelled "SSE Event Stream". This is the real thing: a Datastar
// POST, a Go handler, an SSE response that patches two elements.
//
// The count and the log travel with the client as signals, for the reason
// spelled out in handlers.go — every request to a Cloudflare Worker gets a
// fresh Go runtime, so a package variable here would read 0 forever.

// logLimit is how many events the page keeps. The old page kept five.
const logLimit = 5

// demoSignals is what the page sends up with each request.
type demoSignals struct {
	Count int                  `json:"count"`
	Log   []templates.LogEntry `json:"log"`
}

func mountDemo(r *router.Router) {
	r.DSPost("/api/demo/increment", demoStep("increment", +1))
	r.DSPost("/api/demo/decrement", demoStep("decrement", -1))
}

// demoStep builds the handler for one button.
func demoStep(name string, delta int) func(*router.Context) error {
	return func(ctx *router.Context) error {
		var signals demoSignals
		_ = ctx.ReadSignals(&signals)

		count := signals.Count + delta
		if count < 0 {
			// The old page floored at zero; keep that, so the minus button
			// visibly does nothing rather than going negative.
			count = 0
		}

		now := time.Now().Format("15:04:05")
		log := append([]templates.LogEntry{
			{Kind: "sse", Time: now, Text: "event: datastar-patch-elements"},
			{Kind: "request", Time: now, Text: "POST /api/demo/" + name},
		}, signals.Log...)
		if len(log) > logLimit {
			log = log[:logLimit]
		}

		sse := ctx.SSE()
		sse.PatchTempl(templates.DemoCounterValue(count))
		sse.PatchTempl(templates.DemoLog(log))
		return sse.PatchSignals(map[string]any{
			"count": count,
			"log":   log,
		})
	}
}
