interface AsciiGraphRow {
  sha: string
  subject: string
  lane: number
  passThrough: number[]
}

// One `*`/`|` column per lane, without git log --graph's diagonal connectors.
export function buildAsciiGraph(rows: AsciiGraphRow[]): string {
  if (rows.length === 0) return ''
  const laneCount = rows.reduce((max, r) => Math.max(max, r.lane, ...r.passThrough), 0) + 1
  return rows
    .map((r) => {
      const columns = new Array(laneCount).fill(' ')
      for (const lane of r.passThrough) columns[lane] = '|'
      columns[r.lane] = '*'
      return `${columns.join(' ')}  ${r.sha.slice(0, 7)} ${r.subject}`
    })
    .join('\n')
}
