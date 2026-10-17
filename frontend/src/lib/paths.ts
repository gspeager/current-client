export function baseName(path: string): string {
  return path.split(/[\\/]/).filter(Boolean).pop() ?? path
}

// For display only; uses the separator the parent path already uses.
export function joinPath(parent: string, name: string): string {
  const sep = parent.includes('\\') && !parent.includes('/') ? '\\' : '/'
  return parent.replace(/[\\/]+$/, '') + sep + name
}

// Shows the home folder as ~, as macOS and Linux shells do.
export function tildePath(path: string, home: string): string {
  if (!home) return path
  if (path === home) return '~'
  return path.startsWith(home + '/') ? '~' + path.slice(home.length) : path
}
