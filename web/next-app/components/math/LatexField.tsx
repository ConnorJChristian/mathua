'use client'

// Typed-LaTeX answer entry (fork change).
//
// For a learner who writes LaTeX faster than they drive a visual editor. The
// learner types source (`\frac{3y^2}{2}\,dx`), sees it rendered underneath, and
// submits the same plain answer the visual editor would have sent: MathLive's own
// LaTeX -> ASCIIMath converter, then the same toPlainAnswer shim. So both modes are
// graded identically. Until MathLive's converter has loaded (or if it fails to
// load), the raw LaTeX is sent, which the server also reads (NormalizeTypedMath
// handles \frac, ^{}, \sqrt, \cdot, Greek and spacing commands).

import { useEffect, useRef, useState, type MutableRefObject } from 'react'
import KatexContent from '@/components/KatexContent'
import { toPlainAnswer } from './plainAnswer'
import type { MathInputValue } from './types'

type Converter = (latex: string) => string

let converter: Converter | null = null
let loading: Promise<Converter | null> | null = null

function loadConverter(): Promise<Converter | null> {
  if (converter) return Promise.resolve(converter)
  if (!loading) {
    loading = import('mathlive')
      .then((m) => {
        const fn = (m as unknown as { convertLatexToAsciiMath?: Converter }).convertLatexToAsciiMath
        converter = typeof fn === 'function' ? fn : null
        return converter
      })
      .catch(() => null)
  }
  return loading
}

/** The plain answer for typed LaTeX: the visual editor's pipeline when available. */
export function latexToPlain(latex: string, convert: Converter | null): string {
  const src = latex.trim()
  if (!src) return ''
  if (!convert) return src
  try {
    return toPlainAnswer(convert(src))
  } catch {
    return src
  }
}

interface Props {
  value: string
  onChange: (v: MathInputValue) => void
  onSubmit?: () => void
  disabled?: boolean
  placeholder?: string
  ariaLabel?: string
  id?: string
  inputRef?: MutableRefObject<HTMLInputElement | null>
}

export default function LatexField({ value, onChange, onSubmit, disabled, placeholder, ariaLabel, id, inputRef }: Props) {
  const [text, setText] = useState(value)
  const [ready, setReady] = useState<Converter | null>(converter)
  const textRef = useRef(text)
  textRef.current = text

  useEffect(() => {
    let live = true
    void loadConverter().then((c) => {
      if (!live) return
      setReady(() => c)
      // Re-send what is already typed through the real converter once it arrives.
      if (textRef.current) onChange({ latex: textRef.current, plainAnswer: latexToPlain(textRef.current, c) })
    })
    return () => {
      live = false
    }
    // onChange is the host's setter; re-running on its identity would resend on every render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const update = (next: string) => {
    setText(next)
    onChange({ latex: next, plainAnswer: latexToPlain(next, ready) })
  }

  return (
    <div className="w-full min-w-0">
      <input
        ref={inputRef}
        id={id}
        type="text"
        value={text}
        disabled={disabled}
        placeholder={placeholder ?? 'Type LaTeX, e.g. \\frac{1}{2}x^{2} + 3'}
        aria-label={ariaLabel ?? 'Your answer (LaTeX)'}
        spellCheck={false}
        autoCapitalize="off"
        autoCorrect="off"
        autoComplete="off"
        onChange={(e) => update(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault()
            onSubmit?.()
          }
        }}
        className="w-full min-w-0 min-h-12 px-3 bg-mathua-code font-mono text-base text-mathua-primary border border-mathua-border focus:border-mathua-blue outline-none"
      />
      <div className="mt-1.5 min-h-[28px] px-1" aria-live="polite" data-testid="latex-preview">
        {text.trim() ? (
          <KatexContent className="text-sm text-mathua-primary">{`\\(${text}\\)`}</KatexContent>
        ) : (
          <span className="font-mono text-[11px] text-mathua-muted">Preview appears here.</span>
        )}
      </div>
    </div>
  )
}
