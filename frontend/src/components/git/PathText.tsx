import './PathText.scss'

interface PathTextProps {
  path: string
  className?: string
}

function PathText({ path, className }: PathTextProps) {
  return (
    <span className={className ? `path-text ${className}` : 'path-text'} title={path}>
      <span className="path-text-inner">{path}</span>
    </span>
  )
}

export default PathText
