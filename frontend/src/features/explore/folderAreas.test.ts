import { folderAreas, slotFinder } from './folderAreas'

const file = (path: string, lines = 10) => ({ path, lines })

describe('folderAreas', () => {
  it('gives the eight largest folders a color and the rest none', () => {
    const areas = folderAreas(Array.from({ length: 10 }, (_, i) => file(`f${i}/a.go`, 100 - i)))
    expect(areas.map((area) => [area.path, area.slot])).toEqual([
      ...Array.from({ length: 8 }, (_, i) => [`f${i}`, i + 1]),
      ['f8', 0],
      ['f9', 0],
    ])
  })

  it('splits a folder that holds most of the code', () => {
    const areas = folderAreas([
      file('app/src/one/a.go', 400),
      file('app/src/two/a.go', 300),
      file('app/src/main.go', 50),
      file('docs/a.md', 20),
      file('README.md', 5),
    ])
    expect(areas.map(({ path, own }) => [path, own])).toEqual([
      ['app/src/one', false],
      ['app/src/two', false],
      ['app/src', true],
      ['docs', false],
      ['', true],
    ])
    const slotOf = slotFinder(areas)
    expect(slotOf('app/src/one/deep/er/b.go')).toBe(1)
    expect(slotOf('app/src/main.go')).toBe(3)
    expect(slotOf('app/src/new/a.go')).toBe(3)
    expect(slotOf('README.md')).toBe(5)
  })

  it('colors the same folders the same for the same code in another order', () => {
    const files = [file('b/x'), file('a/x'), file('c/x', 11)]
    expect(folderAreas(files)).toEqual(folderAreas([...files].reverse()))
    expect(folderAreas(files).map((area) => area.path)).toEqual(['c', 'a', 'b'])
  })

  it('has one area for files with no folders, and none for no files', () => {
    expect(folderAreas([file('a'), file('b')])).toEqual([{ path: '', own: false, lines: 20, slot: 1 }])
    expect(folderAreas([]).map((area) => area.lines)).toEqual([0])
    expect(slotFinder([])('gone/a.go')).toBe(0)
  })
})
