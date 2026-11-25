import type { CommitInfo, GraphNode, RefInfo } from '@current-client-bindings/app'
import { writeSvg } from '../../lib/picture'
import { themeTokens, tokenOf } from '../../test/themeTokens'
import { graphPicture, type GraphPictureOptions, type GraphPictureRow } from './graphPicture'

const commit = (sha: string, subject: string): CommitInfo => ({
  sha: sha.repeat(40),
  parentShas: [],
  authorName: 'Ada Lovelace',
  authorEmail: 'ada@example.com',
  date: '2026-01-10T12:00:00Z',
  subject,
  body: '',
  signature: 'none',
  coAuthors: [],
})

const node = (over: Partial<GraphNode>): GraphNode => ({
  sha: 'a',
  lane: 0,
  hasChild: true,
  parentLanes: [0],
  convergingLanes: [],
  passThrough: [],
  ...over,
})

const ref = (name: string, kind: string): RefInfo => ({ name, kind, sha: 'a' })

const ada = { initials: 'AL', color: '#7cb7ff' } as GraphPictureRow['identity']

const ROWS: GraphPictureRow[] = [
  { text: '2 uncommitted files', node: node({ hasChild: false }) },
  {
    commit: commit('a', 'Merge branch feat'),
    node: node({ parentLanes: [0, 1] }),
    head: true,
    refs: [ref('main', 'branch'), ref('origin/main', 'remote'), ref('v1.0', 'tag')],
    identity: ada,
    time: '2h ago',
  },
  { commit: commit('b', 'Add x < y & z'), node: node({ lane: 1, parentLanes: [1], passThrough: [0] }), identity: ada },
  { commit: commit('c', 'Initial commit'), node: node({ parentLanes: [], convergingLanes: [1] }), dimmed: true },
]

const save = async (tokens: Map<string, string>, over: Partial<GraphPictureOptions> = {}) => {
  const options: GraphPictureOptions = {
    rows: ROWS,
    width: 900,
    rowHeight: 38,
    laneCount: 2,
    names: true,
    upstreams: { main: 'origin/main' },
    laneColor: (lane) => ['var(--branch-main)', 'var(--branch-feature)'][lane],
    refColor: () => 'var(--branch-main)',
    color: (color) => (color.startsWith('var(') ? tokens.get(tokenOf(color))! : color),
    ...over,
  }
  const svg = new DOMParser().parseFromString(await writeSvg(graphPicture(options), []), 'image/svg+xml')
  return { svg, said: [...svg.querySelectorAll('text')].map((el) => el.textContent) }
}

beforeEach(() => {
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null)
})

describe.each(Object.entries(themeTokens()))('the saved graph, %s theme', (_theme, tokens) => {
  it("is in the theme's own colors", async () => {
    const { svg } = await save(tokens)
    expect(svg.documentElement.outerHTML).not.toMatch(/var\(|undefined/)
    const row = [...svg.documentElement.children].filter((el) => el.tagName === 'g')
    expect(row).toHaveLength(4)
    // HEAD's ring, then its node.
    const circles = [...row[1].querySelectorAll(':scope > g circle')]
    expect(circles.map((el) => [el.getAttribute('stroke'), el.getAttribute('fill')])).toEqual([
      [tokens.get('--primary'), 'none'],
      [null, tokens.get('--branch-main')],
    ])
    expect(row[1].querySelector('path')?.getAttribute('stroke')).toBe(tokens.get('--branch-feature'))
    expect(row[0].querySelector('circle')?.getAttribute('stroke-dasharray')).toBe('2')
    const subject = [...row[2].querySelectorAll('text')].find((el) => el.textContent === 'Add x < y & z')
    expect(subject?.getAttribute('fill')).toBe(tokens.get('--on-surface'))
    expect(row[3].getAttribute('opacity')).toBe('0.4')
  })

  it('says what each row does', async () => {
    const { said } = await save(tokens)
    expect(said).toEqual([
      '2 uncommitted files',
      ...['HEAD', 'main', 'v1.0', 'Merge branch feat', 'AL', 'Ada Lovelace', 'aaaaaaa', '2h ago'],
      ...['Add x < y & z', 'AL', 'Ada Lovelace', 'bbbbbbb', ''],
      ...['Initial commit', 'Ada Lovelace', 'ccccccc', ''],
    ])
  })
})

describe('the saved graph', () => {
  const { dark } = themeTokens()

  it('leaves out who made each commit', async () => {
    const { said, svg } = await save(dark, { names: false })
    expect(said).not.toContain('Ada Lovelace')
    expect(said).not.toContain('AL')
    expect(svg.documentElement.outerHTML).not.toContain('ada@example.com')
    expect(svg.querySelector('circle[r="10"]')).toBeNull()
  })

  it('counts branches there is no room for', async () => {
    const refs = Array.from({ length: 9 }, (_, i) => ref(`feature/number-${i}`, 'branch'))
    const { said } = await save(dark, { rows: [{ ...ROWS[1], head: false, refs }] })
    const shown = said.filter((text) => text?.startsWith('feat'))
    expect(shown.length).toBeGreaterThan(0)
    expect(said).toContain(`+${9 - shown.length}`)
    expect(said).toContain('Merge branch feat')
  })
})
