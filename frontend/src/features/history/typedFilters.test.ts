import { BREAKING_FILTER } from '../../lib/conventionalCommit'
import { typedFilters, typedText, typedValues, withTyped } from './typedFilters'

const now = new Date(2026, 10, 25, 15, 30)

it('reads each key', () => {
  expect(typedFilters('author:maya type:fix since:2026-01-02 until:2026-03-04', now)).toEqual({
    author: 'maya',
    type: 'fix',
    since: '2026-01-02 00:00:00',
    until: '2026-03-04 23:59:59',
  })
  expect(typedFilters('since:today', now).since).toBe('2026-11-25 00:00:00')
  expect(typedFilters('since:7d until:1d', now)).toMatchObject({
    since: '2026-11-18 00:00:00',
    until: '2026-11-24 23:59:59',
  })
  expect(typedFilters('since:"2 weeks ago"', now).since).toBe('2 weeks ago')
  expect(typedFilters('type:breaking', now).type).toBe(BREAKING_FILTER)
  expect(typedFilters('TYPE:Feat', now).type).toBe('feat')
})

it('reads quoted values, with either quote, closed or not yet', () => {
  expect(typedValues('author:"Maya Lin" type:fix').author).toBe('Maya Lin')
  expect(typedValues("author:'Maya Lin'").author).toBe('Maya Lin')
  expect(typedValues('author:"Maya L').author).toBe('Maya L')
})

it('takes an unknown key, an empty one and the rest as text', () => {
  expect(typedText('fix: login  path:src author:maya')).toBe('fix: login path:src')
  expect(typedValues('path:src').author).toBe('')
  expect(typedFilters('author: login', now).author).toBe('')
  expect(typedText('author: login')).toBe('login')
})

it('mixes filters with text, the last of a key winning', () => {
  const box = 'login author:sam crash author:maya'
  expect(typedText(box)).toBe('login crash')
  expect(typedValues(box).author).toBe('maya')
})

it('writes a key into the box, keeping the rest as typed', () => {
  expect(withTyped('login author:sam', 'author', 'Maya Lin')).toBe('login author:"Maya Lin"')
  expect(withTyped('login author:sam since:7d', 'author', '')).toBe('login since:7d')
  expect(withTyped('', 'since', '30d')).toBe('since:30d')
})
