import { useEffect, useState } from 'react'

const FOCUSABLE = 'a[href], button:not([disabled]), input:not([disabled]), select, summary, [tabindex]'

interface RovingOptions {
  // Selects the row as the arrow keys reach it, as in a commit or file list.
  selectOnMove?: boolean
}

function focusTarget(row: HTMLElement): HTMLElement | null {
  if (row.matches(FOCUSABLE)) return row
  return row.querySelector<HTMLElement>('.file-row-path') ?? row.querySelector<HTMLElement>(FOCUSABLE)
}

// A list is one Tab stop: arrow keys, Home and End move between its rows, and
// left and right reach the controls inside a row.
export function useRovingRows(rowSelector: string, { selectOnMove = false }: RovingOptions = {}) {
  const [container, setContainer] = useState<HTMLElement | null>(null)

  useEffect(() => {
    if (!container) return
    let current: HTMLElement | null = null

    const rowsOf = () => [...container.querySelectorAll<HTMLElement>(rowSelector)]

    const sync = () => {
      const rows = rowsOf()
      const active =
        current && rows.includes(current)
          ? current
          : (container.querySelector<HTMLElement>(
              `${rowSelector}[aria-selected='true'], ${rowSelector}[class*='-selected']`,
            ) ?? rows[0])
      const stop = active ? focusTarget(active) : null
      for (const row of rows) {
        const targets = [...(row.matches(FOCUSABLE) ? [row] : []), ...row.querySelectorAll<HTMLElement>(FOCUSABLE)]
        for (const el of targets) {
          const tabIndex = el === stop ? '0' : '-1'
          if (el.getAttribute('tabindex') !== tabIndex) el.setAttribute('tabindex', tabIndex)
        }
      }
    }

    const moveTo = (row: HTMLElement) => {
      current = row
      sync()
      focusTarget(row)?.focus()
      row.scrollIntoView?.({ block: 'nearest' })
      if (!selectOnMove) return
      const target = focusTarget(row)
      if (target && target !== row && target.classList.contains('file-row-path')) target.click()
      else row.click()
    }

    const onKeyDown = (e: KeyboardEvent) => {
      const el = e.target as HTMLElement
      if (el.matches('input:not([type=checkbox]), textarea, select')) return
      const row = el.closest<HTMLElement>(rowSelector)
      if (!row || !container.contains(row)) return
      const rows = rowsOf()
      const index = rows.indexOf(row)
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp' || e.key === 'Home' || e.key === 'End') {
        const next =
          e.key === 'ArrowDown'
            ? rows[index + 1]
            : e.key === 'ArrowUp'
              ? rows[index - 1]
              : e.key === 'Home'
                ? rows[0]
                : rows[rows.length - 1]
        e.preventDefault()
        if (next) moveTo(next)
        return
      }
      if (e.key === 'ArrowRight' || e.key === 'ArrowLeft') {
        const inside = [...(row.matches(FOCUSABLE) ? [row] : []), ...row.querySelectorAll<HTMLElement>(FOCUSABLE)]
        const at = inside.indexOf(el)
        const next = inside[at + (e.key === 'ArrowRight' ? 1 : -1)]
        if (at >= 0 && next) {
          e.preventDefault()
          next.focus()
        }
      }
    }

    const onFocusIn = (e: FocusEvent) => {
      const row = (e.target as HTMLElement).closest<HTMLElement>(rowSelector)
      if (row && container.contains(row) && row !== current) {
        current = row
        sync()
      }
    }

    sync()
    const observer = new MutationObserver(sync)
    observer.observe(container, { childList: true, subtree: true })
    container.addEventListener('keydown', onKeyDown)
    container.addEventListener('focusin', onFocusIn)
    return () => {
      observer.disconnect()
      container.removeEventListener('keydown', onKeyDown)
      container.removeEventListener('focusin', onFocusIn)
    }
  }, [container, rowSelector, selectOnMove])

  return setContainer
}
