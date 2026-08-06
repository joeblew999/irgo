import type { FC } from "hono/jsx";
import { Layout } from "../components/Layout";
import { CodeBlock } from "../components/CodeBlock";
import { FeatureCard, PlatformCard } from "../components/FeatureCard";
import {
  IOSIcon,
  AndroidIcon,
  DesktopIcon,
  WebIcon,
  BoltIcon,
  RefreshIcon,
  CodeBracketIcon,
  CubeIcon,
  FireIcon,
  TerminalIcon,
} from "../components/Icons";
import {
  ArchitectureDiagram,
  MobileArchitectureDiagram,
} from "../components/ArchitectureDiagram";

export const HomePage: FC = () => {
  return (
    <Layout>
      {/* Hero Section */}
      <section class="section section--hero">
        <div class="hero__glow hero__glow--1"></div>
        <div class="hero__glow hero__glow--2"></div>
        <div class="container hero">
          <div class="hero__badge">
            <span>Now with Desktop Support</span>
          </div>
          <h1 class="hero__title">
            Native Apps with
            <br />
            <span class="hero__title-accent">Go + Datastar</span>
          </h1>
          <p class="hero__subtitle">
            Build native iOS, Android, and desktop apps using Go, HTML, and
            Datastar. The richness of a webapp in a native app. No JavaScript
            frameworks required.
          </p>
          <div class="hero__actions">
            <a href="/docs/getting-started" class="btn btn--primary btn--lg">
              Get Started
              <svg
                class="btn__icon"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <path d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            </a>
            <a
              href="https://github.com/stukennedy/irgo"
              class="btn btn--secondary btn--lg"
              target="_blank"
            >
              <svg class="btn__icon" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
              </svg>
              View on GitHub
            </a>
          </div>

          {/* Install command */}
          <div class="hero__code mt-2xl">
            <div class="install-cmd">
              <span class="install-cmd__prompt">$</span>
              <span class="install-cmd__text">
                go install github.com/stukennedy/irgo/cmd/irgo@latest
              </span>
              <button
                class="install-cmd__copy"
                onclick="navigator.clipboard.writeText('go install github.com/stukennedy/irgo/cmd/irgo@latest')"
              >
                <svg
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                  <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </section>

      {/* Platforms Section */}
      <section class="section platforms">
        <div class="container">
          <div class="features__header">
            <h2>One Codebase. Every Platform.</h2>
            <p>Write your app logic once in Go. Deploy everywhere.</p>
          </div>
          <div class="grid grid--4">
            <PlatformCard
              icon={<IOSIcon size={48} />}
              name="iOS"
              description="Native iOS apps via gomobile"
            />
            <PlatformCard
              icon={<AndroidIcon size={48} />}
              name="Android"
              description="Native Android apps via gomobile"
            />
            <PlatformCard
              icon={<DesktopIcon size={48} />}
              name="Desktop"
              description="macOS, Windows, Linux with webview"
            />
            <PlatformCard
              icon={<WebIcon size={48} />}
              name="Web"
              description="Standard HTTP server for browsers"
            />
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section class="section features">
        <div class="container">
          <div class="features__header">
            <h2>Why irgo?</h2>
            <p>The hypermedia approach to native app development</p>
          </div>
          <div class="grid grid--3">
            <FeatureCard
              icon={<BoltIcon size={24} />}
              title="Go-Powered Runtime"
              description="Write your backend logic in Go. Compile to native mobile frameworks or desktop apps with full access to the Go ecosystem."
            />
            <FeatureCard
              icon={<RefreshIcon size={24} />}
              title="Datastar for Interactivity"
              description="Use Datastar's hypermedia approach with SSE instead of complex JavaScript frameworks. Return HTML via Server-Sent Events. Simple and powerful."
            />
            <FeatureCard
              icon={<CodeBracketIcon size={24} />}
              title="Type-Safe Templates"
              description="Use templ for compile-time checked HTML templates. Catch errors at build time, not runtime. Full Go type safety."
            />
            <FeatureCard
              icon={<CubeIcon size={24} />}
              title="Virtual HTTP (Mobile)"
              description="No network sockets on mobile. Requests are intercepted and handled directly by Go in-process. Fast and secure."
            />
            <FeatureCard
              icon={<FireIcon size={24} />}
              title="Hot Reload Development"
              description="Edit Go and templ code and see changes instantly. Full hot reload support for rapid development."
            />
            <FeatureCard
              icon={<TerminalIcon size={24} />}
              title="Powerful CLI"
              description="Create, develop, and build projects with simple commands. irgo project new, irgo server dev, irgo app build - that's it."
            />
          </div>
        </div>
      </section>

      {/* Code Example Section */}
      <section class="section">
        <div class="container">
          <div class="features__header">
            <h2>Simple, Expressive Code</h2>
            <p>Build interactive UIs with Go handlers and templ templates</p>
          </div>

          <div class="grid grid--2" style="gap: var(--space-xl);">
            <div>
              <h3 style="margin-bottom: var(--space-md); color: var(--text-primary);">
                Handler
              </h3>
              <CodeBlock title="handlers/todos.go" language="go">
                {`func CreateTodo(ctx *router.Context) error {
    var signals struct {
        Title string \`json:"title"\`
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
}`}
              </CodeBlock>
            </div>

            <div>
              <h3 style="margin-bottom: var(--space-md); color: var(--text-primary);">
                Template
              </h3>
              <CodeBlock title="templates/todos.templ" language="templ">
                {`templ TodoItem(todo Todo) {
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
}`}
              </CodeBlock>
            </div>
          </div>
        </div>
      </section>

      {/* Architecture Section */}
      <section class="section architecture">
        <div class="container">
          <div class="features__header">
            <h2>The Thin Client Advantage</h2>
            <p>
              Go compiles your app and serves HTML partials directly. No
              frontend framework, no JSON APIs, no virtual DOM.
            </p>
          </div>

          <ArchitectureDiagram />

          <div
            class="grid grid--2"
            style="gap: var(--space-2xl); margin-top: var(--space-3xl);"
          >
            <div>
              <h3 style="margin-bottom: var(--space-md); color: var(--text-primary);">
                Mobile Architecture
              </h3>
              <MobileArchitectureDiagram />
            </div>

            <div>
              <h3 style="margin-bottom: var(--space-md); color: var(--text-primary);">
                How It's Different
              </h3>
              <div
                class="arch-diagram"
                style="padding: var(--space-xl); height: 100%; box-sizing: border-box;"
              >
                <div style="margin-bottom: var(--space-xl);">
                  <h4
                    style="color: var(--accent-primary); margin-bottom: var(--space-sm); font-size: 1rem;"
                  >
                    Traditional SPA
                  </h4>
                  <p
                    style="font-size: 0.875rem; color: var(--text-tertiary); margin-bottom: var(--space-md);"
                  >
                    {"API → JSON → JS Framework → Virtual DOM → DOM reconciliation"}
                  </p>
                  <div
                    style="display: flex; gap: var(--space-xs); flex-wrap: wrap;"
                  >
                    <span
                      style="background: var(--bg-elevated); padding: 4px 8px; border-radius: 4px; font-size: 0.75rem; color: var(--text-muted);"
                    >
                      React/Vue/Angular
                    </span>
                    <span
                      style="background: var(--bg-elevated); padding: 4px 8px; border-radius: 4px; font-size: 0.75rem; color: var(--text-muted);"
                    >
                      State Management
                    </span>
                    <span
                      style="background: var(--bg-elevated); padding: 4px 8px; border-radius: 4px; font-size: 0.75rem; color: var(--text-muted);"
                    >
                      Build Tools
                    </span>
                  </div>
                </div>

                <div style="margin-bottom: var(--space-xl);">
                  <h4
                    style="color: var(--success); margin-bottom: var(--space-sm); font-size: 1rem;"
                  >
                    irgo Approach
                  </h4>
                  <p
                    style="font-size: 0.875rem; color: var(--text-secondary); margin-bottom: var(--space-md);"
                  >
                    {"Go Handler → templ → SSE → DOM morph"}
                  </p>
                  <div
                    style="display: flex; gap: var(--space-xs); flex-wrap: wrap;"
                  >
                    <span
                      style="background: rgba(34, 197, 94, 0.1); border: 1px solid var(--success); padding: 4px 8px; border-radius: 4px; font-size: 0.75rem; color: var(--success);"
                    >
                      No JS Framework
                    </span>
                    <span
                      style="background: rgba(34, 197, 94, 0.1); border: 1px solid var(--success); padding: 4px 8px; border-radius: 4px; font-size: 0.75rem; color: var(--success);"
                    >
                      Server State
                    </span>
                    <span
                      style="background: rgba(34, 197, 94, 0.1); border: 1px solid var(--success); padding: 4px 8px; border-radius: 4px; font-size: 0.75rem; color: var(--success);"
                    >
                      Just Go
                    </span>
                  </div>
                </div>

                <div
                  style="background: var(--bg-elevated); border-radius: 8px; padding: var(--space-md); border-left: 3px solid var(--accent-primary);"
                >
                  <p
                    style="font-size: 0.875rem; color: var(--text-secondary); margin: 0;"
                  >
                    <strong style="color: var(--text-primary);">
                      The client is thin.
                    </strong>{" "}
                    It just renders what Go sends. Datastar handles the SSE
                    updates and DOM morphing. Your Go code is the single source of truth.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Comparison Table */}
      <section class="section">
        <div class="container">
          <div class="features__header">
            <h2>Platform Comparison</h2>
            <p>Choose the right mode for your target platform</p>
          </div>

          <div class="table-wrapper">
            <table class="comparison-table">
              <thead>
                <tr>
                  <th>Aspect</th>
                  <th>Mobile (iOS/Android)</th>
                  <th>Desktop</th>
                  <th>Web</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td>HTTP</td>
                  <td>Virtual (no sockets)</td>
                  <td>Real localhost server</td>
                  <td>Real HTTP server</td>
                </tr>
                <tr>
                  <td>Bridge</td>
                  <td>gomobile + native code</td>
                  <td>None (direct HTTP)</td>
                  <td>None</td>
                </tr>
                <tr>
                  <td>Entry Point</td>
                  <td>
                    <code>main.go</code>
                  </td>
                  <td>
                    <code>main_desktop.go</code>
                  </td>
                  <td>
                    <code>main.go</code> (serve)
                  </td>
                </tr>
                <tr>
                  <td>Build Tag</td>
                  <td>
                    <code>!desktop</code>
                  </td>
                  <td>
                    <code>desktop</code>
                  </td>
                  <td>
                    <code>!desktop</code>
                  </td>
                </tr>
                <tr>
                  <td>CGO Required</td>
                  <td>No</td>
                  <td>Yes (webview)</td>
                  <td>No</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section class="section" style="text-align: center;">
        <div class="container">
          <h2 style="margin-bottom: var(--space-md);">Ready to Build?</h2>
          <p style="margin-bottom: var(--space-xl); max-width: min(500px, 100%); margin-left: auto; margin-right: auto;">
            Get started in minutes. Create your first irgo app and deploy to any
            platform.
          </p>
          <div class="hero__actions">
            <a href="/docs/getting-started" class="btn btn--primary btn--lg">
              Read the Docs
            </a>
            <a href="/docs/examples" class="btn btn--secondary btn--lg">
              View Examples
            </a>
          </div>
        </div>
      </section>
    </Layout>
  );
};
