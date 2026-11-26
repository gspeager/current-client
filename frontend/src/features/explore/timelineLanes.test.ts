import type { TimelineInfo } from '@current-client-bindings/app'
import { folderAreas, slotFinder } from './folderAreas'
import { areaFinder, linesIn, MOST_LANES, spanStart, timelineLanes } from './timelineLanes'
import { layoutTimeline, MAX_RADIUS, MIN_RADIUS, radiusFinder, timeTicks } from './timelineLayout'

const CODE = [
  { path: 'core/git/log.go', lines: 400 },
  { path: 'core/diff/diff.go', lines: 300 },
  { path: 'app/service.go', lines: 200 },
  { path: 'README.md', lines: 10 },
]
const slotOf = slotFinder(folderAreas(CODE))
const all = () => true
const file = (path: string, lines: number, kind = '') => ({ path, lines, kind })
const info = (over: Partial<TimelineInfo> = {}): TimelineInfo => ({
  commits: [],
  now: false,
  working: [],
  limit: 0,
  target: '',
  ...over,
})

describe('timelineLanes', () => {
  const commits = [
    {
      sha: 'b'.repeat(40),
      time: '2026-11-26T10:00:00Z',
      subject: 'later',
      files: [file('core/git/log.go', 30), file('app/service.go', 5), file('gone/old.go', 4)],
    },
    { sha: 'a'.repeat(40), time: '2026-11-26T09:00:00Z', subject: 'earlier', files: [file('core/diff/diff.go', 12)] },
  ]

  it("makes the folder's changed areas lanes, in the map's order and colors", () => {
    const { lanes, entries } = timelineLanes(info({ commits }), CODE, '', slotOf, all)
    expect(lanes.map((lane) => lane.path)).toEqual(['core/git', 'core/diff', 'app', null])
    expect(lanes.map((lane) => lane.slot)).toEqual([slotOf('core/git/x'), slotOf('core/diff/x'), slotOf('app/x'), 0])
    expect(entries.map((entry) => entry.subject)).toEqual(['earlier', 'later'])
    expect(entries[1].bubbles).toEqual([
      { lane: 0, lines: 30, path: 'core/git/log.go', most: 30 },
      { lane: 2, lines: 5, path: 'app/service.go', most: 5 },
      { lane: 3, lines: 4, path: 'gone/old.go', most: 4 },
    ])
    expect(linesIn(entries[1])).toBe('39 lines in 3 folders, most in core/git')
    expect(linesIn(entries[0])).toBe('12 lines in core/diff')
  })

  it('keeps to the chosen folder and what is shown', () => {
    const { lanes, entries } = timelineLanes(
      info({ commits }),
      CODE,
      'core',
      slotOf,
      (f) => f.path !== 'core/diff/diff.go',
    )
    expect(lanes.map((lane) => lane.path)).toEqual(['core/git'])
    expect(entries).toHaveLength(1)
    expect(entries[0].lines).toBe(30)
  })

  it('puts lanes past the tenth in one for the rest', () => {
    const code = Array.from({ length: 14 }, (_, i) => ({ path: `f${i}/x.go`, lines: 100 - i }))
    const many = [{ sha: 'c', time: '2026-11-26T09:00:00Z', subject: 'all', files: code.map((f) => file(f.path, 1)) }]
    const { lanes, entries } = timelineLanes(info({ commits: many }), code, '', slotFinder(folderAreas(code)), all)
    expect(lanes).toHaveLength(MOST_LANES + 1)
    expect(lanes[MOST_LANES]).toEqual({ path: null, slot: 0 })
    expect(entries[0].bubbles[MOST_LANES].lines).toBe(4)
  })

  it('adds the uncommitted changes when the range reaches now', () => {
    const working = [file('app/service.go', 8)]
    expect(timelineLanes(info({ working }), CODE, '', slotOf, all).entries).toHaveLength(0)
    const { entries } = timelineLanes(info({ now: true, working }), CODE, '', slotOf, all, 123)
    expect(entries).toEqual([
      expect.objectContaining({ sha: null, time: 123, lines: 8, bubbles: [expect.objectContaining({ lane: 0 })] }),
    ])
  })

  it('finds a file in the nearest area above it', () => {
    const areas = folderAreas([...CODE, { path: 'core/x.go', lines: 5 }])
    const areaOf = areaFinder(areas)
    const name = (path: string) => areas[areaOf(path)]?.path
    expect(name('core/git/deeper/y.go')).toBe('core/git')
    expect(name('core/new.go')).toBe('core')
    expect(name('README.md')).toBe('')
    expect(areaOf('elsewhere/z.go')).toBe(-1)
  })
})

describe('spanStart', () => {
  it('starts today at midnight and this week on Monday, local time', () => {
    const thursday = new Date(2026, 10, 26, 15, 30)
    expect(spanStart('today', thursday)).toEqual(new Date(2026, 10, 26))
    expect(spanStart('week', thursday)).toEqual(new Date(2026, 10, 23))
    const sunday = new Date(2026, 10, 29, 9)
    expect(spanStart('week', sunday)).toEqual(new Date(2026, 10, 23))
  })
})

describe('bubble sizes', () => {
  it("makes a bubble's area follow its lines, the largest few capped", () => {
    const radius = radiusFinder([...Array(19).fill(100), 10000])
    expect(radius(100)).toBe(MAX_RADIUS)
    expect(radius(10000)).toBe(MAX_RADIUS)
    expect(radius(25) ** 2 / radius(100) ** 2).toBeCloseTo(0.25)
    expect(radius(0)).toBe(MIN_RADIUS)
  })

  it('places bubbles on their lane at their time, the largest first', () => {
    const { lanes, entries } = timelineLanes(
      info({
        commits: [
          {
            sha: 'a',
            time: '2026-11-26T09:00:00Z',
            subject: 's',
            files: [file('app/a.go', 10), file('core/git/b.go', 40)],
          },
        ],
      }),
      CODE,
      '',
      slotOf,
      all,
    )
    const since = Date.parse('2026-11-26T08:00:00Z')
    const layout = layoutTimeline(
      lanes,
      entries,
      [{ name: 'v1', sha: 'a' }],
      { since, now: since + 2 * 3_600_000 },
      800,
    )
    expect(layout.bubbles.map((b) => b.lane)).toEqual([0, 1])
    expect(layout.bubbles[0].radius).toBeGreaterThan(layout.bubbles[1].radius)
    expect(layout.bubbles[0].y).toBeLessThan(layout.bubbles[1].y)
    const left = layout.gutter + MAX_RADIUS
    expect(layout.xs[0]).toBeCloseTo(left + (layout.now! - left) / 2)
    expect(layout.tags).toEqual([expect.objectContaining({ name: 'v1' })])
  })
})

describe('timeTicks', () => {
  it('steps by hours within a day and by days over a week', () => {
    const start = new Date(2026, 10, 26, 9, 20).getTime()
    const hours = timeTicks(start, new Date(2026, 10, 26, 15, 0).getTime(), 1000)
    expect(hours.map(({ time }) => new Date(time).getHours())).toEqual([10, 11, 12, 13, 14, 15])
    const days = timeTicks(new Date(2026, 10, 23).getTime(), new Date(2026, 10, 30).getTime(), 1000)
    expect(days.map(({ time }) => new Date(time).getDate())).toEqual([23, 24, 25, 26, 27, 28, 29, 30])
  })

  it('names midnight by its date among hours', () => {
    const ticks = timeTicks(new Date(2026, 10, 26, 22).getTime(), new Date(2026, 10, 27, 2).getTime(), 1000)
    const midnight = new Date(2026, 10, 27)
    expect(ticks.find(({ time }) => time === midnight.getTime())?.label).toBe(
      midnight.toLocaleString(undefined, { month: 'short', day: 'numeric' }),
    )
    expect(ticks[0].label).not.toBe(ticks[2].label)
  })
})
