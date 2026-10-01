import './ResizeHandle.scss'

interface ResizeHandleProps {
  onDragStart: (e: React.MouseEvent) => void
  ariaLabel: string
}

function ResizeHandle({ onDragStart, ariaLabel }: ResizeHandleProps) {
  return (
    <div
      className="resize-handle"
      onMouseDown={onDragStart}
      role="separator"
      aria-orientation="vertical"
      aria-label={ariaLabel}
    />
  )
}

export default ResizeHandle
