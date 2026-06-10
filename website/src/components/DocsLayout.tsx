import type { FC } from "hono/jsx";
import { Link, ViteClient } from "vite-ssr-components/hono";

type DocsLayoutProps = {
  title: string;
  currentPath: string;
  children: any;
};

const navSections = [
  {
    title: "Getting Started",
    links: [
      { href: "/docs", label: "Introduction" },
      { href: "/docs/getting-started", label: "Quick Start" },
      { href: "/docs/project-structure", label: "Project Structure" },
    ],
  },
  {
    title: "Core Concepts",
    links: [
      { href: "/docs/routing", label: "Routing & Handlers" },
      { href: "/docs/templates", label: "Templ Templates" },
      { href: "/docs/datastar", label: "Datastar Integration" },
    ],
  },
  {
    title: "Platforms",
    links: [
      { href: "/docs/desktop", label: "Desktop Apps" },
      { href: "/docs/mobile", label: "Mobile Apps" },
      { href: "/docs/web", label: "Web Development" },
    ],
  },
  {
    title: "Advanced",
    links: [
      { href: "/docs/websockets", label: "WebSockets" },
      { href: "/docs/testing", label: "Testing" },
      { href: "/docs/deployment", label: "Deployment" },
    ],
  },
  {
    title: "Resources",
    links: [
      { href: "/docs/examples", label: "Examples" },
      { href: "/docs/cli", label: "CLI Reference" },
      { href: "/docs/api", label: "API Reference" },
    ],
  },
];

export const DocsLayout: FC<DocsLayoutProps> = ({
  title,
  currentPath,
  children,
}) => {
  return (
    <html lang="en">
      <head>
        <meta charset="UTF-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <meta name="description" content={`${title} - irgo documentation`} />
        <title>{title} | irgo Docs</title>
        <link rel="icon" type="image/png" href="/logo_text.png" />
        <ViteClient />
        <Link href="/src/style.css" rel="stylesheet" />
      </head>
      <body>
        <DocsHeader />
        <div class="container docs">
          <aside class="docs__sidebar">
            {navSections.map((section) => (
              <div class="docs__nav-section">
                <div class="docs__nav-title">{section.title}</div>
                {section.links.map((link) => (
                  <a
                    href={link.href}
                    class={`docs__nav-link ${currentPath === link.href ? "docs__nav-link--active" : ""}`}
                  >
                    {link.label}
                  </a>
                ))}
              </div>
            ))}
          </aside>
          <article class="docs__content">{children}</article>
        </div>
        <DocsFooter />
      </body>
    </html>
  );
};

const DocsHeader: FC = () => {
  return (
    <header class="header">
      <div class="container header__inner">
        <a href="/" class="header__logo">
          <img src="/logo_text.png" alt="irgo" style="height: 28px;" />
        </a>
        <nav class="nav">
          <a href="/docs" class="nav__link nav__link--active">
            Docs
          </a>
          <a href="/docs/getting-started" class="nav__link">
            Getting Started
          </a>
          <a href="/docs/examples" class="nav__link">
            Examples
          </a>
          <a
            href="https://github.com/stukennedy/irgo"
            class="nav__link"
            target="_blank"
            rel="noopener"
          >
            GitHub
          </a>
        </nav>
      </div>
    </header>
  );
};

const DocsFooter: FC = () => {
  return (
    <footer class="footer">
      <div class="container footer__inner">
        <div class="footer__brand">
          <img src="/logo_text.png" alt="irgo" style="height: 24px;" />
          <span class="footer__copyright">MIT License</span>
        </div>
        <div class="footer__links">
          <a
            href="https://twitter.com/stukennedydev"
            class="footer__link"
            target="_blank"
            rel="noopener"
          >
            @stukennedydev
          </a>
          <a href="mailto:stu@stukennedy.com" class="footer__link">
            stu@stukennedy.com
          </a>
          <a
            href="https://github.com/stukennedy/irgo"
            class="footer__link"
            target="_blank"
            rel="noopener"
          >
            GitHub
          </a>
        </div>
      </div>
    </footer>
  );
};
