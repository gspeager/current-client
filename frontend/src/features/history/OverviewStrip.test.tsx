import { fireEvent, render, screen } from '@testing-library/react'
import { giveElementsLayout } from '../../test/bindings'
import OverviewStrip from './OverviewStrip'

const iso = (date: string) => new Date(`${date}T00:00:00`).toISOString()

const days = [
  { date: iso('2026-01-01'), commits: 2, firstRow: 8 },
  { date: iso('2026-01-11'), commits: 5, firstRow: 3 },
  { date: iso('2026-01-20'), commits: 1, firstRow: 0 },
]

beforeEach(() => {
  giveElementsLayout(1000)
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({ left: 0 } as DOMRect)
})

const strip = () => screen.getByRole('img', { name: 'Commit activity over time' })

describe('OverviewStrip', () => {
  it('jumps to the newest commit on the nearest day with commits', () => {
    const onJump = vi.fn()
    render(<OverviewStrip days={days} markers={[]} visible={null} onJump={onJump} />)

    fireEvent.click(strip(), { clientX: 10 })
    expect(onJump).toHaveBeenLastCalledWith(8)

    // Between the first two days, nearer the second.
    fireEvent.click(strip(), { clientX: 420 })
    expect(onJump).toHaveBeenLastCalledWith(3)

    fireEvent.click(strip(), { clientX: 999 })
    expect(onJump).toHaveBeenLastCalledWith(0)
  })

  it('colors each kind of marker, with HEAD alone in cyan', () => {
    const markers = [
      { kind: 'head', name: 'HEAD', date: iso('2026-01-20') },
      { kind: 'branch', name: 'main', date: iso('2026-01-20') },
      { kind: 'tag', name: 'v1', date: iso('2026-01-11') },
      { kind: 'stash', name: 'stash', date: iso('2026-01-05') },
      { kind: 'match', name: 'found', date: iso('2026-01-11') },
      { kind: 'tag', name: 'older', date: iso('2020-01-01') },
    ]
    const { container } = render(<OverviewStrip days={days} markers={markers} visible={null} onJump={vi.fn()} />)
    const fills = (kind: string) =>
      [...container.querySelectorAll(`[data-marker="${kind}"]`)].map((tick) => tick.getAttribute('fill'))

    expect(fills('head')).toEqual(['var(--primary)'])
    expect(fills('branch')[0]).not.toContain('--primary')
    expect(fills('stash')).toEqual(['var(--warning)'])
    expect(fills('match')).toEqual(['var(--warning)'])
    // A marker outside the listed range isn't drawn.
    expect(fills('tag')).toEqual(['var(--secondary)'])
  })

  it('draws activity as an open line, not a filled area', () => {
    const { container } = render(<OverviewStrip days={days} markers={[]} visible={null} onJump={vi.fn()} />)
    const path = container.querySelector('.overview-strip-line')?.getAttribute('d') ?? ''
    expect(path).toMatch(/^M0 /)
    expect(path).toContain('C')
    expect(path).not.toContain('Z')
  })

  it('keeps refs in short rows under the line, and HEAD and matches full height', () => {
    const markers = [
      { kind: 'head', name: 'HEAD', date: iso('2026-01-20') },
      { kind: 'match', name: 'found', date: iso('2026-01-11') },
      { kind: 'branch', name: 'main', date: iso('2026-01-20') },
      { kind: 'tag', name: 'v1', date: iso('2026-01-11') },
      { kind: 'stash', name: 'stash', date: iso('2026-01-05') },
    ]
    const { container } = render(<OverviewStrip days={days} markers={markers} visible={null} onJump={vi.fn()} />)
    const box = (kind: string) => {
      const tick = container.querySelector(`[data-marker="${kind}"]`)
      return { y: Number(tick?.getAttribute('y')), height: Number(tick?.getAttribute('height')) }
    }

    expect(box('head')).toEqual({ y: 0, height: 44 })
    expect(box('match')).toEqual({ y: 0, height: 44 })
    expect(box('branch')).toEqual({ y: 32, height: 4 })
    expect(box('tag')).toEqual({ y: 38, height: 4 })
    expect(box('stash')).toEqual({ y: 38, height: 4 })
  })

  it('draws markers over the range band, so it never hides them', () => {
    const at = Date.parse(iso('2026-01-11'))
    const { container } = render(
      <OverviewStrip
        days={days}
        markers={[{ kind: 'match', name: 'found', date: iso('2026-01-11') }]}
        visible={{ newest: at, oldest: at }}
        onJump={vi.fn()}
      />,
    )
    const band = screen.getByTestId('overview-window')
    const match = container.querySelector('[data-marker="match"]')!
    expect(band.compareDocumentPosition(match) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(Number(band.getAttribute('width'))).toBe(4)
  })

  it('draws one tick per pixel and color, however many refs share it', () => {
    const markers = Array.from({ length: 500 }, (_, i) => ({ kind: 'tag', name: `v${i}`, date: iso('2026-01-11') }))
    const { container } = render(<OverviewStrip days={days} markers={markers} visible={null} onJump={vi.fn()} />)
    expect(container.querySelectorAll('[data-marker="tag"]')).toHaveLength(1)
  })

  it('shows where the rows on screen fall, and follows them', () => {
    const at = (date: string) => Date.parse(iso(date))
    const { rerender } = render(
      <OverviewStrip
        days={days}
        markers={[]}
        visible={{ newest: at('2026-01-20'), oldest: at('2026-01-19') }}
        onJump={vi.fn()}
      />,
    )
    const left = () => Number(screen.getByTestId('overview-window').getAttribute('x'))
    const recent = left()

    rerender(
      <OverviewStrip
        days={days}
        markers={[]}
        visible={{ newest: at('2026-01-05'), oldest: at('2026-01-02') }}
        onJump={vi.fn()}
      />,
    )
    expect(left()).toBeLessThan(recent)
  })
})
