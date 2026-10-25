'use client'

import { useEffect, useMemo, Suspense } from 'react'
import { useRouter } from 'next/navigation'
import { useTheme } from '../../hooks/useTheme'
import Header from '../../components/Header'
import BottomTabs from '../../components/BottomTabs'
import Footer from '../../components/Footer'
import Loading from '../../components/Loading'
import QuizHost from './QuizHost'

// This route is the quiz. The domain picker and the goals diagnostic used to be
// rendered here as well, which meant two copies of the same screen with the same
// scope count worded differently in each -- and only one of them got fixed. They
// live in /onboard now, and a visit without ?quiz=1 goes there.
//
// DiagnosticHost still sits in this folder and still has a test suite, but nothing
// renders it any more: /onboard runs the diagnostic inline. Deleting it means
// porting that coverage onto the inline implementation first, which is its own
// change and not something to do by deleting files.

function GoalsContent() {
  const { mounted } = useTheme()
  const { replace } = useRouter()

  const isQuizMode = useMemo(
    () => typeof window !== 'undefined' && new URLSearchParams(window.location.search).get('quiz') === '1',
    [],
  )
  // Fork change: ?course=<id>&scope=review|full is that course's review test.
  const courseTest = useMemo(() => {
    if (typeof window === 'undefined') return { courseId: undefined, scope: 'review' as const }
    const q = new URLSearchParams(window.location.search)
    return { courseId: q.get('course') || undefined, scope: q.get('scope') === 'full' ? ('full' as const) : ('review' as const) }
  }, [])

  useEffect(() => {
    if (!mounted || isQuizMode) return
    replace('/onboard')
  }, [mounted, isQuizMode, replace])

  if (!mounted) return <div style={{ background: 'var(--bg)', minHeight: '100vh' }} />

  // Until the redirect lands, render nothing rather than a picker that is no
  // longer this route's job.
  if (!isQuizMode) return <div style={{ background: 'var(--bg)', minHeight: '100vh' }} />

  return (
    <>
      <Header />
      <div className="max-w-container mx-auto px-4 sm:px-6 pb-[calc(80px+env(safe-area-inset-bottom))] lg:pb-0 overflow-x-hidden min-w-0">
        <section className="pt-8 min-w-0 overflow-hidden">
          <QuizHost courseId={courseTest.courseId} scope={courseTest.scope} />
        </section>
      </div>
      <Footer />
      <BottomTabs />
    </>
  )
}

export default function GoalsPage() {
  return (
    <Suspense fallback={<><div style={{ background: 'var(--bg)', minHeight: '100vh' }}><Loading label="LOADING" /></div></>}>
      <GoalsContent />
    </Suspense>
  )
}
