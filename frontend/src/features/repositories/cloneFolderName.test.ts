import { describe, expect, it } from 'vitest'
import { cloneFolderName } from './cloneFolderName'

describe('cloneFolderName', () => {
  it.each([
    ['https://github.com/gspeager/current-client.git', 'current-client'],
    ['https://github.com/gspeager/current-client', 'current-client'],
    ['https://github.com/gspeager/current-client/', 'current-client'],
    ['git@github.com:gspeager/current-client.git', 'current-client'],
    ['git@host:repo.git', 'repo'],
    ['ssh://git@host:2222/team/app.git', 'app'],
    ['/srv/git/project/.git', 'project'],
    ['C:\\repos\\tool.git', 'tool'],
    ['backup.bundle', 'backup'],
    ['  https://example.com/a/b.git  ', 'b'],
    ['', ''],
  ])('%j → %j', (url, expected) => {
    expect(cloneFolderName(url)).toBe(expected)
  })
})
