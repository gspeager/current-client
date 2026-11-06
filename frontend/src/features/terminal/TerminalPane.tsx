import { useEffect, useRef } from 'react'
import { Events } from '@wailsio/runtime'
import { FitAddon } from '@xterm/addon-fit'
import { Terminal, type ITheme } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'
import { TerminalService } from '@current-client-bindings/app'
import { isTerminalToggle } from './terminalKeys'

const OUTPUT_EVENT = 'terminal:output'
const EXIT_EVENT = 'terminal:exit'

function themeFrom(element: HTMLElement): ITheme {
  const style = getComputedStyle(element)
  const token = (name: string) => style.getPropertyValue(name).trim() || undefined
  return {
    background: token('--surface-lowest') ?? token('--surface'),
    foreground: token('--on-surface'),
    cursor: token('--on-surface'),
    selectionBackground: token('--surface-container-high') ?? token('--surface-container'),
  }
}

function decode(data: string): Uint8Array {
  return Uint8Array.from(atob(data), (char) => char.charCodeAt(0))
}

interface TerminalPaneProps {
  id: string
  visible: boolean
  // Changes when the panel is asked to take the focus.
  focusRequest: number
  theme?: string
  onExit: () => void
}

// One shell. Its scrollback is only here, in memory. It stays mounted while hidden,
// so a shell in a background tab keeps its output.
function TerminalPane({ id, visible, focusRequest, theme, onExit }: TerminalPaneProps) {
  const host = useRef<HTMLDivElement>(null)
  const terminal = useRef<Terminal | null>(null)
  const fit = useRef<FitAddon | null>(null)
  const exited = useRef(onExit)
  exited.current = onExit

  useEffect(() => {
    const element = host.current!
    const term = new Terminal({
      cursorBlink: false,
      fontFamily: getComputedStyle(element).getPropertyValue('--font-mono').trim() || 'monospace',
      fontSize: 12,
      lineHeight: 1.2,
      theme: themeFrom(element),
      scrollback: 5000,
    })
    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(element)
    // The toggle goes on to the app, copy and paste are here since Ctrl+C and Ctrl+V
    // belong to the shell, and everything else goes to the shell.
    term.attachCustomKeyEventHandler((event) => {
      if (isTerminalToggle(event)) return false
      if (event.type === 'keydown' && event.ctrlKey && event.shiftKey && event.code === 'KeyC') {
        void navigator.clipboard?.writeText(term.getSelection())
        return false
      }
      if (event.type === 'keydown' && event.ctrlKey && event.shiftKey && event.code === 'KeyV') {
        void navigator.clipboard?.readText().then((text) => term.paste(text))
        return false
      }
      return true
    })
    const typed = term.onData((data) => void TerminalService.Write(id, data).catch(() => undefined))
    const resized = term.onResize(
      ({ cols, rows }) => void TerminalService.Resize(id, cols, rows).catch(() => undefined),
    )
    terminal.current = term
    fit.current = fitAddon

    const offOutput = Events.On(OUTPUT_EVENT, ({ data }: { data: { id: string; data: string } }) => {
      if (data.id !== id) return
      const bytes = decode(data.data)
      term.write(bytes, () => void TerminalService.Ack(id, bytes.length).catch(() => undefined))
    })
    const offExit = Events.On(EXIT_EVENT, ({ data }: { data: { id: string } }) => {
      if (data.id === id) exited.current()
    })
    void TerminalService.Attach(id).catch(() => undefined)
    return () => {
      offOutput()
      offExit()
      typed.dispose()
      resized.dispose()
      term.dispose()
      terminal.current = null
    }
  }, [id])

  // Keys typed here don't reach the app's own shortcuts on window.
  useEffect(() => {
    const element = host.current!
    const stop = (event: KeyboardEvent) => {
      if (!isTerminalToggle(event)) event.stopPropagation()
    }
    element.addEventListener('keydown', stop)
    return () => element.removeEventListener('keydown', stop)
  }, [])

  useEffect(() => {
    const element = host.current!
    const fitNow = () => {
      if (element.offsetWidth > 0 && element.offsetHeight > 0) fit.current?.fit()
    }
    if (visible) fitNow()
    if (typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(fitNow)
    observer.observe(element)
    return () => observer.disconnect()
  }, [visible])

  // Only for a new request: a repository tab switch shouldn't pull the focus in.
  const focused = useRef(0)
  useEffect(() => {
    if (visible && focusRequest !== focused.current) {
      focused.current = focusRequest
      terminal.current?.focus()
    }
  }, [visible, focusRequest])

  useEffect(() => {
    if (terminal.current && host.current) terminal.current.options.theme = themeFrom(host.current)
  }, [theme])

  return <div ref={host} className="terminal-pane" hidden={!visible} data-terminal={id} />
}

export default TerminalPane
