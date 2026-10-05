export interface SearchFlags {
  ignoreCase: boolean
  wholeWord: boolean
  regexp: boolean
}

// The [start, end) ranges a search matches in one line, to highlight them.
// JavaScript's regular expressions are close to, not the same as, Git's
// extended ones, so a pattern JavaScript can't read just isn't highlighted.
export function matchRanges(text: string, pattern: string, flags: SearchFlags): [number, number][] {
  if (!pattern) return []
  let source = flags.regexp ? pattern : pattern.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  if (flags.wholeWord) source = `\\b(?:${source})\\b`
  let expression: RegExp
  try {
    expression = new RegExp(source, flags.ignoreCase ? 'gi' : 'g')
  } catch {
    return []
  }
  const ranges: [number, number][] = []
  for (const match of text.matchAll(expression)) {
    if (match[0] === '') break
    ranges.push([match.index, match.index + match[0].length])
  }
  return ranges
}
