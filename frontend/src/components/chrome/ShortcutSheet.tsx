import Kbd from '../controls/Kbd'
import Modal from './Modal'
import { SHORTCUT_GROUPS, shortcutLabel } from '../../lib/shortcuts'
import './ShortcutSheet.scss'

function ShortcutSheet({ onClose }: { onClose: () => void }) {
  return (
    <Modal title="Keyboard shortcuts" onClose={onClose} className="shortcut-sheet-modal">
      <div className="shortcut-sheet">
        {SHORTCUT_GROUPS.map((group) => (
          <section key={group.name} className="shortcut-sheet-group">
            <h3 className="shortcut-sheet-name">{group.name}</h3>
            <dl className="shortcut-sheet-list">
              {group.shortcuts.map(({ keys, label }) => (
                <div key={label} className="shortcut-sheet-row">
                  <dt>{label}</dt>
                  <dd>
                    <Kbd>{shortcutLabel(keys)}</Kbd>
                  </dd>
                </div>
              ))}
            </dl>
          </section>
        ))}
      </div>
    </Modal>
  )
}

export default ShortcutSheet
