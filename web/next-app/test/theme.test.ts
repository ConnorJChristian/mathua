import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import fs from 'node:fs'
import path from 'node:path'
import {
  DEFAULT_THEME,
  LEGACY_THEME_STORAGE_KEY,
  THEME_COLOR,
  THEME_STORAGE_KEY,
  applyTheme,
  readStoredTheme,
  resolveTheme,
  writeStoredTheme,
} from '../lib/theme'
import { useTheme } from '../hooks/useTheme'

// Light is the fork's default. These tests pin the three facts the toggle and
// the pre-paint script rely on: no stored choice means light, a stored choice
// is honoured, and toggling writes the key.

const INIT_SCRIPT = fs.readFileSync(path.join(__dirname, '..', 'public', 'js', 'theme-init.js'), 'utf8')

function runInitScript(): void {
  // The script is plain IIFE JavaScript; evaluating it in jsdom exercises the
  // exact file the layout ships, not a re-typed copy.
  new Function(INIT_SCRIPT)()
}

function ensureThemeColorMeta(): HTMLMetaElement {
  let meta = document.querySelector('meta[name="theme-color"]') as HTMLMetaElement | null
  if (!meta) {
    meta = document.createElement('meta')
    meta.setAttribute('name', 'theme-color')
    document.head.appendChild(meta)
  }
  return meta
}

beforeEach(() => {
  localStorage.clear()
  document.documentElement.classList.remove('dark')
  ensureThemeColorMeta().setAttribute('content', '')
})

afterEach(() => {
  localStorage.clear()
  document.documentElement.classList.remove('dark')
})

describe('lib/theme', () => {
  it('defaults to light', () => {
    expect(DEFAULT_THEME).toBe('light')
    expect(resolveTheme(null)).toBe('light')
    expect(resolveTheme(undefined)).toBe('light')
    expect(resolveTheme('system')).toBe('light')
    expect(resolveTheme('')).toBe('light')
  })

  it('honours a stored choice', () => {
    expect(resolveTheme('dark')).toBe('dark')
    expect(resolveTheme('light')).toBe('light')
  })

  it('reads the current key before the legacy one', () => {
    expect(readStoredTheme(localStorage)).toBeNull()
    localStorage.setItem(LEGACY_THEME_STORAGE_KEY, 'dark')
    expect(readStoredTheme(localStorage)).toBe('dark')
    localStorage.setItem(THEME_STORAGE_KEY, 'light')
    expect(readStoredTheme(localStorage)).toBe('light')
  })

  it('writes the current key and retires the legacy one', () => {
    localStorage.setItem(LEGACY_THEME_STORAGE_KEY, 'dark')
    writeStoredTheme(localStorage, 'light')
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('light')
    expect(localStorage.getItem(LEGACY_THEME_STORAGE_KEY)).toBeNull()
  })

  it('survives a storage that throws', () => {
    const broken = {
      getItem: () => {
        throw new Error('blocked')
      },
      setItem: () => {
        throw new Error('blocked')
      },
      removeItem: () => {
        throw new Error('blocked')
      },
    }
    expect(readStoredTheme(broken)).toBeNull()
    expect(() => writeStoredTheme(broken, 'dark')).not.toThrow()
  })

  it('applies the class and the browser chrome colour', () => {
    const meta = ensureThemeColorMeta()
    applyTheme(document, 'dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(meta.getAttribute('content')).toBe(THEME_COLOR.dark)
    applyTheme(document, 'light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(meta.getAttribute('content')).toBe(THEME_COLOR.light)
  })
})

describe('public/js/theme-init.js (pre-paint)', () => {
  it('uses the same storage keys as lib/theme', () => {
    expect(INIT_SCRIPT).toContain(`'${THEME_STORAGE_KEY}'`)
    expect(INIT_SCRIPT).toContain(`'${LEGACY_THEME_STORAGE_KEY}'`)
    expect(INIT_SCRIPT).toContain(`'${THEME_COLOR.dark}'`)
    expect(INIT_SCRIPT).toContain(`'${THEME_COLOR.light}'`)
    // The OS preference is deliberately not consulted: light is the default.
    expect(INIT_SCRIPT).not.toContain('prefers-color-scheme')
  })

  it('no preference: light, even when the document started dark', () => {
    document.documentElement.classList.add('dark')
    runInitScript()
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(ensureThemeColorMeta().getAttribute('content')).toBe(THEME_COLOR.light)
  })

  it('stored dark: dark', () => {
    localStorage.setItem(THEME_STORAGE_KEY, 'dark')
    runInitScript()
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(ensureThemeColorMeta().getAttribute('content')).toBe(THEME_COLOR.dark)
  })

  it('legacy stored dark: dark', () => {
    localStorage.setItem(LEGACY_THEME_STORAGE_KEY, 'dark')
    runInitScript()
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('stored light: light', () => {
    localStorage.setItem(THEME_STORAGE_KEY, 'light')
    document.documentElement.classList.add('dark')
    runInitScript()
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })
})

describe('useTheme', () => {
  it('starts light with no preference and applies no dark class', () => {
    const { result } = renderHook(() => useTheme())
    expect(result.current.theme).toBe('light')
    expect(result.current.mounted).toBe(true)
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('starts dark with a stored dark preference', () => {
    localStorage.setItem(THEME_STORAGE_KEY, 'dark')
    const { result } = renderHook(() => useTheme())
    expect(result.current.theme).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('toggle writes the key and flips the class', () => {
    const { result } = renderHook(() => useTheme())
    act(() => result.current.toggleTheme())
    expect(result.current.theme).toBe('dark')
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    act(() => result.current.toggleTheme())
    expect(result.current.theme).toBe('light')
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })
})

describe('app/layout.tsx', () => {
  it('ships no dark class on <html> and a light theme-color', () => {
    const layout = fs.readFileSync(path.join(__dirname, '..', 'app', 'layout.tsx'), 'utf8')
    expect(layout).not.toMatch(/<html[^>]*className="dark"/)
    expect(layout).toContain(`themeColor: '${THEME_COLOR.light}'`)
    expect(layout).toContain('/js/theme-init.js')
  })

  it('globals.css declares the light palette on :root and dark under .dark only', () => {
    const css = fs.readFileSync(path.join(__dirname, '..', 'app', 'globals.css'), 'utf8')
    expect(css).toMatch(/:root\s*\{[^}]*color-scheme:\s*light;/)
    expect(css).toMatch(/\.dark\s*\{[^}]*color-scheme:\s*dark;/)
  })
})
