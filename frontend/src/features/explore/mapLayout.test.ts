import type { Painter } from '../../lib/picture'
import { drawMap, type MapColors } from './drawMap'
import { headerAt, layoutMap, neighbor, tileAt, type MapItem } from './mapLayout'

const item = (path: string, size: number, over: Partial<MapItem> = {}): MapItem => ({
  path,
  size,
  slot: 1,
  step: 2,
  ...over,
})

const items = [item('src/a.ts', 300), item('src/b.ts', 100), item('docs/guide/x.md', 100), item('empty.txt', 0)]

describe('layoutMap', () => {
  it('sizes tiles by what they hold, an empty file among them', () => {
    const layout = layoutMap(items, '', 800, 600)
    expect(layout.tiles.map((tile) => tile.item.path)).toEqual(['src/a.ts', 'src/b.ts', 'docs/guide/x.md', 'empty.txt'])
    const area = (path: string) => {
      const tile = layout.tiles.find((t) => t.item.path === path)!
      return (tile.x1 - tile.x0) * (tile.y1 - tile.y0)
    }
    expect(area('src/a.ts') / area('src/b.ts')).toBeCloseTo(3, 0)
    expect(area('empty.txt')).toBeGreaterThan(0)
  })

  it('names folders, two in one when one holds only the other', () => {
    const { folders } = layoutMap(items, '', 800, 600)
    expect(folders.map(({ name, path, header, files }) => [name, path, header, files])).toEqual([
      ['src', 'src', true, 2],
      ['docs/guide', 'docs/guide', true, 1],
    ])
  })

  it('fills with one folder', () => {
    const layout = layoutMap(items, 'src', 800, 600)
    expect(layout.tiles.map((tile) => tile.name)).toEqual(['a.ts', 'b.ts'])
    expect(layoutMap(items, 'docs', 800, 600).folders.map((folder) => folder.path)).toEqual(['docs/guide'])
  })

  it('finds what is at a point', () => {
    const layout = layoutMap(items, '', 800, 600)
    const tile = layout.tiles[1]
    expect(tileAt(layout, tile.x0 + 1, tile.y0 + 1)).toBe(1)
    const [src] = layout.folders
    expect(headerAt(layout, src.x0 + 3, src.y0 + 3)?.path).toBe('src')
    expect(headerAt(layout, tile.x0 + 1, tile.y0 + 1)).toBeUndefined()
  })

  it('goes to the next tile that way, and nowhere past the edge', () => {
    const grid = layoutMap(
      ['a', 'b', 'c', 'd'].map((name) => item(name, 10)),
      '',
      400,
      400,
    )
    // Three down the left, one the height of the right.
    const [a, b, c, d] = ['a', 'b', 'c', 'd'].map((name) => grid.tiles.findIndex((tile) => tile.name === name))
    expect(neighbor(grid, a, 'right')).toBe(d)
    expect(neighbor(grid, a, 'down')).toBe(b)
    expect(neighbor(grid, a, 'up')).toBe(a)
    expect(neighbor(grid, c, 'up')).toBe(b)
    expect(neighbor(grid, d, 'left')).toBe(b)
    expect(neighbor(grid, d, 'right')).toBe(d)
  })

  it('lays out 30,000 files, each reachable', () => {
    const many = Array.from({ length: 30_000 }, (_, i) => item(`d${i % 40}/s${i % 7}/f${i}.go`, (i % 300) + 1))
    const layout = layoutMap(many, '', 1400, 800)
    expect(layout.tiles).toHaveLength(30_000)
    expect(layout.tiles.every((tile) => tile.x1 >= tile.x0 && tile.y1 >= tile.y0)).toBe(true)
    const seen = new Set([0])
    for (let i = 0, at = 0; i < 50; i++) {
      at = neighbor(layout, at, i % 2 ? 'right' : 'down')
      seen.add(at)
    }
    expect(seen.size).toBeGreaterThan(10)
  })
})

describe('drawMap', () => {
  const colors: MapColors = {
    ground: 'ground',
    folder: 'folder',
    name: 'name',
    muted: 'muted',
    onFolder: 'on',
    warning: 'warning',
    steps: Array.from({ length: 9 }, (_, slot) => [1, 2, 3, 4].map((step) => `f${slot}-${step}`)),
  }
  const drawn = (list: MapItem[]) => {
    const calls: string[] = []
    const painter: Painter = {
      rect: (_x, _y, _w, _h, fill) => calls.push(`rect ${fill}`),
      frame: (_x, _y, _w, _h, color, line) => calls.push(`frame ${color} ${line}`),
      line: (_x0, _y0, _x1, _y1, color) => calls.push(`line ${color}`),
      text: (_x, _y, text, { color }) => calls.push(`text ${text} ${color}`),
      path: () => undefined,
      circle: () => undefined,
      within: (_x, _y, _o, draw) => draw(),
    }
    drawMap(painter, layoutMap(list, '', 400, 300), colors)
    return calls
  }

  it('fills a tile with its folder and step, and names it', () => {
    expect(drawn([item('src/a.ts', 10, { slot: 3, step: 4 })])).toEqual([
      'rect ground',
      'rect folder',
      'text src name',
      'rect f3-4',
      'text a.ts on',
    ])
  })

  it('says what happened to a file without its color', () => {
    const change = (status: string, over: Partial<MapItem> = {}) => drawn([item('x.ts', 10, { status, ...over })])
    expect(change('A')).toEqual(['rect ground', 'rect f1-2', 'text A on', 'text x.ts on'])
    expect(change('R')).toEqual(['rect ground', 'rect f1-2', 'frame on 1', 'text R on', 'text x.ts on'])
    expect(change('D')).toEqual(['rect ground', 'frame f1-2 1', 'line f1-2', 'text D muted', 'text x.ts muted'])
    expect(change('M', { conflict: true })).toEqual([
      'rect ground',
      'rect f1-2',
      'frame warning 3',
      'text M! on',
      'text x.ts on',
    ])
  })
})
