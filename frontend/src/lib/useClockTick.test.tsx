import { act, render, screen } from '@testing-library/react'
import { relativeTime } from './relativeTime'
import { useClockTick } from './useClockTick'

function FetchedAgo({ at }: { at: Date }) {
  useClockTick()
  return <span>fetched {relativeTime(at)}</span>
}

describe('useClockTick', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('keeps a relative time current as the clock moves', () => {
    render(<FetchedAgo at={new Date()} />)
    expect(screen.getByText('fetched just now')).toBeInTheDocument()

    act(() => vi.advanceTimersByTime(90_000))

    expect(screen.getByText('fetched 1m ago')).toBeInTheDocument()
  })
})
