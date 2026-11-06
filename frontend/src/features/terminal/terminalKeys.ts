export function isTerminalToggle(event: KeyboardEvent): boolean {
  return event.type === 'keydown' && event.ctrlKey && !event.metaKey && !event.altKey && event.code === 'Backquote'
}
