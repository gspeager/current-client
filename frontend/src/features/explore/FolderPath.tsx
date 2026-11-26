import { baseName } from '../../lib/paths'

// The folder that fills the view, each folder above it a step back.
function FolderPath({ folder, onFolder }: { folder: string; onFolder: (folder: string) => void }) {
  const crumbs = folder === '' ? [] : folder.split('/').map((_, i, names) => names.slice(0, i + 1).join('/'))
  return (
    <nav className="explore-path" aria-label="Folder">
      <button type="button" className="explore-crumb" disabled={folder === ''} onClick={() => onFolder('')}>
        All
      </button>
      {crumbs.map((path) => (
        <button
          key={path}
          type="button"
          className="explore-crumb"
          disabled={path === folder}
          onClick={() => onFolder(path)}
        >
          {baseName(path)}
        </button>
      ))}
    </nav>
  )
}

export default FolderPath
