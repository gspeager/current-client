import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { act } from 'react'
import { CurrentClientRootContext } from '../../lib/currentClientRoot'
import { useDialogs, type Dialogs } from '../../lib/useDialogs'
import { DialogProvider } from './DialogProvider'

function renderWithDialogs() {
  const captured: { dialogs?: Dialogs } = {}
  function Capture() {
    captured.dialogs = useDialogs()
    return null
  }
  render(
    <DialogProvider>
      <Capture />
    </DialogProvider>,
  )
  return captured.dialogs!
}

const discardHunk = {
  title: 'Discard hunk',
  message: 'Discard this hunk? This cannot be undone.',
  confirmLabel: 'Discard',
  destructive: true,
}

describe('DialogProvider confirm', () => {
  it('resolves true from the action button', async () => {
    const result = renderWithDialogs().confirm(discardHunk)
    expect(await screen.findByRole('alertdialog', { name: 'Discard hunk' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))
    await expect(result).resolves.toBe(true)
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  })

  it('focuses Cancel and resolves false from it', async () => {
    const result = renderWithDialogs().confirm(discardHunk)
    const cancel = await screen.findByRole('button', { name: 'Cancel' })
    expect(cancel).toHaveFocus()
    await userEvent.click(cancel)
    await expect(result).resolves.toBe(false)
  })

  it('resolves false on Escape', async () => {
    const result = renderWithDialogs().confirm(discardHunk)
    await screen.findByRole('alertdialog')
    await userEvent.keyboard('{Escape}')
    await expect(result).resolves.toBe(false)
  })
})

describe('DialogProvider prompt', () => {
  const rename = { title: 'Rename branch', label: 'New branch name', initialValue: 'main', confirmLabel: 'Rename' }

  it('resolves the trimmed value on submit', async () => {
    const result = renderWithDialogs().prompt(rename)
    const input = await screen.findByRole('textbox', { name: 'New branch name' })
    await userEvent.clear(input)
    await userEvent.type(input, '  develop  {Enter}')
    await expect(result).resolves.toBe('develop')
  })

  it('disables submit while the value is empty or unchanged', async () => {
    renderWithDialogs().prompt(rename)
    const input = await screen.findByRole('textbox', { name: 'New branch name' })
    const submit = screen.getByRole('button', { name: 'Rename' })
    expect(submit).toBeDisabled()
    await userEvent.clear(input)
    expect(submit).toBeDisabled()
    await userEvent.type(input, 'develop')
    expect(submit).toBeEnabled()
  })

  it('shows and resolves the transformed value', async () => {
    const result = renderWithDialogs().prompt({
      ...rename,
      initialValue: undefined,
      transform: (v) => v.replace(/ /g, '-'),
    })
    const input = await screen.findByRole('textbox', { name: 'New branch name' })
    await userEvent.type(input, 'my branch')
    expect(screen.getByRole('status')).toHaveTextContent('Will be saved as my-branch')
    await userEvent.type(input, '{Enter}')
    await expect(result).resolves.toBe('my-branch')
  })

  it('disables submit when the transform leaves nothing', async () => {
    renderWithDialogs().prompt({ ...rename, initialValue: undefined, transform: () => '' })
    await userEvent.type(await screen.findByRole('textbox', { name: 'New branch name' }), '...')
    expect(screen.getByRole('button', { name: 'Rename' })).toBeDisabled()
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('resolves null when cancelled', async () => {
    const result = renderWithDialogs().prompt(rename)
    await userEvent.click(await screen.findByRole('button', { name: 'Cancel' }))
    await expect(result).resolves.toBeNull()
  })
})

describe('DialogProvider inside a Current Client root', () => {
  it('renders dialogs into the root element, so they keep its scoped styles', async () => {
    const root = document.body.appendChild(document.createElement('div'))
    const captured: { dialogs?: Dialogs } = {}
    function Capture() {
      captured.dialogs = useDialogs()
      return null
    }
    render(
      <CurrentClientRootContext.Provider value={{ element: root, active: true }}>
        <DialogProvider>
          <Capture />
        </DialogProvider>
      </CurrentClientRootContext.Provider>,
    )

    void act(() => void captured.dialogs!.confirm(discardHunk))
    expect(root).toContainElement(await screen.findByRole('alertdialog', { name: 'Discard hunk' }))
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))

    void act(() => void captured.dialogs!.prompt({ title: 'Rename branch', label: 'New name', confirmLabel: 'Rename' }))
    expect(root).toContainElement(await screen.findByRole('dialog', { name: 'Rename branch' }))
    root.remove()
  })
})
