import type { Metadata } from 'next'
import Link from 'next/link'
import Header from '../../components/Header'
import Footer from '../../components/Footer'
import BottomTabs from '../../components/BottomTabs'
import ContactBox from '../../components/ContactBox'

export const metadata: Metadata = {
  title: 'Contact — Mathua',
  description:
    'How to report a problem with a question, a bug in the site, or your account, and who reads it.',
}

const monoFont = "var(--font-jetbrains-mono), 'JetBrains Mono', monospace"
const bodyFont = "var(--font-space-grotesk), 'Space Grotesk', serif"

const textStyle: React.CSSProperties = {
  fontFamily: bodyFont,
  fontSize: '0.95rem',
  color: 'var(--text-secondary)',
  lineHeight: 1.85,
  marginBottom: '1rem',
}

const h2Style: React.CSSProperties = {
  fontFamily: bodyFont,
  fontWeight: 400,
  fontSize: '1.3rem',
  color: 'var(--text-primary)',
  marginTop: '2.5rem',
  marginBottom: '1rem',
  borderBottom: '0.5px solid var(--border)',
  paddingBottom: '0.5rem',
}

const linkStyle: React.CSSProperties = {
  color: 'var(--text-primary)',
  textDecoration: 'underline',
}

// The public route to a person.
//
// Three problems arrive here and only one of them belongs in the in-app report box:
// a wrong expected answer needs the captured question to be reproducible at all, a
// broken account needs a channel that is not public, and a code bug is best served
// by an issue tracker where a fix can be referenced. Saying which is which is the
// whole job of this page, because a learner who sends an account problem to a public
// tracker has published it.
//
// No email address: an address on a page is harvested. The box has no separate
// inbox either -- it writes into the report queue, the page the administrator opens
// whenever a question is reported, so a message cannot quietly pile up somewhere
// nobody looks. X and the issue tracker are the two-way channels.
export default function ContactPage() {
  return (
    <>
      <Header />
      <div className="max-w-[680px] mx-auto px-4 sm:px-6 py-12 sm:py-20 pb-[calc(80px+env(safe-area-inset-bottom))] lg:pb-20 overflow-x-hidden min-w-0">
        <Link href="/" style={{ fontFamily: monoFont, fontSize: '13px', color: 'var(--text-muted)' }}>
          ← Back
        </Link>
        <div style={{ fontFamily: monoFont, fontSize: '12px', color: 'var(--text-muted)', marginTop: '1.5rem', marginBottom: '0.5rem' }}>
          WHO TO TELL
        </div>
        <h1
          style={{
            fontFamily: "var(--font-space-grotesk), 'Space Grotesk', serif",
            fontWeight: 400,
            fontSize: '1.9rem',
            color: 'var(--text-primary)',
            marginBottom: '2rem',
          }}
        >
          Contact
        </h1>

        <p style={textStyle}>
          This is a personal install of a fork of Mathua. Its administrator is whoever runs
          it, so what you write here lands in this install&rsquo;s own admin queue, not with
          the upstream Mathua project.
        </p>

        <h2 style={h2Style}>Leave a note</h2>
        <p style={textStyle}>
          Use the box to file a note for later: something to fix, a course to add, a source
          to check. It reaches the same queue as a report about a question.
        </p>
        <ContactBox />

        <h2 style={h2Style}>A question is wrong</h2>
        <p style={textStyle}>
          Use the report control on the question itself. Choose <em>Report a problem</em>, then
          say what you expected. The report keeps the question, the answer you gave, the answer
          Mathua expected and how your answer was being read, so the problem can be replayed
          exactly instead of guessed at from a description.
        </p>
        <p style={textStyle}>
          That is the right route for a wrong expected answer, a question that cannot be typed
          into the box, a figure that does not load, or a mistake in an explanation.
        </p>

        <h2 style={h2Style}>A bug in upstream Mathua</h2>
        <p style={textStyle}>
          If the bug is in Mathua itself rather than in this fork&rsquo;s courses or changes,
          and it is safe to discuss in public, it can go to the upstream project&rsquo;s
          tracker:{' '}
          <a
            href="https://github.com/chuma-beep/mathua/issues"
            target="_blank"
            rel="noreferrer"
            style={linkStyle}
          >
            github.com/chuma-beep/mathua/issues
          </a>
          . Include the page address, what you did, what you expected and what happened.
        </p>

        <p style={textStyle}>
          Never send your password to anyone. Nothing in Mathua will ask for it.
        </p>

        <h2 style={h2Style}>What not to send</h2>
        <p style={textStyle}>
          Do not put an email address, a password or anything private in a public issue. Mathua
          will not ask for them.
        </p>

      </div>
      <Footer />
      <BottomTabs />
    </>
  )
}
