import { fireEvent, render, screen } from '@testing-library/react'
import ScrollCue from './ScrollCue'

function renderIn(sizes: { scrollHeight: number; clientHeight: number; scrollTop?: number }) {
  const { container } = render(
    <div>
      <ScrollCue />
    </div>,
  )
  const scroller = container.firstElementChild as HTMLElement
  scroller.scrollBy = vi.fn()
  for (const [name, value] of Object.entries(sizes)) {
    Object.defineProperty(scroller, name, { configurable: true, value })
  }
  fireEvent.scroll(scroller)
  return scroller
}

const cue = () => screen.getByText('More below')

describe('ScrollCue', () => {
  it('shows while rows continue below and scrolls on to them', () => {
    const scroller = renderIn({ scrollHeight: 900, clientHeight: 500, scrollTop: 0 })
    expect(cue()).toHaveAttribute('data-more', 'true')

    fireEvent.click(cue())
    expect(scroller.scrollBy).toHaveBeenCalledWith({ top: 400 })
  })

  it('hides once only its own space remains', () => {
    renderIn({ scrollHeight: 900, clientHeight: 500, scrollTop: 370 })
    expect(cue()).toHaveAttribute('data-more', 'false')
  })
})
