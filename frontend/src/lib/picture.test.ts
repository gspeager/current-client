import { cut, pictureScale, textWidth, withHeading, writeSvg, type Painter, type Picture } from './picture'

const picture = (draw: (painter: Painter) => void): Picture => ({ width: 200, height: 100, draw })
const parse = (svg: string) => new DOMParser().parseFromString(svg, 'image/svg+xml')

beforeEach(() => {
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null)
})

describe('writeSvg', () => {
  it('keeps the size and weight of text', async () => {
    const svg = parse(
      await writeSvg(
        picture((painter) => {
          painter.text(1, 2, 'HEAD', { color: '#fff', size: 10, weight: 600 })
          painter.text(1, 20, 'a subject', { color: '#fff', face: 'ui' })
        }),
        [],
      ),
    )
    const [head, subject] = svg.querySelectorAll('text')
    expect(head.getAttribute('font-size')).toBe('10')
    expect(head.getAttribute('font-weight')).toBe('600')
    expect(subject.getAttribute('font-size')).toBeNull()
    expect(svg.documentElement.getAttribute('font-size')).toBe('12')
    // A rule would win over the attributes.
    expect(svg.querySelector('style')?.textContent).not.toContain('font-size')
  })

  it('writes any text as text', async () => {
    const said = `<script>alert("&")</script>\u0007 'é' 日本`
    const svg = parse(
      await writeSvg(
        picture((painter) => painter.text(0, 0, said, { color: '"><x/>' })),
        [],
      ),
    )
    expect(svg.querySelector('parsererror, script, x')).toBeNull()
    expect(svg.querySelector('text')?.textContent).toBe(said.replace('\u0007', ''))
  })

  it('holds the fonts its text needs, and no others', async () => {
    const font = (name: string, needs?: RegExp) => ({
      face: 'data' as const,
      weight: 400,
      needs,
      range: needs && 'U+0400-04FF',
      load: vi.fn().mockResolvedValue(`data:font/woff2;base64,${name}`),
    })
    const fonts = [font('LATIN'), font('CYRILLIC', /[Ѐ-ӿ]/)]
    const latin = await writeSvg(
      picture((painter) => painter.text(0, 0, 'main', { color: '#fff' })),
      fonts,
    )
    expect(latin).toContain('base64,LATIN')
    expect(latin).not.toContain('CYRILLIC')
    expect(fonts[1].load).not.toHaveBeenCalled()
    const cyrillic = await writeSvg(
      picture((painter) => painter.text(0, 0, 'ветка', { color: '#fff' })),
      fonts,
    )
    expect(cyrillic).toContain("url(data:font/woff2;base64,CYRILLIC) format('woff2');unicode-range:U+0400-04FF;")
  })

  it('reaches nothing outside the file', async () => {
    const svg = await writeSvg(
      picture((painter) => painter.rect(0, 0, 1, 1, '#fff')),
      [],
    )
    expect(svg.match(/https?:[^"]*/g)).toEqual(['http://www.w3.org/2000/svg'])
  })
})

describe('cut', () => {
  const room = textWidth('12345678', {})

  it('cuts either end, and leaves what fits', () => {
    expect(cut('12345678', room, {}, 'right')).toBe('12345678')
    expect(cut('1234567890', room, {}, 'right')).toBe('1234567…')
    expect(cut('1234567890', room, {}, 'left')).toBe('…4567890')
  })

  it('says nothing when too little fits', () => {
    expect(cut('1234567890', room, {}, 'right', 8)).toBe('')
    expect(cut('1234567890', -5, {}, 'right')).toBe('')
  })
})

describe('pictureScale', () => {
  it('is twice the size on screen, until a file would be too large', () => {
    expect(pictureScale({ width: 1200, height: 800 })).toBe(2)
    const long = pictureScale({ width: 1200, height: 12000 })
    expect(12000 * long).toBeLessThanOrEqual(16000)
    const wide = pictureScale({ width: 7000, height: 7000 })
    expect(wide * wide * 7000 * 7000).toBeLessThanOrEqual(64_000_001)
  })
})

describe('withHeading', () => {
  const heading = { title: 'repo', about: 'Commit graph', date: '2026-02-01' }
  const colors = { ground: '#010203', name: '#fff', muted: '#888' }

  it('names the picture above it', async () => {
    const svg = parse(
      await writeSvg(
        withHeading({ ...picture((painter) => painter.rect(0, 0, 5, 5, '#f00')), width: 600 }, { ...heading, colors }),
        [],
      ),
    )
    expect([...svg.querySelectorAll('text')].map((el) => el.textContent)).toEqual([
      'repo',
      'Commit graph',
      '2026-02-01',
    ])
    expect(svg.documentElement.getAttribute('height')).toBe('140')
    expect(svg.querySelector('rect')?.getAttribute('fill')).toBe('#010203')
    expect(svg.querySelector('g')?.getAttribute('transform')).toBe('translate(0 40)')
  })

  it('keeps the date when there is little room', async () => {
    const svg = parse(
      await writeSvg(
        withHeading(
          picture(() => {}),
          { ...heading, title: 'a-repository-with-a-long-name', colors },
        ),
        [],
      ),
    )
    const said = [...svg.querySelectorAll('text')].map((el) => el.textContent)
    expect(said[said.length - 1]).toBe('2026-02-01')
    expect(said[0]).toMatch(/…$/)
    expect(said).not.toContain('Commit graph')
  })
})
