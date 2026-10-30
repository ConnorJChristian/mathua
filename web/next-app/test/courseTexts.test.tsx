import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import CourseTextLink from '../components/CourseTextLink'
import { bookCounts, bookKindLabel } from '../lib/libraryLabels'

vi.mock('next/link', () => ({ default: ({ href, children, ...rest }: { href: string; children: React.ReactNode }) => <a href={href} {...rest}>{children}</a> }))

// Q62/Q63: course texts link to where they can be read.
describe('CourseTextLink', () => {
  it('links a shelved book to its library page', () => {
    render(<CourseTextLink text={{ key: 'm/b.pdf', title: 'A Book', kind: 'pdf', library_id: 'm-b-pdf' }} />)
    expect(screen.getByRole('link', { name: 'A Book' }).getAttribute('href')).toBe('/library?book=m-b-pdf')
  })
  it('opens an HTML book as itself, not a library page that does not exist', () => {
    render(<CourseTextLink text={{ key: 'hegemony/hss/itr.html', title: 'Hegemony and Socialist Strategy', kind: 'html', url: '/api/modules/hegemony/files/hss/itr.html' }} />)
    const a = screen.getByRole('link', { name: /Hegemony/ })
    expect(a.getAttribute('href')).toContain('/api/modules/hegemony/files/hss/itr.html')
    expect(a.getAttribute('target')).toBe('_blank')
  })
  it('sends lecture notes to the Lectures tab', () => {
    const onLectures = vi.fn()
    render(<CourseTextLink text={{ key: 'notes:ma-program', title: 'Lecture notes', kind: 'notes' }} onLectures={onLectures} />)
    fireEvent.click(screen.getByRole('button', { name: 'Lecture notes' }))
    expect(onLectures).toHaveBeenCalled()
  })
  it('leaves an online text the library lacks as plain text', () => {
    render(<CourseTextLink text={{ key: 'web:Course manual', title: 'Course manual', kind: 'web' }} />)
    expect(screen.getByText('Course manual (online)')).toBeTruthy()
    expect(screen.queryByRole('link')).toBeNull()
  })
})

describe('library labels', () => {
  it('does not call a missing PDF yours', () => {
    expect(bookKindLabel({ kind: 'pdf', available: false })).toBe('PDF, not on this computer')
    expect(bookKindLabel({ kind: 'pdf', available: true })).toBe('Your PDF')
    expect(bookKindLabel({ kind: 'web', available: true })).toBe('Online book')
  })
  it('says a book has no listed contents instead of 0 sections · 0 topics', () => {
    expect(bookCounts({ sections: [], topic_count: 0 })).toBe('Whole book: no contents listed')
    expect(bookCounts({ sections: [{ title: 'a', level: 0, topics: [] }], topic_count: 2 })).toBe('1 section · 2 topics')
  })
})
