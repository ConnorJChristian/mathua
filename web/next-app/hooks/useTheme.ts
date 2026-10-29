'use client'

import { useState, useEffect, useCallback } from 'react'
import {
  type Theme,
  DEFAULT_THEME,
  applyTheme,
  readStoredTheme,
  resolveTheme,
  writeStoredTheme,
} from '../lib/theme'

export type { Theme }

function getInitialTheme(): Theme {
  if (typeof window === 'undefined') return DEFAULT_THEME
  // Light unless the visitor chose dark. The OS preference is not consulted:
  // the fork defaults to light, and only the toggle moves a visitor off it.
  return resolveTheme(readStoredTheme(window.localStorage))
}

export function useTheme() {
  const [theme, setTheme] = useState<Theme>(getInitialTheme)
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
  }, [])

  useEffect(() => {
    if (!mounted) return
    applyTheme(document, theme)
  }, [theme, mounted])

  const toggleTheme = useCallback(() => {
    const next: Theme = theme === 'dark' ? 'light' : 'dark'
    setTheme(next)
    writeStoredTheme(window.localStorage, next)
  }, [theme])

  return { theme, mounted, toggleTheme }
}
