import { act, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { TaskService, type TaskInfo, type TaskListInfo } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import TasksView from './TasksView'
import { useTasks } from './useTasks'

const { handlers } = vi.hoisted(() => ({ handlers: {} as Record<string, (event: { data: unknown }) => void> }))

vi.mock('@wailsio/runtime', async (importOriginal) => ({
  ...(await importOriginal<object>()),
  Events: {
    On: (name: string, handler: (event: { data: unknown }) => void) => {
      handlers[name] = handler
      return () => undefined
    },
  },
}))

const task = (overrides: Partial<TaskInfo> = {}): TaskInfo => ({
  id: 'make:test',
  source: 'Makefile',
  name: 'test',
  command: 'make test',
  file: 'Makefile',
  note: '',
  approved: true,
  fingerprint: 'seen',
  custom: false,
  ...overrides,
})

function list(overrides: Partial<TaskListInfo> = {}): TaskListInfo {
  return { tasks: [task()], beforeCommit: [], beforePush: [], dir: '/repo', ...overrides }
}

function Harness() {
  const tasks = useTasks('/repo', 0)
  return (
    <>
      <button type="button" onClick={() => void tasks.runChecklist('commit')}>
        committed
      </button>
      <TasksView repoPath="/repo" tasks={tasks} />
    </>
  )
}

function renderTasks(info = list()) {
  vi.mocked(TaskService.List).mockResolvedValue(info)
  vi.mocked(TaskService.Run).mockResolvedValue()
  vi.mocked(TaskService.Approve).mockResolvedValue()
  render(
    <DialogProvider>
      <Harness />
    </DialogProvider>,
  )
}

const runId = () => vi.mocked(TaskService.Run).mock.lastCall![2]

const finish = (exitCode: number, stopped = false) =>
  act(async () => handlers['task:done']({ data: { runId: runId(), exitCode, durationMs: 1500, stopped } }))

describe('tasks', () => {
  it('shows the command and folder before a first run, and runs only after a yes', async () => {
    const user = userEvent.setup()
    renderTasks(list({ tasks: [task({ approved: false })] }))

    await user.click(await screen.findByRole('button', { name: 'Run test' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Run this command?' })
    expect(within(dialog).getByText(/make test/)).toBeInTheDocument()
    expect(within(dialog).getByText(/In \/repo/)).toBeInTheDocument()
    expect(TaskService.Run).not.toHaveBeenCalled()

    await user.click(within(dialog).getByRole('button', { name: 'Run' }))
    await vi.waitFor(() => expect(TaskService.Run).toHaveBeenCalledWith('/repo', 'make:test', expect.any(String)))
    expect(TaskService.Approve).toHaveBeenCalledWith('/repo', 'make:test', 'seen')
  })

  it('runs nothing when the question is declined', async () => {
    const user = userEvent.setup()
    renderTasks(list({ tasks: [task({ approved: false })] }))
    await user.click(await screen.findByRole('button', { name: 'Run test' }))
    const dialog = await screen.findByRole('alertdialog')
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }))
    await act(async () => {})
    expect(TaskService.Run).not.toHaveBeenCalled()
    expect(TaskService.Approve).not.toHaveBeenCalled()
  })

  it('keeps output, without colors, and shows the status and duration', async () => {
    const user = userEvent.setup()
    renderTasks()
    await user.click(await screen.findByRole('button', { name: 'Run test' }))
    expect(await screen.findByText('Running')).toBeInTheDocument()

    await act(async () => handlers['task:output']({ data: { runId: runId(), data: btoa('\x1b[31mok\x1b[0m\r\n') } }))
    expect(screen.getByLabelText('Output of test').textContent).toBe('ok\n')

    await finish(0)
    expect(screen.getByText('Passed')).toBeInTheDocument()
    expect(screen.getByText(/Exit 0 · 1\.5 s/)).toBeInTheDocument()
  })

  it('hears from a run that finishes before the call that started it returns', async () => {
    const user = userEvent.setup()
    renderTasks()
    vi.mocked(TaskService.Run).mockImplementation(((_repo: string, _id: string, run: string) => {
      handlers['task:output']({ data: { runId: run, data: btoa('quick') } })
      handlers['task:done']({ data: { runId: run, exitCode: 0, durationMs: 1, stopped: false } })
      return Promise.resolve()
    }) as unknown as typeof TaskService.Run)
    await user.click(await screen.findByRole('button', { name: 'Run test' }))
    expect(await screen.findByText('Passed')).toBeInTheDocument()
    expect(screen.getByLabelText('Output of test').textContent).toBe('quick')
  })

  it("doesn't start a command that's already running", async () => {
    const user = userEvent.setup()
    renderTasks(list({ beforeCommit: ['make:test'] }))
    await user.click(await screen.findByRole('button', { name: 'Run test' }))
    await screen.findByText('Running')
    await user.click(screen.getByRole('button', { name: 'committed' }))
    await act(async () => {})
    expect(TaskService.Run).toHaveBeenCalledTimes(1)
  })

  it('joins an escape split across two reads', async () => {
    const user = userEvent.setup()
    renderTasks()
    await user.click(await screen.findByRole('button', { name: 'Run test' }))
    await screen.findByText('Running')
    await act(async () => handlers['task:output']({ data: { runId: runId(), data: btoa('red\x1b[3') } }))
    await act(async () => handlers['task:output']({ data: { runId: runId(), data: btoa('1mok\r') } }))
    await act(async () => handlers['task:output']({ data: { runId: runId(), data: btoa('\nnext') } }))
    expect(screen.getByLabelText('Output of test').textContent).toBe('redok\nnext')
  })

  it('stops a running command', async () => {
    const user = userEvent.setup()
    vi.mocked(TaskService.Stop).mockResolvedValue()
    renderTasks()
    await user.click(await screen.findByRole('button', { name: 'Run test' }))
    await user.click(await screen.findByRole('button', { name: 'Stop test' }))
    expect(TaskService.Stop).toHaveBeenCalledWith(runId())
    await finish(-1, true)
    expect(screen.getAllByText('Stopped').length).toBeGreaterThan(0)
  })

  it('warns when a checklist item fails, without stopping anything', async () => {
    const user = userEvent.setup()
    renderTasks(list({ beforeCommit: ['make:test'] }))
    await screen.findByRole('button', { name: 'Run test' })

    await user.click(screen.getByRole('button', { name: 'committed' }))
    await vi.waitFor(() => expect(TaskService.Run).toHaveBeenCalled())
    await finish(2)

    expect(await screen.findByText(/test failed after the last commit or push/)).toBeInTheDocument()
    expect(screen.getByText('Failed')).toBeInTheDocument()
  })

  it("doesn't run an unapproved checklist item, and says so", async () => {
    const user = userEvent.setup()
    renderTasks(list({ tasks: [task({ approved: false })], beforeCommit: ['make:test'] }))
    await screen.findByRole('button', { name: 'Run test' })

    await user.click(screen.getByRole('button', { name: 'committed' }))

    expect(await screen.findByText(/Not run, since not approved: test/)).toBeInTheDocument()
    expect(TaskService.Run).not.toHaveBeenCalled()
  })

  it('turns a checklist item on', async () => {
    const user = userEvent.setup()
    vi.mocked(TaskService.SetChecklist).mockResolvedValue()
    renderTasks()
    await user.click(await screen.findByRole('checkbox', { name: 'Run test with each push' }))
    expect(TaskService.SetChecklist).toHaveBeenCalledWith('/repo', 'push', ['make:test'])
  })

  it('adds and removes a custom command', async () => {
    const user = userEvent.setup()
    vi.mocked(TaskService.AddCustom).mockResolvedValue()
    vi.mocked(TaskService.RemoveCustom).mockResolvedValue()
    renderTasks(list({ tasks: [task({ id: 'custom:x', name: 'x', custom: true, file: '' })] }))

    await user.click(await screen.findByRole('button', { name: 'Remove x' }))
    expect(TaskService.RemoveCustom).toHaveBeenCalledWith('/repo', 'x')

    await user.click(screen.getByRole('button', { name: 'Add a command' }))
    let dialog = await screen.findByRole('dialog', { name: 'Add a command' })
    await user.type(within(dialog).getByRole('textbox'), 'deploy')
    await user.click(within(dialog).getByRole('button', { name: 'Next' }))
    dialog = await screen.findByRole('dialog', { name: 'Add a command' })
    await user.type(within(dialog).getByRole('textbox'), 'make deploy')
    await user.click(within(dialog).getByRole('button', { name: 'Add' }))
    await vi.waitFor(() => expect(TaskService.AddCustom).toHaveBeenCalledWith('/repo', 'deploy', 'make deploy'))
  })
})
