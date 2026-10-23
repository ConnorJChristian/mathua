// Domain names and order for pages: the base install's domains
// (lib/graphDomains.ts) plus whatever the enabled modules declare
// (data/tracks.json, written by scripts/merge_concepts).

import tracksData from '../data/tracks.json'
import { MATH_DOMAIN_ORDER, domainLabel as baseLabel } from './graphDomains'

const moduleLabels: Record<string, string> = tracksData.domain_labels

/** A domain's label: the base table, then a module's own label, then the id. */
export function domainLabel(domain: string): string {
  return moduleLabels[domain] ?? baseLabel(domain)
}

/** Base domains, then every module domain in its track's order. */
export function allDomainOrder(): string[] {
  const out: string[] = [...MATH_DOMAIN_ORDER]
  for (const t of tracksData.tracks) for (const d of t.domains) if (!out.includes(d)) out.push(d)
  return out
}
