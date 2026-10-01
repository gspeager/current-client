import Prism from 'prismjs'
import 'prismjs/components/prism-clike'
import 'prismjs/components/prism-javascript'
import 'prismjs/components/prism-typescript'
import 'prismjs/components/prism-go'

type SyntaxRole = 'keyword' | 'string' | 'type' | 'comment' | null

const roleByTokenType: Record<string, SyntaxRole> = {
  comment: 'comment',
  string: 'string',
  'template-string': 'string',
  char: 'string',
  regex: 'string',
  keyword: 'keyword',
  boolean: 'keyword',
  'class-name': 'type',
  builtin: 'type',
  generic: 'type',
  'generic-function': 'type',
  decorator: 'type',
}

const languageByExtension: Record<string, string> = {
  go: 'go',
  ts: 'typescript',
  js: 'javascript',
  mjs: 'javascript',
  cjs: 'javascript',
}

export function languageForPath(path: string): string | null {
  const match = /\.([a-zA-Z0-9]+)$/.exec(path)
  return match ? (languageByExtension[match[1].toLowerCase()] ?? null) : null
}

interface HighlightedSpan {
  text: string
  role: SyntaxRole
}

function flatten(tokens: (string | Prism.Token)[], role: SyntaxRole, out: HighlightedSpan[]): void {
  for (const token of tokens) {
    if (typeof token === 'string') {
      if (token) out.push({ text: token, role })
      continue
    }
    const tokenRole = roleByTokenType[token.type] ?? role
    if (Array.isArray(token.content)) {
      flatten(token.content, tokenRole, out)
    } else {
      out.push({ text: String(token.content), role: tokenRole })
    }
  }
}

export function tokenizeLine(content: string, language: string | null): HighlightedSpan[] {
  const grammar = language ? Prism.languages[language] : undefined
  if (!grammar) {
    return [{ text: content, role: null }]
  }
  const spans: HighlightedSpan[] = []
  flatten(Prism.tokenize(content, grammar), null, spans)
  return spans
}
