import { ReviewThread, ReviewThreads } from '@current-client-bindings/app'
import { groupThreads } from './useReviewThreads'

describe('groupThreads', () => {
  it('splits threads by path into current and outdated, in line order, with the counts', () => {
    const view = groupThreads(
      new ReviewThreads({
        enabled: true,
        number: 7,
        prefix: '!',
        threads: [
          new ReviewThread({ id: 'b', path: 'app/x.go', line: 40 }),
          new ReviewThread({ id: 'a', path: 'app/x.go', line: 12 }),
          new ReviewThread({ id: 'c', path: 'app/x.go', oldLine: 3, outdated: true }),
          new ReviewThread({ id: 'd', path: '', line: 0 }),
        ],
        unresolved: { 'app/x.go': 2 },
        total: 3,
      }),
    )
    expect(view.byPath['app/x.go']?.map((t) => t.id)).toEqual(['a', 'b'])
    expect(view.outdatedByPath['app/x.go']?.map((t) => t.id)).toEqual(['c'])
    expect(view.counts['app/x.go']).toBe(2)
    expect(view.total).toBe(3)
    expect(view.number).toBe(7)
  })
})
