// A picture that can be drawn on a canvas or written as an SVG.
export interface Picture {
  width: number
  height: number
  draw: (painter: Painter) => void
}

type Face = 'data' | 'ui'

export interface TextStyle {
  color: string
  face?: Face
  size?: number
  weight?: 400 | 600
  align?: 'left' | 'right' | 'center'
}

interface Stroke {
  fill?: string
  stroke?: string
  width?: number
  dash?: number
}

export interface Painter {
  rect: (x: number, y: number, width: number, height: number, fill: string, radius?: number) => void
  // An outline drawn inside the rectangle's edge.
  frame: (x: number, y: number, width: number, height: number, color: string, line: number) => void
  line: (x0: number, y0: number, x1: number, y1: number, color: string, width?: number) => void
  path: (d: string, style: Stroke) => void
  circle: (x: number, y: number, radius: number, style: Stroke) => void
  text: (x: number, baseline: number, text: string, style: TextStyle) => void
  // Runs draw with the origin moved to x, y and everything drawn at opacity.
  within: (x: number, y: number, opacity: number, draw: () => void) => void
}

export const TEXT_SIZE = 12
// The data face's character width, in ems. It's monospaced except for what it lacks:
// Chinese, Japanese, Korean and emoji, which are measured.
const CHARACTER = 0.6
const WIDE = /[\u2e80-\uffff]/
// A rough character width, in ems, for the UI face or wide text when no canvas can measure.
const ABOUT = 0.52

const FACES: Record<Face, { name: string; stack: string }> = {
  data: { name: 'JetBrains Mono', stack: "'JetBrains Mono', monospace" },
  ui: { name: 'Geist Sans', stack: "'Geist Sans', sans-serif" },
}

const fontOf = ({ face = 'data', size = TEXT_SIZE, weight = 400 }: TextStyle) =>
  `${weight} ${size}px ${FACES[face].stack}`

let measuring: CanvasRenderingContext2D | null | undefined

export function textWidth(text: string, style: Omit<TextStyle, 'color'>): number {
  const size = style.size ?? TEXT_SIZE
  if ((style.face ?? 'data') === 'data' && !WIDE.test(text)) return text.length * size * CHARACTER
  if (measuring === undefined) measuring = document.createElement('canvas').getContext('2d')
  if (!measuring) return text.length * size * ABOUT
  measuring.font = fontOf({ ...style, color: '' })
  return measuring.measureText(text).width
}

// Half of a surrogate pair, left at the end or the start of a cut.
const HALF_AFTER = /[\ud800-\udbff]$/
const HALF_BEFORE = /^[\udc00-\udfff]/

// As much of the text as fits, with an ellipsis where it's cut. At least this many
// of its characters, or nothing.
export function cut(
  text: string,
  room: number,
  style: Omit<TextStyle, 'color'>,
  end: 'left' | 'right',
  least = 1,
): string {
  if (textWidth(text, style) <= room) return text
  let low = 0
  let high = text.length
  const part = (length: number) =>
    end === 'right'
      ? `${text.slice(0, length).replace(HALF_AFTER, '').trimEnd()}…`
      : `…${text.slice(text.length - length).replace(HALF_BEFORE, '')}`
  while (low < high) {
    const middle = Math.ceil((low + high) / 2)
    if (textWidth(part(middle), style) <= room) low = middle
    else high = middle - 1
  }
  return low < least ? '' : part(low)
}

export function canvasPainter(context: CanvasRenderingContext2D): Painter {
  // Setting a font takes longer than drawing most text.
  let font = ''
  const trace = ({ fill, stroke, width = 1, dash }: Stroke, path?: Path2D) => {
    if (fill) {
      context.fillStyle = fill
      if (path) context.fill(path)
      else context.fill()
    }
    if (stroke) {
      context.strokeStyle = stroke
      context.lineWidth = width
      context.setLineDash(dash ? [dash, dash] : [])
      if (path) context.stroke(path)
      else context.stroke()
      context.setLineDash([])
    }
  }
  return {
    rect: (x, y, width, height, fill, radius = 0) => {
      context.fillStyle = fill
      if (radius === 0) {
        context.fillRect(x, y, width, height)
        return
      }
      context.beginPath()
      context.roundRect(x, y, width, height, radius)
      context.fill()
    },
    frame: (x, y, width, height, color, line) => {
      context.strokeStyle = color
      context.lineWidth = line
      context.strokeRect(x + line / 2, y + line / 2, Math.max(0, width - line), Math.max(0, height - line))
    },
    line: (x0, y0, x1, y1, color, width = 1) => {
      context.beginPath()
      context.moveTo(x0, y0)
      context.lineTo(x1, y1)
      trace({ stroke: color, width })
    },
    path: (d, style) => trace(style, new Path2D(d)),
    circle: (x, y, radius, style) => {
      context.beginPath()
      context.arc(x, y, radius, 0, 2 * Math.PI)
      trace(style)
    },
    text: (x, baseline, text, style) => {
      const wanted = fontOf(style)
      if (wanted !== font) context.font = font = wanted
      context.fillStyle = style.color
      context.textAlign = style.align ?? 'left'
      context.fillText(text, x, baseline)
    },
    within: (x, y, opacity, draw) => {
      context.save()
      context.translate(x, y)
      context.globalAlpha *= opacity
      draw()
      context.restore()
      font = ''
    },
  }
}

// XML doesn't allow these at all.
// eslint-disable-next-line no-control-regex
const NEVER_IN_XML = /[\u0000-\u0008\u000b\u000c\u000e-\u001f\ufffe\uffff]/g

const escaped = (text: string) => text.replace(NEVER_IN_XML, '').replace(/[&<>"]/g, (c) => `&#${c.charCodeAt(0)};`)

const n = (value: number) => String(Math.round(value * 100) / 100)

function svgPainter(out: string[]): Painter {
  const strokes = ({ fill = 'none', stroke, width = 1, dash }: Stroke) =>
    `fill="${escaped(fill)}"${stroke ? ` stroke="${escaped(stroke)}" stroke-width="${n(width)}"` : ''}${dash ? ` stroke-dasharray="${dash}"` : ''}`
  return {
    rect: (x, y, width, height, fill, radius = 0) =>
      out.push(
        `<rect x="${n(x)}" y="${n(y)}" width="${n(width)}" height="${n(height)}"${radius ? ` rx="${n(radius)}"` : ''} fill="${escaped(fill)}"/>`,
      ),
    frame: (x, y, width, height, color, line) =>
      out.push(
        `<rect x="${n(x + line / 2)}" y="${n(y + line / 2)}" width="${n(Math.max(0, width - line))}" height="${n(Math.max(0, height - line))}" ${strokes({ stroke: color, width: line })}/>`,
      ),
    line: (x0, y0, x1, y1, color, width = 1) =>
      out.push(`<line x1="${n(x0)}" y1="${n(y0)}" x2="${n(x1)}" y2="${n(y1)}" ${strokes({ stroke: color, width })}/>`),
    path: (d, style) => out.push(`<path d="${d}" ${strokes(style)}/>`),
    circle: (x, y, radius, style) => out.push(`<circle cx="${n(x)}" cy="${n(y)}" r="${n(radius)}" ${strokes(style)}/>`),
    text: (x, baseline, text, { color, face = 'data', size = TEXT_SIZE, weight = 400, align = 'left' }) =>
      out.push(
        `<text x="${n(x)}" y="${n(baseline)}" class="${face}"${size === TEXT_SIZE ? '' : ` font-size="${size}"`}${weight === 400 ? '' : ` font-weight="${weight}"`}${align === 'left' ? '' : ` text-anchor="${align === 'right' ? 'end' : 'middle'}"`} fill="${escaped(color)}" xml:space="preserve">${escaped(text)}</text>`,
      ),
    within: (x, y, opacity, draw) => {
      out.push(`<g transform="translate(${n(x)} ${n(y)})"${opacity === 1 ? '' : ` opacity="${opacity}"`}>`)
      draw()
      out.push('</g>')
    },
  }
}

export interface PictureFont {
  face: Face
  weight: number
  // Absent for the basic Latin part, which is always embedded.
  needs?: RegExp
  range?: string
  load: () => Promise<string>
}

// The fonts of what the picture says, so it looks the same on any computer.
export async function writeSvg(picture: Picture, fonts: PictureFont[]): Promise<string> {
  const out: string[] = []
  picture.draw(svgPainter(out))
  const body = out.join('\n')
  const faces = await Promise.all(
    fonts
      .filter((font) => !font.needs || font.needs.test(body))
      .map(
        async ({ face, weight, range, load }) =>
          `@font-face{font-family:'${FACES[face].name}';font-weight:${weight};src:url(${await load()}) format('woff2');${range ? `unicode-range:${range};` : ''}}`,
      ),
  )
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${n(picture.width)}" height="${n(picture.height)}" viewBox="0 0 ${n(picture.width)} ${n(picture.height)}" font-size="${TEXT_SIZE}">
<style>${faces.join('')}.data{font-family:${FACES.data.stack}}.ui{font-family:${FACES.ui.stack}}</style>
${body}
</svg>
`
}

// The largest canvas side and area browsers will draw, with room to spare.
const LONGEST_SIDE = 16000
const MOST_PIXELS = 64_000_000

// Twice the size on screen, or as much of that as a file can hold.
export function pictureScale({ width, height }: Pick<Picture, 'width' | 'height'>): number {
  return Math.min(2, LONGEST_SIDE / Math.max(width, height), Math.sqrt(MOST_PIXELS / (width * height)))
}

export async function writePng(picture: Picture): Promise<Blob> {
  const scale = pictureScale(picture)
  const canvas = document.createElement('canvas')
  canvas.width = Math.round(picture.width * scale)
  canvas.height = Math.round(picture.height * scale)
  const context = canvas.getContext('2d')
  if (!context) throw new Error("This picture can't be drawn here.")
  await document.fonts?.ready
  context.scale(scale, scale)
  picture.draw(canvasPainter(context))
  return new Promise((resolve, reject) =>
    canvas.toBlob(
      (blob) => (blob ? resolve(blob) : reject(new Error('This picture is too large to save.'))),
      'image/png',
    ),
  )
}

const HEADING = 40

export interface HeadingColors {
  ground: string
  name: string
  muted: string
}

// The picture under the repository's name, what it shows, and the date.
export function withHeading(
  picture: Picture,
  { title, about, date, colors }: { title: string; about: string; date: string; colors: HeadingColors },
): Picture {
  return {
    width: picture.width,
    height: picture.height + HEADING,
    draw: (painter) => {
      painter.rect(0, 0, picture.width, picture.height + HEADING, colors.ground)
      const name = { color: colors.name, face: 'ui', size: 14, weight: 600 } as const
      const rest = { color: colors.muted, face: 'ui' } as const
      const dated = textWidth(date, rest)
      const shown = cut(title, picture.width - dated - 48, name, 'right')
      painter.text(12, 25, shown, name)
      const after = 12 + textWidth(shown, name) + 12
      const described = cut(about, picture.width - after - dated - 24, rest, 'right', 3)
      if (described) painter.text(after, 25, described, rest)
      painter.text(picture.width - 12, 25, date, { ...rest, align: 'right' })
      painter.within(0, HEADING, 1, () => picture.draw(painter))
    },
  }
}

// A token's value where the picture is drawn, as a saved file has no tokens.
export function colorsAt(element: Element): (color: string) => string {
  const style = getComputedStyle(element)
  return (color) => {
    const token = /^var\((--[\w-]+)\)$/.exec(color)
    return token ? style.getPropertyValue(token[1]).trim() : color
  }
}
