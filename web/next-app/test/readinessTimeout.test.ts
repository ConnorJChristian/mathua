import { describe, it, expect, vi, afterEach } from 'vitest'
import { getLessonReadiness, READINESS_TIMEOUT_MS } from '../lib/api'

afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals() })

// /learn asks for the questions only after the readiness check answers. A
// check that hung held the first question for the default 30 s timeout, and
// then again for each retry; it now gives up after READINESS_TIMEOUT_MS, once.
describe('getLessonReadiness', () => {
  it('gives up on a hung request after its own short timeout, without retrying', async () => {
    vi.useFakeTimers()
    const fetchMock = vi.fn((_u: RequestInfo | URL, init?: RequestInit) => new Promise<Response>((_res, rej) => {
      init?.signal?.addEventListener('abort', () => rej(new DOMException('aborted', 'AbortError')))
    }))
    vi.stubGlobal('fetch', fetchMock)
    expect(READINESS_TIMEOUT_MS).toBeLessThanOrEqual(10000)
    const p = getLessonReadiness('c1')
    const settled = p.then(() => 'resolved', () => 'rejected')
    await vi.advanceTimersByTimeAsync(READINESS_TIMEOUT_MS + 100)
    expect(await Promise.race([settled, Promise.resolve('pending')])).toBe('rejected')
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })
})
