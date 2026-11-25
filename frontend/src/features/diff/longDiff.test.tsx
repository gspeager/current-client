import { fireEvent, render, screen } from '@testing-library/react'
import { FileDiff, ReviewThread, type DiffLine } from '@current-client-bindings/app'
import { giveElementsLayout } from '../../test/bindings'
import DiffViewer from './DiffViewer'
import { DiffWrapContext } from './diffWrap'

const LINES = 50_000

const changed = (n: number): DiffLine => ({
  kind: 'added',
  oldLine: 0,
  newLine: n,
  content: `line ${n}`,
  moved: false,
  inOurs: false,
  inTheirs: false,
})

const diffOf = (lines: number) =>
  new FileDiff({
    hunks: [{ header: '@@ long @@', raw: 'raw', lines: Array.from({ length: lines }, (_, i) => changed(i + 1)) }],
  })

const longDiff = diffOf(LINES)

function renderDiff(
  props: Partial<Parameters<typeof DiffViewer>[0]> = {},
  { wrap = false, viewMode = 'unified' as 'unified' | 'split' } = {},
) {
  const view = render(
    <DiffWrapContext.Provider value={{ wrap, setWrap: () => undefined }}>
      <DiffViewer diff={longDiff} path="a.ts" viewMode={viewMode} onStageHunk={() => undefined} {...props} />
    </DiffWrapContext.Provider>,
  )
  const pane = view.container.querySelector<HTMLElement>('.diff-pane')!
  return { ...view, pane, scroller: pane.querySelector<HTMLElement>('.diff-side') ?? pane }
}

const heightOf = (pane: HTMLElement) =>
  [...pane.children].reduce(
    (sum, el) => sum + (el.classList.contains('diff-gap') ? parseFloat((el as HTMLElement).style.height) : 20),
    0,
  )

const scrollTo = (scroller: HTMLElement, top: number) => fireEvent.scroll(scroller, { target: { scrollTop: top } })

beforeEach(() => {
  giveElementsLayout()
  vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockReturnValue(1000)
  vi.spyOn(HTMLElement.prototype, 'scrollHeight', 'get').mockReturnValue((LINES + 1) * 20)
  Element.prototype.scrollTo = vi.fn(function (this: Element, options?: ScrollToOptions | number) {
    if (typeof options === 'object') scrollTo(this as HTMLElement, options.top ?? 0)
  }) as typeof Element.prototype.scrollTo
})

describe('a 50,000-line diff', () => {
  it('draws only the rows near the screen, at its full height', () => {
    const { pane } = renderDiff()
    expect(pane.querySelectorAll('.diff-row').length).toBeLessThan(200)
    expect(heightOf(pane)).toBe((LINES + 1) * 20)
  })

  it('draws the rows it is scrolled to', () => {
    const { scroller } = renderDiff()
    scrollTo(scroller, 600_000)
    expect(screen.getByText('line 30000')).toBeInTheDocument()
    expect(screen.queryByText('line 1')).toBeNull()
  })

  it('draws the same rows on both sides of a split view with wrapping off', () => {
    const { container, scroller } = renderDiff({}, { viewMode: 'split' })
    scrollTo(scroller, 600_000)
    const [left, right] = [...container.querySelectorAll('.diff-side-content')].map((side) =>
      [...side.children].map((el) => el.getAttribute('data-index') ?? (el as HTMLElement).style.height),
    )
    expect(left.length).toBeGreaterThan(10)
    expect(left).toEqual(right)
  })

  it('keeps the focused line and its selection while scrolled away', () => {
    const { scroller } = renderDiff()
    const line = screen.getByRole('button', { name: 'Select added line 3' })
    fireEvent.click(line)
    line.focus()
    scrollTo(scroller, 600_000)
    expect(screen.getByRole('button', { name: 'Select added line 3' })).toHaveFocus()
    expect(screen.getByText('line 30000')).toBeInTheDocument()
    scrollTo(scroller, 0)
    expect(screen.getByRole('button', { name: 'Select added line 3' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('keeps every review thread, so a fold or a reply survives scrolling', () => {
    const thread = new ReviewThread({
      id: 't1',
      line: 40_000,
      notes: [{ author: 'Ada', createdAt: '2026-11-01T00:00:00Z', bodyHtml: '<p>Why?</p>' }],
    })
    const { scroller } = renderDiff({ threads: [thread] })
    const head = screen.getByRole('button', { name: /Ada/ })
    fireEvent.click(head)
    scrollTo(scroller, 900_000)
    scrollTo(scroller, 0)
    expect(screen.getByRole('button', { name: /Ada/ })).toHaveAttribute('aria-expanded', 'false')
  })

  it('opens at a line that was not drawn', () => {
    const { rerender } = renderDiff()
    rerender(
      <DiffWrapContext.Provider value={{ wrap: false, setWrap: () => undefined }}>
        <DiffViewer diff={longDiff} path="a.ts" viewMode="unified" focusLine={{ line: 45_000 }} />
      </DiffWrapContext.Provider>,
    )
    expect(screen.getByText('line 45000')).toBeInTheDocument()
  })

  it('copies the lines of a selection that are not drawn', () => {
    const { scroller, pane } = renderDiff()
    const start = screen.getByText('line 2')
    document.getSelection()!.setBaseAndExtent(start, 0, start, 0)
    document.getSelection()!.setBaseAndExtent(start, 0, screen.getByText('line 3'), 1)
    scrollTo(scroller, 600_000)
    document.getSelection()!.setBaseAndExtent(screen.getByText('line 2'), 0, screen.getByText('line 30000'), 1)
    const setData = vi.fn()
    fireEvent.copy(pane, { clipboardData: { setData } })
    const lines = (setData.mock.calls[0][1] as string).split('\n')
    expect(lines).toHaveLength(29_999)
    expect([lines[0], lines[29_998]]).toEqual(['line 2', 'line 30000'])
  })
})

describe('a short diff', () => {
  it('draws every row, and copies as the page does', () => {
    const { pane } = renderDiff({ diff: diffOf(400) })
    expect(pane.querySelectorAll('.diff-row')).toHaveLength(400)
    expect(pane.querySelector('.diff-gap')).toBeNull()
  })
})

describe('the minimap of a long diff', () => {
  it('draws no more marks than it has room for', () => {
    const lines = Array.from({ length: LINES }, (_, i): DiffLine => ({
      ...changed(i + 1),
      kind: i % 3 === 0 ? 'added' : 'context',
      oldLine: i + 1,
    }))
    const { container } = renderDiff({ diff: new FileDiff({ hunks: [{ header: '@@', raw: 'raw', lines }] }) })
    expect(container.querySelectorAll('.diff-minimap-mark').length).toBeLessThanOrEqual(1000)
  })
})
