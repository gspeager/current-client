import type { ReactNode } from 'react'

interface DialogActionsProps {
  onCancel: () => void
  // The one primary action's label: the form's submit button, or onConfirm's.
  children: ReactNode
  onConfirm?: () => void
  disabled?: boolean
}

function DialogActions({ onCancel, children, onConfirm, disabled }: DialogActionsProps) {
  return (
    <div className="dialog-actions">
      <button type="button" className="dialog-button" onClick={onCancel}>
        Cancel
      </button>
      <button
        type={onConfirm ? 'button' : 'submit'}
        className="dialog-button dialog-button-primary"
        onClick={onConfirm}
        disabled={disabled}
      >
        {children}
      </button>
    </div>
  )
}

export default DialogActions
