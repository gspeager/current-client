import { ancestorShas } from './commitAncestry'
import type { CommitInfo } from '@current-client-bindings/app'

function commit(sha: string, parentShas: string[]): CommitInfo {
  return {
    sha,
    parentShas,
    authorName: 'Test',
    authorEmail: 'test@example.com',
    date: '2024-01-01T00:00:00Z',
    subject: sha,
    body: '',
    signature: 'none',
    coAuthors: [],
  }
}

describe('ancestorShas', () => {
  it('includes the starting commit and every reachable ancestor', () => {
    const commits = [commit('c', ['b']), commit('b', ['a']), commit('a', [])]
    expect(ancestorShas(commits, 'c')).toEqual(new Set(['c', 'b', 'a']))
  })

  it('excludes commits on an unrelated branch', () => {
    const commits = [commit('main2', ['base']), commit('feature', ['base']), commit('base', [])]
    expect(ancestorShas(commits, 'feature')).toEqual(new Set(['feature', 'base']))
  })

  it('follows both parents of a merge commit', () => {
    const commits = [
      commit('merge', ['left', 'right']),
      commit('left', ['base']),
      commit('right', ['base']),
      commit('base', []),
    ]
    expect(ancestorShas(commits, 'merge')).toEqual(new Set(['merge', 'left', 'right', 'base']))
  })

  it('stops at whatever is loaded, without erroring on an unknown parent', () => {
    const commits = [commit('tip', ['not-loaded'])]
    expect(ancestorShas(commits, 'tip')).toEqual(new Set(['tip', 'not-loaded']))
  })
})
