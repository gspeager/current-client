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
    expect(loadDiff).toHaveBeenCalledWith(files[1])
  })

  it('shows the empty hint when nothing changed', () => {
    render(<ChangedFilesBrowser files={[]} error={null} emptyHint="Nothing here." loadDiff={vi.fn()} />)

    expect(screen.getByText('Nothing here.')).toBeInTheDocument()
  })
})
