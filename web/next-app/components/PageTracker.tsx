'use client'

import { useEffect } from 'react'
import { usePathname } from 'next/navigation'
import { track } from '../lib/events'

// Local event log (fork change): one page_view per page, and on leaving it
// how long the tab was actually visible there, so time spent studying can be
// told apart from time the app sat in a background tab.
export default function PageTracker() {
  const pathname = usePathname()
  useEffect(() => {
    if (!pathname) return
    const start = Date.now()
    let visibleMs = 0
    let visibleSince: number | null = document.visibilityState === 'visible' ? start : null
    const onVis = () => {
      if (document.visibilityState === 'visible') {
        visibleSince = Date.now()
      } else if (visibleSince !== null) {
        visibleMs += Date.now() - visibleSince
        visibleSince = null
      }
    }
    document.addEventListener('visibilitychange', onVis)
    track({ kind: 'page_view', page: pathname, data: { search: window.location.search } })
    return () => {
      document.removeEventListener('visibilitychange', onVis)
      if (visibleSince !== null) visibleMs += Date.now() - visibleSince
      track({ kind: 'page_leave', page: pathname, data: { total_s: (Date.now() - start) / 1000, visible_s: visibleMs / 1000 } })
    }
  }, [pathname])
  return null
}
