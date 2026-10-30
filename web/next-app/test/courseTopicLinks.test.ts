import { describe, it, expect } from 'vitest'
import fs from 'node:fs'
import path from 'node:path'

// A course lists hundreds of topics, each with two links. With Next's
// default prefetch, opening ECON 5153's Modules tab fired 450-odd route
// requests for pages the learner follows one at a time.
describe('course topic links', () => {
  it('do not prefetch', () => {
    const src = fs.readFileSync(path.join(__dirname, '../app/course/page.tsx'), 'utf8')
    const row = src.slice(src.indexOf('function TopicRow('), src.indexOf('\n}\n', src.indexOf('function TopicRow(')))
    const links = row.match(/<Link\b[^>]*>/g) ?? []
    expect(links.length).toBeGreaterThanOrEqual(2)
    for (const l of links) expect(l, l).toContain('prefetch={false}')
  })
})
