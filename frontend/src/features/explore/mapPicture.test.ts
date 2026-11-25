import { writeSvg } from '../../lib/picture'
import { themeTokens } from '../../test/themeTokens'
import { readMapColors } from './drawMap'
import { layoutMap, type MapItem } from './mapLayout'
import { mapPicture, type LegendEntry } from './mapPicture'

const item = (path: string, size: number, over: Partial<MapItem> = {}): MapItem => ({
  path,
  size,
  slot: 1,
  step: 2,
  ...over,
})

const ITEMS = [
  item('src/a.ts', 300, { status: 'M', conflict: true }),
  item('src/b.ts', 100, { status: 'D' }),
  item('docs/x & y.md', 200, { slot: 0, step: 4 }),
]

const LEGEND: LegendEntry[] = [
  { label: 'src', color: '#123456' },
  { label: 'Modified', letter: 'M' },
]

const colorsOf = (tokens: Map<string, string>) => {
  const element = document.createElement('div')
  for (const [token, value] of tokens) element.style.setProperty(token, value)
  document.body.append(element)
  const colors = readMapColors(element)
  element.remove()
  return colors
}

const save = async (tokens: Map<string, string>, legend = LEGEND, width = 800) => {
  const svg = await writeSvg(mapPicture(layoutMap(ITEMS, '', width, 400), legend, colorsOf(tokens)), [])
  return new DOMParser().parseFromString(svg, 'image/svg+xml')
}

beforeEach(() => {
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null)
})

describe.each(Object.entries(themeTokens()))('the saved map, %s theme', (_theme, tokens) => {
  it("is in the theme's own colors", async () => {
    const svg = await save(tokens)
    const fills = [...svg.querySelectorAll('rect')].map((el) => el.getAttribute('fill'))
    expect(fills[0]).toBe(tokens.get('--surface-lowest'))
    expect(fills).toContain(tokens.get('--folder-1-2'))
    expect(fills).toContain(tokens.get('--folder-other-4'))
    expect(fills).toContain('#123456')
    // Deleted is outlined and crossed, and a conflict framed.
    expect(svg.querySelector('line')?.getAttribute('stroke')).toBe(tokens.get('--folder-1-2'))
    expect(svg.querySelector('rect[stroke-width="3"]')?.getAttribute('stroke')).toBe(tokens.get('--warning'))
    expect(svg.documentElement.outerHTML).not.toMatch(/var\(|undefined|""/)
  })

  it('names folders and files, over the legend', async () => {
    const svg = await save(tokens)
    const said = [...svg.querySelectorAll('text')].map((el) => el.textContent)
    expect(said).toEqual(expect.arrayContaining(['src', 'docs', 'M!', 'a.ts', 'D', 'b.ts', 'x & y.md', 'Modified']))
    expect(svg.documentElement.getAttribute('height')).toBe('434')
  })
})

it('wraps a legend wider than the map', async () => {
  const legend = Array.from({ length: 8 }, (_, i) => ({ label: `frontend/src/features/${i}`, color: '#123456' }))
  const svg = await save(themeTokens().dark, legend, 500)
  expect(svg.querySelectorAll('g').length).toBeGreaterThan(1)
  expect(svg.querySelectorAll('g rect')).toHaveLength(8)
})
