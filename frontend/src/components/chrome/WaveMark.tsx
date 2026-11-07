// The product mark from the app icon: the current's single wave.
function WaveMark({ width = 56 }: { width?: number }) {
  return (
    <svg className="wave-mark" width={width} viewBox="0 0 100 40" fill="none" aria-hidden="true">
      <path
        d="M6 28C20 16 30 8 44 9c13 1 22 18 37 17 6-.5 10-3 13-6"
        stroke="currentColor"
        strokeWidth="7.5"
        strokeLinecap="round"
      />
    </svg>
  )
}

export default WaveMark
