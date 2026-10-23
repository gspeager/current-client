import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FileDiff, ReviewThread } from '@current-client-bindings/app'
import { DiffWrapContext } from './diffWrap'
import DiffViewer from './DiffViewer'
import ReviewThreadBlock from './ReviewThreadBlock'

const thread = (overrides: Partial<ReviewThread>) =>
  new ReviewThread({
    id: 't1',
    path: 'app/x.go',
    line: 2,
    notes: [
      {
        author: 'Ada',
        bodyHtml: '<p>Rename this to <strong>client</strong>.</p>',
        createdAt: new Date().toISOString(),
      },
    ],
    ...overrides,
  })

const diff = new FileDiff({
  hunks: [
    {
      header: '@@ -1,2 +1,3 @@',
      raw: '',
      lines: [
        { kind: 'context', oldLine: 1, newLine: 1, content: 'package app' },
        { kind: 'added', oldLine: 0, newLine: 2, content: 'var client = 1' },
        { kind: 'context', oldLine: 2, newLine: 3, content: 'func main() {}' },
      ],
    },
  ],
} as never)

describe('ReviewThreadBlock', () => {
  it('shows an unresolved thread open and a resolved one collapsed until clicked', async () => {
    render(
      <>
        <ReviewThreadBlock thread={thread({})} />
        <ReviewThreadBlock thread={thread({ id: 't2', resolved: true })} />
      </>,
    )
    const open = screen.getByRole('group', { name: 'Unresolved thread by Ada' })
    expect(within(open).getByText('client')).toBeInTheDocument()
    const resolved = screen.getByRole('group', { name: 'Resolved thread by Ada' })
    expect(within(resolved).queryByText('client')).not.toBeInTheDocument()
    await userEvent.click(within(resolved).getByRole('button', { expanded: false }))
    expect(within(resolved).getByText('client')).toBeInTheDocument()
  })
})

describe('DiffViewer with review threads', () => {
  it('pins a thread under its line and lists outdated ones below', () => {
    render(
      <DiffWrapContext.Provider value={{ wrap: true, setWrap: () => undefined }}>
        <DiffViewer
          diff={diff}
          path="app/x.go"
          viewMode="unified"
          threads={[thread({}), thread({ id: 't3', line: 99 })]}
          outdatedThreads={[thread({ id: 't4', line: 0, oldLine: 15, outdated: true })]}
        />
      </DiffWrapContext.Provider>,
    )
    const rows = [...document.querySelectorAll('.diff-pane > *')]
    const lineIndex = rows.findIndex((r) => r.textContent?.includes('var client = 1'))
    expect(rows[lineIndex + 1]).toHaveClass('review-thread')
    const away = screen.getByRole('region', { name: 'Threads not shown at a line' })
    expect(within(away).getByRole('group', { name: 'Unresolved thread by Ada at line 99' })).toBeInTheDocument()
    expect(within(away).getByRole('group', { name: 'Unresolved thread by Ada at old line 15' })).toBeInTheDocument()
  })
})
