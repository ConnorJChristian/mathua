// Fork change: the Math | Python header switch is gone. The fork has no split:
// one Study queue, one review session and one topic list span every module's
// concepts, and the server treats a request with no `track` as every track
// (internal/engine/track.go AllTracks). What remains here is the module
// metadata: which domains the enabled modules claim (data/tracks.json, which
// scripts/merge_concepts writes), for labelling and grouping.

import tracksData from '../data/tracks.json'

export type Track = string

export interface TrackInfo {
  id: string
  label: string
  domains: string[]
}

export const MATH_TRACK = 'math'

export const TRACKS: TrackInfo[] = tracksData.tracks

const domainTrack = new Map<string, string>()
for (const t of TRACKS) for (const d of t.domains) domainTrack.set(d, t.id)

/** The module track a concept domain belongs to. Unclaimed domains are math. */
export function trackOfDomain(domain?: string | null): Track {
  return (domain && domainTrack.get(domain)) || MATH_TRACK
}

/** True for a domain some module's track claims. */
export function isModuleDomain(domain?: string | null): boolean {
  return trackOfDomain(domain) !== MATH_TRACK
}
