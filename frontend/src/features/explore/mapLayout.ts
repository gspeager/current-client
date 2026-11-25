import { hierarchy, treemap, type HierarchyRectangularNode } from 'd3-hierarchy'
import type { Step } from './mapSteps'

export interface MapItem {
  path: string
  // What the tile is sized by.
  size: number
  slot: number
  step: Step
  // In changes: A, M, D, R or T.
  status?: string
  conflict?: boolean
}

interface Box {
  x0: number
  y0: number
  x1: number
  y1: number
}

export interface Tile extends Box {
  item: MapItem
  name: string
}

export interface FolderBox extends Box {
  path: string
  name: string
  // Whether there's room for its name.
  header: boolean
  files: number
}

export interface MapLayout {
  width: number
  height: number
  tiles: Tile[]
  folders: FolderBox[]
}

interface Node {
  name: string
  path: string
  item?: MapItem
  children?: Node[]
}

export const HEADER = 20
const HEADER_MIN_WIDTH = 56
const HEADER_MIN_HEIGHT = 44
const ROOMY = 36
// Pixels a tile. Under this, whole gaps would leave more ground than tiles.
const CROWDED = 60

export const inFolder = (path: string, folder: string) => folder === '' || path.startsWith(`${folder}/`)

function buildTree(items: MapItem[], root: string): Node {
  const top: Node = { name: '', path: root, children: [] }
  const folders = new Map<string, Node>([[root, top]])
  const folderAt = (path: string): Node => {
    const known = folders.get(path)
    if (known) return known
    const cut = path.lastIndexOf('/')
    const node: Node = { name: path.slice(cut + 1), path, children: [] }
    folders.set(path, node)
    folderAt(cut < 0 ? '' : path.slice(0, cut)).children!.push(node)
    return node
  }
  for (const item of items) {
    if (!inFolder(item.path, root)) continue
    const cut = item.path.lastIndexOf('/')
    const folder = folderAt(cut < 0 ? '' : item.path.slice(0, cut))
    folder.children!.push({ name: item.path.slice(cut + 1), path: item.path, item })
  }
  return top
}

// A folder that holds only one folder is one step, named by both.
function joinLone(node: Node): Node {
  if (!node.children) return node
  let { name, path, children } = node
  while (children.length === 1 && children[0].children) {
    const [only]: Node[] = children
    name = name ? `${name}/${only.name}` : only.name
    path = only.path
    children = only.children!
  }
  return { name, path, children: children.map(joinLone) }
}

const area = (node: HierarchyRectangularNode<Node>) => (node.x1 - node.x0) * (node.y1 - node.y0)

const roomFor = (node: HierarchyRectangularNode<Node>) =>
  node.depth > 0 && node.x1 - node.x0 >= HEADER_MIN_WIDTH && node.y1 - node.y0 >= HEADER_MIN_HEIGHT

export function layoutMap(items: MapItem[], root: string, width: number, height: number): MapLayout {
  const layout: MapLayout = { width, height, tiles: [], folders: [] }
  if (width <= 0 || height <= 0) return layout
  const top = buildTree(items, root)
  if (top.children!.length === 0) return layout
  const joined: Node = { ...top, children: top.children!.map(joinLone) }
  const sized = hierarchy(joined)
    // Every file is seen, an empty one too.
    .sum((node) => (node.item ? Math.max(1, node.item.size) : 0))
    .sort((a, b) => b.value! - a.value! || a.data.name.localeCompare(b.data.name))
  const placed = treemap<Node>()
    .size([width, height])
    .paddingInner((node) => (area(node) / (node.children?.length ?? 1) < CROWDED ? 0.4 : 1))
    .paddingOuter((node) => (node.depth > 0 && node.x1 - node.x0 > ROOMY && node.y1 - node.y0 > ROOMY ? 2 : 0))
    .paddingTop((node) => (roomFor(node) ? HEADER : 0))(sized)
  placed.eachBefore((node) => {
    const box = { x0: node.x0, y0: node.y0, x1: node.x1, y1: node.y1 }
    if (node.data.item) layout.tiles.push({ ...box, item: node.data.item, name: node.data.name })
    else if (node.depth > 0) {
      layout.folders.push({
        ...box,
        path: node.data.path,
        name: node.data.name,
        header: roomFor(node),
        files: node.leaves().length,
      })
    }
  })
  return layout
}

const contains = (box: Box, x: number, y: number) => x >= box.x0 && x < box.x1 && y >= box.y0 && y < box.y1

export function tileAt(layout: MapLayout, x: number, y: number): number {
  return layout.tiles.findIndex((tile) => contains(tile, x, y))
}

// The folder whose name is there.
export function headerAt(layout: MapLayout, x: number, y: number): FolderBox | undefined {
  return layout.folders.find((folder) => folder.header && contains({ ...folder, y1: folder.y0 + HEADER }, x, y))
}

export type Direction = 'left' | 'right' | 'up' | 'down'

// The nearest tile that way, straight ahead before to a side.
export function neighbor(layout: MapLayout, from: number, direction: Direction): number {
  const start = layout.tiles[from]
  const across = direction === 'left' || direction === 'right'
  const forward = direction === 'right' || direction === 'down'
  const center = (tile: Tile) => [(tile.x0 + tile.x1) / 2, (tile.y0 + tile.y1) / 2]
  const [x, y] = center(start)
  let best = from
  let least = Infinity
  layout.tiles.forEach((tile, i) => {
    if (i === from) return
    const [low, high] = across ? [tile.x0, tile.x1] : [tile.y0, tile.y1]
    const gap = forward ? low - (across ? start.x1 : start.y1) : (across ? start.x0 : start.y0) - high
    if (gap < -1) return
    const [tx, ty] = center(tile)
    const aside = across
      ? Math.max(0, tile.y0 - y, y - tile.y1) * 3 + Math.abs(ty - y) / 100
      : Math.max(0, tile.x0 - x, x - tile.x1) * 3 + Math.abs(tx - x) / 100
    const distance = gap + aside
    if (distance < least) {
      least = distance
      best = i
    }
  })
  return best
}
