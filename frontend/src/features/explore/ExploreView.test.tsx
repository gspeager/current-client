import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  BranchService,
  CompareService,
  DiffService,
  ExploreService,
  FileDiff,
  HistoryService,
  MapChangesInfo,
  PlatformService,
  StructureInfo,
  TagService,
  type MapChangeInfo,
} from '@current-client-bindings/app'
import { ReadThroughContext } from '../read-through/readThrough'
import { decodeBase64 } from '../../lib/base64'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import ExploreView from './ExploreView'

vi.mock('../../lib/pictureFonts', () => ({ PICTURE_FONTS: [] }))

const file = (path: string, lines: number, over = {}) => ({ path, lines, binary: false, kind: '', ...over })

const FILES = [
  file('src/a.ts', 300),
  file('src/deep/b.ts', 100),
  file('src/a.test.ts', 50, { kind: 'test' }),
  file('docs/x.md', 20),
  file('package-lock.json', 7000, { kind: 'lockfile' }),
  file('logo.png', 0, { binary: true }),
]

const change = (path: string, status: string, over: Partial<MapChangeInfo> = {}): MapChangeInfo => ({
  path,
  origPath: '',
  status,
  added: 3,
  removed: 1,
  binary: false,
  kind: '',
  conflict: false,
  ...over,
})

const readThrough = vi.fn()
const openWorkingFile = vi.fn()

const view = () =>
  render(
    <DialogProvider>
      <ReadThroughContext.Provider value={readThrough}>
        <ExploreView
          repoPath="/repo"
          currentBranch="feat"
          repoVersion={0}
          workingVersion={0}
          onOpenWorkingFile={openWorkingFile}
        />
      </ReadThroughContext.Provider>
    </DialogProvider>,
  )

beforeEach(() => {
  vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockReturnValue(1000)
  vi.spyOn(HTMLElement.prototype, 'clientHeight', 'get').mockReturnValue(600)
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null)
  vi.mocked(ExploreService.Files).mockResolvedValue(FILES)
  vi.mocked(ExploreService.Activity).mockResolvedValue({
    files: { 'src/a.ts': { commits: [1, 4, 9, 12], last: new Date().toISOString() } },
    limit: 0,
  })
  vi.mocked(ExploreService.Changes).mockResolvedValue(new MapChangesInfo({ predicted: true, files: [] }))
  vi.mocked(TagService.ListTags).mockResolvedValue([
    { name: 'v2', sha: 'b', annotated: false, date: '2026-02-01T00:00:00Z' },
    { name: 'v1', sha: 'a', annotated: false, date: '2026-02-01T00:00:00Z' },
  ])
  vi.mocked(BranchService.ListLocal).mockResolvedValue([{ name: 'feat' }, { name: 'main' }] as never)
})

const map = () => screen.findByRole('listbox')
const focusMap = async () => {
  const el = await map()
  act(() => el.focus())
}
const tile = () => document.getElementById('code-map-tile')?.getAttribute('aria-label')
const press = (key: string, over = {}) => fireEvent.keyDown(screen.getByRole('listbox'), { key, ...over })

describe('the code', () => {
  it('leaves lockfiles and binary files out to begin with', async () => {
    view()
    await map()
    expect(screen.getByText('4 files · 470 lines')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('checkbox', { name: 'Lockfiles' }))
    await userEvent.click(screen.getByRole('checkbox', { name: 'Binary' }))
    expect(screen.getByText('6 files · 7,470 lines')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('checkbox', { name: 'Tests' }))
    expect(screen.getByText('5 files · 7,420 lines')).toBeInTheDocument()
  })

  it('names a tile in words, by the keyboard', async () => {
    view()
    await focusMap()
    expect(tile()).toBe('src/a.ts, 300 lines, 4 commits in 90 days, changed just now')
    expect(document.getElementById('code-map-tile')).toHaveAttribute('aria-setsize', '4')
    expect(document.querySelector('.explore-facts')).toHaveTextContent('src/a.ts, 300 lines')
    press('End')
    expect(tile()).toMatch(/^docs\/x\.md, 20 lines, 0 commits in 90 days, changed earlier/)
    press('Home')
    press('ArrowRight')
    expect(tile()).not.toMatch(/^src\/a\.ts/)
    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Period' }), 'All')
    await focusMap()
    press('Home')
    expect(tile()).toMatch(/300 lines, 12 commits, changed/)
  })

  it("opens a file's history", async () => {
    vi.mocked(HistoryService.GetFileHistory).mockResolvedValue([])
    view()
    await focusMap()
    press('Enter')
    await waitFor(() => expect(HistoryService.GetFileHistory).toHaveBeenCalledWith('/repo', 'src/a.ts', 200, 0))
  })

  it('keeps its place while focus is elsewhere', async () => {
    view()
    await focusMap()
    press('End')
    const last = tile()
    act(() => (document.activeElement as HTMLElement).blur())
    expect(tile()).toBeUndefined()
    await focusMap()
    expect(tile()).toBe(last)
  })

  it('says nothing of commits until they are read', async () => {
    vi.mocked(ExploreService.Activity).mockReturnValue(new Promise(() => undefined) as never)
    view()
    await focusMap()
    expect(tile()).toBe('src/a.ts, 300 lines')
  })

  it('opens a folder to fill the map, and goes back', async () => {
    view()
    await focusMap()
    press('Enter', { shiftKey: true })
    expect(screen.getByText('3 files · 450 lines')).toBeInTheDocument()
    const path = within(screen.getByRole('navigation', { name: 'Folder' }))
    expect(path.getAllByRole('button').map((b) => b.textContent)).toEqual(['All', 'src'])
    press('Backspace')
    expect(screen.getByText('4 files · 470 lines')).toBeInTheDocument()
  })

  it('names the folders with colors, and the steps in words', async () => {
    view()
    await map()
    const legend = within(screen.getByLabelText('Legend'))
    expect(legend.getByText('src')).toBeInTheDocument()
    expect(legend.getByText('docs')).toBeInTheDocument()
    expect(legend.getByText('None')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Age' }))
    expect(legend.getByText('Last 7 days')).toBeInTheDocument()
    expect(screen.queryByRole('combobox', { name: 'Period' })).toBeNull()
  })

  it('steps through the releases, oldest first', async () => {
    view()
    await map()
    const slider = screen.getByRole('slider', { name: 'Release' })
    expect(slider).toHaveAttribute('aria-valuetext', 'Now')
    vi.mocked(ExploreService.Files).mockResolvedValue([file('src/old.ts', 10)])
    fireEvent.change(slider, { target: { value: '0' } })
    expect(slider).toHaveAttribute('aria-valuetext', 'v1')
    await waitFor(() => expect(ExploreService.Files).toHaveBeenLastCalledWith('/repo', 'refs/tags/v1'))
    expect(await screen.findByText('1 file · 10 lines')).toBeInTheDocument()
    await focusMap()
    expect(tile()).toBe('src/old.ts, 10 lines at v1')
    act(() => (document.activeElement as HTMLElement).blur())
    expect(screen.getByText('Brightness is off at a release.')).toBeInTheDocument()
    fireEvent.change(slider, { target: { value: '2' } })
    expect(await screen.findByText('4 files · 470 lines')).toBeInTheDocument()
  })

  it('says how far back commits are counted', async () => {
    vi.mocked(ExploreService.Activity).mockResolvedValue({ files: {}, limit: 20000 })
    view()
    expect(await screen.findByText('Commits are counted over the last 20,000.')).toBeInTheDocument()
  })
})

describe('changes', () => {
  const changes = async (files: MapChangeInfo[], over = {}) => {
    vi.mocked(ExploreService.Changes).mockResolvedValue(new MapChangesInfo({ predicted: true, files, ...over }))
    view()
    await map()
    await userEvent.click(screen.getByRole('button', { name: 'Changes' }))
  }
  const compare = () => screen.getByRole('combobox', { name: 'Compare' })

  it('says what happened to each file', async () => {
    await changes([
      change('src/a.ts', 'M', { added: 30 }),
      change('src/new.ts', 'A', { conflict: true, added: 20 }),
      change('src/gone.ts', 'D', { added: 0, removed: 9 }),
      change('src/to.ts', 'R', { origPath: 'src/from.ts' }),
    ])
    expect(await screen.findByText('4 files · 65 lines changed')).toBeInTheDocument()
    await focusMap()
    const said = [tile()]
    for (let i = 0; i < 3; i++) {
      press('PageDown')
      said.push(tile())
    }
    press('PageDown')
    expect(tile()).toBe(said[3])
    expect(new Set(said)).toEqual(
      new Set([
        'src/a.ts, modified, +30 -1',
        'src/new.ts, added, +20 -1, predicted to conflict',
        'src/gone.ts, deleted, +0 -9',
        'src/to.ts, renamed from src/from.ts, +3 -1',
      ]),
    )
    press('Home')
    press('Enter')
    expect(openWorkingFile).toHaveBeenCalledWith('src/a.ts')
  })

  it('compares a branch with its base, and opens it in Read through', async () => {
    await changes([change('src/a.ts', 'M')], { target: 'origin/main' })
    await userEvent.selectOptions(compare(), 'branch:feat')
    await waitFor(() =>
      expect(ExploreService.Changes).toHaveBeenLastCalledWith('/repo', { kind: 'branch', ref: 'feat' }),
    )
    expect(await screen.findByText('against origin/main')).toBeInTheDocument()
    await focusMap()
    press('Enter')
    expect(readThrough).toHaveBeenCalledWith({
      ref: 'refs/heads/feat',
      uncommitted: false,
      at: { path: 'src/a.ts', line: 0 },
    })
  })

  it('compares with a release, and opens the difference', async () => {
    vi.mocked(CompareService.GetChangedFiles).mockResolvedValue([change('src/a.ts', 'M')])
    vi.mocked(DiffService.GetRefDiff).mockResolvedValue(new FileDiff({ hunks: [] }))
    vi.mocked(DiffService.GetDependencySummary).mockResolvedValue({ available: false, changes: [] })
    vi.mocked(BranchService.ListRemote).mockResolvedValue([])
    await changes([change('src/a.ts', 'M')])
    expect(
      within(compare())
        .getAllByRole('option')
        .map((o) => o.textContent),
    ).toEqual(['Uncommitted changes', 'feat (current)', 'main', 'v2', 'v1', 'Choose…'])
    await userEvent.selectOptions(compare(), 'release:v1')
    await focusMap()
    press('Enter')
    await waitFor(() => expect(CompareService.GetChangedFiles).toHaveBeenCalledWith('/repo', 'v1', 'HEAD'))
  })

  it('says when nothing has changed, or there is nothing to compare with', async () => {
    await changes([])
    expect(await screen.findByText('Nothing has changed.')).toBeInTheDocument()
    await userEvent.selectOptions(compare(), 'branch:feat')
    expect(await screen.findByText('This repository has no default branch to compare with.')).toBeInTheDocument()
  })

  it('keeps the folder that fills the map', async () => {
    vi.mocked(ExploreService.Changes).mockResolvedValue(
      new MapChangesInfo({ predicted: true, files: [change('src/a.ts', 'M'), change('docs/x.md', 'M')] }),
    )
    view()
    await focusMap()
    press('Enter', { shiftKey: true })
    await userEvent.click(screen.getByRole('button', { name: 'Changes' }))
    expect(await screen.findByText('1 file · 4 lines changed')).toBeInTheDocument()
  })
})

describe('saving', () => {
  const save = async () => {
    vi.mocked(PlatformService.SaveImage).mockResolvedValue(true)
    await userEvent.click(screen.getByRole('button', { name: 'Save as image' }))
    await userEvent.click(screen.getByRole('button', { name: 'SVG' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save…' }))
    const [data, filename] = vi.mocked(PlatformService.SaveImage).mock.lastCall!
    return { svg: new TextDecoder().decode(decodeBase64(data)), filename }
  }

  it('saves the map on screen, saying what colors it', async () => {
    view()
    await map()
    const { svg, filename } = await save()
    expect(filename).toMatch(/^repo-code-map-.*\.svg$/)
    expect(svg).toContain('>repo</text>')
    expect(svg).toContain('>Code map, commits in 90 days</text>')
    expect(svg).toContain('>a.ts</text>')
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('names the comparison', async () => {
    vi.mocked(ExploreService.Changes).mockResolvedValue(
      new MapChangesInfo({ predicted: true, target: 'main', files: [change('src/a.ts', 'M')] }),
    )
    view()
    await map()
    await userEvent.click(screen.getByRole('button', { name: 'Changes' }))
    await screen.findByText('Modified')
    await waitFor(() => expect(screen.getByRole('button', { name: 'Save as image' })).toBeEnabled())
    const { svg, filename } = await save()
    expect(filename).toMatch(/^repo-changes-/)
    expect(svg).toContain('>Changes, uncommitted</text>')
    expect(svg).toContain('>Predicted to conflict</text>')
  })

  it('names what is on the map while another comparison loads', async () => {
    vi.mocked(ExploreService.Changes).mockResolvedValueOnce(
      new MapChangesInfo({ predicted: true, files: [change('src/a.ts', 'M')] }),
    )
    view()
    await map()
    await userEvent.click(screen.getByRole('button', { name: 'Changes' }))
    await waitFor(() => expect(screen.getByRole('button', { name: 'Save as image' })).toBeEnabled())
    vi.mocked(ExploreService.Changes).mockReturnValue(new Promise(() => {}) as never)
    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Compare' }), 'branch:feat')
    expect((await save()).svg).toContain('>Changes, uncommitted</text>')
  })

  it('has nothing to save without a map', async () => {
    vi.mocked(ExploreService.Files).mockResolvedValue([])
    view()
    await waitFor(() => expect(ExploreService.Files).toHaveBeenCalled())
    expect(screen.getByRole('button', { name: 'Save as image' })).toBeDisabled()
  })
})

describe('the structure', () => {
  it('shares the folder and what is compared with the other modes', async () => {
    vi.mocked(ExploreService.Structure).mockResolvedValue(
      new StructureInfo({ path: 'src', crumbs: [{ path: '', label: '' }], languages: ['Go'] }),
    )
    view()
    await map()
    await userEvent.click(screen.getByRole('button', { name: 'Structure' }))
    await screen.findByText(/Links are read from Go/)
    expect(ExploreService.Structure).toHaveBeenLastCalledWith('/repo', '', null)
    expect(screen.queryByRole('checkbox', { name: 'Tests' })).toBeNull()
    const compare = screen.getByRole('combobox', { name: 'Compare' })
    await userEvent.selectOptions(compare, 'release:v1')
    expect(ExploreService.Structure).toHaveBeenLastCalledWith('/repo', '', { kind: 'release', ref: 'v1' })
    await userEvent.click(screen.getByRole('button', { name: 'Changes' }))
    expect(ExploreService.Changes).toHaveBeenLastCalledWith('/repo', { kind: 'release', ref: 'v1' })
  })

  it('compares with any other point', async () => {
    vi.mocked(ExploreService.Structure).mockResolvedValue(new StructureInfo({ languages: ['Go'] }))
    view()
    await map()
    await userEvent.click(screen.getByRole('button', { name: 'Structure' }))
    await userEvent.selectOptions(await screen.findByRole('combobox', { name: 'Compare' }), 'other')
    await userEvent.type(await screen.findByRole('textbox', { name: 'Commit, branch or tag' }), 'abc123')
    await userEvent.click(screen.getByRole('button', { name: 'Compare' }))
    await waitFor(() =>
      expect(ExploreService.Structure).toHaveBeenLastCalledWith('/repo', '', { kind: 'commit', ref: 'abc123' }),
    )
    expect(screen.getByRole('combobox', { name: 'Compare' })).toHaveDisplayValue('abc123')
  })
})
