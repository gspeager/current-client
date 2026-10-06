import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FileDiff } from '@current-client-bindings/app'
import DiffViewer from './DiffViewer'
import { DiffWrapContext, DiffWrapToggle } from './diffWrap'

const diff = new FileDiff({
  hunks: [
    {
      header: '@@ -1 +1 @@',
      raw: 'raw',
      lines: [{ kind: 'added', oldLine: 0, newLine: 1, content: 'a very long line', moved: false }],
    },
  ],
})

function renderWithWrap(wrap: boolean, setWrap = vi.fn()) {
  const view = render(
    <DiffWrapContext.Provider value={{ wrap, setWrap }}>
      <DiffWrapToggle />
      <DiffViewer diff={diff} path="a.ts" viewMode="split" />
    </DiffWrapContext.Provider>,
  )
  return { ...view, setWrap }
}

describe('diff word wrap', () => {
  it('wraps lines by default, with no saved choice', () => {
    const { container } = render(<DiffViewer diff={diff} path="a.ts" viewMode="split" />)

    expect(container.querySelector('.diff-viewer')).not.toHaveClass('diff-viewer-nowrap')
  })

  it('keeps each line on one row when wrapping is off', () => {
    const { container } = renderWithWrap(false)

    expect(container.querySelector('.diff-viewer')).toHaveClass('diff-viewer-nowrap')
    expect(screen.getByRole('checkbox', { name: 'Wrap lines' })).not.toBeChecked()
  })

  it('scrolls the two sides of a split diff together when wrapping is off', () => {
    const { container } = renderWithWrap(false)
    const [left, right] = container.querySelectorAll<HTMLElement>('.diff-side')

    left.scrollLeft = 120
    left.scrollTop = 40
    fireEvent.scroll(left)
    expect([right.scrollLeft, right.scrollTop]).toEqual([120, 40])

    right.scrollLeft = 10
    fireEvent.scroll(right)
    expect(left.scrollLeft).toBe(10)
  })

  it('saves the choice from the toggle', async () => {
    const { setWrap } = renderWithWrap(true)

    await userEvent.click(screen.getByRole('checkbox', { name: 'Wrap lines' }))

    expect(setWrap).toHaveBeenCalledWith(false)
  })
})
