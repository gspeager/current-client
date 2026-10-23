import { PushWarning } from '@current-client-bindings/app'
import { protectionMessage } from './protection'

const main = new PushWarning({
  host: 'gitlab.com',
  branch: 'main',
  pattern: 'main',
  noPush: true,
  noForcePush: true,
  noDelete: true,
})
const develop = new PushWarning({
  host: 'gitlab.com',
  branch: 'develop',
  pattern: 'develop',
  restrictedPush: true,
  noDelete: true,
})

describe('protectionMessage', () => {
  it('names only what the rule forbids for the action', () => {
    expect(protectionMessage(main, 'push')).toBe(
      'gitlab.com protects main and forbids direct pushes. The push may be rejected.',
    )
    expect(protectionMessage(main, 'force-push')).toBe(
      'gitlab.com protects main and forbids direct pushes and force-pushes. The push may be rejected.',
    )
    expect(protectionMessage(main, 'delete')).toBe(
      'gitlab.com protects main and forbids deleting it. The push may be rejected.',
    )
    expect(protectionMessage(develop, 'force-push')).toBe(
      'gitlab.com protects develop and forbids pushes except by some roles. The push may be rejected.',
    )
    expect(
      protectionMessage(new PushWarning({ host: 'h', branch: 'b', pattern: 'b', noDelete: true }), 'push'),
    ).toBeNull()
  })
})
