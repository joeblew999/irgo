import type { FC } from 'hono/jsx'
import { DocsLayout } from '../../components/DocsLayout'
import { CodeBlock } from '../../components/CodeBlock'
import { Callout } from '../../components/FeatureCard'

export const DeploymentPage: FC = () => {
  return (
    <DocsLayout title="Deployment" currentPath="/docs/deployment">
      <h1>Deployment</h1>

      <p>
        This guide covers deploying irgo apps to various platforms: desktop distribution,
        mobile app stores, and web hosting.
      </p>

      <h2>Desktop Distribution</h2>

      <h3>macOS</h3>

      <CodeBlock language="bash">
{`# Build macOS app bundle
irgo app build desktop macos`}
      </CodeBlock>

      <p>
        This creates a <code>.app</code> bundle in <code>build/desktop/macos/</code>.
      </p>

      <p>For distribution:</p>

      <ol>
        <li>Sign the app with your Apple Developer certificate</li>
        <li>Notarize the app with Apple</li>
        <li>Create a DMG for distribution</li>
      </ol>

      <CodeBlock language="bash">
{`# Sign the app
codesign --deep --force --sign "Developer ID Application: Your Name" MyApp.app

# Create DMG
hdiutil create -volname "MyApp" -srcfolder MyApp.app -ov -format UDZO MyApp.dmg

# Notarize
xcrun notarytool submit MyApp.dmg --apple-id you@email.com --team-id TEAMID --password @keychain:AC_PASSWORD`}
      </CodeBlock>

      <h3>Windows</h3>

      <CodeBlock language="bash">
{`# Build Windows executable
irgo app build desktop windows`}
      </CodeBlock>

      <p>
        Creates <code>.exe</code> in <code>build/desktop/windows/</code>.
      </p>

      <p>For distribution, consider:</p>
      <ul>
        <li>Code signing with a certificate</li>
        <li>Creating an installer (NSIS, WiX, Inno Setup)</li>
        <li>Including Visual C++ redistributables if needed</li>
      </ul>

      <h3>Linux</h3>

      <CodeBlock language="bash">
{`# Build Linux binary
irgo app build desktop linux`}
      </CodeBlock>

      <p>Distribution options:</p>
      <ul>
        <li>AppImage for universal distribution</li>
        <li>Flatpak for sandboxed distribution</li>
        <li>DEB/RPM packages for specific distributions</li>
      </ul>

      <h2>Mobile App Stores</h2>

      <h3>iOS App Store</h3>

      <ol>
        <li>Build the iOS framework:
          <CodeBlock language="bash">{`irgo app build ios`}</CodeBlock>
        </li>
        <li>Open the Xcode project in <code>ios/</code></li>
        <li>Add the framework to your project</li>
        <li>Configure signing with your Apple Developer account</li>
        <li>Archive and upload to App Store Connect</li>
      </ol>

      <Callout type="info">
        <p>
          Ensure your app meets Apple's App Store Review Guidelines.
          Apps using WebViews must provide significant native functionality.
        </p>
      </Callout>

      <h3>Google Play Store</h3>

      <ol>
        <li>Build the Android AAR:
          <CodeBlock language="bash">{`irgo app build android`}</CodeBlock>
        </li>
        <li>Or open <code>android/Example</code> in Android Studio, if you prefer its UI</li>
        <li>Add the AAR to your project</li>
        <li>Configure signing with your keystore</li>
        <li>Generate signed APK or App Bundle</li>
        <li>Upload to Google Play Console</li>
      </ol>

      <CodeBlock language="bash">
{`# Generate signed bundle
./gradlew bundleRelease`}
      </CodeBlock>

      <h2>Web Deployment</h2>

      <h3>Docker</h3>

      <CodeBlock title="Dockerfile" language="dockerfile">
{`FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

# Copy and build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN templ generate
RUN CGO_ENABLED=0 go build -o server .

# Runtime image
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/static ./static

EXPOSE 8080
CMD ["./server", "serve"]`}
      </CodeBlock>

      <CodeBlock language="bash">
{`# Build and run
docker build -t myapp .
docker run -p 8080:8080 myapp`}
      </CodeBlock>

      <h3>Fly.io</h3>

      <CodeBlock title="fly.toml" language="toml">
{`app = "myapp"
primary_region = "ord"

[build]
  dockerfile = "Dockerfile"

[http_service]
  internal_port = 8080
  force_https = true

[[services]]
  internal_port = 8080
  protocol = "tcp"

  [[services.ports]]
    port = 80
    handlers = ["http"]

  [[services.ports]]
    port = 443
    handlers = ["tls", "http"]`}
      </CodeBlock>

      <CodeBlock language="bash">
{`fly launch
fly deploy`}
      </CodeBlock>

      <h3>Railway</h3>

      <p>
        Railway auto-detects Go apps. Just connect your repository and deploy.
      </p>

      <CodeBlock language="bash">
{`# Set start command in Railway dashboard or railway.json
{
  "build": {
    "builder": "DOCKERFILE"
  },
  "deploy": {
    "startCommand": "./server serve"
  }
}`}
      </CodeBlock>

      <h3>Traditional VPS</h3>

      <CodeBlock language="bash">
{`# Build locally
GOOS=linux GOARCH=amd64 go build -o myapp .

# Upload to server
scp myapp user@server:/opt/myapp/
scp -r static user@server:/opt/myapp/

# On server: create systemd service
cat > /etc/systemd/system/myapp.service << EOF
[Unit]
Description=MyApp
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/myapp
ExecStart=/opt/myapp/myapp serve
Restart=always

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
systemctl enable myapp
systemctl start myapp`}
      </CodeBlock>

      <h2>Environment Configuration</h2>

      <CodeBlock language="go">
{`// Use environment variables for configuration
port := os.Getenv("PORT")
if port == "" {
    port = "8080"
}

dbURL := os.Getenv("DATABASE_URL")
apiKey := os.Getenv("API_KEY")`}
      </CodeBlock>

      <h2>Next Steps</h2>

      <ul>
        <li><a href="/docs/desktop">Desktop Apps</a> - Desktop-specific options</li>
        <li><a href="/docs/mobile">Mobile Apps</a> - Mobile-specific options</li>
        <li><a href="/docs/examples">Examples</a> - Deployment examples</li>
      </ul>
    </DocsLayout>
  )
}
