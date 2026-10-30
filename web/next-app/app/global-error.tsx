'use client'

// Root error boundary: replaces the whole layout when the root itself
// throws, so it must render its own <html>/<body> and cannot rely on the
// root layout's fonts, theme script, or Toaster. The palette is inlined for
// the same reason: light by default, dark when the visitor chose dark, read
// from the same storage keys as public/js/theme-init.js.
const THEME_INIT = `(function(){try{var t=localStorage.getItem('mathua.theme');if(t!=='dark'&&t!=='light')t=localStorage.getItem('mathua-theme');if(t==='dark')document.documentElement.classList.add('dark');}catch(e){}})();`

const PALETTE = `
:root{color-scheme:light;--ge-bg:#ffffff;--ge-ink:#1a1b1f;--ge-ink-muted:#4b4f58;--ge-accent:#2f6fc4;--ge-rule:#9a9ea8}
.dark{color-scheme:dark;--ge-bg:#0b0f1a;--ge-ink:#e8e2d5;--ge-ink-muted:#b8b3a8;--ge-accent:#c8a96e;--ge-rule:#444444}
`

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  return (
    <html lang="en">
      <head>
        <style dangerouslySetInnerHTML={{ __html: PALETTE }} />
        <script dangerouslySetInnerHTML={{ __html: THEME_INIT }} />
      </head>
      <body style={{ background: 'var(--ge-bg)', color: 'var(--ge-ink)', fontFamily: 'monospace' }}>
        <main style={{ display: 'flex', minHeight: '100vh', alignItems: 'center', justifyContent: 'center', padding: '24px' }}>
          <section style={{ width: '100%', maxWidth: '640px' }}>
            <div style={{ fontSize: '10px', textTransform: 'uppercase', letterSpacing: '0.18em', color: 'var(--ge-ink-muted)' }}>
              Mathua / error § 500
            </div>
            <h1 style={{ fontSize: '2rem', marginTop: '24px' }}>Something broke.</h1>
            <p role="alert" style={{ marginTop: '12px', fontSize: '14px', color: 'var(--ge-ink-muted)' }}>
              {error?.message || 'An unexpected error interrupted rendering.'}
              {error?.digest ? ` (ref ${error.digest})` : ''}
            </p>
            <div style={{ marginTop: '32px', display: 'flex', gap: '12px' }}>
              <button
                type="button"
                onClick={() => reset()}
                style={{ border: '1px solid var(--ge-accent)', color: 'var(--ge-accent)', background: 'transparent', padding: '10px 20px', fontSize: '12px', cursor: 'pointer' }}
              >
                ↻ Try again
              </button>
              <a href="/" style={{ border: '1px solid var(--ge-rule)', color: 'var(--ge-ink)', padding: '10px 20px', fontSize: '12px', textDecoration: 'none' }}>
                ← Return home
              </a>
            </div>
          </section>
        </main>
      </body>
    </html>
  )
}
