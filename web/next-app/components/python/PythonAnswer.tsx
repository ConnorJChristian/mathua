'use client'

// The answer area for a Python-track question. One component for every host
// (Learn, review, quiz), choosing its input from the question itself:
//
//   code task  -> a code box prefilled with the starter, with Run and Reset
//   output     -> a monospace field for what the snippet prints
//   recall     -> Show answer, then "I had it" / "I missed it" (self-graded)
//   math       -> one math input per answer field, the same editor Mathua's
//                 own questions use, graded by Mathua's math graders
//   term       -> a one-line text field for the concept's name
//   proof      -> a multi-line box for a written proof, with a rendered
//                 preview of its LaTeX; graded on the server (internal/proof)
//   written-answer flashcard -> the proof box, graded the same way, with a
//                 "Just show me the answer" escape to the self-graded card
//   multiple choice -> nothing here; the host's ChoiceOptions is the input
//
// The code box is a plain textarea on purpose. It has to work on a phone, with
// a screen reader and with the browser's own undo, and a learner writing five
// lines of NumPy gains little from an editor that costs a megabyte.

import { useEffect, useMemo, useRef, useState, type KeyboardEvent } from 'react'
import { runPython, type PythonRunRes } from '../../lib/api'
import { isWritingQuestion, isWrittenRecall, pythonKind, recallBack, starterCode, taskKey, typedFields, typedValues, wordCount } from '../../lib/pythonQuestion'
import { MathAnswerInput, type MathFocusHandle } from '../math/MathInput'
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
  if (kind === 'proof') {
    if (isWrittenRecall(question)) {
      return <WrittenRecall question={question} value={value} onChange={onChange} onSubmit={onSubmit} disabled={disabled} id={id} />
    }
    if (isWritingQuestion(question)) {
      return <ProofBox value={value} onChange={onChange} onSubmit={onSubmit} disabled={disabled} id={id} label="Your writing" />
    }
    return <ProofBox value={value} onChange={onChange} onSubmit={onSubmit} disabled={disabled} id={id} />
  }
  if (kind === 'math') {
    return <TypedAnswer question={question} value={value} onChange={onChange} onSubmit={onSubmit} disabled={disabled} id={id} />
  }
  if (kind === 'recall') {
    return <RecallCard question={question} value={value} onChange={onChange} disabled={disabled} />
  }
  if (kind === 'output') {
    return <OutputField value={value} onChange={onChange} onSubmit={onSubmit} disabled={disabled} id={id} />
  }
  if (kind === 'term') {
    return <TermField value={value} onChange={onChange} onSubmit={onSubmit} disabled={disabled} id={id} />
  }
  return <CodeBox question={question} value={value} onChange={onChange} onSubmit={onSubmit} disabled={disabled} id={id} />
}

// One math input per answer field. The answers travel as one string, one per
// line, so every host keeps a plain string in state as it does for any other
// question. Enter moves to the next field, and checks from the last.
function TypedAnswer({ question, value, onChange, onSubmit, disabled, id }: Props) {
  const fields = useMemo(() => typedFields(question), [question])
  // One focus handle per field, so Enter can move to the next one.
  const handles = useMemo(() => fields.map(() => ({ current: null as MathFocusHandle | null })), [fields])
  const values = typedValues(value, fields.length)
  const set = (i: number, v: string) => {
    const next = values.slice()
    next[i] = v.replace(/\n/g, ' ')
    onChange(fields.length > 1 ? next.join('\n') : next[0])
  }
  return (
    <div className="flex w-full min-w-0 flex-col gap-2">
      {fields.map((f, i) => (
        <div key={i} className="flex min-w-0 items-center gap-3">
          {f.label ? (
            <KatexContent className="shrink-0 font-mono text-sm text-mathua-primary">{`\\(${f.label} =\\)`}</KatexContent>
          ) : null}
          <div className="min-w-0 flex-1">
            <MathAnswerInput
              id={i === 0 ? id : id ? `${id}-${i}` : undefined}
              value={values[i]}
              onChange={v => set(i, v)}
              onSubmit={() => {
                if (i < fields.length - 1) handles[i + 1].current?.focus()
                else onSubmit?.()
              }}
              gradingType={f.grading}
              disabled={disabled}
              placeholder="Your answer"
              ariaLabel={f.label ? `Answer for ${f.label}` : 'Your answer'}
              focusRef={handles[i]}
            />
          </div>
        </div>
      ))}
    </div>
  )
}

// A flashcard that asks for a proof or derivation (internal/pytrack
// written.go): the proof box, graded against a marking scheme, and an escape
// to the plain flashcard. Revealing the back is one-way: once the learner has
// read it, a written answer would grade nothing, so the box is cleared and
// only "I had it" / "I missed it" remain, exactly as for any flashcard.
function WrittenRecall({ question, value, onChange, onSubmit, disabled, id }: Props) {
  const [selfGrade, setSelfGrade] = useState(false)
  useEffect(() => setSelfGrade(false), [question])
  if (selfGrade) {
    return <RecallCard question={question} value={value} onChange={onChange} disabled={disabled} revealed />
  }
  return (
    <div className="flex w-full min-w-0 flex-col gap-2">
      <ProofBox value={value} onChange={onChange} onSubmit={onSubmit} disabled={disabled} id={id} label="Your answer" />
      <button
        type="button"
        onClick={() => {
          onChange('')
          setSelfGrade(true)
        }}
        disabled={disabled}
        className="self-start border border-mathua-border-strong px-4 min-h-[40px] font-mono text-xs text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue disabled:opacity-50"
      >
        Just show me the answer (self-grade)
      </button>
    </div>
  )
}

// A flashcard. The back is part of the served question (a flashcard has no
// secret to keep), shown only when the learner asks; the verdict is theirs.
function RecallCard({ question, value, onChange, disabled, revealed = false }: Omit<Props, 'onSubmit' | 'id'> & { revealed?: boolean }) {
  const [shown, setShown] = useState(revealed)
  useEffect(() => setShown(revealed), [question, revealed])
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

// A written proof. A plain textarea for the same reasons as the code box; the
// preview renders the LaTeX the learner typed, so a mistyped delimiter shows
// up before the proof is sent to be graded. Ctrl+Enter checks; Enter is a new
// line, because a proof has many.
function ProofBox({ value, onChange, onSubmit, disabled, id, label = 'Your proof' }: Omit<Props, 'question'> & { label?: string }) {
  const [preview, setPreview] = useState(false)
  const rows = Math.min(28, Math.max(10, value.split('\n').length + 2))
  const writing = label === 'Your writing'
  const words = wordCount(value)
  const tab = (on: boolean, label: string) => (
    <button
      type="button"
      onClick={() => setPreview(on)}
      aria-pressed={preview === on}
      className={`px-3 h-8 font-mono text-[11px] border ${
        preview === on ? 'border-mathua-blue text-mathua-blue' : 'border-mathua-border text-mathua-muted hover:text-mathua-blue'
      }`}
    >
      {label}
    </button>
  )
  return (
    <div className="w-full min-w-0">
      <div role="group" aria-label={label === 'Your proof' ? 'Proof view' : writing ? 'Writing view' : 'Answer view'} className="mb-1.5 flex items-center gap-1">
        {tab(false, 'Write')}
        {tab(true, 'Preview')}
        {writing && (
          <span data-word-count aria-live="polite" className="ml-auto font-mono text-[11px] text-mathua-muted">
            {words} {words === 1 ? 'word' : 'words'}
          </span>
        )}
      </div>
      {preview ? (
        <div data-proof-preview className="min-h-[12rem] border border-mathua-border bg-mathua-bg px-3 py-2">
          {value.trim() ? (
            <KatexContent className="text-sm text-mathua-primary whitespace-pre-wrap break-words">{value}</KatexContent>
          ) : (
            <p className="font-mono text-xs text-mathua-muted">Nothing written yet.</p>
          )}
        </div>
      ) : (
        <textarea
          id={id}
          value={value}
          onChange={e => onChange(e.target.value)}
          onKeyDown={e => {
            if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
              e.preventDefault()
              onSubmit?.()
            }
          }}
          rows={rows}
          disabled={disabled}
          spellCheck
          placeholder={
            label === 'Your proof'
              ? 'Proof. Suppose ...\n\nUse \\( ... \\) for mathematics, e.g. \\(|a_n - a| < \\epsilon\\).'
              : writing
              ? 'Write your text here, in full sentences and paragraphs.'
              : 'Your answer, with every step justified.\n\nUse \\( ... \\) for mathematics, e.g. \\(f\'(x) > 0\\).'
          }
          aria-label={label}
          className="w-full min-h-[12rem] resize-y border border-mathua-border-strong bg-mathua-bg px-3 py-2 font-mono text-sm leading-relaxed text-mathua-primary focus:border-mathua-blue focus:outline-none disabled:opacity-60"
        />
      )}
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

// One line for a term. Enter checks. Spell-check and autocapitalize stay on:
// the answer is a word in a natural language, not code.
function TermField({ value, onChange, onSubmit, disabled, id }: Omit<Props, 'question'>) {
  return (
    <input
      id={id}
      type="text"
      value={value}
      onChange={e => onChange(e.target.value)}
      onKeyDown={e => {
        if (e.key === 'Enter') {
          e.preventDefault()
          onSubmit?.()
        }
      }}
      disabled={disabled}
      autoComplete="off"
      placeholder="The term"
      aria-label="The term"
      className="w-full min-h-[48px] border border-mathua-border-strong bg-mathua-bg px-3 py-2 font-mono text-sm text-mathua-primary focus:border-mathua-blue focus:outline-none disabled:opacity-60"
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
