import { useState } from 'react'
import { Columns2, Layers, SquareSplitHorizontal } from 'lucide-react'
import { DiffService, type FileContent, type FileSource } from '@current-client-bindings/app'
import SegmentedControl from '../../components/controls/SegmentedControl'
import { useAsyncData } from '../../lib/useAsyncData'
import { imageMimeType } from './imageFiles'
import './ImageDiff.scss'

export interface ImageSources {
  repoPath: string
  before: FileSource
  after: FileSource
}

type ImageDiffMode = 'side' | 'swipe' | 'onion'

interface ImageDiffProps extends ImageSources {
  path: string
  // Reloads the images in place when it changes, such as after the file changes on disk.
  refreshKey?: unknown
}

function formatSize(bytes: number): string {
  return bytes < 1024 ? `${bytes} B` : `${(bytes / 1024).toFixed(1)} KB`
}

// Base64 carries 3 bytes in every 4 characters, less the padding.
function decodedSize(base64: string): number {
  return Math.floor((base64.length * 3) / 4) - (base64.endsWith('==') ? 2 : base64.endsWith('=') ? 1 : 0)
}

function ImageVersion({ label, content, src }: { label: string; content: FileContent; src: string }) {
  const [dimensions, setDimensions] = useState<string | null>(null)
  return (
    <figure className="image-diff-version">
      <figcaption className="image-diff-caption">
        <span>{label}</span>
        {!content.tooLarge && (
          <span className="image-diff-meta">
            {dimensions && `${dimensions} · `}
            {formatSize(decodedSize(content.data))}
          </span>
        )}
      </figcaption>
      {content.tooLarge ? (
        <p className="image-diff-note">Too large to show.</p>
      ) : (
        <img
          className="image-diff-image"
          src={src}
          alt={label}
          onLoad={(e) => setDimensions(`${e.currentTarget.naturalWidth} × ${e.currentTarget.naturalHeight}`)}
        />
      )}
    </figure>
  )
}

function ImageDiff({ repoPath, path, before, after, refreshKey }: ImageDiffProps) {
  const [mode, setMode] = useState<ImageDiffMode>('side')
  const [position, setPosition] = useState(50)
  const { data, error } = useAsyncData(
    () =>
      Promise.all([
        DiffService.GetFileContent(repoPath, path, before),
        DiffService.GetFileContent(repoPath, path, after),
      ]),
    // Callers build before/after inline, so compare their fields rather than the objects.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [repoPath, path, before.kind, before.rev, after.kind, after.rev],
    { refreshKey },
  )

  if (error) return <p className="diff-empty-state">Could not load the images: {error}</p>
  if (!data) return <p className="diff-empty-state">Loading images…</p>

  const [old, current] = data
  const src = (content: FileContent) => `data:${imageMimeType(path)};base64,${content.data}`

  // Added or deleted: one version, nothing to compare.
  if (!old.found || !current.found) {
    const only = old.found ? old : current
    if (!only.found) return <p className="diff-empty-state">Binary file changed.</p>
    return (
      <div className="image-diff">
        <ImageVersion label={old.found ? 'Deleted' : 'Added'} content={only} src={src(only)} />
      </div>
    )
  }

  const comparable = !old.tooLarge && !current.tooLarge

  return (
    <div className="image-diff">
      {comparable && (
        <div className="image-diff-toolbar">
          <SegmentedControl
            value={mode}
            onChange={setMode}
            options={[
              { value: 'side', label: 'Side by side', icon: Columns2 },
              { value: 'swipe', label: 'Swipe', icon: SquareSplitHorizontal },
              { value: 'onion', label: 'Onion skin', icon: Layers },
            ]}
          />
          {mode !== 'side' && (
            <input
              type="range"
              className="image-diff-slider"
              min={0}
              max={100}
              value={position}
              onChange={(e) => setPosition(Number(e.target.value))}
              aria-label={mode === 'swipe' ? 'Swipe position' : 'Opacity of the new version'}
            />
          )}
        </div>
      )}
      {mode === 'side' || !comparable ? (
        <div className="image-diff-side">
          <ImageVersion label="Before" content={old} src={src(old)} />
          <ImageVersion label="After" content={current} src={src(current)} />
        </div>
      ) : (
        <div className="image-diff-stack">
          <img className="image-diff-image" src={src(old)} alt="Before" />
          <img
            className="image-diff-image"
            src={src(current)}
            alt="After"
            style={mode === 'swipe' ? { clipPath: `inset(0 0 0 ${position}%)` } : { opacity: position / 100 }}
          />
        </div>
      )}
    </div>
  )
}

export default ImageDiff
