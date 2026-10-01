import { useCallback, useEffect, useRef, useState } from 'react'
import { SettingsService } from '@current-client-bindings/app'

function clamp(width: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, width))
}

type ResizeDirection = 'grow-right' | 'grow-left'

// 'grow-left' is for a pane to the right of its divider.
export function useResizableWidth(
  storageKey: string,
  defaultWidth: number,
  min: number,
  max: number,
  direction: ResizeDirection = 'grow-right',
) {
  const [width, setWidth] = useState(clamp(defaultWidth, min, max))
  const startX = useRef(0)
  const startWidth = useRef(0)
  const sign = direction === 'grow-right' ? 1 : -1

  useEffect(() => {
    SettingsService.GetPaneWidths()
      .then((widths) => {
        const stored = widths?.[storageKey]
        if (typeof stored === 'number' && stored > 0) {
          setWidth(clamp(stored, min, max))
        }
      })
      .catch(() => {})
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [storageKey])

  const onMouseMove = useCallback(
    (e: MouseEvent) => {
      const next = Math.min(max, Math.max(min, startWidth.current + sign * (e.clientX - startX.current)))
      setWidth(next)
    },
    [min, max, sign],
  )

  const onMouseUp = useCallback(() => {
    window.removeEventListener('mousemove', onMouseMove)
    window.removeEventListener('mouseup', onMouseUp)
    setWidth((current) => {
      SettingsService.SetPaneWidth(storageKey, current).catch(() => {})
      return current
    })
  }, [onMouseMove, storageKey])

  const onDragStart = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault()
      startX.current = e.clientX
      startWidth.current = width
      window.addEventListener('mousemove', onMouseMove)
      window.addEventListener('mouseup', onMouseUp)
    },
    [width, onMouseMove, onMouseUp],
  )

  return { width, onDragStart }
}
