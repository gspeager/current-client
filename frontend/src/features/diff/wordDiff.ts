import { diffWordsWithSpace } from 'diff'

interface WordSpan {
  text: string
  changed: boolean
}

export function diffLineWords(oldText: string, newText: string): { left: WordSpan[]; right: WordSpan[] } {
  const left: WordSpan[] = []
  const right: WordSpan[] = []
  for (const part of diffWordsWithSpace(oldText, newText)) {
    if (part.added) {
      right.push({ text: part.value, changed: true })
    } else if (part.removed) {
      left.push({ text: part.value, changed: true })
    } else {
      left.push({ text: part.value, changed: false })
      right.push({ text: part.value, changed: false })
    }
  }
  return { left, right }
}
