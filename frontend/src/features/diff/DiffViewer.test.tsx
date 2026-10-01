import { render, screen, fireEvent, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { FileDiff } from '@current-client-bindings/app'
import DiffViewer from './DiffViewer'

const diff: FileDiff = {
  conflicted: false,
  oldPath: 'a.ts',
  newPath: 'a.ts',
  binary: false,
  tooLarge: false,
  sizeBytes: 0,
  hunks: [
    {
      header: '@@ -1,3 +1,3 @@ first',
      raw: 'raw-hunk-1',
      lines: [{ kind: 'context', oldLine: 1, newLine: 1, content: 'foo', moved: false }],
    },
    {
      header: '@@ -10,3 +10,3 @@ second',
      raw: 'raw-hunk-2',
      lines: [{ kind: 'context', oldLine: 10, newLine: 10, content: 'bar', moved: false }],
    },
    {
      header: '@@ -20,3 +20,3 @@ third',
      raw: 'raw-hunk-3',
      lines: [{ kind: 'context', oldLine: 20, newLine: 20, content: 'baz', moved: false }],
    },
  ],
}

beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn()
})

function currentHeader() {
  return screen.getAllByText(/^@@/, { selector: 'span' }).find((el) => el.closest('.diff-hunk-header-current'))
    ?.textContent
}

describe('DiffViewer hunk navigation', () => {
  it('starts on the first hunk and moves forward/backward with Alt+ArrowDown/Up', () => {
    render(<DiffViewer diff={diff} path="a.ts" viewMode="unified" />)

    expect(currentHeader()).toContain('first')

    fireEvent.keyDown(window, { key: 'ArrowDown', altKey: true })
    expect(currentHeader()).toContain('second')

    fireEvent.keyDown(window, { key: 'ArrowDown', altKey: true })
    expect(currentHeader()).toContain('third')

    // Clamped at the last hunk.
    fireEvent.keyDown(window, { key: 'ArrowDown', altKey: true })
    expect(currentHeader()).toContain('third')

    fireEvent.keyDown(window, { key: 'ArrowUp', altKey: true })
    expect(currentHeader()).toContain('second')
  })

  it('ignores Up/Down without Alt held', () => {
    render(<DiffViewer diff={diff} path="a.ts" viewMode="unified" />)

    fireEvent.keyDown(window, { key: 'ArrowDown' })
    expect(currentHeader()).toContain('first')
  })
})

describe('DiffViewer hunk actions', () => {
  it('offers Discard next to Unstage hunk on a staged hunk', async () => {
    const onUnstageHunk = vi.fn()
    const onDiscardHunk = vi.fn()
    render(
      <DiffViewer
        diff={diff}
        path="a.ts"
        viewMode="unified"
        onUnstageHunk={onUnstageHunk}
        onDiscardHunk={onDiscardHunk}
      />,
    )

    const [unstage] = screen.getAllByRole('button', { name: 'Unstage hunk' })
    await userEvent.click(unstage)
    await userEvent.click(within(unstage.parentElement!).getByRole('button', { name: 'Discard' }))

    expect(onUnstageHunk).toHaveBeenCalledWith(diff.hunks[0].raw)
    expect(onDiscardHunk).toHaveBeenCalledWith(diff.hunks[0].raw)
  })
})

describe('DiffViewer content rendering', () => {
  const changed: FileDiff = {
    conflicted: false,
    oldPath: 'a.txt',
    newPath: 'a.txt',
    binary: false,
    tooLarge: false,
    sizeBytes: 0,
    hunks: [
      {
        header: '@@ -1,2 +1,2 @@',
        raw: 'raw-hunk',
        lines: [
          { kind: 'removed', oldLine: 1, newLine: 0, content: 'old line', moved: false },
          { kind: 'added', oldLine: 0, newLine: 1, content: 'new line', moved: false },
          { kind: 'context', oldLine: 2, newLine: 2, content: 'unchanged line', moved: false },
        ],
      },
    ],
  }

  it('renders removed and added lines with their content and markers', () => {
    const { container } = render(<DiffViewer diff={changed} path="a.txt" viewMode="unified" />)

    const removedRow = container.querySelector('.diff-row-removed')
    const addedRow = container.querySelector('.diff-row-added')
    expect(removedRow?.textContent).toContain('old line')
    expect(removedRow?.querySelector('.diff-marker')?.textContent).toBe('-')
    expect(addedRow?.textContent).toContain('new line')
    expect(addedRow?.querySelector('.diff-marker')?.textContent).toBe('+')
  })

  it('renders unchanged context lines without a marker', () => {
    const { container } = render(<DiffViewer diff={changed} path="a.txt" viewMode="unified" />)

    const contextRow = Array.from(container.querySelectorAll('.diff-row')).find((row) =>
      row.textContent?.includes('unchanged line'),
    )
    expect(contextRow?.querySelector('.diff-marker')?.textContent).toBe('')
  })
})

describe('DiffViewer empty and error-ish states', () => {
  it('shows a too-large message with a manual load action', async () => {
    const user = userEvent.setup()
    const onForceLoad = vi.fn()
    render(
      <DiffViewer
        diff={{
          conflicted: false,
          oldPath: 'big.bin',
          newPath: 'big.bin',
          binary: false,
          tooLarge: true,
          sizeBytes: 5 * 1024 * 1024,
          hunks: [],
        }}
        path="big.bin"
        viewMode="unified"
        onForceLoad={onForceLoad}
      />,
    )
    expect(screen.getByText(/large/i)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Load anyway' }))
    expect(onForceLoad).toHaveBeenCalledTimes(1)
  })

  it('shows a binary-file message', () => {
    render(
      <DiffViewer
        diff={{
          conflicted: false,
          oldPath: 'img.png',
          newPath: 'img.png',
          binary: true,
          tooLarge: false,
          sizeBytes: 0,
          hunks: [],
        }}
        path="img.png"
        viewMode="unified"
      />,
    )
    expect(screen.getByText('Binary file changed.')).toBeInTheDocument()
  })

  it('shows a no-differences message when there are no hunks', () => {
    render(
      <DiffViewer
        diff={{
          conflicted: false,
          oldPath: 'a.txt',
          newPath: 'a.txt',
          binary: false,
          tooLarge: false,
          sizeBytes: 0,
          hunks: [],
        }}
        path="a.txt"
        viewMode="unified"
      />,
    )
    expect(screen.getByText('No differences.')).toBeInTheDocument()
  })
})

describe('DiffViewer conflicted file', () => {
  const conflict: FileDiff = {
    conflicted: true,
    oldPath: 'f.txt',
    newPath: 'f.txt',
    binary: false,
    tooLarge: false,
    sizeBytes: 0,
    hunks: [
      {
        header: '@@@ -1,3 -1,3 +1,7 @@@',
        raw: 'raw',
        lines: [
          { kind: 'conflict-marker', oldLine: 0, newLine: 1, content: '<<<<<<< HEAD', moved: false },
          { kind: 'conflict-ours', oldLine: 1, newLine: 2, content: 'OURS', moved: false },
          { kind: 'conflict-marker', oldLine: 0, newLine: 3, content: '=======', moved: false },
          { kind: 'conflict-theirs', oldLine: 0, newLine: 4, content: 'THEIRS', moved: false },
          { kind: 'conflict-marker', oldLine: 0, newLine: 5, content: '>>>>>>> theirs', moved: false },
        ],
      },
    ],
  }

  it('shows both sides and the markers inline, even when split view is chosen', () => {
    const { container } = render(<DiffViewer diff={conflict} path="f.txt" viewMode="split" />)

    expect(
      screen.getByText('Conflicted. Edit the marked sections to resolve, then stage the file.'),
    ).toBeInTheDocument()
    expect(container.querySelector('.diff-pane-unified')).not.toBeNull()
    expect(screen.getByText('OURS').closest('.diff-row')).toHaveClass('diff-row-conflict-ours')
    expect(screen.getByText('THEIRS').closest('.diff-row')).toHaveClass('diff-row-conflict-theirs')
    expect(screen.getByText('=======').closest('.diff-row')).toHaveClass('diff-row-conflict-marker')
  })

  it('offers no hunk actions', () => {
    render(<DiffViewer diff={conflict} path="f.txt" viewMode="unified" onStageHunk={vi.fn()} onDiscardHunk={vi.fn()} />)

    expect(screen.queryByRole('button', { name: 'Stage hunk' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Discard' })).not.toBeInTheDocument()
  })
})
