import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HookService, HooksInfo } from '@current-client-bindings/app'
import HooksPanel from './HooksPanel'
import { renderWithDialogs } from '../../test/render'

const REPO = '/repo'

function show(hooks: Partial<HooksInfo>) {
  vi.mocked(HookService.Hooks).mockResolvedValue(new HooksInfo({ dir: '/repo/.git/hooks', worktrees: 1, ...hooks }))
  vi.mocked(HookService.SetHookOn).mockResolvedValue(undefined)
  renderWithDialogs(<HooksPanel repoPath={REPO} onClose={() => {}} />)
}

it('lists the hooks and turns one off', async () => {
  const preCommit = { event: 'pre-commit', name: 'pre-commit', path: '/repo/.git/hooks/pre-commit', on: true }
  show({
    hooks: [
      preCommit as never,
      { event: 'pre-push', name: 'lint', command: 'make lint', scope: 'global', on: true } as never,
    ],
  })
  const list = within(await screen.findByRole('list', { name: 'Hooks' }))
  expect(list.getByText('lint · global config · make lint')).toBeInTheDocument()
  await userEvent.click(list.getByRole('checkbox', { name: 'pre-commit on' }))
  expect(HookService.SetHookOn).toHaveBeenCalledWith(REPO, expect.objectContaining(preCommit), false)
  expect(HookService.Hooks).toHaveBeenCalledTimes(2)
})

it('asks first in a tracked folder, and only names a hook a tool writes', async () => {
  show({
    dir: '/repo/.githooks',
    scope: 'local',
    tracked: true,
    hooks: [
      { event: 'pre-commit', name: 'pre-commit', path: '/repo/.githooks/pre-commit', on: true } as never,
      {
        event: 'commit-msg',
        name: 'commit-msg',
        path: '/repo/.githooks/commit-msg',
        on: true,
        manager: 'Husky',
      } as never,
    ],
  })
  expect(await screen.findByText(/Tracked in the repository/)).toBeInTheDocument()
  expect(screen.getByText('Written by Husky')).toBeInTheDocument()
  expect(screen.queryByRole('checkbox', { name: 'commit-msg on' })).toBeNull()

  await userEvent.click(screen.getByRole('checkbox', { name: 'pre-commit on' }))
  const dialog = await screen.findByRole('alertdialog', { name: 'Turn off pre-commit' })
  expect(dialog).toHaveTextContent('.githooks is tracked in the repository, so this is a change to commit.')
  expect(HookService.SetHookOn).not.toHaveBeenCalled()
  await userEvent.click(within(dialog).getByRole('button', { name: 'Turn off' }))
  expect(HookService.SetHookOn).toHaveBeenCalledTimes(1)
})

it('says when hooks are shared, and when Git is too old for config hooks', async () => {
  show({ dir: '/Users/me/.githooks', scope: 'global', worktrees: 3, configUnsupported: true, hooks: [] })
  expect(
    await screen.findByText('Set by core.hooksPath in the global config, so every repository uses it.'),
  ).toBeInTheDocument()
  expect(screen.getByText('Shared by 3 worktrees.')).toBeInTheDocument()
  expect(screen.getByText(/need Git 2.54 or newer/)).toBeInTheDocument()
  expect(screen.getByText('No hooks. Git runs none in this repository.')).toBeInTheDocument()
})
