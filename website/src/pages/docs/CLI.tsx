import type { FC } from "hono/jsx";
import { DocsLayout } from "../../components/DocsLayout";
import { CodeBlock } from "../../components/CodeBlock";

export const CLIPage: FC = () => {
  return (
    <DocsLayout title="CLI Reference" currentPath="/docs/cli">
      <h1>CLI Reference</h1>

      <p>
        The irgo CLI provides commands for creating, developing, and building
        irgo applications.
      </p>

      <h2>Installation</h2>

      <CodeBlock language="bash">
        {`go install github.com/stukennedy/irgo/cmd/irgo@latest`}
      </CodeBlock>

      <h2>Commands</h2>

      <h3>irgo project new</h3>

      <p>Create a new irgo project.</p>

      <CodeBlock language="bash">
        {`# Create new project in directory
irgo project new myapp

# Initialize in current directory
irgo project new .`}
      </CodeBlock>

      <p>This creates a complete project structure with:</p>
      <ul>
        <li>Entry points for mobile, desktop, and web</li>
        <li>Sample handlers and templates</li>
        <li>Tailwind 4 CSS setup (no config file needed)</li>
        <li>Datastar downloaded automatically</li>
        <li>Hot reload configuration via air</li>
      </ul>

      <h3>irgo server dev</h3>

      <p>Start the development server with hot reload.</p>

      <CodeBlock language="bash">{`irgo server dev`}</CodeBlock>

      <p>
        Starts a web server at <code>http://localhost:8080</code> with automatic
        reload when Go or templ files change.
      </p>

      <h3>irgo app run</h3>

      <p>Run the application for a specific platform.</p>

      <CodeBlock language="bash">
        {`# Desktop
irgo app run desktop         # Run desktop app
irgo app run desktop --dev   # With devtools enabled

# iOS
irgo app run ios             # Build and run on Simulator
irgo app run ios --dev       # Hot reload development

# Android
irgo app run android         # Build and run on Emulator
irgo app run android --dev   # Hot reload development`}
      </CodeBlock>

      <h3>irgo app build</h3>

      <p>Build the application for production.</p>

      <CodeBlock language="bash">
        {`# Desktop
irgo app build desktop           # Current platform
irgo app build desktop macos     # macOS .app bundle
irgo app build desktop windows   # Windows .exe
irgo app build desktop linux     # Linux binary

# Mobile
irgo app build ios               # iOS framework
irgo app build android           # Android AAR
irgo app build all               # All mobile platforms`}
      </CodeBlock>

      <h3>irgo project assets</h3>

      <p>Generate Go code from templ templates.</p>

      <CodeBlock language="bash">{`irgo project assets`}</CodeBlock>

      <p>
        Runs <code>templ generate</code> to compile <code>.templ</code> files to{" "}
        <code>_templ.go</code>.
      </p>

      <h3>irgo tools install</h3>

      <p>Install required development tools.</p>

      <CodeBlock language="bash">{`irgo tools install`}</CodeBlock>

      <p>Installs:</p>
      <ul>
        <li>templ - Template compiler</li>
        <li>air - Hot reload tool</li>
        <li>gomobile - Mobile compiler (if not installed)</li>
      </ul>

      <h3>irgo version</h3>

      <p>Print the irgo CLI version.</p>

      <CodeBlock language="bash">{`irgo version`}</CodeBlock>

      <h3>irgo help</h3>

      <p>Show help for any command.</p>

      <CodeBlock language="bash">
        {`irgo help
irgo help build
irgo help run`}
      </CodeBlock>

      <h2>Environment Variables</h2>

      <div class="table-wrapper">
        <table class="comparison-table">
          <thead>
            <tr>
              <th>Variable</th>
              <th>Description</th>
              <th>Default</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>
                <code>IRGO_PORT</code>
              </td>
              <td>Dev server port</td>
              <td>8080</td>
            </tr>
            <tr>
              <td>
                <code>CGO_ENABLED</code>
              </td>
              <td>Required for desktop builds</td>
              <td>0 (must set to 1)</td>
            </tr>
            <tr>
              <td>
                <code>ANDROID_HOME</code>
              </td>
              <td>Android SDK location</td>
              <td>-</td>
            </tr>
            <tr>
              <td>
                <code>ANDROID_NDK_HOME</code>
              </td>
              <td>Android NDK location</td>
              <td>-</td>
            </tr>
          </tbody>
        </table>
      </div>

      <h2>Command Summary</h2>

      <div class="table-wrapper">
        <table class="comparison-table">
          <thead>
            <tr>
              <th>Command</th>
              <th>Description</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>
                <code>irgo project new &lt;name&gt;</code>
              </td>
              <td>Create new project</td>
            </tr>
            <tr>
              <td>
                <code>irgo server dev</code>
              </td>
              <td>Start dev server with hot reload</td>
            </tr>
            <tr>
              <td>
                <code>irgo app run desktop [--dev]</code>
              </td>
              <td>Run as desktop app</td>
            </tr>
            <tr>
              <td>
                <code>irgo app run ios [--dev]</code>
              </td>
              <td>Run on iOS Simulator</td>
            </tr>
            <tr>
              <td>
                <code>irgo app run android [--dev]</code>
              </td>
              <td>Run on Android Emulator</td>
            </tr>
            <tr>
              <td>
                <code>irgo app build desktop [platform]</code>
              </td>
              <td>Build desktop app</td>
            </tr>
            <tr>
              <td>
                <code>irgo app build ios</code>
              </td>
              <td>Build iOS framework</td>
            </tr>
            <tr>
              <td>
                <code>irgo app build android</code>
              </td>
              <td>Build Android AAR</td>
            </tr>
            <tr>
              <td>
                <code>irgo app build all</code>
              </td>
              <td>Build all mobile platforms</td>
            </tr>
            <tr>
              <td>
                <code>irgo project assets</code>
              </td>
              <td>Generate templ files</td>
            </tr>
            <tr>
              <td>
                <code>irgo tools install</code>
              </td>
              <td>Install dev dependencies</td>
            </tr>
            <tr>
              <td>
                <code>irgo version</code>
              </td>
              <td>Print version</td>
            </tr>
            <tr>
              <td>
                <code>irgo help [command]</code>
              </td>
              <td>Show help</td>
            </tr>
          </tbody>
        </table>
      </div>
    </DocsLayout>
  );
};
