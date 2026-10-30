'use client'

import { useEffect, useState } from 'react'
import { homeTarget, parseReturnStack } from '../lib/learnReturn'

/**
 * The home exit for a page that reads its return stack from the URL without
 * useSearchParams (the review page is a static shell): the course it was
 * opened from, or the profile. Read after mount, so the static export renders
 * the profile link first.
 */
export function useReturnHome(): { href: string; label: string } {
  const [home, setHome] = useState(() => homeTarget([]))
  useEffect(() => {
    setHome(homeTarget(parseReturnStack(new URLSearchParams(window.location.search))))
  }, [])
  return home
}
