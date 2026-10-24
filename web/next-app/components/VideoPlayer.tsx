'use client'

import { useState } from 'react'
import { API_BASE } from '../lib/apiBase'
import type { CourseVideo } from '../lib/api'

// Fork change: lecture videos in the course view. A module's videos.json
// attaches videos to a topic or a course: a YouTube link plays in an embedded
// player, a file in the module's files/ folder plays in the browser's own
// <video>, and any other link opens in a new tab. A list of videos shows one
// player with the rest as a playlist beside it.

function youtubeId(url: string): string {
  try {
    const u = new URL(url)
    if (u.hostname === 'youtu.be') return u.pathname.slice(1)
    if (u.pathname.startsWith('/embed/')) return u.pathname.slice('/embed/'.length)
    if (u.pathname.startsWith('/live/')) return u.pathname.slice('/live/'.length)
    return u.searchParams.get('v') ?? ''
  } catch {
    return ''
  }
}

function absolute(url: string): string {
  return /^https?:\/\//.test(url) ? url : `${API_BASE}${url}`
}

export function formatDuration(s?: number): string {
  if (!s) return ''
  const m = Math.floor(s / 60)
  const sec = s % 60
  return m >= 60 ? `${Math.floor(m / 60)}h ${m % 60}m` : `${m}:${String(sec).padStart(2, '0')}`
}

export default function VideoPlayer({ videos, title = 'Lecture' }: { videos: CourseVideo[]; title?: string }) {
  const [index, setIndex] = useState(0)
  if (!videos.length) return null
  const v = videos[Math.min(index, videos.length - 1)]
  const yt = v.kind === 'youtube' ? youtubeId(v.url) : ''
  const btn = 'border border-mathua-border px-2.5 min-h-[32px] font-mono text-[11px] text-mathua-secondary hover:border-mathua-blue hover:text-mathua-blue inline-flex items-center'
  return (
    <section className="border border-mathua-border bg-mathua-bg min-w-0" aria-label={title}>
      <div className="px-3 py-2 border-b border-mathua-border flex flex-wrap items-center justify-between gap-2">
        <div className="min-w-0">
          <p className="font-mono text-[10px] uppercase tracking-wider text-mathua-muted">{title}</p>
          <p className="font-mono text-xs text-mathua-primary break-words [overflow-wrap:anywhere]">{v.title}</p>
          {v.note && <p className="font-mono text-[10px] text-mathua-secondary">{v.note}</p>}
        </div>
        <a href={v.kind === 'file' ? absolute(v.url) : v.url} target="_blank" rel="noreferrer" className={btn}>Open in new tab</a>
      </div>
      <div className={`grid gap-0 ${videos.length > 1 ? 'lg:grid-cols-[minmax(0,1fr)_260px]' : ''}`}>
        <div className="bg-black aspect-video min-w-0">
          {yt ? (
            <iframe
              key={v.url}
              className="w-full h-full"
              src={`https://www.youtube-nocookie.com/embed/${encodeURIComponent(yt)}?rel=0${v.start ? `&start=${v.start}` : ''}`}
              title={v.title}
              allow="accelerometer; encrypted-media; gyroscope; picture-in-picture; fullscreen"
              allowFullScreen
            />
          ) : v.kind === 'file' ? (
            <video key={v.url} className="w-full h-full" controls preload="metadata" src={`${absolute(v.url)}${v.start ? `#t=${v.start}` : ''}`}>
              Your browser cannot play this file. <a href={absolute(v.url)}>Download it</a>.
            </video>
          ) : (
            <div className="w-full h-full flex items-center justify-center">
              <a href={v.url} target="_blank" rel="noreferrer" className="border border-white/60 text-white px-4 py-2 font-mono text-xs">Open the video ↗</a>
            </div>
          )}
        </div>
        {videos.length > 1 && (
          <ol className="divide-y divide-mathua-border border-t lg:border-t-0 lg:border-l border-mathua-border max-h-[50vh] overflow-auto" aria-label="Playlist">
            {videos.map((x, i) => (
              <li key={`${x.url}-${i}`}>
                <button
                  type="button"
                  onClick={() => setIndex(i)}
                  aria-current={i === index}
                  className={`w-full text-left px-3 py-2 font-mono text-[11px] hover:bg-mathua-blue-faint min-w-0 ${i === index ? 'bg-mathua-blue-faint text-mathua-blue' : 'text-mathua-primary'}`}
                >
                  <span className="text-mathua-muted mr-2">{i + 1}.</span>
                  <span className="break-words [overflow-wrap:anywhere]">{x.title}</span>
                  {x.duration ? <span className="ml-2 text-mathua-muted">{formatDuration(x.duration)}</span> : null}
                </button>
              </li>
            ))}
          </ol>
        )}
      </div>
    </section>
  )
}
