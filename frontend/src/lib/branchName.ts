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
