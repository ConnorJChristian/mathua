import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import DomainsPage from '../app/domains/page'
import DomainTable from '../components/DomainTable'
import { concepts } from '../lib/conceptData'
import { buildDomainRows, byNeedsAttention } from '../lib/domainRows'
import { DOMAIN_ORDER, domainLabel } from '../lib/graphDomains'
import * as api from '../lib/api'

const push = vi.fn()
vi.mock('next/navigation', () => ({
  useRouter: () => ({ push }),
}))
vi.mock('next/link', () => ({
  default: ({ children, href }: { children: React.ReactNode; href: string }) => (
    <a href={href}>{children}</a>
  ),
}))
vi.mock('../lib/api', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/api')>()
  return {
    ...mod,
    getCurriculumDomains: vi.fn(),
    getCurriculumDomain: vi.fn(),
    getLessonEligibility: vi.fn(),
  }
})
vi.mock('../components/Header', () => ({ default: () => null }))
vi.mock('../components/Footer', () => ({ default: () => null }))
vi.mock('../components/BottomTabs', () => ({ default: () => null }))
vi.mock('../components/Loading', () => ({ default: ({ label }: { label?: string }) => <div>{label ?? 'Loading'}</div> }))
vi.mock('../components/SectionHeader', () => ({
  default: ({ title }: { title: string }) => <h2>{title}</h2>,
}))

const domains = DOMAIN_ORDER.map((id) => ({
  id,
  conceptCount: 10,
  masteredCount: 0,
  unlockedCount: 2,
  inProgressCount: 1,
}))

beforeEach(() => {
  push.mockReset()
  vi.mocked(api.getCurriculumDomains).mockReset().mockResolvedValue(domains)
  vi.mocked(api.getCurriculumDomain).mockReset()
  vi.mocked(api.getLessonEligibility).mockReset()
})

describe('/domains curriculum picker', () => {
  it('renders every domain in the canonical order with availability', async () => {
    render(<DomainsPage />)
    await waitFor(() => expect(screen.getByText(domainLabel('arithmetic'))).toBeTruthy())
    for (const domain of DOMAIN_ORDER) {
      expect(screen.getByText(domainLabel(domain))).toBeTruthy()
    }
    expect(screen.getAllByText(/\/ \d+ concepts available/).length).toBeGreaterThan(0)
  })

  it('opens a domain and renders its topics with server states', async () => {
    vi.mocked(api.getCurriculumDomain).mockResolvedValue({
      domain: { id: 'arithmetic' },
      topics: [
        { conceptId: 'arith.add', title: 'Single-digit addition', state: 'unlocked', depth: 0 },
        { conceptId: 'arith.sub', title: 'Subtraction', state: 'locked', depth: 1 },
      ],
    })
    render(<DomainsPage />)
    await waitFor(() => expect(screen.getByText(domainLabel('arithmetic'))).toBeTruthy())
    fireEvent.click(screen.getByText(domainLabel('arithmetic')))
    await screen.findByText('Single-digit addition')
    expect(screen.getByText('Available')).toBeTruthy()
    expect(screen.getByText('Locked')).toBeTruthy()
  })

  it('checks eligibility before entering /learn', async () => {
    vi.mocked(api.getCurriculumDomain).mockResolvedValue({
      domain: { id: 'arithmetic' },
      topics: [{ conceptId: 'arith.add', title: 'Single-digit addition', state: 'unlocked', depth: 0 }],
    })
    vi.mocked(api.getLessonEligibility).mockResolvedValue({
      concept_id: 'arith.add', ready: true, weak: [], missing: [], eligible: true,
    })
    render(<DomainsPage />)
    await waitFor(() => expect(screen.getByText(domainLabel('arithmetic'))).toBeTruthy())
    fireEvent.click(screen.getByText(domainLabel('arithmetic')))
    await screen.findByRole('button', { name: 'Learn' })
    fireEvent.click(screen.getByRole('button', { name: 'Learn' }))
    await waitFor(() => expect(push).toHaveBeenCalledWith('/learn?concept=arith.add'))
  })

  it('explains a locked topic instead of offering a start', async () => {
    vi.mocked(api.getCurriculumDomain).mockResolvedValue({
      domain: { id: 'arithmetic' },
      topics: [{ conceptId: 'arith.sub', title: 'Subtraction', state: 'locked', depth: 1 }],
    })
    vi.mocked(api.getLessonEligibility).mockResolvedValue({
      concept_id: 'arith.sub', ready: false, weak: [], missing: [], eligible: false,
      prerequisites: [{ conceptId: 'arith.add', title: 'Single-digit addition', state: 'unlocked', met: false }],
    })
    render(<DomainsPage />)
    await waitFor(() => expect(screen.getByText(domainLabel('arithmetic'))).toBeTruthy())
    fireEvent.click(screen.getByText(domainLabel('arithmetic')))
    fireEvent.click(await screen.findByRole('button', { name: /View prerequisite path/ }))
    const dialog = await screen.findByRole('dialog')
    expect(dialog.textContent).toContain('Subtraction')
    expect(screen.getByRole('link', { name: /Study →/ }).getAttribute('href'))
      .toBe('/study?concept=arith.add&return=%2Fstudy%3Fconcept%3Darith.sub')
    expect(push).not.toHaveBeenCalled()
  })

  it('shows a loading state immediately while readiness loads', async () => {
    vi.mocked(api.getCurriculumDomain).mockResolvedValue({
      domain: { id: 'arithmetic' },
      topics: [{ conceptId: 'arith.sub', title: 'Subtraction', state: 'locked', depth: 1 }],
    })
    let resolveEl: ((v: api.ReadinessRes) => void) | null = null
    vi.mocked(api.getLessonEligibility).mockImplementation(
      () => new Promise((resolve) => { resolveEl = resolve }),
    )
    render(<DomainsPage />)
    await waitFor(() => expect(screen.getByText(domainLabel('arithmetic'))).toBeTruthy())
    fireEvent.click(screen.getByText(domainLabel('arithmetic')))
    fireEvent.click(await screen.findByRole('button', { name: /View prerequisite path/ }))
    await screen.findByRole('dialog')
    expect(screen.getByText(/Loading prerequisites/)).toBeTruthy()
    resolveEl!({
      concept_id: 'arith.sub', ready: false, weak: [], missing: [], eligible: false,
      prerequisites: [{ conceptId: 'arith.add', title: 'Single-digit addition', state: 'locked', met: false }],
    })
    await screen.findByRole('link', { name: /Study →/ })
  })

  it('shows different prerequisites for different rows and never leaves stale data', async () => {
    vi.mocked(api.getCurriculumDomain).mockResolvedValue({
      domain: { id: 'arithmetic' },
      topics: [
        { conceptId: 'arith.sub', title: 'Subtraction', state: 'locked', depth: 1 },
        { conceptId: 'arith.mul', title: 'Multiplication', state: 'locked', depth: 1 },
      ],
    })
    const resolvers: Record<string, (v: api.ReadinessRes) => void> = {}
    vi.mocked(api.getLessonEligibility).mockImplementation((id: string) => new Promise((resolve) => {
      resolvers[id] = resolve as (v: api.ReadinessRes) => void
    }))
    render(<DomainsPage />)
    await waitFor(() => expect(screen.getByText(domainLabel('arithmetic'))).toBeTruthy())
    fireEvent.click(screen.getByText(domainLabel('arithmetic')))
    const buttons = await screen.findAllByRole('button', { name: /View prerequisite path/ })
    // Rapid clicks: A first, then B before either resolves.
    fireEvent.click(buttons[0])
    fireEvent.click(buttons[1])
    // A's late response must not overwrite B.
    resolvers['arith.sub']({
      concept_id: 'arith.sub', ready: false, weak: [], missing: [], eligible: false,
      prerequisites: [{ conceptId: 'arith.add', title: 'Single-digit addition', state: 'locked', met: false }],
    })
    await Promise.resolve()
    resolvers['arith.mul']({
      concept_id: 'arith.mul', ready: false, weak: [], missing: [], eligible: false,
      prerequisites: [{ conceptId: 'arith.dbl', title: 'Doubles', state: 'locked', met: false }],
    })
    await screen.findByText('Doubles')
    expect(screen.queryByText('Single-digit addition')).toBeNull()
    expect(screen.getByRole('dialog').textContent).toContain('Multiplication')
  })

  it('keeps locked prerequisites actionable with target-return context', async () => {
    vi.mocked(api.getCurriculumDomain).mockResolvedValue({
      domain: { id: 'arithmetic' },
      topics: [{ conceptId: 'arith.sub', title: 'Subtraction', state: 'locked', depth: 1 }],
    })
    vi.mocked(api.getLessonEligibility).mockResolvedValue({
      concept_id: 'arith.sub', ready: false, weak: [], missing: [], eligible: false,
      prerequisites: [
        { conceptId: 'arith.add', title: 'Single-digit addition', state: 'locked', met: false },
        { conceptId: 'arith.zero', title: 'Zero', state: 'upcoming', met: false },
      ],
    })
    render(<DomainsPage />)
    await waitFor(() => expect(screen.getByText(domainLabel('arithmetic'))).toBeTruthy())
    fireEvent.click(screen.getByText(domainLabel('arithmetic')))
    fireEvent.click(await screen.findByRole('button', { name: /View prerequisite path/ }))
    await screen.findByRole('dialog')
    const links = await screen.findAllByRole('link', { name: /Study →/ })
    expect(links).toHaveLength(2)
    for (const link of links) {
      expect(link.getAttribute('href')).toContain('/study?concept=')
      expect(link.getAttribute('href')).toContain('return=%2Fstudy%3Fconcept%3Darith.sub')
    }
  })

  it('closes the dialog without navigating or starting practice', async () => {
    vi.mocked(api.getCurriculumDomain).mockResolvedValue({
      domain: { id: 'arithmetic' },
      topics: [{ conceptId: 'arith.sub', title: 'Subtraction', state: 'locked', depth: 1 }],
    })
    vi.mocked(api.getLessonEligibility).mockResolvedValue({
      concept_id: 'arith.sub', ready: false, weak: [], missing: [], eligible: false,
      prerequisites: [{ conceptId: 'arith.add', title: 'Single-digit addition', state: 'locked', met: false }],
    })
    render(<DomainsPage />)
    await waitFor(() => expect(screen.getByText(domainLabel('arithmetic'))).toBeTruthy())
    fireEvent.click(screen.getByText(domainLabel('arithmetic')))
    const trigger = await screen.findByRole('button', { name: /View prerequisite path/ })
    fireEvent.click(trigger)
    await screen.findByRole('dialog')
    fireEvent.click(screen.getByRole('button', { name: 'Close' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(push).not.toHaveBeenCalled()
    expect(api.getLessonEligibility).toHaveBeenCalledTimes(1)
  })

  it('shows an error with retry when readiness fails', async () => {
    vi.mocked(api.getCurriculumDomain).mockResolvedValue({
      domain: { id: 'arithmetic' },
      topics: [{ conceptId: 'arith.sub', title: 'Subtraction', state: 'locked', depth: 1 }],
    })
    vi.mocked(api.getLessonEligibility)
      .mockRejectedValueOnce(new Error('offline'))
      .mockResolvedValueOnce({
        concept_id: 'arith.sub', ready: false, weak: [], missing: [], eligible: false,
        prerequisites: [{ conceptId: 'arith.add', title: 'Single-digit addition', state: 'locked', met: false }],
      })
    render(<DomainsPage />)
    await waitFor(() => expect(screen.getByText(domainLabel('arithmetic'))).toBeTruthy())
    fireEvent.click(screen.getByText(domainLabel('arithmetic')))
    fireEvent.click(await screen.findByRole('button', { name: /View prerequisite path/ }))
    await screen.findByRole('alert')
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    await screen.findByRole('link', { name: /Study →/ })
    expect(api.getLessonEligibility).toHaveBeenCalledTimes(2)
  })

  it('shows the empty state when the concept has no prerequisites', async () => {
    vi.mocked(api.getCurriculumDomain).mockResolvedValue({
      domain: { id: 'arithmetic' },
      topics: [{ conceptId: 'arith.add', title: 'Single-digit addition', state: 'locked', depth: 0 }],
    })
    vi.mocked(api.getLessonEligibility).mockResolvedValue({
      concept_id: 'arith.add', ready: false, weak: [], missing: [], eligible: false, prerequisites: [],
    })
    render(<DomainsPage />)
    await waitFor(() => expect(screen.getByText(domainLabel('arithmetic'))).toBeTruthy())
    fireEvent.click(screen.getByText(domainLabel('arithmetic')))
    fireEvent.click(await screen.findByRole('button', { name: /View prerequisite path/ }))
    await screen.findByText('No prerequisites listed')
  })

  it('opening a prerequisite path performs no practice write', async () => {
    vi.mocked(api.getCurriculumDomain).mockResolvedValue({
      domain: { id: 'arithmetic' },
      topics: [{ conceptId: 'arith.sub', title: 'Subtraction', state: 'locked', depth: 1 }],
    })
    vi.mocked(api.getLessonEligibility).mockResolvedValue({
      concept_id: 'arith.sub', ready: false, weak: [], missing: [], eligible: false,
      prerequisites: [{ conceptId: 'arith.add', title: 'Single-digit addition', state: 'locked', met: false }],
    })
    const practiceSpy = vi.spyOn(api, 'getLessonPractice').mockRejectedValue(new Error('must not be called'))
    render(<DomainsPage />)
    await waitFor(() => expect(screen.getByText(domainLabel('arithmetic'))).toBeTruthy())
    fireEvent.click(screen.getByText(domainLabel('arithmetic')))
    fireEvent.click(await screen.findByRole('button', { name: /View prerequisite path/ }))
    await screen.findByRole('link', { name: /Study →/ })
    expect(practiceSpy).not.toHaveBeenCalled()
    expect(push).not.toHaveBeenCalled()
    practiceSpy.mockRestore()
  })
})

describe('DomainTable', () => {
  it('renders no links when links is off, for the read-only share report', () => {
    const rows = buildDomainRows(concepts, {})
    const { container } = render(<DomainTable rows={rows} links={false} />)
    expect(container.querySelectorAll('a').length).toBe(0)
    expect(container.textContent).toContain('Arithmetic')
  })

  it('omits the expand control when no toggle handler is given', () => {
    const rows = buildDomainRows(concepts, {})
    render(<DomainTable rows={rows} links={false} />)
    expect(screen.queryByRole('button')).toBeNull()
  })
})

describe('buildDomainRows', () => {
  it('marks a concept due for review only from the server\'s due flag; decay is "fading"', () => {
    // FSRS Phase 0 review, F1: DECAYING is the pacing notion, and a topic with a long
    // interval reviewed over a fortnight ago is decaying and not due. The row's `due`
    // comes from the server's one rule (review.IsDue, exclusions applied).
    const [a, b] = concepts.filter(c => (c.prerequisites ?? []).length === 0).map(c => c.id)
    const rows = buildDomainRows(concepts, {
      [a]: { status: 'DECAYING', streak: 3, due: false },
      [b]: { status: 'MASTERED', streak: 3, due: true },
    })
    const all = rows.flatMap(r => r.concepts)
    expect(all.find(c => c.id === a)?.status).toBe('fading')
    expect(all.find(c => c.id === b)?.status).toBe('dueForReview')
    expect(rows.reduce((n, r) => n + r.dueForReview, 0)).toBe(1)
  })

  it('puts the weakest domain first when sorting by needs attention', () => {
    const rows = buildDomainRows(concepts, {})
    const weak = { ...rows[0], mastered: 0, label: 'Zzz' }
    const strong = { ...rows[1], mastered: rows[1].total, label: 'Aaa' }
    expect([weak, strong].sort(byNeedsAttention)[0].label).toBe('Zzz')
  })
})
