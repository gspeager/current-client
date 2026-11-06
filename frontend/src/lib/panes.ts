// Moves focus between the app's panes, marked with data-pane in reading order.
export function cyclePanes(backwards: boolean, root: Document | HTMLElement = document) {
  const panes = [...root.querySelectorAll<HTMLElement>('[data-pane]')].filter(
    (p) => p.offsetParent !== null || p.getClientRects().length > 0,
  )
  if (panes.length === 0) return
  const at = panes.findIndex((p) => p.contains(document.activeElement))
  const next = panes[(at + (backwards ? -1 : 1) + panes.length) % panes.length]
  const target =
    next.querySelector<HTMLElement>('[tabindex="0"]') ??
    next.querySelector<HTMLElement>('button:not([disabled]), a[href], input, select, textarea, summary')
  target?.focus()
}
