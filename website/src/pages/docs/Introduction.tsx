import type { FC } from "hono/jsx";
import { DocsLayout } from "../../components/DocsLayout";
import { CodeBlock } from "../../components/CodeBlock";
import { Callout } from "../../components/FeatureCard";

export const IntroductionPage: FC = () => {
  return (
    <DocsLayout title="Introduction" currentPath="/docs">
      <h1>Introduction to irgo</h1>

      <p>
        <strong>irgo</strong> is a hypermedia-driven application framework that
        uses Go as a runtime kernel with Datastar. Build native iOS, Android, and
        desktop apps using Go, HTML, and Datastar - no JavaScript frameworks
        required.
      </p>

      <h2>The Hypermedia Approach</h2>

      <p>
        Unlike traditional mobile and desktop frameworks that require learning
        platform-specific languages and UI toolkits, irgo takes a different
        approach: <strong>HTML is your UI layer</strong>.
      </p>

      <p>
        Your Go code renders HTML templates. Datastar handles interactivity using
        Server-Sent Events (SSE) to stream HTML updates to the browser. The
        framework handles the rest - whether that's routing requests through
        a native bridge on mobile or serving them over HTTP on desktop.
      </p>

      <CodeBlock title="The irgo pattern" language="text">
        {`User Interaction → Datastar Request → Go Handler → SSE Response → DOM Morph`}
      </CodeBlock>

      <h2>Key Features</h2>

      <ul>
        <li>
          <strong>Go-Powered Apps</strong>: Write your backend logic in Go,
          compile to native mobile frameworks or desktop apps
        </li>
        <li>
          <strong>Datastar for Interactivity</strong>: Use Datastar's SSE-based
          hypermedia approach instead of complex JavaScript
        </li>
        <li>
          <strong>Cross-Platform</strong>: Single codebase for iOS, Android,
          desktop (macOS, Windows, Linux), and web
        </li>
        <li>
          <strong>Virtual HTTP (Mobile)</strong>: No network sockets - requests
          are intercepted and handled directly by Go
        </li>
        <li>
          <strong>Native Webview (Desktop)</strong>: Real HTTP server with
          native webview window
        </li>
        <li>
          <strong>Type-Safe Templates</strong>: Use{" "}
          <a href="https://templ.guide">templ</a> for compile-time checked HTML
          templates
        </li>
        <li>
          <strong>Hot Reload Development</strong>: Edit Go/templ code and see
          changes instantly
        </li>
      </ul>

      <h2>Platform Support</h2>

      <p>irgo supports four deployment modes:</p>

      <div class="table-wrapper">
        <table class="comparison-table">
          <thead>
            <tr>
              <th>Platform</th>
              <th>Architecture</th>
              <th>Build Tag</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>iOS</td>
              <td>Virtual HTTP via gomobile bridge</td>
              <td>
                <code>!desktop</code>
              </td>
            </tr>
            <tr>
              <td>Android</td>
              <td>Virtual HTTP via gomobile bridge</td>
              <td>
                <code>!desktop</code>
              </td>
            </tr>
            <tr>
              <td>Desktop</td>
              <td>Real HTTP server + native webview</td>
              <td>
                <code>desktop</code>
              </td>
            </tr>
            <tr>
              <td>Web</td>
              <td>Standard HTTP server</td>
              <td>
                <code>!desktop</code>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <h2>Prerequisites</h2>

      <p>Before getting started, make sure you have:</p>

      <ul>
        <li>Go 1.21 or later</li>
        <li>
          <a href="https://templ.guide">templ</a>:{" "}
          <code>go install github.com/a-h/templ/cmd/templ@latest</code>
        </li>
        <li>
          <a href="https://github.com/air-verse/air">air</a> (for hot reload):{" "}
          <code>go install github.com/air-verse/air@latest</code>
        </li>
      </ul>

      <Callout type="info" title="Platform-specific requirements">
        <p>
          <strong>Mobile:</strong> Xcode for iOS; the Android toolchain installs itself
          (Android)
          <br />
          <strong>Desktop:</strong> CGO enabled with a C compiler (Xcode CLI on
          macOS, MinGW on Windows, GCC on Linux)
        </p>
      </Callout>

      <h2>Quick Example</h2>

      <p>Here's a taste of what irgo code looks like:</p>

      <CodeBlock title="handlers/hello.go" language="go">
        {`package handlers

import (
    "myapp/templates"
    "github.com/stukennedy/irgo/pkg/router"
)

func Hello(ctx *router.Context) error {
    var signals struct {
        Name string \`json:"name"\`
    }
    ctx.ReadSignals(&signals)

    if signals.Name == "" {
        signals.Name = "World"
    }

    return ctx.SSE().PatchTempl(templates.Greeting(signals.Name))
}`}
      </CodeBlock>

      <CodeBlock title="templates/greeting.templ" language="templ">
        {`package templates

templ Greeting(name string) {
    <div id="greeting" class="greeting" data-signals="{name: ''}">
        <h1>Hello, { name }!</h1>
        <input
            type="text"
            data-bind:name
            placeholder="Enter a name"
        />
        <button data-on:click="@get('/hello')">Greet</button>
    </div>
}`}
      </CodeBlock>

      <h2>AI Coding Assistants</h2>

      <p>
        If you're using an AI coding assistant (like Claude, Cursor, or GitHub
        Copilot), point it to our <a href="/llms.txt">llms.txt</a> file. This
        provides comprehensive instructions for writing irgo applications,
        including handler patterns, templ syntax, Datastar integration, and
        SSE responses.
      </p>

      <h2>Next Steps</h2>

      <p>
        Ready to build your first irgo app? Head to the{" "}
        <a href="/docs/getting-started">Quick Start</a> guide.
      </p>
    </DocsLayout>
  );
};
