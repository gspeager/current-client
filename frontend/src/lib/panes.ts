// Moves focus between the app's panes, marked with data-pane in reading order.
export function cyclePanes(backwards: boolean, root: Document | HTMLElement = document) {
  const panes = [...root.querySelectorAll<HTMLElement>('[data-pane]')].filter(
    (p) => !p.querySelector('[data-pane]') && (p.offsetParent !== null || p.getClientRects().length > 0),
  )
  if (panes.length === 0) return
  const at = panes.findIndex((p) => p.contains(document.activeElement))
  const next = panes[(at + (backwards ? -1 : 1) + panes.length) % panes.length]
  const target =
    next.querySelector<HTMLElement>('[tabindex="0"]') ??
    next.querySelector<HTMLElement>('button:not([disabled]), a[href], input, select, textarea, summary')
  target?.focus()
}

// Where focus goes when a dialog closes: back where it was, else into the main pane.
export function restoreFocus(previous: Element | null) {
  if (previous instanceof HTMLElement && previous.isConnected && previous !== document.body) {
    previous.focus()
    return
  }
  document
    .querySelector<HTMLElement>('[data-pane="main"] [data-roving] [tabindex="0"], [data-pane="main"] button')
    ?.focus()
}
