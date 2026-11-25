import dagre, { type RankerFunction } from '@dagrejs/dagre'
import type { StructureLink, StructureNode } from '@current-client-bindings/app'

export interface Box {
  x: number
  y: number
  width: number
  height: number
}

export interface Point {
  x: number
  y: number
}

export interface StructureLayout {
  width: number
  height: number
  // By node.
  boxes: Box[]
  // By link, from what uses to what it uses.
  lines: Point[][]
  // Where the cards nothing is drawn to begin, below the rest.
  othersAt: number | null
}

// What's drawn: the main links or all of them, every card, and what's outside where a
// line goes to it.
export function drawnOf(
  nodes: StructureNode[],
  links: StructureLink[],
  all: boolean,
): { nodes: StructureNode[]; links: StructureLink[] } {
  const shown = links.filter((link) => all || link.main)
  const linked = new Set(shown.flatMap(({ from, to }) => [from, to]))
  const kept = nodes.flatMap((node, i) => (node.side === 'card' || linked.has(i) ? [i] : []))
  const at = new Map(kept.map((i, k) => [i, k]))
  return {
    nodes: kept.map((i) => nodes[i]),
    links: shown.map((link) => ({ ...link, from: at.get(link.from)!, to: at.get(link.to)! })),
  }
}

const CARD = { width: 188, height: 54 }
const SIDE = { width: 176, height: 34 }
const GAP = 16
const RANK_GAP = 72
const BESIDE = 12
const OTHERS = 40

const sizeOf = (node: StructureNode) => (node.side === 'card' ? CARD : SIDE)

type Graph = InstanceType<typeof dagre.graphlib.Graph>

// Each card one column past the furthest it's used by, with what outside uses them
// first and what they use outside last. One nothing here uses stands next to what it
// uses. The lines are already without cycles, and dagre has a node of its own.
function rankCards(nodes: StructureNode[]) {
  return (graph: Graph) => {
    const ids = graph.nodes().filter((id) => nodes[Number(id)])
    const edges = graph
      .edges()
      .filter(({ v, w }) => nodes[Number(v)] && nodes[Number(w)])
      .map((edge) => ({ from: edge.v, to: edge.w, least: (graph.edge(edge) as { minlen?: number }).minlen ?? 1 }))
    const rank = new Map<string, number>()
    const visit = (id: string): number => {
      const known = rank.get(id)
      if (known !== undefined) return known
      const at = Math.max(0, ...edges.flatMap(({ from, to, least }) => (to === id ? [visit(from) + least] : [])))
      rank.set(id, at)
      return at
    }
    ids.forEach(visit)
    for (const id of ids) {
      const used = edges.some(({ to }) => to === id)
      const after = edges.flatMap(({ from, to, least }) => (from === id ? [rank.get(to)! - least] : []))
      if (!used && after.length > 0) rank.set(id, Math.min(...after))
    }
    const sideOf = (id: string) => nodes[Number(id)].side
    const step = Math.max(1, ...edges.map(({ least }) => least))
    const inner = ids.filter((id) => sideOf(id) === 'card').map((id) => rank.get(id)!)
    const [first, last] = [Math.min(...inner), Math.max(...inner)]
    for (const id of graph.nodes()) {
      const side = nodes[Number(id)]?.side
      const at = side === 'card' ? rank.get(id)! : side === 'uses' ? last + step : first - step
      ;(graph.node(id) as { rank?: number }).rank = at
    }
  }
}

// Cards in dependency order from the left, with lines that go around them.
export function layoutStructure(nodes: StructureNode[], links: StructureLink[]): StructureLayout {
  const linked = new Set(links.flatMap(({ from, to }) => [from, to]))
  const boxes: Box[] = nodes.map((node) => ({ x: 0, y: 0, ...sizeOf(node) }))
  let width = 0
  let height = 0
  let lines: Point[][] = []
  if (linked.size > 0) {
    const graph = new dagre.graphlib.Graph({ multigraph: true })
    graph.setGraph({
      rankdir: 'LR',
      nodesep: GAP,
      ranksep: RANK_GAP,
      edgesep: BESIDE,
      acyclicer: 'greedy',
      ranker: rankCards(nodes) as RankerFunction,
    })
    nodes.forEach((node, i) => linked.has(i) && graph.setNode(String(i), { ...sizeOf(node) }))
    links.forEach(({ from, to, count }, i) =>
      graph.setEdge(String(from), String(to), { weight: Math.min(5, count), minlen: 1 }, String(i)),
    )
    dagre.layout(graph)
    for (const i of linked) {
      const { x, y } = graph.node(String(i))
      boxes[i].x = x! - boxes[i].width / 2
      boxes[i].y = y! - boxes[i].height / 2
    }
    const passing = links.map(
      ({ from, to }, i) => (graph.edge(String(from), String(to), String(i)).points ?? []) as Point[],
    )
    lines = route(
      links,
      boxes,
      [...linked].map((i) => boxes[i]),
      passing,
    )
    const around = lines.flat()
    const left = Math.min(...[...linked].map((i) => boxes[i].x), ...around.map((p) => p.x))
    const top = Math.min(...[...linked].map((i) => boxes[i].y), ...around.map((p) => p.y))
    for (const at of [...[...linked].map((i) => boxes[i]), ...around]) {
      at.x -= left
      at.y -= top
    }
    width = Math.max(...[...linked].map((i) => boxes[i].x + boxes[i].width), ...around.map((p) => p.x))
    height = Math.max(...[...linked].map((i) => boxes[i].y + boxes[i].height), ...around.map((p) => p.y))
  }

  const others = boxes.filter((_, i) => !linked.has(i))
  if (others.length === 0) return { width, height, boxes, lines, othersAt: null }
  const othersAt = linked.size > 0 ? height + OTHERS : 0
  const across = Math.max(4, Math.floor((width + GAP) / (CARD.width + GAP)))
  others.forEach((box, k) => {
    box.x = (k % across) * (CARD.width + GAP)
    box.y = othersAt + Math.floor(k / across) * (CARD.height + GAP)
  })
  return {
    width: Math.max(width, ...others.map((box) => box.x + box.width)),
    height: Math.max(height, ...others.map((box) => box.y + box.height)),
    boxes,
    lines,
    othersAt,
  }
}

// A stop of a line: a column, and how far down it is there.
interface Stop {
  column: number
  y: number
}

// Lines are level except in the gaps between columns, where each has a place of its
// own to go up or down.
function route(links: StructureLink[], boxes: Box[], placed: Box[], passing: Point[][]): Point[][] {
  const middle = (box: Box) => box.x + box.width / 2
  const centers = [...new Set(placed.map((box) => Math.round(middle(box))))].sort((a, b) => a - b)
  const columns = centers.map((center) => {
    const here = placed.filter((box) => Math.round(middle(box)) === center)
    return { left: Math.min(...here.map((box) => box.x)), right: Math.max(...here.map((box) => box.x + box.width)) }
  })
  const columnAt = (x: number) => columns.findIndex(({ left, right }) => x >= left && x <= right)
  const columnOf = (card: number) => columnAt(middle(boxes[card]))
  const ends = endsOf(links, boxes, columnOf)
  const stops = links.map(({ from, to }, i): Stop[] => {
    const [start, end] = [columnOf(from), columnOf(to)]
    const between = passing[i].flatMap(({ x, y }) => {
      const column = columnAt(x)
      return column > Math.min(start, end) && column < Math.max(start, end) ? [{ column, y }] : []
    })
    const each = between.filter(({ column }, k) => between.findIndex((stop) => stop.column === column) === k)
    return [{ column: start, y: ends[i].from }, ...each, { column: end, y: ends[i].to }]
  })

  // By gap, the turns in it.
  const turns = new Map<number, { line: number; hop: number; from: number; to: number }[]>()
  stops.forEach((line, i) => {
    for (let hop = 0; hop < line.length - 1; hop++) {
      const [a, b] = [line[hop], line[hop + 1]]
      const [left, right] = a.column < b.column ? [a, b] : [b, a]
      const gap = a.column < b.column ? left.column : right.column - 1
      turns.set(gap, [...(turns.get(gap) ?? []), { line: i, hop, from: left.y, to: right.y }])
    }
  })
  const turnAt = stops.map((line) => line.map(() => 0))
  for (const [gap, here] of turns) {
    // Going down, the lowest first; going up, the highest, so they cross least.
    here.sort((a, b) => {
      const [downA, downB] = [a.to >= a.from, b.to >= b.from]
      if (downA !== downB) return downA ? 1 : -1
      return downA ? b.from - a.from || b.to - a.to : a.from - b.from || a.to - b.to
    })
    const { right } = columns[gap]
    const { left } = columns[gap + 1]
    here.forEach(({ line, hop }, k) => {
      turnAt[line][hop] = right + ((left - right) * (k + 1)) / (here.length + 1)
    })
  }
  return links.map(({ from, to }, i) => {
    const line = stops[i]
    const forward = line[0].column < line[line.length - 1].column
    const edge = (box: Box, rightSide: boolean) => (rightSide ? box.x + box.width : box.x)
    const points: Point[] = [{ x: edge(boxes[from], forward), y: line[0].y }]
    for (let hop = 0; hop < line.length - 1; hop++) {
      const x = turnAt[i][hop]
      points.push({ x, y: line[hop].y }, { x, y: line[hop + 1].y })
    }
    points.push({ x: edge(boxes[to], !forward), y: line[line.length - 1].y })
    return points
  })
}

// Each line has its own place on a card's edge, in the order of where it goes.
function endsOf(links: StructureLink[], boxes: Box[], columnOf: (card: number) => number) {
  const middle = (box: Box) => box.y + box.height / 2
  const ends = links.map(() => ({ from: 0, to: 0 }))
  const forward = links.map(({ from, to }) => columnOf(from) < columnOf(to))
  boxes.forEach((box, card) => {
    for (const right of [false, true]) {
      const here: { line: number; end: 'from' | 'to'; other: number }[] = []
      links.forEach(({ from, to }, line) => {
        if (from === card && right === forward[line]) here.push({ line, end: 'from', other: middle(boxes[to]) })
        if (to === card && right !== forward[line]) here.push({ line, end: 'to', other: middle(boxes[from]) })
      })
      here.sort((a, b) => a.other - b.other)
      here.forEach(({ line, end }, k) => {
        ends[line][end] = box.y + (box.height * (k + 1)) / (here.length + 1)
      })
    }
  })
  return ends
}

// A path through the points, its corners rounded.
export function linePath(points: Point[], radius = 8): string {
  const [first, ...rest] = points
  if (rest.length === 0) return ''
  const toward = (from: Point, to: Point, by: number) => {
    const length = Math.hypot(to.x - from.x, to.y - from.y) || 1
    const part = Math.min(by, length / 2) / length
    return { x: from.x + (to.x - from.x) * part, y: from.y + (to.y - from.y) * part }
  }
  const n = (value: number) => Math.round(value * 10) / 10
  let d = `M${n(first.x)},${n(first.y)}`
  for (let i = 1; i < points.length - 1; i++) {
    const before = toward(points[i], points[i - 1], radius)
    const after = toward(points[i], points[i + 1], radius)
    d += `L${n(before.x)},${n(before.y)}Q${n(points[i].x)},${n(points[i].y)} ${n(after.x)},${n(after.y)}`
  }
  const last = points[points.length - 1]
  return `${d}L${n(last.x)},${n(last.y)}`
}

export type Arrow = 'left' | 'right' | 'up' | 'down'

// The nearest box that way, those straight ahead first.
export function nextBox(boxes: Box[], from: number, way: Arrow): number {
  const center = (box: Box) => ({ x: box.x + box.width / 2, y: box.y + box.height / 2 })
  const at = center(boxes[from])
  let best = from
  let least = Infinity
  boxes.forEach((box, i) => {
    const to = center(box)
    const [dx, dy] = [to.x - at.x, to.y - at.y]
    const ahead = { left: -dx, right: dx, up: -dy, down: dy }[way]
    const aside = Math.abs(way === 'left' || way === 'right' ? dy : dx)
    if (i === from || ahead <= 1) return
    const cost = ahead + aside * 2
    if (cost < least) [best, least] = [i, cost]
  })
  return best
}

// Names in a folder often share their start, so the middle goes.
export function cutMiddle(name: string, most: number): string {
  if (name.length <= most) return name
  const end = Math.ceil((most - 1) / 2)
  return `${name.slice(0, most - 1 - end)}…${name.slice(name.length - end)}`
}
