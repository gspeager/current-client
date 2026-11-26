import { useState } from 'react'
import SaveImageDialog from '../../components/chrome/SaveImageDialog'
import Checkbox from '../../components/forms/Checkbox'
import { withHeading } from '../../lib/picture'
import { plural } from '../../lib/plural'
import { localDate } from '../../lib/relativeTime'
import { graphPicture, type GraphPictureOptions, type GraphPictureRow } from './graphPicture'
import '../../components/chrome/dialog.scss'

// More would be a file too large to open.
export const MOST_ROWS = 500

interface SaveGraphDialogProps {
  repoName: string
  // The list's filters in words; names false leaves out who they name.
  about: (names: boolean) => string
  rows: GraphPictureRow[]
  // The rows on screen, as [first, end).
  shown: [number, number]
  options: Omit<GraphPictureOptions, 'rows' | 'names'>
  onClose: () => void
}

function SaveGraphDialog({ repoName, about, rows, shown, options, onClose }: SaveGraphDialogProps) {
  const [range, setRange] = useState<'shown' | 'from-top'>('shown')
  const most = Math.min(MOST_ROWS, rows.length)
  const [count, setCount] = useState(String(Math.min(100, most)))
  const [names, setNames] = useState(true)
  const fromTop = Math.max(1, Math.min(most, Math.floor(Number(count)) || 1))
  const picked = range === 'shown' ? rows.slice(...shown) : rows.slice(0, fromTop)

  return (
    <SaveImageDialog
      name={`${repoName}-commit-graph`}
      picture={() =>
        withHeading(graphPicture({ ...options, rows: picked, names }), {
          title: repoName,
          about: `Commit graph, ${about(names)}`,
          date: localDate(),
          colors: {
            ground: options.color('var(--surface)'),
            name: options.color('var(--on-surface)'),
            muted: options.color('var(--on-surface-variant)'),
          },
        })
      }
      onClose={onClose}
    >
      <label className="dialog-field">
        <span className="dialog-message">Rows</span>
        <select className="dialog-input" value={range} onChange={(e) => setRange(e.target.value as typeof range)}>
          <option value="shown">On screen ({plural(shown[1] - shown[0], 'row')})</option>
          <option value="from-top">From the top</option>
        </select>
      </label>
      {range === 'from-top' && (
        <label className="dialog-field">
          <span className="dialog-message">How many, up to {most}</span>
          <input
            className="dialog-input"
            type="number"
            min={1}
            max={most}
            value={count}
            onChange={(e) => setCount(e.target.value)}
            onBlur={() => setCount(String(fromTop))}
          />
        </label>
      )}
      <Checkbox checked={names} onChange={setNames} label="People's names and initials" />
    </SaveImageDialog>
  )
}

export default SaveGraphDialog
