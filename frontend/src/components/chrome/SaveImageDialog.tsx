import { useState, type ReactNode } from 'react'
import { PlatformService } from '@current-client-bindings/app'
import { encodeBase64 } from '../../lib/base64'
import { errorMessage } from '../../lib/errors'
import { writePng, writeSvg, type Picture } from '../../lib/picture'
import { PICTURE_FONTS } from '../../lib/pictureFonts'
import { localDate } from '../../lib/relativeTime'
import SegmentedControl from '../controls/SegmentedControl'
import DialogError from './DialogError'
import Modal from './Modal'

interface SaveImageDialogProps {
  // The file name's start; the date and extension follow it.
  name: string
  picture: () => Picture
  // Further options, shown after the format choice.
  children?: ReactNode
  onClose: () => void
}

const safe = (name: string) => name.replace(/[^\p{L}\p{N}._-]+/gu, '-').replace(/^[-.]+|-+$/g, '')

function SaveImageDialog({ name, picture, children, onClose }: SaveImageDialogProps) {
  const [kind, setKind] = useState<'png' | 'svg'>('png')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const save = async () => {
    setSaving(true)
    setError(null)
    try {
      const drawn = picture()
      const bytes =
        kind === 'png'
          ? new Uint8Array(await (await writePng(drawn)).arrayBuffer())
          : new TextEncoder().encode(await writeSvg(drawn, PICTURE_FONTS))
      const saved = await PlatformService.SaveImage(encodeBase64(bytes), `${safe(name)}-${localDate()}.${kind}`, kind)
      if (saved) onClose()
    } catch (err: unknown) {
      setError(errorMessage(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title="Save as image" onClose={onClose} className="dialog-panel">
      <div className="dialog-body">
        <SegmentedControl
          value={kind}
          onChange={setKind}
          options={[
            { value: 'png', label: 'PNG' },
            { value: 'svg', label: 'SVG' },
          ]}
        />
        {children}
        <p className="dialog-note">Saved where you choose, and nowhere else.</p>
        <DialogError>{error}</DialogError>
        <div className="dialog-actions">
          <button type="button" className="dialog-button" onClick={onClose}>
            Cancel
          </button>
          <button
            type="button"
            className="dialog-button dialog-button-primary"
            disabled={saving}
            onClick={() => void save()}
          >
            {saving ? 'Saving…' : 'Save…'}
          </button>
        </div>
      </div>
    </Modal>
  )
}

export default SaveImageDialog
