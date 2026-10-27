import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HostService } from '@current-client-bindings/app'
import NewIssueDialog from './NewIssueDialog'

describe('NewIssueDialog', () => {
  it('sends nothing until Create issue, then the title, description and labels', async () => {
    const issue = { key: '9', number: 9, title: 'drop the shim', state: 'open', webUrl: '' }
    vi.mocked(HostService.CreateIssue).mockResolvedValue(issue)
    const onCreated = vi.fn()
    render(
      <NewIssueDialog
        repoPath="/repos/app"
        draft={{ title: 'drop the shim', description: 'From a.ts' }}
        onCreated={onCreated}
        onClose={vi.fn()}
      />,
    )
    await userEvent.type(screen.getByRole('textbox', { name: 'Labels, separated by commas' }), 'bug, , tech debt')
    expect(HostService.CreateIssue).not.toHaveBeenCalled()
    await userEvent.click(screen.getByRole('button', { name: 'Create issue' }))
    expect(HostService.CreateIssue).toHaveBeenCalledWith('/repos/app', {
      title: 'drop the shim',
      description: 'From a.ts',
      labels: ['bug', 'tech debt'],
    })
    expect(onCreated).toHaveBeenCalledWith(issue)
  })

  it('needs a title and shows the host refusal', async () => {
    vi.mocked(HostService.CreateIssue).mockRejectedValue(
      new Error('The token for gitlab.example.com lacks the api scope this needs.'),
    )
    render(<NewIssueDialog repoPath="/repos/app" onCreated={vi.fn()} onClose={vi.fn()} />)
    const create = screen.getByRole('button', { name: 'Create issue' })
    expect(create).toBeDisabled()
    await userEvent.type(screen.getByRole('textbox', { name: 'Title' }), 'Flaky test')
    await userEvent.click(create)
    expect(await screen.findByRole('alert')).toHaveTextContent('lacks the api scope')
  })
})
