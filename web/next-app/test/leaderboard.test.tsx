import { fireEvent, render, screen, within, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { LeaderboardBoard, MyLeague } from '../lib/api'
import LeaderboardPage from '../app/leaderboard/page'

const { getLeaderboard, getMyLeague } = vi.hoisted(() => ({
  getLeaderboard: vi.fn(),
  getMyLeague: vi.fn(),
}))

vi.mock('../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../lib/api')>()
  return { ...actual, getLeaderboard, getMyLeague }
})

vi.mock('next/navigation', () => ({
  usePathname: () => '/leaderboard',
  useRouter: () => ({ push: vi.fn() }),
}))
vi.mock('next/link', () => ({
  default: ({ children, href }: { children: React.ReactNode; href: string }) => (
    <a href={href}>{children}</a>
  ),
}))
vi.mock('../hooks/useTheme', () => ({ useTheme: () => ({ mounted: true }) }))
vi.mock('../components/Header', () => ({ default: () => null }))
vi.mock('../components/Footer', () => ({ default: () => null }))
vi.mock('../components/BottomTabs', () => ({ default: () => null }))

function board(overrides: Partial<LeaderboardBoard> = {}): LeaderboardBoard {
  return {
    view: 'weekly',
    page: 1,
    page_size: 25,
    total: 3,
    total_pages: 1,
    week_start: '2026-10-19T00:00:00Z',
    week_end: '2026-10-26T00:00:00Z',
    entries: [
      { rank: 1, id: 'ada', name: 'Ada', xp: 120, xp_total: 900, mastered: 4, level: 'Scholar' },
      { rank: 2, id: 'bob', name: 'Bob', xp: 80, xp_total: 400, mastered: 2, level: 'Student' },
      { rank: 3, id: 'cy', name: 'Cy', xp: 40, xp_total: 100, mastered: 1, level: 'Novice' },
    ],
    ...overrides,
  }
}

function league(overrides: Partial<MyLeague> = {}): MyLeague {
  return {
    week: '2026-10-19',
    week_start: '2026-10-19T00:00:00Z',
    week_end: '2026-10-26T00:00:00Z',
    tier: 'bronze',
    league_no: 1,
    size: 2,
    promote: 5,
    demote: 5,
    members: [
      { rank: 1, id: 'ada', name: 'Ada', xp: 30, is_me: true },
      { rank: 2, id: 'bob', name: 'Bob', xp: 20 },
    ],
    me: { rank: 1, xp: 30 },
    ...overrides,
  }
}

beforeEach(() => {
  getLeaderboard.mockReset()
  getMyLeague.mockReset()
})

// The board answers three different questions, and the old page answered one of
// them wrongly: it ranked lifetime mastery by this week's count and printed a
// scoring formula that no code implemented. These tests pin the separation.
describe('the leaderboard page', () => {
  it('ranks weekly XP and never prints the old formula', async () => {
    getLeaderboard.mockResolvedValue(board())
    render(<LeaderboardPage />)

    await screen.findAllByText('Ada')
    const table = screen.getByRole('table')
    const rows = within(table).getAllByRole('row')
    expect(rows).toHaveLength(4)
    expect(within(rows[1]).getByText('Ada')).toBeInTheDocument()
    expect(within(rows[1]).getByText('120')).toBeInTheDocument()
    expect(screen.queryByText(/speed_bonus/)).toBeNull()
    expect(screen.queryByText(/concepts_mastered_this_week/)).toBeNull()
  })

  it('asks the server for all-time when the tab changes, and resets to page one', async () => {
    getLeaderboard.mockResolvedValue(board())
    render(<LeaderboardPage />)
    await screen.findByRole('table')

    fireEvent.click(screen.getByRole('tab', { name: 'All time' }))
    await waitFor(() => expect(getLeaderboard).toHaveBeenLastCalledWith('all_time', 1, 25))
    expect(screen.getByRole('tab', { name: 'All time' })).toHaveAttribute('aria-selected', 'true')
  })

  it('pages through the standings without dropping the selected view', async () => {
    const scrollTo = vi.spyOn(window, 'scrollTo').mockImplementation(() => {})
    getLeaderboard.mockImplementation((view: string, page: number) =>
      Promise.resolve(board({ view: view as LeaderboardBoard['view'], page, total: 60, total_pages: 3 })),
    )
    render(<LeaderboardPage />)
    await screen.findByRole('table')

    fireEvent.click(screen.getByRole('button', { name: 'Next' }))
    await waitFor(() => expect(getLeaderboard).toHaveBeenLastCalledWith('weekly', 2, 25))
    expect(getLeaderboard).not.toHaveBeenCalledWith('all_time', 2, 25)
    // A page turn is not a route change, so ScrollToTopOnNavigate cannot see
    // it. The page control asks for the top explicitly.
    expect(scrollTo).toHaveBeenCalledWith({ top: 0, left: 0, behavior: 'auto' })
    scrollTo.mockRestore()
  })

  it('keeps a learner outside the page discoverable without duplicating them', async () => {
    getLeaderboard.mockResolvedValue(
      board({
        page: 1,
        total: 60,
        total_pages: 3,
        me: { rank: 57, xp: 4, eligible: true, in_page: false },
      }),
    )
    render(<LeaderboardPage />)
    await screen.findByRole('table')

    expect(screen.getByText(/Your position:/)).toBeInTheDocument()
    expect(screen.getByText('#57')).toBeInTheDocument()
    expect(screen.queryByText('Learner 57')).toBeNull()
  })

  it('explains an empty week instead of showing placeholder learners', async () => {
    getLeaderboard.mockResolvedValue(board({ total: 0, total_pages: 1, entries: [] }))
    render(<LeaderboardPage />)

    expect(await screen.findByText(/No one has earned XP this week yet/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Start learning' })).toHaveAttribute('href', '/learn')
    expect(screen.queryByRole('table')).toBeNull()
  })

  it('shows the caller their own league and the rule it applies', async () => {
    getLeaderboard.mockResolvedValue(board())
    getMyLeague.mockResolvedValue(league())
    render(<LeaderboardPage />)
    await screen.findByRole('table')

    fireEvent.click(screen.getByRole('tab', { name: 'My league' }))
    expect(await screen.findByText('bronze league 1')).toBeInTheDocument()
    expect(screen.getByText(/Top 5 move up a tier/)).toBeInTheDocument()
    expect(screen.getByText('you')).toBeInTheDocument()
    expect(screen.getByRole('table', { name: /league standings/i })).toBeInTheDocument()
  })

  it('tells a signed-out reader to sign in rather than inventing a league', async () => {
    getLeaderboard.mockResolvedValue(board())
    getMyLeague.mockResolvedValue(null)
    render(<LeaderboardPage />)
    await screen.findByRole('table')

    fireEvent.click(screen.getByRole('tab', { name: 'My league' }))
    expect(await screen.findByRole('link', { name: 'Sign in' })).toHaveAttribute('href', '/login')
  })

  it('offers a retry when the board fails, and recovers', async () => {
    getLeaderboard.mockRejectedValueOnce(new Error('offline')).mockResolvedValue(board())
    render(<LeaderboardPage />)

    expect(await screen.findByText(/Could not load this board/)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findAllByText('Ada')).not.toHaveLength(0)
  })
})
