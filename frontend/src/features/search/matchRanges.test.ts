import { matchRanges } from './matchRanges'

const plain = { ignoreCase: false, wholeWord: false, regexp: false }

describe('matchRanges', () => {
  it('finds every occurrence of fixed text, treating special characters literally', () => {
    expect(matchRanges('a.b a.b axb', 'a.b', plain)).toEqual([
      [0, 3],
      [4, 7],
    ])
  })

  it('honours case, whole words and regular expressions', () => {
    expect(matchRanges('Foo foo', 'foo', { ...plain, ignoreCase: true })).toEqual([
      [0, 3],
      [4, 7],
    ])
    expect(matchRanges('food foo', 'foo', { ...plain, wholeWord: true })).toEqual([[5, 8]])
    expect(matchRanges('func Foo()', 'F.o\\(', { ...plain, regexp: true })).toEqual([[5, 9]])
  })

  it('highlights nothing for an empty pattern or one JavaScript cannot read', () => {
    expect(matchRanges('anything', '', plain)).toEqual([])
    expect(matchRanges('a(b', '(', { ...plain, regexp: true })).toEqual([])
  })
})
