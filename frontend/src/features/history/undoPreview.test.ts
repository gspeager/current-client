import type { UndoPlanInfo } from '@current-client-bindings/app'
import { ResetMode } from '@current-client-bindings/core/git'
import { undoConfirmOptions } from './undoPreview'

function plan(overrides: Partial<UndoPlanInfo>): UndoPlanInfo {
  return {
    operation: 'commit',
    detail: 'add login',
    branch: 'main',
    from: 'b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1',
    to: 'a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0',
    switchTo: '',
    mode: ResetMode.ResetSoft,
    removed: 1,
    restored: 0,
    pushed: false,
    ...overrides,
  }
}

describe('undoConfirmOptions', () => {
  it('describes undoing a commit', () => {
    expect(undoConfirmOptions(plan({}))).toEqual({
      title: 'Undo commit',
      message:
        'main moves from b2c3d4e to a1b2c3d. 1 commit leaves the branch. Their changes stay staged. Same as git reset --soft a1b2c3d.',
      confirmLabel: 'Undo commit',
      destructive: false,
    })
  })

  it('counts commits leaving and coming back for a rebase', () => {
    const { message } = undoConfirmOptions(
      plan({ operation: 'rebase', mode: ResetMode.ResetKeep, removed: 3, restored: 2 }),
    )
    expect(message).toContain('3 commits leave the branch. 2 commits come back. Untracked files are kept.')
    expect(message).toContain('git reset --keep a1b2c3d')
  })

  it('warns and marks destructive when the commits are pushed', () => {
    const options = undoConfirmOptions(
      plan({ operation: 'merge', mode: ResetMode.ResetKeep, pushed: true, branch: '' }),
    )
    expect(options.message).toMatch(/^HEAD moves/)
    expect(options.message).toContain('already pushed')
    expect(options.destructive).toBe(true)
  })

  it('describes switching back', () => {
    expect(
      undoConfirmOptions(plan({ operation: 'switch', branch: 'feature', switchTo: 'main', mode: ResetMode.$zero })),
    ).toEqual({
      title: 'Undo branch switch',
      message: 'Switches from feature back to main. Same as git switch main.',
      confirmLabel: 'Undo branch switch',
    })
  })
})
