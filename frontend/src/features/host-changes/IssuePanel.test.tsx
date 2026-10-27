import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HostService, IssueDetails, PlatformService } from '@current-client-bindings/app'
import { IssuePanelContext } from '../../lib/issuePanel'
import IssuePanel from './IssuePanel'

describe('IssuePanel', () => {
  it('shows the issue, its facts, description and comments, and opens it on the host', async () => {
    vi.mocked(HostService.IssueDetails).mockResolvedValue(
      new IssueDetails({
        reference: '#4',
        descriptionHtml: '<p>Steps to <strong>reproduce</strong></p>',
        issue: {
          key: '4',
          number: 4,
          title: 'Login fails',
          state: 'open',
          webUrl: 'https://gitlab.example.com/g/r/-/issues/4',
          description: '',
          labels: ['bug', 'auth'],
          milestone: '0.5.0',
          author: { name: 'Ada', username: 'ada' },
          createdAt: '2026-10-20T10:00:00Z',
          updatedAt: '2026-10-20T10:00:00Z',
        },
        comments: [
          {
            author: { name: 'Grace', username: 'grace' },
            bodyHtml: '<p>Same on Linux</p>',
            createdAt: '2026-10-21T10:00:00Z',
          },
        ],
      } as never),
    )
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue()
    render(<IssuePanel repoPath="/repos/app" issueKey="4" onClose={vi.fn()} />)

    const dialog = await screen.findByRole('dialog', { name: '#4' })
    expect(within(dialog).getByRole('heading', { name: 'Login fails' })).toBeInTheDocument()
    expect(dialog).toHaveTextContent('open')
    expect(dialog).toHaveTextContent('Ada')
    expect(dialog).toHaveTextContent('0.5.0')
    expect(within(dialog).getByText('bug')).toBeInTheDocument()
    expect(within(dialog).getByText('reproduce')).toBeInTheDocument()
    const comments = within(dialog).getByRole('region', { name: 'Comments' })
    expect(comments).toHaveTextContent('1 comment')
    expect(comments).toHaveTextContent('Grace')
    expect(comments).toHaveTextContent('Same on Linux')
    expect(HostService.IssueDetails).toHaveBeenCalledWith('/repos/app', '4')

    await userEvent.click(within(dialog).getByRole('button', { name: 'Open on host' }))
    expect(PlatformService.OpenWebPage).toHaveBeenCalledWith('https://gitlab.example.com/g/r/-/issues/4')
    expect(within(dialog).queryByRole('button', { name: 'Close issue' })).not.toBeInTheDocument()
  })

  it('says when the issue cannot be loaded', async () => {
    vi.mocked(HostService.IssueDetails).mockRejectedValue(new Error('integration is off'))
    render(<IssuePanel repoPath="/repos/app" issueKey="4" onClose={vi.fn()} />)
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not load the issue: integration is off')
  })

  it('comments and closes where the tracker takes writes, then reads the panel again', async () => {
    const details = (state: string, comments: number) =>
      new IssueDetails({
        reference: '#4',
        descriptionHtml: '',
        issue: {
          key: '4',
          number: 4,
          title: 'Login fails',
          state,
          webUrl: '',
          description: '',
          labels: [],
          milestone: '',
          author: { name: 'Ada', username: 'ada' },
          createdAt: '2026-10-20T10:00:00Z',
          updatedAt: '2026-10-20T10:00:00Z',
        },
        comments: Array.from({ length: comments }, () => ({
          author: { name: 'Me', username: 'me' },
          bodyHtml: '<p>Fixed on main</p>',
          createdAt: '2026-10-21T10:00:00Z',
        })),
      } as never)
    vi.mocked(HostService.IssueDetails)
      .mockResolvedValueOnce(details('open', 0))
      .mockResolvedValueOnce(details('open', 1))
      .mockResolvedValueOnce(details('closed', 1))
    vi.mocked(HostService.CommentOnIssue).mockResolvedValue(details('open', 1))
    vi.mocked(HostService.SetIssueState).mockResolvedValue(details('closed', 1))
    const newIssue = vi.fn()
    render(
      <IssuePanelContext.Provider value={{ open: vi.fn(), create: newIssue }}>
        <IssuePanel repoPath="/repos/app" issueKey="4" onClose={vi.fn()} />
      </IssuePanelContext.Provider>,
    )
    const dialog = await screen.findByRole('dialog', { name: '#4' })
    await userEvent.type(within(dialog).getByRole('textbox', { name: 'Comment' }), 'Fixed on main')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Comment' }))
    expect(HostService.CommentOnIssue).toHaveBeenCalledWith('/repos/app', '4', 'Fixed on main')
    expect(await within(dialog).findByText('1 comment')).toBeInTheDocument()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Close issue' }))
    expect(HostService.SetIssueState).toHaveBeenCalledWith('/repos/app', '4', false)
    expect(await within(dialog).findByRole('button', { name: 'Reopen issue' })).toBeInTheDocument()
    await userEvent.click(within(dialog).getByRole('button', { name: 'New issue' }))
    expect(newIssue).toHaveBeenCalled()
  })
})
