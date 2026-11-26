import { useCallback, useEffect, useRef, useState } from 'react'
import { SettingsService } from '@current-client-bindings/app'

function clamp(width: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, width))
}

type ResizeDirection = 'grow-right' | 'grow-left'
type ResizeAxis = 'x' | 'y'

const pointer = (e: { clientX: number; clientY: number }, axis: ResizeAxis) => (axis === 'y' ? e.clientY : e.clientX)

// 'grow-left' is for a pane to the right of its divider.
export function useResizableWidth(
  storageKey: string,
  defaultWidth: number,
  min: number,
  max: number,
  direction: ResizeDirection = 'grow-right',
  axis: ResizeAxis = 'x',
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
  }, [storageKey, min, max])

  const onMouseMove = useCallback(
    (e: MouseEvent) => {
      setWidth(clamp(startWidth.current + sign * (pointer(e, axis) - startX.current), min, max))
    },
    [min, max, sign, axis],
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
      startX.current = pointer(e, axis)
      startWidth.current = width
      window.addEventListener('mousemove', onMouseMove)
      window.addEventListener('mouseup', onMouseUp)
    },
    [width, onMouseMove, onMouseUp, axis],
  )

  return { width, onDragStart }
}
