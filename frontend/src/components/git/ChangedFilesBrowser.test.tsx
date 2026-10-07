import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ChangedFile, FileDiff } from '@current-client-bindings/app'
import ChangedFilesBrowser from './ChangedFilesBrowser'

const files = [
  new ChangedFile({ status: 'M', path: 'src/app.ts', added: 2, removed: 1 }),
  new ChangedFile({ status: '?', path: 'logo.png', binary: true }),
]

describe('ChangedFilesBrowser', () => {
  it('loads the diff of the file picked', async () => {
    const loadDiff = vi.fn(() => Promise.resolve(new FileDiff({ newPath: 'logo.png', binary: true })))
    render(<ChangedFilesBrowser files={files} error={null} emptyHint="Nothing here." loadDiff={loadDiff} />)

    expect(screen.getByText('Select a file to view its diff.')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: /logo\.png/ }))

    expect(await screen.findByText('Binary file changed.')).toBeInTheDocument()
    expect(loadDiff).toHaveBeenCalledWith(files[1], false)
  })

  it('shows the empty hint when nothing changed', () => {
    render(<ChangedFilesBrowser files={[]} error={null} emptyHint="Nothing here." loadDiff={vi.fn()} />)

    expect(screen.getByText('Nothing here.')).toBeInTheDocument()
  })
})

describe('ChangedFilesBrowser large files', () => {
  it('loads a held-back file when asked', async () => {
    const files = [new ChangedFile({ status: 'M', path: 'big.txt' })]
    const loadDiff = vi.fn((_file: ChangedFile, force: boolean) =>
      Promise.resolve(
        force ? new FileDiff({ newPath: 'big.txt' }) : new FileDiff({ tooLarge: true, sizeBytes: 3 << 20 }),
      ),
    )
    render(<ChangedFilesBrowser files={files} error={null} emptyHint="" loadDiff={loadDiff} />)

    await userEvent.click(screen.getByRole('button', { name: /big\.txt/ }))
    await userEvent.click(await screen.findByRole('button', { name: 'Load anyway' }))

    expect(loadDiff).toHaveBeenLastCalledWith(files[0], true)
  })
})
