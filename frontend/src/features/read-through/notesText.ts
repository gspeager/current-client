import type { ReadThroughNote } from '@current-client-bindings/app'

// Each note as a list item naming its file and line, which reads as it is and
// as Markdown.
export function notesText(notes: ReadThroughNote[]): string {
  return notes
    .map(({ path, line, oldLine, outdated, text }) => {
      const at = line || oldLine
      const about = outdated ? ' (line changed since)' : oldLine ? ' (old version)' : ''
      return `- \`${path}${at ? `:${at}` : ''}\`${about} ${text.replace(/\n/g, '\n  ')}`
    })
    .join('\n')
}
