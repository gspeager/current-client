import type { StandupRepo } from '@current-client-bindings/app'
import { standupSince, standupText } from './standup'

const repo = (over: Partial<StandupRepo>): StandupRepo =>
  ({
    path: '/r/app',
    name: 'app',
    commits: [],
    error: '',
    hostEnabled: true,
    fetchedAt: null,
    prefix: '!',
    opened: null,
    merged: null,
    reviewed: null,
    ...over,
  }) as StandupRepo

describe('standup', () => {
  it('starts yesterday at midnight, and the week on Monday', () => {
    const wednesday = new Date(2026, 9, 28, 15, 30)
    expect(standupSince('yesterday', wednesday)).toEqual(new Date(2026, 9, 27))
    expect(standupSince('week', wednesday)).toEqual(new Date(2026, 9, 26))
    expect(standupSince('week', new Date(2026, 10, 1, 9))).toEqual(new Date(2026, 9, 26))
  })

  const repos = [
    repo({
      commits: [{ sha: 'a', subject: 'feat: forms', date: '' }],
      merged: [{ number: 5, title: 'Login' } as never],
      reviewed: [{ number: 3, title: 'Docs', review: 'changes_requested' } as never],
    }),
    repo({ name: 'quiet' }),
  ]

  it('writes Markdown, leaving out a repository with nothing', () => {
    expect(standupText(repos, 'markdown')).toBe(
      '### app\n\n**Commits**\n\n- feat: forms\n\n**Merged**\n\n- !5 Login\n\n**Reviewed**\n\n- !3 Docs (changes requested)',
    )
  })

  it('writes plain text', () => {
    expect(standupText(repos, 'plain')).toBe(
      'app\n  Commits\n  - feat: forms\n  Merged\n  - !5 Login\n  Reviewed\n  - !3 Docs (changes requested)',
    )
  })
})
