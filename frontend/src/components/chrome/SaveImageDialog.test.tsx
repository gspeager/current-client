import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { PlatformService } from '@current-client-bindings/app'
import { decodeBase64 } from '../../lib/base64'
import { localDate } from '../../lib/relativeTime'
import { writePng } from '../../lib/picture'
import SaveImageDialog from './SaveImageDialog'

vi.mock('../../lib/pictureFonts', () => ({ PICTURE_FONTS: [] }))
vi.mock('../../lib/picture', async (original) => ({ ...(await original()), writePng: vi.fn() }))

const picture = () => ({
  width: 10,
  height: 10,
  draw: (painter: { text: (...args: never[]) => void }) =>
    painter.text(...([1, 2, 'café', { color: '#fff' }] as never[])),
})
const onClose = vi.fn()
const open = (name = 'my repo: graph') => render(<SaveImageDialog name={name} picture={picture} onClose={onClose} />)
const save = () => userEvent.click(screen.getByRole('button', { name: 'Save…' }))

it('saves a PNG, named for what it shows and the day', async () => {
  vi.mocked(writePng).mockResolvedValue({
    arrayBuffer: () => Promise.resolve(new Uint8Array([137, 80, 255]).buffer),
  } as Blob)
  vi.mocked(PlatformService.SaveImage).mockResolvedValue(true)
  open()
  await save()
  expect(PlatformService.SaveImage).toHaveBeenCalledWith(
    btoa(String.fromCharCode(137, 80, 255)),
    `my-repo-graph-${localDate()}.png`,
    'png',
  )
  expect(onClose).toHaveBeenCalled()
})

it('saves an SVG', async () => {
  vi.mocked(PlatformService.SaveImage).mockResolvedValue(true)
  open()
  await userEvent.click(screen.getByRole('button', { name: 'SVG' }))
  await save()
  const [data, filename, kind] = vi.mocked(PlatformService.SaveImage).mock.lastCall!
  expect(new TextDecoder().decode(decodeBase64(data))).toContain('café</text>')
  expect([filename, kind]).toEqual([`my-repo-graph-${localDate()}.svg`, 'svg'])
})

it('stays open when nothing was saved', async () => {
  vi.mocked(writePng).mockRejectedValue(new Error('This picture is too large to save.'))
  open()
  await save()
  expect(await screen.findByText('This picture is too large to save.')).toBeInTheDocument()
  expect(onClose).not.toHaveBeenCalled()

  await userEvent.click(screen.getByRole('button', { name: 'SVG' }))
  vi.mocked(PlatformService.SaveImage).mockResolvedValue(false)
  await save()
  await waitFor(() => expect(screen.getByRole('button', { name: 'Save…' })).toBeEnabled())
  expect(screen.queryByText('This picture is too large to save.')).toBeNull()
  expect(onClose).not.toHaveBeenCalled()
})
