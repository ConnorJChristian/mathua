import { describe, it, expect } from 'vitest'
import { precomputedLayout, layoutFromPrecomputed } from '../lib/flowLayout'
import { concepts } from '../lib/conceptData'

describe('precomputedLayout', () => {
  it('covers every concept in the bundled corpus', () => {
    // Guards artifact drift: a content wave that adds concepts without running
    // `npm run flow:build` would silently drop the graph back to dagre.
    expect(precomputedLayout(concepts.map(c => c.id))).not.toBeNull()
  })

  it('returns positions for known concepts', () => {
    const layout = precomputedLayout(['arith.add.single'])
    expect(layout).not.toBeNull()
    expect(layout!.get('arith.add.single')).toEqual({
      x: expect.any(Number),
      y: expect.any(Number),
    })
  })

  it('returns null when any id is missing, so the caller falls back wholesale', () => {
    expect(precomputedLayout(['arith.add.single', 'does.not.exist'])).toBeNull()
  })

  it('returns null for an empty set', () => {
    expect(precomputedLayout([])).toBeNull()
  })
})

describe('layoutFromPrecomputed', () => {
  // One concept the artifact lacks must not send the whole graph back to
  // dagre: it is placed beside its known prerequisite instead.
  it('places a concept the artifact lacks next to its prerequisite', () => {
    const base = concepts.slice(0, 40).map(c => ({ id: c.id, prerequisites: c.prerequisites }))
    const anchor = base[5].id
    const layout = layoutFromPrecomputed([
      ...base,
      { id: 'new.topic', prerequisites: [anchor] },
      { id: 'new.topic.next', prerequisites: ['new.topic'] },
      { id: 'new.orphan', prerequisites: [] },
    ])
    expect(layout).not.toBeNull()
    const a = layout!.get(anchor)!
    const n = layout!.get('new.topic')!
    expect(n.x).toBeGreaterThan(a.x)
    expect(n.y).toBeGreaterThan(a.y)
    // A chain of new concepts flows on to the right.
    expect(layout!.get('new.topic.next')!.x).toBeGreaterThan(n.x)
    // An orphan sits past every known node.
    const maxKnownX = Math.max(...base.map(c => layout!.get(c.id)!.x))
    expect(layout!.get('new.orphan')!.x).toBeGreaterThan(maxKnownX)
    // Known nodes keep the artifact's positions.
    expect(layout!.get(anchor)).toEqual(precomputedLayout([anchor])!.get(anchor))
  })

  it('gives up when the artifact covers less than half of the set', () => {
    expect(layoutFromPrecomputed([
      { id: concepts[0].id, prerequisites: [] },
      { id: 'x.1', prerequisites: [] },
      { id: 'x.2', prerequisites: [] },
    ])).toBeNull()
  })

  it('never stacks two new concepts on the same spot', () => {
    const base = concepts.slice(0, 20).map(c => ({ id: c.id, prerequisites: c.prerequisites }))
    const anchor = base[0].id
    const layout = layoutFromPrecomputed([...base, { id: 'n.1', prerequisites: [anchor] }, { id: 'n.2', prerequisites: [anchor] }])!
    expect(layout.get('n.1')).not.toEqual(layout.get('n.2'))
  })
})
