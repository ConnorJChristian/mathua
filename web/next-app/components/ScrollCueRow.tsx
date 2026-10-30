'use client'

import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'

// A row of tabs that scrolls sideways when it does not fit, and says so. At
// 390px the course page's last tab ("Grades") was cut off at the edge with
// nothing to show that the row scrolls (Q28). While there is more to the
// right, the right edge fades and shows a "›" that scrolls the row on.
export function hasMoreRight(el: { scrollLeft: number; scrollWidth: number; clientWidth: number }): boolean {
  return el.scrollWidth - el.clientWidth - el.scrollLeft > 1
}

export default function ScrollCueRow({ label, className = '', children }: { label: string; className?: string; children: ReactNode }) {
  const ref = useRef<HTMLElement>(null)
  const [more, setMore] = useState(false)
  const update = useCallback(() => {
    if (ref.current) setMore(hasMoreRight(ref.current))
  }, [])
  useEffect(() => {
    update()
    const el = ref.current
    if (!el) return
    el.addEventListener('scroll', update, { passive: true })
    window.addEventListener('resize', update)
    const ro = typeof ResizeObserver !== 'undefined' ? new ResizeObserver(update) : null
    ro?.observe(el)
    return () => {
      el.removeEventListener('scroll', update)
      window.removeEventListener('resize', update)
      ro?.disconnect()
    }
  }, [update, children])
  return (
    <div className={`relative ${className}`}>
      <nav ref={ref} aria-label={label} className="flex gap-1 border-b border-mathua-border overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        {children}
      </nav>
      {more && (
        <button
          type="button"
          aria-label="More tabs"
          data-testid="tab-scroll-cue"
          onClick={() => ref.current?.scrollBy({ left: ref.current.clientWidth / 2, behavior: 'smooth' })}
          className="absolute right-0 top-0 bottom-[1px] flex w-10 items-center justify-end pr-1 font-mono text-base text-mathua-blue bg-gradient-to-l from-[var(--bg)] via-[var(--bg)] to-transparent"
        >
          ›
        </button>
      )}
    </div>
  )
}
