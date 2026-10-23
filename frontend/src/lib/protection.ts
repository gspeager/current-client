import { RemoteService, type PushWarning } from '@current-client-bindings/app'

export type RemoteAction = 'push' | 'force-push' | 'delete'

// What the host forbids on the branch, for the confirmation before the
// action; null when the rule doesn't touch this action. The host still
// enforces: the action is allowed after the warning.
export function protectionMessage(warning: PushWarning, action: RemoteAction): string | null {
  const forbids: string[] = []
  if (action === 'delete') {
    if (warning.noDelete) forbids.push('deleting it')
  } else {
    if (warning.noPush) forbids.push('direct pushes')
    else if (warning.restrictedPush) forbids.push('pushes except by some roles')
    if (action === 'force-push' && warning.noForcePush) forbids.push('force-pushes')
  }
  if (forbids.length === 0) return null
  const rule = warning.pattern === warning.branch ? warning.branch : `${warning.branch} (rule ${warning.pattern})`
  return `${warning.host} protects ${rule} and forbids ${forbids.join(' and ')}. The push may be rejected.`
}

// The cached rules only, never a request; any failure means no warning,
// so the action runs as it would without the host.
export async function protectionFor(
  repoPath: string,
  remoteBranch: string,
  action: RemoteAction,
): Promise<string | null> {
  try {
    const warning = await RemoteService.PushWarning(repoPath, remoteBranch)
    return warning ? protectionMessage(warning, action) : null
  } catch {
    return null
  }
}
