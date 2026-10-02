// The folder `git clone <url>` would create on its own: the last path segment,
// without a trailing .git or .bundle.
export function cloneFolderName(url: string): string {
  const path = url
    .trim()
    .replace(/[\\/]+$/, '')
    .replace(/[\\/]\.git$/, '')
  const last = path.split(/[\\/:]/).pop() ?? ''
  return last.replace(/\.(git|bundle)$/, '').trim()
}
