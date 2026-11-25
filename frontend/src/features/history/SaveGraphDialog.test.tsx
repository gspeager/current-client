import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { PlatformService } from '@current-client-bindings/app'
import { decodeBase64 } from '../../lib/base64'
import SaveGraphDialog from './SaveGraphDialog'

vi.mock('../../lib/pictureFonts', () => ({ PICTURE_FONTS: [] }))

const row = (_: unknown, i: number) => ({
  commit: {
    sha: String(i).padStart(40, '0'),
    parentShas: [],
    authorName: 'Ada Lovelace',
    authorEmail: 'ada@example.com',
    date: '2026-01-10T12:00:00Z',
    subject: `Commit ${i};`,
    body: '',
    signature: 'none',
    coAuthors: [],
  },
})

beforeEach(() => {
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null)
  vi.mocked(PlatformService.SaveImage).mockResolvedValue(true)
})

const open = (rows: number) =>
  render(
    <SaveGraphDialog
      repoName="repo"
      about="all branches"
      rows={Array.from({ length: rows }, row)}
      shown={[10, 30]}
      options={{
        width: 900,
        rowHeight: 38,
        laneCount: 1,
        upstreams: {},
        laneColor: () => '#fff',
        refColor: () => '#fff',
        color: (color) => color,
      }}
      colors={{ ground: '#000', name: '#fff', muted: '#888' }}
      onClose={() => {}}
    />,
  )

const saved = async () => {
  await userEvent.click(screen.getByRole('button', { name: 'SVG' }))
  await userEvent.click(screen.getByRole('button', { name: 'Save…' }))
  return new TextDecoder().decode(decodeBase64(vi.mocked(PlatformService.SaveImage).mock.lastCall![0]))
}
const commits = (svg: string) => [...svg.matchAll(/Commit (\d+);/g)].map((m) => Number(m[1]))

it('saves the rows on screen', async () => {
  open(200)
  expect(screen.getByRole('option', { name: 'On screen (20 rows)' })).toBeInTheDocument()
  const svg = await saved()
  expect(commits(svg)).toEqual(Array.from({ length: 20 }, (_, i) => i + 10))
  expect(svg).toContain('Commit graph, all branches')
  expect(svg).toContain('Ada Lovelace')
})

it('saves as many from the top as asked, up to 500', async () => {
  open(900)
  await userEvent.selectOptions(screen.getByRole('combobox'), 'From the top')
  const field = screen.getByRole('spinbutton', { name: 'How many, up to 500' })
  expect(field).toHaveValue(100)
  await userEvent.clear(field)
  await userEvent.type(field, '250')
  expect(commits(await saved())).toHaveLength(250)
  await userEvent.clear(field)
  await userEvent.type(field, '9000')
  expect(commits(await saved())).toHaveLength(500)
  expect(field).toHaveValue(500)
})

it('leaves people out', async () => {
  open(50)
  await userEvent.click(screen.getByRole('checkbox', { name: "People's names and initials" }))
  expect(await saved()).not.toContain('Ada')
})
