import { X } from 'lucide-react'
import type { HostNotice } from '@current-client-bindings/app'
import './HostNoticeBanner.scss'

interface HostNoticeBannerProps {
  notices: HostNotice[]
  onOpenChange: (number: number) => void
  onOpenFile: (path: string) => void
  onDismiss: () => void
}

// One line per refresh with what changed on the checked-out branch's
// change; each item goes to the relevant place. Nothing modal, no sound.
function HostNoticeBanner({ notices, onOpenChange, onOpenFile, onDismiss }: HostNoticeBannerProps) {
  if (notices.length === 0) return null
  const open = (notice: HostNotice) => {
    if (notice.kind === 'threads' && notice.path) onOpenFile(notice.path)
    else onOpenChange(notice.number)
    onDismiss()
  }
  return (
    <p className="app-inline-notice host-notice" role="status">
      {notices.map((notice, i) => (
        <span key={i} className="host-notice-item">
          {i > 0 && <span className="host-notice-separator">·</span>}
          <button type="button" className="host-notice-link" onClick={() => open(notice)}>
            {notice.text}
          </button>
        </span>
      ))}
      <button type="button" className="host-notice-dismiss" onClick={onDismiss} aria-label="Dismiss">
        <X size={14} strokeWidth={1.75} />
      </button>
    </p>
  )
}

export default HostNoticeBanner
