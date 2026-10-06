import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { DiffService, FileContent, FileDiff } from '@current-client-bindings/app'
import DiffViewer from './DiffViewer'

const REPO = '/repos/app'
const images = { repoPath: REPO, before: { kind: 'commit', rev: 'abc^' }, after: { kind: 'commit', rev: 'abc' } }
const binary = new FileDiff({ binary: true })

function content(data: string) {
  return new FileContent({ found: true, data: btoa(data) })
}

describe('image diffs', () => {
  it('shows both versions side by side, then swipes between them', async () => {
    vi.mocked(DiffService.GetFileContent).mockResolvedValueOnce(content('old')).mockResolvedValueOnce(content('new'))
    render(<DiffViewer diff={binary} path="img/logo.png" viewMode="split" images={images} />)

    const after = await screen.findByRole('img', { name: 'After' })
    expect(after).toHaveAttribute('src', `data:image/png;base64,${btoa('new')}`)
    expect(screen.getByRole('img', { name: 'Before' })).toHaveAttribute('src', `data:image/png;base64,${btoa('old')}`)
    expect(DiffService.GetFileContent).toHaveBeenCalledWith(REPO, 'img/logo.png', images.before)
    expect(DiffService.GetFileContent).toHaveBeenCalledWith(REPO, 'img/logo.png', images.after)

    await userEvent.click(screen.getByRole('button', { name: /Swipe/ }))
    fireEvent.change(screen.getByRole('slider', { name: 'Swipe position' }), { target: { value: '30' } })
    expect(screen.getByRole('img', { name: 'After' })).toHaveStyle({ clipPath: 'inset(0 0 0 30%)' })

    await userEvent.click(screen.getByRole('button', { name: /Onion skin/ }))
    fireEvent.change(screen.getByRole('slider', { name: 'Opacity of the new version' }), { target: { value: '25' } })
    expect(screen.getByRole('img', { name: 'After' })).toHaveStyle({ opacity: '0.25' })
  })

  it('shows an added image on its own', async () => {
    vi.mocked(DiffService.GetFileContent)
      .mockResolvedValueOnce(new FileContent({ found: false }))
      .mockResolvedValueOnce(content('new'))
    render(<DiffViewer diff={binary} path="logo.png" viewMode="split" images={images} />)

    expect(await screen.findByRole('img', { name: 'Added' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Swipe/ })).not.toBeInTheDocument()
  })

  it('says when an image is too large to show', async () => {
    vi.mocked(DiffService.GetFileContent)
      .mockResolvedValueOnce(content('old'))
      .mockResolvedValueOnce(new FileContent({ found: true, tooLarge: true }))
    render(<DiffViewer diff={binary} path="huge.png" viewMode="split" images={images} />)

    expect(await screen.findByText('Too large to show.')).toBeInTheDocument()
    expect(screen.getByRole('img', { name: 'Before' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Swipe/ })).not.toBeInTheDocument()
  })

  it('lets an SVG be read as text too', async () => {
    vi.mocked(DiffService.GetFileContent).mockResolvedValue(content('<svg/>'))
    const svgDiff = new FileDiff({
      hunks: [
        {
          header: '@@ -1 +1 @@',
          raw: '',
          lines: [{ kind: 'add', oldLine: 0, newLine: 1, content: '<svg/>', moved: false }],
        },
      ],
    })
    render(<DiffViewer diff={svgDiff} path="icon.svg" viewMode="unified" images={images} />)

    expect(await screen.findByRole('img', { name: 'After' })).toHaveAttribute(
      'src',
      expect.stringContaining('image/svg+xml'),
    )
    await userEvent.click(screen.getByRole('button', { name: /Text/ }))

    expect(screen.queryByRole('img', { name: 'After' })).not.toBeInTheDocument()
    expect(screen.getByText('<svg/>')).toBeInTheDocument()
  })

  it('keeps the binary message without image sources, or for other files', () => {
    const { unmount } = render(<DiffViewer diff={binary} path="logo.png" viewMode="split" />)
    expect(screen.getByText('Binary file changed.')).toBeInTheDocument()
    unmount()

    render(<DiffViewer diff={binary} path="app.wasm" viewMode="split" images={images} />)
    expect(screen.getByText('Binary file changed.')).toBeInTheDocument()
    expect(DiffService.GetFileContent).not.toHaveBeenCalled()
  })
})
