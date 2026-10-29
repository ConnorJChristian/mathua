import { describe, it, expect } from 'vitest'
import { courseTabs, hasLectures, ungradedTasksLine } from '../lib/courseTabs'
import type { CourseModule, CourseNote, CourseVideo } from '../lib/api'

const none = { notes: [] as CourseNote[], videos: [] as CourseVideo[], modules: [] as CourseModule[] }

// A course without lectures had a Lectures tab whose empty state was developer
// text ("When the course's module lists lectures (videos.json, see FORK.md)…").
describe('the course page tabs', () => {
  it('leave out Lectures when the course has no notes and no videos', () => {
    expect(courseTabs(none).map(t => t.label)).toEqual(['Home', 'Modules', 'Readings', 'Grades'])
  })
  it('keep Lectures for lecture notes, course videos or a topic video', () => {
    expect(hasLectures({ ...none, notes: [{ title: 'L1', url: '/x' }] })).toBe(true)
    expect(hasLectures({ ...none, videos: [{ title: 'V', url: 'https://youtu.be/x', kind: 'youtube' } as CourseVideo] })).toBe(true)
    const topic = { id: 't', label: 'T', status: 'unseen', locked: false, due: false, attempts: 0, videos: [{ title: 'V', url: 'u', kind: 'link' } as CourseVideo] }
    expect(courseTabs({ ...none, modules: [{ title: 'M', topics: [topic], mastered: 0 }] }).map(t => t.label)).toContain('Lectures')
  })
})

describe('the line for written tasks without a grader', () => {
  it('counts them, and says nothing when there are none', () => {
    expect(ungradedTasksLine(12)).toBe('12 writing/proof tasks need an API key to be graded.')
    expect(ungradedTasksLine(1)).toBe('1 writing/proof task needs an API key to be graded.')
    expect(ungradedTasksLine(0)).toBe('')
    expect(ungradedTasksLine(undefined)).toBe('')
  })
})
