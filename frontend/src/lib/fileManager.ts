import { System } from '@wailsio/runtime'

export function fileManagerName(): string {
  if (System.IsMac()) return 'Finder'
  if (System.IsWindows()) return 'File Explorer'
  return 'File Manager'
}

export function revealInFileManagerLabel(): string {
  if (System.IsMac()) return 'Reveal in Finder'
  if (System.IsWindows()) return 'Reveal in File Explorer'
  return 'Open in File Manager'
}
