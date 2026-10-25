// Local event log (fork change, opt-in). See internal/server/events.go.
//
// track() queues an interaction event; the queue is sent in batches to
// /api/events. The server records nothing unless the install sets
// MATHUA_EVENT_LOG_DIR, and the client asks once whether it does: when the
// log is off, track() is a no-op after that first answer, so a public install
// sends nothing beyond one GET.
import { API_BASE } from './apiBase'
import { getAuthHeaders } from './auth'

export interface TrackedEvent {
  kind: string
  concept?: string
  page?: string
  data?: Record<string, unknown>
}

let enabled: boolean | null = null
let checking: Promise<void> | null = null
let queue: (TrackedEvent & { ts: number })[] = []
let timer: ReturnType<typeof setTimeout> | null = null
const FLUSH_MS = 8000
const MAX_QUEUE = 500

function check(): Promise<void> {
  if (checking) return checking
  checking = fetch(`${API_BASE}/api/events`, { cache: 'no-store' })
    .then((r) => (r.ok ? r.json() : { enabled: false }))
    .then((d: { enabled?: boolean }) => {
      enabled = !!d.enabled
      if (!enabled) queue = []
    })
    .catch(() => {
      enabled = false
      queue = []
    })
  return checking
}

export function flushEvents(): void {
  if (timer) {
    clearTimeout(timer)
    timer = null
  }
  if (!enabled || queue.length === 0) return
  const batch = queue.splice(0, 200)
  try {
    void fetch(`${API_BASE}/api/events`, {
      method: 'POST',
      keepalive: true,
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({ events: batch }),
    }).catch(() => {})
  } catch {
    // An event that cannot be sent is lost; tracking never gets in the way.
  }
  if (queue.length) schedule()
}

function schedule(): void {
  if (!timer) timer = setTimeout(flushEvents, FLUSH_MS)
}

export function track(e: TrackedEvent): void {
  if (typeof window === 'undefined' || enabled === false) return
  const ev = { ...e, page: e.page ?? window.location.pathname, ts: Date.now() }
  if (queue.length < MAX_QUEUE) queue.push(ev)
  if (enabled === null) {
    void check().then(() => enabled && schedule())
    return
  }
  schedule()
}

// A short, stable fingerprint of a question's text, so events about the same
// item can be joined to the attempts table without storing the text twice.
export function questionHash(text: string): string {
  let h = 2166136261
  for (let i = 0; i < text.length; i++) {
    h ^= text.charCodeAt(i)
    h = Math.imul(h, 16777619)
  }
  return (h >>> 0).toString(36)
}

// How an answer was typed: the style profile the simulated learner copies.
export function answerStyle(answer: string): Record<string, unknown> {
  return {
    len: answer.length,
    latex: /\\[a-z]+|[\^_]\{/.test(answer),
    unicode: /[^\x00-\x7f]/.test(answer),
    spaces: /\s/.test(answer.trim()),
    fraction: /\d\s*\/\s*\d/.test(answer),
    decimal: /\d\.\d/.test(answer),
    letter_only: /^[A-Ea-e]$/.test(answer.trim()),
  }
}

if (typeof window !== 'undefined') {
  window.addEventListener('pagehide', flushEvents)
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'hidden') flushEvents()
  })
}
