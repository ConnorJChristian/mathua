import Link from 'next/link'
import { API_BASE } from '../lib/apiBase'
import type { CourseText } from '../lib/api'

// One of a course's texts, linked to where it can be read (Q62/Q63): its
// library page when the library has it, the Lectures tab for lecture notes,
// the file itself for an HTML book (the library shelves PDFs only), and plain
// text only for an online book the library does not have.
export default function CourseTextLink({ text, arrow = false, onLectures }: { text: CourseText; arrow?: boolean; onLectures?: () => void }) {
  const link = 'text-mathua-primary hover:text-mathua-blue break-words [overflow-wrap:anywhere]'
  const tail = arrow ? ' →' : ''
  if (text.library_id) {
    return <Link href={`/library?book=${encodeURIComponent(text.library_id)}`} className={link}>{text.title}{tail}</Link>
  }
  if (text.kind === 'notes' && onLectures) {
    return <button type="button" onClick={onLectures} className={`${link} text-left`}>{text.title}{tail}</button>
  }
  if (text.kind === 'html' && text.url) {
    const href = text.url.startsWith('/') ? `${API_BASE}${text.url}` : text.url
    return <a href={href} target="_blank" rel="noopener noreferrer" className={link}>{text.title} ↗</a>
  }
  return <span className="text-mathua-secondary">{text.title}{text.kind === 'web' ? ' (online)' : ''}</span>
}
