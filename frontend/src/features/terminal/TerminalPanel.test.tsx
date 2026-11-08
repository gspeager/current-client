import { useState } from 'react'
import { act, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SettingsService, TerminalService } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import TerminalPanel from './TerminalPanel'
import { isTerminalToggle } from './terminalKeys'
import { useTerminals } from './useTerminals'

const { created, handlers, FakeTerminal } = vi.hoisted(() => {
  const created: {
    written: Uint8Array[]
    focused: number
    dataListener: ((data: string) => void) | null
    keyHandler: ((event: KeyboardEvent) => boolean) | null
  }[] = []
  const handlers: Record<string, (event: { data: unknown }) => void> = {}
  class FakeTerminal {
    options: Record<string, unknown> = {}
    written: Uint8Array[] = []
    focused = 0
    dataListener: ((data: string) => void) | null = null
    keyHandler: ((event: KeyboardEvent) => boolean) | null = null
    constructor() {
      created.push(this)
    }
    loadAddon() {}
    open() {}
    attachCustomKeyEventHandler(handler: (event: KeyboardEvent) => boolean) {
      this.keyHandler = handler
    }
    onData(listener: (data: string) => void) {
      this.dataListener = listener
      return { dispose: () => undefined }
    }
    onResize() {
      return { dispose: () => undefined }
    }
    write(data: Uint8Array, done?: () => void) {
      this.written.push(data)
      done?.()
    }
    focus() {
      this.focused += 1
    }
    dispose() {}
  }
  return { created, handlers, FakeTerminal }
})

vi.mock('@xterm/xterm', () => ({ Terminal: FakeTerminal }))
vi.mock('@xterm/addon-fit', () => ({
  FitAddon: class {
    fit() {}
  },
}))
vi.mock('@xterm/xterm/css/xterm.css', () => ({}))
vi.mock('@wailsio/runtime', async (importOriginal) => ({
  ...(await importOriginal<object>()),
  Events: {
    On: (name: string, handler: (event: { data: unknown }) => void) => {
      handlers[name] = handler
      return () => undefined
    },
  },
}))

function Harness({ repoPath = '/repo' }: { repoPath?: string }) {
  const terminals = useTerminals(repoPath)
  const [focus] = useState(1)
  return (
    <>
      <button type="button" onClick={() => terminals.show(repoPath)}>
        open
      </button>
      <button
        type="button"
        onClick={() =>
          void terminals
            .confirmCloseRepo(repoPath)
            .then((closed) => closed && document.body.setAttribute('data-closed', 'yes'))
        }
      >
        close tab
      </button>
      <TerminalPanel repoPath={repoPath} terminals={terminals} focusRequest={focus} />
    </>
  )
}

function renderPanel() {
  return render(
    <DialogProvider>
      <Harness />
    </DialogProvider>,
  )
}

beforeEach(() => {
  created.length = 0
  document.body.removeAttribute('data-closed')
  vi.mocked(SettingsService.GetPaneWidths).mockResolvedValue({})
  vi.mocked(TerminalService.Start).mockResolvedValue('t1')
  vi.mocked(TerminalService.Attach).mockResolvedValue()
  vi.mocked(TerminalService.Close).mockResolvedValue()
  vi.mocked(TerminalService.CloseRepo).mockResolvedValue()
  vi.mocked(TerminalService.Write).mockResolvedValue()
  vi.mocked(TerminalService.Ack).mockResolvedValue()
  vi.mocked(TerminalService.Resize).mockResolvedValue()
})

describe('TerminalPanel', () => {
  it('opens a shell in the repository and shows its output', async () => {
    const user = userEvent.setup()
    renderPanel()
    await user.click(screen.getByRole('button', { name: 'open' }))

    expect(await screen.findByRole('tab', { name: 'Shell 1' })).toBeInTheDocument()
    expect(TerminalService.Start).toHaveBeenCalledWith('/repo', 80, 24)
    expect(TerminalService.Attach).toHaveBeenCalledWith('t1')

    await act(async () => handlers['terminal:output']({ data: { id: 't1', data: btoa('hi') } }))
    expect(new TextDecoder().decode(created[0].written[0])).toBe('hi')
    expect(TerminalService.Ack).toHaveBeenCalledWith('t1', 2)
  })

  it('sends what is typed to the shell', async () => {
    const user = userEvent.setup()
    renderPanel()
    await user.click(screen.getByRole('button', { name: 'open' }))
    await screen.findByRole('tab', { name: 'Shell 1' })

    created[0].dataListener?.('ls\r')
    expect(TerminalService.Write).toHaveBeenCalledWith('t1', 'ls\r')
  })

  it('takes the focus when asked', async () => {
    const user = userEvent.setup()
    renderPanel()
    await user.click(screen.getByRole('button', { name: 'open' }))
    await screen.findByRole('tab', { name: 'Shell 1' })
    expect(created[0].focused).toBeGreaterThan(0)
  })

  it('lets the toggle shortcut leave the terminal, and keeps every other key', async () => {
    const user = userEvent.setup()
    renderPanel()
    await user.click(screen.getByRole('button', { name: 'open' }))
    await screen.findByRole('tab', { name: 'Shell 1' })

    const toggle = new KeyboardEvent('keydown', { ctrlKey: true, code: 'Backquote' })
    expect(isTerminalToggle(toggle)).toBe(true)
    expect(created[0].keyHandler?.(toggle)).toBe(false)
    expect(created[0].keyHandler?.(new KeyboardEvent('keydown', { ctrlKey: true, code: 'KeyK' }))).toBe(true)

    const reached = vi.fn()
    window.addEventListener('keydown', reached)
    const pane = document.querySelector('[data-terminal="t1"]')!
    pane.dispatchEvent(new KeyboardEvent('keydown', { bubbles: true, ctrlKey: true, code: 'KeyK' }))
    expect(reached).not.toHaveBeenCalled()
    pane.dispatchEvent(new KeyboardEvent('keydown', { bubbles: true, ctrlKey: true, code: 'Backquote' }))
    expect(reached).toHaveBeenCalledTimes(1)
    window.removeEventListener('keydown', reached)
  })

  it('hides without ending the shell, and closes the panel when the shell exits', async () => {
    const user = userEvent.setup()
    renderPanel()
    await user.click(screen.getByRole('button', { name: 'open' }))
    await screen.findByRole('tab', { name: 'Shell 1' })

    await user.click(screen.getByRole('button', { name: 'Hide' }))
    expect(TerminalService.Close).not.toHaveBeenCalled()
    expect(document.querySelector('.terminal-panel')).toHaveAttribute('hidden')

    await user.click(screen.getByRole('button', { name: 'open' }))
    expect(document.querySelector('.terminal-panel')).not.toHaveAttribute('hidden')

    await act(async () => handlers['terminal:exit']({ data: { id: 't1', code: 0 } }))
    expect(screen.queryByRole('tab', { name: 'Shell 1' })).not.toBeInTheDocument()
  })

  it('asks before closing a shell with a process running', async () => {
    const user = userEvent.setup()
    vi.mocked(TerminalService.Busy).mockResolvedValue(true)
    renderPanel()
    await user.click(screen.getByRole('button', { name: 'open' }))
    await user.click(await screen.findByRole('button', { name: 'Close Shell 1' }))

    const dialog = await screen.findByRole('alertdialog', { name: 'Close terminal' })
    expect(TerminalService.Close).not.toHaveBeenCalled()
    await user.click(within(dialog).getByRole('button', { name: 'Close terminal' }))
    await vi.waitFor(() => expect(TerminalService.Close).toHaveBeenCalledWith('t1'))
  })

  it('closes an idle shell straight away', async () => {
    const user = userEvent.setup()
    vi.mocked(TerminalService.Busy).mockResolvedValue(false)
    renderPanel()
    await user.click(screen.getByRole('button', { name: 'open' }))
    await user.click(await screen.findByRole('button', { name: 'Close Shell 1' }))
    await vi.waitFor(() => expect(TerminalService.Close).toHaveBeenCalledWith('t1'))
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  })

  it('asks before closing a repository tab with a process running', async () => {
    const user = userEvent.setup()
    vi.mocked(TerminalService.RepoBusy).mockResolvedValue(true)
    renderPanel()
    await user.click(screen.getByRole('button', { name: 'close tab' }))

    const dialog = await screen.findByRole('alertdialog', { name: 'Close repository tab' })
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }))
    await act(async () => {})
    expect(TerminalService.CloseRepo).not.toHaveBeenCalled()
    expect(document.body.getAttribute('data-closed')).toBeNull()

    await user.click(screen.getByRole('button', { name: 'close tab' }))
    const again = await screen.findByRole('alertdialog', { name: 'Close repository tab' })
    await user.click(within(again).getByRole('button', { name: 'Close tab' }))
    await vi.waitFor(() => expect(document.body.getAttribute('data-closed')).toBe('yes'))
    expect(TerminalService.CloseRepo).toHaveBeenCalledWith('/repo')
  })

  it('says why a shell could not start', async () => {
    const user = userEvent.setup()
    vi.mocked(TerminalService.Start).mockRejectedValue(new Error('turned off by organization policy'))
    function WithError() {
      const terminals = useTerminals('/repo')
      return (
        <>
          <button type="button" onClick={() => terminals.show('/repo')}>
            open
          </button>
          {terminals.error && <p>{terminals.error}</p>}
        </>
      )
    }
    render(
      <DialogProvider>
        <WithError />
      </DialogProvider>,
    )
    await user.click(screen.getByRole('button', { name: 'open' }))
    expect(await screen.findByText(/turned off by organization policy/)).toBeInTheDocument()
  })
})
