import { describe, expect, it } from 'vitest'
import { readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'
import { CONCEPT_COUNT, DOMAIN_COUNT } from '../lib/curriculumFacts'
import { MATH_DOMAIN_ORDER, PYTHON_DOMAIN_ORDER } from '../lib/graphDomains'

// The corpus is the owner of these numbers. The page copy used to say
// "1250 concepts spanning 17 domains" long after the corpus held 21, and no
// test noticed, because a number in JSX is just text. This reads the same files
// `scripts/counts.py` reads and fails when the copy drifts.
const CORPUS = join(__dirname, '../../../data/concepts')

// `enrichment` is a concept file with no domain entry: the ten concepts in it
// are not part of any curriculum domain, so `scripts/counts.py` excludes them
// from `by_domain` and its `concepts` total matches the domain sum.
const UNDOMAINED = new Set(['enrichment'])

function conceptsPerFile(): Map<string, number> {
  const out = new Map<string, number>()
  for (const name of readdirSync(CORPUS)) {
    if (!name.endsWith('.json')) continue
    const parsed: unknown = JSON.parse(readFileSync(join(CORPUS, name), 'utf8'))
    if (!Array.isArray(parsed)) throw new Error(`${name} is not a concept list`)
    out.set(name.slice(0, -'.json'.length), parsed.length)
  }
  return out
}

describe('curriculum facts in the copy', () => {
  const perFile = conceptsPerFile()

  // The copy describes the mathematics curriculum. The Python track is its own
  // side of the site (lib/track.ts) and is counted separately below.
  it('counts the same concepts as the corpus', () => {
    let total = 0
    for (const domain of MATH_DOMAIN_ORDER) {
      const n = perFile.get(domain)
      expect(n, `${domain}.json is missing from data/concepts`).toBeTypeOf('number')
      total += n ?? 0
    }
    expect(total).toBe(CONCEPT_COUNT)
  })

  it('counts the same domains as the corpus', () => {
    expect(DOMAIN_COUNT).toBe(MATH_DOMAIN_ORDER.length)
    const domains = [...perFile.keys()].filter((d) => !UNDOMAINED.has(d) && !d.startsWith('python_'))
    expect(domains.sort()).toEqual([...MATH_DOMAIN_ORDER].sort())
  })

  it('lists every Python chapter domain on disk, and only those', () => {
    const python = [...perFile.keys()].filter((d) => d.startsWith('python_'))
    expect(python.sort()).toEqual([...PYTHON_DOMAIN_ORDER].sort())
  })

  it('does not hardcode a concept count in the page copy', () => {
    // The guard above only works while the copy reads the constant. A literal
    // reintroduced into the section would keep the page wrong and keep this
    // suite green, which is the exact failure it exists to prevent.
    const source = readFileSync(join(__dirname, '../app/how-it-works/sections.tsx'), 'utf8')
    expect(source).not.toMatch(/\b\d{3,4}\s+concepts\b/)
    expect(source).not.toMatch(/\b\d{1,2}\s+domains\b/)
  })

  it('does not hardcode the domain count in the contributing copy', () => {
    // The same class of bug, one page further along: the concept schema table
    // said "One of the 16 domain categories" while the corpus held 21. It is a
    // string inside an array of table cells, so nothing type-checked it either.
    const source = readFileSync(join(__dirname, '../app/docs/contributing/page.tsx'), 'utf8')
    expect(source).not.toMatch(/\b\d{1,2}\s+domain categories\b/)
  })
})
