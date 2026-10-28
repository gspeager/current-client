import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HostService, InboxService, StandupRepo } from '@current-client-bindings/app'
import StandupDialog from './StandupDialog'

const repo = (over: Partial<StandupRepo>) =>
  new StandupRepo({
    path: '/r/app',
    name: 'app',
    commits: [{ sha: 'a', subject: 'feat: forms', date: '' }],
    hostEnabled: false,
    ...over,
  } as never)

describe('StandupDialog', () => {
  // A Wednesday, so "since yesterday" and "this week" start on different days.
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date(2026, 9, 28, 12))
  })
  afterEach(() => vi.useRealTimers())

  it('lists the commits, switches period and format, and copies the text', async () => {
    vi.mocked(InboxService.Standup).mockResolvedValue([
      repo({}),
      repo({ path: '/r/api', name: 'api', hostEnabled: true, commits: [] }),
    ])
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, { clipboard: { writeText } })
    render(<StandupDialog paths={['/r/app', '/r/api']} onClose={vi.fn()} />)

    const text = await screen.findByLabelText('Standup text')
    expect(text).toHaveTextContent('app Commits - feat: forms')
    expect(screen.getByText(/Not refreshed yet, so no host rows: api\./)).toBeInTheDocument()
    expect(vi.mocked(InboxService.Standup).mock.calls[0][0]).toEqual(['/r/app', '/r/api'])

    await userEvent.click(screen.getByRole('button', { name: 'This week' }))
    expect(InboxService.Standup).toHaveBeenCalledTimes(2)
    await userEvent.click(screen.getByRole('button', { name: 'Markdown' }))
    expect(screen.getByLabelText('Standup text')).toHaveTextContent('### app')
    await userEvent.click(screen.getByRole('button', { name: 'Copy' }))
    expect(writeText).toHaveBeenCalledWith('### app\n\n**Commits**\n\n- feat: forms')
    expect(HostService.Refresh).not.toHaveBeenCalled()
  })

  it('shows the Markdown formatted, and copies it as rich text with the Markdown beside', async () => {
    vi.mocked(InboxService.Standup).mockResolvedValue([repo({})])
    const write = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, { clipboard: { write } })
    vi.stubGlobal(
      'ClipboardItem',
      class {
        constructor(public items: Record<string, Blob>) {}
      },
    )
    render(<StandupDialog paths={['/r/app']} onClose={vi.fn()} />)
    await screen.findByLabelText('Standup text')

    await userEvent.click(screen.getByRole('button', { name: 'Rendered' }))
    expect(screen.getByRole('heading', { name: 'app' })).toBeInTheDocument()
    expect(screen.getByRole('listitem')).toHaveTextContent('feat: forms')
    await userEvent.click(screen.getByRole('button', { name: 'Copy' }))
    const { items } = write.mock.calls[0][0][0] as { items: Record<string, Blob> }
    expect(await items['text/plain'].text()).toBe('### app\n\n**Commits**\n\n- feat: forms')
    expect(await items['text/html'].text()).toContain('<h3>app</h3>')
    vi.unstubAllGlobals()
  })

  it('says when there is nothing', async () => {
    vi.mocked(InboxService.Standup).mockResolvedValue([repo({ commits: [] })])
    render(<StandupDialog paths={['/r/app']} onClose={vi.fn()} />)
    expect(await screen.findByLabelText('Standup text')).toHaveTextContent('Nothing since then.')
    expect(screen.getByRole('button', { name: 'Copy' })).toBeDisabled()
  })
})
