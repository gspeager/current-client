import { useMemo, useState, type FormEvent, type ReactNode } from 'react'
import * as AlertDialog from '@radix-ui/react-alert-dialog'
import { useCurrentClientRoot } from '../../lib/currentClientRoot'
import { DialogContext, type ConfirmOptions, type Dialogs, type PromptOptions } from '../../lib/useDialogs'
import Modal from './Modal'
import './DialogProvider.scss'

type Request =
  | { kind: 'confirm'; options: ConfirmOptions; resolve: (confirmed: boolean) => void }
  | { kind: 'prompt'; options: PromptOptions; resolve: (value: string | null) => void }

function ConfirmDialog({ options, onSettle }: { options: ConfirmOptions; onSettle: (confirmed: boolean) => void }) {
  const { element } = useCurrentClientRoot()
  return (
    <AlertDialog.Root open onOpenChange={(open) => !open && onSettle(false)}>
      <AlertDialog.Portal container={element}>
        <AlertDialog.Overlay className="modal-overlay modal-overlay-center">
          <AlertDialog.Content className="modal-panel dialog-panel">
            <div className="modal-header">
              <AlertDialog.Title className="modal-title">{options.title}</AlertDialog.Title>
            </div>
            <div className="dialog-body">
              <AlertDialog.Description className="dialog-message">{options.message}</AlertDialog.Description>
              <div className="dialog-actions">
                <AlertDialog.Cancel className="dialog-button">Cancel</AlertDialog.Cancel>
                <AlertDialog.Action
                  className={`dialog-button ${options.destructive ? 'dialog-button-destructive' : 'dialog-button-primary'}`}
                  onClick={() => onSettle(true)}
                >
                  {options.confirmLabel}
                </AlertDialog.Action>
              </div>
            </div>
          </AlertDialog.Content>
        </AlertDialog.Overlay>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  )
}

function PromptDialog({ options, onSettle }: { options: PromptOptions; onSettle: (value: string | null) => void }) {
  const [value, setValue] = useState(options.initialValue ?? '')
  const trimmed = value.trim()
  const canSubmit = trimmed !== '' && trimmed !== options.initialValue

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (canSubmit) onSettle(trimmed)
  }

  return (
    <Modal title={options.title} onClose={() => onSettle(null)} className="dialog-panel">
      <form className="dialog-body" onSubmit={submit}>
        <label className="dialog-field">
          <span className="dialog-message">{options.label}</span>
          <input className="dialog-input" value={value} onChange={(e) => setValue(e.target.value)} autoFocus />
        </label>
        <div className="dialog-actions">
          <button type="button" className="dialog-button" onClick={() => onSettle(null)}>
            Cancel
          </button>
          <button type="submit" className="dialog-button dialog-button-primary" disabled={!canSubmit}>
            {options.confirmLabel}
          </button>
        </div>
      </form>
    </Modal>
  )
}

export function DialogProvider({ children }: { children: ReactNode }) {
  const [request, setRequest] = useState<Request | null>(null)

  const dialogs = useMemo<Dialogs>(
    () => ({
      confirm: (options) => new Promise((resolve) => setRequest({ kind: 'confirm', options, resolve })),
      prompt: (options) => new Promise((resolve) => setRequest({ kind: 'prompt', options, resolve })),
    }),
    [],
  )

  return (
    <DialogContext.Provider value={dialogs}>
      {children}
      {request?.kind === 'confirm' && (
        <ConfirmDialog
          options={request.options}
          onSettle={(confirmed) => {
            request.resolve(confirmed)
            setRequest(null)
          }}
        />
      )}
      {request?.kind === 'prompt' && (
        <PromptDialog
          options={request.options}
          onSettle={(value) => {
            request.resolve(value)
            setRequest(null)
          }}
        />
      )}
    </DialogContext.Provider>
  )
}
