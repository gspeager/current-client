import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HostService, InboxService } from '@current-client-bindings/app'
import InboxView from './InboxView'

const row = (overrides: Record<string, unknown>) => ({
  kind: '',
  branch: '',
  number: 0,
  prefix: '',
  key: '',
  title: '',
  detail: '',
  count: 0,
  webUrl: '',
  fetchedAt: null,
  id: '',
  new: false,
  todoId: '',
  ...overrides,
})

describe('InboxView', () => {
  it('lists each open repository’s rows and jumps to the thing each row is about', async () => {
    vi.mocked(InboxService.Rows).mockResolvedValue([
      {
        path: '/repos/app',
        name: 'app',
        error: '',
        hostEnabled: true,
        fetchedAt: new Date().toISOString(),
        rows: [
          row({ kind: 'dirty', count: 3 }),
          row({ kind: 'unpushed', branch: 'feature', count: 2 }),
          row({
            kind: 'review',
            number: 7,
            prefix: '!',
            title: 'Add the thing',
            fetchedAt: new Date().toISOString(),
            id: 'review:7',
            new: true,
          }),
          row({
            kind: 'assigned',
            key: '12',
            title: 'Login fails',
            detail: '!4 failing CI',
            fetchedAt: new Date().toISOString(),
          }),
        ],
      },
      { path: '/repos/lib', name: 'lib', error: '', rows: [], hostEnabled: false, fetchedAt: null },
    ] as never)
    const onGo = vi.fn()
    render(<InboxView paths={['/repos/app', '/repos/lib']} activePath="/repos/app" refreshKey={0} onGo={onGo} />)

    const app = await screen.findByRole('region', { name: 'app' })
    expect(screen.getByText('4 items')).toBeInTheDocument()
    expect(InboxService.Rows).toHaveBeenCalledWith(['/repos/app', '/repos/lib'], true)
    // Only the open repository starts expanded; the rest fold to a header.
    const lib = screen.getByRole('region', { name: 'lib' })
    expect(within(lib).getByRole('button', { name: /lib/ })).toHaveAttribute('aria-expanded', 'false')
    expect(within(lib).queryByText('Nothing waiting.')).toBeNull()
    await userEvent.click(within(lib).getByRole('button', { name: /lib/ }))
    expect(within(lib).getByText('Nothing waiting.')).toBeInTheDocument()
    expect(within(app).getByRole('button', { name: /app.*4.*1 new/ })).toHaveAttribute('aria-expanded', 'true')

    await userEvent.click(within(app).getByRole('button', { name: /3 changed files/ }))
    expect(onGo).toHaveBeenLastCalledWith({ path: '/repos/app', kind: 'changes' })
    await userEvent.click(within(app).getByRole('button', { name: /feature.*2 unpushed commits/ }))
    expect(onGo).toHaveBeenLastCalledWith({ path: '/repos/app', kind: 'branch', branch: 'feature' })
    const review = within(app).getByRole('button', { name: /!7 Add the thing/ })
    expect(review).toHaveTextContent('from the host')
    expect(within(review).getByText('new')).toBeInTheDocument()
    expect(within(within(app).getByRole('button', { name: /3 changed files/ })).queryByText('new')).toBeNull()
    await userEvent.click(review)
    expect(onGo).toHaveBeenLastCalledWith({ path: '/repos/app', kind: 'change', number: 7 })
    await userEvent.click(within(app).getByRole('button', { name: /#12 Login fails/ }))
    expect(onGo).toHaveBeenLastCalledWith({ path: '/repos/app', kind: 'issue', key: '12' })
    expect(within(app).getByText('assigned, !4 failing CI')).toBeInTheDocument()
  })

  it('lists the host’s to-dos, marks one done on the host, and says when the host won’t list them', async () => {
    vi.mocked(InboxService.Rows).mockResolvedValue([
      {
        path: '/repos/app',
        name: 'app',
        error: '',
        hostEnabled: true,
        fetchedAt: new Date().toISOString(),
        notice: '',
        rows: [
          row({ kind: 'review', number: 1, prefix: '!', title: 'Bob’s change', id: 'review:1', todoId: '102' }),
          row({
            kind: 'todo',
            number: 7,
            prefix: '!',
            title: 'Add the thing',
            detail: 'mentioned',
            id: 'todo:101',
            todoId: '101',
          }),
          row({ kind: 'todo', key: '12', title: 'Login fails', detail: 'new comment', id: 'todo:103', todoId: '103' }),
          row({
            kind: 'todo',
            title: 'Nightly',
            detail: 'CI activity',
            webUrl: 'https://github.com/o/r/commit/abc',
            id: 'todo:104',
            todoId: '104',
          }),
        ],
      },
      {
        path: '/repos/lib',
        name: 'lib',
        error: '',
        rows: [],
        hostEnabled: true,
        fetchedAt: new Date().toISOString(),
        notice: 'github.com’s notifications need a classic token with the notifications scope',
      },
    ] as never)
    vi.mocked(InboxService.MarkDone).mockResolvedValue(undefined)
    const onGo = vi.fn()
    const onRefreshed = vi.fn()
    render(
      <InboxView
        paths={['/repos/app', '/repos/lib']}
        activePath="/repos/app"
        refreshKey={0}
        onGo={onGo}
        onRefreshed={onRefreshed}
      />,
    )

    const app = await screen.findByRole('region', { name: 'app' })
    expect(within(app).getByText('mentioned')).toBeInTheDocument()
    await userEvent.click(within(app).getByRole('button', { name: /^!7 Add the thing/ }))
    expect(onGo).toHaveBeenLastCalledWith({ path: '/repos/app', kind: 'change', number: 7 })
    await userEvent.click(within(app).getByRole('button', { name: /^#12 Login fails/ }))
    expect(onGo).toHaveBeenLastCalledWith({ path: '/repos/app', kind: 'issue', key: '12' })
    await userEvent.click(within(app).getByRole('button', { name: /^Nightly.*CI activity/ }))
    expect(onGo).toHaveBeenLastCalledWith({
      path: '/repos/app',
      kind: 'host',
      url: 'https://github.com/o/r/commit/abc',
    })
    // A review request that is also a to-do is one row, with Done.
    await userEvent.click(within(app).getByRole('button', { name: 'Mark done: !1 Bob’s change' }))
    expect(InboxService.MarkDone).toHaveBeenCalledWith('/repos/app', '102')
    expect(onRefreshed).toHaveBeenCalled()
    expect(within(app).getAllByRole('button', { name: /^Mark done/ })).toHaveLength(4)

    const lib = screen.getByRole('region', { name: 'lib' })
    await userEvent.click(within(lib).getByRole('button', { name: /lib/ }))
    expect(within(lib).getByText(/classic token/)).toBeInTheDocument()
  })

  it('says when a host is on but not refreshed yet, and refreshes it on request', async () => {
    vi.mocked(InboxService.Rows).mockResolvedValue([
      { path: '/repos/app', name: 'app', error: '', rows: [], hostEnabled: true, fetchedAt: null },
    ] as never)
    vi.mocked(HostService.Refresh).mockResolvedValue()
    const onRefreshed = vi.fn()
    render(
      <InboxView
        paths={['/repos/app']}
        activePath="/repos/app"
        refreshKey={0}
        onGo={vi.fn()}
        onRefreshed={onRefreshed}
      />,
    )

    expect(await screen.findByText(/hasn’t been refreshed yet/)).toBeInTheDocument()
    expect(screen.queryByText('Nothing waiting.')).toBeNull()
    await userEvent.click(screen.getByRole('button', { name: 'Refresh' }))
    expect(HostService.Refresh).toHaveBeenCalledWith('/repos/app')
    await vi.waitFor(() => expect(onRefreshed).toHaveBeenCalled())
  })

  it('closes from its own header', async () => {
    vi.mocked(InboxService.Rows).mockResolvedValue([])
    const onClose = vi.fn()
    render(<InboxView paths={['/repos/app']} refreshKey={0} onGo={vi.fn()} onClose={onClose} />)
    await userEvent.click(screen.getByRole('button', { name: 'Close Inbox' }))
    expect(onClose).toHaveBeenCalled()
  })

  it('shows a repository that could not be read, and says when nothing is open', async () => {
    vi.mocked(InboxService.Rows).mockResolvedValue([
      { path: '/repos/gone', name: 'gone', error: 'not a git repository', rows: [] },
    ] as never)
    render(<InboxView paths={['/repos/gone']} refreshKey={0} onGo={vi.fn()} />)
    expect(await screen.findByText('not a git repository')).toBeInTheDocument()
    expect(screen.getByText('Nothing waiting')).toBeInTheDocument()

    render(<InboxView paths={[]} refreshKey={0} onGo={vi.fn()} />)
    expect(screen.getByText('Open a repository to see what waits in it.')).toBeInTheDocument()
  })
})
