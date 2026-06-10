import { createHighlighter, type Highlighter } from 'shiki'

let highlighter: Highlighter | null = null

async function getHighlighter(): Promise<Highlighter> {
  if (!highlighter) {
    highlighter = await createHighlighter({
      themes: ['github-dark'],
      langs: ['go', 'bash', 'html', 'typescript', 'json', 'toml', 'dockerfile', 'yaml', 'xml'],
    })
  }
  return highlighter
}

export async function highlight(code: string, lang: string): Promise<string> {
  const hl = await getHighlighter()

  // Map language aliases
  const langMap: Record<string, string> = {
    'golang': 'go',
    'sh': 'bash',
    'shell': 'bash',
    'templ': 'go', // templ is Go-like, best approximation
    'text': 'text',
    'plaintext': 'text',
  }

  const actualLang = langMap[lang] || lang

  // Check if language is supported
  const loadedLangs = hl.getLoadedLanguages()
  const langToUse = loadedLangs.includes(actualLang as any) ? actualLang : 'text'

  try {
    const html = hl.codeToHtml(code, {
      lang: langToUse,
      theme: 'github-dark',
    })

    // Extract just the code content, we'll wrap it ourselves
    // Shiki outputs: <pre class="shiki" style="..."><code>...</code></pre>
    // We want just the inner HTML of the code element
    const match = html.match(/<code[^>]*>([\s\S]*)<\/code>/)
    if (match) {
      return match[1]
    }
    return html
  } catch (e) {
    // Fallback: just escape HTML
    return code
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
  }
}
