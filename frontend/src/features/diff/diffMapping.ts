// git's well-known empty tree; diffing a root commit against it matches `git show`.
const EMPTY_TREE_SHA = '4b825dc642cb6eb9a060e54bf8d69288fbee4904'

export function diffBaseFor(parentShas: string[]): string {
  return parentShas[0] ?? EMPTY_TREE_SHA
}
