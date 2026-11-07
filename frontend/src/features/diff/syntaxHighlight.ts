import Prism from 'prismjs'
import 'prismjs/components/prism-clike'
import 'prismjs/components/prism-javascript'
import 'prismjs/components/prism-typescript'
import 'prismjs/components/prism-go'
import 'prismjs/components/prism-markup'
import 'prismjs/components/prism-markup-templating'
import 'prismjs/components/prism-css'
import 'prismjs/components/prism-scss'
import 'prismjs/components/prism-jsx'
import 'prismjs/components/prism-tsx'
import 'prismjs/components/prism-json'
import 'prismjs/components/prism-yaml'
import 'prismjs/components/prism-toml'
import 'prismjs/components/prism-markdown'
import 'prismjs/components/prism-bash'
import 'prismjs/components/prism-python'
import 'prismjs/components/prism-ruby'
import 'prismjs/components/prism-rust'
import 'prismjs/components/prism-java'
import 'prismjs/components/prism-kotlin'
import 'prismjs/components/prism-swift'
import 'prismjs/components/prism-c'
import 'prismjs/components/prism-cpp'
import 'prismjs/components/prism-csharp'
import 'prismjs/components/prism-php'
import 'prismjs/components/prism-sql'

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
  tag: 'keyword',
  atrule: 'keyword',
  doctype: 'keyword',
  'attr-name': 'type',
  'attr-value': 'string',
  selector: 'type',
  property: 'type',
  key: 'type',
  title: 'type',
}

const languageByExtension: Record<string, string> = {
  go: 'go',
  ts: 'typescript',
  mts: 'typescript',
  cts: 'typescript',
  tsx: 'tsx',
  js: 'javascript',
  mjs: 'javascript',
  cjs: 'javascript',
  jsx: 'jsx',
  html: 'markup',
  htm: 'markup',
  xml: 'markup',
  svg: 'markup',
  vue: 'markup',
  css: 'css',
  scss: 'scss',
  json: 'json',
  yml: 'yaml',
  yaml: 'yaml',
  toml: 'toml',
  md: 'markdown',
  markdown: 'markdown',
  sh: 'bash',
  bash: 'bash',
  zsh: 'bash',
  py: 'python',
  rb: 'ruby',
  rs: 'rust',
  java: 'java',
  kt: 'kotlin',
  kts: 'kotlin',
  swift: 'swift',
  c: 'c',
  h: 'c',
  cc: 'cpp',
  cpp: 'cpp',
  hpp: 'cpp',
  cs: 'csharp',
  php: 'php',
  sql: 'sql',
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
