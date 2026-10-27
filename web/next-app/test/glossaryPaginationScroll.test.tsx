import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import GlossaryPage from '../app/glossary/page'

const push = vi.fn()
const replace = vi.fn()

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push, replace }),
  usePathname: () => '/glossary',
  useSearchParams: () => new URLSearchParams(),
}))

vi.mock('../hooks/useTheme', () => ({
  useTheme: () => ({ mounted: true, theme: 'light', toggleTheme: vi.fn() }),
}))

vi.mock('../components/Header', () => ({ default: () => null }))
vi.mock('../components/BottomTabs', () => ({ default: () => null }))
vi.mock('../components/Footer', () => ({ default: () => null }))

beforeEach(() => {
  push.mockReset()
  replace.mockReset()
})

describe('glossary pagination scroll', () => {
  it('moves the document to the top when the next page is opened', async () => {
    const scrollTo = vi.spyOn(window, 'scrollTo').mockImplementation(() => {})
    render(<GlossaryPage />)

    fireEvent.click(await screen.findByRole('button', { name: 'Next page' }))

    expect(push).toHaveBeenCalledWith('/glossary?page=2', { scroll: false })
    expect(scrollTo).toHaveBeenCalledWith({ top: 0, left: 0, behavior: 'auto' })
    scrollTo.mockRestore()
  })
})
