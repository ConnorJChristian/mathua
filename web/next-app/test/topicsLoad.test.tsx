import { describe, it, expect, vi } from 'vitest'
import { render, waitFor } from '@testing-library/react'
import * as api from '../lib/api'
import TopicsPage from '../app/topics/page'

vi.mock('../lib/api', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/api')>()
  return {
    ...mod,
    getCourseCatalog: vi.fn(() => new Promise(() => {})),
    getTranscript: vi.fn(async () => []),
    getProgress: vi.fn(async () => ({})),
    getSettings: vi.fn(async () => ({})),
    getMe: vi.fn(async () => { throw new Error('offline') }),
  }
})

vi.mock('../lib/auth', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/auth')>()
  return { ...mod, getUserInfo: vi.fn(() => ({ student_id: 's1' })) }
})

vi.mock('next/navigation', () => ({
  usePathname: () => '/topics',
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), prefetch: vi.fn() }),
  useSearchParams: () => new URLSearchParams(),
}))

// The transcript does not depend on the catalog: a slow catalog request used
// to hold it back (they were asked for one after the other).
describe('/topics loads', () => {
  it('asks for the transcript without waiting for the catalog', async () => {
    render(<TopicsPage />)
    await waitFor(() => expect(api.getTranscript).toHaveBeenCalled())
  })
})
