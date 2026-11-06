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

  it('marks HEAD in cyan and leaves other refs in their own colors', () => {
    const markers = [
      { kind: 'head', name: 'HEAD', date: iso('2026-01-20') },
      { kind: 'branch', name: 'main', date: iso('2026-01-20') },
      { kind: 'tag', name: 'v1', date: iso('2026-01-11') },
      { kind: 'match', name: 'found', date: iso('2026-01-11') },
      { kind: 'tag', name: 'older', date: iso('2020-01-01') },
    ]
    const { container } = render(<OverviewStrip days={days} markers={markers} visible={null} onJump={vi.fn()} />)
    const fills = (kind: string) =>
      [...container.querySelectorAll(`[data-marker="${kind}"]`)].map((tick) => tick.getAttribute('fill'))

    expect(fills('head')).toEqual(['var(--primary)'])
    expect(fills('branch')[0]).not.toContain('--primary')
    expect(fills('match')).toHaveLength(1)
    // A marker outside the listed range isn't drawn.
    expect(fills('tag')).toHaveLength(1)
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
