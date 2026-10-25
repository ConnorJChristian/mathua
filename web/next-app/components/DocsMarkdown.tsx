'use client'

import React from 'react'
import Link from 'next/link'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { useTheme } from '../hooks/useTheme'
import { DIAGRAMS } from '../lib/docsDiagrams'
import { slugify } from '../lib/docsToc'

// Render a repository markdown document as a web page.
//
// The body comes from content/docs.generated.json, which scripts/build-docs.mjs builds from
// docs/*.md, so the page and the repo doc are the same text. Before this, each docs page was a
// hand-written paraphrase and had drifted on the facts a reader checks first.
//
// Two rewrites make repository markdown fit a website:
//   - a diagram image becomes its interactive React Flow component (see DIAGRAMS)
//   - a link to a sibling .md becomes the route that renders it, or a GitHub URL if the doc is
//     not published

// A sibling document's route, when the website publishes it. Anything not listed resolves to
// the repository, so a link to an internal doc still goes somewhere a reader can follow.
const DOC_ROUTES: Record<string, string> = {
  'architecture.md': '/docs/architecture',
  'administration.md': '/docs/administration',
  'system-design.md': '/docs/system-design',
  'math-input.md': '/docs/math-input',
  'efficacy.md': '/docs/efficacy',
  'CONTRIBUTING.md': '/docs/contributing',
  'THIRD_PARTY_NOTICES.md': '/licenses',
}
const REPO_BLOB = 'https://github.com/chuma-beep/mathua/blob/main/'

// "diagrams/system-design-sm2-mastery-dark.svg" -> "system-design-sm2-mastery"
function diagramName(src: string): string | null {
  const m = src.match(/diagrams\/([\w-]+?)(?:-(?:light|dark))?\.svg$/)
  return m ? m[1] : null
}

function headingId(text: string): string {
  // The same function the generator used to mint the heading ids in the table of contents, so a
  // link and its heading always agree.
  return slugify(text)
}

function extractText(children: React.ReactNode): string {
  let text = ''
  React.Children.forEach(children, (child) => {
    if (typeof child === 'string' || typeof child === 'number') text += child
    else if (child && typeof child === 'object' && 'props' in child) {
      text += extractText((child as React.ReactElement<{ children?: React.ReactNode }>).props.children)
    }
  })
  return text
}

export default function DocsMarkdown({ body, className = '' }: { body: string; className?: string }) {
  const { theme, mounted } = useTheme()
  const variant = theme === 'dark' ? 'dark' : 'light'

  return (
    <div className={`docs-prose ${className}`}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          a: ({ href, children }) => {
            const target = href || ''
            if (target.startsWith('#')) return <a href={target}>{children}</a>
            if (/^https?:/.test(target)) {
              return <a href={target} target="_blank" rel="noreferrer">{children}</a>
            }
            const base = target.split('#')[0].split('/').pop() || ''
            const route = DOC_ROUTES[base]
            if (route) return <Link href={route} className="link-underline">{children}</Link>
            if (base.endsWith('.md')) {
              // A doc the site does not publish: send the reader to the repository.
              return (
                <a href={REPO_BLOB + target.replace(/^\.\//, '')} target="_blank" rel="noreferrer">
                  {children}
                </a>
              )
            }
            return <a href={target}>{children}</a>
          },
          img: ({ src, alt }) => {
            const url = typeof src === 'string' ? src : ''
            const name = diagramName(url)
            const Diagram = name ? DIAGRAMS[name] : undefined
            if (Diagram) return <Diagram />
            // Fallback: the committed SVG, in the theme that matches. `mounted` guards the
            // server render, where the theme is not yet known.
            // Repo-relative figures ("../figures/output/x.svg", right on GitHub) are served
            // from /figures/x.svg (Q23).
            const fig = url.match(/(?:^|\/)figures\/(?:output\/)?([\w.-]+)$/)
            const base = name ? `/diagrams/${name}-${variant}.svg` : fig ? `/figures/${fig[1]}` : url
            // eslint-disable-next-line @next/next/no-img-element
            return <img src={base} alt={alt || name || 'diagram'} loading="lazy" className="max-w-full h-auto my-6 mx-auto" />
          },
          code: ({ children }) => <code className="docs-code">{children}</code>,
          h1: ({ children, node: _n, ...props }) => { void _n; return <h1 id={headingId(extractText(children))} {...props}>{children}</h1> },
          h2: ({ children, node: _n, ...props }) => { void _n; return <h2 id={headingId(extractText(children))} {...props}>{children}</h2> },
          h3: ({ children, node: _n, ...props }) => { void _n; return <h3 id={headingId(extractText(children))} {...props}>{children}</h3> },
        }}
      >
        {body}
      </ReactMarkdown>

      <style jsx global>{`
        .docs-prose { overflow-x: clip; overflow-wrap: break-word; font-family: var(--font-space-grotesk), 'Space Grotesk', serif; color: var(--text-secondary); }
        .docs-prose h1, .docs-prose h2, .docs-prose h3, .docs-prose h4 { scroll-margin-top: 96px; }
        .docs-prose h1 { font-size: clamp(1.4rem, 1.1rem + 1.2vw, 2rem); font-weight: 400; color: var(--text-primary); margin: 0 0 1rem; }
        .docs-prose h2 { font-size: clamp(1.15rem, 1rem + 0.8vw, 1.3rem); font-weight: 400; color: var(--text-primary); border-bottom: 0.5px solid var(--border); padding-bottom: 0.5rem; margin: 2rem 0 1rem; }
        .docs-prose h3 { font-size: clamp(1rem, 0.95rem + 0.4vw, 1.1rem); font-weight: 500; color: var(--text-primary); margin: 1.5rem 0 0.75rem; }
        .docs-prose h4 { font-size: 0.95rem; font-weight: 600; color: var(--text-primary); margin: 1.25rem 0 0.5rem; }
        .docs-prose p, .docs-prose li { font-size: clamp(0.9rem, 0.85rem + 0.5vw, 1rem); line-height: 1.85; margin-bottom: 1rem; }
        .docs-prose ul, .docs-prose ol { padding-left: 1.5rem; margin-bottom: 1rem; }
        .docs-prose li { margin-bottom: 0.4rem; }
        .docs-prose a { color: var(--accent-blue); text-decoration: none; }
        .docs-prose a:hover { text-decoration: underline; }
        .docs-prose strong { color: var(--text-primary); font-weight: 600; }
        .docs-prose .docs-code { font-family: var(--font-jetbrains-mono), 'JetBrains Mono', monospace; font-size: 0.85em; background: var(--code-bg); padding: 0.1em 0.35em; color: var(--text-primary); overflow-wrap: anywhere; }
        .docs-prose pre { background: var(--code-bg); border: 0.5px solid var(--border); padding: 1rem 1.25rem; overflow-x: auto; margin-bottom: 1.25rem; }
        .docs-prose pre .docs-code { background: none; padding: 0; font-size: 0.8rem; line-height: 1.6; }
        .docs-prose blockquote { border-left: 2px solid var(--accent-blue); padding-left: 1.5rem; font-style: italic; color: var(--text-muted); margin: 1.5rem 0; }
        .docs-prose blockquote p { margin-bottom: 0.5rem; }
        .docs-prose table { display: block; overflow-x: auto; max-width: 100%; border-collapse: collapse; margin: 1rem 0; font-size: 0.875rem; }
        .docs-prose th, .docs-prose td { border: 0.5px solid var(--border); padding: 0.5rem 0.75rem; text-align: left; vertical-align: top; }
        .docs-prose th { font-weight: 600; background: var(--code-bg); color: var(--text-primary); }
        .docs-prose hr { border: 0; border-top: 0.5px solid var(--border); margin: 2rem 0; }
        .docs-prose img { border: 0.5px solid var(--border); }
      `}</style>
    </div>
  )
}
