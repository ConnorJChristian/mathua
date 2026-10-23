'use client'

import { usePathname, useRouter } from 'next/navigation'
import { setTrack, useTrack, type Track } from '../lib/track'

// Math | Python in the header. Same site, same login, same progress store:
// the switch only changes which concepts the pages offer (lib/track.ts).
//
// Pages that are about one concept or one session belong to the track they
// were opened on, so switching away from them goes to the new track's Learn
// shelf. Pages that list things (/learn, /domains, /profile, /graph) stay
// put and reload their lists for the new track.
const SESSION_PAGES = ['/review', '/goals', '/onboard', '/study', '/concept', '/session']

export default function TrackSwitch() {
  const track = useTrack()
  const router = useRouter()
  const pathname = usePathname() ?? ''

  function choose(next: Track) {
    if (next === track) return
    setTrack(next)
    // Read at click time rather than through useSearchParams, which would need a
    // Suspense boundary around the header on every statically exported page.
    const onConcept = pathname.startsWith('/learn') && new URLSearchParams(window.location.search).has('concept')
    if (onConcept || SESSION_PAGES.some(p => pathname === p || pathname.startsWith(p + '/'))) {
      router.push('/learn')
    }
  }

  const option = (value: Track, label: string) => (
    <button
      type="button"
      onClick={() => choose(value)}
      aria-pressed={track === value}
      className={`px-1.5 sm:px-2.5 min-h-[32px] font-mono text-[11px] transition-colors ${
        track === value
          ? 'bg-mathua-blue text-white'
          : 'text-mathua-muted hover:text-mathua-blue'
      }`}
    >
      {label}
    </button>
  )

  return (
    <div role="group" aria-label="Track" className="flex items-stretch border border-mathua-border-strong">
      {option('math', 'Math')}
      <span aria-hidden="true" className="w-px bg-mathua-border-strong" />
      {option('python', 'Python')}
    </div>
  )
}
