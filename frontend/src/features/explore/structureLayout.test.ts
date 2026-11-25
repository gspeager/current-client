import type { StructureLink, StructureNode } from '@current-client-bindings/app'
import { cutMiddle, drawnOf, layoutStructure, linePath, nextBox } from './structureLayout'

const node = (path: string, side = 'card'): StructureNode => ({
  path,
  label: path,
  side,
  folder: true,
  count: { files: 1, lines: 1, changed: 0 },
  state: '',
  doc: '',
  slot: 1,
})
const link = (from: number, to: number, main = true): StructureLink => ({ from, to, count: 1, main, state: '' })

describe('layoutStructure', () => {
  it('puts what uses a card left of it, with the sides first and last', () => {
    const nodes = [node('a'), node('b'), node('c'), node('in', 'used-by'), node('out', 'uses')]
    const { boxes } = layoutStructure(nodes, [link(0, 1), link(1, 2), link(3, 2), link(0, 4)])
    const [a, b, c, used, uses] = boxes.map((box) => box.x)
    expect([used, a, b, c, uses]).toEqual([...[used, a, b, c, uses]].sort((x, y) => x - y))
    expect(new Set([used, a, b, c, uses]).size).toBe(5)
  })

  it('draws every line from one card to another, level but for its turns, and through none', () => {
    const nodes = Array.from({ length: 12 }, (_, i) => node(`n${i}`))
    const links = nodes.flatMap((_, i) => [i + 1, i + 3, i + 5].filter((k) => k < 12).map((k) => link(i, k)))
    const { boxes, lines } = layoutStructure(nodes, links)
    links.forEach(({ from, to }, i) => {
      const line = lines[i]
      expect(line[0].x).toBe(boxes[from].x + boxes[from].width)
      expect(line[line.length - 1].x).toBe(boxes[to].x)
      for (let k = 1; k < line.length; k++) {
        const [p, q] = [line[k - 1], line[k]]
        expect(p.x === q.x || p.y === q.y).toBe(true)
        boxes.forEach((box, card) => {
          if (card === from || card === to) return
          const inside =
            Math.max(p.x, q.x) > box.x + 1 &&
            Math.min(p.x, q.x) < box.x + box.width - 1 &&
            Math.max(p.y, q.y) > box.y + 1 &&
            Math.min(p.y, q.y) < box.y + box.height - 1
          expect(inside, `${from} to ${to} crosses ${card}`).toBe(false)
        })
      }
    })
  })

  it('gives cards that use each other a line each way', () => {
    const { boxes, lines } = layoutStructure([node('a'), node('b')], [link(0, 1), link(1, 0)])
    expect(boxes[0].x).not.toBe(boxes[1].x)
    expect(lines[0][0].y).not.toBe(lines[1][lines[1].length - 1].y)
    expect(lines.every((line) => line.length >= 2)).toBe(true)
  })

  it('puts cards with no line below, in rows', () => {
    const nodes = [node('a'), node('b'), ...Array.from({ length: 6 }, (_, i) => node(`alone${i}`))]
    const { boxes, othersAt, height } = layoutStructure(nodes, [link(0, 1)])
    expect(boxes.slice(2).every((box) => box.y >= othersAt!)).toBe(true)
    expect(new Set(boxes.slice(2).map((box) => `${box.x},${box.y}`)).size).toBe(6)
    expect(height).toBeGreaterThan(othersAt!)
    expect(layoutStructure([node('a')], []).othersAt).toBe(0)
    expect(layoutStructure([], [])).toMatchObject({ width: 0, height: 0 })
  })
})

it('draws the main links, and outside only what they go to', () => {
  const nodes = [node('a'), node('out', 'uses'), node('b'), node('in', 'used-by')]
  const links = [link(0, 1, false), link(3, 2)]
  const main = drawnOf(nodes, links, false)
  expect(main.nodes.map((n) => n.path)).toEqual(['a', 'b', 'in'])
  expect(main.links).toMatchObject([{ from: 2, to: 1 }])
  expect(drawnOf(nodes, links, true).nodes).toHaveLength(4)
})

it('rounds corners, and cuts names in the middle', () => {
  expect(
    linePath([
      { x: 0, y: 0 },
      { x: 20, y: 0 },
      { x: 20, y: 40 },
    ]),
  ).toBe('M0,0L12,0Q20,0 20,8L20,40')
  expect(
    linePath([
      { x: 0, y: 0 },
      { x: 5, y: 0 },
    ]),
  ).toBe('M0,0L5,0')
  expect(cutMiddle('MediaBrowser.Controller', 13)).toBe('MediaB…roller')
  expect(cutMiddle('short', 13)).toBe('short')
})

it('goes to the nearest card that way', () => {
  const box = (x: number, y: number) => ({ x, y, width: 10, height: 10 })
  const boxes = [box(0, 0), box(50, 0), box(50, 100), box(100, 5)]
  expect(nextBox(boxes, 0, 'right')).toBe(1)
  expect(nextBox(boxes, 1, 'down')).toBe(2)
  expect(nextBox(boxes, 1, 'right')).toBe(3)
  expect(nextBox(boxes, 0, 'left')).toBe(0)
})
