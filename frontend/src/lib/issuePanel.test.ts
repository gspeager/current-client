import { issueDraftFromLine } from './issuePanel'

describe('issueDraftFromLine', () => {
  it.each([
    ['// TODO: drop the shim', 'drop the shim'],
    ['# FIXME(ada) handle 404', 'handle 404'],
    ['/* XXX - flaky on Windows */', 'flaky on Windows'],
    ['<!-- TODO add alt text -->', 'add alt text'],
    ['return value', ''],
  ])('titles %s as %s', (content, title) => {
    expect(issueDraftFromLine('a.ts', 3, content).title).toBe(title)
  })

  it('says where the line is', () => {
    expect(issueDraftFromLine('src/a.ts', 3, '  x()  ').description).toBe('`src/a.ts` line 3\n\n```\nx()\n```')
  })
})
