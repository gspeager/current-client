import type { UndoPlanInfo } from '@current-client-bindings/app'
import type { ConfirmOptions } from '../../lib/useDialogs'

const commits = (n: number, one: string, many: string) => `${n} commit${n === 1 ? ` ${one}` : `s ${many}`}`

export function undoConfirmOptions(plan: UndoPlanInfo): ConfirmOptions {
  const title = `Undo ${plan.operation === 'switch' ? 'branch switch' : plan.operation}`
  if (plan.operation === 'switch') {
    return {
      title,
      message: `Switches from ${plan.branch || 'detached HEAD'} back to ${plan.switchTo}. Same as git switch ${plan.switchTo}.`,
      confirmLabel: title,
    }
  }
  const to = plan.to.slice(0, 7)
  const sentences = [`${plan.branch || 'HEAD'} moves from ${plan.from.slice(0, 7)} to ${to}.`]
  if (plan.removed > 0) sentences.push(`${commits(plan.removed, 'leaves', 'leave')} the branch.`)
  if (plan.restored > 0) sentences.push(`${commits(plan.restored, 'comes', 'come')} back.`)
  sentences.push(plan.mode === 'soft' ? 'Their changes stay staged.' : 'Untracked files are kept.')
  if (plan.pushed) sentences.push('Some of these commits are already pushed; undoing rewrites published history.')
  sentences.push(`Same as git reset --${plan.mode} ${to}.`)
  return { title, message: sentences.join(' '), confirmLabel: title, destructive: plan.pushed }
}
