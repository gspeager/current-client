import './ResizeHandle.scss'

interface ResizeHandleProps {
  onDragStart: (e: React.MouseEvent) => void
  ariaLabel: string
  // Between panes stacked one above the other.
  horizontal?: boolean
}

function ResizeHandle({ onDragStart, ariaLabel, horizontal = false }: ResizeHandleProps) {
  return (
    <div
      className={horizontal ? 'resize-handle resize-handle-horizontal' : 'resize-handle'}
      onMouseDown={onDragStart}
      role="separator"
      aria-orientation={horizontal ? 'horizontal' : 'vertical'}
      aria-label={ariaLabel}
    />
  )
}

export default ResizeHandle
