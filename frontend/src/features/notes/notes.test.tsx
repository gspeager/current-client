import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { NotesService, type NoteInfo } from '@current-client-bindings/app'
import NotesDialogs from './NotesDialogs'
import { NoteMark, NotesContext, useNotes, useNotesState } from './notes'

const note = (overrides: Partial<NoteInfo> = {}): NoteInfo => ({
  kind: 'branch',
  target: 'side',
  subject: '',
  text: 'Check this next time',
  updatedAt: '2026-01-01T00:00:00Z',
  stale: false,
  ...overrides,
})

function Harness({ onOpen = vi.fn() }: { onOpen?: (note: NoteInfo) => void }) {
  const state = useNotesState('/repo', 0)
  return (
    <NotesContext.Provider value={state.context}>
      <NoteMark kind="branch" target="side" />
      <Buttons />
      <NotesDialogs repoPath="/repo" state={state} onOpen={onOpen} />
    </NotesContext.Provider>
  )
}

function Buttons() {
  const notes = useNotes()
  return (
    <>
      <button type="button" onClick={notes.openList}>
        list
      </button>
      <button type="button" onClick={() => notes.edit({ kind: 'tag', target: 'v1' })}>
        add
      </button>
    </>
  )
}

describe('notes', () => {
  it('shows a mark with the note, and edits it', async () => {
    const user = userEvent.setup()
    vi.mocked(NotesService.List).mockResolvedValue([note()])
    vi.mocked(NotesService.Set).mockResolvedValue()
    render(<Harness />)

    await user.click(await screen.findByRole('button', { name: 'Note: Check this next time' }))
    const dialog = await screen.findByRole('dialog', { name: 'Note on side' })
    const box = within(dialog).getByRole('textbox', { name: 'Note' })
    await user.clear(box)
    await user.type(box, 'Edited')
    await user.click(within(dialog).getByRole('button', { name: 'Save' }))

    expect(NotesService.Set).toHaveBeenCalledWith('/repo', 'branch', 'side', '', 'Edited')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(NotesService.List).toHaveBeenCalledTimes(2)
  })

  it('adds a note, with Save off until there is text and no Remove', async () => {
    const user = userEvent.setup()
    vi.mocked(NotesService.List).mockResolvedValue([])
    vi.mocked(NotesService.Set).mockResolvedValue()
    render(<Harness />)

    await user.click(screen.getByRole('button', { name: 'add' }))
    const dialog = await screen.findByRole('dialog', { name: 'Note on v1' })
    expect(within(dialog).getByRole('button', { name: 'Save' })).toBeDisabled()
    expect(within(dialog).queryByRole('button', { name: 'Remove' })).not.toBeInTheDocument()

    await user.type(within(dialog).getByRole('textbox', { name: 'Note' }), 'New')
    await user.click(within(dialog).getByRole('button', { name: 'Save' }))
    expect(NotesService.Set).toHaveBeenCalledWith('/repo', 'tag', 'v1', '', 'New')
  })

  it('removes a note', async () => {
    const user = userEvent.setup()
    vi.mocked(NotesService.List).mockResolvedValue([note()])
    vi.mocked(NotesService.Set).mockResolvedValue()
    render(<Harness />)

    await user.click(await screen.findByRole('button', { name: /^Note:/ }))
    await user.click(screen.getByRole('button', { name: 'Remove' }))
    expect(NotesService.Set).toHaveBeenCalledWith('/repo', 'branch', 'side', '', '')
  })

  it('lists notes with a stale mark, and only opens what still exists', async () => {
    const user = userEvent.setup()
    const onOpen = vi.fn()
    const here = note()
    vi.mocked(NotesService.List).mockResolvedValue([
      here,
      note({ kind: 'commit', target: 'abcdef1234', subject: 'old work', text: 'Rewritten', stale: true }),
    ])
    render(<Harness onOpen={onOpen} />)
    await screen.findByRole('button', { name: /^Note:/ })

    await user.click(screen.getByRole('button', { name: 'list' }))
    const dialog = await screen.findByRole('dialog', { name: 'Notes' })
    expect(within(dialog).getByText('abcdef1 old work')).toBeInTheDocument()
    expect(within(dialog).getByText('Gone')).toBeInTheDocument()
    expect(within(dialog).getAllByRole('button', { name: 'Open' })).toHaveLength(1)

    await user.click(within(dialog).getByRole('button', { name: 'Open' }))
    expect(onOpen).toHaveBeenCalledWith(here)
  })

  it('shows no mark without a note', async () => {
    vi.mocked(NotesService.List).mockResolvedValue([])
    render(<Harness />)
    await vi.waitFor(() => expect(NotesService.List).toHaveBeenCalled())
    expect(screen.queryByRole('button', { name: /^Note:/ })).not.toBeInTheDocument()
  })
})
