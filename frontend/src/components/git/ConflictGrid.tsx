import { useRef, useState, type KeyboardEvent } from 'react'
import { plural } from '../../lib/plural'
import './ConflictGrid.scss'

export type GridCell = 'clean' | 'unchecked' | { files: string[] }

interface ConflictGridProps {
  label: string
  names: string[]
  cell: (a: number, b: number) => GridCell
  selected: [number, number] | null
  onSelect: (a: number, b: number) => void
}

const STEPS: Record<string, [number, number]> = {
  ArrowUp: [-1, 0],
  ArrowDown: [1, 0],
  ArrowLeft: [0, -1],
  ArrowRight: [0, 1],
}

function inWords(cell: GridCell): string {
  if (cell === 'clean') return 'no conflicts'
  return cell === 'unchecked' ? 'not checked' : `${plural(cell.files.length, 'file')} would conflict`
}

const MARK = { clean: '·', unchecked: '?' }

// Every name against every other. Both halves are filled, so a row can be read across.
function ConflictGrid({ label, names, cell, selected, onSelect }: ConflictGridProps) {
  const table = useRef<HTMLTableElement>(null)
  const [focused, setFocused] = useState<[number, number]>([0, 1])
  const at: [number, number] = focused.every((i) => i < names.length) ? focused : [0, 1]

  const onKeyDown = (e: KeyboardEvent) => {
    const step = STEPS[e.key]
    let next: [number, number]
    if (step) {
      next = [at[0] + step[0], at[1] + step[1]]
      // Nothing to compare with itself.
      if (next[0] === next[1]) next = [next[0] + step[0], next[1] + step[1]]
    } else if (e.key === 'Home') next = [at[0], at[0] === 0 ? 1 : 0]
    else if (e.key === 'End') next = [at[0], names.length - (at[0] === names.length - 1 ? 2 : 1)]
    else if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      onSelect(...at)
      return
    } else return
    e.preventDefault()
    table.current?.querySelector<HTMLElement>(`[data-cell="${next.join(',')}"]`)?.focus()
  }

  return (
    <div className="conflict-grid">
      <table ref={table} role="grid" aria-label={label} onKeyDown={onKeyDown}>
        <thead>
          <tr>
            <td />
            {names.map((name, i) => (
              <th key={name} scope="col" title={name} aria-label={name}>
                {i + 1}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {names.map((name, a) => (
            <tr key={name}>
              <th scope="row" title={name} aria-label={name}>
                <span className="conflict-grid-number">{a + 1}</span>
                <span className="conflict-grid-name">{name}</span>
              </th>
              {names.map((other, b) => {
                if (a === b) return <td key={other} className="conflict-grid-self" aria-hidden="true" />
                const found = cell(a, b)
                const isSelected = selected?.[0] === a && selected[1] === b
                return (
                  <td
                    key={other}
                    role="gridcell"
                    data-cell={`${a},${b}`}
                    tabIndex={at[0] === a && at[1] === b ? 0 : -1}
                    aria-selected={isSelected}
                    aria-label={`${name} and ${other}, ${inWords(found)}`}
                    title={`${name} and ${other}: ${inWords(found)}`}
                    className={typeof found === 'string' ? undefined : 'conflict-grid-conflict'}
                    onFocus={() => setFocused([a, b])}
                    onClick={() => onSelect(a, b)}
                  >
                    {typeof found === 'string' ? MARK[found] : found.files.length}
                  </td>
                )
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export default ConflictGrid
