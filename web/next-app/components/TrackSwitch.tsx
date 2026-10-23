'use client'

import { usePathname, useRouter } from 'next/navigation'
import { TRACKS, setTrack, useTrack, type Track } from '../lib/track'

// The track switch in the header (Math | Python | ...). Same site, same login,
// same progress store: the switch only changes which concepts the pages offer
// (lib/track.ts). It lists the base math track and every track an enabled
// module declares, and hides itself when there is only one.
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

  if (TRACKS.length < 2) return null

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

  // Fork change: with more than three tracks the button row is wider than the
  // header has room for and slides under the nav links, so it becomes a menu.
  if (TRACKS.length > 3) {
    return (
      <select
        aria-label="Track"
        value={track}
        onChange={(e) => choose(e.target.value)}
        className="min-h-[32px] max-w-[6.5rem] sm:max-w-[11rem] border border-mathua-border-strong bg-mathua-surface text-mathua-primary font-mono text-[11px] px-1.5"
      >
        {TRACKS.map((t) => (
          <option key={t.id} value={t.id}>{t.label}</option>
        ))}
      </select>
    )
  }

  return (
    <div role="group" aria-label="Track" className="flex items-stretch border border-mathua-border-strong">
      {TRACKS.map((t, i) => (
        <span key={t.id} className="flex items-stretch">
          {i > 0 && <span aria-hidden="true" className="w-px bg-mathua-border-strong" />}
          <button
            type="button"
            onClick={() => choose(t.id)}
            aria-pressed={track === t.id}
            className={`px-1.5 sm:px-2.5 min-h-[32px] font-mono text-[11px] transition-colors ${
              track === t.id ? 'bg-mathua-blue text-white' : 'text-mathua-muted hover:text-mathua-blue'
            }`}
          >
            {t.label}
          </button>
        </span>
      ))}
    </div>
  )
}
