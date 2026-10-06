import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SubmoduleService, type SubmoduleInfo } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import SubmodulesPanel from './SubmodulesPanel'

const REPO = '/projects/app'

function submodule(path: string, overrides: Partial<SubmoduleInfo> = {}): SubmoduleInfo {
  return { path, commit: 'abc1234def', initialized: true, commitChanged: false, conflicted: false, ...overrides }
}

function renderPanel(submodules: SubmoduleInfo[]) {
  vi.mocked(SubmoduleService.List).mockResolvedValue(submodules)
  const onOpenRepository = vi.fn()
  const onSubmodulesChanged = vi.fn()
  const { container } = render(
    <DialogProvider>
      <SubmodulesPanel repoPath={REPO} onOpenRepository={onOpenRepository} onSubmodulesChanged={onSubmodulesChanged} />
    </DialogProvider>,
  )
  return { container, onOpenRepository, onSubmodulesChanged }
}

describe('SubmodulesPanel', () => {
  it('shows nothing in a repository without submodules', async () => {
    const { container } = renderPanel([])

    await vi.waitFor(() => expect(SubmoduleService.List).toHaveBeenCalled())
    expect(container).toBeEmptyDOMElement()
  })

  it('lists submodules with their state, updates one and opens one', async () => {
    vi.mocked(SubmoduleService.Update).mockResolvedValue()
    const { onOpenRepository, onSubmodulesChanged } = renderPanel([
      submodule('vendor/lib'),
      submodule('vendor/new', { initialized: false }),
      submodule('tools', { commitChanged: true }),
    ])

    expect(await screen.findByText('not initialised')).toBeInTheDocument()
    expect(screen.getByText('different commit')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Open submodule vendor/new' })).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Update submodule vendor/new' }))
    expect(SubmoduleService.Update).toHaveBeenCalledWith(REPO, ['vendor/new'], null)
    await vi.waitFor(() => expect(onSubmodulesChanged).toHaveBeenCalled())

    await userEvent.click(screen.getByRole('button', { name: 'Open submodule vendor/lib' }))
    expect(onOpenRepository).toHaveBeenCalledWith('/projects/app/vendor/lib')
  })

  it('updates every submodule', async () => {
    vi.mocked(SubmoduleService.Update).mockResolvedValue()
    renderPanel([submodule('vendor/lib')])

    await userEvent.click(await screen.findByRole('button', { name: 'Update all' }))

    expect(SubmoduleService.Update).toHaveBeenCalledWith(REPO, [], null)
  })
})
