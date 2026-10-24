// Turns typed text into a name `git check-ref-format --branch` accepts, the way
// other Git clients do: whitespace and characters Git forbids become dashes.
export function toBranchName(input: string): string {
  const name = input
    .replace(/[\s\p{Cc}~^:?*[\\]+|@\{|\.{2,}/gu, '-')
    .replace(/-{2,}/g, '-')
    .split('/')
    .map((part) => part.replace(/^[.-]+/, '').replace(/(\.lock|[.-])+$/, ''))
    .filter((part) => part !== '')
    .join('/')
  return name === '@' ? '' : name
}

const ISSUE_SLUG_MAX = 40

// "123-short-title", the shape branchIssue reads back. The title is cut at a
// word so the name stays short.
export function issueBranchName(number: number, title: string): string {
  const words = title
    .toLowerCase()
    .split(/[^\p{L}\p{N}]+/u)
    .filter(Boolean)
  let slug = words[0]?.slice(0, ISSUE_SLUG_MAX) ?? ''
  for (const word of words.slice(1)) {
    if (slug.length + 1 + word.length > ISSUE_SLUG_MAX) break
    slug += `-${word}`
  }
  return slug ? `${number}-${slug}` : `${number}-issue`
}
