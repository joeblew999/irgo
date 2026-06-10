import type { FC } from "hono/jsx";
import { createHighlighter, type Highlighter } from "shiki";
import { createJavaScriptRegexEngine } from "shiki/engine/javascript";

type CodeBlockProps = {
  title?: string;
  language?: string;
  children: string;
};

// Map language aliases
const langMap: Record<string, string> = {
  golang: "go",
  sh: "bash",
  shell: "bash",
  templ: "go", // templ is Go-like
  text: "plaintext",
};

// All languages we need
const supportedLangs = [
  "go",
  "bash",
  "html",
  "css",
  "typescript",
  "javascript",
  "json",
  "yaml",
  "plaintext",
] as const;

// Singleton highlighter instance
let highlighterPromise: Promise<Highlighter> | null = null;

async function getHighlighter(): Promise<Highlighter> {
  if (!highlighterPromise) {
    highlighterPromise = createHighlighter({
      themes: ["tokyo-night"],
      langs: [...supportedLangs],
      engine: createJavaScriptRegexEngine(),
    });
  }
  return highlighterPromise;
}

export const CodeBlock: FC<CodeBlockProps> = async ({
  title,
  language = "go",
  children,
}) => {
  const code = typeof children === "string" ? children : String(children || "");
  const trimmedCode = code.trim();
  const lang = langMap[language] || language;

  let shikiHtml: string;

  try {
    const highlighter = await getHighlighter();
    const rawHtml = highlighter.codeToHtml(trimmedCode, {
      lang: supportedLangs.includes(lang as any) ? lang : "plaintext",
      theme: "tokyo-night",
    });
    // Remove newlines between tags to prevent extra spacing in pre
    shikiHtml = rawHtml.replace(/>\s+</g, "><");
  } catch (e) {
    // Fallback for unsupported languages
    const escaped = trimmedCode
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");
    shikiHtml = `<pre><code>${escaped}</code></pre>`;
  }

  return (
    <div class="code-block">
      <div class="code-block__header">
        <div class="code-block__dots">
          <span class="code-block__dot code-block__dot--red"></span>
          <span class="code-block__dot code-block__dot--yellow"></span>
          <span class="code-block__dot code-block__dot--green"></span>
        </div>
        {title && <span class="code-block__title">{title}</span>}
        <button
          class="code-block__copy"
          onclick={`navigator.clipboard.writeText(this.closest('.code-block').querySelector('code').textContent)`}
        >
          Copy
        </button>
      </div>
      <div
        class="code-block__content"
        dangerouslySetInnerHTML={{ __html: shikiHtml }}
      />
    </div>
  );
};

export const InlineCode: FC<{ children: string }> = ({ children }) => {
  return <code>{children}</code>;
};
