import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { InboxService } from '@current-client-bindings/app'
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
  ...overrides,
})

describe('InboxView', () => {
  it('lists each open repository’s rows and jumps to the thing each row is about', async () => {
    vi.mocked(InboxService.Rows).mockResolvedValue([
      {
        path: '/repos/app',
        name: 'app',
        error: '',
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
      { path: '/repos/lib', name: 'lib', error: '', rows: [] },
    ] as never)
    const onGo = vi.fn()
    render(<InboxView paths={['/repos/app', '/repos/lib']} refreshKey={0} onGo={onGo} />)

    const app = await screen.findByRole('region', { name: 'app' })
    expect(screen.getByText('4 items')).toBeInTheDocument()
    expect(InboxService.Rows).toHaveBeenCalledWith(['/repos/app', '/repos/lib'], true)
    expect(within(screen.getByRole('region', { name: 'lib' })).getByText('Nothing waiting.')).toBeInTheDocument()

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
