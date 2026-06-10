import type { FC } from "hono/jsx";
import { DocsLayout } from "../../components/DocsLayout";
import { CodeBlock } from "../../components/CodeBlock";

export const WebPage: FC = () => {
  return (
    <DocsLayout title="Web Development" currentPath="/docs/web">
      <h1>Web Development</h1>

      <p>
        irgo apps work as standard web applications out of the box. The web mode
        is primarily used for development, but can also be deployed as a
        traditional web server.
      </p>

      <h2>Development Server</h2>

      <CodeBlock language="bash">{`irgo dev`}</CodeBlock>

      <p>
        This starts a web server at <code>http://localhost:8080</code> with hot
        reload. Changes to Go and templ files are automatically detected and the
        server restarts.
      </p>

      <h2>How It Works</h2>

      <p>
        In web mode, irgo runs as a standard Go HTTP server. The same handlers
        and templates you write for mobile and desktop work identically in the
        browser.
      </p>

      <CodeBlock language="go">
        {`// main.go (web/mobile entry point)
//go:build !desktop

func runDevServer() {
    r := app.NewRouter()

    mux := http.NewServeMux()
    mux.Handle("/static/", http.StripPrefix("/static/",
        http.FileServer(http.Dir("static"))))
    mux.Handle("/", r.Handler())

    fmt.Println("Dev server at http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", mux))
}`}
      </CodeBlock>

      <h2>Hot Reload Configuration</h2>

      <p>
        Hot reload is powered by{" "}
        <a href="https://github.com/air-verse/air">air</a>. The configuration is
        in <code>.air.toml</code>:
      </p>

      <CodeBlock language="toml">
        {`[build]
cmd = "templ generate && npx tailwindcss -i ./static/css/input.css -o ./static/css/output.css --minify && go build -o ./tmp/main ."
bin = "tmp/main serve"
include_ext = ["go", "templ", "html", "css"]
exclude_dir = ["tmp", "vendor", "node_modules"]

[misc]
clean_on_exit = true`}
      </CodeBlock>

      <p>
        This configuration runs templ generation and Tailwind CSS compilation on
        every file change, so you don't need separate watchers.
      </p>

      <h2>Production Deployment</h2>

      <p>For production web deployment, build a standard Go binary:</p>

      <CodeBlock language="bash">
        {`# Generate templates
templ generate

# Build CSS
npx tailwindcss -i static/css/input.css -o static/css/output.css --minify

# Build binary
go build -o myapp .

# Run
./myapp serve`}
      </CodeBlock>

      <h2>Docker Deployment</h2>

      <CodeBlock language="dockerfile">
        {`FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go install github.com/a-h/templ/cmd/templ@latest
RUN templ generate
RUN CGO_ENABLED=0 go build -o myapp .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/myapp .
COPY --from=builder /app/static ./static
EXPOSE 8080
CMD ["./myapp", "serve"]`}
      </CodeBlock>

      <h2>Next Steps</h2>

      <ul>
        <li>
          <a href="/docs/desktop">Desktop Apps</a> - Package as desktop
          application
        </li>
        <li>
          <a href="/docs/mobile">Mobile Apps</a> - Build for iOS and Android
        </li>
        <li>
          <a href="/docs/deployment">Deployment</a> - Deployment strategies
        </li>
      </ul>
    </DocsLayout>
  );
};
