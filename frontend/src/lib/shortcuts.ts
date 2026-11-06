import { System } from '@wailsio/runtime'

export interface Shortcut {
  keys: string
  label: string
}

// Every app-wide shortcut, so the sheet, tooltips and keycaps agree.
export const SHORTCUT_GROUPS: { name: string; shortcuts: Shortcut[] }[] = [
  {
    name: 'Go to',
    shortcuts: [
      { keys: 'mod+K', label: 'Go to anything' },
      { keys: 'mod+shift+F', label: 'Search in files' },
      { keys: 'mod+1…6', label: 'Switch view' },
      { keys: 'F6', label: 'Next pane' },
      { keys: 'mod+B', label: 'Collapse sidebar' },
      { keys: 'ctrl+`', label: 'Terminal panel' },
      { keys: '?', label: 'Keyboard shortcuts' },
    ],
  },
  {
    name: 'Sync',
    shortcuts: [
      { keys: 'mod+shift+T', label: 'Fetch' },
      { keys: 'mod+shift+L', label: 'Pull' },
      { keys: 'mod+shift+K', label: 'Push' },
      { keys: 'mod+shift+Z', label: 'Undo last operation' },
    ],
  },
  {
    name: 'Commit',
    shortcuts: [
      { keys: 'mod+Enter', label: 'Commit staged files' },
      { keys: 'Space', label: 'Stage or unstage the selected file' },
      { keys: 'alt+↑ ↓', label: 'Previous or next hunk' },
      { keys: 'S', label: 'Stage the hunk, from the diff' },
      { keys: 'U', label: 'Unstage the hunk, from the diff' },
      { keys: 'D', label: 'Discard the hunk, from the diff' },
    ],
  },
  {
    name: 'Lists',
    shortcuts: [
      { keys: '↑ ↓', label: 'Move between rows' },
      { keys: 'Home End', label: 'First or last row' },
      { keys: '← →', label: 'Controls in a row' },
      { keys: 'shift+F10', label: 'Open a row’s menu' },
    ],
  },
]

const MAC_KEYS: Record<string, string> = { mod: '⌘', shift: '⇧', ctrl: '⌃', alt: '⌥', Enter: '↩' }
const OTHER_KEYS: Record<string, string> = { mod: 'Ctrl', shift: 'Shift', ctrl: 'Ctrl', alt: 'Alt', Enter: 'Enter' }

export function shortcutLabel(keys: string): string {
  const mac = System.IsMac()
  const names = mac ? MAC_KEYS : OTHER_KEYS
  return keys
    .split('+')
    .map((part) => names[part] ?? part)
    .join(mac ? '' : ' ')
}

export function withShortcut(text: string, keys: string): string {
  return `${text} (${shortcutLabel(keys)})`
}

export function isTypingTarget(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null
  return !!el?.matches?.('input, textarea, select, [contenteditable], .xterm-helper-textarea')
}
