import clsx from 'clsx'
import { useLayoutEffect, useMemo, useRef, useState, type KeyboardEvent } from 'react'
import type { StructureLink, StructureNode } from '@current-client-bindings/app'
import { counted } from './structureWords'
import { cutMiddle, layoutStructure, linePath, nextBox, type Arrow } from './structureLayout'
import './StructureGraph.scss'

interface StructureGraphProps {
  label: string
  nodes: StructureNode[]
  // The ones drawn.
  links: StructureLink[]
  // Commits by path, for a file's card.
  commitsOf: (path: string) => number | undefined
  compared: boolean
  onOpen: (path: string) => void
  onUp?: () => void
  // What a card says under its name, where it isn't its size.
  describe?: (node: StructureNode) => string
}

const ARROWS: Record<string, Arrow> = {
  ArrowLeft: 'left',
  ArrowRight: 'right',
  ArrowUp: 'up',
  ArrowDown: 'down',
}
// A side's label gives way to what happened to it.
const LABEL_CHARACTERS = { card: 22, side: 21, marked: 13 }
// Text is 12px, and no smaller than 9.6.
const SMALLEST = 0.8

const colorOf = (node: StructureNode | undefined) => `var(--folder-${node && node.slot > 0 ? node.slot : 'other'}-2)`

function facts(node: StructureNode, commits: number | undefined, compared: boolean): string {
  if (node.state === 'removed') return 'removed'
  const { files, lines, changed } = node.count
  const isNew = node.state === 'added'
  const isChanged = compared && changed > 0 && !isNew
  const parts = node.folder
    ? [counted(files, 'file'), isChanged ? `${changed.toLocaleString()} changed` : counted(lines, 'line')]
    : [counted(lines, 'line'), isChanged ? 'changed' : commits !== undefined && counted(commits, 'commit')]
  return [isNew && 'new', ...parts].filter(Boolean).join(' · ')
}

// Cards and the lines between them. Everything a line says is in its cards' names for
// a screen reader.
function StructureGraph({ label, nodes, links, commitsOf, compared, onOpen, onUp, describe }: StructureGraphProps) {
  const factsOf = (node: StructureNode) => describe?.(node) ?? facts(node, commitsOf(node.path), compared)
  const box = useRef<HTMLDivElement>(null)
  const [room, setRoom] = useState(0)
  const [pointed, setPointed] = useState<number | null>(null)
  const [focused, setFocused] = useState(0)
  const layout = useMemo(() => layoutStructure(nodes, links), [nodes, links])
  const at = Math.min(focused, Math.max(0, nodes.length - 1))

  useLayoutEffect(() => {
    const el = box.current!
    const measure = () => el.clientWidth > 0 && setRoom(el.clientWidth)
    measure()
    if (typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(measure)
    observer.observe(el)
    return () => observer.disconnect()
  }, [])

  const scale = room > 0 && layout.width > room ? Math.max(SMALLEST, room / layout.width) : 1
  const touching = (i: number) => links.flatMap(({ from, to }) => (from === i ? [to] : to === i ? [from] : []))
  const lit = pointed === null ? null : new Set([pointed, ...touching(pointed)])

  const inWords = (node: StructureNode, i: number) => {
    const names = (others: number[]) => others.map((k) => nodes[k].label).join(', ')
    const uses = names(links.flatMap(({ from, to }) => (from === i ? [to] : [])))
    const usedBy = names(links.flatMap(({ from, to }) => (to === i ? [from] : [])))
    return [
      node.path,
      node.side === 'used-by' ? 'outside, uses this' : node.side === 'uses' ? 'outside, used by this' : '',
      factsOf(node),
      uses && `uses ${uses}`,
      usedBy && `used by ${usedBy}`,
    ]
      .filter(Boolean)
      .join(', ')
  }

  const onKeyDown = (e: KeyboardEvent) => {
    if (e.altKey || e.ctrlKey || e.metaKey) return
    const way = ARROWS[e.key]
    if (way) box.current?.querySelector<HTMLElement>(`[data-node="${nextBox(layout.boxes, at, way)}"]`)?.focus()
    else if ((e.key === 'Backspace' || e.key === 'Escape') && onUp) onUp()
    else return
    e.preventDefault()
    e.stopPropagation()
  }

  const caption = (side: string, text: string) => {
    const first = nodes.findIndex((node) => node.side === side)
    return first < 0 ? null : (
      <span className="structure-caption" style={{ left: layout.boxes[first].x }}>
        {text}
      </span>
    )
  }

  return (
    <div ref={box} className="structure-graph" role="group" aria-label={label} onKeyDown={onKeyDown}>
      <div className="structure-sized" style={{ width: layout.width * scale, height: (layout.height + 28) * scale }}>
        <div
          className="structure-drawn"
          style={{ width: layout.width, height: layout.height, transform: `scale(${scale})` }}
        >
          {caption('used-by', 'Used by')}
          {caption('uses', 'Uses')}
          {layout.othersAt !== null && layout.lines.length > 0 && (
            <span className="structure-caption" style={{ left: 0, top: layout.othersAt }}>
              Other cards
            </span>
          )}
          <svg width={layout.width} height={layout.height} aria-hidden="true">
            {links.map(({ from, to, count, state }, i) => {
              const points = layout.lines[i]
              const end = points[points.length - 1]
              const back = layout.boxes[to].x < layout.boxes[from].x ? 5 : -5
              const middle = points[Math.floor(points.length / 2)]
              const dim = lit !== null && from !== pointed && to !== pointed
              return (
                <g
                  key={`${from}-${to}`}
                  className={clsx('structure-line', dim && 'structure-dim')}
                  stroke={colorOf(nodes[from].side === 'card' ? nodes[from] : nodes[to])}
                  strokeWidth={Math.min(3, 1.2 + Math.sqrt(count) * 0.15)}
                  strokeDasharray={state === 'removed' ? '4 4' : undefined}
                >
                  <path d={linePath(points)} />
                  <path
                    strokeDasharray="none"
                    d={`M${end.x + back},${end.y - 4}L${end.x},${end.y}L${end.x + back},${end.y + 4}`}
                  />
                  {state && (
                    <text className="structure-mark" x={middle.x} y={middle.y - 4} strokeWidth={0}>
                      {state === 'added' ? '+' : '−'}
                    </text>
                  )}
                </g>
              )
            })}
          </svg>
          {nodes.map((node, i) => {
            const place = layout.boxes[i]
            const card = node.side === 'card'
            const classes = ['structure-card']
            if (!card) classes.push('structure-card-side')
            if (node.state) classes.push(`structure-card-${node.state}`)
            if (lit && !lit.has(i)) classes.push('structure-dim')
            return (
              <button
                key={`${node.side}:${node.path}`}
                type="button"
                data-node={i}
                className={classes.join(' ')}
                style={{ left: place.x, top: place.y, width: place.width, height: place.height }}
                tabIndex={i === at ? 0 : -1}
                title={node.doc ? `${node.path}\n${node.doc}` : node.path}
                aria-label={inWords(node, i)}
                onClick={() => onOpen(node.path)}
                onFocus={() => {
                  setFocused(i)
                  setPointed(i)
                }}
                onBlur={() => setPointed(null)}
                onMouseEnter={() => setPointed(i)}
                onMouseLeave={() => setPointed(null)}
              >
                {card && touching(i).length > 0 && (
                  <span className="structure-card-color" style={{ background: colorOf(node) }} />
                )}
                <span className="structure-card-name">
                  {cutMiddle(node.label, LABEL_CHARACTERS[card ? 'card' : node.state ? 'marked' : 'side'])}
                </span>
                {card && <span className="structure-card-facts">{factsOf(node)}</span>}
                {!card && node.state && (
                  <span className="structure-card-state">{node.state === 'added' ? 'new' : 'removed'}</span>
                )}
              </button>
            )
          })}
        </div>
      </div>
    </div>
  )
}

export default StructureGraph
