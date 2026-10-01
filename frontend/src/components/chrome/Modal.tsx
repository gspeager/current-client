import type { ReactNode } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import { useCurrentClientRoot } from '../../lib/currentClientRoot'
import './Modal.scss'

interface ModalProps {
  title: string
  onClose: () => void
  className: string
  placement?: 'center' | 'top'
  headerContent?: ReactNode
  hideHeader?: boolean
  children: ReactNode
}

function Modal({
  title,
  onClose,
  className,
  placement = 'center',
  headerContent,
  hideHeader = false,
  children,
}: ModalProps) {
  const { element } = useCurrentClientRoot()
  return (
    <Dialog.Root open onOpenChange={(open) => !open && onClose()}>
      <Dialog.Portal container={element}>
        <Dialog.Overlay className={`modal-overlay modal-overlay-${placement}`}>
          <Dialog.Content className={`modal-panel ${className}`} aria-describedby={undefined}>
            {hideHeader ? (
              <Dialog.Title className="modal-hidden-title">{title}</Dialog.Title>
            ) : (
              <div className="modal-header">
                <Dialog.Title className="modal-title">{title}</Dialog.Title>
                {headerContent}
                <Dialog.Close className="modal-close" aria-label="Close" title="Close">
                  <X size={16} strokeWidth={1.75} />
                </Dialog.Close>
              </div>
            )}
            {children}
          </Dialog.Content>
        </Dialog.Overlay>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

export default Modal
