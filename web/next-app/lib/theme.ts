// Theme resolution, shared by the React hook (hooks/useTheme.ts) and mirrored by
// the pre-paint script (public/js/theme-init.js). Light is the default: a visitor
// with no stored choice gets the light palette, whatever their OS prefers. Only
// an explicit choice (the toggle) moves a visitor to dark, and that choice is
// kept in localStorage so it survives reloads.
//
// The pre-paint script cannot import this module (it runs before any bundle),
// so the two must agree by hand: the storage keys and the fallback below are the
// contract, and test/theme.test.ts checks both sides against it.

export type Theme = 'dark' | 'light'

/** Where the choice is kept. */
export const THEME_STORAGE_KEY = 'mathua.theme'

/** The key upstream used; still read so an existing visitor's choice carries over. */
export const LEGACY_THEME_STORAGE_KEY = 'mathua-theme'

export const DEFAULT_THEME: Theme = 'light'

/** Browser theme-color per palette, for the `<meta name="theme-color">` tag. */
export const THEME_COLOR: Record<Theme, string> = {
  light: '#ffffff',
  dark: '#0c0d0f',
}

export function isTheme(value: unknown): value is Theme {
  return value === 'dark' || value === 'light'
}

/** The theme a stored value resolves to: a valid choice wins, anything else is the default. */
export function resolveTheme(stored: string | null | undefined): Theme {
  return isTheme(stored) ? stored : DEFAULT_THEME
}

type StorageLike = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>

/** Read the stored choice, honouring the legacy key when the current one is unset. */
export function readStoredTheme(storage: StorageLike | null | undefined): Theme | null {
  if (!storage) return null
  try {
    const current = storage.getItem(THEME_STORAGE_KEY)
    if (isTheme(current)) return current
    const legacy = storage.getItem(LEGACY_THEME_STORAGE_KEY)
    if (isTheme(legacy)) return legacy
  } catch {
    // Storage can throw in private windows or when blocked; treat as unset.
  }
  return null
}

/** Persist a choice under the current key and retire the legacy one. */
export function writeStoredTheme(storage: StorageLike | null | undefined, theme: Theme): void {
  if (!storage) return
  try {
    storage.setItem(THEME_STORAGE_KEY, theme)
    storage.removeItem(LEGACY_THEME_STORAGE_KEY)
  } catch {
    // Nothing to do: the in-memory theme still applies for this page.
  }
}

/** Apply a theme to the document: the `dark` class, and the browser chrome colour. */
export function applyTheme(doc: Document, theme: Theme): void {
  doc.documentElement.classList.toggle('dark', theme === 'dark')
  const meta = doc.querySelector('meta[name="theme-color"]')
  if (meta) meta.setAttribute('content', THEME_COLOR[theme])
}
