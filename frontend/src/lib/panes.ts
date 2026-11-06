const visible = (el: HTMLElement) => el.getClientRects().length > 0

// Moves focus between the app's panes, marked with data-pane in reading order.
// A pane holding smaller panes defers to them.
export function cyclePanes(backwards: boolean, root: Document | HTMLElement = document) {
  const shown = [...root.querySelectorAll<HTMLElement>('[data-pane]')].filter(visible)
  const panes = shown.filter((p) => !shown.some((other) => other !== p && p.contains(other)))
  if (panes.length === 0) return
  const at = panes.findIndex((p) => p.contains(document.activeElement))
  const next = panes[(at + (backwards ? -1 : 1) + panes.length) % panes.length]
  const target =
    [...next.querySelectorAll<HTMLElement>('[tabindex="0"]')].find(visible) ??
    [...next.querySelectorAll<HTMLElement>('button:not([disabled]), a[href], input, select, textarea, summary')].find(
      visible,
    )
  target?.focus()
}

// The first thing to act on in the view on screen: its list's stop, else its first control.
export function focusMain() {
  const stops = [
    ...document.querySelectorAll<HTMLElement>(
      '[data-pane="main"] [data-roving] [tabindex="0"], [data-pane="main"] button:not([disabled])',
    ),
  ]
  stops.find(visible)?.focus()
}

// Where focus goes when a dialog closes: back where it was, else into the main pane.
export function restoreFocus(previous: Element | null) {
  if (previous instanceof HTMLElement && previous.isConnected && previous !== document.body && visible(previous)) {
    previous.focus()
    return
  }
  focusMain()
}
