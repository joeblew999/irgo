import type { FC } from "hono/jsx";
import { Link, ViteClient } from "vite-ssr-components/hono";

type LayoutProps = {
  title?: string;
  description?: string;
  children: any;
};

export const Layout: FC<LayoutProps> = ({
  title = "irgo - Native Apps with Go + Datastar",
  description = "Build native iOS, Android, and desktop apps using Go, HTML, and Datastar. No JavaScript frameworks required.",
  children,
}) => {
  return (
    <html lang="en">
      <head>
        <meta charset="UTF-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <meta name="description" content={description} />
        <title>{title}</title>
        <link rel="icon" type="image/png" href="/logo_text.png" />
        <ViteClient />
        <Link href="/src/style.css" rel="stylesheet" />
      </head>
      <body>
        <Header />
        <main>{children}</main>
        <Footer />
      </body>
    </html>
  );
};

const Header: FC = () => {
  return (
    <header class="header">
      <div class="container header__inner">
        <a href="/" class="header__logo">
          <img src="/logo_text.png" alt="irgo" style="height: 28px;" />
        </a>
        <nav class="nav">
          <a href="/demo" class="nav__link nav__link--accent">
            Demo
          </a>
          <a href="/docs" class="nav__link">
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
        <button
          class="mobile-menu-toggle"
          aria-label="Toggle menu"
          aria-expanded="false"
          onclick="document.body.classList.toggle('mobile-menu-open'); this.setAttribute('aria-expanded', document.body.classList.contains('mobile-menu-open'))"
        >
          <span class="mobile-menu-toggle__bar"></span>
          <span class="mobile-menu-toggle__bar"></span>
          <span class="mobile-menu-toggle__bar"></span>
        </button>
      </div>
      <nav class="mobile-nav">
        <a href="/demo" class="mobile-nav__link mobile-nav__link--accent">
          Demo
        </a>
        <a href="/docs" class="mobile-nav__link">
          Docs
        </a>
        <a href="/docs/getting-started" class="mobile-nav__link">
          Getting Started
        </a>
        <a href="/docs/examples" class="mobile-nav__link">
          Examples
        </a>
        <a
          href="https://github.com/stukennedy/irgo"
          class="mobile-nav__link"
          target="_blank"
          rel="noopener"
        >
          GitHub
        </a>
      </nav>
    </header>
  );
};

const Footer: FC = () => {
  return (
    <footer class="footer">
      <div class="container footer__inner">
        <div class="footer__brand">
          <img src="/logo_text.png" alt="irgo" />
          <span class="footer__copyright">MIT License</span>
        </div>
        <div class="footer__links">
          <a
            href="https://twitter.com/stukennedydev"
            class="footer__link"
            target="_blank"
            rel="noopener"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
              <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
            </svg>
            @stukennedydev
          </a>
          <a href="mailto:stu@stukennedy.com" class="footer__link">
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <rect x="2" y="4" width="20" height="16" rx="2" />
              <path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7" />
            </svg>
            stu@stukennedy.com
          </a>
          <a
            href="https://stukennedy.com"
            class="footer__link"
            target="_blank"
            rel="noopener"
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <circle cx="12" cy="12" r="10" />
              <path d="M2 12h20M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
            </svg>
            stukennedy.com
          </a>
          <a
            href="https://github.com/stukennedy/irgo"
            class="footer__link"
            target="_blank"
            rel="noopener"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
            </svg>
            GitHub
          </a>
        </div>
      </div>
    </footer>
  );
};
