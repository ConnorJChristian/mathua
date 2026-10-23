// The site has one track per side of the curriculum: "math" (the base
// install) and one per track a module declares (internal/modules), such as
// "python". The header switch picks one; the server scopes the recommendation
// shelf, due reviews, the review queue, the quiz and the topic browser to it
// when the request carries `track=<id>`.
//
// The list comes from data/tracks.json, which scripts/merge_concepts writes
// from the enabled modules at build time. The choice lives in localStorage so
// guests have it too and so it survives a reload, the same way the theme does.

import { useEffect, useState } from 'react'
import tracksData from '../data/tracks.json'

export type Track = string

export interface TrackInfo {
  id: string
  label: string
  domains: string[]
}

export const MATH_TRACK = 'math'

const KEY = 'mathua_track'
const EVENT = 'mathua-track-changed'

export const TRACKS: TrackInfo[] = tracksData.tracks

const domainTrack = new Map<string, string>()
for (const t of TRACKS) for (const d of t.domains) domainTrack.set(d, t.id)

/** The track a concept domain belongs to. Unclaimed domains are math. */
export function trackOfDomain(domain?: string | null): Track {
  return (domain && domainTrack.get(domain)) || MATH_TRACK
}

/** True for a domain some module's track claims. */
export function isModuleDomain(domain?: string | null): boolean {
  return trackOfDomain(domain) !== MATH_TRACK
}

function known(track: string | null): Track {
  return track && TRACKS.some(t => t.id === track) ? track : MATH_TRACK
}

export function getTrack(): Track {
  if (typeof window === 'undefined') return MATH_TRACK
  try {
    return known(window.localStorage.getItem(KEY))
  } catch {
    return MATH_TRACK
  }
}

export function setTrack(track: Track): void {
  if (typeof window === 'undefined') return
  const next = known(track)
  if (getTrack() === next) return
  try {
    window.localStorage.setItem(KEY, next)
  } catch {
    // Private mode: the switch still works for this page view.
  }
  window.dispatchEvent(new CustomEvent(EVENT, { detail: next }))
}

/**
 * Appends `track=<id>` to an API URL for any track but math.
 *
 * A request without the parameter is the math track on the server, so every
 * math request stays byte-identical to what it was before tracks existed.
 */
export function withTrack(url: string, track: Track = getTrack()): string {
  if (!track || track === MATH_TRACK) return url
  return url + (url.includes('?') ? '&' : '?') + 'track=' + encodeURIComponent(track)
}

/** The active track, re-rendering when the header switch changes it. */
export function useTrack(): Track {
  const [track, setLocal] = useState<Track>(MATH_TRACK)
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
