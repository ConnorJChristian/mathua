'use client'

// The answer area for a Python-track question. One component for every host
// (Learn, review, quiz), choosing its input from the question itself:
//
//   code task  -> a code box prefilled with the starter, with Run and Reset
//   output     -> a monospace field for what the snippet prints
//   recall     -> Show answer, then "I had it" / "I missed it" (self-graded)
//   multiple choice -> nothing here; the host's ChoiceOptions is the input
//
// The code box is a plain textarea on purpose. It has to work on a phone, with
// a screen reader and with the browser's own undo, and a learner writing five
// lines of NumPy gains little from an editor that costs a megabyte.

import { useEffect, useMemo, useRef, useState, type KeyboardEvent } from 'react'
import { runPython, type PythonRunRes } from '../../lib/api'
import { pythonKind, recallBack, starterCode, taskKey } from '../../lib/pythonQuestion'
import { Play } from '../icons'
import KatexContent from '../KatexContent'

interface Props {
  question: string
  value: string
  onChange: (v: string) => void
  onSubmit?: () => void
  disabled?: boolean
  id?: string
}

const INDENT = '    '

export default function PythonAnswer({ question, value, onChange, onSubmit, disabled, id }: Props) {
  const kind = useMemo(() => pythonKind(question), [question])
  if (kind === 'mc') return null
  if (kind === 'recall') {
    return <RecallCard question={question} value={value} onChange={onChange} disabled={disabled} />
  }
  if (kind === 'output') {
    return <OutputField value={value} onChange={onChange} onSubmit={onSubmit} disabled={disabled} id={id} />
  }
  return <CodeBox question={question} value={value} onChange={onChange} onSubmit={onSubmit} disabled={disabled} id={id} />
}

// A flashcard. The back is part of the served question (a flashcard has no
// secret to keep), shown only when the learner asks; the verdict is theirs.
function RecallCard({ question, value, onChange, disabled }: Omit<Props, 'onSubmit' | 'id'>) {
  const [shown, setShown] = useState(false)
  useEffect(() => setShown(false), [question])
  const back = useMemo(() => recallBack(question), [question])
  if (!shown) {
    return (
      <button
        type="button"
        onClick={() => setShown(true)}
        disabled={disabled}
        className="self-start border border-mathua-border-strong px-4 h-10 font-mono text-xs text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue disabled:opacity-50"
      >
        Show answer
      </button>
    )
  }
  const pick = (v: 'right' | 'wrong', label: string) => (
    <button
      type="button"
      onClick={() => onChange(v)}
      disabled={disabled}
      aria-pressed={value === v}
      className={`border px-4 min-h-[44px] font-mono text-xs disabled:opacity-50 ${
        value === v ? 'border-mathua-blue text-mathua-blue' : 'border-mathua-border-strong text-mathua-primary hover:border-mathua-blue'
      }`}
    >
      {label}
    </button>
  )
  return (
    <div className="w-full min-w-0">
      <div className="border border-mathua-border bg-mathua-bg px-4 py-3">
        <KatexContent className="text-sm text-mathua-primary">{back}</KatexContent>
      </div>
      <div role="group" aria-label="How did you do?" className="mt-2 flex flex-wrap gap-2">
        {pick('right', 'I had it')}
        {pick('wrong', 'I missed it')}
      </div>
    </div>
  )
}

function OutputField({ value, onChange, onSubmit, disabled, id }: Omit<Props, 'question'>) {
  return (
    <textarea
      id={id}
      value={value}
      onChange={e => onChange(e.target.value)}
      onKeyDown={e => {
        // Enter checks; Shift+Enter adds a line for multi-line output.
        if (e.key === 'Enter' && !e.shiftKey) {
          e.preventDefault()
          onSubmit?.()
        }
      }}
      rows={2}
      disabled={disabled}
      spellCheck={false}
      autoCapitalize="off"
      autoCorrect="off"
      placeholder="What it prints"
      aria-label="What the code prints"
      className="w-full min-h-[48px] resize-y border border-mathua-border-strong bg-mathua-bg px-3 py-2 font-mono text-sm text-mathua-primary focus:border-mathua-blue focus:outline-none disabled:opacity-60"
    />
  )
}

function CodeBox({ question, value, onChange, onSubmit, disabled, id }: Props) {
  const starter = useMemo(() => starterCode(question), [question])
  const task = useMemo(() => taskKey(question), [question])
  const areaRef = useRef<HTMLTextAreaElement>(null)
  const [running, setRunning] = useState(false)
  const [result, setResult] = useState<PythonRunRes | null>(null)
  const [runError, setRunError] = useState('')

  // A new question arrives with an empty answer: seed it with the starter code,
  // which is also what makes the host's Check button usable.
  useEffect(() => {
    if (value === '' && starter) onChange(starter)
    setResult(null)
    setRunError('')
    // The starter is a function of the question; value is read only to see
    // whether the learner has typed anything yet.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [question])

  const lines = value.split('\n').length
  const rows = Math.min(24, Math.max(6, lines + 1))

  function edit(next: string, caret: number) {
    onChange(next)
    requestAnimationFrame(() => {
      const el = areaRef.current
      if (el) el.selectionStart = el.selectionEnd = caret
    })
  }

  // Insert at the caret through the browser's own editing command where it
  // exists, so Ctrl+Z still undoes an indent or a newline. Setting the value
  // from script would wipe the textarea's undo history.
  function insert(el: HTMLTextAreaElement, text: string) {
    el.focus()
    const ok = typeof document.execCommand === 'function' && document.execCommand('insertText', false, text)
    if (!ok) {
      const { selectionStart: a, selectionEnd: b } = el
      edit(value.slice(0, a) + text + value.slice(b), a + text.length)
    }
  }

  function onKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    const el = e.currentTarget
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
      e.preventDefault()
      onSubmit?.()
      return
    }
    if (e.key === 'Tab' && !e.shiftKey) {
      // Tab indents, as in any editor. Escape then Tab still leaves the box,
      // because Escape blurs it first (below).
      e.preventDefault()
      insert(el, INDENT)
      return
    }
    if (e.key === 'Tab' && e.shiftKey) {
      e.preventDefault()
      const a = el.selectionStart
      const lineStart = value.lastIndexOf('\n', a - 1) + 1
      const head = value.slice(lineStart, lineStart + INDENT.length)
      const remove = head.length - head.trimStart().length
      if (remove > 0) edit(value.slice(0, lineStart) + value.slice(lineStart + remove), Math.max(lineStart, a - remove))
      return
    }
    if (e.key === 'Enter' && !e.shiftKey) {
      // Keep the current line's indentation, and indent after a colon.
      e.preventDefault()
      const a = el.selectionStart
      const lineStart = value.lastIndexOf('\n', a - 1) + 1
      const line = value.slice(lineStart, a)
      let indent = line.match(/^\s*/)?.[0] ?? ''
      if (line.trimEnd().endsWith(':')) indent += INDENT
      insert(el, '\n' + indent)
      return
    }
    if (e.key === 'Escape') {
      el.blur()
    }
  }

  async function run() {
    setRunning(true)
    setRunError('')
    try {
      setResult(await runPython(value, task))
    } catch (err) {
      setResult(null)
      setRunError(err instanceof Error ? err.message : 'Could not run the code')
    } finally {
      setRunning(false)
    }
  }

  return (
    <div className="w-full min-w-0">
      <textarea
        ref={areaRef}
        id={id}
        value={value}
        onChange={e => onChange(e.target.value)}
        onKeyDown={onKeyDown}
        rows={rows}
        disabled={disabled}
        spellCheck={false}
        autoCapitalize="off"
        autoCorrect="off"
        autoComplete="off"
        wrap="off"
        aria-label="Your Python code"
        className="w-full resize-y overflow-x-auto border border-mathua-border-strong bg-mathua-code px-3 py-2 font-mono text-[13px] leading-5 text-mathua-primary focus:border-mathua-blue focus:outline-none disabled:opacity-60 [tab-size:4]"
      />
      <div className="mt-2 flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={run}
          disabled={disabled || running || !value.trim()}
          className="inline-flex items-center gap-1.5 border border-mathua-border-strong px-4 h-10 font-mono text-xs text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue disabled:opacity-50"
        >
          <Play aria-hidden="true" />
          {running ? 'Running…' : 'Run'}
        </button>
        <button
          type="button"
          onClick={() => {
            onChange(starter)
            setResult(null)
          }}
          disabled={disabled || running || value === starter}
          className="border border-mathua-border px-4 h-10 font-mono text-xs text-mathua-muted hover:border-mathua-blue hover:text-mathua-blue disabled:opacity-50"
        >
          Reset
        </button>
      </div>
      {runError && <p role="alert" className="mt-2 font-mono text-[11px] text-red-400">{runError}</p>}
      {result && <RunOutput result={result} />}
    </div>
  )
}

function RunOutput({ result }: { result: PythonRunRes }) {
  const nothing = !result.stdout && !result.error && !result.timed_out && result.figures.length === 0
  return (
    <div role="status" aria-live="polite" className="mt-2 border border-mathua-border bg-mathua-bg">
      <p className="border-b border-mathua-border px-3 py-1 font-mono text-[10px] uppercase tracking-wider text-mathua-muted">Output</p>
      {result.timed_out && (
        <p className="px-3 py-2 font-mono text-xs text-red-400">
          Stopped after {result.time_limit} seconds. Look for a loop that never ends.
        </p>
      )}
      {result.stdout && (
        <pre className="max-h-64 overflow-auto px-3 py-2 font-mono text-xs text-mathua-primary whitespace-pre">
          {result.stdout}
          {result.stdout_truncated ? '\n… output truncated' : ''}
        </pre>
      )}
      {result.error && (
        <pre className="max-h-64 overflow-auto px-3 py-2 font-mono text-xs text-red-400 whitespace-pre-wrap">{result.error}</pre>
      )}
      {result.figures.map((png, i) => (
        // eslint-disable-next-line @next/next/no-img-element -- an inline data URL from the runner, not a static asset
        <img key={i} src={`data:image/png;base64,${png}`} alt={`Figure ${i + 1} produced by your code`} className="mx-3 my-2 max-w-full bg-white" />
      ))}
      {nothing && <p className="px-3 py-2 font-mono text-xs text-mathua-muted">The code ran and printed nothing.</p>}
    </div>
  )
}
