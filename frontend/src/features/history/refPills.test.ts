import { RefInfo } from '@current-client-bindings/app'
import { pairUpstreams } from './refPills'

const ref = (kind: string, name: string) => new RefInfo({ kind, name, sha: 'abc' })

describe('pairUpstreams', () => {
  it('folds a branch and its upstream into one pill, keeping the rest in order', () => {
    const refs = [
      ref('tag', 'v1'),
      ref('remote', 'origin/feature'),
      ref('branch', 'feature'),
      ref('remote', 'origin/other'),
    ]
    const pills = pairUpstreams(refs, { feature: 'origin/feature' })
    expect(pills.map((p) => [p.ref.name, p.upstream?.name])).toEqual([
      ['feature', 'origin/feature'],
      ['v1', undefined],
      ['origin/other', undefined],
    ])
  })

  it('keeps separate pills when the upstream is elsewhere or unknown', () => {
    const refs = [ref('branch', 'feature'), ref('remote', 'origin/feature')]
    expect(pairUpstreams(refs, { feature: 'origin/main' }).map((p) => p.ref.name)).toEqual([
      'feature',
      'origin/feature',
    ])
    expect(pairUpstreams(refs).map((p) => p.ref.name)).toEqual(['feature', 'origin/feature'])
  })
})
