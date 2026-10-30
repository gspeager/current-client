import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FileDiff, HostService, ReviewThread } from '@current-client-bindings/app'
import { ReviewDraftProvider } from '../../lib/reviewDraft'
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

describe('ReviewThreadBlock with actions', () => {
  it('replies and resolves through the actions, sending nothing until clicked', async () => {
    const actions = {
      canResolve: true,
      reply: vi.fn().mockResolvedValue(undefined),
      resolve: vi.fn().mockResolvedValue(undefined),
    }
    render(<ReviewThreadBlock thread={thread({})} actions={actions} />)
    const group = screen.getByRole('group', { name: 'Unresolved thread by Ada' })
    expect(actions.reply).not.toHaveBeenCalled()
    await userEvent.click(within(group).getByRole('button', { name: 'Reply' }))
    await userEvent.type(within(group).getByRole('textbox', { name: 'Reply' }), 'Agreed.')
    await userEvent.click(within(group).getByRole('button', { name: 'Reply' }))
    await waitFor(() => expect(actions.reply).toHaveBeenCalledWith('t1', 'Agreed.'))
    await waitFor(() => expect(within(group).queryByRole('textbox')).toBeNull())
    await userEvent.click(within(group).getByRole('button', { name: 'Resolve' }))
    expect(actions.resolve).toHaveBeenCalledWith('t1', true)
  })

  it('hides Resolve where the host cannot, and shows a refusal', async () => {
    const actions = {
      canResolve: false,
      reply: vi.fn().mockRejectedValue(new Error('github.example.com refused this')),
      resolve: vi.fn(),
    }
    render(<ReviewThreadBlock thread={thread({})} actions={actions} />)
    const group = screen.getByRole('group', { name: 'Unresolved thread by Ada' })
    expect(within(group).queryByRole('button', { name: 'Resolve' })).toBeNull()
    await userEvent.click(within(group).getByRole('button', { name: 'Reply' }))
    await userEvent.type(within(group).getByRole('textbox', { name: 'Reply' }), 'Hm.')
    await userEvent.click(within(group).getByRole('button', { name: 'Reply' }))
    expect(await within(group).findByRole('alert')).toHaveTextContent('refused this')
  })
})

describe('DiffViewer with a review', () => {
  it('comments on a line now or drafts it for the review, and never sends on its own', async () => {
    vi.mocked(HostService.ReviewCapabilities).mockResolvedValue({
      reply: true,
      resolve: true,
      lineComments: true,
      approve: true,
      requestChanges: false,
    } as never)
    vi.mocked(HostService.CommentOnLine).mockResolvedValue({} as never)
    render(
      <ReviewDraftProvider>
        <DiffWrapContext.Provider value={{ wrap: true, setWrap: () => undefined }}>
          <DiffViewer
            diff={diff}
            path="app/x.go"
            viewMode="unified"
            threads={[thread({})]}
            review={{ repoPath: '/repos/app', number: 7, prefix: '!' }}
          />
        </DiffWrapContext.Provider>
      </ReviewDraftProvider>,
    )
    const bar = await screen.findByRole('toolbar', { name: 'Review' })
    expect(bar).toHaveTextContent('!7')
    // The thread under line 2 takes a reply now.
    expect(
      within(screen.getByRole('group', { name: 'Unresolved thread by Ada' })).getByRole('button', { name: 'Reply' }),
    ).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Comment on line 3' }))
    await userEvent.type(screen.getByRole('textbox', { name: 'Comment on line 3' }), 'Drafted.')
    await userEvent.click(screen.getByRole('button', { name: 'Add to review' }))
    expect(screen.getByRole('group', { name: 'Drafted comment at line 3' })).toHaveTextContent('Drafted.')
    expect(bar).toHaveTextContent('1 drafted comment, not sent yet')
    expect(HostService.CommentOnLine).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: 'Comment on line 2' }))
    await userEvent.type(screen.getByRole('textbox', { name: 'Comment on line 2' }), 'Now.')
    await userEvent.click(screen.getByRole('button', { name: 'Comment' }))
    await waitFor(() =>
      expect(HostService.CommentOnLine).toHaveBeenCalledWith('/repos/app', 7, {
        path: 'app/x.go',
        line: 2,
        oldLine: 0,
        body: 'Now.',
      }),
    )
    await waitFor(() => expect(screen.queryByRole('textbox', { name: 'Comment on line 2' })).toBeNull())

    await userEvent.click(within(bar).getByRole('button', { name: 'Discard drafts' }))
    expect(screen.queryByRole('group', { name: 'Drafted comment at line 3' })).toBeNull()
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
