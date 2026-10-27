import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HostService, TrackerLinks } from '@current-client-bindings/app'
import { IssuePanelContext } from '../../lib/issuePanel'
import IssuePicker from './IssuePicker'

describe('IssuePicker', () => {
  it('lists the tracker’s issues with the host off, and names the branch after the key', async () => {
    vi.mocked(HostService.CurrentChange).mockResolvedValue({ enabled: false } as never)
    vi.mocked(HostService.AssignedIssues).mockResolvedValue([
      { key: 'PROJ-20', number: 0, title: 'Fix the login form', state: 'open', webUrl: '' },
    ])
    const onPick = vi.fn()
    render(
      <IssuePanelContext.Provider value={{ tracker: new TrackerLinks({ reading: true }) }}>
        <IssuePicker repoPath="/repos/app" onPick={onPick} />
      </IssuePanelContext.Provider>,
    )
    await userEvent.click(await screen.findByRole('button', { name: /From an issue/ }))
    await userEvent.click(await screen.findByRole('button', { name: /PROJ-20/ }))
    expect(onPick).toHaveBeenCalledWith('PROJ-20-fix-the-login-form')
  })

  it('is absent with neither the host nor a tracker on', async () => {
    vi.mocked(HostService.CurrentChange).mockResolvedValue({ enabled: false } as never)
    render(<IssuePicker repoPath="/repos/app" onPick={vi.fn()} />)
    await vi.waitFor(() => expect(HostService.CurrentChange).toHaveBeenCalled())
    expect(screen.queryByRole('button', { name: /From an issue/ })).not.toBeInTheDocument()
  })
})
