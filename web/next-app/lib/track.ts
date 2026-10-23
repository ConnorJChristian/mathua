// The site has two tracks over one concept graph: Math (every domain Mathua
// shipped with) and Python (the `python_*` domains, the Python Data Science
// Handbook track). The header switch picks one; the server scopes the
// recommendation shelf, due reviews, the review queue, the quiz and the topic
// browser to it when the request carries `track=python`.
//
// The choice lives in localStorage so guests have it too and so it survives a
// reload, the same way the theme does. Pages that show track-scoped data read
// it through `useTrack`, which re-renders them when the switch is flipped.

import { useEffect, useState } from 'react'

export type Track = 'math' | 'python'

const KEY = 'mathua_track'
const EVENT = 'mathua-track-changed'

export const PYTHON_DOMAIN_PREFIX = 'python_'

export function isPythonDomain(domain?: string | null): boolean {
  return !!domain && domain.startsWith(PYTHON_DOMAIN_PREFIX)
}

export function trackOfDomain(domain?: string | null): Track {
  return isPythonDomain(domain) ? 'python' : 'math'
}

export function getTrack(): Track {
  if (typeof window === 'undefined') return 'math'
  try {
    return window.localStorage.getItem(KEY) === 'python' ? 'python' : 'math'
  } catch {
    return 'math'
  }
}

export function setTrack(track: Track): void {
  if (typeof window === 'undefined') return
  if (getTrack() === track) return
  try {
    window.localStorage.setItem(KEY, track)
  } catch {
    // Private mode: the switch still works for this page view.
  }
  window.dispatchEvent(new CustomEvent(EVENT, { detail: track }))
}

/**
 * Appends `track=python` to an API URL when the Python track is active.
 *
 * Only Python is sent. A request without the parameter is the Math track on the
 * server, so every Math request stays byte-identical to what it was before the
 * Python track existed.
 */
export function withTrack(url: string, track: Track = getTrack()): string {
  if (track !== 'python') return url
  return url + (url.includes('?') ? '&' : '?') + 'track=python'
}

/** The active track, re-rendering when the header switch changes it. */
export function useTrack(): Track {
  const [track, setLocal] = useState<Track>('math')
  useEffect(() => {
    setLocal(getTrack())
    const onChange = () => setLocal(getTrack())
    window.addEventListener(EVENT, onChange)
    window.addEventListener('storage', onChange)
    return () => {
      window.removeEventListener(EVENT, onChange)
      window.removeEventListener('storage', onChange)
    }
  }, [])
  return track
}
