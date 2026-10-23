import { describe, it, expect } from 'vitest'
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'
import { REPO_ROOT as REPO } from './helpers/roots'

// The bundled concept list (data/concepts.json) and the track list
// (data/tracks.json) are committed. They are generated from the base corpus
// and the committed modules only; your modules-local/ content reaches the app
// you run through the Docker build, never through these files. A local build
// (`make local`, MATHUA_INCLUDE_LOCAL_MODULES=1) regenerates them with your
// modules included, and this test is what stops that copy being committed.
function idsIn(dir: string): Set<string> {
  const out = new Set<string>()
  if (!existsSync(dir)) return out
  for (const f of readdirSync(dir)) {
    if (!f.endsWith('.json') || f === 'enrichment.json') continue
    for (const c of JSON.parse(readFileSync(join(dir, f), 'utf8'))) out.add(c.id)
  }
  return out
}

describe('the committed bundle', () => {
  const allowed = idsIn(join(REPO, 'data', 'concepts'))
  for (const m of readdirSync(join(REPO, 'modules'))) {
    for (const id of idsIn(join(REPO, 'modules', m, 'concepts'))) allowed.add(id)
  }
  const bundled: { id: string }[] = JSON.parse(readFileSync(join(REPO, 'web', 'next-app', 'data', 'concepts.json'), 'utf8'))

  it('reads both sides, so a pass is not two empty lists agreeing', () => {
    expect(allowed.size).toBeGreaterThan(1000)
    expect(bundled.length).toBeGreaterThan(1000)
  })

  it('holds only base and committed-module concepts', () => {
    const foreign = bundled.map(c => c.id).filter(id => !allowed.has(id))
    expect(
      foreign.length === 0
        ? 'clean'
        : `${foreign.length} concepts from modules-local/ (e.g. ${foreign.slice(0, 3).join(', ')}): run "go run ./scripts/merge_concepts" without -local before committing`,
    ).toBe('clean')
  })
})
